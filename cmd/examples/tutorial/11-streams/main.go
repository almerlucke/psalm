// Step 11: streams, parts played by generators.
//
// Written parts play the same notes every time. A stream makes its notes as it plays, from
// generators (the genny package): a rhythm generator gives the length of every step in beats (a
// negative step is a rest) and a template the message of every note, its values from generators
// too. A stream never plays a song the same way twice.
//
//	&arrange.Stream{
//		Rhythm:  sequence.NewLoop(0.5, 0.5, 1.0, -1.0),      // eighth, eighth, quarter, a quarter rest
//		Message: template.Template{"pitch": melody},       // a genny.Generator[float64]; a []float64 is a chord
//		Amp:     0.3,                                      // a number or a generator
//	}
//
// form.Stream(p, synth, name, func(s Section) *Stream) plays it per section like a part: nil is
// silent, the same stream goes on where it was, a new stream starts fresh. template.NewParameter
// takes a value from the section's Params, so a section can set how a stream plays.
//
// A stream can drive any module, not only synths: Output Messages sends the template as messages,
// Output Values sends its "value" as a number to a value input (form.NewStream, then connect it).
//
// Here a glass melody wanders over the pentatonic scale, brighter in the second section (its
// formant from the section), a bass follows the chords (they change at every bar, from an event),
// and the glass drifts through the stereo field (values for the pan of its channel).
//
//	go run ./cmd/examples/tutorial/11-streams
package main

import (
	"github.com/almerlucke/genny/function"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
)

// choose is a generator that picks one of its values at random every time
func choose[T any](values ...T) *function.Function[T] {
	return function.New(func() T { return values[psalm.Rand().IntN(len(values))] })
}

func main() {
	p := tutorial.New(100)

	form := arrange.NewForm(
		arrange.Section{Name: "Soft", Bars: 8, Params: map[string]any{"formant": 1200.0}},
		arrange.Section{Name: "Bright", Bars: 8, Params: map[string]any{"formant": 3200.0}},
	)
	scale := harmony.MustScale("C", "major pentatonic")
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")
	chord := chords[0]

	glass := pulsar.NewPoly(pulsar.GlassPreset(), 8).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)

	// the chord changes at every bar: an event (step 9) sets it, the streams read it
	arrangement := arrange.NewArrangement(form).
		OnBar(func(pos arrange.Position) { chord = chords[pos.Bar%4] }).
		PrintSections(true)
	arrangement.AddTo(p)

	// a written part for contrast: the pad holds the chords
	form.Part(p, pad, "pad", score.Config{Amp: 0.12, Legato: 1}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note { return []score.Note{score.Chord(chords[bar%4].Close(55), 4)} })
	})

	// the glass: a step or two up or down the pentatonic scale at a time, in a rhythm of 8ths,
	// 16ths and rests; its formant from the section
	pitch := 76.0
	melody := &arrange.Stream{
		Rhythm: choose(0.5, 0.5, 0.25, 0.25, 1.0, -0.5),
		Message: template.Template{
			"pitch": function.New(func() float64 {
				pitch = scale.Step(pitch, psalm.Rand().IntN(5)-2)
				if pitch < 67 || pitch > 88 {
					pitch = 76
				}
				return pitch
			}),
			"formant": template.NewParameter("formant", 1200.0),
		},
		Amp:    choose(0.3, 0.4, 0.5),
		Legato: 0.8,
	}
	form.Stream(p, glass, "glass", func(s arrange.Section) *arrange.Stream { return melody })

	// the bass: a dotted rhythm on the root of the chord of the bar
	form.Stream(p, bass, "bass", func(s arrange.Section) *arrange.Stream {
		return &arrange.Stream{
			Rhythm:  sequence.NewLoop(1.5, 1.5, 1.0),
			Message: template.Template{"pitch": function.New(func() float64 { return chord.BassPitch(36) })},
			Amp:     0.3,
			Legato:  0.7,
		}
	})

	mix := mixer.New(p)
	mix.Channel("pad", pad, mixer.StripConfig{Volume: -3})
	glassChannel := mix.Channel("glass", glass, mixer.StripConfig{Volume: -3})
	mix.Channel("bass", bass, mixer.StripConfig{})

	// the glass drifts through the stereo field: a value stream into the pan input (2) of its channel
	pan := 0.0
	panStream := form.NewStream(p, "glass pan", func(s arrange.Section) *arrange.Stream {
		return &arrange.Stream{
			Output: arrange.Values,
			Rhythm: sequence.NewLoop(1.0),
			Message: template.Template{"value": function.New(func() float64 {
				pan = max(-0.8, min(0.8, pan+(psalm.Rand().Float64()-0.5)*0.4))
				return pan
			})},
		}
	})
	glassChannel.ValueIn(panStream, 0, 2)

	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
