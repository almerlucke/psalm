// Package modal plays struck things: bars, tubes, bells, tines. Each rings in its modes, a set of
// partials with their own pitch ratios and decay times, the way a marimba bar or a church bell
// does. A mallet hits it: the softer the mallet, the longer it touches, the fewer high modes it
// wakes; where it hits picks the modes too.
//
//	mallets := modal.NewPoly(modal.Marimba(), 8).AddTo(p)
package modal

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

// Mode is a partial: its pitch over the note, its loudness and how long it rings relative to the
// note's Decay (0 is treated as 1)
type Mode struct {
	Ratio float64
	Amp   float64
	Decay float64
}

// Excitation is what sets the modes ringing
type Excitation int

const (
	Strike Excitation = iota // a mallet hits once (the default)
	Bow                      // a bow drives them as long as the note is held, fading in
	Burst                    // a burst of noise, shorter the harder the mallet: a brush, a scrape
)

// Excitations are the names of the excitations, by number
var Excitations = []string{"strike", "bow", "noise burst"}

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

	Excitation    Excitation // what rings the modes (Strike is 0)
	Inharmonicity float64    // 0-1: stretches the modes' ratios, the higher ones more (stiff metal, glass)
	LFORate       float64    // Hz of the matrix's LFO 1 (0 is 5)
	LFO2Rate      float64    // Hz of the matrix's LFO 2, a plain sine from the strike (0 is 1), running only when a slot uses it
	// Filter is a filter after the modes, its envelope starting at the strike (off unless its Type is set)
	Filter voicefilter.Config
	Mods   [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext    [modmatrix.Exts]float64         // the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)
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
	filter    *voicefilter.Filter
	mods      *modmatrix.Matrix
	random    float64
	ampEnv    float64 // the loudness of the note, falling with the decay (a source of the matrix)
	lfoPhase  float64
	lfo2Phase float64
	bowEnv    float64 // how hard the bow presses
	bowLP     float64
	bowSigma  float64
	pitchMod  float64 // the semitones the matrix bends the modes by
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 440}
	v.rng = uint32(v.Rand().Uint64())
	v.filter = voicefilter.New(cfg.Filter, v.Config().SampleRate, 1)
	v.mods = modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		v.mods.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		v.mods.SetExt(i, e)
	}
	v.SetSelf(v)
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	v.filter.Receive(msg)
	v.cfg.Filter = v.filter.Config()
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
	msg.IfIntVal("excitation", func(i int) { v.cfg.Excitation = Excitation(max(0, min(i, int(Burst)))) })
	msg.IfFltVal("inharmonicity", func(f float64) { v.cfg.Inharmonicity = f })
	msg.IfFltVal("lfoRate", func(f float64) { v.cfg.LFORate = f })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = f })
	envelope.Seconds(msg, "amp", nil, &v.cfg.Decay, &v.cfg.Release)
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

