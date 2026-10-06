package gain

import (
	"math"
	"testing"
)

func TestVolume(t *testing.T) {
	for _, db := range []float64{0, -6, 3, -130} {
		g := New(db)
		for i := range g.Config().BufferSize {
			g.Input(0).Channels[0][i] = 0.5
			g.Input(0).Channels[1][i] = -0.5
		}
		g.Synthesize()
		want := 0.5 * math.Pow(10, db/20)
		if db <= -120 {
			want = 0
		}
		if l, r := float64(g.Output(0).Channels[0][0]), float64(g.Output(0).Channels[1][0]); math.Abs(l-want) > 1e-6 || math.Abs(r+want) > 1e-6 {
			t.Fatalf("%v dB: got %v %v, want %v", db, l, r, want)
		}
	}
}

func TestVolumeGlides(t *testing.T) {
	g := New(0)
	bs := g.Config().BufferSize
	for i := range bs {
		g.Input(0).Channels[0][i] = 1
		g.Input(0).Channels[1][i] = 1
	}
	g.Synthesize()
	g.Receive(-120.0, 1)
	g.Synthesize()
	out := g.Output(0).Channels[0]
	for i := 1; i < bs; i++ {
		if d := out[i-1] - out[i]; d < 0 || d > 0.001 {
			t.Fatalf("expected a glide down, step %v at %d", d, i)
		}
	}
	if out[bs-1] > 0.5 {
		t.Fatalf("expected most of the way down after a buffer, at %v", out[bs-1])
	}
}
