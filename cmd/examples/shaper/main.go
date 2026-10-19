// Shaper: a piece of electronic music without drums, played by scripts, to hear what the shaper synth
// can do. Seven scripts: one holds the score (the chords of the piece, D minor with an Eb over the D and a lift to
// Bb major, a notation for lines), five play a shaper synth each: a sequencer (a short cycle of notes with rests
// in it, a cycle of seven against sixteen steps), an arpeggio (a euclidean rhythm turned a step every bar), a pad
// (voice led chords), a lead (a motif of four notes with silence round it, a melody in the lift) and glass (a few
// bells); a sixth plays the joins (hits and risers). Each rewrites its synth for every section with set(): a
// preset of the synth, then the macros, the morph and the filter. The effects are on the mixer, four buses: a hall, an
// echo of a dotted eighth, a granulator and a shimmer. Eight sections at 100 bpm, about five and a half minutes:
// Drone (an airy pad, bells, a kalimba far off), Pulse (the sequencer comes in note by note, the lead at the end),
// Glow (a gentle melody over a climbing pulse), Depth (everything, an octave higher in the second half), Hollow (a choir far
// off, the pulse gone), Lift (the relative major, a melody, a harp five against sixteen), Bloom (the high one: a melody in
// the upper octave, a marimba nine against sixteen) and Dissolve (the layers leave one by one and the
// granulator keeps their grains).
//
// The scripts are in scripts/ (embedded); in the page double tap a script to read it and change it while
// it plays (Ctrl+Enter takes it), the song's tiles play a section next. Every setting of the synths is
// in their side panels, their chains too.
//
//	go run ./cmd/examples/shaper
//	go run ./cmd/examples/shaper -render shaper.wav -seed 7      render it to a file (-stems: the five parts too)
//	go run ./cmd/examples/shaper -profile                        the time every module takes
//	go run ./cmd/examples/shaper -theme midnight -addr localhost:8088
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
	stems := flag.Bool("stems", false, "with -render, also write the lead, the bass and the pad to their own files")
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
	ctl := panel.New(p, "Shaper")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Theme(t))
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