// Receive takes messages, or the hardness on input 1 and the tremolo depth on input 2, and on 3 to 6
// (InExt1...) values 0-1 that the matrix takes as sources Ext 1 to 4
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
	case InExt1, InExt1 + 1, InExt1 + 2, InExt1 + 3:
		v.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
		v.cfg.Ext[index-InExt1] = v.mods.Ext(index - InExt1)
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
	if !v.active {
		v.filter.Clear()
	}
	v.filter.Trigger(0, adsr.NoteOff)
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}
	v.ampEnv, v.lfoPhase, v.lfo2Phase, v.pitchMod = 1, 0, 0, 0
	v.prime()
	mods := v.mods
	pos := cfg.Position
	if pos <= 0 {
		pos = 0.25
	}
	pos = math.Max(0.01, math.Min(pos+mods.Mod(dPosition), 1))
	beat := math.Max(0, cfg.Beat+mods.Mod(dBeat))
	highDamping := math.Max(0, cfg.HighDamping+mods.Mod(dHighDamping))
	inharm := math.Max(0, math.Min(cfg.Inharmonicity+mods.Mod(dInharmonicity), 1))
	decay := math.Max(0.005, cfg.Decay*math.Exp2(mods.Mod(dDecay)))
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
		if inharm > 0 && cfg.Modes[0].Ratio > 0 {
			// stiffness stretches the partials: relative to the first mode, the higher the sharper
			x := mode.Ratio / cfg.Modes[0].Ratio
			f = v.freq * cfg.Modes[0].Ratio * x * math.Sqrt(math.Max(0.1, 1+0.02*inharm*(x*x-1)))
		}
		rel := mode.Decay
		if rel <= 0 {
			rel = 1
		}
		t60 := decay * rel * math.Pow(cfg.Modes[0].Ratio*v.freq/f, highDamping)
		// where it is struck: the modes with a node there sound less
		amp := mode.Amp * (0.2 + 0.8*math.Abs(math.Sin(math.Pi*pos*float64(k+1))))
		if beat > 0 {
			add(f*(1-beat/2), amp/2, t60)
			add(f*(1+beat/2), amp/2, t60)
		} else {
			add(f, amp, t60)
		}
	}
	// the mallet: a half sine force, as long as it touches (3 ms soft, 0.2 ms hard); its area is 1
	hard := math.Max(0, math.Min(cfg.Hardness+mods.Mod(dHardness)+0.5*(v.velocity-0.5), 1))
	touch := 0.003 * math.Pow(0.2/3, hard)
	n := max(1, int(touch*sr))
	if cfg.Excitation == Burst {
		n = max(1, int((0.01+0.04*(1-hard))*sr))
	}
	if cap(v.pulse) < n {
		v.pulse = make([]float64, n)
	}
	v.pulse = v.pulse[:n]
	switch cfg.Excitation {
	case Burst: // a noise burst under a half sine, with the energy of a strike
		for i := range v.pulse {
			v.rng = v.rng*1664525 + 1013904223
			v.pulse[i] = float64(int32(v.rng)) / 2147483648 * math.Sin(math.Pi*(float64(i)+0.5)/float64(n)) * 1.6 / math.Sqrt(float64(n))
		}
	case Bow: // no blow: the bow drives the modes in Synthesize
		v.pulse = v.pulse[:0]
		v.bowEnv, v.bowLP = 0, 0
		r0 := 0.999
		if v.n > 0 {
			r0 = v.modes[0].r
		}
		v.bowSigma = 0.5 * math.Sqrt(2*(1-r0))
	default:
		for i := range v.pulse {
			v.pulse[i] = math.Sin(math.Pi*(float64(i)+0.5)/float64(n)) * math.Pi / (2 * float64(n))
		}
	}
	v.pulsePos = 0
	v.knock = math.Max(0, cfg.Noise+mods.Mod(dNoise)) * (0.3 + 0.7*v.velocity)
	if cfg.Excitation == Bow {
		v.knock = 0
	}
	v.bentTo = 0 // the modes are set unbent: bend them in the next buffer
	v.tremPhase = 0
	v.active = true
	v.quiet = 0
}

func (v *Voice) NoteOff() {
	v.released = true
	v.noteOff = 0
	v.filter.Release()
}

// prime sets the sources of the matrix that are known at a strike and works out what it moves
// the settings of the note by (the cabled values need a moment to glide to where they are)
func (v *Voice) prime() {
	mods := v.mods
	if !mods.Active() {
		return
	}
	sr := v.Config().SampleRate
	mv := &mods.Values
	mv[modmatrix.Velocity], mv[modmatrix.Random] = v.velocity, v.random
	mv[modmatrix.Key] = v.keyValue()
	mv[modmatrix.Wheel] = v.expr.Wheel()
	mv[modmatrix.AmpEnv], mv[modmatrix.FilterEnv], mv[modmatrix.LFO1] = 1, 0, 0
	for range int(0.02*sr) + 1 {
		mods.Eval(sr)
	}
}

// turn tunes every mode to the bend
func (v *Voice) turn(bend float64) {
	for k := range v.n {
		m := &v.modes[k]
		w := math.Min(m.w*bend, math.Pi*0.96)
		m.c, m.s = m.r*math.Cos(w), m.r*math.Sin(w)
	}
	v.bentTo = bend
}

// keyValue is the note for the matrix: -1 to 1, five octaves below middle C to five above
func (v *Voice) keyValue() float64 {
	return math.Max(-1, math.Min(math.Log2(v.freq/middleC)/5, 1))
}

const middleC = 261.6255653005986

