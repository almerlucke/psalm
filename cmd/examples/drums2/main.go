package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/sndfile"
)

func newTimer(mult float64, s ...any) *timer.Timer {
	return timer.WithSteps(
		steps.Multiply(steps.QuickSteps(s...), mult),
	)
}

func drumTemplate(amp any, speed any, sound any) *templ.Templ {
	return templ.NewTrigger(amp, template.Template{
		"speed": speed,
		"sound": sound,
	})
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        120,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	soundBank := sndfile.SoundBank[float32]{}

	soundBank["kick"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/kick/CarbonKick_01_634.wav", 4)
	soundBank["hihat"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4)
	soundBank["synth1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/shots/Cymatics - Odyssey Synth One Shot 4 - C.wav", 4)
	soundBank["synth2"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/shots/Cymatics - Odyssey Synth One Shot 16 - F.wav", 4)

	sf1 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/chants/Johannes.Hymnus.wav", 4)
	sf2 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/chants/Crux_fidelis_(Graduale_1974).wav", 4)
	sf3 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-cinematic-angelical-choir-transition-663.wav", 4)

	drms := drums.NewDrums(soundBank, 16, 2).AddTo(p).SetID("drums")

	drms.ValueIn(drumTemplate(0.5,
		bucket.NewLoop(0.75, 0.6, 0.85, 0.7), "kick").ValueIn(newTimer(
		0.25,
		// Bar 1
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0,
		// Bar 2
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0,
		// Bar 3
		1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0,
		// Bar 4 (turnaround variation)
		1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 0,
	).AddTo(p)))

	drms.ValueIn(drumTemplate(0.5,
		bucket.NewLoop(2.0, 1.0, 0.5, 0.75, 1.5), bucket.NewLoop("synth1", "synth2")).ValueIn(newTimer(
		0.25,
		// Bar 1
		0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		// Bar 2
		0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		// Bar 3
		0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0,
		// Bar 4 (with snare roll / pickup into loop)
		0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 1,
	).AddTo(p)))

	drms.ValueIn(drumTemplate(0.375,
		bucket.NewLoop(2.0, 1.0, 1.5, 1.75, 4.5, 3.0), "hihat").ValueIn(newTimer(
		0.5,
		// Bar 1
		1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, &steps.Step{
			BurstChance: 0.6,
			NumBurst:    3,
		}, 0,
		// Bar 2 (trap roll accents)
		1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 0, 1, &steps.Step{
			BurstChance: 0.7,
			NumBurst:    4,
		}, 0,
		// Bar 3 (syncopated 16th groove)
		1, 0, 0, 1, 1, 0, 1, 0, 1, 0, 0, 1, 1, 0, &steps.Step{
			BurstChance: 0.5,
			NumBurst:    6,
		}, 0,
		// Bar 4 (high-energy fill before turnaround)
		1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 1, 0, &steps.Step{
			BurstChance: 0.8,
			NumBurst:    4,
		}, 0, 1, 0,
	).AddTo(p)))

	pl1 := player.New(player.Config{
		SndFile:          sf1,
		StartOffset:      0.1,
		EndOffset:        0.2,
		UseZeroCrossings: true,
		Loop:             false,
		Speed:            1.0,
		Amp:              0.25,
	}).AddTo(p).ValueIn(templ.New(template.Template{
		"startOffset": bucket.NewLoop(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8),
		"offsetRange": bucket.NewLoop(0.05, 0.1, 0.2, 0.3, 0.2),
		"bang":        true,
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 8)).AddTo(p)))

	pl2 := player.New(player.Config{
		SndFile:          sf2,
		StartOffset:      0.1,
		EndOffset:        0.2,
		UseZeroCrossings: true,
		Loop:             false,
		Speed:            1.0,
		Amp:              0.25,
	}).AddTo(p).ValueIn(templ.New(template.Template{
		"startOffset": bucket.NewLoop(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8),
		"offsetRange": bucket.NewLoop(0.05, 0.1, 0.2, 0.3),
		"bang":        true,
	}).ValueIn(newTimer(
		2.0, 1, 1, 0, 1, 1, 0, 0, 0,
	).AddTo(p)))

	pl3 := player.New(player.Config{
		SndFile:          sf3,
		StartOffset:      0.1,
		EndOffset:        0.2,
		UseZeroCrossings: true,
		Loop:             false,
		Speed:            0.25,
		Amp:              0.375,
	}).AddTo(p).ValueIn(templ.New(template.Template{
		"startOffset": bucket.NewLoop(0.0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8),
		"offsetRange": bucket.NewLoop(0.05, 0.1, 0.1, 0.1, 0.1),
		"speed":       bucket.NewLoop(0.25, 0.375, 0.5, 0.25, 0.25, 0.25),
		"bang":        true,
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 6)).AddTo(p)))

	// the mix: every part on its own strip, all through the pingpong delay into the reverb
	mix := mixer.New(p)
	mix.Group("fx", mixer.StripConfig{},
		pingpong.New(pingpong.Config{Time: 1.25, Feedback: 0.075, Mix: 0.125}).AddTo(p),
		freeverb.New(0.8, 0.7, 0.9, 0.075, 0.8).AddTo(p),
	)
	mix.Channel("drums", drms, mixer.StripConfig{Group: "fx"})
	// the voices start muted, their sections unmute them
	mix.Channel("hymnus", pl1, mixer.StripConfig{Group: "fx", Mute: true})
	mix.Channel("crux", pl2, mixer.StripConfig{Group: "fx", Mute: true})
	mix.Channel("choir", pl3, mixer.StripConfig{Group: "fx", Mute: true})
	p.Out(mix.Master())

	// the arrangement: in blocks of 8 bars, the parts play (unmuted) in their sections. The parts
	// keep running while muted, so they go on where they would have been
	form := arrange.NewForm(
		arrange.Section{Name: "Drums", Bars: 8},
		arrange.Section{Name: "Choir", Bars: 8},
		arrange.Section{Name: "Hymnus", Bars: 8},
		arrange.Section{Name: "Crux", Bars: 24},
		arrange.Section{Name: "Break", Bars: 24},
		arrange.Section{Name: "Return", Bars: 16},
		arrange.Section{Name: "Swap", Bars: 16},
		arrange.Section{Name: "Full", Bars: 16},
		arrange.Section{Name: "Fade", Bars: 8},
		arrange.Section{Name: "Crux alone", Bars: 16},
		arrange.Section{Name: "End", Bars: 8},
	)
	form.Loop = false // after the end everything stays muted

	arrange.NewArrangement(form).
		Play(mix.Strip("drums"), "Drums", "Choir", "Hymnus", "Crux", "Return", "Swap", "Full").
		Play(mix.Strip("choir"), "Choir", "Hymnus", "Crux", "Break", "Return", "Swap", "Full").
		Play(mix.Strip("hymnus"), "Hymnus", "Crux", "Swap", "Full", "Fade").
		Play(mix.Strip("crux"), "Crux", "Break", "Return", "Full", "Fade", "Crux alone").
		PrintSections(true).
		AddTo(p)

	//err := p.ToSoundFile("crux_fidelis.wav", writer.WAV, form.Beats()*float64(psalm.PPQ())*psalm.TickDuration(), 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
