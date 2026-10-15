// Patcher: make a modular patch on a page. Add modules from the palette (every synth with its
// presets, every effect, a keyboard, LFOs, random modulation, a mixer, and generators: a clock, a
// sequence of values, notes, a quantizer), move them around by their heads, and draw cables from
// an output to an input: audio (blue), notes to a synth (green), modulation from an LFO, a random
// source or the mod wheel to any setting of a module (yellow: choose the setting and the part of
// its range it moves below the patch), triggers (purple dots) and values (grey). Patterns and
// lists of values are typed on the modules themselves. Click a
// module for its settings, a cable to take it away. Keep patches by name (patches), every module's
// presets of your own in editors-presets. Play the page's keyboard, the computer's keys (A W S E D
// ...) or a MIDI keyboard. Open http://localhost:8087.
//
//	go run ./cmd/examples/patcher
//	go run ./cmd/examples/patcher -theme midnight -patches mypatches
package main

import (
	"cmp"
	"flag"
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
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/sndfile"
)

// inputs reads several inputs as one, in order of offset
type inputs []midi.Input

func (in inputs) Read(events []midi.Event, frames int, sampleRate float64) []midi.Event {
	start := len(events)
	for _, i := range in {
		events = i.Read(events, frames, sampleRate)
	}
	slices.SortStableFunc(events[start:], func(a, b midi.Event) int { return cmp.Compare(a.Offset, b.Offset) })
	return events
}

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	patches := flag.String("patches", "patches", "the folder patches are kept in")
	presets := flag.String("presets", "editors-presets", "the folder the modules' presets of your own are kept in (the editors example's too)")
	device := flag.String("device", "", "a raw MIDI device to play from too (empty: the first keyboard, if there is one)")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// what the keyboards of the patch play from: the page's keys, and a MIDI keyboard when there is one
	keys := midi.NewQueue()
	in := inputs{keys}
	if d, err := midi.OpenDevice(*device); err == nil {
		defer d.Close()
		in = append(in, d)
		log.Printf("playing from %s too", d.Path())
	} else if *device != "" {
		log.Fatal(err)
	}

	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	ctl := panel.New(p, "Patcher")
	ctl.SetTheme(t)
	hall := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	ctl.Section("Patch")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: in, IR: hall}),
		patcher.Keep(*patches), patcher.Presets(*presets), patcher.Theme(t))
	mix := mixer.New(p)
	mix.Channel("patch", pt.Out(), mixer.StripConfig{Volume: -3})
	p.Out(mix.Master())

	// a patch to start from: a clock of sixteenths through a sequence of C minor, up and down, playing
	// a wavetable pluck; a script on the same clock playing an FM bass; the keyboard playing a lead;
	// all on a mixer whose send A goes to a reverb. A song of four sections: the clock plays sparser
	// in the intro and the outro, the sequence other notes in the chorus
	if len(pt.Patches()) == 0 || pt.Load(pt.Patches()[0]) != nil {
		must := func(id string, err error) string {
			if err != nil {
				log.Fatal(err)
			}
			return id
		}
		clock := must(pt.Add("clock", "", 20, 20))
		seq := must(pt.Add("sequence", "", 230, 20))
		notes := must(pt.Add("notes", "", 450, 20))
		pluck := must(pt.Add("wavetable", "pluck", 680, 20))
		kb := must(pt.Add("keyboard", "", 20, 330))
		lead := must(pt.Add("subtractive", "lead", 230, 330))
		lfo := must(pt.Add("lfo", "", 20, 470))
		mix := must(pt.Add("mixer", "4 channels", 900, 140))
		rev := must(pt.Add("reverb", "", 900, 440))
		bassLine := must(pt.Add("script", "", 460, 250))
		bass := must(pt.Add("dx", "bass", 680, 130))
		song := must(pt.Add("song", "", 1150, 320)) // its sections: the clock and the sequence follow them
		_ = pt.SetText(song, "sections", "Intro 4, Verse 8, Chorus 8, Outro 4")
		_ = pt.SetText(clock, "pattern", "x x . x x . x . x x . x . x x .\nIntro: x . . . x . . . x . . x . . . .\nOutro: x . . . . . x . . . . . x . . .")
		_ = pt.SetText(seq, "values", "C3 Eb3 G3 Bb3 C4 D4\nChorus: F3 Ab3 C4 Eb4 F4 G4")
		pt.SettingsOf(seq).Apply(map[string]float64{"order": 3}) // up and down
		pt.SettingsOf(notes).Apply(map[string]float64{"length": 0.6})
		_ = pt.SetText(bassLine, "code", `// a bass under the pluck: a root a bar, a note on every beat, now and then its fifth
const roots = ["C2", "Ab1", "Bb1", "G1"].map(pitch)
const fifths = genny.bucket([false, false, false, true])

function trigger(ev) {
  const root = roots[Math.floor(ev.step / 16) % roots.length]
  if (ev.step % 4 === 0) note(root, 0.9, 3)
  else if (ev.step % 4 === 3 && fifths.next()) note(root + 7, 0.6, 1)
}
`)
		must(pt.Connect(patcher.Cable{From: clock, To: seq}))
		must(pt.Connect(patcher.Cable{From: seq, Out: 0, To: notes, In: 1}))
		must(pt.Connect(patcher.Cable{From: seq, Out: 1, To: notes, In: 0}))
		must(pt.Connect(patcher.Cable{From: notes, To: pluck}))
		must(pt.Connect(patcher.Cable{From: pluck, To: mix, In: 0}))
		must(pt.Connect(patcher.Cable{From: kb, To: lead}))
		must(pt.Connect(patcher.Cable{From: lead, To: mix, In: 1}))
		must(pt.Connect(patcher.Cable{From: lfo, To: lead, In: 1, Key: "cutoff", Lo: 0.45, Hi: 0.8}))
		must(pt.Connect(patcher.Cable{From: clock, To: bassLine}))
		must(pt.Connect(patcher.Cable{From: bassLine, To: bass}))
		must(pt.Connect(patcher.Cable{From: bass, To: mix, In: 2}))
		must(pt.Connect(patcher.Cable{From: mix, Out: 1, To: rev}))
		must(pt.Connect(patcher.Cable{From: rev, To: mix, In: 4}))
		must(pt.Connect(patcher.Cable{From: mix, Out: 0, To: "out"}))
		pt.SettingsOf(mix).Apply(map[string]float64{"sendA1": 0.4, "sendA2": 0.25, "level1": 0.7, "level3": 0.6, "sendA3": 0.1})
		pt.Move("out", 1150, 160)
	}

	// in the bar at the top, beside the patch's tempo: the volume, the level, the sound, the keys (a
	// button showing them in a sidebar)
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03)).Top()
	ctl.Meter("Out", mix.Master()).Top()
	widgets.Scope(ctl, "Scope", mix.Master()).Top()
	widgets.Keys(ctl, "Keys", func(note int, velocity float64, on bool) {
		if on {
			keys.Push(midi.NoteOnMsg(1, note, max(1, int(velocity*127))))
		} else {
			keys.Push(midi.NoteOffMsg(1, note, 0))
		}
	}, widgets.From(36), widgets.Octaves(4)).Top()

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
