package grains

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

// Config describes a granular sampler voice: every note starts a stream of grains, short windowed
// slices of the sample played at the pitch of the note
type Config struct {
	NumChannels int

	Sample sndfile.SoundFiler[float32] // the recording the grains read ...
	Source Source                      // ... or any other source (a recorder of the sound that comes in, an oscillator); it is read instead of Sample

	// what the grains read, as the settings pick it (SourceRecording ...): a recording (Sample, or the sound of the bank), a single cycle wave of
	// the shared oscillator (Wave: 0 sine, 1 triangle, 2 saw, 3 square, 4 as the position says), or the grain buffer called Buffer ("live" without)
	SourceKind int
	Wave       int
	Buffer     string
	SoundBank  sndfile.SoundBank[float32] // optional, notes can pick a sample with the "sound" key
	RootNote   float64                    // MIDI pitch of the sample, playing this note plays it at its own pitch
	Roots      map[string]float64         // the MIDI pitch of the sounds of the bank, by name (a note that picks a sound plays it at its own pitch around this key); without, RootNote

	Position        float64 // 0-1, where in the sample the grains come from
	Scan            float64 // how fast the position moves through the sample: 0 freezes, 1 is the original speed, negative goes backwards
	Spray           float64 // seconds, random spread of the grain positions around the position
	Density         float64 // grains per second
	Sync            float64 // beats between the grains, following the tempo (0: Density is used); 0.25 is a grain every sixteenth
	GrainSize       float64 // milliseconds
	PitchSpread     float64 // semitones, random detune per grain (chorus, shimmer)
	Octave          float64 // 0-1, chance a grain plays an octave up (sparkle, shimmer)
	StereoSpread    float64 // 0-1, random pan per grain
	Shape           float64 // 0-1, grain window from smooth (0) to percussive (1)
	Window          int     // the shape the grain window starts from: WindowHann, WindowGauss, WindowTukey or WindowTriangle
	Reverse         float64 // 0-1, chance a grain plays backwards
	Jitter          float64 // 0-1, randomness of the time between grains
	VelocityDensity float64 // 0-1, how much the note velocity scales the density
	Level           float64 // output level (0 is treated as 1)
	Normalize       bool    // play every sample at the same loudness (recordings differ a lot in level)

	Scale     []float64 // hold the pitch of every grain to this scale: the semitones above its root (see ScaleNamed); none leaves the pitches as they are
	ScaleRoot float64   // the root of the scale, as a MIDI pitch (any octave)
	Stack     []float64 // every grain is a chord: one grain for each of these semitones above the pitch (0, 4, 7 is a major triad)
	Streams   Streams   // values for each grain that start from lists or generators (see Streams)

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
		top = max(top, rms)
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

// the sources a voice picks between by its settings (SourceKind)
const (
	SourceRecording = iota
	SourceWave
	SourceLive
	numSourceKinds
	sourceCustom = -1 // a source that a patch (a text of sounds, a script) picked by its name
)

// SourceNames are the names of the sources a voice picks between, in the order of SourceKind
var SourceNames = []string{"recording", "wave", "live"}

// WaveChoices are the choices of Wave: the waves, then "morph" (the position picks)
var WaveChoices = []string{"sine", "triangle", "saw", "square", "morph"}

// grain is one playing grain
type grain struct {
	pos    float64 // frames in the source
	lo, hi float64 // the part of the source it reads, and wraps in
	inc    float64 // frames per output sample (negative plays backwards)
	depth  int     // mipmap depth for the speed
	length int     // samples
	age    int
	delay  int // samples into the buffer where it starts (only in the buffer it was made in)
	gainL  float64
	gainR  float64
	src    Source
}

// maxGrains is the maximum number of simultaneous grains of a voice
const maxGrains = 96

// Voice is a granular sampler voice. Notes take an optional "velocity" (0-1, default 1) in their
// message, the note amplitude sets the volume. Value inputs: 1 position, 2 density, 3 scan,
// 4 spray, 5 grain size. They only affect new grains, so changing them never clicks
type Voice struct {
	*psalm.Module
	cfg       Config
	src       Source
	kind      int    // which source of the settings is read (SourceKind), sourceCustom for one picked by name
	rec       Source // the recording of the settings
	live      Source
	bank      map[string]Source // the sounds of the bank as sources
	streams   Streams
	st        voiceStreams
	ampEnv    *adsr.ADSR
	grains    []grain
	pitch     float64
	expr      expression.Expression // the pitch bend (new grains); the mod wheel sprays them wider
	velocity  float64
	scanned   float64 // frames the position has moved on since the note began (or the position was set)
	countdown float64 // samples until the next grain
	waitSamps int
	env       []float64 // the envelope of the buffer
	left      []float64 // the grains of the buffer, summed
	right     []float64
	scratch   []float64 // what a grain reads
}

func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}

	v := &Voice{
		Module:   psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:      cfg,
		src:      cfg.Source,
		streams:  cfg.Streams,
		pitch:    cfg.RootNote,
		velocity: 1.0,
		grains:   make([]grain, 0, maxGrains),
	}
	if v.src == nil && cfg.Sample != nil {
		v.src = NewSampleSource(cfg.Sample)
	}
	if len(cfg.SoundBank) > 0 {
		v.bank = make(map[string]Source, len(cfg.SoundBank))
		for name, s := range cfg.SoundBank {
			v.bank[name] = NewSampleSource(s)
		}
	}
	// the source of the settings: a Source in the config is the recording
	v.rec = v.src
	v.kind = cfg.SourceKind
	if cfg.Source != nil && cfg.SourceKind == SourceRecording {
		v.kind = sourceCustom
	}
	v.live = LiveSource(cfg.liveName())
	v.pick()
	v.st = v.streams.make()
	n := v.Config().BufferSize
	v.env, v.left, v.right, v.scratch = make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, v.Config().SampleRate)
	v.SetSelf(v)

	return v
}

