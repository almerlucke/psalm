package score

import (
	"math"
	"math/rand/v2"
	"sort"
	"sync/atomic"

	"github.com/almerlucke/psalm"
)

// Config sets how a sequencer plays its notes
type Config struct {
	Amp      float64 // amplitude of the notes (0 is 1)
	Legato   float64 // part of its length a note sounds when it has no hold (0 is 0.9)
	Velocity float64 // velocity of notes without one, 0 sends none (the synth default)
	Key      string  // message key of the pitches: "pitch" (the default) or "frequency"
	Tuning   Tuning  // plays the pitches (MIDI) in a tuning: sent as "frequency" (Key is not used)
	Loop     bool    // start again after the last note
	Length   float64 // loop length in beats, 0 is the length of the notes
	Feel     Feel    // swing, humanized timing and dynamics, accents
}

// Feel takes notes off the grid: swing, random timing and dynamics like a player, strummed
// chords, playing ahead of or behind the beat and accents through the bar. The random parts are
// drawn again every time a loop starts, so repeats are never exactly the same
type Feel struct {
	Swing     float64 // where the second note of a swung pair lands: 0 (or 0.5) is straight, 0.58 light, 0.66 a triplet feel, 0.75 hard
	SwingUnit float64 // beats per swung note: 0.5 swings 8ths (the default), 0.25 16ths

	Timing float64 // ± milliseconds of random timing per note, the notes of a chord move together
	Spread float64 // ± milliseconds of random timing per note within a chord, an ensemble never starts exactly together
	Strum  float64 // milliseconds between the notes of a chord, low to high (negative is high to low)
	// StrumFade is how much softer the last note of a strummed chord is than the first (0 is as loud, 0.5 half as hard, 1 silent),
	// the notes between fall in a line: the finger or the pick plucks the first string hardest and passes the others
	StrumFade float64
	Push      float64 // milliseconds every note moves: negative plays ahead of the beat, positive lays back
	// (a note on the very first beat of the sequence can't play early, it plays at the start)

	Velocity float64 // ± random variation of velocity and amplitude (0.1 is ±10%)

	Accents     []float64 // amplitude per accent unit through the bar, like 1, 0.7, 0.85, 0.7
	AccentUnit  float64   // beats per accent (0 is 1)
	BeatsPerBar float64   // for the accents (0 is 4)

	Seed uint64 // the same seed gives the same feel, 0 is a new feel every run
}

// random returns true when the feel draws random numbers
func (f Feel) random() bool {
	return f.Timing != 0 || f.Spread != 0 || f.Velocity != 0
}

// warp moves a position in beats by the swing: within every pair of swing units the first unit
// is stretched and the second compressed, so the second unit starts at Swing of the pair
func (f Feel) warp(beat float64) float64 {
	if f.Swing <= 0 || f.Swing == 0.5 {
		return beat
	}
	u := f.SwingUnit
	if u <= 0 {
		u = 0.5
	}
	pair := 2 * u
	base := math.Floor(beat/pair) * pair
	x := beat - base
	split := pair * max(0.01, min(f.Swing, 0.99))
	if x < u {
		return base + x*split/u
	}
	return base + split + (x-u)*(pair-split)/u
}

// accent returns the accent at a position
func (f Feel) accent(beat float64) float64 {
	if len(f.Accents) == 0 {
		return 1
	}
	unit, bar := f.AccentUnit, f.BeatsPerBar
	if unit <= 0 {
		unit = 1
	}
	if bar <= 0 {
		bar = 4
	}
	inBar := beat - math.Floor(beat/bar)*bar
	return f.Accents[int(math.Floor(inBar/unit+1e-9))%len(f.Accents)]
}

// placed is a note at its position in beats
type placed struct {
	beat float64
	note Note
}

// event is a pitch played at a tick, after the feel
type event struct {
	tick  int64
	pitch float64
	hold  float64 // beats
	amp   float64
	vel   float64
	note  *Note
}

// Sequencer plays notes into a poly synth: connect it to the synth with synth.ValueIn(sequencer).
// Every note is sent as a trigger at its tick, a chord as one trigger per pitch, shaped by the
// feel. SetGain scales the amplitude of the notes that follow, for example for dynamics
type Sequencer struct {
	*psalm.Module
	cfg    Config
	notes  []placed
	events []event
	length int64 // ticks
	next   int
	gain   float64
	last   int64
	rng    *rand.Rand
	clock  func(tick int64) (int64, bool)
	off    atomic.Bool // switched off: no new notes (SetEnabled)
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
	seed := cfg.Feel.Seed
	if seed == 0 {
		seed = 0 // set below from the module stream
	}
	s := &Sequencer{Module: psalm.NewControl(), cfg: cfg, gain: 1, last: -1}
	if seed == 0 {
		seed = s.Rand().Uint64() // the patch seed makes it repeat
	}
	s.rng = rand.New(rand.NewPCG(seed, 0x853c49e6748fea9b))
	s.SetSelf(s)
	s.SetNotes(notes)
	return s
}

// SetNotes replaces the notes, playing continues from the current position
func (s *Sequencer) SetNotes(notes []Note) {
	s.notes = s.notes[:0]
	beat := 0.0
	for _, n := range notes {
		if !n.IsRest() {
			s.notes = append(s.notes, placed{beat, n})
		}
		beat += n.Beats
	}
	if s.cfg.Length > 0 {
		beat = s.cfg.Length
	}
	s.length = max(1, int64(math.Round(beat*float64(s.Config().PPQ))))
	s.build()
	s.resume()
}

