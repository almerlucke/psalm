package organ

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

// render runs the organ for seconds and returns the left channel
func render(o *Organ, seconds float64) []float64 {
	var x []float64
	for range int(math.Ceil(seconds * sr() / float64(o.Config().BufferSize))) {
		o.Synthesize()
		for _, s := range o.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func down(o *Organ, id string, pitch float64) {
	o.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOn": id, "amplitude": 1.0,
		"message": psalm.Message{"pitch": pitch}})
}

func up(o *Organ, id string) {
	o.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOff": id})
}

// amplitude returns the amplitude of a frequency in x (Goertzel, Hann window)
func amplitude(x []float64, freq float64) float64 {
	w := 2 * math.Pi * freq / sr()
	var s1, s2, norm float64
	for i, v := range x {
		h := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(x)-1))
		s := v*h + 2*math.Cos(w)*s1 - s2
		s2, s1 = s1, s
		norm += h
	}
	re := s1 - s2*math.Cos(w)
	im := s2 * math.Sin(w)
	return 2 * math.Hypot(re, im) / norm
}

func db(a float64) float64 { return 20 * math.Log10(a) }

func pitchHz(p float64) float64 { return 440 * math.Exp2((p-69)/12) }

func clean(bars string) Config {
	cfg := DefaultConfig()
	cfg.Drawbars, cfg.Click = Bars(bars), 0
	return cfg
}

func TestDrawbarsPlayTheirFootages(t *testing.T) {
	o := New(clean("888000000"))
	down(o, "a", 57)
	x := render(o, 1)[int(0.2*sr()):]
	f := pitchHz(57)
	for _, c := range []struct {
		freq float64
		on   bool
	}{{f / 2, true}, {pitchHz(64), true}, {f, true}, {2 * f, false}, {pitchHz(76), false}, {4 * f, false}} {
		a := amplitude(x, c.freq)
		if c.on && a < 0.01 || !c.on && a > 1e-4 {
			t.Errorf("%.1f Hz: amplitude %g, want it on: %v", c.freq, a, c.on)
		}
	}
	// a drawbar step is 3 dB
	o = New(clean("008000000"))
	down(o, "a", 57)
	full := amplitude(render(o, 1)[int(0.2*sr()):], f)
	o = New(clean("005000000"))
	down(o, "a", 57)
	three := amplitude(render(o, 1)[int(0.2*sr()):], f)
	if d := db(full) - db(three); math.Abs(d-9) > 0.1 {
		t.Errorf("3 steps down is %.2f dB, want 9", d)
	}
}

func TestKeysShareTheWheels(t *testing.T) {
	// the 4' of A3 and the 8' of A4 are the same wheel: they add up in phase, 6 dB
	o := New(clean("008800000"))
	down(o, "a", 57)
	one := amplitude(render(o, 1)[int(0.2*sr()):], 440)
	o = New(clean("008800000"))
	down(o, "a", 57)
	down(o, "b", 69)
	both := amplitude(render(o, 1)[int(0.2*sr()):], 440)
	if d := db(both) - db(one); math.Abs(d-6.02) > 0.1 {
		t.Errorf("a shared wheel is %.2f dB louder, want 6", d)
	}
	// the wheels fold back at the ends: a 1' of the top key plays an octave down
	if wheelOf(120) != wheelOf(108) || wheelOf(12) != wheelOf(24) {
		t.Errorf("the wheels don't fold back")
	}
}

func TestPercussionSingleTrigger(t *testing.T) {
	cfg := clean("888000000")
	cfg.Percussion, cfg.PercFast = PercThird, true
	o := New(cfg)
	third := pitchHz(57 + 19)
	down(o, "a", 57)
	x := render(o, 0.5)
	early := amplitude(x[:int(0.05*sr())], third)
	late := amplitude(x[int(0.4*sr()):], third)
	if early < 0.01 || late > early*0.05 {
		t.Errorf("the percussion: %g at the start, %g after 0.4 s", early, late)
	}
	// a key played while another is down has no percussion
	down(o, "b", 64)
	x = render(o, 0.1)
	if a := amplitude(x, pitchHz(64+19)); a > early*0.1 {
		t.Errorf("legato: the second key has percussion %g", a)
	}
	// after all keys are up it comes again
	up(o, "a")
	up(o, "b")
	render(o, 0.1)
	down(o, "c", 57)
	if a := amplitude(render(o, 0.05), third); a < early*0.8 {
		t.Errorf("a new phrase: percussion %g, want %g", a, early)
	}
	// it takes the 1' away
	cfg = clean("000000008")
	cfg.Percussion = PercSecond
	o = New(cfg)
	down(o, "a", 57)
	if a := amplitude(render(o, 1)[int(0.5*sr()):], 8*pitchHz(57)); a > 1e-4 {
		t.Errorf("the 1' with the percussion on: %g", a)
	}
}

