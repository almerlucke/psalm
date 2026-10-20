package shaper

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
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
	re := s1 - s2*math.Cos(w)
	im := s2 * math.Sin(w)
	return 2 * math.Hypot(re, im) / norm
}

func peak(x []float64) float64 {
	p := 0.0
	for _, v := range x {
		p = math.Max(p, math.Abs(v))
	}
	return p
}

// plain is a held sound without envelopes moving the wave, for measuring
func plain(chains ...string) Config {
	cfg := Config{Line1: Line{Level: 1}, AmpEnv: env(1, 100, 1, 50), Level: 1}
	copy(cfg.Chains[:], chains)
	return cfg
}

func TestPresetsPlay(t *testing.T) {
	if len(Presets) < 12 {
		t.Fatalf("%d presets", len(Presets))
	}
	for name, f := range Presets {
		t.Run(name, func(t *testing.T) {
			cfg := f()
			for i, text := range cfg.Chains {
				if text == "" {
					continue
				}
				if _, err := shapechain.Parse(text); err != nil {
					t.Errorf("chain %c %q: %v", 'A'+i, text, err)
				}
			}
			if cfg.Chains[0] == "" {
				t.Error("no chain A")
			}
			v, x := play(cfg, psalm.Message{"pitch": 60.0}, 1.5)
			for i, s := range x {
				if math.IsNaN(s) || math.IsInf(s, 0) {
					t.Fatalf("sample %d is %v", i, s)
				}
			}
			p := peak(x)
			if p < 0.02 {
				t.Errorf("silent: peak %v", p)
			}
			if p > 1.2 {
				t.Errorf("too loud: peak %v", p)
			}
			if n := v.numChains; n < 1 {
				t.Errorf("%d chains", n)
			}
			// let go: it ends
			v.NoteOff()
			tail := run(v, 6)
			if v.IsActive() {
				t.Errorf("still sounding 6 s after the key was let go (peak of the tail %v)", peak(tail[len(tail)-4410:]))
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	for name, f := range Presets {
		v := New(f())
		t.Run(name, func(t *testing.T) {
			paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
		})
	}
}

// the morph goes from the first chain to the last: a sine has no third harmonic, a saw has
func TestMorph(t *testing.T) {
	f := 440.0
	third := func(morph float64) float64 {
		cfg := plain("sin", "bipolar")
		cfg.Morph = morph
		cfg.Oversample = 1
		_, x := play(cfg, psalm.Message{"frequency": f}, 0.6)
		return amplitude(x[len(x)/3:], 3*f)
	}
	a, mid, b := third(0), third(0.5), third(1)
	if a > 0.01 {
		t.Errorf("morph 0 (a sine) has a third harmonic of %v", a)
	}
	if b < 0.1 {
		t.Errorf("morph 1 (a saw) has a third harmonic of only %v", b)
	}
	if !(mid > a+0.03 && mid < b) {
		t.Errorf("morph 0.5: %v between %v and %v", mid, a, b)
	}
}

// four chains: the morph passes through the ones between
func TestMorphFourChains(t *testing.T) {
	cfg := plain("sin", "par(sin, mult 2 | sin)", "par(sin, mult 3 | sin)", "par(sin, mult 4 | sin)")
	cfg.Oversample = 1
	at := func(morph, h float64) float64 {
		cfg.Morph = morph
		_, x := play(cfg, psalm.Message{"frequency": 400.0}, 0.5)
		return amplitude(x[len(x)/3:], h*400)
	}
	for i, h := range []float64{0, 2, 3, 4} {
		morph := float64(i) / 3
		if h > 0 && at(morph, h) < 0.2 {
			t.Errorf("morph %v should have harmonic %v", morph, h)
		}
		if h > 0 && at(0, h) > 0.01 {
			t.Errorf("chain A has harmonic %v", h)
		}
	}
	// between the second and third: both harmonics, each partly
	two, three := at(0.5, 2), at(0.5, 3)
	if two < 0.05 || three < 0.05 || two > 1 || three > 1 {
		t.Errorf("halfway: %v %v", two, three)
	}
}

// a macro sets a number in the chain, and the matrix moves it
func TestMacros(t *testing.T) {
	cfg := plain("mult m1[1,4] | sin") // m1 sets the number of cycles: the pitch
	cfg.Oversample = 1
	for _, c := range []struct{ m, harmonic float64 }{{0, 1}, {1, 4}} {
		cfg.Macros[0] = c.m
		_, x := play(cfg, psalm.Message{"frequency": 300.0}, 0.5)
		if a := amplitude(x[len(x)/3:], c.harmonic*300); a < 0.4 {
			t.Errorf("m1=%v: harmonic %v at %v", c.m, c.harmonic, a)
		}
	}
	cfg.Macros[0] = 0
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "macro1", Amount: 1}
	_, x := play(cfg, psalm.Message{"frequency": 300.0}, 0.5)
	if a := amplitude(x[len(x)/3:], 4*300); a < 0.4 {
		t.Errorf("velocity 1 to m1: harmonic 4 at %v", a)
	}
}

// the morph envelope moves the wave over the note
func TestMorphEnvelope(t *testing.T) {
	cfg := plain("sin", "bipolar")
	cfg.Oversample = 1
	cfg.MorphAmount, cfg.MorphEnv = 1, env(1, 300, 0, 100)
	_, x := play(cfg, psalm.Message{"frequency": 400.0}, 1.2)
	n := int(sr() * 0.05)
	early := amplitude(x[n:2*n], 1200)
	late := amplitude(x[len(x)-2*n:len(x)-n], 1200)
	if !(early > 0.05 && late < early/5) {
		t.Errorf("third harmonic early %v, late %v: the wave should fall to a sine", early, late)
	}
}

// the matrix: the velocity pulls the morph
func TestMatrixMorph(t *testing.T) {
	cfg := plain("sin", "bipolar")
	cfg.Oversample = 1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "morph", Amount: 1}
	third := func(vel float64) float64 {
		v := New(cfg)
		v.NoteOn(vel, psalm.Message{"frequency": 440.0}, psalm.CtrlTick{}, v.Config())
		x := run(v, 0.6)
		return amplitude(x[len(x)/3:], 3*440) / vel
	}
	if soft, hard := third(0.1), third(1); !(hard > 0.1 && soft < hard/3) {
		t.Errorf("third harmonic: soft %v, hard %v", soft, hard)
	}
}

// the oversampling takes aliasing away: a saw high up, the 8th harmonic of 3 kHz folds to 20.1 kHz
func TestOversampleAliasing(t *testing.T) {
	alias := func(over int) float64 {
		cfg := plain("bipolar")
		cfg.Oversample = over
		_, x := play(cfg, psalm.Message{"frequency": 3000.0}, 0.6)
		return amplitude(x[len(x)/3:], 44100-8*3000)
	}
	off, four := alias(1), alias(4)
	if !(four < off/3) {
		t.Errorf("alias at 20.1 kHz: %v with 4x, %v without", four, off)
	}
}

// chains by message: text, a chain read already; text that can't be read keeps the one before
func TestChainMessages(t *testing.T) {
	cfg := plain("sin")
	cfg.Oversample = 1
	v := New(cfg)
	if v.numChains != 1 {
		t.Fatalf("%d chains", v.numChains)
	}
	v.ReceiveMessage(psalm.Message{"chainB": "bipolar"})
	if v.numChains != 2 || v.cfg.Chains[1] != "bipolar" {
		t.Errorf("chain B: %d chains, %q", v.numChains, v.cfg.Chains[1])
	}
	v.ReceiveMessage(psalm.Message{"chainB": "wobble 3"})
	if v.cfg.Chains[1] != "bipolar" {
		t.Errorf("a chain that can't be read replaced the one before: %q", v.cfg.Chains[1])
	}
	v.ReceiveMessage(psalm.Message{"chainA": shapechain.MustParse("tri")})
	if v.cfg.Chains[0] != "tri" {
		t.Errorf("a chain read already: %q", v.cfg.Chains[0])
	}
	v.ReceiveMessage(psalm.Message{"chainB": ""})
	if v.numChains != 1 {
		t.Errorf("an empty chain B leaves %d chains", v.numChains)
	}
	v.ReceiveMessage(psalm.Message{"chainA": "mod 0"})
	if v.cfg.Chains[0] != "tri" {
		t.Errorf("chain A: %q", v.cfg.Chains[0])
	}
	if v := New(Config{}); v.numChains != 1 || v.cfg.Chains[0] != DefaultChain {
		t.Errorf("no chains: %d, %q", v.numChains, v.cfg.Chains[0])
	}
}

// a chain that gives NaN or huge values is made safe
func TestSafe(t *testing.T) {
	cfg := plain("mult m1 | mod m1 | mult 1e9") // m1 at 0: mod by nothing
	_, x := play(cfg, psalm.Message{"frequency": 300.0}, 0.3)
	for _, s := range x {
		if math.IsNaN(s) || math.Abs(s) > 2 {
			t.Fatalf("sample %v", s)
		}
	}
}

// the same notes sound the same (the phases of a unison and the drift are random: from the seed of the environment)
func TestRepeatable(t *testing.T) {
	for name, f := range Presets {
		psalm.CurrentEnv().Seed(5)
		_, a := play(f(), psalm.Message{"pitch": 64.0}, 0.4)
		psalm.CurrentEnv().Seed(5)
		_, b := play(f(), psalm.Message{"pitch": 64.0}, 0.4)
		for i := range a {
			if a[i] != b[i] {
				t.Errorf("%s differs at %d", name, i)
				break
			}
		}
	}
}

// the filter is the one of the other synths: it takes the highs away
func TestFilter(t *testing.T) {
	cfg := plain("bipolar")
	cfg.Oversample = 1
	_, open := play(cfg, psalm.Message{"frequency": 200.0}, 0.6)
	_, shut := play(cfg, psalm.Message{"frequency": 200.0, "filterType": 1.0, "cutoff": 400.0}, 0.6)
	h := func(x []float64) float64 { return amplitude(x[len(x)/3:], 4000) }
	if !(h(shut) < h(open)/10) {
		t.Errorf("4 kHz: %v open, %v through a low pass at 400 Hz", h(open), h(shut))
	}
}

// two lines: the second is an octave up, on the right
func TestLines(t *testing.T) {
	cfg := plain("sin")
	cfg.Oversample = 1
	cfg.Width = 1
	cfg.Line2 = Line{Octave: 1, Level: 1}
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 300.0}, psalm.CtrlTick{}, v.Config())
	var l, r []float64
	for range 40 {
		v.Synthesize()
		for i := range v.Output(0).Channels[0] {
			l = append(l, float64(v.Output(0).Channels[0][i]))
			r = append(r, float64(v.Output(0).Channels[1][i]))
		}
	}
	l, r = l[len(l)/3:], r[len(r)/3:]
	if !(amplitude(l, 300) > 5*amplitude(l, 600) && amplitude(r, 600) > 5*amplitude(r, 300)) {
		t.Errorf("line 1 left: %v %v, line 2 right: %v %v", amplitude(l, 300), amplitude(l, 600), amplitude(r, 300), amplitude(r, 600))
	}
}

