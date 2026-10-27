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

// a resonance of 1 sings on by itself, at the cutoff; 0.9 only rings and dies
func TestSelfOscillation(t *testing.T) {
	for _, c := range []struct {
		res    float64
		sings  bool
		cutoff float64
	}{{0.9, false, 1000}, {1.0, true, 1000}, {1.0, true, 300}} {
		l := New(c.cutoff, c.res, 44100)
		l.Process(1, LowPass24)
		peak, crossings, last := 0.0, 0, 0.0

		for i := 1; i < 88200; i++ {
			y := l.Process(0, LowPass24)
			if i > 44100 {
				peak = math.Max(peak, math.Abs(y))
				if last < 0 && y >= 0 {
					crossings++
				}
			}

			last = y
		}

		if sings := peak > 0.02; sings != c.sings {
			t.Errorf("resonance %v: the peak after a second is %v, sings should be %v", c.res, peak, c.sings)
		}

		if c.sings && math.Abs(float64(crossings)-c.cutoff) > c.cutoff*0.02 {
			t.Errorf("sings at %d Hz, want %v", crossings, c.cutoff)
		}
	}
}

// peakGain is the level a sine of an amplitude comes out at, as dB of what went in
func peakGain(fc, res, f, amp, drive float64) float64 {
	l := New(fc, res, 44100)
	l.SetDrive(drive)
	peak := 0.0

	for i := 0; i < 44100; i++ {
		y := l.Process(amp*math.Sin(2*math.Pi*f*float64(i)/44100), LowPass24)
		if i > 30000 {
			peak = math.Max(peak, math.Abs(y))
		}
	}

	return 20 * math.Log10(peak/amp)
}

// the stages clip softly: a small signal sees the linear filter, a loud one at a high resonance is held back where the resonance would grow it
func TestStagesClip(t *testing.T) {
	quiet := peakGain(1000, 0.9, 900, 0.01, 1)
	loud := peakGain(1000, 0.9, 900, 1, 1)

	if quiet < 8 {
		t.Errorf("a quiet signal at the resonance gains %.1f dB, want the peak of the linear filter (over 8)", quiet)
	}

	if loud > quiet-10 {
		t.Errorf("a loud signal gains %.1f dB against %.1f dB for a quiet one: the stages should hold the resonance back", loud, quiet)
	}
}

// whatever goes in, nothing blows up
func TestStagesStayBounded(t *testing.T) {
	for _, res := range []float64{0, 0.9, 1} {
		l := New(800, res, 44100)
		l.SetDrive(6)

		for i := 0; i < 20000; i++ {
			x := 40 * math.Sin(float64(i)*0.05)
			if i%700 < 3 {
				x = 100
			}

			if y := l.Process(x, LowPass24); math.IsNaN(y) || math.Abs(y) > 20 {
				t.Fatalf("resonance %v: sample %d is %v", res, i, y)
			}
		}
	}
}

// a lower level for the stages makes the same signal clip more: the resonance of a signal of normal size is held back further
func TestStageLevelSetsTheClip(t *testing.T) {
	gain := func(level float64) float64 {
		l := New(1000, 0.9, 44100)
		l.SetStageLevel(level)
		peak := 0.0

		for i := 0; i < 44100; i++ {
			y := l.Process(0.5*math.Sin(2*math.Pi*900*float64(i)/44100), LowPass24)
			if i > 30000 {
				peak = math.Max(peak, math.Abs(y))
			}
		}

		return peak
	}

	if hard, soft := gain(1.25), gain(4); hard >= soft {
		t.Errorf("a stage level of 1.25 gives a peak of %v, 4 gives %v: the lower should clip more", hard, soft)
	}
}
