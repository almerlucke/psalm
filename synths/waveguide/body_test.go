package waveguide

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/almerlucke/psalm"
)

func TestGuitarIR(t *testing.T) {
	ir := GuitarIR()
	if len(ir) != 1 || len(ir[0]) != 17640 {
		t.Fatalf("channels %d, samples %d: want one of 17640", len(ir), len(ir[0]))
	}
	e := 0.0
	for _, x := range ir[0] {
		e += x * x
	}
	if math.Abs(e-1) > 1e-9 {
		t.Errorf("energy %g, want 1", e)
	}
}

func TestDecodeWavRefuses(t *testing.T) {
	for _, b := range [][]byte{nil, []byte("RIFF"), []byte("not a wave file at all")} {
		if _, _, err := decodeWav(b); err == nil {
			t.Errorf("%q: no error", b)
		}
	}
}

func TestResample(t *testing.T) {
	ir := GuitarIR()
	for _, to := range []float64{48000, 88200, 22050} {
		out := resample(ir, 44100, to)
		if want := float64(len(ir[0])) * to / 44100; math.Abs(float64(len(out[0]))-want) > 1 {
			t.Errorf("%g: %d samples, want %g", to, len(out[0]), want)
		}
		// the same sound: about the energy per second
		e0, e1 := 0.0, 0.0
		for _, x := range ir[0] {
			e0 += x * x
		}
		for _, x := range out[0] {
			e1 += x * x
		}
		if r := e1 * 44100 / (e0 * to); r < 0.8 || r > 1.25 {
			t.Errorf("%g: energy x%.2f", to, r)
		}
	}
}

// feed puts noise through a stage in blocks of the buffer size and returns what came out
func feed(s *bodyStage, blocks int, seed uint64) (in, out []float64) {
	size := psalm.CurrentConfig().BufferSize
	r := rand.New(rand.NewPCG(seed, 1))
	for range blocks {
		ch := psalm.NewChannels(psalm.Stereo, size)
		for i := range size {
			x := float32(r.Float64()*2 - 1)
			ch[0][i], ch[1][i] = x, x
			in = append(in, float64(x))
		}
		s.process(ch)
		for _, x := range ch[0] {
			out = append(out, float64(x))
		}
	}
	return
}

// the blend is the resonators, held back by the convolution's delay, and the measured body, in their shares
func TestBodyStageBlend(t *testing.T) {
	m := &Measured{IR: GuitarIR(), SampleRate: 44100, Blend: 0.5, Trim: -1.5}
	solo := *m
	solo.Blend = 1
	_, both := feed(newBodyStage(GuitarBody, 0.8, m, 2), 20, 1)
	_, wet := feed(newBodyStage(GuitarBody, 0.8, &solo, 2), 20, 1)
	_, dry := feed(newBodyStage(GuitarBody, 0.8, nil, 2), 20, 1)

	peak := 0
	for i, x := range GuitarIR()[0] {
		if math.Abs(x) > math.Abs(GuitarIR()[0][peak]) {
			peak = i
		}
	}
	d := bodyBlock + peak
	db := func(x float64) float64 { return math.Pow(10, x/20) }
	dryGain, wetGain := db(blendTrim)*0.5, db(blendTrim+m.Trim)*0.5
	worst := 0.0
	for i := d; i < len(both); i++ {
		want := dryGain*dry[i-d] + wetGain*wet[i]
		worst = max(worst, math.Abs(both[i]-want))
	}
	if worst > 1e-4 {
		t.Errorf("the blend is off by %g from its parts", worst)
	}
	if rms(wet) < 0.01 {
		t.Error("the measured body is silent")
	}
}

// the resonators alone are what a voice has: the stage with no measured body is not a stage of NewPoly
func TestPolyStage(t *testing.T) {
	play := func(cfg Config) []float64 {
		sameNoise()
		p := NewPoly(cfg, 4)
		monoOn(p, "a", 52)
		return polyRun(p, 1)
	}
	own := OwnGuitar()
	if own.Measured == nil {
		t.Fatal("OwnGuitar has no measured body")
	}
	with := play(own)
	plain := own
	plain.Measured = nil
	without := play(plain)
	if rms(with) < 0.001 || rms(without) < 0.001 {
		t.Fatalf("silent: %g with, %g without", rms(with), rms(without))
	}
	// the two bodies sound alike in loudness (within 9 dB: one low note, which the measured body takes more of than) and not the same
	if r := 20 * math.Log10(rms(with)/rms(without)); math.Abs(r) > 9 {
		t.Errorf("the measured body is %+.1f dB against the resonators", r)
	}
	same := true
	for i := range with {
		if math.Abs(with[i]-without[i]) > 1e-4 {
			same = false
			break
		}
	}
	if same {
		t.Error("the measured body changed nothing")
	}
	for _, x := range with {
		if math.IsNaN(x) || math.Abs(x) > 1.5 {
			t.Fatalf("bad sample %g", x)
		}
	}
}

func TestBodyEffect(t *testing.T) {
	e := NewBodyEffect(GuitarBody, 0.8, OwnGuitarBody())
	size := psalm.CurrentConfig().BufferSize
	for i := range size {
		e.Input(0).Channels[0][i], e.Input(0).Channels[1][i] = 0.1, 0.1
	}
	e.Synthesize()
	if e.Tail() <= 0 {
		t.Error("no tail")
	}
}
