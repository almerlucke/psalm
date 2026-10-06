package sine

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestAccuracy(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	worst := 0.0
	for range 1_000_000 {
		x := (r.Float64() - 0.5) * 2000 // also far from 0
		worst = math.Max(worst, math.Abs(Sin(x)-math.Sin(x)))
		worst = math.Max(worst, math.Abs(Cos(x)-math.Cos(x)))
	}
	if worst > 5e-7 {
		t.Fatalf("worst error %v", worst)
	}
	for _, p := range []float64{0, 0.25, 0.5, 0.75, 1, -0.25, 3.75} {
		if d := math.Abs(Cycle(p) - math.Sin(2*math.Pi*p)); d > 1e-12 {
			t.Fatalf("Cycle(%v) off by %v", p, d)
		}
	}
}

var sink float64

func BenchmarkMathSin(b *testing.B) {
	x := 0.0
	for b.Loop() {
		sink += math.Sin(x)
		x += 0.0123
	}
}

func BenchmarkSin(b *testing.B) {
	x := 0.0
	for b.Loop() {
		sink += Sin(x)
		x += 0.0123
	}
}

func BenchmarkCycle(b *testing.B) {
	x := 0.0
	for b.Loop() {
		sink += Cycle(x)
		x += 0.0019
	}
}
