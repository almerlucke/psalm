// Step 16: the transport, playing the song live.
//
// A form plays its sections in order. Its transport is the song clock behind that: it says where
// the song is and decides which section comes next, and you can change that while it plays. Every
// part of the form follows it: the parts, the streams, the lanes, the tempo and the events.
//
//	tr := form.Transport()
//	tr.SetEnabled("Bridge", false)   a section that is off is skipped (until you turn it on)
//	tr.Queue("Chorus")               play the chorus next
//	tr.SetRepeat(true)               play the section playing again and again
//	tr.SetJump(arrange.AtBar)        a queued section starts at the next bar (instead of when the
//	                                 section playing ends)
//	tr.Controls(ctl, "Song")         all of it on the panel (step 14)
//
// The parts of the song can be switched off and on while it plays, the way a mixer's channels are
// muted, but in time: a part that is off starts no new notes (what sounds rings out), and keeps its
// place, so it comes back on the beat. It stays off until it is switched on again.
//
//	form.SetPartEnabled("bass", false)
//	form.PartControls(ctl, "Parts")  a tile per part: tap to switch it; the ones playing in this section light up
//
// On the page a tile per section: tap one to play it next (tap it again to take that back), tap its
// dot to turn it on or off. The section playing shows how far it is, the one queued blinks. Repeat
// holds the section playing, Jump at says when a queued section starts. Jumps land on a bar line,
// so the song stays in time; a section always starts from its first bar.
//
// Restart in the page's header goes back to the start at once; the Again button below asks for the
// first section the musical way, at the end of the section playing (or the next bar).
//
// This song has a bridge that is off: it plays only when you turn it on, or tap it. Duration and
// Schedule leave the sections that are off out, so a render plays the song without the bridge.
// Which sections are on is kept in presets, with everything else on the panel.
//
// The pads of a MIDI controller can play the sections: every tile is a button of its own on the
// panel ("Song/Chorus"). Start with -midi, press MIDI learn on the page, tap a tile and hit a pad (a
// note, or a controller); the bindings are kept in transport-midi.json. In code:
//
//	ctl.Param("Song/Chorus").BindNote(0, 36)                       a note on any channel
//
//	go run ./cmd/examples/tutorial/16-transport           then open http://localhost:8087
//	go run ./cmd/examples/tutorial/16-transport -midi     with the first MIDI controller
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
)

