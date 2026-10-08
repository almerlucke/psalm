// Package harmony holds music theory helpers for writing parts: chords from symbols ("Dm", "A7",
// "Bbmaj7/D"), voicings and voice leading, scales and keys with roman numeral progressions, an
// arpeggiator and a check that finds clashing notes between a part and its chords
package harmony

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// Chord is a chord: a root, the intervals above it and the bass (another pitch class for slash
// chords like "G/B")
type Chord struct {
	Name      string
	Root      int   // pitch class, 0 is C
	Intervals []int // semitones above the root, starting with 0
	Bass      int   // pitch class of the bass, the root unless a slash chord
}

// chordTypes maps chord symbol suffixes to intervals
var chordTypes = map[string][]int{
	"":        {0, 4, 7},
	"maj":     {0, 4, 7},
	"M":       {0, 4, 7},
	"m":       {0, 3, 7},
	"min":     {0, 3, 7},
	"-":       {0, 3, 7},
	"dim":     {0, 3, 6},
	"°":       {0, 3, 6},
	"o":       {0, 3, 6},
	"aug":     {0, 4, 8},
	"+":       {0, 4, 8},
	"sus2":    {0, 2, 7},
	"sus4":    {0, 5, 7},
	"sus":     {0, 5, 7},
	"5":       {0, 7},
	"6":       {0, 4, 7, 9},
	"m6":      {0, 3, 7, 9},
	"7":       {0, 4, 7, 10},
	"maj7":    {0, 4, 7, 11},
	"M7":      {0, 4, 7, 11},
	"Δ":       {0, 4, 7, 11},
	"Δ7":      {0, 4, 7, 11},
	"m7":      {0, 3, 7, 10},
	"min7":    {0, 3, 7, 10},
	"-7":      {0, 3, 7, 10},
	"mMaj7":   {0, 3, 7, 11},
	"m(maj7)": {0, 3, 7, 11},
	"dim7":    {0, 3, 6, 9},
	"°7":      {0, 3, 6, 9},
	"o7":      {0, 3, 6, 9},
	"m7b5":    {0, 3, 6, 10},
	"ø":       {0, 3, 6, 10},
	"ø7":      {0, 3, 6, 10},
	"aug7":    {0, 4, 8, 10},
	"+7":      {0, 4, 8, 10},
	"7#5":     {0, 4, 8, 10},
	"7sus4":   {0, 5, 7, 10},
	"9":       {0, 4, 7, 10, 14},
	"maj9":    {0, 4, 7, 11, 14},
	"m9":      {0, 3, 7, 10, 14},
	"add9":    {0, 4, 7, 14},
	"madd9":   {0, 3, 7, 14},
	"7b9":     {0, 4, 7, 10, 13},
	"7#9":     {0, 4, 7, 10, 15},
	"11":      {0, 4, 7, 10, 14, 17},
	"m11":     {0, 3, 7, 10, 14, 17},
	"13":      {0, 4, 7, 10, 14, 21},
}

var (
	sharpNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	flatNames  = []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	letters    = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
)

// PitchClass returns the pitch class (0-11) of a note name like "C", "F#" or "Bb"
func PitchClass(name string) (int, error) {
	if name == "" {
		return 0, fmt.Errorf("harmony: empty note name")
	}
	pc, ok := letters[name[0]]
	if !ok {
		return 0, fmt.Errorf("harmony: bad note name %q", name)
	}
	for _, r := range name[1:] {
		switch r {
		case '#':
			pc++
		case 'b':
			pc--
		default:
			return 0, fmt.Errorf("harmony: bad note name %q", name)
		}
	}
	return mod12(pc), nil
}

// noteName splits a note name off the front of a symbol
func noteName(s string) (string, string) {
	if s == "" {
		return "", ""
	}
	i := 1
	for i < len(s) && (s[i] == '#' || s[i] == 'b') {
		i++
	}
	return s[:i], s[i:]
}

// ParseChord reads a chord symbol: a root ("D", "Bb", "F#"), a type ("m", "7", "maj7", "m7b5",
// "dim", "sus4", "add9" and more, nothing is major) and an optional bass ("/B")
func ParseChord(symbol string) (Chord, error) {
	main, bass, hasBass := strings.Cut(strings.TrimSpace(symbol), "/")
	rootName, suffix := noteName(main)
	root, err := PitchClass(rootName)
	if err != nil {
		return Chord{}, fmt.Errorf("harmony: bad chord %q: %w", symbol, err)
	}
	intervals, ok := chordTypes[suffix]
	if !ok {
		return Chord{}, fmt.Errorf("harmony: unknown chord type %q in %q", suffix, symbol)
	}
	c := Chord{Name: strings.TrimSpace(symbol), Root: root, Intervals: slices.Clone(intervals), Bass: root}
	if hasBass {
		b, err := PitchClass(bass)
		if err != nil {
			return Chord{}, fmt.Errorf("harmony: bad bass in %q: %w", symbol, err)
		}
		c.Bass = b
	}
	return c, nil
}

// MustChord parses a chord symbol and panics on an error
func MustChord(symbol string) Chord {
	c, err := ParseChord(symbol)
	if err != nil {
		panic(err)
	}
	return c
}

