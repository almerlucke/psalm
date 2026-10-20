// Ensemble: a short piece for the synths that model instruments, or the machines that made a sound
// their own.
//
//	Intro     a church bell, a choir on "u", a harp, a singing bowl
//	Groove    the 909 (drumsynth), a fingered bass (waveguide), a marimba (modal), the jazz organ
//	          through a rotary speaker, a wood block (modal), a sampled kalimba-ish tine (sampler)
//	Lift      a trance supersaw pad and plucks, the 808, the cello bowing the melody, the shaper
//	          synth's electric piano
//	Outro     slow strings, tubular bells, the choir again on "a"
//
//	go run ./cmd/examples/ensemble                       play it
//	go run ./cmd/examples/ensemble -render ensemble.wav  render it
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/rotary"
	"github.com/almerlucke/psalm/modules/effects/tape"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
)

type (
	S = arrange.Section
	N = score.Note
)

func main() {
	renderTo := flag.String("render", "", "render to this WAV file instead of playing")
	seed := flag.Uint64("seed", 0, "seed for everything random")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}
	psalm.PushConfig(psalm.Config{BPM: 104, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	form := arrange.NewForm(
		S{Name: "Intro", Bars: 4, Dynamics: 0.8},
		S{Name: "Groove", Bars: 8},
		S{Name: "Lift", Bars: 8, Dynamics: 1.1},
		S{Name: "Outro", Bars: 4, Dynamics: 0.85},
	)
	chords := harmony.MustKey("A minor").MustProgression("i VI III VII")
	chordOf := func(s S, bar int) harmony.Chord { return chords[bar%4] }
	played := score.Feel{Timing: 6, Velocity: 0.08}

	// --- the drums: the 909 in the groove, the 808 in the lift ---

	tr909 := drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kit909()}, 12).AddTo(p)
	tr808 := drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kit808()}, 12).AddTo(p)
	// beat writes a bar from text: X an accent, x a hit, o a ghost, . nothing (16ths)
	beat := func(sound, text string) []N {
		amp := map[rune]float64{'X': 1, 'x': 0.7, 'o': 0.35}
		var ns []N
		for _, c := range text {
			if c == ' ' {
				continue
			}
			if a, ok := amp[c]; ok {
				ns = append(ns, N{Pitches: []float64{60}, Beats: 0.25, Amp: a, Params: map[string]any{"sound": sound}})
			} else {
				ns = append(ns, score.Rest(0.25))
			}
		}
		return ns
	}
	drumPart := func(kit psalm.Modular, name, section string, pattern func(bar int) []N) {
		form.Part(p, kit, name, score.Config{Feel: score.Feel{Timing: 3, Velocity: 0.05}}, arrange.Only(section, func(s S) []N {
			return arrange.PerBar(s, pattern)
		}))
	}
	drumPart(tr909, "909 kick", "Groove", func(int) []N { return beat("kick", "X... X... X... X...") })
	drumPart(tr909, "909 snare", "Groove", func(bar int) []N {
		if bar == 7 {
			return beat("snare", ".... X... ..o. XxXX")
		}
		return beat("snare", ".... X... .... X..o")
	})
	drumPart(tr909, "909 hats", "Groove", func(int) []N {
		return append(beat("hat", "x.x. x.x. x.x. x."), beat("open", "x.")...)
	})
	drumPart(tr808, "808 kick", "Lift", func(bar int) []N { return beat("kick", "X... ...x ..X. ....") })
	drumPart(tr808, "808 clap", "Lift", func(int) []N { return beat("clap", ".... X... .... X...") })
	drumPart(tr808, "808 hats", "Lift", func(int) []N { return beat("hat", "xxox xxox xxox xxox") })
	drumPart(tr808, "808 cowbell", "Lift", func(bar int) []N {
		if bar%2 == 1 {
			return beat("cowbell", "..x. .... ..x. x...")
		}
		return nil
	})

	// --- the instruments ---

	bell := modal.NewPoly(modal.ChurchBell(), 2).AddTo(p)
	bowl := modal.NewPoly(modal.SingingBowl(), 2).AddTo(p)
	marimba := modal.NewPoly(modal.Marimba(), 8).AddTo(p)
	chimes := modal.NewPoly(modal.TubularBells(), 4).AddTo(p)
	choir := vocal.NewPoly(vocal.Ooh(), 6).AddTo(p)
	harp := waveguide.NewPoly(waveguide.Harp(), 12).AddTo(p)
	bass := waveguide.NewPoly(waveguide.Bass(), 2).AddTo(p)
	cello := waveguide.NewPoly(waveguide.Cello(), 2).AddTo(p)
	strings := waveguide.NewPoly(waveguide.SlowStrings(), 8).AddTo(p)
	jazz := organ.New(organ.Jazz()).AddTo(p)
	blocks := modal.NewPoly(modal.WoodBlock(), 6).AddTo(p)
	pad := supersaw.NewPoly(supersaw.TrancePad(), 8).AddTo(p)
	plucks := supersaw.NewPoly(supersaw.Pluck(), 8).AddTo(p)
	ePiano := shaper.NewPoly(shaper.EPiano(), 8).AddTo(p)

	// a tine, sampled from a percussion hit, played over the keys
	tine, err := sampler.LoadFile("assets/sounds/drums/percussion/Cymatics - Odyssey Percussion 3.wav")
	if err != nil {
		log.Fatal(err)
	}
	region := sampler.NewRegion(tine).Keys(36, 96, 72)
	region.Loop, region.Env.Release = sampler.OneShot, 0.1
	tines := sampler.NewPoly(sampler.DefaultConfig(sampler.MustInstrument(region)), 8).AddTo(p)

	// the intro and the outro
	form.Part(p, bell, "bell", score.Config{Velocity: 0.7}, arrange.Only("Intro", func(s S) []N {
		return score.MustParse(`A3:16`)
	}))
	form.Part(p, bowl, "bowl", score.Config{Velocity: 0.5}, arrange.Only("Intro", func(s S) []N {
		return score.MustParse(`r:8 E4:8`)
	}))
	form.Part(p, choir, "choir", score.Config{Amp: 0.7}, func(s S) []N {
		if !arrange.In(s, "Intro", "Outro") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordOf(s, bar).Close(57), 4)} })
	})
	form.Part(p, harp, "harp", score.Config{Velocity: 0.55, Feel: played}, arrange.Only("Intro", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := chordOf(s, bar)
			v := append([]float64{c.BassPitch(45)}, c.Close(60)...)
			return score.Notes(0.5, v[0], v[1], v[2], v[3], v[2], v[1], v[0]+12, v[1]+12)
		})
	}))
	form.Part(p, strings, "strings", score.Config{Amp: 0.6}, arrange.Only("Outro", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordOf(s, bar).Close(64), 4)} })
	}))
	form.Part(p, chimes, "chimes", score.Config{Velocity: 0.75}, arrange.Only("Outro", func(s S) []N {
		return score.MustParse(`E5:4 | C5:4 | G5:4 | A5:4`)
	}))

	// the groove
	form.Part(p, bass, "bass", score.Config{Velocity: 0.8, Feel: played}, func(s S) []N {
		if !arrange.In(s, "Groove", "Lift") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			r := chordOf(s, bar).BassPitch(33)
			return score.Shape(score.Notes(0.5, r, 0, r, r+12, 0, r+7, r+10, r+12), score.Velocities(0.9, 0, 0.6, 0.7, 0, 0.6, 0.5, 0.6), score.Hold(0.8))
		})
	})
	form.Part(p, marimba, "marimba", score.Config{Velocity: 0.6, Feel: played}, arrange.Only("Groove", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			v := chordOf(s, bar).Close(69)
			return score.Notes(0.25, v[0], 0, v[2], v[1], 0, v[0]+12, v[2], 0, v[1], 0, v[0], v[2], 0, v[1], v[2], 0)
		})
	}))
	form.Part(p, jazz, "organ", score.Config{Amp: 0.6, Feel: played}, arrange.Only("Groove", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := score.Chord(chordOf(s, bar).Close(60), 0.75)
			return []N{score.Rest(0.5), c, score.Rest(0.75), c, score.Rest(0.5), c}
		})
	}))
	form.Part(p, blocks, "wood block", score.Config{Feel: played}, arrange.Only("Groove", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			return score.Shape(score.Notes(0.25, 62, 0, 67, 62, 0, 62, 67, 0, 62, 0, 67, 0, 62, 67, 69, 67),
				score.Velocities(0.8, 0, 0.5, 0.6, 0, 0.7, 0.5, 0, 0.8, 0, 0.5, 0, 0.7, 0.5, 0.6, 0.5))
		})
	}))
	form.Part(p, tines, "tines", score.Config{Velocity: 0.75, Feel: played}, arrange.Only("Groove", func(s S) []N {
		return score.MustParse(`r:4 | r:2 E6:0.5 A6:0.5 C7:1 | r:4 | r:2 D7:0.5 C7:0.5 G6:1 |
			r:4 | r:2 E6:0.5 A6:0.5 C7:1 | r:4 | B6:1 G6:1 E6:2`)
	}))

	// the lift
	form.Part(p, pad, "pad", score.Config{Amp: 0.55}, arrange.Only("Lift", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordOf(s, bar).Close(60), 4)} })
	}))
	form.Part(p, plucks, "plucks", score.Config{Velocity: 0.6, Feel: played}, arrange.Only("Lift", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			v := chordOf(s, bar).Close(72)
			return score.Notes(0.25, v[0], v[1], v[2], v[1], v[0]+12, v[2], v[1], v[2], v[0], v[1], v[2], v[1], v[0]+12, v[2], v[1], v[0])
		})
	}))
	form.Part(p, ePiano, "e-piano", score.Config{Amp: 0.6, Feel: played}, arrange.Only("Lift", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := score.Chord(chordOf(s, bar).Close(55), 1.5)
			return []N{c, score.Rest(0.5), score.Chord(chordOf(s, bar).Close(55), 1), score.Rest(1)}
		})
	}))
	form.Part(p, cello, "cello", score.Config{Amp: 0.9, Feel: score.Feel{Timing: 10, Push: 4}}, arrange.Only("Lift", func(s S) []N {
		return score.MustParse(`A3:3 C4:1 | E4:4 | D4:2 C4:1 B3:1 | C4:4 |
			A3:3 C4:1 | E4:3 G4:1 | F4:2 E4:1 D4:1 | E4:4`)
	}))

	// --- the effects and the mix ---

	leslie := rotary.New(rotary.DefaultConfig())
	leslie.AddTo(p).In(jazz)
	tinesEcho := tape.New(tape.Config{Time: 0.75, Feedback: 0.45, Wow: 0.3, Flutter: 0.2, HighCut: 4000, LowCut: 300, Mix: 0.3}).AddTo(p).In(tines)

	arrangement := arrange.NewArrangement(form).
		At("Groove", 4, func() { leslie.SetFast(true) }).
		PrintSections(true)
	arrangement.AddTo(p)

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay, hall.Mix = 2.6, 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Group("drums", mixer.StripConfig{Volume: -8, Sends: map[string]float64{"hall": 0.08}})
	mix.Channel("909", tr909, mixer.StripConfig{Group: "drums"})
	mix.Channel("808", tr808, mixer.StripConfig{Group: "drums"})
	for _, c := range []struct {
		name   string
		synth  psalm.Modular
		volume float64
		pan    float64
		hall   float64
	}{
		{"bell", bell, -10, 0, 0.5},
		{"bowl", bowl, -12, 0.3, 0.5},
		{"choir", choir, -17, 0, 0.4},
		{"harp", harp, -15, -0.25, 0.35},
		{"strings", strings, -14, 0, 0.4},
		{"chimes", chimes, -10, 0.3, 0.45},
		{"bass", bass, -10, 0, 0.05},
		{"marimba", marimba, -15, -0.3, 0.25},
		{"organ", leslie, -16, 0, 0.2},
		{"wood block", blocks, -10, 0.35, 0.2},
		{"tines", tinesEcho, -10, 0.2, 0.3},
		{"pad", pad, -17, 0, 0.3},
		{"plucks", plucks, -20, 0, 0.3},
		{"e-piano", ePiano, -18, -0.2, 0.25},
		{"cello", cello, -5, 0.1, 0.35},
	} {
		mix.Channel(c.name, c.synth, mixer.StripConfig{Volume: c.volume, Pan: c.pan, Sends: map[string]float64{"hall": c.hall}})
	}
	p.Out(mix.Master())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 5})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stats)
		fmt.Print(mix.Meters())
		return
	}
	fmt.Println("playing, Ctrl-C to stop")
	_ = p.Listen()
}
