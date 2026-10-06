package mix

import (
	"math"
	"testing"
)

func TestEqualPowerKeepsPower(t *testing.T) {
	for _, m := range []float64{-1, 0, 0.1, 0.25, 0.5, 0.8, 1, 2} {
		dry, wet := EqualPower(m)
		if p := dry*dry + wet*wet; math.Abs(p-1) > 1e-5 {
			t.Fatalf("mix %v: power %v", m, p)
		}
	}
	if dry, wet := EqualPower(0); dry != 1 || math.Abs(wet) > 1e-9 {
		t.Fatalf("mix 0 is dry, got %v %v", dry, wet)
	}
	if dry, wet := EqualPower(1); math.Abs(dry) > 1e-6 || math.Abs(wet-1) > 1e-9 {
		t.Fatalf("mix 1 is wet, got %v %v", dry, wet)
	}
}

func TestLinearToEqualPower(t *testing.T) {
	for _, m := range []float64{0, 0.125, 0.2, 0.5, 0.7, 1} {
		em, gain := LinearToEqualPower(m)
		dry, wet := EqualPower(em)
		if math.Abs(dry*gain-(1-m)) > 1e-5 || math.Abs(wet*gain-m) > 1e-5 {
			t.Fatalf("mix %v: %v, %v times %v is not %v, %v", m, dry, wet, gain, 1-m, m)
		}
	}
}

func TestSmootherGlides(t *testing.T) {
	s := NewSmoother(0, SmoothMs, 44100)
	n := 0
	for s.Next(1) < 1-math.Exp(-1) {
		n++
	}
	if ms := float64(n) / 44.1; math.Abs(ms-SmoothMs) > 0.5 {
		t.Fatalf("two thirds of the way after %.1f ms", ms)
	}
}
