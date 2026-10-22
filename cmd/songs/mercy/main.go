// Mercy: Psalm 136, "O give thanks unto the LORD; for his mercy endureth for ever", twenty-six verses, each ending with the same
// refrain, made of sounds cut into pieces: a tabla loop, the whole psalm spoken (goedertierenheid), wooden tubes, birds and a congas
// kit, played by a sampler for each (the pieces of a sound are on the keys of its sampler), with the grains synth, the choirs and
// the horn of the Virtual Playing Orchestra (shaper synths instead when it is not in assets) and a drone and a harp of the shaper.
//
// Every verse is four bars, the length of the tabla loop. The loop is cut at its hits and remixed as the verse goes: beats thrown to
// other places, a few repeated (a stutter, a roll), bars swapped, turned round, thinned; the last bar is always the loop as it
// was, the refrain that returns. The recording is the psalm spoken, each verse with its refrain, cut at its pauses: verse n of the song says verse n of the recording, and its refrain is said in the last bar, the refrain bar. The
// wooden tubes wander through their file by a markov chain (on to the next piece, a skip, back, the same again). The verses are
// the six parts of the psalm (sections): give thanks (3 verses), creation (6), the Egyptians and the sea (6), the wilderness and the
// kings (7), "he remembered us" (3) and "the God of heaven" (1): 12, 24, 24, 28, 12 and 8 bars at 90 beats a minute, about 4 1/2 minutes,
// and it loops. The last verse is followed by a coda of four bars: the choir holds the chord under the last refrain and goes out, then the quiet before it begins again. The key is F minor.
//
//	go run ./cmd/songs/mercy
//	go run ./cmd/songs/mercy -render mercy.wav -seed 7      render it to a file (-stems: the parts too)
//	go run ./cmd/songs/mercy -profile                       the time every module takes
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
	ctl := panel.New(p, "Mercy")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Theme(t))
	ids, err := buildPatch(pt, *renderTo != "" || *profile)
	if err != nil {
		log.Fatal(err)
	}
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: 6})
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
