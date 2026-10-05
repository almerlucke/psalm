package biquad

import (
	"math"
	"testing"
)

const sr = 44100.0

func gainDB(b *Biquad, freq float64) float64 {
	b.Clear()
	in, out := 0.0, 0.0
	for i := range int(sr) {
		x := math.Sin(2 * math.Pi * freq * float64(i) / sr)
		y := b.Process(x)
		if i > int(sr)/2 {
			in += x * x
			out += y * y
		}
	}
	return 10 * math.Log10(out/in)
}

func near(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s: got %.2f dB, want %.2f dB", name, got, want)
	}
}

func TestTypes(t *testing.T) {
	near(t, "lowpass passband", gainDB(New(LowPass, 1000, 0.707, 0, sr), 50), 0, 0.2)
	near(t, "lowpass at cutoff", gainDB(New(LowPass, 1000, 0.707, 0, sr), 1000), -3, 0.3)
	near(t, "highpass passband", gainDB(New(HighPass, 1000, 0.707, 0, sr), 15000), 0, 0.3)
	near(t, "bandpass center", gainDB(New(BandPass, 1000, 2, 0, sr), 1000), 0, 0.2)
	near(t, "peak boost", gainDB(New(Peak, 1000, 1, 6, sr), 1000), 6, 0.2)
	near(t, "peak cut", gainDB(New(Peak, 1000, 1, -9, sr), 1000), -9, 0.2)
	near(t, "low shelf", gainDB(New(LowShelf, 500, 0.707, 6, sr), 30), 6, 0.3)
	near(t, "low shelf above", gainDB(New(LowShelf, 500, 0.707, 6, sr), 10000), 0, 0.3)
	near(t, "high shelf", gainDB(New(HighShelf, 2000, 0.707, -6, sr), 15000), -6, 0.3)
	near(t, "allpass", gainDB(New(AllPass, 1000, 0.707, 0, sr), 3000), 0, 0.1)

	if g := gainDB(New(Notch, 1000, 2, 0, sr), 1000); g > -30 {
		t.Fatalf("notch center %v dB", g)
	}
	if g := gainDB(New(LowPass, 1000, 0.707, 0, sr), 10000); g > -35 {
		t.Fatalf("lowpass stopband %v dB", g)
	}
}
