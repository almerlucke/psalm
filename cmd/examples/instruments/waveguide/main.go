// Waveguide: audition the physically modelled strings, bows, drums and pipes on a page. Choose a
// preset (and yours: type a name, Save, kept in editors-presets/waveguide, the folder the editors
// example uses too), play it on the page's keyboard (the computer's keys: A W S E D ..., or a MIDI
// keyboard), shape every setting of the model, and test it with a phrase: a scale, a chord held, or
// the whole range from the lowest note to the highest, to hear its tuning and its registers.
// Open http://localhost:8087.
//
//	go run ./cmd/examples/instruments/waveguide
//	go run ./cmd/examples/instruments/waveguide -preset violin -hall 0.3 -theme midnight
//	go run ./cmd/examples/instruments/waveguide -list        the presets
package main

import (
	"maps"
	"slices"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/synths/waveguide"

	"github.com/almerlucke/psalm/cmd/examples/instruments/audition"
)

type step = audition.Step

// the phrases the buttons play. They sit in the middle of the keyboard where plucks, bows and
// pipes all speak; the range goes from the lowest note a string or a drum has to the highest
var phrases = map[string][]step{
	"Scale": func() []step {
		var s []step
		for i, p := range []float64{57, 59, 60, 62, 64, 65, 67, 69, 67, 65, 64, 62, 60, 59, 57} {
			s = append(s, step{Pitch: p, At: float64(i) * 0.45, Hold: 0.4, Velocity: 0.5 + 0.4*float64(i%3)/2})
		}
		return s
	}(),
	"Chord": {
		{Pitch: 48, At: 0, Hold: 3.5, Velocity: 0.7}, {Pitch: 55, At: 0.12, Hold: 3.4, Velocity: 0.7}, {Pitch: 60, At: 0.24, Hold: 3.3, Velocity: 0.7},
		{Pitch: 64, At: 0.36, Hold: 3.2, Velocity: 0.7}, {Pitch: 67, At: 0.48, Hold: 3.1, Velocity: 0.7},
	},
	"Range": func() []step {
		var s []step
		for i := 0; i <= 12; i++ { // every fifth semitone, up, from C1 to C7
			s = append(s, step{Pitch: 24 + float64(i)*6, At: float64(i) * 0.7, Hold: 0.6, Velocity: 0.8})
		}
		return s
	}(),
	"Dynamics": {
		{Pitch: 57, At: 0, Hold: 0.8, Velocity: 0.15}, {Pitch: 57, At: 1, Hold: 0.8, Velocity: 0.3}, {Pitch: 57, At: 2, Hold: 0.8, Velocity: 0.5},
		{Pitch: 57, At: 3, Hold: 0.8, Velocity: 0.7}, {Pitch: 57, At: 4, Hold: 0.8, Velocity: 0.9}, {Pitch: 57, At: 5, Hold: 0.8, Velocity: 1.0},
	},
}

func main() {
	audition.Main(spec())
}

// spec is the waveguide on the page; its friction bows tune themselves in the background in the
// first second or two (a note at a pitch not yet tuned plays on an estimate)
func spec() audition.Spec {
	return audition.Spec{
		Name:    "Waveguide",
		Presets: slices.Sorted(maps.Keys(waveguide.Presets)),
		Default: "guitar",
		Describe: func(n string) params.Set {
			return waveguide.Describe(waveguide.Presets[n]())
		},
		New: func(n string, voices int) psalm.Modular {
			return waveguide.NewPoly(waveguide.Presets[n](), voices)
		},
		Voices:  12,
		Hall:    0.25,
		Phrases: phrases,
		Order:   []string{"Scale", "Chord", "Range", "Dynamics"},
		Keep:    "waveguide",
	}
}
