package wavetable

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	ovs "github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/blocks/sine"
	"github.com/almerlucke/psalm/modules/poly"
)

// FilterMode selects the filter after the oscillators
type FilterMode int

const (
	FilterOff FilterMode = iota
	LowPass
	BandPass
	HighPass
)

// WarpMode selects how the phase of the oscillators is bent before the table is read
type WarpMode int

const (
	WarpOff      WarpMode = iota
	WarpBendUp            // the cycle is pushed towards its end: the first part of the frame is stretched
	WarpBendDown          // the cycle is pushed towards its start: the last part of the frame is stretched
	WarpSync              // the frame is played 1 to 8 times in a cycle, like a hard synced oscillator
	WarpMirror            // the cycle is played forward, then backward (a triangle of the phase)
	WarpFMB               // oscillator B (as it sounds, before its level) modulates the phase
	WarpFMSine            // a sine at the pitch of the note modulates the phase
	numWarpModes
)

// MaxUnison is the largest number of unison oscillators
const MaxUnison = 7

// Config describes a wavetable voice: an oscillator morphing through the frames of a table (the
// position), stacked in unison and spread over the stereo field, a sine sub oscillator and a
// state variable filter
type Config struct {
	NumChannels int

	Table *Table // nil plays the Basic table

	// Position through the table
	Position          float64 // 0-1, morphs through the frames
	PositionEnvAmount float64 // added to the position at full position envelope (-1 to 1)
	PositionLFORate   float64 // Hz
	PositionLFODepth  float64 // 0-1, the most the LFO moves the position either way

	// Unison
	Unison int     // 1-7 oscillators (0 is 1)
	Detune float64 // cents between the lowest and highest unison oscillator
	Spread float64 // 0-1, stereo width of the unison oscillators

	SubLevel float64 // sine an octave below

	// Warp bends the phase of oscillator A (and B, but for the FM modes) before the table is read
	Warp     float64  // 0-1, how far
	WarpMode WarpMode // see WarpMode

	// Oscillator B reads a table of its own position at its own pitch, mixed with oscillator A (and
	// the sub): one oscillator, in the middle, with the same table and normalization
	OscBLevel    float64 // 0-1, 0 is off
	OscBPosition float64 // 0-1, position through the table
	OscBOctave   int     // -3 to 3 octaves from the note
	OscBSemi     float64 // -12 to 12 semitones from the note
	OscBDetune   float64 // -50 to 50 cents from the note
	// OscBUnison stacks oscillator B in 1-7 oscillators (0 is 1), spread over OscBSpread cents (0 takes
	// Detune) and the stereo width Spread
	OscBUnison int
	OscBSpread float64

	// Two LFOs (sines) for the sources LFO 1 and LFO 2 of the matrix; they run only when a slot uses them
	LFO1Rate float64 // Hz (0 is 1 Hz)
	LFO2Rate float64 // Hz (0 is 1 Hz)

	// The matrix: a source to a destination (see Dests) over an amount
	Mods [modmatrix.Slots]modmatrix.Slot
	// Ext are the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)
	Ext [modmatrix.Exts]float64

	// Normalize brings every frame of the table to the same loudness, so moving the position
	// changes the timbre but not the level (quiet frames are brought up at most 4 times)
	Normalize bool

	// Filter
	Filter          FilterMode
	Cutoff          float64 // Hz
	Resonance       float64 // 0-1
	FilterEnvAmount float64 // octaves added to the cutoff at full filter envelope
	KeyTrack        float64 // 0-1, how much the cutoff follows the pitch (relative to middle C)

	// Pitch
	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	Glide        float64 // milliseconds from the previous note of the voice

	Level float64 // output level (0 is treated as 1)

	// Oversample plays the oscillators once (0), twice (1) or four times (2) a sample, filtered down: the warps (bend, mirror, sync, FM) make corners and steps that
	// fold back at a high note. Costs that many times the oscillators
	Oversample int

	// Envelopes (durations in milliseconds)
	PositionEnv adsr.Setting
	FilterEnv   adsr.Setting
	AmpEnv      adsr.Setting
}

