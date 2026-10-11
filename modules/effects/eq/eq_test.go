package eq

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// gain returns the gain (dB) of the EQ at a frequency
func gain(e *EQ, freq float64) float64 {
	bs, sr := e.Config().BufferSize, e.Config().SampleRate
	peak, n := 0.0, 0
	for b := range int(sr / float64(bs)) {
		for i := range bs {
			x := float32(0.5 * math.Sin(2*math.Pi*freq*float64(n)/sr))
			e.Input(0).Channels[0][i], e.Input(0).Channels[1][i] = x, x
			n++
		}
		e.Synthesize()
		if b > int(0.5*sr/float64(bs)) {
			for _, x := range e.Output(0).Channels[0] {
				peak = math.Max(peak, math.Abs(float64(x)))
			}
		}
	}
	return 20 * math.Log10(peak/0.5)
}

func TestFlat(t *testing.T) {
	psalm.New(nil)
	for _, f := range []float64{50, 500, 5000, 15000} {
		if g := gain(New(DefaultConfig()), f); math.Abs(g) > 0.05 {
			t.Fatalf("flat at %v Hz: %.2f dB", f, g)
		}
	}
}

func TestBands(t *testing.T) {
	psalm.New(nil)
	cfg := DefaultConfig()
	cfg.Low = Band{Freq: 150, Gain: 6}
	cfg.Mid1 = Band{Freq: 1000, Gain: -12, Q: 2}
	cfg.High = Band{Freq: 6000, Gain: 4}
	for _, c := range []struct{ freq, want, tol float64 }{{40, 6, 0.5}, {1000, -12, 0.3}, {15000, 4, 0.5}, {400, 0, 1.5}} {
		if g := gain(New(cfg), c.freq); math.Abs(g-c.want) > c.tol {
			t.Fatalf("%v Hz: %.2f dB, expected %v", c.freq, g, c.want)
		}
	}
}

func TestCuts(t *testing.T) {
	psalm.New(nil)
	cfg := DefaultConfig()
	cfg.LowCut, cfg.HighCut = 200, 3000
	if g := gain(New(cfg), 50); g > -20 {
		t.Fatalf("the low cut takes 50 Hz away: %.1f dB", g)
	}
	if g := gain(New(cfg), 12000); g > -20 {
		t.Fatalf("the high cut takes 12 kHz away: %.1f dB", g)
	}
	if g := gain(New(cfg), 800); math.Abs(g) > 0.5 {
		t.Fatalf("in between it passes: %.1f dB", g)
	}
}

func TestGlides(t *testing.T) {
	psalm.New(nil)
	e := New(DefaultConfig())
	gain(e, 1000)
	e.ReceiveMessage(psalm.Message{"mid1Freq": 1000.0, "mid1Gain": -12.0, "mid1Q": 2.0})
	e.Synthesize()
	if g := e.now[mid1].Gain; g >= 0 || g <= -12 {
		t.Fatalf("a change glides: a part of the way after a buffer, %.2f", g)
	}
	if g := gain(e, 1000); math.Abs(g+12) > 0.3 {
		t.Fatalf("and gets there: %.2f dB", g)
	}
}
