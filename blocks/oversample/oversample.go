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
	down   []float64 // the history of the processed samples, written twice (n apart): the last n are always one run
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
	return &Oversampler{Factor: factor, taps: taps, up: make([]float64, n), down: make([]float64, 2*n), buf: make([]float64, factor)}
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
		o.push(o.buf[k])
	}
	return o.filter()
}

// push puts a processed sample in the history
func (o *Oversampler) push(v float64) {
	n := len(o.taps)
	o.down[o.dnPos] = v
	o.down[o.dnPos+n] = v
	if o.dnPos++; o.dnPos == n {
		o.dnPos = 0
	}
}

// filter is the history through the lowpass. The taps are symmetric, so the newest sample meets the
// last of them either way round, and the run of the last n samples is a plain dot product
func (o *Oversampler) filter() float64 {
	n := len(o.taps)
	w := o.down[o.dnPos : o.dnPos+n] // oldest first
	taps := o.taps[:n]
	var a0, a1, a2, a3 float64
	i := 0
	for ; i+4 <= n; i += 4 {
		a0 += taps[i] * w[i]
		a1 += taps[i+1] * w[i+1]
		a2 += taps[i+2] * w[i+2]
		a3 += taps[i+3] * w[i+3]
	}
	for ; i < n; i++ {
		a0 += taps[i] * w[i]
	}
	return (a0 + a1) + (a2 + a3)
}

// Down runs f for each of the factor oversampled points of one sample (k from 0 to factor-1, in
// time) and returns the one sample at the original rate that they make, through the same lowpass:
// for a source that can be asked for its value at any moment (a wave read at a phase), where Process
// would first have to make the points from a sample
func (o *Oversampler) Down(f func(k int) float64) float64 {
	for k := range o.Factor {
		o.push(f(k))
	}
	return o.filter()
}

// Push puts one oversampled point in the history, and Out gives the sample at the original rate that the last Factor points make: Down for a loop that is
// not a function (call Push Factor times, then Out)
func (o *Oversampler) Push(v float64) { o.push(v) }

// Out is the history through the lowpass (see Push)
func (o *Oversampler) Out() float64 { return o.filter() }

// DownDelay is how many samples at the original rate Down delays what it is given: read a wave this far ahead
// and the sample it gives is the wave at the moment
func (o *Oversampler) DownDelay() float64 { return float64(len(o.taps)-1) / (2 * float64(o.Factor)) }

// Reset clears the history
func (o *Oversampler) Reset() {
	clear(o.up)
	clear(o.down)
	o.dnPos = 0
}
