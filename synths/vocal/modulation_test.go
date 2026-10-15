package vocal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func rms(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

func sung(cfg Config, msg psalm.Message, seconds float64) ([]float64, []float64) {
	psalm.Seed(1)
	_, l, r := sing(cfg, msg, seconds)
	return l, r
}

func TestFilterDarkens(t *testing.T) {
	cfg := steady(Solo())
	cfg.Range = Tenor
	open, _ := sung(cfg, psalm.Message{"pitch": 48.0}, 0.6)
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 300}
	dark, _ := sung(cfg, psalm.Message{"pitch": 48.0}, 0.6)
	o, d := open[int(0.3*sr()):], dark[int(0.3*sr()):]
	if harmonicsIn(d, hz(48), 1500, 4000) > 0.05*harmonicsIn(o, hz(48), 1500, 4000) {
		t.Errorf("the filter leaves the top")
	}
}

func TestMatrixMovesDestinations(t *testing.T) {
	// the vowel: from a to i with Ext 1
	cfg := steady(Solo())
	cfg.Range, cfg.VowelGlide = Tenor, 0
	cfg.Ext[0] = 1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "vowel", Amount: 1} // two steps: i
	x, _ := sung(cfg, psalm.Message{"pitch": 48.0}, 0.6)
	y := x[int(0.3*sr()):]
	f0 := hz(48)
	if low, mid := harmonicsIn(y, f0, 200, 400), harmonicsIn(y, f0, 550, 800); low < mid {
		t.Errorf("the matrix doesn't move the vowel to i: %.2g around 300 Hz, %.2g around 650", low, mid)
	}
	// level
	cfg = steady(Solo())
	cfg.Ext[0] = 1
	a, _ := sung(cfg, psalm.Message{"pitch": 55.0}, 0.4)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: 1}
	b, _ := sung(cfg, psalm.Message{"pitch": 55.0}, 0.4)
	if rms(b) < 1.5*rms(a) {
		t.Errorf("level: %g vs %g", rms(b), rms(a))
	}
	// pitch
	cfg = steady(Solo())
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "pitch", Amount: 1}
	x, _ = sung(cfg, psalm.Message{"pitch": 48.0, "modWheel": 1.0}, 0.6)
	y = x[int(0.3*sr()):]
	if amplitude(y, hz(60)) < 5*amplitude(y, hz(48)) {
		t.Errorf("the wheel doesn't raise the pitch an octave")
	}
	// spread: a solo voice with the spread moved is no longer in the middle
	cfg = Choir()
	cfg.Spread = 0
	l, r := sung(cfg, psalm.Message{"pitch": 57.0}, 0.5)
	same := true
	for i := range l {
		if l[i] != r[i] {
			same = false
		}
	}
	cfg.Ext[0] = 1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "spread", Amount: 1}
	l2, r2 := sung(cfg, psalm.Message{"pitch": 57.0}, 0.5)
	if !same || rms(l2[2000:]) == rms(r2[2000:]) {
		t.Errorf("spread: same %v, left %g right %g", same, rms(l2[2000:]), rms(r2[2000:]))
	}
}

func TestConsonants(t *testing.T) {
	cfg := steady(Solo())
	cfg.Attack = 0.2 // the vowel hasn't come in during the burst
	none, _ := sung(cfg, psalm.Message{"pitch": 55.0}, 0.15)
	n := int(0.04 * sr())
	for c := T; c < numConsonants; c++ {
		cfg.Consonant = c
		cfg.ConsonantLevel = 1
		x, _ := sung(cfg, psalm.Message{"pitch": 55.0}, 0.15)
		if rms(x[:n]) < 3*rms(none[:n]) {
			t.Errorf("%s: the burst %g against %g", ConsonantNames[c], rms(x[:n]), rms(none[:n]))
		}
		// it is over in a fraction of a second
		if tail := rms(x[int(0.14*sr()):]); tail > 0.5*rms(x[:n]) {
			t.Errorf("%s: still going", ConsonantNames[c])
		}
	}
	// sh is lower than s: less of its energy changes fast (the difference of neighbouring samples)
	bright := func(c Consonant) float64 {
		cfg.Consonant = c
		x, _ := sung(cfg, psalm.Message{"pitch": 55.0}, 0.1)
		x = x[:int(0.08*sr())]
		var e, d float64
		for i := 1; i < len(x); i++ {
			e += x[i] * x[i]
			d += (x[i] - x[i-1]) * (x[i] - x[i-1])
		}
		return d / e
	}
	if bs, bsh := bright(S), bright(Sh); bsh >= bs {
		t.Errorf("sh is not darker than s: %g against %g", bsh, bs)
	}
}

func TestMatrixMessages(t *testing.T) {
	v := New(Solo())
	v.ReceiveMessage(psalm.Message{"mod2Src": int(modmatrix.Velocity), "mod2Dest": 1 + dBreath, "mod2Amount": -0.5, "consonant": 3})
	if got := v.cfg.Mods[1]; got.Src != modmatrix.Velocity || got.Dest != "breath" || got.Amount != -0.5 {
		t.Errorf("slot %+v", got)
	}
	if v.cfg.Consonant != S {
		t.Errorf("consonant %v", v.cfg.Consonant)
	}
	v.Receive(0.3, InExt1+2)
	if v.cfg.Ext[2] != 0.3 {
		t.Errorf("ext %v", v.cfg.Ext)
	}
}

func TestDescribeModulation(t *testing.T) {
	cfg := Choir()
	cfg.Consonant = K
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 900}
	cfg.Mods[3] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "vowel", Amount: 0.3}
	v := New(cfg)
	paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
}

func TestNoNaNAcrossMatrix(t *testing.T) {
	for _, cfg := range []Config{Solo(), Choir()} {
		for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
			for d := range Dests {
				c := cfg
				c.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 1500, Resonance: 0.5}
				c.Consonant = Sh
				c.Mods[0] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: 1}
				c.Mods[1] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: -1}
				c.Ext = [4]float64{1, 1, 1, 1}
				l, r := sung(c, psalm.Message{"pitch": 48.0, "modWheel": 1.0}, 0.25)
				for _, x := range append(l, r...) {
					if math.IsNaN(x) || math.IsInf(x, 0) || math.Abs(x) > 40 {
						t.Fatalf("src %v dest %s: %g", src, Dests[d].Key, x)
					}
				}
			}
		}
	}
}