// the automatic oversampling reads a chain as often as it and the pitch need: a sine not at all, a
// saw four times, a fold twice at a low pitch and up to four times high up; a number in the config is kept
func TestAutoOversample(t *testing.T) {
	factor := func(cfg Config, pitch float64) int {
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())
		return v.osFactor
	}
	if f := factor(plain("sin"), 60); f != 1 {
		t.Errorf("a sine: %d", f)
	}
	if f := factor(plain("par(sin, mult 3 | sin | mult 0.4)"), 84); f != 1 {
		t.Errorf("a few partials high up: %d", f)
	}
	if f := factor(plain("bipolar"), 40); f != MaxAuto {
		t.Errorf("a saw: %d", f)
	}
	low, high := factor(plain("sin | mult 3 | mirror -1 1"), 36), factor(plain("sin | mult 3 | mirror -1 1"), 84)
	if low != 1 || high < 2 {
		t.Errorf("a fold: %d low, %d high", low, high)
	}
	fixed := plain("sin")
	fixed.Oversample = 8
	if f := factor(fixed, 60); f != 8 {
		t.Errorf("a fixed 8: %d", f)
	}
	// a chain sent to a voice playing changes it
	v := New(plain("sin"))
	v.NoteOn(1, psalm.Message{"pitch": 72.0}, psalm.CtrlTick{}, v.Config())
	v.ReceiveMessage(psalm.Message{"chainA": "bipolar"})
	if v.osFactor != MaxAuto {
		t.Errorf("a saw sent to a sine: %d", v.osFactor)
	}
}

