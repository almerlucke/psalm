// Fracture: an avant garde electronic piece that is never the same twice. Open http://localhost:8087.
//
//	go run ./cmd/songs/fracture                  a new one every time
//	go run ./cmd/songs/fracture -seed 7          the same one again (the same drawing of the rhythms, the chords and the melody)
//	go run ./cmd/songs/fracture -theme midnight
//
// It is made with what the patcher has for change:
//
//   - rhythms that are made while it plays: a euclidean kick that turns a step every pass (E(5,16,+1)), cellular automata that run on (rule 30, 90, 110: a new row every pass), a
//     Markov rhythm that is drawn again every few passes, and mutate, which takes hits away and puts others in (a hit with a ! after it stays); conditions (the second time through in every
//     four, a fill in the last bar of a section); lanes of 3, 4, 5 and 7 steps against the 16 and 32 of the patterns, so the accents drift against each other; the groove of Dilla and a
//     drift of the timing
//
//   - samplers: a sung recording cut into short syllables and played by the chords (each syllable tuned to the tone it is played at), and a tabla loop cut at its strokes and played in any
//     order, by the same kind of rhythms, through the crusher and the granulator: the glitch
//
//   - a toy box: a melancholic melody on a glockenspiel (the dx crystalGlock) in two layers, and a counterpoint to it on the left (melodists over the chords, in the phrygian scale of the harmony, a question and its answer, a falling tune and a rising
//     answer, the tune of the section before changed a little each time through)
//
//   - an 808 groove under it (a kick, hats and a rim), a synth that drives the tune in sixteenths on the chord, timpani, a bass that plays the chords by notes relative to them (an acid bass:
//     the buzz bass through a low filter that the envelope snaps open), a pad held voice led; one curve of the song moves the chance and the velocity of the groove, the length of the bass and
//     the filter of the synth, another the acid of the bass, in every section the same way; a random source moves the bit depth of a crusher and a slow wave the feedback of the echo
//
//   - a master that is loud: a compressor on the sum and a limiter after it. A shimmer (a fifth up) on the fourth bus, for the two glockenspiels only
//
// The sections: Static (a heart, a held chord, the tune alone), Fracture (the rhythms break in), the Ridge and the Rift (the Fracture again, a little different each time: other rotations and
// seeds of the same rhythms, the chords of the Fracture in another order), the Bare (the Rift without the tune), the Skeleton (eight bars of the drums, a rim, the bass, the glitch and the metal) and the Collapse (the Fracture again without the tune, the drums and the bass: the
// glitch, the sung syllables, the tabla, the synth and the pad that are left). It goes round: every pass is another.
package main

import (
	"flag"
	"log"
	"maps"
	"slices"
	"time"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/mixer"
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	seed := flag.Int("seed", 0, "the seed of the drawing (0: a new one every time)")
	flag.Parse()
	if *seed == 0 {
		*seed = int(time.Now().UnixNano()%100000) + 1
	}
	log.Printf("seed %d (play it again with -seed %d)", *seed, *seed)

	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	psalm.Seed(uint64(*seed))
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	ctl := panel.New(p, "Fracture")
	ctl.SetTheme(t)
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Audio: p.PatchInput(0)}), patcher.Theme(t))
	mix := mixer.New(p)
	mix.Channel("patch", pt.Out(), mixer.StripConfig{Volume: -3})
	p.Out(mix.Master())
	if _, err := buildPatch(pt, *seed); err != nil {
		log.Fatal(err)
	}
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03)).Top()
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
