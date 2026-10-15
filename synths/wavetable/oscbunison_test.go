package wavetable

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

func bOnly() Config {
	cfg := steady()
	cfg.Position = 2.0 / 3.0
	cfg.OscBLevel, cfg.OscBPosition = 1, 2.0/3.0
	return cfg
}

func TestOscBUnisonThickens(t *testing.T) {
	msg := psalm.Message{"frequency": 110.0}
	one := bOnly()
	seven := one
	seven.OscBUnison, seven.OscBSpread, seven.Spread = 7, 40, 1
	l1, _ := render(one, msg, 80)
	l7, r7 := render(seven, msg, 80)
	for _, x := range [][]float64{l1, l7, r7} {
		for _, v := range x {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatal("NaN")
			}
		}
	}
	// the level stays about the same, the energy between the harmonics (the beating) grows
	if ratio := (rms(l7) + rms(r7)) / 2 / rms(l1); ratio < 0.5 || ratio > 1.6 {
		t.Errorf("level ratio %v", ratio)
	}
	h1, b1 := harmonicEnergy(spectrum(l1), 110)
	h7, b7 := harmonicEnergy(spectrum(l7), 110)
	if b7/h7 < 5*b1/h1 {
		t.Errorf("no thicker: between/harmonic %g vs %g", b7/h7, b1/h1)
	}
	corr, el, er := 0.0, 0.0, 0.0
	for i := range l7 {
		corr += l7[i] * r7[i]
		el += l7[i] * l7[i]
		er += r7[i] * r7[i]
	}
	if c := corr / math.Sqrt(el*er); c > 0.9 {
		t.Errorf("not wide: correlation %v", c)
	}
}

func TestOscBUnisonDefaultsOff(t *testing.T) {
	// 0 and 1 are the same single oscillator
	a, _ := render(bOnly(), psalm.Message{"pitch": 48.0}, 20)
	c := bOnly()
	c.OscBUnison = 1
	b, _ := render(c, psalm.Message{"pitch": 48.0}, 20)
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("unison 1 differs from 0")
		}
	}
	v := New(bOnly())
	v.ReceiveMessage(psalm.Message{"oscBUnison": 99, "oscBSpread": 25.0})
	if v.cfg.OscBUnison != MaxUnison || v.cfg.OscBSpread != 25 {
		t.Errorf("%d %g", v.cfg.OscBUnison, v.cfg.OscBSpread)
	}
}

func TestOscBUnisonWithWarpAndMatrix(t *testing.T) {
	for mode := WarpOff; mode < numWarpModes; mode++ {
		cfg := bOnly()
		cfg.OscBUnison, cfg.OscBSpread = 5, 30
		cfg.WarpMode, cfg.Warp = mode, 0.7
		l, r := render(cfg, psalm.Message{"pitch": 60.0}, 20)
		if rms(l) == 0 {
			t.Errorf("mode %d silent", mode)
		}
		for i := range l {
			if math.IsNaN(l[i]) || math.IsNaN(r[i]) {
				t.Fatalf("mode %d: NaN", mode)
			}
		}
	}
	// the matrix opens B
	cfg := steady()
	cfg.OscBUnison, cfg.OscBSpread, cfg.OscBOctave = 3, 20, 1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "oscBLevel", Amount: 1}
	with, _ := render(cfg, psalm.Message{"pitch": 48.0}, 30)
	cfg.Mods[0] = modmatrix.Slot{}
	without, _ := render(cfg, psalm.Message{"pitch": 48.0}, 30)
	if rms(with) < 1.1*rms(without) {
		t.Errorf("matrix: %g vs %g", rms(with), rms(without))
	}
}
