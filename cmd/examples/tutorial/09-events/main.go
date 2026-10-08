// Step 9: events, things that happen at a moment.
//
// Lanes change a value continuously. Some things happen once, at a moment: a sample starts, a part
// changes its feel, a channel comes in. The arrangement calls your functions for those:
//
//	At("Chorus", 0, bells.Bang)        at the start of bar 0 of the chorus, every time the song gets there
//	OnSection(func(pos Position) {...})  at the start of every section
//	OnBar(func(pos Position) {...})      at the start of every bar
//	Play(channel, "Chorus")            the channel sounds in the chorus only (muted with a short fade elsewhere)
//
// A Position tells where the song is: pos.Section (its Name, Bars, ...), pos.Bar (in the song,
// from 0), pos.SectionBar() (in the section), pos.Progress (0-1 through the section) and pos.Loop
// (how many times the song looped).
//
// Here sleigh bells (a recording) ring at the start of every chorus, the keys swing in the chorus
// and play straight in the verse (OnSection changes the feel of their part), a counter is printed
// at every bar, and the lead plays only in the chorus.
//
//	go run ./cmd/examples/tutorial/09-events
package main

import (
	"fmt"

	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/sndfile"
)

func main() {
	p := tutorial.New(104)

	form := arrange.NewForm(
		arrange.Section{Name: "Verse", Bars: 4},
		arrange.Section{Name: "Chorus", Bars: 4},
	)
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")

	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	lead := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	// a recording, played once from its start every time it gets a bang
	bells := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-sleigh-bells-584.wav", 4), Speed: 1, Amp: 0.5})
	bells.AddTo(p)

	keysPart := form.Part(p, keys, "keys", score.Config{Amp: 0.18}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			chord := score.Chord(chords[bar%4].Close(60), 0.5)
			return score.Shape(score.Repeat([]score.Note{chord}, 8), score.Velocities(0.7, 0.35, 0.55, 0.35), score.Hold(0.3))
		})
	})
	// the lead plays all the time, but its channel only sounds in the chorus (Play, below)
	form.Part(p, lead, "lead", score.Config{Amp: 0.35, Velocity: 0.6}, func(s arrange.Section) []score.Note {
		return score.MustParse(`G5:2 E5:1 C5 | A5:2 E5:1 C5 | F5:1 A5 C6:2 | B5:1 A5 G5:2`)
	})

	mix := mixer.New(p)
	mix.Channel("keys", keys, mixer.StripConfig{})
	leadChannel := mix.Channel("lead", lead, mixer.StripConfig{})
	mix.Channel("bells", bells, mixer.StripConfig{})

	arrangement := arrange.NewArrangement(form).
		// the bells ring at the start of every chorus
		At("Chorus", 0, bells.Bang).
		// the keys swing in the chorus, play straight in the verse
		OnSection(func(pos arrange.Position) {
			fmt.Printf("%s (loop %d)\n", pos.Section.Name, pos.Loop+1)
			if pos.Section.Name == "Chorus" {
				keysPart.SetFeel(score.Feel{Swing: 0.66, Timing: 6})
			} else {
				keysPart.SetFeel(score.Feel{})
			}
		}).
		// a counter at every bar
		OnBar(func(pos arrange.Position) { fmt.Printf("  bar %d of %d\n", pos.SectionBar()+1, pos.Section.Bars) }).
		// the lead's channel sounds in the chorus only
		Play(leadChannel, "Chorus")
	arrangement.AddTo(p)

	tutorial.Play(p, mix.Master(), 2*arrangement.Duration()) // twice round, to hear it loop
}