// frequency of x by its rising zero crossings
func crossings(x []float64) float64 {
	n := 0
	for i := 1; i < len(x); i++ {
		if x[i-1] < 0 && x[i] >= 0 {
			n++
		}
	}
	return float64(n) * sr() / float64(len(x))
}

// the pitch envelope bends the pitch over the note: a kick that starts high and falls to the note
func TestPitchEnvelope(t *testing.T) {
	cfg := plain("sin")
	cfg.PitchAmount, cfg.PitchEnv = 24, env(1, 120, 0, 100)
	_, x := play(cfg, psalm.Message{"frequency": 110.0}, 0.6)
	early, late := crossings(x[int(0.004*sr()):int(0.054*sr())]), crossings(x[len(x)-int(0.2*sr()):])
	if early < 300 || math.Abs(late-110) > 6 {
		t.Errorf("early %v Hz (the note is 110, 24 st up is 440), late %v", early, late)
	}
	// off at 0
	cfg.PitchAmount = 0
	_, x = play(cfg, psalm.Message{"frequency": 110.0}, 0.4)
	if f := crossings(x[int(0.004*sr()):int(0.054*sr())]); math.Abs(f-110) > 12 {
		t.Errorf("no pitch envelope: %v Hz", f)
	}
}

// with a glide a note slides from the pitch the voice was at
func TestGlide(t *testing.T) {
	cfg := plain("sin")
	cfg.Glide = 150
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 110.0}, psalm.CtrlTick{}, v.Config())
	run(v, 0.4)
	v.NoteOn(1, psalm.Message{"frequency": 440.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 1.2)
	first, last := crossings(x[int(0.01*sr()):int(0.06*sr())]), crossings(x[len(x)-int(0.2*sr()):])
	if first > 200 || math.Abs(last-440) > 12 {
		t.Errorf("slides from %v Hz (it was at 110) to %v (the note is 440)", first, last)
	}
	// the first note of a voice does not slide from nothing
	w := New(cfg)
	w.NoteOn(1, psalm.Message{"frequency": 440.0}, psalm.CtrlTick{}, w.Config())
	y := run(w, 0.4)
	if f := crossings(y[int(0.01*sr()):int(0.11*sr())]); math.Abs(f-440) > 20 {
		t.Errorf("the first note: %v Hz", f)
	}
}

