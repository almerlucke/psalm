package grains

import (
	"math"
	"math/rand/v2"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

// Config describes a granular sampler voice: every note starts a stream of grains, short windowed
// slices of the sample played at the pitch of the note
type Config struct {
	NumChannels int

	Sample    sndfile.SoundFiler[float32]
	SoundBank sndfile.SoundBank[float32] // optional, notes can pick a sample with the "sound" key
	RootNote  float64                    // MIDI pitch of the sample, playing this note plays it at its own pitch

	Position        float64 // 0-1, where in the sample the grains come from
	Scan            float64 // how fast the position moves through the sample: 0 freezes, 1 is the original speed, negative goes backwards
	Spray           float64 // seconds, random spread of the grain positions around the position
	Density         float64 // grains per second
	GrainSize       float64 // milliseconds
	PitchSpread     float64 // semitones, random detune per grain (chorus, shimmer)
	Octave          float64 // 0-1, chance a grain plays an octave up (sparkle, shimmer)
	StereoSpread    float64 // 0-1, random pan per grain
	Shape           float64 // 0-1, grain window from smooth (0) to percussive (1)
	Reverse         float64 // 0-1, chance a grain plays backwards
	Jitter          float64 // 0-1, randomness of the time between grains
	VelocityDensity float64 // 0-1, how much the note velocity scales the density
	Level           float64 // output level (0 is treated as 1)
	Normalize       bool    // play every sample at the same loudness (recordings differ a lot in level)

	AmpEnv adsr.Setting // envelope of the whole note (durations in milliseconds)
}

func DefaultConfig() Config {
	return Config{
		NumChannels:  psalm.Stereo,
		RootNote:     60,
		Position:     0.3,
		Scan:         0.0,
		Spray:        0.05,
		Density:      30,
		GrainSize:    120,
		StereoSpread: 0.5,
		Jitter:       0.3,
		Level:        1.0,
		Normalize:    true,
		AmpEnv:       *adsr.NewSetting(1.0, 50.0, 0.9, 200.0, 1.0, 500.0),
	}
}

// loudness caches the loudness of every sample used for normalizing, so switching samples per note
// stays cheap (the audio thread is the only user)
var loudness = map[sndfile.SoundFiler[float32]]float64{}

// normalizedRMS is the loudness every sample is normalized to
const normalizedRMS = 0.2

// sampleLoudness returns the RMS of the parts of the sample that sound: the RMS of 50 ms blocks,
// ignoring blocks more than 30 dB below the loudest one, so silence in a recording doesn't make
// it count as quiet
func sampleLoudness(s sndfile.SoundFiler[float32]) float64 {
	if l, ok := loudness[s]; ok {
		return l
	}

	n := int(s.NumFrames())
	block := max(1, int(0.05*s.SampleRate()))
	var blocks []float64
	top := 0.0
	for start := 0; start < n; start += block {
		end := min(n, start+block)
		sum := 0.0
		for ch := range s.NumChannels() {
			buf := s.Buffer(ch, 0)
			for _, v := range buf[start:end] {
				sum += float64(v) * float64(v)
			}
		}
		rms := math.Sqrt(sum / float64((end-start)*s.NumChannels()))
		blocks = append(blocks, rms)
		top = math.Max(top, rms)
	}

	sum, count := 0.0, 0
	for _, r := range blocks {
		if r > top*0.0316 { // within 30 dB of the loudest block
			sum += r * r
			count++
		}
	}

	l := 0.0
	if count > 0 {
		l = math.Sqrt(sum / float64(count))
	}
	loudness[s] = l
	return l
}

// windowCompensation scales a grain shape to the energy of the smooth window, so percussive grains
// (which decay fast and carry less energy) aren't quieter
func windowCompensation(shape float64) float64 {
	if shape <= 0 {
		return 1
	}
	const steps = 256
	smooth, sh := 0.0, 0.0
	for i := range steps {
		t := (float64(i) + 0.5) / steps
		ws, w := window(t, 0), window(t, shape)
		smooth += ws * ws
		sh += w * w
	}
	return math.Sqrt(smooth / sh)
}

// grain is one playing grain
type grain struct {
	pos    float64 // frames in the sample
	inc    float64 // frames per output sample (negative plays backwards)
	depth  int     // mipmap depth for the speed
	length int     // samples
	age    int
	gainL  float64
	gainR  float64
	sample sndfile.SoundFiler[float32]
}

// maxGrains is the maximum number of simultaneous grains of a voice
const maxGrains = 96

// Voice is a granular sampler voice. Notes take an optional "velocity" (0-1, default 1) in their
// message, the note amplitude sets the volume. Value inputs: 1 position, 2 density, 3 scan,
// 4 spray, 5 grain size. They only affect new grains, so changing them never clicks
type Voice struct {
	*psalm.Module
	cfg       Config
	sample    sndfile.SoundFiler[float32]
	ampEnv    *adsr.ADSR
	grains    []grain
	pitch     float64
	velocity  float64
	position  float64 // frames, moves with the scan
	countdown float64 // samples until the next grain
	waitSamps int
}

func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}

	v := &Voice{
		Module:   psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:      cfg,
		sample:   cfg.Sample,
		pitch:    cfg.RootNote,
		velocity: 1.0,
		grains:   make([]grain, 0, maxGrains),
	}
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, v.Config().SampleRate)
	v.SetSelf(v)

	return v
}

