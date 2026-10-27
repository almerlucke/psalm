package vocal

import (
	"math"
	"math/cmplx"
	"testing"
)

func fftv(x []complex128) []complex128 {
	n := len(x)
	if n == 1 {
		return x
	}

	even, odd := make([]complex128, n/2), make([]complex128, n/2)
	for i := range n / 2 {
		even[i], odd[i] = x[2*i], x[2*i+1]
	}

	even, odd = fftv(even), fftv(odd)
	out := make([]complex128, n)

	for k := range n / 2 {
		t := cmplx.Rect(1, -2*math.Pi*float64(k)/float64(n)) * odd[k]
		out[k], out[k+n/2] = even[k]+t, even[k]-t
	}

	return out
}

// aliasOf is the energy that is not at a harmonic of f (within four bins) as dB of the energy that is
func aliasOf(sig []float64, f, sr float64) float64 {
	n := len(sig)
	x := make([]complex128, n)

	for i, v := range sig {
		w := 0.35875 - 0.48829*math.Cos(2*math.Pi*float64(i)/float64(n)) + 0.14128*math.Cos(4*math.Pi*float64(i)/float64(n)) - 0.01168*math.Cos(6*math.Pi*float64(i)/float64(n))
		x[i] = complex(v*w, 0)
	}

	spectrum := fftv(x)
	bin := sr / float64(n)
	var harmonic, alias float64

	for k := 1; k < n/2; k++ {
		p := real(spectrum[k] * cmplx.Conj(spectrum[k]))
		if h := float64(k) * bin / f; math.Abs(h-math.Round(h)) < 4*bin/f {
			harmonic += p
		} else {
			alias += p
		}
	}

	return 10 * math.Log10(alias/harmonic)
}

// the closing of the folds is a hard edge: its correction keeps the alias of a pulse far below its harmonics, up to the top of the singing range
func TestGlottisIsBandLimited(t *testing.T) {
	const sr = 44100.0

	for _, f := range []float64{110, 220, 440, 880, 1320, 1760} {
		f *= 1.0001
		g := &glottis{}
		sig := make([]float64, 1<<15)

		for i := range sig {
			sig[i], _ = g.tick(f/sr, 0.6, 1.4, 0, 0)
		}

		t.Logf("%.0f Hz: alias %.1f dB", f, aliasOf(sig, f, sr))

		if level := aliasOf(sig, f, sr); level > -50 {
			t.Errorf("%.0f Hz: the alias is at %.1f dB, want below -50", f, level)
		}
	}
}

// the same for the Liljencrants-Fant pulse: its two corners are corrected. The pressed end of its quality (Rd 0.5) closes in less than a sample at a high pitch, which
// is a hard edge that is not corrected (the limits say how far the alias is kept down)
func TestLFPulseIsBandLimited(t *testing.T) {
	const sr = 44100.0

	for _, c := range []struct{ rd, f, limit float64 }{
		{0.5, 110, -70}, {0.5, 440, -45},
		{1, 110, -70}, {1, 440, -60}, {1, 880, -50},
		{2.2, 110, -70}, {2.2, 440, -70}, {2.2, 880, -65}, {2.2, 1760, -55},
	} {
		pl := lfSolve(lfRd(c.rd))
		f := c.f * 1.0001
		g := &glottis{}
		sig := make([]float64, 1<<15)

		for i := range sig {
			sig[i], _ = g.tickLF(f/sr, &pl, 0, 0)
		}

		if level := aliasOf(sig, f, sr); level > c.limit {
			t.Errorf("Rd %.1f, %.0f Hz: the alias is at %.1f dB, want below %v", c.rd, c.f, level, c.limit)
		}
	}
}
