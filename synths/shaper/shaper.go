// Package shaper is a synth that builds its wave from a chain of shapers (genny's float/shape, see
// blocks/shapechain): a phase from 0 to 1 goes in, the chain bends it into a wave. Up to four
// chains (A to D) are written as text and the synth morphs from one to the next as its morph goes
// from 0 to 1, so the wave itself moves: an envelope, an LFO, the velocity or the mod wheel can
// pull the morph and four macros (m1 to m4 in the chains) that set the numbers inside them. Two
// lines (octave, detune, level, a morph of their own) go through the filter every synth has
// (blocks/voicefilter) and an amp envelope.
//
//	cfg := shaper.Glass()
//	cfg.Chains[1] = "bipolar | mirror -0.4 0.4 | mult 2.5 | sin"  // chain B
//	v := shaper.NewPoly(cfg, 8).AddTo(p)
//
// The shaping is nonlinear, so it makes harmonics far above the pitch: the synth reads the chain
// several times a sample (Oversample) and filters them away before they fold back as aliasing.
package shaper

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

// NumChains is how many chains a sound morphs through
const NumChains = 4

// ChainKeys are the messages that set a chain (a string, or a *shapechain.Chain read already), A to D
var ChainKeys = [NumChains]string{"chainA", "chainB", "chainC", "chainD"}

// Oversamples are the factors the chains are read at, by the number of the message "oversample": 0 is
// automatic, then 1 (the chain read once a sample), 2, 4 and 8 times
var Oversamples = [...]int{0, 1, 2, 4, 8}

// DefaultChain is what a chain A that is empty or can't be read is
const DefaultChain = "sin"

// Line is an oscillator: the same chains read at its pitch and its own place in the morph
type Line struct {
	Octave int
	Detune float64 // cents
	Level  float64 // line 1: 0 is 1; line 2: 0 is silent
	Morph  float64 // -1 to 1: added to the morph of the voice
}

