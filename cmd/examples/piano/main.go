package main

// A tour of the physically modelled piano. The grand plays a nocturne (with the sustain pedal
// changed every bar, the strings ringing along), then shows its dynamics, the whole keyboard, the
// sustain pedal (and half pedaling), the sostenuto and the soft pedal; after that every other preset plays a short piece that suits
// it. All music is original.
//
//	go run ./cmd/examples/piano                                  play it
//	go run ./cmd/examples/piano -render piano.wav -seed 3        render a pass

import (
	"flag"
	"fmt"
	"math"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

type (
	S = arrange.Section
	N = score.Note
)

const (
	nocturne = "Grand: a nocturne, the pedal changed every bar"
	dynamics = "Grand: dynamics, one chord from pianissimo to fortissimo"
	keyboard = "Grand: the whole keyboard, from A0 to the top"
	pedal    = "Grand: pedal, two bars dry, a bar half pedal, a bar with the pedal down"
	sost     = "Grand: sostenuto, a low C held under short chords"
	soft     = "Grand: the soft pedal, a phrase and again with the soft pedal down"
	pop      = "Bright: pop chords"
	hymn     = "Upright: a hymn"
	boogie   = "Honky tonk: boogie"
	ambient  = "Felt: ambient"
	musicBox = "Toy: a music box"
)

// the parade: the other presets play the same phrase one after the other
var parade = []struct {
	name, section string
	preset        func() piano.Config
}{
	{"concert", "Parade: concert, the grand from the audience", piano.ConcertPreset},
	{"warm", "Parade: warm, a grand in a jazz club", piano.WarmPreset},
	{"baby", "Parade: baby, a small grand", piano.BabyPreset},
	{"vintage", "Parade: vintage, an old upright", piano.VintagePreset},
	{"cinematic", "Parade: cinematic, soft and dark from the hall", piano.CinematicPreset},
}

func main() {
	renderTo := flag.String("render", "", "render one pass to this WAV file instead of playing it")
	seed := flag.Uint64("seed", 0, "seed for the human feel: the same seed plays the same, 0 is new every run")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}

	psalm.PushConfig(psalm.Config{BPM: 66, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	sections := []S{
		{Name: nocturne, Bars: 8},
		{Name: dynamics, Bars: 4},
		{Name: keyboard, Bars: 3},
		{Name: pedal, Bars: 4},
		{Name: sost, Bars: 4},
		{Name: soft, Bars: 4},
		{Name: pop, Bars: 4},
		{Name: hymn, Bars: 4},
		{Name: boogie, Bars: 4},
		{Name: ambient, Bars: 4},
		{Name: musicBox, Bars: 4},
	}
	for _, pr := range parade {
		sections = append(sections, S{Name: pr.section, Bars: 4})
	}
	form := arrange.NewForm(sections...)

	// --- the pianos, all on the mixer with a hall: the Musikverein in Vienna, a recorded impulse
	// response through the convolver (large partitions keep it cheap, on a bus their latency is a
	// pre delay) ---

	mix := mixer.New(p)
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	mix.Bus("hall", mixer.StripConfig{}, convolver.New(vienna, convolver.Config{PartitionSize: 2048, Mix: 1, Gain: 0.7}).AddTo(p))

	type instrument struct {
		name   string
		preset func() piano.Config
		voices int
		hall   float64
	}
	pianos := map[string]psalm.Modular{}
	for _, in := range []instrument{
		{"grand", piano.GrandPreset, 24, 0.3},
		{"bright", piano.BrightPreset, 12, 0.2},
		{"upright", piano.UprightPreset, 12, 0.2},
		{"honkyTonk", piano.HonkyTonkPreset, 12, 0.12},
		{"felt", piano.FeltPreset, 16, 0.45},
		{"toy", piano.ToyPreset, 8, 0.35},
		{"concert", piano.ConcertPreset, 16, 0.3},
		{"warm", piano.WarmPreset, 16, 0.25},
		{"baby", piano.BabyPreset, 16, 0.25},
		{"vintage", piano.VintagePreset, 16, 0.2},
		{"cinematic", piano.CinematicPreset, 16, 0.4},
	} {
		synth := piano.NewPoly(in.preset(), in.voices).AddTo(p)
		pianos[in.name] = synth
		mix.Channel(in.name, synth, mixer.StripConfig{Volume: -3.5, Sends: map[string]float64{"hall": in.hall}})
	}
	p.Out(mix.Master())

	part := func(instr, name string, amp float64, feel score.Feel, sc func(s S) []N) {
		form.Part(p, pianos[instr], name, score.Config{Amp: amp, Legato: 0.95, Velocity: 0.7, Feel: feel}, sc)
	}
	human := score.Feel{Timing: 8, Velocity: 0.08}

	// --- the nocturne: E flat, an arpeggiated left hand and a melody ---

	nocturneChords := harmony.MustProgression("Ebmaj7 Cm7 Abmaj7 Bb7 Ebmaj7 Gm7 Fm7 Eb")
	part("grand", "nocturne melody", 0.65, score.Feel{Timing: 12, Velocity: 0.08, Push: 10}, func(s S) []N {
		if s.Name != nocturne {
			return nil
		}
		return score.MustParse(`
			G5:1.5@.7 F5:.5@.55 Eb5:1@.6 Bb4:1@.55 | C5:1.5@.65 D5:.5@.55 Eb5:1@.6 G5:1@.7 |
			F5:1@.65 Eb5:1@.6 C5:1.5@.6 Ab4:.5@.5 | Bb4:2@.6 D5:1@.6 F5:1@.65 |
			G5:1.5@.75 Bb5:.5@.7 Ab5:1@.7 G5:1@.65 | F5:2@.7 D5:1@.6 Bb4:1@.55 |
			Eb5:1@.6 F5:1@.65 Ab5:1@.7 C5:1@.55 | Eb5:4_6@.6`)
	})
	part("grand", "nocturne left hand", 0.5, human, func(s S) []N {
		if s.Name != nocturne {
			return nil
		}
		return arrange.PerBar(s, func(b int) []N {
			c := nocturneChords[b%len(nocturneChords)]
			ps := c.Pitches(c.BassPitch(34))
			root, third, fifth := ps[0], ps[1], ps[2]
			var ns []N
			for i, pitch := range []float64{root, fifth, root + 12, third + 12, fifth + 12, third + 12, root + 12, fifth} {
				n := score.N(pitch, 0.5)
				n.Velocity = []float64{0.6, 0.4, 0.45, 0.4, 0.5, 0.4, 0.45, 0.4}[i]
				n.Hold = 3
				ns = append(ns, n)
			}
			return ns
		})
	})

	// --- dynamics: the same chord eight times, from 0.1 to 1 ---

	part("grand", "dynamics", 0.11, score.Feel{}, func(s S) []N {
		if s.Name != dynamics {
			return nil
		}
		var ns []N
		for i := range 8 {
			n := score.Chord([]float64{39, 46, 55, 62, 67, 74}, 2)
			n.Velocity = 0.1 + 0.9*float64(i)/7
			n.Hold = 1.6
			ns = append(ns, n)
		}
		return ns
	})

	// --- the whole keyboard: E flat major up from E flat 1 and down, then the lowest and highest keys ---

	part("grand", "keyboard", 0.35, score.Feel{Timing: 4, Velocity: 0.05}, func(s S) []N {
		if s.Name != keyboard {
			return nil
		}
		var up []float64
		for pitch := 27.0; pitch <= 99; pitch++ {
			if pc := int(pitch) % 12; pc == 3 || pc == 7 || pc == 10 {
				up = append(up, pitch)
			}
		}
		var ns []N
		for i, pitch := range up {
			n := score.N(pitch, 0.25)
			n.Velocity = 0.4 + 0.4*float64(i)/float64(len(up))
			ns = append(ns, n)
		}
		for i := len(up) - 2; i >= 1; i-- {
			n := score.N(up[i], 0.25)
			n.Velocity = 0.4 + 0.4*float64(i)/float64(len(up))
			ns = append(ns, n)
		}
		used := score.Beats(ns)
		ends := score.Chord([]float64{21, 33, 96, 108}, 12-used)
		ends.Velocity = 0.7
		return append(ns, ends)
	})

	// --- pedal: short chords, dry and then held by the pedal ---

	pedalChords := harmony.MustProgression("Ab Bb Gm Cm")
	part("grand", "pedal", 0.3, human, func(s S) []N {
		if s.Name != pedal {
			return nil
		}
		return arrange.PerBar(s, func(b int) []N {
			c := pedalChords[b%len(pedalChords)]
			v := append([]float64{c.BassPitch(36)}, c.Close(60)...)
			var ns []N
			for range 4 {
				n := score.Chord(v, 1)
				n.Hold = 0.25
				ns = append(ns, n)
			}
			return ns
		})
	})

	// --- sostenuto: a low C caught by the pedal, short chords above it stay dry ---

	part("grand", "sostenuto", 0.5, human, func(s S) []N {
		if s.Name != sost {
			return nil
		}
		low := score.Chord([]float64{36, 48}, 1)
		low.Hold = 0.8
		low.Velocity = 0.75
		ns := []N{low}
		for i, c := range harmony.MustProgression("C F G C") {
			for b := range 4 {
				if i == 0 && b == 0 {
					continue // the low C takes the first beat
				}
				n := score.Chord(c.Close(60), 1)
				n.Hold = 0.2
				n.Velocity = 0.55
				ns = append(ns, n)
			}
		}
		return ns
	})

	// --- the soft pedal: a phrase, then again una corda ---

	part("grand", "soft", 0.65, human, func(s S) []N {
		if s.Name != soft {
			return nil
		}
		phrase := score.MustParse(`[48 55 64]:1_3@.6 E5:.5@.6 D5:.5@.55 C5:1@.6 G4:1@.5 | [41 57 60]:1_3@.6 A4:.5@.55 C5:.5@.6 F5:2@.65`)
		return score.Join(phrase, phrase)
	})

	// --- bright: pumping 8th chords and root octaves ---

	popChords := harmony.MustProgression("G D Em C")
	popVoicings := [][]float64{popChords[0].Close(59)}
	for _, c := range popChords[1:] {
		popVoicings = append(popVoicings, harmony.VoiceLead(popVoicings[len(popVoicings)-1], c))
	}
	part("bright", "pop chords", 0.24, score.Feel{Timing: 5, Velocity: 0.06, Accents: []float64{1, 0.7, 0.85, 0.7}, AccentUnit: 0.5}, func(s S) []N {
		if s.Name != pop {
			return nil
		}
		return arrange.PerBar(s, func(b int) []N {
			n := score.Chord(popVoicings[b%4], 0.5)
			n.Hold = 0.4
			return score.Repeat([]N{n}, 8)
		})
	})
	part("bright", "pop bass", 0.25, human, func(s S) []N {
		if s.Name != pop {
			return nil
		}
		return arrange.PerBar(s, func(b int) []N {
			r := popChords[b%4].BassPitch(31)
			return []N{score.Chord([]float64{r, r + 12}, 2), score.Chord([]float64{r, r + 12}, 1.5), score.Chord([]float64{r + 7, r + 19}, 0.5)}
		})
	})

	// --- upright: a hymn in four parts, half notes ---

	part("upright", "hymn", 0.47, score.Feel{Timing: 10, Spread: 6, Velocity: 0.06}, func(s S) []N {
		if s.Name != hymn {
			return nil
		}
		return score.MustParse(`
			[41 57 60 69]:2@.6 [38 57 62 65]:2@.55 | [46 58 62 70]:2@.65 [48 55 64 67]:2@.6 |
			[45 57 60 72]:2@.7 [43 58 62 70]:2@.6 | [48 55 60 64]:2@.55 [41 57 60 65]:2_3@.5`)
	})

	// --- honky tonk: a boogie left hand and swung stabs ---

	boogieRoots := []float64{36, 36, 41, 36}
	swing := score.Feel{Swing: 0.64, Timing: 6, Velocity: 0.08}
	part("honkyTonk", "boogie left hand", 0.33, swing, func(s S) []N {
		if s.Name != boogie {
			return nil
		}
		return arrange.PerBar(s, func(b int) []N {
			r := boogieRoots[b%4]
			return score.Notes(0.5, r, r+4, r+7, r+9, r+10, r+9, r+7, r+4)
		})
	})
	part("honkyTonk", "boogie stabs", 0.26, swing, func(s S) []N {
		if s.Name != boogie {
			return nil
		}
		return arrange.PerBar(s, func(b int) []N {
			r := boogieRoots[b%4] + 24
			stab := score.Chord([]float64{r + 4, r + 7, r + 10, r + 12}, 1)
			stab.Hold = 0.3
			return []N{score.Rest(0.5), stab, score.Rest(0.5), stab, {Beats: 1}}
		})
	})

	// --- felt: slow arpeggios in the middle of the keyboard, the pedal down ---

	feltChords := harmony.MustProgression("Dmaj7 Bm7 Gmaj7 A")
	part("felt", "ambient", 0.52, score.Feel{Timing: 15, Velocity: 0.1}, func(s S) []N {
		if s.Name != ambient {
			return nil
		}
		return arrange.PerBar(s, func(b int) []N {
			c := feltChords[b%4]
			ns := harmony.Arp{Pattern: harmony.UpDown, Octaves: 2, Steps: 8, Velocities: []float64{0.5, 0.35, 0.4, 0.35}}.Notes(c.Close(62))
			return append([]N{score.Chord([]float64{c.BassPitch(38)}, 0)}, ns...)
		})
	})

	// --- toy: a music box tune ---

	part("toy", "music box", 0.4, score.Feel{Timing: 4, Velocity: 0.05}, func(s S) []N {
		if s.Name != musicBox {
			return nil
		}
		return score.MustParse(`
			E6:.5 G6 C7:1 B6:.5 G6 A6:1 | F6:.5 A6 D7:1 C7:.5 A6 B6:1 |
			G6:.5 B6 E7:1 D7:.5 B6 C7:.5 A6 | G6:1 E6:1 C6:2`)
	})
	part("toy", "music box bass", 0.3, score.Feel{}, func(s S) []N {
		if s.Name != musicBox {
			return nil
		}
		return score.MustParse(`[48 55]:2 [48 55] | [41 53] [43 55] | [43 55] [45 52] | [43 53] [48 55]`)
	})

	// --- the parade: one phrase in A minor on every other preset ---

	paradeChords := harmony.MustProgression("Am F C G")
	melody := score.MustParse(`
		E5:1.5@.65 D5:.5@.55 C5:1@.6 B4:1@.55 | A4:1.5@.6 C5:.5@.55 F5:2@.7 |
		E5:1@.6 G5:1@.65 C5:2@.6 | B4:1@.55 D5:1@.6 G5:2_3@.65`)
	for _, pr := range parade {
		section := pr.section
		part(pr.name, pr.name+" melody", 0.6, score.Feel{Timing: 12, Velocity: 0.08, Push: 8}, func(s S) []N {
			if s.Name != section {
				return nil
			}
			return melody
		})
		part(pr.name, pr.name+" left hand", 0.45, human, func(s S) []N {
			if s.Name != section {
				return nil
			}
			return arrange.PerBar(s, func(b int) []N {
				c := paradeChords[b%4]
				ps := c.Pitches(c.BassPitch(33))
				root, third, fifth := ps[0], ps[1], ps[2]
				var ns []N
				for i, pitch := range []float64{root, fifth, root + 12, third + 12, fifth + 12, third + 12, root + 12, fifth} {
					n := score.N(pitch, 0.5)
					n.Velocity = []float64{0.6, 0.4, 0.45, 0.4, 0.5, 0.4, 0.45, 0.4}[i]
					n.Hold = 2
					ns = append(ns, n)
				}
				return ns
			})
		})
	}

	// --- the arrangement: tempo per section and the sustain pedals ---

	// the pedal goes up at every bar line, just long enough to clear the old chord, and down again
	changed := func(bars int) arrange.Curve {
		return func(progress float64) float64 {
			at := progress * float64(bars)
			if at-math.Floor(at) < 0.05 {
				return 0
			}
			return 1
		}
	}
	secondHalf := func(progress float64) float64 {
		if progress < 0.5 {
			return 0
		}
		return 1
	}
	arrangement := arrange.NewArrangement(form).
		Tempo(arrange.Lane{Default: 66, Sections: map[string]arrange.Curve{
			keyboard: arrange.Const(72),
			pedal:    arrange.Const(80),
			sost:     arrange.Const(72),
			soft:     arrange.Const(66),
			pop:      arrange.Const(100),
			hymn:     arrange.Const(70),
			boogie:   arrange.Const(126),
			ambient:  arrange.Const(60),
			musicBox: arrange.Const(96),
		}}).
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{
			nocturne: changed(8),
			keyboard: arrange.Const(1),
			pedal: func(progress float64) float64 { // dry, half, down
				switch {
				case progress < 0.5:
					return 0
				case progress < 0.75:
					return 0.5
				}
				return 1
			},
			soft: changed(2),
		}}, pianos["grand"], "sustain").
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{soft: secondHalf}}, pianos["grand"], "soft").
		// the sostenuto goes down while the low C is held (its first beat) and up at the end
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{sost: func(progress float64) float64 {
			if progress > 0.03 && progress < 0.97 {
				return 1
			}
			return 0
		}}}, pianos["grand"], "sostenuto").
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{hymn: changed(4)}}, pianos["upright"], "sustain").
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{ambient: arrange.Const(1)}}, pianos["felt"], "sustain").
		PrintSections(true)
	for _, pr := range parade {
		arrangement.AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{pr.section: changed(4)}}, pianos[pr.name], "sustain")
	}
	arrangement.AddTo(p)

	fmt.Println("The piano")
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
