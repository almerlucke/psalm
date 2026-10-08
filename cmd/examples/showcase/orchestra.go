package main

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
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

// orchestra is everything that sounds, on a mixer: a hall and a shimmering sky as effect buses,
// the strings and the choir as groups with their own inserts
type orchestra struct {
	mix          *mixer.Mixer
	stringFilter *filters.Filter        // the dawn and dusk filter on the strings
	gran         *granulator.Granulator // the granulator on the harp
	harpStrip    *mixer.Strip

	violins, celli, pizz, harp, flute, horns, hornSolo psalm.Modular
	soprano, choir, ooh, celesta, bellPad, carrier     psalm.Modular
	droplets, aurora, grand                            psalm.Modular
}

type sends = map[string]float64

func newOrchestra(p *psalm.Psalm) *orchestra {
	// --- the mixer: a hall and a shimmering sky as effect buses, the strings and the choir as
	// groups with their own inserts ---

	mix := mixer.New(p)

	hcfg := fdnreverb.DefaultConfig()
	hcfg.Decay = 3.8
	hcfg.Size = 0.8
	hcfg.Damping = 0.55
	hcfg.PreDelay = 30
	hcfg.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hcfg).AddTo(p))

	scfg := shimmer.DefaultConfig()
	scfg.Reverb.Decay = 7
	scfg.Shimmer = 0.45
	scfg.Mix = 1
	mix.Bus("sky", mixer.StripConfig{}, shimmer.New(scfg).AddTo(p))

	// the strings play through the dawn and dusk filter
	stringFilter := filters.NewLadder(ladder.LowPass12, 250, 0.1, 1.0)
	stringFilter.AddTo(p)
	mix.Group("strings", mixer.StripConfig{Sends: sends{"hall": 0.4}}, stringFilter)

	// the choir sings in a church: a highpass cleans the lows, a recorded church response (large
	// partitions keep it cheap, their latency is a natural pre delay)
	choirClean := filters.NewSVF(svf.HighPass, 120, 0.1)
	choirClean.AddTo(p)
	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	mix.Group("choir", mixer.StripConfig{Volume: -2.8}, choirClean,
		convolver.New(church, convolver.Config{PartitionSize: 4096, Mix: 0.37, Gain: 2.27}).AddTo(p))

	// --- strings ---

	violinsCfg := wavetable.StringsPreset(gen.Brightness(64))
	violinsCfg.Position = 0.8
	violinsCfg.Unison = 3 // the chorus widens the section
	violins := wavetable.NewPoly(violinsCfg, 16).AddTo(p)
	violins.ValueIn(lfo.New(true, 0.05, 0.7, 0.9, lfo.Sine).AddTo(p), 0, 1) // the bowing breathes
	violinsChorus := chorus.New(chorus.Config{Rate: 0.3, Depth: 6, Delay: 23, Voices: 3, Spread: 1, Feedback: 0.1, Tone: 9000, Mix: 0.35}).AddTo(p).In(violins)
	mix.Channel("violins", violinsChorus, mixer.StripConfig{Volume: -0.4, Pan: -0.35, Group: "strings"})

	celli := subtractive.NewPoly(subtractive.StringsPreset(), 8).AddTo(p)
	mix.Channel("celli", celli, mixer.StripConfig{Pan: 0.35, Group: "strings"})

	pizz := fm.NewPoly(fm.UprightBassPreset(), 2).AddTo(p)
	mix.Channel("pizzicato", pizz, mixer.StripConfig{Volume: -3, Pan: 0.45, Sends: sends{"hall": 0.25}})

	// --- harp: freeverb, then the granulator ---

	harp := fm.NewPoly(fm.HarpPreset(), 12).AddTo(p)
	harpRoom := freeverb.New(0.5, 0.4, 1.0, 0.25, 0.9).AddTo(p).In(harp)
	gcfg := granulator.DefaultConfig()
	gcfg.Mix = 0
	gcfg.Delay = 300
	gcfg.Spray = 250
	gcfg.GrainSize = 220
	gcfg.Density = 30
	gcfg.Octave = 0.25
	gran := granulator.New(gcfg)
	gran.AddTo(p)
	gran.In(harpRoom)
	harpStrip := mix.Channel("harp", gran, mixer.StripConfig{Volume: -3, Pan: 0.55, Sends: sends{"hall": 0.5}})

	// --- winds and brass ---

	flute := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	piccolo := pitchshift.New(12, 60, 0.11).AddTo(p).In(flute) // an octave shadow above
	mix.Channel("flute", piccolo, mixer.StripConfig{Volume: -4.3, Pan: -0.1, Sends: sends{"hall": 0.4, "sky": 0.1}})

	horns := dx.NewPoly(dx.SoftBrass(), 12, psalm.Stereo).AddTo(p)
	mix.Channel("horns", horns, mixer.StripConfig{Volume: -3, Pan: -0.2, Sends: sends{"hall": 0.45}})

	hornSolo := fm.NewPoly(fm.FrenchHornPreset(), 2).AddTo(p)
	mix.Channel("horn solo", hornSolo, mixer.StripConfig{Volume: -3, Pan: 0.25, Sends: sends{"hall": 0.4}})

	// --- choir: grains on a choir recording, a pulsar choir and the soprano ---

	choirRec := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	choir := grains.NewPoly(grains.ChoirPad(choirRec, 60), 12).AddTo(p)
	mix.Channel("grains choir", choir, mixer.StripConfig{Group: "choir"})
	ooh := pulsar.NewPoly(pulsar.ChoirPreset(), 12).AddTo(p)
	mix.Channel("ooh", ooh, mixer.StripConfig{Group: "choir"})
	soprano := pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
	mix.Channel("soprano", soprano, mixer.StripConfig{Group: "choir", Sends: sends{"sky": 0.15}})

	// --- interlude colours: celesta, bell pad, vocoder, droplets ---

	celesta := fm.NewPoly(fm.CelestaPreset(), 8).AddTo(p)
	celestaEcho := pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.35, Mix: 0.26, HighCut: 5000, LowCut: 200}).AddTo(p).In(celesta)
	mix.Channel("celesta", celestaEcho, mixer.StripConfig{Volume: -7.4, Pan: 0.3, Sends: sends{"sky": 0.8}})

	speech := player.New(player.Config{
		SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4),
		Loop:    true,
		Speed:   1.0,
		Amp:     0.8,
	}).AddTo(p)
	carrierCfg := wavetable.PadPreset(gen.Brightness(16))
	carrierCfg.Position = 1
	carrierCfg.PositionLFODepth = 0
	carrierCfg.Filter = wavetable.FilterOff
	carrierCfg.Unison = 3
	carrierCfg.AmpEnv.AttackDuration = 200
	carrier := wavetable.NewPoly(carrierCfg, 12).AddTo(p)
	voc := vocoder.New(vocoder.DefaultConfig())
	voc.AddTo(p)
	voc.In(speech, 0, 0, carrier, 0, 1)
	mix.Channel("voices", voc, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.4}})

	bellPad := dx.NewPoly(dx.BellPad(), 12, psalm.Stereo).AddTo(p)
	bellFlange := flanger.New(flanger.Config{Rate: 0.08, Manual: 1, Depth: 9.5, Feedback: 0.4, Stereo: 0.5, Shape: flanger.Triangle, Mix: 0.4}).AddTo(p).In(bellPad)
	// the vocoder sounds an FFT frame late: the mixer delays the other strips to line them up
	mix.Channel("bell pad", bellFlange, mixer.StripConfig{Volume: 0.3, Sends: sends{"hall": 0.4}})

	droplets := pulsar.NewPoly(pulsar.RaindropsPreset(), 2).AddTo(p)
	mix.Channel("droplets", droplets, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.4, "sky": 0.2}})

	// --- aurora: a drone on a table of random spectra, at dawn and dusk ---

	aurora := wavetable.NewPoly(wavetable.DronePreset(gen.Random(64, 6, 32, 11)), 4).AddTo(p)
	mix.Channel("aurora", aurora, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.4}})

	// --- piano: the grand, its sustain pedal changed at every bar line ---

	grand := piano.NewPoly(piano.GrandPreset(), 24).AddTo(p)
	mix.Channel("piano", grand, mixer.StripConfig{Volume: -3, Pan: -0.05, Sends: sends{"hall": 0.35}})

	return &orchestra{
		mix: mix, stringFilter: stringFilter, gran: gran, harpStrip: harpStrip,
		violins:  violins,
		celli:    celli,
		pizz:     pizz,
		harp:     harp,
		flute:    flute,
		horns:    horns,
		hornSolo: hornSolo,
		soprano:  soprano,
		choir:    choir,
		ooh:      ooh,
		celesta:  celesta,
		bellPad:  bellPad,
		carrier:  carrier,
		droplets: droplets,
		aurora:   aurora,
		grand:    grand,
	}
}
