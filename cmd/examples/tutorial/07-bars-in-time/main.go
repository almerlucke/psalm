// Step 7: curves at bars.
//
// A curve runs over its whole section. Music thinks in bars instead: "open the filter over bars
// 3 and 4", "the last two bars", "a pad that comes in at bar 3". A Span measures a section in bars
// and places curves there (bars count from 0, fractions are fine):
//
//	span := form.Span("A")            the 8 bars of section A (arrange.Span(8) works too)
//	span.Between(2, 4, curve)         the curve from bar 2 to bar 4, its first value before and its last after
//	span.From(4, curve)               from bar 4 to the end of the section
//	span.Last(2, curve)               over the last 2 bars
//	span.Steps(0.4, 0.6, 0.8, 1)      a value per bar, the last one holding on
//	span.On(4, 6)                     1 from bar 4 to bar 6, 0 elsewhere: a switch
//	span.Fade(-8, 2, 7)               a level in dB from bar 2 to bar 7, faded in and out, silent outside
//	span.Pedal(0, 8)                  a sustain pedal changed at every bar line, the way a pianist does
//
// Here a felt piano plays chords with its pedal changed at every bar, an arpeggio grows bar by bar
// and is switched off for two bars, a pad fades in and out within the first section, and a filter
// opens over two bars of the first section and closes over the last two of the second.
//
//	go run ./cmd/examples/tutorial/07-bars-in-time
package main

import (
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
)

func main() {
	p := tutorial.New(90)

	form := arrange.NewForm(
		arrange.Section{Name: "A", Bars: 8},
		arrange.Section{Name: "B", Bars: 8},
	)
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")

	felt := piano.NewPoly(piano.FeltPreset(), 16).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 8).AddTo(p)
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)

	// the piano: a chord on every beat, short; the pedal (below) lets them ring through the bar
	form.Part(p, felt, "piano", score.Config{Amp: 0.4, Velocity: 0.5}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			v := append([]float64{chords[bar%4].BassPitch(36)}, chords[bar%4].Close(55)...)
			return score.Shape(score.Repeat([]score.Note{score.Chord(v, 1)}, 4), score.Hold(0.3))
		})
	})
	arp := harmony.Arp{Pattern: harmony.Up, Octaves: 2, Steps: 8}
	arpPart := form.Part(p, keys, "arpeggio", score.Config{Amp: 0.3, Velocity: 0.5}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note { return arp.Notes(chords[bar%4].Close(67)) })
	})
	form.Part(p, pad, "pad", score.Config{Amp: 0.2, Legato: 1}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			return []score.Note{score.Chord(chords[bar%4].Close(48), 4)}
		})
	})

	filter := filters.NewLadder(ladder.LowPass12, 500, 0.2, 1.0)
	filter.AddTo(p)
	mix := mixer.New(p)
	mix.Group("filtered", mixer.StripConfig{}, filter)
	mix.Channel("piano", felt, mixer.StripConfig{Group: "filtered"})
	keysChannel := mix.Channel("keys", keys, mixer.StripConfig{Group: "filtered"})
	padChannel := mix.Channel("pad", pad, mixer.StripConfig{Group: "filtered", Volume: -120})

	a, b := form.Span("A"), form.Span("B")
	arrangement := arrange.NewArrangement(form).
		// the pedal changes at every bar line of both sections
		AutomateMessage(arrange.Sections{"A": a.Pedal(0, 8), "B": b.Pedal(0, 8)}, felt, "sustain").
		// the filter opens over bars 2 and 3 of A, and closes over the last two bars of B
		AutomateMessage(arrange.Sections{
			"A": a.Between(2, 4, arrange.Exp(500, 12000)),
			"B": b.Last(2, arrange.Exp(12000, 500)),
		}, filter, "cutoff").
		// the arpeggio grows bar by bar in A
		AutomateFunc(arrange.Sections{"A": a.Steps(0.3, 0.5, 0.7, 0.9, 1)}.Else(1), func(v float64) { arpPart.SetGain(v) }).
		// and is switched off (muted) for bars 4 and 5 of B
		AutomateMessage(arrange.Sections{"B": b.On(4, 6)}, keysChannel, "mute").
		// the pad comes in at bar 2 of A and goes at bar 7
		AutomateMessage(arrange.Sections{"A": a.Fade(-6, 2, 7)}.Else(arrange.Silent), padChannel, "volume").
		PrintSections(true)
	arrangement.AddTo(p)

	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