// a config can say mono and legato: one note at a time on one voice
func TestMonoConfig(t *testing.T) {
	cfg := plain("sin")
	cfg.Mono, cfg.Legato, cfg.Glide = true, true, 120
	p := NewPoly(cfg, 4)
	var out []float64
	collect := func(seconds float64) {
		for range int(seconds * sr() / float64(p.Config().BufferSize)) {
			p.ResetRun()
			p.Run()
			for _, s := range p.Output(0).Channels[0] {
				out = append(out, float64(s))
			}
		}
	}
	p.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "a", "tick": psalm.CtrlTick{}, "amplitude": 1.0, "message": psalm.Message{"frequency": 200.0}})
	collect(0.4)
	p.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "b", "tick": psalm.CtrlTick{}, "amplitude": 1.0, "message": psalm.Message{"frequency": 400.0}})
	mark := len(out)
	collect(0.8)
	first, last := crossings(out[mark+int(0.01*sr()):mark+int(0.06*sr())]), crossings(out[len(out)-int(0.2*sr()):])
	if !(first < 330 && math.Abs(last-400) < 12) {
		t.Errorf("legato glide from %v Hz to %v (200 to 400)", first, last)
	}
}

// d2 is the largest second difference of x: a smooth tone has a small one, a click (a step in the wave)
// a big one
func d2(x []float64) float64 {
	m := 0.0
	for i := 2; i < len(x); i++ {
		m = math.Max(m, math.Abs(x[i]-2*x[i-1]+x[i-2]))
	}
	return m
}

