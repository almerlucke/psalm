package adsr

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/granular"
)

type Parameter interface {
	granular.Parameter
	ADSRSetting() *adsr.Setting
}

type Envelope[P Parameter] struct {
	adsr *adsr.ADSR
}

func (e *Envelope[P]) New(cfg any) *Envelope[P] {
	return &Envelope[P]{
		adsr: adsr.New(&adsr.Setting{}, adsr.Automatic, cfg.(psalm.Config).SampleRate),
	}
}

func (e *Envelope[P]) Activate(amplitude float64, durationSamples int64, param P, _ psalm.Config) {
	var (
		inputSetting        = param.ADSRSetting()
		outputSetting       = *inputSetting
		envRelativeDuration float64
		grainDuration       = param.Duration()
		scale               = 1.0
	)

	e.adsr.Clear()

	envRelativeDuration += inputSetting.AttackDuration

	if !inputSetting.SkipDecay {
		envRelativeDuration += inputSetting.DecayDuration
	}

	if !inputSetting.SkipSustain {
		envRelativeDuration += inputSetting.SustainDuration
	}

	envRelativeDuration += inputSetting.ReleaseDuration

	scale = 1.0 / envRelativeDuration

	outputSetting.AttackDuration = grainDuration * 1000 * scale * inputSetting.AttackDuration

	if !inputSetting.SkipDecay {
		outputSetting.DecayDuration = grainDuration * 1000 * scale * inputSetting.DecayDuration
	}

	if !inputSetting.SkipSustain {
		outputSetting.SustainDuration = grainDuration * 1000 * scale * inputSetting.SustainDuration
	}

	outputSetting.ReleaseDuration = grainDuration * 1000 * scale * inputSetting.ReleaseDuration

	e.adsr.TriggerFull(grainDuration, amplitude, &outputSetting, adsr.Automatic)
}

func (e *Envelope[P]) Synthesize(bufs [][]float32, bufSize int) {
	for i := range bufSize {
		amp := float32(e.adsr.Generate())
		for _, buf := range bufs {
			buf[i] *= amp
		}
	}
}
