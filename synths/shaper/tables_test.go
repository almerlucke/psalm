package shaper

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// the tables are used for the chains that need more than one time a sample, with the oversampling automatic
func TestTableUsed(t *testing.T) {
	use := func(chain string, over int, pitch float64) bool {
		cfg := plain(chain)
		cfg.Oversample = over
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())
		return v.table
	}
	if use("sin", 0, 60) {
		t.Errorf("a sine is read as it is")
	}
	if !use("bipolar", 0, 60) {
		t.Errorf("a saw is not read from a table")
	}
	if use("bipolar", 4, 60) || use("bipolar", 1, 60) {
		t.Errorf("a fixed oversampling uses a table")
	}
}

// a table has no aliasing: as clean as the oversampling, from a saw high up
func TestTableAliasing(t *testing.T) {
	alias := func(over int) float64 {
		cfg := plain("bipolar")
		cfg.Oversample = over
		_, x := play(cfg, psalm.Message{"frequency": 3000.0}, 0.6)
		return amplitude(x[len(x)/3:], 44100-8*3000)
	}
	off, tab := alias(1), alias(0)
	if !(tab < off/10) {
		t.Errorf("alias at 20.1 kHz: %v from a table, %v without", tab, off)
	}
}

// the harmonics of a table are those of the wave read at eight times a sample
func TestTableHarmonics(t *testing.T) {
	for _, chain := range []string{"bipolar", "pulse 0.3 | bipolar", "sin | mult 3 | mirror -1 1 | softclip 1.4"} {
		harm := func(over int, k int) float64 {
			cfg := plain(chain)
			cfg.Oversample = over
			_, x := play(cfg, psalm.Message{"frequency": 1000.0}, 0.5)
			return amplitude(x[len(x)/4:], 1000*float64(k))
		}
		for k := 1; k <= 8; k++ {
			a, b := harm(8, k), harm(0, k)
			if a < 0.005 {
				continue
			}
			if r := 20 * math.Log10(b/a); math.Abs(r) > 1 {
				t.Errorf("%s: harmonic %d is %.2f dB off the oversampled wave (%.3f, %.3f)", chain, k, r, a, b)
			}
		}
	}
}

func TestTableSweepNoClicks(t *testing.T) {
	still := func(mod bool) float64 {
		cfg := plain("bipolar | mult m1[1,2] | mirror -1 1", "pulse 0.3 | bipolar")
		cfg.Macros[0] = 0.5
		cfg.Morph = 0.5
		cfg.LFO1Rate = 3
		cfg.Unison, cfg.Spread = 3, 10
		if mod {
			cfg.Mods = slots(modmatrix.Slot{Src: modmatrix.LFO1, Dest: "morph", Amount: 0.4}, modmatrix.Slot{Src: modmatrix.LFO1, Dest: "macro1", Amount: 0.4})
		}
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"frequency": 220.3}, psalm.CtrlTick{}, v.Config())
		if !v.table {
			t.Fatal("not from tables")
		}
		run(v, 0.2)
		return d2(run(v, 1.5))
	}
	base, swept := still(false), still(true)
	if swept > 2.5*base {
		t.Errorf("second difference %.3f swept against %.3f still: the tables step", swept, base)
	}
}

func TestTableChainSwapNoClick(t *testing.T) {
	cfg := plain("bipolar | mult 1.5 | mirror -1 1")
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 217.3}, psalm.CtrlTick{}, v.Config())
	run(v, 0.3)
	before := run(v, 0.1)
	v.ReceiveMessage(psalm.Message{"chainA": "pulse 0.3 | bipolar"})
	after := run(v, 0.2)
	base := d2(before)
	if got := d2(append(append([]float64{}, before[len(before)-8:]...), after...)); got > 2.5*base+0.05 {
		t.Errorf("second difference %.3f across the swap of a chain, %.3f before", got, base)
	}
}

// the pitch envelope of a kick or a laser takes a table through harmonics that fit at the start and not later
func TestTablePitchEnvelope(t *testing.T) {
	cfg := plain("bipolar | softclip 1.5")
	cfg.PitchAmount, cfg.PitchEnv = 36, env(1, 150, 0, 60)
	cfg.AmpEnv = env(1, 400, 0.5, 100)
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 40.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.6)
	if p := peak(x); p < 0.1 || p > 3 || math.IsNaN(p) {
		t.Errorf("peak %v", p)
	}
	if d2(x) > 3 {
		t.Errorf("a click in the fall: %.3f", d2(x))
	}
}

// a unison reads one table for all its copies: the same sound as the single, spread
func TestTableUnison(t *testing.T) {
	cfg := plain("bipolar | softclip 1.2")
	cfg.Unison, cfg.Spread, cfg.UniStereo = 5, 15, 1
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 180.0}, psalm.CtrlTick{}, v.Config())
	l, r := stereoRun(v, 0.6)
	if !v.table || rms(l) < 0.05 || rms(r) < 0.05 {
		t.Errorf("table %v, rms %v %v", v.table, rms(l), rms(r))
	}
}

// --- the cross modulation ---

func xmodCfg() Config {
	cfg := plain("sin")
	cfg.Oversample = 1
	cfg.Line1 = Line{Level: 1, Octave: 1} // twice the pitch
	cfg.Line2 = Line{Level: 1}
	return cfg
}

