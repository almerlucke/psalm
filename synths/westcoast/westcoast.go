// Package westcoast is a synth in the Buchla way: rich sounds are made, not filtered out. A complex
// oscillator (a principal sine, frequency and amplitude modulated by a second oscillator) goes into
// a wave folder that folds the wave over on itself for brighter overtones, then through a low pass
// gate: a filter and an amplifier opened together by a vactrol, a lamp and a light sensitive cell
// that light up at once and fade slowly. Struck short, the gate rings and dies like a hand drum
// (the famous bongo).
//
//	wc := westcoast.NewPoly(westcoast.Bongo(), 8).AddTo(p)
package westcoast

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/modules/poly"
)

// ModWave is the waveform of the modulation oscillator
type ModWave int

const (
	ModSine ModWave = iota
	ModTriangle
	ModSaw
	ModSquare
)

// Gate is what the low pass gate does
type Gate int

const (
	Combo   Gate = iota // the filter and the amplifier together
	LowPass             // the filter only: the gate closes darker, never silent
	VCA                 // the amplifier only
)

// Config is a patch
type Config struct {
	NumChannels int

	// the complex oscillator
	ModWave  ModWave
	ModRatio float64 // the modulator's frequency over the note's (0 is 1)
	FM       float64 // the index of the frequency modulation
	AM       float64 // 0-1: amplitude modulation

	// a second complex oscillator, off at Osc2Mix 0: its own modulator (wave and ratio) with the same
	// FM index and AM, detuned from the first, mixed in before the folder
	Osc2Mix      float64 // 0-1
	Osc2Detune   float64 // semitones
	Osc2ModWave  ModWave
	Osc2ModRatio float64 // the second modulator's frequency over the second oscillator's (0 is 1)

	// the timbre
	Fold     float64 // 0-1: how far the wave folds (0 leaves the sine)
	Symmetry float64 // -1 to 1: folds one side more than the other (even overtones)

	// the function generator: rises in Attack, falls in Decay, holds while the key is down with
	// Sustain. It opens the gate, and adds to the folding and the modulation
	Attack    float64 // seconds
	Decay     float64 // seconds
	Sustain   bool
	EnvToFold float64 // 0-1
	EnvToFM   float64 // index added at the top of the envelope
	Cycle     bool    // the function generator retriggers itself, an LFO (it never sustains)
	CycleRate float64 // Hz, how often it retriggers (0 is 2)

	// the low pass gate
	Gate      Gate
	Resonance float64 // 0-1: the gate rings when struck
	Response  float64 // seconds the vactrol takes to let go (0 is 0.15)
	Color     float64 // 0-1: how bright it opens (0 is 0.8)
	Strike    float64 // 0-1: how far the lamp lights at once at a note, the gate opens with a hit
	KeyTrack  float64 // 0-1: how much it opens brighter for higher notes (0 is 0.6, negative is none)

	// an LFO, a sine from the start of every note: Hz, or beats a cycle (LFOSync, follows the
	// tempo); it moves the fold (0-1, up and down) and the FM index
	LFORate float64
	LFOSync float64
	LFOFold float64
	LFOFM   float64

	// the matrix: a source to a destination (see Dests) over an amount. FilterEnv is the function
	// generator, AmpEnv the vactrol (the gate), LFO1 the LFO above, LFO2 a second sine from the
	// start of every note at LFO2Rate Hz (0 is 1)
	LFO2Rate float64
	Mods     [modmatrix.Slots]modmatrix.Slot
	Ext      [modmatrix.Exts]float64 // the values 0-1 of the sources Ext 1 to 4 (cabled in on InExt1...)

	Level float64 // output level (0 is treated as 1)
}

// DefaultConfig is the bongo
func DefaultConfig() Config {
	return Bongo()
}

