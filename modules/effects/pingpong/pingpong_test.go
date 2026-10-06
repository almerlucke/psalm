package pingpong

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// run feeds an impulse (on the left, right or both sides) and returns the output channels
func run(pp *PingPong, seconds float64, left, right float64) ([]float64, []float64) {
	bs := pp.Config().BufferSize
	n := int(seconds * pp.Config().SampleRate / float64(bs))
	var l, r []float64
	for b := range n {
		pp.Input(0).Channels.Clear()
		if b == 0 {
			pp.Input(0).Channels[0][0] = float32(left)
			pp.Input(0).Channels[1][0] = float32(right)
		}
		pp.Synthesize()
		for i := range bs {
			l = append(l, float64(pp.Output(0).Channels[0][i]))
			r = append(r, float64(pp.Output(0).Channels[1][i]))
		}
	}
	return l, r
}

// peakNear returns the largest value within a few samples of position
func peakNear(x []float64, pos int) float64 {
	m := 0.0
	for i := max(0, pos-4); i < min(len(x), pos+5); i++ {
		m = math.Max(m, math.Abs(x[i]))
	}
	return m
}

func wetOnly() Config {
	cfg := DefaultConfig()
	cfg.Mix = 1
	cfg.HighCut = 0
	cfg.LowCut = 0
	cfg.Feedback = 0.5
	return cfg
}

func TestEchoTimeFollowsTheTempo(t *testing.T) {
	sr := psalm.SampleRate()
	for _, bpm := range []float64{120, 90} {
		cfg := psalm.CurrentConfig()
		cfg.BPM = bpm
		psalm.PushConfig(cfg)

		c := wetOnly()
		c.Time = 0.5
		l, _ := run(New(c), 1, 1, 1)
		echo := int(math.Round(0.5 * 60 / bpm * sr))
		if peakNear(l, echo) < 0.5 {
			t.Errorf("%v BPM: expected the first echo at %d samples", bpm, echo)
		}
		if peakNear(l, echo/2) > 1e-6 {
			t.Errorf("%v BPM: echo too early", bpm)
		}
		psalm.PopConfig()
	}
}

func TestTempoChangeWhilePlaying(t *testing.T) {
	cfg := psalm.CurrentConfig()
	cfg.BPM = 120
	psalm.PushConfig(cfg)
	defer psalm.PopConfig()

	pp := New(wetOnly()) // 0.75 beats
	run(pp, 0.1, 0, 0)

	cfg.BPM = 60
	psalm.PushConfig(cfg)
	defer psalm.PopConfig()
	run(pp, 1, 0, 0) // the time glides to the new tempo
	if want := 0.75 * psalm.SampleRate(); math.Abs(pp.samples-want) > 1 {
		t.Fatalf("expected the echo time to follow the tempo to %v samples, got %v", want, pp.samples)
	}
}

func TestTimeMsIgnoresTheTempo(t *testing.T) {
	c := wetOnly()
	c.TimeMs = 100
	l, _ := run(New(c), 0.5, 1, 1)
	if peakNear(l, int(0.1*psalm.SampleRate())) < 0.5 {
		t.Fatal("expected the echo after 100 ms")
	}
}

// A centred sound bounces: the first echo on the left, the second on the right, each quieter by
// the feedback
func TestPingPong(t *testing.T) {
	c := wetOnly()
	c.TimeMs = 100
	l, r := run(New(c), 0.5, 1, 1)
	step := int(0.1 * psalm.SampleRate())
	if peakNear(l, step) < 0.9 || peakNear(r, step) > 1e-6 {
		t.Fatalf("first echo: left %v right %v", peakNear(l, step), peakNear(r, step))
	}
	if peakNear(r, 2*step) < 0.45 || peakNear(r, 2*step) > 0.55 || peakNear(l, 2*step) > 1e-6 {
		t.Fatalf("second echo: left %v right %v", peakNear(l, 2*step), peakNear(r, 2*step))
	}
	if third := peakNear(l, 3*step); math.Abs(third-0.25) > 0.03 {
		t.Fatalf("third echo %v, expected 0.25", third)
	}
}

// A plain stereo delay keeps every side to itself
func TestStereo(t *testing.T) {
	c := wetOnly()
	c.TimeMs = 100
	c.Stereo = 1
	l, r := run(New(c), 0.5, 1, 0)
	step := int(0.1 * psalm.SampleRate())
	if peakNear(l, step) < 0.9 || peakNear(l, 2*step) < 0.45 {
		t.Fatal("expected the left echoes on the left")
	}
	for i := range r {
		if math.Abs(r[i]) > 1e-9 {
			t.Fatalf("nothing should echo on the right, got %v at %d", r[i], i)
		}
	}
}

func TestRepeatsDarken(t *testing.T) {
	c := wetOnly()
	c.TimeMs = 50
	c.Feedback = 0.9
	c.HighCut = 3000
	l, r := run(New(c), 1, 1, 1)
	// a darker echo is spread out in time: its peak falls faster than its energy
	crest := func(x []float64, pos int) float64 {
		peak, sum := 0.0, 0.0
		for i := pos - 20; i < pos+200; i++ {
			peak = math.Max(peak, math.Abs(x[i]))
			sum += x[i] * x[i]
		}
		return peak / math.Sqrt(sum)
	}
	step := int(0.05 * psalm.SampleRate())
	if first, fifth := crest(l, step), crest(l, 5*step); fifth > first*0.8 {
		t.Fatalf("expected later echoes to be smoother (darker), crest %v vs %v", first, fifth)
	}
	_ = r
}

func TestFeedbackIsBounded(t *testing.T) {
	c := wetOnly()
	c.TimeMs = 20
	c.Feedback = 5
	l, r := run(New(c), 10, 1, 1)
	for i := range l {
		if math.IsNaN(l[i]) || math.Abs(l[i]) > 1 || math.Abs(r[i]) > 1 {
			t.Fatalf("unbounded feedback at %d: %v %v", i, l[i], r[i])
		}
	}
}

func TestDry(t *testing.T) {
	c := DefaultConfig()
	c.Mix = 0
	l, r := run(New(c), 0.2, 0.7, -0.3)
	if l[0] != float64(float32(0.7)) || r[0] != float64(float32(-0.3)) {
		t.Fatal("mix 0 should pass the input")
	}
}

func TestMessages(t *testing.T) {
	pp := New(DefaultConfig())
	pp.ReceiveMessage(psalm.Message{"time": 1.5, "feedback": 0.2, "mix": 0.6, "stereo": 0.5, "highCut": 2000, "lowCut": 300})
	s := pp.Settings()
	if s.Time != 1.5 || s.Feedback != 0.2 || s.Mix != 0.6 || s.Stereo != 0.5 || s.HighCut != 2000 || s.LowCut != 300 {
		t.Fatalf("unexpected settings %+v", s)
	}
	pp.ReceiveMessage(psalm.Message{"timeMs": 250})
	pp.Receive(0.25, 0) // back to beats
	if s := pp.Settings(); s.TimeMs != 0 || s.Time != 0.25 {
		t.Fatalf("expected the time back in beats, got %+v", s)
	}
}
