package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/internal/timing"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/triggers/stepper"
	"github.com/almerlucke/psalm/modules/controls/triggers/stepper/duration/swing"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/sndfile"
)

func newStepper(multiplier float64, steps ...any) *stepper.Stepper {
	return stepper.NewStepper(swing.New(sequence.NewLoop[*swing.Step](
		swing.Multiply(swing.QuickSteps(steps...), multiplier)...,
	)))
}

func drumTemplate(amp any, speed any, sound any) *templ.Templ {
	return templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": amp,
		"message": template.Template{
			"speed": speed,
			"sound": sound,
			"tick":  template.NewParameter("tick", nil),
		},
	})
}

func main() {
	soundBank := sndfile.SoundBank[float32]{}

	soundBank["kick"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/kick/81_Kick_05_255_SP.wav", 4)
	soundBank["hihat"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4)
	soundBank["clap"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/snare/Cymatics - Odyssey Trap Snare 4 - F.wav", 4)

	sf1 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/Cymatics - Blurry Vocal - 80 BPM F Min.wav", 4)
	sf2 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/elisa.wav", 4)

	p := psalm.New()

	psalm.PushConfig(psalm.Config{
		BPM:        160,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	drms := drums.NewDrums(soundBank, 8).AddTo(p).SetID("drums")

	drms.ValueIn(drumTemplate(0.8,
		bucket.NewLoop(1.0, 1.4, 1.3, 0.75, 0.4, 1.2, 0.85), "kick").ValueIn(newStepper(
		0.5,
		1, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
	).AddTo(p)))

	drms.ValueIn(drumTemplate(0.6,
		bucket.NewLoop(1.0, 1.4, 1.3, 1.75, 0.85, 1.8, 0.95), "clap").ValueIn(newStepper(
		0.5,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0,
	).AddTo(p)))

	drms.ValueIn(drumTemplate(0.6,
		bucket.NewLoop(2.0, 2.4, 2.3, 1.75, 1.4, 1.7, 1.85), "hihat").ValueIn(newStepper(
		0.5,
		0, 1, 0, 1, 0, 1, &swing.Step{
			BurstChance: 0.3,
			NumBurst:    3,
		}, 1,
		1, 1, 0, 1, 1, 0, &swing.Step{
			BurstChance: 0.3,
			NumBurst:    6,
		}, 1,
	).AddTo(p)))

	pl1 := player.New(player.Config{
		SndFile:          sf1,
		StartOffset:      0.1,
		EndOffset:        0.2,
		UseZeroCrossings: true,
		Loop:             true,
		Speed:            1.5,
		Amp:              0.4,
	}).AddTo(p).ValueIn(templ.New(template.Template{
		"startOffset": bucket.NewLoop(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8),
		"offsetRange": bucket.NewLoop(0.05, 0.1, 0.2, 0.3, 0.4),
	}).ValueIn(timer.New(psalm.PPQ()*8, true).AddTo(p)))

	pl2 := player.New(player.Config{
		SndFile:          sf2,
		StartOffset:      0.1,
		EndOffset:        0.2,
		UseZeroCrossings: true,
		Loop:             false,
		Speed:            1.0,
		Amp:              0.2,
	}).AddTo(p).ValueIn(templ.New(template.Template{
		"startOffset": bucket.NewLoop(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8),
		"offsetRange": bucket.NewLoop(0.05, 0.1, 0.1, 0.1, 0.1),
		"speed":       bucket.NewLoop(1.0, 1.25, 1.15, 1.25, 1.0, 0.75, 0.85),
		"bang":        true,
	}).ValueIn(newStepper(
		2.0, 1, 1, 0, 1, 1, 0, 0, 0,
	).AddTo(p)))

	bpmMs := timing.BPMToMilli(psalm.BPM())

	pp := pingpong.New(bpmMs*4.0, bpmMs*1.75, 0.0175, 0.125).AddTo(p).In(drms, pl1, 0, 0, pl2, 0, 0)
	rv := freeverb.New()
	rv.SetRoomSize(0.7)
	rv.SetDry(0.7)
	rv.SetWet(0.1)
	rv.SetDamp(0.8)
	rv.SetWidth(0.9)
	rv.AddTo(p).In(pp)

	p.Out(rv)

	//err := p.ToSoundFile("7x7.wav", writer.WAV, 60*6, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
