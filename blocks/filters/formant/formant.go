package formant

import (
	"math"

	"github.com/almerlucke/psalm/blocks/filters/svf"
)

// vowel holds the first three formant frequencies (Hz) and their relative gains
type vowel struct {
	freqs [3]float64
	gains [3]float64
}

// Approximate average formants of an adult male voice
var vowels = []vowel{
	{[3]float64{730, 1090, 2440}, [3]float64{1.0, 0.5, 0.25}}, // a
	{[3]float64{530, 1840, 2480}, [3]float64{1.0, 0.4, 0.3}},  // e
	{[3]float64{270, 2290, 3010}, [3]float64{1.0, 0.3, 0.25}}, // i
	{[3]float64{570, 840, 2410}, [3]float64{1.0, 0.6, 0.15}},  // o
	{[3]float64{300, 870, 2240}, [3]float64{1.0, 0.35, 0.1}},  // u
}

// NumVowels is the number of vowels, Vowel positions range from 0 to NumVowels-1 (a, e, i, o, u)
const NumVowels = 5

// Formant is a vowel filter: three parallel resonant bandpass filters tuned to the formants of a
// vowel. The vowel position morphs smoothly between a (0), e (1), i (2), o (3) and u (4)
type Formant struct {
	sr      float64
	pos     float64
	q       float64
	shift   float64
	filters [3]*svf.SVF
	gains   [3]float64
}

// New creates a formant filter, q sets the sharpness of the formants (around 5-20)
func New(vowelPos float64, q float64, sr float64) *Formant {
	f := &Formant{sr: sr, pos: vowelPos, q: q, shift: 1.0}
	for i := range f.filters {
		f.filters[i] = svf.New(1000.0, 0.0, sr)
	}
	f.update()
	return f
}

// Set sets the vowel position (0-4, fractional values morph) and formant sharpness
func (f *Formant) Set(vowelPos float64, q float64) {
	f.pos = vowelPos
	f.q = q
	f.update()
}

// SetShift scales all formant frequencies, > 1 sounds like a smaller voice, < 1 like a bigger one
func (f *Formant) SetShift(shift float64) {
	f.shift = max(0.1, shift)
	f.update()
}

func (f *Formant) update() {
	pos := max(0.0, min(f.pos, NumVowels-1))
	idx := int(pos)
	if idx >= NumVowels-1 {
		idx = NumVowels - 2
	}
	frac := pos - float64(idx)
	v1, v2 := vowels[idx], vowels[idx+1]

	for i, filter := range f.filters {
		// morph formant frequencies geometrically, it sounds more natural than linear
		freq := v1.freqs[i] * math.Pow(v2.freqs[i]/v1.freqs[i], frac) * f.shift
		filter.SetFrequencyQ(freq, f.q)
		f.gains[i] = v1.gains[i] + (v2.gains[i]-v1.gains[i])*frac
	}
}

func (f *Formant) Clear() {
	for _, filter := range f.filters {
		filter.Clear()
	}
}

func (f *Formant) Process(x float64) float64 {
	out := 0.0
	for i, filter := range f.filters {
		_, band, _ := filter.Tick(x)
		// band * k has a constant 0 dB peak gain
		out += band * filter.K() * f.gains[i]
	}
	return out
}
