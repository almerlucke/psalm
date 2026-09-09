package main

import (
	"math"
	"math/rand"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/genny/float/rwalk"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/granular"
	grainAdsr "github.com/almerlucke/psalm/modules/granular/adsr"
	"github.com/almerlucke/psalm/tools/timing"
)

type Param struct {
	onset       int64
	duration    float64
	amplitude   float64
	freq        float64
	pan         float64
	adsrSetting *adsr.Setting
}

func (p *Param) Onset() int64 {
	return p.onset
}

func (p *Param) Duration() float64 {
	return p.duration
}

func (p *Param) Amplitude() float64 {
	return p.amplitude
}

func (p *Param) ADSRSetting() *adsr.Setting {
	return p.adsrSetting
}

type Source struct {
	osc             *blosc.Blosc
	pitchEnv        *adsr.ADSR
	pitchEnvSetting *adsr.Setting
	freq            float64
	sr              float64
	pan             [2]float64
}

func NewSource(sr float64) *Source {
	return &Source{
		osc:             blosc.New(100.0, sr, 0.0, 0.0, 1.0, blosc.Sine),
		pitchEnv:        adsr.New(nil, adsr.Automatic, sr),
		pitchEnvSetting: adsr.NewADSetting(1.0, 0.1, 1.0),
		sr:              sr,
	}
}

func (s *Source) Synthesize(outBuffers [][]float32, bufSize int) {
	for i := range bufSize {
		out := s.osc.Tick(0, false, 0)
		freq := s.freq + s.pitchEnv.Generate()*20.0
		s.osc.SetFrequency(freq, s.sr)

		for j, outBuf := range outBuffers {
			outBuf[i] = float32(out * s.pan[j])
		}
	}
}

func (s *Source) Activate(sampsToGo int64, p *Param, cfg psalm.Config) {
	duration := timing.SampsToMs(sampsToGo, cfg.SampleRate)

	s.pitchEnvSetting.AttackDuration = duration * 0.01
	s.pitchEnvSetting.ReleaseDuration = duration * 0.05
	s.pitchEnv.TriggerFull(0, 1.0, s.pitchEnvSetting, adsr.Automatic)
	s.freq = p.freq
	s.osc.SetFrequency(p.freq, cfg.SampleRate)
	s.osc.SetAmplitude(p.amplitude)
	s.pan[0] = math.Cos(p.pan * math.Pi / 2.0)
	s.pan[1] = math.Sin(p.pan * math.Pi / 2.0)
}

func (s *Source) New(cfg any) *Source {
	return NewSource(cfg.(psalm.Config).SampleRate)
}

type ParamGen struct {
	granular.ParameterGenerator[*Param]
	parameter Param // single parameter can be passed for each NextParameter call, prevent from allocating new params
	freqGen   genny.Generator[float64]
	panGen    genny.Generator[float64]
}

func NewParamGen() *ParamGen {
	g := &ParamGen{}
	g.parameter.adsrSetting = adsr.NewADSetting(1.0, 0.0025, 0.95)
	g.parameter.adsrSetting.ReleaseShape = -0.5
	g.panGen = rwalk.New(0.5, 0.0, 1.0, 0.01, rwalk.Bounce)
	g.freqGen = sequence.NewLoop(
		// Phrase 1: Solemn Invocation & Multi-Octave Ascent (Notes 1 - 25)
		146.83, 220.00, 293.66, 369.99, 440.00, 587.33, 554.37, 587.33,
		739.99, 880.00, 1174.66, 1108.73, 987.77, 880.00, 739.99, 659.25,
		587.33, 440.00, 369.99, 293.66, 440.00, 587.33, 739.99, 880.00, 880.00,

		// Phrase 2: Celestial Exultation & Soaring Apex (Notes 26 - 50)
		880.00, 1174.66, 1318.51, 1479.98, 1760.00, 2349.32, 2217.46, 1760.00,
		1479.98, 1760.00, 1975.53, 1760.00, 1479.98, 1318.51, 1174.66, 987.77,
		1174.66, 1479.98, 1318.51, 1174.66, 880.00, 987.77, 1174.66, 1318.51, 1174.66,

		// Phrase 3: Deep Contemplation & Antiphonal Leaps (Notes 51 - 75)
		587.33, 293.66, 146.83, 196.00, 220.00, 293.66, 440.00, 739.99,
		659.25, 587.33, 369.99, 246.94, 293.66, 392.00, 493.88, 587.33,
		880.00, 1108.73, 1174.66, 1479.98, 1318.51, 987.77, 880.00, 739.99, 587.33,

		// Phrase 4: Majestic Full-Range Climax & Final Grounding (Notes 76 - 100)
		440.00, 587.33, 739.99, 880.00, 1174.66, 1479.98, 1760.00, 1479.98,
		1174.66, 987.77, 880.00, 739.99, 587.33, 659.25, 587.33, 440.00,
		369.99, 293.66, 220.00, 146.83, 220.00, 293.66, 369.99, 293.66, 146.83,
	)

	return g
}

func (pgen *ParamGen) NextOnset(_ int64, config psalm.Config) *Param {
	p := &pgen.parameter

	p.duration = rand.Float64()*2.5 + 0.1
	p.amplitude = rand.Float64()*0.4 + 0.1
	p.freq = pgen.freqGen.Generate()
	p.onset = int64((rand.Float64()*0.25 + 0.0001) * config.SampleRate)
	p.pan = pgen.panGen.Generate()

	return p
}

func (pgen *ParamGen) Type() granular.ParameterGeneratorType {
	return granular.Onset
}

func main() {
	p := psalm.New()

	gl := granular.New[*Param, *Source, *grainAdsr.Envelope[*Param]](
		2, &Source{},
		&grainAdsr.Envelope[*Param]{},
		40,
		NewParamGen(),
	).AddTo(p)

	p.Out(freeverb.New(0.9, 0.5, 0.95, 0.3, 0.6).AddTo(p).In(
		pingpong.New(4000.0, 2000.0, 0.1, 0.175).AddTo(p).In(gl),
	))

	_ = p.Listen()
}
