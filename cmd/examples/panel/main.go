// Panel: control a patch from a web page while it plays.
//
// A pad and an arpeggio loop through a filter. The panel next to the patch gives them faders,
// knobs, toggles, buttons, a select and meters, and serves them as a page: open it in a browser on
// this computer, or (with -addr :8080) on a phone or tablet on the same network, and play.
//
//	go run ./cmd/examples/panel                  then open http://localhost:8080
//	go run ./cmd/examples/panel -addr :8080      reachable from other devices on the network
//	go run ./cmd/examples/panel -midi            with the first MIDI controller (-midi-device for another)
//
// With a MIDI controller, press MIDI learn on the page, tap a control and turn a knob: the knob
// moves the control from then on. -bindings keeps what you learned in a file for the next time.
//
// The controls are declared in code and bound to settings of the modules: a message key
// (panel.Message), a value input (panel.Input) or a function (panel.Func). The panel applies their
// changes on the audio thread, between two control ticks: the page (or any goroutine) never makes
// the audio wait.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
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
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/sndfile"
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	useMIDI := flag.Bool("midi", false, "move the controls from a MIDI controller (MIDI learn on the page)")
	device := flag.String("midi-device", "", "the raw MIDI device (empty is the first controller)")
	bindingsFile := flag.String("bindings", "panel-midi.json", "the file the MIDI bindings are kept in")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: 96, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// --- the patch: a pad and an arpeggio through a filter, a hall, sleigh bells ---

	form := arrange.NewForm(arrange.Section{Name: "Loop", Bars: 4})
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	bells := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-sleigh-bells-584.wav", 4), Speed: 1, Amp: 0.5})
	bells.AddTo(p)

	form.Part(p, pad, "pad", score.Config{Amp: 0.24, Legato: 1}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note { return []score.Note{score.Chord(chords[bar%4].Close(55), 4)} })
	})
	arpeggio := func(pattern harmony.Pattern) arrange.Score {
		arp := harmony.Arp{Pattern: pattern, Octaves: 2, Steps: 8}
		return func(s arrange.Section) []score.Note {
			return arrange.PerBar(s, func(bar int) []score.Note { return arp.Notes(chords[bar%4].Close(60)) })
		}
	}
	arpPart := form.Part(p, keys, "arpeggio", score.Config{Amp: 0.35, Velocity: 0.55}, arpeggio(harmony.Up))

	filter := filters.NewLadder(ladder.LowPass12, 2400, 0.2, 1.0)
	filter.AddTo(p)
	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Group("filtered", mixer.StripConfig{}, filter)
	padStrip := mix.Channel("pad", pad, mixer.StripConfig{Group: "filtered"})
	keysStrip := mix.Channel("keys", keys, mixer.StripConfig{Group: "filtered"})
	mix.Channel("bells", bells, mixer.StripConfig{Volume: -6})
	p.Out(mix.Master())

	// --- the panel ---

	ctl := panel.New(p, "Panel")

	ctl.Section("Filter")
	ctl.Fader("Cutoff", 200, 12000, panel.Exp, panel.Message(filter, "cutoff"), panel.Default(2400), panel.Unit("Hz"), panel.Smooth(0.05))
	ctl.Knob("Resonance", 0, 0.95, panel.Linear, panel.Message(filter, "resonance"), panel.Default(0.2))

	ctl.Section("Mix")
	ctl.Fader("Pad", -40, 6, panel.Linear, panel.Message(padStrip, "volume"), panel.Default(0), panel.Unit("dB"), panel.Smooth(0.03))
	ctl.Fader("Keys", -40, 6, panel.Linear, panel.Message(keysStrip, "volume"), panel.Default(0), panel.Unit("dB"), panel.Smooth(0.03))
	ctl.Knob("Hall", 0, 1, panel.Linear, panel.Targets(panel.Message(padStrip, "send.hall"), panel.Message(keysStrip, "send.hall")), panel.Default(0.3))
	ctl.Meter("Pad out", pad)
	ctl.Meter("Keys out", keys)
	ctl.Meter("Master", mix.Master())

	ctl.Section("Play")
	ctl.Fader("Tempo", 60, 160, panel.Linear, panel.Func(func(v float64) { p.Env().SetBPM(v) }), panel.Default(96), panel.Step(1), panel.Unit("BPM"))
	ctl.Toggle("Mute keys", panel.Message(keysStrip, "mute"))
	ctl.Button("Bells", bells.Bang)
	patterns := []harmony.Pattern{harmony.Up, harmony.Down, harmony.UpDown, harmony.RandomWalk}
	ctl.Select("Arpeggio", []string{"up", "down", "up and down", "random"}, func(i int) {
		arpPart.SetNotes(form.Compile("arpeggio", arpeggio(patterns[i])))
	})

	// --- a MIDI controller: its knobs and faders move the controls bound to them ---

	if *useMIDI {
		d, err := midi.OpenDevice(*device)
		if err != nil {
			log.Fatalf("%v (go run ./cmd/examples/keyboard -list shows the devices)", err)
		}
		defer d.Close()
		fmt.Printf("MIDI from %s\n", d.Path())
		kb := keyboard.New(d, keyboard.Config{Controls: map[int]string{}}) // no notes to play, only its controllers
		kb.AddTo(p)
		ctl.MIDI(kb)
		// the bindings of the last time, saved again every time they change
		if err := ctl.KeepBindings(*bindingsFile); err != nil {
			log.Fatal(err)
		}
	}

	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close() // the page's connections end and the address is free again
	fmt.Printf("open http://%s (Ctrl-C to stop)\n", *addr)
	_ = p.Listen()
}
