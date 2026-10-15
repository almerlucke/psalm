// Step 2: the form, sections and parts.
//
// A song is a list of sections: an intro, a verse, a chorus, an outro. An arrange.Form holds them,
// each with a name and a number of bars (4 beats to the bar unless you set form.BeatsPerBar).
//
// A part is what one instrument plays through the whole song. You write it as a function that
// gets a section and returns the notes for it, or nil when the instrument is silent there.
// form.Part compiles that over all sections into one sequencer that loops with the song and
// connects it to the synth.
//
// Every section has to last exactly its bars: a part that writes 15 beats for a 4 bar section
// stops the program at the start with the name of the part and the section, instead of drifting
// out of time.
//
// Here the keys play chords everywhere, the bass enters in the verse and the melody plays only in
// the verse and the chorus (a different melody in each). An arrangement prints every section as it
// starts: more about arrangements in step 6.
//
//	go run ./cmd/examples/tutorial/02-form
package main

import (
	"fmt"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/subtractive"
)

func main() {
	p := tutorial.New(96)

	form := arrange.NewForm(
		arrange.Section{Name: "Intro", Bars: 2},
		arrange.Section{Name: "Verse", Bars: 4},
		arrange.Section{Name: "Chorus", Bars: 4},
		arrange.Section{Name: "Outro", Bars: 2},
	)

	keys := dx.NewPoly(dx.EPianoPreset(), 8, psalm.Stereo).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)

	// the chords of the song, a bar each: C Am F G
	progression := score.MustParse(`[C4 E4 G4]:4 | [C4 E4 A4]:4 | [C4 F4 A4]:4 | [B3 D4 G4]:4`)

	// the keys play in every section; the intro and the outro are 2 bars, half the progression
	form.Part(p, keys, "chords", score.Config{Amp: 0.2, Velocity: 0.5, Legato: 0.95}, func(s arrange.Section) []score.Note {
		if s.Bars == 2 {
			return progression[:2]
		}
		return progression
	})

	// the bass is silent (nil) in the intro: it enters with the verse
	form.Part(p, bass, "bass", score.Config{Amp: 0.35, Legato: 0.8}, func(s arrange.Section) []score.Note {
		switch s.Name {
		case "Intro":
			return nil
		case "Outro":
			return score.MustParse(`C2:4 | C2:4`)
		}
		return score.MustParse(`C2:2 C3:1 G2 | A1:2 A2:1 E2 | F1:2 F2:1 C2 | G1:2 G2:1 D2`)
	})

	// the melody: one for the verse, another for the chorus, silent elsewhere
	form.Part(p, keys, "melody", score.Config{Amp: 0.35, Velocity: 0.6}, func(s arrange.Section) []score.Note {
		switch s.Name {
		case "Verse":
			return score.MustParse(`E5:1 G5 C6:2 | E5:1 A5 C6:2 | A5:1 F5 C5:2 | D5:1 G5 B4:2`)
		case "Chorus":
			return score.MustParse(`G5:2@.75 E5:1 C5 | A5:2@.75 E5:1 C5 | F5:1 A5 C6:2@.8 | B5:1 A5 G5:2`)
		}
		return nil
	})

	// the schedule: where every section starts in time
	fmt.Print(form.Schedule())
	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)

	// two instruments: a mixer adds them up, a channel each (more about the mixer in step 10)
	mix := mixer.New(p)
	mix.Channel("keys", keys, mixer.StripConfig{})
	mix.Channel("bass", bass, mixer.StripConfig{})
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
