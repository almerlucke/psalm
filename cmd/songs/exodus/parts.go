package main

import (
	"slices"

	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
)

type (
	S = arrange.Section
	N = score.Note
)

// how every part plays: ensembles never start a chord exactly together, soloists breathe
var feels = map[string]score.Feel{
	"contrabass":        {Timing: 8, Velocity: 0.05},
	"alto flute":        {Timing: 10, Push: 5, Velocity: 0.08},
	"clarinets":         {Timing: 8, Spread: 12, Velocity: 0.06},
	"oud":               {Timing: 8, Velocity: 0.1},
	"harp":              {Timing: 5, Velocity: 0.1},
	"bansuri":           {Timing: 10, Push: 6, Velocity: 0.08},
	"recorder":          {Timing: 8, Push: 4, Velocity: 0.08},
	"bass clarinet":     {Timing: 10, Push: 6, Velocity: 0.08},
	"shakuhachi":        {Timing: 12, Push: 8, Velocity: 0.08},
	"frame drum":        {Timing: 6, Velocity: 0.12},
	"taiko":             {Timing: 4, Velocity: 0.08},
	"bodhran":           {Timing: 5, Velocity: 0.12},
	"floor tom":         {Timing: 5, Velocity: 0.1},
	"tambourine":        {Timing: 5, Velocity: 0.12},
	"piano":             {Timing: 6, Strum: 14, Velocity: 0.06},
	"organ (cathedral)": {Timing: 4, Spread: 6},
	"organ (soft)":      {Timing: 4, Spread: 6},
	"organ (plenum)":    {Timing: 4, Spread: 6},
	"organ (strings)":   {Timing: 4, Spread: 6},
	"tubular bells":     {Timing: 6},
	"handbells":         {Strum: 30, Velocity: 0.08},
	"vibraphone":        {Timing: 6, Velocity: 0.1},
	"bowed vibraphone":  {Timing: 10, Spread: 15},
	"kalimba":           {Timing: 15, Velocity: 0.15},
	"glockenspiel":      {Timing: 6, Velocity: 0.12},
}

// perBar writes a section bar by bar with the chord and voicings of every bar; a bar without notes
// (nil) is a rest
func perBar(s S, f func(b int, br bar) []N) []N {
	return arrange.PerBar(s, func(b int) []N { return f(b, at(s, b)) })
}

func rest(bars float64) []N { return []N{score.Rest(bars * form.BeatsPerBar)} }

// from plays f from a bar of the section on (before it a rest)
func from(first int, f func(b int, br bar) []N) func(b int, br bar) []N {
	return func(b int, br bar) []N {
		if b < first {
			return nil
		}
		return f(b, br)
	}
}

// within plays f in the bars first to last - 1
func within(first, last int, f func(b int, br bar) []N) func(b int, br bar) []N {
	return func(b int, br bar) []N {
		if b < first || b >= last {
			return nil
		}
		return f(b, br)
	}
}

// hold is a chord for a bar
func hold(p []float64) []N { return []N{{Pitches: p, Beats: 4, Hold: 4}} }

// held is a chord for a bar at an amplitude
func held(p []float64, amp float64) []N { return []N{{Pitches: p, Beats: 4, Hold: 4, Amp: amp}} }

// roll is a bar of one chord struck again and again, from slow and soft to fast and hard (speeding up
// by the factor accel from the first stroke to the last), the strokes left to ring on their own
func roll(p []float64, strokes int, accel, from, to float64) []N {
	w := make([]float64, strokes)
	sum := 0.0
	for i := range w {
		w[i] = 1 - (1-1/accel)*float64(i)/float64(max(strokes-1, 1))
		sum += w[i]
	}
	var notes []N
	for i := range w {
		beats := 4 * w[i] / sum
		notes = append(notes, N{Pitches: p, Beats: beats, Hold: min(beats, 0.15), Velocity: from + (to-from)*float64(i)/float64(max(strokes-1, 1))})
	}
	return notes
}

// arp8 is a bar of eighth notes up and down the notes of a chord
func arp8(p []float64) []N {
	ps := slices.Clone(p)
	slices.Sort(ps)
	var notes []N
	for _, i := range []int{0, 1, 2, 1, 0, 1, 2, 1} {
		notes = append(notes, score.N(ps[min(i, len(ps)-1)], 0.5))
	}
	return notes
}

// sweep is a run of n notes of the chord's tones from a pitch to another (up or down), within beats
func sweep(c harmony.Chord, from, to float64, n int, beats float64) []N {
	var tones []float64
	lo, hi := min(from, to), max(from, to)
	for p := lo; p <= hi; p++ {
		if c.Contains(p) {
			tones = append(tones, p)
		}
	}
	var notes []N
	for i := range n {
		k := i * (len(tones) - 1) / max(n-1, 1)
		if from > to {
			k = len(tones) - 1 - k
		}
		notes = append(notes, score.N(tones[k], beats/float64(n)))
	}
	return notes
}

// strokes plays a note on the given beats of a bar (the rest silent)
func strokes(pitch float64, beats ...float64) []N {
	var notes []N
	at := 0.0
	for _, b := range beats {
		if b > at {
			notes = append(notes, score.Rest(b-at))
		}
		notes = append(notes, score.N(pitch, 1))
		at = b + 1
	}
	if at < 4 {
		notes = append(notes, score.Rest(4-at))
	}
	return notes
}

