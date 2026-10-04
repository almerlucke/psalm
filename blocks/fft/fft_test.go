package fft

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestForwardMatchesDFT(t *testing.T) {
	const n = 64
	f := New(n)

	re := make([]float64, n)
	im := make([]float64, n)
	for i := range n {
		re[i] = rand.Float64()*2 - 1
		im[i] = rand.Float64()*2 - 1
	}

	// naive DFT
	wantRe := make([]float64, n)
	wantIm := make([]float64, n)
	for k := range n {
		for j := range n {
			a := -2 * math.Pi * float64(k*j) / n
			wantRe[k] += re[j]*math.Cos(a) - im[j]*math.Sin(a)
			wantIm[k] += re[j]*math.Sin(a) + im[j]*math.Cos(a)
		}
	}

	f.Forward(re, im)

	for k := range n {
		if math.Abs(re[k]-wantRe[k]) > 1e-9 || math.Abs(im[k]-wantIm[k]) > 1e-9 {
			t.Fatalf("bin %d: got (%v, %v), want (%v, %v)", k, re[k], im[k], wantRe[k], wantIm[k])
		}
	}
}

func TestRoundTrip(t *testing.T) {
	const n = 1024
	f := New(n)

	orig := make([]float64, n)
	re := make([]float64, n)
	im := make([]float64, n)
	for i := range n {
		orig[i] = rand.Float64()*2 - 1
		re[i] = orig[i]
	}

	f.Forward(re, im)
	f.Inverse(re, im)

	for i := range n {
		if math.Abs(re[i]-orig[i]) > 1e-9 || math.Abs(im[i]) > 1e-9 {
			t.Fatalf("sample %d: got (%v, %v), want %v", i, re[i], im[i], orig[i])
		}
	}
}

func TestNonPowerOfTwoPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	New(100)
}
