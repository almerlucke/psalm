package patcher

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/tools/harmony"
)

// Harmony is the chord track of the patch: the chords of the piece, a number of bars each, per section of the song. A part with chord-relative notes (@1 the root, @3 the fifth...)
// asks it which chord it is at the moment of a note, so a figure written once follows the harmony, and its outputs tell the others: the chord as a value (an array of pitches,
// the chord in root position from the lowest note it is voiced at), its bass note, and a trigger when the chord changes. One harmony a patch (the last made), like the song.
//
// The chords are typed as symbols (Dm7, F#m7b5, Bb/D, see tools/harmony) or roman numerals in the key and scale of its settings (i, VI, V7), a number of bars each (`Dm7*2`, the
// default is one, `*0.5` half a bar), `-` holds the chord before it a bar longer, `.` is no chord. They loop over the bars of the section; per section of a song, a line `Chorus:
// F Am Dm Bb` each (the others, the lines without one); without a song they loop from the start of the patch
type Harmony struct {
	*psalm.Module
	Key, Scale int     // the key and the scale the roman numerals are in
	Low        float64 // the lowest note of the chord it sends (the root at or above it)
	text       atomic.Value
	prog       atomic.Pointer[harmonies]
	song       *Song
	now        atomic.Pointer[string] // the chord at the last tick, for the page

	// on the audio thread: the chord last sent
	sent    string
	hasSent bool
}

// progression is the chords of a section (or of the others): a chord and the bars it lasts (a nil chord: none)
type progression struct {
	chords []*harmony.Chord
	bars   []float64
	total  float64
}

// harmonies are the progressions of a harmony: per section, and for the others
type harmonies struct {
	each    map[string]*progression
	rest    *progression
	hasRest bool
}

var harmonyTracks sync.Map // *psalm.Env -> *Harmony

// HarmonyOf returns the harmony of the patch of an environment (a module's: m.Env()); nil when the patch has none
func HarmonyOf(env *psalm.Env) *Harmony {
	v, ok := harmonyTracks.Load(env)
	if !ok {
		return nil
	}
	return v.(*Harmony)
}

// DefaultChords are the chords of a new harmony
const DefaultChords = "Dm7 G7 Cmaj7 Fmaj7"

// NewHarmony returns a harmony of four chords in C major, voiced from C3
func NewHarmony() *Harmony {
	h := &Harmony{Module: psalm.NewControl(), Low: 48}
	h.Scale = indexOf(Scales, "major")
	h.SetSelf(h)
	h.song = SongOf(h.Env())
	harmonyTracks.Store(h.Env(), h)
	_ = h.SetChords(DefaultChords)
	return h
}

func indexOf(list []string, name string) int {
	for i, n := range list {
		if n == name {
			return i
		}
	}
	return 0
}

// SetChords sets the chords typed (from any goroutine); an error leaves the ones before
func (h *Harmony) SetChords(text string) error {
	each, rest, hasRest, err := h.read(text)
	if err != nil {
		return err
	}
	h.text.Store(text)
	h.prog.Store(&harmonies{each: each, rest: rest, hasRest: hasRest})
	return nil
}

func (h *Harmony) read(text string) (map[string]*progression, *progression, bool, error) {
	scale, err := harmony.NewScale(Keys[h.Key], Scales[h.Scale])
	if err != nil {
		return nil, nil, false, err
	}
	return perSection(text, func(t string) (*progression, error) { return ParseChords(t, scale) })
}

// ParseChords reads chords: symbols or roman numerals (in a scale), a number of bars each (`*2`), `-` the chord before held a bar longer, `.` no chord; spaces, `|` and new lines
// between them
func ParseChords(text string, scale harmony.Scale) (*progression, error) {
	text = strings.NewReplacer("|", " ", "\n", " ", "\t", " ").Replace(text)
	var p progression
	for _, tok := range strings.Fields(text) {
		bars := 1.0
		name := tok
		if k := strings.LastIndex(tok, "*"); k >= 0 {
			v, err := strconv.ParseFloat(tok[k+1:], 64)
			if err != nil || v <= 0 || v > 256 {
				return nil, fmt.Errorf("%s: how many bars the chord lasts (Dm7*2, *0.5)", tok)
			}
			name, bars = tok[:k], v
		}
		switch name {
		case "", "-":
			if len(p.chords) == 0 {
				return nil, fmt.Errorf("a - without a chord before it")
			}
			p.bars[len(p.bars)-1] += bars
		case ".":
			p.chords, p.bars = append(p.chords, nil), append(p.bars, bars)
		default:
			var c harmony.Chord
			var err error
			if strings.ContainsRune("ABCDEFG", rune(name[0])) {
				c, err = harmony.ParseChord(name)
			} else {
				c, err = scale.Roman(name)
			}
			if err != nil {
				return nil, fmt.Errorf("%s: a chord (Dm7, Bb/D, F#m7b5) or a roman numeral (i, VI, V7)", name)
			}
			p.chords, p.bars = append(p.chords, &c), append(p.bars, bars)
		}
		p.total += bars
	}
	if len(p.chords) == 0 {
		return nil, fmt.Errorf("no chords: Dm7 G7 Cmaj7")
	}
	return &p, nil
}