// changing a setting under a sounding note never clicks: the macros, the morph, the levels and the width
// glide, and a chain swapped for another is faded to. A script does all of these every step
func TestNoClicksOnChange(t *testing.T) {
	for _, c := range []struct {
		name string
		msg  psalm.Message
	}{
		{"macro", psalm.Message{"macro1": 1.0}},
		{"morph", psalm.Message{"morph": 1.0}},
		{"level", psalm.Message{"level": 3.0}},
		{"level of line 2", psalm.Message{"level2": 1.0}},
		{"width", psalm.Message{"width": 1.0}},
		{"chain", psalm.Message{"chainA": "mult 2 | sin"}},
		{"chains", psalm.Message{"chainA": "mult 2 | sin", "chainB": "mult 5 | sin"}},
		{"everything", psalm.Message{"macro1": 1.0, "morph": 0.7, "level": 2.0, "width": 1.0, "chainA": "par(sin, mult 3 | sin | mult 0.3)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			cfg := plain("par(sin, mult 2 | sin | mult m1[0,0.8])", "sin")
			cfg.Macros[0] = 0
			cfg.AmpEnv = env(5, 100, 1, 100)
			cfg.Line2 = Line{Detune: 7, Level: 0.3}
			cfg.Oversample = 1
			v := New(cfg)
			v.NoteOn(1, psalm.Message{"frequency": 217.3}, psalm.CtrlTick{}, v.Config()) // not a whole number of cycles by the change
			run(v, 0.3)
			before := run(v, 0.1)
			v.ReceiveMessage(c.msg)
			after := run(v, 0.1)
			// across the change too: the step is between the last sample before it and the first after
			base, got := d2(before), d2(append(append([]float64{}, before[len(before)-8:]...), after...))
			if got > 0.02 {
				t.Errorf("second difference %.4f after the change, %.4f before: a click", got, base)
			}
		})
	}
}

// a lopsided wave (a narrow pulse, mostly one side of zero) does not carry its offset in with the envelope: the
// mean of the sound stays near zero from the first cycle, where the offset used to come in with the
// attack and go out again in the release, a thump with every note
func TestNoOffsetThump(t *testing.T) {
	cfg := plain("pulse 0.1 | bipolar")
	cfg.Oversample = 1
	cfg.AmpEnv = env(4, 100, 1, 100)
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 110.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.4)
	win := int(sr() * 2 / 110) // two cycles
	worst := 0.0
	for i := 0; i+win < len(x); i += win {
		s, s2 := 0.0, 0.0
		for _, y := range x[i : i+win] {
			s += y
			s2 += y * y
		}
		if s2 > 0 {
			worst = math.Max(worst, math.Abs(s/float64(win))/math.Sqrt(s2/float64(win)))
		}
	}
	if worst > 0.15 {
		t.Errorf("the mean of the sound is %.0f%% of its rms in some window: an offset", worst*100)
	}
	v.NoteOff()
	x = run(v, 0.4)
	worst = 0
	for i := 0; i+win < len(x); i += win {
		s, s2 := 0.0, 0.0
		for _, y := range x[i : i+win] {
			s += y
			s2 += y * y
		}
		if s2/float64(win) > 1e-4 { // louder than -40 dB: below it the offset is of the blocker settling, far below hearing
			worst = math.Max(worst, math.Abs(s/float64(win))/math.Sqrt(s2/float64(win)))
		}
	}
	if worst > 0.15 {
		t.Errorf("in the release the mean is %.0f%% of the rms", worst*100)
	}
}

