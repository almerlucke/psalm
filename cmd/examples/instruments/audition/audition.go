// Package audition is the page the examples in cmd/examples/instruments share: one polyphonic synth
// on a mixer with a hall, its keys (the page's, the computer's, a MIDI keyboard), test phrases, and the
// editor of every setting with its presets (a preset chosen is the synth made at it, swapped in).
// An example describes its synth with a Spec and calls Main.
package audition

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/sndfile"
)

// Step is a note of a phrase: its pitch, seconds from the start of the phrase, how long it is held
// (0: a second) and a velocity
type Step struct {
	Pitch, At, Hold, Velocity float64
}

// Spec describes a synth for the page
type Spec struct {
	Name     string                                        // "Waveguide": the title of the page and of the mixer channel
	Presets  []string                                      // the names of its presets, sorted
	Default  string                                        // the preset it starts on
	Describe func(preset string) params.Set                // its settings at a preset
	New      func(preset string, voices int) psalm.Modular // the synth at a preset: a polyphonic synth (*poly.Poly)
	Voices   int                                           // how many notes sound at once (the flag's default)
	Hall     float64                                       // how much of the sound goes through the hall (the flag's default)
	Phrases  map[string][]Step                             // the test phrases
	Order    []string                                      // their order on the page
	Keep     string                                        // the folder of your presets in the presets folder ("waveguide")
}

// Options are what a page is built with
type Options struct {
	Start     string      // the preset to start on
	Voices    int         // how many notes sound at once
	Hall      float64     // how much of the sound goes through the hall (0 is dry)
	PresetDir string      // the folder your presets are kept in
	Theme     panel.Theme // the theme of the page
}

// Instrument is the synth on its page
type Instrument struct {
	Synth  psalm.Modular
	Mix    *mixer.Mixer
	Page   *panel.Panel
	Editor *editor.Editor
	Swap   *poly.Swap          // a preset chosen on the page is swapped in
	Run    func(phrase string) // plays a test phrase
	Stop   func()              // ends the phrase and lets go of the notes
}

// Build makes the synth, its mixer and its page on p
func Build(p *psalm.Psalm, spec Spec, o Options) (*Instrument, error) {
	if !slices.Contains(spec.Presets, o.Start) {
		return nil, fmt.Errorf("no preset %q: one of %v", o.Start, spec.Presets)
	}
	in := &Instrument{}

	// the synth, at the preset chosen
	in.Synth = spec.New(o.Start, o.Voices)
	in.Synth.(interface {
		AddTo(psalm.Patcher) psalm.Modular
	}).AddTo(p)

	// the synth on the mixer, with the hall as a send when there is one
	in.Mix = mixer.New(p)
	sends := map[string]float64{}
	if o.Hall > 0 {
		if sf, err := sndfile.NewSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"); err == nil {
			ir := convolver.IRFromSoundFile(sf)
			in.Mix.Bus("hall", mixer.StripConfig{}, convolver.New(ir, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))
			sends["hall"] = o.Hall
		} else {
			log.Printf("no hall (%v): dry", err)
		}
	}
	in.Mix.Channel(strings.ToLower(spec.Name), in.Synth, mixer.StripConfig{Volume: -3, Sends: sends})
	p.Out(in.Mix.Master())

	// the page: the keys, the phrases, the editor of the synth with its presets
	in.Page = panel.New(p, spec.Name)
	in.Page.SetTheme(o.Theme)

	in.Page.Section("Play")
	widgets.Keys(in.Page, "Keys", widgets.Play(in.Synth), widgets.From(36), widgets.Octaves(4))
	widgets.Scope(in.Page, "Scope", in.Mix.Master())
	in.Page.Fader("Volume", -40, 6, panel.Linear, panel.Message(in.Mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
	in.Page.Meter("Out", in.Mix.Master())

	// the phrases: notes timed from a goroutine, each handed to the audio thread between buffers
	in.Page.Section("Test phrases")
	var playing atomic.Int64 // the phrase running (a new one, or Stop, ends it)
	in.Run = func(name string) {
		id := playing.Add(1)
		go func() {
			began := time.Now()
			for _, s := range spec.Phrases[name] {
				if d := time.Duration(s.At*float64(time.Second)) - time.Since(began); d > 0 {
					time.Sleep(d)
				}
				if playing.Load() != id {
					return
				}
				hold := s.Hold
				if hold <= 0 {
					hold = 1
				}
				msg := psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "duration": hold,
					"amplitude": s.Velocity, "message": psalm.Message{"pitch": s.Pitch, "velocity": s.Velocity}}
				p.Do(func() { in.Synth.ReceiveMessage(msg) })
			}
		}()
	}
	in.Stop = func() {
		playing.Add(1)
		if all, ok := in.Synth.(interface{ AllNotesOff() }); ok {
			go p.Do(all.AllNotesOff)
		}
	}
	for _, name := range spec.Order {
		in.Page.Button(name, func() { in.Run(name) })
	}
	in.Page.Button("Stop", in.Stop)

	in.Page.Section("Preset")
	in.Editor = editor.Add(in.Page, spec.Name, in.Synth, spec.Describe(o.Start), panel.NoDefault())
	// a preset is the synth made at it, its voices swapped in (a model, a mode list or a word isn't a
	// setting of the one before)
	in.Swap = poly.NewSwap(p.Do, in.Synth, func(n string) psalm.Modular {
		var m psalm.Modular
		p.Within(func() { m = spec.New(n, o.Voices) })
		return m
	})
	in.Editor.FactoryApply(spec.Presets, spec.Describe, in.Swap.Load)
	if err := in.Editor.Keep(filepath.Join(o.PresetDir, spec.Keep)); err != nil {
		return nil, err
	}
	return in, nil
}

// Main reads the flags, builds the page and plays until terminated
func Main(spec Spec) {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	start := flag.String("preset", spec.Default, "the preset to start with")
	presetDir := flag.String("presets", "editors-presets", "the folder your presets are kept in (a folder for the synth in it)")
	voices := flag.Int("voices", spec.Voices, "how many notes sound at once")
	hall := flag.Float64("hall", spec.Hall, "how much of the sound goes through the hall (the Musikverein, 0 is dry)")
	device := flag.String("device", "", "a raw MIDI device to play from too (empty: the first keyboard, if there is one)")
	list := flag.Bool("list", false, "print the presets and stop")
	flag.Parse()

	if *list {
		fmt.Println(len(spec.Presets), "presets:")
		fmt.Println(strings.Join(spec.Presets, " "))
		return
	}
	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	in, err := Build(p, spec, Options{Start: *start, Voices: *voices, Hall: *hall, PresetDir: *presetDir, Theme: t})
	if err != nil {
		log.Fatal(err)
	}

	// a MIDI keyboard, when there is one: it plays the synth
	if d, err := midi.OpenDevice(*device); err == nil {
		defer d.Close()
		keyboard.New(d, keyboard.Config{Dynamics: 1, Controls: map[int]string{midi.CCModWheel: "modWheel"}, HoldNotes: true,
			PitchBend: "pitchBend", Pressure: "pressure"}, in.Synth).AddTo(p)
		log.Printf("playing from %s too", d.Path())
	} else if *device != "" {
		log.Fatal(err)
	}

	go func() {
		if err := in.Page.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer in.Page.Close()
	log.Printf("open http://%s: %d presets, starting with %s", *addr, len(spec.Presets), *start)
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}
