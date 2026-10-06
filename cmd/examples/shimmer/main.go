package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// random picks one of the values at random every time
func random[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Random)
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        70,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// Slow, sparse, soft notes from the A major pentatonic, the shimmer turns them into a rising wash
	voiceCfg := subtractive.PadPreset()
	voiceCfg.Cutoff = 1200.0
	voiceCfg.AmpEnv = *adsr.NewSetting(1.0, 300.0, 0.8, 800.0, 1.0, 1500.0)

	voice := subtractive.NewPoly(voiceCfg, 8).AddTo(p)

	ppq := float64(psalm.PPQ())
	voice.ValueIn(templ.NewDurationTrigger(random(1.5, 2.5, 3.5), random(0.232, 0.31, 0.387), template.Template{
		"pitch": random(57.0, 59.0, 61.0, 64.0, 66.0, 69.0, 71.0, 73.0),
	}).ValueIn(timer.New(random(2*ppq, 3*ppq, 4*ppq, -2*ppq, -4*ppq)).AddTo(p)))

	// Shimmer reverb: an octave up in the feedback loop of a long, large reverb
	cfg := shimmer.DefaultConfig()
	cfg.Semitones = 4
	cfg.Reverb.Decay = 8.0
	cfg.Shimmer = 0.6
	cfg.Mix = 0.5

	reverb := shimmer.New(cfg).AddTo(p).In(voice)

	p.Out(reverb)

	//err := p.ToSoundFile("shimmer.wav", writer.WAV, 40, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
