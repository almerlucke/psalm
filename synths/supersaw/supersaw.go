// Package supersaw is the supersaw of the Roland JP-8000: seven sawtooths detuned around the note
// into one wide, shimmering sound, the sound of trance pads and leads. The detune and the mix of the
// center saw against the others follow the curves Adam Szabo measured on the original; the saws
// here are band limited (no aliasing) and spread over the stereo field, then through a filter with
// its envelope.
//
//	pad := supersaw.NewPoly(supersaw.TrancePad(), 8).AddTo(p)
package supersaw

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/poly"
)

// Config is a supersaw patch
type Config struct {
	NumChannels int
	Saws        int     // 0 is 7
	Detune      float64 // 0-1: from unison to a wide smear
	Mix         float64 // 0-1: the side saws against the center one
	Stereo      float64 // 0-1: the saws spread left and right
	Sub         float64 // 0-1: a square an octave down
	HighPass    bool    // takes out what lies below the note (as the original does)
	HPCutoff    float64 // Hz, above 20: a high pass at that fixed frequency instead (in series, before the filter)

	Cutoff          float64 // Hz
	Resonance       float64 // 0-1
	FilterEnvAmount float64 // octaves the envelope opens the filter
	KeyTrack        float64 // 0-1: the cutoff follows the note
	Velocity        float64 // octaves a hard note opens the filter more
	FilterEnv       adsr.Setting
	AmpEnv          adsr.Setting

	Glide float64 // seconds from the previous note of the voice

	// an LFO, a sine from the start of every note: Hz, or beats a cycle (LFOSync, follows the
	// tempo); it moves the cutoff (octaves up and down) and the detune (added, 0-1)
	LFORate   float64
	LFOSync   float64
	LFOCutoff float64
	LFODetune float64
	Level     float64 // output level (0 is treated as 1)

	// a second LFO, a plain sine (LFO2Shape 0) or triangle (1) from the start of every note, running
	// only when the matrix uses it
	LFO2Rate  float64 // Hz
	LFO2Shape int

	// the ensemble: three modulated delays on each channel, the chorus of a string machine
	Ensemble     float64 // 0-1: how much of it (0 is off)
	EnsembleRate float64 // Hz (0 is 0.6)

	Mods [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext  [modmatrix.Exts]float64         // the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)
}

// The destinations of the matrix (see Dests), by number
const (
	dDetune = iota
	dMix
	dStereo
	dSub
	dCutoff
	dResonance
	dPitch
	dLevel
	dLFORate
	dEnsemble
	numDests
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (semitones, octaves, or the setting itself)
var Dests = []modmatrix.Dest{
	dDetune:    {Key: "detune", Label: "detune", Span: 0.5},
	dMix:       {Key: "mix", Label: "mix", Span: 0.5},
	dStereo:    {Key: "stereo", Label: "stereo", Span: 0.5},
	dSub:       {Key: "sub", Label: "sub", Span: 0.5},
	dCutoff:    {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance: {Key: "resonance", Label: "resonance", Span: 1},
	dPitch:     {Key: "pitch", Label: "pitch", Span: 12},
	dLevel:     {Key: "level", Label: "level", Span: 1},
	dLFORate:   {Key: "lfoRate", Label: "lfo rate", Span: 3},
	dEnsemble:  {Key: "ensemble", Label: "ensemble", Span: 0.5},
}

// The value inputs of a voice: 1 sets the cutoff, 2 the detune, 5 to 8 the values of the matrix
// (Ext 1 to 4) that a slot can move any setting by
const (
	InCutoff = 1
	InDetune = 2
	InExt1   = 5 // to InExt1+3
)

// DefaultConfig is a trance pad
func DefaultConfig() Config {
	return TrancePad()
}

// the detunes of the seven saws of the original, at full detune (relative frequency)
var offsets7 = [7]float64{-0.11002313, -0.06288439, -0.01952356, 0, 0.01991221, 0.06216538, 0.10745242}

// detuneCurve is the JP-8000's response of the detune knob (Szabo)
func detuneCurve(d float64) float64 {
	d = max(0, min(d, 1))
	c := []float64{10028.7312891634, -50818.8652045924, 111363.4808729368, -138150.6761080548, 106649.6679158292,
		-53046.9642751875, 17019.9518580080, -3425.0836591318, 404.2703938388, -24.1878824391, 0.6717417634, 0.0030115596}
	y := 0.0
	for _, k := range c {
		y = y*d + k
	}
	return y
}

// mixLevels returns the level of the center saw and of the others (Szabo)
func mixLevels(m float64) (center, side float64) {
	m = max(0, min(m, 1))
	return -0.55366*m + 0.99785, -0.73764*m*m + 1.2841*m + 0.044372
}

// maxSaws is the most saws a voice plays
const maxSaws = 16

type saw struct {
	osc    *blosc.Blosc
	offset float64 // its detune at full detune
	pos    float64 // -1 to 1, before the stereo spread
	gainL  float64
	gainR  float64
	center bool
}

// Voice plays a note
type Voice struct {
	*psalm.Module
	cfg       Config
	saws      [maxSaws]saw
	sub       *blosc.Blosc
	n         int
	freq      float64
	glideFreq float64
	velocity  float64
	filters   [2]*svf.SVF
	hp        [2]*svf.SVF
	filterEnv *adsr.ADSR
	ampEnv    *adsr.ADSR
	count     int
	lfoPhase  float64
	expr      expression.Expression // the pitch bend; the mod wheel opens the filter
	waitSamps int
	rng       uint64
	mods      *modmatrix.Matrix
	amplitude float64
	random    float64 // -1 to 1, for the note
	lfo2Phase float64
	ens       [2]ensemble
	ensPhase  float64
	spreadMod bool // the matrix moves the stereo spread
}

// New returns a voice
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, freq: 220, glideFreq: 220}
	sr := v.Config().SampleRate
	for i := range v.saws {
		v.saws[i].osc = blosc.New(220, sr, 0, 0.5, 1, blosc.Sawtooth)
	}
	v.sub = blosc.New(110, sr, 0, 0.5, 1, blosc.Square)
	for c := range v.filters {
		v.filters[c] = svf.New(1000, 0, sr)
		v.hp[c] = svf.New(100, 0, sr)
	}
	v.filterEnv = adsr.New(&v.cfg.FilterEnv, adsr.NoteOff, sr)
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)
	v.rng = v.Rand().Uint64() | 1
	v.mods = modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		v.mods.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		v.mods.SetExt(i, e)
	}
	v.layout()
	v.SetSelf(v)
	return v
}

