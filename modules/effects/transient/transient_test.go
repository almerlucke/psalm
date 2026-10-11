package transient

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// hit plays a drum like hit (a 150 Hz tone, sharp start, 200 ms decay) and returns the peak of its
// first 10 ms and of 150 to 250 ms
func hit(s *Shaper) (start, tail float64) {
	bs, sr := s.Config().BufferSize, s.Config().SampleRate
	n := 0
	for range int(0.4 * sr / float64(bs)) {
		for i := range bs {
			t := float64(n) / sr
			x := float32(0.8 * math.Exp(-t/0.2) * math.Sin(2*math.Pi*150*t))
			s.Input(0).Channels[0][i], s.Input(0).Channels[1][i] = x, x
			n++
		}
		s.Synthesize()
		for i, x := range s.Output(0).Channels[0] {
			t := float64(n-bs+i) / sr
			switch {
			case t < 0.01:
				start = math.Max(start, math.Abs(float64(x)))
			case t > 0.15 && t < 0.25:
				tail = math.Max(tail, math.Abs(float64(x)))
			}
		}
	}
	return start, tail
}

func TestShapes(t *testing.T) {
	psalm.New(nil)
	s0, t0 := hit(New(Config{Mix: 1}))
	s1, t1 := hit(New(Config{Attack: 12, Mix: 1}))
	if s1 < s0*1.5 || math.Abs(t1-t0) > t0*0.2 {
		t.Fatalf("attack turns up the start, not the decay: start %.3f->%.3f, decay %.3f->%.3f", s0, s1, t0, t1)
	}
	s2, t2 := hit(New(Config{Sustain: -12, Mix: 1}))
	if t2 > t0*0.6 || s2 < s0*0.8 {
		t.Fatalf("less sustain turns the decay down, not the start: start %.3f->%.3f, decay %.3f->%.3f", s0, s2, t0, t2)
	}
	s3, t3 := hit(New(Config{Attack: 12, Sustain: -12, Mix: 0}))
	if s3 != s0 || t3 != t0 {
		t.Fatal("mix 0 is dry")
	}
}
