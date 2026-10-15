package fm

import (
	"fmt"
	"github.com/almerlucke/psalm/blocks/sine"
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

// OperatorConfig configures one sine operator
type OperatorConfig struct {
	Ratio        float64 // frequency ratio to the note frequency (0 is treated as 1)
	Fixed        float64 // fixed frequency in Hz, used instead of the ratio when > 0
	Detune       float64 // cents
	Level        float64 // carriers: output level, modulators: modulation index (1 is 2*pi radians)
	VelocitySens float64 // 0-1, how much the note velocity scales the level (brighter when played harder)
	Env          adsr.Setting
}

// Config describes a 4 operator FM voice
type Config struct {
	NumChannels  int
	Algorithm    int // index in Algorithms
	Ops          [NumOperators]OperatorConfig
	Feedback     float64            // 0-1, operator 4 modulates itself (towards a saw, then noise)
	ModDepth     float64            // scales the level of all modulators (brightness), 0 is treated as 1
	Glide        float64            // portamento in milliseconds from the previous note of the voice
	VibratoRate  float64            // Hz
	VibratoDepth float64            // semitones
	VibratoDelay float64            // fade in time in milliseconds after note on
	Level        float64            // output level (0 is treated as 1)
	Filter       voicefilter.Config // a filter after the operators (off unless its Type is set)

	// the matrix: a source to a destination (Dests) over an amount; the two LFOs it can use
	Mods     [modmatrix.Slots]modmatrix.Slot
	Ext      [modmatrix.Exts]float64 // the values of Ext 1 to 4 (cabled in on InExt1...)
	LFO1Rate float64                 // Hz (0 is still)
	LFO2Rate float64
}

// The destinations of the matrix (see Dests), by number
const (
	dModDepth = iota
	dFeedback
	dOp1Level // to dOp1Level+3: the levels of the operators (the modulation index of a modulator)
	_
	_
	_
	dPitch
	dLevel
	dCutoff
	dResonance
	dLFO1Rate
	dLFO2Rate
)

// Dests are the settings the matrix of a voice moves, by their keys, with how far a full source moves them
var Dests = []modmatrix.Dest{
	dModDepth:     {Key: "modDepth", Label: "mod depth", Span: 2},
	dFeedback:     {Key: "feedback", Label: "feedback", Span: 1},
	dOp1Level:     {Key: "op1Level", Label: "op 1 level", Span: 1},
	dOp1Level + 1: {Key: "op2Level", Label: "op 2 level", Span: 1},
	dOp1Level + 2: {Key: "op3Level", Label: "op 3 level", Span: 1},
	dOp1Level + 3: {Key: "op4Level", Label: "op 4 level", Span: 1},
	dPitch:        {Key: "pitch", Label: "pitch", Span: 24},
	dLevel:        {Key: "level", Label: "level", Span: 1},
	dCutoff:       {Key: "cutoff", Label: "cutoff (filter)", Span: 6},
	dResonance:    {Key: "resonance", Label: "resonance (filter)", Span: 1},
	dLFO1Rate:     {Key: "lfo1Rate", Label: "lfo 1 rate", Span: 3},
	dLFO2Rate:     {Key: "lfo2Rate", Label: "lfo 2 rate", Span: 3},
}

// The value inputs of a voice: 1 the mod depth, 2 the feedback; 5 to 8 the values of the matrix (Ext 1 to 4)
const InExt1 = 5

func DefaultConfig() Config {
	cfg := Config{
		NumChannels: psalm.Stereo,
		Algorithm:   0,
		ModDepth:    1.0,
		Level:       1.0,
	}

	for i := range cfg.Ops {
		cfg.Ops[i] = OperatorConfig{
			Ratio: 1.0,
			Env:   *adsr.NewSetting(1.0, 5.0, 0.7, 300.0, 1.0, 300.0),
		}
	}

	// a simple two operator tone: operator 2 modulates operator 1
	cfg.Ops[0].Level = 1.0
	cfg.Ops[1].Level = 0.4

	return cfg
}

// Voice is a 4 operator FM synth voice (phase modulation of sine operators), see Algorithms for
// the routings. Value inputs: 1 mod depth (brightness), 2 feedback
type Voice struct {
	*psalm.Module
	cfg        Config
	envs       [NumOperators]*adsr.ADSR
	phase      [NumOperators]float64
	out        [NumOperators]float64
	fb         [2]float64
	isCarrier  [NumOperators]bool
	modTargets [NumOperators][]int // modulators per target operator
	frequency  float64
	expr       expression.Expression // the pitch bend; the mod wheel brightens (more modulation), or brings in a vibrato when no operator modulates
	glideFreq  float64
	velocity   float64
	vibPhase   float64
	vibFade    float64
	vibFadeInc float64
	waitSamps  int
	filter     *voicefilter.Filter
	mods       *modmatrix.Matrix
	random     float64 // for the note, -1 to 1
	lfoPhase   [2]float64
	ampLast    float64 // the carriers' envelopes at the last sample

	// operator levels and feedback glide over a buffer to their new values, so changing them (for
	// example the mod depth from an LFO) doesn't step and click
	curLevels   [NumOperators]float64
	curFeedback float64
	levelsSet   bool
}

const middleC = 261.6255653005986

func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}

	v := &Voice{
		Module:    psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:       cfg,
		frequency: middleC,
		glideFreq: middleC,
		velocity:  1.0,
	}

	sr := v.Config().SampleRate
	for i := range v.envs {
		v.envs[i] = adsr.New(&v.cfg.Ops[i].Env, adsr.NoteOff, sr)
	}

	v.filter = voicefilter.New(cfg.Filter, sr, cfg.NumChannels)
	v.mods = modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		v.mods.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		v.mods.SetExt(i, e)
	}
	v.setAlgorithm(cfg.Algorithm)
	v.SetSelf(v)

	return v
}

