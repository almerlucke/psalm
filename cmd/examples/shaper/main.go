// Shaper: a tour of the shaper synth, played by scripts. Three scripts (lead, bass, pad) play three
// voices of the synth, and each rewrites its voice for every section of the song: the chains (as text),
// the macros the chains read, the morph, the envelopes, the filter. Twelve sections, a different sound
// each: a sine that grows overtones, a folded sine, hard sync, a pulse width moved by chaos, the morph
// through sine, triangle, saw and square, struck pipes of whole number partials, mouth shapes, a saw
// cut into steps, chains the script swaps and the script makes up, a quick pluck, the sines again.
//
// The scripts are in scripts/ (embedded); in the page double tap a script to read it and change it while
// it plays (Ctrl+Enter takes it), the song's tiles play a section next. Every setting of the synths is
// in their side panels, their chains too.
//
//	go run ./cmd/examples/shaper
//	go run ./cmd/examples/shaper -render shaper.wav -seed 7      render the tour to a file
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
	seed := flag.Uint64("seed", 0, "seed the random choices (0: different every time); with it a render is the same every time")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	if *seed != 0 {
		psalm.Seed(*seed)
	}
	var p *psalm.Psalm
	if *renderTo != "" {
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
	scripts, err := buildPatch(pt, *renderTo != "")
	if err != nil {
		log.Fatal(err)
	}
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: -3})
	p.Out(master.Master())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: tourSeconds(), Tail: 8})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("rendered %.0f s to %s\n", stats.Seconds, *renderTo)
		for _, s := range scripts {
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