// spread returns how much the frequency moves: the shortest and longest period, relative
func spread(x []float64) float64 {
	var crossings []float64
	for i := 1; i < len(x); i++ {
		if x[i-1] < 0 && x[i] >= 0 {
			crossings = append(crossings, float64(i-1)+x[i-1]/(x[i-1]-x[i]))
		}
	}
	lo, hi := math.Inf(1), 0.0
	for i := 1; i < len(crossings); i++ {
		p := crossings[i] - crossings[i-1]
		lo, hi = math.Min(lo, p), math.Max(hi, p)
	}
	return hi/lo - 1
}

func TestVibrato(t *testing.T) {
	for _, c := range []struct {
		v        Vibrato
		min, max float64
	}{{VibratoOff, 0, 0.001}, {V1, 0.005, 0.03}, {V3, 0.03, 0.08}} {
		cfg := clean("008000000")
		cfg.Vibrato = c.v
		o := New(cfg)
		down(o, "a", 69)
		if s := spread(render(o, 1)[int(0.2*sr()):]); s < c.min || s > c.max {
			t.Errorf("vibrato %d: the period moves %.4f, want %g-%g", c.v, s, c.min, c.max)
		}
	}
}

func TestNotesAndPedal(t *testing.T) {
	o := New(clean("008000000"))
	// a note of a duration, starting on its sample
	o.Note(0.1, 1, psalm.Message{"pitch": 69.0}, psalm.CtrlTick{SampleOffset: 50}, o.Config())
	o.Synthesize()
	if v := o.Output(0).Channels[0]; v[49] != 0 || v[51] == 0 {
		t.Errorf("the note starts at %v, want sample 50", v[48:52])
	}
	render(o, 0.2)
	if len(o.keys) != 0 {
		t.Errorf("a note of 0.1 s still holds its key")
	}
	// the same key twice: it lets go when both notes have
	o.Note(0.1, 1, psalm.Message{"pitch": 69.0}, psalm.CtrlTick{}, o.Config())
	o.Note(0.3, 1, psalm.Message{"pitch": 69.0}, psalm.CtrlTick{}, o.Config())
	render(o, 0.2)
	if len(o.keys) != 1 {
		t.Errorf("the key let go with a note still holding it")
	}
	render(o, 0.2)
	// the sustain pedal holds the keys let go
	o.ReceiveMessage(psalm.Message{"sustain": 1.0})
	down(o, "a", 60)
	up(o, "a")
	render(o, 0.2)
	if len(o.keys) != 1 {
		t.Errorf("the pedal doesn't hold the key")
	}
	o.ReceiveMessage(psalm.Message{"sustain": 0.0})
	render(o, 0.1)
	if len(o.keys) != 0 {
		t.Errorf("the pedal up doesn't let go")
	}
	down(o, "b", 60)
	o.AllNotesOff()
	render(o, 0.1)
	if len(o.keys) != 0 || peak(render(o, 0.1)) != 0 {
		t.Errorf("all notes off still sounds")
	}
}

func peak(x []float64) float64 {
	p := 0.0
	for _, v := range x {
		p = math.Max(p, math.Abs(v))
	}
	return p
}

func TestClick(t *testing.T) {
	onset := func(click float64) float64 {
		cfg := clean("008000000")
		cfg.Click = click
		o := New(cfg)
		down(o, "a", 45)
		x := render(o, 0.02)[:256]
		// high frequency energy: the differences of the samples
		e := 0.0
		for i := 1; i < len(x); i++ {
			e += (x[i] - x[i-1]) * (x[i] - x[i-1])
		}
		return e
	}
	if c, n := onset(1), onset(0); c < 10*n {
		t.Errorf("the click adds %g to %g", c, n)
	}
}

func TestSettings(t *testing.T) {
	o := New(DefaultConfig())
	o.ReceiveMessage(psalm.Message{"drawbars": "123456780", "percussion": 2, "vibrato": 6, "drive": 0.5})
	o.Receive(4.0, 9)
	if r := o.cfg.Registration(); r != "123456784" || o.cfg.Percussion != PercThird || o.cfg.Vibrato != C3 || o.cfg.Drive != 0.5 {
		t.Errorf("settings: %s %+v", r, o.cfg)
	}
	// every preset plays a chord (the levels are set for one note: synths/levels)
	for name, f := range Presets {
		o := New(f())
		for i, p := range []float64{57, 60, 64, 67} {
			down(o, string(rune('a'+i)), p)
		}
		x := render(o, 0.5)
		if p := peak(x); p < 0.05 || p > 4 || math.IsNaN(p) {
			t.Errorf("%s: a chord peaks at %g", name, p)
		}
	}
}
