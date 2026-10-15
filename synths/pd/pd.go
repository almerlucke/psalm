// Package pd is a phase distortion synth after the Casio CZ: a cosine read at a bent speed. Bent a
// little it stays a sine; bent further it turns into a saw, a square, a pulse, or a resonant sweep,
// and an envelope (the DCW) bends it over the note, the way a filter envelope opens a subtractive
// synth, but with the CZ's own glassy, digital edge. Two lines of an oscillator, its DCW and its
// amp (DCA) envelope sound together, detuned or ring modulated.
//
//	cz := pd.NewPoly(pd.Brass(), 8).AddTo(p)
package pd

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

// Waveform is the shape a cosine is bent into at full distortion
type Waveform int

const (
	Saw      Waveform = iota // the cosine's first half squeezed: a sawtooth
	Square                   // both halves squeezed: a square
	Pulse                    // the cycle squeezed into its start: a narrow pulse
	SawPulse                 // a saw, then a pulse
	Reso1                    // a resonance, a sine at the distortion's pitch, under a falling window
	Reso2                    // the resonance under a rise and fall
	Reso3                    // the resonance under a window that holds, then falls
)

// Line is an oscillator with its two envelopes
type Line struct {
	Waveform  Waveform
	Second    Waveform // the waveform of every other cycle when Alternate is set
	Alternate bool
	Octave    int
	Detune    float64 // cents
	Level     float64 // 0 is silent
	DCW       float64 // 0-1: the distortion at the top of its envelope (0 is a sine)
	DCWEnv    adsr.Setting
	KeyFollow float64 // 0-1: higher notes are distorted less (as on the CZ)
	Velocity  float64 // 0-1: how much a harder note distorts more
	DCAEnv    adsr.Setting
}

