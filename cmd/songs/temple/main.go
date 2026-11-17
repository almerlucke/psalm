// Temple: a piece from a MIDI file (assets/midi/genny-temple.mid: 66 bpm, four minutes, sixteen channels; its tracks are named after the furnishings of the temple of
// Solomon). A midiplayer plays the file, each channel to a synth of psalm of its own, as the General MIDI programs of the file ask: tubular bells, a vibraphone (twice) and a glockenspiel
// (modal), two church organs (organ), two harps (waveguide), choirs and a halo pad (vocal), a warm pad (subtractive), a grand piano (piano) and the drums (drumsynth, the 808 kit). All are on a mixer with
// a hall, an echo and a shimmer. In the page the midiplayer's window has the file picker, and the synths' windows their editors, to change the sounds while it plays.
//
//	go run ./cmd/songs/temple
//	go run ./cmd/songs/temple -render part.wav -from 60 -seconds 30      a part of it (to tune the mix)
//	go run ./cmd/songs/temple -save temple      save the patch as patches/temple.json, to load in the patcher and change
//	go run ./cmd/songs/temple -render temple.wav -seed 7      render it to a file (-stems: the parts too)
package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"slices"

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
	from := flag.Float64("from", 0, "with -render, start the file at this second")
	length := flag.Float64("seconds", 0, "with -render, render this many seconds (0: to the end of the file), to tune a part without waiting for all of it")
	save := flag.String("save", "", "save the patch of the piece as a patch of the patcher (patches/<name>.json, to load in the patcher and change) and stop")
	seed := flag.Uint64("seed", 0, "seed the random choices (0: different every time); with it a render is the same every time")
	flag.Parse()

	if duration() == 0 {
		log.Fatalf("%s: the file is not there or can't be read (run it from the root of the repository)", midiFile)
	}
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
	ctl := panel.New(p, "Temple")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Audio: p.PatchInput(0)}), patcher.Theme(t), patcher.Keep("patches"))
	ids, err := buildPatch(pt)
	if err != nil {
		log.Fatal(err)
	}
	if *save != "" {
		if err := pt.Save(*save); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("saved patches/%s.json\n", *save)
		return
	}
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: 2})
	p.Out(master.Master())

	if *profile {
		fmt.Print(render.RunProfiled(p, render.Options{Seconds: duration(), Tail: 2}))
		return
	}
	secs := duration()
	if *from > 0 {
		pt.Module(ids.player).ReceiveMessage(psalm.Message{"seek": *from})
		secs -= *from
	}
	if *length > 0 {
		secs = min(secs, *length)
	}
	if *renderTo != "" {
		var sources map[string]psalm.Modular
		if *stems {
			sources = map[string]psalm.Modular{}
			for part, id := range ids.synths {
				sources[part] = pt.Module(id)
			}
		}
		stats, err := render.Stems(p, *renderTo, render.Options{Seconds: secs, Tail: 4}, sources)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("rendered %.0f s to %s\n", stats.Seconds, *renderTo)
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
