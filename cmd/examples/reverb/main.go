package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// seq plays the values in order and loops (a bucket would shuffle them)
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// random picks one of the values at random every time
func random[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Random)
}

// chords plays a chord progression, every chord lasts the given number of beats. The pitch
// generator produces a slice per chord, which makes the template send one note message per pitch
func chords(p *psalm.Psalm, synth psalm.Modular, beats float64, amp float64, progression ...[]float64) {
	beatSec := 60.0 / float64(psalm.BPM())

	synth.ValueIn(templ.NewDurationTrigger(beats*beatSec*0.95, amp, template.Template{
		"pitch": seq(progression...),
	}).ValueIn(timer.WithTicks(beats * float64(psalm.PPQ())).AddTo(p)))
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        80,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// Sparse random plucks from the E minor pentatonic
	plucks := subtractive.NewPoly(subtractive.PluckPreset(), 8).AddTo(p)

	ppq := float64(psalm.PPQ())
	plucks.ValueIn(templ.NewDurationTrigger(0.2, random(0.2, 0.3, 0.4), template.Template{
		"pitch": random(64.0, 67.0, 69.0, 71.0, 74.0, 76.0, 79.0),
	}).ValueIn(timer.New(random(0.5*ppq, ppq, 1.5*ppq, 2*ppq, -ppq, -2*ppq)).AddTo(p)))

	// A soft pad underneath: Em C G D, one bar per chord
	pad := subtractive.NewPoly(subtractive.HollowPadPreset(), 12).AddTo(p)
	chords(p, pad, 4, 0.08,
		[]float64{52, 59, 64}, // Em: E3 B3 E4
		[]float64{48, 55, 64}, // C:  C3 G3 E4
		[]float64{55, 59, 62}, // G:  G3 B3 D4
		[]float64{50, 57, 62}, // D:  D3 A3 D4
	)

	// A large, dark hall: long decay with damped highs, a little pre-delay and gentle modulation
	reverb := fdnreverb.New(fdnreverb.Config{
		Size:      0.8,
		Decay:     4.5,
		Damping:   0.6,
		PreDelay:  30.0,
		Diffusion: 0.75,
		ModDepth:  0.5,
		ModRate:   0.5,
		Width:     1.0,
		Mix:       0.4,
	}).AddTo(p).In(plucks, 0, 0, pad, 0, 0)

	p.Out(reverb)

	//err := p.ToSoundFile("reverb.wav", writer.WAV, 30, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
