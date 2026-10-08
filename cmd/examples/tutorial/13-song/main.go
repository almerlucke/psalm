// Step 13: a whole song.
//
// Everything from the tutorial together in one short song, each step where it belongs:
//
//	the form (step 2)          intro, verse, chorus, outro, the chorus louder (its dynamics)
//	bars and chords (3)        a progression in C, voice led, written bar by bar
//	feel (4)                   accents shaped into the keys, players' timing on every part
//	harmony check (5)          every written part checked against the chords at the start
//	automation (6)             a filter opening over the intro
//	curves at bars (7)         the pad fading in, the pedal of the piano, the waves of the outro
//	tempo (8)                  a ritardando over the last two bars
//	events (9)                 bells at the start of the chorus
//	the mixer (10)             a hall, a group with the filter, a channel per instrument
//	streams (11)               glass sparkles in the chorus, never the same twice
//	tuning (12)                in just intonation
//
//	go run ./cmd/examples/tutorial/13-song
//	go run ./cmd/examples/tutorial/13-song -render song.wav -seed 3
package main

import (
	"fmt"

	"github.com/almerlucke/genny/function"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/sndfile"
)

type (
	S = arrange.Section
	N = score.Note
)

func main() {
	p := tutorial.New(92)

	// --- the form and the harmony ---

	form := arrange.NewForm(
		S{Name: "Intro", Bars: 4},
		S{Name: "Verse", Bars: 8},
		S{Name: "Chorus", Bars: 8, Dynamics: 1.15},
		S{Name: "Outro", Bars: 4},
	)
	key := harmony.MustKey("C major")
	chords := key.MustProgression("I vi IV V")
	chordOf := func(bar int) harmony.Chord { return chords[bar%4] }
	voicings := [][]float64{chords[0].Close(55)}
	for i := 1; i < 4; i++ {
		voicings = append(voicings, harmony.VoiceLead(voicings[i-1], chords[i]))
	}
	tune := key.Just(440)

	// the arrangement comes first in the patch: its events (the chord of the bar, below) happen
	// before the parts play the bar. Its lanes are added further down, once the mixer is there
	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)

	// --- the instruments ---

	felt := piano.NewPoly(piano.FeltPreset(), 16).AddTo(p)
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	lead := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	glass := pulsar.NewPoly(pulsar.GlassPreset(), 8).AddTo(p)
	recording := func(file string, loop bool) *player.Player {
		pl := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32](file, 4), Loop: loop, Speed: 1, Amp: 0.5})
		pl.AddTo(p)
		return pl
	}
	bells := recording("assets/sounds/various/mixkit-sleigh-bells-584.wav", false)
	waves := recording("assets/sounds/various/boat_waves-6099.wav", true)

	// --- the written parts, each checked against the chords ---

	timeline := harmony.Timeline(4, chords...)
	clashes := 0
	part := func(synth psalm.Modular, name string, amp float64, feel score.Feel, sc arrange.Score) *score.Sequencer {
		issues := harmony.Check(form.Compile(name, sc), timeline)
		clashes += len(issues)
		fmt.Print(harmony.Report(name, issues))
		return form.Part(p, synth, name, score.Config{Amp: amp, Velocity: 0.6, Tuning: tune, Feel: feel}, sc)
	}
	played := score.Feel{Timing: 8, Velocity: 0.08}

	// the felt piano: a chord a beat, its pedal changed every bar (below); everywhere but the chorus
	part(felt, "piano", 0.35, played, func(s S) []N {
		if s.Name == "Chorus" {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			v := append([]float64{chordOf(bar).BassPitch(36)}, voicings[bar%4]...)
			if s.Name == "Outro" && bar == 3 {
				return []N{score.Chord(v, 4)} // the last chord
			}
			return score.Shape(score.Repeat([]N{score.Chord(v, 1)}, 4), score.Velocities(0.6, 0.4, 0.5, 0.4), score.Hold(0.4))
		})
	})
	part(pad, "pad", 0.15, score.Feel{Spread: 20}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(voicings[bar%4], 4)} })
	})
	// the keys and the bass: the groove of the verse and the chorus
	inGroove := func(s S) bool { return arrange.In(s, "Verse", "Chorus") }
	part(keys, "keys", 0.14, played, func(s S) []N {
		if !inGroove(s) {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			c := score.Chord(score.Pitches(voicings[bar%4], 12), 0.5)
			return score.Shape(score.Repeat([]N{c}, 8), score.Velocities(0.7, 0.3, 0.5, 0.3), score.Hold(0.3))
		})
	})
	part(bass, "bass", 0.25, played, func(s S) []N {
		if !inGroove(s) {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			r := chordOf(bar).BassPitch(36)
			return score.Shape(score.Notes(1, r, 0, r+7, r+12), score.Velocities(0.8, 0, 0.6, 0.5))
		})
	})
	// the melody: one for the verse, another for the chorus
	part(lead, "melody", 0.3, score.Feel{Timing: 10, Push: 6, Velocity: 0.06}, func(s S) []N {
		switch s.Name {
		case "Verse":
			return score.MustParse(`E5:1 G5 C6:2 | E5:1 A5 C6:2 | A5:1 F5 C5:2 | D5:1 G5 B4:2 |
				E5:1 G5 C6:1.5 B5:.5 | A5:1 E5 C5:2 | F5:1 A5 C6:2 | B5:1 A5 G5:2`)
		case "Chorus":
			return score.MustParse(`G5:2@.75 E5:1 C5 | A5:2@.75 E5:1 C5 | F5:1 A5 C6:2@.8 | B5:1 A5 G5:2 |
				G5:2@.75 E5:1 C5 | A5:2@.75 C6:1 A5 | F5:1 A5 C6:1 D6 | B5:2 G5:2`)
		}
		return nil
	})

	if clashes == 0 {
		fmt.Println("harmony check: every part fits its chords")
	}

	// --- a stream: glass sparkling over the chorus, on the tones of the chord of the bar ---

	chord := chords[0]
	sparkle := &arrange.Stream{
		Rhythm: function.New(func() float64 { return []float64{0.25, 0.5, -0.5, -0.25}[psalm.Rand().IntN(4)] }),
		Message: template.Template{"pitch": function.New(func() float64 {
			tones := chord.Close(84)
			return tones[psalm.Rand().IntN(len(tones))]
		})},
		Amp:    0.18,
		Legato: 0.6,
	}
	form.Stream(p, glass, "glass", func(s S) *arrange.Stream {
		if s.Name != "Chorus" {
			return nil
		}
		return sparkle
	})

	// --- the mixer: a hall, the pad and the piano through a filter ---

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay = 2.8
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	filter := filters.NewLadder(ladder.LowPass12, 400, 0.2, 1.0)
	filter.AddTo(p)
	mix.Group("soft", mixer.StripConfig{Sends: map[string]float64{"hall": 0.4}}, filter)

	mix.Channel("piano", felt, mixer.StripConfig{Group: "soft", Pan: -0.2})
	padChannel := mix.Channel("pad", pad, mixer.StripConfig{Group: "soft", Volume: arrange.Silent, Pan: 0.2})
	mix.Channel("keys", keys, mixer.StripConfig{Pan: -0.3, Sends: map[string]float64{"hall": 0.2}})
	mix.Channel("bass", bass, mixer.StripConfig{})
	mix.Channel("melody", lead, mixer.StripConfig{Pan: 0.1, Sends: map[string]float64{"hall": 0.3}})
	mix.Channel("glass", glass, mixer.StripConfig{Volume: -4, Pan: 0.4, Sends: map[string]float64{"hall": 0.5}})
	mix.Channel("bells", bells, mixer.StripConfig{Volume: -6, Sends: map[string]float64{"hall": 0.4}})
	wavesChannel := mix.Channel("waves", waves, mixer.StripConfig{Volume: arrange.Silent})

	// --- the arrangement ---

	intro, outro := form.Span("Intro"), form.Span("Outro")
	arrangement.
		// the chord of the bar, for the stream
		OnBar(func(pos arrange.Position) { chord = chordOf(pos.SectionBar()) }).
		// the filter opens over the intro
		AutomateMessage(arrange.Sections{"Intro": arrange.Exp(400, 12000)}.Else(12000), filter, "cutoff").
		// the pad comes in from the second bar and stays (a bar past the song's end: no fade out)
		AutomateMessage(arrange.Sections{"Intro": intro.Fade(-4, 1, 5)}.Else(-4), padChannel, "volume").
		// the piano's pedal changes every bar
		AutomateMessage(arrange.Sections{"Intro": intro.Pedal(0, 4), "Verse": form.Span("Verse").Pedal(0, 8), "Outro": outro.Pedal(0, 4)}, felt, "sustain").
		// bells at the start of the chorus, waves in the outro
		At("Chorus", 0, bells.Bang).
		AutomateMessage(arrange.Sections{"Outro": outro.Fade(-8, 0, 5)}.Else(arrange.Silent), wavesChannel, "volume").
		// a ritardando over the last two bars
		Tempo(arrange.Sections{"Outro": outro.Last(2, arrange.Smooth(92, 60))})

	fmt.Print(arrangement.Schedule())
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
