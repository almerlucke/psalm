package subtractive

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/formant"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/poly"
)

// FilterType selects the filter of the voice
type FilterType int

const (
	FilterSVF     FilterType = iota // state variable filter, 12 dB/oct, modes set by FilterMode
	FilterLadder                    // Moog style ladder, 24 or 12 dB/oct, modes set by LadderMode
	FilterFormant                   // vowel filter, the cutoff modulation moves the vowel instead
)

// FilterMode selects the output of the SVF filter
type FilterMode int

const (
	LowPass  = FilterMode(svf.LowPass)
	BandPass = FilterMode(svf.BandPass)
	HighPass = FilterMode(svf.HighPass)
	Notch    = FilterMode(svf.Notch)
	Peak     = FilterMode(svf.Peak)
	AllPass  = FilterMode(svf.AllPass)
)

// Config describes a classic subtractive voice: 2 oscillators + noise -> filter -> amp
type Config struct {
	NumChannels int

	// Oscillators
	Osc1Waveform blosc.Waveform
	Osc2Waveform blosc.Waveform
	Osc1Level    float64
	Osc2Level    float64
	Osc2Detune   float64 // semitones (fractional for cents)
	Osc2Octave   int
	PulseWidth   float64 // 0-1, used by pulse-width aware waveforms (Rectangle etc)
	SubLevel     float64 // square wave sub oscillator one octave below osc 1
	NoiseLevel   float64
	NoiseType    NoiseType
	RingModLevel float64 // osc 1 * osc 2
	Sync         bool    // hard sync osc 2 to osc 1
	FMAmount     float64 // osc 1 modulates the phase of osc 2
	Drive        float64 // pre filter saturation, 1.0 is unity

	// Pitch
	Glide            float64 // portamento time in milliseconds from the previous note of this voice
	FilterEnvToPitch float64 // semitones of pitch sweep driven by the filter envelope

	// Filter
	FilterType      FilterType
	FilterMode      FilterMode  // SVF output
	LadderMode      ladder.Mode // ladder output
	FilterDrive     float64     // ladder input saturation (0 is treated as 1)
	Cutoff          float64     // Hz
	Resonance       float64     // 0-1, for the formant filter it sets the sharpness of the formants
	FilterEnvAmount float64     // octaves added to cutoff at full envelope (vowel steps for the formant filter)
	KeyTrack        float64     // 0-1, 1 means cutoff follows pitch fully (relative to middle C)

	// Formant filter
	Vowel        float64 // 0-4: a e i o u, fractions morph. Envelope and LFO cutoff amounts move the vowel
	FormantShift float64 // scales the formant frequencies (0 is treated as 1)

	// Envelopes (durations in milliseconds)
	FilterEnv adsr.Setting
	AmpEnv    adsr.Setting

	// Modulation
	LFO1 LFOConfig
	LFO2 LFOConfig

	Level float64 // output level (0 is treated as 1)
}

func DefaultConfig() Config {
	return Config{
		NumChannels:     psalm.Stereo,
		Osc1Waveform:    blosc.Sawtooth,
		Osc2Waveform:    blosc.Sawtooth,
		Osc1Level:       0.5,
		Osc2Level:       0.5,
		Osc2Detune:      0.07,
		PulseWidth:      0.5,
		Drive:           1.0,
		FilterMode:      LowPass,
		Cutoff:          800.0,
		Resonance:       0.3,
		FilterEnvAmount: 3.0,
		KeyTrack:        0.5,
		FilterEnv:       *adsr.NewSetting(1.0, 5.0, 0.3, 300.0, 1.0, 400.0),
		AmpEnv:          *adsr.NewSetting(1.0, 5.0, 0.7, 200.0, 1.0, 300.0),
	}
}

const middleC = 261.6255653005986

