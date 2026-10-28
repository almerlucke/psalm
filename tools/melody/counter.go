package melody

import (
	"math"
	"slices"

	"github.com/almerlucke/psalm/tools/harmony"
)

// CounterParams describe a counter-line: a second voice for a melody
type CounterParams struct {
	Scale    harmony.Scale
	Chords   []Span  // the chords under the melody (none: the scale only)
	Low      int     // the lowest key of the voice (48)
	High     int     // the highest (72)
	Every    float64 // a note every so many beats (0: one under every note of the melody, 1: a note a beat, 2, 4...)
	Motion   float64 // 0 any to 1 contrary to the melody wherever it can (0.6)
	Steps    float64 // 0 leaps to 1 steps (0.7)
	Below    bool    // keep under the melody (the default is also under: it is a bass-ish voice; false lets it cross)
	Velocity float64
}

// Counter makes a second voice for a melody by dynamic programming over the notes it plays: every note a choice of the keys of the scale in its range, scored by how well it sounds
// with the melody note it falls under (thirds and sixths and tenths best, a perfect fifth or octave allowed, seconds, sevenths and tritones only passing), by being a tone of the
// chord on a strong beat, by moving by step, by moving against the melody, and never moving in parallel fifths or octaves, never crossing above the melody
func Counter(m []Note, p CounterParams) []Note {
	if len(m) == 0 {
		return nil
	}
	if p.High <= p.Low {
		p.Low, p.High = 48, 72
	}
	if p.Motion == 0 {
		p.Motion = 0.6
	}
	if p.Steps == 0 {
		p.Steps = 0.7
	}
	if p.Velocity == 0 {
		p.Velocity = 0.7
	}
	var pool []int
	for _, k := range p.Scale.Pitches(float64(p.Low), float64(p.High)) {
		pool = append(pool, int(k))
	}
	if len(pool) < 3 {
		return nil
	}
	// the slots of the counter-line: where it plays and the melody note it sits under
	type slot struct {
		at, len float64
		mel     int // the key of the melody there
		chord   *Span
	}
	melodyAt := func(t float64) int {
		best, key := math.Inf(1), m[0].Key
		for _, n := range m {
			if t >= n.Start-1e-9 && t < n.Start+n.Length-1e-9 {
				return n.Key
			}
			if d := math.Abs(n.Start - t); d < best {
				best, key = d, n.Key
			}
		}
		return key
	}
	chordAt := func(t float64) *Span {
		for i := range p.Chords {
			if t >= p.Chords[i].At-1e-9 && t < p.Chords[i].At+p.Chords[i].Len-1e-9 {
				return &p.Chords[i]
			}
		}
		return nil
	}
	var slots []slot
	end := m[len(m)-1].Start + m[len(m)-1].Length
	if p.Every <= 0 {
		for i, n := range m {
			l := n.Length
			if i+1 < len(m) {
				l = m[i+1].Start - n.Start
			}
			slots = append(slots, slot{at: n.Start, len: math.Max(l, 0.05), mel: n.Key, chord: chordAt(n.Start)})
		}
	} else {
		for at := m[0].Start; at < end-1e-9; at += p.Every {
			slots = append(slots, slot{at: at, len: math.Min(p.Every, end-at), mel: melodyAt(at), chord: chordAt(at)})
		}
	}
	if len(slots) == 0 {
		return nil
	}
	has := func(c *Span, key int) bool { return c != nil && slices.Contains(c.Pcs, ((key%12)+12)%12) }
	// the cost of a key under a melody note: lower is better
	vertical := func(s slot, key int) float64 {
		iv := ((s.mel-key)%12 + 12) % 12
		cost := 0.0
		switch iv {
		case 3, 4, 8, 9: // thirds and sixths
		case 0, 7:
			cost += 1.2
		case 5:
			cost += 2
		default: // seconds, sevenths, the tritone
			cost += 4
		}
		if key >= s.mel-2 { // not under the melody, or in unison with it
			cost += 6
		}
		if s.mel-key > 19 { // too far from it
			cost += 2
		}
		if !has(s.chord, key) {
			if strong(s.at, s.len) {
				cost += 3
			} else {
				cost += 0.8
			}
		}
		return cost
	}
	// the cost of a move between two keys, while the melody moves from a to b
	motion := func(prev, key, ma, mb int) float64 {
		cost := 0.0
		d := key - prev
		step := 0
		if d != 0 {
			step = 1
			if d < 0 {
				step = -1
			}
		}
		leap := math.Abs(float64(d))
		cost += (1 - p.Steps) * 0.3 * leap
		cost += p.Steps * 0.8 * math.Max(0, leap-2)
		md := mb - ma
		ms := 0
		if md > 0 {
			ms = 1
		} else if md < 0 {
			ms = -1
		}
		if step != 0 && ms != 0 && step == ms {
			cost += 1.5 * p.Motion // similar motion
		}
		// parallel perfect intervals: the same perfect interval class before and after, both moving
		ia, ib := ((ma-prev)%12+12)%12, ((mb-key)%12+12)%12
		if (ia == 0 || ia == 7) && ia == ib && step != 0 && ms != 0 {
			cost += 40
		}
		// hidden perfect: to a perfect interval in similar motion
		if (ib == 0 || ib == 7) && step == ms && step != 0 && math.Abs(float64(md)) > 2 {
			cost += 3
		}
		return cost
	}
	n, k := len(slots), len(pool)
	cost := make([][]float64, n)
	back := make([][]int, n)
	for i := range slots {
		cost[i], back[i] = make([]float64, k), make([]int, k)
		for j, key := range pool {
			v := vertical(slots[i], key)
			if i == 0 {
				cost[i][j] = v
				continue
			}
			best, from := math.Inf(1), 0
			for q, pk := range pool {
				c := cost[i-1][q] + motion(pk, key, slots[i-1].mel, slots[i].mel)
				if c < best {
					best, from = c, q
				}
			}
			cost[i][j], back[i][j] = best+v, from
		}
	}
	at := 0
	for j := range pool {
		if cost[n-1][j] < cost[n-1][at] {
			at = j
		}
	}
	keys := make([]int, n)
	for i := n - 1; i >= 0; i-- {
		keys[i] = pool[at]
		at = back[i][at]
	}
	out := make([]Note, n)
	for i, s := range slots {
		out[i] = Note{Start: s.at, Length: s.len * 0.97, Key: keys[i], Velocity: p.Velocity}
	}
	return out
}