// The destinations of the matrix (see Dests), by number
const (
	dFold = iota
	dSymmetry
	dFM
	dAM
	dModRatio
	dPitch
	dLevel
	dColor
	dResponse
	dResonance
	dLFORate
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (octaves, semitones, or the setting itself)
var Dests = []modmatrix.Dest{
	dFold:      {Key: "fold", Label: "fold", Span: 1},
	dSymmetry:  {Key: "symmetry", Label: "symmetry", Span: 1},
	dFM:        {Key: "fm", Label: "fm", Span: 4},
	dAM:        {Key: "am", Label: "am", Span: 1},
	dModRatio:  {Key: "modRatio", Label: "mod ratio", Span: 2},
	dPitch:     {Key: "pitch", Label: "pitch", Span: 24},
	dLevel:     {Key: "level", Label: "level", Span: 1},
	dColor:     {Key: "color", Label: "color", Span: 0.5},
	dResponse:  {Key: "response", Label: "response", Span: 2},
	dResonance: {Key: "resonance", Label: "resonance", Span: 1},
	dLFORate:   {Key: "lfoRate", Label: "lfo rate", Span: 3},
}

// The value inputs of a voice: 1 sets the fold and 2 the FM index, 5 to 8 the values of the matrix
// (Ext 1 to 4) that a slot can move any setting by
const (
	InFold = 1
	InFM   = 2
	InExt1 = 5 // to InExt1+3
)

const middleC = 261.6255653005986

// Voice plays a note
type Voice struct {
	*psalm.Module
	cfg       Config
	freq      float64
	velocity  float64
	amp       float64
	phase     float64 // the principal oscillator
	modPhase  float64
	phase2    float64 // the second oscillator
	modPhase2 float64
	lfoPhase  float64
	lfo2Phase float64
	cycle     float64 // where the function generator is in its cycle
	mods      *modmatrix.Matrix
	random    float64 // a value for the note, -1 to 1
	env       float64 // the function generator
	stage     int     // 0 rising, 1 holding, 2 falling, 3 done
	vactrol   float64
	lpg       *svf.SVF
	count     int
	t         int                   // samples since the note started
	expr      expression.Expression // the pitch bend; the mod wheel folds the wave further
	released  bool
	noteOff   int
	waitSamps int
	active    bool
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 220}
	v.lpg = svf.New(1000, 0, v.Config().SampleRate)
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
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
	// the function generator: "ampAttack", "ampDecay"; a note held (Sustain) falls in Decay once let
	// go, so "ampRelease" sets it too
	envelope.Seconds(msg, "amp", &v.cfg.Attack, &v.cfg.Decay, &v.cfg.Decay)
	msg.IfFltVal("frequency", func(f float64) { v.freq = f })
	msg.IfFltVal("pitch", func(p float64) { v.freq = 440 * math.Exp2((p-69)/12) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("fold", func(f float64) { v.cfg.Fold = f })
	msg.IfFltVal("symmetry", func(f float64) { v.cfg.Symmetry = f })
	msg.IfFltVal("fm", func(f float64) { v.cfg.FM = f })
	msg.IfFltVal("am", func(f float64) { v.cfg.AM = f })
	msg.IfFltVal("modRatio", func(f float64) { v.cfg.ModRatio = f })
	msg.IfFltVal("decay", func(f float64) { v.cfg.Decay = f })
	msg.IfFltVal("resonance", func(f float64) { v.cfg.Resonance = f })
	msg.IfFltVal("response", func(f float64) { v.cfg.Response = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	msg.IfFltVal("color", func(f float64) { v.cfg.Color = f })
	msg.IfFltVal("strike", func(f float64) { v.cfg.Strike = f })
	msg.IfIntVal("modWave", func(w int) { v.cfg.ModWave = ModWave(max(0, min(w, int(ModSquare)))) })
	msg.IfFltVal("osc2Mix", func(f float64) { v.cfg.Osc2Mix = f })
	msg.IfFltVal("osc2Detune", func(f float64) { v.cfg.Osc2Detune = f })
	msg.IfIntVal("osc2ModWave", func(w int) { v.cfg.Osc2ModWave = ModWave(max(0, min(w, int(ModSquare)))) })
	msg.IfFltVal("osc2ModRatio", func(f float64) { v.cfg.Osc2ModRatio = f })
	msg.IfBoolVal("cycle", func(b bool) { v.cfg.Cycle = b })
	msg.IfFltVal("cycleRate", func(f float64) { v.cfg.CycleRate = f })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = f })
}

// Receive takes messages, or the fold on input 1 and the FM index on input 2 (for LFOs), or on 5 to
// 8 (InExt1...) values 0-1 that the matrix takes as sources Ext 1 to 4
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case 1:
		v.cfg.Fold = psalm.FltVal[float64](value)
	case 2:
		v.cfg.FM = psalm.FltVal[float64](value)
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

func (v *Voice) strike(duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.velocity = 0.8
	v.ReceiveMessage(message)
	v.waitSamps = tick.SampleOffset
	v.amp = amplitude
	v.released = false
	v.noteOff = 0
	if duration > 0 {
		v.noteOff = max(1, int(duration*v.Config().SampleRate))
	}
	v.stage = 0
	if !v.active {
		v.env, v.vactrol = 0, 0
		v.lpg.Clear()
		v.phase, v.modPhase = 0, 0
		v.phase2, v.modPhase2 = 0, 0
	}
	if v.cfg.Strike > 0 {
		v.vactrol = math.Max(v.vactrol, math.Min(v.cfg.Strike, 1)*(0.5+0.5*v.velocity))
	}
	v.random = v.Rand().Float64()*2 - 1
	v.lfoPhase, v.lfo2Phase, v.cycle = 0, 0, 0
	v.count = 0
	v.t = 0
	v.active = true
}

func (v *Voice) NoteOff() {
	v.released = true
	v.noteOff = 0
}

func (v *Voice) Clear() {
	v.active = false
	v.env, v.vactrol = 0, 0
	v.lpg.Clear()
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return v.active
}

// fold folds a wave (-1 to 1) back on itself: a sine of it, gain times as far round. Folding a
// sine this way stays smooth, its overtones fall off fast (it hardly aliases)
func fold(x, gain, symmetry float64) float64 {
	return math.Sin(math.Pi / 2 * (x*gain + symmetry))
}

func modulator(wave ModWave, phase float64) float64 {
	switch wave {
	case ModTriangle:
		return 1 - 4*math.Abs(phase-0.5)
	case ModSaw: // softened at its edge: no hard jump
		return math.Sin(math.Pi*phase) * (1 - 2*phase) * 1.3
	case ModSquare: // a rounded square
		return math.Tanh(4 * math.Sin(2*math.Pi*phase))
	default:
		return math.Sin(2 * math.Pi * phase)
	}
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
	gain := v.amp
	ratio := cfg.ModRatio
	if ratio <= 0 {
		ratio = 1
	}
	ratio2 := cfg.Osc2ModRatio
	if ratio2 <= 0 {
		ratio2 = 1
	}
	detune2 := math.Exp2(cfg.Osc2Detune / 12)
	attack := 1 / math.Max(1, cfg.Attack*sr)
	decay := math.Pow(10, -3/(math.Max(0.002, cfg.Decay)*sr))
	response := cfg.Response
	if response <= 0 {
		response = 0.15
	}
	color := cfg.Color
	if color <= 0 {
		color = 0.8
	}
	vel := 0.5 + 0.5*v.velocity
	// the gate follows the key: higher notes open brighter, so they don't come out dull
	track := cfg.KeyTrack
	if track == 0 {
		track = 0.6
	}
	keyFactor := math.Pow(v.freq/220, math.Max(0, track))
	// the vactrol: it lights up in a few milliseconds and lets go slowly, slower still as it dims
	rise := 1 - math.Exp(-1/(0.003*sr))
	resonance := math.Max(0, math.Min(cfg.Resonance, 1))
	lfoInc := cfg.LFORate / sr
	if cfg.LFOSync > 0 {
		lfoInc = v.BPM() / 60 / cfg.LFOSync / sr
	}
	lfo2Rate := cfg.LFO2Rate
	if lfo2Rate <= 0 {
		lfo2Rate = 1
	}
	cycleRate := cfg.CycleRate
	if cycleRate <= 0 {
		cycleRate = 2
	}
	mods := v.mods
	modded := mods.Active()
	lfoOn := cfg.LFOFold != 0 || cfg.LFOFM != 0 || mods.Uses(modmatrix.LFO1)
	lfo2On := mods.Uses(modmatrix.LFO2)
	if modded {
		mods.Values[modmatrix.Velocity] = v.velocity
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(v.freq/middleC)/5, 1))
	}
	channels := out.Channels
	for i := v.waitSamps; i < n; i++ {
		lfo, lfo2 := 0.0, 0.0
		rateMod := 1.0
		if modded {
			rateMod = math.Exp2(mods.Mod(dLFORate))
		}
		if lfoInc > 0 && lfoOn {
			lfo = math.Sin(2 * math.Pi * v.lfoPhase)
			v.lfoPhase += lfoInc * rateMod
			v.lfoPhase -= math.Floor(v.lfoPhase)
		}
		if lfo2On {
			lfo2 = math.Sin(2 * math.Pi * v.lfo2Phase)
			v.lfo2Phase += lfo2Rate * rateMod / sr
			v.lfo2Phase -= math.Floor(v.lfo2Phase)
		}
		v.t++
		// the function generator
		if cfg.Cycle && !v.released {
			v.cycle += cycleRate / sr
			if v.cycle >= 1 {
				v.cycle -= math.Floor(v.cycle)
				v.stage = 0
			}
		}
		switch v.stage {
		case 0:
			v.env += attack
			if v.env >= 1 {
				v.env = 1
				v.stage = 1
				if !cfg.Sustain || cfg.Cycle {
					v.stage = 2
				}
			}
		case 1:
			if v.released {
				v.stage = 2
			}
		case 2:
			v.env *= decay
			if v.env < 1e-4 {
				v.env, v.stage = 0, 3
			}
		}
		c := v.env * vel
		if c > v.vactrol {
			v.vactrol += (c - v.vactrol) * rise
		} else {
			resp := response
			if modded {
				resp *= math.Exp2(mods.Mod(dResponse))
			}
			fall := 1 / ((0.3 + 1.7*(1-v.vactrol)) * math.Max(0.005, resp) * sr)
			v.vactrol += (c - v.vactrol) * fall
		}
		if modded {
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = v.env, v.vactrol
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = lfo, lfo2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
		}

		// the complex oscillators
		v.expr.Tick(sr)
		freq := v.freq * v.expr.Ratio()
		r := ratio
		am := cfg.AM
		index := cfg.FM + cfg.EnvToFM*v.env + cfg.LFOFM*lfo
		symmetry := cfg.Symmetry
		fl := cfg.Fold + cfg.EnvToFold*v.env + cfg.LFOFold*lfo + 0.6*v.expr.Wheel()
		if modded {
			freq *= math.Exp2(mods.Mod(dPitch) / 12)
			r *= math.Exp2(mods.Mod(dModRatio))
			am = math.Max(0, math.Min(am+mods.Mod(dAM), 1))
			index = math.Max(0, index+mods.Mod(dFM))
			symmetry = math.Max(-1, math.Min(symmetry+mods.Mod(dSymmetry), 1))
			fl += mods.Mod(dFold)
		}
		v.modPhase += freq * r / sr
		v.modPhase -= math.Floor(v.modPhase)
		mod := modulator(cfg.ModWave, v.modPhase)
		v.phase += freq / sr
		v.phase -= math.Floor(v.phase)
		s := math.Sin(2*math.Pi*v.phase + index*mod)
		if am > 0 {
			s *= 1 - am*0.5*(1-mod)
		}
		if cfg.Osc2Mix > 0 {
			f2 := freq * detune2
			v.modPhase2 += f2 * ratio2 * (r / ratio) / sr
			v.modPhase2 -= math.Floor(v.modPhase2)
			mod2 := modulator(cfg.Osc2ModWave, v.modPhase2)
			v.phase2 += f2 / sr
			v.phase2 -= math.Floor(v.phase2)
			s2 := math.Sin(2*math.Pi*v.phase2 + index*mod2)
			if am > 0 {
				s2 *= 1 - am*0.5*(1-mod2)
			}
			mix := math.Min(cfg.Osc2Mix, 1)
			s = (s + mix*s2) / (1 + mix)
		}
		// the folder
		f := math.Max(0, math.Min(fl, 1))
		if f > 0 || symmetry != 0 {
			s = fold(s, 1+f*7, symmetry*f)
		}
		// the gate
		vc := v.vactrol
		if cfg.Gate != VCA {
			if v.count == 0 {
				col, res := color, resonance
				if modded {
					col = math.Max(0.02, math.Min(col+mods.Mod(dColor), 2))
					res = math.Max(0, math.Min(res+mods.Mod(dResonance), 1))
				}
				cutoff := 30 * math.Exp2(vc*col*9.5) * keyFactor
				v.lpg.SetFrequencyQ(math.Min(cutoff, sr*0.45), (0.5+8*res)*(0.3+0.7*vc))
			}
			v.count = (v.count + 1) % 4
			s = v.lpg.Process(s, svf.LowPass)
		}
		if cfg.Gate != LowPass {
			s *= vc * math.Sqrt(vc)
		}
		g := gain * level
		if modded {
			g = gain * math.Max(0, level+mods.Mod(dLevel))
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
	if v.stage == 3 && v.vactrol < 2e-3 && !(cfg.Cycle && !v.released) { // the gate lets through vactrol^1.5: -80 dB
		v.active = false
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
