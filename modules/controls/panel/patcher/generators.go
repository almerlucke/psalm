package patcher

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/arpeggio"
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/tools/harmony"
)

// --- Clock ---

// Divisions are the clock's steps, and how many of them make a beat
var Divisions = []string{"1/4", "1/8", "1/8T", "1/16", "1/16T", "1/32"}

var stepsPerBeat = []float64{1, 2, 3, 4, 6, 8}

// Clock sends a trigger on every step of the patch's tempo: a division of the beat, swung (every
// second step later), a step pattern ("x . x x": a hit or a rest, looping) and a chance for every
// hit. Settings: "division" (an index of Divisions), "swing" (0.5 straight to 0.75), "chance" (0-1);
// the pattern is a setting of text
type Clock struct {
	*psalm.Module
	Division int
	Swing    float64
	Chance   float64
	pattern  atomic.Pointer[[]bool]
	next     int64 // the next step to send
	last     int64 // the tick timestamp of the last control tick
	started  bool
	rng      *rand.Rand
}

// NewClock returns a clock of sixteenths, straight, every step a hit
func NewClock() *Clock {
	c := &Clock{Module: psalm.NewControl(), Division: 3, Swing: 0.5, Chance: 1, rng: rand.New(rand.NewPCG(rand.Uint64(), 3))}
	all := []bool{true}
	c.pattern.Store(&all)
	c.SetSelf(c)
	return c
}

// ParsePattern reads a step pattern: x (or X, 1, o) a hit, . (or -, 0, _) a rest; spaces and bar
// lines (|) are left out
func ParsePattern(text string) ([]bool, error) {
	var p []bool
	for _, r := range text {
		switch r {
		case 'x', 'X', '1', 'o', 'O':
			p = append(p, true)
		case '.', '-', '0', '_':
			p = append(p, false)
		case ' ', '\t', '\n', '|':
		default:
			return nil, fmt.Errorf("%q in a pattern: x is a hit, . a rest", r)
		}
	}
	if len(p) == 0 {
		return nil, fmt.Errorf("an empty pattern")
	}
	return p, nil
}

// SetPattern sets the step pattern (from any goroutine)
func (c *Clock) SetPattern(text string) error {
	p, err := ParsePattern(text)
	if err == nil {
		c.pattern.Store(&p)
	}
	return err
}

func (c *Clock) ReceiveMessage(msg psalm.Message) {
	msg.IfIntVal("division", func(d int) { c.Division = max(0, min(d, len(Divisions)-1)) })
	msg.IfFltVal("swing", func(f float64) { c.Swing = math.Max(0.5, math.Min(f, 0.75)) })
	msg.IfFltVal("chance", func(f float64) { c.Chance = math.Max(0, math.Min(f, 1)) })
}

func (c *Clock) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		c.ReceiveMessage(m)
	}
}

// stepTicks returns how many control ticks a step lasts
func (c *Clock) stepTicks() float64 {
	return float64(c.Config().PPQ) / stepsPerBeat[c.Division]
}

// at returns the tick a step starts on: every second one later by the swing
func (c *Clock) at(step int64) float64 {
	l := c.stepTicks()
	t := float64(step) * l
	if step%2 == 1 {
		t += (c.Swing - 0.5) * 2 * l
	}
	return t
}

// CtrlTick sends the steps starting on this tick; it follows the patch's clock (a restart, a jump
// back starts the steps over from where it is)
func (c *Clock) CtrlTick(tick psalm.CtrlTick) {
	t := float64(tick.TickTimestamp)
	if !c.started || tick.TickTimestamp < c.last {
		c.started = true
		c.next = int64(math.Ceil(t/c.stepTicks() - 1e-9))
		if c.next > 0 && c.at(c.next-1) >= t-1e-9 { // a swung step that is still to come
			c.next--
		}
	}
	c.last = tick.TickTimestamp
	for c.at(c.next) <= t+1e-9 {
		step := c.next
		c.next++
		if c.at(step) < t-1 { // missed (the tempo jumped): no burst of old steps
			continue
		}
		p := *c.pattern.Load()
		if !p[int(step%int64(len(p)))] || (c.Chance < 1 && c.rng.Float64() >= c.Chance) {
			continue
		}
		seconds := c.stepTicks() * 60 / (c.BPM() * float64(c.Config().PPQ))
		c.Send(Event{Tick: tick, Step: step, Seconds: seconds}, 0)
	}
}

