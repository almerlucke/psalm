package wavetable

import (
	"github.com/almerlucke/psalm/blocks/sine"
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/expression"
	"github.com/almerlucke/psalm/blocks/filters/svf"
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

// Voice is a wavetable synth voice. Value inputs: 1 position, 2 cutoff, 3 resonance. They glide
// (the position over a few milliseconds, the filter over a buffer), so modulating them never clicks
type Voice struct {
	*psalm.Module
	cfg         Config
	ampEnv      *adsr.ADSR
	filterEnv   *adsr.ADSR
	positionEnv *adsr.ADSR
	filters     [2]*svf.SVF

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
	count     int // samples until the next filter coefficient update
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
	case 1:
		v.cfg.Position = f
	case 2:
		v.cfg.Cutoff = f
	case 3:
		v.cfg.Resonance = f
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
		// unison oscillators start at random phases so they don't start out as one loud wave
		for u := range v.phases {
			v.phases[u] = 0
			if u > 0 {
				v.phases[u] = v.Rand().Float64()
			}
		}
		v.subPhase = 0
		v.vibPhase = 0
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
	)
	if table == nil {
		table = basic
	}
	if level <= 0 {
		level = 1
	}

	// unison: detune ratios and pans, spread evenly, the sum keeps the level of one oscillator
	n := max(1, min(cfg.Unison, MaxUnison))
	var ratios, gainL, gainR [MaxUnison]float64
	norm := 1.0 / math.Sqrt(float64(n))
	for u := range n {
		offset := 0.0
		if n > 1 {
			offset = float64(u)/float64(n-1) - 0.5
		}
		ratios[u] = math.Exp2(offset * cfg.Detune / 1200.0)
		pan := 0.5 + offset*cfg.Spread
		gainL[u] = math.Cos(pan*math.Pi/2) * math.Sqrt2 * norm
		gainR[u] = math.Sin(pan*math.Pi/2) * math.Sqrt2 * norm
		if !stereo {
			gainL[u] = norm
		}
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

		// pitch with glide and vibrato, and the bend
		v.expr.Tick(sr)
		v.glideFreq += (v.frequency - v.glideFreq) * glideCoef
		freq := v.glideFreq * v.expr.Ratio()
		if cfg.VibratoDepth != 0 && cfg.VibratoRate > 0 {
			freq *= math.Exp2(cfg.VibratoDepth * sine.Cycle(v.vibPhase) / 12.0)
			v.vibPhase += cfg.VibratoRate / sr
			v.vibPhase -= math.Floor(v.vibPhase)
		}

		// position: the smoothed setting plus envelope and LFO
		v.position += (cfg.Position - v.position) * posCoef
		pos := v.position + cfg.PositionEnvAmount*penv + 0.6*v.expr.Wheel()
		if cfg.PositionLFODepth != 0 && cfg.PositionLFORate > 0 {
			pos += cfg.PositionLFODepth * sine.Cycle(v.lfoPhase)
			v.lfoPhase += cfg.PositionLFORate / sr
			v.lfoPhase -= math.Floor(v.lfoPhase)
		}

		// oscillators, the highest unison oscillator picks the band limited level for all
		inc := freq / sr
		c := table.cursor(pos, inc*ratios[n-1])
		frameGain := 1.0
		if cfg.Normalize {
			frameGain = c.gain
		}
		var l, r float64
		for u := range n {
			x := c.read(v.phases[u]) * frameGain
			l += x * gainL[u]
			r += x * gainR[u]
			v.phases[u] += inc * ratios[u]
			v.phases[u] -= math.Floor(v.phases[u])
		}
		if cfg.SubLevel > 0 {
			sub := cfg.SubLevel * sine.Cycle(v.subPhase)
			l += sub
			r += sub
		}
		v.subPhase += inc * 0.5
		v.subPhase -= math.Floor(v.subPhase)

		// filter
		if cfg.Filter != FilterOff {
			v.cutoff += cutoffInc
			v.res += resInc
			if v.count == 0 {
				fc := v.cutoff * keyTrack * math.Exp2(cfg.FilterEnvAmount*fenv+1.5*v.expr.Wheel())
				for _, f := range v.filters {
					f.Set(fc, v.res)
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

		g := amp * level
		outs[0][i] = float32(l * g)
		if stereo {
			outs[1][i] = float32(r * g)
		}
	}

	v.cutoff = cfg.Cutoff
	v.res = cfg.Resonance
	v.waitSamps = 0
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