func TestXModOffIsOff(t *testing.T) {
	a := xmodCfg()
	b := xmodCfg()
	b.XModType = 1 // the type does nothing at an amount of 0
	_, x := play(a, psalm.Message{"frequency": 200.0}, 0.3)
	_, y := play(b, psalm.Message{"frequency": 200.0}, 0.3)
	for i := range x {
		if x[i] != y[i] {
			t.Fatalf("differ at %d", i)
		}
	}
}

// phase modulation puts sidebands round line 2 at multiples of line 1; the silent line 1 is not heard
func TestXModPhase(t *testing.T) {
	cfg := xmodCfg()
	_, plainX := play(cfg, psalm.Message{"frequency": 200.0}, 0.6)
	cfg.XMod = 0.15
	cfg.Line1.Level = 1
	_, x := play(cfg, psalm.Message{"frequency": 200.0}, 0.6)
	side := func(x []float64) float64 { return amplitude(x[4000:], 600) } // 200 + 2 * 200
	if side(x) < 3*side(plainX)+0.02 {
		t.Errorf("sideband at 600 Hz: %.3f with the modulation, %.3f without", side(x), side(plainX))
	}
	cfg.XModSilent = true
	_, quiet := play(cfg, psalm.Message{"frequency": 200.0}, 0.6)
	if a, b := amplitude(x[4000:], 400), amplitude(quiet[4000:], 400); !(b < a/3) {
		t.Errorf("line 1 at 400 Hz: %.3f heard, %.3f silent", a, b)
	}
	if a := amplitude(quiet[4000:], 200); a < 0.2 {
		t.Errorf("line 2 at 200 Hz is %.3f when line 1 is silent", a)
	}
}

// ring modulation: the sum and difference of the two
func TestXModRing(t *testing.T) {
	cfg := xmodCfg()
	cfg.Line1.Octave = 0
	cfg.Line1.Detune = 1200 * math.Log2(1.5) // 300 Hz against 200
	cfg.XMod, cfg.XModType, cfg.XModSilent = 1, 1, true
	_, x := play(cfg, psalm.Message{"frequency": 200.0}, 0.6)
	x = x[4000:]
	if sum, diff, own := amplitude(x, 500), amplitude(x, 100), amplitude(x, 200); sum < 0.08 || diff < 0.08 || own > sum/2 {
		t.Errorf("200 Hz times 300 Hz: %.3f at 500, %.3f at 100, %.3f at 200", sum, diff, own)
	}
}

// the matrix moves it: an envelope of the amp on the amount
func TestXModMatrix(t *testing.T) {
	cfg := xmodCfg()
	cfg.XModSilent = true
	_, off := play(cfg, psalm.Message{"frequency": 200.0}, 0.5)
	cfg.Mods = slots(modmatrix.Slot{Src: modmatrix.AmpEnv, Dest: "xmod", Amount: 0.3})
	_, on := play(cfg, psalm.Message{"frequency": 200.0}, 0.5)
	side := func(x []float64) float64 { return amplitude(x[4000:], 600) }
	if side(on) < side(off)+0.03 {
		t.Errorf("sideband at 600 Hz: %.3f with the matrix, %.3f without", side(on), side(off))
	}
}

// from tables, in a unison, with a silent modulator: finite and sounding
func TestXModTables(t *testing.T) {
	cfg := plain("bipolar | softclip 1.3")
	cfg.Line1 = Line{Level: 1, Octave: 1}
	cfg.Line2 = Line{Level: 1, Detune: 7}
	cfg.XMod, cfg.XModSilent = 0.3, true
	cfg.Unison, cfg.Spread = 3, 10
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 150.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.5)
	if p := peak(x); p < 0.05 || math.IsNaN(p) || p > 4 {
		t.Errorf("peak %v", p)
	}
}

// a chain swapped while the tables are on their way to a new morph (a pad on an LFO, every moment): the mix the sound
// had reached is kept, and it fades to the new chain with no step
func TestTableSwapDuringSweep(t *testing.T) {
	for _, swapAfter := range []float64{0.301, 0.337, 0.349, 0.4} {
		cfg := plain("bipolar | mult m1[1,2] | mirror -1 1", "pulse 0.3 | bipolar")
		cfg.Macros[0], cfg.Morph, cfg.LFO1Rate = 0.5, 0.5, 4
		cfg.Unison, cfg.Spread = 3, 10
		cfg.Mods = slots(modmatrix.Slot{Src: modmatrix.LFO1, Dest: "morph", Amount: 0.5}, modmatrix.Slot{Src: modmatrix.LFO1, Dest: "macro1", Amount: 0.5})
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"frequency": 220.3}, psalm.CtrlTick{}, v.Config())
		run(v, swapAfter)
		before := run(v, 0.05)
		v.ReceiveMessage(psalm.Message{"chainA": "sin | mult 3 | mirror -1 1 | softclip 1.4", "chainB": "bipolar | mult 2 | mirror -1 1"})
		after := run(v, 0.15)
		base := d2(before)
		if got := d2(append(append([]float64{}, before[len(before)-8:]...), after...)); got > 2.5*base+0.05 {
			t.Errorf("swap at %.3f s: second difference %.3f across the swap, %.3f before", swapAfter, got, base)
		}
	}
}
