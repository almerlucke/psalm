package pan

import (
	"math"
	"testing"
)

func render(p *Pan, value float32) (float32, float32) {
	for i := range p.Input(0).Channels[0] {
		p.Input(0).Channels[0][i] = value
	}
	p.Synthesize()
	out := p.Output(0).Channels
	last := len(out[0]) - 1
	return out[0][last], out[1][last]
}

func TestPanPositions(t *testing.T) {
	cases := []struct {
		pan         float64
		left, right float64
	}{
		{0.0, 1.0, 1.0},
		{-1.0, math.Sqrt2, 0.0},
		{1.0, 0.0, math.Sqrt2},
	}

	for _, c := range cases {
		l, r := render(New(c.pan), 1.0)
		if math.Abs(float64(l)-c.left) > 1e-5 || math.Abs(float64(r)-c.right) > 1e-5 {
			t.Fatalf("pan %v: got (%v, %v), want (%v, %v)", c.pan, l, r, c.left, c.right)
		}
	}
}

func TestEqualPower(t *testing.T) {
	for pan := -1.0; pan <= 1.0; pan += 0.1 {
		gl, gr := gains(pan)
		if power := gl*gl + gr*gr; math.Abs(power-2.0) > 1e-9 {
			t.Fatalf("pan %v: power %v, want constant 2", pan, power)
		}
	}
}

func TestPanChangeIsRamped(t *testing.T) {
	p := New(-1.0)
	render(p, 1.0)

	p.SetPan(1.0)
	for i := range p.Input(0).Channels[0] {
		p.Input(0).Channels[0][i] = 1.0
	}
	p.Synthesize()

	right := p.Output(0).Channels[1]
	if right[0] > 0.05 {
		t.Fatalf("expected ramp to start near the old position, got %v", right[0])
	}
	if math.Abs(float64(right[len(right)-1])-math.Sqrt2) > 1e-5 {
		t.Fatalf("expected ramp to end at the new position, got %v", right[len(right)-1])
	}
	for i := 1; i < len(right); i++ {
		if right[i] < right[i-1] {
			t.Fatalf("ramp not monotonic at %d", i)
		}
	}
}