func (c *Config) liveName() string {
	if c.Buffer == "" {
		return DefaultRecorderName
	}
	return c.Buffer
}

// pick makes the source of the settings the one that is read (not when a patch picked one by its name)
func (v *Voice) pick() {
	switch v.kind {
	case SourceRecording:
		v.src = v.rec
	case SourceWave:
		v.src = SharedOscillator()
	case SourceLive:
		v.src = v.live
	}
}

// SetSample makes a recording what the grains read
func (v *Voice) SetSample(sample sndfile.SoundFiler[float32]) {
	v.rec = NewSampleSource(sample)
	v.kind = SourceRecording
	v.pick()
}

// SetSource sets what the grains read (nil: nothing, the voice is silent)
func (v *Voice) SetSource(src Source) {
	v.src = src
	v.kind = sourceCustom
}

// useNamed reads the source a sounds text calls by a name: a recording is the recording of the settings, a made source (a wave, a
// buffer) is read as it is
func (v *Voice) useNamed(s Source) {
	if _, ok := s.(*sampleSource); ok {
		v.rec = s
		v.kind = SourceRecording
		v.pick()
		return
	}
	v.src = s
	v.kind = sourceCustom
}

// seqList reads a list of numbers of a message, nil when it is not one
func seqList(x any) ([]float64, bool) {
	l, ok := x.([]float64)
	return l, ok
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	envelope.Receive(msg, "amp", &v.cfg.AmpEnv)
	cfg := &v.cfg
	// a bank of sounds (and the pitch of each) sent by a patch that built it off the audio thread; the first sound is played until a note picks one
	if bank, ok := msg["bank"].(sndfile.SoundBank[float32]); ok {
		cfg.SoundBank = bank
		v.bank = make(map[string]Source, len(bank))
		for name, s := range bank {
			v.bank[name] = NewSampleSource(s)
		}
		if roots, ok := msg["roots"].(map[string]float64); ok {
			cfg.Roots = roots
		}
		if first, ok := msg["first"].(string); ok {
			if s, ok := v.bank[first]; ok {
				v.useNamed(s)
				if r, ok := cfg.Roots[first]; ok {
					cfg.RootNote = r
				}
			}
		}
	}
	// a source built by a patch (a recorder's, an oscillator): the key "source" (a Source) or "sources" (named ones, "first" is played)
	if s, ok := msg["source"].(Source); ok {
		v.SetSource(s)
	}
	if all, ok := msg["sources"].(map[string]Source); ok {
		if v.bank == nil {
			v.bank = map[string]Source{}
		}
		for name, s := range all {
			v.bank[name] = s
		}
		if first, ok := msg["first"].(string); ok && v.bank[first] != nil {
			v.useNamed(v.bank[first])
		}
	}
	msg.IfStrVal("sound", func(name string) {
		if s, ok := v.bank[name]; ok {
			v.useNamed(s)
			if r, ok := cfg.Roots[name]; ok {
				cfg.RootNote = r
			}
		}
	})
	msg.IfFltVal("source", func(f float64) {
		cfg.SourceKind = max(0, min(int(math.Round(f)), numSourceKinds-1))
		v.kind = cfg.SourceKind
		v.pick()
	})
	msg.IfFltVal("wave", func(f float64) { cfg.Wave = max(0, min(int(math.Round(f)), len(WaveChoices)-1)) })
	msg.IfStrVal("buffer", func(name string) {
		cfg.Buffer = name
		v.live = LiveSource(cfg.liveName())
		if v.kind == SourceLive {
			v.pick()
		}
	})
	msg.IfFltVal("rootNote", func(f float64) { cfg.RootNote = f })
	msg.IfFltVal("position", func(f float64) { cfg.Position = f })
	msg.IfFltVal("scan", func(f float64) { cfg.Scan = f })
	msg.IfFltVal("spray", func(f float64) { cfg.Spray = f })
	msg.IfFltVal("density", func(f float64) { cfg.Density = f })
	msg.IfFltVal("sync", func(f float64) { cfg.Sync = max(0, f) })
	msg.IfFltVal("grainSize", func(f float64) { cfg.GrainSize = f })
	msg.IfFltVal("pitchSpread", func(f float64) { cfg.PitchSpread = f })
	msg.IfFltVal("stereoSpread", func(f float64) { cfg.StereoSpread = f })
	msg.IfFltVal("octave", func(f float64) { cfg.Octave = f })
	msg.IfFltVal("shape", func(f float64) { cfg.Shape = f })
	msg.IfFltVal("window", func(f float64) { cfg.Window = max(0, min(int(math.Round(f)), numWindows-1)) })
	msg.IfFltVal("reverse", func(f float64) { cfg.Reverse = f })
	msg.IfFltVal("jitter", func(f float64) { cfg.Jitter = f })
	msg.IfFltVal("level", func(f float64) { cfg.Level = f })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = max(0.0, min(f, 1.0)) })
	msg.IfFltVal("pitch", func(f float64) { v.pitch = f })
	msg.IfFltVal("frequency", func(f float64) { v.pitch = 69.0 + 12.0*math.Log2(f/440.0) })
	// the pitch of the grains: held to a scale (a list of the semitones above its root, or a name), several at once (a stack)
	msg.IfFltVal("scaleRoot", func(f float64) { cfg.ScaleRoot = f })
	// the scale and the stack: a list (the semitones), a name, or the number of the choice of the setting (see ScaleChoices and StackChoices)
	switch x := msg["scale"].(type) {
	case []float64:
		cfg.Scale = append([]float64(nil), x...)
	case string:
		cfg.Scale = ScaleNamed(x)
	case nil:
	default:
		if i := int(math.Round(psalm.FltVal[float64](x))); i > 0 && i <= len(ScaleNames) {
			cfg.Scale = ScaleNamed(ScaleNames[i-1])
		} else {
			cfg.Scale = nil
		}
	}
	switch x := msg["stack"].(type) {
	case []float64:
		cfg.Stack = append([]float64(nil), x...)
	case nil:
	default:
		cfg.Stack = stackOf(int(math.Round(psalm.FltVal[float64](x))))
	}
	// a stream of a list of numbers (see streamKeys)
	for i, key := range streamKeys {
		l, ok := seqList(msg[key])
		if !ok {
			continue
		}
		var s Stream
		if len(l) > 0 {
			s = Seq(l...)
		}
		slots := []*Stream{&v.st.interval, &v.st.position, &v.st.pitch, &v.st.size, &v.st.pan, &v.st.amp}
		*slots[i] = s
	}
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
		v.scanned = 0 // jump there, the scan continues from the new position
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
	v.scanned = 0
	v.st.reset()
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