func DefaultConfig() Config {
	return Config{
		NumChannels: psalm.Stereo,
		Unison:      1,
		Detune:      20,
		Spread:      0.7,
		Cutoff:      8000,
		Resonance:   0.2,
		KeyTrack:    0.5,
		Level:       0.6,
		Normalize:   true,
		PositionEnv: *adsr.NewSetting(1.0, 5.0, 0.0, 500.0, 1.0, 300.0),
		FilterEnv:   *adsr.NewSetting(1.0, 5.0, 0.3, 300.0, 1.0, 300.0),
		AmpEnv:      *adsr.NewSetting(1.0, 5.0, 0.8, 300.0, 1.0, 300.0),
	}
}

const middleC = 261.6255653005986

// basic is the table played when the config has none
var basic = Basic()

// The destinations of the matrix (see Dests), by number
const (
	dPosition = iota
	dCutoff
	dResonance
	dPitch
	dDetune
	dSpread
	dSubLevel
	dLevel
	dWarp
	dOscBLevel
	dOscBPosition
	dOscBDetune
	dLFO1Rate
	dLFO2Rate
	numDests
)

// Dests are the settings the matrix of a voice moves: by their keys (the pitch is the pitch of the
// voice, not the message), with how far a full source moves them (octaves for the cutoff, semitones
// for the pitch, cents for the detunes, the setting itself for the rest)
var Dests = []modmatrix.Dest{
	dPosition:     {Key: "position", Label: "position", Span: 1},
	dCutoff:       {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance:    {Key: "resonance", Label: "resonance", Span: 1},
	dPitch:        {Key: "pitch", Label: "pitch", Span: 24},
	dDetune:       {Key: "detune", Label: "detune", Span: 100},
	dSpread:       {Key: "spread", Label: "spread", Span: 1},
	dSubLevel:     {Key: "subLevel", Label: "sub", Span: 1},
	dLevel:        {Key: "level", Label: "level", Span: 1},
	dWarp:         {Key: "warp", Label: "warp", Span: 1},
	dOscBLevel:    {Key: "oscBLevel", Label: "level b", Span: 1},
	dOscBPosition: {Key: "oscBPosition", Label: "position b", Span: 1},
	dOscBDetune:   {Key: "oscBDetune", Label: "detune b", Span: 100},
	dLFO1Rate:     {Key: "lfo1Rate", Label: "lfo 1 rate", Span: 3},
	dLFO2Rate:     {Key: "lfo2Rate", Label: "lfo 2 rate", Span: 3},
}

// The value inputs of a voice: 1 to 3 set the position, the cutoff and the resonance, 5 to 8 the
// values of the matrix (Ext 1 to 4) that a slot can move any setting by
const (
	InPosition  = 1
	InCutoff    = 2
	InResonance = 3
	InExt1      = 5 // to InExt1+3
)

// Voice is a wavetable synth voice. Value inputs: 1 position, 2 cutoff, 3 resonance, 5-8 ext 1-4
// of the matrix. They glide (the position over a few milliseconds, the filter over a buffer), so
// modulating them never clicks
type Voice struct {
	*psalm.Module
	cfg         Config
	ampEnv      *adsr.ADSR
	filterEnv   *adsr.ADSR
	positionEnv *adsr.ADSR
	filters     [2]*svf.SVF
	overL       *ovs.Oversampler // the oscillators played several times a sample (Config.Oversample), nil for once
	overR       *ovs.Oversampler

	frequency float64               // target
	expr      expression.Expression // the pitch bend; the mod wheel moves through the table and opens the filter
	glideFreq float64
	phases    [MaxUnison]float64
	subPhase  float64
	position  float64 // smoothed cfg.Position
	cutoff    float64 // smoothed cfg.Cutoff
	res       float64 // smoothed cfg.Resonance
	vibPhase  float64
	lfoPhase  float64
	phaseB    float64
	phasesB   [MaxUnison]float64 // the unison oscillators of B after the first (phaseB)
	warpPhase float64
	lfo1Phase float64
	lfo2Phase float64
	mods      *modmatrix.Matrix
	velocity  float64 // the amplitude of the note
	random    float64 // a random value for the note, -1 to 1
	count     int     // samples until the next filter coefficient update
	waitSamps int
}

func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}

	v := &Voice{
		Module:    psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:       cfg,
		frequency: middleC,
		glideFreq: middleC,
		position:  cfg.Position,
		cutoff:    cfg.Cutoff,
		res:       cfg.Resonance,
	}

	sr := v.Config().SampleRate
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)
	v.filterEnv = adsr.New(&v.cfg.FilterEnv, adsr.NoteOff, sr)
	v.positionEnv = adsr.New(&v.cfg.PositionEnv, adsr.NoteOff, sr)
	for c := range v.filters {
		v.filters[c] = svf.New(cfg.Cutoff, cfg.Resonance, sr)
	}
	v.setOversample()
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

