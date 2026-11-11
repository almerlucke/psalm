package chorus

import (
	"math"
	"testing"
)

const sr = 44100.0

// run feeds a signal per channel and returns both outputs
func run(c *Chorus, buffers int, left, right func(n int) float64) ([]float64, []float64) {
	var l, r []float64
	bs := c.Config().BufferSize
	for b := range buffers {
		for i := range bs {
			c.Input(0).Channels[0][i] = float32(left(b*bs + i))
			c.Input(0).Channels[1][i] = float32(right(b*bs + i))
		}
		c.Synthesize()
		for i := range bs {
			l = append(l, float64(c.Output(0).Channels[0][i]))
			r = append(r, float64(c.Output(0).Channels[1][i]))
		}
	}
	return l, r
}

func tone(f float64) func(int) float64 {
	return func(n int) float64 { return 0.5 * math.Sin(2*math.Pi*f*float64(n)/sr) }
}

func silent(int) float64 { return 0 }

func rms(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

// Fully wet the chorus keeps about the level of its input, and every side its own input
func TestLevelAndStereo(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mix = 1
	l, r := run(New(cfg), 80, tone(220), silent)
	l, r = l[20*1024:], r[20*1024:]
	if g := 20 * math.Log10(rms(l)/(0.5/math.Sqrt2)); math.Abs(g) > 1.5 {
		t.Fatalf("expected about unity gain, got %.1f dB", g)
	}
	if rms(r) > 1e-9 {
		t.Fatalf("the right side should stay silent with a left only input, rms %v", rms(r))
	}
}

// The sides are modulated differently with spread, the same without
func TestSpread(t *testing.T) {
	corr := func(spread float64) float64 {
		cfg := DefaultConfig()
		cfg.Mix = 1
		cfg.Spread = spread
		l, r := run(New(cfg), 80, tone(3000), tone(3000))
		l, r = l[20*1024:], r[20*1024:]
		s, a, b := 0.0, 0.0, 0.0
		for i := range l {
			s += l[i] * r[i]
			a += l[i] * l[i]
			b += r[i] * r[i]
		}
		return s / math.Sqrt(a*b)
	}
	if c := corr(0); c < 0.999 {
		t.Fatalf("without spread both sides are the same, correlation %v", c)
	}
	if c := corr(1); c > 0.9 {
		t.Fatalf("with spread the sides differ, correlation %v", c)
	}
}

// Moving the settings while playing glides: no steps in the output
func TestNoClicksWhenChanging(t *testing.T) {
	c := New(DefaultConfig())
	n := 0
	in := func(int) float64 { n++; return tone(110)(n) } // one tone through both runs
	run(c, 10, in, silent)
	c.ReceiveMessage(map[string]any{"delay": 30.0, "depth": 8.0, "mix": 1.0})
	l, _ := run(c, 10, in, silent)
	step := 0.0
	for i := 1; i < len(l); i++ {
		step = max(step, math.Abs(l[i]-l[i-1]))
	}
	// a 110 Hz sine of 0.5 moves at most 0.008 per sample, the voices add up to sqrt(3) and the
	// glide bends the pitch a little
	if step > 0.02 {
		t.Fatalf("the output stepped by %v", step)
	}
}

func TestFeedbackIsBounded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Feedback = 5
	cfg.Mix = 1
	l, _ := run(New(cfg), 200, tone(500), tone(500))
	for _, v := range l {
		if math.IsNaN(v) || math.Abs(v) > 10 {
			t.Fatalf("unbounded %v", v)
		}
	}
}

// noise returns noise rolled off above about 1 kHz, more like music than white noise, the same
// for the same seed
func noise(seed uint64) func(int) float64 {
	lp := 0.0
	return func(int) float64 {
		seed = seed*6364136223846793005 + 1442695040888963407
		lp += 0.15 * (float64(seed>>11)/(1<<53)*2 - 1 - lp)
		return lp
	}
}

// The mix crossfades at equal power: dry, halfway and wet are about as loud
func TestMixKeepsLoudness(t *testing.T) {
	level := func(mix float64) float64 {
		cfg := DefaultConfig()
		cfg.Mix = mix
		cfg.Tone = 0 // only the crossfade
		l, _ := run(New(cfg), 80, noise(1), noise(2))
		return 20 * math.Log10(rms(l[20*1024:]))
	}
	dry := level(0)
	for _, mix := range []float64{0.25, 0.5, 0.75, 1} {
		if d := level(mix) - dry; math.Abs(d) > 1 {
			t.Fatalf("mix %v changes the level by %.1f dB", mix, d)
		}
	}
}
