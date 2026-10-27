package blosc

import (
	"math"
	"math/cmplx"
	"testing"
)

func fft(x []complex128) []complex128 {
	n := len(x)
	if n == 1 {
		return x
	}

	even, odd := make([]complex128, n/2), make([]complex128, n/2)
	for i := range n / 2 {
		even[i], odd[i] = x[2*i], x[2*i+1]
	}

	even, odd = fft(even), fft(odd)
	out := make([]complex128, n)

	for k := range n / 2 {
		t := cmplx.Rect(1, -2*math.Pi*float64(k)/float64(n)) * odd[k]
		out[k], out[k+n/2] = even[k]+t, even[k]-t
	}

	return out
}

// aliasLevel is the energy that is not at a harmonic of f (within four bins) as dB of the energy that is
func aliasLevel(sig []float64, f, sr float64) float64 {
	n := len(sig)
	x := make([]complex128, n)

	for i, v := range sig {
		w := 0.35875 - 0.48829*math.Cos(2*math.Pi*float64(i)/float64(n)) + 0.14128*math.Cos(4*math.Pi*float64(i)/float64(n)) - 0.01168*math.Cos(6*math.Pi*float64(i)/float64(n))
		x[i] = complex(v*w, 0)
	}

	spectrum := fft(x)
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

// every waveform is band limited: its jumps and its corners are corrected by the windowed sinc, so what folds back from above the Nyquist frequency stays far below
// the harmonics, up the keyboard
func TestWaveformsAreBandLimited(t *testing.T) {
	const sr = 44100.0

	names := []string{"sine", "cosine", "triangle", "square", "rectangle", "sawtooth", "ramp", "modified triangle", "modified square", "half wave rectified sine",
		"full wave rectified sine", "triangular pulse", "trapezoid", "variable trapezoid"}

	for wf := Sine; wf <= TrapezoidVariable; wf++ {
		for _, f := range []float64{65.4, 261.6, 1046.5, 2093, 4186} {
			f *= 1.0001
			o := New(f, sr, 0, 0.3, 1, wf)
			sig := make([]float64, 1<<15)

			for i := range sig {
				sig[i] = o.Tick(0, false, 0)
			}

			if level := aliasLevel(sig, f, sr); level > -55 {
				t.Errorf("%s at %.0f Hz: the alias is at %.1f dB, want below -55", names[wf], f, level)
			}
		}
	}
}

// below the Nyquist frequency the corrected edge keeps the wave: a saw is a ramp of the right slope between its edges
func TestSawKeepsItsSlope(t *testing.T) {
	const sr, f = 44100.0, 100.0
	o := New(f, sr, 0, 0.5, 1, Sawtooth)
	prev := o.Tick(0, false, 0)

	for i := 1; i < 2000; i++ {
		v := o.Tick(0, false, 0)
		if m := i % 441; m < 203 || m > 239 { // away (more than the 16 samples of the correction) from the edge at 220.5 of every 441 samples
			if d := v - prev; math.Abs(d-2*f/sr) > 1e-9 {
				t.Fatalf("sample %d: the slope is %v, want %v", i, d, 2*f/sr)
			}
		}

		prev = v
	}
}
