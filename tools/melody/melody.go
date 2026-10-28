// Package melody makes melodies: notes for a stretch of beats, in a scale and a range, that follow the chords under them. A melody goes in phrases that repeat and vary (a period, A B A B,
// A A' B A, a call and its answer) because a melody without repetition does not sound written; each phrase has a rhythm (even, mixed, syncopated, dotted, triplets, as busy as the density
// says, with rests) and a line of pitches that moves by steps mostly and by leaps now and then, goes back the other way after a big leap, keeps to a contour (an arch, a rise, a fall,
// a wave), puts the tones of the chord on the strong beats and the long notes, and ends on a tone of the chord. The same seed makes the same melody.
//
//	notes := melody.Generate(melody.Params{Scale: harmony.MustScale("D", "minor"), Beats: 8, Chords: spans, Form: "AABA", Seed: 3})
package melody

import (
	"math"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/almerlucke/psalm/tools/harmony"
)

// Note is a note of a melody: where it starts and how long it lasts (beats from the start of the stretch), its key (MIDI) and its velocity (0-1)
type Note struct {
	Start, Length float64
	Key           int
	Velocity      float64
}

// Span is a chord over a part of the stretch: where it starts and how long it lasts (beats), the pitch classes of its tones and its root
type Span struct {
	At, Len float64
	Pcs     []int
	Root    int
}

// Params describe a melody; zero values are the defaults
type Params struct {
	Scale    harmony.Scale
	Beats    float64 // how long the stretch is (8)
	Low      int     // the lowest key (60)
	High     int     // the highest (84)
	Chords   []Span  // the chords under it (none: the scale only)
	Grid     float64 // the shortest note in beats (0.25; 1/3 for the triplets)
	Density  float64 // 0 sparse to 1 busy (0.5)
	Rest     float64 // the chance a note is a rest (0.15)
	Rhythm   string  // "mixed" (the default), "even", "syncopated", "dotted", "triplet"
	Steps    float64 // 0 leaps a lot to 1 moves by steps (0.7)
	Chord    float64 // 0 to 1: how much the strong beats and the long notes keep to the tones of the chord (0.7)
	Contour  string  // "free" (the default), "arch", "rise", "fall", "wave": the shape of the line over a phrase
	Form     string  // "free" (the default: one phrase), "AA'" (a period), "ABAB", "AABA", "call" (a question and its answer)
	Grammar  string  // rules that write the form (see Expand): when there are some they are the form and Form is not used
	Phrase   float64 // how long a phrase is in beats (4)
	Velocity float64 // the loudness of a note, accents on top (0.8)
	Seed     uint64
}

func (p *Params) defaults() {
	if p.Beats <= 0 {
		p.Beats = 8
	}
	if p.Low == 0 && p.High == 0 {
		p.Low, p.High = 60, 84
	}
	if p.High < p.Low+4 {
		p.High = p.Low + 4
	}
	if p.Grid <= 0 {
		p.Grid = 0.25
	}
	if p.Density == 0 {
		p.Density = 0.5
	}
	if p.Steps == 0 {
		p.Steps = 0.7
	}
	if p.Chord == 0 {
		p.Chord = 0.7
	}
	if p.Phrase <= 0 {
		p.Phrase = 4
	}
	if p.Velocity <= 0 {
		p.Velocity = 0.8
	}
	if p.Rest == 0 {
		p.Rest = 0.15
	}
	if p.Rhythm == "triplet" && p.Grid > 1.0/3 {
		p.Grid = 1.0 / 3
	}
	p.Density = math.Max(0, math.Min(p.Density, 1))
	p.Rest = math.Max(0, math.Min(p.Rest, 0.8))
	p.Steps = math.Max(0, math.Min(p.Steps, 1))
	p.Chord = math.Max(0, math.Min(p.Chord, 1))
}

// slot is a note to be: its place in a phrase, how long, and whether it is a rest
type slot struct {
	at, len float64
	rest    bool
}

type gen struct {
	p    Params
	rng  *rand.Rand
	pool []int // the keys of the scale in the range
}

