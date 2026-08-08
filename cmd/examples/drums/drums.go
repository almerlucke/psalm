package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/internal/timing"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/sndfile"
)

func main() {
	soundBank := sndfile.SoundBank{}

	soundBank["perc1_1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 1.wav", 4)
	soundBank["perc2_1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 2.wav", 4)
	soundBank["perc3_1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 3.wav", 4)
	soundBank["perc4_1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 4.wav", 4)
	soundBank["perc5_1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 5.wav", 4)
	soundBank["perc6_1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 6.wav", 4)
	soundBank["perc7_1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 7.wav", 4)

	soundBank["harp1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/harp/E3_mf1.wav", 4)
	soundBank["harp2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/harp/B1_f1.wav", 4)
	soundBank["harp3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/harp/B3_mf1.wav", 4)
	soundBank["harp4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/harp/C3_f2.wav", 4)
	soundBank["harp5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/harp/D2_f1.wav", 4)
	soundBank["harp6"], _ = sndfile.NewMipMapSoundFile("assets/sounds/harp/F2_f1.wav", 4)
	soundBank["harp7"], _ = sndfile.NewMipMapSoundFile("assets/sounds/harp/G1_f1.wav", 4)

	soundBank["glass1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/world_percussion/Maracas/01_01.wav", 4)
	soundBank["glass2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/world_percussion/LowConga/v3_01_01.wav", 4)
	soundBank["glass3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/world_percussion/LowConga/v3_01_02.wav", 4)
	soundBank["glass4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/world_percussion/LowConga/v4_01_01.wav", 4)
	soundBank["glass5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/world_percussion/Castanets/01.wav", 4)
	soundBank["glass6"], _ = sndfile.NewMipMapSoundFile("assets/sounds/world_percussion/Darbuka/doom_01_01.wav", 4)
	soundBank["glass7"], _ = sndfile.NewMipMapSoundFile("assets/sounds/world_percussion/Darbuka/doom_01_05.wav", 4)

	soundBank["kick1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/long-boomy-real-acoustic-kick-one-shot_120bpm_C_major.wav", 4)
	soundBank["kick2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/Cymatics - Orchid Kick - Dancehall (A#).wav", 4)
	soundBank["kick3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/Cymatics - Orchid Kick - Layered (F#).wav", 4)
	soundBank["kick4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/Cymatics - Orchid Kick - Tight (G).wav", 4)
	soundBank["kick5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/CarbonKick_01_634.wav", 4)
	soundBank["kick6"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/long-boomy-real-acoustic-kick-one-shot_120bpm_C_major.wav", 4)
	soundBank["kick7"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/kick/WaterKick_01_693.wav", 4)

	soundBank["hihat1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 1.wav", 4)
	soundBank["hihat2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 2.wav", 4)
	soundBank["hihat3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 3.wav", 4)
	soundBank["hihat4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4)
	soundBank["hihat5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 5.wav", 4)
	soundBank["hihat6"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 6.wav", 4)
	soundBank["hihat7"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 7.wav", 4)

	soundBank["ride1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Ride - Flanged.wav", 4)
	soundBank["ride2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Ride - Mysterious.wav", 4)
	soundBank["ride3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Rimshot - Basic.wav", 4)
	soundBank["ride4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Rimshot - Bunker.wav", 4)
	soundBank["ride5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Shaker - Banana.wav", 4)
	soundBank["ride6"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Shaker - Crisp.wav", 4)
	soundBank["ride7"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/hihat/Cymatics - Orchid Shaker - Drew.wav", 4)

	soundBank["clap1"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/clap/Cymatics - Humble Divine Clap.wav", 4)
	soundBank["clap2"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/clap/Cymatics - Humble Infatuation Clap.wav", 4)
	soundBank["clap3"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/clap/Cymatics - Humble Sit Down Clap.wav", 4)
	soundBank["clap4"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/clap/Cymatics - Humble Stars Clap.wav", 4)
	soundBank["clap5"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/clap/Cymatics - Orchid Clap - Basic.wav", 4)
	soundBank["clap6"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/clap/Cymatics - Orchid Clap - Flam.wav", 4)
	soundBank["clap7"], _ = sndfile.NewMipMapSoundFile("assets/sounds/drums/clap/Cymatics - Orchid Clap - Pop.wav", 4)

	soundBank["angels"], _ = sndfile.NewMipMapSoundFile("assets/sounds/various/mixkit-cinematic-angelical-choir-transition-663.wav", 4)

	p := psalm.New()

	drms := drums.NewDrums(soundBank, 32).AddTo(p).SetID("drums")

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.35,
		"message": template.Template{
			"speed": bucket.New(1.0, 2.0, 0.2, 0.75, 0.4, 3.0, 2.5).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("perc1_1", "perc2_1", "perc3_1", "perc4_1", "perc5_1", "perc6_1", "perc7_1").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*2, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.15,
		"message": template.Template{
			"speed": 1,
			"sound": bucket.New("harp1", "harp2", "harp3", "harp4", "harp5", "harp6", "harp7").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*7, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.45,
		"message": template.Template{
			"speed": bucket.New(1.0, 2.0, 0.25, 0.75, 0.5, 1.25, 1.5).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("glass1", "glass2", "glass3", "glass4", "glass5", "glass6", "glass7").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*4, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 1.0,
		"message": template.Template{
			"speed": bucket.New(1.0, 2.0, 2.2, 0.75, 0.4, 1.2, 0.85).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("kick1", "kick2", "kick3", "kick4", "kick5", "kick6", "kick7").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*3, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.5,
		"message": template.Template{
			"speed": bucket.New(2.0, 3.0, 1.0, 1.25, 1.5, 2.5, 0.75).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("hihat1", "hihat2", "hihat3", "hihat4", "hihat5", "hihat6", "hihat7").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ(), true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.75,
		"message": template.Template{
			"speed": bucket.New(0.6, 1.0, 1.2, 0.5, 1.5, 0.45, 0.75).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("ride1", "ride2", "ride3", "ride4", "ride5", "ride6", "ride7").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*5, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.4,
		"message": template.Template{
			"speed": bucket.New(0.4, 1.2, 1.0, 1.25, 1.5, 1.75, 0.75).SetMode(bucket.Indexed).Loop(true),
			"sound": bucket.New("clap1", "clap2", "clap3", "clap4", "clap5", "clap6", "clap7").SetMode(bucket.Indexed).Loop(true),
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*6, true).AddTo(p)))

	drms.ValueIn(templ.New(template.Template{
		"command":   "trigger",
		"duration":  0.0,
		"amplitude": 0.10,
		"message": template.Template{
			"speed": bucket.New(1.0, 0.75, 1.25).SetMode(bucket.Indexed).Loop(true),
			"sound": "angels",
			"tick":  template.NewParameter("tick", nil),
		},
	}).ValueIn(timer.New(p.PPQ()*30, true).AddTo(p)))

	bpmMs := timing.BPMToMilli(p.BPM())

	pp := pingpong.New(bpmMs*4.0, bpmMs*1.75, 0.1, 0.125).AddTo(p).In(drms)
	rv := freeverb.New()
	rv.SetRoomSize(0.7)
	rv.SetDry(0.7)
	rv.SetWet(0.05)
	rv.SetDamp(0.8)
	rv.SetWidth(0.9)
	rv.AddTo(p).In(pp)

	p.Out(rv)

	_ = p.Listen()
}