type (
	S = arrange.Section
	N = score.Note
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the panel's page")
	useMIDI := flag.Bool("midi", false, "play the sections from the pads of the first MIDI controller (MIDI learn on the page)")
	p := tutorial.New(100)

	// --- the form: the bridge is written, but off ---

	form := arrange.NewForm(
		S{Name: "Intro", Bars: 4},
		S{Name: "Verse", Bars: 8},
		S{Name: "Chorus", Bars: 8, Dynamics: 1.15},
		S{Name: "Bridge", Bars: 4, Dynamics: 0.85},
		S{Name: "Outro", Bars: 4},
	)
	tr := form.Transport().SetEnabled("Bridge", false)

	// the chords: I vi IV V, and the bridge goes somewhere else
	key := harmony.MustKey("C major")
	song, bridge := key.MustProgression("I vi IV V"), key.MustProgression("IV V iii vi")
	chordOf := func(s S, bar int) harmony.Chord {
		if s.Name == "Bridge" {
			return bridge[bar%4]
		}
		return song[bar%4]
	}

	// --- the instruments and their parts ---

	keys := piano.NewPoly(piano.WarmPreset(), 16).AddTo(p)
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	lead := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	cfg := score.Config{Velocity: 0.6, Feel: score.Feel{Timing: 8, Velocity: 0.08}}

	form.Part(p, keys, "piano", cfg, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := chordOf(s, bar)
			v := append([]float64{c.BassPitch(36)}, c.Close(55)...)
			if s.Name == "Outro" && bar == 3 {
				return []N{score.Chord(v, 4)}
			}
			if s.Name == "Intro" || s.Name == "Outro" || s.Name == "Bridge" {
				return []N{score.Chord(v, 2), score.Chord(c.Close(60), 2)} // calm: half notes
			}
			return score.Shape(score.Repeat([]N{score.Chord(v, 1)}, 4), score.Velocities(0.6, 0.4, 0.5, 0.4), score.Hold(0.4))
		})
	})
	form.Part(p, pad, "pad", score.Config{Amp: 0.5}, func(s S) []N {
		if !arrange.In(s, "Chorus", "Bridge") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordOf(s, bar).Close(60), 4)} })
	})
	form.Part(p, bass, "bass", cfg, func(s S) []N {
		if !arrange.In(s, "Verse", "Chorus") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			r := chordOf(s, bar).BassPitch(36)
			return score.Shape(score.Notes(1, r, 0, r+7, r+12), score.Velocities(0.8, 0, 0.6, 0.5))
		})
	})
	form.Part(p, lead, "melody", score.Config{Amp: 0.8, Feel: score.Feel{Timing: 10, Push: 6}}, func(s S) []N {
		switch s.Name {
		case "Verse":
			return score.MustParse(`E5:1 G5 C6:2 | E5:1 A5 C6:2 | A5:1 F5 C5:2 | D5:1 G5 B4:2 |
				E5:1 G5 C6:1.5 B5:.5 | A5:1 E5 C5:2 | F5:1 A5 C6:2 | B5:1 A5 G5:2`)
		case "Chorus":
			return score.MustParse(`G5:2 E5:1 C5 | A5:2 E5:1 C5 | F5:1 A5 C6:2 | B5:1 A5 G5:2 |
				G5:2 E5:1 C5 | A5:2 C6:1 A5 | F5:1 A5 C6:1 D6 | B5:2 G5:2`)
		case "Bridge":
			return score.MustParse(`A5:4 | B5:4 | G5:2 E5:2 | A5:4`)
		}
		return nil
	})

	// --- the arrangement: a ritardando at the end, the sections as they play ---

	arrangement := arrange.NewArrangement(form).
		Tempo(arrange.Sections{"Outro": arrange.Linear(100, 84)}).
		PrintSections(true)
	arrangement.AddTo(p)

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay = 2.4
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Channel("piano", keys, mixer.StripConfig{Volume: -10, Pan: -0.15, Sends: map[string]float64{"hall": 0.3}})
	mix.Channel("pad", pad, mixer.StripConfig{Volume: -20, Pan: 0.2, Sends: map[string]float64{"hall": 0.4}})
	mix.Channel("bass", bass, mixer.StripConfig{Volume: -14})
	mix.Channel("melody", lead, mixer.StripConfig{Volume: -12, Pan: 0.1, Sends: map[string]float64{"hall": 0.3}})

	fmt.Print(arrangement.Schedule()) // the bridge shows as off

	// --- the panel: the song, and a button that queues from code ---

	ctl := panel.New(p, "Transport")
	ctl.Section("Song")
	tr.Controls(ctl, "Song")
	ctl.Button("Again", func() { // the song from its first section that is on, at the jump of Jump at
		if order := tr.Order(); len(order) > 0 {
			tr.Queue(order[0].Name)
		}
	})
	ctl.Section("Parts")
	form.PartControls(ctl, "Parts")
	ctl.Section("Mix")
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Default(0), panel.Unit("dB"), panel.Smooth(0.03))
	ctl.Meter("Master", mix.Master())

	if *useMIDI {
		d, err := midi.OpenDevice("")
		if err != nil {
			log.Fatalf("%v (go run ./cmd/examples/keyboard -list shows the devices)", err)
		}
		defer d.Close()
		kb := keyboard.New(d, keyboard.Config{Controls: map[int]string{}}) // its pads and knobs only
		kb.AddTo(p)
		ctl.MIDI(kb)
		if err := ctl.KeepBindings("transport-midi.json"); err != nil {
			log.Fatal(err)
		}
	}
	if !tutorial.Rendering() {
		if err := ctl.Presets("transport-presets"); err != nil {
			log.Fatal(err)
		}
		go func() {
			if err := ctl.Serve(*addr); err != nil {
				log.Fatal(err)
			}
		}()
		defer ctl.Close()
		fmt.Printf("the song: http://%s\n", *addr)
	}
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