func (v *Voice) SetFrequency(frequency float64) {
	v.frequency = frequency
	if v.cfg.Glide <= 0 {
		v.glideFreq = frequency
	}
}

func (v *Voice) SetPitch(pitch float64) {
	v.SetFrequency(440.0 * math.Pow(2.0, (pitch-69.0)/12.0))
}

// setOversample makes (or drops) the samplers for the oversampling of the config
func (v *Voice) setOversample() {
	factor := 1 << max(0, min(v.cfg.Oversample, 2))
	if factor == 1 {
		v.overL, v.overR = nil, nil
		return
	}
	if v.overL == nil || v.overL.Factor != factor {
		v.overL = ovs.New(factor, ovs.TapsPerPhase)
		v.overR = ovs.New(factor, ovs.TapsPerPhase)
	}
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
	envelope.Receive(msg, "amp", &v.cfg.AmpEnv)
	envelope.Receive(msg, "filter", &v.cfg.FilterEnv)
	envelope.Receive(msg, "position", &v.cfg.PositionEnv)
	cfg := &v.cfg
	if t, ok := msg["table"].(*Table); ok {
		cfg.Table = t
	}
	msg.IfFltVal("position", func(f float64) { cfg.Position = f })
	msg.IfFltVal("positionEnvAmount", func(f float64) { cfg.PositionEnvAmount = f })
	msg.IfFltVal("positionLFORate", func(f float64) { cfg.PositionLFORate = f })
	msg.IfFltVal("positionLFODepth", func(f float64) { cfg.PositionLFODepth = f })
	msg.IfIntVal("unison", func(n int) { cfg.Unison = n })
	msg.IfFltVal("detune", func(f float64) { cfg.Detune = f })
	msg.IfFltVal("spread", func(f float64) { cfg.Spread = f })
	msg.IfFltVal("subLevel", func(f float64) { cfg.SubLevel = f })
	msg.IfFltVal("warp", func(f float64) { cfg.Warp = f })
	msg.IfIntVal("oversample", func(n int) { cfg.Oversample = max(0, min(n, 2)); v.setOversample() })
	msg.IfIntVal("warpMode", func(m int) { cfg.WarpMode = WarpMode(max(0, min(m, int(numWarpModes)-1))) })
	msg.IfFltVal("oscBLevel", func(f float64) { cfg.OscBLevel = f })
	msg.IfFltVal("oscBPosition", func(f float64) { cfg.OscBPosition = f })
	msg.IfIntVal("oscBOctave", func(n int) { cfg.OscBOctave = n })
	msg.IfFltVal("oscBSemi", func(f float64) { cfg.OscBSemi = f })
	msg.IfFltVal("oscBDetune", func(f float64) { cfg.OscBDetune = f })
	msg.IfIntVal("oscBUnison", func(n int) { cfg.OscBUnison = max(1, min(n, MaxUnison)) })
	msg.IfFltVal("oscBSpread", func(f float64) { cfg.OscBSpread = max(0, f) })
	msg.IfFltVal("lfo1Rate", func(f float64) { cfg.LFO1Rate = f })
	msg.IfFltVal("lfo2Rate", func(f float64) { cfg.LFO2Rate = f })
	msg.IfFltVal("normalize", func(f float64) { cfg.Normalize = f >= 0.5 })
	msg.IfIntVal("filter", func(m int) { cfg.Filter = FilterMode(m) })
	msg.IfFltVal("cutoff", func(f float64) { cfg.Cutoff = f })
	msg.IfFltVal("resonance", func(f float64) { cfg.Resonance = f })
	msg.IfFltVal("filterEnvAmount", func(f float64) { cfg.FilterEnvAmount = f })
	msg.IfFltVal("keyTrack", func(f float64) { cfg.KeyTrack = f })
	msg.IfFltVal("vibratoRate", func(f float64) { cfg.VibratoRate = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { cfg.VibratoDepth = f })
	msg.IfFltVal("glide", func(f float64) { cfg.Glide = f })
	msg.IfFltVal("level", func(f float64) { cfg.Level = f })
	msg.IfFltVal("frequency", v.SetFrequency)
	msg.IfFltVal("pitch", v.SetPitch)
}

func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	f := psalm.FltVal[float64](value)
	switch index {
	case InPosition:
		v.cfg.Position = f
	case InCutoff:
		v.cfg.Cutoff = f
	case InResonance:
		v.cfg.Resonance = f
	case InExt1, InExt1 + 1, InExt1 + 2, InExt1 + 3:
		v.mods.SetExt(index-InExt1, f)
		v.cfg.Ext[index-InExt1] = v.mods.Ext(index - InExt1)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.NoteOff, 0.0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.Duration, duration, amplitude, message, tick)
}

