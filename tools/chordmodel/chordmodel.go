// Package chordmodel is a Markov model of chord progressions: the chords are roman numerals of a key (I, vi, bVII, V7), and the model says how likely each is to follow
// another. It generates progressions of a number of bars, a chord lasting 1, 2 or half a bar by weights of its own, and can keep some of the chords of a progression (locked)
// while the others are drawn again, each fitting the chords on both sides of it. It learns from progressions typed in numerals (what follows what, and which pair of chords
// leads to which), and has presets for a few styles.
//
//	m, _ := chordmodel.Preset("pop")
//	slots := m.Fill(8, rand.New(rand.NewPCG(1, 2)))                  // eight bars
//	slots[2].Lock = true
//	m.Reroll(slots, rng)                                             // the others again
//	text := m.Text(slots, harmony.MustScale("D", "major"), true)     // "D A Bm G*2"
package chordmodel

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/tools/harmony"
)

// Length is how many bars a chord lasts, and how often it does
type Length struct {
	Bars   float64 `json:"bars"`
	Weight float64 `json:"weight"`
}

// Slot is a chord of a progression: the state it is (an index of Model.States), the bars it lasts, and whether it is kept when the others are drawn again
type Slot struct {
	State int     `json:"s"`
	Bars  float64 `json:"bars"`
	Lock  bool    `json:"lock"`
}

// Model is the chords and how likely each follows another. W[a][b] is the weight of chord b after chord a (rows need not add up to anything: they are weights); Start is
// how likely each is to begin; Pairs are what was learned of two chords in a row ("a,b" -> the weight of each chord after them), trusted by Memory (0: only W, 1: the pairs
// where there are any); End is the chord to end on (-1: none)
type Model struct {
	States  []string             `json:"states"`
	Start   []float64            `json:"start"`
	W       [][]float64          `json:"w"`
	Pairs   map[string][]float64 `json:"pairs,omitempty"`
	Memory  float64              `json:"memory"`
	Lengths []Length             `json:"lengths"`
	End     int                  `json:"end"`
}

// New returns a model of chords (roman numerals) in which every chord is as likely as any other
func New(states ...string) *Model {
	n := len(states)
	m := &Model{States: slices.Clone(states), Start: make([]float64, n), W: make([][]float64, n), End: -1,
		Lengths: []Length{{0.5, 0}, {1, 6}, {2, 3}, {4, 0}}}
	for i := range n {
		m.Start[i] = 1
		m.W[i] = make([]float64, n)
		for j := range n {
			m.W[i][j] = 1
		}
	}
	return m
}

// Clone returns a copy
func (m *Model) Clone() *Model {
	c := *m
	c.States = slices.Clone(m.States)
	c.Start = slices.Clone(m.Start)
	c.W = make([][]float64, len(m.W))
	for i, r := range m.W {
		c.W[i] = slices.Clone(r)
	}
	c.Pairs = map[string][]float64{}
	for k, v := range m.Pairs {
		c.Pairs[k] = slices.Clone(v)
	}
	c.Lengths = slices.Clone(m.Lengths)
	return &c
}

// Index is the state of a chord (a numeral), -1 for none
func (m *Model) Index(name string) int { return slices.Index(m.States, name) }

// AddState adds a chord to the model (a numeral, read in a key to see it is one), and returns its state; one it has already is returned as it is
func (m *Model) AddState(name string) (int, error) {
	if i := m.Index(name); i >= 0 {
		return i, nil
	}
	if _, err := harmony.MustScale("C", "major").Roman(name); err != nil {
		return -1, fmt.Errorf("%s: a roman numeral (I, vi, bVII, V7, iiø)", name)
	}
	n := len(m.States)
	m.States = append(m.States, name)
	m.Start = append(m.Start, 0)
	for i := range m.W {
		m.W[i] = append(m.W[i], 0)
	}
	m.W = append(m.W, make([]float64, n+1))
	for k, v := range m.Pairs {
		m.Pairs[k] = append(v, 0)
	}
	return n, nil
}

