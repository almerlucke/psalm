// Step 6: automation, settings that change through the song.
//
// The parts so far play notes. An arrangement changes settings while the song plays: a filter
// opening over the intro, a channel fading in. It follows the song position and runs lanes.
//
// A lane gives a value for every moment: a curve per section, and a value for the sections without
// one. A curve is a function of the progress through its section (0 at the start, 1 at the end):
//
//	arrange.Const(400)          stays at 400
//	arrange.Linear(-30, 0)      straight from -30 to 0
//	arrange.Exp(400, 12000)     from 400 to 12000 in even steps per octave: right for frequencies
//	arrange.Smooth(12000, 400)  eases in and out
//	func(progress float64) float64 { ... }   or any function of your own
//
// arrange.Sections{"Rise": curve} is a lane: that curve in the Rise, 0 everywhere else, and
// .Else(12000) makes everywhere else 12000. Send a lane to:
//
//	AutomateMessage(lane, filter, "cutoff")   a setting by name ("volume" of a mixer channel, ...)
//	Automate(lane, filter, 1)                 a setting by its value input number
//	AutomateFunc(lane, func(v float64) {...}) your own function
//
// Values are updated 48 times per beat and only sent when they change. Here a lowpass filter on the
// pad and the keys closes, opens, stays open and closes again, the pad fades in over the first
// section, and the arpeggio's level follows its own lane through a function.
//
//	go run ./cmd/examples/tutorial/06-automation
package main

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
)

func main() {
	p := tutorial.New(96)

	form := arrange.NewForm(
		arrange.Section{Name: "Dark", Bars: 4},
		arrange.Section{Name: "Rise", Bars: 4},
		arrange.Section{Name: "Bright", Bars: 4},
		arrange.Section{Name: "Fall", Bars: 4},
	)
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")

	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	keys := dx.NewPoly(dx.EPianoPreset(), 8, psalm.Stereo).AddTo(p)

	form.Part(p, pad, "pad", score.Config{Amp: 0.24, Legato: 1}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			return []score.Note{score.Chord(chords[bar%4].Close(55), 4)}
		})
	})
	arp := harmony.Arp{Pattern: harmony.UpDown, Octaves: 2, Steps: 8}
	arpPart := form.Part(p, keys, "arpeggio", score.Config{Amp: 0.4, Velocity: 0.55}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note { return arp.Notes(chords[bar%4].Close(60)) })
	})

	// the pad and the keys play through a lowpass filter: a group on the mixer with the filter in it
	// (more about the mixer in step 10)
	filter := filters.NewLadder(ladder.LowPass12, 400, 0.3, 1.0)
	filter.AddTo(p)
	mix := mixer.New(p)
	mix.Group("filtered", mixer.StripConfig{}, filter)
	padChannel := mix.Channel("pad", pad, mixer.StripConfig{Group: "filtered"})
	mix.Channel("keys", keys, mixer.StripConfig{Group: "filtered"})

	arrangement := arrange.NewArrangement(form).
		// the filter: closed, opening, open (Else: the sections without a curve), closing
		AutomateMessage(arrange.Sections{
			"Dark": arrange.Const(400),
			"Rise": arrange.Exp(400, 12000),
			"Fall": arrange.Smooth(12000, 400),
		}.Else(12000), filter, "cutoff").
		// the pad fades in over the first section (in dB), then stays at 0 dB
		AutomateMessage(arrange.Sections{"Dark": arrange.Linear(-30, 0)}, padChannel, "volume").
		// your own function: the arpeggio louder in the bright section, softer elsewhere
		AutomateFunc(arrange.Sections{"Bright": arrange.Const(1)}.Else(0.6), func(v float64) { arpPart.SetGain(v) }).
		PrintSections(true)
	arrangement.AddTo(p)

	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
