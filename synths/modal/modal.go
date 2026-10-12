// Package modal plays struck things: bars, tubes, bells, tines. Each rings in its modes, a set of
// partials with their own pitch ratios and decay times, the way a marimba bar or a church bell
// does. A mallet hits it: the softer the mallet, the longer it touches, the fewer high modes it
// wakes; where it hits picks the modes too.
//
//	mallets := modal.NewPoly(modal.Marimba(), 8).AddTo(p)
package modal

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/modules/poly"
)

// Mode is a partial: its pitch over the note, its loudness and how long it rings relative to the
// note's Decay (0 is treated as 1)
type Mode struct {
	Ratio float64
	Amp   float64
	Decay float64
}

// Config is the instrument and the mallet
type Config struct {
	NumChannels  int
	Modes        []Mode
	Decay        float64 // seconds the lowest mode rings (60 dB down)
	HighDamping  float64 // the higher modes die sooner: their decay times (f0/f)^HighDamping
	Hardness     float64 // 0-1: the mallet, from soft yarn to hard plastic; the velocity adds to it
	Position     float64 // 0-1: where it is struck, it mutes the modes with a node there (0 is 0.25)
	Release      float64 // seconds it takes to die once let go (a damper), 0 lets it ring
	Tremolo      float64 // Hz of a vibraphone's motor, 0 is none
	TremoloDepth float64 // 0-1
	Beat         float64 // each mode splits in two this fraction apart, beating (bells)
	Noise        float64 // 0-1: the knock of the mallet
	Level        float64 // output level (0 is treated as 1)
}

// DefaultConfig is a marimba
func DefaultConfig() Config {
	return Marimba()
}

// maxModes is the most resonators a voice rings (beating modes take two)
const maxModes = 32

// resonator is a decaying sine: a complex pole, rung by the mallet
type resonator struct {
	re, im float64
	c, s   float64 // the rotation per sample, with the decay
	gain   float64 // per unit of force
	fast   float64 // the decay per sample once let go
	r, w   float64 // the decay and the angle per sample, unbent
}

