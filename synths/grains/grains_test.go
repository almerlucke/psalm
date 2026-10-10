package grains

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

var _ poly.Voice = (*Voice)(nil)

const sr = 44100.0

// testSample is an in memory mono sample implementing sndfile.SoundFiler
type testSample struct {
	buf []float32
	out []float32
}

func sine(freq float64, seconds float64) *testSample {
	buf := make([]float32, int(seconds*sr))
	for i := range buf {
		buf[i] = float32(math.Sin(2 * math.Pi * freq * float64(i) / sr))
	}
	return &testSample{buf: buf, out: make([]float32, 1)}
}

func (s *testSample) NumChannels() int                        { return 1 }
func (s *testSample) SampleRate() float64                     { return sr }
func (s *testSample) NumFrames() int64                        { return int64(len(s.buf)) }
func (s *testSample) Duration() float64                       { return float64(len(s.buf)) / sr }
func (s *testSample) Depth() int                              { return 1 }
func (s *testSample) Buffer(int, int) []float32               { return s.buf }
func (s *testSample) ZeroCrossings(int) sndfile.ZeroCrossings { return nil }
func (s *testSample) LookupAll(pos float64, _ int, wrap bool) []float32 {
	s.out[0] = s.Lookup(pos, 0, 0, wrap)
	return s.out
}
func (s *testSample) Lookup(pos float64, _ int, _ int, wrap bool) float32 {
	return sndfile.NewLookupParam[float32](pos, int64(len(s.buf)), wrap).Lookup(s.buf)
}

// render holds a note and returns both channels after the attack
func render(cfg Config, msg psalm.Message) ([]float64, []float64) {
	v := New(cfg)
	v.NoteOn(0.5, msg, psalm.CtrlTick{}, v.Config())
	var left, right []float64
	for b := range 30 {
		v.Synthesize()
		if b >= 6 {
			for i := range v.Output(0).Channels[0] {
				left = append(left, float64(v.Output(0).Channels[0][i]))
				right = append(right, float64(v.Output(0).Channels[1][i]))
			}
		}
	}
	return left, right
}

// renderLong holds a note and returns the left channel of the given number of buffers
func renderLong(cfg Config, msg psalm.Message, buffers int) []float64 {
	v := New(cfg)
	v.NoteOn(0.5, msg, psalm.CtrlTick{}, v.Config())
	var left []float64
	for b := range buffers {
		v.Synthesize()
		if b >= 6 {
			for _, s := range v.Output(0).Channels[0] {
				left = append(left, float64(s))
			}
		}
	}
	return left
}

func base() Config {
	cfg := DefaultConfig()
	cfg.Sample = sine(440, 2.0)
	cfg.RootNote = 69
	cfg.StereoSpread = 0
	return cfg
}

// peakFrequency returns the strongest frequency of the power spectrum averaged over overlapping
// windows (Welch): every grain has a random phase, so a single snapshot is noisy
func peakFrequency(x []float64) float64 {
	const n = 16384
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	power := make([]float64, n/2)

	for start := 0; start+n <= len(x); start += n / 2 {
		for i := range n {
			re[i] = x[start+i] * w[i]
			im[i] = 0
		}
		f.Forward(re, im)
		for k := range power {
			power[k] += re[k]*re[k] + im[k]*im[k]
		}
	}

	best := 1
	for k := 1; k < n/2; k++ {
		if power[k] > power[best] {
			best = k
		}
	}
	return float64(best) * sr / n
}

func rms(x []float64) float64 {
	sum := 0.0
	for _, v := range x {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(x)))
}

func TestPitchFollowsTheNote(t *testing.T) {
	binHz := sr / 16384
	for _, c := range []struct{ note, want float64 }{{69, 440}, {81, 880}, {57, 220}} {
		// longer grains give a cleaner tone per grain, a longer render more windows to average
		cfg := base()
		cfg.Scan = 1
		cfg.GrainSize = 300
		cfg.Density = 12
		left := renderLong(cfg, psalm.Message{"pitch": c.note}, 120)
		if f := peakFrequency(left); math.Abs(f-c.want) > 2*binHz {
			t.Fatalf("note %v: expected %v Hz, got %v", c.note, c.want, f)
		}
	}
}

func TestWindowIsSmooth(t *testing.T) {
	for _, shape := range []float64{0, 0.5, 1} {
		if window(0, shape) != 0 || math.Abs(window(1, shape)) > 1e-9 {
			t.Fatalf("shape %v: the window must start and end at zero", shape)
		}
		prev := 0.0
		for i := 1; i <= 10000; i++ {
			w := window(float64(i)/10000, shape)
			if math.Abs(w-prev) > 0.05 {
				t.Fatalf("shape %v: the window jumps at %v (%v to %v)", shape, float64(i)/10000, prev, w)
			}
			prev = w
		}
	}
}

