package pulsar

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/sine"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

// Window is the envelope of every pulsaret
type Window int

const (
	Hann       Window = iota // smooth
	Gaussian                 // smooth and narrow, a purer formant
	Percussive               // sharp attack, exponential decay
	Rectangle                // hard edges, bright and buzzy
	Tapered                  // a rectangle with short smooth edges: buzzy without the aliasing
)

// Pulsaret2 is the second shape of the pulse content, mixed with the formant sines by ShapeMix
type Pulsaret2 int

const (
	Saw      Pulsaret2 = iota // a saw at the formant frequency (all harmonics)
	Square                    // a square (odd harmonics)
	Triangle                  // a triangle (odd harmonics, soft)
)

// Pulsaret2Names are the names of the second pulsaret shapes, by number
var Pulsaret2Names = []string{"saw", "square", "triangle"}

// maxShapeHarmonics limits the harmonics summed for the second pulsaret
const maxShapeHarmonics = 8

// tukeyTaper is the part of the pulse each edge of the Tapered window takes
const tukeyTaper = 0.1

// Config describes a pulsar voice. A pulsar is a train of pulses at the pitch of the note; every
// pulse is a short burst of a sine at the formant frequency, shaped by a window. The pulse rate
// sets the pitch (at very low notes the pulses become rhythms), the formant sets a spectral peak
// that stays put when the pitch changes, the duty sets how much of every period the pulse fills
type Config struct {
	NumChannels int

	Formant          float64 // Hz, the frequency inside every pulse (the spectral peak)
	Formant2         float64 // Hz, a second formant (0 is off)
	Formant2Level    float64 // 0-1
	FormantTracking  float64 // 0-1, how much the formants follow the pitch (relative to middle C)
	Duty             float64 // 0-1, part of every period the pulse sounds
	Window           Window
	Pulsaret2        Pulsaret2 // the second shape of the pulse content
	ShapeMix         float64   // 0-1: from the formant sines (0) to the second pulsaret (1)
	Burst            int       // pulses played in a burst (0 is off: every pulse plays)
	BurstRest        int       // pulses silent after a burst
	Masking          float64   // 0-1, chance a pulse is silent (stochastic, crackling textures)
	Jitter           float64   // 0-1, randomness of the pulse period
	StereoSpread     float64   // 0-1, random pan per pulse, for sparse textures (on a tone it is noise)
	NoteSpread       float64   // 0-1, random pan per note, to widen chords and pads
	FormantEnvAmount float64   // octaves the formant envelope adds to the formants
	FormantLFORate   float64   // Hz
	FormantLFODepth  float64   // octaves
	VibratoRate      float64   // Hz
	VibratoDepth     float64   // semitones
	Glide            float64   // milliseconds from the previous note of the voice
	Level            float64   // output level (0 is treated as 1)

	Filter voicefilter.Config // a filter after the pulse train (off unless its Type is set)

	// the matrix: a source to a destination (see Dests) over an amount. FilterEnv is the formant
	// envelope, LFO1 the formant LFO (its rate, any depth), LFO2 a second sine at LFO2Rate Hz (0 is 1)
	LFO2Rate float64
	Mods     [modmatrix.Slots]modmatrix.Slot
	Ext      [modmatrix.Exts]float64 // the values 0-1 of the sources Ext 1 to 4 (cabled in on InExt1...)

	FormantEnv adsr.Setting
	AmpEnv     adsr.Setting
}

func DefaultConfig() Config {
	return Config{
		NumChannels:   psalm.Stereo,
		Formant:       800,
		Formant2Level: 0.5,
		Duty:          0.5,
		Window:        Hann,
		Level:         1,
		FormantEnv:    *adsr.NewSetting(1.0, 5.0, 0.0, 300.0, 1.0, 300.0),
		AmpEnv:        *adsr.NewSetting(1.0, 10.0, 0.8, 300.0, 1.0, 300.0),
	}
}

const middleC = 261.6255653005986

