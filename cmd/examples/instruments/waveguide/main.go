// Waveguide: audition the physically modelled strings, bows, drums and pipes on a page. Choose a
// preset (and yours: type a name, Save, kept in editors-presets/waveguide, the folder the editors
// example uses too), play it on the page's keyboard (the computer's keys: A W S E D ..., or a MIDI
// keyboard), shape every setting of the model, and test it with a phrase: a scale, a chord held, or
// the whole range from the lowest note to the highest, to hear its tuning and its registers.
// Open http://localhost:8087.
//
//	go run ./cmd/examples/instruments/waveguide
//	go run ./cmd/examples/instruments/waveguide -preset violin -hall 0.3 -theme midnight
//	go run ./cmd/examples/instruments/waveguide -list        the presets
package main

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
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/sndfile"
)

// a phrase is notes at times: pitch, seconds from the start, how long it is held (0: until the
// phrase is over), and a velocity
type step struct {
	pitch, at, hold, velocity float64
}

// the phrases the buttons play. They sit in the middle of the keyboard where plucks, bows and
// pipes all speak; the range goes from the lowest note a string or a drum has to the highest
var phrases = map[string][]step{
	"Scale": func() []step {
		var s []step
		for i, p := range []float64{57, 59, 60, 62, 64, 65, 67, 69, 67, 65, 64, 62, 60, 59, 57} {
			s = append(s, step{p, float64(i) * 0.45, 0.4, 0.5 + 0.4*float64(i%3)/2})
		}
		return s
	}(),
	"Chord": {
		{48, 0, 3.5, 0.7}, {55, 0.12, 3.4, 0.7}, {60, 0.24, 3.3, 0.7}, {64, 0.36, 3.2, 0.7}, {67, 0.48, 3.1, 0.7},
	},
	"Range": func() []step {
		var s []step
		for i := 0; i <= 12; i++ { // every fifth semitone, up, from C1 to C7
			p := 24 + float64(i)*6
			s = append(s, step{p, float64(i) * 0.7, 0.6, 0.8})
		}
		return s
	}(),
	"Dynamics": {
		{57, 0, 0.8, 0.15}, {57, 1, 0.8, 0.3}, {57, 2, 0.8, 0.5}, {57, 3, 0.8, 0.7}, {57, 4, 0.8, 0.9}, {57, 5, 0.8, 1.0},
	},
}

var phraseOrder = []string{"Scale", "Chord", "Range", "Dynamics"}

// options are what the page is built with
type options struct {
	start     string  // the preset to start with
	voices    int     // how many notes sound at once
	hall      float64 // how much of the sound goes through the hall (0 is dry)
	presetDir string  // the folder your presets are kept in
	theme     panel.Theme
}

// instrument is the synth on its page
type instrument struct {
	synth  *poly.Poly[*waveguide.Voice]
	mix    *mixer.Mixer
	ctl    *panel.Panel
	editor *editor.Editor
	run    func(phrase string) // plays a test phrase
	stop   func()              // ends the phrase and lets go of the notes
}