// the message "preset" makes a voice a preset: its chains, envelopes, filter and matrix; later keys of the message
// change it, a name that is not a preset changes nothing
func TestPresetMessage(t *testing.T) {
	v := New(Config{})
	v.ReceiveMessage(psalm.Message{"preset": "kickSub"})
	want := KickSub()
	if v.cfg.Chains != want.Chains || v.cfg.PitchAmount != want.PitchAmount || v.cfg.AmpEnv != want.AmpEnv || v.cfg.Level != want.Level {
		t.Errorf("the voice is not the preset: %+v", v.cfg.Chains)
	}
	v.ReceiveMessage(psalm.Message{"preset": "acid", "cutoff": 900.0})
	acid := Acid()
	if v.cfg.Chains != acid.Chains || v.cfg.Glide != acid.Glide || v.cfg.Filter.Resonance != acid.Filter.Resonance || v.cfg.Filter.Cutoff != 900 {
		t.Errorf("acid with its cutoff changed: %v %v %+v", v.cfg.Chains, v.cfg.Glide, v.cfg.Filter)
	}
	if v.cfg.Mods[0].Dest != "cutoff" {
		t.Errorf("the matrix: %+v", v.cfg.Mods[0])
	}
	v.ReceiveMessage(psalm.Message{"preset": "nothing"})
	if v.cfg.Chains != acid.Chains {
		t.Error("a name that is not a preset changed the voice")
	}
	// every preset can be made of any voice, and plays
	for name := range Presets {
		w := New(Config{})
		w.NoteOn(1, psalm.Message{"preset": name, "pitch": 48.0}, psalm.CtrlTick{}, w.Config())
		x := run(w, 0.5)
		if peak(x) < 0.01 || math.IsNaN(peak(x)) {
			t.Errorf("%s as a message: peak %v", name, peak(x))
		}
	}
	// a sounding voice made another preset does not click: not past what the new sound has of its own (a fold has
	// corners). The voice has no filter; the preset has one, which fades in
	w := New(plain("par(sin, mult 2 | sin | mult 0.3)"))
	w.NoteOn(1, psalm.Message{"frequency": 217.3}, psalm.CtrlTick{}, w.Config())
	run(w, 0.3)
	before := run(w, 0.1)
	w.ReceiveMessage(psalm.Message{"preset": "foldPad"})
	after := run(w, 0.1)
	run(w, 1)
	steady := d2(run(w, 0.1))
	got := d2(append(append([]float64{}, before[len(before)-8:]...), after...))
	if got > 3*steady+0.01 {
		t.Errorf("second difference %.4f after a preset was loaded under a note, %.4f once settled: a click", got, steady)
	}
}

