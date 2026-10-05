package svf

import (
	"math"
	"testing"
)

const sr = 44100.0

// gainDB measures the steady state gain of the filter for a sine at freq
func gainDB(f *SVF, mode Mode, freq float64) float64 {
	f.Clear()
	in, out := 0.0, 0.0
	for i := range int(sr) {
		x := math.Sin(2 * math.Pi * freq * float64(i) / sr)
		y := f.Process(x, mode)
		if i > int(sr)/2 {
			in += x * x
			out += y * y
		}
	}
	return 10 * math.Log10(out/in)
}

func TestModes(t *testing.T) {
	f := New(1000, 0, sr)
	f.SetFrequencyQ(1000, 0.707)

	if g := gainDB(f, LowPass, 100); math.Abs(g) > 0.5 {
		t.Fatalf("lowpass passband %v dB", g)
	}
	if g := gainDB(f, LowPass, 8000); g > -30 {
		t.Fatalf("lowpass stopband %v dB", g)
	}
	if g := gainDB(f, HighPass, 8000); math.Abs(g) > 0.5 {
		t.Fatalf("highpass passband %v dB", g)
	}
	if g := gainDB(f, HighPass, 100); g > -30 {
		t.Fatalf("highpass stopband %v dB", g)
	}
	if g := gainDB(f, Notch, 1000); g > -30 {
		t.Fatalf("notch at center %v dB", g)
	}
	for _, freq := range []float64{100, 1000, 5000} {
		if g := gainDB(f, AllPass, freq); math.Abs(g) > 0.1 {
			t.Fatalf("allpass at %v Hz: %v dB", freq, g)
		}
	}
}

func TestBandPassQ(t *testing.T) {
	f := New(1000, 0, sr)
	f.SetFrequencyQ(1000, 10)

	// band * k has 0 dB at the center
	f.Clear()
	in, out := 0.0, 0.0
	for i := range int(sr) {
		x := math.Sin(2 * math.Pi * 1000 * float64(i) / sr)
		_, band, _ := f.Tick(x)
		y := band * f.K()
		if i > int(sr)/2 {
			in += x * x
			out += y * y
		}
	}
	if g := 10 * math.Log10(out/in); math.Abs(g) > 0.5 {
		t.Fatalf("normalized bandpass center gain %v dB", g)
	}
	if g := gainDB(f, BandPass, 3000) - gainDB(f, BandPass, 1000); g > -15 {
		t.Fatalf("expected narrow band with Q 10, off center %v dB", g)
	}
}
