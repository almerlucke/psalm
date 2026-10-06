package harmony

import (
	"math/rand/v2"
	"slices"

	"github.com/almerlucke/psalm/modules/controls/score"
)

// Pattern is the order an arpeggio plays its notes in
type Pattern int

const (
	Up         Pattern = iota
	Down               //
	UpDown             // up and down again, without repeating the top and bottom
	DownUp             //
	OutsideIn          // lowest, highest, second lowest, second highest, ...
	RandomWalk         // random order, from a seed so it is the same every time
)

// Arp describes an arpeggio
type Arp struct {
	Pattern    Pattern
	Octaves    int       // the chord repeated in this many octaves (0 is 1)
	Steps      int       // number of notes, 0 is one pass through the pattern
	Beats      float64   // length of every note (0 is 0.5, 8ths)
	Hold       float64   // beats every note sounds, 0 is the legato of the part
	Velocities []float64 // velocities in turn, for accents
	Seed       uint64    // for RandomWalk
}

// Notes returns the arpeggio of a chord's pitches as notes
func (a Arp) Notes(pitches []float64) []score.Note {
	if len(pitches) == 0 {
		return nil
	}
	octaves := max(1, a.Octaves)
	beats := a.Beats
	if beats == 0 {
		beats = 0.5
	}

	base := slices.Clone(pitches)
	slices.Sort(base)
	var tones []float64
	for o := range octaves {
		for _, p := range base {
			tones = append(tones, p+float64(12*o))
		}
	}

	var order []float64
	switch a.Pattern {
	case Down:
		order = reversed(tones)
	case UpDown:
		order = append(slices.Clone(tones), reversed(tones)[1:max(1, len(tones)-1)]...)
	case DownUp:
		order = append(reversed(tones), tones[1:max(1, len(tones)-1)]...)
	case OutsideIn:
		for lo, hi := 0, len(tones)-1; lo <= hi; lo, hi = lo+1, hi-1 {
			order = append(order, tones[lo])
			if hi != lo {
				order = append(order, tones[hi])
			}
		}
	case RandomWalk:
		r := rand.New(rand.NewPCG(a.Seed, 0x5bd1e995))
		n := a.Steps
		if n == 0 {
			n = len(tones)
		}
		for range n {
			order = append(order, tones[r.IntN(len(tones))])
		}
	default:
		order = tones
	}

	steps := a.Steps
	if steps == 0 {
		steps = len(order)
	}
	notes := make([]score.Note, steps)
	for i := range steps {
		notes[i] = score.Note{Pitches: []float64{order[i%len(order)]}, Beats: beats, Hold: a.Hold}
		if len(a.Velocities) > 0 {
			notes[i].Velocity = a.Velocities[i%len(a.Velocities)]
		}
	}
	return notes
}

func reversed(x []float64) []float64 {
	r := slices.Clone(x)
	slices.Reverse(r)
	return r
}
