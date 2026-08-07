package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/internal/timing"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/sndfile"
)

func main() {
	soundBank := sndfile.SoundBank{}

	soundBank["sh1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/glitch/CH_Med_22_259_SP.wav", 4)
	soundBank["sh2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/glitch/98_Hit_319_SP_02.wav", 4)
	soundBank["sh3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/glitch/DigiDisintegrate_SP_44_01.wav", 4)
	soundBank["sh4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/glitch/Hat_Glitch_03_321_SP.wav", 4)
	soundBank["sh5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/glitch/Rubber_Glitch_01_321_SP.wav", 4)

	soundBank["shot1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/shots/Cymatics - Odyssey Synth One Shot 1 - C.wav", 4)
	soundBank["shot2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/shots/Cymatics - Odyssey Synth One Shot 6 - C#.wav", 4)
	soundBank["shot3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/shots/Cymatics - Odyssey Synth One Shot 9 - E.wav", 4)
	soundBank["shot4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/shots/Cymatics - Odyssey Synth One Shot 11 - E.wav", 4)
	soundBank["shot5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/shots/Cymatics - Odyssey Synth One Shot 17 - F#.wav", 4)

	soundBank["perc1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 1.wav", 4)
	soundBank["perc2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 2.wav", 4)
	soundBank["perc3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 3.wav", 4)
	soundBank["perc4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 4.wav", 4)
	soundBank["perc5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 5.wav", 4)

	soundBank["kick1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/long-boomy-real-acoustic-kick-one-shot_120bpm_C_major.wav", 4)
	soundBank["kick2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/Cymatics - Orchid Kick - Dancehall (A#).wav", 4)
	soundBank["kick3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/Cymatics - Orchid Kick - Layered (F#).wav", 4)
	soundBank["kick4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/Cymatics - Orchid Kick - Tight (G).wav", 4)

	soundBank["hihat1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 1.wav", 4)
	soundBank["hihat2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Odyssey Closed Hihat 1.wav", 4)
	soundBank["hihat3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Odyssey Closed Hihat 2.wav", 4)
	soundBank["hihat4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Hihat - Closed 2.wav", 4)
	soundBank["hihat5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Hihat - Closed 2.wav", 4)

	p := psalm.New()

	drms := drums.NewDrums(soundBank, 32).AddTo(p).SetID("drums")

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.35,
		"tick":      template.NewParameter("tick", nil),
		"message": template.Template{
			"speed": bucket.New(1.0, 2.0, 0.2, 0.75, 0.4, 3.0, 2.5).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("sh1", "sh2", "sh3", "sh4", "sh5").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*3, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.75,
		"tick":      template.NewParameter("tick", nil),
		"message": template.Template{
			"speed": bucket.New(1.0, 2.0, 0.8, 0.75, 1.4, 3.0, 2.5).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("shot1", "shot2", "shot3", "shot4", "shot5").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ(), true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.65,
		"tick":      template.NewParameter("tick", nil),
		"message": template.Template{
			"speed": bucket.New(1.0, 2.0, 0.2, 0.75, 0.4, 3.0, 2.5).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("perc1", "perc2", "perc3", "perc4", "perc5").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()/2, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 1.0,
		"message": template.Template{
			"speed": bucket.New(1.0, 2.0, 2.2, 0.75, 0.4).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("kick1", "kick2", "kick3", "kick4").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ(), true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.5,
		"message": template.Template{
			"speed": bucket.New(2.0, 3.0, 1.0, 1.25, 1.5, 2.5, 0.75).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("hihat1", "hihat2", "hihat3", "hihat4", "hihat5").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()/3, true).AddTo(p)))

	pp := pingpong.New(2000.0, timing.BPMToMilli(p.BPM())*1.5, 0.1, 0.075).AddTo(p).In(drms)

	p.Out(pp)

	_ = p.Listen()
}
