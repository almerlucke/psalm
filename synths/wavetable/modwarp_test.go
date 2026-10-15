package wavetable

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestMatrixMovesDestination(t *testing.T) {
	base := steady()
	base.Filter = LowPass
	base.Cutoff = 40
	base.KeyTrack = 0
	l0, _ := render(base, psalm.Message{"pitch": 48.0}, 30)

	moved := base
	moved.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 1}
	l1, _ := render(moved, psalm.Message{"pitch": 48.0}, 30)
	if rms(l1) < rms(l0)*3 {
		t.Errorf("cutoff mod: rms %g, without %g", rms(l1), rms(l0))
	}

	// level: a negative amount on a full source silences
	quiet := base
	quiet.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "level", Amount: -1}
	l2, _ := render(quiet, psalm.Message{"pitch": 48.0}, 10)
	if rms(l2) > rms(l0)*0.5 {
		t.Errorf("level mod: rms %g, without %g", rms(l2), rms(l0))
	}

	// messages and ext inputs
	v := New(base)
	v.ReceiveMessage(psalm.Message{"mod1Src": int(modmatrix.Ext1), "mod1Dest": 1, "mod1Amount": 0.5})
	if s := v.cfg.Mods[0]; s.Src != modmatrix.Ext1 || s.Dest != "position" || s.Amount != 0.5 {
		t.Errorf("slot %+v", s)
	}
	v.Receive(0.7, InExt1+1)
	if v.cfg.Ext[1] != 0.7 {
		t.Errorf("ext 2 at %g", v.cfg.Ext[1])
	}
}

func TestOscBAddsEnergy(t *testing.T) {
	cfg := steady()
	a, _ := render(cfg, psalm.Message{"pitch": 48.0}, 30)
	cfg.OscBLevel, cfg.OscBOctave = 0.8, 1
	b, _ := render(cfg, psalm.Message{"pitch": 48.0}, 30)
	if rms(b) <= rms(a)*1.1 {
		t.Errorf("osc B: rms %g, without %g", rms(b), rms(a))
	}
	// B an octave up adds energy at the octave
	f0 := 130.8128
	pa, pb := spectrum(a), spectrum(b)
	k := int(2 * f0 / (sr / 16384))
	if pb[k] < pa[k]*1.2 {
		t.Errorf("no more at the octave: %g vs %g", pb[k], pa[k])
	}
}

func TestWarpChangesSpectrum(t *testing.T) {
	cfg := steady()
	ref, _ := render(cfg, psalm.Message{"pitch": 48.0}, 30)
	pr := spectrum(ref)
	for mode := WarpBendUp; mode < numWarpModes; mode++ {
		c := cfg
		c.WarpMode, c.Warp = mode, 0.6
		c.OscBLevel = 0
		x, _ := render(c, psalm.Message{"pitch": 48.0}, 30)
		p := spectrum(x)
		diff, tot := 0.0, 0.0
		for k := range p {
			diff += math.Abs(p[k] - pr[k])
			tot += pr[k]
		}
		if diff < tot*0.05 {
			t.Errorf("%s: spectrum hardly changed (%g of %g)", WarpModes[mode], diff, tot)
		}
	}
	// warp of 0 is no warp
	c := cfg
	c.WarpMode = WarpSync
	x, _ := render(c, psalm.Message{"pitch": 48.0}, 30)
	for i := range x {
		if x[i] != ref[i] {
			t.Fatal("warp 0 changed the sound")
		}
	}
}

func TestDescribeNew(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mods[2] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "warp", Amount: -0.4}
	v := New(cfg)
	set := Describe(v.cfg)
	for _, k := range []string{"warp", "warpMode", "oscBLevel", "oscBOctave", "mod3Src", "mod8Amount", "ext4", "lfo1Rate"} {
		if _, ok := set.Find(k); !ok {
			t.Errorf("%s not described", k)
		}
	}
	if p, _ := set.Find("mod3Dest"); p.Options[int(p.Value)] != "warp" {
		t.Errorf("mod3Dest at %v", p.Value)
	}
	paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
}

func TestMatrixEverything(t *testing.T) {
	for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
		for _, d := range Dests {
			for _, mode := range []WarpMode{WarpOff, WarpBendUp, WarpFMB, WarpSync} {
				cfg := DefaultConfig()
				cfg.Filter, cfg.Unison = LowPass, 3
				cfg.OscBLevel, cfg.WarpMode, cfg.Warp = 0.5, mode, 0.5
				cfg.LFO1Rate, cfg.LFO2Rate = 3, 4
				cfg.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
				cfg.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
				v := New(cfg)
				v.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, v.Config())
				for range 20 {
					v.Synthesize()
					for _, s := range v.Output(0).Channels[0] {
						if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) || math.Abs(float64(s)) > 20 {
							t.Fatalf("%s to %s (warp %d): sample %v", modmatrix.SourceNames[src], d.Key, mode, s)
						}
					}
				}
			}
		}
	}
}
