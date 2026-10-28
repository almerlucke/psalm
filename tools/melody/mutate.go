package melody

import (
	"math"
	"math/rand/v2"
	"slices"
)

// Mutate changes a melody a little: every note, with a chance of amount (0 leaves it as it is, 1 changes most of them), moves a step or two in the scale, repeats the note before it,
// goes silent, or is split in two (the second a step on); the notes that matter (on a beat, or long) are then fitted to the chord under them again, and the last note of the melody
// ends on the root. The same seed makes the same changes; what is left of the melody is what makes it still the melody
func Mutate(m []Note, p Params, amount float64, seed uint64) []Note {
	amount = math.Max(0, math.Min(amount, 1))
	if amount == 0 || len(m) == 0 {
		return slices.Clone(m)
	}
	p.defaults()
	g := &gen{p: p, rng: rand.New(rand.NewPCG(seed, seed*29+3))}
	for _, k := range p.Scale.Pitches(float64(p.Low), float64(p.High)) {
		g.pool = append(g.pool, int(k))
	}
	if len(g.pool) < 3 {
		return slices.Clone(m)
	}
	q := amount * 0.6
	var out []Note
	for i, n := range m {
		if g.rng.Float64() >= q {
			out = append(out, n)
			continue
		}
		switch x := g.rng.Float64(); {
		case x < 0.6: // a step or two
			j := g.nearest(n.Key) + []int{-2, -1, 1, 2}[g.rng.IntN(4)]
			n.Key = g.pool[max(0, min(j, len(g.pool)-1))]
			out = append(out, n)
		case x < 0.72 && i > 0: // the note before it again
			n.Key = m[i-1].Key
			out = append(out, n)
		case x < 0.84 && i < len(m)-1: // silent
		case n.Length >= 2*p.Grid: // split in two, the second a step on
			half := math.Round(n.Length/2/p.Grid) * p.Grid
			if half <= 0 || half >= n.Length {
				out = append(out, n)
				continue
			}
			second := n
			second.Start, second.Length = n.Start+half, n.Length-half
			second.Key = g.pool[max(0, min(g.nearest(n.Key)+[]int{-1, 1}[g.rng.IntN(2)], len(g.pool)-1))]
			n.Length = half * 0.97
			out = append(out, n, second)
		default:
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return slices.Clone(m)
	}
	ph := &phraseNotes{notes: out, length: p.Beats}
	g.fit(ph, true)
	slices.SortStableFunc(ph.notes, func(a, b Note) int {
		if a.Start < b.Start {
			return -1
		} else if a.Start > b.Start {
			return 1
		}
		return a.Key - b.Key
	})
	return ph.notes
}