// Describe describes the clock's settings
func (c *Clock) Describe() params.Set {
	return params.Set{Name: "clock", Params: []params.Param{
		params.C("Clock", "division", Divisions, c.Division),
		params.K("Clock", "swing", 0.5, 0.75, c.Swing),
		params.K("Clock", "chance", 0, 1, c.Chance),
	}}
}

// --- Sequence ---

// Orders are how a sequence goes through its values
var Orders = []string{"in order", "shuffled", "random", "up and down", "converge", "alternate", "backwards"}

// Sequence gives a value on every trigger, from a list of values (typed: numbers or note names,
// "C3 Eb3 G3", a chord in brackets "[C3 E3 G3]") in an order (genny: a sequence, a bucket, an
// arpeggio). Inputs: trigger, reset (back to the start); outputs: the value and a trigger after it
// (to chain modules that need the value first). Setting: "order" (an index of Orders)
type Sequence struct {
	*psalm.Module
	Order  int
	values atomic.Pointer[[]any]
	gen    genny.Generator[any] // on the audio thread
	built  *[]any               // the values gen was made from
	order  int                  // the order gen was made for
}

// NewSequence returns a sequence of values
func NewSequence() *Sequence {
	s := &Sequence{Module: psalm.NewModule(nil, nil), order: -1}
	s.SetSelf(s)
	_ = s.SetValues("C3 Eb3 G3 Bb3")
	return s
}

// ParseValues reads a list of values: numbers, note names (C3, F#4, Bb2), chords in brackets
func ParseValues(text string) ([]any, error) {
	var vals []any
	var chord []float64
	inChord := false
	for _, f := range strings.Fields(strings.NewReplacer("[", " [ ", "]", " ] ", ",", " ").Replace(text)) {
		switch f {
		case "[":
			if inChord {
				return nil, fmt.Errorf("a chord in a chord")
			}
			inChord, chord = true, nil
			continue
		case "]":
			if !inChord || len(chord) == 0 {
				return nil, fmt.Errorf("a ] without a chord")
			}
			inChord = false
			vals = append(vals, chord)
			continue
		}
		v, err := score.Pitch(f)
		if err != nil {
			return nil, fmt.Errorf("%q: a number or a note name (C3, F#4, Bb2)", f)
		}
		if inChord {
			chord = append(chord, v)
		} else {
			vals = append(vals, v)
		}
	}
	if inChord {
		return nil, fmt.Errorf("a chord without its ]")
	}
	if len(vals) == 0 {
		return nil, fmt.Errorf("no values")
	}
	return vals, nil
}

// SetValues sets the values (from any goroutine)
func (s *Sequence) SetValues(text string) error {
	v, err := ParseValues(text)
	if err == nil {
		s.values.Store(&v)
	}
	return err
}

func (s *Sequence) ReceiveMessage(msg psalm.Message) {
	msg.IfIntVal("order", func(o int) { s.Order = max(0, min(o, len(Orders)-1)) })
}

// generator returns the generator of the values in the order (made again when either changed)
func (s *Sequence) generator() genny.Generator[any] {
	vals := s.values.Load()
	if s.gen != nil && vals == s.built && s.Order == s.order {
		return s.gen
	}
	s.built, s.order = vals, s.Order
	v := *vals
	switch s.Order {
	case 1:
		s.gen = bucket.NewLoop(v...).SetMode(bucket.Indexed)
	case 2:
		s.gen = bucket.NewLoop(v...).SetMode(bucket.Random)
	case 3:
		s.gen = arpeggio.NewLoop(v, arpeggio.Up, arpeggio.Exclusive, false)
	case 4:
		s.gen = arpeggio.NewLoop(v, arpeggio.Converge, arpeggio.None, false)
	case 5:
		s.gen = arpeggio.NewLoop(v, arpeggio.Alternate, arpeggio.None, false)
	case 6:
		back := slices.Clone(v)
		slices.Reverse(back)
		s.gen = sequence.NewLoop(back...)
	default:
		s.gen = sequence.NewLoop(v...)
	}
	return s.gen
}

