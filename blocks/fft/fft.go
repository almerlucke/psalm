package fft

import (
	"fmt"
	"math"
)

// FFT is an in-place iterative radix-2 complex FFT with precomputed twiddles and bit reversal
type FFT struct {
	n   int
	rev []int
	cos []float64
	sin []float64
}

// New creates an FFT of size n, n must be a power of 2
func New(n int) *FFT {
	if n < 2 || n&(n-1) != 0 {
		panic(fmt.Sprintf("fft size must be a power of 2, got %d", n))
	}

	f := &FFT{
		n:   n,
		rev: make([]int, n),
		cos: make([]float64, n/2),
		sin: make([]float64, n/2),
	}

	bits := 0
	for 1<<bits < n {
		bits++
	}

	for i := range n {
		r := 0
		for b := range bits {
			if i&(1<<b) != 0 {
				r |= 1 << (bits - 1 - b)
			}
		}
		f.rev[i] = r
	}

	for i := range n / 2 {
		a := -2.0 * math.Pi * float64(i) / float64(n)
		f.cos[i] = math.Cos(a)
		f.sin[i] = math.Sin(a)
	}

	return f
}

func (f *FFT) Size() int {
	return f.n
}

// Forward transforms re/im in place
func (f *FFT) Forward(re []float64, im []float64) {
	f.transform(re, im, false)
}

// Inverse transforms re/im in place, including the 1/n scaling
func (f *FFT) Inverse(re []float64, im []float64) {
	f.transform(re, im, true)

	scale := 1.0 / float64(f.n)
	for i := range f.n {
		re[i] *= scale
		im[i] *= scale
	}
}

func (f *FFT) transform(re []float64, im []float64, inverse bool) {
	n := f.n

	for i, r := range f.rev {
		if i < r {
			re[i], re[r] = re[r], re[i]
			im[i], im[r] = im[r], im[i]
		}
	}

	sign := 1.0
	if inverse {
		sign = -1.0
	}

	for size := 2; size <= n; size <<= 1 {
		half := size >> 1
		step := n / size

		for start := 0; start < n; start += size {
			for j := range half {
				wr := f.cos[j*step]
				wi := sign * f.sin[j*step]

				a := start + j
				b := a + half

				tr := wr*re[b] - wi*im[b]
				ti := wr*im[b] + wi*re[b]

				re[b] = re[a] - tr
				im[b] = im[a] - ti
				re[a] += tr
				im[a] += ti
			}
		}
	}
}

// HannWindow returns a periodic Hann window of size n, suited for overlap-add
func HannWindow(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 0.5 - 0.5*math.Cos(2.0*math.Pi*float64(i)/float64(n))
	}
	return w
}
