package compressor

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// level runs a sine (or nothing) of an amplitude through the compressor for some seconds, the
// sidechain optional, and returns the peak of the output over its last 50 ms in dB
func level(c *Compressor, amp, seconds float64, side func(n int) float64) float64 {
	bs := c.Config().BufferSize
	sr := c.Config().SampleRate
	buffers := int(seconds * sr / float64(bs))
	peak, n := 0.0, 0
	for b := range buffers {
		for i := range bs {
			x := float32(amp * math.Sin(2*math.Pi*440*float64(n)/sr))
			c.Input(0).Channels[0][i], c.Input(0).Channels[1][i] = x, x
			if side != nil {
				s := float32(side(n))
				c.Input(1).Channels[0][i], c.Input(1).Channels[1][i] = s, s
			}
			n++
		}
		c.Synthesize()
		if float64(b) >= float64(buffers)-0.05*sr/float64(bs) {
			for _, x := range c.Output(0).Channels[0] {
				peak = max(peak, math.Abs(float64(x)))
			}
		}
	}
	return 20 * math.Log10(peak)
}

func db(x float64) float64 { return math.Pow(10, x/20) }

func TestBelowThreshold(t *testing.T) {
	psalm.New(nil)
	c := New(Config{Threshold: -10, Ratio: 4, Attack: 5, Release: 50, Mix: 1})
	if out := level(c, db(-20), 0.5, nil); math.Abs(out+20) > 0.1 {
		t.Fatalf("below the threshold nothing should change: %.2f dB", out)
	}
}

func TestRatio(t *testing.T) {
	psalm.New(nil)
	c := New(Config{Threshold: -20, Ratio: 4, Attack: 1, Release: 50, Mix: 1})
	// 12 dB over the threshold at 4:1 comes out 3 dB over it
	if out := level(c, db(-8), 1, nil); math.Abs(out+17) > 0.5 {
		t.Fatalf("12 dB over at 4:1 should be 3 dB over (-17 dB): %.2f dB", out)
	}
	if r := c.Reduction(); math.Abs(r-9) > 0.6 {
		t.Fatalf("the reduction should be about 9 dB: %.2f", r)
	}
	if r := c.Reduction(); r != 0 {
		t.Fatal("the reduction is the most since it was read last")
	}
}

func TestMakeupAndMix(t *testing.T) {
	psalm.New(nil)
	c := New(Config{Threshold: -20, Ratio: 4, Attack: 1, Release: 50, Makeup: 6, Mix: 1})
	if out := level(c, db(-8), 1, nil); math.Abs(out+11) > 0.5 {
		t.Fatalf("6 dB of makeup: %.2f dB", out)
	}
	c = New(Config{Threshold: -20, Ratio: 4, Attack: 1, Release: 50, Mix: 0})
	if out := level(c, db(-8), 1, nil); math.Abs(out+8) > 0.1 {
		t.Fatalf("mix 0 is dry: %.2f dB", out)
	}
	c = New(Config{Threshold: -20, Ratio: 2, Knee: 0, AutoMakeup: true, Attack: 1, Release: 50, Mix: 1})
	// at 0 dB in a 2:1 above -20 gives -10 out: auto makeup is half of the 10 dB, 5 dB
	if out := level(c, 1, 1, nil); math.Abs(out+5) > 0.5 {
		t.Fatalf("auto makeup should give half of the reduction at 0 dB: %.2f dB", out)
	}
}

func TestAttackAndRelease(t *testing.T) {
	psalm.New(nil)
	c := New(Config{Threshold: -30, Ratio: 10, Attack: 300, Release: 500, Mix: 1})
	level(c, db(-40), 0.5, nil)
	fast := level(c, db(-6), 0.1, nil) // 100 ms after a loud start: not all the way down yet
	settled := level(c, db(-6), 2, nil)
	if fast < settled+3 {
		t.Fatalf("a 300 ms attack should not have settled after 100 ms: %.1f vs %.1f dB", fast, settled)
	}
	quiet := level(c, db(-40), 0.1, nil) // just after: still turned down (release 500 ms)
	if quiet > -42 {
		t.Fatalf("a slow release should still hold the level down: %.1f dB", quiet)
	}
}

func TestSidechainDucks(t *testing.T) {
	psalm.New(nil)
	c := New(Ducker(12, 100))
	sr := c.Config().SampleRate
	c.Input(1).AddConnection(&psalm.Connection{Module: psalm.NewControl(), Index: 0}) // something on the sidechain
	quiet := func(int) float64 { return 0 }
	kick := func(n int) float64 { return 0.9 * math.Sin(2*math.Pi*60*float64(n)/sr) }
	if out := level(c, db(-12), 0.5, quiet); math.Abs(out+12) > 0.3 {
		t.Fatalf("a silent sidechain leaves the sound alone: %.1f dB", out)
	}
	if out := level(c, db(-12), 0.5, kick); out > -20 {
		t.Fatalf("a loud sidechain should duck the sound: %.1f dB", out)
	}
	c.SetMix(0)
	if out := level(c, db(-12), 0.5, kick); math.Abs(out+12) > 0.3 {
		t.Fatalf("mix 0 leaves it dry: %.1f dB", out)
	}
}
