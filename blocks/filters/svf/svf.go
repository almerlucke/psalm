package svf

import "math"

// Mode selects the output of Process
type Mode int

const (
	LowPass Mode = iota
	BandPass
	HighPass
	Notch
	Peak
	AllPass
)

// SVF is a zero-delay feedback (TPT) state variable filter after Andrew Simper (Cytomic)
type SVF struct {
	sr    float64
	fc    float64
	res   float64
	q     float64 // when > 0 the damping is set from Q instead of resonance
	g     float64
	k     float64
	a1    float64
	a2    float64
	a3    float64
	ic1eq float64
	ic2eq float64
}

// New creates a new state variable filter, resonance is in the range 0-1
func New(fc float64, res float64, sr float64) *SVF {
	f := &SVF{sr: sr, fc: fc, res: res}
	f.update()
	return f
}

func (f *SVF) SetSampleRate(sr float64) {
	f.sr = sr
	f.update()
}

func (f *SVF) SetFrequency(fc float64) {
	f.fc = fc
	f.update()
}

func (f *SVF) SetResonance(res float64) {
	f.res = res
	f.update()
}

func (f *SVF) Set(fc float64, res float64) {
	f.fc = fc
	f.res = res
	f.q = 0.0
	f.update()
}

// SetFrequencyQ sets cutoff and Q directly (Q 0.5 is critically damped, higher is more resonant),
// handy for bandpass based filters like formant filters
func (f *SVF) SetFrequencyQ(fc float64, q float64) {
	f.fc = fc
	f.q = max(0.1, q)
	f.update()
}

// K returns the damping factor (1 / Q)
func (f *SVF) K() float64 {
	return f.k
}

func (f *SVF) update() {
	fc := max(1.0, min(f.fc, f.sr*0.49))
	res := max(0.0, min(f.res, 0.99))

	f.g = math.Tan(math.Pi * fc / f.sr)
	if f.q > 0.0 {
		f.k = 1.0 / f.q
	} else {
		f.k = 2.0 - 2.0*res
	}
	f.a1 = 1.0 / (1.0 + f.g*(f.g+f.k))
	f.a2 = f.g * f.a1
	f.a3 = f.g * f.a2
}

func (f *SVF) Clear() {
	f.ic1eq = 0.0
	f.ic2eq = 0.0
}

// Tick processes a single sample and returns the lowpass, bandpass and highpass outputs
func (f *SVF) Tick(v0 float64) (low float64, band float64, high float64) {
	v3 := v0 - f.ic2eq
	v1 := f.a1*f.ic1eq + f.a2*v3
	v2 := f.ic2eq + f.a2*f.ic1eq + f.a3*v3

	f.ic1eq = 2.0*v1 - f.ic1eq
	f.ic2eq = 2.0*v2 - f.ic2eq

	return v2, v1, v0 - f.k*v1 - v2
}

// Process filters a single sample and returns the output for the given mode
func (f *SVF) Process(v0 float64, mode Mode) float64 {
	low, band, high := f.Tick(v0)

	switch mode {
	case BandPass:
		return band
	case HighPass:
		return high
	case Notch:
		return low + high
	case Peak:
		return low - high
	case AllPass:
		return low + high - f.k*band
	default:
		return low
	}
}
