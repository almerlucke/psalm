// Modal: audition the struck, rung and bowed instruments of the modal synth on a page: bars (marimba,
// vibraphone, xylophone, glockenspiel, kalimba), bells and bowls, plates, gongs and cymbals, steel
// pans, blocks and bottles. Choose a preset (and yours: type a name, Save, kept in
// editors-presets/modal, the folder the editors example uses too), play it on the page's keyboard (the
// computer's keys: A W S E D ..., or a MIDI keyboard), shape every setting of the model (the
// instrument, the mallet, where it is struck, the resonator tube, the beating of the modes) and test
// it with a phrase: a scale, a chord, the whole range, one note struck from soft to hard, a mallet
// roll, or one low note left to ring. Open http://localhost:8087.
//
//	go run ./cmd/examples/instruments/modal
//	go run ./cmd/examples/instruments/modal -preset tubularBells -hall 0.4 -theme midnight
//	go run ./cmd/examples/instruments/modal -list        the presets
package main

import (
	"maps"
	"slices"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/synths/modal"

	"github.com/almerlucke/psalm/cmd/examples/instruments/audition"
)

type step = audition.Step

// the phrases the buttons play: bars and bells ring on their own after the blow, so the notes are
// short and the ring is left to the instrument
var phrases = map[string][]step{
	"Scale": func() []step {
		var s []step
		for i, p := range []float64{60, 62, 64, 65, 67, 69, 71, 72, 71, 69, 67, 65, 64, 62, 60} {
			s = append(s, step{Pitch: p, At: float64(i) * 0.4, Hold: 0.3, Velocity: 0.55 + 0.35*float64(i%3)/2})
		}
		return s
	}(),
	"Chord": {
		{Pitch: 48, At: 0, Hold: 0.4, Velocity: 0.7}, {Pitch: 55, At: 0.1, Hold: 0.4, Velocity: 0.7}, {Pitch: 60, At: 0.2, Hold: 0.4, Velocity: 0.7},
		{Pitch: 64, At: 0.3, Hold: 0.4, Velocity: 0.7}, {Pitch: 67, At: 0.4, Hold: 0.4, Velocity: 0.7},
	},
	"Range": func() []step {
		var s []step
		for i := 0; i <= 10; i++ { // every fifth semitone, up, from C2 to C7
			s = append(s, step{Pitch: 36 + float64(i)*6, At: float64(i) * 0.8, Hold: 0.4, Velocity: 0.8})
		}
		return s
	}(),
	// one note struck from soft to hard: a harder blow is shorter in contact, so brighter
	"Dynamics": {
		{Pitch: 60, At: 0, Hold: 0.3, Velocity: 0.1}, {Pitch: 60, At: 2, Hold: 0.3, Velocity: 0.25}, {Pitch: 60, At: 4, Hold: 0.3, Velocity: 0.45},
		{Pitch: 60, At: 6, Hold: 0.3, Velocity: 0.65}, {Pitch: 60, At: 8, Hold: 0.3, Velocity: 0.85}, {Pitch: 60, At: 10, Hold: 0.3, Velocity: 1.0},
	},
	// a roll: one note struck again and again, faster and louder
	"Roll": func() []step {
		var s []step
		at := 0.0
		for i := 0; i < 28; i++ {
			s = append(s, step{Pitch: 67, At: at, Hold: 0.15, Velocity: 0.25 + 0.65*float64(i)/27})
			at += 0.16 - 0.1*float64(i)/27
		}
		return s
	}(),
	// one low note, left to ring: the decay of every mode, the beating of bells and bowls
	"Ring": {
		{Pitch: 48, At: 0, Hold: 0.3, Velocity: 0.8},
	},
}

func main() {
	audition.Main(spec())
}

// spec is the modal synth on the page
func spec() audition.Spec {
	return audition.Spec{
		Name:    "Modal",
		Presets: slices.Sorted(maps.Keys(modal.Presets)),
		Default: "vibraphone",
		Describe: func(n string) params.Set {
			return modal.Describe(modal.Presets[n]())
		},
		New: func(n string, voices int) psalm.Modular {
			return modal.NewPoly(modal.Presets[n](), voices)
		},
		Voices:  16,
		Hall:    0.3,
		Phrases: phrases,
		Order:   []string{"Scale", "Chord", "Range", "Dynamics", "Roll", "Ring"},
		Keep:    "modal",
	}
}
