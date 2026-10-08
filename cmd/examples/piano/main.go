package main

// A tour of the physically modelled piano. The grand plays a nocturne (with the sustain pedal
// changed every bar, the strings ringing along), then shows its dynamics, the whole keyboard, the
// sustain pedal (and half pedaling), the sostenuto and the soft pedal; after that every other preset plays a short piece that suits
// it. All music is original.
//
//	go run ./cmd/examples/piano                                  play it
//	go run ./cmd/examples/piano -render piano.wav -seed 3        render a pass
//
// The files: main.go (the sections of the tour and the pianos), parts.go (what every piano plays)
// and arrangement.go (the tempo of every section and the pedals).

import (
	"flag"
	"fmt"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

type (
	S = arrange.Section
	N = score.Note
)

const (
	nocturne = "Grand: a nocturne, the pedal changed every bar"
	dynamics = "Grand: dynamics, one chord from pianissimo to fortissimo"
	keyboard = "Grand: the whole keyboard, from A0 to the top"
	pedal    = "Grand: pedal, two bars dry, a bar half pedal, a bar with the pedal down"
	sost     = "Grand: sostenuto, a low C held under short chords"
	soft     = "Grand: the soft pedal, a phrase and again with the soft pedal down"
	pop      = "Bright: pop chords"
	hymn     = "Upright: a hymn"
	boogie   = "Honky tonk: boogie"
	ambient  = "Felt: ambient"
	musicBox = "Toy: a music box"
)

// the parade: the other presets play the same phrase one after the other
var parade = []struct {
	name, section string
	preset        func() piano.Config
}{
	{"concert", "Parade: concert, the grand from the audience", piano.ConcertPreset},
	{"warm", "Parade: warm, a grand in a jazz club", piano.WarmPreset},
	{"baby", "Parade: baby, a small grand", piano.BabyPreset},
	{"vintage", "Parade: vintage, an old upright", piano.VintagePreset},
	{"cinematic", "Parade: cinematic, soft and dark from the hall", piano.CinematicPreset},
}

func main() {
	renderTo := flag.String("render", "", "render one pass to this WAV file instead of playing it")
	stems := flag.Bool("stems", false, "with -render, also write every piano to its own file")
	seed := flag.Uint64("seed", 0, "seed for the human feel: the same seed plays the same, 0 is new every run")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}

	psalm.PushConfig(psalm.Config{BPM: 66, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	sections := []S{
		{Name: nocturne, Bars: 8},
		{Name: dynamics, Bars: 4},
		{Name: keyboard, Bars: 3},
		{Name: pedal, Bars: 4},
		{Name: sost, Bars: 4},
		{Name: soft, Bars: 4},
		{Name: pop, Bars: 4},
		{Name: hymn, Bars: 4},
		{Name: boogie, Bars: 4},
		{Name: ambient, Bars: 4},
		{Name: musicBox, Bars: 4},
	}
	for _, pr := range parade {
		sections = append(sections, S{Name: pr.section, Bars: 4})
	}
	form := arrange.NewForm(sections...)

	// --- the pianos, all on the mixer with a hall: the Musikverein in Vienna, a recorded impulse
	// response through the convolver ---

	mix := mixer.New(p)
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	mix.Bus("hall", mixer.StripConfig{}, convolver.New(vienna, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))

	type instrument struct {
		name   string
		preset func() piano.Config
		voices int
		hall   float64
	}
	pianos := map[string]psalm.Modular{}
	for _, in := range []instrument{
		{"grand", piano.GrandPreset, 24, 0.5},
		{"bright", piano.BrightPreset, 12, 0.4},
		{"upright", piano.UprightPreset, 12, 0.35},
		{"honkyTonk", piano.HonkyTonkPreset, 12, 0.12},
		{"felt", piano.FeltPreset, 16, 0.45},
		{"toy", piano.ToyPreset, 8, 0.35},
		{"concert", piano.ConcertPreset, 16, 0.5},
		{"warm", piano.WarmPreset, 16, 0.45},
		{"baby", piano.BabyPreset, 16, 0.25},
		{"vintage", piano.VintagePreset, 16, 0.2},
		{"cinematic", piano.CinematicPreset, 16, 0.6},
	} {
		synth := piano.NewPoly(in.preset(), in.voices).AddTo(p)
		pianos[in.name] = synth
		mix.Channel(in.name, synth, mixer.StripConfig{Volume: -3.5, Sends: map[string]float64{"hall": in.hall}})
	}
	p.Out(mix.Master())

	writeParts(p, form, pianos)
	arrangement := newArrangement(form, pianos)
	arrangement.AddTo(p)

	fmt.Println("The piano")
	fmt.Print(arrangement.Schedule())

	if *renderTo != "" {
		var sources map[string]psalm.Modular
		if *stems {
			sources = render.MixerStems(mix)
		}
		stats, err := render.Stems(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 5}, sources)
		if err != nil {
			panic(err)
		}
		fmt.Println(stats)
		fmt.Print(mix.Meters())
		return
	}
	_ = p.Listen()
}
