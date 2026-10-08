package harmony

import (
	"math"

	"github.com/almerlucke/psalm/modules/controls/score"
)

// EqualTemperament is the tuning of a piano: twelve even semitones, A4 at a4 Hz (440 is the
// standard)
func EqualTemperament(a4 float64) score.Tuning {
	return func(pitch float64) float64 { return a4 * math.Exp2((pitch-69)/12) }
}

// JustRatios are the pure ratios (five limit) of the twelve semitones above a tonic: 1, 16/15,
// 9/8, 6/5, 5/4, 4/3, 45/32, 3/2, 8/5, 5/3, 9/5 and 15/8
var JustRatios = [12]float64{1, 16.0 / 15, 9.0 / 8, 6.0 / 5, 5.0 / 4, 4.0 / 3, 45.0 / 32, 3.0 / 2, 8.0 / 5, 5.0 / 3, 9.0 / 5, 15.0 / 8}

// Just tunes the key in just intonation: every note a pure ratio of the tonic (JustRatios), the
// tonic where equal temperament with A4 at a4 Hz has it. The chords on the tonic, the fourth and
// the fifth are pure triads, in a major key also the ones on the sixth and the third; the chord on
// the second is not (its fifth is a comma narrow). A fraction of a semitone stays a fraction
// (evenly) above the note it rounds to
func (s Scale) Just(a4 float64) score.Tuning {
	root := float64(60 + s.Root) // the tonic in the middle of the keyboard (any octave tunes the same)
	tonic := a4 * math.Exp2((root-69)/12)
	return func(pitch float64) float64 {
		n := math.Round(pitch)
		steps := int(n - root)
		octave := math.Floor(float64(steps) / 12)
		f := tonic * math.Exp2(octave) * JustRatios[steps-int(octave)*12]
		if frac := pitch - n; frac != 0 {
			f *= math.Exp2(frac / 12)
		}
		return f
	}
}
