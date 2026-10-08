// Step 8: tempo.
//
// The tempo is a lane too: arrangement.Tempo(lane) sets the tempo of the patch in beats per minute,
// and every sequencer follows it. Where the lane is 0 (a section without a curve) the tempo goes
// back to the tempo the patch started with, so you only write curves where the tempo changes:
//
//	Tempo(arrange.Sections{
//		"Slow":        arrange.Const(72),                                // slower
//		"Accelerando": arrange.Linear(72, 120),                          // speeding up
//		"End":         form.Span("End").Last(2, arrange.Smooth(120, 60)), // slowing down at the end
//	})
//
// The tempo is updated 48 times per beat, so a ritardando is smooth, and the parts stay together:
// the song counts in beats, not in seconds. arrangement.Schedule() prints when every section
// starts following the tempo lane, and arrangement.Duration() is the length of the song in seconds.
//
//	go run ./cmd/examples/tutorial/08-tempo
package main

import (
	"fmt"

	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/tools/harmony"
)

func main() {
	p := tutorial.New(120) // the base tempo

	form := arrange.NewForm(
		arrange.Section{Name: "Slow", Bars: 4},
		arrange.Section{Name: "Accelerando", Bars: 4},
		arrange.Section{Name: "Base", Bars: 4},
		arrange.Section{Name: "End", Bars: 4},
	)
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")

	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)

	form.Part(p, keys, "keys", score.Config{Amp: 0.18, Feel: score.Feel{Timing: 6}}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			if s.Name == "End" && bar == 3 {
				return []score.Note{score.Chord(harmony.MustChord("C").Close(60), 4)} // the last chord
			}
			chord := score.Chord(chords[bar%4].Close(60), 0.5)
			return score.Shape(score.Repeat([]score.Note{chord}, 8), score.Velocities(0.7, 0.35, 0.55, 0.35), score.Hold(0.3))
		})
	})
	form.Part(p, bass, "bass", score.Config{Amp: 0.25, Legato: 0.7}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			r := chords[bar%4].BassPitch(36)
			if s.Name == "End" && bar == 3 {
				return score.Notes(4, 36)
			}
			return score.Notes(1, r, r, r+7, r+12)
		})
	})

	arrangement := arrange.NewArrangement(form).
		Tempo(arrange.Sections{
			"Slow":        arrange.Const(72),
			"Accelerando": arrange.Linear(72, 120),
			"End":         form.Span("End").Last(2, arrange.Smooth(120, 60)),
		}).
		PrintSections(true)
	arrangement.AddTo(p)

	// the schedule follows the tempo lane: the slow section takes longer than the base one
	fmt.Print(arrangement.Schedule())

	mix := mixer.New(p)
	mix.Channel("keys", keys, mixer.StripConfig{})
	mix.Channel("bass", bass, mixer.StripConfig{})
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