func (v *Voice) Clear() {
	v.active = false
	for i := range v.modes {
		v.modes[i].re, v.modes[i].im = 0, 0
	}
	v.knock = 0
	v.bowEnv = 0
	v.filter.Clear()
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
	mods := v.mods
	modded := mods.Active()
	gain := v.amp * (0.3 + 0.7*v.velocity)
	filtered := v.filter.On()
	envUsed := modded && mods.Uses(modmatrix.FilterEnv)
	lfoUsed := modded && mods.Uses(modmatrix.LFO1)
	lfo2Used := modded && mods.Uses(modmatrix.LFO2)
	lfo2Inc := 1 / sr
	if cfg.LFO2Rate > 0 {
		lfo2Inc = cfg.LFO2Rate / sr
	}
	pitchOn := modded && mods.Targets(dPitch)
	bow := cfg.Excitation == Bow
	lfoInc := 5 / sr
	if cfg.LFORate > 0 {
		lfoInc = cfg.LFORate / sr
	}
	decayCoef := math.Pow(10, -3/(math.Max(0.005, cfg.Decay)*sr))
	damp := v.released && cfg.Release > 0
	knockCoef := math.Exp(-1 / (0.002 * sr))
	// the bend turns every mode (the modes ring on smoothly; the matrix's pitch every 8 samples)
	v.expr.Settle()
	if modded {
		v.pitchMod = mods.Mod(dPitch)
	}
	if bend := v.expr.Ratio() * math.Exp2(v.pitchMod/12); bend != v.bentTo || v.bentTo == 0 {
		v.turn(bend)
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
	if tremolo <= 0 && modded && mods.Targets(dTremoloDepth) {
		tremolo = 5.5
	}
	tremInc := tremolo / sr
	bowCoef := 0.05 + 0.5*cfg.Hardness
	bowAttack := 1 - math.Exp(-1/(0.15*sr))
	bowLetGo := math.Exp(-1 / (0.03 * sr))
	filterFreq := v.freq
	loud := 0.0
	channels := out.Channels
	for i := v.waitSamps; i < n; i++ {
		force := 0.0
		if v.pulsePos < len(v.pulse) {
			force = v.pulse[v.pulsePos]
			v.pulsePos++
		}
		if bow && (!v.released || v.bowEnv > 1e-4) {
			// the bow: noise pressed against the modes, fading in, lifted when let go
			v.rng = v.rng*1664525 + 1013904223
			v.bowLP += (float64(int32(v.rng))/2147483648 - v.bowLP) * bowCoef
			if v.released {
				v.bowEnv *= bowLetGo
			} else {
				v.bowEnv += (1 - v.bowEnv) * bowAttack
			}
			force += v.bowLP * v.bowEnv * v.bowSigma * 2.5
		}
		var dLev, dTrem float64
		if modded {
			v.ampEnv *= decayCoef
			mv := &mods.Values
			mv[modmatrix.AmpEnv], mv[modmatrix.FilterEnv] = v.ampEnv, v.filter.Env()
			mv[modmatrix.Wheel] = v.expr.Wheel()
			if lfoUsed {
				mv[modmatrix.LFO1] = math.Sin(2 * math.Pi * v.lfoPhase)
				if v.lfoPhase += lfoInc; v.lfoPhase >= 1 {
					v.lfoPhase--
				}
			}
			if lfo2Used {
				mv[modmatrix.LFO2] = math.Sin(2 * math.Pi * v.lfo2Phase)
				if v.lfo2Phase += lfo2Inc; v.lfo2Phase >= 1 {
					v.lfo2Phase--
				}
			}
			mods.Eval(sr)
			dLev, dTrem = mods.Mod(dLevel), mods.Mod(dTremoloDepth)
			// the pitch follows the matrix every 8 samples, so a fast LFO glides instead of stepping
			if pitchOn && i&7 == 0 {
				v.pitchMod = mods.Mod(dPitch)
				if bend := v.expr.Ratio() * math.Exp2(v.pitchMod/12); bend != v.bentTo {
					v.turn(bend)
				}
			}
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
		if d := math.Max(0, math.Min(depth+dTrem, 1)); tremolo > 0 && d > 0 {
			s *= 1 - d*0.5*(1-math.Cos(2*math.Pi*v.tremPhase))
			v.tremPhase += tremInc
			if v.tremPhase >= 1 {
				v.tremPhase--
			}
		}
		if filtered || envUsed {
			var cut, res float64
			if modded {
				cut, res = mods.Mod(dCutoff), mods.Mod(dResonance)
			}
			v.filter.Tick(filterFreq, cut, res)
		}
		if filtered {
			s = v.filter.Process(0, s)
		}
		loud = math.Max(loud, math.Abs(s))
		g := gain * level
		if modded {
			g = gain * math.Max(0, level+dLev)
		}
		val := float32(s * g)
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
	if loud < 1e-5 && v.pulsePos >= len(v.pulse) && v.knock <= 1e-5 && !(bow && !v.released) {
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
