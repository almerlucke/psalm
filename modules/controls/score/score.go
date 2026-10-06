// Package score plays notes written in beats: notes, chords and rests with a length, how long
// they sound, a velocity and extra synth settings. A Sequencer plays a list of notes into a poly
// synth at the exact tick and loops, replacing a timer, a template and sequences of pitches and
// durations that have to be kept in step by hand
package score

import (
	"fmt"
	"strconv"
	"strings"
)

// Note is a note, a chord (more than one pitch) or a rest (no pitches)
type Note struct {
	Pitches  []float64      // MIDI pitches (or frequencies, see Config.Key)
	Beats    float64        // time until the next note
	Hold     float64        // beats the note sounds, 0 is Beats times the legato of the part
	Velocity float64        // sent as "velocity" when above 0, 0 is the part default
	Amp      float64        // scales the amplitude of the note, 0 is 1
	Params   map[string]any // extra values for the synth voice, for example "position" or "formant"
}

// IsRest returns true for a note without pitches
func (n Note) IsRest() bool {
	return len(n.Pitches) == 0
}

// N is a single note
func N(pitch float64, beats float64) Note {
	return Note{Pitches: []float64{pitch}, Beats: beats}
}

// Chord is a chord
func Chord(pitches []float64, beats float64) Note {
	return Note{Pitches: append([]float64(nil), pitches...), Beats: beats}
}

// Rest is a rest
func Rest(beats float64) Note {
	return Note{Beats: beats}
}

// Notes builds single notes of equal length, a pitch of 0 is a rest
func Notes(beats float64, pitches ...float64) []Note {
	ns := make([]Note, len(pitches))
	for i, p := range pitches {
		if p == 0 {
			ns[i] = Rest(beats)
		} else {
			ns[i] = N(p, beats)
		}
	}
	return ns
}

// Chords builds chords of equal length
func Chords(beats float64, chords ...[]float64) []Note {
	ns := make([]Note, len(chords))
	for i, c := range chords {
		ns[i] = Chord(c, beats)
	}
	return ns
}

// Join joins groups of notes
func Join(groups ...[]Note) []Note {
	var ns []Note
	for _, g := range groups {
		ns = append(ns, g...)
	}
	return ns
}

// Repeat repeats notes n times
func Repeat(notes []Note, n int) []Note {
	var ns []Note
	for range n {
		ns = append(ns, notes...)
	}
	return ns
}

// Beats returns the total length of notes in beats
func Beats(notes []Note) float64 {
	total := 0.0
	for _, n := range notes {
		total += n.Beats
	}
	return total
}

// Transpose returns the notes moved by semitones
func Transpose(notes []Note, semitones float64) []Note {
	out := make([]Note, len(notes))
	for i, n := range notes {
		out[i] = n
		out[i].Pitches = make([]float64, len(n.Pitches))
		for j, p := range n.Pitches {
			out[i].Pitches[j] = p + semitones
		}
	}
	return out
}

// With returns the notes with a change applied to every note, for example a velocity or a setting
func With(notes []Note, f func(n *Note)) []Note {
	out := make([]Note, len(notes))
	for i, n := range notes {
		out[i] = n
		f(&out[i])
	}
	return out
}

// Pitches returns the pitches moved by semitones, for building chords
func Pitches(pitches []float64, semitones float64) []float64 {
	out := make([]float64, len(pitches))
	for i, p := range pitches {
		out[i] = p + semitones
	}
	return out
}

var noteNames = map[byte]float64{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// Pitch returns the MIDI pitch of a note name like "C4" (60), "F#3", "Bb2" or "A-1", or of a
// plain number like "62"
func Pitch(name string) (float64, error) {
	if f, err := strconv.ParseFloat(name, 64); err == nil {
		return f, nil
	}
	if len(name) < 2 {
		return 0, fmt.Errorf("score: bad pitch %q", name)
	}
	base, ok := noteNames[strings.ToUpper(name[:1])[0]]
	if !ok {
		return 0, fmt.Errorf("score: bad note name %q", name)
	}
	rest := name[1:]
	for len(rest) > 0 && (rest[0] == '#' || rest[0] == 'b') {
		if rest[0] == '#' {
			base++
		} else {
			base--
		}
		rest = rest[1:]
	}
	octave, err := strconv.Atoi(rest)
	if err != nil {
		return 0, fmt.Errorf("score: bad octave in %q", name)
	}
	return base + float64(octave+1)*12, nil
}

// Parse reads notes from a compact notation, tokens separated by spaces:
//
//	A3:1        a note (name or MIDI number) lasting 1 beat
//	[D4 F4 A4]:2  a chord lasting 2 beats
//	r:1         a rest (also "-:1")
//	E4:1@.6     velocity 0.6
//	C5:1_3      lasting 1 beat, but sounding for 3
//	G4          without a length the note lasts as long as the one before (1 beat at the start)
//	|           bar lines are ignored, use them for reading
//
// Combine them as in "A3:.5_2@.9"
func Parse(notation string) ([]Note, error) {
	var (
		notes []Note
		beats = 1.0
	)
	tokens := tokenize(notation)
	for _, tok := range tokens {
		if tok == "|" {
			continue
		}
		var n Note

		// velocity, hold and length come after the pitch part
		head, vel, hasVel := strings.Cut(tok, "@")
		if hasVel {
			v, err := strconv.ParseFloat(vel, 64)
			if err != nil {
				return nil, fmt.Errorf("score: bad velocity in %q", tok)
			}
			n.Velocity = v
		}
		head, length, hasLength := cutLast(head, ":")
		if hasLength {
			length, hold, hasHold := strings.Cut(length, "_")
			b, err := strconv.ParseFloat(length, 64)
			if err != nil {
				return nil, fmt.Errorf("score: bad length in %q", tok)
			}
			beats = b
			if hasHold {
				h, err := strconv.ParseFloat(hold, 64)
				if err != nil {
					return nil, fmt.Errorf("score: bad hold in %q", tok)
				}
				n.Hold = h
			}
		}
		n.Beats = beats

		switch {
		case head == "r" || head == "-":
		case strings.HasPrefix(head, "["):
			inner := strings.TrimSuffix(strings.TrimPrefix(head, "["), "]")
			for _, name := range strings.Fields(inner) {
				p, err := Pitch(name)
				if err != nil {
					return nil, err
				}
				n.Pitches = append(n.Pitches, p)
			}
		default:
			p, err := Pitch(head)
			if err != nil {
				return nil, err
			}
			n.Pitches = []float64{p}
		}
		notes = append(notes, n)
	}
	return notes, nil
}

// MustParse parses notes and panics on an error
func MustParse(notation string) []Note {
	notes, err := Parse(notation)
	if err != nil {
		panic(err)
	}
	return notes
}

// tokenize splits on spaces, keeping chords in brackets together
func tokenize(s string) []string {
	var (
		tokens []string
		cur    strings.Builder
		depth  int
	)
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case r == '[':
			depth++
			cur.WriteRune(r)
		case r == ']':
			depth--
			cur.WriteRune(r)
		case (r == ' ' || r == '\t' || r == '\n') && depth == 0:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// cutLast cuts around the last separator outside brackets
func cutLast(s string, sep string) (string, string, bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 || i < strings.LastIndex(s, "]") {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}
