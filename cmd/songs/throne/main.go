// Throne: the throne room of Revelation 4, John's door opened in heaven, in about seven minutes of calm electronic music, made to be
// left on for hours (it loops: the end is the quiet the beginning starts from, nothing in it builds up or startles). There are no
// drums. Seven shaper synths play,
// most of them with the presets that only this synth makes: a pad that travels through four colours in one chord (the rainbow
// round the throne, like unto an emerald), a drone of the partials 1, 3, 7 and 12, a sea of glass that never repeats itself, seven
// lamps of fire (a bell whose partials belong to no pitch), a voice as of a trumpet that is a breath played softly and brass played
// hard, eyes that draw every note anew and a crown thrown down in a clatter of gold;
// with harps, a music box, glass bells and vibes of the others (a flute, a marimba, a voice, a choir) to make it heavenly. The tinklings
// of the throne (the music box) evolve all through the piece and come home to it in the worthy (scripts/sparkle.js).
//
// The numbers are holy ones. The tempo is 84 beats a minute (seven times twelve) and the piece 144 bars (twelve times twelve):
// the door of 7 bars; the throne of 12; the seven lamps of 21 bars, three for each lamp, one more of the twelve steps lit with
// every lamp; the sea of glass of 12; the four living creatures, six bars for each, six wings each, 24 bars, each with its own
// voice (a flute, a marimba, a voice, an airy pad, rising); the 24 elders, two bars for each chord, who throw down their crowns;
// "Holy, holy, holy" in 28 bars (four times seven), the call said three times, three notes at a time; and the worthy of 16.
// The rhythms are euclidean ones against the sixteen steps of the bar: one to seven hits in a cycle of twelve steps (it comes
// round every three bars), three of seven for the stars. The lines are walks of a markov chain whose weights are 3, 7 and 12
// (staying on a note is 12, a step 7, a leap 3), fitted to the chord of the bar, with now and then the last three steps again; the harp, the
// lamps and the stars are not scales going up and down: every phrase has a shape (an arch, a wave, a low note under a climbing line,
// broken wide...) that a markov chain of the same weights picks (scripts/harmony.js, shared.shapes). The whole piece is in D major, light.
//
//	Door       a door opened in heaven, a voice as of a trumpet: "Come up hither" (open fifths)
//	Throne     a throne set in heaven, and one sat on it: a rainbow round it, like an emerald (the light, D major)
//	Lamps      seven lamps of fire before the throne (21 bars, one lamp more every three)
//	Glass      before the throne a sea of glass like unto crystal
//	Creatures  four beasts full of eyes: a lion, a calf, a man, a flying eagle, and they rest not day and night
//	Elders     four and twenty elders sit round the throne in white, and they cast their crowns before it
//	Holy       "Holy, holy, holy, Lord God Almighty, which was, and is, and is to come"
//	Worthy     "Thou art worthy, O Lord, to receive glory and honour and power: for thou hast created all things"
//
// The scripts are in scripts/ (embedded); in the page double tap a script to read it and change it while
// it plays (Ctrl+Enter takes it), the song's tiles play a section next. Every setting of the synths is
// in their side panels, their chains too.
//
//	go run ./cmd/songs/throne
//	go run ./cmd/songs/throne -render throne.wav -seed 7      render it to a file (-stems: the seven parts too)
//	go run ./cmd/songs/throne -profile                        the time every module takes
//	go run ./cmd/songs/throne -theme midnight -addr localhost:8088
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
	ctl := panel.New(p, "Throne")
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
