package tanh

import (
	"math"
	"testing"
)

func TestAccuracy(t *testing.T) {
	worst, prev := 0.0, -2.0
	for x := -10.0; x <= 10; x += 1e-4 {
		y := Tanh(x)
		worst = max(worst, math.Abs(y-math.Tanh(x)))
		if y < prev {
			t.Fatalf("not rising at %g: %g after %g", x, y, prev)
		}
		prev = y
		if y != -Tanh(-x) {
			t.Fatalf("not odd at %g", x)
		}
	}
	if worst > 1e-4 {
		t.Errorf("the largest error is %g, over 1e-4", worst)
	}
}

func TestSmallSignalGain(t *testing.T) {
	// a gain of 1 around zero, as math.Tanh has: the saturation does not touch a quiet signal
	for _, x := range []float64{1e-6, 1e-3, 0.01} {
		if got, want := Tanh(x)/x, math.Tanh(x)/x; math.Abs(got-want) > 1e-9 {
			t.Errorf("gain at %g is %g, math.Tanh has %g", x, got, want)
		}
	}
	if Tanh(0) != 0 {
		t.Error("tanh(0) is not 0")
	}
}

func TestEdges(t *testing.T) {
	if Tanh(math.Inf(1)) != 1 || Tanh(math.Inf(-1)) != -1 || Tanh(100) != 1 {
		t.Error("the ends are not ±1")
	}
	if !math.IsNaN(Tanh(math.NaN())) {
		t.Error("NaN does not stay NaN")
	}
}

func BenchmarkTanh(b *testing.B) {
	s := 0.0
	for i := range b.N {
		s += Tanh(float64(i&1023)/170 - 3)
	}
	_ = s
}

func BenchmarkMathTanh(b *testing.B) {
	s := 0.0
	for i := range b.N {
		s += math.Tanh(float64(i&1023)/170 - 3)
	}
	_ = s
}
