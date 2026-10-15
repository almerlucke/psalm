// Step 3: bars and chords.
//
// Most parts follow the chords, and the chords change by the bar. Instead of writing every note
// out, write one bar at a time:
//
//	arrange.PerBar(s, func(bar int) []score.Note { ... })     every bar of a section (from 0)
//	arrange.Bars("Verse", func(s, bar int) []score.Note {...})  the same, for one section only
//
// A bar that returns nil is a rest: the bass below waits out the first bar of the verse.
//
// The harmony package knows chords and keys. A key turns roman numerals into chords
// (MustKey("C major").MustProgression("I vi IV V") is C Am F G), and a chord gives its notes in
// many shapes:
//
//	c.Close(60)           the chord in root position from C4 up: C E G
//	harmony.VoiceLead(v, c) the next chord for the same voices with the least movement, like a
//	                      pianist or a choir would sing it (C E G, then C E A, then C F A ...)
//	c.BassPitch(36)       the bass note from C2 up
//	c.Triad(48)           root, third and fifth from C3 up, for figures on the chord tones
//	harmony.Arp{...}      an arpeggio over the chord, up, down or both
//
//	go run ./cmd/examples/tutorial/03-bars
package main

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
)

func main() {
	p := tutorial.New(96)

	form := arrange.NewForm(
		arrange.Section{Name: "Intro", Bars: 4},
		arrange.Section{Name: "Verse", Bars: 8},
	)

	key := harmony.MustKey("C major")
	chords := key.MustProgression("I vi IV V") // C Am F G
	chordOf := func(bar int) harmony.Chord { return chords[bar%len(chords)] }

	// the voicings of the pad: the first chord close, every next one voice led from the one before
	voicings := [][]float64{chords[0].Close(60)}
	for i := 1; i < len(chords); i++ {
		voicings = append(voicings, harmony.VoiceLead(voicings[i-1], chords[i]))
	}

	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	keys := dx.NewPoly(dx.EPianoPreset(), 8, psalm.Stereo).AddTo(p)

	// the pad holds a voice led chord in every bar of every section
	form.Part(p, pad, "pad", score.Config{Amp: 0.12, Legato: 1}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			return []score.Note{score.Chord(voicings[bar%len(voicings)], 4)}
		})
	})

	// the bass plays in the verse only, from its second bar: the root, then the fifth
	form.Part(p, bass, "bass", score.Config{Amp: 0.35, Legato: 0.8}, arrange.Bars("Verse", func(s arrange.Section, bar int) []score.Note {
		if bar == 0 {
			return nil // a bar of rest
		}
		root := chordOf(bar).BassPitch(33)
		return score.Notes(1, root, 0, root+7, root+12) // 0 is a rest: root, rest, fifth, octave
	}))

	// the keys break the chord in the verse: up from the root through two octaves and back
	arp := harmony.Arp{Pattern: harmony.UpDown, Octaves: 2, Steps: 8}
	form.Part(p, keys, "arpeggio", score.Config{Amp: 0.25, Velocity: 0.5}, arrange.Bars("Verse", func(s arrange.Section, bar int) []score.Note {
		root, third, fifth := chordOf(bar).Triad(60)
		return arp.Notes([]float64{root, third, fifth})
	}))

	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)

	mix := mixer.New(p)
	mix.Channel("pad", pad, mixer.StripConfig{})
	mix.Channel("bass", bass, mixer.StripConfig{})
	mix.Channel("keys", keys, mixer.StripConfig{})
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
