// Ceilidh: an electro folk piece made from three jigs (assets/midi/jig-1.mid, jig-2.mid, jig-3.mid: 6/8, G major, 165 bpm,
// 32, 32 and 48 bars). The program reads the files and puts their notes in the score script; the other scripts
// play them: an electronic flute (the tunes as they are written) and a tin whistle (an octave up, or alone), a mandolin that
// chops the chords, a bodhran, and under them the electro side: a drum machine kit, a sub bass, an arpeggio, a pad, a
// saw lead that doubles the flute and the joins (hits and risers). A hurdy-gurdy drones a G and a D.
// The chords of every bar are written in the score (shared.chordAt), the layers of every section too
// (shared.level), so that the form can be read in one place. Seven sections in 6/8 at 150 bpm, 144 bars, about three
// minutes: Hearth (the first jig on a whistle over a drone), First (the first jig, the band comes in layer by layer),
// Break (the chords alone, a build), Second (the second jig, a synth doubles it), Bridge (a whistle, the third jig
// far off, a riser), Third (the third jig, everything) and Outro (the first jig again, the layers leave).
//
// The scripts are in scripts/ (embedded); in the page double tap a script to read it and change it while
// it plays (Ctrl+Enter takes it), the song's tiles play a section next.
//
//	go run ./cmd/songs/ceilidh
//	go run ./cmd/songs/ceilidh -render ceilidh.wav -seed 7      render it to a file (-stems: the parts too)
//	go run ./cmd/songs/ceilidh -profile                         the time every module takes
//	go run ./cmd/songs/ceilidh -theme midnight -addr localhost:8088
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
	ctl := panel.New(p, "Ceilidh")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Audio: p.PatchInput(0)}), patcher.Theme(t))
	ids, err := buildPatch(pt, *renderTo != "" || *profile)
	if err != nil {
		log.Fatal(err)
	}
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: 2})
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
		stats, err := render.Stems(p, *renderTo, render.Options{Seconds: tourSeconds(), Tail: 8}, sources)
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
