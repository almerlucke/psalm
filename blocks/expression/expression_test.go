package expression

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func TestBendAndWheel(t *testing.T) {
	var e Expression
	if e.Ratio() != 1 || e.Wheel() != 0 || e.Moved() {
		t.Fatalf("at rest: ratio %g wheel %g", e.Ratio(), e.Wheel())
	}
	e.Receive(psalm.Message{"pitchBend": 1.0, "modWheel": 0.5})
	// it glides there: after a sample barely moved, after 50 ms there
	e.Tick(44100)
	if r := e.Ratio(); r <= 1 || r > 1.01 {
		t.Errorf("a sample after the bend: ratio %g", r)
	}
	for range 2205 {
		e.Tick(44100)
	}
	if r := e.Ratio(); math.Abs(r-math.Exp2(2.0/12)) > 1e-4 {
		t.Errorf("bent all the way up: ratio %g, want two semitones", r)
	}
	if math.Abs(e.Wheel()-0.5) > 1e-4 {
		t.Errorf("the wheel: %g", e.Wheel())
	}
	e.Receive(psalm.Message{"pitchBend": -0.5, "bendRange": 12.0})
	e.Settle()
	if r := e.Ratio(); math.Abs(r-math.Exp2(-6.0/12)) > 1e-9 {
		t.Errorf("half down over an octave: ratio %g", r)
	}
	// the wheel's vibrato: none at rest, depth up and down
	var f Expression
	if v := f.Vibrato(44100, 5, 0.5); v != 0 {
		t.Errorf("a vibrato with the wheel at rest: %g", v)
	}
	f.Receive(psalm.Message{"modWheel": 1.0})
	f.Settle()
	lo, hi := 0.0, 0.0
	for range 44100 {
		v := f.Vibrato(44100, 5, 0.5)
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if math.Abs(lo+0.5) > 1e-3 || math.Abs(hi-0.5) > 1e-3 {
		t.Errorf("the vibrato swings %g to %g, want ±0.5", lo, hi)
	}
}
