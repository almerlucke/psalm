package melody

import (
	"math"
	"slices"
	"testing"

	"github.com/almerlucke/psalm/tools/harmony"
)

var dMinor = harmony.MustScale("D", "minor")

// the chords of a bar each: Dm Bb Gm A
func chords(bars int) []Span {
	cs := []string{"Dm", "Bb", "Gm", "A"}
	var out []Span
	for b := range bars {
		c := harmony.MustChord(cs[b%4])
		out = append(out, Span{At: float64(b * 4), Len: 4, Pcs: c.PitchClasses(), Root: c.Root})
	}
	return out
}

func avg(f func(seed uint64) float64) float64 {
	sum := 0.0
	for seed := uint64(1); seed <= 60; seed++ {
		sum += f(seed)
	}
	return sum / 60
}

// every melody stays in the range and in the scale, begins at 0 and ends inside the stretch, in order; the same seed makes the same melody
func TestBounds(t *testing.T) {
	for _, form := range []string{"free", "AA'", "ABAB", "AABA", "call"} {
		for _, rhythm := range []string{"mixed", "even", "syncopated", "dotted", "triplet"} {
			for seed := uint64(1); seed <= 8; seed++ {
				p := Params{Scale: dMinor, Beats: 16, Low: 57, High: 81, Chords: chords(4), Form: form, Rhythm: rhythm, Seed: seed}
				notes := Generate(p)
				if len(notes) == 0 {
					t.Fatalf("%s %s: no notes", form, rhythm)
				}
				for i, n := range notes {
					if n.Key < 57 || n.Key > 81 || !dMinor.Contains(float64(n.Key)) {
						t.Fatalf("%s %s: key %d is out of the range or the scale", form, rhythm, n.Key)
					}
					if n.Start < 0 || n.Start >= 16 || n.Start+n.Length > 16+1e-6 || n.Length <= 0 {
						t.Fatalf("%s %s: note %v is outside the stretch", form, rhythm, n)
					}
					if n.Velocity <= 0 || n.Velocity > 1 {
						t.Fatalf("velocity %v", n.Velocity)
					}
					if i > 0 && n.Start < notes[i-1].Start {
						t.Fatalf("not in order: %v", notes)
					}
				}
				if again := Generate(p); !slices.Equal(notes, again) {
					t.Fatalf("%s %s seed %d: not the same twice", form, rhythm, seed)
				}
			}
		}
	}
	a := Generate(Params{Scale: dMinor, Seed: 1})
	b := Generate(Params{Scale: dMinor, Seed: 2})
	if slices.Equal(a, b) {
		t.Error("two seeds, one melody")
	}
}

// with the chord bias up, the notes on the beat are tones of the chord; with it down, fewer are
func TestChordTones(t *testing.T) {
	share := func(bias float64) float64 {
		return avg(func(seed uint64) float64 {
			notes := Generate(Params{Scale: dMinor, Beats: 16, Chords: chords(4), Chord: bias, Seed: seed})
			cs := chords(4)
			on, in := 0, 0
			for _, n := range notes {
				if math.Abs(n.Start-math.Round(n.Start)) < 1e-6 {
					on++
					for _, c := range cs {
						if n.Start >= c.At && n.Start < c.At+c.Len && slices.Contains(c.Pcs, n.Key%12) {
							in++
						}
					}
				}
			}
			if on == 0 {
				return 1
			}
			return float64(in) / float64(on)
		})
	}
	hi, lo := share(1), share(0.02)
	if hi < 0.85 {
		t.Errorf("with the chords all the way up %.2f of the notes on a beat are tones of the chord", hi)
	}
	if lo > hi-0.15 {
		t.Errorf("chord bias 1: %.2f, 0: %.2f; want a clear difference", hi, lo)
	}
}

// the melody ends on the root of the chord it ends in
func TestEndsOnTheRoot(t *testing.T) {
	ok := 0
	for seed := uint64(1); seed <= 60; seed++ {
		notes := Generate(Params{Scale: dMinor, Beats: 16, Chords: chords(4), Form: "AABA", Seed: seed})
		last := notes[len(notes)-1]
		if last.Key%12 == harmony.MustChord("A").Root { // the chord of the last bar
			ok++
		}
	}
	if ok < 55 {
		t.Errorf("the last note is the root %d of 60 times", ok)
	}
}

