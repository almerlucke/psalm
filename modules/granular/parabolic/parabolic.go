package parabolic

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/granular"
)

type Envelope[P granular.Parameter] struct {
	amp   float64
	slope float64
	curve float64
}

func (e *Envelope[P]) New(_ any) *Envelope[P] {
	return &Envelope[P]{}
}

func (e *Envelope[P]) Activate(amplitude float64, durationSamples int64, _ P, _ psalm.Config) {
	e.amp = 0
	rdur := 1.0 / float64(durationSamples)
	rdur2 := rdur * rdur
	e.slope = 4.0 * amplitude * (rdur - rdur2)
	e.curve = -8.0 * amplitude * rdur2
}

func (e *Envelope[P]) Synthesize(bufs [][]float32, bufSize int) {
	for i := range bufSize {
		amp := float32(e.amp)
		e.amp = e.amp + e.slope
		e.slope = e.slope + e.curve
		for _, buf := range bufs {
			buf[i] *= amp
		}
	}
}
