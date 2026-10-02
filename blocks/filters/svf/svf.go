package svf

import "math"

// SVF is a zero-delay feedback (TPT) state variable filter after Andrew Simper (Cytomic)
type SVF struct {
	sr    float64
	fc    float64
	res   float64
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
	f.update()
}

func (f *SVF) update() {
	fc := math.Max(1.0, math.Min(f.fc, f.sr*0.49))
	res := math.Max(0.0, math.Min(f.res, 0.99))

	f.g = math.Tan(math.Pi * fc / f.sr)
	f.k = 2.0 - 2.0*res
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