// a note that starts on a voice that is still sounding, in the middle of a buffer (a mono synth's next note), goes on
// from the sound: the old note sounds until the new one starts, and the wave, the envelope and the settings carry on
func TestRetriggerOnSoundingVoice(t *testing.T) {
	cfg := plain("sin")
	cfg.Oversample = 1
	cfg.AmpEnv = env(5, 100, 1, 300)
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 217.3}, psalm.CtrlTick{}, v.Config())
	run(v, 0.3)
	before := run(v, 0.05)
	v.NoteOn(1, psalm.Message{"frequency": 311.1}, psalm.CtrlTick{SampleOffset: 100}, v.Config())
	after := run(v, 0.2)
	// the first 100 samples are the old note, not a gap
	gap := 0
	for _, y := range after[:100] {
		if y == 0 {
			gap++
		}
	}
	if gap > 5 {
		t.Errorf("%d silent samples before the new note: the old one was cut at the start of the buffer", gap)
	}
	if got := d2(append(append([]float64{}, before[len(before)-8:]...), after...)); got > 0.01 {
		t.Errorf("second difference %.4f across the new note: a click", got)
	}
	// a voice that was silent starts its note at the offset as it always did
	w := New(cfg)
	w.NoteOn(1, psalm.Message{"frequency": 217.3}, psalm.CtrlTick{SampleOffset: 100}, w.Config())
	x := run(w, 0.05)
	for _, y := range x[:100] {
		if y != 0 {
			t.Fatal("a fresh note sounds before its offset")
		}
	}
}

// a preset without a filter loaded on a sounding voice that has one leaves the filter on, open: it does not step
func TestPresetWithoutFilterUnderNote(t *testing.T) {
	v := New(Config{})
	v.NoteOn(1, psalm.Message{"preset": "warmPad", "frequency": 217.3}, psalm.CtrlTick{}, v.Config())
	run(v, 0.5)
	before := run(v, 0.05)
	v.ReceiveMessage(psalm.Message{"preset": "choirPad"}) // no filter of its own
	after := run(v, 0.2)
	steady := d2(run(v, 0.2))
	got := d2(append(append([]float64{}, before[len(before)-8:]...), after...))
	if got > 4*steady+0.02 {
		t.Errorf("second difference %.4f when the filter went, %.4f once settled: a click", got, steady)
	}
	if !v.filter.On() {
		t.Error("the filter was switched off under the note")
	}
}

// a preset swapped under a sounding note never ticks, whether a filter comes in (the saw of the new sound must not come
// through before the filter is there) or goes (the saw of the old one must not come through the opening one)
func TestPresetSwapsDoNotTick(t *testing.T) {
	maxd2 := func(x []float64) float64 {
		m := 0.0
		for i := 2; i < len(x); i++ {
			m = math.Max(m, math.Abs(x[i]-2*x[i-1]+x[i-2]))
		}
		return m
	}
	for _, p := range [][2]string{{"choirPad", "warmPad"}, {"warmPad", "choirPad"}, {"evolve", "choirPad"}, {"warmPad", "evolve"}, {"airPad", "stringPad"}} {
		v := New(Config{})
		v.NoteOn(1, psalm.Message{"preset": p[0], "frequency": 98.0}, psalm.CtrlTick{}, v.Config())
		run(v, 1)
		before := run(v, 0.1)
		v.ReceiveMessage(psalm.Message{"preset": p[1]})
		after := run(v, 0.5)
		x := append(append([]float64{}, before[len(before)-8:]...), after...)
		early, steady := maxd2(x[:int(0.03*sr())]), maxd2(x[int(0.2*sr()):])
		if early > 4*math.Max(steady, maxd2(before))+0.003 {
			t.Errorf("%s to %s: second difference %.4f in the first 30 ms, %.4f in the sound once settled, %.4f before: a tick", p[0], p[1], early, steady, maxd2(before))
		}
	}
}

// a bass note does not start with a click: the attack of FMBass takes the onset down to the level of the note
func TestFMBassOnset(t *testing.T) {
	v := New(FMBass())
	v.NoteOn(1, psalm.Message{"pitch": 38.0, "velocity": 0.95}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.5)
	on, body := 0.0, 0.0
	for i := 2; i < len(x); i++ {
		d := math.Abs(x[i] - 2*x[i-1] + x[i-2])
		if i < int(0.02*sr()) {
			on = math.Max(on, d)
		} else if i > int(0.1*sr()) {
			body = math.Max(body, d)
		}
	}
	if on > 4*body {
		t.Errorf("the onset of FMBass has a second difference of %.4f, %.1f times the body of the note", on, on/body)
	}
}
