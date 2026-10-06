package fdnreverb

import (
	"math"
	"testing"
)

// impulseResponse feeds an impulse into the left input and returns the stereo wet output
func impulseResponse(r *FDNReverb, seconds float64) ([]float64, []float64) {
	bufSize := r.Config().BufferSize
	numBuffers := int(seconds * r.Config().SampleRate / float64(bufSize))
	var left, right []float64

	for b := range numBuffers {
		for i := range bufSize {
			v := float32(0)
			if b == 0 && i == 0 {
				v = 1
			}
			r.Input(0).Channels[0][i] = v
			r.Input(0).Channels[1][i] = v
		}
		r.Synthesize()
		for i := range bufSize {
			left = append(left, float64(r.Output(0).Channels[0][i]))
			right = append(right, float64(r.Output(0).Channels[1][i]))
		}
	}

	return left, right
}

// measureRT60 estimates the reverb time from the decay rate of the tail: it fits a line to the
// level (dB) of 50 ms windows between 0.3 and 1.0 times the expected decay, after the echo density
// has built up, and extrapolates to -60 dB
func measureRT60(ir []float64, sr float64, expected float64) float64 {
	win := int(0.05 * sr)
	var xs, ys []float64
	for start := int(0.3 * expected * sr); start+win < int(expected*sr); start += win {
		e := 0.0
		for _, v := range ir[start : start+win] {
			e += v * v
		}
		xs = append(xs, float64(start)/sr)
		ys = append(ys, 10*math.Log10(e/float64(win)))
	}

	var mx, my float64
	for i := range xs {
		mx += xs[i]
		my += ys[i]
	}
	mx /= float64(len(xs))
	my /= float64(len(ys))

	var num, den float64
	for i := range xs {
		num += (xs[i] - mx) * (ys[i] - my)
		den += (xs[i] - mx) * (xs[i] - mx)
	}

	return -60.0 / (num / den)
}

func TestDecayTimeMatchesRT60(t *testing.T) {
	for _, mod := range []float64{0.0, 0.4, 2.0} {
		for _, decay := range []float64{1.0, 2.5} {
			cfg := DefaultConfig()
			cfg.Decay = decay
			cfg.Damping = 0.0
			cfg.ModDepth = mod
			cfg.Mix = 1.0
			cfg.PreDelay = 0.0
			r := New(cfg)

			left, _ := impulseResponse(r, decay*1.2)
			rt60 := measureRT60(left, r.Config().SampleRate, decay)

			if math.Abs(rt60-decay)/decay > 0.05 {
				t.Fatalf("decay %v s, modulation %v ms: measured RT60 %v s", decay, mod, rt60)
			}
		}
	}
}

func TestAbsorptionFilterGains(t *testing.T) {
	var f absorption
	f.set(0.9, 0.5)

	dc := f.b / (1.0 - f.a)
	nyquist := f.b / (1.0 + f.a)

	if math.Abs(dc-0.9) > 1e-12 || math.Abs(nyquist-0.5) > 1e-12 {
		t.Fatalf("expected gains 0.9 and 0.5, got %v and %v", dc, nyquist)
	}
}

func TestDampingDarkensTheTail(t *testing.T) {
	// high frequency energy (sample differences) relative to the total drops over time when damped
	brightness := func(ir []float64, from, to int) float64 {
		hf, total := 0.0, 0.0
		for i := from + 1; i < to; i++ {
			d := ir[i] - ir[i-1]
			hf += d * d
			total += ir[i] * ir[i]
		}
		return hf / total
	}

	cfg := DefaultConfig()
	cfg.Mix = 1.0
	cfg.Damping = 0.9
	r := New(cfg)
	ir, _ := impulseResponse(r, 2.0)
	sr := int(r.Config().SampleRate)

	early := brightness(ir, sr/10, sr/5)
	late := brightness(ir, sr, sr+sr/10)
	if late > early*0.5 {
		t.Fatalf("expected the tail to get darker: early %v late %v", early, late)
	}
}

func TestLongDecayIsStable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Decay = 60.0
	cfg.Damping = 0.0
	cfg.Mix = 1.0
	r := New(cfg)

	left, right := impulseResponse(r, 5.0)
	for i := range left {
		if math.IsNaN(left[i]) || math.Abs(left[i]) > 1.0 || math.Abs(right[i]) > 1.0 {
			t.Fatalf("unstable output at %d: %v %v", i, left[i], right[i])
		}
	}
}

func TestStereoIsDecorrelated(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mix = 1.0
	r := New(cfg)

	left, right := impulseResponse(r, 1.0)
	var lr, ll, rr float64
	for i := range left {
		lr += left[i] * right[i]
		ll += left[i] * left[i]
		rr += right[i] * right[i]
	}
	if corr := lr / math.Sqrt(ll*rr); math.Abs(corr) > 0.5 {
		t.Fatalf("expected decorrelated channels, correlation %v", corr)
	}
}

func TestDryMixPassesInput(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mix = 0.0
	r := New(cfg)

	bufSize := r.Config().BufferSize
	for i := range bufSize {
		r.Input(0).Channels[0][i] = float32(math.Sin(float64(i) * 0.1))
		r.Input(0).Channels[1][i] = float32(math.Cos(float64(i) * 0.1))
	}
	r.Synthesize()

	for i := range bufSize {
		if r.Output(0).Channels[0][i] != r.Input(0).Channels[0][i] || r.Output(0).Channels[1][i] != r.Input(0).Channels[1][i] {
			t.Fatalf("dry signal changed at %d", i)
		}
	}
}

// Jumping the mix glides: no steps in the output
func TestMixGlides(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mix = 0
	r := New(cfg)
	bs := r.Config().BufferSize
	n := 0
	play := func(buffers int) []float64 {
		var out []float64
		for range buffers {
			for i := range bs {
				v := float32(0.5 * math.Sin(2*math.Pi*110*float64(n)/r.Config().SampleRate))
				r.Input(0).Channels[0][i], r.Input(0).Channels[1][i] = v, v
				n++
			}
			r.Synthesize()
			for i := range bs {
				out = append(out, float64(r.Output(0).Channels[0][i]))
			}
		}
		return out
	}
	play(40)
	r.ReceiveMessage(map[string]any{"mix": 1.0})
	out := play(10)
	for i := 1; i < len(out); i++ {
		if d := math.Abs(out[i] - out[i-1]); d > 0.03 {
			t.Fatalf("the output stepped by %v at %d", d, i)
		}
	}
}
