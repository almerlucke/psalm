package dx

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

// Voice plays a 6 operator Patch. Notes take an optional "velocity" (0-1, default 1) in their
// message, the note amplitude sets the volume. Value input 1 scales all modulators (brightness)
type Voice struct {
	*psalm.Module
	patch      Patch
	alg        Algorithm
	isCarrier  [NumOperators]bool
	modTargets [NumOperators][]int

	envs     [NumOperators]envelope
	pitchEnv envelope
	phase    [NumOperators]float64
	out      [NumOperators]float64
	fb       [2]float64

	// per note values
	opLevel  [NumOperators]float64 // amplitude from output level, keyboard scaling and velocity
	freqMult [NumOperators]float64 // ratio (and detune) for ratio operators
	fixed    [NumOperators]float64 // frequency for fixed operators, 0 for ratio operators
	note     float64               // MIDI note
	velocity float64
	outScale float64 // note amplitude divided by the summed level of the carriers

	// LFO
	lfoPhase float64
	lfoSH    float64
	lfoTime  float64

	modScale    float64 // brightness, value input 1
	curModScale float64 // glides to modScale over a buffer

	noteOffSamps int
	waitSamps    int

	// operator amplitudes are converted from the envelope levels every ampUpdate samples (an
	// exponential per operator per sample is costly) and glide linearly in between
	opAmp    [NumOperators]float64
	opAmpInc [NumOperators]float64
	ampCount int

	// DC blocker on the output: FM puts a sideband at 0 Hz when operator ratios coincide, the
	// original hardware removes that offset in its analog output stage
	dcX, dcY float64
	dcCoef   float64
}

func New(patch Patch, numChannels int) *Voice {
	if numChannels == 0 {
		numChannels = psalm.Mono
	}

	v := &Voice{
		Module:      psalm.NewModule(nil, []int{numChannels}),
		note:        60,
		velocity:    1.0,
		modScale:    1.0,
		curModScale: 1.0,
	}

	for i := range v.envs {
		v.envs[i].reset()
	}
	v.pitchEnv.reset()
	v.dcCoef = 1.0 - 2.0*math.Pi*10.0/v.Config().SampleRate // 10 Hz high pass
	v.SetPatch(patch)
	v.SetSelf(v)

	return v
}

// SetPatch changes the patch, it is used from the next note on
func (v *Voice) SetPatch(patch Patch) {
	v.patch = patch
	v.alg = Algorithms[max(0, min(patch.Algorithm, 31))]

	v.isCarrier = [NumOperators]bool{}
	for _, c := range v.alg.Carriers {
		v.isCarrier[c] = true
	}

	v.modTargets = [NumOperators][]int{}
	for _, mod := range v.alg.Mods {
		v.modTargets[mod[1]] = append(v.modTargets[mod[1]], mod[0])
	}
}

func (v *Voice) Patch() Patch {
	return v.patch
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("velocity", func(f float64) { v.velocity = math.Max(0.0, math.Min(f, 1.0)) })
	msg.IfFltVal("modScale", func(f float64) { v.modScale = f })
	msg.IfFltVal("pitch", func(f float64) { v.note = f })
	msg.IfFltVal("frequency", func(f float64) { v.note = 69.0 + 12.0*math.Log2(f/440.0) })
}

