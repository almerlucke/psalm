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
	"github.com/almerlucke/psalm/blocks/expression"
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
}

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
}

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
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 220}
	sr := v.Config().SampleRate
	v.lines[0].dcw = adsr.New(&v.cfg.Line1.DCWEnv, adsr.NoteOff, sr)
	v.lines[0].dca = adsr.New(&v.cfg.Line1.DCAEnv, adsr.NoteOff, sr)
	v.lines[1].dcw = adsr.New(&v.cfg.Line2.DCWEnv, adsr.NoteOff, sr)
	v.lines[1].dca = adsr.New(&v.cfg.Line2.DCAEnv, adsr.NoteOff, sr)
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
	msg.IfFltVal("frequency", func(f float64) { v.freq = f })
	msg.IfFltVal("pitch", func(p float64) { v.freq = 440 * math.Exp2((p-69)/12) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("dcw1", func(f float64) { v.cfg.Line1.DCW = f })
	msg.IfFltVal("dcw2", func(f float64) { v.cfg.Line2.DCW = f })
	msg.IfIntVal("waveform1", func(w int) { v.cfg.Line1.Waveform = Waveform(w) })
	msg.IfIntVal("waveform2", func(w int) { v.cfg.Line2.Waveform = Waveform(w) })
	msg.IfFltVal("detune", func(f float64) { v.cfg.Line2.Detune = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { v.cfg.VibratoDepth = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
}

// Receive takes messages, or the distortion of both lines (0-1) on input 1
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	if index == 1 {
		f := psalm.FltVal[float64](value)
		v.cfg.Line1.DCW, v.cfg.Line2.DCW = f, f
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
	for i := range v.lines {
		v.lines[i].dcw.Release()
		v.lines[i].dca.Release()
	}
}

func (v *Voice) Clear() {
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
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
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
	pitchTime := math.Max(0.001, cfg.PitchTime) * sr
	dcCoef := 1 - 2*math.Pi*10/sr
	var ratio [2]float64
	for i := range v.lines {
		l := v.line(i)
		ratio[i] = math.Exp2(float64(l.Octave) + l.Detune/1200)
	}
	channels := out.Channels
	stereo := len(channels) > 1
	for i := v.waitSamps; i < n; i++ {
		v.expr.Tick(sr)
		semis := v.expr.Semitones()
		if cfg.PitchSweep != 0 {
			semis += cfg.PitchSweep * math.Exp(-float64(v.t)/pitchTime*3)
		}
		if cfg.VibratoDepth != 0 && v.t > vibDelay {
			semis += cfg.VibratoDepth * math.Sin(2*math.Pi*cfg.VibratoRate*float64(v.t-vibDelay)/sr)
		}
		f := v.freq
		if semis != 0 {
			f *= math.Exp2(semis / 12)
		}
		sweep := 0.0
		if cfg.SweepDepth != 0 && sweepInc > 0 {
			sweep = cfg.SweepDepth * math.Sin(2*math.Pi*sweepInc*float64(v.t))
		}
		var y [2]float64
		for k := range v.lines {
			l, s := v.line(k), &v.lines[k]
			dcw := s.dcw.Generate()
			dca := s.dca.Generate()
			if l.Level <= 0 {
				continue
			}
			inc := f * ratio[k] / sr
			w := l.Waveform
			if l.Alternate && s.odd {
				w = l.Second
			}
			d := s.amount * dcw
			if wheel := v.expr.Wheel(); wheel != 0 {
				d = math.Min(d+0.6*wheel, 1)
			}
			if sweep != 0 {
				d = math.Max(0, math.Min(d+sweep, 1))
			}
			y[k] = shape(w, s.phase, d, inc) * dca * l.Level
			s.phase += inc
			if s.phase >= 1 {
				s.phase -= math.Floor(s.phase)
				s.odd = !s.odd
			}
		}
		if cfg.Ring {
			y[1] *= y[0]
		}
		var o [2]float64
		if stereo {
			o[0] = y[0]*l1L + y[1]*l2L
			o[1] = y[0]*l1R + y[1]*l2R
		} else {
			o[0] = y[0] + y[1]
		}
		for c := range o {
			// the resonance waveforms are not centered: a DC blocker
			x := o[c]
			v.dc[c] = x - v.dcIn[c] + dcCoef*v.dc[c]
			v.dcIn[c] = x
			o[c] = v.dc[c]
		}
		channels[0][i] = float32(o[0] * gain * 0.5)
		if stereo {
			channels[1][i] = float32(o[1] * gain * 0.5)
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
