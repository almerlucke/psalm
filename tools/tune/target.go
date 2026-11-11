package tune

import (
	"math"
	"slices"
)

// Allowed says which notes the voice may be on at a time (seconds): the pitch classes of the chord or the scale there, 0 to 11
type Allowed func(seconds float64) []int

// Mode is how the voice is brought to the notes
type Mode int

const (
	// Snap puts every moment of the voice on the nearest allowed note: the voice moves as it did, in steps of the notes (the effect of a pitch corrector)
	Snap Mode = iota
	// Chant puts a stretch of the voice (a word or a few) on one allowed note near where the voice speaks, and keeps some of how it rises and falls: a voice that recites on the
	// notes of the chord, like the tone of a psalm that is chanted
	Chant
)

// Options is how a voice is brought to the notes
type Options struct {
	Mode      Mode
	Allowed   Allowed
	Speed     float64 // milliseconds the voice takes to move to its note (0: at once, a hard correction; 100 to 300 is softer)
	Keep      float64 // for Chant: how much of the rise and fall of the voice stays, 0 to 1
	Strength  float64 // how far toward the note, 0 to 1 (1)
	MaxShift  float64 // the most the voice is moved, in semitones (4): more than that and it does not sound like the same voice
	Ramp      float64 // milliseconds a word takes to go to its note at the start and to come back at the end (60): a word does not begin on another pitch than it was said on
	Offset    float64 // semitones added to the notes (a voice that speaks low may sound better an octave or a few semitones from where it is)
	PhraseGap float64 // seconds of silence between two phrases (0.35)
}

// nearest is the allowed note (in MIDI, a whole number) nearest to a pitch
func nearest(m float64, classes []int) float64 {
	best, bestD := math.Round(m), math.Inf(1)
	for n := math.Floor(m) - 7; n <= math.Ceil(m)+7; n++ {
		if !slices.Contains(classes, ((int(n)%12)+12)%12) {
			continue
		}
		if d := math.Abs(n - m); d < bestD {
			best, bestD = n, d
		}
	}
	return best
}

// Target makes the track the voice is moved to, from the pitch of the voice
func Target(t Track, o Options) Track {
	if o.Strength == 0 {
		o.Strength = 1
	}
	if o.MaxShift == 0 {
		o.MaxShift = 4
	}
	if o.Ramp == 0 {
		o.Ramp = 60
	}
	if o.PhraseGap == 0 {
		o.PhraseGap = 0.35
	}
	out := Track{Hop: t.Hop, SampleRate: t.SampleRate, F0: make([]float64, len(t.F0))}
	hopSec := float64(t.Hop) / t.SampleRate
	// the runs of voiced frames, and the phrases they make (runs close together)
	type run struct{ from, to int }
	var runs []run
	for f := 0; f < len(t.F0); {
		if t.F0[f] == 0 {
			f++
			continue
		}
		g := f
		for g < len(t.F0) && t.F0[g] > 0 {
			g++
		}
		runs = append(runs, run{f, g})
		f = g
	}
	median := func(from, to int) float64 {
		var ms []float64
		for f := from; f < to; f++ {
			if t.F0[f] > 0 {
				ms = append(ms, Midi(t.F0[f]))
			}
		}
		slices.Sort(ms)
		return ms[len(ms)/2]
	}
	phraseMedian := make([]float64, len(runs))
	for i := 0; i < len(runs); {
		j := i + 1
		for j < len(runs) && float64(runs[j].from-runs[j-1].to)*hopSec < o.PhraseGap {
			j++
		}
		m := median(runs[i].from, runs[j-1].to)
		for k := i; k < j; k++ {
			phraseMedian[k] = m
		}
		i = j
	}
	for i, r := range runs {
		var shift float64 // semitones, smoothed over the run
		for f := r.from; f < r.to; f++ {
			m := Midi(t.F0[f])
			classes := o.Allowed(float64(f) * hopSec)
			var want float64
			switch o.Mode {
			case Chant:
				// the note near the register of the phrase, the rise and fall of the voice kept in part
				want = nearest(phraseMedian[i]+o.Offset, classes) + o.Keep*(m-median(r.from, r.to))
			default:
				want = nearest(m+o.Offset, classes)
			}
			d := (want - m) * o.Strength
			d = max(-o.MaxShift, min(o.MaxShift, d))
			if f == r.from || o.Speed <= 0 {
				shift = d
			} else {
				a := 1 - math.Exp(-hopSec*1000/o.Speed)
				shift += a * (d - shift)
			}
			// the start and the end of the word: the shift grows from nothing and falls back to nothing
			edge := min(float64(f-r.from)+1, float64(r.to-f)) * hopSec * 1000 / o.Ramp
			edge = min(edge, 1)
			edge = edge * edge * (3 - 2*edge)
			out.F0[f] = t.F0[f] * math.Pow(2, shift*edge/12)
		}
	}
	return out
}
