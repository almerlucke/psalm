// Composer: a patch to write music in, on a page. A song of four sections; a chord model (a Markov matrix of how likely each chord is to follow another: draw a
// progression, lock the chords you like, draw the others again, hear it, write it to the harmony); the harmony (the chords of every section); three parts that write
// their notes relative to the chords (`@1 @3 @5` the root, the fifth, `@b` the bass, `@c` the chord: write a figure once and it follows the harmony); and a piano roll
// with a clip for every section and the chords over it, for a melody. Open http://localhost:8087.
//
//	go run ./cmd/examples/composer
//	go run ./cmd/examples/composer -theme midnight
//
// Try: open the chord model, choose a style, press "New progression", lock the chords you like and "Draw the others again", "Hear" it, choose a section and "Write to the
// harmony"; then press Restart (the song plays the chords you wrote). In the piano roll choose the clip of a section, draw a melody (the keys of the chord are lit), copy it to the other sections.
package main

import (
	"flag"
	"log"
	"maps"
	"slices"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/chordmodel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/sndfile"
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	patches := flag.String("patches", "patches", "the folder patches are kept in")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: 96, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	ctl := panel.New(p, "Composer")
	ctl.SetTheme(t)
	hall := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), IR: hall}), patcher.Keep(*patches), patcher.Theme(t))
	mix := mixer.New(p)
	mix.Channel("patch", pt.Out(), mixer.StripConfig{Volume: -3})
	p.Out(mix.Master())

	must := func(id string, err error) string {
		if err != nil {
			log.Fatal(err)
		}
		return id
	}
	// the song and its chords: D minor, drawn by the chord model (the first chords are written in the harmony to start from)
	song := must(pt.Add("song", "", 20, 20))
	harmony := must(pt.Add("harmony", "", 20, 150))
	model := must(pt.Add("chordmodel", "", 20, 300))
	_ = pt.SetText(song, "sections", "Intro 4, Verse 8, Chorus 8, Outro 4")
	_ = pt.SetText(harmony, "chords", "Dm Dm Bb A7\nVerse: Dm Bb Gm A7\nChorus: Bb F Gm A*2 Dm Bb Gm A")
	pt.SettingsOf(harmony).Apply(map[string]float64{"key": 2, "scale": float64(slices.Index(patcher.Scales, "minor"))})
	if m, ok := pt.Module(model).(*chordmodel.Module); ok {
		m.SetKey(2, 1) // D minor
		m.Preset("minor")
	}

	// the parts: notes written relative to the chords, each to a synth of its own
	bass := must(pt.Add("part", "", 260, 20))
	pad := must(pt.Add("part", "", 260, 170))
	arp := must(pt.Add("part", "", 260, 320))
	roll := must(pt.Add("noteroll", "", 260, 470))
	bassSynth := must(pt.Add("subtractive", "subBass", 520, 20))
	padSynth := must(pt.Add("supersaw", "airPad", 520, 170))
	arpSynth := must(pt.Add("wavetable", "pluck", 520, 320))
	leadSynth := must(pt.Add("dx", "ePiano", 520, 470))
	_ = pt.SetText(bass, "notes", "@b . . . @b . . . @b . @b . @b . . .\nIntro: .")
	_ = pt.SetText(pad, "notes", "@c*16")
	_ = pt.SetText(arp, "notes", "@1 @3 @5 @3' @1' @3' @5 @3 @1 @3 @5 @3' @1' @5 @3' @5\nIntro: .")
	pt.SettingsOf(pad).Apply(map[string]float64{"division": 3, "length": 1, "velocity": 0.55, "register": 50})
	pt.SettingsOf(arp).Apply(map[string]float64{"velocity": 0.6, "length": 0.5, "register": 57})
	pt.SettingsOf(bass).Apply(map[string]float64{"velocity": 0.9, "length": 0.7})

	// the mix: a mixer with a reverb on send A
	m8 := must(pt.Add("mixer", "8 channels", 800, 150))
	rev := must(pt.Add("reverb", "", 1060, 150))
	for _, c := range []patcher.Cable{
		{From: bass, To: bassSynth}, {From: pad, To: padSynth}, {From: arp, To: arpSynth}, {From: roll, To: leadSynth},
		{From: model, To: padSynth}, // hear the chord model through the pad
		{From: bassSynth, To: m8, In: 0}, {From: padSynth, To: m8, In: 1}, {From: arpSynth, To: m8, In: 2}, {From: leadSynth, To: m8, In: 3},
		{From: m8, Out: 1, To: rev}, {From: rev, To: m8, In: 8}, {From: m8, Out: 0, To: "out"},
	} {
		if _, err := pt.Connect(c); err != nil {
			log.Fatal(err)
		}
	}
	pt.SettingsOf(m8).Apply(map[string]float64{"level1": 0.7, "level2": 0.45, "level3": 0.5, "level4": 0.6, "sendA2": 0.4, "sendA3": 0.35, "sendA4": 0.4, "pan3": -0.2, "pan4": 0.2, "returnA": 1})
	pt.SettingsOf(rev).Apply(map[string]float64{"mix": 1, "decay": 3})

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