// RemoveState takes a chord out of the model
func (m *Model) RemoveState(i int) {
	if i < 0 || i >= len(m.States) || len(m.States) <= 2 {
		return
	}
	m.States = slices.Delete(m.States, i, i+1)
	m.Start = slices.Delete(m.Start, i, i+1)
	m.W = slices.Delete(m.W, i, i+1)
	for r := range m.W {
		m.W[r] = slices.Delete(m.W[r], i, i+1)
	}
	m.Pairs = nil // the pairs name chords by their places
	switch {
	case m.End == i:
		m.End = -1
	case m.End > i:
		m.End--
	}
}

// normalized is weights as chances, adding up to 1 (all equal when none is above 0)
func normalized(w []float64) []float64 {
	sum := 0.0
	for _, x := range w {
		sum += max(0, x)
	}
	out := make([]float64, len(w))
	for i, x := range w {
		if sum > 0 {
			out[i] = max(0, x) / sum
		} else {
			out[i] = 1 / float64(len(w))
		}
	}
	return out
}

func pairKey(a, b int) string { return strconv.Itoa(a) + "," + strconv.Itoa(b) }

// Next is how likely each chord is after a chord (and the one before it, -1 for none): the weights of the chord, and where pairs were learned for the two, mixed with them by
// Memory. Chances that add up to 1
func (m *Model) Next(prev, before int) []float64 {
	if prev < 0 {
		return normalized(m.Start)
	}
	p := normalized(m.W[prev])
	if before >= 0 && m.Memory > 0 {
		if row, ok := m.Pairs[pairKey(before, prev)]; ok {
			q := normalized(row)
			for i := range p {
				p[i] = (1-m.Memory)*p[i] + m.Memory*q[i]
			}
		}
	}
	return p
}

// pick draws a chord after prev (before it: before), each weighted by how likely the chord after it, ahead, is to follow it (ahead -1: no such chord)
func (m *Model) pick(rng *rand.Rand, prev, before, ahead int) int {
	p := m.Next(prev, before)
	if ahead >= 0 {
		for i := range p {
			p[i] *= normalized(m.W[i])[ahead] + 1e-6
		}
		p = normalized(p)
	}
	x := rng.Float64()
	for i, c := range p {
		if x < c {
			return i
		}
		x -= c
	}
	return len(p) - 1
}

// lengths draws the lengths of chords until they fill a number of bars (the last cut to fit)
func (m *Model) lengths(bars float64, rng *rand.Rand) []float64 {
	var ls []float64
	sum := 0.0
	for _, l := range m.Lengths {
		sum += max(0, l.Weight)
	}
	for left := bars; left > 1e-9; {
		bar := 1.0
		if sum > 0 {
			x := rng.Float64() * sum
			for _, l := range m.Lengths {
				if x < max(0, l.Weight) {
					bar = l.Bars
					break
				}
				x -= max(0, l.Weight)
			}
		}
		bar = min(bar, left)
		ls = append(ls, bar)
		left -= bar
	}
	return ls
}

// Fill draws a progression of a number of bars: the chords from the model, the first from Start, the last End (when it has one, and the one before it a chord that leads there)
func (m *Model) Fill(bars float64, rng *rand.Rand) []Slot {
	ls := m.lengths(max(bars, 0.5), rng)
	slots := make([]Slot, len(ls))
	for i, l := range ls {
		slots[i].Bars = l
	}
	m.Reroll(slots, rng)
	return slots
}

