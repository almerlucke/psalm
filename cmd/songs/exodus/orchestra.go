package main

import (
	"fmt"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/rotary"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/tape"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/sndfile"
)

// orchestra is everything that sounds: the physically modelled instruments of the library and
// nothing else (the waveguide plucked strings, pipes and drums, the modal bells and bars, the piano
// and the organ), on a mixer with a hall and a church, and the voice that speaks the verses
type orchestra struct {
	mix *mixer.Mixer

	// the granulators on the glass harmonica and the shakuhachi, whose mix follows the story
	glassCloud, shakuCloud *granulator.Granulator
	// the voice that speaks the verses
	voice psalm.Modular

	// the low and solo winds, the plucked strings
	low, solo, section, oud, harp psalm.Modular
	// the winds
	bansuri, recorder, clarinet, shakuhachi psalm.Modular
	// the drums
	frame, taiko, bodhran, tom, tambourine psalm.Modular
	// the bells and bars
	tubular, bell, hand, anvil, vibe, bowedVibe, bowl, thunder, tamtam, gong, kalimba, glock, glass, marimba, tine psalm.Modular
	// the keys
	grand                                                 psalm.Modular
	organLow, organSoft, organPlenum, organRaw, organBass psalm.Modular
}

type sends = map[string]float64

func newOrchestra(p *psalm.Psalm) *orchestra {
	o := &orchestra{mix: mixer.New(p)}
	mix := o.mix

	hcfg := fdnreverb.DefaultConfig()
	hcfg.Decay = 3.6
	hcfg.Size = 0.85
	hcfg.Damping = 0.5
	hcfg.PreDelay = 28
	hcfg.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hcfg).AddTo(p))

	scfg := shimmer.DefaultConfig()
	scfg.Reverb.Decay = 7
	scfg.Shimmer = 0.55
	scfg.Mix = 1
	mix.Bus("heavens", mixer.StripConfig{}, shimmer.New(scfg).AddTo(p))

	// everything of the music plays into one group, which gives way a little while the voice speaks
	mix.Group("music", mixer.StripConfig{})
	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	mix.Group("church", mixer.StripConfig{Volume: -2, Group: "music"},
		convolver.New(church, convolver.Config{PartitionSize: 4096, Mix: 0.4, Gain: 2.2}).AddTo(p))

	wg := func(name string, voices int) psalm.Modular {
		f, ok := waveguide.Presets[name]
		if !ok {
			panic(fmt.Sprintf("no waveguide preset %q", name))
		}
		return waveguide.NewPoly(f(), voices).AddTo(p)
	}
	md := func(name string, voices int) psalm.Modular {
		f, ok := modal.Presets[name]
		if !ok {
			panic(fmt.Sprintf("no modal preset %q", name))
		}
		return modal.NewPoly(f(), voices).AddTo(p)
	}
	og := func(name string) psalm.Modular {
		f, ok := organ.Presets[name]
		if !ok {
			panic(fmt.Sprintf("no organ preset %q", name))
		}
		return organ.New(f()).AddTo(p)
	}
	ch := func(name string, source psalm.Modular, cfg mixer.StripConfig) *mixer.Strip {
		if cfg.Group == "" {
			cfg.Group = "music"
		}
		return mix.Channel(name, source, cfg)
	}

	// --- winds and plucked strings ---
	o.low = wg("bassClarinet", 3)
	ch("low clarinet", o.low, mixer.StripConfig{Volume: -8, Pan: 0.3, Sends: sends{"hall": 0.4}})
	o.solo = wg("altoFlute", 2)
	ch("alto flute", o.solo, mixer.StripConfig{Volume: -8, Pan: -0.25, Sends: sends{"hall": 0.45}})
	o.section = wg("clarinet", 4)
	sectionChorus := chorus.New(chorus.Config{Rate: 0.25, Depth: 5, Delay: 22, Voices: 3, Spread: 1, Tone: 7000, Mix: 0.45}).AddTo(p).In(o.section)
	ch("clarinets", sectionChorus, mixer.StripConfig{Volume: -8, Pan: -0.35, Sends: sends{"hall": 0.45}})
	o.glass = md("glassHarmonica", 6)
	gcfg := granulator.DefaultConfig()
	gcfg.Mix = 0.15
	gcfg.Delay = 350
	gcfg.Spray = 300
	gcfg.GrainSize = 260
	gcfg.Density = 24
	gcfg.Octave = 0.3
	gcfg.StereoSpread = 0.8
	gcfg.Feedback = 0.25
	o.glassCloud = granulator.New(gcfg).AddTo(p).In(o.glass).(*granulator.Granulator)
	ch("glass harmonica", o.glassCloud, mixer.StripConfig{Volume: 6, Pan: 0.35, Sends: sends{"hall": 0.5, "heavens": 0.2}})
	o.tine = md("spaceTine", 16)
	ch("space tine", o.tine, mixer.StripConfig{Volume: -11, Pan: 0.05, Sends: sends{"hall": 0.5, "heavens": 0.45}})
	o.marimba = md("marimba", 4)
	ch("marimba", o.marimba, mixer.StripConfig{Pan: 0.3, Sends: sends{"hall": 0.3}})
	// the oud rings longer than its preset: a plucked note that lasts, the lament has room for it
	oudCfg := waveguide.Oud()
	oudCfg.Decay, oudCfg.Damping, oudCfg.Release = 4.5, 0.3, 0.45
	o.oud = waveguide.NewPoly(oudCfg, 4).AddTo(p)
	oudEcho := tape.New(tape.Config{Time: 0.75, Feedback: 0.42, Wow: 0.3, Flutter: 0.2, HighCut: 3200, LowCut: 250, Mix: 0.35}).AddTo(p).In(o.oud)
	ch("oud", oudEcho, mixer.StripConfig{Volume: 4, Pan: 0.4, Sends: sends{"hall": 0.3}})
	o.harp = wg("celticHarp", 10)
	harpChorus := chorus.New(chorus.Config{Rate: 0.2, Depth: 4, Delay: 20, Voices: 3, Spread: 1, Tone: 8000, Mix: 0.4}).AddTo(p).In(o.harp)
	ch("harp", harpChorus, mixer.StripConfig{Pan: 0.5, Sends: sends{"hall": 0.45, "heavens": 0.1}})

	// --- winds ---
	o.bansuri = wg("bansuri", 2)
	ch("bansuri", o.bansuri, mixer.StripConfig{Volume: -8, Pan: -0.2, Sends: sends{"hall": 0.45, "heavens": 0.1}})
	o.recorder = wg("recorder", 2)
	ch("recorder", o.recorder, mixer.StripConfig{Volume: -8, Pan: 0.15, Sends: sends{"hall": 0.35}})
	o.clarinet = wg("bassClarinet", 2)
	ch("bass clarinet", o.clarinet, mixer.StripConfig{Volume: -8, Pan: -0.4, Sends: sends{"hall": 0.4}})
	o.shakuhachi = wg("shakuhachi", 2)
	scfg2 := granulator.DefaultConfig()
	scfg2.Mix = 0
	scfg2.Delay = 450
	scfg2.Spray = 350
	scfg2.GrainSize = 300
	scfg2.Density = 20
	scfg2.Octave = 0.25
	scfg2.StereoSpread = 0.7
	o.shakuCloud = granulator.New(scfg2).AddTo(p).In(o.shakuhachi).(*granulator.Granulator)
	ch("shakuhachi", o.shakuCloud, mixer.StripConfig{Volume: -8, Pan: 0.2, Sends: sends{"hall": 0.5, "heavens": 0.15}})

	// --- drums ---
	o.frame = wg("frameDrum", 4)
	ch("frame drum", o.frame, mixer.StripConfig{Pan: -0.1, Sends: sends{"hall": 0.3}})
	o.taiko = wg("taiko", 4)
	ch("taiko", o.taiko, mixer.StripConfig{Sends: sends{"hall": 0.4}})
	o.bodhran = wg("bodhran", 4)
	ch("bodhran", o.bodhran, mixer.StripConfig{Pan: 0.2, Sends: sends{"hall": 0.25}})
	o.tom = wg("floorTom", 4)
	ch("floor tom", o.tom, mixer.StripConfig{Pan: -0.3, Sends: sends{"hall": 0.35}})
	o.tambourine = wg("tambourine", 4)
	ch("tambourine", o.tambourine, mixer.StripConfig{Volume: 4, Pan: 0.45, Sends: sends{"hall": 0.25}})

	// --- bells and bars ---
	o.tubular = md("tubularBells", 8)
	ch("tubular bells", o.tubular, mixer.StripConfig{Group: "church", Pan: 0.1})
	o.bell = md("templeBell", 3)
	ch("temple bell", o.bell, mixer.StripConfig{Group: "church", Pan: -0.1})
	o.hand = md("handbellChoir", 8)
	ch("handbells", o.hand, mixer.StripConfig{Volume: 10, Group: "church", Pan: 0.2})
	o.anvil = md("anvil", 4)
	ch("anvil", o.anvil, mixer.StripConfig{Volume: 8, Pan: -0.3, Sends: sends{"hall": 0.2}})
	o.vibe = md("vibraphone", 8)
	ch("vibraphone", o.vibe, mixer.StripConfig{Pan: 0.3, Sends: sends{"hall": 0.4}})
	o.bowedVibe = md("bowedVibraphone", 6)
	ch("bowed vibraphone", o.bowedVibe, mixer.StripConfig{Pan: -0.3, Sends: sends{"hall": 0.5, "heavens": 0.2}})
	o.bowl = md("singingBowl", 3)
	ch("singing bowl", o.bowl, mixer.StripConfig{Volume: 6, Sends: sends{"hall": 0.5, "heavens": 0.3}})
	o.thunder = md("thunderSheet", 2)
	ch("thunder sheet", o.thunder, mixer.StripConfig{Volume: 6, Sends: sends{"hall": 0.4}})
	o.tamtam = md("tamTam", 2)
	ch("tam-tam", o.tamtam, mixer.StripConfig{Sends: sends{"hall": 0.45}})
	o.gong = md("gongAgeng", 2)
	ch("gong", o.gong, mixer.StripConfig{Sends: sends{"hall": 0.4}})
	o.kalimba = md("kalimba", 6)
	kalimbaEcho := pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.42, Mix: 0.32, HighCut: 5000, LowCut: 300}).AddTo(p).In(o.kalimba)
	ch("kalimba", kalimbaEcho, mixer.StripConfig{Volume: 4, Pan: -0.35, Sends: sends{"hall": 0.45, "heavens": 0.2}})
	o.glock = md("glockenspiel", 6)
	ch("glockenspiel", o.glock, mixer.StripConfig{Pan: 0.4, Sends: sends{"hall": 0.4, "heavens": 0.3}})

	// --- the keys ---
	o.grand = piano.NewPoly(piano.Presets["grand"](), 28).AddTo(p)
	ch("piano", o.grand, mixer.StripConfig{Pan: -0.05, Sends: sends{"hall": 0.3}})

	o.organLow = og("cathedral")
	ch("organ (cathedral)", o.organLow, mixer.StripConfig{Group: "church"})
	o.organSoft = og("softPad")
	organTurn := rotary.New(rotary.Config{Crossover: 800, Horn: 0.4, Spread: 0.8, Mix: 0.7}).AddTo(p).In(o.organSoft)
	ch("organ (soft)", organTurn, mixer.StripConfig{Group: "church"})
	o.organPlenum = og("churchPlenum")
	ch("organ (plenum)", o.organPlenum, mixer.StripConfig{Group: "church"})
	o.organBass = og("subBass")
	ch("organ (pedal)", o.organBass, mixer.StripConfig{Volume: 3, Group: "church"})
	o.organRaw = og("stringOrgan")
	ch("organ (strings)", o.organRaw, mixer.StripConfig{Group: "church"})

	// --- the voice: not in the music, its own reverb sends ---
	o.voice = newVoice(p)
	mix.Channel("voice", o.voice, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.25, "heavens": 0.2}})
	return o
}

var _ = arrange.Silent
