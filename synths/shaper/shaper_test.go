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

// the same notes sound the same
func TestRepeatable(t *testing.T) {
	for name, f := range Presets {
		_, a := play(f(), psalm.Message{"pitch": 64.0}, 0.4)
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
