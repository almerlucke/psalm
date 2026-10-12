package supersaw

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

func run(v *Voice, seconds float64) ([]float64, []float64) {
	var l, r []float64
	for range int(math.Ceil(seconds * sr() / float64(v.Config().BufferSize))) {
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			l = append(l, float64(ch[0][i]))
			r = append(r, float64(ch[len(ch)-1][i]))
		}
	}
	return l, r
}

func play(cfg Config, msg psalm.Message, seconds float64) (*Voice, []float64, []float64) {
	v := New(cfg)
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	l, r := run(v, seconds)
	return v, l, r
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

func open() Config {
	return Config{Cutoff: 20000, FilterEnv: env(1, 10, 1, 10), AmpEnv: env(1, 10, 1, 10), Level: 1}
}

func TestCurves(t *testing.T) {
	// Szabo's curves: no detune at 0, about 1 at full; the mix crosses over
	if d := detuneCurve(0); math.Abs(d) > 0.01 {
		t.Errorf("the detune at 0: %g", d)
	}
	if d := detuneCurve(1); math.Abs(d-1) > 0.05 {
		t.Errorf("the detune at 1: %g", d)
	}
	prev := -1.0
	for d := 0.0; d <= 1; d += 0.05 {
		if c := detuneCurve(d); c < prev-1e-3 {
			t.Errorf("the detune curve falls at %g", d)
		} else {
			prev = c
		}
	}
	if c, s := mixLevels(0); c < 0.9 || s > 0.1 {
		t.Errorf("mix 0: center %g side %g", c, s)
	}
}

func TestDetuneSpreads(t *testing.T) {
	// with no detune all saws are one pitch: the fundamental is all there; detuned, the energy
	// spreads around it
	at := func(detune float64) (on, beside float64) {
		cfg := open()
		cfg.Detune, cfg.Mix = detune, 0.8
		_, l, r := play(cfg, psalm.Message{"frequency": 220.0}, 1)
		x := make([]float64, len(l))
		for i := range l {
			x[i] = l[i] + r[i]
		}
		y := x[int(0.2*sr()):]
		return amplitude(y, 220), amplitude(y, 220*1.02)
	}
	on0, beside0 := at(0)
	on1, beside1 := at(0.7)
	if beside0 > on0*0.01 {
		t.Errorf("no detune: %.3f beside the note, %.3f on it", beside0, on0)
	}
	if beside1/on1 < 10*beside0/on0 || beside1 < on1*0.1 {
		t.Errorf("detuned: %.3f beside the note, %.3f on it", beside1, on1)
	}
}

func TestStereo(t *testing.T) {
	cfg := open()
	cfg.Detune, cfg.Mix, cfg.Stereo = 0.5, 0.7, 1
	_, l, r := play(cfg, psalm.Message{"frequency": 220.0}, 1)
	var ll, rr, lr float64
	for i := int(0.2 * sr()); i < len(l); i++ {
		ll += l[i] * l[i]
		rr += r[i] * r[i]
		lr += l[i] * r[i]
	}
	if c := lr / math.Sqrt(ll*rr); c > 0.7 {
		t.Errorf("spread wide, the channels correlate %.2f", c)
	}
	if math.Abs(10*math.Log10(ll/rr)) > 1.5 {
		t.Errorf("spread wide, the channels differ %.1f dB", 10*math.Log10(ll/rr))
	}
	cfg.Stereo = 0
	_, l, r = play(cfg, psalm.Message{"frequency": 220.0}, 0.3)
	for i := range l {
		if math.Abs(l[i]-r[i]) > 1e-6 {
			t.Fatalf("not spread, the channels differ")
		}
	}
}

func TestFilterAndHighPass(t *testing.T) {
	cfg := open()
	cfg.Cutoff = 400
	_, l, _ := play(cfg, psalm.Message{"frequency": 110.0}, 0.5)
	y := l[int(0.2*sr()):]
	if h := amplitude(y, 110*20) / amplitude(y, 110); h > 0.005 { // unfiltered 0.05
		t.Errorf("filtered at 400 Hz: the 20th harmonic %.4f", h)
	}
	// the high pass takes out the sub
	cfg = open()
	cfg.Sub = 1
	_, l, _ = play(cfg, psalm.Message{"frequency": 110.0}, 0.5)
	plain := amplitude(l[int(0.2*sr()):], 55)
	cfg.HighPass = true
	_, l, _ = play(cfg, psalm.Message{"frequency": 110.0}, 0.5)
	if hp := amplitude(l[int(0.2*sr()):], 55); hp > plain*0.4 {
		t.Errorf("the high pass leaves the sub at %.3f of %.3f", hp, plain)
	}
}

func TestNotes(t *testing.T) {
	v := New(Lead())
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{SampleOffset: 33}, v.Config())
	v.Synthesize()
	if out := v.Output(0).Channels[0]; out[32] != 0 {
		t.Errorf("it plays before its sample")
	}
	v.Clear()
	v.Note(0.2, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, 1)
	if v.IsActive() {
		t.Errorf("a note of 0.2 s plays after 1 s")
	}
	for name, f := range Presets {
		_, l, _ := play(f(), psalm.Message{"pitch": 57.0}, 1)
		p := 0.0
		for _, s := range l {
			p = math.Max(p, math.Abs(s))
		}
		if p < 0.01 || p > 4 || math.IsNaN(p) {
			t.Errorf("%s peaks at %g", name, p)
		}
	}
}

func TestMonoGlides(t *testing.T) {
	// a poly of four voices played mono legato: the second note slides from the first (seeded: the
	// saws start at random phases)
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	p := NewPoly(GlideLead(), 4).SetMono(true, true)
	trig := func(id string, pitch float64) {
		p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOn": id, "amplitude": 1.0,
			"message": psalm.Message{"pitch": pitch}})
	}
	render := func(seconds float64) []float64 {
		var x []float64
		for range int(seconds * sr() / float64(p.Config().BufferSize)) {
			p.Synthesize()
			for _, s := range p.Output(0).Channels[0] {
				x = append(x, float64(s))
			}
		}
		return x
	}
	trig("a", 57)
	render(0.5)
	trig("b", 69)
	x := render(1)
	// the glide takes about 0.25 s: a quarter second in it is between the notes, after a second on the new one
	mid := strongest(x[int(0.2*sr()):int(0.3*sr())])
	late := strongest(x[int(0.8*sr()):])
	if mid < 250 || mid > 420 {
		t.Errorf("a quarter second into the glide at %.0f Hz, want between 220 and 440", mid)
	}
	if math.Abs(late-440) > 12 { // the detuned saws spread the energy a little around it
		t.Errorf("after the glide at %.0f Hz, want 440", late)
	}
	active := 0
	p.CallVoices(func(v *Voice) {
		if v.IsActive() {
			active++
		}
	})
	if active != 1 {
		t.Errorf("mono: %d voices sound", active)
	}
}

// strongest returns the frequency between 150 and 500 Hz with the most energy
func strongest(x []float64) float64 {
	best, bestF := 0.0, 0.0
	for f := 150.0; f <= 500; f += 2 {
		if a := amplitude(x, f); a > best {
			best, bestF = a, f
		}
	}
	return bestF
}
