package reference

import (
	"math"
	"testing"
)

func TestNotes(t *testing.T) {
	ns := Notes()
	if len(ns) != 26 {
		t.Fatalf("%d notes, want the 26 of the recording", len(ns))
	}
	for _, n := range ns {
		if n.Pitch < 36 || n.Pitch > 84 || n.DropDBps <= 0 {
			t.Errorf("note at %.1f s: pitch %d, falls %.1f dB/s", n.Onset, n.Pitch, n.DropDBps)
		}
		if f := 440 * math.Exp2(float64(n.Pitch-69)/12); math.Abs(f-n.Hz)/f > 0.02 {
			t.Errorf("note at %.1f s: %.1f Hz is not pitch %d", n.Onset, n.Hz, n.Pitch)
		}
	}
}

// the high notes of the real guitar fall faster than its low ones
func TestTargetRateRisesWithPitch(t *testing.T) {
	if lo, hi := TargetRate(40), TargetRate(66); !(lo > 0 && hi > 2*lo) {
		t.Errorf("falls %.1f dB/s at 40 and %.1f at 66", lo, hi)
	}
}