// Receive takes a trigger (input 0: the next value), a reset (input 1) or a message (the order)
func (s *Sequence) Receive(v any, index int) {
	if m, ok := psalm.MsgVal(v); ok {
		s.ReceiveMessage(m)
		return
	}
	switch index {
	case 0:
		ev, _ := v.(Event)
		s.Send(s.generator().Generate(), 0)
		s.Send(ev, 1)
	case 1:
		s.generator().Reset()
	}
}

// Describe describes the sequence's settings
func (s *Sequence) Describe() params.Set {
	return params.Set{Name: "sequence", Params: []params.Param{params.C("Sequence", "order", Orders, s.Order)}}
}

// --- Notes ---

// Note plays a note on every trigger, to the synths its notes cable goes to: its pitch, velocity
// and length (in steps of the clock that triggered it) from its value inputs, or from its settings
// for an input not patched. Inputs: trigger, pitch, velocity, length. A pitch that is a chord plays
// every note of it. Settings: "pitch", "velocity", "length" (steps), "transpose" (semitones)
type Note struct {
	*psalm.Module
	Pitch, Velocity, Length, Transpose float64

	pitch            any // from the cables, nil: the setting
	velocity, length float64
	hasVel, hasLen   bool
	pending          []Event
	targets          []psalm.Receiver
}

// NewNote returns notes of C3 (48) at a velocity of 0.8, a step long
func NewNote() *Note {
	n := &Note{Module: psalm.NewModule(nil, nil), Pitch: 48, Velocity: 0.8, Length: 0.9}
	n.SetSelf(n)
	return n
}

func (n *Note) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("pitch", func(f float64) { n.Pitch = f })
	msg.IfFltVal("velocity", func(f float64) { n.Velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("length", func(f float64) { n.Length = math.Max(0.01, f) })
	msg.IfFltVal("transpose", func(f float64) { n.Transpose = f })
}

// Receive takes a trigger (input 0, played at the end of the tick), a pitch (1), a velocity (2), a
// length in steps (3), or a message (the settings)
func (n *Note) Receive(v any, index int) {
	if m, ok := psalm.MsgVal(v); ok {
		n.ReceiveMessage(m)
		return
	}
	switch index {
	case 0:
		if ev, ok := v.(Event); ok {
			n.pending = append(n.pending, ev)
		}
	case 1:
		n.pitch = v
	case 2:
		n.velocity, n.hasVel = psalm.FltVal[float64](v), true
	case 3:
		n.length, n.hasLen = psalm.FltVal[float64](v), true
	}
}

// SetTargets sets the synths it plays (on the audio thread)
func (n *Note) SetTargets(targets []psalm.Receiver) { n.targets = targets }

// late plays the triggers of the tick, with the values as they are now
func (n *Note) late(psalm.CtrlTick) {
	if len(n.pending) == 0 {
		return
	}
	for _, ev := range n.pending {
		vel := n.Velocity
		if n.hasVel {
			vel = n.velocity
		}
		steps := n.Length
		if n.hasLen {
			steps = n.length
		}
		seconds := math.Max(0.005, steps*ev.Seconds)
		var pitches []float64
		switch p := n.pitch.(type) {
		case nil:
			pitches = []float64{n.Pitch}
		case []float64:
			pitches = p
		default:
			pitches = []float64{psalm.FltVal[float64](p)}
		}
		for _, pitch := range pitches {
			msg := psalm.Message{"command": "trigger", "tick": ev.Tick, "duration": seconds, "amplitude": vel,
				"message": psalm.Message{"pitch": pitch + n.Transpose, "velocity": vel}}
			for _, t := range n.targets {
				t.ReceiveMessage(msg)
			}
		}
	}
	n.pending = n.pending[:0]
}

// Describe describes the settings of the notes
func (n *Note) Describe() params.Set {
	return params.Set{Name: "notes", Params: []params.Param{
		params.K("Notes", "pitch", 0, 127, n.Pitch, params.Step(1)),
		params.K("Notes", "velocity", 0, 1, n.Velocity),
		params.K("Notes", "length", 0.05, 16, n.Length, params.Exp(), params.Label("length (steps)")),
		params.K("Notes", "transpose", -36, 36, n.Transpose, params.Step(1), params.Unit("st")),
	}}
}

// --- Quantize ---

// Keys are the roots of a scale
var Keys = []string{"C", "Db", "D", "Eb", "E", "F", "F#", "G", "Ab", "A", "Bb", "B"}

// Scales are the scales to quantize to (tools/harmony)
var Scales = func() []string {
	names := make([]string, 0, len(harmony.Modes))
	for n := range harmony.Modes {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}()

// Quantize moves every value it gets (a pitch, a chord) to the nearest note of a scale. Settings:
// "key" (an index of Keys), "scale" (an index of Scales)
type Quantize struct {
	*psalm.Module
	Key, Scale int
	scale      harmony.Scale
	made       [2]int
}

// NewQuantize returns a quantizer to C minor
func NewQuantize() *Quantize {
	q := &Quantize{Module: psalm.NewModule(nil, nil), Scale: slices.Index(Scales, "minor"), made: [2]int{-1, -1}}
	q.SetSelf(q)
	return q
}

func (q *Quantize) ReceiveMessage(msg psalm.Message) {
	msg.IfIntVal("key", func(k int) { q.Key = max(0, min(k, len(Keys)-1)) })
	msg.IfIntVal("scale", func(s int) { q.Scale = max(0, min(s, len(Scales)-1)) })
}

// snap moves a pitch into the scale
func (q *Quantize) snap(p float64) float64 {
	if q.made != [2]int{q.Key, q.Scale} {
		q.scale = harmony.MustScale(Keys[q.Key], Scales[q.Scale])
		q.made = [2]int{q.Key, q.Scale}
	}
	return q.scale.Snap(p)
}

// Receive takes a value (input 0) and sends it quantized, or a message (the settings)
func (q *Quantize) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		q.ReceiveMessage(m)
		return
	}
	switch p := v.(type) {
	case []float64:
		out := make([]float64, len(p))
		for i, x := range p {
			out[i] = q.snap(x)
		}
		q.Send(out, 0)
	default:
		q.Send(q.snap(psalm.FltVal[float64](v)), 0)
	}
}