func (v *Voice) SetSample(sample sndfile.SoundFiler[float32]) {
	v.sample = sample
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	cfg := &v.cfg
	msg.IfStrVal("sound", func(name string) {
		if s, ok := cfg.SoundBank[name]; ok {
			v.sample = s
		}
	})
	msg.IfFltVal("rootNote", func(f float64) { cfg.RootNote = f })
	msg.IfFltVal("position", func(f float64) { cfg.Position = f })
	msg.IfFltVal("scan", func(f float64) { cfg.Scan = f })
	msg.IfFltVal("spray", func(f float64) { cfg.Spray = f })
	msg.IfFltVal("density", func(f float64) { cfg.Density = f })
	msg.IfFltVal("grainSize", func(f float64) { cfg.GrainSize = f })
	msg.IfFltVal("pitchSpread", func(f float64) { cfg.PitchSpread = f })
	msg.IfFltVal("stereoSpread", func(f float64) { cfg.StereoSpread = f })
	msg.IfFltVal("octave", func(f float64) { cfg.Octave = f })
	msg.IfFltVal("shape", func(f float64) { cfg.Shape = f })
	msg.IfFltVal("reverse", func(f float64) { cfg.Reverse = f })
	msg.IfFltVal("jitter", func(f float64) { cfg.Jitter = f })
	msg.IfFltVal("level", func(f float64) { cfg.Level = f })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0.0, math.Min(f, 1.0)) })
	msg.IfFltVal("pitch", func(f float64) { v.pitch = f })
	msg.IfFltVal("frequency", func(f float64) { v.pitch = 69.0 + 12.0*math.Log2(f/440.0) })
}

func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}

	f := psalm.FltVal[float64](value)
	switch index {
	case 1:
		v.cfg.Position = f
		v.position = v.positionFrames() // jump there, the scan continues from the new position
	case 2:
		v.cfg.Density = f
	case 3:
		v.cfg.Scan = f
	case 4:
		v.cfg.Spray = f
	case 5:
		v.cfg.GrainSize = f
	}
}

func (v *Voice) positionFrames() float64 {
	if v.sample == nil {
		return 0
	}
	return math.Max(0.0, math.Min(v.cfg.Position, 1.0)) * float64(v.sample.NumFrames()-1)
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.NoteOff, 0.0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.Duration, duration, amplitude, message, tick)
}

func (v *Voice) note(mode adsr.ReleaseMode, duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 1.0
	v.ReceiveMessage(message)

	v.waitSamps = tick.SampleOffset
	v.position = v.positionFrames()
	v.countdown = 0 // the first grain starts right away
	v.ampEnv.TriggerFull(duration*1000.0, amplitude, &v.cfg.AmpEnv, mode)
}

func (v *Voice) NoteOff() {
	v.ampEnv.Release()
}

func (v *Voice) Clear() {
	v.ampEnv.Clear()
	v.grains = v.grains[:0]
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return !v.ampEnv.Done()
}

// window returns the grain envelope at t (0-1): a smooth sin^2 window, morphing to a percussive
// shape with a fast attack and an exponential decay
func window(t float64, shape float64) float64 {
	smooth := math.Sin(math.Pi * t)
	smooth *= smooth

	if shape <= 0.0 {
		return smooth
	}

	const attack = 0.03
	var perc float64
	if t < attack {
		perc = t / attack
	} else {
		// starts at 1 where the attack ends and reaches 0 at the end of the grain
		perc = math.Exp(-6.0*(t-attack)/(1.0-attack)) * (1.0 - t) / (1.0 - attack)
	}

	return smooth + (perc-smooth)*shape
}