// Voice is one struck note
type Voice struct {
	*psalm.Module
	cfg       Config
	freq      float64
	velocity  float64
	amp       float64
	modes     [maxModes]resonator
	n         int
	pulse     []float64 // the force of the mallet
	pulsePos  int
	knock     float64
	knockLP   float64
	rng       uint32
	tremPhase float64
	released  bool
	noteOff   int
	waitSamps int
	active    bool
	quiet     int
	expr      expression.Expression // the pitch bend (all the modes); the mod wheel brings in a tremolo
	bentTo    float64               // the bend the modes are turned for now
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 440}
	v.rng = uint32(v.Rand().Uint64())
	v.SetSelf(v)
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	msg.IfFltVal("frequency", func(f float64) { v.freq = f })
	msg.IfFltVal("pitch", func(p float64) { v.freq = 440 * math.Exp2((p-69)/12) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("decay", func(f float64) { v.cfg.Decay = f })
	msg.IfFltVal("highDamping", func(f float64) { v.cfg.HighDamping = f })
	msg.IfFltVal("hardness", func(f float64) { v.cfg.Hardness = f })
	msg.IfFltVal("position", func(f float64) { v.cfg.Position = f })
	msg.IfFltVal("release", func(f float64) { v.cfg.Release = f })
	msg.IfFltVal("tremolo", func(f float64) { v.cfg.Tremolo = f })
	msg.IfFltVal("tremoloDepth", func(f float64) { v.cfg.TremoloDepth = f })
	msg.IfFltVal("noise", func(f float64) { v.cfg.Noise = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
}

// Receive takes messages, or the hardness on input 1 and the tremolo depth on input 2
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case 1:
		v.cfg.Hardness = psalm.FltVal[float64](value)
	case 2:
		v.cfg.TremoloDepth = psalm.FltVal[float64](value)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.strike(0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.strike(duration, amplitude, message, tick)
}

// strike sets the modes for the note and the mallet's force
func (v *Voice) strike(duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 0.8
	v.ReceiveMessage(message)
	cfg := &v.cfg
	sr := v.Config().SampleRate
	v.waitSamps = tick.SampleOffset
	v.amp = amplitude
	v.released = false
	v.noteOff = 0
	if duration > 0 {
		v.noteOff = max(1, int(duration*sr))
	}
	pos := cfg.Position
	if pos <= 0 {
		pos = 0.25
	}
	decay := math.Max(0.005, cfg.Decay)
	release := math.Max(0.005, cfg.Release)
	v.n = 0
	add := func(f, amp, t60 float64) {
		if v.n >= maxModes || f >= sr*0.48 || amp == 0 {
			return
		}
		r := math.Pow(10, -3/(t60*sr))
		w := 2 * math.Pi * f / sr
		m := &v.modes[v.n]
		if !v.active {
			m.re, m.im = 0, 0
		}
		m.r, m.w = r, w
		m.c, m.s = r*math.Cos(w), r*math.Sin(w)
		m.gain = amp
		m.fast = math.Pow(10, -3/(math.Min(t60, release)*sr)) / r
		v.n++
	}
	for k, mode := range cfg.Modes {
		f := v.freq * mode.Ratio
		rel := mode.Decay
		if rel <= 0 {
			rel = 1
		}
		t60 := decay * rel * math.Pow(cfg.Modes[0].Ratio*v.freq/f, cfg.HighDamping)
		// where it is struck: the modes with a node there sound less
		amp := mode.Amp * (0.2 + 0.8*math.Abs(math.Sin(math.Pi*pos*float64(k+1))))
		if cfg.Beat > 0 {
			add(f*(1-cfg.Beat/2), amp/2, t60)
			add(f*(1+cfg.Beat/2), amp/2, t60)
		} else {
			add(f, amp, t60)
		}
	}
	// the mallet: a half sine force, as long as it touches (3 ms soft, 0.2 ms hard); its area is 1
	hard := math.Max(0, math.Min(cfg.Hardness+0.5*(v.velocity-0.5), 1))
	touch := 0.003 * math.Pow(0.2/3, hard)
	n := max(1, int(touch*sr))
	if cap(v.pulse) < n {
		v.pulse = make([]float64, n)
	}
	v.pulse = v.pulse[:n]
	for i := range v.pulse {
		v.pulse[i] = math.Sin(math.Pi*(float64(i)+0.5)/float64(n)) * math.Pi / (2 * float64(n))
	}
	v.pulsePos = 0
	v.knock = cfg.Noise * (0.3 + 0.7*v.velocity)
	v.bentTo = 0 // the modes are set unbent: bend them in the next buffer
	v.tremPhase = 0
	v.active = true
	v.quiet = 0
}

func (v *Voice) NoteOff() {
	v.released = true
	v.noteOff = 0
}

func (v *Voice) Clear() {
	v.active = false
	for i := range v.modes {
		v.modes[i].re, v.modes[i].im = 0, 0
	}
	v.knock = 0
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return v.active
}

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	if !v.active {
		return
	}
	cfg := &v.cfg
	n := v.Config().BufferSize
	sr := v.Config().SampleRate
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	gain := level * v.amp * (0.3 + 0.7*v.velocity)
	damp := v.released && cfg.Release > 0
	knockCoef := math.Exp(-1 / (0.002 * sr))
	// the bend turns every mode (once a buffer, the modes ring on smoothly)
	v.expr.Settle()
	if bend := v.expr.Ratio(); bend != v.bentTo || v.bentTo == 0 {
		for k := range v.n {
			m := &v.modes[k]
			w := math.Min(m.w*bend, math.Pi*0.96)
			m.c, m.s = m.r*math.Cos(w), m.r*math.Sin(w)
		}
		v.bentTo = bend
	}
	// the tremolo: the motor's, or the wheel's
	tremolo := cfg.Tremolo
	depth := math.Max(0, math.Min(cfg.TremoloDepth, 1))
	if wheel := 0.6 * v.expr.Wheel(); wheel > depth {
		depth = wheel
		if tremolo <= 0 {
			tremolo = 5.5
		}
	}
	tremInc := tremolo / sr
	loud := 0.0
	channels := out.Channels
	for i := v.waitSamps; i < n; i++ {
		force := 0.0
		if v.pulsePos < len(v.pulse) {
			force = v.pulse[v.pulsePos]
			v.pulsePos++
		}
		s := 0.0
		for k := range v.n {
			m := &v.modes[k]
			re := m.c*m.re - m.s*m.im + force*m.gain
			im := m.s*m.re + m.c*m.im
			if damp {
				re *= m.fast
				im *= m.fast
			}
			m.re, m.im = re, im
			s += im
		}
		if v.knock > 1e-5 {
			v.rng = v.rng*1664525 + 1013904223
			v.knockLP += (float64(int32(v.rng))/2147483648 - v.knockLP) * 0.3
			s += v.knockLP * v.knock * 0.3
			v.knock *= knockCoef
		}
		if tremolo > 0 && depth > 0 {
			s *= 1 - depth*0.5*(1-math.Cos(2*math.Pi*v.tremPhase))
			v.tremPhase += tremInc
			if v.tremPhase >= 1 {
				v.tremPhase--
			}
		}
		loud = math.Max(loud, math.Abs(s))
		val := float32(s * gain)
		for _, ch := range channels {
			ch[i] = val
		}
	}
	if v.noteOff > 0 {
		if left := n - v.waitSamps; v.noteOff > left {
			v.noteOff -= left
		} else {
			v.NoteOff()
		}
	}
	v.waitSamps = 0
	if loud < 1e-5 && v.pulsePos >= len(v.pulse) && v.knock <= 1e-5 {
		v.quiet += n
		if v.quiet > int(0.02*sr) {
			v.active = false
		}
	} else {
		v.quiet = 0
	}
}

// NewPoly creates a set of voices
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	for i := range numVoices {
		voices[i] = New(cfg)
	}
	return poly.New[*Voice](cfg.NumChannels, voices)
}