func (v *Voice) note(mode adsr.ReleaseMode, duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	wasActive := v.IsActive()
	v.ReceiveMessage(message)
	v.waitSamps = tick.SampleOffset
	v.velocity = amplitude
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}

	if !wasActive {
		if v.overL != nil {
			v.overL.Reset() // what is left in the filters of the note before
			v.overR.Reset()
		}
		// unison oscillators start at random phases so they don't start out as one loud wave
		for u := range v.phases {
			v.phases[u] = 0
			if u > 0 {
				v.phases[u] = v.Rand().Float64()
			}
		}
		v.subPhase = 0
		v.vibPhase = 0
		v.phaseB, v.warpPhase = 0, 0
		if v.cfg.OscBUnison > 1 {
			for u := 1; u < MaxUnison; u++ {
				v.phasesB[u] = v.Rand().Float64()
			}
		}
		v.lfo1Phase, v.lfo2Phase = 0, 0
		v.position = v.cfg.Position
		v.cutoff = v.cfg.Cutoff
		v.res = v.cfg.Resonance
		for _, f := range v.filters {
			f.Clear()
		}
	}

	ms := duration * 1000.0
	v.ampEnv.TriggerFull(ms, amplitude, &v.cfg.AmpEnv, mode)
	v.filterEnv.TriggerFull(ms, 1.0, &v.cfg.FilterEnv, mode)
	v.positionEnv.TriggerFull(ms, 1.0, &v.cfg.PositionEnv, mode)
}

func (v *Voice) NoteOff() {
	v.ampEnv.Release()
	v.filterEnv.Release()
	v.positionEnv.Release()
}

func (v *Voice) Clear() {
	v.ampEnv.Clear()
	v.filterEnv.Clear()
	v.positionEnv.Clear()
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return !v.ampEnv.Done()
}

// filterUpdate is the interval in samples at which the filter coefficients are recalculated
const filterUpdate = 4

// positionSmoothing is the time in milliseconds the position takes to follow a change
const positionSmoothing = 5.0

// warpSlope is the most a warp steepens the phase: the table is read at this times the pitch, so
// the band limited level is picked for the harmonics the warp moves up
func warpSlope(mode WarpMode, a float64) float64 {
	switch mode {
	case WarpBendUp, WarpBendDown:
		return 1 + 3*a
	case WarpSync:
		return 1 + 7*a
	case WarpMirror:
		return 1 + a
	case WarpFMB, WarpFMSine:
		return 1 + 4*a
	}
	return 1
}

