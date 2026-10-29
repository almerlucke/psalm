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
//   - chords that are drawn by a Markov model of progressions each time the song leaves a section (the Shatter, the Splinter and the Drive have no chords of their own: they get the last drawing)
//
//   - samplers: a sung recording cut into short syllables and a tabla loop cut at its strokes, played in any order by the same kind of rhythms (rolls, ratchets, mutations), through the crusher and the
//     granulator: the glitch
//
//   - a toy box: a melancholic melody on a music box in three layers (melodists over the chords, in the phrygian scale of the harmony, a question and its answer, a falling tune and a rising
//     answer, the tune of the section before changed a little each time through): the tune alone in the Static, a lower answer under it in the Fracture, a glittering high layer in the Drive,
//     the glitter and the answer in the Drive, and in the Collapse, when it has all fallen apart, the tune alone
//
//   - a groove that builds the tune up: an 808 kick, hats, a rim and congas whose chance and velocity are moved by one curve of the song (a lane: nothing in the Static, fuller and louder
//     through the Fracture to full in the Shatter, the Splinter and the Drive, away in the Collapse); a synth plays the chords in sixteenths to drive the tune, its filter opening with the curve
//
//   - a bass that plays the chords by notes relative to them, its notes lengthening with the same curve (short and plucked early, long and buzzing in the Drive, so it drives the tune), a pad that is held voice led, a crusher whose bit depth is moved by a random source and an echo whose feedback is moved by a
//     slow wave
//
//   - a beat to dance to from the Shatter on: a kick on every beat, a clap on the two and the four, hats on the eighths with the offbeat accented, steady (nothing mutates), on the 808, louder with the
//     curve; the Fracture keeps its own rhythms, the beat is what the later sections stand on
//
//   - it gets more and more epic: one curve rises through the whole piece to the end of the Drive and everything grand follows it, coming in one by one and louder as it goes (timpani on the
//     root of the chord with a roll in the last bar of each section, strings held voice led and doubled an octave up, brass on the root and the fifth, a choir, a longer room, a brighter shimmer),
//     and the acid bass gets deeper and deeper (the resonance, the amount of the filter envelope and the drive of the buzz bass rise with it)
//
// The sections: Static (a heart, dust, a held chord, the tune alone), Fracture (the rhythms break in, timpani), Shatter and Splinter (two more sections of the same kind as the Fracture, each its own
// pattern of rhythms and figures, the tune building: a glitter on top in the Shatter, an upper voice, the synth, strings and brass in the Splinter), Drive (the build goes on to full, the choir, and
// it explodes in the roll at the end), Collapse (one section after it: everything falls apart, automata and dust under the tune alone, a held chord). It goes round: every pass is another.
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
