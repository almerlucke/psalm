package phaser

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// level returns the level (dB) of a sine through the phaser after a settling half second
func level(p *Phaser, freq float64) float64 {
	bs, sr := p.Config().BufferSize, p.Config().SampleRate
	peak, n := 0.0, 0
	for b := range int(sr / float64(bs)) {
		for i := range bs {
			x := float32(0.5 * math.Sin(2*math.Pi*freq*float64(n)/sr))
			p.Input(0).Channels[0][i], p.Input(0).Channels[1][i] = x, x
			n++
		}
		p.Synthesize()
		if b > int(0.5*sr/float64(bs)) {
			for _, x := range p.Output(0).Channels[0] {
				peak = max(peak, math.Abs(float64(x)))
			}
		}
	}
	return 20 * math.Log10(peak/0.5)
}

func TestNotches(t *testing.T) {
	psalm.New(nil)
	// the sweep held at 1 kHz: four stages turn 1 kHz a full turn (it passes), and half a turn at
	// tan(22.5°) and tan(67.5°) times it: the notches at 414 Hz and 2414 Hz
	frozen := func() *Phaser { return New(Config{Stages: 4, Rate: 0, Min: 1000, Max: 1000, Mix: 0.5}) }
	if l := level(frozen(), 1000); math.Abs(l) > 0.5 {
		t.Fatalf("at the frequency of the stages it passes: %.1f dB", l)
	}
	for _, f := range []float64{1000 * math.Tan(math.Pi/8), 1000 * math.Tan(3*math.Pi/8)} {
		if l := level(frozen(), f); l > -20 {
			t.Fatalf("a notch at %.0f Hz: %.1f dB", f, l)
		}
	}
	if l := level(New(Config{Stages: 4, Rate: 0, Min: 1000, Max: 1000, Mix: 0}), 414); math.Abs(l) > 0.1 {
		t.Fatalf("mix 0 is dry: %.1f dB", l)
	}
}

func TestSweep(t *testing.T) {
	psalm.New(nil)
	// sweeping, the notch passes over a tone: its level goes up and down
	p := New(Config{Stages: 4, Rate: 2, Min: 200, Max: 4000, Mix: 0.5})
	bs, sr := p.Config().BufferSize, p.Config().SampleRate
	lo, hi, n := 1.0, 0.0, 0
	for range int(2 * sr / float64(bs)) {
		for i := range bs {
			x := float32(0.5 * math.Sin(2*math.Pi*800*float64(n)/sr))
			p.Input(0).Channels[0][i], p.Input(0).Channels[1][i] = x, x
			n++
		}
		p.Synthesize()
		peak := 0.0
		for _, x := range p.Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(x)))
		}
		lo, hi = min(lo, peak), max(hi, peak)
	}
	if hi < 0.45 || lo > 0.25 {
		t.Fatalf("the sweep should move a notch over the tone: %.2f to %.2f", lo, hi)
	}
}

func TestSync(t *testing.T) {
	psalm.New(nil)
	p := New(Config{Sync: 2})
	if r := p.rate(); math.Abs(r-psalm.BPM()/120) > 1e-9 {
		t.Fatalf("a sweep of 2 beats: %v per second", r)
	}
}
