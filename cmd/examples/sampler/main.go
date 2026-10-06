package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/sndfile"
)

func main() {
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	sf := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4)

	pl := player.New(player.Config{
		SndFile:          sf,
		StartOffset:      0.1,
		EndOffset:        0.2,
		UseZeroCrossings: true,
		Loop:             true,
		Speed:            1.0,
		Amp:              1.1,
	}).AddTo(p).ValueIn(templ.New(template.Template{
		"startOffset": bucket.New(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8).SetMode(bucket.Indexed).Loop(true),
		"offsetRange": bucket.New(0.05, 0.1, 0.2, 0.3, 0.4).SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 8)).AddTo(p)))

	p.Out(freeverb.New(0.7, 0.8, 0.9, 0.044, 0.71).AddTo(p).In(
		pingpong.New(pingpong.Config{Time: 1.75, Feedback: 0.1, Mix: 0.09}).AddTo(p).In(
			flanger.New(flanger.Config{Rate: 0.05, Manual: 1, Depth: 18, Feedback: 0.85, Stereo: 0.5, Shape: flanger.Triangle, Mix: 0.5}).AddTo(p).In(pl),
		),
	))

	//err := p.ToSoundFile("7x7.wav", writer.WAV, 60*6, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
