package main

// "Aurora": a slow orchestral piece in D minor, the showcase of psalm. Every synth plays a part
// of the orchestra and every effect has a role:
//
//	violins     wavetable strings on a generated sawtooth table, chorus, an LFO on the position
//	celli       subtractive strings
//	pizzicato   FM upright bass
//	harp        FM harp through freeverb and the granulator (frozen in the interlude)
//	flute       FM flute with a pitch shifted octave above, like a piccolo shadow
//	horns       DX soft brass chords
//	horn solo   FM french horn counter melody
//	soprano     pulsar vowel lead, the theme in the climax
//	choir       grains choir pad on a choir recording and a pulsar choir, in a convolved church
//	celesta     FM celesta through a pingpong delay into a shimmer reverb
//	bell pad    DX bell pad through a flanger (the mixer lines everything up with the vocoder)
//	voices      speech vocoding a wavetable pad
//	droplets    pulsar raindrops
//	aurora      wavetable drone on a generated random spectra table
//	piano       the modelled grand: rolled chords at dawn, weight under the chorale, the last cadence
//
// The strings run through a ladder lowpass that opens at dawn (the prelude) and closes at dusk
// (the coda), which slows down in its last four bars.
//
// Everything plays through a mixer: the strings and the choir are groups with their own inserts
// (the dawn filter, the church), the hall reverb and the shimmering sky are effect buses every
// instrument sends to in its own amount, and a lookahead limiter guards the master.
//
// The piece is built with the arrange and score packages: the form lists the sections, every part
// is written per section in beats (the melodies in score notation) and the arrangement automates
// the string filter, the granulator and the tempo. The piece loops, the schedule is printed at start
//
//	go run ./cmd/examples/showcase                                    play it
//	go run ./cmd/examples/showcase -render aurora.wav -stems -seed 7  render it, a stem per mixer strip
//	go run ./cmd/examples/showcase -profile                           the time every module takes
//
// The files: music.go (the chords, the form and the melodies), orchestra.go (the instruments on
// the mixer), parts.go (what every instrument plays, section by section) and arrangement.go (the
// automation and the tempo).

import (
	"flag"
	"fmt"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/tools/render"
)

func main() {
	var (
		renderTo = flag.String("render", "", "render one pass of the piece to this WAV file instead of playing it")
		stems    = flag.Bool("stems", false, "with -render, also write every mixer strip to its own file")
		profile  = flag.Bool("profile", false, "render the piece offline and report the time every module takes")
		seed     = flag.Uint64("seed", 0, "seed for the random parts (grains, feel, pans): the same seed plays the same, 0 is new every run")
	)
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}

	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})
	var p *psalm.Psalm
	if *renderTo != "" || *profile {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	o := newOrchestra(p)
	if writeParts(p, o) == 0 {
		fmt.Println("harmony check: every part fits its chords")
	}
	arrangement := o.arrangement()
	arrangement.AddTo(p)

	fmt.Println("Aurora")
	fmt.Print(arrangement.Schedule())

	p.Out(o.mix.Master())

	opts := render.Options{Seconds: arrangement.Duration(), Tail: 10}
	switch {
	case *profile:
		fmt.Print(render.RunProfiled(p, opts))
	case *renderTo != "":
		var sources map[string]psalm.Modular
		if *stems {
			sources = render.MixerStems(o.mix)
		}
		stats, err := render.Stems(p, *renderTo, opts, sources)
		if err != nil {
			panic(err)
		}
		fmt.Println(stats)
		fmt.Print(o.mix.Meters())
	default:
		_ = p.Listen()
	}
}
