package subtractive

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/formant"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/modules/poly"
)

// FilterType selects the filter of the voice
type FilterType int

const (
	FilterSVF     FilterType = iota // state variable filter, 12 dB/oct, modes set by FilterMode
	FilterLadder                    // Moog style ladder, 24 or 12 dB/oct, modes set by LadderMode
	FilterFormant                   // vowel filter, the cutoff modulation moves the vowel instead
)

// Filter2Mode selects the second filter
type Filter2Mode int

const (
	Filter2Off Filter2Mode = iota
	Filter2Low
	Filter2Band
	Filter2High
	Filter2Notch
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
	Unison       int     // oscillator pairs per voice, 1-7 (0 is 1): detuned copies of both oscillators
	UnisonDetune float64 // cents between the lowest and the highest of them
	Drift        float64 // cents, a slow random wander of the pitch (an analog oscillator is never still)

	// Pitch
	Glide            float64 // portamento time in milliseconds from the previous note of this voice
	GlideLegato      bool    // glide only when the voice is still sounding as it gets the note (a stolen voice, a note played over another), a free voice jumps
	FilterEnvToPitch float64 // semitones of pitch sweep driven by the filter envelope

	// Filter
	FilterType       FilterType
	FilterMode       FilterMode  // SVF output
	LadderMode       ladder.Mode // ladder output
	FilterDrive      float64     // ladder input saturation (0 is treated as 1)
	FilterOversample int         // the ladder runs once (0), twice (1) or four times (2) a sample, so the harmonics of its saturation do not fold back
	Cutoff           float64     // Hz
	Resonance        float64     // 0-1, for the formant filter it sets the sharpness of the formants
	FilterEnvAmount  float64     // octaves added to cutoff at full envelope (vowel steps for the formant filter)
	KeyTrack         float64     // 0-1, 1 means cutoff follows pitch fully (relative to middle C)

	// A second filter, a state variable one after the first (serial) or beside it (parallel): the
	// two bands of a split, a low pass with a high pass for a band, a notch for a hole
	Filter2Mode      Filter2Mode
	Filter2Cutoff    float64 // Hz
	Filter2Resonance float64 // 0-1
	Filter2EnvAmount float64 // octaves the filter envelope adds to its cutoff
	Filter2KeyTrack  float64 // 0-1
	Filter2Parallel  bool    // beside the first filter (the two mixed) instead of after it

	// Formant filter
	Vowel        float64 // 0-4: a e i o u, fractions morph. Envelope and LFO cutoff amounts move the vowel
	FormantShift float64 // scales the formant frequencies (0 is treated as 1)

	// Envelopes (durations in milliseconds)
	FilterEnv adsr.Setting
	AmpEnv    adsr.Setting

	// Modulation
	LFO1 LFOConfig
	LFO2 LFOConfig
	Mods [modmatrix.Slots]modmatrix.Slot // the matrix: a source to a destination (see Dests) over an amount
	Ext  [modmatrix.Exts]float64         // the values 0-1 of the matrix's sources Ext 1 to 4 (cabled in on InExt1...)

	Level float64 // output level (0 is treated as 1)
}

func DefaultConfig() Config {
	return Config{
		NumChannels:      psalm.Stereo,
		FilterOversample: 1, // the ladder twice a sample: its saturation does not fold back
		Osc1Waveform:     blosc.Sawtooth,
		Osc2Waveform:     blosc.Sawtooth,
		Osc1Level:        0.5,
		Osc2Level:        0.5,
		Osc2Detune:       0.07,
		PulseWidth:       0.5,
		Drive:            1.0,
		FilterMode:       LowPass,
		Cutoff:           800.0,
		Resonance:        0.3,
		FilterEnvAmount:  3.0,
		KeyTrack:         0.5,
		FilterEnv:        *adsr.NewSetting(1.0, 5.0, 0.3, 300.0, 1.0, 400.0),
		AmpEnv:           *adsr.NewSetting(1.0, 5.0, 0.7, 200.0, 1.0, 300.0),
	}
}

const middleC = 261.6255653005986

// maxUnison is the most oscillator pairs a voice plays
const maxUnison = 7

