package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/sndfile"
)

// random picks one of the values at random every time
func random[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Random)
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

	// Sparse random plucks from the D minor pentatonic
	plucks := subtractive.NewPoly(subtractive.PluckPreset(), 8).AddTo(p)

	ppq := float64(psalm.PPQ())
	plucks.ValueIn(templ.NewDurationTrigger(0.2, random(0.23, 0.344, 0.459), template.Template{
		"pitch": random(62.0, 65.0, 67.0, 69.0, 72.0, 74.0, 77.0),
	}).ValueIn(timer.New(random(0.5*ppq, ppq, 1.5*ppq, -ppq, -2*ppq)).AddTo(p)))

	// Convolution reverb with a synthetic 3 second impulse response. To use a recorded impulse
	// response (at the same sample rate) instead:
	ir := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	//ir := convolver.SyntheticReverbIR(3.0, 0.6, psalm.SampleRate(), 1)

	reverb := convolver.New(ir, convolver.Config{PartitionSize: 512, Mix: 0.35, Gain: 0.3}).AddTo(p).In(plucks)

	p.Out(reverb)

	//err := p.ToSoundFile("convolution.wav", writer.WAV, 30, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
