package waveguide

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

var pipes = []string{"flute", "clarinet", "panpipe"}

// still is a pipe with no vibrato
func still(name string) Config {
	cfg := dry(Presets[name]())
	cfg.VibratoDepth = 0
	return cfg
}

func TestPipeInTune(t *testing.T) {
	sameNoise()
	for _, name := range pipes {
		for _, p := range []float64{48, 55, 60, 67, 72, 79, 84, 91} {
			_, x := play(still(name), p, 1.8)
			y := at(x, 1, 1.8)
			if peakOf(y) < 0.05 {
				t.Errorf("%s %g: silent", name, p)
				continue
			}
			if c := cents(pitchOf(y), hz(p)); math.Abs(c) > 4 {
				t.Errorf("%s %g: %.1f cents out", name, p, c)
			}
		}
	}
}

func TestPipeInTuneWhateverTheBlowing(t *testing.T) {
	// the breath, the embouchure and the damping bend a jet; the pitch is kept
	sameNoise()
	for _, br := range []float64{0.65, 0.8, 0.9} {
		for _, emb := range []float64{0.2, 0.5, 0.8} {
			for _, d := range []float64{0.5, 0.9} {
				cfg := still("flute")
				cfg.Breath, cfg.Embouchure, cfg.Damping = br, emb, d
				_, x := play(cfg, 66, 1.5)
				y := at(x, 0.8, 1.5)
				if peakOf(y) < 0.05 {
					continue // not enough breath to sound
				}
				if c := cents(pitchOf(y), hz(66)); math.Abs(c) > 4 {
					t.Errorf("breath %g embouchure %g damping %g: %.1f cents out", br, emb, d, c)
				}
			}
		}
	}
}

func TestPipeBreathAttackRelease(t *testing.T) {
	sameNoise()
	for _, name := range pipes {
		cfg := still(name)
		cfg.BowAttack, cfg.BowRelease = 0.3, 0.1
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 64.0}, psalm.CtrlTick{}, v.Config())
		x := run(v, 1.2)
		if a, b := rms(at(x, 0, 0.1)), rms(at(x, 0.9, 1.2)); a > 0.5*b {
			t.Errorf("%s: no attack: %g then %g", name, a, b)
		}
		v.NoteOff()
		run(v, cfg.BowRelease+0.6)
		if v.IsActive() {
			t.Errorf("%s: plays on after the breath stopped", name)
		}
		// it holds
		if a, b := rms(at(x, 0.7, 0.9)), rms(at(x, 1, 1.2)); b < 0.6*a || b > 1.6*a {
			t.Errorf("%s: the held note changes: %g then %g", name, a, b)
		}
	}
}

func TestPipeBreathAndNoise(t *testing.T) {
	sameNoise()
	level := func(mod func(*Config)) float64 {
		cfg := still("flute")
		mod(&cfg)
		_, x := play(cfg, 64, 1.2)
		return rms(at(x, 0.7, 1.2))
	}
	if soft, hard := level(func(c *Config) { c.Breath = 0.62 }), level(func(c *Config) { c.Breath = 0.95 }); hard < 1.3*soft {
		t.Errorf("more breath is not louder: %g vs %g", soft, hard)
	}
	if none := level(func(c *Config) { c.Breath = 0.2 }); none > 0.05 {
		t.Errorf("a breath too light still sounds: %g", none)
	}
	// the air is noise: between the partials there is more of it
	hiss := func(noise float64) float64 {
		cfg := still("flute")
		cfg.Noise = noise
		sameNoise()
		_, x := play(cfg, 64, 1.2)
		y := at(x, 0.7, 1.2)
		return rms(diff(diff(y))) / rms(y)
	}
	if clean, airy := hiss(0), hiss(1); airy < 1.2*clean {
		t.Errorf("noise does not add air: %g vs %g", clean, airy)
	}
}

func TestPipeReedIsOdd(t *testing.T) {
	// a reed on a closed tube: the odd partials; a flute has the even ones too
	sameNoise()
	evenOdd := func(name string) float64 {
		_, x := play(still(name), 55, 1.5)
		y := at(x, 0.8, 1.5)
		f := hz(55)
		return (amplitude(y, 2*f) + amplitude(y, 4*f)) / (amplitude(y, 3*f) + amplitude(y, 5*f))
	}
	if cl, fl := evenOdd("clarinet"), evenOdd("flute"); cl > 0.5*fl || cl > 0.5 {
		t.Errorf("even to odd partials: clarinet %g, flute %g", cl, fl)
	}
}

func TestPipeVibrato(t *testing.T) {
	sameNoise()
	cfg := dry(Flute())
	cfg.VibratoDelay, cfg.VibratoDepth, cfg.VibratoRate = 0, 0.3, 5
	_, x := play(cfg, 64, 1.8)
	lo, hi := math.Inf(1), 0.0
	for s := 0.6; s < 1.6; s += 0.025 {
		f := pitchOf(x[int(s*sr()) : int(s*sr())+int(0.03*sr())+int(sr()/30)+4])
		if f > 0 {
			lo, hi = math.Min(lo, f), math.Max(hi, f)
		}
	}
	if spread := cents(hi, lo); spread < 25 || spread > 110 {
		t.Errorf("a vibrato of 0.3 semitones swings %.0f cents", spread)
	}
}

func TestPipeNoNaN(t *testing.T) {
	for _, name := range pipes {
		for _, p := range []float64{0, 12, 24, 48, 72, 96, 110, 127} {
			for _, vel := range []float64{0, 0.5, 1} {
				for _, br := range []float64{0.3, 0.8, 1} {
					cfg := Presets[name]()
					cfg.Breath = br
					v := New(cfg)
					v.NoteOn(1, psalm.Message{"pitch": p, "velocity": vel}, psalm.CtrlTick{}, v.Config())
					for _, s := range run(v, 0.3) {
						if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 20 {
							t.Fatalf("%s pitch %g velocity %g breath %g: %g", name, p, vel, br, s)
						}
					}
				}
			}
		}
	}
}

func TestPipeMatrixAndFilter(t *testing.T) {
	sameNoise()
	// the matrix moves the breath: loud with a source that is full
	cfg := still("flute")
	cfg.Breath = 0.62
	cfg.Ext[0] = 1
	a := rms(at(playMsg(cfg, psalm.Message{"pitch": 64.0}, 1.2), 0.7, 1.2))
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "breath", Amount: 1}
	b := rms(at(playMsg(cfg, psalm.Message{"pitch": 64.0}, 1.2), 0.7, 1.2))
	if b < 1.3*a {
		t.Errorf("the matrix does not raise the breath: %g vs %g", b, a)
	}
	// the filter darkens it
	cfg = still("clarinet")
	open := at(playMsg(cfg, psalm.Message{"pitch": 55.0}, 1.2), 0.6, 1.2)
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 400}
	dark := at(playMsg(cfg, psalm.Message{"pitch": 55.0}, 1.2), 0.6, 1.2)
	if f := 9 * hz(55); amplitude(dark, f) > 0.2*amplitude(open, f) {
		t.Errorf("the filter leaves the 9th partial at %g of %g", amplitude(dark, f), amplitude(open, f))
	}
	if amplitude(dark, hz(55)) < 0.3*amplitude(open, hz(55)) {
		t.Errorf("the filter takes the fundamental")
	}
}
