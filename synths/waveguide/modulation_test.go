package waveguide

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

func playMsg(cfg Config, msg psalm.Message, seconds float64) []float64 {
	psalm.Seed(1)
	v := New(cfg)
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	return run(v, seconds)
}

func TestFilterDarkens(t *testing.T) {
	cfg := dry(Steel())
	_, open := play(cfg, 60, 0.5)
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 400}
	_, dark := play(cfg, 60, 0.5)
	f := hz(60) * 8
	if amplitude(dark, f) > 0.2*amplitude(open, f) {
		t.Errorf("the filter leaves the 8th partial at %g of %g", amplitude(dark, f), amplitude(open, f))
	}
	if amplitude(dark, hz(60)) < 0.3*amplitude(open, hz(60)) {
		t.Errorf("the filter takes the fundamental")
	}
}

func TestMatrixMovesDestinations(t *testing.T) {
	// decay: a longer ring
	cfg := dry(Guitar())
	cfg.Decay = 0.5
	cfg.Ext[0] = 1
	base := playMsg(cfg, psalm.Message{"pitch": 57.0}, 1)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "decay", Amount: 1}
	long := playMsg(cfg, psalm.Message{"pitch": 57.0}, 1)
	tail := func(x []float64) float64 { return rms(x[int(0.7*sr()):]) }
	if tail(long) < 3*tail(base) {
		t.Errorf("the matrix doesn't lengthen the decay: %g vs %g", tail(long), tail(base))
	}
	// level
	cfg = dry(Guitar())
	cfg.Ext[0] = 1
	a := rms(playMsg(cfg, psalm.Message{"pitch": 57.0}, 0.3))
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: 1}
	b := rms(playMsg(cfg, psalm.Message{"pitch": 57.0}, 0.3))
	if b < 1.3*a {
		t.Errorf("the matrix doesn't raise the level: %g vs %g", b, a)
	}
	// pitch: a key track of the wheel, an octave up with the wheel full
	cfg = dry(Violin())
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "pitch", Amount: 1}
	x := playMsg(cfg, psalm.Message{"pitch": 57.0, "modWheel": 1.0}, 1)
	if p := pitchOf(x[int(0.5*sr()):]); math.Abs(cents(p, hz(69))) > 40 {
		t.Errorf("the wheel moves the pitch to %g Hz, want %g", p, hz(69))
	}
	// cutoff, with the filter on
	cfg = dry(Steel())
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 300}
	closed := playMsg(cfg, psalm.Message{"pitch": 60.0}, 0.4)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "cutoff", Amount: 1}
	opened := playMsg(cfg, psalm.Message{"pitch": 60.0, "modWheel": 1.0}, 0.4)
	if amplitude(opened, hz(60)*8) < 3*amplitude(closed, hz(60)*8) {
		t.Errorf("the matrix doesn't open the filter")
	}
}

func TestMatrixMessages(t *testing.T) {
	v := New(Violin())
	v.ReceiveMessage(psalm.Message{"mod1Src": int(modmatrix.LFO1), "mod1Dest": 1 + dBodyMix, "mod1Amount": 0.5})
	if got := v.cfg.Mods[0]; got.Src != modmatrix.LFO1 || got.Dest != "bodyMix" || got.Amount != 0.5 {
		t.Errorf("slot %+v", got)
	}
	v.Receive(0.7, InExt1+1)
	if v.cfg.Ext[1] != 0.7 {
		t.Errorf("ext %v", v.cfg.Ext)
	}
}

func TestSympathetic(t *testing.T) {
	cfg := dry(Steel())
	cfg.Decay = 3
	_, off := play(cfg, 52, 1.5)
	cfg.Sympathetic = 1
	_, on := play(cfg, 52, 1.5)
	// the strings tuned to the fifth above the octave take the third partial along, and the output differs
	w := on[int(0.3*sr()):]
	o := off[int(0.3*sr()):]
	if amplitude(w, hz(52)*3) < 1.1*amplitude(o, hz(52)*3) {
		t.Errorf("the sympathetic strings don't add: %g vs %g", amplitude(w, hz(52)*3), amplitude(o, hz(52)*3))
	}
}

func TestNoNaNAcrossMatrix(t *testing.T) {
	for _, cfg := range []Config{Steel(), Violin()} {
		for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
			for d := range Dests {
				c := cfg
				c.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 1500, Resonance: 0.5}
				c.Sympathetic = 0.5
				c.Mods[0] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: 1}
				c.Mods[1] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: -1}
				c.Ext = [4]float64{1, 1, 1, 1}
				x := playMsg(c, psalm.Message{"pitch": 48.0, "modWheel": 1.0}, 0.25)
				for _, s := range x {
					if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 20 {
						t.Fatalf("model %v src %v dest %s: %g", c.Model, src, Dests[d].Key, s)
					}
				}
			}
		}
	}
}

func TestDescribeModulation(t *testing.T) {
	cfg := Violin()
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 900}
	cfg.Mods[2] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "pitch", Amount: 0.3}
	cfg.Sympathetic = 0.4
	v := New(cfg)
	paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
}
