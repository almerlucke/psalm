// Step 4: feel and dynamics.
//
// Notes exactly on the grid, all equally loud, sound like a machine. Three things make a part
// sound played:
//
// Accents, written in the notes: score.Shape gives a figure velocities in turn (strong, weak,
// medium, weak ...) and how long its notes sound (score.Hold), without a loop over the notes.
//
// Feel, set on the part (score.Config.Feel), changes the notes as they play:
//
//	Timing   ± milliseconds of random timing per note (a chord moves together)
//	Spread   ± milliseconds between the notes of a chord (an ensemble never starts exactly together)
//	Strum    milliseconds between the notes of a chord, low to high, like a guitar or a rolled chord
//	Swing    where the second 8th of a pair lands: 0.5 straight, 0.58 light, 0.66 a triplet feel
//	Push     milliseconds every note moves: negative plays ahead of the beat, positive lays back
//	Velocity ± random variation of the velocity and amplitude
//
// The random parts are drawn again every time the song loops: no two passes are the same (run it
// with -seed 3 to make them the same every run).
//
// Dynamics, set on a section (Section.Dynamics), scales the amplitude of every part in it: a
// quiet verse, a loud chorus, without touching the parts.
//
// The same groove plays four times: on the grid, played like a person, swung, and loud with
// strummed chords.
//
//	go run ./cmd/examples/tutorial/04-feel
package main

import (
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/tools/harmony"
)

func main() {
	p := tutorial.New(100)

	form := arrange.NewForm(
		arrange.Section{Name: "Grid", Bars: 4},
		arrange.Section{Name: "Human", Bars: 4},
		arrange.Section{Name: "Swing", Bars: 4},
		arrange.Section{Name: "Loud", Bars: 4, Dynamics: 1.5}, // every part 1.5 times as loud
	)

	chords := harmony.MustKey("C major").MustProgression("I vi IV V")
	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)

	// the groove: a chord on every 8th, accented on the beats (strong, weak, medium, weak ...),
	// short; the bass on the beat
	comp := func(bar int) []score.Note {
		chord := chords[bar%4].Close(60)
		return score.Shape(score.Repeat([]score.Note{score.Chord(chord, 0.5)}, 8),
			score.Velocities(0.7, 0.35, 0.55, 0.35), score.Hold(0.3))
	}
	bassline := func(bar int) []score.Note {
		r := chords[bar%4].BassPitch(36)
		return score.Shape(score.Notes(1, r, r, r+7, r+12), score.Velocities(0.8, 0.5, 0.6, 0.5))
	}

	// the same notes in every section, a part per feel: each plays in its own section only
	feels := map[string]score.Feel{
		"Grid":  {},
		"Human": {Timing: 10, Spread: 8, Velocity: 0.1},
		"Swing": {Swing: 0.64, Timing: 8, Velocity: 0.1},
		"Loud":  {Timing: 8, Strum: 25, Velocity: 0.1},
	}
	for _, s := range form.Sections {
		feel := feels[s.Name]
		form.Part(p, keys, "keys "+s.Name, score.Config{Amp: 0.18, Feel: feel}, arrange.Bars(s.Name, func(_ arrange.Section, bar int) []score.Note {
			return comp(bar)
		}))
		form.Part(p, bass, "bass "+s.Name, score.Config{Amp: 0.25, Legato: 0.7, Feel: feel}, arrange.Bars(s.Name, func(_ arrange.Section, bar int) []score.Note {
			return bassline(bar)
		}))
	}

	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)

	mix := mixer.New(p)
	mix.Channel("keys", keys, mixer.StripConfig{})
	mix.Channel("bass", bass, mixer.StripConfig{})
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
