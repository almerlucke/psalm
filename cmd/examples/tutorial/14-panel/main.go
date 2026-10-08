// Step 14: a control panel.
//
// Everything so far was decided in code before the song played. A panel lets you change things
// while it plays: faders, knobs, toggles, buttons and selects on a web page, bound to settings of
// the modules, and meters showing levels. Run this step and open http://localhost:8080 in a
// browser (start it with -addr :8080 to open it on a phone or tablet on the same network).
//
//	ctl := panel.New(p, "Tutorial")
//	ctl.Section("Filter")                                       the controls after it are shown together
//	ctl.Fader("Cutoff", 200, 12000, panel.Exp, target, options) a fader over a range
//	ctl.Knob(...)                                               the same as a knob
//	ctl.Toggle("Mute bass", target)                             on (1) and off (0)
//	ctl.Button("Bells", bells.Bang)                             calls a function when pressed
//	ctl.Select("Arpeggio", names, func(i int) { ... })          one of some options
//	ctl.Meter("Master", mix.Master())                           the level of a module
//	go ctl.Serve("localhost:8080")                              the page
//
// A fader moves by a curve: panel.Linear in even steps (levels in dB, amounts), panel.Exp in even
// steps per octave (frequencies). Options: panel.Default (where it starts, and where a double click
// returns), panel.Unit, panel.Step (whole numbers), panel.Smooth (glide to a new value instead of
// jumping, so a quick move on a cutoff doesn't step).
//
// A control sends its value to a target: panel.Message(module, "cutoff") as a setting by name,
// panel.Input(module, 1) to a value input, panel.Func(f) to your own function, panel.Targets(...)
// to several.
//
// The threads: the page runs on its own goroutines, the sound on the audio thread. The panel is a
// module in the patch: it applies the changes between two control ticks, so the audio never waits
// for the page. Button and select functions run on the audio thread too: keep them short, and
// start anything slow (like saving a file) on a goroutine, as the Save and Load buttons below do.
//
// Snapshots: ctl.Save and ctl.Load keep the values of all controls in a file, a preset of the
// patch. MIDI: with -midi the knobs of a MIDI keyboard or controller can move the controls: press
// MIDI learn on the page, tap a control and turn a knob.
//
// The panel and the arrangement work side by side: lanes (step 6) move the settings they automate,
// the panel the settings you give it. Give a setting to one of them, not both, or they take turns
// moving it.
//
//	go run ./cmd/examples/tutorial/14-panel
//	go run ./cmd/examples/tutorial/14-panel -midi
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/sndfile"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "the address of the panel's page")
	useMIDI := flag.Bool("midi", false, "move the controls from the first MIDI keyboard or controller too")
	p := tutorial.New(96)

	// --- the music: a pad, an arpeggio and a bass, looping, through a filter and a hall ---

	form := arrange.NewForm(arrange.Section{Name: "Loop", Bars: 4})
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	bells := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-sleigh-bells-584.wav", 4), Speed: 1, Amp: 0.5})
	bells.AddTo(p)

	form.Part(p, pad, "pad", score.Config{Amp: 0.24, Legato: 1}, arrange.Bars("Loop", func(_ arrange.Section, bar int) []score.Note {
		return []score.Note{score.Chord(chords[bar%4].Close(55), 4)}
	}))
	arpeggio := func(pattern harmony.Pattern) arrange.Score {
		arp := harmony.Arp{Pattern: pattern, Octaves: 2, Steps: 8}
		return arrange.Bars("Loop", func(_ arrange.Section, bar int) []score.Note { return arp.Notes(chords[bar%4].Close(60)) })
	}
	arpPart := form.Part(p, keys, "arpeggio", score.Config{Amp: 0.35, Velocity: 0.55}, arpeggio(harmony.Up))
	form.Part(p, bass, "bass", score.Config{Amp: 0.25, Legato: 0.7}, arrange.Bars("Loop", func(_ arrange.Section, bar int) []score.Note {
		r := chords[bar%4].BassPitch(36)
		return score.Notes(1, r, r, r+7, r+12)
	}))

	filter := filters.NewLadder(ladder.LowPass12, 2400, 0.2, 1.0)
	filter.AddTo(p)
	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Group("filtered", mixer.StripConfig{}, filter)
	padStrip := mix.Channel("pad", pad, mixer.StripConfig{Group: "filtered"})
	keysStrip := mix.Channel("keys", keys, mixer.StripConfig{Group: "filtered"})
	bassStrip := mix.Channel("bass", bass, mixer.StripConfig{})
	mix.Channel("bells", bells, mixer.StripConfig{Volume: -6})

	// --- the panel ---

	ctl := panel.New(p, "Tutorial")

	ctl.Section("Filter")
	// an exponential fader for a frequency, gliding over 50 milliseconds
	ctl.Fader("Cutoff", 200, 12000, panel.Exp, panel.Message(filter, "cutoff"), panel.Default(2400), panel.Unit("Hz"), panel.Smooth(0.05))
	ctl.Knob("Resonance", 0, 0.95, panel.Linear, panel.Message(filter, "resonance"), panel.Default(0.2))

	ctl.Section("Mix")
	// levels in dB: linear faders on the volumes of the mixer channels
	level := func(name string, strip *mixer.Strip) {
		ctl.Fader(name, -40, 6, panel.Linear, panel.Message(strip, "volume"), panel.Default(0), panel.Unit("dB"), panel.Smooth(0.03))
	}
	level("Pad", padStrip)
	level("Keys", keysStrip)
	level("Bass", bassStrip)
	// one knob, two targets: the hall send of the pad and of the keys
	ctl.Knob("Hall", 0, 1, panel.Linear, panel.Targets(panel.Message(padStrip, "send.hall"), panel.Message(keysStrip, "send.hall")), panel.Default(0.3))
	ctl.Meter("Master", mix.Master())

	ctl.Section("Play")
	// your own function: the tempo of the patch, in whole beats per minute
	ctl.Fader("Tempo", 60, 160, panel.Linear, panel.Func(func(v float64) { p.Env().SetBPM(v) }), panel.Default(96), panel.Step(1), panel.Unit("BPM"))
	ctl.Toggle("Mute bass", panel.Message(bassStrip, "mute"))
	ctl.Button("Bells", bells.Bang)
	// a select changing what a part plays, while it plays
	patterns := []harmony.Pattern{harmony.Up, harmony.Down, harmony.UpDown, harmony.RandomWalk}
	ctl.Select("Arpeggio", []string{"up", "down", "up and down", "random"}, func(i int) {
		arpPart.SetNotes(form.Compile("arpeggio", arpeggio(patterns[i])))
	})

	ctl.Section("Preset")
	// buttons run on the audio thread: the file work goes to a goroutine
	const presetFile = "tutorial-preset.json"
	ctl.Button("Save", func() {
		go func() {
			if err := ctl.Save(presetFile); err != nil {
				log.Print(err)
			} else {
				fmt.Println("saved", presetFile)
			}
		}()
	})
	ctl.Button("Load", func() {
		go func() {
			if err := ctl.Load(presetFile); err != nil {
				log.Print(err)
			}
		}()
	})

	// --- MIDI: the knobs of a keyboard or controller move the controls they are bound to ---

	if *useMIDI {
		d, err := midi.OpenDevice("")
		if err != nil {
			log.Fatalf("%v (go run ./cmd/examples/keyboard -list shows the devices)", err)
		}
		defer d.Close()
		kb := keyboard.New(d, keyboard.Config{Controls: map[int]string{}}) // its controllers only
		kb.AddTo(p)
		ctl.MIDI(kb)
		if err := ctl.KeepBindings("tutorial-midi.json"); err != nil { // what you learn, kept for next time
			log.Fatal(err)
		}
	}

	if !tutorial.Rendering() { // a rendered file has no one to move the controls
		go func() {
			if err := ctl.Serve(*addr); err != nil {
				log.Fatal(err)
			}
		}()
		defer ctl.Close() // when the program stops: the pages' connections end, the address is free
		fmt.Printf("the panel: http://%s\n", *addr)
	}

	tutorial.Play(p, mix.Master(), 4*4*60/96.0*2)
}