// Progression parses chord symbols separated by spaces (bar lines "|" are ignored)
func Progression(symbols string) ([]Chord, error) {
	var chords []Chord
	for _, s := range strings.Fields(symbols) {
		if s == "|" {
			continue
		}
		c, err := ParseChord(s)
		if err != nil {
			return nil, err
		}
		chords = append(chords, c)
	}
	return chords, nil
}

// MustProgression parses chord symbols and panics on an error
func MustProgression(symbols string) []Chord {
	chords, err := Progression(symbols)
	if err != nil {
		panic(err)
	}
	return chords
}

func (c Chord) String() string { return c.Name }

// PitchClasses returns the pitch classes of the chord (with the bass of a slash chord)
func (c Chord) PitchClasses() []int {
	pcs := make([]int, 0, len(c.Intervals)+1)
	for _, iv := range c.Intervals {
		pcs = append(pcs, mod12(c.Root+iv))
	}
	if !slices.Contains(pcs, c.Bass) {
		pcs = append(pcs, c.Bass)
	}
	return pcs
}

// Contains returns true when the pitch (MIDI) is a chord tone in any octave
func (c Chord) Contains(pitch float64) bool {
	return slices.Contains(c.PitchClasses(), pitchClass(pitch))
}

// Pitches returns the chord in root position from a root pitch (MIDI), a slash chord with its bass
// below the root: MustChord("G/B").Pitches(55) is B2 G3 B3 D4
func (c Chord) Pitches(root float64) []float64 {
	var ps []float64
	if c.Bass != c.Root {
		down := mod12(c.Root - c.Bass)
		ps = append(ps, root-float64(down))
	}
	for _, iv := range c.Intervals {
		ps = append(ps, root+float64(iv))
	}
	return ps
}

// Close returns the chord in root position with the root at or above the lowest pitch
func (c Chord) Close(lowest float64) []float64 {
	root := lowest + float64(mod12(c.Root-pitchClass(lowest)))
	return c.Pitches(root)
}

// BassPitch returns the bass note of the chord at or above the lowest pitch
// Triad returns the root, the third and the fifth of the chord (its first three tones, root
// position), the root the lowest at or above lowest: for figures on the chord tones, like a broken
// chord in the left hand. A chord of fewer than three tones (a power chord) repeats its last
func (c Chord) Triad(lowest float64) (root, third, fifth float64) {
	root = lowest + float64(mod12(c.Root-pitchClass(lowest)))
	tone := func(i int) float64 { return root + float64(c.Intervals[min(i, len(c.Intervals)-1)]) }
	return root, tone(1), tone(2)
}

func (c Chord) BassPitch(lowest float64) float64 {
	return lowest + float64(mod12(c.Bass-pitchClass(lowest)))
}

// Inversion moves the lowest pitch up an octave n times (a negative n moves the highest down)
func Inversion(pitches []float64, n int) []float64 {
	ps := slices.Clone(pitches)
	slices.Sort(ps)
	for ; n > 0; n-- {
		ps = append(ps[1:], ps[0]+12)
	}
	for ; n < 0; n++ {
		last := len(ps) - 1
		ps = append([]float64{ps[last] - 12}, ps[:last]...)
	}
	return ps
}

// Drop2 moves the second highest pitch down an octave, an open voicing from a close one
func Drop2(pitches []float64) []float64 {
	ps := slices.Clone(pitches)
	slices.Sort(ps)
	if len(ps) >= 2 {
		ps[len(ps)-2] -= 12
	}
	slices.Sort(ps)
	return ps
}

// VoiceLead voices a chord for the same number of voices as the previous chord, with as little
// movement as possible: every voice moves to a nearby chord tone, and all chord tones are used
// when there are enough voices
func VoiceLead(prev []float64, c Chord) []float64 {
	pcs := c.PitchClasses()
	n := len(prev)
	if n == 0 {
		return nil
	}
	// the nearest pitch of a pitch class to a pitch
	nearest := func(from float64, pc int) float64 {
		up := float64(mod12(pc - pitchClass(from)))
		if up > 6 {
			return from + up - 12
		}
		return from + up
	}
	need := min(n, len(pcs)) // distinct pitch classes the voicing has to cover

	best := math.Inf(1)
	var bestPitches []float64
	choice := make([]int, n)
	var search func(v int)
	search = func(v int) {
		if v == n {
			used := map[int]bool{}
			cost := 0.0
			for i, k := range choice {
				used[pcs[k]] = true
				cost += math.Abs(nearest(prev[i], pcs[k]) - prev[i])
			}
			if len(used) < need {
				return
			}
			if cost < best {
				best = cost
				bestPitches = make([]float64, n)
				for i, k := range choice {
					bestPitches[i] = nearest(prev[i], pcs[k])
				}
			}
			return
		}
		for k := range pcs {
			choice[v] = k
			search(v + 1)
		}
	}
	search(0)
	slices.Sort(bestPitches)
	return bestPitches
}

func mod12(x int) int {
	return ((x % 12) + 12) % 12
}

func pitchClass(p float64) int {
	return mod12(int(math.Round(p)))
}
