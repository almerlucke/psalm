package dx

import (
	"fmt"
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/sine"
	"math"

	"github.com/almerlucke/psalm"
	msgenv "github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/blocks/voicefilter"
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

	expr expression.Expression // the pitch bend; the mod wheel brings in a vibrato (as on a DX7)

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

	filter *voicefilter.Filter // after the operators, set by the patch (its Filter)

	// the free operators (Operator.FreeOn): an ADSR each, and their levels at this note
	freeEnv   [NumOperators]*adsr.ADSR
	freeLevel [NumOperators]float64
	free      bool    // a carrier is free: the output is its own, scaled by the number of carriers
	glideFreq float64 // the pitch glides to the note (Patch.Glide)
	vibPhase  float64
	vibFade   float64
	vibInc    float64

	mods    *modmatrix.Matrix
	random  float64 // for the note, -1 to 1
	lfo2    float64 // the phase of the second LFO
	ampLast float64 // the loudest carrier at the last sample

	over *oversample.Oversampler // the operators are played this many times a sample (Patch.Oversample), nil for once
}

func New(patch Patch, numChannels int) *Voice {
	if numChannels == 0 {
		numChannels = psalm.Stereo
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
	v.filter = voicefilter.New(voicefilter.Config{}, v.Config().SampleRate, numChannels)
	for i := range v.freeEnv {
		v.freeEnv[i] = adsr.New(&v.patch.Ops[i].Free.Env, adsr.NoteOff, v.Config().SampleRate)
	}
	v.glideFreq = 261.6255653005986
	v.mods = modmatrix.New(Dests)
	v.SetPatch(patch)
	v.SetSelf(v)

	return v
}

// SetPatch changes the patch, it is used from the next note on
func (v *Voice) SetPatch(patch Patch) {
	if patch.Routing != nil && patch.Routing.Check() != nil {
		patch.Routing = nil // one that can't play: the algorithm instead
	}
	v.patch = patch
	v.alg = patch.Routed()
	if factor := patch.OversampleFactor(); factor < 2 {
		v.over = nil
	} else if v.over == nil || v.over.Factor != factor {
		v.over = oversample.New(factor, oversample.TapsPerPhase)
	}
	v.filter.SetConfig(patch.filter().Config())
	for i, sl := range patch.Mods {
		v.mods.Set(i, sl)
	}
	for i, e := range patch.Ext {
		v.mods.SetExt(i, e)
	}

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

// ReceiveMessage takes the bend, the wheel and the pressure, the note, and every setting of the
// patch by its name (Patch.Set: "algorithm", "feedback", "op1Coarse", "op3Rate2", ...), a whole
// "patch" (a Patch) or a "routing" of your own (an Algorithm, or nil for the patch's algorithm).
// The settings are played from the next note
func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	changed := false
	if p, ok := msg["patch"].(Patch); ok {
		v.patch, changed = p, true
	}
	if r, ok := msg["routing"]; ok {
		switch a := r.(type) {
		case Algorithm:
			v.patch.Routing, changed = &a, true
		case *Algorithm:
			v.patch.Routing, changed = a, true
		case nil:
			v.patch.Routing, changed = nil, true
		}
	}
	for key, val := range msg {
		if f, ok := val.(float64); ok && v.patch.Set(key, f) {
			changed = true
		}
	}
	// the free operators and what goes with them, from the next note on: op1Ratio, op1Fixed
	// is the DX7's flag, so op1FixedHz, op1Cents, op1Gain (the level, or the index of a modulator),
	// modDepth, freeFeedback, glide, vibratoRate, vibratoDepth, vibratoDelay; fmAlgorithm (0-7) is
	// one of the four operator routings of the former FM synth
	for i := range v.patch.Ops {
		f, key := &v.patch.Ops[i].Free, fmt.Sprintf("op%d", i+1)
		msgenv.Receive(msg, key, &f.Env) // op1Attack, op1Decay, op1Sustain, op1Release: milliseconds
		msg.IfFltVal(key+"Ratio", func(x float64) { f.Ratio = x })
		msg.IfFltVal(key+"FixedHz", func(x float64) { f.Fixed = x })
		msg.IfFltVal(key+"Cents", func(x float64) { f.Detune = x })
		msg.IfFltVal(key+"Gain", func(x float64) { f.Level = x })
	}
	for _, c := range v.alg.Carriers { // "amp" is the envelope of the free carriers: the operators heard
		if v.patch.Ops[c].FreeOn {
			msgenv.Receive(msg, "amp", &v.patch.Ops[c].Free.Env)
		}
	}
	v.mods.Receive(msg)
	for i := range v.patch.Mods {
		v.patch.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.patch.Ext {
		v.patch.Ext[i] = v.mods.Ext(i)
	}
	msg.IfFltVal("lfo2Rate", func(x float64) { v.patch.LFO2Rate = math.Max(0, x) })
	msg.IfFltVal("modDepth", func(x float64) { v.patch.ModDepth = x })
	msg.IfFltVal("freeFeedback", func(x float64) { v.patch.FreeFeedback = x })
	msg.IfFltVal("glide", func(x float64) { v.patch.Glide = x })
	msg.IfFltVal("vibratoRate", func(x float64) { v.patch.VibratoRate = x })
	msg.IfFltVal("vibratoDepth", func(x float64) { v.patch.VibratoDepth = x })
	msg.IfFltVal("vibratoDelay", func(x float64) { v.patch.VibratoDelay = x })
	msg.IfIntVal("fmAlgorithm", func(n int) {
		route := fmAlgorithms[max(0, min(n, len(fmAlgorithms)-1))]
		route.FeedbackFrom, route.FeedbackTo = 3, 3
		v.patch.Routing, changed = &route, true
	})
	if changed {
		v.SetPatch(v.patch)
	}
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
	if index >= InExt1 && index < InExt1+modmatrix.Exts {
		v.mods.SetExt(index-InExt1, psalm.FltVal[float64](value))
		v.patch.Ext[index-InExt1] = v.mods.Ext(index - InExt1)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.start(0.0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.start(duration, amplitude, message, tick)
}

// CutsOnRestart: the operators start from zero again (poly fades out the note that was sounding)
func (v *Voice) CutsOnRestart() bool { return true }

func (v *Voice) start(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	sr := v.Config().SampleRate
	wasActive := v.IsActive()

	v.velocity = 1.0
	v.ReceiveMessage(message)

	if v.mods.Uses(modmatrix.Random) { // drawn only when used: a voice that doesn't keeps its random stream
		v.random = v.Rand().Float64()*2 - 1
	}
	v.waitSamps = tick.SampleOffset
	v.noteOffSamps = 0
	if v.over != nil && !wasActive {
		v.over.Reset() // what is left in the filter of the note before
	}
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

		// a free operator: its own frequency, level and envelope instead
		if op.FreeOn {
			f := &op.Free
			detune := math.Pow(2.0, f.Detune/1200.0)
			if f.Fixed > 0.0 {
				v.fixed[i] = f.Fixed * detune
			} else {
				v.fixed[i] = 0.0
				v.freqMult[i] = math.Max(f.Ratio, 0.0) * detune
				if f.Ratio <= 0.0 {
					v.freqMult[i] = detune
				}
			}
			level := f.Level * (1.0 - f.VelocitySens + f.VelocitySens*v.velocity)
			trigger := 1.0
			if v.isCarrier[i] {
				trigger = amplitude // carriers sound as loud as the note
			} else if p.ModDepth > 0.0 {
				level *= p.ModDepth
			}
			v.freeLevel[i] = level
			v.freeEnv[i].TriggerFull(0, trigger, &op.Free.Env, adsr.NoteOff)
		}
	}

	// pitch: portamento starts from the last note, a vibrato fades in
	if p.Glide <= 0.0 {
		v.glideFreq = 440.0 * math.Pow(2.0, (note-69.0)/12.0)
	}
	v.vibFade, v.vibInc = 1.0, 0.0
	if p.VibratoDelay > 0.0 {
		v.vibFade, v.vibInc = 0.0, 1.0/(p.VibratoDelay*0.001*sr)
	}

	v.pitchEnv.keyOn(p.PitchEGRates, p.PitchEGLevels, 0.0, sr)
	v.filter.Trigger(0, adsr.NoteOff)

	// carriers add up, scale by their summed level so several loud carriers don't clip, while a
	// single carrier (or carriers that are turned down) keeps its full level
	carrierSum := 0.0
	v.free = false
	for _, c := range v.alg.Carriers {
		carrierSum += v.opLevel[c]
		v.free = v.free || p.Ops[c].FreeOn
	}
	v.outScale = amplitude / math.Max(1.0, carrierSum)
	if v.patch.Level > 0 {
		v.outScale *= v.patch.Level
	}
	if v.free { // the free carriers' envelopes have the note's amplitude in them; their levels add up to a share each
		v.outScale = 1.0 / float64(len(v.alg.Carriers))
		if v.patch.Level > 0 {
			v.outScale *= v.patch.Level
		}
	}

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
	for _, e := range v.freeEnv {
		e.Release()
	}
	v.filter.Release()
}

func (v *Voice) Clear() {
	for i := range v.envs {
		v.envs[i].reset()
	}
	v.opAmp = [NumOperators]float64{}
	v.ampCount = 0
	v.pitchEnv.reset()
	for _, e := range v.freeEnv {
		e.Clear()
	}
	v.filter.Clear()
	v.Output(0).Clear()
}

// IsActive is true while any carrier envelope is running
func (v *Voice) IsActive() bool {
	for i := range v.envs {
		if v.isCarrier[i] && !v.envs[i].idle() && !v.patch.Ops[i].FreeOn {
			return true
		}
		if v.isCarrier[i] && v.patch.Ops[i].FreeOn && !v.freeEnv[i].Done() {
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
		return sine.Cycle(ph)
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

	glideCoef := 1.0
	if p.Glide > 0.0 {
		glideCoef = 1.0 - math.Exp(-1.0/(p.Glide*0.001*sr))
	}
	vibrato := p.VibratoDepth != 0.0 && p.VibratoRate > 0.0
	mods := v.mods
	modded := mods.Active()
	lfo2Active := mods.Uses(modmatrix.LFO2)
	if modded {
		mods.Values[modmatrix.Velocity] = v.velocity
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(baseFreq/261.6255653005986)/5, 1))
	}
	filtered := v.filter.On()
	over, factor := v.over, 1
	if over != nil {
		factor = over.Factor
	}
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

		// pitch from the pitch envelope and LFO pitch modulation, the bend and the wheel's vibrato
		v.expr.Tick(sr)
		if modded {
			mv := &mods.Values
			mv[modmatrix.LFO1] = lfo
			if lfo2Active {
				mv[modmatrix.LFO2] = sine.Cycle(v.lfo2)
				v.lfo2 += p.LFO2Rate * math.Exp2(mods.Mod(dLFO2Rate)) / sr
				v.lfo2 -= math.Floor(v.lfo2)
			}
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = v.filter.Env(), v.ampLast
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
		}
		semis := pitchLevelToSemitones(v.pitchEnv.next()) + lfo*pmDepth + v.expr.Semitones() + v.expr.Vibrato(sr, 5.5, 0.4) + mods.Mod(dPitch)
		freq := baseFreq
		if glideCoef < 1.0 {
			v.glideFreq += (baseFreq - v.glideFreq) * glideCoef
			freq = v.glideFreq
		}
		if semis != 0.0 {
			freq *= math.Exp2(semis / 12.0)
		}
		if vibrato {
			freq *= math.Exp2(p.VibratoDepth * v.vibFade * sine.Cycle(v.vibPhase) / 12.0)
			v.vibPhase += p.VibratoRate / sr
			v.vibPhase -= math.Floor(v.vibPhase)
			v.vibFade = math.Min(1.0, v.vibFade+v.vibInc)
		}

		// what every operator is this sample, apart from the wave it plays: the envelopes move once a sample, however often the operators are played
		var outMul, carMul, fbMul [NumOperators]float64
		ampNow := 0.0
		for op := range NumOperators {
			free := p.Ops[op].FreeOn
			if free { // an envelope and a level of its own, in the sound's units
				env := v.freeEnv[op].Generate()
				level := env * math.Max(0, v.freeLevel[op]+mods.Mod(dOp1Gain+op))
				fbMul[op] = env
				outMul[op] = level
				if v.isCarrier[op] {
					carMul[op] = level
					ampNow = math.Max(ampNow, env)
				} else {
					outMul[op] *= v.curModScale
				}
				continue
			}
			level := v.envs[op].next()
			if v.ampCount == 0 {
				v.opAmpInc[op] = (levelToAmp(level)*v.opLevel[op] - v.opAmp[op]) / ampUpdate
			}
			v.opAmp[op] += v.opAmpInc[op]
			amp := v.opAmp[op]

			if ams := p.Ops[op].AmpModSens; ams > 0 && amDepth > 0.0 {
				amp *= 1.0 - amDepth*ampModDepth[ams]*(0.5+0.5*lfo)
			}

			amp = math.Max(0, amp+mods.Mod(dOp1Gain+op))
			fbMul[op] = amp
			// what it modulates with (an algorithm of your own can have a carrier modulate too), what it adds to the sound
			outMul[op] = amp * modIndex * v.curModScale
			if v.isCarrier[op] {
				carMul[op] = amp
				ampNow = math.Max(ampNow, amp)
			}
		}
		pmScale := math.Max(0, 1+mods.Mod(dModDepth))
		fbPM := 0.0
		if fbOp := v.alg.FeedbackTo; p.Ops[fbOp].FreeOn {
			if fb := math.Max(0, p.FreeFeedback+mods.Mod(dFeedback)); fb > 0.0 {
				fbPM = fb
			}
		} else if fb := math.Max(0, fbAmount+mods.Mod(dFeedback)); fb > 0.0 {
			fbPM = fb
		}
		var inc [NumOperators]float64 // the phase an operator moves a play, a play being a sample over the factor
		for op := range NumOperators {
			f := v.fixed[op]
			if f == 0.0 {
				f = freq * v.freqMult[op]
			}
			inc[op] = f / sr / float64(factor)
		}

		// operators from 6 down to 1, modulators always have a higher number than their targets
		play := func() float64 {
			sample := 0.0
			for op := NumOperators - 1; op >= 0; op-- {
				pm := 0.0
				for _, m := range v.modTargets[op] {
					pm += v.out[m]
				}
				pm *= pmScale
				if op == v.alg.FeedbackTo && fbPM > 0 {
					pm += (v.fb[0] + v.fb[1]) * 0.5 * fbPM
				}

				var s float64
				if w := p.Ops[op].Wave; w == 0 {
					s = sine.Cycle(v.phase[op] + pm)
				} else {
					s = wave(w, v.phase[op]+pm)
				}
				if op == v.alg.FeedbackFrom {
					v.fb[1], v.fb[0] = v.fb[0], s*fbMul[op]
				}
				v.out[op] = s * outMul[op]
				sample += s * carMul[op]

				v.phase[op] += inc[op]
				v.phase[op] -= math.Floor(v.phase[op])
			}
			return sample
		}
		var sample float64
		if over != nil {
			sample = over.Down(func(int) float64 { return play() })
		} else {
			sample = play()
		}

		v.ampCount = (v.ampCount + 1) % ampUpdate

		// remove the DC offset
		v.ampLast = ampNow
		x := sample * v.outScale * math.Max(0, 1+mods.Mod(dLevel))
		if v.free { // the free operators are the FM synth's, which has no DC blocker
			v.dcY = x
		} else {
			v.dcY = x - v.dcX + v.dcCoef*v.dcY
			v.dcX = x
		}

		if filtered {
			v.filter.Tick(freq, mods.Mod(dCutoff), mods.Mod(dResonance))
			for c, ch := range outs {
				ch[i] = float32(v.filter.Process(c, v.dcY))
			}
		} else {
			val := float32(v.dcY)
			for _, ch := range outs {
				ch[i] = val
			}
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
