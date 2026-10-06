package main

// "Weather": a generative piece in D dorian. Nothing is written note by note: generators play the
// parts (streams), the arrangement shapes the form, and every section is a scene with its own
// parameters. The song loops and is different every time around.
//
//	pad    wavetable chords, from a shuffle bag of the chords of the key, changed every two bars
//	glass  pulsar notes wandering through the scale in random rhythms, the formant set per section
//	bass   a tresillo on the root of the chord, the cutoff set per section
//
// The pad and the glass play through a granulator whose scene (mix, freeze, octave) comes from
// the section parameters, while streams also drive the granulator's grains (messages) and the
// pan of the glass strip (values). Random choices come from the patch seed:
//
//	go run ./cmd/examples/generative                               play it
//	go run ./cmd/examples/generative -render weather.wav -seed 5   render a pass

import (
	"flag"
	"fmt"

	"github.com/almerlucke/genny/function"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
)

// the sections are scenes: the parameters go to the templates of the streams (formant, cutoff)
// and to the granulator (mix, freeze, octave)
var form = arrange.NewForm(
	arrange.Section{Name: "Mist", Bars: 8, Params: map[string]any{
		"formant": 2400.0, "cutoff": 300.0, "mix": 0.3, "freeze": 0.0, "octave": 0.0}},
	arrange.Section{Name: "Drops", Bars: 8, Params: map[string]any{
		"formant": 2400.0, "cutoff": 300.0, "mix": 0.4, "freeze": 0.0, "octave": 0.1}},
	arrange.Section{Name: "Pulse", Bars: 8, Params: map[string]any{
		"formant": 3200.0, "cutoff": 600.0, "mix": 0.3, "freeze": 0.0, "octave": 0.0}},
	arrange.Section{Name: "Freeze", Bars: 8, Params: map[string]any{
		"formant": 2000.0, "cutoff": 900.0, "mix": 0.7, "freeze": 1.0, "octave": 0.3}},
	arrange.Section{Name: "Bloom", Bars: 8, Dynamics: 1.1, Params: map[string]any{
		"formant": 1800.0, "cutoff": 1200.0, "mix": 0.5, "freeze": 0.0, "octave": 0.2}},
	arrange.Section{Name: "Fade", Bars: 8, Params: map[string]any{
		"formant": 2400.0, "cutoff": 300.0, "mix": 0.6, "freeze": 0.0, "octave": 0.1}},
)

// choose picks one of the values at random every time, from the patch seed
func choose[T any](values ...T) *function.Function[T] {
	return function.New(func() T { return values[psalm.Rand().IntN(len(values))] })
}

// shuffled plays the values in a random order, every value once per pass, from the patch seed
func shuffled[T any](values ...T) func() T {
	vals := append([]T(nil), values...)
	i := len(vals)
	return func() T {
		if i >= len(vals) {
			psalm.Rand().Shuffle(len(vals), func(a, b int) { vals[a], vals[b] = vals[b], vals[a] })
			i = 0
		}
		i++
		return vals[i-1]
	}
}

