package shimmer

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm/blocks/fft"
)

// run feeds the input function into both channels and returns the left output
func run(s *Shimmer, seconds float64, input func(n int) float64) []float64 {
	bufSize := s.Config().BufferSize
	numBuffers := int(seconds * s.Config().SampleRate / float64(bufSize))
	var out []float64

	for b := range numBuffers {
		for i := range bufSize {
			v := float32(input(b*bufSize + i))
			s.Input(0).Channels[0][i] = v
			s.Input(0).Channels[1][i] = v
		}
		s.Synthesize()
		for _, v := range s.Output(0).Channels[0] {
			out = append(out, float64(v))
		}
	}

	return out
}

// bandEnergy returns the spectral energy around freq of the signal segment
func bandEnergy(x []float64, sr float64, freq float64) float64 {
	const n = 16384
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	for i := range n {
		re[i] = x[i] * w[i]
	}
	f.Forward(re, im)

	e := 0.0
	center := int(freq * n / sr)
	for k := center - 8; k <= center+8; k++ {
		e += re[k]*re[k] + im[k]*im[k]
	}
	return e
}

func TestShimmerAddsOctavesToTheTail(t *testing.T) {
	burst := func(sr float64) func(n int) float64 {
		return func(n int) float64 {
			if float64(n) > 0.3*sr {
				return 0
			}
			return 0.3 * math.Sin(2*math.Pi*440*float64(n)/sr)
		}
	}

	tailRatio := func(amount float64) float64 {
		cfg := DefaultConfig()
		cfg.Shimmer = amount
		cfg.Mix = 1.0
		s := New(cfg)
		sr := s.Config().SampleRate

		out := run(s, 3.0, burst(sr))
		tail := out[int(1.5*sr):]
		up := bandEnergy(tail, sr, 880) + bandEnergy(tail, sr, 1760)
		return up / bandEnergy(tail, sr, 440)
	}

	plain, shimmer := tailRatio(0.0), tailRatio(0.7)
	if shimmer < plain*20 {
		t.Fatalf("expected much more octave-up energy with shimmer: %v vs %v", shimmer, plain)
	}
}

func TestMaximumShimmerIsBounded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Shimmer = 1.0
	cfg.Reverb.Decay = 20.0
	cfg.Reverb.Damping = 0.0
	cfg.Mix = 1.0
	s := New(cfg)
	sr := s.Config().SampleRate

	out := run(s, 10.0, func(n int) float64 {
		if float64(n) > 0.5*sr {
			return 0
		}
		return 0.8 * math.Sin(2*math.Pi*220*float64(n)/sr)
	})

	for i, v := range out {
		if math.IsNaN(v) || math.Abs(v) > 2.0 {
			t.Fatalf("unbounded output at %d: %v", i, v)
		}
	}
}

func TestDryMixPassesInput(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mix = 0.0
	s := New(cfg)

	out := run(s, 0.1, func(n int) float64 { return math.Sin(float64(n) * 0.05) })
	for i, v := range out {
		if math.Abs(v-float64(float32(math.Sin(float64(i)*0.05)))) > 1e-6 {
			t.Fatalf("dry signal changed at %d", i)
		}
	}
}
