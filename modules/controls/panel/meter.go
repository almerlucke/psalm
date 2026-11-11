package panel

import (
	"math"
	"sync/atomic"

	"github.com/almerlucke/psalm"
)

// meterRelease is how fast the peak of a meter falls back, dB per second
const meterRelease = 20.0

// meter measures the output of a module, on the audio thread, for the page to read
type meter struct {
	module psalm.Modular
	peakDB float64       // the falling peak, on the audio thread
	peak   atomic.Uint64 // dB, float64 bits
	rms    atomic.Uint64
}

// measure takes the levels of the buffer the module made last (the patch runs the control ticks
// of a buffer before it renders it, so the output still holds the buffer before)
func (m *meter) measure(sr float64, bufferSize int) {
	out := m.module.Output(0)
	if out == nil {
		return
	}
	peak, sum, n := 0.0, 0.0, 0
	for _, ch := range out.Channels {
		for _, x := range ch {
			a := math.Abs(float64(x))
			peak = max(peak, a)
			sum += a * a
			n++
		}
	}
	db := func(g float64) float64 {
		if g <= 1e-6 {
			return -120
		}
		return 20 * math.Log10(g)
	}
	rms := -120.0
	if n > 0 {
		rms = db(math.Sqrt(sum / float64(n)))
	}
	fall := m.peakDB - meterRelease*float64(bufferSize)/sr
	m.peakDB = max(db(peak), max(fall, -120))
	m.peak.Store(math.Float64bits(m.peakDB))
	m.rms.Store(math.Float64bits(rms))
}

func (m *meter) levels() (float64, float64) {
	return math.Float64frombits(m.peak.Load()), math.Float64frombits(m.rms.Load())
}