// The destinations of the matrix (see Dests), by number
const (
	dFormant = iota
	dFormant2
	dDuty
	dMasking
	dJitter
	dSpread
	dPitch
	dLevel
	dCutoff
	dResonance
	dShapeMix
	dLFORate
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (octaves, semitones, or the setting itself). They are picked up
// by the next pulse, except the pitch, the level, the filter and the rates
var Dests = []modmatrix.Dest{
	dFormant:   {Key: "formant", Label: "formant", Span: 3},
	dFormant2:  {Key: "formant2", Label: "formant 2", Span: 3},
	dDuty:      {Key: "duty", Label: "duty", Span: 0.5},
	dMasking:   {Key: "masking", Label: "masking", Span: 1},
	dJitter:    {Key: "jitter", Label: "jitter", Span: 1},
	dSpread:    {Key: "stereoSpread", Label: "pulse spread", Span: 1},
	dPitch:     {Key: "pitch", Label: "pitch", Span: 24},
	dLevel:     {Key: "level", Label: "level", Span: 1},
	dCutoff:    {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance: {Key: "resonance", Label: "resonance", Span: 1},
	dShapeMix:  {Key: "shapeMix", Label: "shape mix", Span: 1},
	dLFORate:   {Key: "lfoRate", Label: "lfo rate", Span: 3},
}

// The value inputs beyond the four that set the formant, the duty, the masking and the second
// formant: 5 to 8 are the values of the matrix (Ext 1 to 4) that a slot can move any setting by
const InExt1 = 5 // to InExt1+3

// Voice is a pulsar synth voice. Value inputs: 1 formant, 2 duty, 3 masking, 4 second formant,
// 5 to 8 the Ext values of the matrix.
// They are picked up by the next pulse, so changing them never clicks
type Voice struct {
	*psalm.Module
	cfg        Config
	ampEnv     *adsr.ADSR
	formantEnv *adsr.ADSR

	frequency float64               // target
	expr      expression.Expression // the pitch bend; the mod wheel lifts the formant
	glideFreq float64
	phase     float64 // position in the current period, 0-1
	period    float64 // period length relative to the pitch (jitter)

	// the current pulse
	masked   bool
	duty     float64
	formant1 float64
	formant2 float64
	mix      float64 // the shape mix of the pulse
	pulses   int     // pulses started in this note
	gainL    float64
	gainR    float64
	notePan  float64
	compGain float64

	vibPhase  float64
	lfoPhase  float64
	dcX, dcY  [2]float64 // per channel, the pan is applied before the DC blocker
	dcCoef    float64
	lfo2Phase float64
	waitSamps int
	filter    *voicefilter.Filter
	mods      *modmatrix.Matrix
	velocity  float64
	random    float64
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
		period:    1,
		notePan:   0.5,
	}

	sr := v.Config().SampleRate
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)
	v.formantEnv = adsr.New(&v.cfg.FormantEnv, adsr.NoteOff, sr)
	v.dcCoef = 1.0 - 2.0*math.Pi*10.0/sr // 10 Hz DC blocker: short sine bursts carry an offset
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

func (v *Voice) SetFrequency(frequency float64) {
	v.frequency = frequency
	if v.cfg.Glide <= 0 {
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
	envelope.Receive(msg, "amp", &v.cfg.AmpEnv)
	envelope.Receive(msg, "formant", &v.cfg.FormantEnv)
	cfg := &v.cfg
	msg.IfFltVal("formant", func(f float64) { cfg.Formant = f })
	msg.IfFltVal("formant2", func(f float64) { cfg.Formant2 = f })
	msg.IfFltVal("formant2Level", func(f float64) { cfg.Formant2Level = f })
	msg.IfFltVal("formantTracking", func(f float64) { cfg.FormantTracking = f })
	msg.IfFltVal("duty", func(f float64) { cfg.Duty = f })
	msg.IfIntVal("window", func(w int) { cfg.Window = Window(w) })
	msg.IfIntVal("pulsaret2", func(w int) { cfg.Pulsaret2 = Pulsaret2(max(0, min(w, int(Triangle)))) })
	msg.IfFltVal("shapeMix", func(f float64) { cfg.ShapeMix = f })
	msg.IfIntVal("burst", func(n int) { cfg.Burst = max(0, n) })
	msg.IfIntVal("burstRest", func(n int) { cfg.BurstRest = max(0, n) })
	msg.IfFltVal("lfo2Rate", func(f float64) { cfg.LFO2Rate = f })
	msg.IfFltVal("masking", func(f float64) { cfg.Masking = f })
	msg.IfFltVal("jitter", func(f float64) { cfg.Jitter = f })
	msg.IfFltVal("stereoSpread", func(f float64) { cfg.StereoSpread = f })
	msg.IfFltVal("noteSpread", func(f float64) { cfg.NoteSpread = f })
	msg.IfFltVal("formantEnvAmount", func(f float64) { cfg.FormantEnvAmount = f })
	msg.IfFltVal("formantLFORate", func(f float64) { cfg.FormantLFORate = f })
	msg.IfFltVal("formantLFODepth", func(f float64) { cfg.FormantLFODepth = f })
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
	case 1:
		v.cfg.Formant = f
	case 2:
		v.cfg.Duty = f
	case 3:
		v.cfg.Masking = f
	case 4:
		v.cfg.Formant2 = f
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

	if !wasActive {
		v.phase = 1 // the first pulse starts right away
		v.vibPhase = 0
		v.dcX, v.dcY = [2]float64{}, [2]float64{}
	}

	// a new pan for every note, picked up by the next pulse
	v.pulses = 0
	v.velocity = amplitude
	v.random = v.Rand().Float64()*2 - 1
	v.lfo2Phase = 0
	v.notePan = 0.5 + v.cfg.NoteSpread*(v.Rand().Float64()-0.5)

	v.ampEnv.TriggerFull(duration*1000.0, amplitude, &v.cfg.AmpEnv, mode)
	v.formantEnv.TriggerFull(duration*1000.0, 1.0, &v.cfg.FormantEnv, mode)
	v.filter.Trigger(duration*1000.0, mode)
}

func (v *Voice) NoteOff() {
	v.ampEnv.Release()
	v.formantEnv.Release()
	v.filter.Release()
}

func (v *Voice) Clear() {
	v.ampEnv.Clear()
	v.formantEnv.Clear()
	v.filter.Clear()
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return !v.ampEnv.Done()
}

// window returns the pulsaret envelope at t (0-1)
func window(w Window, t float64) float64 {
	switch w {
	case Gaussian:
		x := (t - 0.5) / 0.15
		return math.Exp(-0.5 * x * x)
	case Percussive:
		const attack = 0.02
		if t < attack {
			return t / attack
		}
		return math.Exp(-5.0*(t-attack)/(1.0-attack)) * (1.0 - t) / (1.0 - attack)
	case Rectangle:
		return 1
	case Tapered:
		if t < tukeyTaper {
			return 0.5 - 0.5*sine.Cos(math.Pi*t/tukeyTaper)
		}
		if t > 1.0-tukeyTaper {
			return 0.5 - 0.5*sine.Cos(math.Pi*(1.0-t)/tukeyTaper)
		}
		return 1
	default:
		s := sine.Cycle(0.5 * t)
		return s * s
	}
}

// maxFormant is the highest a formant goes, as a part of the sample rate (just below half of it)
const maxFormant = 0.45

// maxCompensation limits how much short pulses are boosted, very short pulses get quieter
// instead of turning into huge spikes
const maxCompensation = 3.0

// pulse returns the pulsaret at t (0-1 through the pulse): the formants (sine bursts, mixed so a
// second formant doesn't raise the peak) shaped by the window. Under the percussive window the
// formants start at their peak (its short attack keeps that click free), so the struck start of the
// pulse carries the sound instead of the zero crossing of a sine
func (v *Voice) pulse(t float64, seconds float64) float64 {
	cfg := &v.cfg
	phase := 0.0
	if cfg.Window == Percussive {
		phase = 0.5 * math.Pi
	}
	x := sine.Sin(2.0*math.Pi*v.formant1*seconds + phase)
	if v.formant2 > 0 {
		x += cfg.Formant2Level * sine.Sin(2.0*math.Pi*v.formant2*seconds+phase)
		x /= math.Sqrt(1.0 + cfg.Formant2Level*cfg.Formant2Level)
	}
	if v.mix > 0 {
		x = (1-v.mix)*x + v.mix*v.shape(seconds)
	}
	return x * window(cfg.Window, t)
}

// shape returns the second pulsaret at a time into the pulse: a few harmonics of the first formant,
// below the Nyquist frequency, so a saw or a square doesn't alias
func (v *Voice) shape(seconds float64) float64 {
	n := min(maxShapeHarmonics, int(maxFormant*v.Config().SampleRate/math.Max(1, v.formant1)))
	x := 0.0
	for k := 1; k <= n; k++ {
		a := 2.0 * math.Pi * float64(k) * v.formant1 * seconds
		switch v.cfg.Pulsaret2 {
		case Square:
			if k%2 == 1 {
				x += sine.Sin(a) / float64(k)
			}
		case Triangle:
			if k%2 == 1 {
				sign := 1.0
				if k%4 == 3 {
					sign = -1
				}
				x += sign * sine.Sin(a) / float64(k*k)
			}
		default:
			x += sine.Sin(a) / float64(k)
		}
	}
	switch v.cfg.Pulsaret2 {
	case Square:
		return x * 4 / math.Pi
	case Triangle:
		return x * 8 / (math.Pi * math.Pi)
	}
	return x * 2 / math.Pi
}

// compensation measures the energy of the current pulse (its length, formants and window) and
// returns the gain that gives every pulse train the level of a continuous sine. A short pulse can
// hold only a part of a formant cycle, so the energy is measured instead of assumed
func (v *Voice) compensation(freq float64) float64 {
	const steps = 32
	pulseSeconds := v.duty / freq
	sum := 0.0
	for i := range steps {
		t := (float64(i) + 0.5) / steps
		x := v.pulse(t, t*pulseSeconds)
		sum += x * x
	}
	energy := v.duty * sum / steps // mean square over the whole period
	if energy <= 1e-12 {
		return 0
	}
	return math.Min(maxCompensation, 1.0/math.Sqrt(2.0*energy))
}

// startPulse sets up the next pulse: masking, pan, the formants at this moment and the level
func (v *Voice) startPulse(formantMod float64, freq float64) {
	cfg := &v.cfg
	mods := v.mods

	masking := math.Max(0, math.Min(cfg.Masking+mods.Mod(dMasking), 1))
	v.masked = masking > 0 && v.Rand().Float64() < masking
	if cfg.Burst > 0 && v.pulses%(cfg.Burst+cfg.BurstRest) >= cfg.Burst {
		v.masked = true
	}
	v.pulses++
	v.duty = math.Max(0.01, math.Min(cfg.Duty+mods.Mod(dDuty), 1.0))
	v.mix = math.Max(0, math.Min(cfg.ShapeMix+mods.Mod(dShapeMix), 1))

	tracking := math.Pow(v.glideFreq/middleC, cfg.FormantTracking)
	octaves := math.Exp2(formantMod + mods.Mod(dFormant))
	// a sine burst above half the sample rate folds back (aliases) into inharmonic noise: the
	// first formant stays below it, a second formant past it is left out (high notes with full
	// formant tracking get there)
	ceiling := maxFormant * v.Config().SampleRate
	v.formant1 = math.Min(cfg.Formant*tracking*octaves, ceiling)
	v.formant2 = cfg.Formant2 * tracking * octaves * math.Exp2(mods.Mod(dFormant2))
	if v.formant2 > ceiling {
		v.formant2 = 0
	}

	spread := math.Max(0, math.Min(cfg.StereoSpread+mods.Mod(dSpread), 1))
	pan := math.Max(0, math.Min(v.notePan+spread*(v.Rand().Float64()-0.5), 1))
	v.gainL = math.Cos(pan * math.Pi / 2)
	v.gainR = math.Sin(pan * math.Pi / 2)

	v.compGain = v.compensation(freq)

	jitter := math.Max(0, math.Min(cfg.Jitter+mods.Mod(dJitter), 1))
	v.period = 1.0 + jitter*(v.Rand().Float64()*2.0-1.0)*0.5
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
		level   = cfg.Level
	)
	if level <= 0 {
		level = 1
	}

	glideCoef := 1.0
	if cfg.Glide > 0 {
		glideCoef = 1.0 - math.Exp(-1.0/(cfg.Glide*0.001*sr))
	}

	mods := v.mods
	modded := mods.Active()
	lfoOn := cfg.FormantLFORate > 0 && (cfg.FormantLFODepth != 0 || mods.Uses(modmatrix.LFO1))
	lfo2Rate := cfg.LFO2Rate
	if lfo2Rate <= 0 {
		lfo2Rate = 1
	}
	lfo2On := mods.Uses(modmatrix.LFO2)
	filtered := v.filter.On()
	if modded {
		mods.Values[modmatrix.Velocity] = v.velocity
		mods.Values[modmatrix.Random] = v.random
	}

	for i := v.waitSamps; i < bufSize; i++ {
		amp := v.ampEnv.Generate()
		fenv := v.formantEnv.Generate()

		rateMod := 1.0
		if modded {
			rateMod = math.Exp2(mods.Mod(dLFORate))
		}
		unitLFO, lfo2 := 0.0, 0.0
		if lfoOn {
			unitLFO = sine.Cycle(v.lfoPhase)
			v.lfoPhase += cfg.FormantLFORate * rateMod / sr
			v.lfoPhase -= math.Floor(v.lfoPhase)
		}
		if lfo2On {
			lfo2 = sine.Cycle(v.lfo2Phase)
			v.lfo2Phase += lfo2Rate * rateMod / sr
			v.lfo2Phase -= math.Floor(v.lfo2Phase)
		}
		if modded {
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = fenv, amp
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = unitLFO, lfo2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mv[modmatrix.Key] = math.Max(-1, math.Min(math.Log2(v.glideFreq/middleC)/5, 1))
			mods.Eval(sr)
		}

		// pitch with glide and vibrato, and the bend
		v.expr.Tick(sr)
		v.glideFreq += (v.frequency - v.glideFreq) * glideCoef
		freq := v.glideFreq * v.expr.Ratio()
		if modded {
			freq *= math.Exp2(mods.Mod(dPitch) / 12)
		}
		if cfg.VibratoDepth != 0 && cfg.VibratoRate > 0 {
			freq *= math.Exp2(cfg.VibratoDepth * sine.Cycle(v.vibPhase) / 12.0)
			v.vibPhase += cfg.VibratoRate / sr
			v.vibPhase -= math.Floor(v.vibPhase)
		}

		formantLFO := cfg.FormantLFODepth * unitLFO

		// next pulse
		if v.phase >= v.period {
			v.phase -= v.period
			v.startPulse(cfg.FormantEnvAmount*fenv+formantLFO+1.5*v.expr.Wheel(), freq)
		}

		x := 0.0
		if !v.masked && v.phase < v.duty {
			x = v.pulse(v.phase/v.duty, v.phase/freq) * v.compGain
		}

		v.phase += freq / sr

		if filtered {
			v.filter.Tick(freq, mods.Mod(dCutoff), mods.Mod(dResonance))
		} else {
			v.filter.Tick(freq, 0, 0)
		}
		g := amp * level
		if modded {
			g = amp * math.Max(0, level+mods.Mod(dLevel))
		}

		// pan, then remove the offset of the sine bursts (panning after the DC blocker would put a
		// step in its tail at every pan change), then the filter
		if len(outs) == 1 {
			outs[0][i] = float32(v.filterOut(filtered, 0, v.dcBlock(0, x)) * g)
		} else {
			outs[0][i] = float32(v.filterOut(filtered, 0, v.dcBlock(0, x*v.gainL)) * g)
			outs[1][i] = float32(v.filterOut(filtered, 1, v.dcBlock(1, x*v.gainR)) * g)
		}
	}

	v.waitSamps = 0
}

func (v *Voice) filterOut(on bool, c int, x float64) float64 {
	if on {
		return v.filter.Process(c, x)
	}
	return x
}

func (v *Voice) dcBlock(c int, x float64) float64 {
	v.dcY[c] = x - v.dcX[c] + v.dcCoef*v.dcY[c]
	v.dcX[c] = x
	return v.dcY[c]
}

// NewPoly creates a polyphonic pulsar synth
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