// layout sets the detunes and places of the saws
func (v *Voice) layout() {
	n := v.cfg.Saws
	if n <= 0 {
		n = 7
	}
	n = min(n, maxSaws)
	v.n = n
	stereo := max(0, min(v.cfg.Stereo, 1))
	for i := range n {
		s := &v.saws[i]
		if n == 7 {
			s.offset = offsets7[i]
		} else if n > 1 {
			s.offset = -0.11 + 0.22*float64(i)/float64(n-1)
		} else {
			s.offset = 0
		}
		s.center = n%2 == 1 && i == n/2
		// the saws alternate left and right, the outer ones furthest
		pos := 0.0
		if n > 1 {
			pos = float64(i)/float64(n-1)*2 - 1
			if i%2 == 1 {
				pos = -pos
			}
		}
		s.pos = pos
	}
	v.spread(stereo)
}

// spread sets the left and right gains of the saws for a stereo width
func (v *Voice) spread(stereo float64) {
	stereo = max(0, min(stereo, 1))
	for i := range v.n {
		s := &v.saws[i]
		a := (s.pos*stereo + 1) * math.Pi / 4
		s.gainL, s.gainR = math.Cos(a)*math.Sqrt2, math.Sin(a)*math.Sqrt2
	}
}

func (v *Voice) random01() float64 {
	v.rng ^= v.rng << 13
	v.rng ^= v.rng >> 7
	v.rng ^= v.rng << 17
	return float64(v.rng>>11) / (1 << 53)
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	envelope.Receive(msg, "amp", &v.cfg.AmpEnv)
	envelope.Receive(msg, "filter", &v.cfg.FilterEnv)
	msg.IfFltVal("frequency", func(f float64) { v.setFrequency(f) })
	msg.IfFltVal("pitch", func(p float64) { v.setFrequency(440 * math.Exp2((p-69)/12)) })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = max(0, min(f, 1)) })
	msg.IfFltVal("detune", func(f float64) { v.cfg.Detune = f })
	msg.IfFltVal("mix", func(f float64) { v.cfg.Mix = f })
	msg.IfFltVal("stereo", func(f float64) { v.cfg.Stereo = f; v.layout() })
	msg.IfIntVal("saws", func(n int) { v.cfg.Saws = n; v.layout() })
	msg.IfFltVal("sub", func(f float64) { v.cfg.Sub = f })
	msg.IfFltVal("cutoff", func(f float64) { v.cfg.Cutoff = f })
	msg.IfFltVal("resonance", func(f float64) { v.cfg.Resonance = f })
	msg.IfFltVal("filterEnvAmount", func(f float64) { v.cfg.FilterEnvAmount = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	msg.IfFltVal("hpCutoff", func(f float64) { v.cfg.HPCutoff = f })
	msg.IfFltVal("lfoRate", func(f float64) { v.cfg.LFORate = f })
	msg.IfFltVal("lfoCutoff", func(f float64) { v.cfg.LFOCutoff = f })
	msg.IfFltVal("lfoDetune", func(f float64) { v.cfg.LFODetune = f })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = f })
	msg.IfIntVal("lfo2Shape", func(n int) { v.cfg.LFO2Shape = max(0, min(n, 1)) })
	msg.IfFltVal("ensemble", func(f float64) { v.cfg.Ensemble = max(0, min(f, 1)) })
	msg.IfFltVal("ensembleRate", func(f float64) { v.cfg.EnsembleRate = f })
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
}