// Reroll draws the chords again, all but the ones that are locked: each fits what is before it and the chord after it, when that is locked (or the end of the model)
func (m *Model) Reroll(slots []Slot, rng *rand.Rand) {
	prev, before := -1, -1
	for i := range slots {
		if !slots[i].Lock {
			ahead := -1
			switch {
			case i == len(slots)-1 && m.End >= 0:
				slots[i].State = m.End
				prev, before = slots[i].State, prev
				continue
			case i+1 < len(slots) && slots[i+1].Lock:
				ahead = slots[i+1].State
			case i+2 == len(slots) && m.End >= 0 && !slots[i+1].Lock:
				ahead = m.End // the one before the last leads to the end
			}
			slots[i].State = m.pick(rng, prev, before, ahead)
		}
		prev, before = slots[i].State, prev
	}
}

// Learn teaches the model a progression (states, in order): what follows what and which pair leads to which are made likelier, by rate for every time it was seen (the chances
// of the rows are 1: a rate of 0.25 means four times seen is as likely as all the rest together)
func (m *Model) Learn(seq []int, rate float64) {
	if len(seq) == 0 {
		return
	}
	if m.Pairs == nil {
		m.Pairs = map[string][]float64{}
	}
	for i := range m.W {
		m.W[i] = normalized(m.W[i])
	}
	m.Start = normalized(m.Start)
	m.Start[seq[0]] += rate
	for i := 1; i < len(seq); i++ {
		m.W[seq[i-1]][seq[i]] += rate
		if i >= 2 {
			k := pairKey(seq[i-2], seq[i-1])
			if m.Pairs[k] == nil {
				m.Pairs[k] = make([]float64, len(m.States))
			}
			m.Pairs[k][seq[i]] += 1
		}
	}
}

// LearnText teaches the model progressions typed as numerals (a line each, the chords separated by spaces, bars or dashes): chords it does not have are added. It returns how many
// progressions it learned from
func (m *Model) LearnText(text string, rate float64) (int, error) {
	n := 0
	for _, line := range strings.Split(text, "\n") {
		fields := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '|' || r == '-' || r == ',' || r == '\t' })
		if len(fields) == 0 {
			continue
		}
		var seq []int
		for _, f := range fields {
			i, err := m.AddState(f)
			if err != nil {
				return n, err
			}
			seq = append(seq, i)
		}
		m.Learn(seq, rate)
		n++
	}
	return n, nil
}

// ChordName is the chord of a state in a scale (D for the I of D major)
func (m *Model) ChordName(state int, scale harmony.Scale) string {
	if state < 0 || state >= len(m.States) {
		return ""
	}
	c, err := scale.Roman(m.States[state])
	if err != nil {
		return m.States[state]
	}
	return c.Name
}

