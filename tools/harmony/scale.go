package harmony

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// Modes maps scale names to the intervals of their steps above the root
var Modes = map[string][]int{
	"major":            {0, 2, 4, 5, 7, 9, 11},
	"ionian":           {0, 2, 4, 5, 7, 9, 11},
	"minor":            {0, 2, 3, 5, 7, 8, 10},
	"aeolian":          {0, 2, 3, 5, 7, 8, 10},
	"harmonic minor":   {0, 2, 3, 5, 7, 8, 11},
	"melodic minor":    {0, 2, 3, 5, 7, 9, 11},
	"dorian":           {0, 2, 3, 5, 7, 9, 10},
	"phrygian":         {0, 1, 3, 5, 7, 8, 10},
	"lydian":           {0, 2, 4, 6, 7, 9, 11},
	"mixolydian":       {0, 2, 4, 5, 7, 9, 10},
	"locrian":          {0, 1, 3, 5, 6, 8, 10},
	"major pentatonic": {0, 2, 4, 7, 9},
	"minor pentatonic": {0, 3, 5, 7, 10},
	"blues":            {0, 3, 5, 6, 7, 10},
	"whole tone":       {0, 2, 4, 6, 8, 10},
	"chromatic":        {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
}

// Scale is a scale (or key) on a root
type Scale struct {
	Root  int // pitch class
	Mode  string
	Steps []int // intervals above the root
	flats bool  // spell names with flats
}

// NewScale creates a scale on a root note name ("D", "Bb") in a mode ("minor", "dorian", ...)
func NewScale(root string, mode string) (Scale, error) {
	pc, err := PitchClass(root)
	if err != nil {
		return Scale{}, err
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	steps, ok := Modes[mode]
	if !ok {
		return Scale{}, fmt.Errorf("harmony: unknown mode %q", mode)
	}
	s := Scale{Root: pc, Mode: mode, Steps: slices.Clone(steps)}
	s.flats = strings.Contains(root[1:], "b") || (!strings.Contains(root, "#") && flatKey(pc, steps))
	return s, nil
}

// MustScale creates a scale and panics on an error
func MustScale(root string, mode string) Scale {
	s, err := NewScale(root, mode)
	if err != nil {
		panic(err)
	}
	return s
}

// Key parses a key like "D minor", "Eb major" or "A dorian"
func Key(key string) (Scale, error) {
	root, mode, _ := strings.Cut(strings.TrimSpace(key), " ")
	if mode == "" {
		mode = "major"
	}
	return NewScale(root, mode)
}

// MustKey parses a key and panics on an error
func MustKey(key string) Scale {
	s, err := Key(key)
	if err != nil {
		panic(err)
	}
	return s
}

// flatKey returns true for scales whose key signature has flats (the related major scale is F,
// Bb, Eb, Ab, Db or Gb)
func flatKey(root int, steps []int) bool {
	major := Modes["major"]
	if len(steps) != 7 {
		return root == 5 || root == 10 || root == 3 || root == 8 || root == 1
	}
	for shift := range 12 {
		ionian := mod12(root + shift)
		same := true
		for _, st := range steps {
			if !slices.Contains(major, mod12(root+st-ionian)) {
				same = false
				break
			}
		}
		if same {
			return slices.Contains([]int{5, 10, 3, 8, 1, 6}, ionian)
		}
	}
	return false
}

// Name returns the name of a pitch class spelled for the scale
func (s Scale) Name(pc int) string {
	if s.flats {
		return flatNames[mod12(pc)]
	}
	return sharpNames[mod12(pc)]
}

// Contains returns true when the pitch is in the scale
func (s Scale) Contains(pitch float64) bool {
	return slices.Contains(s.Steps, mod12(pitchClass(pitch)-s.Root))
}

// Pitch returns the pitch of a scale degree (1 is the root, 8 the root an octave up, 0 and below go
// down) in an octave (the root of octave 4 is in the octave of middle C)
func (s Scale) Pitch(degree int, octave int) float64 {
	n := len(s.Steps)
	d := degree - 1
	oct := int(math.Floor(float64(d) / float64(n)))
	idx := d - oct*n
	return float64(s.Root + (octave+1)*12 + oct*12 + s.Steps[idx])
}

// Snap returns the nearest pitch in the scale (going down on a tie)
func (s Scale) Snap(pitch float64) float64 {
	p := math.Round(pitch)
	for d := 0.0; d <= 12; d++ {
		if s.Contains(p - d) {
			return p - d
		}
		if s.Contains(p + d) {
			return p + d
		}
	}
	return p
}

// Step moves a pitch n steps through the scale (negative goes down), from the nearest scale pitch
func (s Scale) Step(pitch float64, n int) float64 {
	p := s.Snap(pitch)
	for n > 0 {
		p++
		for !s.Contains(p) {
			p++
		}
		n--
	}
	for n < 0 {
		p--
		for !s.Contains(p) {
			p--
		}
		n++
	}
	return p
}

// Pitches returns all scale pitches from low to high
func (s Scale) Pitches(low, high float64) []float64 {
	var ps []float64
	for p := math.Ceil(low); p <= high; p++ {
		if s.Contains(p) {
			ps = append(ps, p)
		}
	}
	return ps
}

// preferredNames are the chord types used to name the chords of a scale
var preferredNames = []string{"", "m", "dim", "aug", "7", "maj7", "m7", "m7b5", "dim7", "mMaj7", "aug7"}

// Triad returns the chord of thirds of the scale on a degree (1-7)
func (s Scale) Triad(degree int) Chord {
	return s.stack(degree, 3)
}

// Seventh returns the seventh chord of the scale on a degree
func (s Scale) Seventh(degree int) Chord {
	return s.stack(degree, 4)
}

// stack stacks notes of the scale in thirds and names the chord
func (s Scale) stack(degree int, notes int) Chord {
	base := s.Pitch(degree, 4)
	intervals := make([]int, notes)
	for i := range notes {
		intervals[i] = int(s.Pitch(degree+2*i, 4) - base)
	}
	root := pitchClass(base)
	name := s.Name(root)
	for _, suffix := range preferredNames {
		if slices.Equal(chordTypes[suffix], intervals) {
			name += suffix
			break
		}
	}
	return Chord{Name: name, Root: root, Intervals: intervals, Bass: root}
}

var romans = []string{"vii", "iii", "vi", "iv", "ii", "v", "i"} // longest first

// Roman returns the chord of a roman numeral in the scale: the degree of the scale from the
// numeral (in D minor VII is C), an accidental "b" or "#" in front moves it (#vii° is C#dim), major for upper case and minor for lower case, "°"
// (or "o", "dim") diminished, "+" augmented, and an optional chord type after it like "7", "maj7"
// or "sus4": "i", "VI", "iv", "V7", "bVII", "vii°7", "ii7"
func (s Scale) Roman(numeral string) (Chord, error) {
	rest := strings.TrimSpace(numeral)
	shift := 0
	for len(rest) > 0 && (rest[0] == 'b' || rest[0] == '#') {
		if rest[0] == 'b' {
			shift--
		} else {
			shift++
		}
		rest = rest[1:]
	}
	degree := 0
	upper := false
	for _, r := range romans {
		if strings.HasPrefix(strings.ToLower(rest), r) {
			degree = map[string]int{"i": 1, "ii": 2, "iii": 3, "iv": 4, "v": 5, "vi": 6, "vii": 7}[r]
			upper = rest[:len(r)] == strings.ToUpper(r)
			rest = rest[len(r):]
			break
		}
	}
	if degree == 0 {
		return Chord{}, fmt.Errorf("harmony: bad roman numeral %q", numeral)
	}

	quality := "m"
	if upper {
		quality = ""
	}
	switch {
	case strings.HasPrefix(rest, "°"), strings.HasPrefix(rest, "o"), strings.HasPrefix(rest, "dim"):
		quality = "dim"
		rest = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(rest, "°"), "o"), "dim")
		if rest == "7" {
			quality, rest = "dim7", ""
		}
	case strings.HasPrefix(rest, "ø"):
		quality, rest = "m7b5", strings.TrimPrefix(strings.TrimPrefix(rest, "ø"), "7")
	case strings.HasPrefix(rest, "+"):
		quality, rest = "aug", rest[1:]
	}
	suffix := quality + rest
	switch {
	case quality == "" && rest != "":
		suffix = rest // V7, Vmaj7, Vsus4
	case quality == "m" && rest == "7":
		suffix = "m7"
	case quality == "m" && rest == "maj7":
		suffix = "mMaj7"
	}

	root := mod12(s.Root + s.Steps[(degree-1)%len(s.Steps)] + shift)
	name := s.Name(root)
	if shift > 0 {
		name = sharpNames[root] // a raised degree is spelled with a sharp
	} else if shift < 0 {
		name = flatNames[root]
	}
	return ParseChord(name + suffix)
}

// Progression returns the chords of roman numerals separated by spaces: "i VI iv V"
func (s Scale) Progression(numerals string) ([]Chord, error) {
	var chords []Chord
	for _, n := range strings.Fields(numerals) {
		if n == "|" {
			continue
		}
		c, err := s.Roman(n)
		if err != nil {
			return nil, err
		}
		chords = append(chords, c)
	}
	return chords, nil
}

// MustProgression returns the chords of roman numerals and panics on an error
func (s Scale) MustProgression(numerals string) []Chord {
	chords, err := s.Progression(numerals)
	if err != nil {
		panic(err)
	}
	return chords
}
