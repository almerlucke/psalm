package main

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
)

// brokenChord is a bar of a left hand breaking a chord in eighths, up and down from its root at or
// above lowest, every note held for some beats under the pedal
func brokenChord(c harmony.Chord, lowest, hold float64) []N {
	root, third, fifth := c.Triad(lowest)
	return score.Shape(score.Notes(0.5, root, fifth, root+12, third+12, fifth+12, third+12, root+12, fifth),
		score.Velocities(0.6, 0.4, 0.45, 0.4, 0.5, 0.4, 0.45, 0.4), score.Hold(hold))
}

// writeParts writes what every piano plays, section by section
func writeParts(p *psalm.Psalm, form *arrange.Form, pianos map[string]psalm.Modular) {
	part := func(instr, name string, amp float64, feel score.Feel, sc func(s S) []N) {
		form.Part(p, pianos[instr], name, score.Config{Amp: amp, Legato: 0.95, Velocity: 0.7, Feel: feel}, sc)
	}
	human := score.Feel{Timing: 8, Velocity: 0.08}

	// --- the nocturne: E flat, an arpeggiated left hand and a melody ---

	nocturneChords := harmony.MustProgression("Ebmaj7 Cm7 Abmaj7 Bb7 Ebmaj7 Gm7 Fm7 Eb")
	part("grand", "nocturne melody", 0.65, score.Feel{Timing: 12, Velocity: 0.08, Push: 10}, arrange.Only(nocturne, func(s S) []N {
		return score.MustParse(`
			G5:1.5@.7 F5:.5@.55 Eb5:1@.6 Bb4:1@.55 | C5:1.5@.65 D5:.5@.55 Eb5:1@.6 G5:1@.7 |
			F5:1@.65 Eb5:1@.6 C5:1.5@.6 Ab4:.5@.5 | Bb4:2@.6 D5:1@.6 F5:1@.65 |
			G5:1.5@.75 Bb5:.5@.7 Ab5:1@.7 G5:1@.65 | F5:2@.7 D5:1@.6 Bb4:1@.55 |
			Eb5:1@.6 F5:1@.65 Ab5:1@.7 C5:1@.55 | Eb5:4_6@.6`)
	}))
	part("grand", "nocturne left hand", 0.5, human, arrange.Bars(nocturne, func(_ S, b int) []N {
		return brokenChord(nocturneChords[b%len(nocturneChords)], 34, 3)
	}))

	// --- dynamics: the same chord eight times, from 0.1 to 1 ---

	part("grand", "dynamics", 0.11, score.Feel{}, arrange.Only(dynamics, func(s S) []N {
		ramp := make([]float64, 8)
		for i := range ramp {
			ramp[i] = 0.1 + 0.9*float64(i)/7
		}
		chord := score.Chord([]float64{39, 46, 55, 62, 67, 74}, 2)
		return score.Shape(score.Repeat([]N{chord}, 8), score.Velocities(ramp...), score.Hold(1.6))
	}))

	// --- the whole keyboard: E flat major up from E flat 1 and down, then the lowest and highest keys ---

	part("grand", "keyboard", 0.35, score.Feel{Timing: 4, Velocity: 0.05}, arrange.Only(keyboard, func(s S) []N {
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
	}))

	// --- pedal: short chords, dry and then held by the pedal ---

	pedalChords := harmony.MustProgression("Ab Bb Gm Cm")
	part("grand", "pedal", 0.3, human, arrange.Bars(pedal, func(_ S, b int) []N {
		c := pedalChords[b%len(pedalChords)]
		v := append([]float64{c.BassPitch(36)}, c.Close(60)...)
		return score.Shape(score.Repeat([]N{score.Chord(v, 1)}, 4), score.Hold(0.25))
	}))

	// --- sostenuto: a low C caught by the pedal, short chords above it stay dry ---

	part("grand", "sostenuto", 0.5, human, arrange.Only(sost, func(s S) []N {
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
	}))

	// --- the soft pedal: a phrase, then again una corda ---

	part("grand", "soft", 0.65, human, arrange.Only(soft, func(s S) []N {
		phrase := score.MustParse(`[48 55 64]:1_3@.6 E5:.5@.6 D5:.5@.55 C5:1@.6 G4:1@.5 | [41 57 60]:1_3@.6 A4:.5@.55 C5:.5@.6 F5:2@.65`)
		return score.Join(phrase, phrase)
	}))

	// --- bright: pumping 8th chords and root octaves ---

	popChords := harmony.MustProgression("G D Em C")
	popVoicings := [][]float64{popChords[0].Close(59)}
	for _, c := range popChords[1:] {
		popVoicings = append(popVoicings, harmony.VoiceLead(popVoicings[len(popVoicings)-1], c))
	}
	part("bright", "pop chords", 0.24, score.Feel{Timing: 5, Velocity: 0.06, Accents: []float64{1, 0.7, 0.85, 0.7}, AccentUnit: 0.5}, arrange.Bars(pop, func(_ S, b int) []N {
		return score.Shape(score.Repeat([]N{score.Chord(popVoicings[b%4], 0.5)}, 8), score.Hold(0.4))
	}))
	part("bright", "pop bass", 0.25, human, arrange.Bars(pop, func(_ S, b int) []N {
		r := popChords[b%4].BassPitch(31)
		return []N{score.Chord([]float64{r, r + 12}, 2), score.Chord([]float64{r, r + 12}, 1.5), score.Chord([]float64{r + 7, r + 19}, 0.5)}
	}))

	// --- upright: a hymn in four parts, half notes ---

	part("upright", "hymn", 0.47, score.Feel{Timing: 10, Spread: 6, Velocity: 0.06}, arrange.Only(hymn, func(s S) []N {
		return score.MustParse(`
			[41 57 60 69]:2@.6 [38 57 62 65]:2@.55 | [46 58 62 70]:2@.65 [48 55 64 67]:2@.6 |
			[45 57 60 72]:2@.7 [43 58 62 70]:2@.6 | [48 55 60 64]:2@.55 [41 57 60 65]:2_3@.5`)
	}))

	// --- honky tonk: a boogie left hand and swung stabs ---

	boogieRoots := []float64{36, 36, 41, 36}
	swing := score.Feel{Swing: 0.64, Timing: 6, Velocity: 0.08}
	part("honkyTonk", "boogie left hand", 0.33, swing, arrange.Bars(boogie, func(_ S, b int) []N {
		r := boogieRoots[b%4]
		return score.Notes(0.5, r, r+4, r+7, r+9, r+10, r+9, r+7, r+4)
	}))
	part("honkyTonk", "boogie stabs", 0.26, swing, arrange.Bars(boogie, func(_ S, b int) []N {
		r := boogieRoots[b%4] + 24
		stab := score.Chord([]float64{r + 4, r + 7, r + 10, r + 12}, 1)
		stab.Hold = 0.3
		return []N{score.Rest(0.5), stab, score.Rest(0.5), stab, {Beats: 1}}
	}))

	// --- felt: slow arpeggios in the middle of the keyboard, the pedal down ---

	feltChords := harmony.MustProgression("Dmaj7 Bm7 Gmaj7 A")
	part("felt", "ambient", 0.52, score.Feel{Timing: 15, Velocity: 0.1}, arrange.Bars(ambient, func(_ S, b int) []N {
		c := feltChords[b%4]
		ns := harmony.Arp{Pattern: harmony.UpDown, Octaves: 2, Steps: 8, Velocities: []float64{0.5, 0.35, 0.4, 0.35}}.Notes(c.Close(62))
		return append([]N{score.Chord([]float64{c.BassPitch(38)}, 0)}, ns...)
	}))

	// --- toy: a music box tune ---

	part("toy", "music box", 0.27, score.Feel{Timing: 4, Velocity: 0.05}, arrange.Only(musicBox, func(s S) []N {
		return score.MustParse(`
			E6:.5 G6 C7:1 B6:.5 G6 A6:1 | F6:.5 A6 D7:1 C7:.5 A6 B6:1 |
			G6:.5 B6 E7:1 D7:.5 B6 C7:.5 A6 | G6:1 E6:1 C6:2`)
	}))
	part("toy", "music box bass", 0.19, score.Feel{}, arrange.Only(musicBox, func(s S) []N {
		return score.MustParse(`[48 55]:2 [48 55] | [41 53] [43 55] | [43 55] [45 52] | [43 53] [48 55]`)
	}))

	// --- the parade: one phrase in A minor on every other preset ---

	paradeChords := harmony.MustProgression("Am F C G")
	melody := score.MustParse(`
		E5:1.5@.65 D5:.5@.55 C5:1@.6 B4:1@.55 | A4:1.5@.6 C5:.5@.55 F5:2@.7 |
		E5:1@.6 G5:1@.65 C5:2@.6 | B4:1@.55 D5:1@.6 G5:2_3@.65`)
	for _, pr := range parade {
		section := pr.section
		part(pr.name, pr.name+" melody", 0.6, score.Feel{Timing: 12, Velocity: 0.08, Push: 8}, arrange.Only(section, func(s S) []N {
			return melody
		}))
		part(pr.name, pr.name+" left hand", 0.45, human, arrange.Bars(section, func(_ S, b int) []N {
			return brokenChord(paradeChords[b%4], 33, 2)
		}))
	}
}