// Text writes a progression for a harmony module: the chords in the scale (or the numerals), a number of bars each (`*2`)
func (m *Model) Text(slots []Slot, scale harmony.Scale, symbols bool) string {
	var parts []string
	for _, s := range slots {
		name := m.States[s.State]
		if symbols {
			name = m.ChordName(s.State, scale)
		}
		if s.Bars != 1 {
			name += "*" + strconv.FormatFloat(s.Bars, 'f', -1, 64)
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, " ")
}

// --- presets ---

type preset struct {
	states []string
	start  map[string]float64
	rows   map[string]map[string]float64
	end    string
}

var presets = map[string]preset{
	"pop": {
		states: []string{"I", "ii", "iii", "IV", "V", "vi", "vii°"},
		start:  map[string]float64{"I": 80, "vi": 10, "IV": 10},
		rows: map[string]map[string]float64{
			"I":    {"IV": 30, "V": 25, "vi": 25, "ii": 10, "iii": 5, "I": 5},
			"ii":   {"V": 55, "IV": 15, "I": 10, "vi": 10, "vii°": 5, "iii": 5},
			"iii":  {"vi": 40, "IV": 30, "ii": 20, "I": 10},
			"IV":   {"V": 35, "I": 25, "ii": 15, "vi": 15, "IV": 5, "iii": 5},
			"V":    {"I": 50, "vi": 30, "IV": 10, "iii": 5, "V": 5},
			"vi":   {"IV": 35, "ii": 25, "V": 20, "I": 10, "iii": 10},
			"vii°": {"I": 70, "iii": 20, "vi": 10},
		},
	},
	"minor": {
		states: []string{"i", "ii°", "III", "iv", "v", "VI", "VII"},
		start:  map[string]float64{"i": 80, "VI": 10, "iv": 10},
		rows: map[string]map[string]float64{
			"i":   {"VI": 25, "iv": 20, "VII": 20, "III": 15, "v": 10, "i": 5, "ii°": 5},
			"ii°": {"v": 60, "i": 20, "iv": 10, "VII": 10},
			"III": {"VI": 35, "iv": 25, "VII": 20, "i": 20},
			"iv":  {"v": 25, "i": 25, "VII": 20, "VI": 15, "ii°": 10, "iv": 5},
			"v":   {"i": 55, "VI": 25, "iv": 10, "III": 10},
			"VI":  {"VII": 30, "iv": 25, "i": 20, "III": 15, "v": 10},
			"VII": {"i": 40, "III": 30, "VI": 20, "iv": 10},
		},
	},
	"jazz": {
		states: []string{"Imaj7", "ii7", "iii7", "IVmaj7", "V7", "vi7", "viiø"},
		start:  map[string]float64{"Imaj7": 70, "ii7": 20, "vi7": 10},
		rows: map[string]map[string]float64{
			"Imaj7":  {"ii7": 25, "IVmaj7": 20, "vi7": 20, "iii7": 15, "V7": 10, "Imaj7": 5, "viiø": 5},
			"ii7":    {"V7": 65, "viiø": 10, "IVmaj7": 10, "Imaj7": 10, "vi7": 5},
			"iii7":   {"vi7": 45, "ii7": 25, "IVmaj7": 20, "Imaj7": 10},
			"IVmaj7": {"V7": 30, "Imaj7": 20, "ii7": 20, "iii7": 15, "viiø": 10, "vi7": 5},
			"V7":     {"Imaj7": 50, "vi7": 25, "iii7": 10, "IVmaj7": 10, "V7": 5},
			"vi7":    {"ii7": 45, "IVmaj7": 20, "V7": 15, "iii7": 10, "Imaj7": 10},
			"viiø":   {"iii7": 50, "Imaj7": 20, "vi7": 20, "V7": 10},
		},
	},
	"rock": {
		states: []string{"I", "ii", "IV", "V", "vi", "bVI", "bVII"},
		start:  map[string]float64{"I": 85, "vi": 10, "IV": 5},
		rows: map[string]map[string]float64{
			"I":    {"IV": 30, "bVII": 25, "V": 15, "vi": 15, "bVI": 10, "I": 5},
			"ii":   {"V": 50, "IV": 30, "I": 20},
			"IV":   {"I": 30, "V": 25, "bVII": 20, "ii": 10, "vi": 10, "IV": 5},
			"V":    {"I": 40, "IV": 25, "vi": 20, "bVII": 10, "V": 5},
			"vi":   {"IV": 40, "V": 25, "I": 15, "bVII": 10, "ii": 10},
			"bVI":  {"bVII": 60, "I": 20, "IV": 20},
			"bVII": {"IV": 40, "I": 40, "bVI": 10, "V": 10},
		},
	},
}

// PresetNames are the names of the presets
func PresetNames() []string {
	names := make([]string, 0, len(presets))
	for n := range presets {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Preset returns a model of a style: the chords and their chances
func Preset(name string) (*Model, bool) {
	p, ok := presets[name]
	if !ok {
		return nil, false
	}
	m := New(p.states...)
	for i, s := range p.states {
		m.Start[i] = p.start[s]
		for j, t := range p.states {
			m.W[i][j] = p.rows[s][t]
		}
	}
	if p.end != "" {
		m.End = m.Index(p.end)
	}
	return m, true
}