func TestLevelStaysStableWithDensity(t *testing.T) {
	sparse := base()
	sparse.Density = 10
	sparse.GrainSize = 100
	dense := sparse
	dense.Density = 80

	ls, _ := render(sparse, psalm.Message{"pitch": 69.0})
	ld, _ := render(dense, psalm.Message{"pitch": 69.0})
	if ratio := rms(ld) / rms(ls); ratio < 0.6 || ratio > 1.7 {
		t.Fatalf("expected a similar level at 8 times the density, ratio %v", ratio)
	}
}

func TestStereoSpread(t *testing.T) {
	mono := base()
	l, r := render(mono, psalm.Message{"pitch": 69.0})
	for i := range l {
		if l[i] != r[i] {
			t.Fatal("without stereo spread the channels should be equal")
		}
	}

	wide := base()
	wide.StereoSpread = 1
	l, r = render(wide, psalm.Message{"pitch": 69.0})
	diff := make([]float64, len(l))
	for i := range l {
		diff[i] = l[i] - r[i]
	}
	if rms(diff) < rms(l)*0.1 {
		t.Fatal("with stereo spread the channels should differ")
	}
}

func TestNoteLifecycle(t *testing.T) {
	v := New(base())
	v.Note(0.3, 0.5, psalm.Message{"pitch": 69.0}, psalm.CtrlTick{}, v.Config())
	for blocks := 0; v.IsActive(); blocks++ {
		if blocks > 200 {
			t.Fatal("timed note never ended")
		}
		v.Synthesize()
	}

	v.NoteOn(0.5, psalm.Message{"pitch": 69.0}, psalm.CtrlTick{}, v.Config())
	for range 100 {
		v.Synthesize()
	}
	if !v.IsActive() {
		t.Fatal("a held note should sustain")
	}
	v.NoteOff()
	for blocks := 0; v.IsActive(); blocks++ {
		if blocks > 200 {
			t.Fatal("released note never ended")
		}
		v.Synthesize()
	}
}

func TestSoundBank(t *testing.T) {
	cfg := base()
	cfg.SoundBank = sndfile.SoundBank[float32]{"low": sine(220, 2.0), "high": sine(880, 2.0)}
	cfg.Scan = 1

	left, _ := render(cfg, psalm.Message{"pitch": 69.0, "sound": "high"})
	if f := peakFrequency(left); math.Abs(f-880) > 6 {
		t.Fatalf("expected the high sample at 880 Hz, got %v", f)
	}
}

func TestExtremeSettingsAreBounded(t *testing.T) {
	cfg := base()
	cfg.Density = 500
	cfg.GrainSize = 500
	cfg.PitchSpread = 12
	cfg.Reverse = 1
	cfg.Shape = 1
	cfg.Spray = 1
	cfg.Scan = -3
	cfg.StereoSpread = 1

	l, r := render(cfg, psalm.Message{"pitch": 100.0})
	for i := range l {
		if math.IsNaN(l[i]) || math.IsNaN(r[i]) || math.Abs(l[i]) > 2 || math.Abs(r[i]) > 2 {
			t.Fatalf("invalid sample %v %v", l[i], r[i])
		}
	}
	if rms(l) < 0.01 {
		t.Fatal("expected sound")
	}
}

// bandEnergy returns the averaged spectral energy around freq
func bandEnergy(x []float64, freq float64) float64 {
	const n = 16384
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
		for k := c - 20; k <= c+20; k++ {
			e += re[k]*re[k] + im[k]*im[k]
		}
	}
	return e
}

func TestOctaveGrains(t *testing.T) {
	cfg := base()
	cfg.Scan = 1
	plain := renderLong(cfg, psalm.Message{"pitch": 69.0}, 80)

	cfg.Octave = 0.5
	sparkle := renderLong(cfg, psalm.Message{"pitch": 69.0}, 80)

	ratio := func(x []float64) float64 { return bandEnergy(x, 880) / bandEnergy(x, 440) }
	if ratio(sparkle) < 20*ratio(plain) || ratio(sparkle) < 0.3 {
		t.Fatalf("expected octave grains at 880 Hz: ratio %v vs %v", ratio(sparkle), ratio(plain))
	}
}

func TestPresets(t *testing.T) {
	for name, preset := range Presets {
		t.Run(name, func(t *testing.T) {
			// a seeded environment of its own: the grains are random, the test the same every run
			psalm.Seed(1)
			psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
			v := New(preset(sine(440, 3.0), 69))
			v.Note(1.0, 0.5, psalm.Message{"pitch": 69.0, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())

			peak := 0.0
			for blocks := 0; v.IsActive(); blocks++ {
				if blocks > 1000 {
					t.Fatal("voice never finished")
				}
				v.Synthesize()
				for _, ch := range v.Output(0).Channels {
					for _, s := range ch {
						if math.IsNaN(float64(s)) {
							t.Fatal("NaN output")
						}
						peak = math.Max(peak, math.Abs(float64(s)))
					}
				}
			}
			// a full scale sine is hotter than the recordings the levels are calibrated on (see
			// synths/levels): the pluck, calibrated to peak at -1 dB, peaks up to 1.8 on it
			if peak < 0.02 || peak > 2 {
				t.Fatalf("unexpected peak %v", peak)
			}
			t.Logf("peak %.3f", peak)
		})
	}
}
