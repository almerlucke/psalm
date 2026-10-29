// Fracture: an avant garde electronic piece that is never the same twice. Open http://localhost:8087.
//
//	go run ./cmd/songs/fracture                  a new one every time
//	go run ./cmd/songs/fracture -seed 7          the same one again (the same drawing of the rhythms, the chords and the melody)
//	go run ./cmd/songs/fracture -theme midnight
//
// It is made with what the patcher has for change:
//   - rhythms that are made while it plays: a euclidean kick that turns a step every pass (E(5,16,+1)), cellular automata that run on (rule 30, 90, 110: a new row every pass), a
//     Markov rhythm that is drawn again every few passes, and mutate, which takes hits away and puts others in (a hit with a ! after it stays); conditions (the second time through in every
//     four, a fill in the last bar of a section); lanes of 3, 4, 5 and 7 steps against the 16 and 32 of the patterns, so the accents drift against each other; the groove of Dilla and a
//     drift of the timing
//   - chords that are drawn by a Markov model of progressions each time the song leaves a section (the Machine and the Drift have no chords of their own: they get the last drawing)
//   - samplers: a sung recording cut into short syllables and a tabla loop cut at its strokes, played in any order by the same kind of rhythms (rolls, ratchets, mutations), through the crusher and the
//     granulator: the glitch
//   - a toy box: a melancholic melody on a music box in three layers (melodists over the chords, in the phrygian scale of the harmony, a question and its answer, a falling tune and a rising
//     answer, the tune of the section before changed a little each time through): the tune alone in the Static, a lower answer under it in the Fracture, a glittering high layer in the Machine,
//     nothing in the Collapse, the tune and the glitter in the Drift, the tune alone thinning in the Residue
//   - a groove that builds the tune up: an 808 kick, hats, a rim and congas whose chance and velocity are moved by one curve of the song (a lane: nothing in the Static, fuller and louder
//     through the Fracture to full in the Machine, away in the Collapse, back and fading in the Drift and the Residue)
//   - a bass that plays the chords by notes relative to them, a pad that is held voice led, a crusher whose bit depth is moved by a random source and an echo whose feedback is moved by a
//     slow wave
//
// The sections: Static (a heart, dust, a held chord), Fracture (the rhythms break in), Machine (a floor of four that falls apart), Collapse (automata alone), Drift (what is left, turning),
// Residue (dust and a held chord). It goes round: every pass is another.
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
