package patcher

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/rhythm"
)

// Part plays written notes on the steps of the patch's tempo, as a clock plays its pattern (a
// division, swing, chance; per section of a song too), to the synths its notes cable goes to:
// "C3 Eb3 . G3": a note a step, . a rest, - the note before held a step longer, C3+Eb3+G3 a chord,
// [C3 D3 Eb3] a group filling a step, *2 a note (or group) lasting 2 steps, ? by chance. Settings:
// the clock's, "velocity", "length" (of its step, 0-1: staccato to legato), "transpose"
type Part struct {
	*Clock
	Velocity, Length, Transpose float64
	Register, Bass              float64 // the lowest note of the root of the chord notes (@1), and of the bass note (@b)
	pending                     []queued
	voiced                      []float64 // on the audio thread: the voicing of the chord last played by @v, and the chord
	voicedChord                 string
	targets                     []psalm.Receiver
}

// Rel is a note written relative to the chord of the harmony at the moment it plays: @1 the root, @2 the next tone of the chord (the third), @3 the fifth... the tones in the
// order of the chord's symbol (a Dm9: root, third, fifth, seventh, ninth) and the same an octave up after them; @-1 the tone before the root (an octave down); @b the bass
// note; @c the whole chord; ' after it an octave up, , an octave down
type Rel struct {
	Kind   byte // 'n' a tone of the chord by its number, 'b' the bass, 'c' the whole chord
	N      int  // the number of the tone, from 1
	Octave int
}

// queued is a note to play at the end of a tick
type queued struct {
	tick    psalm.CtrlTick
	pitches []float64
	seconds float64
	vel     float64 // a factor from the lanes and the groove (0: none)
	pitch   float64
}

// resolve gives the pitches of a chord note for a chord
func (r Rel) resolve(c harmony.Chord, register, bass float64) []float64 {
	up := float64(12 * r.Octave)
	switch r.Kind {
	case 'b':
		return []float64{c.BassPitch(bass) + up}
	case 'c':
		out := c.Close(register)
		for i := range out {
			out[i] += up
		}
		return out
	}
	root := math.Floor(register)
	root += float64(((c.Root-int(root))%12 + 12) % 12)
	n := len(c.Intervals)
	i := r.N - 1
	if r.N < 0 {
		i = r.N
	}
	oct := int(math.Floor(float64(i) / float64(n)))
	return []float64{root + float64(c.Intervals[i-oct*n]) + float64(12*oct) + up}
}

// pitchesOf is what a hit plays at a tick: its notes, and its chord notes in the chord of the harmony then (none of them when there is no chord)
func (p *Part) pitchesOf(h Hit, tick psalm.CtrlTick) []float64 {
	if len(h.Rel) == 0 {
		return h.Pitches
	}
	out := append([]float64(nil), h.Pitches...)
	if hm := HarmonyOf(p.Env()); hm != nil {
		if c, ok := hm.ChordAt(tick); ok {
			for _, r := range h.Rel {
				if r.Kind == 'v' {
					out = append(out, p.voiceLed(c, r.Octave)...)
					continue
				}
				out = append(out, r.resolve(c, p.Register, p.Bass)...)
			}
		}
	}
	return out
}

// voiceLed is the chord voiced as close as it can to the chord @v played before (the first time close from the register); it stays as it is while the chord does
func (p *Part) voiceLed(c harmony.Chord, octave int) []float64 {
	if c.Name != p.voicedChord || len(p.voiced) == 0 {
		if len(p.voiced) == 0 {
			p.voiced = c.Close(p.Register)
		} else {
			p.voiced = harmony.VoiceLead(p.voiced, c)
		}
		p.voicedChord = c.Name
	}
	out := make([]float64, len(p.voiced))
	for i, v := range p.voiced {
		out[i] = v + float64(12*octave)
	}
	return out
}

// DefaultPart are the notes of a new part
const DefaultPart = "C3 Eb3 G3 - Bb3 . G3 Eb3"

// NewPart returns a part of sixteenths
func NewPart() *Part {
	p := &Part{Clock: NewClock(), Velocity: 0.8, Length: 0.9, Register: 48, Bass: 36}
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
	msg.IfFltVal("register", func(f float64) { p.Register = math.Max(0, math.Min(f, 96)) })
	msg.IfFltVal("bass", func(f float64) { p.Bass = math.Max(0, math.Min(f, 96)) })
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
		if pitches := p.pitchesOf(h, tick); len(pitches) > 0 {
			p.pending = append(p.pending, queued{tick: tick, pitches: pitches, seconds: math.Max(0.005, seconds*p.Length), vel: h.Vel, pitch: h.Pitch})
		}
	})
}

