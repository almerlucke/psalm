// Step 15: a drum machine.
//
// A step sequencer plays rows of steps: a row per sound, 16 steps to the bar. The steps live in a
// grid on the panel (step 14), so you can edit the beat on the page while it plays, and in code.
//
//	grid := stepseq.NewGrid(ctl, "Beat", []string{"kick", "snare", "hat"}, 16, 2) // rows, steps, patterns
//	seq := stepseq.New(p, grid, stepseq.Config{StepsPerBeat: 4, Swing: 0.56})
//	seq.Row("kick", stepseq.Sound(kit, "kick"))                                 // what a row plays
//	grid.Set(pattern, row, step, stepseq.Velocity, 0.8)                         // a step, from code
//
// Every step is more than on and off. Its fields:
//
//	velocity    how hard (0 is off)
//	chance      the chance it plays: random skips
//	roll        hits the step is split into (a ratchet), roll chance, roll ramp (the hits rising or falling)
//	nudge       a little early or late
//	pitch       semitones up or down
//	condition   when it plays: A:B (the Ath of every B times round), fill (only with the fill on), first
//
// A row can be shorter than the grid: it comes round sooner and drifts against the others
// (polymeter). Rows in a choke group cut each other off (the closed hihat cuts the open one).
// seq.Controls(ctl) puts swing, humanizing, the fill switch and a card per row (level, length, mute,
// solo) on the panel.
//
// The sequencer runs on the clock of the patch, like the parts of the form: here it plays along
// with a written bass part, and the arrangement switches the drums to pattern B for the chorus
// (an event, step 9: asked for in the bar before, since a new pattern starts when the one playing
// comes round). On the page: tap a step to switch it, drag to paint, right click (or long
// press) a step to edit its fields, tap a row's name for its tools. Copy next to the pattern tabs
// copies the pattern on view to another: A to B, then make B your own.
//
// A beat you like is kept with a preset: the folder for them is given to the panel, and the page
// saves the whole setup (every pattern, the swing, the levels) by name and loads it again. Download
// and Upload do the same with a file.
//
//	ctl.Presets("drum-presets")                                                 // Save and Load on the page
//	ctl.LoadPreset("Groove")                                                    // or from code
//
//	go run ./cmd/examples/tutorial/15-drum-machine          then open http://localhost:8087
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/controls/stepseq"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/sndfile"
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the panel's page")
	p := tutorial.New(100)

	// --- the kit: samples in a sound bank, played by a drum synth by name ---

	samples := map[string]string{
		"kick":  "kick/Cymatics - Humble Friday Kick - E.wav",
		"snare": "snare/Cymatics - Humble King Snare 1 - A.wav",
		"hat":   "hihat/Cymatics - Humble Closed Hihat 4.wav",
		"open":  "hihat/Cymatics - Humble Open Hihat 1.wav",
		"perc":  "percussion/Cymatics - Odyssey Percussion 3.wav",
	}
	rows := []string{"kick", "snare", "hat", "open", "perc"}
	bank := sndfile.SoundBank[float32]{}
	for name, file := range samples {
		bank[name] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/"+file, 4)
	}
	kit := drums.NewDrums(bank, 16, 2).AddTo(p)

	// --- the grid and the sequencer ---

	ctl := panel.New(p, "Drum machine")
	ctl.Section("Beat")
	grid := stepseq.NewGrid(ctl, "Beat", rows, 16, 2)
	seq := stepseq.New(p, grid, stepseq.Config{StepsPerBeat: 4, Swing: 0.56, Humanize: 3})
	seq.Row("kick", stepseq.Sound(kit, "kick")).
		Row("snare", stepseq.Sound(kit, "snare")).
		Row("hat", stepseq.Sound(kit, "hat"), stepseq.Choke(1)). // the closed hihat cuts the open one
		Row("open", stepseq.Sound(kit, "open"), stepseq.Choke(1)).
		Row("perc", stepseq.Sound(kit, "perc"), stepseq.Length(12)) // 12 steps against 16: it drifts
	seq.Controls(ctl)

	// --- the beat, written in code: A for the verse, B (busier) for the chorus ---

	on := func(pattern int, row string, velocity float64, steps ...int) {
		for _, s := range steps {
			grid.Set(pattern, grid.Row(row), s, stepseq.Velocity, velocity)
		}
	}
	set := func(pattern int, row string, step, field int, v float64) {
		grid.Set(pattern, grid.Row(row), step, field, v)
	}
	condition := func(name string) float64 {
		for i, c := range stepseq.Conditions {
			if c == name {
				return float64(i)
			}
		}
		return 0
	}
	const A, B = 0, 1
	on(A, "kick", 1, 0, 7, 10)
	on(A, "snare", 0.9, 4, 12)
	on(A, "hat", 0.6, 0, 2, 4, 6, 8, 10, 12, 14)
	on(A, "hat", 0.3, 3, 11)                                  // ghost notes
	on(A, "perc", 0.5, 2, 5, 9)                               // 12 steps long: it drifts against the bar
	set(A, "hat", 14, stepseq.Roll, 3)                        // a roll of 3 hits ...
	set(A, "hat", 14, stepseq.RollChance, 0.5)                // ... half of the time
	set(A, "perc", 5, stepseq.Chance, 0.5)                    // a random skip
	set(A, "perc", 9, stepseq.Pitch, 5)                       // a step higher
	set(A, "kick", 10, stepseq.Nudge, 0.15)                   // a little late
	on(A, "snare", 0.7, 15)                                   // a snare roll into the next bar ...
	set(A, "snare", 15, stepseq.Roll, 4)                      //
	set(A, "snare", 15, stepseq.RollRamp, 0.8)                // ... rising
	set(A, "snare", 15, stepseq.Condition, condition("fill")) // only with the fill switch on

	on(B, "kick", 1, 0, 6, 8, 11)
	on(B, "snare", 1, 4, 12)
	on(B, "hat", 0.6, 0, 1, 2, 4, 5, 6, 8, 9, 10, 12, 13)
	on(B, "open", 0.7, 14) // cut off by the next closed hihat
	on(B, "perc", 0.6, 1, 4, 7, 10)
	on(B, "snare", 0.6, 7)
	set(B, "snare", 7, stepseq.Condition, condition("2:2")) // every second time round

	// --- a written bass part along with it, on the same clock ---

	form := arrange.NewForm(
		arrange.Section{Name: "Verse", Bars: 4},
		arrange.Section{Name: "Chorus", Bars: 4},
	)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")
	form.Part(p, bass, "bass", score.Config{Amp: 0.3, Legato: 0.6}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			r := chords[bar%4].BassPitch(36)
			return score.Notes(0.5, r, 0, r, r+12, 0, r, r+7, 0)
		})
	})

	// the arrangement switches the drums: B for the chorus, A again for the verse. A new pattern
	// starts when the one playing comes round, and the sequencer works a step ahead: ask for it
	// during the last bar before, and it starts right on the bar line
	arrangement := arrange.NewArrangement(form).
		At("Verse", 3, func() { grid.SetPattern(B) }).
		At("Chorus", 3, func() { grid.SetPattern(A) }).
		PrintSections(true)
	arrangement.AddTo(p)

	// --- the mix, and the page ---

	mix := mixer.New(p)
	mix.Channel("drums", kit, mixer.StripConfig{Volume: -8})
	mix.Channel("bass", bass, mixer.StripConfig{})

	if !tutorial.Rendering() {
		if err := ctl.Presets("drum-presets"); err != nil {
			log.Fatal(err)
		}
		go func() {
			if err := ctl.Serve(*addr); err != nil {
				log.Fatal(err)
			}
		}()
		defer ctl.Close()
		fmt.Printf("the drum machine: http://%s\n", *addr)
	}
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