// a busier melody has more notes, rests thin it out, steps keep the line smooth, the triplets are on the third of a beat
func TestSettings(t *testing.T) {
	count := func(p Params) float64 {
		return avg(func(seed uint64) float64 { p.Seed = seed; return float64(len(Generate(p))) })
	}
	sparse := count(Params{Scale: dMinor, Beats: 16, Density: 0.1, Rest: 0.05})
	busy := count(Params{Scale: dMinor, Beats: 16, Density: 0.95, Rest: 0.05})
	if busy < 1.8*sparse {
		t.Errorf("sparse %.1f notes, busy %.1f", sparse, busy)
	}
	few := count(Params{Scale: dMinor, Beats: 16, Rest: 0.6})
	many := count(Params{Scale: dMinor, Beats: 16, Rest: 0.02})
	if few > 0.8*many {
		t.Errorf("rests: %.1f notes against %.1f", few, many)
	}
	jump := func(steps float64) float64 {
		return avg(func(seed uint64) float64 {
			notes := Generate(Params{Scale: dMinor, Beats: 16, Steps: steps, Chord: 0.01, Seed: seed})
			sum := 0.0
			for i := 1; i < len(notes); i++ {
				sum += math.Abs(float64(notes[i].Key - notes[i-1].Key))
			}
			return sum / float64(max(1, len(notes)-1))
		})
	}
	if smooth, leaps := jump(1), jump(0.05); smooth > leaps-0.8 {
		t.Errorf("average interval: %.2f with steps, %.2f with leaps", smooth, leaps)
	}
	for _, n := range Generate(Params{Scale: dMinor, Beats: 8, Rhythm: "triplet", Seed: 4}) {
		if x := n.Start * 3; math.Abs(x-math.Round(x)) > 0.01 {
			t.Errorf("a triplet note at %v", n.Start)
		}
	}
}

// the contour: an arch has its highest note in the middle, a fall begins higher than it ends, a rise the other way
func TestContour(t *testing.T) {
	mid := 0
	for seed := uint64(1); seed <= 60; seed++ {
		notes := Generate(Params{Scale: dMinor, Beats: 8, Phrase: 8, Contour: "arch", Chord: 0.01, Density: 0.7, Seed: seed})
		top := notes[0]
		for _, n := range notes {
			if n.Key > top.Key {
				top = n
			}
		}
		if top.Start >= 2 && top.Start <= 6 {
			mid++
		}
	}
	if mid < 42 {
		t.Errorf("the arch peaks in the middle %d of 60 times", mid)
	}
	diff := func(shape string) float64 {
		return avg(func(seed uint64) float64 {
			notes := Generate(Params{Scale: dMinor, Beats: 8, Phrase: 8, Contour: shape, Chord: 0.01, Density: 0.7, Seed: seed})
			h := len(notes) / 2
			var a, b float64
			for i, n := range notes {
				if i < h {
					a += float64(n.Key) / float64(h)
				} else {
					b += float64(n.Key) / float64(len(notes)-h)
				}
			}
			return b - a
		})
	}
	if rise, fall := diff("rise"), diff("fall"); rise < 2 || fall > -2 {
		t.Errorf("the second half is %.1f above the first for a rise, %.1f for a fall", rise, fall)
	}
}

// the forms repeat: A B A B has the third phrase as the first and the fourth as the second; in a period the second phrase begins as the first does
func TestForms(t *testing.T) {
	notes := Generate(Params{Scale: dMinor, Beats: 16, Form: "ABAB", Seed: 7})
	phrase := func(from float64) []Note {
		var out []Note
		for _, n := range notes {
			if n.Start >= from && n.Start < from+4 {
				n.Start -= from
				out = append(out, n)
			}
		}
		return out
	}
	// the velocities are drawn again, the notes and the rhythm are the same
	same := func(a, b []Note) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i].Start != b[i].Start || a[i].Key != b[i].Key || a[i].Length != b[i].Length {
				return false
			}
		}
		return true
	}
	if !same(phrase(0), phrase(8)) || !same(phrase(4), phrase(12)) {
		t.Errorf("A B A B: %v", notes)
	}
	if same(phrase(0), phrase(4)) {
		t.Error("A and B are the same phrase")
	}
	// a period: the second phrase has something of the first (a variation of it)
	shared := 0
	for seed := uint64(1); seed <= 40; seed++ {
		n := Generate(Params{Scale: dMinor, Beats: 8, Form: "AA'", Seed: seed})
		a, b := map[float64]int{}, map[float64]int{}
		for _, x := range n {
			if x.Start < 4 {
				a[x.Start] = x.Key
			} else {
				b[x.Start-4] = x.Key
			}
		}
		hit := 0
		for s := range a {
			if _, ok := b[s]; ok {
				hit++
			}
		}
		if hit*2 >= len(a) {
			shared++
		}
	}
	if shared < 24 {
		t.Errorf("the second phrase of a period shares half its beginning with the first %d of 40 times", shared)
	}
}