// Config is a sound
type Config struct {
	NumChannels int
	// Chains are the chains written as text, A first. The ones from the first empty one on are not
	// used; a morph goes through the ones that are
	Chains [NumChains]string
	Morph  float64                       // 0-1: 0 is chain A, 1 the last chain, in between a blend of two neighbours
	Macros [shapechain.NumMacros]float64 // 0-1: m1 to m4 in the chains
	Line1  Line
	Line2  Line

	Oversample int     // how many times a sample the chains are read: 1 (off), 2, 4 or 8; 0 is automatic: as many as the chains and the pitch ask for (see pickFactor), up to 4
	Width      float64 // 0-1: line 1 to the left, line 2 to the right
	Level      float64 // output level (0 is treated as 1)

	AmpEnv adsr.Setting
	// MorphEnv moves the morph over the note by MorphAmount (-1 to 1) at its peak; ShapeEnv moves
	// macro m1 by ShapeAmount. An amount of 0 leaves them off
	MorphEnv    adsr.Setting
	MorphAmount float64
	ShapeEnv    adsr.Setting
	ShapeAmount float64

	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	VibratoDelay float64 // seconds
	// the LFOs of the matrix, a sine from the start of every note (LFO1: Hz, or beats a cycle with
	// LFO1Sync, following the tempo), and a second one a sine (LFO2Shape 0) or a triangle (1)
	LFO1Rate  float64
	LFO1Sync  float64
	LFO2Rate  float64
	LFO2Shape int

	Filter voicefilter.Config // after the lines (off unless its Type is set)

	Mods [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext  [modmatrix.Exts]float64         // the values 0-1 of the sources Ext 1 to 4 (cabled in on InExt1...)
}

// MaxAuto is the most an automatic oversample goes to
const MaxAuto = 4

// oversampleFactor is the factor a Config's setting stands for: 0 stays 0 (automatic), others the
// nearest of 1, 2, 4, 8 below
func oversampleFactor(n int) int {
	switch {
	case n <= 0:
		return 0
	case n >= 8:
		return 8
	case n >= 4:
		return 4
	case n >= 2:
		return 2
	}
	return 1
}

// The destinations of the matrix (see Dests), by number
const (
	dMorph = iota
	dMacro1
	dMacro2
	dMacro3
	dMacro4
	dPitch
	dVibrato
	dWidth
	dLevel
	dDetune
	dCutoff
	dResonance
	numDests
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (semitones, octaves, cents, or the setting itself). The cutoff and
// the resonance are those of the filter, and do nothing while it is off
var Dests = []modmatrix.Dest{
	dMorph:     {Key: "morph", Label: "morph", Span: 1},
	dMacro1:    {Key: "macro1", Label: "macro 1", Span: 1},
	dMacro2:    {Key: "macro2", Label: "macro 2", Span: 1},
	dMacro3:    {Key: "macro3", Label: "macro 3", Span: 1},
	dMacro4:    {Key: "macro4", Label: "macro 4", Span: 1},
	dPitch:     {Key: "pitch", Label: "pitch", Span: 12},
	dVibrato:   {Key: "vibratoDepth", Label: "vibrato", Span: 1},
	dWidth:     {Key: "width", Label: "width", Span: 1},
	dLevel:     {Key: "level", Label: "level", Span: 1},
	dDetune:    {Key: "detune2", Label: "detune 2", Span: 100},
	dCutoff:    {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance: {Key: "resonance", Label: "resonance", Span: 1},
}

// The value inputs of a voice: 1 sets the morph, 5 to 8 the values of the matrix (Ext 1 to 4)
const (
	InMorph = 1
	InExt1  = 5 // to InExt1+3
)

// DefaultConfig is a glassy sound
func DefaultConfig() Config { return Glass() }

// Voice plays a note
type Voice struct {
	*psalm.Module
	cfg       Config
	chains    [NumChains]*shapechain.Chain
	numChains int
	freq      float64
	velocity  float64
	amp       float64
	phase     [2]float64
	os        [2][len(Oversamples)]*oversample.Oversampler // by factor, made when one is first needed
	osFactor  int                                          // how many times it reads the chain now
	amped     *adsr.ADSR
	morphEnv  *adsr.ADSR
	shapeEnv  *adsr.ADSR
	t         int
	waitSamps int
	dc, dcIn  [2]float64
	expr      expression.Expression
	filter    *voicefilter.Filter
	mods      *modmatrix.Matrix
	random    float64
	lfo1Ph    float64
	lfo2Ph    float64
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	if cfg.AmpEnv == (adsr.Setting{}) {
		cfg.AmpEnv = *adsr.NewSetting(1, 5, 0.8, 200, 1, 300)
	}
	if cfg.MorphEnv == (adsr.Setting{}) {
		cfg.MorphEnv = *adsr.NewSetting(1, 5, 0.5, 400, 1, 300)
	}
	if cfg.ShapeEnv == (adsr.Setting{}) {
		cfg.ShapeEnv = *adsr.NewSetting(1, 5, 0.5, 400, 1, 300)
	}
	cfg.Oversample = oversampleFactor(cfg.Oversample)
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 220}
	sr := v.Config().SampleRate
	v.amped = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)
	v.morphEnv = adsr.New(&v.cfg.MorphEnv, adsr.NoteOff, sr)
	v.shapeEnv = adsr.New(&v.cfg.ShapeEnv, adsr.NoteOff, sr)
	v.filter = voicefilter.New(cfg.Filter, sr, cfg.NumChannels)
	v.mods = modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		v.mods.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		v.mods.SetExt(i, e)
	}
	for i, text := range cfg.Chains {
		v.setChain(i, text)
	}
	v.pickFactor()
	v.SetSelf(v)
	return v
}

// setChain sets a chain from text or a chain read already. Text that can't be read leaves the chain as it was
// (chain A that has none is a sine)
func (v *Voice) setChain(i int, chain any) {
	var c *shapechain.Chain
	switch x := chain.(type) {
	case *shapechain.Chain:
		c = x
	case string:
		if x == "" {
			c = nil
			if i == 0 {
				c = shapechain.MustParse(DefaultChain)
			}
		} else {
			var err error
			if c, err = shapechain.Parse(x); err != nil {
				if v.chains[i] != nil || i != 0 {
					return
				}
				c = shapechain.MustParse(DefaultChain)
			}
		}
	default:
		return
	}
	v.chains[i] = c
	if c != nil {
		v.cfg.Chains[i] = c.Text()
	} else {
		v.cfg.Chains[i] = ""
	}
	v.numChains = 0
	for _, ch := range v.chains {
		if ch == nil {
			break
		}
		v.numChains++
	}
}

