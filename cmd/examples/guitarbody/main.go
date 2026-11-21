// Guitar body: hear the waveguide guitar through its five resonators or through a measured guitar body
// (an impulse response, convolved), or both together. The same notes play every way (the random choices
// are seeded), so run it twice and listen to the difference. The strings are those of a nylon guitar
// (the preset ownGuitar, fitted to a recording of it), the body is the taps on the same guitar:
//
//	go run ./cmd/examples/guitarbody              both: the guitar with its own body and, next to it, the string through
//	                                              the measured body, half and half
//	go run ./cmd/examples/guitarbody -blend 0     the guitar as it is, with its own body
//	go run ./cmd/examples/guitarbody -ir          the string through the measured body alone
//	go run ./cmd/examples/guitarbody -ir -mix 0.7 less of the body (0 is the string alone, 1 only the body)
//	go run ./cmd/examples/guitarbody -blend 0.3   less of the measured body (0 is the guitar's own alone, 1 the measured alone)
//	go run ./cmd/examples/guitarbody -piece       a miniature (chords and a melody) instead of random plucks (all of these take it)
//	go run ./cmd/examples/guitarbody -render      write guitarbody.wav, guitarbody-ir.wav and guitarbody-blend.wav (60 s each) and stop
//
// The impulse response is assets/IR44100/Guitar Body Own Combined.wav: the body of that guitar, from taps
// of a finger (averaged: the response below 600 Hz) and of a harder object (one of them: above it). The
// finger taps alone are Guitar Body Own Taps.wav (-file, with -wettrim 0). The response of another guitar, from
// the HAW Hamburg archive of Spanish guitars (CC BY 4.0, doi 10.5281/zenodo.4604576), is
// assets/IR44100/Guitar Body HAW 56.wav; its tone is cmd/internal/guitartone:
//
//	go run ./cmd/examples/guitarbody -file "assets/IR44100/Guitar Body HAW 56.wav" -eq -wettrim 2
//
// -file takes another one (at 44100 Hz).
package main

import (
	"flag"
	"fmt"
	"math"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/audio"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/cmd/internal/guitartone"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/eq"
	"github.com/almerlucke/psalm/modules/effects/gain"
	"github.com/almerlucke/psalm/modules/effects/latency"
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
	file := flag.String("file", "assets/IR44100/Guitar Body Own Combined.wav", "the impulse response of the body")
	mix := flag.Float64("mix", 1, "with -ir: how much is the body (0 is the string alone, 1 only the body)")
	level := flag.Float64("gain", -3, "the output level in dB")
	ring := flag.Float64("ring", 1, "how long the strings ring out, as a factor of the real nylon guitar they are fitted to (see cmd/tools/guitarcompare); 1 is that guitar")
	tone := flag.Bool("eq", false, "the tone of cmd/internal/guitartone after the measured body: it takes the low end the close microphones heard out of the HAW response and adds the air they missed (the taps of this guitar do not want it)")
	blend := flag.Float64("blend", 0.5, "the guitar with its own body and the string through the measured body together: how much of the second (0 none, 1 only that, the same as -ir); -render writes 0.5 when this is 0 or 1")
	wetTrimFlag := flag.Float64("wettrim", -1.5, "with -blend: the level (dB) of the string through the measured body next to the other: a response with much low end (the taps) wants less")
	align := flag.Int("align", 0, "with -blend: samples more (or fewer) the guitar with its own body is delayed, to line it up with the convolution")
	only := flag.String("only", "", "with -render: write only this version (own, ir or blend)")
	guitarName := flag.String("guitar", "ownGuitar", "the waveguide preset of the strings (ownGuitar is fitted to the recording of this guitar)")
	piece := flag.Bool("piece", false, "play a miniature, eight bars in E minor, instead of random plucks")
	render := flag.Bool("render", false, "write the three versions (own body, measured body, both) to wav files and stop")
	flag.Parse()

	if *render {
		w := *blend
		if w <= 0 || w >= 1 {
			w = 0.5
		}
		for _, v := range []struct {
			name, key string
			mode      mode
		}{{"guitarbody.wav", "own", mode{}}, {"guitarbody-ir.wav", "ir", mode{ir: true}}, {"guitarbody-blend.wav", "blend", mode{blend: w}}} {
			if *only != "" && *only != v.key {
				continue
			}
			alignSamples, wetTrimDB = *align, *wetTrimFlag
			v.mode.guitar = *guitarName
			p := patch(v.mode, *file, *mix, *level, *ring, *tone, *piece, nil)
			if err := p.ToSoundFile(v.name, writer.WAV, 60, 44100.0, false); err != nil {
				panic(err)
			}
			p.Close()
			fmt.Println("wrote", v.name)
		}
		return
	}

	alignSamples, wetTrimDB = *align, *wetTrimFlag
	m := mode{ir: *useIR || *blend >= 1}
	if !*useIR && *blend > 0 && *blend < 1 {
		m = mode{blend: *blend}
	}
	m.guitar = *guitarName
	p := patch(m, *file, *mix, *level, *ring, *tone, *piece, paengine.New(44100, 1024))
	defer p.Close()
	_ = p.Listen()
}

// mode is the body of the guitar: its own (the zero), the measured one (ir), or the two together, a part
// of the second (blend, between 0 and 1)
type mode struct {
	ir     bool
	blend  float64
	guitar string // the waveguide preset of the strings
}

// wetTrim is how much louder (dB) the string through the measured body would sit next to the guitar with its own
// (-wettrim: -1.5 for the combined response, 0 for the finger taps, 2 for the HAW response, which comes through
// the convolver about 2 dB quieter)
const wetTrim = 0.0