func (v *Voice) setAlgorithm(index int) {
	index = max(0, min(index, len(Algorithms)-1))
	v.cfg.Algorithm = index
	alg := Algorithms[index]

	v.isCarrier = [NumOperators]bool{}
	for _, c := range alg.Carriers {
		v.isCarrier[c] = true
	}

	v.modTargets = [NumOperators][]int{}
	for _, m := range alg.Mods {
		v.modTargets[m[1]] = append(v.modTargets[m[1]], m[0])
	}
}

// SetFrequency sets the target frequency, the voice glides there when Glide is set
func (v *Voice) SetFrequency(frequency float64) {
	v.frequency = frequency
	if v.cfg.Glide <= 0.0 {
		v.glideFreq = frequency
	}
}

func (v *Voice) SetPitch(pitch float64) {
	v.SetFrequency(440.0 * math.Pow(2.0, (pitch-69.0)/12.0))
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
	msg.IfFltVal("lfo1Rate", func(f float64) { v.cfg.LFO1Rate = math.Max(0, f) })
	msg.IfFltVal("lfo2Rate", func(f float64) { v.cfg.LFO2Rate = math.Max(0, f) })
	// "amp" is the envelope of the carriers (the operators heard), "op1".."op4" each operator's
	for _, c := range Algorithms[v.cfg.Algorithm%len(Algorithms)].Carriers {
		envelope.Receive(msg, "amp", &v.cfg.Ops[c].Env)
	}
	msg.IfIntVal("algorithm", v.setAlgorithm)
	msg.IfFltVal("feedback", func(f float64) { v.cfg.Feedback = f })
	msg.IfFltVal("modDepth", func(f float64) { v.cfg.ModDepth = f })
	msg.IfFltVal("glide", func(f float64) { v.cfg.Glide = f })
	msg.IfFltVal("vibratoRate", func(f float64) { v.cfg.VibratoRate = f })
	msg.IfFltVal("vibratoDepth", func(f float64) { v.cfg.VibratoDepth = f })
	msg.IfFltVal("vibratoDelay", func(f float64) { v.cfg.VibratoDelay = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0.0, math.Min(f, 1.0)) })

	for i := range v.cfg.Ops {
		op := &v.cfg.Ops[i]
		prefix := fmt.Sprintf("op%d", i+1)
		envelope.Receive(msg, prefix, &op.Env)
		msg.IfFltVal(prefix+"Ratio", func(f float64) { op.Ratio = f })
		msg.IfFltVal(prefix+"Fixed", func(f float64) { op.Fixed = f })
		msg.IfFltVal(prefix+"Detune", func(f float64) { op.Detune = f })
		msg.IfFltVal(prefix+"Level", func(f float64) { op.Level = f })
		msg.IfFltVal(prefix+"VelocitySens", func(f float64) { op.VelocitySens = f })
	}

	msg.IfFltVal("frequency", v.SetFrequency)
	msg.IfFltVal("pitch", v.SetPitch)
}

