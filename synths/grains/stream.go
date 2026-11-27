package grains

import (
	"github.com/almerlucke/genny"
)

// A Stream gives the value of a parameter for every grain that starts: where it looks, how long it is, which pitch it has, when the
// next one comes. The streams of a voice replace the settings of the same name for the grains: a list that is cycled, or any
// generator of genny (a euclidean rhythm for the time between grains, a walk for the position, a scale for the pitch)
type Stream interface {
	// Next is the value for the next grain
	Next() float64
	// Reset starts the stream over (at every note)
	Reset()
}

// seq cycles a list
type seq struct {
	values []float64
	i      int
}

// Seq is a stream that plays the values one after the other and starts again at the end (a copy of the list)
func Seq(values ...float64) Stream { return &seq{values: append([]float64(nil), values...)} }

func (s *seq) Next() float64 {
	v := s.values[s.i]
	s.i = (s.i + 1) % len(s.values)
	return v
}
func (s *seq) Reset() { s.i = 0 }

// gen is a generator of genny as a stream: the last value goes on when it is done
type gen struct {
	g    genny.Generator[float64]
	last float64
}

// Gen is a stream of the values of a generator of genny. A generator that ends goes on with its last value
func Gen(g genny.Generator[float64]) Stream { return &gen{g: g} }

func (s *gen) Next() float64 {
	if !s.g.Done() {
		s.last = s.g.Generate()
	}
	return s.last
}
func (s *gen) Reset() { s.g.Reset() }

// Streams are the streams of a voice, as functions that make them (a voice of a poly synth each makes its own, so they do not
// share a position in a list or a generator). A nil one leaves the setting as it is
//
//	Interval  seconds until the next grain (instead of density and jitter, and of sync)
//	Position  0 to 1, where the grain looks in the source (the scan still moves it on)
//	Pitch     semitones, added to the pitch of the note
//	Size      milliseconds, the length of the grain
//	Pan       0 to 1, left to right
//	Amp       the level of the grain (0 is a rest: the grain does not sound)
//	Cutoff    Hz, the cutoff of the filter of the grain
//	Slice     the slice the grain plays (counting from 0, see Config.Slices)
type Streams struct {
	Interval, Position, Pitch, Size, Pan, Amp, Cutoff, Slice func() Stream
}

// voiceStreams are the streams of one voice
type voiceStreams struct {
	interval, position, pitch, size, pan, amp, cutoff, slice Stream
}

func (s Streams) make() voiceStreams {
	mk := func(f func() Stream) Stream {
		if f == nil {
			return nil
		}
		return f()
	}
	return voiceStreams{mk(s.Interval), mk(s.Position), mk(s.Pitch), mk(s.Size), mk(s.Pan), mk(s.Amp), mk(s.Cutoff), mk(s.Slice)}
}

func (s *voiceStreams) reset() {
	for _, st := range []Stream{s.interval, s.position, s.pitch, s.size, s.pan, s.amp, s.cutoff, s.slice} {
		if st != nil {
			st.Reset()
		}
	}
}

// streamKeys are the messages that set a stream from a list of numbers (a script's set("positionSeq", [0.1, 0.5, 0.3]), an empty list
// takes the stream away again)
var streamKeys = []string{"intervalSeq", "positionSeq", "pitchSeq", "sizeSeq", "panSeq", "ampSeq", "cutoffSeq", "sliceSeq"}