// resume continues after a rebuild with the first event after the last tick played, so notes of
// the current tick are not played twice when the notes or the feel change during a tick
func (s *Sequencer) resume() {
	last := s.last
	s.next = sort.Search(len(s.events), func(i int) bool { return s.events[i].tick > last })
}

// SetClock makes the sequencer follow a clock instead of the ticks of the patch: the clock turns a
// tick of the patch into the tick of the sequence to play (false: nothing plays). The sequence can
// jump: it goes on from wherever the clock says. A song transport (arrange) sets it
func (s *Sequencer) SetClock(clock func(tick int64) (int64, bool)) {
	s.clock = clock
}

// Restart plays the notes from the start again (psalm.Restarter)
func (s *Sequencer) Restart() {
	if s.last >= 0 && s.cfg.Feel.random() {
		s.build() // a new pass, a new performance
	}
	s.last, s.next = -1, 0
}

// SetFeel changes the feel from the next note on
func (s *Sequencer) SetFeel(feel Feel) {
	s.cfg.Feel = feel
	s.build()
	s.resume()
}

// SetGain scales the amplitude of the notes that follow
func (s *Sequencer) SetGain(gain float64) {
	s.gain = gain
}

// Length returns the length of the sequence in beats
func (s *Sequencer) Length() float64 {
	return float64(s.length) / float64(s.Config().PPQ)
}

// tri returns a random number from -1 to 1, most often near 0
func (s *Sequencer) tri() float64 {
	return s.rng.Float64() + s.rng.Float64() - 1
}

// build places every pitch at its tick, with the feel applied
func (s *Sequencer) build() {
	var (
		f        = s.cfg.Feel
		ppq      = float64(s.Config().PPQ)
		msToBeat = s.BPM() / 60000.0
	)
	s.events = s.events[:0]
	for i := range s.notes {
		pl := &s.notes[i]
		n := &pl.note

		hold := n.Hold
		if hold == 0 {
			hold = n.Beats * s.cfg.Legato
		}
		start := f.warp(pl.beat)
		holdBeats := f.warp(pl.beat+hold) - start

		offset := f.Push
		if f.Timing != 0 {
			offset += s.tri() * f.Timing
		}

		amp := s.cfg.Amp * f.accent(pl.beat)
		if n.Amp > 0 {
			amp *= n.Amp
		}
		vel := n.Velocity
		if vel == 0 {
			vel = s.cfg.Velocity
		}
		if f.Velocity != 0 {
			v := 1 + s.tri()*f.Velocity
			amp *= v
			if vel > 0 {
				vel = max(0.01, min(vel*v, 1))
			}
		}

		// strum order: low to high, or high to low
		order := make([]int, len(n.Pitches))
		for k := range order {
			order[k] = k
		}
		sort.SliceStable(order, func(a, b int) bool {
			if f.Strum < 0 {
				return n.Pitches[order[a]] > n.Pitches[order[b]]
			}
			return n.Pitches[order[a]] < n.Pitches[order[b]]
		})

		for k, idx := range order {
			evVel := vel
			if f.StrumFade > 0 && len(order) > 1 && vel > 0 {
				evVel = max(0.01, vel*(1-min(f.StrumFade, 1)*float64(k)/float64(len(order)-1)))
			}
			ms := offset + math.Abs(f.Strum)*float64(k)
			if f.Spread != 0 {
				ms += s.tri() * f.Spread
			}
			tick := int64(math.Round((start + ms*msToBeat) * ppq))
			tick = max(0, min(tick, s.length-1))
			s.events = append(s.events, event{tick: tick, pitch: n.Pitches[idx], hold: holdBeats, amp: amp, vel: evVel, note: n})
		}
	}
	sort.SliceStable(s.events, func(i, j int) bool { return s.events[i].tick < s.events[j].tick })
}

func (s *Sequencer) CtrlTick(tick psalm.CtrlTick) {
	t := tick.TickTimestamp
	if s.clock != nil {
		c, ok := s.clock(t)
		if !ok {
			return
		}
		t = c
	} else if !s.cfg.Loop && t >= s.length {
		return
	}
	pos := t % s.length

	// a new loop (or a jump back): start from the first event at or after the position
	if pos < s.last {
		if s.cfg.Feel.random() {
			s.build() // a new pass, a new performance
		}
		s.next = sort.Search(len(s.events), func(i int) bool { return s.events[i].tick >= pos })
	} else if pos > s.last+1 { // a jump forward
		s.next = sort.Search(len(s.events), func(i int) bool { return s.events[i].tick >= pos })
	}
	s.last = pos

	off := s.off.Load()
	for s.next < len(s.events) && s.events[s.next].tick == pos {
		if !off {
			s.play(&s.events[s.next], tick)
		}
		s.next++
	}
}

// SetEnabled switches the part on or off while it plays (from any goroutine): off, it starts no new
// notes (the notes sounding ring out) and keeps its place, on again it goes on from there
func (s *Sequencer) SetEnabled(on bool) { s.off.Store(!on) }

// Enabled tells whether the part plays its notes
func (s *Sequencer) Enabled() bool { return !s.off.Load() }

func (s *Sequencer) play(e *event, tick psalm.CtrlTick) {
	msg := psalm.Message{s.cfg.Key: e.pitch}
	if s.cfg.Tuning != nil {
		msg = psalm.Message{"frequency": s.cfg.Tuning(e.pitch)}
	}
	if e.vel > 0 {
		msg["velocity"] = e.vel
	}
	for k, v := range e.note.Params {
		msg[k] = v
	}
	s.Send(psalm.Message{
		"command":   "trigger",
		"duration":  e.hold * 60.0 / s.BPM(),
		"amplitude": e.amp * s.gain,
		"tick":      tick,
		"message":   msg,
	}, 0)
}