func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}

	switch index {
	case 1:
		v.cfg.ModDepth = psalm.FltVal[float64](value)
	case 2:
		v.cfg.Feedback = psalm.FltVal[float64](value)
	case InExt1, InExt1 + 1, InExt1 + 2, InExt1 + 3:
		v.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
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

	v.velocity = 1.0
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}
	v.ReceiveMessage(message)

	v.waitSamps = tick.SampleOffset

	// a fresh note starts all operators in phase, so the attack sounds the same every time
	if !wasActive {
		v.phase = [NumOperators]float64{}
		v.fb = [2]float64{}
		v.levelsSet = false // start at the exact levels, nothing to glide from
	}

	v.vibFade = 1.0
	v.vibFadeInc = 0.0
	if v.cfg.VibratoDelay > 0.0 {
		v.vibFade = 0.0
		v.vibFadeInc = 1.0 / (v.cfg.VibratoDelay * 0.001 * v.Config().SampleRate)
	}

	v.filter.Trigger(duration*1000.0, mode)

	// carriers get the note amplitude, modulators shape the timbre
	for i, env := range v.envs {
		level := 1.0
		if v.isCarrier[i] {
			level = amplitude
		}
		env.TriggerFull(duration*1000.0, level, &v.cfg.Ops[i].Env, mode)
	}
}

func (v *Voice) NoteOff() {
	for _, env := range v.envs {
		env.Release()
	}
	v.filter.Release()
}

func (v *Voice) Clear() {
	for _, env := range v.envs {
		env.Clear()
	}
	v.filter.Clear()
	v.Output(0).Clear()
}

// IsActive is true while any carrier envelope is running
func (v *Voice) IsActive() bool {
	for i, env := range v.envs {
		if v.isCarrier[i] && !env.Done() {
			return true
		}
	}
	return false
}

