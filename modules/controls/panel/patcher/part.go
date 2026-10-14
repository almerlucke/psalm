package patcher

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/params"
)

// Part plays written notes on the steps of the patch's tempo, as a clock plays its pattern (a
// division, swing, chance; per section of a song too), to the synths its notes cable goes to:
// "C3 Eb3 . G3": a note a step, . a rest, - the note before held a step longer, C3+Eb3+G3 a chord,
// [C3 D3 Eb3] a group filling a step, *2 a note (or group) lasting 2 steps, ? by chance. Settings:
// the clock's, "velocity", "length" (of its step, 0-1: staccato to legato), "transpose"
type Part struct {
	*Clock
	Velocity, Length, Transpose float64
	pending                     []queued
	targets                     []psalm.Receiver
}

// queued is a note to play at the end of a tick
type queued struct {
	tick    psalm.CtrlTick
	pitches []float64
	seconds float64
}

// DefaultPart are the notes of a new part
const DefaultPart = "C3 Eb3 G3 - Bb3 . G3 Eb3"

// NewPart returns a part of sixteenths
func NewPart() *Part {
	p := &Part{Clock: NewClock(), Velocity: 0.8, Length: 0.9}
	p.SetSelf(p)
	_ = p.SetNotes(DefaultPart)
	return p
}

// SetNotes sets the notes written (from any goroutine)
func (p *Part) SetNotes(text string) error {
	each, rest, hasRest, err := perSection(text, pointer(ParsePart))
	if err == nil {
		p.pattern.Store(&patterns{each: each, rest: rest, hasRest: hasRest})
	}
	return err
}

func (p *Part) ReceiveMessage(msg psalm.Message) {
	p.Clock.ReceiveMessage(msg)
	msg.IfFltVal("velocity", func(f float64) { p.Velocity = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("length", func(f float64) { p.Length = math.Max(0.05, math.Min(f, 1)) })
	msg.IfFltVal("transpose", func(f float64) { p.Transpose = f })
}

func (p *Part) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		p.ReceiveMessage(m)
	}
}

// SetTargets sets the synths it plays (on the audio thread)
func (p *Part) SetTargets(targets []psalm.Receiver) { p.targets = targets }

// CtrlTick finds the notes of the tick (played at its end, Late)
func (p *Part) CtrlTick(tick psalm.CtrlTick) {
	p.play(tick, func(h Hit, _ int64, seconds float64) {
		p.pending = append(p.pending, queued{tick: tick, pitches: h.Pitches, seconds: math.Max(0.005, seconds*p.Length)})
	})
}

// late plays the notes of the tick
func (p *Part) late(psalm.CtrlTick) {
	for _, n := range p.pending {
		for _, pitch := range n.pitches {
			msg := psalm.Message{"command": "trigger", "tick": n.tick, "duration": n.seconds, "amplitude": p.Velocity,
				"message": psalm.Message{"pitch": pitch + p.Transpose, "velocity": p.Velocity}}
			for _, t := range p.targets {
				t.ReceiveMessage(msg)
			}
		}
	}
	p.pending = p.pending[:0]
}

// Describe describes the part's settings: the clock's, its velocity, length and transposition
func (p *Part) Describe() params.Set {
	set := p.Clock.Describe()
	set.Name = "part"
	set.Params = append(set.Params,
		params.K("Notes", "velocity", 0, 1, p.Velocity),
		params.K("Notes", "length", 0.05, 1, p.Length, params.Label("length (of its step)")),
		params.K("Notes", "transpose", -36, 36, p.Transpose, params.Step(1), params.Unit("st")))
	return set
}

// ParsePart reads written notes: a note a step (C3, F#4, Bb2, 60), . (or _) a rest, - the note
// before held a step longer, C3+Eb3+G3 a chord, [C3 D3 Eb3] a group filling a step, *2 after a note
// or a group: it lasts 2 steps, ? (or ?25) by chance. Spaces, bar lines and new lines between them
func ParsePart(text string) (Pattern, error) {
	toks := partTokens(text)
	i := 0
	els, err := partSequence(toks, &i, false)
	if err != nil {
		return Pattern{}, err
	}
	if len(els) == 0 {
		return Pattern{}, fmt.Errorf("no notes: C3 Eb3 . G3")
	}
	hits, steps := assemble(els)
	return Pattern{Hits: hits, Steps: steps}, nil
}

// partTokens splits written notes: words, and the brackets of groups (a ] keeps what follows it:
// "]*2")
func partTokens(text string) []string {
	var toks []string
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			toks = append(toks, word.String())
			word.Reset()
		}
	}
	rs := []rune(text)
	for i := 0; i < len(rs); i++ {
		switch r := rs[i]; {
		case r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '|':
			flush()
		case r == '[':
			flush()
			toks = append(toks, "[")
		case r == ']':
			flush()
			word.WriteRune(']')
		default:
			word.WriteRune(r)
		}
	}
	flush()
	return toks
}

// partElement is a note, a rest, a tie or a group of written notes: how many steps it lasts, its
// hits within it (0-1)
type partElement struct {
	len  float64
	hits []Hit
	tie  bool
}

