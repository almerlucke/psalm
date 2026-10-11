package pd

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

func run(v *Voice, seconds float64) []float64 {
	var x []float64
	for range int(math.Ceil(seconds * sr() / float64(v.Config().BufferSize))) {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func play(cfg Config, msg psalm.Message, seconds float64) (*Voice, []float64) {
	v := New(cfg)
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	return v, run(v, seconds)
}

// amplitude of a frequency in x (Goertzel, Hann window)
func amplitude(x []float64, freq float64) float64 {
	w := 2 * math.Pi * freq / sr()
	var s1, s2, norm float64
	for i, v := range x {
		h := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(x)-1))
		s := v*h + 2*math.Cos(w)*s1 - s2
		s2, s1 = s1, s
		norm += h
	}
	return 2 * math.Hypot(s1-s2*math.Cos(w), s2*math.Sin(w)) / norm
}

// steady is one line holding a waveform at a fixed distortion
func steady(w Waveform, dcw float64) Config {
	hold := env(1, 10, 1, 10)
	return Config{Line1: Line{Waveform: w, Level: 1, DCW: dcw, DCWEnv: hold, DCAEnv: hold}, Level: 1}
}

func harmonics(cfg Config, f float64, n int) []float64 {
	_, x := play(cfg, psalm.Message{"frequency": f, "velocity": 0.5}, 0.4)
	y := x[int(0.1*sr()):]
	h := make([]float64, n+1)
	for k := 1; k <= n; k++ {
		h[k] = amplitude(y, float64(k)*f) / amplitude(y, f)
	}
	return h
}

func TestUndistortedIsASine(t *testing.T) {
	for _, w := range []Waveform{Saw, Square, Pulse, SawPulse} {
		h := harmonics(steady(w, 0), 220, 5)
		for k := 2; k <= 5; k++ {
			if h[k] > 0.01 {
				t.Errorf("waveform %d undistorted has harmonic %d at %.3f", w, k, h[k])
			}
		}
	}
}

func TestShapes(t *testing.T) {
	// a distorted saw has all harmonics, a square mostly the odd ones
	saw := harmonics(steady(Saw, 0.95), 110, 6)
	if saw[2] < 0.2 || saw[3] < 0.1 {
		t.Errorf("the saw's harmonics: %.3f", saw[1:])
	}
	sq := harmonics(steady(Square, 0.95), 110, 6)
	if sq[3] < 0.15 || sq[2] > sq[3]*0.2 || sq[4] > sq[3]*0.2 {
		t.Errorf("the square's harmonics: %.3f", sq[1:])
	}
	// a resonance waveform peaks at the distortion's pitch: 1 + 15 d times the note
	res := harmonics(steady(Reso1, 0.6), 110, 12)
	best := 2
	for k := 2; k <= 12; k++ {
		if res[k] > res[best] {
			best = k
		}
	}
	if best < 9 || best > 11 {
		t.Errorf("the resonance peaks at harmonic %d, want 10: %.2f", best, res[1:])
	}
}

func TestNoAliasing(t *testing.T) {
	// at 3 kHz, fully distorted: what lies between the harmonics is far below them
	f := 3000.0
	_, x := play(steady(Saw, 1), psalm.Message{"frequency": f}, 0.4)
	y := x[int(0.1*sr()):]
	fund := amplitude(y, f)
	for _, between := range []float64{1.5, 2.5, 3.5, 4.5, 5.5} {
		if a := amplitude(y, between*f); a > fund*0.01 {
			t.Errorf("at %.0f Hz (between harmonics): %.1f dB", between*f, 20*math.Log10(a/fund))
		}
	}
}

func TestDCWEnvelope(t *testing.T) {
	// the bass's distortion falls after the attack: its second harmonic too
	_, x := play(Bass(), psalm.Message{"pitch": 45.0}, 1)
	f := 110.0
	early := x[int(0.01*sr()):int(0.06*sr())]
	late := x[int(0.6*sr()):int(0.65*sr())]
	ratio := func(y []float64) float64 { return amplitude(y, 4*f) / amplitude(y, f) }
	if ratio(early) < 2*ratio(late) {
		t.Errorf("the 4th harmonic: %.3f at the start, %.3f later", ratio(early), ratio(late))
	}
	// key follow: the same distortion is gentler high up
	cfg := steady(Saw, 0.8)
	cfg.Line1.KeyFollow = 1
	low, high := harmonics(cfg, 131, 2), harmonics(cfg, 1046, 2)
	if high[2] > low[2]*0.8 {
		t.Errorf("key follow: the 2nd harmonic %.3f low, %.3f high", low[2], high[2])
	}
}

func TestRing(t *testing.T) {
	hold := env(1, 10, 1, 10)
	cfg := Config{
		Line1: Line{Waveform: Saw, Level: 1, DCWEnv: hold, DCAEnv: hold},
		Line2: Line{Waveform: Saw, Level: 1, Detune: 702 + 1200, DCWEnv: hold, DCAEnv: hold},
		Ring:  true, Level: 1,
	}
	_, x := play(cfg, psalm.Message{"frequency": 200.0}, 0.4)
	y := x[int(0.1*sr()):]
	// two sines, 200 and 600 Hz (an octave and a fifth): ring modulated, 400 and 800, and 200 from line 1
	if a, b := amplitude(y, 400), amplitude(y, 600); a < 5*b {
		t.Errorf("ring modulation: the difference %.3f, line 2 itself %.3f", a, b)
	}
}

func TestNotes(t *testing.T) {
	v := New(Brass())
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{SampleOffset: 25}, v.Config())
	v.Synthesize()
	if out := v.Output(0).Channels[0]; out[24] != 0 {
		t.Errorf("it plays before its sample")
	}
	v.Clear()
	v.Note(0.2, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, 1.5)
	if v.IsActive() {
		t.Errorf("a note of 0.2 s plays after 1.5 s")
	}
	for name, f := range Presets {
		_, x := play(f(), psalm.Message{"pitch": 57.0}, 1)
		p := 0.0
		for _, s := range x {
			p = math.Max(p, math.Abs(s))
		}
		if p < 0.01 || p > 4 || math.IsNaN(p) {
			t.Errorf("%s peaks at %g", name, p)
		}
	}
}