// warp bends a phase (0-1) by a (0-1); mod (-1 to 1) is what modulates the phase in the FM modes
func warp(mode WarpMode, a, p, mod float64) float64 {
	switch mode {
	case WarpBendUp:
		return math.Pow(p, 1+3*a)
	case WarpBendDown:
		return 1 - math.Pow(1-p, 1+3*a)
	case WarpSync:
		q := p * (1 + 7*a)
		return q - math.Floor(q)
	case WarpMirror:
		return p + (1-math.Abs(2*p-1)-p)*a
	case WarpFMB, WarpFMSine:
		q := p + 0.75*a*mod
		return q - math.Floor(q)
	}
	return p
}

// unison fills the detune ratios and the gains of n unison oscillators, spread evenly: the sum
// keeps the level of one oscillator
func unison(n int, detune, spread float64, stereo bool, ratios, gainL, gainR *[MaxUnison]float64) {
	norm := 1.0 / math.Sqrt(float64(n))
	for u := range n {
		offset := 0.0
		if n > 1 {
			offset = float64(u)/float64(n-1) - 0.5
		}
		ratios[u] = math.Exp2(offset * detune / 1200.0)
		pan := 0.5 + offset*spread
		gainL[u] = math.Cos(pan*math.Pi/2) * math.Sqrt2 * norm
		gainR[u] = math.Sin(pan*math.Pi/2) * math.Sqrt2 * norm
		if !stereo {
			gainL[u] = norm
		}
	}
}