type Voice struct {
	*psalm.Module
	cfg       Config
	osc1      *blosc.Blosc
	osc2      *blosc.Blosc
	sub       *blosc.Blosc
	noise     noise
	lfo1      lfo
	lfo2      lfo
	svf       *svf.SVF
	ladder    *ladder.Ladder
	formant   *formant.Formant
	ladderDrv float64
	formShift float64
	formCount int // counts samples until the next filter coefficient update
	filterEnv *adsr.ADSR
	ampEnv    *adsr.ADSR
	frequency float64 // target frequency
	glideFreq float64 // current frequency, glides towards frequency
	cutoff    float64 // smoothed cutoff, ramps towards cfg.Cutoff over a buffer
	res       float64 // smoothed resonance, ramps towards cfg.Resonance over a buffer
	vowel     float64 // smoothed vowel, ramps towards cfg.Vowel over a buffer
	waitSamps int
	expr      expression.Expression // the pitch bend; the mod wheel opens the filter
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
		cutoff:    cfg.Cutoff,
		res:       cfg.Resonance,
		vowel:     cfg.Vowel,
	}

	sr := v.Config().SampleRate

	v.osc1 = blosc.New(v.frequency, sr, 0.0, cfg.PulseWidth, 1.0, cfg.Osc1Waveform)
	v.osc2 = blosc.New(v.frequency*v.osc2Ratio(), sr, 0.0, cfg.PulseWidth, 1.0, cfg.Osc2Waveform)
	v.sub = blosc.New(v.frequency*0.5, sr, 0.0, 0.5, 1.0, blosc.Square)
	v.svf = svf.New(cfg.Cutoff, cfg.Resonance, sr)
	v.ladder = ladder.New(cfg.Cutoff, cfg.Resonance, sr)
	v.formant = formant.New(cfg.Vowel, formantQ(cfg.Resonance), sr)
	v.filterEnv = adsr.New(&v.cfg.FilterEnv, adsr.NoteOff, sr)
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)

	v.lfo1.rng = v.Rand()
	v.lfo2.rng = v.Rand()
	v.noise.rng = v.Rand()

	v.SetSelf(v)

	return v
}

func (v *Voice) osc2Ratio() float64 {
	return math.Pow(2.0, float64(v.cfg.Osc2Octave)+v.cfg.Osc2Detune/12.0)
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

func (v *Voice) SetPulseWidth(pw float64) {
	v.cfg.PulseWidth = pw
	v.osc1.SetPulseWidth(pw)
	v.osc2.SetPulseWidth(pw)
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg)
	msg.IfIntVal("osc1Waveform", func(wf int) {
		v.cfg.Osc1Waveform = blosc.Waveform(wf)
		v.osc1.SetWaveform(v.cfg.Osc1Waveform)
	})
	msg.IfIntVal("osc2Waveform", func(wf int) {
		v.cfg.Osc2Waveform = blosc.Waveform(wf)
		v.osc2.SetWaveform(v.cfg.Osc2Waveform)
	})
	msg.IfFltVal("osc1Level", func(f float64) { v.cfg.Osc1Level = f })
	msg.IfFltVal("osc2Level", func(f float64) { v.cfg.Osc2Level = f })
	msg.IfFltVal("osc2Detune", func(f float64) { v.cfg.Osc2Detune = f })
	msg.IfIntVal("osc2Octave", func(o int) { v.cfg.Osc2Octave = o })
	msg.IfFltVal("pulseWidth", v.SetPulseWidth)
	msg.IfFltVal("subLevel", func(f float64) { v.cfg.SubLevel = f })
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
	msg.IfFltVal("noiseLevel", func(f float64) { v.cfg.NoiseLevel = f })
	msg.IfIntVal("noiseType", func(t int) { v.cfg.NoiseType = NoiseType(t) })
	msg.IfFltVal("ringModLevel", func(f float64) { v.cfg.RingModLevel = f })
	msg.IfBoolVal("sync", func(b bool) { v.cfg.Sync = b })
	msg.IfFltVal("fmAmount", func(f float64) { v.cfg.FMAmount = f })
	msg.IfFltVal("drive", func(f float64) { v.cfg.Drive = f })
	msg.IfFltVal("glide", func(f float64) { v.cfg.Glide = f })
	msg.IfFltVal("filterEnvToPitch", func(f float64) { v.cfg.FilterEnvToPitch = f })
	receiveLFOMessage(msg, "lfo1", &v.cfg.LFO1)
	receiveLFOMessage(msg, "lfo2", &v.cfg.LFO2)
	msg.IfIntVal("filterType", func(t int) { v.cfg.FilterType = FilterType(t) })
	msg.IfIntVal("filterMode", func(m int) { v.cfg.FilterMode = FilterMode(m) })
	msg.IfIntVal("ladderMode", func(m int) { v.cfg.LadderMode = ladder.Mode(m) })
	msg.IfFltVal("filterDrive", func(f float64) { v.cfg.FilterDrive = f })
	msg.IfFltVal("vowel", func(f float64) { v.cfg.Vowel = f })
	msg.IfFltVal("formantShift", func(f float64) { v.cfg.FormantShift = f })
	msg.IfFltVal("cutoff", func(f float64) { v.cfg.Cutoff = f })
	msg.IfFltVal("resonance", func(f float64) { v.cfg.Resonance = f })
	msg.IfFltVal("filterEnvAmount", func(f float64) { v.cfg.FilterEnvAmount = f })
	msg.IfFltVal("keyTrack", func(f float64) { v.cfg.KeyTrack = f })
	msg.IfFltVal("frequency", v.SetFrequency)
	msg.IfFltVal("pitch", v.SetPitch)
}

