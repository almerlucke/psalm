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

// A normalized impulse response keeps about the loudness of the input, whatever its own level
func TestNormalizeIR(t *testing.T) {
	quiet := SyntheticReverbIR(1, 0.5, 44100, 3)
	for _, ch := range quiet {
		for i := range ch {
			ch[i] *= 0.01 // a recording 40 dB down
		}
	}
	norm := NormalizeIR(quiet)
	energy := 0.0
	for _, ch := range norm {
		for _, x := range ch {
			energy += x * x
		}
	}
	if math.Abs(energy/float64(len(norm))-1) > 1e-9 {
		t.Fatalf("expected unit energy per channel, got %v", energy/float64(len(norm)))
	}
}

// A long response is convolved in segments of growing blocks: still exactly a convolution, with the
// latency of the first block only
func TestLongResponseMatchesDirectConvolution(t *testing.T) {
	ir := randomSignal(40000, 3)
	for i := range ir {
		ir[i] *= 0.01 * math.Exp(-float64(i)/8000)
	}
	c := New([][]float64{ir}, Config{PartitionSize: 64, Mix: 1.0, Gain: 1.0})
	if len(c.segments) < 5 {
		t.Fatalf("expected the response in segments of growing blocks, %d", len(c.segments))
	}

	x := randomSignal(60*c.Config().BufferSize, 4)
	outL, _ := run(c, x, x)
	want := directConvolution(x, ir[:c.irLength])
	lat := c.Latency()
	maxErr := 0.0
	for n := lat; n < len(outL); n++ {
		maxErr = math.Max(maxErr, math.Abs(outL[n]-want[n-lat]))
	}
	if maxErr > 1e-4 {
		t.Fatalf("max error vs direct convolution %v", maxErr)
	}
}

// The inaudible end of a response is left out
func TestTrimsTheSilentEnd(t *testing.T) {
	ir := make([]float64, 20000)
	for i := range 5000 {
		ir[i] = math.Exp(-float64(i) / 500) // gone below -90 dB long before the end
	}
	c := New([][]float64{ir}, Config{PartitionSize: 128, Mix: 1.0, Gain: 1.0})
	if c.irLength >= 12000 || c.irLength < 4000 {
		t.Fatalf("expected the silent end trimmed, %d of 20000 samples kept", c.irLength)
	}
}

// With the same input on both sides the input is transformed once: the same output
func TestMonoInputSharesTheTransform(t *testing.T) {
	ir := [][]float64{randomSignal(20000, 5), randomSignal(20000, 6)}
	for _, ch := range ir {
		for i := range ch {
			ch[i] *= 0.01
		}
	}
	x := randomSignal(30*1024, 7)
	a := New(ir, Config{PartitionSize: 128, Mix: 1, Gain: 1})
	b := New(ir, Config{PartitionSize: 128, Mix: 1, Gain: 1, MonoInput: true})
	al, ar := run(a, x, x)
	bl, br := run(b, x, x)
	for i := range al {
		if math.Abs(al[i]-bl[i]) > 1e-6 || math.Abs(ar[i]-br[i]) > 1e-6 {
			t.Fatalf("sample %d differs: %v %v, %v %v", i, al[i], ar[i], bl[i], br[i])
		}
	}
}
