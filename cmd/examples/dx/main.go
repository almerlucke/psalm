package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/synths/dx"
)

// Plays every voice of all DX7 compatible SysEx banks in assets/dx7 (or of the bank files given as
// arguments), followed by the built in patches, two bars each, and loops:
//
//	go run ./cmd/examples/dx
//	go run ./cmd/examples/dx path/to/bank.syx path/to/other.syx

const voiceBeats = 8.0 // 2 bars per voice

// seq plays the values in order and loops
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// voice is a patch with the bank it came from
type voice struct {
	bank  string
	patch dx.Patch
}

func loadVoices() []voice {
	files := os.Args[1:]
	if len(files) == 0 {
		files, _ = filepath.Glob("assets/dx7/*.syx")
		slices.Sort(files)
	}

	var voices []voice
	for _, file := range files {
		patches, err := dx.LoadSysEx(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "can't load", file+":", err)
			continue
		}
		for _, patch := range patches {
			voices = append(voices, voice{filepath.Base(file), patch})
		}
	}

	names := make([]string, 0, len(dx.Presets))
	for name := range dx.Presets {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		voices = append(voices, voice{"built in", dx.Presets[name]()})
	}
	return voices
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        100,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	voices := loadVoices()
	beatSec := 60.0 / float64(psalm.BPM())
	total := voiceBeats * float64(len(voices))

	// Every voice plays the same phrase in its slot: two chords and a short melody on top
	var mix []any

	for i, v := range voices {
		start := voiceBeats * float64(i)
		fmt.Printf("%6.1fs  %3d  %-10s %s\n", start*beatSec, i+1, v.bank, v.patch.Name)

		synth := dx.NewPoly(v.patch, 8, psalm.Stereo).AddTo(p)

		// rests before and after the slot, so each voice only plays in its own two bars
		var chordSteps, melodySteps []any
		if start > 0 {
			chordSteps = append(chordSteps, -start)
			melodySteps = append(melodySteps, -start)
		}
		chordSteps = append(chordSteps, 4.0, 4.0)
		melodySteps = append(melodySteps, -0.5, 0.5, 0.5, 0.5, 1.0, 1.0, 0.5, 0.5, 0.5, 0.5, 2.0)
		if after := total - start - voiceBeats; after > 0 {
			chordSteps = append(chordSteps, -after)
			melodySteps = append(melodySteps, -after)
		}

		synth.ValueIn(templ.NewDurationTrigger(3.6*beatSec, 0.15, template.Template{
			"pitch":    seq([]float64{48, 55, 60, 64}, []float64{45, 52, 57, 60}), // C, Am
			"velocity": 0.7,
		}).ValueIn(timer.WithSteps(steps.QuickSteps(chordSteps...)).AddTo(p)))

		synth.ValueIn(templ.NewDurationTrigger(seq(0.45, 0.45, 0.45, 0.9, 0.9, 0.45, 0.45, 0.45, 0.45, 1.8), 0.2, template.Template{
			"pitch":    seq(72.0, 74.0, 76.0, 79.0, 76.0, 74.0, 72.0, 71.0, 72.0, 69.0),
			"velocity": seq(0.9, 0.6, 0.7, 1.0, 0.8, 0.6, 0.7, 0.6, 0.8, 0.9),
		}).ValueIn(timer.WithSteps(steps.QuickSteps(melodySteps...)).AddTo(p)))

		mix = append(mix, synth, 0, 0)
	}

	fmt.Printf("%6.1fs  (loops)\n", total*beatSec)

	rcfg := fdnreverb.DefaultConfig()
	rcfg.Decay = 2.0
	rcfg.Mix = 0.2

	p.Out(fdnreverb.New(rcfg).AddTo(p).In(mix...))

	_ = p.Listen()
}
