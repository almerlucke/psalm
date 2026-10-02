package vco

import (
	"math"
	"math/rand/v2"

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
	NoiseLevel   float64
	Drive        float64 // pre filter saturation, 1.0 is unity

	// Filter
	FilterMode      FilterMode
	Cutoff          float64 // Hz
	Resonance       float64 // 0-1
	FilterEnvAmount float64 // octaves added to cutoff at full envelope
	KeyTrack        float64 // 0-1, 1 means cutoff follows pitch fully (relative to middle C)

	// Envelopes (durations in milliseconds)
	FilterEnv adsr.Setting
	AmpEnv    adsr.Setting
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
	filter    *svf.SVF
	filterEnv *adsr.ADSR
	ampEnv    *adsr.ADSR
	frequency float64
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
	}

	sr := v.Config().SampleRate

	v.osc1 = blosc.New(v.frequency, sr, 0.0, cfg.PulseWidth, 1.0, cfg.Osc1Waveform)
	v.osc2 = blosc.New(v.osc2Frequency(), sr, 0.0, cfg.PulseWidth, 1.0, cfg.Osc2Waveform)
	v.filter = svf.New(cfg.Cutoff, cfg.Resonance, sr)
	v.filterEnv = adsr.New(&v.cfg.FilterEnv, adsr.NoteOff, sr)
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)

	v.SetSelf(v)

	return v
}

func (v *VCO) osc2Frequency() float64 {
	return v.frequency * math.Pow(2.0, float64(v.cfg.Osc2Octave)+v.cfg.Osc2Detune/12.0)
}

func (v *VCO) SetFrequency(frequency float64) {
	sr := v.Config().SampleRate
	v.frequency = frequency
	v.osc1.SetFrequency(frequency, sr)
	v.osc2.SetFrequency(v.osc2Frequency(), sr)
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
	msg.IfFltVal("osc2Detune", func(f float64) {
		v.cfg.Osc2Detune = f
		v.osc2.SetFrequency(v.osc2Frequency(), v.Config().SampleRate)
	})
	msg.IfIntVal("osc2Octave", func(o int) {
		v.cfg.Osc2Octave = o
		v.osc2.SetFrequency(v.osc2Frequency(), v.Config().SampleRate)
	})
	msg.IfFltVal("pulseWidth", v.SetPulseWidth)
	msg.IfFltVal("noiseLevel", func(f float64) { v.cfg.NoiseLevel = f })
	msg.IfFltVal("drive", func(f float64) { v.cfg.Drive = f })
	msg.IfIntVal("filterMode", func(m int) { v.cfg.FilterMode = FilterMode(m) })
	msg.IfFltVal("cutoff", func(f float64) { v.cfg.Cutoff = f })
	msg.IfFltVal("resonance", func(f float64) { v.cfg.Resonance = f })
	msg.IfFltVal("filterEnvAmount", func(f float64) { v.cfg.FilterEnvAmount = f })
	msg.IfFltVal("keyTrack", func(f float64) { v.cfg.KeyTrack = f })
	msg.IfFltVal("frequency", v.SetFrequency)
	msg.IfFltVal("pitch", v.SetPitch)
}

func (v *VCO) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
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
		return
	}

	cfg := &v.cfg
	outChannels := outSocket.Channels
	bufferSize := v.Config().BufferSize
	keyTrackMult := math.Pow(v.frequency/middleC, cfg.KeyTrack)
	baseCutoff := cfg.Cutoff * keyTrackMult
	drive := cfg.Drive
	if drive <= 0.0 {
		drive = 1.0
	}
	driveComp := 1.0 / math.Tanh(drive)

	for i := v.waitSamps; i < bufferSize; i++ {
		// Oscillators + noise
		s := v.osc1.Tick(0, false, 0)*cfg.Osc1Level + v.osc2.Tick(0, false, 0)*cfg.Osc2Level
		if cfg.NoiseLevel > 0.0 {
			s += (rand.Float64()*2.0 - 1.0) * cfg.NoiseLevel
		}

		// Saturation
		s = math.Tanh(s*drive) * driveComp

		// Filter with envelope modulation in octaves
		v.filter.Set(baseCutoff*math.Pow(2.0, cfg.FilterEnvAmount*v.filterEnv.Generate()), cfg.Resonance)
		low, band, high := v.filter.Tick(s)
		switch cfg.FilterMode {
		case BandPass:
			s = band
		case HighPass:
			s = high
		default:
			s = low
		}

		// Amp
		val := float32(s * v.ampEnv.Generate())
		for _, channel := range outChannels {
			channel[i] = val
		}

		if v.ampEnv.Done() {
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
