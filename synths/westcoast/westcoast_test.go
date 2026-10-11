package westcoast

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

// open is a held note through a gate wide open: the oscillator and the folder as they are
func open() Config {
	return Config{Sustain: true, Decay: 1, Gate: VCA, Level: 1}
}

func harmonics(cfg Config, f float64, n int) []float64 {
	_, x := play(cfg, psalm.Message{"frequency": f, "velocity": 1.0}, 0.5)
	y := x[int(0.2*sr()):]
	h := make([]float64, n+1)
	for k := 1; k <= n; k++ {
		h[k] = amplitude(y, float64(k)*f) / amplitude(y, f)
	}
	return h
}

func TestFolder(t *testing.T) {
	plain := harmonics(open(), 220, 5)
	for k := 2; k <= 5; k++ {
		if plain[k] > 0.005 {
			t.Errorf("unfolded, harmonic %d is %.4f", k, plain[k])
		}
	}
	cfg := open()
	cfg.Fold = 0.5
	folded := harmonics(cfg, 220, 5)
	if folded[3] < 0.1 || folded[2] > 0.01 {
		t.Errorf("folded: the odd harmonics %.3f, the even %.3f", folded[3], folded[2])
	}
	cfg.Symmetry = 0.5
	if h := harmonics(cfg, 220, 2); h[2] < 0.05 {
		t.Errorf("folded off center: the second harmonic %.3f", h[2])
	}
}

func TestFM(t *testing.T) {
	cfg := open()
	cfg.ModRatio, cfg.FM = 1.5, 1
	_, x := play(cfg, psalm.Message{"frequency": 200.0, "velocity": 1.0}, 0.5)
	y := x[int(0.2*sr()):]
	// sidebands at 200 ± 300 Hz: 500 (and 100)
	if a, c := amplitude(y, 500), amplitude(y, 200); a < 0.2*c {
		t.Errorf("FM: the sideband %.3f against the carrier %.3f", a, c)
	}
	if a := amplitude(y, 350); a > 0.01*amplitude(y, 200) {
		t.Errorf("FM: energy between the sidebands %.4f", a)
	}
}

func TestLowPassGate(t *testing.T) {
	// struck, the vactrol lights at once and dies slowly; it gets darker as it dies
	cfg := Pluck()
	cfg.EnvToFold, cfg.Fold = 0, 0.5
	_, x := play(cfg, psalm.Message{"frequency": 220.0, "velocity": 1.0}, 0.6)
	peakAt, peak := 0, 0.0
	for i, s := range x {
		if math.Abs(s) > peak {
			peakAt, peak = i, math.Abs(s)
		}
	}
	if float64(peakAt)/sr() > 0.03 {
		t.Errorf("the gate peaks after %.0f ms", float64(peakAt)/sr()*1000)
	}
	early, late := x[int(0.02*sr()):int(0.07*sr())], x[int(0.25*sr()):int(0.3*sr())]
	bright := func(y []float64) float64 { return amplitude(y, 5*220) / amplitude(y, 220) }
	if bright(late) > bright(early)*0.5 {
		t.Errorf("the gate closing: the 5th harmonic %.3f early, %.3f late", bright(early), bright(late))
	}
	if amplitude(late, 220) > amplitude(early, 220)*0.7 {
		t.Errorf("the gate doesn't close")
	}
}

func TestBongo(t *testing.T) {
	v, x := play(Bongo(), psalm.Message{"pitch": 57.0}, 1.5)
	if v.IsActive() {
		t.Errorf("the bongo still sounds after 1.5 s")
	}
	p := 0.0
	for _, s := range x[:int(0.05*sr())] {
		p = math.Max(p, math.Abs(s))
	}
	if p < 0.05 {
		t.Errorf("the bongo peaks at %g", p)
	}
	if a, b := amplitude(x[:int(0.1*sr())], 220), amplitude(x[:int(0.1*sr())], 330); a < 5*b {
		t.Errorf("the bongo rings at %g at its pitch, %g beside it", a, b)
	}
}

func TestNotes(t *testing.T) {
	v := New(Bass())
	v.NoteOn(1, psalm.Message{"pitch": 40.0}, psalm.CtrlTick{SampleOffset: 15}, v.Config())
	v.Synthesize()
	if out := v.Output(0).Channels[0]; out[14] != 0 {
		t.Errorf("it plays before its sample")
	}
	v.Clear()
	v.Note(0.2, 1, psalm.Message{"pitch": 40.0}, psalm.CtrlTick{}, v.Config())
	run(v, 3)
	if v.IsActive() {
		t.Errorf("a held note of 0.2 s plays after 3 s")
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
