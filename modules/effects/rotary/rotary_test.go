package rotary

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// envelope runs a sine for some seconds and returns the peak of every 10 ms of the left output
func envelope(r *Rotary, freq, seconds float64) []float64 {
	bs, sr := r.Config().BufferSize, r.Config().SampleRate
	win := int(0.01 * sr)
	var env []float64
	peak, k, n := 0.0, 0, 0
	for range int(seconds * sr / float64(bs)) {
		for i := range bs {
			x := float32(0.5 * math.Sin(2*math.Pi*freq*float64(n)/sr))
			r.Input(0).Channels[0][i], r.Input(0).Channels[1][i] = x, x
			n++
		}
		r.Synthesize()
		for _, x := range r.Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(x)))
			if k++; k == win {
				env, peak, k = append(env, peak), 0, 0
			}
		}
	}
	return env
}

// cycles counts the times the envelope rises through its middle
func cycles(env []float64) int {
	lo, hi := math.Inf(1), 0.0
	for _, e := range env {
		lo, hi = min(lo, e), max(hi, e)
	}
	mid, n := (lo+hi)/2, 0
	for i := 1; i < len(env); i++ {
		if env[i-1] < mid && env[i] >= mid {
			n++
		}
	}
	return n
}

func TestSpeeds(t *testing.T) {
	psalm.New(nil)
	slow := New(Config{Mix: 1})
	envelope(slow, 3000, 1) // settle
	if n := cycles(envelope(slow, 3000, 3)); n < 2 || n > 3 {
		t.Fatalf("slow, the horn turns about 0.8 times a second: %d in 3 s", n)
	}
	fast := New(Config{Fast: true, Mix: 1})
	envelope(fast, 3000, 1)
	if n := cycles(envelope(fast, 3000, 2)); n < 11 || n > 15 {
		t.Fatalf("fast, the horn turns about 6.7 times a second: %d in 2 s", n)
	}
}

func TestSpeedingUp(t *testing.T) {
	psalm.New(nil)
	r := New(Config{Mix: 1})
	r.SetFast(true)
	envelope(r, 440, 0.25)
	horn, drum := r.Speeds()
	if horn < 2 || horn > 6.5 || drum > 2 {
		t.Fatalf("a quarter second after switching the horn is on its way, the drum barely: %.2f, %.2f", horn, drum)
	}
	envelope(r, 440, 6)
	horn, drum = r.Speeds()
	if math.Abs(horn-6.7) > 0.1 || math.Abs(drum-5.8) > 0.3 {
		t.Fatalf("after six seconds both are fast: %.2f, %.2f", horn, drum)
	}
}

func TestStereoAndMix(t *testing.T) {
	psalm.New(nil)
	r := New(Config{Mix: 1})
	envelope(r, 3000, 0.5)
	if l, rr := r.Output(0).Channels[0], r.Output(0).Channels[1]; l[100] == rr[100] {
		t.Fatal("two microphones: left and right differ")
	}
	dry := New(Config{Mix: 0})
	env := envelope(dry, 440, 0.5)
	for _, e := range env[5:] {
		if math.Abs(e-0.5) > 0.01 {
			t.Fatalf("mix 0 is dry: %v", e)
		}
	}
}