// Config is a sound of two lines
type Config struct {
	NumChannels  int
	Line1        Line
	Line2        Line
	Ring         bool    // line 2 is multiplied by line 1
	PitchSweep   float64 // semitones the pitch starts away from the note...
	PitchTime    float64 // ...and the seconds it takes to get there
	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	VibratoDelay float64 // seconds
	Width        float64 // 0-1: line 1 to the left, line 2 to the right
	// an LFO on the distortion of both lines (the CZ's wah), a sine from the start of every note:
	// Hz, or beats a cycle (SweepSync, follows the tempo); SweepDepth is added to the DCW, up and down
	SweepRate  float64
	SweepSync  float64
	SweepDepth float64
	Level      float64 // output level (0 is treated as 1)

	// the DCO envelope: the pitch (semitones, DCOAmount at the top of the envelope, added to the pitch)
	// and a distortion envelope over both lines (DepthAmount, 0-1, added to the DCW), apart from
	// the DCW envelopes of the lines. An amount of 0 leaves them off
	DCOAmount   float64
	DCOEnv      adsr.Setting
	DepthAmount float64
	DepthEnv    adsr.Setting

	// a second LFO, a plain sine (LFO2Shape 0) or triangle (1), only running when the matrix uses it
	LFO2Rate  float64 // Hz
	LFO2Shape int

	Filter voicefilter.Config // a filter after the lines (off unless its Type is set)

	Mods [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext  [modmatrix.Exts]float64         // the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)
}

// The destinations of the matrix (see Dests), by number
const (
	dDCW1 = iota
	dDCW2
	dPitch
	dVibrato
	dWidth
	dLevel
	dCutoff
	dResonance
	dDetune
	dSweepRate
	dSweepDepth
	numDests
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (semitones, octaves, cents, or the setting itself). The cutoff and
// the resonance are those of the filter after the lines, and do nothing while it is off
var Dests = []modmatrix.Dest{
	dDCW1:       {Key: "dcw1", Label: "dcw 1", Span: 0.5},
	dDCW2:       {Key: "dcw2", Label: "dcw 2", Span: 0.5},
	dPitch:      {Key: "pitch", Label: "pitch", Span: 12},
	dVibrato:    {Key: "vibratoDepth", Label: "vibrato", Span: 1},
	dWidth:      {Key: "width", Label: "width", Span: 1},
	dLevel:      {Key: "level", Label: "level", Span: 1},
	dCutoff:     {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance:  {Key: "resonance", Label: "resonance", Span: 1},
	dDetune:     {Key: "detune", Label: "detune 2", Span: 100},
	dSweepRate:  {Key: "sweepRate", Label: "sweep rate", Span: 3},
	dSweepDepth: {Key: "sweepDepth", Label: "sweep depth", Span: 0.5},
}

// The value inputs of a voice: 1 sets the distortion of both lines, 5 to 8 the values of the
// matrix (Ext 1 to 4) that a slot can move any setting by
const (
	InDCW  = 1
	InExt1 = 5 // to InExt1+3
)

// DefaultConfig is a brass sound
func DefaultConfig() Config {
	return Brass()
}

// Voice plays a note
type Voice struct {
	*psalm.Module
	cfg       Config
	freq      float64
	velocity  float64
	amp       float64
	lines     [2]lineState
	t         int
	waitSamps int
	dc        [2]float64            // the DC blockers of the channels
	expr      expression.Expression // the pitch bend; the mod wheel distorts more (the DCW of both lines)
	dcIn      [2]float64
	filter    *voicefilter.Filter
	mods      *modmatrix.Matrix
	dco       *adsr.ADSR
	depth     *adsr.ADSR
	random    float64 // -1 to 1, for the note
	sweepPh   float64
	lfo2Ph    float64
}

// defaultEnv is the envelope of the DCO and the depth when none is set
func defaultEnv() adsr.Setting { return env(5, 400, 0.5, 300) }

type lineState struct {
	phase  float64
	odd    bool // every other cycle, for Alternate
	dcw    *adsr.ADSR
	dca    *adsr.ADSR
	amount float64 // the distortion at the top of the envelope, for this note
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	if cfg.DCOEnv == (adsr.Setting{}) {
		cfg.DCOEnv = defaultEnv()
	}
	if cfg.DepthEnv == (adsr.Setting{}) {
		cfg.DepthEnv = defaultEnv()
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 220}
	sr := v.Config().SampleRate
	v.lines[0].dcw = adsr.New(&v.cfg.Line1.DCWEnv, adsr.NoteOff, sr)
	v.lines[0].dca = adsr.New(&v.cfg.Line1.DCAEnv, adsr.NoteOff, sr)
	v.lines[1].dcw = adsr.New(&v.cfg.Line2.DCWEnv, adsr.NoteOff, sr)
	v.lines[1].dca = adsr.New(&v.cfg.Line2.DCAEnv, adsr.NoteOff, sr)
	v.dco = adsr.New(&v.cfg.DCOEnv, adsr.NoteOff, sr)
	v.depth = adsr.New(&v.cfg.DepthEnv, adsr.NoteOff, sr)
	v.filter = voicefilter.New(cfg.Filter, sr, cfg.NumChannels)
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
	envelope.Receive(msg, "dco", &v.cfg.DCOEnv)
	envelope.Receive(msg, "depth", &v.cfg.DepthEnv)
	// both lines: "amp" their loudness (DCA), "dcw" their brightness
	for _, l := range []*Line{&v.cfg.Line1, &v.cfg.Line2} {
		envelope.Receive(msg, "amp", &l.DCAEnv)
		envelope.Receive(msg, "dcw", &l.DCWEnv)
	}
	msg.IfFltVal("frequency", func(f float64) { v.freq = f })
	msg.IfFltVal("pitch", func(p float64) { v.freq = 440 * math.Exp2((p-69)/12) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("dcw1", func(f float64) { v.cfg.Line1.DCW = f })
	msg.IfFltVal("dcw2", func(f float64) { v.cfg.Line2.DCW = f })
	msg.IfIntVal("waveform1", func(w int) { v.cfg.Line1.Waveform = Waveform(w) })
	msg.IfIntVal("waveform2", func(w int) { v.cfg.Line2.Waveform = Waveform(w) })
	msg.IfFltVal("detune", func(f float64) { v.cfg.Line2.Detune = f })
	msg.IfFltVal("detune1", func(f float64) { v.cfg.Line1.Detune = f })
	msg.IfFltVal("detune2", func(f float64) { v.cfg.Line2.Detune = f })
	msg.IfFltVal("width", func(f float64) { v.cfg.Width = f })
	msg.IfFltVal("vibratoRate", func(f float64) { v.cfg.VibratoRate = f })
	msg.IfFltVal("sweepRate", func(f float64) { v.cfg.SweepRate = f })
	msg.IfFltVal("sweepDepth", func(f float64) { v.cfg.SweepDepth = f })
	msg.IfFltVal("dcoAmount", func(f float64) { v.cfg.DCOAmount = f })
	msg.IfFltVal("depthAmount", func(f float64) { v.cfg.DepthAmount = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = f })
	msg.IfIntVal("lfo2Shape", func(n int) { v.cfg.LFO2Shape = max(0, min(n, 1)) })
	msg.IfFltVal("vibratoDepth", func(f float64) { v.cfg.VibratoDepth = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
}

// Receive takes messages, or the distortion of both lines (0-1) on input 1, and on 5 to 8 (InExt1...)
// values 0-1 that the matrix takes as sources Ext 1 to 4
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case InDCW:
		f := psalm.FltVal[float64](value)
		v.cfg.Line1.DCW, v.cfg.Line2.DCW = f, f
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
	v.waitSamps = tick.SampleOffset
	v.amp = amplitude
	v.t = 0
	v.sweepPh, v.lfo2Ph = 0, 0
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}
	v.filter.Trigger(duration*1000, mode)
	if v.cfg.DCOAmount != 0 {
		v.dco.TriggerFull(duration*1000, 1, &v.cfg.DCOEnv, mode)
	}
	if v.cfg.DepthAmount != 0 {
		v.depth.TriggerFull(duration*1000, 1, &v.cfg.DepthEnv, mode)
	}
	for i := range v.lines {
		l, s := v.line(i), &v.lines[i]
		// the CZ's key follow: the higher the note, the less it is distorted
		follow := 1 - math.Max(0, math.Min(l.KeyFollow, 1))*math.Max(0, math.Log2(v.freq/130.81)/5)
		s.amount = math.Max(0, math.Min((l.DCW+l.Velocity*(v.velocity-0.5))*math.Max(0, follow), 1))
		s.dcw.TriggerFull(duration*1000, 1, &l.DCWEnv, mode)
		s.dca.TriggerFull(duration*1000, 1, &l.DCAEnv, mode)
	}
}

func (v *Voice) NoteOff() {
	v.filter.Release()
	v.dco.Release()
	v.depth.Release()
	for i := range v.lines {
		v.lines[i].dcw.Release()
		v.lines[i].dca.Release()
	}
}

func (v *Voice) Clear() {
	v.filter.Clear()
	v.dco.Clear()
	v.depth.Clear()
	for i := range v.lines {
		v.lines[i].dcw.Clear()
		v.lines[i].dca.Clear()
		v.lines[i].phase, v.lines[i].odd = 0, false
	}
	v.dc, v.dcIn = [2]float64{}, [2]float64{}
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return !v.lines[0].dca.Done() || (v.cfg.Line2.Level > 0 && !v.lines[1].dca.Done())
}

// shape returns the sample of a waveform at a phase (0-1), distorted d (0-1), for a pitch that
// takes inc of the cycle a sample: the bends are never steeper than a few samples (no aliasing).
// Every waveform starts and ends its cycle at -1, so they can alternate without a jump
func shape(w Waveform, x, d, inc float64) float64 {
	min := math.Min(0.49, 2.5*inc) // the shortest a squeezed part may be
	switch w {
	case Square:
		m := math.Max(0.5*(1-d), min)
		var p float64
		if x < 0.5 {
			p = 0.5 * math.Min(x/m, 1)
		} else {
			p = 0.5 + 0.5*math.Min((x-0.5)/m, 1)
		}
		return -math.Cos(2 * math.Pi * p)
	case Pulse:
		m := math.Max(1-d, 2*min)
		return -math.Cos(2 * math.Pi * math.Min(x/m, 1))
	case SawPulse:
		// the first half of the cycle bent as a saw, the second as a pulse
		var p float64
		if x < 0.5 {
			p = 0.5 * sawPhase(2*x, d, 2*inc)
		} else {
			p = 0.5 + 0.5*math.Min((2*x-1)/math.Max(1-d, 4*min), 1)
		}
		return -math.Cos(2 * math.Pi * p)
	case Reso1, Reso2, Reso3:
		// a sine at up to 16 times the pitch, reset every cycle, under a window
		r := math.Min(1+d*15, 0.45/inc)
		var win float64
		switch w {
		case Reso1:
			win = 1 - x
		case Reso2:
			win = 1 - math.Abs(2*x-1)
		default:
			win = math.Min(1, 2*(1-x))
		}
		return win*(1-math.Cos(2*math.Pi*r*x)) - 1
	default: // Saw
		return -math.Cos(2 * math.Pi * sawPhase(x, d, inc))
	}
}

// sawPhase bends the phase for a saw: the first half of the cosine squeezed into the start
func sawPhase(x, d, inc float64) float64 {
	m := math.Max(0.5-0.5*d, math.Min(0.49, 2.5*inc))
	if x < m {
		return 0.5 * x / m
	}
	return 0.5 + 0.5*(x-m)/(1-m)
}

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
	var dcwModded [2]bool
	dcwModded[0], dcwModded[1] = modded && mods.Targets(dDCW1), modded && mods.Targets(dDCW2)
	pitchModded := modded && mods.Targets(dPitch)
	vibModded := modded && mods.Targets(dVibrato)
	widthModded := modded && mods.Targets(dWidth)
	levelModded := modded && mods.Targets(dLevel)
	detuneModded := modded && mods.Targets(dDetune)
	sweepRateModded := modded && mods.Targets(dSweepRate)
	sweepDepthModded := modded && mods.Targets(dSweepDepth)
	filtered := v.filter.On()
	filterTick := filtered || (modded && mods.Uses(modmatrix.FilterEnv))
	dcoOn := cfg.DCOAmount != 0
	depthOn := cfg.DepthAmount != 0
	gain := level * v.amp
	width := math.Max(0, math.Min(cfg.Width, 1))
	pan := func(p float64) (float64, float64) {
		a := (p + 1) * math.Pi / 4
		return math.Cos(a) * math.Sqrt2, math.Sin(a) * math.Sqrt2
	}
	l1L, l1R := pan(-width)
	l2L, l2R := pan(width)
	vibDelay := int(cfg.VibratoDelay * sr)
	sweepInc := cfg.SweepRate / sr
	if cfg.SweepSync > 0 {
		sweepInc = v.BPM() / 60 / cfg.SweepSync / sr
	}
	sweepOn := sweepInc > 0 && (cfg.SweepDepth != 0 || sweepDepthModded || (modded && mods.Uses(modmatrix.LFO1)))
	lfo2On := modded && mods.Uses(modmatrix.LFO2) && cfg.LFO2Rate > 0
	lfo2Inc := cfg.LFO2Rate / sr
	pitchTime := math.Max(0.001, cfg.PitchTime) * sr
	dcCoef := 1 - 2*math.Pi*10/sr
	var ratio [2]float64
	for i := range v.lines {
		l := v.line(i)
		ratio[i] = math.Exp2(float64(l.Octave) + l.Detune/1200)
	}
	if modded {
		mods.Values[modmatrix.Velocity] = v.amp
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(v.freq/130.81/2)/5, 1))
	}
	var lfo1, lfo2, depthEnv, dcoEnv float64
	channels := out.Channels
	stereo := len(channels) > 1
	for i := v.waitSamps; i < n; i++ {
		v.expr.Tick(sr)
		var dcw, dca [2]float64
		for k := range v.lines {
			dcw[k] = v.lines[k].dcw.Generate()
			dca[k] = v.lines[k].dca.Generate()
		}
		if depthOn {
			depthEnv = v.depth.Generate() * cfg.DepthAmount
		}
		if dcoOn {
			dcoEnv = v.dco.Generate() * cfg.DCOAmount
		}
		if sweepOn {
			lfo1 = math.Sin(2 * math.Pi * v.sweepPh)
			inc := sweepInc
			if sweepRateModded {
				inc *= math.Exp2(mods.Mod(dSweepRate))
			}
			v.sweepPh += inc
			v.sweepPh -= math.Floor(v.sweepPh)
		}
		if lfo2On {
			if cfg.LFO2Shape == 1 {
				t := v.lfo2Ph + 0.25
				t -= math.Floor(t)
				lfo2 = 1 - 4*math.Abs(t-0.5)
			} else {
				lfo2 = math.Sin(2 * math.Pi * v.lfo2Ph)
			}
			v.lfo2Ph += lfo2Inc
			v.lfo2Ph -= math.Floor(v.lfo2Ph)
		}
		if modded {
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = v.filter.Env(), dca[0]
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = lfo1, lfo2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
		}
		semis := v.expr.Semitones() + dcoEnv
		if pitchModded {
			semis += mods.Mod(dPitch)
		}
		if cfg.PitchSweep != 0 {
			semis += cfg.PitchSweep * math.Exp(-float64(v.t)/pitchTime*3)
		}
		if vd := cfg.VibratoDepth + mods.Mod(dVibrato); vd != 0 && (v.t > vibDelay) && (cfg.VibratoDepth != 0 || vibModded) {
			semis += vd * math.Sin(2*math.Pi*cfg.VibratoRate*float64(v.t-vibDelay)/sr)
		}
		f := v.freq
		if semis != 0 {
			f *= math.Exp2(semis / 12)
		}
		sweep := 0.0
		if sweepOn {
			sweep = (cfg.SweepDepth + mods.Mod(dSweepDepth)) * lfo1
		}
		if detuneModded {
			ratio[1] = math.Exp2(float64(cfg.Line2.Octave) + (cfg.Line2.Detune+mods.Mod(dDetune))/1200)
		}
		var y [2]float64
		for k := range v.lines {
			l, s := v.line(k), &v.lines[k]
			if l.Level <= 0 {
				continue
			}
			inc := f * ratio[k] / sr
			w := l.Waveform
			if l.Alternate && s.odd {
				w = l.Second
			}
			d := s.amount * dcw[k]
			if depthOn || dcwModded[k] {
				d = math.Max(0, math.Min(d+depthEnv+mods.Mod(dDCW1+k), 1))
			}
			if wheel := v.expr.Wheel(); wheel != 0 {
				d = math.Min(d+0.6*wheel, 1)
			}
			if sweep != 0 {
				d = math.Max(0, math.Min(d+sweep, 1))
			}
			y[k] = shape(w, s.phase, d, inc) * dca[k] * l.Level
			s.phase += inc
			if s.phase >= 1 {
				s.phase -= math.Floor(s.phase)
				s.odd = !s.odd
			}
		}
		if cfg.Ring {
			y[1] *= y[0]
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
			v.filter.Tick(f, 1.5*v.expr.Wheel()+mods.Mod(dCutoff), mods.Mod(dResonance))
		}
		for c := range o {
			if c > 0 && !stereo {
				break
			}
			// the resonance waveforms are not centered: a DC blocker
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
