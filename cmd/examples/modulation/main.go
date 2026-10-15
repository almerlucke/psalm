package main

// A tour of the chorus and the flanger. Every section compares: two bars dry, then two bars with
// the effect, set up for that section (the effect settings come from the section parameters, the
// switch from dry to wet is an automation lane on the mix). A dry bass plays throughout.
//
//	go run ./cmd/examples/modulation                                    play it
//	go run ./cmd/examples/modulation -render modulation.wav -seed 3     render a pass

import (
	"flag"
	"fmt"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

// a scene for one of the effects: its settings and the mix once it comes in
type scene struct {
	sources  []string // the parts playing
	settings psalm.Message
	mix      float64
}

// the sections, in order, with the chorus or the flanger scene
var tour = []struct {
	name    string
	chorus  *scene
	flanger *scene
}{
	{"Chorus: electric piano, a subtle classic chorus", &scene{[]string{"epiano"},
		psalm.Message{"rate": 0.5, "depth": 2.0, "delay": 12.0, "spread": 1.0, "feedback": 0.0, "tone": 9000.0}, 0.5}, nil},
	{"Chorus: lush strings, deep and slow", &scene{[]string{"strings"},
		psalm.Message{"rate": 0.35, "depth": 5.0, "delay": 20.0, "spread": 1.0, "feedback": 0.0, "tone": 9000.0}, 0.6}, nil},
	{"Chorus: width, the spread sweeps from mono to wide", &scene{[]string{"epiano", "strings"},
		psalm.Message{"rate": 0.5, "depth": 3.0, "delay": 15.0, "feedback": 0.0, "tone": 9000.0}, 0.6}, nil},
	{"Chorus: doubler, a long delay that hardly moves", &scene{[]string{"guitar"},
		psalm.Message{"rate": 0.2, "depth": 1.0, "delay": 30.0, "spread": 1.0, "feedback": 0.0, "tone": 9000.0}, 0.5}, nil},
	{"Chorus: resonant, feedback and a darker tone", &scene{[]string{"guitar"},
		psalm.Message{"rate": 1.2, "depth": 3.0, "delay": 8.0, "spread": 1.0, "feedback": 0.7, "tone": 4000.0}, 0.5}, nil},
	{"Flanger: jet, a sweep every two bars with strong feedback", nil, &scene{[]string{"drums"},
		psalm.Message{"beats": 8.0, "manual": 0.3, "depth": 5.0, "feedback": 0.7, "stereo": 0.25}, 0.5}},
	{"Flanger: negative feedback, hollow", nil, &scene{[]string{"saw"},
		psalm.Message{"rate": 0.15, "manual": 0.5, "depth": 6.0, "feedback": -0.75, "stereo": 0.5}, 0.5}},
	{"Flanger: stereo swirl, the sides sweep in opposite directions", nil, &scene{[]string{"drums", "saw"},
		psalm.Message{"rate": 0.3, "manual": 0.5, "depth": 4.0, "feedback": 0.5, "stereo": 1.0}, 0.5}},
	{"Flanger: manual sweep, the shortest delay swept by automation", nil, &scene{[]string{"saw"},
		psalm.Message{"rate": 0.0, "depth": 0.0, "feedback": 0.8, "stereo": 0.0}, 0.5}},
}

func main() {
	renderTo := flag.String("render", "", "render one pass to this WAV file instead of playing it")
	seed := flag.Uint64("seed", 0, "seed for the random parts: the same seed plays the same, 0 is new every run")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}

	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	// --- the form: a section of 4 bars per scene ---

	var sections []arrange.Section
	for _, t := range tour {
		sections = append(sections, arrange.Section{Name: t.name, Bars: 4})
	}
	form := arrange.NewForm(sections...)
	plays := func(s arrange.Section, part string) bool {
		for _, t := range tour {
			if t.name != s.Name {
				continue
			}
			for _, sc := range []*scene{t.chorus, t.flanger} {
				if sc == nil {
					continue
				}
				for _, src := range sc.sources {
					if src == part {
						return true
					}
				}
			}
		}
		return false
	}

	// --- harmony: Am F C G, a bar each, voice led ---

	chords := harmony.MustProgression("Am F C G")
	voicings := [][]float64{chords[0].Close(57)}
	for _, c := range chords[1:] {
		voicings = append(voicings, harmony.VoiceLead(voicings[len(voicings)-1], c))
	}

	// --- instruments ---

	epiano := dx.NewPoly(dx.EPianoPreset(), 12, psalm.Stereo).AddTo(p)
	strings := wavetable.NewPoly(wavetable.StringsPreset(gen.Brightness(32)), 12).AddTo(p)
	guitar := dx.NewPoly(dx.NylonGuitarPreset(), 8, psalm.Stereo).AddTo(p)
	saw := subtractive.NewPoly(subtractive.PWMPadPreset(), 8).AddTo(p)
	bass := dx.NewPoly(dx.BassPreset(), 2, psalm.Stereo).AddTo(p)
	kit := drums.NewDrums(sndfile.SoundBank[float32]{
		"kick":  sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/kick/Cymatics - Humble Sit Down Kick - D.wav", 4),
		"snare": sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/snare/Cymatics - Humble Adequate Snare - E.wav", 4),
		"hat":   sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4),
	}, 16, 2).AddTo(p)

	// --- the parts, playing in the sections that show them ---

	feel := score.Feel{Timing: 6, Velocity: 0.1}
	part := func(synth psalm.Modular, name string, amp, legato float64, bar func(c int) []score.Note) {
		form.Part(p, synth, name, score.Config{Amp: amp, Legato: legato, Velocity: 0.8, Feel: feel}, func(s arrange.Section) []score.Note {
			if !plays(s, name) {
				return nil
			}
			return arrange.PerBar(s, func(b int) []score.Note { return bar(b % len(chords)) })
		})
	}

	part(epiano, "epiano", 0.25, 0.9, func(c int) []score.Note { // comping on the off beats
		v := voicings[c]
		return []score.Note{score.Rest(0.5), score.Chord(v, 1), score.Rest(0.5), score.Chord(v, 1.5), score.Chord(v, 0.5)}
	})
	part(strings, "strings", 0.3, 1.0, func(c int) []score.Note {
		return []score.Note{score.Chord(score.Pitches(voicings[c], 12), 4)}
	})
	part(guitar, "guitar", 0.4, 1.4, func(c int) []score.Note {
		return harmony.Arp{Pattern: harmony.UpDown, Octaves: 2, Steps: 8, Velocities: []float64{1, 0.6, 0.8, 0.6}}.Notes(voicings[c])
	})
	part(saw, "saw", 0.3, 1.0, func(c int) []score.Note {
		return []score.Note{score.Chord(voicings[c], 4)}
	})
	hit := func(sound string, beats float64) score.Note {
		return score.Note{Pitches: []float64{60}, Beats: beats, Params: map[string]any{"sound": sound, "speed": 1.0}}
	}
	form.Part(p, kit, "drums", score.Config{Amp: 0.5, Feel: score.Feel{Swing: 0.56, SwingUnit: 0.25, Velocity: 0.15}}, func(s arrange.Section) []score.Note {
		if !plays(s, "drums") {
			return nil
		}
		return arrange.PerBar(s, func(int) []score.Note {
			var ns []score.Note
			for step := range 16 {
				switch {
				case step == 0 || step == 10:
					ns = append(ns, hit("kick", 0.25))
				case step == 4 || step == 12:
					ns = append(ns, hit("snare", 0.25))
				default:
					n := hit("hat", 0.25)
					n.Amp = 0.5
					ns = append(ns, n)
				}
			}
			return ns
		})
	})
	form.Part(p, bass, "bass", score.Config{Amp: 0.5, Legato: 0.8, Velocity: 0.8}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(b int) []score.Note {
			root := chords[b%len(chords)].BassPitch(33)
			return score.Notes(1, root, 0, root+12, root+7)
		})
	})

	// --- the mix: the chorus and the flanger as inserts on two groups ---

	ens := chorus.New(chorus.Config{Voices: 4, Rate: 0.5, Depth: 3, Delay: 15, Spread: 1, Tone: 9000})
	ens.AddTo(p)
	jet := flanger.New(flanger.Config{Rate: 0.2, Manual: 0.5, Depth: 4, Feedback: 0.5, Shape: flanger.Triangle})
	jet.AddTo(p)

	mix := mixer.New(p)
	room := fdnreverb.DefaultConfig()
	room.Decay = 1.8
	room.Mix = 1
	mix.Bus("room", mixer.StripConfig{}, fdnreverb.New(room).AddTo(p))
	mix.Group("chorus", mixer.StripConfig{Volume: -2, Sends: map[string]float64{"room": 0.2}}, ens)
	mix.Group("flanger", mixer.StripConfig{Sends: map[string]float64{"room": 0.15}}, jet)
	mix.Channel("epiano", epiano, mixer.StripConfig{Group: "chorus"})
	mix.Channel("strings", strings, mixer.StripConfig{Group: "chorus"})
	mix.Channel("guitar", guitar, mixer.StripConfig{Group: "chorus"})
	mix.Channel("drums", kit, mixer.StripConfig{Group: "flanger", Volume: -3})
	mix.Channel("saw", saw, mixer.StripConfig{Group: "flanger", Volume: -3})
	mix.Channel("bass", bass, mixer.StripConfig{})
	p.Out(mix.Master())

	// --- the arrangement: every section sets up its effect, the mix comes in halfway ---

	chorusMix := arrange.Lane{Sections: map[string]arrange.Curve{}}
	flangerMix := arrange.Lane{Sections: map[string]arrange.Curve{}}
	halfway := func(mix float64) arrange.Curve {
		return func(progress float64) float64 {
			if progress < 0.5 {
				return 0 // the first two bars dry
			}
			return mix
		}
	}
	for _, t := range tour {
		if t.chorus != nil {
			chorusMix.Sections[t.name] = halfway(t.chorus.mix)
		}
		if t.flanger != nil {
			flangerMix.Sections[t.name] = halfway(t.flanger.mix)
		}
	}

	arrangement := arrange.NewArrangement(form).
		OnSection(func(pos arrange.Position) {
			for _, t := range tour {
				if t.name != pos.Section.Name {
					continue
				}
				fmt.Printf("bar %2d  %s\n", pos.Bar+1, t.name)
				if t.chorus != nil {
					ens.ReceiveMessage(t.chorus.settings)
				}
				if t.flanger != nil {
					jet.ReceiveMessage(t.flanger.settings)
				}
			}
		}).
		AutomateMessage(chorusMix, ens, "mix").
		AutomateMessage(flangerMix, jet, "mix").
		// the width section sweeps the spread from mono to wide, the manual section the delay
		AutomateMessage(arrange.Lane{Default: 1, Sections: map[string]arrange.Curve{
			tour[2].name: arrange.Linear(0, 1),
		}}, ens, "spread").
		AutomateMessage(arrange.Lane{Default: 0.5, Sections: map[string]arrange.Curve{
			tour[8].name: arrange.Exp(8, 0.3),
		}}, jet, "manual")
	arrangement.AddTo(p)

	fmt.Println("Chorus and flanger")
	fmt.Print(arrangement.Schedule())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 5})
		if err != nil {
			panic(err)
		}
		fmt.Println(stats)
		fmt.Print(mix.Meters())
		return
	}
	_ = p.Listen()
}
