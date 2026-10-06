package score

import (
	"math"
	"sort"

	"github.com/almerlucke/psalm"
)

// Config sets how a sequencer plays its notes
type Config struct {
	Amp      float64 // amplitude of the notes (0 is 1)
	Legato   float64 // part of its length a note sounds when it has no hold (0 is 0.9)
	Velocity float64 // velocity of notes without one, 0 sends none (the synth default)
	Key      string  // message key of the pitches: "pitch" (the default) or "frequency"
	Loop     bool    // start again after the last note
	Length   float64 // loop length in beats, 0 is the length of the notes
}

// event is a note at its tick
type event struct {
	tick int64
	note Note
}

// Sequencer plays notes into a poly synth: connect it to the synth with synth.ValueIn(sequencer).
// Every note is sent as a trigger at its exact tick, a chord as one trigger per pitch. SetGain
// scales the amplitude of the notes that follow, for example for dynamics
type Sequencer struct {
	*psalm.Module
	cfg    Config
	events []event
	length int64 // ticks
	next   int
	gain   float64
	last   int64
}

// NewSequencer creates a sequencer playing notes
func NewSequencer(notes []Note, cfg Config) *Sequencer {
	if cfg.Amp == 0 {
		cfg.Amp = 1
	}
	if cfg.Legato == 0 {
		cfg.Legato = 0.9
	}
	if cfg.Key == "" {
		cfg.Key = "pitch"
	}
	s := &Sequencer{Module: psalm.NewControl(), cfg: cfg, gain: 1, last: -1}
	s.SetSelf(s)
	s.SetNotes(notes)
	return s
}

// SetNotes replaces the notes, playing continues from the current position
func (s *Sequencer) SetNotes(notes []Note) {
	ppq := float64(psalm.PPQ())
	s.events = s.events[:0]
	beat := 0.0
	for _, n := range notes {
		if !n.IsRest() {
			s.events = append(s.events, event{tick: int64(math.Round(beat * ppq)), note: n})
		}
		beat += n.Beats
	}
	if s.cfg.Length > 0 {
		beat = s.cfg.Length
	}
	s.length = max(1, int64(math.Round(beat*ppq)))
	sort.SliceStable(s.events, func(i, j int) bool { return s.events[i].tick < s.events[j].tick })
	s.next = -1 // find the position again at the next tick
}

// SetGain scales the amplitude of the notes that follow
func (s *Sequencer) SetGain(gain float64) {
	s.gain = gain
}

// Length returns the length of the sequence in beats
func (s *Sequencer) Length() float64 {
	return float64(s.length) / float64(psalm.PPQ())
}

func (s *Sequencer) CtrlTick(tick psalm.CtrlTick) {
	t := tick.TickTimestamp - 1
	if !s.cfg.Loop && t >= s.length {
		return
	}
	pos := t % s.length

	// a new loop, or the position is unknown: find the first event at or after the position
	if s.next < 0 || pos < s.last {
		s.next = sort.Search(len(s.events), func(i int) bool { return s.events[i].tick >= pos })
	}
	s.last = pos

	for s.next < len(s.events) && s.events[s.next].tick == pos {
		s.play(s.events[s.next].note, tick)
		s.next++
	}
}

func (s *Sequencer) play(n Note, tick psalm.CtrlTick) {
	hold := n.Hold
	if hold == 0 {
		hold = n.Beats * s.cfg.Legato
	}
	seconds := hold * 60.0 / psalm.BPM()

	amp := s.cfg.Amp * s.gain
	if n.Amp > 0 {
		amp *= n.Amp
	}

	vel := n.Velocity
	if vel == 0 {
		vel = s.cfg.Velocity
	}

	for _, p := range n.Pitches {
		msg := psalm.Message{s.cfg.Key: p}
		if vel > 0 {
			msg["velocity"] = vel
		}
		for k, v := range n.Params {
			msg[k] = v
		}
		s.Send(psalm.Message{
			"command":   "trigger",
			"duration":  seconds,
			"amplitude": amp,
			"tick":      tick,
			"message":   msg,
		}, 0)
	}
}
