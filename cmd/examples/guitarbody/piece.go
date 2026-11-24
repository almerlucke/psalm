package main

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/score"
)

// The piece: twenty bars of 4/4 in E minor, a minute at 80 beats per minute. A guitarist plays it with a thumb and
// fingers, then a hand: fingerpicking (the thumb on the low strings, the fingers on the high ones) for twelve bars,
// the melody over it from the fifth bar, strummed chords (down and up strokes) for the next four, a softer strum,
// and one chord rolled slowly at the end that rings out in the last bar.
//
// The feel is the player's: every part has its own timing (the melody lays back a little, the chords play on the beat), the
// notes of a chord are strummed (down strokes from the low string, up strokes from the high one, lighter), accents run
// through the bar, and the dynamics follow the phrases: each phrase of the melody swells to its middle and falls, the
// sections get louder as the piece goes and soften before the end.

const (
	pieceBPM  = 80
	pieceBars = 20
	pieceBeat = 4 * pieceBars
)

var (
	em = []float64{40, 47, 52, 55, 59, 64}
	c  = []float64{48, 52, 55, 60, 64}
	g  = []float64{43, 47, 50, 55, 59, 67}
	d  = []float64{50, 57, 62, 66}
	am = []float64{45, 52, 57, 60, 64}
	b7 = []float64{47, 51, 57, 59, 66}
)

// the chord of every bar, and how loud it is played (velocity)
var (
	harmony = [pieceBars][]float64{em, c, g, d, em, c, g, d, am, em, b7, em, em, c, g, d, am, b7, em, nil}
	loud    = [pieceBars]float64{
		0.42, 0.46, 0.5, 0.54, // the fingers alone, softly
		0.5, 0.5, 0.5, 0.5, // under the melody
		0.5, 0.55, 0.6, 0.55,
		0.78, 0.8, 0.8, 0.85, // strummed
		0.6, 0.5, // softer
		0.55, 0, // the chord that is rolled
	}
)

func rests(beats float64) score.Note { return score.Rest(beats) }

// pad checks that a part is as long as the piece (the parts loop together) and rests the beats that are left
func pad(name string, notes []score.Note) []score.Note {
	total := 0.0
	for _, n := range notes {
		total += n.Beats
	}
	if total > pieceBeat {
		panic(name + " is longer than the piece")
	}
	if total < pieceBeat {
		notes = append(notes, rests(pieceBeat-total))
	}
	return notes
}

// fingers is the right hand picking: the thumb plays the bass, then three fingers the high strings, eight notes to the bar
func fingers() []score.Note {
	var out []score.Note
	for bar := range 12 {
		ch := harmony[bar]
		n := len(ch)
		hi := [3]int{n - 3, n - 2, n - 1}
		for i, k := range [8]int{0, hi[0], hi[1], hi[2], 1, hi[2], hi[1], hi[0]} {
			note := score.N(ch[k], 0.5)
			note.Velocity = loud[bar]
			note.Hold = 3 // the strings are let ring
			if i == 0 {
				note.Hold = 4
				note.Velocity = min(loud[bar]+0.1, 1) // the thumb on the first beat
			}
			out = append(out, note)
		}
	}
	return pad("fingers", out)
}

// phrase sets the velocity of the notes of a phrase to an arch: soft at the start, a swell to the middle, soft at the end
func phrase(notes []score.Note, soft, peak float64) []score.Note {
	total := 0.0
	for _, n := range notes {
		total += n.Beats
	}
	pos := 0.0
	for i := range notes {
		x := (pos + notes[i].Beats/2) / total
		notes[i].Velocity = soft + (peak-soft)*(1-(2*x-1)*(2*x-1))
		notes[i].Hold = notes[i].Beats * 1.6 // a note rings a bit longer than it is long
		pos += notes[i].Beats
	}
	return notes
}

