package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/internal/timing"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/sndfile"
)

func main() {
	sf := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4)

	p := psalm.New()

	pl := player.New(player.Config{
		SndFile:          sf,
		StartOffset:      0.1,
		EndOffset:        0.2,
		UseZeroCrossings: true,
		Loop:             true,
		Speed:            1.0,
		Amp:              0.6,
	}).AddTo(p).ValueIn(templ.New(template.Template{
		"startOffset": bucket.New(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8).SetMode(bucket.Indexed).Loop(true),
		"offsetRange": bucket.New(0.05, 0.1, 0.2, 0.3, 0.4).SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.New(psalm.PPQ()*8, true).AddTo(p)))

	bpmMs := timing.BPMToMilli(psalm.BPM())

	p.Out(freeverb.New(0.7, 0.8, 0.9, 0.05, 0.8).AddTo(p).In(
		pingpong.New(bpmMs*4.0, bpmMs*1.75, 0.1, 0.125).AddTo(p).In(
			flanger.New(0.05, 0.95, 0.85, 0.9, nil).AddTo(p).In(pl),
		),
	))

	//err := p.ToSoundFile("7x7.wav", writer.WAV, 60*6, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
