package crusher

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func run(c *Crusher, amp float64) []float64 {
	bs, sr := c.Config().BufferSize, c.Config().SampleRate
	var out []float64
	for i := range bs {
		x := float32(amp * math.Sin(2*math.Pi*220*float64(i)/sr))
		c.Input(0).Channels[0][i], c.Input(0).Channels[1][i] = x, x
	}
	c.Synthesize()
	for _, x := range c.Output(0).Channels[0] {
		out = append(out, float64(x))
	}
	return out
}

func TestBits(t *testing.T) {
	psalm.New(nil)
	out := run(New(Config{Bits: 2, Mix: 1}), 0.9)
	for _, x := range out {
		if q := x * 2; math.Abs(q-math.Round(q)) > 1e-6 {
			t.Fatalf("2 bits leave the steps -1, -0.5, 0, 0.5, 1: %v", x)
		}
	}
	if q := quantize(0.3, 0); q != 0.3 {
		t.Fatal("0 bits is off")
	}
}

func TestRate(t *testing.T) {
	psalm.New(nil)
	sr := psalm.SampleRate()
	out := run(New(Config{Rate: sr / 8, Mix: 1}), 0.9)
	changes := 0
	for i := 1; i < len(out); i++ {
		if out[i] != out[i-1] {
			changes++
		}
	}
	if want := len(out) / 8; changes > want+1 || changes < want-1 {
		t.Fatalf("an eighth of the rate holds every sample 8 times: %d changes in %d", changes, len(out))
	}
}

func TestMix(t *testing.T) {
	psalm.New(nil)
	c := New(Config{Bits: 1, Rate: 1000, Mix: 0})
	out := run(c, 0.5)
	sr := c.Config().SampleRate
	for i, x := range out {
		if want := 0.5 * math.Sin(2*math.Pi*220*float64(i)/sr); math.Abs(x-want) > 1e-6 {
			t.Fatal("mix 0 is dry")
		}
	}
}
