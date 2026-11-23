package waveguide

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func TestKeyTrackOf(t *testing.T) {
	var none *KeyTrack
	if d, k := none.of(60); d != 0 || k != 1 {
		t.Errorf("no track: %g %g", d, k)
	}
	if d, k := (&KeyTrack{From: 40, Ref: 50}).of(60); d != 0 || k != 1 {
		t.Errorf("a track without tables: %g %g", d, k)
	}
	k := &KeyTrack{From: 40, Ref: 41, Decay: []float64{8, 4, 2}, Damping: []float64{0.2, 0.4, 0.6}}
	if d, s := k.of(41); d != 0 || s != 1 {
		t.Errorf("at the reference pitch: %g %g", d, s)
	}
	if d, s := k.of(42); math.Abs(d-0.2) > 1e-9 || math.Abs(s-0.5) > 1e-9 {
		t.Errorf("a semitone above: damping %+g (want +0.2), decay x%g (want 0.5)", d, s)
	}
	// between the semitones, and clamped outside the table
	if d, s := k.of(41.5); math.Abs(d-0.1) > 1e-9 || math.Abs(s-math.Sqrt(0.5)) > 1e-9 {
		t.Errorf("half a semitone: damping %+g, decay x%g", d, s)
	}
	if d, s := k.of(90); math.Abs(d-0.2) > 1e-9 || math.Abs(s-0.5) > 1e-9 {
		t.Errorf("above the table: damping %+g, decay x%g", d, s)
	}
	if d, s := k.of(20); math.Abs(d+0.2) > 1e-9 || math.Abs(s-2) > 1e-9 {
		t.Errorf("below the table: damping %+g, decay x%g", d, s)
	}
}

// the preset has the settings of the real guitar at its reference pitch, so a plain note there is
// what the fit made
func TestOwnGuitarSettings(t *testing.T) {
	c := OwnGuitar()
	if c.Track == nil || len(c.Track.Decay) == 0 || len(c.Track.Decay) != len(c.Track.Damping) {
		t.Fatal("no fitted track: go run ./cmd/tools/guitarcompare -fit")
	}
	ref := float64(c.Track.Ref)
	if c.Decay != c.Track.DecayAt(ref) || c.Damping != c.Track.DampingAt(ref) {
		t.Errorf("decay %g damping %g, the table has %g %g at %g", c.Decay, c.Damping, c.Track.DecayAt(ref), c.Track.DampingAt(ref), ref)
	}
	for p := range len(c.Track.Decay) {
		if d, g := c.Track.Decay[p], c.Track.Damping[p]; !(d > 0.3 && d < 120 && g >= -minDamping && g <= 0.7) {
			t.Errorf("pitch %d: decay %g damping %g", c.Track.From+p, d, g)
		}
	}
	if _, ok := Presets["ownGuitar"]; !ok {
		t.Error("not in Presets")
	}
}

// a string of the fitted guitar falls at the pace of the real one at its own pitch: the low strings
// slowly, the high ones quickly
func TestOwnGuitarFalls(t *testing.T) {
	fall := func(cfg Config, pitch float64) float64 {
		cfg.BodyMix = 0
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": pitch, "velocity": 0.65}, psalm.CtrlTick{}, v.Config())
		x := run(v, 2.5)
		level := func(from, to float64) float64 {
			e := 0.0
			for _, s := range x[int(from*sr()):int(to*sr())] {
				e += s * s
			}
			return 10 * math.Log10(e/((to-from)*sr())+1e-12)
		}
		return (level(0.3, 0.5) - level(2.0, 2.2)) / 1.7
	}
	low, high := fall(OwnGuitar(), 41), fall(OwnGuitar(), 69)
	if low > 8 || high < 12 || high > 18 {
		t.Errorf("falls %.1f dB/s at 41 and %.1f at 69: the real guitar 3-5 and 14", low, high)
	}
	// the track is more than the settings of A3: the real guitar falls faster at D4 than the model does
	// without it (the pitch scaling of the string alone gives 7.6 dB/s there, the real guitar 12.5)
	plain := OwnGuitar()
	plain.Track = nil
	if with, without := fall(OwnGuitar(), 62), fall(plain, 62); without > with-3 {
		t.Errorf("at 62 the string falls %.1f dB/s with the track and %.1f without", with, without)
	}
}

// the damping goes below 0 down to a loop filter that takes nothing: a string that keeps its top partials
func TestDampingBelowZero(t *testing.T) {
	top := func(damping float64) float64 {
		cfg := Guitar()
		cfg.BodyMix = 0
		cfg.Damping = damping
		sameNoise()
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 76.0, "velocity": 0.65}, psalm.CtrlTick{}, v.Config())
		x := run(v, 1.5)
		hi, all := 0.0, 0.0
		from := int(0.5 * sr())
		for _, s := range x[from:] {
			all += s * s
		}
		// the energy above 2 kHz: what is left of a first difference taken three times
		y := x[from:]
		for range 3 {
			d := make([]float64, len(y)-1)
			for i := range d {
				d[i] = y[i+1] - y[i]
			}
			y = d
		}
		for _, s := range y {
			hi += s * s
		}
		return 10 * math.Log10(hi/all)
	}
	plain, open, further := top(0), top(-1.0/9), top(-0.5)
	if open < plain+6 {
		t.Errorf("the top partials are %.1f dB at damping 0 and %.1f at -1/9: they should be much more", plain, open)
	}
	if math.Abs(further-open) > 0.01 {
		t.Errorf("below -1/9 the loop filter takes nothing, the sound is the same: %.2f and %.2f dB", open, further)
	}
}
