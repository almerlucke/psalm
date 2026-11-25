// Guitar body: hear the waveguide guitar through its five resonators or through a measured guitar body
// (an impulse response, convolved), or both together. The same notes play every way (the random choices
// are seeded), so run it twice and listen to the difference. The strings are those of a nylon guitar
// (the preset ownGuitar, fitted to a recording of it), the body is the taps on the same guitar. That body is
// part of the synth (waveguide.OwnGuitarBody, embedded in it), and ownGuitar plays through it in NewPoly,
// half and half with the resonators:
//
//	go run ./cmd/examples/guitarbody              both: the guitar with its own body and the measured one, half and half
//	go run ./cmd/examples/guitarbody -blend 0     the guitar as it is, with the resonators of its own
//	go run ./cmd/examples/guitarbody -ir          the string through the measured body alone
//	go run ./cmd/examples/guitarbody -blend 0.3   less of the measured body (0 is the resonators alone, 1 the measured alone)
//	go run ./cmd/examples/guitarbody -piece       a piece of one minute (fingerpicking, a melody, strummed chords) instead of random plucks (all of these take it)
//	go run ./cmd/examples/guitarbody -guitar steelGuitar -blend 0   a steel string guitar (fitted to GuitarSet) with the resonators of its own: the measured body is that of the nylon guitar
//	go run ./cmd/examples/guitarbody -render      write guitarbody.wav, guitarbody-ir.wav and guitarbody-blend.wav (60 s each) and stop
//
// The body is the response of that guitar: taps of a finger (averaged: the response below 600 Hz) and of a harder
// object (one of them: above it). -file takes another response (a wav at 44100 Hz). The response of another guitar,
// from the HAW Hamburg archive of Spanish guitars (CC BY 4.0, doi 10.5281/zenodo.4604576), is not part of the
// repository; its tone is cmd/internal/guitartone:
//
//	go run ./cmd/examples/guitarbody -file "assets/IR44100/Guitar Body HAW 56.wav" -eq -wettrim 2
package main

import (
	"flag"
	"fmt"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/audio"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/cmd/internal/guitartone"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/eq"
	"github.com/almerlucke/psalm/modules/effects/gain"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/sndfile"
	"github.com/almerlucke/sndfile/writer"
)

// random picks one of the values at random every time
func random[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Random)
}

func main() {
	useIR := flag.Bool("ir", false, "the string through the measured body instead of the five resonators")
	file := flag.String("file", "", "the impulse response of the body (a wav at 44100 Hz; the embedded one of the guitar when empty)")
	level := flag.Float64("gain", -3, "the output level in dB")
	ring := flag.Float64("ring", 1, "how long the strings ring out, as a factor of the real nylon guitar they are fitted to (see cmd/tools/guitarcompare); 1 is that guitar")
	tone := flag.Bool("eq", false, "the tone of cmd/internal/guitartone after the guitar: it takes the low end the close microphones heard out of the HAW response and adds the air they missed (the taps of this guitar do not want it)")
	blend := flag.Float64("blend", 0.5, "how much of the sound is the measured body (0 none, 1 only that, the same as -ir); -render writes 0.5 when this is 0 or 1")
	wetTrim := flag.Float64("wettrim", -1.5, "with -blend: the level (dB) of the measured body next to the resonators: a response with much low end (the taps) wants less")
	only := flag.String("only", "", "with -render: write only this version (own, ir or blend)")
	guitarName := flag.String("guitar", "ownGuitar", "the waveguide preset of the strings (ownGuitar is fitted to the recording of this guitar)")
	piece := flag.Bool("piece", false, "play a piece of one minute in E minor (fingerpicked, then strummed) instead of random plucks")
	thump := flag.Float64("thump", -1, "the knock of the pluck on the body, 0-1 (the preset's own when negative)")
	render := flag.Bool("render", false, "write the three versions (own body, measured body, both) to wav files and stop")
	flag.Parse()

	// the measured body of the synth, or one from a file
	ir := waveguide.GuitarIR()
	if *file != "" {
		ir = convolver.IRFromSoundFile(sndfile.MustSoundFile[float32](*file))
	}
	body := func(blend float64) *waveguide.Measured {
		return &waveguide.Measured{IR: ir, SampleRate: 44100, Blend: blend, Trim: *wetTrim}
	}

	if *render {
		w := *blend
		if w <= 0 || w >= 1 {
			w = 0.5
		}
		for _, v := range []struct {
			name, key string
			body      *waveguide.Measured
		}{{"guitarbody.wav", "own", nil}, {"guitarbody-ir.wav", "ir", body(1)}, {"guitarbody-blend.wav", "blend", body(w)}} {
			if *only != "" && *only != v.key {
				continue
			}
			p := patch(*guitarName, v.body, *level, *ring, *thump, *tone, *piece, nil)
			if err := p.ToSoundFile(v.name, writer.WAV, 60, 44100.0, false); err != nil {
				panic(err)
			}
			p.Close()
			fmt.Println("wrote", v.name)
		}
		return
	}

	var m *waveguide.Measured
	switch {
	case *useIR || *blend >= 1:
		m = body(1)
	case *blend > 0:
		m = body(*blend)
	}
	p := patch(*guitarName, m, *level, *ring, *thump, *tone, *piece, paengine.New(44100, 1024))
	defer p.Close()
	_ = p.Listen()
}

// patch is the guitar, a phrase of plucks, and the body (measured: the resonators alone when nil)
func patch(guitar string, measured *waveguide.Measured, level, ring, thump float64, tone, piece bool, engine audio.Engine) *psalm.Psalm {
	genny.Seed(7) // the same notes every time

	bpm := 100.0
	if piece {
		bpm = pieceBPM
	}
	psalm.PushConfig(psalm.Config{
		BPM:        bpm,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(engine)

	// the strings die as those of the real guitar they are fitted to do, each at its own pitch
	preset, ok := waveguide.Presets[guitar]
	if !ok {
		panic("no waveguide preset " + guitar)
	}
	cfg := preset()
	cfg.Release *= 2.5 // a string let go goes on ringing a little
	cfg.Decay *= ring
	cfg.Measured = measured
	if thump >= 0 {
		cfg.Thump = thump
	}
	var out psalm.Modular = waveguide.NewPoly(cfg, 24).AddTo(p)
	if piece {
		play(p, out)
	} else {
		pluck(p, out)
	}
	if tone {
		out = eq.New(guitartone.Tone()).AddTo(p).In(out)
	}

	p.Out(gain.New(level).AddTo(p).In(out))
	return p
}

// pluck is plucks from the E minor pentatonic, across the strings of a guitar
func pluck(p *psalm.Psalm, guitars ...psalm.Modular) {
	ppq := float64(psalm.PPQ())
	trigger := templ.NewDurationTrigger(1.5, random(0.5, 0.7, 0.9), template.Template{
		"pitch": random(40.0, 43.0, 45.0, 47.0, 50.0, 52.0, 55.0, 57.0, 59.0, 62.0, 64.0, 67.0, 71.0),
	}).ValueIn(timer.New(random(0.5*ppq, 0.5*ppq, ppq, -0.5*ppq, 1.5*ppq)).AddTo(p))
	for _, g := range guitars {
		g.ValueIn(trigger)
	}
}
