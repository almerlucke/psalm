// Spiritus: a piece for the Holy Spirit, who works in us to make us more like Jesus (from glory to glory, 2 Corinthians 3:18).
// Everything that sounds is granular: nothing is played by a synth that is heard. Two choirs of chants (grains of the
// Ad te levavi, the Factus est repente of Pentecost, the Epistle of Easter and the Crux fidelis, taken from the places
// where the singers rest on their home note, so the chords are in tune), a cloud of grains of a pad and a sparkle
// of grains of bells (two synths that play into grain buffers and are not on the mixer), a drone of grains of a sine and
// the tongues of fire (short sine grains in the pentatonic scale). Every grain has a lowpass filter that a slow lfo opens and
// closes (the grain filter and the modulation matrix of the grains synth), the high choir echoes itself softly (the feedback), and
// there is only a hall, no shimmer. Gentle, slow, in D major; the Glory lifts a whole step
// to E major. Six sections at 60 bpm, about six minutes: Waiting (a drone, a chant far off), Wind (a sound from heaven:
// the chant swells), Fire (flames on every one of them: bells and sparks), Indwelling (the Spirit in us: the choir of
// the Crux, a heartbeat of grains), Glory (the lift, from glory to glory) and Amen (the chant of the beginning, and rest).
//
// The scripts are in scripts/ (embedded); in the page double tap a script to read it and change it while
// it plays (Ctrl+Enter takes it), the song's tiles play a section next.
//
//	go run ./cmd/songs/spiritus
//	go run ./cmd/songs/spiritus -render spiritus.wav -seed 7      render it to a file (-stems: the parts too)
//	go run ./cmd/songs/spiritus -profile                          the time every module takes
//	go run ./cmd/songs/spiritus -theme midnight -addr localhost:8088
package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/tools/render"
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	renderTo := flag.String("render", "", "render the tour once to this WAV file instead of playing it")
	profile := flag.Bool("profile", false, "play the piece offline and report the time every module takes")
	stems := flag.Bool("stems", false, "with -render, also write the parts to their own files")
	seed := flag.Uint64("seed", 0, "seed the random choices (0: different every time); with it a render is the same every time")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	if *seed != 0 {
		psalm.Seed(*seed)
	}
	var p *psalm.Psalm
	if *renderTo != "" || *profile {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	ctl := panel.New(p, "Spiritus")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Theme(t))
	ids, err := buildPatch(pt, *renderTo != "" || *profile)
	if err != nil {
		log.Fatal(err)
	}
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: 3.2})
	p.Out(master.Master())

	if *profile {
		fmt.Print(render.RunProfiled(p, render.Options{Seconds: tourSeconds(), Tail: 2}))
		return
	}
	if *renderTo != "" {
		var sources map[string]psalm.Modular
		if *stems {
			sources = map[string]psalm.Modular{}
			for part, id := range ids.synths {
				sources[part] = pt.Module(id)
			}
		}
		stats, err := render.Stems(p, *renderTo, render.Options{Seconds: tourSeconds(), Tail: 12}, sources)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("rendered %.0f s to %s\n", stats.Seconds, *renderTo)
		for _, s := range ids.scripts {
			if st := pt.Status(s); strings.HasPrefix(st, "error") {
				log.Fatalf("script %s: %s", s, st)
			}
		}
		return
	}

	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(master.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03)).Top()
	widgets.Readout(ctl, "Time", pt.SongClock).Top()
	ctl.Meter("Out", master.Master()).Top()
	widgets.Scope(ctl, "Scope", master.Master()).Top()
	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close()
	log.Printf("open http://%s", *addr)
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}
