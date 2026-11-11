package harmony

import (
	"fmt"
	"math"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/score"
)

// TimedChord is a chord at a time in beats
type TimedChord struct {
	Beat  float64
	Beats float64
	Chord Chord
}

// Timeline lays out chords of equal length, one after the other
func Timeline(beatsPerChord float64, chords ...Chord) []TimedChord {
	tl := make([]TimedChord, len(chords))
	for i, c := range chords {
		tl[i] = TimedChord{Beat: float64(i) * beatsPerChord, Beats: beatsPerChord, Chord: c}
	}
	return tl
}

// Issue is a note that clashes with its chord
type Issue struct {
	Beat   float64 // where the clash starts, in beats from the start of the part
	Pitch  float64
	Chord  Chord
	Reason string
}

func (i Issue) String() string {
	return fmt.Sprintf("beat %g: %s against %s, %s", i.Beat, pitchName(i.Pitch), i.Chord.Name, i.Reason)
}

// tolerance is how long a note may ring into the next chord before it counts, in beats: the release
// of a note played up to the chord change
const tolerance = 0.25

// Check finds the notes of a part that clash with the chords under them. A note clashes when it
// is a semitone above a chord tone (an "avoid note": C# over C, F over C major's E), unless it is a
// passing or neighbour note: a short note (at most a beat) of a single line, reached or left by a
// step. Other notes outside the chord (sevenths, ninths, sixths) are colour, not clashes; write
// them in the chord symbol (Dm9) when the chord should hold them. A note held into the next chord
// also clashes when it ends up a semitone below a chord tone. When the part is longer than the
// timeline, the timeline repeats
func Check(notes []score.Note, timeline []TimedChord) []Issue {
	if len(timeline) == 0 {
		return nil
	}
	length := timeline[len(timeline)-1].Beat + timeline[len(timeline)-1].Beats

	// single notes of the part with their starts, to find steps around a note
	type played struct {
		start, end float64
		pitches    []float64
	}
	var ps []played
	beat := 0.0
	for _, n := range notes {
		if !n.IsRest() {
			hold := n.Hold
			if hold == 0 {
				hold = n.Beats
			}
			ps = append(ps, played{beat, beat + hold, n.Pitches})
		}
		beat += n.Beats
	}

	var issues []Issue
	for i, p := range ps {
		single := len(p.pitches) == 1
		step := false
		if single {
			if i > 0 && len(ps[i-1].pitches) == 1 && math.Abs(ps[i-1].pitches[0]-p.pitches[0]) <= 2 {
				step = true
			}
			if i+1 < len(ps) && len(ps[i+1].pitches) == 1 && math.Abs(ps[i+1].pitches[0]-p.pitches[0]) <= 2 {
				step = true
			}
		}

		for _, seg := range chordsBetween(timeline, length, p.start, p.end) {
			held := seg.start > p.start // the note was already sounding when this chord came
			for _, pitch := range p.pitches {
				if seg.chord.Contains(pitch) || !rubs(pitch, seg.chord, held) {
					continue
				}
				if !held && step && p.end-p.start <= 1 {
					continue // a passing or neighbour note
				}
				reason := "a semitone above a chord tone"
				if held {
					reason = "held from the chord before"
				}
				issues = append(issues, Issue{Beat: seg.start, Pitch: pitch, Chord: seg.chord, Reason: reason})
			}
		}
	}
	return issues
}

// segment is the part of a note under one chord
type segment struct {
	start float64
	chord Chord
}

// chordsBetween returns the chords under a note from start to end (the timeline repeating),
// leaving out a chord the note only rings into briefly
func chordsBetween(timeline []TimedChord, length, start, end float64) []segment {
	var segs []segment
	loop := math.Floor(start/length) * length
	for off := loop; off < end; off += length {
		for _, tc := range timeline {
			cs, ce := off+tc.Beat, off+tc.Beat+tc.Beats
			if ce <= start || cs >= end {
				continue
			}
			from := max(cs, start)
			if from > start && end-from <= tolerance {
				continue
			}
			segs = append(segs, segment{from, tc.Chord})
		}
	}
	return segs
}

// rubs returns true when a pitch is a semitone above a tone of the chord, or (for a note held
// over from the chord before) a semitone from it either way
func rubs(pitch float64, c Chord, held bool) bool {
	pc := pitchClass(pitch)
	for _, t := range c.PitchClasses() {
		if d := mod12(pc - t); d == 1 || (held && d == 11) {
			return true
		}
	}
	return false
}

// Report returns the issues as text, one per line, or "" when there are none
func Report(name string, issues []Issue) string {
	var b strings.Builder
	for _, i := range issues {
		fmt.Fprintf(&b, "%s: %s\n", name, i)
	}
	return b.String()
}

// pitchName returns a note name like "C#4"
func pitchName(p float64) string {
	n := int(math.Round(p))
	return sharpNames[mod12(n)] + fmt.Sprint(n/12-1)
}