func (v *Voice) Synthesize() {
	outSocket := v.Output(0)
	outSocket.Clear()

	if v.ampEnv.Done() {
		return
	}

	var (
		cfg     = &v.cfg
		sr      = v.Config().SampleRate
		outs    = outSocket.Channels
		bufSize = v.Config().BufferSize
		table   = cfg.Table
		level   = cfg.Level
		stereo  = len(outs) > 1
		mods    = v.mods
		modded  = mods.Active()
	)
	if table == nil {
		table = basic
	}
	if level <= 0 {
		level = 1
	}

	n := max(1, min(cfg.Unison, MaxUnison))
	var ratios, gainL, gainR [MaxUnison]float64
	unison(n, cfg.Detune, cfg.Spread, stereo, &ratios, &gainL, &gainR)
	unisonModded := modded && (mods.Targets(dDetune) || mods.Targets(dSpread))

	// oscillator B, and the warp: only worked out when they are on
	baseRatioB := math.Exp2(float64(cfg.OscBOctave) + cfg.OscBSemi/12)
	nB := max(1, min(cfg.OscBUnison, MaxUnison))
	var ratiosB, gainBL, gainBR [MaxUnison]float64
	if nB > 1 {
		spreadB := cfg.OscBSpread
		if spreadB <= 0 {
			spreadB = cfg.Detune
		}
		unison(nB, spreadB, cfg.Spread, stereo, &ratiosB, &gainBL, &gainBR)
	}
	bModded := modded && (mods.Targets(dOscBLevel) || mods.Targets(dOscBDetune) || mods.Targets(dOscBPosition))
	warpModded := modded && mods.Targets(dWarp)
	lfo1Active := mods.Uses(modmatrix.LFO1) || mods.Targets(dLFO1Rate)
	lfo2Active := mods.Uses(modmatrix.LFO2) || mods.Targets(dLFO2Rate)
	lfo1Rate, lfo2Rate := orOneF(cfg.LFO1Rate), orOneF(cfg.LFO2Rate)
	if modded {
		mods.Values[modmatrix.Velocity] = v.velocity
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = max(-1, min(math.Log2(v.glideFreq/middleC)/5, 1))
	}

	glideCoef := 1.0
	if cfg.Glide > 0 {
		glideCoef = 1.0 - math.Exp(-1.0/(cfg.Glide*0.001*sr))
	}
	posCoef := 1.0 - math.Exp(-1.0/(positionSmoothing*0.001*sr))

	// the filter settings glide over the buffer
	rem := float64(bufSize - v.waitSamps)
	cutoffInc := (cfg.Cutoff - v.cutoff) / rem
	resInc := (cfg.Resonance - v.res) / rem
	keyTrack := math.Pow(v.glideFreq/middleC, cfg.KeyTrack)

	for i := v.waitSamps; i < bufSize; i++ {
		amp := v.ampEnv.Generate()
		fenv := v.filterEnv.Generate()
		penv := v.positionEnv.Generate()

		// the matrix: the sources of this sample, and what each destination is moved by
		if modded {
			var l1, l2 float64
			if lfo1Active {
				l1 = sine.Cycle(v.lfo1Phase)
				v.lfo1Phase += lfo1Rate * math.Exp2(mods.Mod(dLFO1Rate)) / sr
				v.lfo1Phase -= math.Floor(v.lfo1Phase)
			}
			if lfo2Active {
				l2 = sine.Cycle(v.lfo2Phase)
				v.lfo2Phase += lfo2Rate * math.Exp2(mods.Mod(dLFO2Rate)) / sr
				v.lfo2Phase -= math.Floor(v.lfo2Phase)
			}
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = fenv, amp
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = l1, l2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
		}

		// pitch with glide and vibrato, and the bend
		v.expr.Tick(sr)
		v.glideFreq += (v.frequency - v.glideFreq) * glideCoef
		freq := v.glideFreq * v.expr.Ratio()
		if cfg.VibratoDepth != 0 && cfg.VibratoRate > 0 {
			freq *= math.Exp2(cfg.VibratoDepth * sine.Cycle(v.vibPhase) / 12.0)
			v.vibPhase += cfg.VibratoRate / sr
			v.vibPhase -= math.Floor(v.vibPhase)
		}
		if modded {
			if m := mods.Mod(dPitch); m != 0 {
				freq *= math.Exp2(m / 12)
			}
		}

		// position: the smoothed setting plus envelope and LFO
		v.position += (cfg.Position - v.position) * posCoef
		pos := v.position + cfg.PositionEnvAmount*penv + 0.6*v.expr.Wheel() + mods.Mod(dPosition)
		if cfg.PositionLFODepth != 0 && cfg.PositionLFORate > 0 {
			pos += cfg.PositionLFODepth * sine.Cycle(v.lfoPhase)
			v.lfoPhase += cfg.PositionLFORate / sr
			v.lfoPhase -= math.Floor(v.lfoPhase)
		}

		if unisonModded {
			unison(n, max(0, cfg.Detune+mods.Mod(dDetune)), max(0, min(cfg.Spread+mods.Mod(dSpread), 1)), stereo, &ratios, &gainL, &gainR)
		}

		// oscillator B (also the modulator of the FM warp)
		inc0 := freq / sr
		levelB := cfg.OscBLevel
		if bModded {
			levelB += mods.Mod(dOscBLevel)
		}
		wmode := cfg.WarpMode
		wamt := cfg.Warp
		if warpModded {
			wamt += mods.Mod(dWarp)
		}
		wamt = max(0, min(wamt, 1))
		if wamt == 0 {
			wmode = WarpOff
		}
		// the oscillators, played this many times a sample (Config.Oversample): the warps bend the phase and make corners and steps that fold back at the pitch itself
		factor := 1
		if v.overL != nil {
			factor = v.overL.Factor
		}
		inc := inc0 / float64(factor)
		var l, r float64
		for range factor {
			var oB, oBl, oBr float64
			l, r = 0, 0
			if levelB > 0 || (wmode == WarpFMB) {
				posB, centsB := cfg.OscBPosition, cfg.OscBDetune
				if bModded {
					posB += mods.Mod(dOscBPosition)
					centsB += mods.Mod(dOscBDetune)
				}
				incB := inc * baseRatioB * math.Exp2(centsB/1200)
				cb := table.cursor(posB, incB)
				gB := 1.0
				if cfg.Normalize {
					gB = cb.gain
				}
				if nB == 1 {
					pb := v.phaseB
					if wmode != WarpFMB && wmode != WarpFMSine {
						pb = warp(wmode, wamt, pb, 0)
						cb = table.cursor(posB, incB*warpSlope(wmode, wamt))
					}
					oB = cb.read(pb) * gB
					oBl, oBr = oB, oB
					v.phaseB += incB
					v.phaseB -= math.Floor(v.phaseB)
				} else {
					fm := wmode == WarpFMB || wmode == WarpFMSine
					slopeB := 1.0
					if !fm {
						slopeB = warpSlope(wmode, wamt)
					}
					cb = table.cursor(posB, incB*ratiosB[nB-1]*slopeB)
					if cfg.Normalize {
						gB = cb.gain
					}
					for u := range nB {
						p := &v.phaseB
						if u > 0 {
							p = &v.phasesB[u]
						}
						pb := *p
						if !fm {
							pb = warp(wmode, wamt, pb, 0)
						}
						x := cb.read(pb) * gB
						oBl += x * gainBL[u]
						oBr += x * gainBR[u]
						*p += incB * ratiosB[u]
						*p -= math.Floor(*p)
					}
					oB = (oBl + oBr) / 2
				}
			}

			// oscillator A, the highest unison oscillator picks the band limited level for all
			slope := warpSlope(wmode, wamt)
			c := table.cursor(pos, inc*ratios[n-1]*slope)
			frameGain := 1.0
			if cfg.Normalize {
				frameGain = c.gain
			}
			mod := 0.0
			switch wmode {
			case WarpFMB:
				mod = oB
			case WarpFMSine:
				mod = sine.Cycle(v.warpPhase)
			}
			for u := range n {
				p := v.phases[u]
				if wmode != WarpOff {
					p = warp(wmode, wamt, p, mod)
				}
				x := c.read(p) * frameGain
				l += x * gainL[u]
				r += x * gainR[u]
				v.phases[u] += inc * ratios[u]
				v.phases[u] -= math.Floor(v.phases[u])
			}
			v.warpPhase += inc
			v.warpPhase -= math.Floor(v.warpPhase)
			if levelB > 0 {
				l += oBl * levelB
				r += oBr * levelB
			}
			if sub := cfg.SubLevel + mods.Mod(dSubLevel); sub > 0 {
				sub *= sine.Cycle(v.subPhase)
				l += sub
				r += sub
			}
			v.subPhase += inc * 0.5
			v.subPhase -= math.Floor(v.subPhase)
			if factor > 1 {
				v.overL.Push(l)
				if stereo {
					v.overR.Push(r)
				}
			}
		}
		if factor > 1 {
			l = v.overL.Out()
			if stereo {
				r = v.overR.Out()
			}
		}

		// filter
		if cfg.Filter != FilterOff {
			v.cutoff += cutoffInc
			v.res += resInc
			if v.count == 0 {
				fc := v.cutoff * keyTrack * math.Exp2(cfg.FilterEnvAmount*fenv+1.5*v.expr.Wheel()+mods.Mod(dCutoff))
				res := v.res
				if modded {
					res = max(0, min(res+mods.Mod(dResonance), 1))
				}
				for _, f := range v.filters {
					f.Set(fc, res)
				}
			}
			v.count = (v.count + 1) % filterUpdate
			mode := svf.LowPass
			switch cfg.Filter {
			case BandPass:
				mode = svf.BandPass
			case HighPass:
				mode = svf.HighPass
			}
			l = v.filters[0].Process(l, mode)
			if stereo {
				r = v.filters[1].Process(r, mode)
			}
		}

		g := amp * max(0, level+mods.Mod(dLevel))
		outs[0][i] = float32(l * g)
		if stereo {
			outs[1][i] = float32(r * g)
		}
	}

	v.cutoff = cfg.Cutoff
	v.res = cfg.Resonance
	v.waitSamps = 0
}

func orOneF(f float64) float64 {
	if f <= 0 {
		return 1
	}
	return f
}

// NewPoly creates a polyphonic wavetable synth
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