// receiveLFOMessage handles prefixed LFO keys like "lfo1Rate", "lfo1Pitch", "lfo1Cutoff"
func receiveLFOMessage(msg psalm.Message, prefix string, cfg *LFOConfig) {
	msg.IfIntVal(prefix+"Waveform", func(wf int) { cfg.Waveform = LFOWaveform(wf) })
	msg.IfFltVal(prefix+"Rate", func(f float64) { cfg.Rate = f })
	msg.IfFltVal(prefix+"Delay", func(f float64) { cfg.Delay = f })
	msg.IfBoolVal(prefix+"KeySync", func(b bool) { cfg.KeySync = b })
	msg.IfFltVal(prefix+"Pitch", func(f float64) { cfg.Pitch = f })
	msg.IfFltVal(prefix+"Cutoff", func(f float64) { cfg.Cutoff = f })
	msg.IfFltVal(prefix+"PulseWidth", func(f float64) { cfg.PulseWidth = f })
	msg.IfFltVal(prefix+"Amp", func(f float64) { cfg.Amp = f })
}

// Receive accepts messages, or plain values on inputs 1 (cutoff), 2 (resonance), 3 (pulse width)
// and 4 (vowel), which makes it easy to connect LFOs
func (v *Voice) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
		return
	}

	switch index {
	case 1:
		v.cfg.Cutoff = psalm.FltVal[float64](value)
	case 2:
		v.cfg.Resonance = psalm.FltVal[float64](value)
	case 3:
		v.SetPulseWidth(psalm.FltVal[float64](value))
	case 4:
		v.cfg.Vowel = psalm.FltVal[float64](value)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.NoteOff, 0.0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.Duration, duration, amplitude, message, tick)
}

func (v *Voice) note(mode adsr.ReleaseMode, duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.ReceiveMessage(message)

	v.waitSamps = tick.SampleOffset

	sr := v.Config().SampleRate
	v.lfo1.trigger(&v.cfg.LFO1, sr)
	v.lfo2.trigger(&v.cfg.LFO2, sr)

	// adsr durations are in milliseconds
	v.filterEnv.TriggerFull(duration*1000.0, 1.0, &v.cfg.FilterEnv, mode)
	v.ampEnv.TriggerFull(duration*1000.0, amplitude, &v.cfg.AmpEnv, mode)
}