// a stretch that is shorter than a phrase is one phrase; a range that is too narrow makes no melody
func TestShort(t *testing.T) {
	notes := Generate(Params{Scale: dMinor, Beats: 2, Form: "AABA", Seed: 1})
	if len(notes) == 0 {
		t.Fatal("no notes in two beats")
	}
	for _, n := range notes {
		if n.Start+n.Length > 2+1e-6 {
			t.Errorf("note %v is past the end", n)
		}
	}
	// a range that is too narrow is made a major third wide; a scale with nothing in it makes no melody
	for _, n := range Generate(Params{Scale: dMinor, Low: 60, High: 61, Seed: 1}) {
		if n.Key < 60 || n.Key > 64 {
			t.Errorf("key %d out of the range made of 60 to 61", n.Key)
		}
	}
	if got := Generate(Params{Scale: harmony.MustScale("C", "major pentatonic"), Low: 61, High: 62, Seed: 1}); got != nil {
		t.Errorf("a melody in a range with no scale notes: %v", got)
	}
}

func TestMutate(t *testing.T) {
	scale, _ := harmony.NewScale("C", "major")
	chords := []Span{{At: 0, Len: 4, Pcs: []int{0, 4, 7}, Root: 0}, {At: 4, Len: 4, Pcs: []int{5, 9, 0}, Root: 5}, {At: 8, Len: 4, Pcs: []int{7, 11, 2}, Root: 7}, {At: 12, Len: 4, Pcs: []int{0, 4, 7}, Root: 0}}
	p := Params{Scale: scale, Beats: 16, Low: 60, High: 84, Chords: chords, Form: "AA'", Phrase: 8, Seed: 4}
	m := Generate(p)
	if got := Mutate(m, p, 0, 1); !slices.Equal(got, m) {
		t.Error("a mutation of nothing changed the melody")
	}
	a, b := Mutate(m, p, 0.3, 5), Mutate(m, p, 0.3, 5)
	if !slices.Equal(a, b) {
		t.Error("not repeatable")
	}
	if slices.Equal(a, m) || slices.Equal(Mutate(m, p, 0.3, 6), a) {
		t.Error("a mutation changes nothing, or the same every seed")
	}
	// most of it stays: the same pitch at the same place for most of the notes
	same := 0
	for _, n := range a {
		for _, o := range m {
			if o.Start == n.Start && o.Key == n.Key {
				same++
				break
			}
		}
	}
	if float64(same)/float64(len(m)) < 0.45 {
		t.Errorf("only %d of %d notes are as they were", same, len(m))
	}
	// in the scale, in the range, on the roots at the end; the notes on a beat are tones of the chord
	inScale := map[int]bool{0: true, 2: true, 4: true, 5: true, 7: true, 9: true, 11: true}
	for _, n := range a {
		if !inScale[n.Key%12] || n.Key < 60 || n.Key > 84 || n.Start >= 16 {
			t.Fatalf("note %+v out of scale or range", n)
		}
	}
	if last := a[len(a)-1]; last.Key%12 != 0 {
		t.Errorf("the last note %d is not the root", last.Key)
	}
}

// dynamics: the notes differ in loudness (a spread of the velocities that grows with it), the beats louder than the notes between
func TestMelodyDynamics(t *testing.T) {
	scale, _ := harmony.NewScale("C", "major")
	spread := func(d float64) (lo, hi, std float64) {
		var vs []float64
		for seed := uint64(1); seed <= 20; seed++ {
			for _, n := range Generate(Params{Scale: scale, Beats: 32, Phrase: 8, Velocity: 0.5, Dynamics: d, Seed: seed}) {
				vs = append(vs, n.Velocity)
			}
		}
		lo, hi = 1, 0
		mean := 0.0
		for _, v := range vs {
			lo, hi, mean = min(lo, v), max(hi, v), mean+v
		}
		mean /= float64(len(vs))
		for _, v := range vs {
			std += (v - mean) * (v - mean)
		}
		return lo, hi, math.Sqrt(std / float64(len(vs)))
	}
	_, _, flat := spread(0)
	lo, hi, lively := spread(1)
	if lively < 2*flat || hi-lo < 0.35 {
		t.Errorf("dynamics 1: std %.3f against %.3f, spread %.2f to %.2f: the notes do not differ enough", lively, flat, lo, hi)
	}
	if lo < 0.1-1e-9 || hi > 1+1e-9 {
		t.Errorf("velocities %v to %v are out of range", lo, hi)
	}
}