func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}
	if index == 1 {
		v.modScale = psalm.FltVal[float64](value)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.start(0.0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.start(duration, amplitude, message, tick)
}

func (v *Voice) start(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	sr := v.Config().SampleRate
	wasActive := v.IsActive()

	v.velocity = 1.0
	v.ReceiveMessage(message)

	v.waitSamps = tick.SampleOffset
	v.noteOffSamps = 0
	if duration > 0.0 {
		v.noteOffSamps = max(1, int(duration*sr))
	}

	p := &v.patch
	note := v.note + float64(p.Transpose-24)

	for i := range v.patch.Ops {
		op := &p.Ops[i]

		// static level: output level, keyboard level scaling and velocity
		level := float64(op.OutputLevel) + keyboardScaling(op, note) - float64(op.VelocitySens)*(1.0-v.velocity)*6.0
		v.opLevel[i] = levelToAmp(math.Min(level, 99.0))

		// frequency
		detune := math.Pow(2.0, float64(op.Detune-7)*1.5/1200.0)
		if op.FixedFreq {
			v.fixed[i] = math.Pow(10.0, float64(op.Coarse&3)) * math.Pow(10.0, float64(op.Fine)/100.0) * detune
		} else {
			ratio := float64(op.Coarse)
			if op.Coarse == 0 {
				ratio = 0.5
			}
			v.fixed[i] = 0.0
			v.freqMult[i] = ratio * (1.0 + float64(op.Fine)/100.0) * detune
		}

		// envelopes, rate scaling makes higher notes faster
		rateAdd := float64(op.RateScaling) * math.Max(0.0, note-21.0) / 24.0
		v.envs[i].keyOn(op.EGRates, op.EGLevels, rateAdd, sr)
	}

	v.pitchEnv.keyOn(p.PitchEGRates, p.PitchEGLevels, 0.0, sr)

	// carriers add up, scale by their summed level so several loud carriers don't clip, while a
	// single carrier (or carriers that are turned down) keeps its full level
	carrierSum := 0.0
	for _, c := range v.alg.Carriers {
		carrierSum += v.opLevel[c]
	}
	v.outScale = amplitude / math.Max(1.0, carrierSum)

	if p.OscKeySync || !wasActive {
		v.phase = [NumOperators]float64{}
		v.fb = [2]float64{}
	}
	if p.LFOKeySync || !wasActive {
		v.lfoPhase = 0.0
		v.lfoTime = 0.0
		v.lfoSH = v.Rand().Float64()*2.0 - 1.0
	}
	if !wasActive {
		v.curModScale = v.modScale
		v.dcX, v.dcY = 0.0, 0.0
		for i := range v.envs {
			v.opAmp[i] = levelToAmp(v.envs[i].level) * v.opLevel[i]
		}
		v.ampCount = 0
	}

}

func (v *Voice) NoteOff() {
	for i := range v.envs {
		v.envs[i].keyOff()
	}
	v.pitchEnv.keyOff()
}

func (v *Voice) Clear() {
	for i := range v.envs {
		v.envs[i].reset()
	}
	v.opAmp = [NumOperators]float64{}
	v.ampCount = 0
	v.pitchEnv.reset()
	v.Output(0).Clear()
}

// IsActive is true while any carrier envelope is running
func (v *Voice) IsActive() bool {
	for i := range v.envs {
		if v.isCarrier[i] && !v.envs[i].idle() {
			return true
		}
	}
	return false
}

// lfo returns the LFO value (-1 to 1) for the current phase
func (v *Voice) lfo() float64 {
	ph := v.lfoPhase
	switch v.patch.LFOWaveform {
	case 0: // triangle
		return 1.0 - 4.0*math.Abs(ph-0.5)
	case 1: // saw down
		return 1.0 - 2.0*ph
	case 2: // saw up
		return 2.0*ph - 1.0
	case 3: // square
		if ph < 0.5 {
			return 1.0
		}
		return -1.0
	case 5: // sample & hold
		return v.lfoSH
	default: // sine
		return math.Sin(2.0 * math.Pi * ph)
	}
}

func (v *Voice) Synthesize() {
	outSocket := v.Output(0)
	outSocket.Clear()

	if !v.IsActive() {
		return
	}

	var (
		p        = &v.patch
		sr       = v.Config().SampleRate
		outs     = outSocket.Channels
		bufSize  = v.Config().BufferSize
		twoPi    = 2.0 * math.Pi
		note     = v.note + float64(p.Transpose-24)
		baseFreq = 440.0 * math.Pow(2.0, (note-69.0)/12.0)
		fbAmount = feedbackAmount(p.Feedback)
		lfoInc   = lfoSpeedToHz(p.LFOSpeed) / sr
		lfoDelay = lfoDelayToSeconds(p.LFODelay)
		lfoRamp  = math.Max(0.05, lfoDelay*0.5)
		pmDepth  = float64(p.LFOPitchDepth) / 99.0 * pitchModSemitones[max(0, min(p.PitchModSens, 7))]
		amDepth  = float64(p.LFOAmpDepth) / 99.0
		modInc   = (v.modScale - v.curModScale) / float64(bufSize-v.waitSamps)
	)

	for i := v.waitSamps; i < bufSize; i++ {
		if v.noteOffSamps > 0 {
			v.noteOffSamps--
			if v.noteOffSamps == 0 {
				v.NoteOff()
			}
		}

		v.curModScale += modInc

		// LFO with delay and fade in
		lfo := v.lfo()
		fade := math.Max(0.0, math.Min(1.0, (v.lfoTime-lfoDelay)/lfoRamp))
		lfo *= fade
		v.lfoTime += 1.0 / sr
		v.lfoPhase += lfoInc
		if v.lfoPhase >= 1.0 {
			v.lfoPhase -= 1.0
			v.lfoSH = v.Rand().Float64()*2.0 - 1.0
		}

		// pitch from the pitch envelope and LFO pitch modulation
		semis := pitchLevelToSemitones(v.pitchEnv.next()) + lfo*pmDepth
		freq := baseFreq
		if semis != 0.0 {
			freq *= math.Exp2(semis / 12.0)
		}

		// operators from 6 down to 1, modulators always have a higher number than their targets
		sample := 0.0
		for op := NumOperators - 1; op >= 0; op-- {
			pm := 0.0
			for _, m := range v.modTargets[op] {
				pm += v.out[m]
			}
			if op == v.alg.FeedbackTo && fbAmount > 0.0 {
				pm += (v.fb[0] + v.fb[1]) * 0.5 * fbAmount
			}

			s := math.Sin(twoPi * (v.phase[op] + pm))
			level := v.envs[op].next()
			if v.ampCount == 0 {
				v.opAmpInc[op] = (levelToAmp(level)*v.opLevel[op] - v.opAmp[op]) / ampUpdate
			}
			v.opAmp[op] += v.opAmpInc[op]
			amp := v.opAmp[op]

			if ams := p.Ops[op].AmpModSens; ams > 0 && amDepth > 0.0 {
				amp *= 1.0 - amDepth*ampModDepth[ams]*(0.5+0.5*lfo)
			}

			if op == v.alg.FeedbackFrom {
				v.fb[1], v.fb[0] = v.fb[0], s*amp
			}

			if v.isCarrier[op] {
				v.out[op] = s * amp
				sample += v.out[op]
			} else {
				v.out[op] = s * amp * modIndex * v.curModScale
			}

			f := v.fixed[op]
			if f == 0.0 {
				f = freq * v.freqMult[op]
			}
			v.phase[op] += f / sr
			v.phase[op] -= math.Floor(v.phase[op])
		}

		v.ampCount = (v.ampCount + 1) % ampUpdate

		// remove the DC offset
		x := sample * v.outScale
		v.dcY = x - v.dcX + v.dcCoef*v.dcY
		v.dcX = x

		val := float32(v.dcY)
		for _, ch := range outs {
			ch[i] = val
		}
	}

	v.curModScale = v.modScale
	v.waitSamps = 0
}

// ampUpdate is the interval in samples at which operator amplitudes are recalculated
const ampUpdate = 4

// NewPoly creates a polyphonic synth playing the patch
func NewPoly(patch Patch, numVoices int, numChannels int) *poly.Poly[*Voice] {
	if numChannels == 0 {
		numChannels = psalm.Stereo
	}

	voices := make([]*Voice, numVoices)
	for i := range voices {
		voices[i] = New(patch, numChannels)
	}

	return poly.New[*Voice](numChannels, voices)
}

// SetPatch changes the patch of all voices of a poly synth
func SetPatch(synth *poly.Poly[*Voice], patch Patch) {
	synth.CallVoices(func(v *Voice) { v.SetPatch(patch) })
}