// spawn starts a new grain
func (v *Voice) spawn(sr float64) {
	cfg := &v.cfg
	s := v.sample
	frames := float64(s.NumFrames())

	if len(v.grains) >= maxGrains || frames < 2 {
		return
	}

	semis := v.pitch - cfg.RootNote
	if cfg.PitchSpread != 0 {
		semis += cfg.PitchSpread * (rand.Float64()*2.0 - 1.0)
	}
	if cfg.Octave > 0 && rand.Float64() < cfg.Octave {
		semis += 12
	}
	speed := math.Exp2(semis/12.0) * s.SampleRate() / sr

	pos := v.position + cfg.Spray*(rand.Float64()*2.0-1.0)*s.SampleRate()
	pos -= math.Floor(pos/frames) * frames // wrap into the sample

	if rand.Float64() < cfg.Reverse {
		speed = -speed
	}

	pan := 0.5 + cfg.StereoSpread*(rand.Float64()-0.5)

	depth := sndfile.SpeedToMipMapDepth(speed)
	if depth >= s.Depth() {
		depth = s.Depth() - 1
	}

	v.grains = append(v.grains, grain{
		pos:    pos,
		inc:    speed,
		depth:  depth,
		length: max(1, int(cfg.GrainSize*0.001*sr)),
		gainL:  math.Cos(pan * math.Pi / 2.0),
		gainR:  math.Sin(pan * math.Pi / 2.0),
		sample: s,
	})
}

func (v *Voice) Synthesize() {
	outSocket := v.Output(0)
	outSocket.Clear()

	if v.ampEnv.Done() || v.sample == nil {
		v.grains = v.grains[:0]
		return
	}

	var (
		cfg     = &v.cfg
		sr      = v.Config().SampleRate
		outs    = outSocket.Channels
		bufSize = v.Config().BufferSize
		frames  = float64(v.sample.NumFrames())
		scanInc = cfg.Scan * v.sample.SampleRate() / sr
		vel     = 1.0 - cfg.VelocityDensity + cfg.VelocityDensity*v.velocity
		density = math.Max(0.1, cfg.Density*vel)
		level   = cfg.Level
	)

	if level <= 0 {
		level = 1
	}

	// overlapping grains add up: scale by the square root of the average overlap so the level
	// stays about the same when the density or grain size changes
	overlap := math.Max(1.0, density*cfg.GrainSize*0.001)
	norm := level / math.Sqrt(overlap)

	norm *= windowCompensation(cfg.Shape)

	if cfg.Normalize {
		if l := sampleLoudness(v.sample); l > 1e-6 {
			norm *= normalizedRMS / l
		}
	}

	interval := sr / density

	for i := v.waitSamps; i < bufSize; i++ {
		env := v.ampEnv.Generate()

		// start grains while the note sounds
		v.countdown--
		if v.countdown <= 0 {
			v.spawn(sr)
			v.countdown += interval * (1.0 + cfg.Jitter*(rand.Float64()*2.0-1.0))
			if v.countdown < 1 {
				v.countdown = 1
			}
		}

		// move the position through the sample
		v.position += scanInc
		v.position -= math.Floor(v.position/frames) * frames

		var left, right float64
		for g := 0; g < len(v.grains); {
			gr := &v.grains[g]
			t := float64(gr.age) / float64(gr.length)
			w := window(t, cfg.Shape)

			n := float64(gr.sample.NumFrames())
			values := gr.sample.LookupAll(gr.pos, gr.depth, true)
			x := 0.0
			for _, c := range values {
				x += float64(c)
			}
			x = x / float64(len(values)) * w

			left += x * gr.gainL
			right += x * gr.gainR

			gr.pos += gr.inc
			gr.pos -= math.Floor(gr.pos/n) * n
			gr.age++

			if gr.age >= gr.length {
				// swap with the last grain and drop it
				v.grains[g] = v.grains[len(v.grains)-1]
				v.grains = v.grains[:len(v.grains)-1]
			} else {
				g++
			}
		}

		gain := env * norm
		if len(outs) == 1 {
			outs[0][i] = float32((left + right) * 0.5 * gain)
		} else {
			outs[0][i] = float32(left * gain)
			outs[1][i] = float32(right * gain)
		}

		if v.ampEnv.Done() {
			v.grains = v.grains[:0]
			break
		}
	}

	v.waitSamps = 0
}

// NewPoly creates a polyphonic granular sampler
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}

	voices := make([]*Voice, numVoices)
	for i := range voices {
		voices[i] = New(cfg)
	}

	return poly.New[*Voice](cfg.NumChannels, voices)
}
