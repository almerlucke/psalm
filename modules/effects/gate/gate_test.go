package gate

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// level runs a sine of an amplitude for some seconds and returns the peak of the last 50 ms in dB
func level(g *Gate, amp, seconds float64) float64 {
	bs, sr := g.Config().BufferSize, g.Config().SampleRate
	buffers := int(seconds * sr / float64(bs))
	peak, n := 0.0, 0
	for b := range buffers {
		for i := range bs {
			x := float32(amp * math.Sin(2*math.Pi*440*float64(n)/sr))
			g.Input(0).Channels[0][i], g.Input(0).Channels[1][i] = x, x
			n++
		}
		g.Synthesize()
		if float64(b) >= float64(buffers)-0.05*sr/float64(bs) {
			for _, x := range g.Output(0).Channels[0] {
				peak = math.Max(peak, math.Abs(float64(x)))
			}
		}
	}
	if peak == 0 {
		return -200
	}
	return 20 * math.Log10(peak)
}

func db(x float64) float64 { return math.Pow(10, x/20) }

func TestGate(t *testing.T) {
	psalm.New(nil)
	g := New(Config{Threshold: -40, Range: 60, Attack: 1, Hold: 20, Release: 50})
	if out := level(g, db(-20), 0.5); math.Abs(out+20) > 0.2 {
		t.Fatalf("above the threshold the gate is open: %.1f dB", out)
	}
	if out := level(g, db(-50), 1); math.Abs(out+110) > 1 {
		t.Fatalf("below it the gate closes by its range (60 dB): %.1f dB", out)
	}
	if out := level(g, db(-20), 0.2); math.Abs(out+20) > 0.2 {
		t.Fatalf("and opens again: %.1f dB", out)
	}
}

func TestExpander(t *testing.T) {
	psalm.New(nil)
	g := New(Config{Threshold: -30, Ratio: 2, Range: 40, Attack: 1, Hold: 0, Release: 20})
	// 10 dB under the threshold at 1:2 goes 10 dB further down
	if out := level(g, db(-40), 1); math.Abs(out+50) > 0.5 {
		t.Fatalf("an expander: 10 dB under becomes 20 dB under: %.1f dB", out)
	}
	// 30 dB under would be 30 dB more, the range stops it at 40 in all
	if out := level(g, db(-70), 1); math.Abs(out+110) > 0.5 {
		t.Fatalf("the range limits it: %.1f dB", out)
	}
}

func TestHold(t *testing.T) {
	psalm.New(nil)
	g := New(Config{Threshold: -40, Range: 60, Attack: 1, Hold: 300, Release: 5})
	level(g, db(-20), 0.5)
	if out := level(g, db(-50), 0.15); out < -51 {
		t.Fatalf("within the hold the gate stays open: %.1f dB", out)
	}
	if out := level(g, db(-50), 0.5); out > -100 {
		t.Fatalf("after the hold it closes: %.1f dB", out)
	}
}
