// Package oversample runs a nonlinear process (a waveshaper, a saturating feedback) at a multiple
// of the sample rate, so the harmonics it makes above the audible band are filtered away before
// they fold back as aliasing: up (zero stuffing and a windowed sinc lowpass), the process, down
// (the same lowpass, every factor-th sample)
package oversample

import "math"

// TapsPerPhase is a good length of the lowpass
const TapsPerPhase = 32

// Oversampler is one channel's up and down sampler
type Oversampler struct {
	Factor int
	taps   []float64 // the lowpass, at the oversampled rate
	up     []float64 // the history of the zero stuffed input (a ring of len(taps))
	down   []float64 // the history of the processed samples
	upPos  int
	dnPos  int
	buf    []float64 // the oversampled block of one sample
}

// New creates an oversampler by a factor (2 or more), with a lowpass of tapsPerPhase taps per
// phase: 32 (TapsPerPhase) keeps the band up to 16 kHz within half a dB and stops above 24 kHz,
// where aliases would fold back into the audible band
func New(factor, tapsPerPhase int) *Oversampler {
	if factor < 2 {
		factor = 2
	}
	if tapsPerPhase < 4 {
		tapsPerPhase = 4
	}
	n := factor * tapsPerPhase
	taps := make([]float64, n)
	fc := 0.5 / float64(factor) * 0.907 // 20 kHz of 22.05: the middle of the slope
	mid := float64(n-1) / 2
	sum := 0.0
	for i := range taps {
		x := float64(i) - mid
		sinc := 2 * fc
		if x != 0 {
			sinc = math.Sin(2*math.Pi*fc*x) / (math.Pi * x)
		}
		// a Blackman window
		w := 0.42 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1)) + 0.08*math.Cos(4*math.Pi*float64(i)/float64(n-1))
		taps[i] = sinc * w
		sum += taps[i]
	}
	for i := range taps {
		taps[i] /= sum
	}
	return &Oversampler{Factor: factor, taps: taps, up: make([]float64, n), down: make([]float64, n), buf: make([]float64, factor)}
}

// Latency returns the delay of up and down, in samples at the original rate
func (o *Oversampler) Latency() int { return (len(o.taps) - 1) / o.Factor }

// Process runs f on the oversampled version of one sample and returns the processed sample at the
// original rate
func (o *Oversampler) Process(x float64, f func(float64) float64) float64 {
	n := len(o.taps)
	// up: the sample, then factor-1 zeros, through the lowpass (times factor: the zeros took energy)
	for k := range o.Factor {
		v := 0.0
		if k == 0 {
			v = x * float64(o.Factor)
		}
		o.up[o.upPos] = v
		acc := 0.0
		for j := k; j < n; j += o.Factor { // only the taps on the non-zero samples (polyphase)
			idx := o.upPos - j
			if idx < 0 {
				idx += n
			}
			acc += o.taps[j] * o.up[idx]
		}
		o.upPos = (o.upPos + 1) % n
		o.buf[k] = f(acc)
	}
	// down: the processed samples through the lowpass, keeping one
	for k := range o.Factor {
		o.down[o.dnPos] = o.buf[k]
		o.dnPos = (o.dnPos + 1) % n
	}
	acc := 0.0
	pos := o.dnPos - 1
	for j := range n {
		idx := pos - j
		if idx < 0 {
			idx += n
		}
		acc += o.taps[j] * o.down[idx]
	}
	return acc
}

// Reset clears the history
func (o *Oversampler) Reset() {
	clear(o.up)
	clear(o.down)
}
