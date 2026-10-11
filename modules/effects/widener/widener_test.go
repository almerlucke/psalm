package widener

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// run plays different sines on left and right and returns the last buffer of both
func run(w *Widener, freq float64) ([]float32, []float32) {
	bs, sr := w.Config().BufferSize, w.Config().SampleRate
	n := 0
	for range int(sr / float64(bs)) {
		for i := range bs {
			a := 2 * math.Pi * freq * float64(n) / sr
			w.Input(0).Channels[0][i] = float32(0.5 * math.Sin(a))
			w.Input(0).Channels[1][i] = float32(0.3 * math.Sin(a+1))
			n++
		}
		w.Synthesize()
	}
	return w.Output(0).Channels[0], w.Output(0).Channels[1]
}

// sideLevel returns the peak of (l-r)/2
func sideLevel(l, r []float32) float64 {
	m := 0.0
	for i := range l {
		m = math.Max(m, math.Abs(float64(l[i]-r[i])/2))
	}
	return m
}

func TestWidth(t *testing.T) {
	psalm.New(nil)
	l, r := run(New(Config{Width: 0}), 1000)
	if sideLevel(l, r) > 1e-6 {
		t.Fatal("width 0 is mono")
	}
	l1, r1 := run(New(Config{Width: 1}), 1000)
	l2, r2 := run(New(Config{Width: 2}), 1000)
	if s1, s2 := sideLevel(l1, r1), sideLevel(l2, r2); math.Abs(s2-2*s1) > 0.002 {
		t.Fatalf("width 2 has twice the sides: %.4f vs %.4f", s2, s1)
	}
	for i := range l1 {
		if math.Abs(float64((l1[i]+r1[i])-(l2[i]+r2[i]))) > 1e-5 {
			t.Fatal("the middle stays the same")
		}
	}
}

func TestMonoBass(t *testing.T) {
	psalm.New(nil)
	l, r := run(New(Config{Width: 1, MonoBelow: 150}), 40)
	if s := sideLevel(l, r); s > 0.03 {
		t.Fatalf("40 Hz should be kept in the middle: sides %.3f", s)
	}
	l, r = run(New(Config{Width: 1, MonoBelow: 150}), 2000)
	if s := sideLevel(l, r); s < 0.18 {
		t.Fatalf("2 kHz keeps its sides: %.3f", s)
	}
}