// pickFactor works out how many times a sample the chains are read: what the config says, or for
// the automatic one the least that keeps the harmonics of the chains (Bandwidth) that lie above
// the audible band from folding back, at the pitch of the voice (and a little over it, for vibrato
// and bend): 1 while the highest harmonic is under half the sample rate, then 2, then 4
func (v *Voice) pickFactor() {
	f := v.cfg.Oversample
	if f == 0 {
		sr := v.Config().SampleRate
		top := v.freq * 1.12
		for k := range 2 {
			top = math.Max(top, v.freq*1.12*math.Exp2(float64(v.line(k).Octave)+v.line(k).Detune/1200))
		}
		f = 1
		for i := 0; i < v.numChains; i++ {
			reach := top * float64(v.chains[i].Bandwidth()) // the highest frequency with something in it
			need := 1
			switch {
			case reach <= sr/2:
			case reach <= 2*sr-20000:
				need = 2
			default:
				need = MaxAuto
			}
			f = max(f, need)
		}
	}
	if f != v.osFactor {
		v.osFactor = f
		for k := range v.os {
			for _, o := range v.os[k] {
				if o != nil {
					o.Reset()
				}
			}
		}
	}
}

// oversampler is the up and down sampler of a line at the factor now
func (v *Voice) oversampler(line int) *oversample.Oversampler {
	idx := 0
	for i, f := range Oversamples {
		if f == v.osFactor {
			idx = i
		}
	}
	if v.os[line][idx] == nil {
		v.os[line][idx] = oversample.New(v.osFactor, 16)
	}
	return v.os[line][idx]
}

func (v *Voice) line(i int) *Line {
	if i == 0 {
		return &v.cfg.Line1
	}
	return &v.cfg.Line2
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	v.filter.Receive(msg)
	v.cfg.Filter = v.filter.Config()
	envelope.Receive(msg, "amp", &v.cfg.AmpEnv)
	envelope.Receive(msg, "morphEnv", &v.cfg.MorphEnv)
	envelope.Receive(msg, "shapeEnv", &v.cfg.ShapeEnv)
	for i, key := range ChainKeys {
		if c, ok := msg[key]; ok {
			v.setChain(i, c)
		}
	}
	msg.IfFltVal("frequency", func(f float64) { v.freq = f })
	msg.IfFltVal("pitch", func(p float64) { v.freq = 440 * math.Exp2((p-69)/12) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("morph", func(f float64) { v.cfg.Morph = math.Max(0, math.Min(f, 1)) })
	for i := range v.cfg.Macros {
		msg.IfFltVal([]string{"macro1", "macro2", "macro3", "macro4"}[i], func(f float64) { v.cfg.Macros[i] = math.Max(0, math.Min(f, 1)) })
	}
	for i, n := range []string{"1", "2"} {
		l := v.line(i)
		msg.IfIntVal("octave"+n, func(o int) { l.Octave = max(-4, min(o, 4)) })
		msg.IfFltVal("detune"+n, func(f float64) { l.Detune = f })
		msg.IfFltVal("level"+n, func(f float64) { l.Level = math.Max(0, f) })
		msg.IfFltVal("morph"+n, func(f float64) { l.Morph = math.Max(-1, math.Min(f, 1)) })
	}
	msg.IfFltVal("detune", func(f float64) { v.cfg.Line2.Detune = f }) // line 2, as on the other synths with two
	msg.IfIntVal("oversample", func(o int) { v.cfg.Oversample = Oversamples[max(0, min(o, len(Oversamples)-1))] })
	msg.IfFltVal("width", func(f float64) { v.cfg.Width = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	msg.IfFltVal("morphAmount", func(f float64) { v.cfg.MorphAmount = math.Max(-1, math.Min(f, 1)) })
	msg.IfFltVal("shapeAmount", func(f float64) { v.cfg.ShapeAmount = math.Max(-1, math.Min(f, 1)) })
	msg.IfFltVal("vibratoRate", func(f float64) { v.cfg.VibratoRate = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { v.cfg.VibratoDepth = f })
	msg.IfFltVal("lfo1Rate", func(f float64) { v.cfg.LFO1Rate = f })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = f })
	msg.IfIntVal("lfo2Shape", func(n int) { v.cfg.LFO2Shape = max(0, min(n, 1)) })
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
	v.pickFactor() // the chains, or the factor, may have changed
}

// Receive takes messages, or the morph (0-1) on input 1, and on 5 to 8 (InExt1...) values 0-1 that
// the matrix takes as sources Ext 1 to 4
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case InMorph:
		v.cfg.Morph = math.Max(0, math.Min(psalm.FltVal[float64](value), 1))
	case InExt1, InExt1 + 1, InExt1 + 2, InExt1 + 3:
		v.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
		v.cfg.Ext[index-InExt1] = v.mods.Ext(index - InExt1)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.NoteOff, 0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.Duration, duration, amplitude, message, tick)
}

func (v *Voice) note(mode adsr.ReleaseMode, duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 0.8
	v.ReceiveMessage(message)
	v.pickFactor() // the pitch of the note
	v.waitSamps = tick.SampleOffset
	v.amp = amplitude
	v.t = 0
	v.phase = [2]float64{} // every note starts the wave at the same place
	v.lfo1Ph, v.lfo2Ph = 0, 0
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}
	v.filter.Trigger(duration*1000, mode)
	v.amped.TriggerFull(duration*1000, 1, &v.cfg.AmpEnv, mode)
	if v.cfg.MorphAmount != 0 {
		v.morphEnv.TriggerFull(duration*1000, 1, &v.cfg.MorphEnv, mode)
	}
	if v.cfg.ShapeAmount != 0 {
		v.shapeEnv.TriggerFull(duration*1000, 1, &v.cfg.ShapeEnv, mode)
	}
}

