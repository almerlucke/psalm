package vocoder

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/almerlucke/psalm/blocks/fft"
)

// run feeds modulator and carrier sample functions through the vocoder and returns the output
func run(v *Vocoder, numBuffers int, mod func(i int) float64, car func(i int) float64) []float64 {
	bufSize := v.Config().BufferSize
	out := make([]float64, 0, numBuffers*bufSize)

	for b := range numBuffers {
		for i := range bufSize {
			n := b*bufSize + i
			v.Input(0).Channels[0][i] = float32(mod(n))
			v.Input(1).Channels[0][i] = float32(car(n))
		}
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}

	return out
}

func TestIdentityWhenModulatorEqualsCarrier(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Sibilance = 0.0
	cfg.Attack = 0.0
	cfg.Release = 0.0
	v := New(cfg)

	noise := make([]float64, 40*v.Config().BufferSize)
	for i := range noise {
		noise[i] = rand.Float64()*2 - 1
	}
	sig := func(i int) float64 { return noise[i] * 0.5 }

	out := run(v, 40, sig, sig)

	// output should be the input delayed by one frame, check after the first frames settled
	lat := v.Latency()
	maxErr := 0.0
	for i := 4 * lat; i < len(out); i++ {
		maxErr = max(maxErr, math.Abs(out[i]-sig(i-lat)))
	}

	if maxErr > 1e-3 {
		t.Fatalf("expected delayed identity, max error %v", maxErr)
	}
}

func TestSilentModulatorGivesSilence(t *testing.T) {
	v := New(DefaultConfig())

	out := run(v, 20, func(int) float64 { return 0 }, func(i int) float64 {
		return math.Sin(float64(i) * 0.05)
	})

	for i, s := range out {
		if math.Abs(s) > 1e-6 {
			t.Fatalf("sample %d not silent: %v", i, s)
		}
	}
}

func TestModulatorShapesCarrierSpectrum(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Sibilance = 0.0
	v := New(cfg)
	sr := v.Config().SampleRate

	// broadband carrier (white noise), modulator a 1 kHz sine
	out := run(v, 40,
		func(i int) float64 { return 0.5 * math.Sin(2*math.Pi*1000*float64(i)/sr) },
		func(int) float64 { return rand.Float64()*2 - 1 },
	)

	for _, s := range out {
		if math.IsNaN(s) || math.IsInf(s, 0) {
			t.Fatal("invalid sample")
		}
	}

	// measure the output spectrum of a late frame
	const n = 4096
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	start := len(out) - n
	for i := range n {
		re[i] = out[start+i] * w[i]
	}
	f.Forward(re, im)

	energy := func(lo, hi float64) float64 {
		e := 0.0
		for k := int(lo * n / sr); k <= int(hi*n/sr); k++ {
			e += re[k]*re[k] + im[k]*im[k]
		}
		return e
	}

	near := energy(800, 1250)
	far := energy(3000, 3450) + energy(200, 650)

	if near < 10*far {
		t.Fatalf("expected energy concentrated around 1 kHz, near %v far %v", near, far)
	}
}

// Without a carrier the vocoder is silent, the consonants of the voice included
func TestSilentCarrierGivesSilence(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Sibilance = 1
	v := New(cfg)
	noise := func(i int) float64 { return math.Sin(float64(i)*1.7) * math.Sin(float64(i)*0.013) } // bright, voice like
	out := run(v, 40, noise, func(int) float64 { return 0 })
	for i, x := range out {
		if math.Abs(x) > 1e-9 {
			t.Fatalf("expected silence without a carrier, got %v at %d", x, i)
		}
	}
}
