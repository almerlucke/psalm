package shaper

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// brightness is the spectral centroid of x in harmonics of f0 (the first thirty, in steps of an eighth of f0, so a
// unison spread over a few cents is counted whole)
func brightness(x []float64, f0 float64) float64 {
	var num, den float64
	for f := f0 / 2; f <= 30*f0 && f < sr()/2.2; f += f0 / 8 {
		a := amplitude(x, f)
		num += f / f0 * a * a
		den += a * a
	}
	return num / (den + 1e-12)
}

// upper is the energy from the second to the fourteenth harmonic against that round the first
func upper(x []float64, f0 float64) float64 {
	var hi, lo float64
	for f := 0.8 * f0; f <= 14*f0; f += f0 / 8 {
		a := amplitude(x, f)
		if f < 1.3*f0 {
			lo += a * a
		} else if f > 1.8*f0 {
			hi += a * a
		}
	}
	return hi / (lo + 1e-12)
}

func midiHz(p float64) float64 { return 440 * math.Pow(2, (p-69)/12) }

func after(x []float64, from, to float64) []float64 { return x[int(from*sr()):int(to*sr())] }

// Four beasts is another instrument on every register: the morph follows the key
func TestFourBeastsFollowTheKey(t *testing.T) {
	psalm.CurrentEnv().Seed(3)
	var b []float64
	for _, p := range []float64{36, 48, 60, 72, 84} {
		_, x := play(FourBeasts(), psalm.Message{"pitch": p}, 1.2)
		b = append(b, brightness(after(x, 0.4, 1.1), midiHz(p)))
	}
	t.Log("harmonic centroid by octave:", b)
	// the instrument changes: the centroids (in harmonics of each pitch) are not all alike
	lo, hi := b[0], b[0]
	for _, v := range b {
		lo, hi = min(lo, v), max(hi, v)
	}
	if hi < lo*1.4 {
		t.Errorf("the register does not change the timbre: %v", b)
	}
}

// the touch changes the instrument, not only the level
func TestTouchChangesTheInstrument(t *testing.T) {
	for _, name := range []string{"trumpetVoice", "lionBass"} {
		touch := func(vel float64) []float64 {
			v := New(Presets[name]())
			v.NoteOn(vel, psalm.Message{"pitch": 48.0}, psalm.CtrlTick{}, v.Config())
			return run(v, 1.5)
		}
		soft, hard := touch(0.2), touch(1)
		bs, bh := brightness(after(soft, 0.5, 1.4), midiHz(48)), brightness(after(hard, 0.5, 1.4), midiHz(48))
		t.Logf("%s: soft %.2f hard %.2f", name, bs, bh)
		if bh < bs*1.5 {
			t.Errorf("%s: a hard touch is not much brighter than a soft one (%.2f, %.2f)", name, bs, bh)
		}
	}
}

// a pad that travels: the start of the chord and its sixth second are not the same wave
func TestRainbowTravels(t *testing.T) {
	psalm.CurrentEnv().Seed(3)
	_, x := play(Rainbow(), psalm.Message{"pitch": 55.0}, 9)
	f := midiHz(55)
	b0, b1 := upper(after(x, 0.8, 1.8), f), upper(after(x, 6, 7), f)
	t.Logf("rainbow: %.3f then %.3f", b0, b1)
	if b1 < 3*b0 {
		t.Errorf("the wave does not travel: %.2f, %.2f", b0, b1)
	}
}

// the drone and the sea keep changing and do not come back: no two seconds of a long note alike
func TestSlowWanderDoesNotRepeat(t *testing.T) {
	psalm.CurrentEnv().Seed(3)
	for _, name := range []string{"throneDrone", "crystalSea"} {
		_, x := play(Presets[name](), psalm.Message{"pitch": 50.0}, 26)
		f := midiHz(50)
		var b []float64
		for s := 6.0; s+1 <= 25; s += 3 {
			b = append(b, brightness(after(x, s, s+1), f))
		}
		lo, hi := b[0], b[0]
		for _, v := range b {
			lo, hi = min(lo, v), max(hi, v)
		}
		t.Logf("%s: %.2f", name, b)
		if hi < lo*1.1 {
			t.Errorf("%s does not move: %v", name, b)
		}
	}
}

// every note of the lead draws itself anew
func TestManyEyesDrawsEveryNote(t *testing.T) {
	psalm.CurrentEnv().Seed(3)
	var b []float64
	for range 6 {
		_, x := play(ManyEyes(), psalm.Message{"pitch": 64.0}, 1.0)
		b = append(b, brightness(after(x, 0.3, 0.9), midiHz(64)))
	}
	lo, hi := b[0], b[0]
	for _, v := range b {
		lo, hi = min(lo, v), max(hi, v)
	}
	t.Logf("many eyes: %.2f", b)
	if hi < lo*1.2 {
		t.Errorf("the notes are alike: %v", b)
	}
}

// the thunder falls; the lightning is a snare low and a hat high
func TestKitFromTheThrone(t *testing.T) {
	_, th := play(Thunder(), psalm.Message{"pitch": 33.0}, 1)
	// the pitch falls two octaves: in the first 20 ms the energy sits far above the 55 Hz of the note
	early, late := amplitude(after(th, 0.0, 0.02), 110), amplitude(after(th, 0.3, 0.4), 55)
	if early == 0 || late == 0 {
		t.Errorf("thunder: %v %v", early, late)
	}
	_, snare := play(Lightning(), psalm.Message{"pitch": 50.0}, 0.5)
	_, hat := play(Lightning(), psalm.Message{"pitch": 90.0}, 0.5)
	cs, ch := centroidHz(snare), centroidHz(hat)
	t.Logf("lightning: snare %.0f Hz, hat %.0f Hz", cs, ch)
	if ch < cs*1.8 {
		t.Errorf("the lightning does not follow the key: %.0f, %.0f", cs, ch)
	}
}

// centroidHz is the spectral centroid of x (steps of 100 Hz up to 16 kHz)
func centroidHz(x []float64) float64 {
	var num, den float64
	for f := 100.0; f < 16000; f += 100 {
		a := amplitude(x, f)
		num += f * a * a
		den += a * a
	}
	return num / (den + 1e-12)
}