// Describe describes the quantizer's settings
func (q *Quantize) Describe() params.Set {
	return params.Set{Name: "quantize", Params: []params.Param{
		params.C("Quantize", "key", Keys, q.Key),
		params.C("Quantize", "scale", Scales, q.Scale),
	}}
}

// Generators are the kinds of generator modules: a clock, a sequence of values, notes, a quantizer
func Generators() []Kind {
	trig := Port{Name: "trigger", Type: Trigger}
	return []Kind{
		{Name: "clock", Group: "Generators", Outs: []Port{trig}, Make: func(string) (*Made, error) {
			c := NewClock()
			return &Made{Module: c, Describe: func(string) params.Set { return c.Describe() },
				Texts: []Text{{Key: "pattern", Label: "pattern", Value: "x"}}, SetText: func(_, v string) error { return c.SetPattern(v) }}, nil
		}},
		{Name: "sequence", Group: "Generators",
			Ins:  []Port{trig, {Name: "reset", Type: Trigger, Index: 1}},
			Outs: []Port{{Name: "value", Type: Value}, {Name: "after", Type: Trigger, Index: 1}},
			Make: func(string) (*Made, error) {
				s := NewSequence()
				return &Made{Module: s, Describe: func(string) params.Set { return s.Describe() },
					Texts: []Text{{Key: "values", Label: "values", Value: "C3 Eb3 G3 Bb3"}}, SetText: func(_, v string) error { return s.SetValues(v) }}, nil
			}},
		{Name: "notes", Group: "Generators",
			Ins:  []Port{trig, {Name: "pitch", Type: Value, Index: 1}, {Name: "velocity", Type: Value, Index: 2}, {Name: "length", Type: Value, Index: 3}},
			Outs: []Port{{Name: "notes", Type: Notes}},
			Make: func(string) (*Made, error) {
				n := NewNote()
				return &Made{Module: n, Describe: func(string) params.Set { return n.Describe() },
					Notes: n.SetTargets, Late: n.late}, nil
			}},
		{Name: "quantize", Group: "Generators", Ins: []Port{{Name: "in", Type: Value}}, Outs: []Port{{Name: "out", Type: Value}},
			Make: func(string) (*Made, error) {
				q := NewQuantize()
				return &Made{Module: q, Describe: func(string) params.Set { return q.Describe() }}, nil
			}},
	}
}