// quantize holds a pitch to the nearest tone of a scale (the semitones above its root, within an octave)
func quantize(pitch float64, scale []float64, root float64) float64 {
	if len(scale) == 0 {
		return pitch
	}
	rel := pitch - root
	oct := math.Floor(rel / 12)
	in := rel - oct*12
	best, dist := scale[0], math.Inf(1)
	for _, d := range scale {
		for _, c := range []float64{d, d + 12, d - 12} {
			if x := math.Abs(in - c); x < dist {
				best, dist = c, x
			}
		}
	}
	return root + oct*12 + best
}

// spawn starts the grains of one grain event, at sample i of the buffer (a stack makes more than one)
func (v *Voice) spawn(i int, sr float64) {
	cfg := &v.cfg
	s := v.src

	position := cfg.Position
	if v.st.position != nil {
		position = v.st.position.Next()
	}
	if w, ok := s.(Waved); ok && v.kind == SourceWave && cfg.Wave < len(WaveChoices)-1 {
		position = float64(cfg.Wave) / float64(max(1, w.NumWaves()-1)) // one wave of the settings, not where the position is
	}
	extra := 0.0
	if v.st.pitch != nil {
		extra = v.st.pitch.Next()
	}
	size := cfg.GrainSize
	if v.st.size != nil {
		size = v.st.size.Next()
	}
	pan, panSet := 0.0, false
	if v.st.pan != nil {
		pan, panSet = max(0, min(v.st.pan.Next(), 1)), true
	}
	amp := 1.0
	if v.st.amp != nil {
		amp = max(0, v.st.amp.Next())
	}
	if amp == 0 {
		return // a rest
	}

	stack := cfg.Stack
	if len(stack) == 0 {
		stack = []float64{0}
	}
	length := max(1, int(size*0.001*sr))

	for _, up := range stack {
		if len(v.grains) >= maxGrains {
			return
		}

		pitch := v.pitch + extra + up
		if cfg.PitchSpread != 0 {
			pitch += cfg.PitchSpread * (v.Rand().Float64()*2.0 - 1.0)
		}
		if cfg.Octave > 0 && v.Rand().Float64() < cfg.Octave {
			pitch += 12
		}
		pitch = quantize(pitch, cfg.Scale, cfg.ScaleRoot)
		root := cfg.RootNote
		if v.kind == SourceWave {
			root = 60 // the shared waves sound at middle C
		}
		semis := pitch - root + v.expr.Semitones()
		speed := math.Exp2(semis/12.0) * s.Rate() / sr

		spray := (cfg.Spray + 0.3*v.expr.Wheel()) * (v.Rand().Float64()*2.0 - 1.0) * s.Rate()

		if v.Rand().Float64() < cfg.Reverse {
			speed = -speed
		}

		lead := float64(length) * (math.Abs(speed)*sr/s.Rate() - 1) // frames ahead of a source that is written as it is read
		start, lo, hi := s.Locate(position, v.scanned, spray, max(0, lead))

		if !panSet {
			pan = 0.5 + cfg.StereoSpread*(v.Rand().Float64()-0.5)
		}

		depth := sndfile.SpeedToMipMapDepth(speed)
		if depth >= s.Depth() {
			depth = s.Depth() - 1
		}

		v.grains = append(v.grains, grain{
			pos:    start,
			lo:     lo,
			hi:     hi,
			inc:    speed,
			depth:  depth,
			length: length,
			delay:  i,
			gainL:  amp * math.Cos(pan*math.Pi/2.0),
			gainR:  amp * math.Sin(pan*math.Pi/2.0),
			src:    s,
		})
	}
}