// blendTrim is how much louder (dB) the two guitars together are made: the sounds do not add up the same at all
// frequencies, and together they come out about 3 dB quieter than either alone
const blendTrim = 2.8

// alignSamples is -align, wetTrimDB is -wettrim
var (
	alignSamples int
	wetTrimDB    = wetTrim
)

// patch is the guitar, a phrase of plucks, and the body
func patch(m mode, file string, mix, level, ring float64, tone, piece bool, engine audio.Engine) *psalm.Psalm {
	genny.Seed(7) // the same notes every time

	bpm := 100.0
	if piece {
		bpm = 76
	}
	psalm.PushConfig(psalm.Config{
		BPM:        bpm,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(engine)

	// the strings die as those of the real guitar they are fitted to do, each at its own pitch
	preset, ok := waveguide.Presets[m.guitar]
	if !ok {
		panic("no waveguide preset " + m.guitar)
	}
	cfg := preset()
	cfg.Release *= 2.5 // a string let go goes on ringing a little
	cfg.Decay *= ring
	poly := func(bodyless bool) psalm.Modular {
		c := cfg
		if bodyless {
			c.BodyMix = 0 // the convolution is the body
		}
		return waveguide.NewPoly(c, 16).AddTo(p)
	}
	// the measured body: the string through the convolver, and the tone after it
	body := func(guitar psalm.Modular, mix float64) (out psalm.Modular, delay int) {
		ir := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32](file))
		c := convolver.New(ir, convolver.Config{PartitionSize: 128, Mix: mix, Gain: guitartone.ConvolverGain})
		out = c.AddTo(p).In(guitar)
		if tone {
			out = eq.New(guitartone.Tone()).AddTo(p).In(out)
		}
		// the convolution is late by its blocks and by where the response begins: the sound a signal
		// next to it has to wait for
		peak := 0
		for i, v := range ir[0] {
			if math.Abs(v) > math.Abs(ir[0][peak]) {
				peak = i
			}
		}
		return out, c.SignalLatency() + peak
	}
	// the notes go to every guitar that plays
	notes := func(guitars ...psalm.Modular) {
		if piece {
			play(p, guitars...)
		} else {
			pluck(p, guitars...)
		}
	}

	var out psalm.Modular
	switch {
	case m.blend > 0 && m.blend < 1:
		// two guitars play the same notes: one with its own body, one without it through the measured one,
		// the second a part of the sound, and the first held back by the delay of the convolution so that the
		// two add up instead of combing (the sounds are the same, so the parts add as amplitudes)
		dry, wet := poly(false), poly(true)
		notes(dry, wet)
		measured, delay := body(wet, 1)
		wetDB := 20*math.Log10(m.blend) + wetTrimDB
		dryDB := 20 * math.Log10(1-m.blend)
		late := latency.New(psalm.Stereo, max(0, delay+alignSamples)).AddTo(p).In(dry)
		// two connections to the same input add up
		out = gain.New(blendTrim).AddTo(p).In(gain.New(dryDB).AddTo(p).In(late)).In(gain.New(wetDB).AddTo(p).In(measured))
	case m.ir:
		guitar := poly(true)
		notes(guitar)
		out, _ = body(guitar, mix)
	default:
		guitar := poly(false)
		notes(guitar)
		out = guitar
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

// play is a miniature: eight bars in E minor, a chord strummed on the first beat of each bar with a
// melody over it, played again and again. The chords are open guitar chords, low string first
func play(p *psalm.Psalm, guitars ...psalm.Modular) {
	em := []float64{40, 47, 52, 55, 59, 64}
	c := []float64{48, 52, 55, 60, 64}
	g := []float64{43, 47, 50, 55, 59, 67}
	d := []float64{50, 57, 62, 66}
	am := []float64{45, 52, 57, 60, 64}
	b7 := []float64{47, 51, 57, 59, 66}

	// a chord rings the whole bar (the last one the whole two), and goes on ringing when let go
	chord := func(pitches []float64, bars float64) score.Note {
		n := score.Chord(pitches, 4*bars)
		n.Hold = 4 * bars
		return n
	}
	chords := []score.Note{chord(em, 1), chord(c, 1), chord(g, 1), chord(d, 1), chord(em, 1), chord(am, 1), chord(b7, 1), chord(em, 2)}

	// the melody: a note rings a bit longer than it is long
	note := func(pitch, beats float64) score.Note {
		n := score.N(pitch, beats)
		n.Hold = beats * 1.6
		return n
	}
	melody := []score.Note{
		note(71, 1.5), note(67, 0.5), note(64, 2), // Em
		note(67, 1), note(64, 1), note(72, 1.5), note(71, 0.5), // C
		note(74, 1.5), note(71, 0.5), note(67, 2), // G
		note(66, 1), note(69, 1), note(74, 2), // D
		note(71, 1), note(76, 1), note(74, 1), note(71, 1), // Em
		note(72, 1.5), note(69, 0.5), note(64, 2), // Am
		note(63, 1), note(66, 1), note(69, 1), note(71, 1), // B7
		note(64, 4), score.Rest(4), // Em, and the last chord rings alone
	}

	for _, part := range []psalm.Modular{
		score.NewSequencer(chords, score.Config{
			Amp: 0.5, Velocity: 0.55, Loop: true,
			Feel: score.Feel{Strum: 38, Timing: 6, Velocity: 0.08, Seed: 3},
		}).AddTo(p),
		score.NewSequencer(melody, score.Config{
			Amp: 0.62, Velocity: 0.75, Loop: true,
			Feel: score.Feel{Timing: 8, Velocity: 0.1, Seed: 5},
		}).AddTo(p),
	} {
		for _, g := range guitars {
			g.ValueIn(part)
		}
	}
}
