package convolver

import (
	"math"
	"math/rand/v2"
	"testing"
)

// run feeds left/right input through the convolver and returns the output channels
func run(c *Convolver, left []float64, right []float64) ([]float64, []float64) {
	bufSize := c.Config().BufferSize
	var outL, outR []float64

	for start := 0; start+bufSize <= len(left); start += bufSize {
		for i := range bufSize {
			c.Input(0).Channels[0][i] = float32(left[start+i])
			c.Input(0).Channels[1][i] = float32(right[start+i])
		}
		c.Synthesize()
		for i := range bufSize {
			outL = append(outL, float64(c.Output(0).Channels[0][i]))
			outR = append(outR, float64(c.Output(0).Channels[1][i]))
		}
	}

	return outL, outR
}

func directConvolution(x []float64, h []float64) []float64 {
	y := make([]float64, len(x))
	for n := range y {
		for k := 0; k < len(h) && k <= n; k++ {
			y[n] += h[k] * x[n-k]
		}
	}
	return y
}

func randomSignal(n int, seed uint64) []float64 {
	rng := rand.New(rand.NewPCG(seed, 0))
	x := make([]float64, n)
	for i := range x {
		x[i] = rng.Float64()*2 - 1
	}
	return x
}

func TestMatchesDirectConvolution(t *testing.T) {
	// an impulse response spanning several partitions, with a partial last partition
	ir := randomSignal(1000, 1)
	for i := range ir {
		ir[i] *= 0.05
	}

	c := New([][]float64{ir}, Config{PartitionSize: 128, Mix: 1.0, Gain: 1.0})

	x := randomSignal(8*c.Config().BufferSize, 2)
	outL, outR := run(c, x, x)
	want := directConvolution(x, ir)

	lat := c.Latency()
	maxErr := 0.0
	for n := lat; n < len(outL); n++ {
		maxErr = math.Max(maxErr, math.Abs(outL[n]-want[n-lat]))
		maxErr = math.Max(maxErr, math.Abs(outR[n]-want[n-lat]))
	}

	if maxErr > 1e-4 {
		t.Fatalf("max error vs direct convolution %v", maxErr)
	}
}

func TestStereoImpulseResponseMapsChannels(t *testing.T) {
	// left IR delays by 10 samples, right IR inverts and delays by 20
	left := make([]float64, 11)
	left[10] = 1.0
	right := make([]float64, 21)
	right[20] = -1.0

	c := New([][]float64{left, right}, Config{PartitionSize: 64, Mix: 1.0, Gain: 1.0})

	x := randomSignal(4*c.Config().BufferSize, 3)
	y := randomSignal(len(x), 4)
	outL, outR := run(c, x, y)

	lat := c.Latency()
	for n := lat + 20; n < len(outL); n++ {
		if math.Abs(outL[n]-x[n-lat-10]) > 1e-5 {
			t.Fatalf("left mismatch at %d", n)
		}
		if math.Abs(outR[n]+y[n-lat-20]) > 1e-5 {
			t.Fatalf("right mismatch at %d", n)
		}
	}
}

func TestDryMix(t *testing.T) {
	c := New(SyntheticReverbIR(0.5, 0.5, 44100, 1), Config{PartitionSize: 256, Mix: 0.0, Gain: 1.0})

	x := randomSignal(2*c.Config().BufferSize, 5)
	outL, _ := run(c, x, x)

	for n := range outL {
		if math.Abs(outL[n]-x[n]) > 1e-6 {
			t.Fatalf("dry signal changed at %d", n)
		}
	}
}

func TestSyntheticReverbIRHasUnitEnergy(t *testing.T) {
	ir := SyntheticReverbIR(1.0, 0.7, 44100, 42)
	if len(ir) != 2 || len(ir[0]) != 44100 {
		t.Fatalf("unexpected shape %d x %d", len(ir), len(ir[0]))
	}

	for ch, buf := range ir {
		energy := 0.0
		for _, v := range buf {
			energy += v * v
		}
		if math.Abs(energy-1.0) > 1e-9 {
			t.Fatalf("channel %d energy %v", ch, energy)
		}
	}

	// channels are decorrelated for stereo width
	if ir[0][1000] == ir[1][1000] {
		t.Fatal("expected different noise per channel")
	}
}