// The destinations of the matrix (see Dests), by number
const (
	dPitch = iota
	dCutoff
	dResonance
	dPulseWidth
	dOsc1Level
	dOsc2Level
	dSubLevel
	dNoiseLevel
	dOsc2Detune
	dFM
	dDrive
	dVowel
	dLevel
	dLFO1Rate
	dLFO2Rate
	dUnisonDetune
	dCutoff2
	numDests
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (semitones, octaves, or the setting itself)
var Dests = []modmatrix.Dest{
	dPitch:        {Key: "pitch", Label: "pitch", Span: 24},
	dCutoff:       {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance:    {Key: "resonance", Label: "resonance", Span: 1},
	dPulseWidth:   {Key: "pulseWidth", Label: "pulse width", Span: 0.45},
	dOsc1Level:    {Key: "osc1Level", Label: "level 1", Span: 1},
	dOsc2Level:    {Key: "osc2Level", Label: "level 2", Span: 1},
	dSubLevel:     {Key: "subLevel", Label: "sub", Span: 1},
	dNoiseLevel:   {Key: "noiseLevel", Label: "noise", Span: 1},
	dOsc2Detune:   {Key: "osc2Detune", Label: "detune 2", Span: 12},
	dFM:           {Key: "fmAmount", Label: "fm", Span: 5},
	dDrive:        {Key: "drive", Label: "drive", Span: 3},
	dVowel:        {Key: "vowel", Label: "vowel", Span: 4},
	dLevel:        {Key: "level", Label: "level", Span: 1},
	dLFO1Rate:     {Key: "lfo1Rate", Label: "lfo 1 rate", Span: 3},
	dLFO2Rate:     {Key: "lfo2Rate", Label: "lfo 2 rate", Span: 3},
	dUnisonDetune: {Key: "unisonDetune", Label: "unison detune", Span: 50},
	dCutoff2:      {Key: "filter2Cutoff", Label: "cutoff 2", Span: 6},
}

// The value inputs of a voice: 1 to 4 set the cutoff, the resonance, the pulse width and the vowel,
// 5 to 8 the values of the matrix (Ext 1 to 4) that a slot can move any setting by
const (
	InCutoff     = 1
	InResonance  = 2
	InPulseWidth = 3
	InVowel      = 4
	InExt1       = 5 // to InExt1+3
)

// ladderStageLevel is where the stages of the ladder of the synth clip: a signal of normal level feels it a little (see ladder.SetStageLevel)
const ladderStageLevel = 1.25

type Voice struct {
	*psalm.Module
	cfg        Config
	osc1       *blosc.Blosc
	osc2       *blosc.Blosc
	sub        *blosc.Blosc
	noise      noise
	lfo1       lfo
	lfo2       lfo
	svf        *svf.SVF
	svf2       *svf.SVF
	ladder     *ladder.Ladder
	formant    *formant.Formant
	ladderDrv  float64
	ladderOver int                     // how many times a sample the ladder runs (set in the ladder)
	drvOver    *oversample.Oversampler // the saturation of the oscillators (Drive) runs this many times a sample, with the ladder (FilterOversample); nil for once
	drvD       float64                 // the drive of the sample in the oversampled run, and the gain that brings it back
	drvComp    float64
	drvShape   func(float64) float64
	formShift  float64
	formCount  int // counts samples until the next filter coefficient update
	filterEnv  *adsr.ADSR
	ampEnv     *adsr.ADSR
	frequency  float64 // target frequency
	glideFreq  float64 // current frequency, glides towards frequency
	cutoff     float64 // smoothed cutoff, ramps towards cfg.Cutoff over a buffer
	res        float64 // smoothed resonance, ramps towards cfg.Resonance over a buffer
	vowel      float64 // smoothed vowel, ramps towards cfg.Vowel over a buffer
	waitSamps  int
	expr       expression.Expression // the pitch bend; the mod wheel opens the filter
	mods       *modmatrix.Matrix
	uni1       [maxUnison - 1]*blosc.Blosc // the extra oscillators of unison
	uni2       [maxUnison - 1]*blosc.Blosc
	drift      float64 // -1 to 1, wanders
	driftTo    float64
	driftLeft  int
	velocity   float64 // the amplitude of the note
	random     float64 // a random value for the note, -1 to 1
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
	for k := range v.uni1 {
		// spread phases (the golden ratio), without drawing from the voice's random stream
		v.uni1[k] = blosc.New(v.frequency, sr, math.Mod(0.618034*float64(k+1), 1), cfg.PulseWidth, 1.0, cfg.Osc1Waveform)
		v.uni2[k] = blosc.New(v.frequency, sr, math.Mod(0.381966*float64(k+2), 1), cfg.PulseWidth, 1.0, cfg.Osc2Waveform)
	}
	v.svf = svf.New(cfg.Cutoff, cfg.Resonance, sr)
	v.svf2 = svf.New(max(20, cfg.Filter2Cutoff), cfg.Filter2Resonance, sr)
	v.ladder = ladder.New(cfg.Cutoff, cfg.Resonance, sr)
	v.ladder.SetStageLevel(ladderStageLevel)
	v.ladderOver = 1
	v.drvShape = func(x float64) float64 { return math.Tanh(x*v.drvD) * v.drvComp }
	v.formant = formant.New(cfg.Vowel, formantQ(cfg.Resonance), sr)
	v.filterEnv = adsr.New(&v.cfg.FilterEnv, adsr.NoteOff, sr)
	v.ampEnv = adsr.New(&v.cfg.AmpEnv, adsr.NoteOff, sr)

	v.lfo1.rng = v.Rand()
	v.lfo2.rng = v.Rand()
	v.noise.rng = v.Rand()

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
	v.mods.Receive(msg)
	for i := range v.cfg.Mods {
		v.cfg.Mods[i] = v.mods.Slot(i)
	}
	for i := range v.cfg.Ext {
		v.cfg.Ext[i] = v.mods.Ext(i)
	}
	envelope.Receive(msg, "amp", &v.cfg.AmpEnv)
	envelope.Receive(msg, "filter", &v.cfg.FilterEnv)
	msg.IfIntVal("osc1Waveform", func(wf int) {
		v.cfg.Osc1Waveform = blosc.Waveform(wf)
		v.osc1.SetWaveform(v.cfg.Osc1Waveform)
		for _, u := range v.uni1 {
			u.SetWaveform(v.cfg.Osc1Waveform)
		}
	})
	msg.IfIntVal("osc2Waveform", func(wf int) {
		v.cfg.Osc2Waveform = blosc.Waveform(wf)
		v.osc2.SetWaveform(v.cfg.Osc2Waveform)
		for _, u := range v.uni2 {
			u.SetWaveform(v.cfg.Osc2Waveform)
		}
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
	msg.IfIntVal("unison", func(n int) { v.cfg.Unison = max(1, min(n, maxUnison)) })
	msg.IfFltVal("unisonDetune", func(f float64) { v.cfg.UnisonDetune = max(0, f) })
	msg.IfFltVal("drift", func(f float64) { v.cfg.Drift = max(0, f) })
	msg.IfFltVal("glide", func(f float64) { v.cfg.Glide = f })
	msg.IfFltVal("filterEnvToPitch", func(f float64) { v.cfg.FilterEnvToPitch = f })
	receiveLFOMessage(msg, "lfo1", &v.cfg.LFO1)
	receiveLFOMessage(msg, "lfo2", &v.cfg.LFO2)
	msg.IfIntVal("filterType", func(t int) { v.cfg.FilterType = FilterType(t) })
	msg.IfIntVal("filterMode", func(m int) { v.cfg.FilterMode = FilterMode(m) })
	msg.IfIntVal("ladderMode", func(m int) { v.cfg.LadderMode = ladder.Mode(m) })
	msg.IfFltVal("filterDrive", func(f float64) { v.cfg.FilterDrive = f })
	msg.IfIntVal("filterOversample", func(n int) { v.cfg.FilterOversample = max(0, min(n, 2)) })
	msg.IfFltVal("vowel", func(f float64) { v.cfg.Vowel = f })
	msg.IfFltVal("formantShift", func(f float64) { v.cfg.FormantShift = f })
	msg.IfFltVal("cutoff", func(f float64) { v.cfg.Cutoff = f })
	msg.IfFltVal("resonance", func(f float64) { v.cfg.Resonance = f })
	msg.IfFltVal("filterEnvAmount", func(f float64) { v.cfg.FilterEnvAmount = f })
	msg.IfFltVal("keyTrack", func(f float64) { v.cfg.KeyTrack = f })
	msg.IfIntVal("filter2Mode", func(m int) { v.cfg.Filter2Mode = Filter2Mode(max(0, min(m, int(Filter2Notch)))) })
	msg.IfFltVal("filter2Cutoff", func(f float64) { v.cfg.Filter2Cutoff = max(20, f) })
	msg.IfFltVal("filter2Resonance", func(f float64) { v.cfg.Filter2Resonance = max(0, min(f, 1)) })
	msg.IfFltVal("filter2EnvAmount", func(f float64) { v.cfg.Filter2EnvAmount = f })
	msg.IfFltVal("filter2KeyTrack", func(f float64) { v.cfg.Filter2KeyTrack = f })
	msg.IfFltVal("filter2Parallel", func(f float64) { v.cfg.Filter2Parallel = f > 0.5 })
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
// and 4 (vowel), which makes it easy to connect LFOs; and on 5 to 8 (InExt1...) values 0-1 that the
// matrix takes as sources Ext 1 to 4, to move any setting it has a slot for
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
	v.ReceiveMessage(message)
	if v.cfg.GlideLegato && !wasActive {
		v.glideFreq = v.frequency
	}
	if !wasActive && v.drvOver != nil {
		v.drvOver.Reset() // what is left in the filter of the note before
	}

	v.waitSamps = tick.SampleOffset
	v.velocity = amplitude
	// drawn only when a slot uses it: a voice that doesn't keeps the random stream it had
	if v.mods.Uses(modmatrix.Random) {
		v.random = v.Rand().Float64()*2 - 1
	}

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
	v.svf2.Clear()
	v.ladder.Clear()
	v.formant.Clear()
	if v.drvOver != nil {
		v.drvOver.Reset()
	}
	v.formCount = 0
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool {
	return !v.ampEnv.Done()
}

// syncDriveOversample makes the sampler of the saturation of the oscillators to go with the oversampling of the ladder, FilterOversample
func (v *Voice) syncDriveOversample() {
	factor := 1 << max(0, min(v.cfg.FilterOversample, 2))
	switch {
	case factor == 1:
		v.drvOver = nil
	case v.drvOver == nil || v.drvOver.Factor != factor:
		v.drvOver = oversample.New(factor, oversample.TapsPerPhase)
	}
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
	v.syncDriveOversample()
	keyTrackMult := math.Pow(v.glideFreq/middleC, cfg.KeyTrack)
	// cutoff, resonance and vowel glide over the buffer, so changing them (for example from an LFO
	// on the value inputs) doesn't step and click
	n := float64(bufferSize - v.waitSamps)
	cutoffInc := (cfg.Cutoff - v.cutoff) / n
	resInc := (cfg.Resonance - v.res) / n
	vowelInc := (cfg.Vowel - v.vowel) / n
	osc2Ratio := v.osc2Ratio()
	mods := v.mods
	modded := mods.Active()
	lfo1Active := cfg.LFO1.active() || mods.Uses(modmatrix.LFO1)
	lfo2Active := cfg.LFO2.active() || mods.Uses(modmatrix.LFO2)
	pwModded := modded && mods.Targets(dPulseWidth)
	detuneModded := modded && mods.Targets(dOsc2Detune)
	driveModded := modded && mods.Targets(dDrive)
	filter2 := cfg.Filter2Mode != Filter2Off
	svf2Mode := [...]svf.Mode{svf.LowPass, svf.LowPass, svf.BandPass, svf.HighPass, svf.Notch}[cfg.Filter2Mode]
	unison := max(1, min(cfg.Unison, maxUnison))
	uniDetune := cfg.UnisonDetune
	if modded {
		uniDetune = max(0, uniDetune+mods.Mod(dUnisonDetune))
	}
	var uniMul [maxUnison - 1]float64
	for k := range unison - 1 { // alternately above and below, evenly over the detune
		off := float64(k/2+1) / float64(unison/2)
		if k%2 == 1 {
			off = -off
		}
		uniMul[k] = math.Exp2(off * uniDetune / 2 / 1200)
	}
	uniGain := 1 / math.Sqrt(float64(unison))
	drive := cfg.Drive
	if drive <= 0.0 {
		drive = 1.0
	}
	driveComp := 1.0 / math.Tanh(drive)
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	if modded {
		mods.Values[modmatrix.Velocity] = v.velocity
		mods.Values[modmatrix.Random] = v.random
		mods.Values[modmatrix.Key] = max(-1, min(math.Log2(v.glideFreq/middleC)/5, 1))
	}

	glideCoef := 1.0
	if cfg.Glide > 0.0 {
		glideCoef = 1.0 - math.Exp(-1.0/(cfg.Glide*0.001*sr))
	}

	for i := v.waitSamps; i < bufferSize; i++ {
		v.expr.Tick(sr)
		// Modulation sources
		fenv := v.filterEnv.Generate()

		amp := v.ampEnv.Generate()

		var l1, l2 float64
		if lfo1Active {
			l1 = v.lfo1.tick(&cfg.LFO1, sr, cfg.LFO1.Rate*math.Exp2(mods.Mod(dLFO1Rate)))
		}
		if lfo2Active {
			l2 = v.lfo2.tick(&cfg.LFO2, sr, cfg.LFO2.Rate*math.Exp2(mods.Mod(dLFO2Rate)))
		}
		if modded {
			mv := &mods.Values
			mv[modmatrix.FilterEnv], mv[modmatrix.AmpEnv] = fenv, amp
			mv[modmatrix.LFO1], mv[modmatrix.LFO2] = l1, l2
			mv[modmatrix.Wheel] = v.expr.Wheel()
			mods.Eval(sr)
		}

		// Pitch: glide, LFO vibrato and filter envelope pitch sweep
		v.glideFreq += (v.frequency - v.glideFreq) * glideCoef
		freq := v.glideFreq
		if semis := cfg.LFO1.Pitch*l1 + cfg.LFO2.Pitch*l2 + cfg.FilterEnvToPitch*fenv + v.expr.Semitones() + mods.Mod(dPitch) + v.wander(cfg.Drift, sr); semis != 0.0 {
			freq *= math.Pow(2.0, semis/12.0)
		}

		ratio := osc2Ratio
		if detuneModded {
			ratio *= math.Exp2(mods.Mod(dOsc2Detune) / 12)
		}
		v.osc1.SetFrequency(freq, sr)
		v.osc2.SetFrequency(freq*ratio, sr)

		// Pulse width modulation
		if lfo1Active || lfo2Active || pwModded {
			pw := max(0.02, min(0.98, cfg.PulseWidth+cfg.LFO1.PulseWidth*l1+cfg.LFO2.PulseWidth*l2+mods.Mod(dPulseWidth)))
			v.osc1.SetPulseWidth(pw)
			v.osc2.SetPulseWidth(pw)
			for k := range unison - 1 {
				v.uni1[k].SetPulseWidth(pw)
				v.uni2[k].SetPulseWidth(pw)
			}
		}

		// Oscillators: osc 2 can be phase modulated by and hard synced to osc 1
		prevPhase := v.osc1.Phase()
		o1 := v.osc1.Tick(0, false, 0)
		phase := v.osc1.Phase()
		wrapped := phase < prevPhase

		fm := 0.0
		if fmAmount := cfg.FMAmount + mods.Mod(dFM); fmAmount != 0.0 {
			fm = fmAmount * o1 * freq * ratio / sr
		}

		syncPhase := 0.0
		if cfg.Sync && wrapped {
			syncPhase = phase * ratio
			syncPhase -= math.Floor(syncPhase)
		}

		o2 := v.osc2.Tick(fm, cfg.Sync && wrapped, syncPhase)

		// unison: detuned copies of both oscillators (free running), summed with the pair
		m1, m2 := o1, o2
		if unison > 1 {
			for k := range unison - 1 {
				v.uni1[k].SetFrequency(freq*uniMul[k], sr)
				v.uni2[k].SetFrequency(freq*ratio*uniMul[k], sr)
				m1 += v.uni1[k].Tick(0, false, 0)
				m2 += v.uni2[k].Tick(0, false, 0)
			}
			m1, m2 = m1*uniGain, m2*uniGain
		}

		s := m1*max(0, cfg.Osc1Level+mods.Mod(dOsc1Level)) + m2*max(0, cfg.Osc2Level+mods.Mod(dOsc2Level))

		if sub := cfg.SubLevel + mods.Mod(dSubLevel); sub > 0.0 {
			v.sub.SetFrequency(freq*0.5, sr)
			s += v.sub.Tick(0, false, 0) * sub
		}
		if cfg.RingModLevel > 0.0 {
			s += o1 * o2 * cfg.RingModLevel
		}
		if noiseLevel := cfg.NoiseLevel + mods.Mod(dNoiseLevel); noiseLevel > 0.0 {
			s += v.noise.tick(cfg.NoiseType) * noiseLevel
		}

		// Saturation, played twice or four times a sample with the ladder when the drive is more than a touch
		if driveModded {
			d := drive * math.Exp2(mods.Mod(dDrive))
			if v.drvOver != nil {
				v.drvD, v.drvComp = d, 1/math.Tanh(d)
				s = v.drvOver.Process(s, v.drvShape)
			} else {
				s = math.Tanh(s*d) / math.Tanh(d)
			}
		} else if v.drvOver != nil && drive > 1.05 {
			v.drvD, v.drvComp = drive, driveComp
			s = v.drvOver.Process(s, v.drvShape)
		} else {
			s = math.Tanh(s*drive) * driveComp
		}

		// Filter with envelope and LFO modulation in octaves (in vowel steps for the formant filter)
		v.cutoff += cutoffInc
		v.res += resInc
		v.vowel += vowelInc
		if v.formCount == 0 {
			mod := cfg.FilterEnvAmount*fenv + cfg.LFO1.Cutoff*l1 + cfg.LFO2.Cutoff*l2 + 2*v.expr.Wheel() + mods.Mod(dCutoff)
			v.setFilter(v.cutoff*keyTrackMult*math.Exp2(mod), mod, max(0, min(v.res+mods.Mod(dResonance), 1)), v.vowel+mods.Mod(dVowel))
			if filter2 {
				c2 := max(20, cfg.Filter2Cutoff) * math.Pow(v.glideFreq/middleC, cfg.Filter2KeyTrack) * math.Exp2(cfg.Filter2EnvAmount*fenv+mods.Mod(dCutoff2))
				v.svf2.Set(max(20, min(c2, 0.45*sr)), cfg.Filter2Resonance)
			}
		}
		v.formCount = (v.formCount + 1) % filterUpdate
		if filter2 {
			in := s
			s = v.processFilter(in)
			if cfg.Filter2Parallel {
				s = 0.5 * (s + v.svf2.Process(in, svf2Mode))
			} else {
				s = v.svf2.Process(s, svf2Mode)
			}
		} else {
			s = v.processFilter(s)
		}

		// Amp with LFO tremolo
		if cfg.LFO1.Amp != 0.0 {
			amp *= 1.0 - cfg.LFO1.Amp*(1.0-l1)*0.5
		}
		if cfg.LFO2.Amp != 0.0 {
			amp *= 1.0 - cfg.LFO2.Amp*(1.0-l2)*0.5
		}

		val := float32(s * amp * max(0, level+mods.Mod(dLevel)))
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

// wander moves the drift: a new place to go to every quarter second, gliding there; it returns the
// semitones it moves the pitch (cents of drift)
func (v *Voice) wander(cents, sr float64) float64 {
	if cents <= 0 {
		return 0
	}
	if v.driftLeft <= 0 {
		v.driftTo = v.Rand().Float64()*2 - 1
		v.driftLeft = int(0.25 * sr)
	}
	v.driftLeft--
	v.drift += (v.driftTo - v.drift) * (1 / (0.2 * sr))
	return v.drift * cents / 100
}

// resetSmoothing sets the smoothed filter settings to their targets
func (v *Voice) resetSmoothing() {
	v.cutoff = v.cfg.Cutoff
	v.res = v.cfg.Resonance
	v.vowel = v.cfg.Vowel
}

// formantQ maps resonance 0-1 to the formant sharpness
func formantQ(res float64) float64 {
	return 4.0 + max(0.0, min(res, 1.0))*16.0
}

// filterUpdate is the interval in samples at which the filter coefficients are recalculated
// (about 11 kHz at 44.1 kHz). Calculating them every sample costs a lot (tan and exp) and is far
// more often than any modulation needs
const filterUpdate = 4

// setFilter updates the coefficients of the selected filter
func (v *Voice) setFilter(cutoff, mod, res, vowel float64) {
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
		if over := 1 << max(0, min(cfg.FilterOversample, 2)); over != v.ladderOver {
			v.ladderOver = over
			v.ladder.SetOversample(over)
		}
		v.ladder.Set(cutoff, res)

	case FilterFormant:
		shift := cfg.FormantShift
		if shift <= 0.0 {
			shift = 1.0
		}
		if shift != v.formShift {
			v.formShift = shift
			v.formant.SetShift(shift)
		}
		v.formant.Set(vowel+mod, formantQ(res))

	default:
		v.svf.Set(cutoff, res)
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
