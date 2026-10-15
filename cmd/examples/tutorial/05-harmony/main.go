// Step 5: checking parts against the chords.
//
// When you write parts in code, a wrong note is easy to miss. harmony.Check compares the notes of a
// part with the chords under them and reports the ones that clash, before anything plays:
//
//   - a note a semitone above a chord tone (an "avoid note": F over C major rubs against its E),
//     unless it is a passing or neighbour note: short (a beat at most) and reached or left by a step
//   - a note held into the next chord where it ends up a semitone from a chord tone
//
// Other notes outside the chord (sevenths, ninths, sixths) are colour, not mistakes.
//
// harmony.Timeline lays out the chords (4 beats each here), form.Compile gives the notes of a part
// over the whole song, harmony.Report prints what Check found. The song below plays the melody
// twice: first with two wrong notes, then corrected. The program prints the two it found:
//
//	melody (wrong): beat 1: F5 against C, a semitone above a chord tone
//	melody (wrong): beat 12: C5 against G, a semitone above a chord tone
//
// Listen for the rubs in the first section, and for the quick F in the first bar of the second:
// the same note, but short and between two Es, a neighbour note the check lets through.
//
//	go run ./cmd/examples/tutorial/05-harmony
package main

import (
	"fmt"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/tools/harmony"
)

func main() {
	p := tutorial.New(80)

	form := arrange.NewForm(
		arrange.Section{Name: "Wrong", Bars: 4},
		arrange.Section{Name: "Right", Bars: 4},
	)
	chords := harmony.MustKey("C major").MustProgression("I vi IV V") // C Am F G, a bar each
	timeline := harmony.Timeline(4, chords...)                        // repeats when the song is longer

	keys := dx.NewPoly(dx.EPianoPreset(), 8, psalm.Stereo).AddTo(p)
	lead := dx.NewPoly(dx.FlutePreset(), 2, psalm.Stereo).AddTo(p)

	form.Part(p, keys, "chords", score.Config{Amp: 0.2, Velocity: 0.45, Legato: 1}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			return []score.Note{score.Chord(chords[bar%4].Close(55), 4)}
		})
	})

	// the melody, with two wrong notes held over the chord: F5 over C, C5 over G
	wrong := arrange.Only("Wrong", func(s arrange.Section) []score.Note {
		return score.MustParse(`E5:1 F5:2 G5:1 | E5:2 C5 | A4:1 B4:.5 C5:.5 F5:2 | C5:2 B4`)
	})
	// corrected: E5 and B4. The F5 is back in the first bar, but short and between two Es: a
	// neighbour note, which the check lets through
	right := arrange.Only("Right", func(s arrange.Section) []score.Note {
		return score.MustParse(`E5:1 F5:.5 E5:.5 G5:2 | E5:2 C5 | A4:1 B4:.5 C5:.5 F5:2 | B4:2 B4`)
	})

	clashes := 0
	for name, sc := range map[string]arrange.Score{"melody (wrong)": wrong, "melody (right)": right} {
		form.Part(p, lead, name, score.Config{Amp: 0.4, Velocity: 0.6}, sc)
		issues := harmony.Check(form.Compile(name, sc), timeline)
		clashes += len(issues)
		fmt.Print(harmony.Report(name, issues))
	}
	fmt.Printf("%d notes clash\n", clashes)

	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)

	mix := mixer.New(p)
	mix.Channel("keys", keys, mixer.StripConfig{})
	mix.Channel("lead", lead, mixer.StripConfig{})
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