// interval is the number of samples until the next grain
func (v *Voice) interval(sr, density float64) float64 {
	cfg := &v.cfg
	var iv float64
	switch {
	case v.st.interval != nil:
		return max(1, v.st.interval.Next()*sr)
	case cfg.Sync > 0:
		iv = cfg.Sync * 60 / v.BPM() * sr
	default:
		iv = sr / density
	}
	return max(1, iv*(1.0+cfg.Jitter*(v.Rand().Float64()*2.0-1.0)))
}

func (v *Voice) Synthesize() {
	outSocket := v.Output(0)
	outSocket.Clear()

	if v.ampEnv.Done() || v.src == nil || !v.src.Ready() {
		v.grains = v.grains[:0]
		return
	}
	v.expr.Settle() // the grains take the bend and the wheel when they start

	var (
		cfg     = &v.cfg
		src     = v.src
		sr      = v.Config().SampleRate
		outs    = outSocket.Channels
		bufSize = v.Config().BufferSize
		scanInc = cfg.Scan * src.Rate() / sr
		vel     = 1.0 - cfg.VelocityDensity + cfg.VelocityDensity*v.velocity
		density = max(0.1, cfg.Density*vel)
		level   = cfg.Level
		span    = src.Span()
	)

	if level <= 0 {
		level = 1
	}

	// overlapping grains add up: scale by the square root of the average overlap so the level
	// stays about the same when the density or grain size changes
	grainMs := cfg.GrainSize
	if v.st.size != nil {
		grainMs = 100 // sizes that come from a stream: an average one
	}
	grainsPerSec := density
	if cfg.Sync > 0 {
		grainsPerSec = v.BPM() / 60 / cfg.Sync
	}
	overlap := max(1.0, grainsPerSec*grainMs*0.001*float64(max(1, len(cfg.Stack))))
	norm := level / math.Sqrt(overlap)

	norm *= windowCompensation(cfg.Shape, cfg.Window)

	if cfg.Normalize {
		if l := src.Loudness(); l > 1e-6 {
			norm *= normalizedRMS / l
		}
	}

	// the envelope of the note, and where the grains start in this buffer
	end := bufSize
	done := false
	clear(v.left[:bufSize])
	clear(v.right[:bufSize])
	for i := v.waitSamps; i < bufSize; i++ {
		v.env[i] = v.ampEnv.Generate()

		// start grains while the note sounds
		v.countdown--
		if v.countdown <= 0 {
			v.spawn(i, sr)
			v.countdown += v.interval(sr, density)
		}

		// move the position through the source
		v.scanned += scanInc
		v.scanned -= math.Floor(v.scanned/span) * span

		if v.ampEnv.Done() {
			end = i + 1
			done = true
			break
		}
	}

	// the grains, one at a time over the whole buffer
	for g := 0; g < len(v.grains); {
		gr := &v.grains[g]
		first := gr.delay
		gr.delay = 0
		n := min(end-first, gr.length-gr.age)
		if n > 0 {
			buf := v.scratch[:n]
			gr.pos = gr.src.Read(buf, gr.pos, gr.inc, gr.depth, gr.lo, gr.hi)
			for j, x := range buf {
				w := windowOf(float64(gr.age+j)/float64(gr.length), cfg.Shape, cfg.Window)
				x *= w
				v.left[first+j] += x * gr.gainL
				v.right[first+j] += x * gr.gainR
			}
			gr.age += n
		}
		if gr.age >= gr.length {
			// swap with the last grain and drop it
			v.grains[g] = v.grains[len(v.grains)-1]
			v.grains = v.grains[:len(v.grains)-1]
		} else {
			g++
		}
	}

	for i := v.waitSamps; i < end; i++ {
		gain := v.env[i] * norm
		if len(outs) == 1 {
			outs[0][i] = float32((v.left[i] + v.right[i]) * 0.5 * gain)
		} else {
			outs[0][i] = float32(v.left[i] * gain)
			outs[1][i] = float32(v.right[i] * gain)
		}
	}

	if done {
		v.grains = v.grains[:0]
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