func main() {
	renderTo := flag.String("render", "", "render one pass to this WAV file instead of playing it")
	seed := flag.Uint64("seed", 0, "seed for the random choices: the same seed plays the same, 0 is new every run")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}

	psalm.PushConfig(psalm.Config{BPM: 84, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	// --- harmony: the chords of D dorian, voice led, chosen every two bars ---

	key := harmony.MustKey("D dorian")
	nextChord := shuffled(key.MustProgression("i ii III IV v VII")...)
	chord := key.Triad(1)
	voicing := chord.Close(57)

	// the arrangement comes first in the patch, so the chord changes before the parts play
	arrangement := arrange.NewArrangement(form).
		PrintSections(true).
		OnBar(func(pos arrange.Position) {
			if pos.Bar%2 == 0 {
				chord = nextChord()
				voicing = harmony.VoiceLead(voicing, chord)
			}
		})
	arrangement.AddTo(p)

	// --- instruments ---

	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Random(64, 6, 32, 5)), 12).AddTo(p)
	glass := pulsar.NewPoly(pulsar.GlassPreset(), 8).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)

	// --- the parts, played by generators ---

	padStream := &arrange.Stream{
		Rhythm:  sequence.NewLoop(8.0),
		Message: template.Template{"pitch": function.New(func() []float64 { return voicing })}, // a slice: a chord
		Amp:     0.25,
		Legato:  1.0,
	}
	form.Stream(p, pad, "pad", func(s arrange.Section) *arrange.Stream {
		if s.Name == "Freeze" {
			return nil
		}
		return padStream // the same stream in every section: it goes on without a break
	})

	// glass wanders through the scale a step or two at a time
	glassPitch := 69.0
	glassStream := &arrange.Stream{
		Rhythm: choose(0.5, 0.5, 0.25, 0.25, 1.0, -0.5, -1.0, -0.25),
		Message: template.Template{
			"pitch": function.New(func() float64 {
				glassPitch = key.Step(glassPitch, psalm.Rand().IntN(5)-2)
				if glassPitch > 84 || glassPitch < 62 {
					glassPitch = key.Step(74, 0)
				}
				return glassPitch
			}),
			"formant": template.NewParameter("formant", 2400.0), // from the section
		},
		Amp:    choose(0.15, 0.2, 0.25, 0.3),
		Legato: 0.8,
	}
	form.Stream(p, glass, "glass", func(s arrange.Section) *arrange.Stream {
		if arrange.In(s, "Mist", "Fade") {
			return nil
		}
		return glassStream
	})

	// the bass: a tresillo on the root of the chord, now and then an octave up
	bassStream := &arrange.Stream{
		Rhythm: sequence.NewLoop(1.5, 1.5, 1.0),
		Message: template.Template{
			"pitch":  function.New(func() float64 { return chord.BassPitch(36) + choose(0.0, 0.0, 0.0, 12.0).Generate() }),
			"cutoff": template.NewParameter("cutoff", 300.0), // from the section
		},
		Amp:    choose(0.35, 0.4, 0.45),
		Legato: 0.7,
	}
	form.Stream(p, bass, "bass", func(s arrange.Section) *arrange.Stream {
		if !arrange.In(s, "Pulse", "Freeze", "Bloom") {
			return nil
		}
		return bassStream
	})

	// --- the mix: pad and glass through the granulator, everything in a hall ---

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay = 3.5
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))

	gcfg := granulator.DefaultConfig()
	gcfg.GrainSize = 200
	gcfg.Spray = 200
	gcfg.Density = 30
	gran := granulator.New(gcfg)
	gran.AddTo(p)
	mix.Group("atmos", mixer.StripConfig{Sends: map[string]float64{"hall": 0.4}}, gran)
	mix.Channel("pad", pad, mixer.StripConfig{Group: "atmos", Pan: -0.2})
	mix.Channel("glass", glass, mixer.StripConfig{Group: "atmos", Pan: 0.3, Volume: -4})
	mix.Channel("bass", bass, mixer.StripConfig{Sends: map[string]float64{"hall": 0.1}})
	p.Out(mix.Master())

	// the scenes: the granulator takes the section parameters it knows (mix, freeze, octave)
	arrangement.SendParams(gran)

	// streams can drive any module: new grain settings every bar for the granulator (messages)...
	grains := &arrange.Stream{
		Output: arrange.Messages,
		Rhythm: sequence.NewLoop(4.0),
		Message: template.Template{
			"grainSize":   choose(60.0, 120.0, 200.0, 350.0),
			"spray":       choose(50.0, 150.0, 400.0),
			"pitchSpread": choose(0.0, 0.1, 0.3),
		},
	}
	form.Stream(p, gran, "grains", func(s arrange.Section) *arrange.Stream {
		if arrange.In(s, "Drops", "Pulse", "Bloom") {
			return grains
		}
		return nil
	})

	// ...and the glass wandering through the stereo field: values for the pan input (2) of its strip
	glassPan := 0.3
	pan := form.NewStream(p, "glass pan", func(s arrange.Section) *arrange.Stream {
		return &arrange.Stream{
			Output: arrange.Values,
			Rhythm: sequence.NewLoop(1.0),
			Message: template.Template{"value": function.New(func() float64 {
				glassPan = max(-0.8, min(0.8, glassPan+(psalm.Rand().Float64()-0.5)*0.4))
				return glassPan
			})},
		}
	})
	mix.Strip("glass").ValueIn(pan, 0, 2)

	fmt.Println("Weather")
	fmt.Print(arrangement.Schedule())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 10})
		if err != nil {
			panic(err)
		}
		fmt.Println(stats)
		fmt.Print(mix.Meters())
		return
	}
	_ = p.Listen()
}
