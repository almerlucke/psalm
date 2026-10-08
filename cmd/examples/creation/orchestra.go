package main

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/sndfile"
)

// orchestra is everything that sounds, on a mixer: a hall and the heavens as effect buses,
// creation playing through the darkness that lifts on the first day, the choir and the organ in a
// church
type orchestra struct {
	mix      *mixer.Mixer
	darkness *filters.Filter

	// recordings of creation itself, faded in on their days
	waves, seaBirds, gulls, animals, children *mixer.Strip
	word, laughter                            *player.Player // started on their bars

	void, spirit, wordPad               psalm.Modular
	violins, celli, pizz, harp, timpani psalm.Modular
	flute, oboe, clarinet, horns, horn  psalm.Modular
	celesta, bellPad, bells, deep       psalm.Modular
	grand, soprano, choir, organ        psalm.Modular
}

type sends = map[string]float64

func newOrchestra(p *psalm.Psalm) *orchestra {
	o := &orchestra{mix: mixer.New(p)}
	mix := o.mix

	hcfg := fdnreverb.DefaultConfig()
	hcfg.Decay = 4.2
	hcfg.Size = 0.85
	hcfg.Damping = 0.5
	hcfg.PreDelay = 30
	hcfg.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hcfg).AddTo(p))

	scfg := shimmer.DefaultConfig()
	scfg.Reverb.Decay = 8
	scfg.Shimmer = 0.5
	scfg.Mix = 1
	mix.Bus("heavens", mixer.StripConfig{}, shimmer.New(scfg).AddTo(p))

	o.darkness = filters.NewLadder(ladder.LowPass12, 300, 0.1, 1.0)
	o.darkness.AddTo(p)
	mix.Group("creation", mixer.StripConfig{}, o.darkness)

	choirClean := filters.NewSVF(svf.HighPass, 110, 0.1)
	choirClean.AddTo(p)
	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	mix.Group("church", mixer.StripConfig{Volume: -2}, choirClean,
		convolver.New(church, convolver.Config{PartitionSize: 4096, Mix: 0.4, Gain: 2.2}).AddTo(p))

	// a strip of creation (in the darkness), unless it says otherwise
	channel := func(name string, m psalm.Modular, cfg mixer.StripConfig) *mixer.Strip {
		if cfg.Group == "" {
			cfg.Group = "creation"
		}
		return mix.Channel(name, m, cfg)
	}
	// a recording, silent until it is faded in
	recording := func(name, file string, loop bool, pan, hall float64) (*player.Player, *mixer.Strip) {
		pl := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32](file, 4), Loop: loop, Speed: 1, Amp: 1})
		pl.AddTo(p)
		return pl, channel(name, pl, mixer.StripConfig{Volume: mixer.MinDB, Pan: pan, Sends: sends{"hall": hall}})
	}

	// --- the waters, the land and the living ---

	_, o.waves = recording("waves", "assets/sounds/various/boat_waves-6099.wav", true, 0, 0.15)
	_, o.seaBirds = recording("sea and birds", "assets/sounds/various/mixkit-sea-waves-with-birds-loop-1185.wav", true, 0.2, 0.15)
	_, o.gulls = recording("gulls", "assets/sounds/various/seagulls-by-the-sea-7042.wav", true, -0.3, 0.15)
	_, o.animals = recording("animals", "assets/sounds/various/animals.wav", true, 0, 0.15)
	// the children laugh once, from the start of the recording
	o.laughter, o.children = recording("children", "assets/sounds/various/mixkit-laughing-children-indoors-427.wav", false, 0.1, 0.25)

	// --- the void and the Spirit ---

	o.void = wavetable.NewPoly(wavetable.DronePreset(gen.Random(64, 6, 32, 7)), 4).AddTo(p)
	channel("void", o.void, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.4}})

	o.spirit = pulsar.NewPoly(pulsar.ChoirPreset(), 12).AddTo(p)
	channel("spirit", o.spirit, mixer.StripConfig{Group: "church", Sends: sends{"heavens": 0.2}})

	// the Word: John 1:1 spoken through a pad of open fifths, heard clearly in the darkness (not
	// through it)
	o.word = player.New(player.Config{
		SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4),
		Speed:   1.0,
		Amp:     0.8,
	})
	o.word.AddTo(p)
	wordCfg := wavetable.PadPreset(gen.Brightness(16))
	wordCfg.Position = 1
	wordCfg.PositionLFODepth = 0
	wordCfg.Filter = wavetable.FilterOff
	wordCfg.Unison = 3
	wordCfg.AmpEnv.AttackDuration = 200
	o.wordPad = wavetable.NewPoly(wordCfg, 8).AddTo(p)
	voc := vocoder.New(vocoder.DefaultConfig())
	voc.AddTo(p)
	voc.In(o.word, 0, 0, o.wordPad, 0, 1)
	mix.Channel("the word", voc, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.35, "heavens": 0.1}})

	// --- strings ---

	violinsCfg := wavetable.StringsPreset(gen.Brightness(64))
	violinsCfg.Position = 0.8
	o.violins = wavetable.NewPoly(violinsCfg, 16).AddTo(p)
	channel("violins", o.violins, mixer.StripConfig{Pan: -0.35, Sends: sends{"hall": 0.45}})

	o.celli = subtractive.NewPoly(subtractive.StringsPreset(), 8).AddTo(p)
	channel("celli", o.celli, mixer.StripConfig{Pan: 0.35, Sends: sends{"hall": 0.4}})

	o.pizz = fm.NewPoly(fm.PizzicatoPreset(), 4).AddTo(p)
	channel("pizzicato", o.pizz, mixer.StripConfig{Pan: 0.45, Sends: sends{"hall": 0.3}})

	o.harp = fm.NewPoly(fm.HarpPreset(), 12).AddTo(p)
	channel("harp", o.harp, mixer.StripConfig{Pan: 0.55, Sends: sends{"hall": 0.5}})

	o.timpani = fm.NewPoly(fm.TimpaniPreset(), 4).AddTo(p)
	channel("timpani", o.timpani, mixer.StripConfig{Pan: -0.1, Sends: sends{"hall": 0.45}})

	// --- winds and brass ---

	o.flute = fm.NewPoly(fm.FlutePreset(), 4).AddTo(p)
	piccolo := pitchshift.New(12, 60, 0.15).AddTo(p).In(o.flute) // on the fifth day, the birds an octave above
	channel("flute", piccolo, mixer.StripConfig{Pan: -0.15, Sends: sends{"hall": 0.4, "heavens": 0.1}})

	o.oboe = fm.NewPoly(fm.OboePreset(), 2).AddTo(p)
	channel("oboe", o.oboe, mixer.StripConfig{Pan: 0.2, Sends: sends{"hall": 0.4}})

	o.clarinet = fm.NewPoly(fm.ClarinetPreset(), 2).AddTo(p)
	channel("clarinet", o.clarinet, mixer.StripConfig{Pan: -0.4, Sends: sends{"hall": 0.4}})

	o.horns = dx.NewPoly(dx.SoftBrass(), 12, psalm.Stereo).AddTo(p)
	channel("horns", o.horns, mixer.StripConfig{Pan: -0.2, Sends: sends{"hall": 0.45}})

	o.horn = fm.NewPoly(fm.FrenchHornPreset(), 2).AddTo(p)
	channel("horn", o.horn, mixer.StripConfig{Pan: 0.25, Sends: sends{"hall": 0.4}})

	// --- the lights: celesta and bells, stars in the heavens ---

	o.celesta = fm.NewPoly(fm.CelestaPreset(), 8).AddTo(p)
	celestaEcho := pingpong.New(pingpong.Config{Time: 0.833, Feedback: 0.35, Mix: 0.25, HighCut: 6000, LowCut: 250}).AddTo(p).In(o.celesta)
	channel("celesta", celestaEcho, mixer.StripConfig{Pan: 0.3, Sends: sends{"heavens": 0.8}})

	o.bellPad = dx.NewPoly(dx.BellPad(), 12, psalm.Stereo).AddTo(p)
	channel("bell pad", o.bellPad, mixer.StripConfig{Sends: sends{"hall": 0.4, "heavens": 0.3}})

	// tuned bells: every partial a whole number times the note (the 1, 2 and 4 of the music box, the
	// 7 of its modulator only adding harmonics), ringing long and soft
	bellCfg := fm.MusicBoxPreset()
	bellCfg.Ops[0].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 6000.0, 1.0, 3000.0)
	bellCfg.Ops[1].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 3500.0, 1.0, 2000.0)
	bellCfg.Ops[2].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 1800.0, 1.0, 1000.0)
	bellCfg.Ops[3].Level = 0.1
	bellCfg.Ops[3].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 300.0, 1.0, 200.0)
	o.bells = fm.NewPoly(bellCfg, 4).AddTo(p)
	channel("bells", o.bells, mixer.StripConfig{Pan: 0.15, Sends: sends{"hall": 0.5, "heavens": 0.3}})

	// --- the living: the creatures of the sea, mankind ---

	o.deep = pulsar.NewPoly(pulsar.FormantPadPreset(), 4).AddTo(p)
	channel("the deep", o.deep, mixer.StripConfig{Sends: sends{"hall": 0.5}})

	o.grand = piano.NewPoly(piano.GrandPreset(), 24).AddTo(p)
	channel("piano", o.grand, mixer.StripConfig{Pan: -0.05, Sends: sends{"hall": 0.35}})

	o.soprano = pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
	channel("soprano", o.soprano, mixer.StripConfig{Group: "church", Sends: sends{"heavens": 0.15}})

	choirRec := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	o.choir = grains.NewPoly(grains.ChoirPad(choirRec, 60), 12).AddTo(p)
	channel("choir", o.choir, mixer.StripConfig{Group: "church"})

	o.organ = fm.NewPoly(fm.ChurchOrganPreset(), 12).AddTo(p)
	channel("organ", o.organ, mixer.StripConfig{Group: "church"})

	return o
}
