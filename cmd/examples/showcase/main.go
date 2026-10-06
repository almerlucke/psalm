package main

// "Aurora": a slow orchestral piece in D minor, the showcase of psalm. Every synth plays a part
// of the orchestra and every effect has a role:
//
//	violins     wavetable strings on a generated sawtooth table, chorus, an LFO on the position
//	celli       subtractive strings
//	pizzicato   FM upright bass
//	harp        FM harp through freeverb and the granulator (frozen in the interlude)
//	flute       FM flute with a pitch shifted octave above, like a piccolo shadow
//	horns       DX soft brass chords
//	horn solo   FM french horn counter melody
//	soprano     pulsar vowel lead, the theme in the climax
//	choir       grains choir pad on a choir recording and a pulsar choir, in a convolved church
//	celesta     FM celesta through a pingpong delay into a shimmer reverb
//	bell pad    DX bell pad through a flanger, lined up with the vocoder by latency
//	voices      speech vocoding a wavetable pad
//	droplets    pulsar raindrops
//	aurora      wavetable drone on a generated random spectra table
//
// The strings run through a ladder lowpass that opens at dawn (the prelude) and closes at dusk
// (the coda), which slows down in its last four bars. The whole orchestra sits in an FDN hall and
// a soft clipper guards the master.
//
// The piece is built with the arrange and score packages: the form lists the sections, every part
// is written per section in beats (the melodies in score notation) and the arrangement automates
// the string filter, the granulator and the tempo. The piece loops, the schedule is printed at start

import (
	"fmt"
	"math"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/latency"
	"github.com/almerlucke/psalm/modules/effects/pan"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/sndfile"
)

const bpm = 66

// --- harmony ---

// chord holds the voicings of a chord for the sections of the orchestra
type chord struct {
	root    float64   // pizzicato bass
	celli   []float64 // root and fifth
	choir   []float64 // middle voices, also the horns an octave down
	violins []float64 // upper voices
}

var (
	Dm = chord{38, []float64{50, 57}, []float64{57, 62, 65}, []float64{62, 65, 69, 74}}
	Bb = chord{34, []float64{46, 53}, []float64{58, 62, 65}, []float64{62, 65, 70, 74}}
	Gm = chord{31, []float64{43, 50}, []float64{58, 62, 67}, []float64{62, 67, 70, 74}}
	A  = chord{33, []float64{45, 52}, []float64{57, 61, 64}, []float64{61, 64, 69, 73}}
	F  = chord{41, []float64{41, 48}, []float64{57, 60, 65}, []float64{60, 65, 69, 72}}
	C  = chord{36, []float64{48, 55}, []float64{55, 60, 64}, []float64{60, 64, 67, 72}}
)

var (
	// the progression, one chord per bar
	progression = []chord{Dm, Bb, Gm, A, F, C, Gm, A}
	// the coda comes home
	homecoming = []chord{Dm, Bb, F, C, Gm, A, Dm, Dm}
)

// chordAt returns the chord of a bar in a section
func chordAt(s arrange.Section, bar int) chord {
	if s.Name == "Coda" {
		return homecoming[bar%len(homecoming)]
	}
	return progression[bar%len(progression)]
}

// --- the form ---

var form = arrange.NewForm(
	arrange.Section{Name: "Prelude", Bars: 8, Dynamics: 0.9},
	arrange.Section{Name: "Theme", Bars: 16, Dynamics: 0.85},
	arrange.Section{Name: "Chorale", Bars: 8, Dynamics: 1.6},
	arrange.Section{Name: "Interlude", Bars: 8, Dynamics: 1.1},
	arrange.Section{Name: "Climax", Bars: 16, Dynamics: 0.95},
	arrange.Section{Name: "Coda", Bars: 8},
)

// --- melodies ---

// the theme, 16 bars over the progression twice
var themeBars = []string{
	"A4:2 F4:1 E4", "D4:3 F4:1", "G4:2 Bb4:1 A4", "E4:2 C#4:1 E4",
	"A4:2 C5:1 A4", "G4:2 E4:1 G4", "Bb4:1.5 A4:.5 G4:1 D5", "C#5:2 A4",
	"D5:2 C5:1 A4", "Bb4:2 A4:1 F4", "G4:1 A4 Bb4 D5", "C#5:3 E5:1",
	"F5:2 E5:1 C5", "E5:2 D5:1 C5", "Bb4:2 A4:1 G4", "A4:4",
}