// build makes the synth, its mixer and its page on p
func build(p *psalm.Psalm, o options) (*instrument, error) {
	names := slices.Sorted(maps.Keys(waveguide.Presets))
	if !slices.Contains(names, o.start) {
		return nil, fmt.Errorf("no preset %q: one of %v", o.start, names)
	}
	in := &instrument{}

	// the synth, at the preset chosen; its friction bows tune themselves in the background in the
	// first second or two (a note at a pitch not yet tuned plays on an estimate)
	in.synth = waveguide.NewPoly(waveguide.Presets[o.start](), o.voices)
	in.synth.AddTo(p)

	// the synth on the mixer, with the hall as a send when there is one
	in.mix = mixer.New(p)
	sends := map[string]float64{}
	if o.hall > 0 {
		if sf, err := sndfile.NewSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"); err == nil {
			ir := convolver.IRFromSoundFile(sf)
			in.mix.Bus("hall", mixer.StripConfig{}, convolver.New(ir, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))
			sends["hall"] = o.hall
		} else {
			log.Printf("no hall (%v): dry", err)
		}
	}
	in.mix.Channel("waveguide", in.synth, mixer.StripConfig{Volume: -3, Sends: sends})
	p.Out(in.mix.Master())

	// the page: the keys, the phrases, the editor of the synth with its presets
	in.ctl = panel.New(p, "Waveguide")
	in.ctl.SetTheme(o.theme)

	in.ctl.Section("Play")
	widgets.Keys(in.ctl, "Keys", widgets.Play(in.synth), widgets.From(36), widgets.Octaves(4))
	widgets.Scope(in.ctl, "Scope", in.mix.Master())
	in.ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(in.mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
	in.ctl.Meter("Out", in.mix.Master())

	// the phrases: notes timed from a goroutine, each handed to the audio thread between buffers
	in.ctl.Section("Test phrases")
	var playing atomic.Int64 // the phrase running (a new one, or Stop, ends it)
	in.run = func(name string) {
		id := playing.Add(1)
		go func() {
			began := time.Now()
			for _, s := range phrases[name] {
				if d := time.Duration(s.at*float64(time.Second)) - time.Since(began); d > 0 {
					time.Sleep(d)
				}
				if playing.Load() != id {
					return
				}
				hold := s.hold
				if hold <= 0 {
					hold = 1
				}
				msg := psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "duration": hold,
					"amplitude": s.velocity, "message": psalm.Message{"pitch": s.pitch, "velocity": s.velocity}}
				p.Do(func() { in.synth.ReceiveMessage(msg) })
			}
		}()
	}
	in.stop = func() {
		playing.Add(1)
		go p.Do(in.synth.AllNotesOff)
	}
	for _, name := range phraseOrder {
		in.ctl.Button(name, func() { in.run(name) })
	}
	in.ctl.Button("Stop", in.stop)

	in.ctl.Section("Preset")
	in.editor = editor.Add(in.ctl, "Waveguide", in.synth, waveguide.Describe(waveguide.Presets[o.start]()), panel.NoDefault())
	in.editor.Factory(names, func(n string) params.Set { return waveguide.Describe(waveguide.Presets[n]()) })
	if err := in.editor.Keep(filepath.Join(o.presetDir, "waveguide")); err != nil {
		return nil, err
	}
	return in, nil
}

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	start := flag.String("preset", "guitar", "the preset to start with")
	presetDir := flag.String("presets", "editors-presets", "the folder your presets are kept in (a folder for the waveguide in it)")
	voices := flag.Int("voices", 12, "how many notes sound at once")
	hall := flag.Float64("hall", 0.25, "how much of the sound goes through the hall (the Musikverein, 0 is dry)")
	device := flag.String("device", "", "a raw MIDI device to play from too (empty: the first keyboard, if there is one)")
	list := flag.Bool("list", false, "print the presets and stop")
	flag.Parse()

	names := slices.Sorted(maps.Keys(waveguide.Presets))
	if *list {
		fmt.Println(len(names), "presets:")
		fmt.Println(strings.Join(names, " "))
		return
	}
	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	in, err := build(p, options{start: *start, voices: *voices, hall: *hall, presetDir: *presetDir, theme: t})
	if err != nil {
		log.Fatal(err)
	}

	// a MIDI keyboard, when there is one: it plays the synth
	if d, err := midi.OpenDevice(*device); err == nil {
		defer d.Close()
		keyboard.New(d, keyboard.Config{Dynamics: 1, Controls: map[int]string{midi.CCModWheel: "modWheel"}, HoldNotes: true,
			PitchBend: "pitchBend", Pressure: "pressure"}, in.synth).AddTo(p)
		log.Printf("playing from %s too", d.Path())
	} else if *device != "" {
		log.Fatal(err)
	}

	go func() {
		if err := in.ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer in.ctl.Close()
	log.Printf("open http://%s: %d presets, starting with %s", *addr, len(names), *start)
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}