// Generate makes a melody for a stretch
func Generate(p Params) []Note {
	p.defaults()
	g := &gen{p: p, rng: rand.New(rand.NewPCG(p.Seed, p.Seed*13+5))}
	for _, k := range p.Scale.Pitches(float64(p.Low), float64(p.High)) {
		g.pool = append(g.pool, int(k))
	}
	if len(g.pool) < 3 {
		return nil
	}
	phrase := math.Min(p.Phrase, p.Beats)
	count := int(math.Ceil(p.Beats/phrase - 1e-9))
	var toks []Token
	if strings.TrimSpace(p.Grammar) != "" {
		toks, _ = Expand(p.Grammar, g.rng, count)
	}
	if len(toks) == 0 {
		for _, k := range g.plan(p.Form, count) {
			t := Token{Letter: k &^ 0x20}
			if k >= 'a' {
				t.Ops = "'"
			}
			toks = append(toks, t)
		}
		if p.Form == "" || p.Form == "free" {
			phrase = p.Beats
		}
	}
	kinds := toks
	var out []Note
	var last *phraseNotes // the phrase A and B to repeat or vary
	byKind := map[byte]*phraseNotes{}
	prevEnd := -1
	for i, kind := range kinds {
		off := float64(i) * phrase
		length := math.Min(phrase, p.Beats-off)
		if length < p.Grid-1e-9 {
			break
		}
		var ph *phraseNotes
		src := byKind[kind.Letter] // A and A' (its variation) come from A
		end := g.endFor(p.Form, i, len(kinds))
		if strings.TrimSpace(p.Grammar) != "" {
			end = ""
			if i == len(kinds)-1 {
				end = "closed"
			} else if strings.Contains(kind.Ops, "?") {
				end = "open"
			}
		}
		final := i == len(kinds)-1
		switch {
		case src == nil: // a phrase of its own
			ph = g.phrase(off, length, prevEnd, final, end)
			byKind[kind.Letter] = ph
		case strings.Contains(kind.Ops, "'"): // the variation of A
			ph = g.vary(src, off, length, final, end)
		default: // A again
			ph = g.repeat(src, off, length, final)
		}
		g.operate(ph, kind.Ops, final)
		last = ph
		if n := len(ph.notes); n > 0 {
			prevEnd = ph.notes[n-1].Key
		}
		out = append(out, ph.notes...)
	}
	_ = last
	// the notes inside the stretch, in order
	out = slices.DeleteFunc(out, func(n Note) bool { return n.Start >= p.Beats-1e-9 })
	for i := range out {
		out[i].Length = math.Max(0.05, math.Min(out[i].Length, p.Beats-out[i].Start))
		out[i].Start = math.Round(out[i].Start*1000) / 1000
		out[i].Length = math.Round(out[i].Length*1000) / 1000
	}
	slices.SortStableFunc(out, func(a, b Note) int {
		if a.Start < b.Start {
			return -1
		} else if a.Start > b.Start {
			return 1
		}
		return a.Key - b.Key
	})
	return out
}

// plan is the kind of every phrase of a form: A and B are phrases of their own, a is a variation of A
func (g *gen) plan(form string, n int) []byte {
	var kinds []byte
	pattern := map[string]string{"AA'": "Aa", "ABAB": "ABAB", "AABA": "AaBA", "call": "Aa"}[form]
	if pattern == "" {
		return []byte{'A'}
	}
	for i := range n {
		kinds = append(kinds, pattern[i%len(pattern)])
	}
	return kinds
}

// endFor is how a phrase ends: "open" (a question: on the third or fifth of the chord), "closed" (on the root), "" (a tone of the chord)
func (g *gen) endFor(form string, i, n int) string {
	switch {
	case i == n-1:
		return "closed"
	case form == "call" && i%2 == 0:
		return "open"
	}
	return ""
}

// phraseNotes is a phrase as made: its notes (in the stretch, from the offset of the phrase), and what it was made of, to repeat or vary it
type phraseNotes struct {
	notes  []Note
	slots  []slot
	length float64
	off    float64
}

