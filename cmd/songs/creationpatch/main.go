package main

// "Creation" as a patch: the seven days of Genesis 1 and 2, to the glory of God the Father, the Son and the Holy Spirit, the song of cmd/songs/creation made
// with the modules of the patcher instead of Go code. Every part is a script (a synth each, on a mixer with a hall, the heavens and a church), the days
// and their chords are in the first script, and the page shows it all while it plays: change a score, a sound or a send and hear it.
//
// "In the beginning was the Word, and the Word was with God, and the Word was God. All things were made by him" (John 1:1, 3): the Word through whom all was
// made is Jesus Christ, our Messiah and Redeemer, and the piece ends at his cross (Colossians 1:16, 20). Every day is an evening and a morning (six bars of
// evening, six of morning); the seventh day, the day God blessed and rested, is 21 bars: the Doxology on the Old Hundredth and a plagal Amen. Three beats to
// the bar (the Trinity) in E flat major, tuned in just intonation: every note a pure ratio of the harmonic series God put into every vibrating string.
//
//	go run ./cmd/songs/creationpatch                                      play it (the page is at http://localhost:8087)
//	go run ./cmd/songs/creationpatch -tuning equal                        in equal temperament instead
//	go run ./cmd/songs/creationpatch -render creation.wav -stems -seed 7  render it, a stem per part
//	go run ./cmd/songs/creationpatch -profile                             the time every module takes

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
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	renderTo := flag.String("render", "", "render the piece once to this WAV file instead of playing it")
	profile := flag.Bool("profile", false, "play the piece offline and report the time every module takes")
	stems := flag.Bool("stems", false, "with -render, also write the parts to their own files")
	seed := flag.Uint64("seed", 0, "seed the random choices (0: different every time); with it a render is the same every time")
	tuning := flag.String("tuning", "just", "just (pure ratios) or equal (equal temperament)")
	a4 := flag.Float64("a4", 440, "the reference pitch of A4 in Hz")
	flag.Parse()
	if *tuning != "just" && *tuning != "equal" {
		log.Fatalf("tuning %q: just or equal", *tuning)
	}

	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})
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
	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	ctl := panel.New(p, "Creation")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), IR: church}), patcher.Theme(t))
	ids, err := buildPatch(pt, options{just: *tuning == "just", a4: *a4, once: *renderTo != "" || *profile})
	if err != nil {
		log.Fatal(err)
	}
	(&tempoLane{Module: psalm.NewControl(), song: patcher.SongOf(p.Env())}).init().AddTo(p)
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: 0})
	p.Out(master.Master())

	fmt.Println("Creation")
	fmt.Println("In the beginning was the Word, and the Word was with God, and the Word was God. All things were made by him. (John 1:1, 3)")
	fmt.Print(schedule())

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
