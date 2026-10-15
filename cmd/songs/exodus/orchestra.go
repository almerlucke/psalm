package main

import (
	"fmt"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/sndfile"
)

// orchestra is everything that sounds: the physically modelled instruments of the library and
// nothing else (the waveguide strings, bows, pipes and drums, the modal bells and bars, the piano
// and the organ), on a mixer with a hall and a church
type orchestra struct {
	mix *mixer.Mixer

	// the strings
	bass, cello, solo, section, tremolo, oud, harp psalm.Modular
	// the winds
	bansuri, recorder, clarinet, shakuhachi psalm.Modular
	// the drums
	frame, taiko, bodhran, tom, tambourine psalm.Modular
	// the bells and bars
	tubular, bell, hand, anvil, vibe, bowedVibe, bowl, thunder, tamtam, gong, kalimba, glock psalm.Modular
	// the keys
	grand                                      psalm.Modular
	organLow, organSoft, organPlenum, organRaw psalm.Modular
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
	scfg.Shimmer = 0.4
	scfg.Mix = 1
	mix.Bus("heavens", mixer.StripConfig{}, shimmer.New(scfg).AddTo(p))

	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	mix.Group("church", mixer.StripConfig{Volume: -2},
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
	ch := mix.Channel

	// --- strings ---
	o.bass = wg("contrabass", 3)
	ch("contrabass", o.bass, mixer.StripConfig{Pan: -0.1, Sends: sends{"hall": 0.25}})
	o.cello = wg("cello", 4)
	ch("cello", o.cello, mixer.StripConfig{Pan: 0.3, Sends: sends{"hall": 0.35}})
	o.solo = wg("violinSolo", 2)
	ch("solo violin", o.solo, mixer.StripConfig{Pan: -0.25, Sends: sends{"hall": 0.4}})
	o.section = wg("violinSection", 8)
	ch("violins", o.section, mixer.StripConfig{Pan: -0.35, Sends: sends{"hall": 0.45}})
	o.tremolo = wg("tremoloStrings", 6)
	ch("tremolo strings", o.tremolo, mixer.StripConfig{Volume: -4, Pan: 0.35, Sends: sends{"hall": 0.4}})
	o.oud = wg("oud", 4)
	ch("oud", o.oud, mixer.StripConfig{Volume: 4, Pan: 0.4, Sends: sends{"hall": 0.3}})
	o.harp = wg("celticHarp", 10)
	ch("harp", o.harp, mixer.StripConfig{Pan: 0.5, Sends: sends{"hall": 0.45, "heavens": 0.1}})

	// --- winds ---
	o.bansuri = wg("bansuri", 2)
	ch("bansuri", o.bansuri, mixer.StripConfig{Volume: -8, Pan: -0.2, Sends: sends{"hall": 0.45, "heavens": 0.1}})
	o.recorder = wg("recorder", 2)
	ch("recorder", o.recorder, mixer.StripConfig{Volume: -8, Pan: 0.15, Sends: sends{"hall": 0.35}})
	o.clarinet = wg("bassClarinet", 2)
	ch("bass clarinet", o.clarinet, mixer.StripConfig{Volume: -8, Pan: -0.4, Sends: sends{"hall": 0.4}})
	o.shakuhachi = wg("shakuhachi", 2)
	ch("shakuhachi", o.shakuhachi, mixer.StripConfig{Volume: -8, Pan: 0.2, Sends: sends{"hall": 0.5, "heavens": 0.15}})

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
	ch("kalimba", o.kalimba, mixer.StripConfig{Volume: 4, Pan: -0.35, Sends: sends{"hall": 0.45, "heavens": 0.2}})
	o.glock = md("glockenspiel", 6)
	ch("glockenspiel", o.glock, mixer.StripConfig{Pan: 0.4, Sends: sends{"hall": 0.4, "heavens": 0.3}})

	// --- the keys ---
	o.grand = piano.NewPoly(piano.Presets["grand"](), 28).AddTo(p)
	ch("piano", o.grand, mixer.StripConfig{Pan: -0.05, Sends: sends{"hall": 0.3}})

	o.organLow = og("cathedral")
	ch("organ (cathedral)", o.organLow, mixer.StripConfig{Group: "church"})
	o.organSoft = og("softPad")
	ch("organ (soft)", o.organSoft, mixer.StripConfig{Group: "church"})
	o.organPlenum = og("churchPlenum")
	ch("organ (plenum)", o.organPlenum, mixer.StripConfig{Group: "church"})
	o.organRaw = og("stringOrgan")
	ch("organ (strings)", o.organRaw, mixer.StripConfig{Group: "church"})
	return o
}

// warm makes the bowed strings' tunings before the first note (a friction bow looks its tuning up
// per pitch; offline renders wait for them)
func warm() {
	for _, n := range []string{"contrabass", "cello", "violinSolo", "violinSection", "tremoloStrings"} {
		waveguide.WarmBow(waveguide.Presets[n]())
	}
	waveguide.WaitBow()
}

var _ = arrange.Silent