// --- the tunes ---

// the lament of the slaves in Egypt (12 bars over Dm Dm Gm Dm Bb A Dm Gm Eb A Dm A)
var lament = score.MustParse("A4:2 F4:1 G4:1 | A4:3 D5:1 | Bb4:2 A4:1 G4:1 | F4:3 D4:1 | D5:2 C5:1 Bb4:1 | C#5:1 A4 E5:2 | D5:3 E5:1 | D5:2 Bb4:2 | Eb5:2 D5:1 Bb4:1 | C#5:2 A4:2 | D5:4 | E5:4")

// the march of the exodus (16 bars)
var march = score.MustParse("A4:1 A4 D5 F5 | E5:2 D5:1 C5:1 | C5:1 C5 F5 A5 | G5:2 F5:1 E5:1 | D5:1 D5 F5 Bb5 | A5:2 G5:1 F5:1 | E5:1 E5 G5 C6 | G5:2 E5:1 D5:1 | " +
	"A4:1 A4 D5 F5 | G5:2 F5:1 D5:1 | A5:1 G5 F5 C5 | E5:2 G5:2 | Bb5:1 A5 G5 F5 | A5:2 C6:2 | C6:2 G5:2 | F5:2 D5:2")

// the wonder at the sea, the bansuri over the first eight bars
var wonder = score.MustParse("D5:4 | Eb5:3 D5:1 | D5:2 F5:2 | A4:2 C5:2 | D5:4 | Eb5:2 G5:2 | A5:2 F5:2 | E5:4")

// and the crossing on dry ground, the last four
var crossing = score.MustParse("A4:2 D5:2 | D5:3 C5:1 | C5:2 A4:2 | C#5:2 E5:2")

// Miriam's song (12 bars over F C Bb F Dm Gm C F F Bb C F)
var song = score.MustParse("F5:1 A5 C6 A5 | G5:1 E5 G5 C6 | D6:1 C6 Bb5 A5 | A5:2 G5:1 F5:1 | D6:1 A5 F5 A5 | Bb5:1 G5 Bb5 D6 | C6:1 G5 E5 G5 | F5:2 A5:2 | " +
	"F5:1 A5 C6 F6 | D6:2 Bb5:2 | G5:1 C6 E6:1 C6 | F6:4")

// the wanderer in the wilderness (12 bars over Dm Am C Dm Gm Dm Bb C Dm Am Bb A)
var wanderer = score.MustParse("A4:3 D5:1 | C5:2 E5:2 | G5:2 E5:2 | F5:2 D5:2 | D5:3 Bb4:1 | A4:2 F4:2 | Bb4:2 D5:2 | C5:2 G4:2 | A4:3 D5:1 | E5:2 C5:2 | D5:2 Bb4:2 | C#5:4")

// the lament again after forty years (8 bars over Dm Am Bb A Dm Gm A Dm)
var memory = score.MustParse("A4:3 D5:1 | C5:2 E5:2 | D5:2 Bb4:2 | C#5:2 E5:2 | D5:3 F5:1 | D5:2 Bb4:2 | E5:2 C#5:2 | D5:4")

// the lament in the promised land, in D major (16 bars over Dm Gm A A D G A D Bm Em A D G A D D)
var promised = score.MustParse("A4:2 D5:2 | D5:2 Bb4:2 | C#5:2 E5:2 | E5:4 | F#5:1 A5 D6:2 | B5:2 G5:2 | A5:2 C#6:2 | D6:4 | " +
	"F#5:2 B5:2 | E5:1 G5 B5:2 | A5:1 C#6 E6:2 | D6:2 A5:2 | B5:2 D6:2 | C#6:2 A5:2 | D6:4 | D6:4")

// plays holds what a part plays in every section
type plays map[string]func(S) []N

func (m plays) score() arrange.Score {
	return func(s S) []N {
		if f, ok := m[s.Name]; ok {
			return f(s)
		}
		return nil
	}
}

// bar plays f in every bar of a section
func every(f func(b int, br bar) []N) func(S) []N {
	return func(s S) []N { return perBar(s, f) }
}

// tune plays notes written for the whole section
func tune(ns []N) func(S) []N { return func(S) []N { return ns } }

// slice takes the bars first to last - 1 of a tune written in whole bars
func slice(ns []N, first, last int) []N {
	var out []N
	at := 0.0
	for _, n := range ns {
		if at >= float64(first)*4-1e-9 && at < float64(last)*4-1e-9 {
			out = append(out, n)
		}
		at += n.Beats
	}
	return out
}

// tuneBars plays the bars first to last - 1 of a tune written for a section of bars bars, in
// their place in it (a rest before and after)
func tuneBars(ns []N, first, last, bars int) func(S) []N {
	return func(S) []N {
		out := slice(ns, first, last)
		if first > 0 {
			out = score.Join(rest(float64(first)), out)
		}
		if last < bars {
			out = score.Join(out, rest(float64(bars-last)))
		}
		return out
	}
}
