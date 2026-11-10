// Noah: a score for a soundscape, local/HeavenlyNoah.wav (six minutes and twenty-four seconds: Noah's ark on the water, rain, thunder, the animals, bells in D and at the
// end a voice). A streamer plays the recording from its first second to its last, and under it a sparse, light music plays that follows what the recording does: a
// piano (the melody, gentle phrases with long rests), a kalimba, a bell marimba, a carillon, a soft handbell, a soft chime and a vibe bell, high bowls, nothing bowed, nothing heavy. Six sections at 60 bpm in 4/4, 96 bars of four
// seconds: Water (the ark creaks), Rain (a drop of music in the rain), Bells (the music answers the bells, in D), Beasts (the animals walk; the scale has no
// third), Storm (the music stands back for the thunder and moves down to C) and Voice (almost nothing; the music is a third of a semitone sharp, as the voice is).
//
// The recording is not in the repository (local/ is not): put it there, or name it with -sound. The scripts are in scripts/ (embedded); in the page double tap a
// script to read it and change it while it plays (Ctrl+Enter takes it).
//
//	go run ./cmd/songs/noah
//	go run ./cmd/songs/noah -render noah.wav -seed 7      render it to a file (-stems: the parts too)
//	go run ./cmd/songs/noah -sound other.wav              another recording
package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
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
	renderTo := flag.String("render", "", "render the piece once to this WAV file instead of playing it")
	profile := flag.Bool("profile", false, "play the piece offline and report the time every module takes")
	stems := flag.Bool("stems", false, "with -render, also write the parts to their own files")
	seed := flag.Uint64("seed", 0, "seed the random choices (0: different every time); with it a render is the same every time")
	flag.StringVar(&soundPath, "sound", soundPath, "the recording the music is made for")
	flag.Parse()

	if _, err := os.Stat(soundPath); err != nil {
		log.Fatalf("the recording is not there (%v): put HeavenlyNoah.wav in local/, or name the file with -sound", err)
	}
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 48000, BufferSize: 256})
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
	ctl := panel.New(p, "Noah")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Audio: p.PatchInput(0)}), patcher.Theme(t))
	ids, err := buildPatch(pt, *renderTo != "" || *profile, *seed)
	if err != nil {
		log.Fatal(err)
	}
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: 1})
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
		stats, err := render.Stems(p, *renderTo, render.Options{Seconds: tourSeconds(), Tail: 6}, sources)
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