func themeBar(i int) []score.Note {
	return score.MustParse(themeBars[i])
}

func theme() []score.Note {
	var notes []score.Note
	for i := range themeBars {
		notes = append(notes, themeBar(i)...)
	}
	return notes
}

// the horn counter melody of the climax, two half notes per bar
var counter = score.MustParse("F3:2 A3 | F3 D3 | D3 G3 | E3 C#3 | C3 F3 | E3 G3 | Bb3 A3 | A3 E3")

// harpArp is an 8th note arpeggio up and down the chord
func harpArp(c chord) []score.Note {
	tones := []float64{c.celli[0], c.celli[1], c.choir[0], c.choir[1], c.choir[2], c.choir[1], c.choir[0], c.celli[1]}
	notes := score.Notes(0.5, tones...)
	for i, v := range []float64{0.9, 0.6, 0.7, 0.6, 0.8, 0.6, 0.7, 0.5} {
		notes[i].Velocity = v
	}
	return notes
}

// softClip keeps the master below full scale: transparent below 0.7, smoothly rounding above
type softClip struct {
	*psalm.Module
}

func newSoftClip() *softClip {
	c := &softClip{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo})}
	c.SetSelf(c)
	return c
}

func (c *softClip) Synthesize() {
	const knee = 0.7
	ins := c.Input(0).Channels
	outs := c.Output(0).Channels
	for ch := range outs {
		for i, x := range ins[ch] {
			v := float64(x)
			if a := math.Abs(v); a > knee {
				v = math.Copysign(knee+(1-knee)*math.Tanh((a-knee)/(1-knee)), v)
			}
			outs[ch][i] = float32(v)
		}
	}
}

