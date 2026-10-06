package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/sndfile"
)

func main() {
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	soundBank := sndfile.SoundBank[float32]{}

	soundBank["perc1_1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 1.wav", 4)
	soundBank["perc2_1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 2.wav", 4)
	soundBank["perc3_1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 3.wav", 4)
	soundBank["perc4_1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 4.wav", 4)
	soundBank["perc5_1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 5.wav", 4)
	soundBank["perc6_1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 6.wav", 4)
	soundBank["perc7_1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 7.wav", 4)

	soundBank["harp1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/harp/E3_mf1.wav", 4)
	soundBank["harp2"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/harp/B1_f1.wav", 4)
	soundBank["harp3"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/harp/B3_mf1.wav", 4)
	soundBank["harp4"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/harp/C3_f2.wav", 4)
	soundBank["harp5"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/harp/D2_f1.wav", 4)
	soundBank["harp6"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/harp/F2_f1.wav", 4)
	soundBank["harp7"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/harp/G1_f1.wav", 4)

	soundBank["perc1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/world_percussion/Maracas/01_01.wav", 4)
	soundBank["perc2"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/world_percussion/LowConga/v3_01_01.wav", 4)
	soundBank["perc3"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/world_percussion/LowConga/v3_01_02.wav", 4)
	soundBank["perc4"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/world_percussion/LowConga/v4_01_01.wav", 4)
	soundBank["perc5"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/world_percussion/Castanets/01.wav", 4)
	soundBank["perc6"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/world_percussion/Darbuka/doom_01_01.wav", 4)
	soundBank["perc7"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/world_percussion/Darbuka/doom_01_05.wav", 4)

	soundBank["hihat1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 1.wav", 4)
	soundBank["hihat2"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 2.wav", 4)
	soundBank["hihat3"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 3.wav", 4)
	soundBank["hihat4"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4)
	soundBank["hihat5"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 5.wav", 4)
	soundBank["hihat6"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 6.wav", 4)
	soundBank["hihat7"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 7.wav", 4)

	soundBank["ride1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Ride - Flanged.wav", 4)
	soundBank["ride2"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Ride - Mysterious.wav", 4)
	soundBank["ride3"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Rimshot - Basic.wav", 4)
	soundBank["ride4"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Rimshot - Bunker.wav", 4)
	soundBank["ride5"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Shaker - Banana.wav", 4)
	soundBank["ride6"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Shaker - Crisp.wav", 4)
	soundBank["ride7"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Shaker - Drew.wav", 4)

	soundBank["clap1"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Humble Divine Clap.wav", 4)
	soundBank["clap2"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Humble Infatuation Clap.wav", 4)
	soundBank["clap3"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Humble Sit Down Clap.wav", 4)
	soundBank["clap4"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Humble Stars Clap.wav", 4)
	soundBank["clap5"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Orchid Clap - Basic.wav", 4)
	soundBank["clap6"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Orchid Clap - Flam.wav", 4)
	soundBank["clap7"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Orchid Clap - Pop.wav", 4)

	soundBank["angels"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-cinematic-angelical-choir-transition-663.wav", 4)

	drms := drums.NewDrums(soundBank, 32, 2).AddTo(p).SetID("drums")

	drms.ValueIn(templ.NewTrigger(0.35, template.Template{
		"speed": bucket.New(1.0, 2.0, 0.2, 0.75, 0.4, 3.0, 2.5).SetMode(bucket.Indexed).Loop(true),
		"sound": bucket.New("perc1_1", "perc2_1", "perc3_1", "perc4_1", "perc5_1", "perc6_1", "perc7_1").SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 2)).AddTo(p)))

	drms.ValueIn(templ.NewTrigger(0.15, template.Template{
		"speed": 1.0,
		"sound": bucket.New("harp1", "harp2", "harp3", "harp4", "harp5", "harp6", "harp7").SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 7)).AddTo(p)))

	drms.ValueIn(templ.NewTrigger(0.45, template.Template{
		"speed": bucket.New(1.0, 2.0, 0.25, 0.75, 0.5, 1.25, 1.5).SetMode(bucket.Indexed).Loop(true),
		"sound": bucket.New("perc1", "perc2", "perc3", "perc4", "perc5", "perc6", "perc7").SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 4)).AddTo(p)))

	drms.ValueIn(templ.NewTrigger(0.5, template.Template{
		"speed": bucket.New(2.0, 3.0, 1.0, 1.25, 1.5, 2.5, 0.75).SetMode(bucket.Indexed).Loop(true),
		"sound": bucket.New("hihat1", "hihat2", "hihat3", "hihat4", "hihat5", "hihat6", "hihat7").SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))

	drms.ValueIn(templ.NewTrigger(0.75, template.Template{
		"speed": bucket.New(0.6, 1.0, 1.2, 0.5, 1.5, 0.45, 0.75).SetMode(bucket.Indexed).Loop(true),
		"sound": bucket.New("ride1", "ride2", "ride3", "ride4", "ride5", "ride6", "ride7").SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 5)).AddTo(p)))

	drms.ValueIn(templ.NewTrigger(0.4, template.Template{
		"speed": bucket.New(0.4, 1.2, 1.0, 1.25, 1.5, 1.75, 0.75).SetMode(bucket.Indexed).Loop(true),
		"sound": bucket.New("clap1", "clap2", "clap3", "clap4", "clap5", "clap6", "clap7").SetMode(bucket.Indexed).Loop(true),
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 6)).AddTo(p)))

	drms.ValueIn(templ.NewTrigger(0.10, template.Template{
		"speed": bucket.New(1.0, 0.75, 1.25).SetMode(bucket.Indexed).Loop(true),
		"sound": "angels",
	}).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 30)).AddTo(p)))

	p.Out(freeverb.New(0.7, 0.8, 0.9, 0.044, 0.62).AddTo(p).In(
		pingpong.New(pingpong.Config{Time: 1.75, Feedback: 0.1, Mix: 0.09}).AddTo(p).In(drms),
	))

	//err := p.ToSoundFile("7x7.wav", writer.WAV, 60*6, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
