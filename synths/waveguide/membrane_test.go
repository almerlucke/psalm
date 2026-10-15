package waveguide

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

var drums = []string{"tom", "floorTom", "conga", "tabla", "frameDrum", "snareDrum"}

func peakOf(x []float64) float64 {
	p := 0.0
	for _, s := range x {
		p = math.Max(p, math.Abs(s))
	}
	return p
}

// highRatio is how much of x is in the modes of the skin from 2.1 times the lowest up, against the
// two lowest
func highRatio(x []float64, f float64) float64 {
	high, low := 0.0, 0.0
	for _, r := range []float64{2.136, 2.296, 2.653, 2.918, 3.156, 3.5} {
		high += amplitude(x, f*r)
	}
	for _, r := range []float64{1, 1.594} {
		low += amplitude(x, f*r)
	}
	return high / low
}

// diff is the first difference of x: its high part
func diff(x []float64) []float64 {
	d := make([]float64, len(x)-1)
	for i := range d {
		d[i] = x[i+1] - x[i]
	}
	return d
}

func at(x []float64, from, to float64) []float64 { return x[int(from*sr()):int(to*sr())] }

func TestMembraneSoundsAndDecays(t *testing.T) {
	sameNoise()
	for _, name := range drums {
		cfg := Presets[name]()
		_, x := play(cfg, 50, 4)
		if p := peakOf(at(x, 0, 0.1)); p < 0.1 || math.IsNaN(p) {
			t.Errorf("%s: peaks at %g", name, p)
		}
		first, late := rms(at(x, 0, 0.1)), rms(at(x, 3.5, 4))
		if late > 0.01*first {
			t.Errorf("%s: does not decay: %g then %g", name, first, late)
		}
	}
	// the voice rests once it has died
	v, _ := play(Tom(), 50, 0.1)
	run(v, 6)
	if v.IsActive() {
		t.Errorf("a struck skin rings for ever")
	}
}

func TestMembraneDecayAndDamping(t *testing.T) {
	tail := func(mod func(*Config)) float64 {
		cfg := Tom()
		cfg.Snare = 0
		mod(&cfg)
		_, x := play(cfg, 50, 1.2)
		return rms(at(x, 0.3, 0.6))
	}
	// the higher modes die first: the sound dulls (its first difference falls against its level)
	dull := func(damping float64) float64 {
		cfg := Tom()
		cfg.Snare, cfg.Damping, cfg.Tension = 0, damping, 0
		_, x := play(cfg, 50, 1)
		return highRatio(at(x, 0.2, 0.5), hz(50))
	}
	if lo, hi := dull(0), dull(1); hi > 0.7*lo {
		t.Errorf("damping does not dull the ring: %g vs %g", lo, hi)
	}
	// (the fundamental is untouched by damping and carries most of this window, so the ring is no longer
	// than it was, and the high modes' share is the dulling above)
	if lo, hi := tail(func(c *Config) { c.Damping = 0 }), tail(func(c *Config) { c.Damping = 1 }); hi > lo {
		t.Errorf("damping lengthens the ring: %g vs %g", lo, hi)
	}
	if short, long := tail(func(c *Config) { c.Decay = 0.3 }), tail(func(c *Config) { c.Decay = 1.5 }); long < 5*short {
		t.Errorf("decay does not lengthen the ring: %g vs %g", short, long)
	}
	// the fundamental falls 60 dB in Decay: 20 dB in a third of it
	cfg := Tom()
	cfg.Damping, cfg.Snare, cfg.Strike, cfg.Decay, cfg.Tension = 0, 0, 0, 1.5, 0
	_, x := play(cfg, 57, 1.2)
	f := hz(57)
	a, b := amplitude(at(x, 0.05, 0.15), f), amplitude(at(x, 0.55, 0.65), f)
	if d := 20 * math.Log10(a/b); math.Abs(d-20) > 3 {
		t.Errorf("in 0.5 s the lowest mode falls %.1f dB, want 20", d)
	}
}

func TestMembraneModes(t *testing.T) {
	// struck in the middle, the modes that are not round (the second, 1.59 times the first) have a node
	// there: only the round ones ring; near the rim they are there
	second := func(strike float64) float64 {
		cfg := Tom()
		cfg.Snare, cfg.Strike, cfg.Tension, cfg.Damping = 0, strike, 0, 0
		_, x := play(cfg, 50, 0.5)
		y := at(x, 0.02, 0.4)
		return amplitude(y, hz(50)*1.5934) / amplitude(y, hz(50))
	}
	if mid, rim := second(0), second(0.6); mid > 0.05*rim {
		t.Errorf("the second mode struck in the middle %g, near the rim %g", mid, rim)
	}
	// the modes are not harmonic: nothing rings at the octave, the modes of 2.30 and 2.14 do
	cfg := Tom()
	cfg.Snare, cfg.Strike, cfg.Tension, cfg.Damping = 0, 0.5, 0, 0
	_, x := play(cfg, 50, 0.5)
	y := at(x, 0.02, 0.4)
	if oct, mode := amplitude(y, 2*hz(50)), amplitude(y, 2.1360*hz(50)); oct > 0.2*mode {
		t.Errorf("the octave rings %g beside the mode at 2.14: %g", oct, mode)
	}
}