func (v *Voice) setFrequency(f float64) {
	v.freq = f
	if v.cfg.Glide <= 0 {
		v.glideFreq = f
	}
}

// Receive takes messages, or the cutoff on input 1 and the detune on input 2 (for LFOs), and on 5 to 8
// (InExt1...) values 0-1 that the matrix takes as sources Ext 1 to 4
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	switch index {
	case InCutoff:
		v.cfg.Cutoff = psalm.FltVal[float64](value)
	case InDetune:
		v.cfg.Detune = psalm.FltVal[float64](value)
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
	v.amplitude = amplitude
	if v.mods.Uses(modmatrix.Random) { // drawn only when used: the random stream stays as it was
		v.random = v.random01()*2 - 1
	}
	fresh := v.ampEnv.Done()
	v.ReceiveMessage(message)
	if fresh {
		v.glideFreq = v.freq
		// every saw starts somewhere else: no two notes sound the same
		for i := range v.n {
			v.saws[i].osc.SetPhase(v.random01())
		}
		for c := range v.filters {
			v.filters[c].Clear()
			v.hp[c].Clear()
		}
		v.ens[0].clear()
		v.ens[1].clear()
	}
	v.waitSamps = tick.SampleOffset
	v.lfoPhase, v.lfo2Phase = 0, 0
	v.filterEnv.TriggerFull(duration*1000, 1, &v.cfg.FilterEnv, mode)
	v.ampEnv.TriggerFull(duration*1000, amplitude, &v.cfg.AmpEnv, mode)
}

func (v *Voice) NoteOff() {
	v.filterEnv.Release()
	v.ampEnv.Release()
}

