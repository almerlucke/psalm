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
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
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

	// the low pass gate
	Gate      Gate
	Resonance float64 // 0-1: the gate rings when struck
	Response  float64 // seconds the vactrol takes to let go (0 is 0.15)
	Color     float64 // 0-1: how bright it opens (0 is 0.8)
	KeyTrack  float64 // 0-1: how much it opens brighter for higher notes (0 is 0.6, negative is none)

	// an LFO, a sine from the start of every note: Hz, or beats a cycle (LFOSync, follows the
	// tempo); it moves the fold (0-1, up and down) and the FM index
	LFORate float64
	LFOSync float64
	LFOFold float64
	LFOFM   float64

	Level float64 // output level (0 is treated as 1)
}

// DefaultConfig is the bongo
func DefaultConfig() Config {
	return Bongo()
}

// Voice plays a note
type Voice struct {
	*psalm.Module
	cfg       Config
	freq      float64
	velocity  float64
	amp       float64
	phase     float64 // the principal oscillator
	modPhase  float64
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
	v.SetSelf(v)
	return v
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
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
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
}

// Receive takes messages, or the fold on input 1 and the FM index on input 2 (for LFOs)
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
	}
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

func (v *Voice) modulator(phase float64) float64 {
	switch v.cfg.ModWave {
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
	gain := level * v.amp
	ratio := cfg.ModRatio
	if ratio <= 0 {
		ratio = 1
	}
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
	q := 0.5 + 8*math.Max(0, math.Min(cfg.Resonance, 1))
	lfoInc := cfg.LFORate / sr
	if cfg.LFOSync > 0 {
		lfoInc = v.BPM() / 60 / cfg.LFOSync / sr
	}
	channels := out.Channels
	for i := v.waitSamps; i < n; i++ {
		lfo := 0.0
		if lfoInc > 0 && (cfg.LFOFold != 0 || cfg.LFOFM != 0) {
			lfo = math.Sin(2 * math.Pi * lfoInc * float64(v.t))
		}
		v.t++
		// the function generator
		switch v.stage {
		case 0:
			v.env += attack
			if v.env >= 1 {
				v.env = 1
				v.stage = 1
				if !cfg.Sustain {
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
			fall := 1 / ((0.3 + 1.7*(1-v.vactrol)) * response * sr)
			v.vactrol += (c - v.vactrol) * fall
		}

		// the complex oscillator
		v.expr.Tick(sr)
		freq := v.freq * v.expr.Ratio()
		v.modPhase += freq * ratio / sr
		v.modPhase -= math.Floor(v.modPhase)
		mod := v.modulator(v.modPhase)
		index := cfg.FM + cfg.EnvToFM*v.env + cfg.LFOFM*lfo
		v.phase += freq / sr
		v.phase -= math.Floor(v.phase)
		s := math.Sin(2*math.Pi*v.phase + index*mod)
		if cfg.AM > 0 {
			s *= 1 - cfg.AM*0.5*(1-mod)
		}
		// the folder
		f := math.Max(0, math.Min(cfg.Fold+cfg.EnvToFold*v.env+cfg.LFOFold*lfo+0.6*v.expr.Wheel(), 1))
		if f > 0 || cfg.Symmetry != 0 {
			s = fold(s, 1+f*7, cfg.Symmetry*f)
		}
		// the gate
		vc := v.vactrol
		if cfg.Gate != VCA {
			if v.count == 0 {
				cutoff := 30 * math.Exp2(vc*color*9.5) * keyFactor
				v.lpg.SetFrequencyQ(math.Min(cutoff, sr*0.45), q*(0.3+0.7*vc))
			}
			v.count = (v.count + 1) % 4
			s = v.lpg.Process(s, svf.LowPass)
		}
		if cfg.Gate != LowPass {
			s *= vc * math.Sqrt(vc)
		}
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
	if v.stage == 3 && v.vactrol < 2e-3 { // the gate lets through vactrol^1.5: -80 dB
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
