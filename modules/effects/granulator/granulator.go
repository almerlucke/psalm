package granulator

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
	fastsine "github.com/almerlucke/psalm/blocks/sine"
)

// Config for the granulator
type Config struct {
	BufferSeconds float64 // length of the recording buffer (and the longest delay)
	Delay         float64 // milliseconds, how far behind the input the grains read
	Spray         float64 // milliseconds, random spread of the grain positions
	Density       float64 // grains per second
	GrainSize     float64 // milliseconds
	Pitch         float64 // semitones
	PitchSpread   float64 // semitones, random detune per grain
	Octave        float64 // 0-1, chance a grain plays an octave up
	Reverse       float64 // 0-1, chance a grain plays backwards
	Shape         float64 // 0-1, grain window from smooth to percussive
	Jitter        float64 // 0-1, randomness of the time between grains
	StereoSpread  float64 // 0-1, random pan per grain
	Feedback      float64 // 0-1, grains fed back into the buffer
	Mix           float64 // 0 is dry only, 1 is wet only, an equal power crossfade that keeps the loudness
	Freeze        bool    // stop recording, the grains keep playing what is in the buffer
}

func DefaultConfig() Config {
	return Config{
		BufferSeconds: 4,
		Delay:         150,
		Spray:         80,
		Density:       25,
		GrainSize:     120,
		Jitter:        0.4,
		StereoSpread:  0.7,
		Mix:           0.5,
	}
}

type grain struct {
	pos    float64 // absolute read position in samples since the start (not wrapped)
	inc    float64
	length int
	age    int
	gainL  float64
	gainR  float64
}

const (
	maxGrains = 128
	margin    = 4.0 // samples kept away from the edges of the valid buffer
)

// Granulator records its stereo input into a buffer and plays grains from it: short windowed
// slices, read a little behind the input, with pitch, spray and stereo spread. Freezing stops the
// recording, so the grains keep playing the frozen moment. Feedback sends the grains back into
// the buffer. Value inputs: 1 freeze (above 0.5 freezes), 2 delay, 3 density, 4 grain size, 5 pitch
type Granulator struct {
	*psalm.Module
	cfg       Config
	buf       [2][]float32
	written   int64 // samples recorded so far, the write head
	grains    []grain
	countdown float64
	fbL, fbR  float64
	mix       mix.Smoother // mix, feedback and the grain level glide when they change
	fb        mix.Smoother
	norm      mix.Smoother
}

func New(cfg Config) *Granulator {
	if cfg.BufferSeconds <= 0 {
		cfg.BufferSeconds = DefaultConfig().BufferSeconds
	}

	g := &Granulator{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		grains: make([]grain, 0, maxGrains),
	}

	n := int(cfg.BufferSeconds * g.Config().SampleRate)
	g.buf[0] = make([]float32, n)
	g.buf[1] = make([]float32, n)

	sr := g.Config().SampleRate
	g.mix = mix.NewSmoother(math.Max(0, math.Min(cfg.Mix, 1)), mix.SmoothMs, sr)
	g.fb = mix.NewSmoother(math.Max(0, math.Min(cfg.Feedback, 0.98)), mix.SmoothMs, sr)
	g.norm = mix.NewSmoother(g.grainNorm(), mix.SmoothMs, sr)
	g.SetSelf(g)

	return g
}

// Settings returns the settings
func (g *Granulator) Settings() Config { return g.cfg }

// SetFreeze freezes or unfreezes the buffer
func (g *Granulator) SetFreeze(freeze bool) {
	g.cfg.Freeze = freeze
}

func (g *Granulator) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		g.ReceiveMessage(msg)
		return
	}

	f := psalm.FltVal[float64](value)
	switch index {
	case 1:
		g.SetFreeze(f > 0.5)
	case 2:
		g.cfg.Delay = f
	case 3:
		g.cfg.Density = f
	case 4:
		g.cfg.GrainSize = f
	case 5:
		g.cfg.Pitch = f
	}
}

func (g *Granulator) ReceiveMessage(msg psalm.Message) {
	cfg := &g.cfg
	msg.IfFltVal("freeze", func(f float64) { g.SetFreeze(f > 0.5) })
	msg.IfFltVal("delay", func(f float64) { cfg.Delay = f })
	msg.IfFltVal("spray", func(f float64) { cfg.Spray = f })
	msg.IfFltVal("density", func(f float64) { cfg.Density = f })
	msg.IfFltVal("grainSize", func(f float64) { cfg.GrainSize = f })
	msg.IfFltVal("pitch", func(f float64) { cfg.Pitch = f })
	msg.IfFltVal("pitchSpread", func(f float64) { cfg.PitchSpread = f })
	msg.IfFltVal("octave", func(f float64) { cfg.Octave = f })
	msg.IfFltVal("reverse", func(f float64) { cfg.Reverse = f })
	msg.IfFltVal("shape", func(f float64) { cfg.Shape = f })
	msg.IfFltVal("jitter", func(f float64) { cfg.Jitter = f })
	msg.IfFltVal("stereoSpread", func(f float64) { cfg.StereoSpread = f })
	msg.IfFltVal("feedback", func(f float64) { cfg.Feedback = f })
	msg.IfFltVal("mix", func(f float64) { cfg.Mix = f })
}

// window returns the grain envelope at t (0-1), from a smooth sin^2 window to a percussive shape
func window(t float64, shape float64) float64 {
	smooth := fastsine.Cycle(0.5 * t)
	smooth *= smooth
	if shape <= 0 {
		return smooth
	}
	const attack = 0.03
	var perc float64
	if t < attack {
		perc = t / attack
	} else {
		perc = math.Exp(-6.0*(t-attack)/(1.0-attack)) * (1.0 - t) / (1.0 - attack)
	}
	return smooth + (perc-smooth)*shape
}

