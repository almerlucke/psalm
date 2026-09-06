package main

import (
	"math"
	"math/rand"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
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
		freq := s.freq + s.pitchEnv.Generate()*220.0
		s.osc.SetFrequency(freq, s.sr)

		for j, outBuf := range outBuffers {
			outBuf[i] = float32(out * s.pan[j])
		}
	}
}

func (s *Source) Activate(sampsToGo int64, p granular.Parameter, cfg psalm.Config) {
	param := p.(*Param)

	duration := timing.SampsToMs(sampsToGo, cfg.SampleRate)

	s.pitchEnvSetting.AttackDuration = duration * 0.01
	s.pitchEnvSetting.ReleaseDuration = duration * 0.05
	s.pitchEnv.TriggerFull(0, 1.0, s.pitchEnvSetting, adsr.Automatic)
	s.freq = param.freq
	s.osc.SetFrequency(param.freq, cfg.SampleRate)
	s.osc.SetAmplitude(param.amplitude)
	s.pan[0] = math.Cos(param.pan * math.Pi / 2.0)
	s.pan[1] = math.Sin(param.pan * math.Pi / 2.0)
}

func (s *Source) New(cfg any) *Source {
	return NewSource(cfg.(psalm.Config).SampleRate)
}

type ParamGen struct {
	parameter    Param // single parameter can be passed for each NextParameter call, prevent from allocating new params
	returnParams [1]granular.Parameter
}

func NewParamGen() *ParamGen {
	g := &ParamGen{}
	g.parameter.adsrSetting = adsr.NewADSetting(1.0, 0.05, 0.95)
	g.parameter.adsrSetting.ReleaseShape = -0.5
	return g
}

func (pgen *ParamGen) Receive(_ any, _ int)           {}
func (pgen *ParamGen) ReceiveMessage(_ psalm.Message) {}

func (pgen *ParamGen) Next(_ int64, config psalm.Config) []granular.Parameter {
	p := &pgen.parameter

	p.duration = rand.Float64()*0.1 + 0.05
	p.amplitude = rand.Float64()*0.4 + 0.2
	p.freq = rand.Float64()*2000.0 + 400.0
	p.onset = int64((rand.Float64()*0.2 + 0.0001) * config.SampleRate)
	p.pan = rand.Float64() * 1.0

	pgen.returnParams[0] = p

	return pgen.returnParams[:]
}

func (pgen *ParamGen) Type() granular.ParameterGeneratorType {
	return granular.Onset
}

func main() {
	p := psalm.New()

	gl := granular.New[*Source, *grainAdsr.Envelope](2, &Source{}, &grainAdsr.Envelope{}, 40, NewParamGen()).AddTo(p)

	p.Out(freeverb.New(0.9, 0.6, 0.9, 0.3, 0.6).AddTo(p).In(gl))

	_ = p.Listen()
}
