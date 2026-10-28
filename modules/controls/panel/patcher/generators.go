package patcher

import (
	"fmt"
	"math"
	"slices"
	"strconv"
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

// Clock sends triggers on the steps of the patch's tempo: a division of the beat, swung (every
// second step later), a step pattern (looping) and a chance for every hit. The pattern: x (or X, o,
// 1) a hit, . (or -, _, 0) a rest, a step each; x3 a roll (3 hits in the step); [x x x] a group
// filling a step (a triplet); *2 after a step or a group: it lasts 2 steps; ? after a hit or a
// group: its hits by chance (50%, ?25: 25%). Spaces, bar lines and new lines are left out. A
// trigger carries how long its hit lasts (a note of a step follows it). Settings: "division" (an
// index of Divisions), "swing" (0.5 straight to 0.75), "chance" (0-1); the pattern is a setting of
// text, per section of a song too
type Clock struct {
	*psalm.Module
	Division int
	Swing    float64
	Chance   float64
	pattern  atomic.Pointer[patterns]
	song     *Song

	// on the audio thread: the pattern playing, where it starts, the next hit
	playing *Pattern
	origin  float64 // the tick its first step starts on
	cycle   int64   // the loop of the pattern the next hit is in
	hit     int     // the next hit in it
	div     int     // the division it counts in
	last    int64   // the tick timestamp of the last control tick
	started bool
}

// NewClock returns a clock of sixteenths, straight, every step a hit
func NewClock() *Clock {
	c := &Clock{Module: psalm.NewControl(), Division: 3, Swing: 0.5, Chance: 1}
	all, _ := ParsePattern("x")
	c.pattern.Store(&patterns{rest: &all, hasRest: true})
	c.SetSelf(c)
	c.song = SongOf(c.Env())
	return c
}

// Pattern is a clock's step pattern read: its hits and how many steps it lasts (then it loops)
type Pattern struct {
	Hits  []Hit
	Steps float64
}

// Hit is a hit of a pattern: where it starts and how long it lasts, in steps from the start of the
// pattern, and the chance it is played (0-1)
type Hit struct {
	At, Len, Chance float64
	Pitches         []float64 // what it plays (a part's: a note, a chord)
	Rel             []Rel     // chord notes of a part: found in the chord of the harmony when they play
}

// ParsePattern reads a step pattern: x (or X, o, 1) a hit, . (or -, _, 0) a rest, a step each; x3
// a roll of 3 hits in a step; [x . x] a group filling a step; *2 after a step or a group: it lasts
// 2 steps (*1.5 too); ? after a hit or a group: its hits by chance, 50% (?25: 25%). Spaces, bar
// lines (|) and new lines are left out
func ParsePattern(text string) (Pattern, error) {
	pr := &patternReader{rs: []rune(text)}
	els, err := pr.sequence(false)
	if err != nil {
		return Pattern{}, err
	}
	if len(els) == 0 {
		return Pattern{}, fmt.Errorf("an empty pattern")
	}
	var p Pattern
	for _, e := range els {
		for _, h := range e.hits {
			p.Hits = append(p.Hits, Hit{At: p.Steps + h.At*e.len, Len: h.Len * e.len, Chance: h.Chance})
		}
		p.Steps += e.len
	}
	return p, nil
}

// patternReader reads a pattern, a rune at a time
type patternReader struct {
	rs []rune
	i  int
}

// element is a step or a group of a pattern: how many steps it lasts, its hits within it (0-1)
type element struct {
	len  float64
	hits []Hit
}

func (pr *patternReader) skip() {
	for pr.i < len(pr.rs) && strings.ContainsRune(" \t\n\r|", pr.rs[pr.i]) {
		pr.i++
	}
}

// number reads a number (digits, a point); "" none
func (pr *patternReader) number() string {
	j := pr.i
	for pr.i < len(pr.rs) && (pr.rs[pr.i] >= '0' && pr.rs[pr.i] <= '9' || pr.rs[pr.i] == '.' && pr.i > j) {
		pr.i++
	}
	return string(pr.rs[j:pr.i])
}

// sequence reads elements until the end (or a ], in a group)
func (pr *patternReader) sequence(inGroup bool) ([]element, error) {
	var els []element
	for {
		pr.skip()
		if pr.i >= len(pr.rs) {
			if inGroup {
				return nil, fmt.Errorf("a group without its ]")
			}
			return els, nil
		}
		r := pr.rs[pr.i]
		if r == ']' {
			if !inGroup {
				return nil, fmt.Errorf("a ] without a group")
			}
			pr.i++
			return els, nil
		}
		e, err := pr.element()
		if err != nil {
			return nil, err
		}
		els = append(els, e)
	}
}

// element reads a step or a group, and what comes after it (*2, ?50)
func (pr *patternReader) element() (element, error) {
	r := pr.rs[pr.i]
	pr.i++
	var e element
	switch {
	case strings.ContainsRune("xXoO1", r):
		n := 1
		if r == '1' { // 1 and 0 a hit and a rest: 1010 is four steps, no roll
		} else if d := pr.number(); d != "" { // a roll
			v, err := strconv.Atoi(d)
			if err != nil || v < 1 || v > 64 {
				return e, fmt.Errorf("x%s: a roll of 1 to 64 hits", d)
			}
			n = v
		}
		e = element{len: 1}
		for k := range n {
			e.hits = append(e.hits, Hit{At: float64(k) / float64(n), Len: 1 / float64(n), Chance: 1})
		}
	case strings.ContainsRune(".-_0", r):
		e = element{len: 1}
	case r == '[':
		inner, err := pr.sequence(true)
		if err != nil {
			return e, err
		}
		if len(inner) == 0 {
			return e, fmt.Errorf("an empty group []")
		}
		total := 0.0
		for _, in := range inner {
			total += in.len
		}
		e = element{len: 1}
		at := 0.0
		for _, in := range inner {
			for _, h := range in.hits {
				e.hits = append(e.hits, Hit{At: (at + h.At*in.len) / total, Len: h.Len * in.len / total, Chance: h.Chance})
			}
			at += in.len
		}
	default:
		return e, fmt.Errorf("%q in a pattern: x is a hit, . a rest, x3 a roll, [x x x] a group, *2 longer, ? by chance", r)
	}
	for pr.i < len(pr.rs) { // what comes after it
		switch pr.rs[pr.i] {
		case '*':
			pr.i++
			d := pr.number()
			v, err := strconv.ParseFloat(d, 64)
			if err != nil || v <= 0 || v > 64 {
				return e, fmt.Errorf("*%s: how many steps it lasts (*2, *0.5)", d)
			}
			e.len = v
		case '?':
			pr.i++
			chance := 0.5
			if d := pr.number(); d != "" {
				v, err := strconv.ParseFloat(d, 64)
				if err != nil || v < 0 || v > 100 {
					return e, fmt.Errorf("?%s: a chance of 0 to 100%%", d)
				}
				chance = v / 100
			}
			for k := range e.hits {
				e.hits[k].Chance *= chance
			}
		default:
			return e, nil
		}
	}
	return e, nil
}

// SetPattern sets the step pattern (from any goroutine)
func (c *Clock) SetPattern(text string) error {
	each, rest, hasRest, err := perSection(text, pointer(ParsePattern))
	if err == nil {
		c.pattern.Store(&patterns{each: each, rest: rest, hasRest: hasRest})
	}
	return err
}

// patterns are a clock's patterns: per section of the song, and for the others
type patterns struct {
	each    map[string]*Pattern
	rest    *Pattern
	hasRest bool
}

// pointer makes a reader of patterns give them by pointer (one stays the same while it plays)
func pointer(read func(string) (Pattern, error)) func(string) (*Pattern, error) {
	return func(text string) (*Pattern, error) {
		p, err := read(text)
		return &p, err
	}
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

// tickAt returns the tick a place in the pattern falls on (in steps from where it starts): every
// second step later by the swing, what is in a step with it
func (c *Clock) tickAt(steps float64) float64 {
	l := c.stepTicks()
	pair := math.Floor(steps / 2)
	f := steps - 2*pair
	t := pair * 2 * l
	if f < 1 {
		t += f * c.Swing * 2 * l
	} else {
		t += c.Swing*2*l + (f-1)*(1-c.Swing)*2*l
	}
	return c.origin + t
}

// hitTick is the tick of the next hit
func (c *Clock) hitTick() float64 {
	p := c.playing
	return c.tickAt(float64(c.cycle)*p.Steps + p.Hits[c.hit].At)
}

// advance moves on to the hit after the next
func (c *Clock) advance() {
	if c.hit++; c.hit >= len(c.playing.Hits) {
		c.hit, c.cycle = 0, c.cycle+1
	}
}

// seek finds the first hit at the tick or after it
func (c *Clock) seek(t float64) {
	p := c.playing
	if p == nil || len(p.Hits) == 0 || p.Steps <= 0 {
		return
	}
	c.cycle = max(0, int64(math.Floor((t-c.origin)/c.stepTicks()/p.Steps))-1)
	c.hit = 0
	for c.hitTick() < t-1e-9 {
		c.advance()
	}
}

// now is the pattern to play at a tick and where it starts: the section's of the song (from the
// section's start), else the pattern (from the start of the patch); nil for none
func (c *Clock) now(tick psalm.CtrlTick) (*Pattern, float64) {
	ps := c.pattern.Load()
	if pos, ok := c.song.At(tick, c.Config().PPQ); ok && !pos.Ended {
		p, has := ps.each[pos.Section.Name]
		if !has {
			if !ps.hasRest {
				return nil, 0 // the pattern says nothing for this section: it rests
			}
			p = ps.rest
		}
		bpb := c.song.Transport().Form().BeatsPerBar
		in := (pos.Beat - float64(pos.Section.StartBar)*bpb) * float64(c.Config().PPQ)
		return p, math.Round(float64(tick.TickTimestamp) - in)
	}
	if !ps.hasRest {
		return nil, 0
	}
	return ps.rest, 0
}

// Play gives the hits of the clock falling on this tick, as the events it would send (for a script that keeps its own clock)
func (c *Clock) Play(tick psalm.CtrlTick, f func(Event)) {
	c.play(tick, func(_ Hit, step int64, seconds float64) { f(Event{Tick: tick, Step: step, Seconds: seconds}) })
}

// Set sets the division (one of Divisions, by name) and the swing (0.5 to 0.75), for a script that keeps its own clock; an error for a division
// that is not one
func (c *Clock) Set(division string, swing float64) error {
	for i, d := range Divisions {
		if d == division {
			c.Division, c.Swing = i, math.Max(0.5, math.Min(swing, 0.75))
			return nil
		}
	}
	return fmt.Errorf("a division is one of %s", strings.Join(Divisions, ", "))
}

// CtrlTick sends the hits falling on this tick
func (c *Clock) CtrlTick(tick psalm.CtrlTick) {
	c.play(tick, func(_ Hit, step int64, seconds float64) {
		c.Send(Event{Tick: tick, Step: step, Seconds: seconds}, 0)
	})
}

// play gives the hits falling on this tick (the step they are in, how long they last); it follows
// the patch's clock (a restart, a jump back, another division, pattern or section starts it over
// from where it is)
func (c *Clock) play(tick psalm.CtrlTick, hit func(h Hit, step int64, seconds float64)) {
	t := float64(tick.TickTimestamp)
	p, origin := c.now(tick)
	if p == nil {
		c.playing, c.started = nil, false
		return
	}
	if !c.started || tick.TickTimestamp < c.last || c.Division != c.div || origin != c.origin || c.playing != p {
		c.started, c.div, c.origin, c.playing = true, c.Division, origin, p
		c.seek(t)
	}
	c.last = tick.TickTimestamp
	if len(p.Hits) == 0 {
		return
	}
	l := c.stepTicks()
	stepSeconds := l * 60 / (c.BPM() * float64(c.Config().PPQ))
	for at := c.hitTick(); at <= t+1e-9; at = c.hitTick() {
		h := p.Hits[c.hit]
		c.advance()
		if at < t-1 { // missed (the tempo jumped): no burst of old hits
			continue
		}
		if chance := h.Chance * c.Chance; chance < 1 && c.Rand().Float64() >= chance {
			continue
		}
		hit(h, int64(math.Floor(at/l+1e-6)), h.Len*stepSeconds)
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
	Order   int
	values  atomic.Pointer[seqValues]
	song    *Song
	gens    map[string]genny.Generator[any] // on the audio thread: per section ("" the others)
	built   *seqValues                      // the values gens were made from
	order   int                             // the order gens were made for
	passing [2]int                          // the section of the song passing (its generator starts over in a new one)
	inSong  bool
}

// NewSequence returns a sequence of values
func NewSequence() *Sequence {
	s := &Sequence{Module: psalm.NewModule(nil, nil), order: -1}
	s.SetSelf(s)
	s.song = SongOf(s.Env())
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
	each, rest, hasRest, err := perSection(text, ParseValues)
	if err == nil {
		s.values.Store(&seqValues{each: each, rest: rest, hasRest: hasRest})
	}
	return err
}

// seqValues are a sequence's values: per section of the song, and for the others
type seqValues struct {
	each    map[string][]any
	rest    []any
	hasRest bool
}

func (s *Sequence) ReceiveMessage(msg psalm.Message) {
	msg.IfIntVal("order", func(o int) { s.Order = max(0, min(o, len(Orders)-1)) })
}

// generator returns the generator of a section's values in the order ("" the others; made again
// when the values or the order changed), nil when it has none
func (s *Sequence) generator(section string) genny.Generator[any] {
	vals := s.values.Load()
	if vals != s.built || s.Order != s.order {
		s.built, s.order, s.gens = vals, s.Order, map[string]genny.Generator[any]{}
	}
	if g, ok := s.gens[section]; ok {
		return g
	}
	v, ok := vals.each[section]
	if section == "" || !ok {
		section, v = "", vals.rest
		if !vals.hasRest {
			return nil
		}
		if g, ok := s.gens[""]; ok {
			return g
		}
	}
	var g genny.Generator[any]
	switch s.Order {
	case 1:
		g = bucket.NewLoop(v...).SetMode(bucket.Indexed)
	case 2:
		g = bucket.NewLoop(v...).SetMode(bucket.Random)
	case 3:
		g = arpeggio.NewLoop(v, arpeggio.Up, arpeggio.Exclusive, false)
	case 4:
		g = arpeggio.NewLoop(v, arpeggio.Converge, arpeggio.None, false)
	case 5:
		g = arpeggio.NewLoop(v, arpeggio.Alternate, arpeggio.None, false)
	case 6:
		back := slices.Clone(v)
		slices.Reverse(back)
		g = sequence.NewLoop(back...)
	default:
		g = sequence.NewLoop(v...)
	}
	s.gens[section] = g
	return g
}

// current is the generator for a trigger at a tick: the section's of the song (from its first value
// when the section starts), or the one of all values without a song
func (s *Sequence) current(tick psalm.CtrlTick) genny.Generator[any] {
	pos, ok := s.song.At(tick, s.Config().PPQ)
	if !ok || pos.Ended {
		s.inSong = false
		return s.generator("")
	}
	g := s.generator(pos.Section.Name)
	if key := sectionKey(pos); !s.inSong || key != s.passing {
		s.inSong, s.passing = true, key
		if g != nil {
			g.Reset()
		}
	}
	return g
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
		g := s.current(ev.Tick)
		if g == nil {
			return // nothing for this section of the song
		}
		s.Send(g.Generate(), 0)
		s.Send(ev, 1)
	case 1:
		for _, g := range s.gens {
			g.Reset()
		}
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

// PatternHelp is what to know to type a clock's pattern
const PatternHelp = "`x` (or `X`, `o`) is a hit, `.` (or `-`, `_`) a rest: a step of the division each; spaces, `|` and new lines are left out\n" +
	"`x3` a roll: 3 hits in the step; `[x x x]` a group filling one step (a triplet), `[x . x x]` with rests\n" +
	"`*2` after a step or a group: it lasts 2 steps (`x*2`, `.*4`, `[x x x]*2` a triplet over two steps); a note of a step lasts as long\n" +
	"`?` after a hit or a group: by chance, half the time (`x?25`: a quarter); the chance below is for every hit\n" +
	"the pattern loops; swing (below) makes every second step later, what is in it too\n" +
	"with a song: a pattern per section, a line `Chorus: x x x x` each (the others, the lines without one); it counts from the section's start\n" +
	"Ctrl+Enter (or Run) takes it; one that can't be read is shown in red, the clock keeps the one before"

// ValuesHelp is what to know to type a sequence's values
const ValuesHelp = "numbers or note names: `60 C3 Eb3 F#4 Bb2` (C3 is 48)\n" +
	"a chord in brackets: `[C3 Eb3 G3]`, played together\n" +
	"spaces, commas or new lines between them\n" +
	"the order (in order, shuffled, up and down...) is below; a trigger on reset starts it over\n" +
	"with a song: values per section, a line `Verse: C3 Eb3 G3` each (the others, the lines without one); a section starts them over\n" +
	"Ctrl+Enter (or Run) takes them"

// perSection reads a text that may say something per section ("Verse: ..." lines): what it says for
// each section and for the others (rest: false when only sections are given)
func perSection[T any](text string, parse func(string) (T, error)) (each map[string]T, rest T, hasRest bool, err error) {
	lines, others := PerSection(text)
	for name, t := range lines {
		v, e := parse(t)
		if e != nil {
			return nil, rest, false, fmt.Errorf("%s: %v", name, e)
		}
		if each == nil {
			each = map[string]T{}
		}
		each[name] = v
	}
	if strings.TrimSpace(others) == "" && len(each) > 0 {
		return each, rest, false, nil
	}
	rest, err = parse(others)
	return each, rest, err == nil, err
}

// Generators are the kinds of generator modules: a clock, a sequence of values, notes, a quantizer
func Generators() []Kind {
	trig := Port{Name: "trigger", Type: Trigger}
	return []Kind{
		{Name: "clock", Group: "Generators", Outs: []Port{trig}, Make: func(string) (*Made, error) {
			c := NewClock()
			return &Made{Module: c, Describe: func(string) params.Set { return c.Describe() },
				Texts: []Text{{Key: "pattern", Label: "pattern", Value: "x", Help: PatternHelp}}, SetText: func(_, v string) error { return c.SetPattern(v) }}, nil
		}},
		{Name: "sequence", Group: "Generators",
			Ins:  []Port{trig, {Name: "reset", Type: Trigger, Index: 1}},
			Outs: []Port{{Name: "value", Type: Value}, {Name: "after", Type: Trigger, Index: 1}},
			Make: func(string) (*Made, error) {
				s := NewSequence()
				return &Made{Module: s, Describe: func(string) params.Set { return s.Describe() },
					Texts: []Text{{Key: "values", Label: "values", Value: "C3 Eb3 G3 Bb3", Help: ValuesHelp}}, SetText: func(_, v string) error { return s.SetValues(v) }}, nil
			}},
		{Name: "notes", Group: "Generators",
			Ins:  []Port{trig, {Name: "pitch", Type: Value, Index: 1}, {Name: "velocity", Type: Value, Index: 2}, {Name: "length (steps)", Type: Value, Index: 3}},
			Outs: []Port{{Name: "notes", Type: Notes}},
			Make: func(string) (*Made, error) {
				n := NewNote()
				return &Made{Module: n, Describe: func(string) params.Set { return n.Describe() },
					Notes: n.SetTargets, Late: n.late}, nil
			}},
		SongKind(), laneKind(), automationKind(), gateKind(), partKind(), harmonyKind(), LogKind(), NoteRollKind(), ArpKind(), LooperKind(),
		{Name: "quantize", Group: "Generators", Ins: []Port{{Name: "in", Type: Value}}, Outs: []Port{{Name: "out", Type: Value}},
			Make: func(string) (*Made, error) {
				q := NewQuantize()
				return &Made{Module: q, Describe: func(string) params.Set { return q.Describe() }}, nil
			}},
	}
}