func (v *Voice) NoteOff() {
	v.filter.Release()
	v.amped.Release()
	v.morphEnv.Release()
	v.shapeEnv.Release()
}

func (v *Voice) Clear() {
	v.filter.Clear()
	v.amped.Clear()
	v.morphEnv.Clear()
	v.shapeEnv.Clear()
	v.phase = [2]float64{}
	for _, line := range v.os {
		for _, o := range line {
			if o != nil {
				o.Reset()
			}
		}
	}
	v.dc, v.dcIn = [2]float64{}, [2]float64{}
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool { return !v.amped.Done() }

// wave is the chains at a phase, blended at a place in the morph (0-1 over the chains in use)
func (v *Voice) wave(ph, pos float64, m *shapechain.Macros) float64 {
	n := v.numChains
	if n <= 1 {
		return v.chains[0].Eval(ph, m)
	}
	f := pos * float64(n-1)
	i := int(f)
	if i >= n-1 {
		return v.chains[n-1].Eval(ph, m)
	}
	a := v.chains[i].Eval(ph, m)
	frac := f - float64(i)
	if frac <= 0 {
		return a
	}
	return a + (v.chains[i+1].Eval(ph, m)-a)*frac
}

// silent is the level of the amp envelope (-100 dB) below which a line makes no sound
const silent = 1e-5

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(x, 1)) }

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	if !v.IsActive() {
		return
	}
	cfg := &v.cfg
	n := v.Config().BufferSize
	sr := v.Config().SampleRate
	level := orOne(cfg.Level)
	mods := v.mods
	modded := mods.Active()
	morphModded := modded && mods.Targets(dMorph)
	var macroModded [shapechain.NumMacros]bool
	for i := range macroModded {
		macroModded[i] = modded && mods.Targets(dMacro1+i)
	}
	pitchModded := modded && mods.Targets(dPitch)
	vibModded := modded && mods.Targets(dVibrato)
	widthModded := modded && mods.Targets(dWidth)
	levelModded := modded && mods.Targets(dLevel)
	detuneModded := modded && mods.Targets(dDetune)
	filtered := v.filter.On()
	filterTick := filtered || (modded && mods.Uses(modmatrix.FilterEnv))
	morphEnvOn := cfg.MorphAmount != 0
	shapeEnvOn := cfg.ShapeAmount != 0
	gain := level * v.amp
	width := math.Max(0, math.Min(cfg.Width, 1))
	pan := func(p float64) (float64, float64) {
		a := (p + 1) * math.Pi / 4
		return math.Cos(a) * math.Sqrt2, math.Sin(a) * math.Sqrt2
	}
	l1L, l1R := pan(-width)
	l2L, l2R := pan(width)
	vibDelay := int(cfg.VibratoDelay * sr)
	lfo1Inc := cfg.LFO1Rate / sr
	if cfg.LFO1Sync > 0 {
		lfo1Inc = v.BPM() / 60 / cfg.LFO1Sync / sr
	}
	lfo1On := lfo1Inc > 0 && modded && mods.Uses(modmatrix.LFO1)
	lfo2On := modded && mods.Uses(modmatrix.LFO2) && cfg.LFO2Rate > 0
	lfo2Inc := cfg.LFO2Rate / sr
	dcCoef := 1 - 2*math.Pi*10/sr
	line2On := cfg.Line2.Level > 0
	var ratio [2]float64
	for i := range ratio {
		l := v.line(i)
		ratio[i] = math.Exp2(float64(l.Octave) + l.Detune/1200)
	}
	if modded {
		mods.Values[modmatrix.Velocity] = v.amp
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(v.freq/130.81/2)/5, 1))
	}
	factor := v.osFactor
	invFactor := 1 / float64(factor)
	var lfo1, lfo2, morphEnv, shapeEnv float64
	channels := out.Channels
	stereo := len(channels) > 1
	for i := v.waitSamps; i < n; i++ {
		v.expr.Tick(sr)
		dca := v.amped.Generate()
		if morphEnvOn {
			morphEnv = v.morphEnv.Generate() * cfg.MorphAmount
		}
		if shapeEnvOn {
			shapeEnv = v.shapeEnv.Generate() * cfg.ShapeAmount
		}
		if lfo1On {
			lfo1 = shapechain.SinTurns(v.lfo1Ph)
			v.lfo1Ph += lfo1Inc
			v.lfo1Ph -= math.Floor(v.lfo1Ph)
		}
		if lfo2On {
			if cfg.LFO2Shape == 1 {
				t := v.lfo2Ph + 0.25
				t -= math.Floor(t)
				lfo2 = 1 - 4*math.Abs(t-0.5)
			} else {
				lfo2 = shapechain.SinTurns(v.lfo2Ph)
			}
			v.lfo2Ph += lfo2Inc
			v.lfo2Ph -= math.Floor(v.lfo2Ph)
		}
		if modded {
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = v.filter.Env(), dca
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = lfo1, lfo2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
		}
		morph := cfg.Morph + morphEnv
		if morphModded {
			morph += mods.Mod(dMorph)
		}
		var macros shapechain.Macros
		for k := range macros {
			macros[k] = cfg.Macros[k]
			if k == 0 {
				macros[k] += shapeEnv
			}
			if macroModded[k] {
				macros[k] += mods.Mod(dMacro1 + k)
			}
			macros[k] = clamp01(macros[k])
		}
		semis := v.expr.Semitones()
		if pitchModded {
			semis += mods.Mod(dPitch)
		}
		if vd := cfg.VibratoDepth + mods.Mod(dVibrato); vd != 0 && v.t > vibDelay && (cfg.VibratoDepth != 0 || vibModded) {
			semis += vd * shapechain.SinTurns(cfg.VibratoRate*float64(v.t-vibDelay)/sr)
		}
		f := v.freq
		if semis != 0 {
			f *= math.Exp2(semis / 12)
		}
		if detuneModded {
			ratio[1] = math.Exp2(float64(cfg.Line2.Octave) + (cfg.Line2.Detune+mods.Mod(dDetune))/1200)
		}
		var y [2]float64
		for k := 0; k < 2; k++ {
			l := v.line(k)
			if k == 1 && !line2On {
				continue
			}
			lvl := l.Level
			if k == 0 {
				lvl = orOne(lvl)
			}
			if lvl <= 0 {
				continue
			}
			inc := f * ratio[k] / sr
			if dca < silent { // the end of a long release: nothing to hear, nothing to work out
				ph := v.phase[k] + inc
				v.phase[k] = ph - math.Floor(ph)
				continue
			}
			pos := clamp01(morph + l.Morph)
			ph := v.phase[k]
			var s float64
			if factor == 1 {
				s = v.wave(ph, pos, &macros)
			} else {
				step := inc * invFactor
				s = v.oversampler(k).Down(func(j int) float64 {
					p := ph + float64(j)*step
					return v.wave(p-math.Floor(p), pos, &macros)
				})
			}
			if s != s { // NaN: a chain that divided by nothing
				s = 0
			}
			y[k] = math.Max(-2, math.Min(s, 2)) * dca * lvl
			ph += inc
			v.phase[k] = ph - math.Floor(ph)
		}
		if widthModded && v.t%4 == 0 {
			wd := math.Max(0, math.Min(cfg.Width+mods.Mod(dWidth), 1))
			l1L, l1R = pan(-wd)
			l2L, l2R = pan(wd)
		}
		var o [2]float64
		if stereo {
			o[0] = y[0]*l1L + y[1]*l2L
			o[1] = y[0]*l1R + y[1]*l2R
		} else {
			o[0] = y[0] + y[1]
		}
		g := gain
		if levelModded {
			g = math.Max(0, level+mods.Mod(dLevel)) * v.amp
		}
		if filterTick {
			v.filter.Tick(f, mods.Mod(dCutoff)+1.5*v.expr.Wheel(), mods.Mod(dResonance))
		}
		for c := range o {
			if c > 0 && !stereo {
				break
			}
			// the chains are not always centred: a DC blocker
			x := o[c]
			v.dc[c] = x - v.dcIn[c] + dcCoef*v.dc[c]
			v.dcIn[c] = x
			val := v.dc[c] * g * 0.5
			if filtered {
				val = v.filter.Process(c, val)
			}
			channels[c][i] = float32(val)
		}
		v.t++
	}
	v.waitSamps = 0
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