// late plays the notes of the tick
func (p *Part) late(psalm.CtrlTick) {
	for _, n := range p.pending {
		vel := p.Velocity
		if n.vel > 0 {
			vel = math.Min(1, vel*n.vel)
		}
		for _, pitch := range n.pitches {
			msg := psalm.Message{"command": "trigger", "tick": n.tick, "duration": n.seconds, "amplitude": vel,
				"message": psalm.Message{"pitch": pitch + p.Transpose + n.pitch, "velocity": vel}}
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
		params.K("Notes", "transpose", -36, 36, p.Transpose, params.Step(1), params.Unit("st")),
		params.K("Chord notes", "register", 24, 84, p.Register, params.Step(1), params.Label("root (@1) from")),
		params.K("Chord notes", "bass", 12, 60, p.Bass, params.Step(1), params.Label("bass (@b) from")))
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
			if k := strings.IndexAny(tok, "*?^%"); k >= 0 {
				base, mods = tok[:k], tok[k:]
			}
			switch base {
			case ".", "_":
				e = partElement{len: 1}
			case "-":
				e = partElement{len: 1, tie: true}
			default:
				var pitches []float64
				var rels []Rel
				for _, name := range strings.Split(base, "+") {
					if strings.HasPrefix(name, "@") {
						r, err := parseRel(name)
						if err != nil {
							return nil, err
						}
						rels = append(rels, r)
						continue
					}
					p, err := score.Pitch(name)
					if err != nil {
						return nil, fmt.Errorf("%q: a note (C3, F#4, Bb2, 60), @3 a note of the chord, . a rest, - held longer", name)
					}
					pitches = append(pitches, p)
				}
				e = partElement{len: 1, hits: []Hit{{At: 0, Len: 1, Chance: 1, Pitches: pitches, Rel: rels}}}
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

// parseRel reads a chord note: @1 (a tone of the chord by its number, @-1 the one below the root), @b the bass, @c the chord, then ' for each octave up and , for each down
func parseRel(tok string) (Rel, error) {
	body := strings.TrimPrefix(tok, "@")
	r := Rel{Kind: 'n'}
	for strings.HasSuffix(body, "'") || strings.HasSuffix(body, ",") {
		if body[len(body)-1] == '\'' {
			r.Octave++
		} else {
			r.Octave--
		}
		body = body[:len(body)-1]
	}
	switch body {
	case "b":
		r.Kind = 'b'
	case "c":
		r.Kind = 'c'
	case "v":
		r.Kind = 'v'
	default:
		n, err := strconv.Atoi(body)
		if err != nil || n == 0 || n < -24 || n > 24 {
			return Rel{}, fmt.Errorf("%q: @1 the root, @3 the fifth, @b the bass, @c the chord, @v the chord voice led (' an octave up, , down)", tok)
		}
		r.N = n // @-1 is the tone before the root: the last of the chord, an octave down
	}
	return r, nil
}

// partModifiers reads what comes after a note or a group: *2 (how many steps), ?25 (the chance)
func partModifiers(e *partElement, mods string) error {
	for mods != "" {
		op := mods[0]
		mods = mods[1:]
		j := 0
		if op == '%' { // a condition: up to the next modifier
			for j < len(mods) && !strings.ContainsRune("*?^%", rune(mods[j])) {
				j++
			}
		} else {
			for j < len(mods) && (mods[j] >= '0' && mods[j] <= '9' || mods[j] == '.') {
				j++
			}
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
		case '%':
			cond, err := rhythm.ParseCond(num)
			if err != nil {
				return err
			}
			for k := range e.hits {
				e.hits[k].Cond = cond
			}
		case '^':
			v, err := strconv.Atoi(num)
			if err != nil || v < 2 || v > 16 {
				return fmt.Errorf("^%s: a ratchet of 2 to 16 hits in the note", num)
			}
			var hits []Hit
			for _, h := range e.hits {
				for k := range v {
					r := h
					r.At, r.Len = h.At+float64(k)*h.Len/float64(v), h.Len/float64(v)
					hits = append(hits, r)
				}
			}
			e.hits = hits
		default:
			return fmt.Errorf("%q after a note: *2 longer, ? by chance, ^3 a ratchet, %%2/4 on the second of every four", string(op))
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
	"`C3+Eb3+G3` a chord; `[C3 D3 Eb3]` a group filling a step (a triplet); `*2` after a note or a group: it lasts 2 steps; `?` by chance (`?25`); `^3` a ratchet: 3 hits in the note; `%2/4` only the second time through in every four, `%fill` in the last bar of a section, `%first`, `!` for the other way (`%!fill`)\n" +
	"with a harmony module: chord notes, found in the chord at the moment they play: `@1` the root, `@2` the next tone of the chord (the third), `@3` the fifth, `@4` the seventh (a triad: the\n" +
	"root an octave up), `@-1` the tone below the root, `@b` the bass, `@c` the whole chord, `@v` the chord voice led from the one before (each note moves as little as it can, the same shape of chord in a smooth line); `'` after one an octave up, `,` down; `@1 @3 @5 @3`, `@c*4`; `C3+@3` together\n" +
	"the division, swing and chance below, as a clock's; velocity, length (of a step: staccato to legato), transpose\n" +
	"with a song: notes per section, a line `Chorus: F3 Ab3 C4 -` each (the others, the lines without one); from the section's start\n" +
	"its notes cable to a synth; Ctrl+Enter (or Run) takes them"

// partKind is the kind of parts
func partKind() Kind {
	return Kind{Name: "part", Group: "Song", Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			p := NewPart()
			return &Made{Module: p, Describe: func(string) params.Set { return p.Describe() },
				Texts: []Text{{Key: "notes", Label: "notes", Value: DefaultPart, Help: PartHelp}, {Key: "lanes", Label: "lanes", Lines: 5, Value: "", Help: LanesHelp}},
				SetText: func(k, v string) error {
					if k == "lanes" {
						return p.SetLanes(v)
					}
					return p.SetNotes(v)
				},
				Notes: p.SetTargets, Late: p.late}, nil
		}}
}
