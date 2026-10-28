package main

// "Woven": Psalm 139, God who knits us together in our mother's womb. "For thou hast possessed my reins: thou hast covered me in my mother's womb. I will praise thee; for I
// am fearfully and wonderfully made: marvellous are thy works. My substance was not hid from thee, when I was made in secret, and curiously wrought in the lowest parts
// of the earth" (Psalm 139:13-15): curiously wrought is woven, embroidered, in the Hebrew, and the piece is a weaving. It is made as a patch, with the scripts and the
// instruments of the patcher, for the page to show while it plays.
//
// The weave: a harp goes up and down the chord in a zigzag, over and under, the warp that stays; the voices that sing the tune are the weft, passing through it: one tune
// in two halves that fit over each other (every note of their strong beats a note of the chord), so a flute, a violin and a cello can sing them in turn, a canon, a bar
// or four apart, and meet in consonance wherever they cross. Struck glass, hand drums and eggs are threads of other lengths (cycles of seven, nine and twelve sixteenths
// through a bar of sixteen) that run through the same cloth and come back to the same place only after a long while. A heart beats under it, D and the A above it.
//
//	Thread  (8)   "thou hast searched me": the harp and a heart, the tune once, alone, over a held D
//	Knit    (16)  "covered me in my mother's womb": the weave begins; the tune is taken up by three voices in turn
//	Wonder  (16)  "fearfully and wonderfully made": a hymn in the violin and the voices, the star-lit celesta
//	Shadow  (12)  "if I say, Surely the darkness shall cover me": the relative minor, the cello and the alto flute, a heart far off
//	Light   (12)  "the night shineth as the day": the same weave in the major, all the voices together
//	Sand    (16)  "more in number than the sand": three sparkling cycles of plucked notes, the tune alone above them
//	Awake   (12)  "when I awake, I am still with thee": back to a single thread and a heart, and the last chord slowing as a breath is let go
//
// The key is D major, 68 beats a minute, tuned in just intonation (every note a pure ratio of the harmonic series, the tonic D); the verses are written to the log as the
// sections begin. The instruments are the recordings of the Virtual Playing Orchestra where it is in assets (a harp, a celesta, solo flute, alto flute, violin and cello,
// violas, a female choir, plucked strings), models of them where it is not; the piano, the heart and the percussion are the same either way.
//
// Under the written parts there are parts that are made while it plays (addLive in patch.go): a harmony module with the chords, a melodist that thinks aloud on the glass in the Wonder,
// the Light and the Sand (a question and its answer over the chord of the bar, a little different each time through a section, in just intonation), hand drum ghosts of three hits in
// twelve that turn a step every pass, and in the Sand eggs of a cellular automaton (rule 30, a new row every pass): the grains more in number than the sand.
//
//	go run ./cmd/songs/psalm139                                      play it (the page is at http://localhost:8087)
//	go run ./cmd/songs/psalm139 -tuning equal                        in equal temperament instead
//	go run ./cmd/songs/psalm139 -orchestra=false                     the models of the instruments instead of the recordings
//	go run ./cmd/songs/psalm139 -human=false                         exactly on the grid
//	go run ./cmd/songs/psalm139 -render woven.wav -stems -seed 7     render it, a stem per part
//	go run ./cmd/songs/psalm139 -profile                             the time every module takes

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"slices"
	"strings"
	"time"

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
	human := flag.Bool("human", true, "the parts play like players: a little off the beat, a chord not quite together, a little different in loudness (off: exactly on the grid)")
	orch := flag.Bool("orchestra", true, "the instruments the Virtual Playing Orchestra has recorded play its recordings (if it is in assets), off: models of them")
	a4 := flag.Float64("a4", 440, "the reference pitch of A4 in Hz")
	flag.Parse()
	if *tuning != "just" && *tuning != "equal" {
		log.Fatalf("tuning %q: just or equal", *tuning)
	}

	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})
	feelSeed := *seed
	if *seed != 0 {
		psalm.Seed(*seed)
	} else {
		feelSeed = uint64(time.Now().UnixNano())
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
	ctl := panel.New(p, "Woven")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), IR: church, Audio: p.PatchInput(0)}), patcher.Theme(t))
	ctl.InputSelect(p.PatchInput(0)) // the sound input to listen to: a headset, the microphone of the computer
	ids, err := buildPatch(pt, options{just: *tuning == "just", a4: *a4, seed: feelSeed, human: *human, real: *orch && hasOrchestra(), once: *renderTo != "" || *profile})
	if err != nil {
		log.Fatal(err)
	}
	(&tempoLane{Module: psalm.NewControl(), song: patcher.SongOf(p.Env())}).init().AddTo(p)
	master := mixer.New(p)
	master.Channel("patch", pt.Out(), mixer.StripConfig{Volume: 0})
	p.Out(master.Master())

	fmt.Println("Woven")
	fmt.Println("I will praise thee; for I am fearfully and wonderfully made: marvellous are thy works. (Psalm 139:14)")
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
