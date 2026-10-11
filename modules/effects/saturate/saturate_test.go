package saturate

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// render runs a sine through the saturation and returns the left output after a settling second
func render(s *Saturate, freq, amp float64) []float64 {
	bs, sr := s.Config().BufferSize, s.Config().SampleRate
	var out []float64
	n := 0
	for b := range int(2 * sr / float64(bs)) {
		for i := range bs {
			x := float32(amp * math.Sin(2*math.Pi*freq*float64(n)/sr))
			s.Input(0).Channels[0][i], s.Input(0).Channels[1][i] = x, x
			n++
		}
		s.Synthesize()
		if b >= int(sr/float64(bs)) {
			for _, x := range s.Output(0).Channels[0] {
				out = append(out, float64(x))
			}
		}
	}
	return out
}

// magnitude returns the level of a frequency in a signal (Goertzel), in dB relative to a full
// scale sine
func magnitude(x []float64, freq, sr float64) float64 {
	w := 2 * math.Pi * freq / sr
	c := 2 * math.Cos(w)
	var s1, s2 float64
	for _, v := range x {
		s1, s2 = v+c*s1-s2, s1
	}
	p := s1*s1 + s2*s2 - c*s1*s2
	return 20 * math.Log10(2*math.Sqrt(p)/float64(len(x))+1e-12)
}

func TestCleanAtLowLevels(t *testing.T) {
	psalm.New(nil)
	s := New(Config{Shape: Soft, Drive: 0, Output: 0, Mix: 1})
	out := render(s, 440, 0.01)
	sr := s.Config().SampleRate
	if l := magnitude(out, 440, sr); math.Abs(l+40) > 0.3 {
		t.Fatalf("a quiet sound passes at its level: %.2f dB", l)
	}
	if h := magnitude(out, 1320, sr) - magnitude(out, 440, sr); h > -60 {
		t.Fatalf("and nearly clean: third harmonic %.1f dB", h)
	}
}

func TestHarmonics(t *testing.T) {
	psalm.New(nil)
	sr := psalm.SampleRate()
	soft := render(New(Config{Shape: Soft, Drive: 12, Mix: 1}), 220, 0.5)
	if h3 := magnitude(soft, 660, sr) - magnitude(soft, 220, sr); h3 < -30 {
		t.Fatalf("driven, a soft curve adds odd harmonics: third %.1f dB", h3)
	}
	if h2 := magnitude(soft, 440, sr) - magnitude(soft, 220, sr); h2 > -50 { // leakage of the measurement only
		t.Fatalf("a symmetric curve adds no even ones: second %.1f dB", h2)
	}
	tube := render(New(Config{Shape: Tube, Drive: 12, Mix: 1}), 220, 0.5)
	if h2 := magnitude(tube, 440, sr) - magnitude(tube, 220, sr); h2 < -40 {
		t.Fatalf("a lopsided curve adds even harmonics: second %.1f dB", h2)
	}
}

func TestNoAliasing(t *testing.T) {
	psalm.New(nil)
	sr := psalm.SampleRate()
	// a 7 kHz sine clipped hard: its 5th harmonic (35 kHz) would fold back to 9.1 kHz
	out := render(New(Config{Shape: Hard, Drive: 24, Mix: 1}), 7000, 0.5)
	folded := magnitude(out, sr-35000, sr) - magnitude(out, 7000, sr)
	if folded > -40 {
		t.Fatalf("the folded 5th harmonic should be filtered away: %.1f dB", folded)
	}
}

func TestDryLinesUp(t *testing.T) {
	psalm.New(nil)
	// mix halfway with no drive: the dry and the wet sound are the same, in phase: no comb filter
	s := New(Config{Shape: Soft, Drive: 0, Output: 0, Mix: 0.5})
	out := render(s, 5000, 0.01)
	if l := magnitude(out, 5000, s.Config().SampleRate); math.Abs(l+40) > 0.5 {
		t.Fatalf("the dry sound should line up with the saturated one: %.2f dB", l)
	}
	if s.SignalLatency() <= 0 {
		t.Fatal("it should tell its latency")
	}
}

func TestFoldStaysInRange(t *testing.T) {
	for _, x := range []float64{-7.3, -2.5, -1, 0, 0.4, 1, 1.5, 3, 9.9} {
		if y := curve(Fold, x, 0); y < -1 || y > 1 {
			t.Fatalf("folding should stay within ±1: %v -> %v", x, y)
		}
	}
	if y := curve(Fold, 1.5, 0); math.Abs(y-0.5) > 1e-9 {
		t.Fatalf("1.5 folds back to 0.5: %v", y)
	}
}