// rhythm lays the notes of a phrase: how long each lasts, filling the length
func (g *gen) rhythm(length float64) []slot {
	p := g.p
	d := p.Density
	type choice struct{ len, w float64 }
	var cs []choice
	switch p.Rhythm {
	case "even":
		if d > 0.6 {
			cs = []choice{{0.5, 6}, {0.25, d * 2}, {1, 1}}
		} else if d > 0.3 {
			cs = []choice{{0.5, 4}, {1, 3}, {2, 0.5}}
		} else {
			cs = []choice{{1, 5}, {2, 2}, {0.5, 1}}
		}
	case "dotted":
		cs = []choice{{0.75, 4}, {0.25, 3}, {1.5, 2 * (1.2 - d)}, {0.5, 2}, {1, 1}}
	case "triplet":
		cs = []choice{{1.0 / 3, 3 * d}, {2.0 / 3, 3}, {1, 2 * (1.1 - d)}, {2, 0.6 * (1 - d)}}
	case "syncopated":
		cs = []choice{{0.75, 4}, {0.5, 3 + 2*d}, {1.5, 2}, {0.25, 1 + 2*d}, {1, 1}, {2, 0.3}}
	default:
		cs = []choice{{0.25, 3 * d * d}, {0.5, 2 + 3*d}, {0.75, 0.5}, {1, 3 - 2*d}, {1.5, 1.2 * (1 - d)}, {2, 1.6 * (1 - d)}, {3, 0.3 * (1 - d)}}
	}
	cs = slices.DeleteFunc(cs, func(c choice) bool { return c.w <= 0 || c.len < p.Grid-1e-9 })
	if len(cs) == 0 {
		cs = []choice{{math.Max(p.Grid, 0.5), 1}}
	}
	total := 0.0
	for _, c := range cs {
		total += c.w
	}
	var slots []slot
	for at := 0.0; at < length-1e-9; {
		x := g.rng.Float64() * total
		l := cs[len(cs)-1].len
		for _, c := range cs {
			if x < c.w {
				l = c.len
				break
			}
			x -= c.w
		}
		l = math.Min(l, length-at)
		if l < p.Grid-1e-9 && len(slots) > 0 { // what is left is too short for a note: the one before takes it
			slots[len(slots)-1].len += l
			break
		}
		rest := len(slots) > 0 && !slots[len(slots)-1].rest && g.rng.Float64() < p.Rest && length-at-l > 0.1
		slots = append(slots, slot{at: at, len: l, rest: rest})
		at += l
	}
	return slots
}

// chordAt is the chord at a time of the stretch (nil: none)
func (g *gen) chordAt(t float64) *Span {
	for i := range g.p.Chords {
		c := &g.p.Chords[i]
		if t >= c.At-1e-9 && t < c.At+c.Len-1e-9 {
			return c
		}
	}
	return nil
}

func (s *Span) has(key int) bool { return s != nil && slices.Contains(s.Pcs, ((key%12)+12)%12) }

// strong says whether a note matters: it is on a beat, or it lasts a beat or more
func strong(at, length float64) bool {
	return math.Abs(at-math.Round(at)) < 1e-6 || length >= 1-1e-9
}

// contourAt is the height (0 to 1) the line aims for at a part of the phrase (0 to 1)
func contourAt(shape string, x float64) float64 {
	switch shape {
	case "arch":
		return 0.15 + 0.8*math.Sin(math.Pi*x)
	case "rise":
		return 0.1 + 0.8*x
	case "fall":
		return 0.9 - 0.8*x
	case "wave":
		return 0.5 + 0.35*math.Sin(4*math.Pi*x)
	}
	return -1
}

// phrase makes a phrase of its own: a rhythm, and a line of pitches for it
func (g *gen) phrase(off, length float64, prevKey int, final bool, end string) *phraseNotes {
	slots := g.rhythm(length)
	ph := &phraseNotes{slots: slots, length: length, off: off}
	ph.notes = g.line(slots, off, length, prevKey, end)
	return ph
}