func (v *Voice) NoteOff() {
	v.filterEnv.Release()
	v.ampEnv.Release()
}

func (v *Voice) Clear() {
	v.filterEnv.Clear()
	v.ampEnv.Clear()
	v.svf.Clear()
	v.ladder.Clear()
	v.formant.Clear()
	v.formCount = 0
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return !v.ampEnv.Done()
}

func (v *Voice) Synthesize() {
	outSocket := v.Output(0)
	outSocket.Clear()

	if v.ampEnv.Done() {
		v.resetSmoothing()
		return
	}

	cfg := &v.cfg
	sr := v.Config().SampleRate
	outChannels := outSocket.Channels
	bufferSize := v.Config().BufferSize
	keyTrackMult := math.Pow(v.glideFreq/middleC, cfg.KeyTrack)
	// cutoff, resonance and vowel glide over the buffer, so changing them (for example from an LFO
	// on the value inputs) doesn't step and click
	n := float64(bufferSize - v.waitSamps)
	cutoffInc := (cfg.Cutoff - v.cutoff) / n
	resInc := (cfg.Resonance - v.res) / n
	vowelInc := (cfg.Vowel - v.vowel) / n
	osc2Ratio := v.osc2Ratio()
	lfo1Active := cfg.LFO1.active()
	lfo2Active := cfg.LFO2.active()
	drive := cfg.Drive
	if drive <= 0.0 {
		drive = 1.0
	}
	driveComp := 1.0 / math.Tanh(drive)
	level := cfg.Level
	if level <= 0 {
		level = 1
	}

	glideCoef := 1.0
	if cfg.Glide > 0.0 {
		glideCoef = 1.0 - math.Exp(-1.0/(cfg.Glide*0.001*sr))
	}

	for i := v.waitSamps; i < bufferSize; i++ {
		v.expr.Tick(sr)
		// Modulation sources
		fenv := v.filterEnv.Generate()

		var l1, l2 float64
		if lfo1Active {
			l1 = v.lfo1.tick(&cfg.LFO1, sr)
		}
		if lfo2Active {
			l2 = v.lfo2.tick(&cfg.LFO2, sr)
		}

		// Pitch: glide, LFO vibrato and filter envelope pitch sweep
		v.glideFreq += (v.frequency - v.glideFreq) * glideCoef
		freq := v.glideFreq
		if semis := cfg.LFO1.Pitch*l1 + cfg.LFO2.Pitch*l2 + cfg.FilterEnvToPitch*fenv + v.expr.Semitones(); semis != 0.0 {
			freq *= math.Pow(2.0, semis/12.0)
		}

		v.osc1.SetFrequency(freq, sr)
		v.osc2.SetFrequency(freq*osc2Ratio, sr)

		// Pulse width modulation
		if lfo1Active || lfo2Active {
			pw := math.Max(0.02, math.Min(0.98, cfg.PulseWidth+cfg.LFO1.PulseWidth*l1+cfg.LFO2.PulseWidth*l2))
			v.osc1.SetPulseWidth(pw)
			v.osc2.SetPulseWidth(pw)
		}

		// Oscillators: osc 2 can be phase modulated by and hard synced to osc 1
		prevPhase := v.osc1.Phase()
		o1 := v.osc1.Tick(0, false, 0)
		phase := v.osc1.Phase()
		wrapped := phase < prevPhase

		fm := 0.0
		if cfg.FMAmount != 0.0 {
			fm = cfg.FMAmount * o1 * freq * osc2Ratio / sr
		}

		syncPhase := 0.0
		if cfg.Sync && wrapped {
			syncPhase = phase * osc2Ratio
			syncPhase -= math.Floor(syncPhase)
		}

		o2 := v.osc2.Tick(fm, cfg.Sync && wrapped, syncPhase)

		s := o1*cfg.Osc1Level + o2*cfg.Osc2Level

		if cfg.SubLevel > 0.0 {
			v.sub.SetFrequency(freq*0.5, sr)
			s += v.sub.Tick(0, false, 0) * cfg.SubLevel
		}
		if cfg.RingModLevel > 0.0 {
			s += o1 * o2 * cfg.RingModLevel
		}
		if cfg.NoiseLevel > 0.0 {
			s += v.noise.tick(cfg.NoiseType) * cfg.NoiseLevel
		}

		// Saturation
		s = math.Tanh(s*drive) * driveComp

		// Filter with envelope and LFO modulation in octaves (in vowel steps for the formant filter)
		v.cutoff += cutoffInc
		v.res += resInc
		v.vowel += vowelInc
		if v.formCount == 0 {
			mod := cfg.FilterEnvAmount*fenv + cfg.LFO1.Cutoff*l1 + cfg.LFO2.Cutoff*l2 + 2*v.expr.Wheel()
			v.setFilter(v.cutoff*keyTrackMult*math.Exp2(mod), mod)
		}
		v.formCount = (v.formCount + 1) % filterUpdate
		s = v.processFilter(s)

		// Amp with LFO tremolo
		amp := v.ampEnv.Generate()
		if cfg.LFO1.Amp != 0.0 {
			amp *= 1.0 - cfg.LFO1.Amp*(1.0-l1)*0.5
		}
		if cfg.LFO2.Amp != 0.0 {
			amp *= 1.0 - cfg.LFO2.Amp*(1.0-l2)*0.5
		}

		val := float32(s * amp * level)
		for _, channel := range outChannels {
			channel[i] = val
		}

		if v.ampEnv.Done() {
			v.resetSmoothing()
			break
		}
	}

	// land exactly on the targets, no rounding drift
	v.resetSmoothing()
	v.waitSamps = 0
}

