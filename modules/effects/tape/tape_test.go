package tape

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// run feeds a short burst (or a steady sine) and returns the left output
func run(t *Tape, seconds float64, in func(n int) float64) []float64 {
	bs, sr := t.Config().BufferSize, t.Config().SampleRate
	var out []float64
	n := 0
	for range int(seconds * sr / float64(bs)) {
		for i := range bs {
			x := float32(in(n))
			t.Input(0).Channels[0][i], t.Input(0).Channels[1][i] = x, x
			n++
		}
		t.Synthesize()
		for _, x := range t.Output(0).Channels[0] {
			out = append(out, float64(x))
		}
	}
	return out
}

func burst(n int) float64 {
	if n < 200 {
		return 0.5 * math.Sin(2*math.Pi*1000*float64(n)/44100)
	}
	return 0
}

func peakAround(x []float64, at, width int) float64 {
	m := 0.0
	for i := max(0, at-width); i < min(len(x), at+width); i++ {
		m = max(m, math.Abs(x[i]))
	}
	return m
}

func TestEchoes(t *testing.T) {
	psalm.New(nil)
	tp := New(Config{TimeMs: 250, Feedback: 0.5, Mix: 1, HighCut: 0, LowCut: 0})
	out := run(tp, 1.5, burst)
	echo := int(0.25 * psalm.SampleRate())
	first, second := peakAround(out, echo, 800), peakAround(out, 2*echo, 800)
	if first < 0.3 || second > first*0.7 || second < first*0.3 {
		t.Fatalf("an echo after 250 ms, the next about half as loud: %.3f, %.3f", first, second)
	}
	if early := peakAround(out, echo/2, 300); early > 0.01 {
		t.Fatalf("nothing before the echo: %.3f", early)
	}
}

func TestWaver(t *testing.T) {
	psalm.New(nil)
	tp := New(Config{TimeMs: 100, Feedback: 0, Wow: 1, Flutter: 1, Mix: 1})
	// a steady tone: its echo wavers in pitch, so wet and the tone apart drift in and out of phase
	sr := psalm.SampleRate()
	tone := func(n int) float64 { return 0.5 * math.Sin(2*math.Pi*2000*float64(n)/sr) }
	out := run(tp, 2, tone)
	// the zero crossings of the echo don't keep the spacing of 2 kHz
	var gaps []float64
	last := -1.0
	for i := int(sr); i < len(out); i++ {
		if out[i-1] < 0 && out[i] >= 0 {
			pos := float64(i-1) + out[i-1]/(out[i-1]-out[i])
			if last > 0 {
				gaps = append(gaps, pos-last)
			}
			last = pos
		}
	}
	lo, hi := math.Inf(1), 0.0
	for _, g := range gaps {
		lo, hi = min(lo, g), max(hi, g)
	}
	if hi-lo < 0.05 {
		t.Fatalf("wow and flutter should waver the pitch: periods from %.3f to %.3f samples", lo, hi)
	}
}

func TestSelfOscillationStaysInBounds(t *testing.T) {
	psalm.New(nil)
	tp := New(Config{TimeMs: 120, Feedback: 1.2, Mix: 1, HighCut: 5000, LowCut: 100})
	out := run(tp, 5, burst)
	peak := 0.0
	for _, x := range out {
		peak = max(peak, math.Abs(x))
	}
	if peak > 1.5 || math.IsNaN(peak) {
		t.Fatalf("feedback above 1 builds up, the saturation keeps it in bounds: %.2f", peak)
	}
	if late := peakAround(out, len(out)-2000, 1500); late < 0.1 {
		t.Fatalf("and it goes on singing: %.3f", late)
	}
}

func TestFollowsTheTempo(t *testing.T) {
	psalm.New(nil)
	tp := New(Config{Time: 0.5, Feedback: 0, Mix: 1})
	if want := 0.5 * 60 / psalm.BPM() * psalm.SampleRate(); math.Abs(tp.EchoSamples()-want) > 1 {
		t.Fatalf("half a beat: %.1f samples, not %.1f", tp.EchoSamples(), want)
	}
}