func (v *Voice) Synthesize() {
	outSocket := v.Output(0)
	outSocket.Clear()

	if !v.IsActive() {
		return
	}

	var (
		cfg      = &v.cfg
		sr       = v.Config().SampleRate
		outs     = outSocket.Channels
		bufSize  = v.Config().BufferSize
		alg      = Algorithms[cfg.Algorithm]
		modDepth = cfg.ModDepth
		level    = cfg.Level
		levels   [NumOperators]float64
		levelInc [NumOperators]float64
		freqMult [NumOperators]float64
		fixed    [NumOperators]float64
	)

	if modDepth <= 0.0 {
		modDepth = 1.0
	}
	if level <= 0.0 {
		level = 1.0
	}

	for i, op := range cfg.Ops {
		vel := 1.0 - op.VelocitySens + op.VelocitySens*v.velocity
		levels[i] = op.Level * vel
		if !v.isCarrier[i] {
			levels[i] *= modDepth
		}

		detune := math.Pow(2.0, op.Detune/1200.0)
		if op.Fixed > 0.0 {
			fixed[i] = op.Fixed * detune
		} else {
			ratio := op.Ratio
			if ratio <= 0.0 {
				ratio = 1.0
			}
			freqMult[i] = ratio * detune
		}
	}

	// glide levels and feedback from their previous values over the buffer
	if !v.levelsSet {
		v.curLevels = levels
		v.curFeedback = cfg.Feedback
		v.levelsSet = true
	}
	n := float64(bufSize - v.waitSamps)
	for i := range levels {
		levelInc[i] = (levels[i] - v.curLevels[i]) / n
	}
	feedbackInc := (cfg.Feedback - v.curFeedback) / n

	glideCoef := 1.0
	if cfg.Glide > 0.0 {
		glideCoef = 1.0 - math.Exp(-1.0/(cfg.Glide*0.001*sr))
	}

	filtered := v.filter.On()
	mods := v.mods
	modded := mods.Active()
	if modded {
		mods.Values[modmatrix.Velocity] = v.velocity
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(v.glideFreq/middleC)/5, 1))
	}
	carrierScale := level / float64(len(alg.Carriers))
	hasMod := false // whether any operator modulates another (the wheel brightens), or all are carriers
	for op := range v.modTargets {
		if len(v.modTargets[op]) > 0 {
			hasMod = true
		}
	}

	for i := v.waitSamps; i < bufSize; i++ {
		// pitch with glide and vibrato, and the bend
		v.expr.Tick(sr)
		v.glideFreq += (v.frequency - v.glideFreq) * glideCoef
		freq := v.glideFreq
		if modded {
			mv := &mods.Values
			for n := range v.lfoPhase {
				rate := cfg.LFO1Rate
				if n == 1 {
					rate = cfg.LFO2Rate
				}
				rate *= math.Exp2(mods.Mod(dLFO1Rate + n))
				mv[modmatrix.LFO1+modmatrix.Source(n)] = sine.Cycle(v.lfoPhase[n])
				v.lfoPhase[n] += rate / sr
				v.lfoPhase[n] -= math.Floor(v.lfoPhase[n])
			}
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = v.filter.Env(), v.ampLast
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
			freq *= math.Exp2(mods.Mod(dPitch) / 12)
		}
		bright := math.Max(0, 1+1.5*v.expr.Wheel()+mods.Mod(dModDepth))
		if hasMod {
			freq *= v.expr.Ratio()
		} else {
			freq *= v.expr.VibratoRatio(sr, 5.5, 0.35) // nothing to brighten: the wheel brings in a vibrato
		}
		if cfg.VibratoDepth != 0.0 && cfg.VibratoRate > 0.0 {
			freq *= math.Pow(2.0, cfg.VibratoDepth*v.vibFade*sine.Cycle(v.vibPhase)/12.0)
			v.vibPhase += cfg.VibratoRate / sr
			v.vibPhase -= math.Floor(v.vibPhase)
			v.vibFade = math.Min(1.0, v.vibFade+v.vibFadeInc)
		}

		for op := range levels {
			v.curLevels[op] += levelInc[op]
		}
		v.curFeedback += feedbackInc
		feedback := math.Max(0, v.curFeedback+mods.Mod(dFeedback))

		// operators from 4 down to 1, modulators always have a higher number than their targets
		sample := 0.0
		ampNow := 0.0
		for op := NumOperators - 1; op >= 0; op-- {
			pm := 0.0
			for _, m := range v.modTargets[op] {
				pm += v.out[m]
			}
			pm *= bright
			if op == NumOperators-1 && feedback > 0.0 {
				// averaging the last two outputs keeps strong feedback from oscillating
				pm += (v.fb[0] + v.fb[1]) * 0.5 * feedback
			}

			s := sine.Cycle(v.phase[op] + pm)
			env := v.envs[op].Generate()

			if op == NumOperators-1 {
				v.fb[1], v.fb[0] = v.fb[0], s*env
			}

			v.out[op] = s * env * math.Max(0, v.curLevels[op]+mods.Mod(dOp1Level+op))

			f := fixed[op]
			if f == 0.0 {
				f = freq * freqMult[op]
			}
			v.phase[op] += f / sr
			v.phase[op] -= math.Floor(v.phase[op])

			if v.isCarrier[op] {
				sample += v.out[op]
				ampNow = math.Max(ampNow, env)
			}
		}

		v.ampLast = ampNow
		val := sample * carrierScale
		if modded {
			val *= math.Max(0, 1+mods.Mod(dLevel))
		}
		if filtered {
			v.filter.Tick(freq, 1.5*v.expr.Wheel()+mods.Mod(dCutoff), mods.Mod(dResonance))
			for c, ch := range outs {
				ch[i] = float32(v.filter.Process(c, val))
			}
		} else {
			for _, ch := range outs {
				ch[i] = float32(val)
			}
		}
	}

	// land exactly on the targets, no rounding drift
	v.curLevels = levels
	v.curFeedback = cfg.Feedback
	v.waitSamps = 0
}

// NewPoly creates a polyphonic FM synth
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