// line chooses the pitches of the notes of a rhythm
func (g *gen) line(slots []slot, off, length float64, prevKey int, end string) []Note {
	p := g.p
	pool := g.pool
	height := func(key int) float64 { return float64(key-p.Low) / float64(max(1, p.High-p.Low)) }
	// where it starts: the nearest tone of the chord to where the last phrase ended, or the middle of the range
	cur := len(pool) / 2
	if prevKey >= 0 {
		cur = g.nearest(prevKey)
	} else if c := g.chordAt(off); c != nil {
		best := math.MaxFloat64
		for j, k := range pool {
			if c.has(k) && math.Abs(float64(k-(p.Low+p.High)/2)) < best {
				best, cur = math.Abs(float64(k-(p.Low+p.High)/2)), j
			}
		}
	}
	var notes []Note
	lastDir, lastStep := 0, 0
	firstNote := true
	lastNote := -1
	for i, s := range slots {
		if !s.rest {
			lastNote = i
		}
	}
	for i, s := range slots {
		if s.rest {
			continue
		}
		at := off + s.at
		isLast := i == lastNote
		c := g.chordAt(at)
		str := strong(s.at, s.len)
		weights := make([]float64, len(pool))
		total := 0.0
		for j := range pool {
			step := j - cur
			w := g.stepWeight(step, lastDir, lastStep)
			if firstNote && prevKey < 0 {
				w = 1
				if j != cur {
					w = 0.15
				}
			}
			if c != nil {
				switch {
				case c.has(pool[j]):
					w *= 1 + 5*p.Chord
					if !str {
						w *= 0.6 // a weak note is as likely a passing note
						w += 0
					}
				case str:
					w *= math.Max(0.04, 1-0.9*p.Chord)
				}
			}
			if t := contourAt(p.Contour, (s.at+s.len/2)/math.Max(length, 1e-6)); t >= 0 {
				d := (height(pool[j]) - t) * float64(p.High-p.Low)
				w *= math.Exp(-d * d / (2 * 6 * 6))
			}
			if isLast {
				switch {
				case end == "closed" && c != nil && int(((pool[j]%12)+12)%12) == ((c.Root%12)+12)%12:
					w *= 8
				case end == "open" && c != nil && c.has(pool[j]) && ((pool[j]%12)+12)%12 != ((c.Root%12)+12)%12:
					w *= 8
				case c != nil && !c.has(pool[j]):
					w *= 0.05
				}
			}
			weights[j] = w
			total += w
		}
		next := cur
		if total > 0 {
			x := g.rng.Float64() * total
			for j, w := range weights {
				if x < w {
					next = j
					break
				}
				x -= w
			}
		}
		lastStep = next - cur
		switch {
		case lastStep > 0:
			lastDir = 1
		case lastStep < 0:
			lastDir = -1
		}
		cur = next
		firstNote = false
		notes = append(notes, Note{Start: at, Length: s.len * 0.97, Key: pool[cur], Velocity: g.velocity(s, i == 0)})
	}
	return notes
}

// stepWeight is how likely a move of some steps through the scale is, after the last move
func (g *gen) stepWeight(step, lastDir, lastStep int) float64 {
	s := g.p.Steps
	a := step
	if a < 0 {
		a = -a
	}
	var w float64
	switch a {
	case 0:
		w = 0.10 * (1.5 - s)
	case 1:
		w = 0.42*s + 0.15
	case 2:
		w = 0.22
	case 3:
		w = 0.10 * (1.3 - s)
	case 4:
		w = 0.06 * (1.2 - s)
	default:
		w = 0.03 * (1.2 - s) / float64(a-3)
	}
	if lastDir != 0 && step*lastDir > 0 {
		w *= 1.25 // it goes on the way it went
	}
	if ls := lastStep; (ls >= 3 || ls <= -3) && step*ls < 0 && a <= 2 {
		w *= 2.2 // after a leap it comes back
	}
	return math.Max(w, 0)
}

func (g *gen) velocity(s slot, first bool) float64 {
	v := g.p.Velocity
	if strong(s.at, s.len) {
		v += 0.06
	}
	if first {
		v += 0.04
	}
	v += (g.rng.Float64() - 0.5) * 0.08
	return math.Max(0.1, math.Min(v, 1))
}

// nearest is the place in the pool of the key nearest to a key
func (g *gen) nearest(key int) int {
	best, at := math.MaxInt, 0
	for j, k := range g.pool {
		d := k - key
		if d < 0 {
			d = -d
		}
		if d < best {
			best, at = d, j
		}
	}
	return at
}

// repeat is a phrase again, later, its strong notes fitted to the chord there
func (g *gen) repeat(src *phraseNotes, off, length float64, final bool) *phraseNotes {
	ph := &phraseNotes{slots: src.slots, length: length, off: off}
	for _, n := range src.notes {
		n.Start += off - src.off
		ph.notes = append(ph.notes, n)
	}
	g.fit(ph, final)
	return ph
}

