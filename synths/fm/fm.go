package fm

import (
	"fmt"
	"github.com/almerlucke/psalm/blocks/sine"
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
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
	Feedback     float64 // 0-1, operator 4 modulates itself (towards a saw, then noise)
	ModDepth     float64 // scales the level of all modulators (brightness), 0 is treated as 1
	Glide        float64 // portamento in milliseconds from the previous note of the voice
	VibratoRate  float64 // Hz
	VibratoDepth float64 // semitones
	VibratoDelay float64 // fade in time in milliseconds after note on
	Level        float64 // output level (0 is treated as 1)
}

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
}

func (v *Voice) Clear() {
	for _, env := range v.envs {
		env.Clear()
	}
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
		bright := 1 + 1.5*v.expr.Wheel()
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

		// operators from 4 down to 1, modulators always have a higher number than their targets
		sample := 0.0
		for op := NumOperators - 1; op >= 0; op-- {
			pm := 0.0
			for _, m := range v.modTargets[op] {
				pm += v.out[m]
			}
			pm *= bright
			if op == NumOperators-1 && v.curFeedback > 0.0 {
				// averaging the last two outputs keeps strong feedback from oscillating
				pm += (v.fb[0] + v.fb[1]) * 0.5 * v.curFeedback
			}

			s := sine.Cycle(v.phase[op] + pm)
			env := v.envs[op].Generate()

			if op == NumOperators-1 {
				v.fb[1], v.fb[0] = v.fb[0], s*env
			}

			v.out[op] = s * env * v.curLevels[op]

			f := fixed[op]
			if f == 0.0 {
				f = freq * freqMult[op]
			}
			v.phase[op] += f / sr
			v.phase[op] -= math.Floor(v.phase[op])

			if v.isCarrier[op] {
				sample += v.out[op]
			}
		}

		val := float32(sample * carrierScale)
		for _, ch := range outs {
			ch[i] = val
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