// resetSmoothing sets the smoothed filter settings to their targets
func (v *Voice) resetSmoothing() {
	v.cutoff = v.cfg.Cutoff
	v.res = v.cfg.Resonance
	v.vowel = v.cfg.Vowel
}

// formantQ maps resonance 0-1 to the formant sharpness
func formantQ(res float64) float64 {
	return 4.0 + math.Max(0.0, math.Min(res, 1.0))*16.0
}

// filterUpdate is the interval in samples at which the filter coefficients are recalculated
// (about 11 kHz at 44.1 kHz). Calculating them every sample costs a lot (tan and exp) and is far
// more often than any modulation needs
const filterUpdate = 4

// setFilter updates the coefficients of the selected filter
func (v *Voice) setFilter(cutoff float64, mod float64) {
	cfg := &v.cfg

	switch cfg.FilterType {
	case FilterLadder:
		drive := cfg.FilterDrive
		if drive <= 0.0 {
			drive = 1.0
		}
		if drive != v.ladderDrv {
			v.ladderDrv = drive
			v.ladder.SetDrive(drive)
		}
		v.ladder.Set(cutoff, v.res)

	case FilterFormant:
		shift := cfg.FormantShift
		if shift <= 0.0 {
			shift = 1.0
		}
		if shift != v.formShift {
			v.formShift = shift
			v.formant.SetShift(shift)
		}
		v.formant.Set(v.vowel+mod, formantQ(v.res))

	default:
		v.svf.Set(cutoff, v.res)
	}
}

// processFilter runs a sample through the selected filter
func (v *Voice) processFilter(s float64) float64 {
	switch v.cfg.FilterType {
	case FilterLadder:
		return v.ladder.Process(s, v.cfg.LadderMode)
	case FilterFormant:
		return v.formant.Process(s)
	default:
		return v.svf.Process(s, svf.Mode(v.cfg.FilterMode))
	}
}

// NewPoly creates a polyphonic subtractive synth
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
