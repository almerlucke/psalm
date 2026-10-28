// Everything: a song in D minor that uses what the patcher can do for rhythm, harmony and melody, on a page. Open http://localhost:8087.
//
//	go run ./cmd/examples/everything
//	go run ./cmd/examples/everything -theme midnight
//
// What is in it, and where to look:
//
//   - harmony: the chords of every section, typed (`Dm Bb Gm A7`), and a chord model (a Markov matrix of chord progressions) to draw new ones from and write to the harmony
//   - the pad is a part of `@v`: the chord voice led from the one before, each note moving as little as it can; the bass is `@b`, the arpeggio `@1 @3 @5` with ratchets (`^2`)
//     and a condition (`%2/4`: only the second time through in every four); its lanes (open its window: bars for the chance, velocity, timing, length and pitch of each step)
//     are of 3, 4 and 5 steps against a pattern of 16, so the figure never repeats the same way twice (polymeter)
//   - the drums are clocks with patterns that are made, not written: a euclidean kick `E(4,16)`, a snare with a fill that is only played in the last bar of a section
//     (`x%fill`), hats that are a Markov chain on an example `M(...)` with a groove (mpc 58) and a drift, an open hat of `E(5,16,2)` on `%2/4`, a clap of the cellular automaton rule 30
//   - the lead is in the piano roll: a clip for every section (the melody made with a grammar, `S -> A A' B A~`, over the chords of the section), and a second roll with a
//     counter-line under it (no parallel fifths or octaves, against the melody). The rolls have a clip for the Verse and the Chorus (choose the section above the roll; the loop has none). Open the roll's Melody panel to make others: Generate, Variations, Every section, a
//     grammar or an L-system, Counter-line, Invert, Reverse, Groove
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
	"github.com/almerlucke/psalm/modules/controls/panel/noteroll"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/melody"
	"github.com/almerlucke/psalm/tools/rhythm"
	"github.com/almerlucke/sndfile"
)

// the sections of the song and how many bars each
var sections = []struct {
	name string
	bars int
}{{"Intro", 4}, {"Verse", 8}, {"Chorus", 8}, {"Outro", 4}}

const beatsPerBar = 4

