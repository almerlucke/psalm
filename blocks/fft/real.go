package fft

import "math"

// Real is an FFT of real signals of size n: a complex FFT of half the size, the even samples in
// the real part and the odd ones in the imaginary part, then untangled. About twice as fast as a
// complex FFT of size n, and it returns only the bins 0..n/2 (the others mirror them)
type Real struct {
	n        int
	half     *FFT
	cos, sin []float64 // the twiddles e^(-2 pi i k/n), k up to n/2
	re, im   []float64 // scratch of n/2
}

// NewReal creates a real FFT of size n, n must be a power of 2 (at least 4)
func NewReal(n int) *Real {
	if n < 4 || n&(n-1) != 0 {
		panic("fft: real size must be a power of 2, at least 4")
	}
	r := &Real{
		n:    n,
		half: New(n / 2),
		cos:  make([]float64, n/2+1),
		sin:  make([]float64, n/2+1),
		re:   make([]float64, n/2),
		im:   make([]float64, n/2),
	}
	for k := range r.cos {
		r.cos[k] = math.Cos(2 * math.Pi * float64(k) / float64(n))
		r.sin[k] = -math.Sin(2 * math.Pi * float64(k) / float64(n))
	}
	return r
}

// Size returns n
func (r *Real) Size() int { return r.n }

// Forward transforms the n samples of x into the bins 0..n/2 of re and im (n/2+1 each)
func (r *Real) Forward(x, re, im []float64) {
	h := r.n / 2
	for k := range h {
		r.re[k], r.im[k] = x[2*k], x[2*k+1]
	}
	r.half.Forward(r.re, r.im)

	// Z = E + i O: E[k] = (Z[k] + conj Z[h-k]) / 2, O[k] = (Z[k] - conj Z[h-k]) / 2i,
	// X[k] = E[k] + W^k O[k]
	for k := 0; k <= h; k++ {
		zr, zi := r.re[k%h], r.im[k%h]
		cr, ci := r.re[(h-k)%h], -r.im[(h-k)%h]
		er, ei := (zr+cr)/2, (zi+ci)/2
		or, oi := (zi-ci)/2, -(zr-cr)/2 // (Z - conj) / 2i
		wr, wi := r.cos[k], r.sin[k]
		re[k] = er + wr*or - wi*oi
		im[k] = ei + wr*oi + wi*or
	}
}

// Inverse transforms the bins 0..n/2 of re and im back into n samples of x (scaled, so Inverse
// undoes Forward)
func (r *Real) Inverse(re, im, x []float64) {
	h := r.n / 2
	// E[k] = (X[k] + conj X[h-k]) / 2, O[k] = (X[k] - conj X[h-k]) / (2 W^k), Z = E + i O
	for k := range h {
		xr, xi := re[k], im[k]
		cr, ci := re[h-k], -im[h-k]
		er, ei := (xr+cr)/2, (xi+ci)/2
		dr, di := (xr-cr)/2, (xi-ci)/2
		wr, wi := r.cos[k], -r.sin[k] // 1/W^k = conj W^k
		or, oi := dr*wr-di*wi, dr*wi+di*wr
		r.re[k], r.im[k] = er-oi, ei+or
	}
	r.half.Inverse(r.re, r.im)
	for k := range h {
		x[2*k], x[2*k+1] = r.re[k], r.im[k]
	}
}
