package waveguide

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
)

// What a player meets: a drum doesn't start with a click, a pipe speaks at once and is in tune while
// it does, and it is the register it was asked to be in.

func pipePresets(reed bool) []string {
	var names []string
	for n, f := range Presets {
		if c := f(); c.Model == Pipe && c.Reed == reed && c.Breath <= 1 { // above 1 asks for the overblown note
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func rmsWin(x []float64, from, to float64) float64 {
	seg := x[int(from*sr()):min(len(x), int(to*sr()))]
	var e float64
	for _, v := range seg {
		e += v * v
	}
	return math.Sqrt(e / float64(len(seg)))
}

// the pitch in cents from the target over a window, by autocorrelation near the period
func windowCents(x []float64, from, target float64) float64 {
	pt := sr() / target
	w := int(0.04 * sr())
	lo, hi := int(pt*0.88), int(pt*1.12)
	s := int(from * sr())
	corr := make([]float64, hi+2)
	best, bestLag := -1e18, 0
	for lag := lo - 1; lag <= hi+1; lag++ {
		var c float64
		for i := 0; i < w; i++ {
			c += x[s+i] * x[s+i+lag]
		}
		corr[lag] = c
		if lag >= lo && lag <= hi && c > best {
			best, bestLag = c, lag
		}
	}
	a, b, c := corr[bestLag-1], corr[bestLag], corr[bestLag+1]
	lag := float64(bestLag)
	if d := a - 2*b + c; d != 0 {
		lag += 0.5 * (a - c) / d
	}
	return 1200 * math.Log2(pt/lag)
}

// a stick presses for a time: the skin's sound starts at nothing and rises, it does not begin with a
// step (which clicked, and was most of what a drum sounded like)
func TestDrumsDoNotStartWithAStep(t *testing.T) {
	for n, f := range Presets {
		if f().Model != Membrane {
			continue
		}
		for _, pitch := range []float64{45, 57, 69} {
			sameNoise()
			_, x := play(f(), pitch, 0.1)
			var peak float64
			for _, v := range x[:int(0.05*sr())] {
				peak = math.Max(peak, math.Abs(v))
			}
			if math.Abs(x[0]) > 0.02*peak {
				t.Errorf("%s %g: the first sample is %.3f against a peak of %.3f: a step", n, pitch, x[0], peak)
			}
		}
	}
}

// the pitch keeper trims the tube a little at a time: the length of a delay line that jumps is a click
func TestPipeLengthNeverJumps(t *testing.T) {
	for _, name := range []string{"bassClarinet", "clarinet", "chalumeau", "altoFlute", "flute", "panpipe", "shakuhachi"} {
		for _, pitch := range []float64{43, 55, 67} {
			sameNoise()
			v := New(Presets[name]())
			v.NoteOn(0.8, psalm.Message{"pitch": pitch, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
			last := v.pipe.trimNow
			buf := v.Config().BufferSize
			for b := 0; b < int(1.5*sr()/float64(buf)); b++ {
				v.Synthesize()
				if d := math.Abs(v.pipe.trimNow - last); d > trimSlew*float64(buf)*1.01 {
					t.Fatalf("%s %g: the tube changed by %.2f samples in a buffer", name, pitch, d)
				}
				last = v.pipe.trimNow
			}
		}
	}
}

// a reed speaks in a fifth of a second, low notes too (they took half a second, and sounded through
// pitches on the way up), and is in tune from then on
func TestReedsSpeakQuicklyAndInTune(t *testing.T) {
	for _, name := range pipePresets(true) {
		for _, pitch := range []float64{38, 45, 55, 62} {
			sameNoise()
			_, x := play(Presets[name](), pitch, 1.6)
			steady := rmsWin(x, 1.2, 1.5)
			speaks := -1.0
			for s := 0.0; s+0.02 < 1.5; s += 0.005 {
				if rmsWin(x, s, s+0.02) >= 0.9*steady {
					speaks = s
					break
				}
			}
			if speaks < 0 || speaks > 0.3 {
				t.Errorf("%s %g: speaks after %.2f s", name, pitch, speaks)
				continue
			}
			for at := math.Max(0.25, speaks+0.05); at < 0.9; at += 0.05 {
				if c := windowCents(x, at, hz(pitch)); math.Abs(c) > 30 {
					t.Errorf("%s %g: %.0f cents out at %.2f s", name, pitch, c, at)
					break
				}
			}
		}
	}
}

// a flute stays in the register it is asked for: not an octave up (overblown) and not dying away
func TestFlutesKeepTheirRegister(t *testing.T) {
	for _, name := range pipePresets(false) {
		for pitch := 57.0; pitch <= 90; pitch += 3 { // the flutes' own range: a piccolo below its first note is no flute
			sameNoise()
			_, x := play(Presets[name](), pitch, 2)
			var peak float64
			for _, v := range x[int(1.0*sr()):int(1.8*sr())] {
				peak = math.Max(peak, math.Abs(v))
			}
			if peak < 0.05 {
				t.Errorf("%s %g: dies away (peak %.3f after a second)", name, pitch, peak)
				continue
			}
			if c := cents(pitchOf(x[int(1.0*sr()):int(1.0*sr())+8192]), hz(pitch)); math.Abs(c) > 150 {
				t.Errorf("%s %g: sounds %+.0f cents off, a register", name, pitch, c)
			}
		}
	}
}

// the start of a note is not louder than the note: no burst while the tone builds up
func TestPipeStartDoesNotOvershoot(t *testing.T) {
	for _, name := range append(pipePresets(false), pipePresets(true)...) {
		for _, pitch := range []float64{57, 69} {
			sameNoise()
			_, x := play(Presets[name](), pitch, 1.4)
			var early, late float64
			for i, v := range x {
				switch a := math.Abs(v); {
				case i < int(0.3*sr()):
					early = math.Max(early, a)
				case i > int(0.8*sr()):
					late = math.Max(late, a)
				}
			}
			if late > 0.01 && early > 2*late {
				t.Errorf("%s %g: the start peaks at %.2f against %.2f later", name, pitch, early, late)
			}
		}
	}
}
