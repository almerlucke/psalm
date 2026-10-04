package vco

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/poly"
)

type FilterMode int

const (
	LowPass FilterMode = iota
	BandPass
	HighPass
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
	FilterMode      FilterMode
	Cutoff          float64 // Hz
	Resonance       float64 // 0-1
	FilterEnvAmount float64 // octaves added to cutoff at full envelope
	KeyTrack        float64 // 0-1, 1 means cutoff follows pitch fully (relative to middle C)

	// Envelopes (durations in milliseconds)
	FilterEnv adsr.Setting
	AmpEnv    adsr.Setting

	// Modulation
	LFO1 LFOConfig
	LFO2 LFOConfig
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

type VCO struct {
	*psalm.Module
	cfg       Config
	osc1      *blosc.Blosc
	osc2      *blosc.Blosc
	sub       *blosc.Blosc
	noise     noise
	lfo1      lfo
	lfo2      lfo
	filter    *svf.SVF
	filterEnv *adsr.ADSR
	ampEnv    *adsr.ADSR
	frequency float64 // target frequency
	glideFreq float64 // current frequency, glides towards frequency
	cutoff    float64 // smoothed cutoff, ramps towards cfg.Cutoff over a buffer
	waitSamps int
}

func New(cfg Config) *VCO {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Mono
	}

	v := &VCO{
		Module:    psalm.NewModule(nil, []int{cfg.NumChannels}),
		cfg:       cfg,
		frequency: middleC,
		glideFreq: middleC,
		cutoff:    cfg.Cutoff,
	}

	sr := v.Config().SampleRate

	v.osc1 = blosc.New(v.frequency, sr, 0.0, cfg.PulseWidth, 1.0, cfg.Osc1Waveform)
	v.osc2 = blosc.New(v.frequency*v.osc2Ratio(), sr, 0.0, cfg.PulseWidth, 1.0, cfg.Osc2Waveform)
	v.sub = blosc.New(v.frequency*0.5, sr, 0.0, 0.5, 1.0, blosc.Square)
	v.filter = svf.New(cfg.Cutoff, cfg.Resonance, sr)
	v.filterEnv = adsr.New(&v.cfg.FilterEnv, adsr.NoteOff, sr)
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)

	v.SetSelf(v)

	return v
}

func (v *VCO) osc2Ratio() float64 {
	return math.Pow(2.0, float64(v.cfg.Osc2Octave)+v.cfg.Osc2Detune/12.0)
}

// SetFrequency sets the target frequency, the voice glides there when Glide is set
func (v *VCO) SetFrequency(frequency float64) {
	v.frequency = frequency
	if v.cfg.Glide <= 0.0 {
		v.glideFreq = frequency
	}
}

func (v *VCO) SetPitch(pitch float64) {
	v.SetFrequency(440.0 * math.Pow(2.0, (pitch-69.0)/12.0))
}

func (v *VCO) SetPulseWidth(pw float64) {
	v.cfg.PulseWidth = pw
	v.osc1.SetPulseWidth(pw)
	v.osc2.SetPulseWidth(pw)
}

func (v *VCO) ReceiveMessage(msg psalm.Message) {
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
	msg.IfIntVal("filterMode", func(m int) { v.cfg.FilterMode = FilterMode(m) })
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

// Receive accepts messages, or plain values on inputs 1 (cutoff), 2 (resonance) and 3 (pulse width),
// which makes it easy to connect LFOs
func (v *VCO) Receive(value any, index int) {
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
	}
}

func (v *VCO) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.NoteOff, 0.0, amplitude, message, tick)
}

func (v *VCO) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(adsr.Duration, duration, amplitude, message, tick)
}

func (v *VCO) note(mode adsr.ReleaseMode, duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.ReceiveMessage(message)

	v.waitSamps = tick.SampleOffset

	sr := v.Config().SampleRate
	v.lfo1.trigger(&v.cfg.LFO1, sr)
	v.lfo2.trigger(&v.cfg.LFO2, sr)

	// adsr durations are in milliseconds
	v.filterEnv.TriggerFull(duration*1000.0, 1.0, &v.cfg.FilterEnv, mode)
	v.ampEnv.TriggerFull(duration*1000.0, amplitude, &v.cfg.AmpEnv, mode)
}

func (v *VCO) NoteOff() {
	v.filterEnv.Release()
	v.ampEnv.Release()
}

func (v *VCO) Clear() {
	v.filterEnv.Clear()
	v.ampEnv.Clear()
	v.filter.Clear()
	v.Output(0).Clear()
}

func (v *VCO) IsActive() bool {
	return !v.ampEnv.Done()
}

func (v *VCO) Synthesize() {
	outSocket := v.Output(0)
	outSocket.Clear()

	if v.ampEnv.Done() {
		v.cutoff = v.cfg.Cutoff
		return
	}

	cfg := &v.cfg
	sr := v.Config().SampleRate
	outChannels := outSocket.Channels
	bufferSize := v.Config().BufferSize
	keyTrackMult := math.Pow(v.glideFreq/middleC, cfg.KeyTrack)
	cutoffInc := (cfg.Cutoff - v.cutoff) / float64(bufferSize-v.waitSamps)
	osc2Ratio := v.osc2Ratio()
	lfo1Active := cfg.LFO1.active()
	lfo2Active := cfg.LFO2.active()
	drive := cfg.Drive
	if drive <= 0.0 {
		drive = 1.0
	}
	driveComp := 1.0 / math.Tanh(drive)

	glideCoef := 1.0
	if cfg.Glide > 0.0 {
		glideCoef = 1.0 - math.Exp(-1.0/(cfg.Glide*0.001*sr))
	}

	for i := v.waitSamps; i < bufferSize; i++ {
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
		if semis := cfg.LFO1.Pitch*l1 + cfg.LFO2.Pitch*l2 + cfg.FilterEnvToPitch*fenv; semis != 0.0 {
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

		// Filter with envelope and LFO modulation in octaves
		v.cutoff += cutoffInc
		octaves := cfg.FilterEnvAmount*fenv + cfg.LFO1.Cutoff*l1 + cfg.LFO2.Cutoff*l2
		v.filter.Set(v.cutoff*keyTrackMult*math.Pow(2.0, octaves), cfg.Resonance)
		low, band, high := v.filter.Tick(s)
		switch cfg.FilterMode {
		case BandPass:
			s = band
		case HighPass:
			s = high
		default:
			s = low
		}

		// Amp with LFO tremolo
		amp := v.ampEnv.Generate()
		if cfg.LFO1.Amp != 0.0 {
			amp *= 1.0 - cfg.LFO1.Amp*(1.0-l1)*0.5
		}
		if cfg.LFO2.Amp != 0.0 {
			amp *= 1.0 - cfg.LFO2.Amp*(1.0-l2)*0.5
		}

		val := float32(s * amp)
		for _, channel := range outChannels {
			channel[i] = val
		}

		if v.ampEnv.Done() {
			v.cutoff = cfg.Cutoff
			break
		}
	}

	v.waitSamps = 0
}

// NewPoly creates a polyphonic VCO synth
func NewPoly(cfg Config, numVoices int) *poly.Poly[*VCO] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Mono
	}

	voices := make([]*VCO, numVoices)
	for i := range numVoices {
		voices[i] = New(cfg)
	}

	return poly.New[*VCO](cfg.NumChannels, voices)
}