// melody is two phrases of four bars over the fingerpicking, the second one higher
func melody() []score.Note {
	n := score.N
	list := []score.Note{rests(16)}
	list = append(list, phrase([]score.Note{
		n(71, 1.5), n(67, 0.5), n(64, 2), // Em
		n(67, 1), n(64, 1), n(72, 1.5), n(71, 0.5), // C
		n(74, 1.5), n(71, 0.5), n(67, 2), // G
		n(66, 1), n(69, 1), n(74, 2), // D
	}, 0.5, 0.78)...)
	list = append(list, phrase([]score.Note{
		n(69, 1.5), n(72, 0.5), n(76, 2), // Am
		n(79, 1), n(76, 1), n(74, 1), n(71, 1), // Em
		n(75, 1.5), n(71, 0.5), n(66, 2), // B7
		n(64, 4), // Em, held
	}, 0.55, 0.9)...)
	return pad("melody", list)
}

// strummed are the down strokes (the chord, low string first) and the up strokes (the high strings only, lighter) of the
// bars that are strummed: a pattern of eight notes to the bar, D a down stroke, U an up stroke
func strummed() (down, up []score.Note) {
	patterns := map[int]string{12: "D.DU.UDU", 13: "D.DU.UDU", 14: "D.DU.UDU", 15: "D.DU.UDU", 16: "D...D.U.", 17: "D...D.U."}
	down, up = []score.Note{rests(48)}, []score.Note{rests(48)}
	for bar := 12; bar < 18; bar++ {
		ch := harmony[bar]
		for i, k := range patterns[bar] {
			dn, un := rests(0.5), rests(0.5)
			switch k {
			case 'D':
				dn = score.Chord(ch, 0.5)
				dn.Velocity = loud[bar]
				if i == 0 {
					dn.Velocity = min(loud[bar]+0.1, 1) // the first beat is the heavy one
				}
				dn.Hold = 2
			case 'U':
				un = score.Chord(ch[max(0, len(ch)-4):], 0.5)
				un.Velocity = loud[bar] * 0.7
				un.Hold = 1
			}
			down, up = append(down, dn), append(up, un)
		}
	}
	return pad("down", down), pad("up", up)
}

// finale is the last chord, rolled slowly from the low string to the high one, and let ring
func finale() []score.Note {
	ch := score.Chord(harmony[18], 4)
	ch.Velocity = loud[18]
	ch.Hold = 5.5 // it rings into the last bar
	return pad("finale", []score.Note{rests(72), ch})
}

// play is the piece, for the guitars
func play(p *psalm.Psalm, guitars ...psalm.Modular) {
	eighths := []float64{1, 0.6, 0.8, 0.6, 0.9, 0.6, 0.8, 0.6} // the accents through the bar
	down, up := strummed()
	for _, part := range []psalm.Modular{
		score.NewSequencer(fingers(), score.Config{
			Amp: 0.4, Loop: true, Length: pieceBeat,
			Feel: score.Feel{Timing: 5, Velocity: 0.08, Accents: eighths, AccentUnit: 0.5, BeatsPerBar: 4, Seed: 3},
		}).AddTo(p),
		score.NewSequencer(melody(), score.Config{
			Amp: 0.38, Loop: true, Length: pieceBeat,
			Feel: score.Feel{Timing: 8, Push: 9, Velocity: 0.06, Seed: 5}, // laid back a little
		}).AddTo(p),
		score.NewSequencer(down, score.Config{
			Amp: 0.33, Loop: true, Length: pieceBeat,
			Feel: score.Feel{Strum: 26, Timing: 6, Velocity: 0.08, Seed: 7},
		}).AddTo(p),
		score.NewSequencer(up, score.Config{
			Amp: 0.33, Loop: true, Length: pieceBeat,
			Feel: score.Feel{Strum: -16, Timing: 7, Velocity: 0.1, Seed: 9},
		}).AddTo(p),
		score.NewSequencer(finale(), score.Config{
			Amp: 0.3, Loop: true, Length: pieceBeat,
			Feel: score.Feel{Strum: 85, Timing: 4, Velocity: 0.04, Seed: 11},
		}).AddTo(p),
	} {
		for _, g := range guitars {
			g.ValueIn(part)
		}
	}
}