func main() {
	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// the buses
	var hall, stringBus, choirBus []any
	route := func(bus *[]any, m psalm.Modular) { *bus = append(*bus, m, 0, 0) }

	// --- strings ---

	violinsCfg := wavetable.StringsPreset(gen.Brightness(64))
	violinsCfg.Position = 0.8
	violinsCfg.Unison = 3 // the chorus widens the section
	violins := wavetable.NewPoly(violinsCfg, 16).AddTo(p)
	violins.ValueIn(lfo.New(true, 0.05, 0.7, 0.9, lfo.Sine).AddTo(p), 0, 1) // the bowing breathes
	violinsChorus := chorus.New(0.3, 0.3, 0.4, 0.1, 1.0, 0.35, nil).AddTo(p).In(violins)
	route(&stringBus, pan.New(-0.35).AddTo(p).In(violinsChorus))

	celli := subtractive.NewPoly(subtractive.StringsPreset(), 8).AddTo(p)
	route(&stringBus, pan.New(0.35).AddTo(p).In(celli))

	stringFilter := filters.NewLadder(ladder.LowPass12, 250, 0.1, 1.0)
	stringFilter.AddTo(p)
	stringFilter.In(stringBus...)
	route(&hall, stringFilter)

	pizz := fm.NewPoly(fm.UprightBassPreset(), 2).AddTo(p)
	route(&hall, pan.New(0.45).AddTo(p).In(pizz))

	// --- harp: freeverb, then the granulator ---

	harp := fm.NewPoly(fm.HarpPreset(), 12).AddTo(p)
	harpRoom := freeverb.New(0.5, 0.4, 1.0, 0.25, 0.9).AddTo(p).In(pan.New(0.55).AddTo(p).In(harp))
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
	route(&hall, gran)

	// --- winds and brass ---

	flute := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	piccolo := pitchshift.New(12, 60, 0.15).AddTo(p).In(flute) // an octave shadow above
	route(&hall, pan.New(-0.1).AddTo(p).In(piccolo))

	horns := dx.NewPoly(dx.SoftBrass(), 12, psalm.Stereo).AddTo(p)
	route(&hall, pan.New(-0.2).AddTo(p).In(horns))

	hornSolo := fm.NewPoly(fm.FrenchHornPreset(), 2).AddTo(p)
	route(&hall, pan.New(0.25).AddTo(p).In(hornSolo))

	// --- choir: grains on a choir recording, a pulsar choir and the soprano, in a church ---

	choirRec := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	choir := grains.NewPoly(grains.ChoirPad(choirRec, 60), 12).AddTo(p)
	route(&choirBus, choir)
	ooh := pulsar.NewPoly(pulsar.ChoirPreset(), 12).AddTo(p)
	route(&choirBus, ooh)
	soprano := pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
	route(&choirBus, soprano)

	choirClean := filters.NewSVF(svf.HighPass, 120, 0.1)
	choirClean.AddTo(p)
	choirClean.In(choirBus...)
	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	// large partitions keep the long church response cheap, their latency is a natural pre delay
	choirChurch := convolver.New(church, convolver.Config{PartitionSize: 4096, Mix: 0.4, Gain: 0.15}).AddTo(p).In(choirClean)

	// --- interlude colours: celesta, bell pad, vocoder, droplets ---

	celesta := fm.NewPoly(fm.CelestaPreset(), 8).AddTo(p)
	celestaEcho := pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.35, Mix: 0.3, HighCut: 5000, LowCut: 200}).AddTo(p).In(pan.New(0.3).AddTo(p).In(celesta))
	scfg := shimmer.DefaultConfig()
	scfg.Reverb.Decay = 7
	scfg.Shimmer = 0.45
	scfg.Mix = 0.45
	celestaSky := shimmer.New(scfg).AddTo(p).In(celestaEcho)

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
	route(&hall, voc)

	bellPad := dx.NewPoly(dx.BellPad(), 12, psalm.Stereo).AddTo(p)
	bellFlange := flanger.New(0.08, 0.5, 0.4, 0.4, nil).AddTo(p).In(bellPad)
	// the vocoder sounds an FFT frame late, the bell pad waits for it
	route(&hall, latency.New(psalm.Stereo, voc.Latency()).AddTo(p).In(bellFlange))

	droplets := pulsar.NewPoly(pulsar.RaindropsPreset(), 2).AddTo(p)
	route(&hall, droplets)

	// --- aurora: a drone on a table of random spectra, at dawn and dusk ---

	aurora := wavetable.NewPoly(wavetable.DronePreset(gen.Random(64, 6, 32, 11)), 4).AddTo(p)
	route(&hall, aurora)

	// --- the scores ---

	type S = arrange.Section
	type N = score.Note
	part := func(synth psalm.Modular, name string, amp float64, legato float64, sc arrange.Score) {
		form.Part(p, synth, name, score.Config{Amp: amp, Legato: legato, Velocity: 0.8}, sc)
	}

	part(violins, "violins", 0.12, 1.0, func(s S) []N {
		if !arrange.In(s, "Prelude", "Theme", "Climax", "Coda") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			c := chordAt(s, bar)
			if s.Name == "Coda" && bar == s.Bars-1 {
				return []N{{Pitches: c.violins, Beats: 4, Hold: 6}} // the last chord rings
			}
			return []N{score.Chord(c.violins, 4)}
		})
	})

	part(celli, "celli", 0.08, 1.0, func(s S) []N {
		if s.Name == "Interlude" {
			return arrange.PerBar(s, func(bar int) []N {
				return []N{{Pitches: chordAt(s, bar).celli[:1], Beats: 4, Velocity: 0.5}}
			})
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordAt(s, bar).celli, 4)} })
	})

	part(pizz, "pizzicato", 0.19, 0.5, func(s S) []N {
		if !arrange.In(s, "Theme", "Climax") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			r := chordAt(s, bar).root
			return score.Join(score.Notes(2, r), score.Notes(1, r+7, r+12))
		})
	})

	part(harp, "harp", 0.12, 1.6, func(s S) []N {
		if !arrange.In(s, "Prelude", "Theme", "Interlude", "Climax", "Coda") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			if (s.Name == "Prelude" && bar < 4) || (s.Name == "Coda" && bar == s.Bars-1) {
				return []N{score.Rest(4)} // the harp enters halfway the prelude and rests on the last chord
			}
			return harpArp(chordAt(s, bar))
		})
	})

	part(flute, "flute", 0.35, 0.95, func(s S) []N {
		if s.Name != "Theme" {
			return nil
		}
		return theme()
	})

	part(horns, "horns", 0.075, 0.95, func(s S) []N {
		if !arrange.In(s, "Chorale", "Climax") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			low := score.Pitches(chordAt(s, bar).choir, -12)
			if s.Name == "Chorale" {
				return score.Chords(2, low, low) // a chorale in half notes
			}
			return []N{score.Chord(low, 4)}
		})
	})

	part(hornSolo, "horn solo", 0.14, 0.95, func(s S) []N {
		if s.Name != "Climax" {
			return nil
		}
		return score.Repeat(counter, 2)
	})

	part(soprano, "soprano", 0.15, 0.95, func(s S) []N {
		if s.Name != "Climax" {
			return nil
		}
		return theme()
	})

	part(choir, "choir", 0.25, 1.0, func(s S) []N {
		if !arrange.In(s, "Prelude", "Chorale", "Climax", "Coda") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			c := chordAt(s, bar)
			if s.Name == "Prelude" {
				if bar%2 == 1 {
					return []N{score.Rest(4)}
				}
				return []N{{Pitches: c.choir, Beats: 4, Hold: 3.8}} // every other chord, within its bar
			}
			return []N{score.Chord(c.choir, 4)}
		})
	})

	part(ooh, "ooh", 0.045, 1.0, func(s S) []N {
		if !arrange.In(s, "Chorale", "Climax") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(score.Pitches(chordAt(s, bar).choir, 12), 4)} })
	})

	part(celesta, "celesta", 0.22, 0.9, func(s S) []N {
		switch s.Name {
		case "Interlude":
			return arrange.PerBar(s, func(bar int) []N {
				v := score.Pitches(chordAt(s, bar).violins, 12)
				if bar%2 == 0 {
					return score.Join(score.Notes(1, v[3]), score.Notes(0.5, v[2], v[1]), score.Notes(1, v[0], 0))
				}
				return score.Join(score.Notes(1, 0), score.Notes(0.5, v[1], v[2]), score.Notes(2, v[3]))
			})
		case "Coda":
			// the theme returns as a memory (the bars that fit Dm Bb F C), then the last chord
			memory := score.Join(themeBar(0), themeBar(1), themeBar(4), themeBar(5))
			memory = score.With(score.Transpose(memory, 12), func(n *N) { n.Velocity = 0.6 })
			return score.Join(memory, score.MustParse("r:4 D5:2 A4 F5 D5 r:4"))
		}
		return nil
	})

	part(bellPad, "bell pad", 0.05, 1.0, func(s S) []N {
		if s.Name != "Interlude" {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordAt(s, bar).choir, 4)} })
	})

	part(carrier, "voices", 0.1, 1.0, func(s S) []N {
		if s.Name != "Interlude" {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			c := chordAt(s, bar)
			return []N{score.Chord(append([]float64{c.celli[0]}, c.choir...), 4)}
		})
	})

	form.Part(p, droplets, "droplets", score.Config{Amp: 0.1, Legato: 1, Key: "frequency"}, func(s S) []N {
		if s.Name != "Interlude" {
			return nil
		}
		return score.Notes(16, 7, 9) // drops per second
	})

	part(aurora, "aurora", 0.17, 1.0, func(s S) []N {
		if !arrange.In(s, "Prelude", "Coda") {
			return nil
		}
		return []N{{Pitches: []float64{38, 45}, Beats: 32, Hold: 26}}
	})

	// --- the arrangement: dawn and dusk on the strings, the frozen harp of the interlude and a
	// ritardando over the last four bars ---

	arrangement := arrange.NewArrangement(form).
		Tempo(arrange.Lane{Sections: map[string]arrange.Curve{"Coda": func(progress float64) float64 {
			if progress < 0.5 {
				return bpm
			}
			return arrange.Smooth(bpm, 52)((progress - 0.5) * 2)
		}}}).
		Automate(arrange.Lane{Default: 12000, Sections: map[string]arrange.Curve{
			"Prelude": arrange.Exp(250, 12000),
			"Coda":    arrange.Exp(12000, 400),
		}}, stringFilter, 1).
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{"Interlude": arrange.Const(0.6)}}, gran, "mix").
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{"Interlude": func(progress float64) float64 {
			if progress >= 0.5 {
				return 1
			}
			return 0
		}}}, gran, "freeze").
		PrintSections(true)
	arrangement.AddTo(p)

	// --- the hall and the master ---

	hcfg := fdnreverb.DefaultConfig()
	hcfg.Decay = 3.8
	hcfg.Size = 0.8
	hcfg.Damping = 0.55
	hcfg.PreDelay = 30
	hcfg.Mix = 0.28
	orchestra := fdnreverb.New(hcfg).AddTo(p).In(hall...)

	master := newSoftClip()
	master.AddTo(p)
	master.In(orchestra, 0, 0, choirChurch, 0, 0, celestaSky, 0, 0)

	fmt.Println("Aurora")
	fmt.Print(arrangement.Schedule())

	p.Out(master)

	_ = p.Listen()
}