// partSequence reads elements until the end (or a ], in a group)
func partSequence(toks []string, i *int, inGroup bool) ([]partElement, error) {
	var els []partElement
	for *i < len(toks) {
		tok := toks[*i]
		*i++
		var e partElement
		var mods string
		switch {
		case strings.HasPrefix(tok, "]"):
			if !inGroup {
				return nil, fmt.Errorf("a ] without a group")
			}
			if len(els) == 0 {
				return nil, fmt.Errorf("an empty group []")
			}
			// the group's modifiers are the caller's: put them back
			toks[*i-1] = tok[1:]
			*i--
			return els, nil
		case tok == "[":
			inner, err := partSequence(toks, i, true)
			if err != nil {
				return nil, err
			}
			if *i >= len(toks) {
				return nil, fmt.Errorf("a group without its ]")
			}
			mods = toks[*i] // what came after its ]
			*i++
			hits, total := assemble(inner)
			e = partElement{len: 1}
			for _, h := range hits {
				h.At, h.Len = h.At/total, h.Len/total
				e.hits = append(e.hits, h)
			}
		default:
			base := tok
			if k := strings.IndexAny(tok, "*?"); k >= 0 {
				base, mods = tok[:k], tok[k:]
			}
			switch base {
			case ".", "_":
				e = partElement{len: 1}
			case "-":
				e = partElement{len: 1, tie: true}
			default:
				var pitches []float64
				for _, name := range strings.Split(base, "+") {
					p, err := score.Pitch(name)
					if err != nil {
						return nil, fmt.Errorf("%q: a note (C3, F#4, Bb2, 60), . a rest, - held longer", name)
					}
					pitches = append(pitches, p)
				}
				e = partElement{len: 1, hits: []Hit{{At: 0, Len: 1, Chance: 1, Pitches: pitches}}}
			}
		}
		if err := partModifiers(&e, mods); err != nil {
			return nil, err
		}
		els = append(els, e)
	}
	if inGroup {
		return nil, fmt.Errorf("a group without its ]")
	}
	return els, nil
}

// partModifiers reads what comes after a note or a group: *2 (how many steps), ?25 (the chance)
func partModifiers(e *partElement, mods string) error {
	for mods != "" {
		op := mods[0]
		mods = mods[1:]
		j := 0
		for j < len(mods) && (mods[j] >= '0' && mods[j] <= '9' || mods[j] == '.') {
			j++
		}
		num := mods[:j]
		mods = mods[j:]
		switch op {
		case '*':
			v, err := strconv.ParseFloat(num, 64)
			if err != nil || v <= 0 || v > 64 {
				return fmt.Errorf("*%s: how many steps it lasts (*2, *0.5)", num)
			}
			e.len = v
		case '?':
			chance := 0.5
			if num != "" {
				v, err := strconv.ParseFloat(num, 64)
				if err != nil || v < 0 || v > 100 {
					return fmt.Errorf("?%s: a chance of 0 to 100%%", num)
				}
				chance = v / 100
			}
			for k := range e.hits {
				e.hits[k].Chance *= chance
			}
		default:
			return fmt.Errorf("%q after a note: *2 longer, ? by chance", string(op))
		}
	}
	return nil
}

// assemble lays elements one after the other: their hits in steps from the first, and how many
// steps they last; a tie holds the note before it longer
func assemble(els []partElement) ([]Hit, float64) {
	var hits []Hit
	pos := 0.0
	last := -1
	for _, e := range els {
		if e.tie {
			if last >= 0 && math.Abs(hits[last].At+hits[last].Len-pos) < 1e-9 {
				hits[last].Len += e.len
			}
			pos += e.len
			continue
		}
		for _, h := range e.hits {
			h.At, h.Len = pos+h.At*e.len, h.Len*e.len
			hits = append(hits, h)
			last = len(hits) - 1
		}
		if len(e.hits) == 0 {
			last = -1 // a rest: nothing to hold
		}
		pos += e.len
	}
	return hits, pos
}

// PartHelp is what to know to write a part
const PartHelp = "notes a step each: `C3 Eb3 G3 Bb3` (C3 is 48; F#4, Bb2, 60 too), `.` a rest, `-` the note before held a step longer\n" +
	"`C3+Eb3+G3` a chord; `[C3 D3 Eb3]` a group filling a step (a triplet); `*2` after a note or a group: it lasts 2 steps; `?` by chance (`?25`)\n" +
	"the division, swing and chance below, as a clock's; velocity, length (of a step: staccato to legato), transpose\n" +
	"with a song: notes per section, a line `Chorus: F3 Ab3 C4 -` each (the others, the lines without one); from the section's start\n" +
	"its notes cable to a synth; Ctrl+Enter (or Run) takes them"

// partKind is the kind of parts
func partKind() Kind {
	return Kind{Name: "part", Group: "Song", Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			p := NewPart()
			return &Made{Module: p, Describe: func(string) params.Set { return p.Describe() },
				Texts:   []Text{{Key: "notes", Label: "notes", Value: DefaultPart, Help: PartHelp}},
				SetText: func(_, v string) error { return p.SetNotes(v) },
				Notes:   p.SetTargets, Late: p.late}, nil
		}}
}