// spawn starts a grain. Its start is chosen so the whole grain stays in valid audio: it never
// catches up with the write head (a grain pitched up reads faster than the input is written) and
// never reaches audio that has already been overwritten (or the seam of a frozen buffer)
func (g *Granulator) spawn(sr float64) {
	if len(g.grains) >= maxGrains {
		return
	}

	cfg := &g.cfg
	n := float64(len(g.buf[0]))
	length := max(1, int(cfg.GrainSize*0.001*sr))

	semis := cfg.Pitch
	if cfg.PitchSpread != 0 {
		semis += cfg.PitchSpread * (g.Rand().Float64()*2.0 - 1.0)
	}
	if cfg.Octave > 0 && g.Rand().Float64() < cfg.Octave {
		semis += 12
	}
	inc := math.Exp2(semis / 12.0)
	if g.Rand().Float64() < cfg.Reverse {
		inc = -inc
	}

	// valid start delays (behind the write head) for this grain
	l := float64(length)
	var minDelay, maxDelay float64
	if cfg.Freeze {
		// the write head stands still: stay between the oldest and newest frozen sample
		minDelay = margin + math.Max(0, l*inc)
		maxDelay = n - margin + math.Min(0, l*inc)
	} else {
		// the write head moves one sample per sample
		minDelay = margin + math.Max(0, l*(inc-1))
		maxDelay = n - margin + math.Min(0, l*(inc-1))
	}
	// only audio that has been recorded is valid
	maxDelay = math.Min(maxDelay, float64(g.written)-margin+math.Min(0, l*inc))
	if minDelay > maxDelay {
		return
	}

	delay := (cfg.Delay + cfg.Spray*(g.Rand().Float64()*2.0-1.0)) * 0.001 * sr
	delay = math.Max(minDelay, math.Min(delay, maxDelay))

	pan := 0.5 + cfg.StereoSpread*(g.Rand().Float64()-0.5)

	g.grains = append(g.grains, grain{
		pos:    float64(g.written) - delay,
		inc:    inc,
		length: length,
		gainL:  math.Cos(pan * math.Pi / 2.0),
		gainR:  math.Sin(pan * math.Pi / 2.0),
	})
}

// grainNorm returns the grain level: overlapping grains add up, scaled by the square root of the
// overlap the level stays steady
func (g *Granulator) grainNorm() float64 {
	return 1.0 / math.Sqrt(math.Max(1.0, math.Max(0.1, g.cfg.Density)*g.cfg.GrainSize*0.001))
}

// read returns the buffer at an absolute position with linear interpolation
func (g *Granulator) read(ch int, pos float64) float64 {
	buf := g.buf[ch]
	n := int64(len(buf))
	i := int64(math.Floor(pos))
	frac := pos - float64(i)
	a := buf[((i%n)+n)%n]
	b := buf[(((i+1)%n)+n)%n]
	return float64(a) + float64(b-a)*frac
}

func (g *Granulator) Synthesize() {
	var (
		cfg     = &g.cfg
		sr      = g.Config().SampleRate
		ins     = g.Input(0).Channels
		outs    = g.Output(0).Channels
		n       = int64(len(g.buf[0]))
		density = math.Max(0.1, cfg.Density)
		mixTo   = math.Max(0, math.Min(cfg.Mix, 1))
		fbTo    = math.Max(0, math.Min(cfg.Feedback, 0.98))
		normTo  = g.grainNorm()
	)
	interval := sr / density

	for i := range g.Config().BufferSize {
		inL, inR := float64(ins[0][i]), float64(ins[1][i])
		dry, wet := mix.EqualPower(g.mix.Next(mixTo))
		fb := g.fb.Next(fbTo)
		norm := g.norm.Next(normTo)

		// record the input (plus feedback), unless frozen
		if !cfg.Freeze {
			w := g.written % n
			g.buf[0][w] = float32(inL + math.Tanh(g.fbL*fb))
			g.buf[1][w] = float32(inR + math.Tanh(g.fbR*fb))
			g.written++
		}

		g.countdown--
		if g.countdown <= 0 {
			g.spawn(sr)
			g.countdown += interval * (1.0 + cfg.Jitter*(g.Rand().Float64()*2.0-1.0))
			g.countdown = math.Max(1, g.countdown)
		}

		var left, right float64
		for k := 0; k < len(g.grains); {
			gr := &g.grains[k]
			w := window(float64(gr.age)/float64(gr.length), cfg.Shape)
			x := (g.read(0, gr.pos) + g.read(1, gr.pos)) * 0.5 * w
			left += x * gr.gainL
			right += x * gr.gainR

			gr.pos += gr.inc
			gr.age++
			if gr.age >= gr.length {
				g.grains[k] = g.grains[len(g.grains)-1]
				g.grains = g.grains[:len(g.grains)-1]
			} else {
				k++
			}
		}

		left *= norm
		right *= norm
		g.fbL, g.fbR = left, right

		outs[0][i] = float32(dry*inL + wet*left)
		outs[1][i] = float32(dry*inR + wet*right)
	}
}

// Tail lets the granulator rest while silent: grains read up to the delay and spray behind the
// input, a frozen granulator plays its buffer and never rests
func (g *Granulator) Tail() float64 {
	if g.cfg.Freeze {
		return -1
	}
	return (g.cfg.Delay+g.cfg.Spray+g.cfg.GrainSize)/1000 + 0.05
}