func TestMembraneHardnessAndVelocity(t *testing.T) {
	bright := func(hard, vel float64) float64 {
		cfg := Tom()
		cfg.Snare, cfg.Hardness = 0, hard
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 50.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
		return highRatio(run(v, 0.3), hz(50))
	}
	if soft, hard := bright(0.1, 0.5), bright(0.9, 0.5); hard < 1.5*soft {
		t.Errorf("a hard stick is not brighter: %g vs %g", soft, hard)
	}
	if soft, hard := bright(0.5, 0.2), bright(0.5, 1); hard < 1.1*soft {
		t.Errorf("a hard hit is not brighter: %g vs %g", soft, hard)
	}
	loud := func(vel float64) float64 {
		v := New(Tom())
		v.NoteOn(1, psalm.Message{"pitch": 50.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
		return peakOf(run(v, 0.2))
	}
	if soft, hard := loud(0.2), loud(1); hard < 2*soft {
		t.Errorf("velocity: peaks %g and %g", soft, hard)
	}
}

func TestMembraneTension(t *testing.T) {
	// struck hard on a tight skin the pitch is above its note and falls back
	peakAt := func(x []float64) float64 {
		best, bf := 0.0, 0.0
		for f := hz(50) * 0.9; f < hz(50)*1.5; f += 1 {
			if a := amplitude(x, f); a > best {
				best, bf = a, f
			}
		}
		return bf
	}
	cfg := Tom()
	cfg.Snare, cfg.Strike, cfg.Damping, cfg.Decay = 0, 0, 0, 2
	cfg.Tension = 1
	_, x := play(cfg, 50, 1)
	early, late := peakAt(at(x, 0, 0.08)), peakAt(at(x, 0.6, 0.9))
	if early < 1.04*late {
		t.Errorf("the pitch of a tight skin: %.1f Hz at the strike, %.1f Hz after", early, late)
	}
	if math.Abs(cents(late, hz(50))) > 15 {
		t.Errorf("the skin settles at %.1f Hz, want %.1f", late, hz(50))
	}
	cfg.Tension = 0
	_, x = play(cfg, 50, 1)
	if f := peakAt(at(x, 0, 0.08)); math.Abs(cents(f, hz(50))) > 40 {
		t.Errorf("without tension the strike is at %.1f Hz, want %.1f", f, hz(50))
	}
}

func TestMembraneSnareAndRelease(t *testing.T) {
	buzz := func(snare float64) float64 {
		cfg := SnareDrum()
		cfg.Snare = snare
		sameNoise()
		_, x := play(cfg, 50, 0.6)
		return rms(diff(at(x, 0.15, 0.5)))
	}
	if off, on := buzz(0), buzz(1); on < 1.5*off {
		t.Errorf("the snare does not buzz: %g vs %g", off, on)
	}
	// a hand on the skin stops it
	ring := func(release float64) float64 {
		cfg := Conga()
		cfg.Release = release
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 50.0}, psalm.CtrlTick{}, v.Config())
		run(v, 0.05)
		v.NoteOff()
		return rms(at(run(v, 0.5), 0.2, 0.5))
	}
	if free, held := ring(0), ring(0.05); held > 0.2*free {
		t.Errorf("a hand on the skin: %g, free %g", held, free)
	}
}

func TestMembraneNoNaN(t *testing.T) {
	for _, name := range drums {
		for _, p := range []float64{0, 12, 24, 40, 60, 84, 100, 127} {
			for _, vel := range []float64{0, 0.3, 1} {
				cfg := Presets[name]()
				v := New(cfg)
				v.NoteOn(1, psalm.Message{"pitch": p, "velocity": vel}, psalm.CtrlTick{}, v.Config())
				for _, s := range run(v, 0.4) {
					if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 10 {
						t.Fatalf("%s pitch %g velocity %g: %g", name, p, vel, s)
					}
				}
			}
		}
	}
}

func TestMembraneMatrixAndFilter(t *testing.T) {
	// the matrix moves the decay
	cfg := Tom()
	cfg.Snare, cfg.Decay, cfg.Ext[0] = 0, 0.3, 1
	sameNoise()
	base := playMsg(cfg, psalm.Message{"pitch": 50.0}, 1)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "decay", Amount: 1}
	long := playMsg(cfg, psalm.Message{"pitch": 50.0}, 1)
	if tail := func(x []float64) float64 { return rms(at(x, 0.4, 0.8)) }; tail(long) < 3*tail(base) {
		t.Errorf("the matrix does not lengthen the ring: %g vs %g", tail(long), tail(base))
	}
	// and the filter darkens it
	cfg = Tom()
	open := playMsg(cfg, psalm.Message{"pitch": 50.0}, 0.3)
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 200}
	dark := playMsg(cfg, psalm.Message{"pitch": 50.0}, 0.3)
	if f := 2.296 * hz(50); amplitude(dark, f) > 0.3*amplitude(open, f) {
		t.Errorf("the filter does not darken the drum")
	}
}
