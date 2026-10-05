package ladder

import (
	"math"
	"testing"
)

const sr = 44100.0

func gainDB(l *Ladder, mode Mode, freq float64) float64 {
	l.Clear()
	in, out := 0.0, 0.0
	for i := range int(sr) {
		// small level so the input saturation stays linear
		x := 0.05 * math.Sin(2*math.Pi*freq*float64(i)/sr)
		y := l.Process(x, mode)
		if i > int(sr)/2 {
			in += x * x
			out += y * y
		}
	}
	return 10 * math.Log10(out/in)
}

func TestLowPassSlopes(t *testing.T) {
	l := New(500, 0, sr)

	pass := gainDB(l, LowPass24, 50)
	if math.Abs(pass) > 1.0 {
		t.Fatalf("passband %v dB", pass)
	}

	// two octaves above cutoff: about -48 dB for 24 dB/oct, about -24 dB for 12 dB/oct
	lp24 := gainDB(l, LowPass24, 2000)
	lp12 := gainDB(l, LowPass12, 2000)
	if lp24 > -35 {
		t.Fatalf("24 dB slope too shallow: %v dB", lp24)
	}
	if lp12 > -18 || lp12 < lp24 {
		t.Fatalf("12 dB slope unexpected: %v dB (24 dB: %v)", lp12, lp24)
	}
}

func TestHighPassAndBandPass(t *testing.T) {
	l := New(1000, 0, sr)

	if g := gainDB(l, HighPass24, 100); g > -35 {
		t.Fatalf("highpass stopband %v dB", g)
	}
	if g := gainDB(l, HighPass24, 10000); math.Abs(g) > 2.0 {
		t.Fatalf("highpass passband %v dB", g)
	}
	// the 12 dB bandpass has 6 dB/oct skirts (about -14 dB a decade away), the 24 dB one 12 dB/oct
	if center := gainDB(l, BandPass12, 1000); math.Abs(center) > 0.5 {
		t.Fatalf("bandpass 12 center %v dB", center)
	}
	for _, freq := range []float64{100, 10000} {
		bp12 := gainDB(l, BandPass12, freq)
		bp24 := gainDB(l, BandPass24, freq)
		if bp12 > -12 {
			t.Fatalf("bandpass 12 at %v Hz: %v dB", freq, bp12)
		}
		if bp24 > bp12-6 {
			t.Fatalf("bandpass 24 at %v Hz should be steeper: %v vs %v dB", freq, bp24, bp12)
		}
	}
}

func TestResonancePeakAndSelfOscillation(t *testing.T) {
	flat := New(1000, 0.0, sr)
	res := New(1000, 0.9, sr)
	if peak, base := gainDB(res, LowPass24, 1000), gainDB(flat, LowPass24, 1000); peak-base < 10 {
		t.Fatalf("expected a resonant peak: %v vs %v dB", peak, base)
	}

	// at full resonance an impulse keeps ringing, but stays bounded
	l := New(1000, 1.0, sr)
	maxOut, tail := 0.0, 0.0
	for i := range int(sr) {
		x := 0.0
		if i == 0 {
			x = 1.0
		}
		y := l.Process(x, LowPass24)
		if math.IsNaN(y) || math.IsInf(y, 0) {
			t.Fatal("unstable")
		}
		maxOut = math.Max(maxOut, math.Abs(y))
		if i > int(sr)-1000 {
			tail = math.Max(tail, math.Abs(y))
		}
	}
	if maxOut > 10 {
		t.Fatalf("self oscillation not bounded: %v", maxOut)
	}
	if tail < 1e-3 {
		t.Fatalf("expected self oscillation to keep ringing, tail %v", tail)
	}
}
