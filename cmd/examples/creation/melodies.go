package main

import (
	"slices"

	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
)

// --- melodies ---

// the theme of man, made in the image of God (day 6, over I vi IV V IV I)
var theme = score.MustParse("G4:2 Bb4:1 | C5:2 Eb5:1 | Eb5:1 D5 C5 | Bb4:2 D5:1 | C5:1 Bb4 Ab4 | G4:1 F4 Eb4")

// the horn answers: its fourth bar and fifth bar hold the figure of the cross (Bb G C Bb: the first
// and last note the beam, the second and third crossing it)
var answer = score.MustParse("Eb3:3 | C3:3 | Ab2:3 | Bb2:1 G2 C3 | Bb2:3 | Eb3:3")

// cross is the figure of the cross by itself, two bars
var cross = score.MustParse("Bb3:1 G3 C4 | Bb3:3")

// the moon (day 4, over vi IV I V vi V)
var moon = score.MustParse("Eb5:2 C5:1 | C5:2 Ab4:1 | Bb4:2 G4:1 | F4:2 Bb4:1 | C5:1 D5 Eb5 | D5:3")

// the canon of the plants, every phrase two bars over I and IV: the flute sings A B C, the oboe
// follows two bars later with A B, the clarinet two more with A, an octave down
var (
	seedA = score.MustParse("Bb4:1 G4 Eb4 | C5:2 Ab4:1")
	seedB = score.MustParse("G4:2 Bb4:1 | Ab4:1 C5 Eb5")
	seedC = score.MustParse("Eb5:2 D5:1 | C5:3")
)

// the Doxology: the Old Hundredth in E flat, a long first and last note in every line, three beats
// to the bar
var doxology = score.MustParse(
	"Eb4:3 | Eb4:1 D4 C4 | Bb3:1 Eb4 F4 | G4:3 | " + // Praise God, from whom all blessings flow
		"G4:3 | G4:1 G4 F4 | Eb4:1 Ab4 G4 | F4:3 | " + // praise him, all creatures here below
		"Eb4:3 | F4:1 G4 F4 | Eb4:1 C4 D4 | Eb4:3 | " + // praise him above, ye heavenly host
		"Bb4:3 | G4:1 Eb4 F4 | Ab4:1 G4 F4 | Eb4:3") // praise Father, Son, and Holy Ghost

// arp is six eighth notes up and down a chord
func arp(pitches []float64) []score.Note {
	ps := slices.Clone(pitches)
	slices.Sort(ps)
	order := []int{0, 1, 2, 3, 2, 1}
	var notes []score.Note
	for _, i := range order {
		notes = append(notes, score.N(ps[min(i, len(ps)-1)], 0.5))
	}
	return notes
}

// birdsong is a bar of quick calls on the tones of a chord
func birdsong(c harmony.Chord, alt bool) []score.Note {
	p := c.Close(72)
	p = append(p, p[0]+12)
	if alt {
		return score.Join(score.Notes(0.25, p[3], p[2], p[3], p[2]), score.Notes(1, 0), score.Notes(0.5, p[1], p[3]))
	}
	return score.Join(score.Notes(0.25, p[1], p[2]), score.Notes(0.5, p[3], 0), score.Notes(0.25, p[2], p[3]), score.Notes(1, p[1]))
}

// stars is a bar of slow twinkling high in a chord
func stars(c harmony.Chord, b int) []score.Note {
	p := c.Close(79)
	switch b % 3 {
	case 0:
		return score.Notes(1, p[2], 0, p[0])
	case 1:
		return score.Join(score.Notes(0.5, 0, p[1]), score.Notes(2, p[2]))
	}
	return score.Notes(1.5, p[0]+12, p[1])
}

// rest is a rest of a number of bars
func rest(bars float64) []score.Note { return []score.Note{score.Rest(bars * 3)} }
