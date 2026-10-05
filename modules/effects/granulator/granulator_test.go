package granulator

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/almerlucke/psalm/blocks/fft"
)

const sr = 44100.0

// run feeds a mono signal (by sample index) into both inputs and returns the left output
func run(g *Granulator, buffers int, input func(n int) float64) []float64 {
	bs := g.Config().BufferSize
	var out []float64
	for b := range buffers {
		for i := range bs {
			v := float32(input(b*bs + i))
			g.Input(0).Channels[0][i] = v
			g.Input(0).Channels[1][i] = v
		}
		g.Synthesize()
		for _, s := range g.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}
	return out
}

func sine(freq float64) func(int) float64 {
	return func(n int) float64 { return 0.5 * math.Sin(2*math.Pi*freq*float64(n)/sr) }
}

// bandEnergy returns the spectral energy around freq, averaged over overlapping windows
func bandEnergy(x []float64, freq float64) float64 {
	const n = 8192
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	e := 0.0
	for start := 0; start+n <= len(x); start += n / 2 {
		for i := range n {
			re[i] = x[start+i] * w[i]
			im[i] = 0
		}
		f.Forward(re, im)
		c := int(freq * n / sr)
		for k := c - 8; k <= c+8; k++ {
			e += re[k]*re[k] + im[k]*im[k]
		}
	}
	return e
}

func wetOnly() Config {
	cfg := DefaultConfig()
	cfg.Mix = 1
	return cfg
}

func TestDryMixPassesInput(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mix = 0
	g := New(cfg)
	out := run(g, 4, sine(440))
	for i, v := range out {
		if math.Abs(v-float64(float32(sine(440)(i)))) > 1e-6 {
			t.Fatalf("dry signal changed at %d", i)
		}
	}
}

func TestPitchShift(t *testing.T) {
	cfg := wetOnly()
	cfg.Pitch = 12
	out := run(New(cfg), 80, sine(440))[20*1024:]
	if up, root := bandEnergy(out, 880), bandEnergy(out, 440); up < 20*root {
		t.Fatalf("expected the grains an octave up: 880 Hz %v, 440 Hz %v", up, root)
	}
}

func TestFreezeHoldsTheBuffer(t *testing.T) {
	// 2 seconds of 440 Hz, then the input switches to 1000 Hz
	input := func(n int) float64 {
		if n < int(2*sr) {
			return sine(440)(n)
		}
		return sine(1000)(n)
	}
	switchBuffer := int(2*sr) / 1024

	measure := func(freeze bool) (float64, float64) {
		g := New(wetOnly())
		run(g, switchBuffer, input)
		g.SetFreeze(freeze)
		bs := g.Config().BufferSize
		var out []float64
		for b := switchBuffer; b < switchBuffer+80; b++ {
			for i := range bs {
				v := float32(input(b*bs + i))
				g.Input(0).Channels[0][i] = v
				g.Input(0).Channels[1][i] = v
			}
			g.Synthesize()
			for _, s := range g.Output(0).Channels[0] {
				out = append(out, float64(s))
			}
		}
		out = out[20*bs:] // after the grains of the old input are gone
		return bandEnergy(out, 440), bandEnergy(out, 1000)
	}

	old, now := measure(true)
	if old < 20*now {
		t.Fatalf("frozen: expected the old 440 Hz to keep playing, 440 %v, 1000 %v", old, now)
	}
	old, now = measure(false)
	if now < 20*old {
		t.Fatalf("not frozen: expected the new 1000 Hz input, 440 %v, 1000 %v", old, now)
	}
}

func TestSilenceDiesAwayWithoutFreeze(t *testing.T) {
	g := New(wetOnly())
	input := func(n int) float64 {
		if n < int(sr) {
			return sine(440)(n)
		}
		return 0
	}
	out := run(g, int(7*sr)/1024, input)
	tail := out[len(out)-int(sr):]
	for _, v := range tail {
		if math.Abs(v) > 1e-6 {
			t.Fatalf("expected silence once the input is silent and the buffer has passed, got %v", v)
		}
	}
}

func TestGrainsStayInValidAudio(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for trial := range 2000 {
		cfg := DefaultConfig()
		cfg.Pitch = r.Float64()*48 - 24
		cfg.Reverse = r.Float64()
		cfg.GrainSize = r.Float64()*800 + 5
		cfg.Delay = r.Float64() * 4000
		cfg.Spray = r.Float64() * 2000
		cfg.Freeze = r.IntN(2) == 0

		g := New(cfg)
		n := int64(len(g.buf[0]))
		g.written = int64(r.IntN(int(3 * n))) // anywhere from a part to a fully recorded buffer

		for range 10 {
			g.spawn(sr)
		}
		for _, gr := range g.grains {
			for k := 0; k <= gr.length; k++ {
				pos := gr.pos + float64(k)*gr.inc
				head := float64(g.written)
				if !cfg.Freeze {
					head += float64(k) // the write head moves on while the grain plays
				}
				oldest := math.Max(0, head-float64(n))
				if pos > head-margin+1e-6 || pos < oldest-1e-6 {
					t.Fatalf("trial %d (freeze %v): grain reads %v at step %d, valid is %v - %v",
						trial, cfg.Freeze, pos, k, oldest, head-margin)
				}
			}
		}
		g.grains = g.grains[:0]
	}
}

func TestFeedbackIsBounded(t *testing.T) {
	cfg := wetOnly()
	cfg.Feedback = 1
	cfg.Density = 100
	cfg.GrainSize = 300
	cfg.Delay = 50
	out := run(New(cfg), int(10*sr)/1024, sine(220))
	for i, v := range out {
		if math.IsNaN(v) || math.Abs(v) > 4 {
			t.Fatalf("unbounded output at %d: %v", i, v)
		}
	}
}