func (v *Voice) Clear() {
	v.filterEnv.Clear()
	v.ampEnv.Clear()
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return !v.ampEnv.Done()
}

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	if v.ampEnv.Done() {
		return
	}
	cfg := &v.cfg
	n := v.Config().BufferSize
	sr := v.Config().SampleRate
	level := orOne(cfg.Level)
	mods := v.mods
	modded := mods.Active()
	detuneModded := modded && mods.Targets(dDetune)
	mixModded := modded && mods.Targets(dMix)
	stereoModded := modded && mods.Targets(dStereo)
	subModded := modded && mods.Targets(dSub)
	resModded := modded && mods.Targets(dResonance)
	pitchModded := modded && mods.Targets(dPitch)
	ensModded := modded && mods.Targets(dEnsemble)
	if v.spreadMod && !stereoModded {
		v.layout()
	}
	v.spreadMod = stereoModded
	detune := detuneCurve(cfg.Detune)
	lfoInc := cfg.LFORate / sr
	if cfg.LFOSync > 0 {
		lfoInc = v.BPM() / 60 / cfg.LFOSync / sr
	}
	lfoOn := lfoInc > 0 && (cfg.LFOCutoff != 0 || cfg.LFODetune != 0 || mods.Uses(modmatrix.LFO1))
	lfo2On := modded && mods.Uses(modmatrix.LFO2) && cfg.LFO2Rate > 0
	lfo2Inc := cfg.LFO2Rate / sr
	lfo, lfo2 := 0.0, 0.0
	center, side := mixLevels(cfg.Mix)
	// the level of the sum: the saws add up as noise does (they are not in phase), but every so
	// often they line up: room for that
	normOf := func(center, side float64) float64 {
		return 0.4 / math.Sqrt(center*center+float64(v.n-1)*side*side)
	}
	norm := normOf(center, side)
	glide := 1.0
	if cfg.Glide > 0 {
		glide = 1 - math.Exp(-1/(cfg.Glide*sr))
	}
	resonance := max(0, min(cfg.Resonance, 1))
	q := 0.5 + 9.5*resonance
	keyTrack := max(0, min(cfg.KeyTrack, 1))
	vel := cfg.Velocity * (v.velocity - 0.5)
	hpFixed := cfg.HPCutoff > 20
	hpOn := cfg.HighPass || hpFixed
	subLevel := cfg.Sub
	ensRate := cfg.EnsembleRate
	if ensRate <= 0 {
		ensRate = 0.6
	}
	ensInc := ensRate / sr
	ensOn := cfg.Ensemble > 0 || ensModded
	if modded {
		mods.Values[modmatrix.Velocity] = v.amplitude
		mods.Values[modmatrix.Random] = v.random
	}
	channels := out.Channels
	stereo := len(channels) > 1
	for i := v.waitSamps; i < n; i++ {
		v.expr.Tick(sr)
		v.glideFreq += (v.freq - v.glideFreq) * glide
		f := v.glideFreq * v.expr.Ratio()
		if pitchModded {
			f *= math.Exp2(mods.Mod(dPitch) / 12)
		}
		fenv := v.filterEnv.Generate()
		amp := v.ampEnv.Generate()
		if lfoOn {
			if v.count == 0 {
				lfo = math.Sin(2 * math.Pi * v.lfoPhase)
			}
			inc := lfoInc
			if modded && mods.Targets(dLFORate) {
				inc *= math.Exp2(mods.Mod(dLFORate))
			}
			v.lfoPhase += inc
			v.lfoPhase -= math.Floor(v.lfoPhase)
		}
		if lfo2On {
			if v.count == 0 {
				if cfg.LFO2Shape == 1 {
					t := v.lfo2Phase + 0.25
					t -= math.Floor(t)
					lfo2 = 1 - 4*math.Abs(t-0.5)
				} else {
					lfo2 = math.Sin(2 * math.Pi * v.lfo2Phase)
				}
			}
			v.lfo2Phase += lfo2Inc
			v.lfo2Phase -= math.Floor(v.lfo2Phase)
		}
		if modded {
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = fenv, amp
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = lfo, lfo2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			if v.count == 0 {
				mv[modmatrix.Key] = max(-1, min(math.Log2(v.glideFreq/middleC)/5, 1))
			}
			mods.Eval(sr)
		}
		if v.count == 0 {
			if (lfoOn && cfg.LFODetune != 0) || detuneModded {
				d := cfg.Detune
				if lfoOn {
					d += cfg.LFODetune * (lfo + 1) / 2
				}
				detune = detuneCurve(d + mods.Mod(dDetune))
			}
			if mixModded {
				center, side = mixLevels(cfg.Mix + mods.Mod(dMix))
				norm = normOf(center, side)
			}
			if stereoModded {
				v.spread(cfg.Stereo + mods.Mod(dStereo))
			}
			if subModded {
				subLevel = cfg.Sub + mods.Mod(dSub)
			}
			if resModded {
				q = 0.5 + 9.5*max(0, min(resonance+mods.Mod(dResonance), 1))
			}
		}
		var y [2]float64
		for k := range v.n {
			s := &v.saws[k]
			if v.count == 0 {
				s.osc.SetFrequency(f*(1+s.offset*detune), sr)
			}
			x := s.osc.Tick(0, false, 0)
			if s.center {
				x *= center
			} else {
				x *= side
			}
			y[0] += x * s.gainL
			y[1] += x * s.gainR
		}
		y[0] *= norm
		y[1] *= norm
		if subLevel > 0 {
			if v.count == 0 {
				v.sub.SetFrequency(f/2, sr)
			}
			sub := v.sub.Tick(0, false, 0) * subLevel * 0.7
			y[0] += sub
			y[1] += sub
		}
		if v.count == 0 {
			cutoff := cfg.Cutoff * math.Pow(f/261.63, keyTrack) * math.Exp2(cfg.FilterEnvAmount*fenv+vel+cfg.LFOCutoff*lfo+3*v.expr.Wheel()+mods.Mod(dCutoff))
			cutoff = max(20, min(cutoff, sr*0.45))
			for c := range v.filters {
				v.filters[c].SetFrequencyQ(cutoff, q)
				if hpFixed {
					v.hp[c].SetFrequencyQ(min(cfg.HPCutoff, sr*0.45), 0.7)
				} else if cfg.HighPass {
					v.hp[c].SetFrequencyQ(min(f, sr*0.45), 0.7)
				}
			}
		}
		v.count = (v.count + 1) % 4
		amp *= max(0, level+mods.Mod(dLevel))
		chans := 1
		if stereo {
			chans = 2
		} else {
			y[0] = (y[0] + y[1]) / 2
		}
		var o [2]float64
		for c := range chans {
			s := y[c]
			if hpOn {
				s = v.hp[c].Process(s, svf.HighPass)
			}
			s = v.filters[c].Process(s, svf.LowPass)
			o[c] = s * amp
		}
		if ensOn {
			a := max(0, min(cfg.Ensemble+mods.Mod(dEnsemble), 1))
			for c := range chans {
				o[c] = v.ens[c].process(o[c], a, v.ensPhase+0.17*float64(c), sr)
			}
			v.ensPhase += ensInc
			v.ensPhase -= math.Floor(v.ensPhase)
		}
		for c := range chans {
			channels[c][i] = float32(o[c])
		}
		if v.ampEnv.Done() {
			break
		}
	}
	v.waitSamps = 0
}

const middleC = 261.63

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