// at is the chord x bars into the progression (it loops); nil for none
func (p *progression) at(x float64) *harmony.Chord {
	x = math.Mod(x, p.total)
	if x < 0 {
		x += p.total
	}
	for i, b := range p.bars {
		if x < b-1e-9 {
			return p.chords[i]
		}
		x -= b
	}
	return p.chords[len(p.chords)-1]
}

// ChordAt is the chord at a tick (on the audio thread): the chord of the section the song is in, or the chords from the start of the patch when there is no song; false for none
func (h *Harmony) ChordAt(tick psalm.CtrlTick) (harmony.Chord, bool) {
	ps := h.prog.Load()
	ppq := h.Config().PPQ
	var p *progression
	var bars float64
	if pos, ok := h.song.At(tick, ppq); ok {
		if pos.Ended {
			return harmony.Chord{}, false
		}
		var has bool
		if p, has = ps.each[pos.Section.Name]; !has {
			if !ps.hasRest {
				return harmony.Chord{}, false
			}
			p = ps.rest
		}
		bpb := h.song.Transport().Form().BeatsPerBar
		bars = (pos.Beat - float64(pos.Section.StartBar)*bpb) / bpb
	} else {
		if !ps.hasRest {
			return harmony.Chord{}, false
		}
		p = ps.rest
		bars = float64(tick.TickTimestamp) / float64(ppq) / 4
	}
	c := p.at(bars + 1e-9)
	if c == nil {
		return harmony.Chord{}, false
	}
	return *c, true
}

// Voiced is the chord as it is sent: in root position, the root at or above the lowest note
func (h *Harmony) Voiced(c harmony.Chord) []float64 { return c.Close(h.Low) }

func (h *Harmony) ReceiveMessage(msg psalm.Message) {
	changed := false
	msg.IfIntVal("key", func(k int) { h.Key, changed = max(0, min(k, len(Keys)-1)), true })
	msg.IfIntVal("scale", func(s int) { h.Scale, changed = max(0, min(s, len(Scales)-1)), true })
	msg.IfFltVal("low", func(f float64) { h.Low = math.Max(0, math.Min(f, 96)); h.hasSent = false })
	if changed {
		if t, ok := h.text.Load().(string); ok {
			_ = h.SetChords(t) // the numerals mean other chords in another key
		}
	}
}

func (h *Harmony) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		h.ReceiveMessage(m)
	}
}

// CtrlTick sends the chord when it changes: the chord (an array of pitches) on output 0, its bass on 1, a trigger on 2
func (h *Harmony) CtrlTick(tick psalm.CtrlTick) {
	c, ok := h.ChordAt(tick)
	name := ""
	if ok {
		name = c.Name
	}
	h.now.Store(&name)
	if h.hasSent && name == h.sent {
		return
	}
	h.sent, h.hasSent = name, true
	if !ok {
		return
	}
	h.Send(h.Voiced(c), 0)
	h.Send(c.BassPitch(h.Low-12), 1)
	h.Send(Event{Tick: tick}, 2)
}

// Current is the chord at the last tick, "" for none (from any goroutine)
func (h *Harmony) Current() string {
	if n := h.now.Load(); n != nil {
		return *n
	}
	return ""
}

// Describe describes the harmony's settings
func (h *Harmony) Describe() params.Set {
	return params.Set{Name: "harmony", Params: []params.Param{
		params.C("Harmony", "key", Keys, h.Key, params.Label("key (of the numerals)")),
		params.C("Harmony", "scale", Scales, h.Scale, params.Label("scale (of the numerals)")),
		params.K("Harmony", "low", 24, 84, h.Low, params.Step(1), params.Label("lowest note")),
	}}
}

// ChordsHelp is what to know to write the chords
const ChordsHelp = "chords as symbols (`Dm7`, `F#m7b5`, `Bb/D`, `Csus4`) or roman numerals in the key and scale below (`i`, `VI`, `V7`, `bVII`), a bar each\n" +
	"`*2` after a chord: it lasts 2 bars (`*0.5` half a bar); `-` holds the chord before it a bar longer; `.` is no chord\n" +
	"they loop over the bars of the section; with a song: chords per section, a line `Chorus: F Am Dm Bb` each (the others, the lines without one); without one from the start\n" +
	"parts with chord notes (`@1 @3 @5`, `@b` the bass, `@c` the chord) follow them; the outputs: the chord (an array of pitches), its bass and a trigger when it changes"

// harmonyKind is the kind of harmonies
func harmonyKind() Kind {
	return Kind{Name: "harmony", Group: "Song",
		Outs: []Port{{Name: "chord", Type: Value}, {Name: "bass", Type: Value, Index: 1}, {Name: "change", Type: Trigger, Index: 2}},
		Make: func(string) (*Made, error) {
			h := NewHarmony()
			return &Made{Module: h, Describe: func(string) params.Set { return h.Describe() },
				Texts:   []Text{{Key: "chords", Label: "chords", Value: DefaultChords, Help: ChordsHelp}},
				SetText: func(_, v string) error { return h.SetChords(v) },
				Status:  h.Current}, nil
		}}
}
