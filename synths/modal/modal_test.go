package modal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

func hz(p float64) float64 { return 440 * math.Exp2((p-69)/12) }

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

func strike(cfg Config, pitch, seconds float64) (*Voice, []float64) {
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())
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

func window(x []float64, from, length float64) []float64 {
	return x[int(from*sr()) : int(from*sr())+int(length*sr())]
}

func db(a float64) float64 { return 20 * math.Log10(a) }

func TestModesRingAtTheirRatios(t *testing.T) {
	cfg := Marimba()
	cfg.Noise, cfg.Hardness, cfg.Position = 0, 1, 0.3
	_, x := strike(cfg, 57, 0.5)
	y := window(x, 0.005, 0.1)
	f := hz(57)
	ref := amplitude(y, f)
	for _, r := range []float64{4, 10} {
		if a := amplitude(y, r*f); a < ref*0.003 {
			t.Errorf("the mode at %g is %.1f dB down", r, db(a/ref))
		}
	}
	for _, r := range []float64{2, 3, 5} {
		if a := amplitude(y, r*f); a > ref*0.001 {
			t.Errorf("a marimba sounds at %g: %.1f dB", r, db(a/ref))
		}
	}
}

func TestDecay(t *testing.T) {
	one := Config{Modes: []Mode{{1, 1, 1}}, Decay: 1.5, Hardness: 0.5, Level: 1}
	_, x := strike(one, 69, 1)
	// 60 dB in 1.5 s: 20 dB in 0.5 s
	if d := db(amplitude(window(x, 0.1, 0.1), 440) / amplitude(window(x, 0.6, 0.1), 440)); math.Abs(d-20) > 0.5 {
		t.Errorf("in 0.5 s it falls %.1f dB, want 20", d)
	}
	// the higher modes die sooner
	two := Config{Modes: []Mode{{1, 1, 1}, {4, 1, 1}}, Decay: 1.5, HighDamping: 1, Hardness: 1, Level: 1}
	_, x = strike(two, 57, 1)
	fall := func(f float64) float64 {
		return db(amplitude(window(x, 0.05, 0.1), f) / amplitude(window(x, 0.35, 0.1), f))
	}
	if low, high := fall(hz(57)), fall(4*hz(57)); math.Abs(high/low-4) > 0.2 {
		t.Errorf("the mode 2 octaves up falls %.1f dB, the lowest %.1f: want 4 times", high, low)
	}
}

func TestMallet(t *testing.T) {
	high := func(hardness float64) float64 {
		cfg := Glockenspiel()
		cfg.Hardness, cfg.Noise = hardness, 0
		_, x := strike(cfg, 72, 0.3)
		y := window(x, 0.01, 0.1)
		return amplitude(y, 5.404*hz(72)) / amplitude(y, hz(72))
	}
	if soft, hard := high(0), high(1); db(hard/soft) < 10 {
		t.Errorf("a hard mallet wakes the high mode %.1f dB more than a soft one, want 10", db(hard/soft))
	}
	// struck in the middle, the second mode has a node there
	second := func(pos float64) float64 {
		cfg := Config{Modes: []Mode{{1, 1, 1}, {2.756, 1, 1}}, Decay: 1, Hardness: 1, Position: pos, Level: 1}
		_, x := strike(cfg, 60, 0.3)
		y := window(x, 0.01, 0.1)
		return amplitude(y, 2.756*hz(60)) / amplitude(y, hz(60))
	}
	if mid, side := second(0.5), second(0.25); mid > side*0.3 {
		t.Errorf("the second mode struck in the middle %.3f, at a quarter %.3f", mid, side)
	}
}

func TestReleaseAndTremolo(t *testing.T) {
	v, _ := strike(Vibraphone(), 69, 0.2)
	v.NoteOff()
	run(v, 0.5)
	if v.IsActive() {
		t.Errorf("the damper doesn't stop a vibraphone")
	}
	v, _ = strike(Glockenspiel(), 81, 0.2) // no damper
	v.NoteOff()
	if x := run(v, 0.3); amplitude(x[len(x)-4096:], hz(81)) < 0.05 {
		t.Errorf("a glockenspiel stops when let go")
	}
	// the motor: the loudness swings 5 times a second
	cfg := Vibraphone()
	cfg.Decay, cfg.HighDamping = 100, 0 // hardly decaying, to see the tremolo
	cfg.Noise = 0
	_, x := strike(cfg, 69, 1.2)
	env := func(at float64) float64 { return amplitude(window(x, at, 0.02), 440) }
	lo, hi := math.Inf(1), 0.0
	for at := 0.2; at < 1.1; at += 0.01 {
		lo, hi = min(lo, env(at)), max(hi, env(at))
	}
	// the fans swing the resonator tube, the bar rings on: a smaller swing than the whole sound's
	if d := db(hi / lo); d < 2 || d > 8 {
		t.Errorf("the tremolo swings %.1f dB, want about 3 (depth 0.6 on the tube)", d)
	}
	cfg.Resonator = 0
	_, x = strike(cfg, 69, 1.2)
	lo, hi = math.Inf(1), 0.0
	for at := 0.2; at < 1.1; at += 0.01 {
		lo, hi = min(lo, env(at)), max(hi, env(at))
	}
	if d := db(hi / lo); d < 5 || d > 9 {
		t.Errorf("without a tube the tremolo swings %.1f dB, want 8 (depth 0.6)", d)
	}
}

func TestBeating(t *testing.T) {
	cfg := Config{Modes: []Mode{{1, 1, 1}}, Decay: 100, Hardness: 0.5, Beat: 0.01, Level: 1}
	_, x := strike(cfg, 69, 1.2) // two partials 4.4 Hz apart: the loudness beats 4.4 times a second
	env := func(at float64) float64 { return amplitude(window(x, at, 0.02), 440) }
	lo, hi := math.Inf(1), 0.0
	for at := 0.05; at < 1.1; at += 0.005 {
		lo, hi = min(lo, env(at)), max(hi, env(at))
	}
	if hi < 10*lo {
		t.Errorf("a split mode beats from %.3f to %.3f", lo, hi)
	}
}

func TestNotes(t *testing.T) {
	v := New(Marimba())
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{SampleOffset: 30}, v.Config())
	v.Synthesize()
	out := v.Output(0).Channels[0]
	if out[29] != 0 || out[31] == 0 {
		t.Errorf("the strike starts at %v, want sample 30", out[28:33])
	}
	v.Clear()
	v.Note(0.1, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, 5)
	if v.IsActive() {
		t.Errorf("a marimba note still plays after 5 s")
	}
	for name, f := range Presets {
		_, x := strike(f(), 69, 0.5)
		p := 0.0
		for _, s := range x {
			p = max(p, math.Abs(s))
		}
		if p < 0.01 || p > 4 || math.IsNaN(p) {
			t.Errorf("%s peaks at %g", name, p)
		}
	}
}
