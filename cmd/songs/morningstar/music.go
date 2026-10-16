package main

import (
	"slices"

	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
)

const bpm = 66

// --- harmony ---

// chord holds the voicings of a chord for the sections of the orchestra
type chord struct {
	symbol  harmony.Chord
	root    float64   // pizzicato bass
	celli   []float64 // root and fifth
	choir   []float64 // middle voices, also the horns an octave down
	violins []float64 // upper voices
}

var (
	Dm = chord{harmony.MustChord("Dm"), 38, []float64{50, 57}, []float64{57, 62, 65}, []float64{62, 65, 69, 74}}
	Bb = chord{harmony.MustChord("Bb"), 34, []float64{46, 53}, []float64{58, 62, 65}, []float64{62, 65, 70, 74}}
	Gm = chord{harmony.MustChord("Gm"), 31, []float64{43, 50}, []float64{58, 62, 67}, []float64{62, 67, 70, 74}}
	A  = chord{harmony.MustChord("A"), 33, []float64{45, 52}, []float64{57, 61, 64}, []float64{61, 64, 69, 73}}
	F  = chord{harmony.MustChord("F"), 41, []float64{41, 48}, []float64{57, 60, 65}, []float64{60, 65, 69, 72}}
	C  = chord{harmony.MustChord("C"), 36, []float64{48, 55}, []float64{55, 60, 64}, []float64{60, 64, 67, 72}}
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

// songChords lays out the chord of every bar of the song, for the harmony check
func songChords() []harmony.TimedChord {
	var tl []harmony.TimedChord
	for _, s := range form.Sections {
		for bar := range s.Bars {
			start := float64(s.StartBar+bar) * form.BeatsPerBar
			tl = append(tl, harmony.TimedChord{Beat: start, Beats: form.BeatsPerBar, Chord: chordAt(s, bar).symbol})
		}
	}
	return tl
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

// harpArp is an 8th note arpeggio up and down the chord, from the celli up through the choir voices
func harpArp(c chord) []score.Note {
	arp := harmony.Arp{Pattern: harmony.UpDown, Velocities: []float64{0.9, 0.6, 0.7, 0.6, 0.8, 0.6, 0.7, 0.5}}
	return arp.Notes(append(slices.Clone(c.celli), c.choir...))
}