// vary is a phrase that is a variation of another: a sequence (every note a step or two higher or lower), or the same beginning with another end, or the same pitches
// with another rhythm for one note
func (g *gen) vary(src *phraseNotes, off, length float64, final bool, end string) *phraseNotes {
	switch g.rng.IntN(3) {
	case 0: // a sequence
		ph := g.repeat(src, off, length, false)
		shift := []int{-2, -1, 1, 2}[g.rng.IntN(4)]
		for i := range ph.notes {
			j := g.nearest(ph.notes[i].Key) + shift
			j = max(0, min(j, len(g.pool)-1))
			ph.notes[i].Key = g.pool[j]
		}
		g.fit(ph, final)
		return ph
	case 1: // the beginning, another end
		keep := max(1, len(src.notes)*6/10)
		ph := &phraseNotes{slots: src.slots, length: length, off: off}
		for _, n := range src.notes[:min(keep, len(src.notes))] {
			n.Start += off - src.off
			ph.notes = append(ph.notes, n)
		}
		var rest []slot
		from := 0.0
		if len(ph.notes) > 0 {
			l := ph.notes[len(ph.notes)-1]
			from = l.Start + l.Length/0.97 - off
		}
		for _, s := range src.slots {
			if s.at >= from-1e-9 {
				s.at -= 0
				rest = append(rest, s)
			}
		}
		prev := -1
		if len(ph.notes) > 0 {
			prev = ph.notes[len(ph.notes)-1].Key
		}
		ph.notes = append(ph.notes, g.line(rest, off, length, prev, end)...)
		g.fit(ph, final)
		return ph
	}
	// the same line with a note split in two (the second a step on)
	ph := g.repeat(src, off, length, final)
	if len(ph.notes) > 1 {
		i := g.rng.IntN(len(ph.notes))
		n := ph.notes[i]
		if n.Length >= 2*g.p.Grid {
			half := math.Round(n.Length/2/g.p.Grid) * g.p.Grid
			if half > 0 && half < n.Length {
				second := n
				second.Start, second.Length = n.Start+half, n.Length-half
				j := max(0, min(g.nearest(n.Key)+[]int{-1, 1}[g.rng.IntN(2)], len(g.pool)-1))
				second.Key = g.pool[j]
				ph.notes[i].Length = half * 0.97
				ph.notes = slices.Insert(ph.notes, i+1, second)
				g.fit(ph, final)
			}
		}
	}
	return ph
}

// operate turns a phrase by the operators after its letter: ~ upside down (around its first note, in the scale), < backwards (the pitches over the same rhythm), ^ and v a step
// of the scale higher or lower each
func (g *gen) operate(ph *phraseNotes, ops string, final bool) {
	if len(ph.notes) == 0 {
		return
	}
	did := false
	for _, op := range ops {
		switch op {
		case '~':
			pivot := g.nearest(ph.notes[0].Key)
			for i := range ph.notes {
				j := max(0, min(2*pivot-g.nearest(ph.notes[i].Key), len(g.pool)-1))
				ph.notes[i].Key = g.pool[j]
			}
		case '<':
			for i, j := 0, len(ph.notes)-1; i < j; i, j = i+1, j-1 {
				ph.notes[i].Key, ph.notes[j].Key = ph.notes[j].Key, ph.notes[i].Key
			}
		case '^', 'v':
			d := 1
			if op == 'v' {
				d = -1
			}
			for i := range ph.notes {
				ph.notes[i].Key = g.pool[max(0, min(g.nearest(ph.notes[i].Key)+d, len(g.pool)-1))]
			}
		default:
			continue
		}
		did = true
	}
	if did {
		g.fit(ph, final)
	}
}

// fit moves the notes of a phrase that matter (on a beat, or long) and are no tone of the chord at their time to the nearest tone of it (a phrase repeated over another chord);
// the last note of the last phrase ends on the root
func (g *gen) fit(ph *phraseNotes, final bool) {
	for i := range ph.notes {
		n := &ph.notes[i]
		c := g.chordAt(n.Start)
		if c == nil || c.has(n.Key) || !strong(n.Start-ph.off, n.Length) {
			continue
		}
		at := g.nearest(n.Key)
		for d := 1; d < len(g.pool); d++ {
			for _, j := range []int{at - d, at + d} {
				if j >= 0 && j < len(g.pool) && c.has(g.pool[j]) {
					n.Key = g.pool[j]
					goto next
				}
			}
		}
	next:
	}
	if final && len(ph.notes) > 0 {
		n := &ph.notes[len(ph.notes)-1]
		if c := g.chordAt(n.Start); c != nil {
			at := g.nearest(n.Key)
			for d := 0; d < len(g.pool); d++ {
				for _, j := range []int{at - d, at + d} {
					if j >= 0 && j < len(g.pool) && ((g.pool[j]%12)+12)%12 == ((c.Root%12)+12)%12 {
						n.Key = g.pool[j]
						return
					}
				}
			}
		}
	}
}
