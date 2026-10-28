// Laudate: Psalm 150, "Praise ye the LORD", a slow, jazzy piece of rhythms that interlock (swung sixteenths and eighth triplets, a ride pattern, a walking bass, piano comping on jazz chords with their dominants), made with the scripts of the patcher and no drum kit (hand percussion, a tambourine and a cymbal, cut at their hits). The six sections
// are the verses of the psalm (the sanctuary, the mighty acts, the trumpet, harp and dance, the cymbals, everything that hath breath, and the Hallelujah), and the
// verse is written to the log of the patch as its section begins. The rhythms are euclidean cycles (5, 7 or 9 in 16, 5 in 12, 4 in 9, 3 in 7) that run against
// one another, each part with a clock of its own on a division of the beat of its own (sixteenths, eighth triplets, sixteenth triplets, quarters), and a markov
// chain that chooses what a hit is: a single, a ratchet of two or three, a late one, a rest. The melodies are markov chains too: a piano that goes by steps and
// leaps through the scale of F minor and settles on the tones of the chord when the beat is strong, a glass bell far above it, a harp over the chord. The chords
// are a chain of their own. The sounds are those of the sampler (bells of glass, harp notes, wooden tubes, the hand percussion and the tambourine loop cut at their hits, an organ trumpet, a cymbal, the long notes I sang, each on its keys), the piano, an organ pad and the shaper (a sine bass). The piece goes in phrases of two bars, four to a group: something said, said again, answered, said again changed. At 66 beats a minute, four minutes; it loops.
//
//	go run ./cmd/songs/laudate
//	go run ./cmd/songs/laudate -render laudate.wav -seed 7      render it to a file (-stems: the parts too; -fade 3: the parts fade out over the last three bars, -lead 4: ending four seconds before the end, so the echoes ring out alone; -ring 20: at most 20 seconds more)
//	go run ./cmd/songs/laudate -profile                         the time every module takes
package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"math"
	"regexp"
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
	fade := flag.Float64("fade", 3, "with -render, the bars the end of the piece fades out over: the parts fade, the echoes and the hall ring out after (0: no fade)")
	lead := flag.Float64("lead", 4, "with -render, the seconds before the end of the piece the parts are silent: the echoes, the hall and the shimmer ring on alone")
	ring := flag.Float64("ring", 20, "with -render, the most seconds the echoes may ring on after the piece (it stops sooner when it is silent)")
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
	ctl := panel.New(p, "Laudate")
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
		// the fade is on the strips of the mixer, before the sends: the parts fade out and the echoes, the hall and the shimmer go on ringing after
		// them, until they are silent. The levels are taken from the mixer as the patch set them
		mix := pt.SettingsOf(ids.mixer)
		base := map[string]float64{} // the levels and the sends of the channels (not the main level, nor the returns: they carry the echoes)
		chanKey := regexp.MustCompile(`^(level|sendA|sendB|sendC|sendD)\d+$`)
		for k, v := range mix.Values() {
			if chanKey.MatchString(k) {
				base[k] = v
			}
		}
		length, fadeLen := tourSeconds()-*lead, *fade*4*60/tempo // the parts are faded out at length
		lastGain := 1.0
		before := func(t float64) {
			g := 1.0
			if fadeLen > 0 && t > length-fadeLen {
				g = 0.5 * (1 + math.Cos(math.Pi*math.Min(1, (t-(length-fadeLen))/fadeLen)))
			}
			if g == lastGain {
				return
			}
			lastGain = g
			// the sound of a part fades with g, what it sends to the echoes, the hall and the shimmer more slowly (the sends are after the level, so a send
			// is made larger by the part that the level takes away), so there is more of the echoes than of the parts as it ends, and they ring on after
			scaled := map[string]float64{}
			for k, v := range base {
				switch {
				case strings.HasPrefix(k, "level"):
					scaled[k] = v * g
				default:
					scaled[k] = math.Min(1, v*math.Pow(math.Max(g, 1e-3), -0.5))
				}
			}
			mix.Apply(scaled)
		}
		stats, err := render.Stems(p, *renderTo, render.Options{Seconds: tourSeconds(), Tail: *ring, Before: before}, sources)
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