// ids are the modules of the patch that tests look at
type ids struct {
	harmony, roll, counter string
	drums                  []string
}

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	patches := flag.String("patches", "patches", "the folder patches are kept in")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	ctl := panel.New(p, "Everything")
	ctl.SetTheme(t)
	hall := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), IR: hall}), patcher.Keep(*patches), patcher.Theme(t))
	mix := mixer.New(p)
	mix.Channel("patch", pt.Out(), mixer.StripConfig{Volume: -3})
	p.Out(mix.Master())
	if _, err := buildPatch(pt); err != nil {
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

// buildPatch puts the song in a patcher
func buildPatch(pt *patcher.Patcher) (ids, error) {
	var out ids
	var firstErr error
	must := func(id string, err error) string {
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return id
	}
	check := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// the song and its chords: D minor
	song := must(pt.Add("song", "", 20, 20))
	harm := must(pt.Add("harmony", "", 20, 150))
	model := must(pt.Add("chordmodel", "", 20, 300))
	check(pt.SetText(song, "sections", "Intro 4, Verse 8, Chorus 8, Outro 4"))
	check(pt.SetText(harm, "chords", "Dm Dm Bb A7\nVerse: Dm Bb Gm A7\nChorus: Bb F Gm A*2 Dm Bb Gm A"))
	pt.SettingsOf(harm).Apply(map[string]float64{"key": 2, "scale": float64(slices.Index(patcher.Scales, "minor"))})
	if m, ok := pt.Module(model).(*chordmodel.Module); ok {
		m.SetKey(2, 1) // D minor
		m.Preset("minor")
	}
	out.harmony = harm

	// the parts: bass and pad follow the chords, the arpeggio too, with lanes
	bass := must(pt.Add("part", "", 260, 20))
	pad := must(pt.Add("part", "", 260, 170))
	arp := must(pt.Add("part", "", 260, 320))
	check(pt.SetText(bass, "notes", "@b . . . @b . . . @b . @b . @b . . .\nIntro: ."))
	check(pt.SetText(pad, "notes", "@v*16"))
	check(pt.SetText(arp, "notes", "@1 @3 @5^2 @3' @1' @3'%2/4 @5 @3 @1 @3 @5^2 @3' @1' @5 @3' @5\nIntro: ."))
	// lanes of 3, 4 and 5 steps against the 16 of the pattern: the figure drifts
	check(pt.SetText(arp, "lanes", "vel: 100 70 85\nprob: 100 100 90 100\nlen: 100 60 100 80 60\npitch: 0 0 0 0 12 0 0"))
	pt.SettingsOf(pad).Apply(map[string]float64{"division": 3, "length": 1, "velocity": 0.55, "register": 50})
	pt.SettingsOf(arp).Apply(map[string]float64{"velocity": 0.6, "length": 0.5, "register": 57, "drift": 0.04})
	pt.SettingsOf(bass).Apply(map[string]float64{"velocity": 0.9, "length": 0.7})
	bassSynth := must(pt.Add("subtractive", "subBass", 520, 20))
	padSynth := must(pt.Add("supersaw", "airPad", 520, 170))
	arpSynth := must(pt.Add("wavetable", "pluck", 520, 320))

	// the drums: clocks with patterns that are made, a notes module for each to say what they play (GM: 36 kick, 38 snare, 42 closed hat, 46 open hat, 39 clap)
	drumSynth := must(pt.Add("drumsynth", "909", 520, 620))
	drum := func(name string, note, y float64, pattern, lanes string, settings map[string]float64) {
		clock := must(pt.Add("clock", "", 260, y))
		notes := must(pt.Add("notes", "", 390, y))
		check(pt.SetText(clock, "pattern", pattern))
		if lanes != "" {
			check(pt.SetText(clock, "lanes", lanes))
		}
		pt.SettingsOf(clock).Apply(settings)
		pt.SettingsOf(notes).Apply(map[string]float64{"pitch": note, "velocity": 0.8, "length": 0.3})
		for _, c := range []patcher.Cable{{From: clock, To: notes}, {From: notes, To: drumSynth}} {
			if _, err := pt.Connect(c); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		out.drums = append(out.drums, clock)
	}
	mpc58 := float64(slices.Index(rhythm.GrooveNames(), "mpc 58"))
	// the kick: 4 hits evenly over 16 steps, the intro without
	drum("kick", 36, 470, "Intro: .\nE(4,16)", "", map[string]float64{"division": 3})
	// the snare: the 5th and the 13th steps, and a roll in the last bar of a section only
	drum("snare", 38, 520, "Intro: .\n. . . . x . . . . . . . x . . [x x]%fill", "vel: 100 90", map[string]float64{"division": 3, "groove": mpc58})
	// the hats: a Markov chain on an example, a groove and a smooth drift; a lane of 5 steps for the chance and one of 3 for the velocity
	drum("hat", 42, 570, "M(x . x x . x . x x . x . x x . x;16;7)", "prob: 100 70 100 60 90\nvel: 100 55 80", map[string]float64{"division": 3, "groove": mpc58, "drift": 0.06})
	// the open hat: 5 over 16, turned 2 steps, on the second time through in every four
	drum("open", 46, 620, "Intro: .\nVerse: .\nE(5,16,2)%2/4", "", map[string]float64{"division": 3})
	// a clap of the cellular automaton (rule 30), the chorus only
	drum("clap", 39, 670, "Intro: .\nVerse: .\nOutro: .\nCA(30,16,5)", "", map[string]float64{"division": 3, "chance": 0.8})

	// the lead: a clip for every section, made with a grammar over the chords of the section; a counter-line under it in a second roll
	roll := must(pt.Add("noteroll", "", 260, 720))
	counter := must(pt.Add("noteroll", "", 260, 880))
	leadSynth := must(pt.Add("dx", "ePiano", 520, 770))
	counterSynth := must(pt.Add("wavetable", "pluck", 520, 920))
	out.roll, out.counter = roll, counter
	if err := writeMelody(pt, harm, roll, counter); err != nil && firstErr == nil {
		firstErr = err
	}

	// the mix: a mixer with a reverb on send A
	m8 := must(pt.Add("mixer", "8 channels", 800, 300))
	rev := must(pt.Add("reverb", "", 1060, 300))
	for _, c := range []patcher.Cable{
		{From: bass, To: bassSynth}, {From: pad, To: padSynth}, {From: arp, To: arpSynth}, {From: roll, To: leadSynth}, {From: counter, To: counterSynth},
		{From: model, To: padSynth}, // hear the chord model through the pad
		{From: bassSynth, To: m8, In: 0}, {From: padSynth, To: m8, In: 1}, {From: arpSynth, To: m8, In: 2}, {From: leadSynth, To: m8, In: 3},
		{From: drumSynth, To: m8, In: 4}, {From: counterSynth, To: m8, In: 5},
		{From: m8, Out: 1, To: rev}, {From: rev, To: m8, In: 8}, {From: m8, Out: 0, To: "out"},
	} {
		if _, err := pt.Connect(c); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	pt.SettingsOf(m8).Apply(map[string]float64{"level1": 0.7, "level2": 0.4, "level3": 0.45, "level4": 0.6, "level5": 0.6, "level6": 0.3,
		"sendA2": 0.4, "sendA3": 0.35, "sendA4": 0.4, "sendA6": 0.4, "pan3": -0.2, "pan4": 0.2, "pan6": -0.3, "returnA": 1})
	pt.SettingsOf(rev).Apply(map[string]float64{"mix": 1, "decay": 3})
	return out, firstErr
}

// writeMelody makes the lead of the verse and the chorus with a grammar over the chords of those sections, and a counter-line under it
func writeMelody(pt *patcher.Patcher, harm, roll, counter string) error {
	h := pt.Module(harm).(*patcher.Harmony)
	scale, err := harmony.NewScale("D", "minor")
	if err != nil {
		return err
	}
	lead, under := pt.Module(roll).(*noteroll.Roll), pt.Module(counter).(*noteroll.Roll)
	for i, s := range sections {
		if s.name == "Intro" || s.name == "Outro" {
			continue
		}
		beats := float64(s.bars * beatsPerBar)
		var chords []melody.Span
		for _, sp := range h.Spans(s.name, beats, beatsPerBar) {
			root := 0
			if len(sp.Pcs) > 0 {
				root = sp.Pcs[0]
			}
			chords = append(chords, melody.Span{At: sp.At, Len: sp.Len, Pcs: sp.Pcs, Root: root})
		}
		grammar := "S -> A A' B A~"
		if s.name == "Chorus" {
			grammar = "S -> A B A' B~?"
		}
		m := melody.Generate(melody.Params{Scale: scale, Beats: beats, Low: 62, High: 86, Chords: chords, Grammar: grammar, Phrase: 8, Density: 0.45, Seed: uint64(10 + i)})
		c := melody.Counter(m, melody.CounterParams{Scale: scale, Chords: chords, Low: 45, High: 69, Every: 2})
		lead.SetClip(s.name, toNotes(m), beats)
		under.SetClip(s.name, toNotes(c), beats)
		lead.SetView(s.name) // the editors open on a clip with notes in it (the loop has none: the notes are in the clips of the sections)
		under.SetView(s.name)
	}
	return nil
}

func toNotes(m []melody.Note) []noteroll.Note {
	out := make([]noteroll.Note, len(m))
	for i, n := range m {
		out[i] = noteroll.Note{Start: n.Start, Length: n.Length, Key: n.Key, Velocity: n.Velocity}
	}
	return out
}
