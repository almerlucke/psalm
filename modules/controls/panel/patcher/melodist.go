package patcher

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/melodyui"
	"github.com/almerlucke/psalm/modules/controls/panel/noteroll"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/melody"
)

// Melodist makes a melody while the song plays: a pass (a section of the song, else so many beats) at a time, over the chords of the harmony, in its scale, in phrases that repeat
// and vary (a form: A A' B A, or a grammar). The next pass is made ahead, off the audio thread, from the one before: how much of it changes is `mutate` (0: the same melody every
// time through the section, 1: a new one), so the chorus the second time is the chorus again, changed a little. `freeze` keeps what it plays, a second voice (`counter`) goes with
// it, and the buttons of its window draw it again or keep what it plays in a roll
type Melodist struct {
	*psalm.Module
	song *Song

	mu  sync.Mutex
	cfg MelodistConfig
	gr  string // the grammar, when the form is one

	req        chan melodyReq
	ready      atomic.Pointer[[]*melodyPass] // the passes made, the newest last
	last       atomic.Pointer[melodyPass]    // the pass playing (for Keep)
	status     atomic.Pointer[string]
	grammarErr atomic.Pointer[string]
	kept       atomic.Pointer[string]
	made       atomic.Int64
	closed     atomic.Bool

	plays atomic.Pointer[[]string] // the sections it plays in (nil: all of them)
	pos   atomic.Uint64            // where it is in the pass playing, in beats (float64 bits)
	cur   atomic.Pointer[string]   // the section of the pass playing

	// on the audio thread
	targets  []psalm.Receiver
	n        int64 // the pass it is in, counted from the first
	section  string
	began    int64
	inPass   bool
	lastTick int64
	asked    int64 // the newest pass asked for
}

// MelodistConfig are the settings of a melodist
type MelodistConfig struct {
	Density, Rest, Steps, Chord float64
	Contour, Rhythm, Form       int // indexes of MelodyContours, MelodyRhythms, MelodyForms
	Phrase                      int // an index of MelodyPhrases
	Low, High                   float64
	Key, Scale                  int // Scale 0: the scale of the harmony; else one of Scales (Scale-1), in Key
	Seed                        int
	Mutate                      float64
	Freeze, Counter, Just       bool // Just: the notes in just intonation (pure ratios from the tonic, the key of the scale) instead of equal temperament
	Velocity                    float64
	Dynamics                    float64 // 0 to 1: how much the notes differ in loudness
	Grid                        float64 // the shortest note in beats (0.25): 1 or 2 makes a slow tune of long notes
	Length                      float64 // beats a pass without a song
}

// the choices of a melodist
var (
	MelodyContours = []string{"free", "arch", "rise", "fall", "wave"}
	MelodyRhythms  = []string{"mixed", "even", "syncopated", "dotted", "triplet"}
	MelodyForms    = []string{"free", "AA'", "ABAB", "AABA", "call", "grammar"}
	MelodyPhrases  = []float64{2, 4, 8, 16}
)

// justCents are the twelve notes above the tonic in just intonation, in cents from the equal temperament: 16/15, 9/8, 6/5, 5/4, 4/3, 45/32, 3/2, 8/5, 5/3, 9/5, 15/8
var justCents = [12]float64{0, 11.73, 3.91, 15.64, -13.69, -1.96, -9.78, 1.96, 13.69, -15.64, 17.6, -11.73}

// melodyPass is a pass made: the section and the number it is for, and its notes (beats from the start of the pass)
type melodyPass struct {
	n       int64
	section string
	beats   float64
	notes   []melody.Note
	counter []melody.Note
	tonic   int // the pitch class of the tonic (for just intonation)
	just    bool
}

// melodyReq is a pass to make; clear sends the memory of the sections away (a new take)
type melodyReq struct {
	n       int64
	section string
	beats   float64
	clear   bool
}

// NewMelodist returns a melodist of the settings that are good to start with
func NewMelodist() *Melodist {
	m := &Melodist{Module: psalm.NewControl(), req: make(chan melodyReq, 8), lastTick: -1, asked: -1,
		cfg: MelodistConfig{Density: 0.5, Rest: 0.15, Steps: 0.7, Chord: 0.7, Form: 3, Phrase: 2, Low: 60, High: 84, Seed: 1, Mutate: 0.25, Velocity: 0.8, Grid: 0.25, Length: 16}}
	m.SetSelf(m)
	m.song = SongOf(m.Env())
	go m.work()
	return m
}

func (m *Melodist) config() MelodistConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// SetGrammar sets the grammar of the form (used when the form is "grammar")
func (m *Melodist) SetGrammar(text string) error {
	if strings.TrimSpace(text) != "" {
		if err := melody.ParseForm(text); err != nil {
			return err
		}
	}
	m.mu.Lock()
	m.gr = text
	m.mu.Unlock()
	return nil
}

func (m *Melodist) ReceiveMessage(msg psalm.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := &m.cfg
	f01 := func(dst *float64) func(float64) { return func(v float64) { *dst = math.Max(0, math.Min(v, 1)) } }
	msg.IfFltVal("density", f01(&c.Density))
	msg.IfFltVal("rest", f01(&c.Rest))
	msg.IfFltVal("steps", f01(&c.Steps))
	msg.IfFltVal("chord", f01(&c.Chord))
	msg.IfFltVal("mutate", f01(&c.Mutate))
	msg.IfFltVal("velocity", func(v float64) { c.Velocity = math.Max(0.05, math.Min(v, 1)) })
	msg.IfFltVal("dynamics", f01(&c.Dynamics))
	msg.IfIntVal("contour", func(i int) { c.Contour = max(0, min(i, len(MelodyContours)-1)) })
	msg.IfIntVal("rhythm", func(i int) { c.Rhythm = max(0, min(i, len(MelodyRhythms)-1)) })
	msg.IfIntVal("form", func(i int) { c.Form = max(0, min(i, len(MelodyForms)-1)) })
	msg.IfIntVal("phrase", func(i int) { c.Phrase = max(0, min(i, len(MelodyPhrases)-1)) })
	msg.IfIntVal("key", func(i int) { c.Key = max(0, min(i, len(Keys)-1)) })
	msg.IfIntVal("scale", func(i int) { c.Scale = max(0, min(i, len(Scales))) })
	msg.IfFltVal("low", func(v float64) { c.Low = math.Max(0, math.Min(v, 120)) })
	msg.IfFltVal("high", func(v float64) { c.High = math.Max(0, math.Min(v, 127)) })
	msg.IfFltVal("seed", func(v float64) { c.Seed = int(math.Max(0, math.Min(v, 9999))) })
	msg.IfFltVal("length", func(v float64) { c.Length = math.Max(2, math.Min(v, 128)) })
	msg.IfFltVal("grid", func(v float64) { c.Grid = math.Max(0.25, math.Min(v, 4)) })
	msg.IfBoolVal("freeze", func(b bool) { c.Freeze = b })
	msg.IfBoolVal("counter", func(b bool) { c.Counter = b })
	msg.IfBoolVal("just", func(b bool) { c.Just = b })
}

func (m *Melodist) Receive(v any, _ int) {
	if msg, ok := psalm.MsgVal(v); ok {
		m.ReceiveMessage(msg)
	}
}

// SetTargets sets the synths it plays (on the audio thread)
func (m *Melodist) SetTargets(targets []psalm.Receiver) { m.targets = targets }

// Describe describes the settings
func (m *Melodist) Describe() params.Set {
	c := m.config()
	scales := append([]string{"the harmony's"}, Scales...)
	phrases := []string{"2 beats", "4 beats", "8 beats", "16 beats"}
	return params.Set{Name: "melodist", Params: []params.Param{
		params.C("Melody", "form", MelodyForms, c.Form),
		params.C("Melody", "contour", MelodyContours, c.Contour),
		params.C("Melody", "rhythm", MelodyRhythms, c.Rhythm),
		params.C("Melody", "phrase", phrases, c.Phrase),
		params.K("Melody", "density", 0, 1, c.Density, params.Label("busy")),
		params.K("Melody", "rest", 0, 1, c.Rest, params.Label("rests")),
		params.K("Melody", "steps", 0, 1, c.Steps, params.Label("smooth")),
		params.K("Melody", "chord", 0, 1, c.Chord, params.Label("chord tones")),
		params.K("Melody", "grid", 0.25, 4, c.Grid, params.Exp(), params.Unit("beats"), params.Label("shortest note")),
		params.K("Range", "low", 24, 100, c.Low, params.Step(1)),
		params.K("Range", "high", 40, 120, c.High, params.Step(1)),
		params.C("Range", "key", Keys, c.Key),
		params.C("Range", "scale", scales, c.Scale),
		params.K("Live", "mutate", 0, 1, c.Mutate),
		params.K("Live", "seed", 0, 9999, float64(c.Seed), params.Step(1)),
		params.S("Live", "freeze", c.Freeze),
		params.S("Live", "counter", c.Counter, params.Label("second voice")),
		params.S("Range", "just", c.Just, params.Label("just intonation")),
		params.K("Live", "length", 2, 128, c.Length, params.Step(1), params.Label("beats a pass (no song)")),
		params.K("Live", "velocity", 0.05, 1, c.Velocity),
		params.K("Live", "dynamics", 0, 1, c.Dynamics, params.Label("dynamics (notes of different loudness)")),
	}}
}

// the work: passes are made off the audio thread, from the one of the same section before (mutated), or new
func (m *Melodist) work() {
	last := map[string]*melodyPass{}
	for r := range m.req {
		if r.clear {
			last = map[string]*melodyPass{}
			continue
		}
		if cur := m.ready.Load(); cur != nil {
			dup := false
			for _, p := range *cur {
				dup = dup || p.n == r.n && p.section == r.section
			}
			if dup {
				continue
			}
		}
		cfg := m.config()
		m.mu.Lock()
		gr := m.gr
		m.mu.Unlock()
		pass := m.makePass(cfg, gr, r, last[r.section])
		if pass == nil {
			continue
		}
		last[r.section] = pass
		m.made.Add(1)
		var keep []*melodyPass
		if cur := m.ready.Load(); cur != nil {
			keep = append(keep, *cur...)
		}
		keep = append(keep, pass)
		if len(keep) > 6 {
			keep = keep[len(keep)-6:]
		}
		m.ready.Store(&keep)
	}
}

// scaleOf is the scale a melody is in: the harmony's, or one of the settings
func (m *Melodist) scaleOf(c MelodistConfig) harmony.Scale {
	key, name := Keys[c.Key], "major"
	if c.Scale > 0 {
		name = Scales[c.Scale-1]
	} else if h := HarmonyOf(m.Env()); h != nil {
		key, name = Keys[h.Key], Scales[h.Scale]
	}
	if s, err := harmony.NewScale(key, name); err == nil {
		return s
	}
	return harmony.MustScale("C", "major")
}

func (m *Melodist) makePass(c MelodistConfig, grammar string, r melodyReq, before *melodyPass) *melodyPass {
	scale := m.scaleOf(c)
	bpb := 4.0
	if tr := m.song.Transport(); tr != nil {
		bpb = tr.Form().BeatsPerBar
	}
	p := melody.Params{Scale: scale, Beats: r.beats, Low: int(c.Low), High: int(math.Max(c.High, c.Low+4)), Density: c.Density, Rest: c.Rest, Steps: c.Steps, Chord: c.Chord,
		Grid: c.Grid, Rhythm: MelodyRhythms[c.Rhythm], Contour: MelodyContours[c.Contour], Form: MelodyForms[c.Form], Phrase: MelodyPhrases[c.Phrase], Velocity: c.Velocity, Dynamics: c.Dynamics}
	if p.Form == "grammar" {
		p.Grammar = grammar
	}
	if h := HarmonyOf(m.Env()); h != nil {
		for _, sp := range h.Spans(r.section, r.beats, bpb) {
			root := 0
			if len(sp.Pcs) > 0 {
				root = sp.Pcs[0]
			}
			p.Chords = append(p.Chords, melody.Span{At: sp.At, Len: sp.Len, Pcs: sp.Pcs, Root: root})
		}
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(r.section))
	base := uint64(c.Seed)*7919 + h.Sum64()%100003
	var notes []melody.Note
	switch {
	case before != nil && before.beats == r.beats && (c.Freeze || c.Mutate < 1):
		if c.Freeze || c.Mutate == 0 {
			notes = before.notes
		} else {
			notes = melody.Mutate(before.notes, p, c.Mutate, base+uint64(r.n)*31+1)
		}
	default:
		p.Seed = base + uint64(r.n)
		notes = melody.Generate(p)
	}
	tonic := 0
	if c.Scale > 0 {
		tonic, _ = harmony.PitchClass(Keys[c.Key])
	} else if hm := HarmonyOf(m.Env()); hm != nil {
		tonic, _ = harmony.PitchClass(Keys[hm.Key])
	}
	out := &melodyPass{n: r.n, section: r.section, beats: r.beats, notes: notes, tonic: tonic, just: c.Just}
	if c.Counter && len(notes) > 0 {
		lo, hi := 127, 0
		for _, n := range notes {
			lo, hi = min(lo, n.Key), max(hi, n.Key)
		}
		cp := melody.CounterParams{Scale: scale, Chords: p.Chords, Low: max(24, lo-17), High: max(hi-3, max(24, lo-17)+7), Every: 2, Velocity: c.Velocity * 0.8}
		out.counter = melody.Counter(notes, cp)
	}
	return out
}

// CtrlTick plays the notes that start on the tick: the pass of the section the song is in, made ahead
func (m *Melodist) CtrlTick(tick psalm.CtrlTick) {
	ppq := m.Config().PPQ
	t := tick.TickTimestamp
	section, beats, began := "", 0.0, int64(0)
	var next string
	nextBeats := 0.0
	if pos, ok := m.song.At(tick, ppq); ok && !pos.Ended {
		form := m.song.Transport().Form()
		bpb := form.BeatsPerBar
		section, beats = pos.Section.Name, float64(pos.Section.Bars)*bpb
		began = int64(math.Round(float64(t) - (pos.Beat-float64(pos.Section.StartBar)*bpb)*float64(ppq)))
		if len(form.Sections) > 0 {
			nx := form.Sections[(pos.Section.Index+1)%len(form.Sections)]
			next, nextBeats = nx.Name, float64(nx.Bars)*bpb
		}
	} else {
		if ok { // the song is over
			return
		}
		cfg := m.config()
		beats = cfg.Length
		loop := int64(math.Round(beats * float64(ppq)))
		began = (t / loop) * loop
		next, nextBeats = "", beats
	}
	if !m.inPass || section != m.section || began != m.began || t < m.lastTick {
		m.n++
		m.section, m.began, m.inPass = section, began, true
		m.lastTick = -1
	}
	if m.asked < m.n {
		m.asked = m.n
		m.ask(melodyReq{n: m.n, section: section, beats: beats})
		m.ask(melodyReq{n: m.n + 1, section: next, beats: nextBeats})
	}
	var pass *melodyPass
	if cur := m.ready.Load(); cur != nil {
		for i := len(*cur) - 1; i >= 0; i-- {
			if p := (*cur)[i]; p.n == m.n && p.section == section {
				pass = p
				break
			}
		}
	}
	if pass == nil {
		return
	}
	if m.last.Load() != pass {
		m.last.Store(pass)
		st := fmt.Sprintf("%s · pass %d · %d notes", section, m.n, len(pass.notes))
		if section == "" {
			st = fmt.Sprintf("pass %d · %d notes", m.n, len(pass.notes))
		}
		m.status.Store(&st)
	}
	from := m.lastTick + 1
	if m.lastTick < 0 {
		from = t
	}
	m.lastTick = t
	m.pos.Store(math.Float64bits(float64(t-began) / float64(ppq)))
	m.cur.Store(&section)
	if pl := m.plays.Load(); pl != nil && !slices.Contains(*pl, section) { // not a section it plays in
		return
	}
	second := m.Env().TickDuration() * float64(ppq)
	for _, list := range [][]melody.Note{pass.notes, pass.counter} {
		for _, n := range list {
			start := began + int64(math.Round(n.Start*float64(ppq)))
			if start < from || start > t {
				continue
			}
			pitch := float64(n.Key)
			if pass.just {
				pitch += justCents[((n.Key-pass.tonic)%12+12)%12] / 100
			}
			msg := psalm.Message{"command": "trigger", "tick": tick, "duration": math.Max(0.005, n.Length*second), "amplitude": n.Velocity,
				"message": psalm.Message{"pitch": pitch, "velocity": n.Velocity}}
			for _, target := range m.targets {
				target.ReceiveMessage(msg)
			}
		}
	}
}

// ask puts a pass to make in the queue (the audio thread never waits)
func (m *Melodist) ask(r melodyReq) {
	if r.beats <= 0 || m.closed.Load() {
		return
	}
	select {
	case m.req <- r:
	default:
	}
}

// Passes is how many passes have been made
func (m *Melodist) Passes() int64 { return m.made.Load() }

// Current is the pass that plays now (nil before the first)
func (m *Melodist) Current() (section string, beats float64, notes []melody.Note) {
	if p := m.last.Load(); p != nil {
		return p.section, p.beats, p.notes
	}
	return "", 0, nil
}

// Status is what the module says on its tile
func (m *Melodist) Status() string {
	if s := m.status.Load(); s != nil {
		return *s
	}
	return ""
}

// DrawAgain forgets what the sections had: the next pass of every section is new
func (m *Melodist) DrawAgain() {
	m.mu.Lock()
	m.cfg.Seed = (m.cfg.Seed + 1) % 10000
	m.mu.Unlock()
	if !m.closed.Load() {
		m.req <- melodyReq{clear: true}
	}
}

// Keep writes the pass that plays into the clip of its section in the first note editor of the patch (the loop for a pass without a song)
func (m *Melodist) Keep() error {
	section, beats, notes := m.Current()
	if len(notes) == 0 {
		return fmt.Errorf("nothing has played yet")
	}
	v, ok := patchers.Load(m.Env())
	if !ok {
		return fmt.Errorf("this module is not in a patch")
	}
	pt := v.(*Patcher)
	pt.mu.Lock()
	var roll *noteroll.Roll
	for _, n := range pt.nodes {
		if r, ok := n.made.Module.(*noteroll.Roll); ok {
			roll = r
			break
		}
	}
	pt.mu.Unlock()
	if roll == nil {
		return fmt.Errorf("the patch has no note editor to keep the melody in: add one")
	}
	out := make([]noteroll.Note, len(notes))
	for i, n := range notes {
		out[i] = noteroll.Note{Start: n.Start, Length: n.Length, Key: n.Key, Velocity: n.Velocity}
	}
	roll.SetClip(section, out, beats)
	return nil
}

// MelodyHelp is what to know to write the grammar
const MelodyHelp = "the form of the melody, when it is a grammar: rules that rewrite themselves, a rule a line, `S -> A A' B A~`\n" +
	"the phrases are the letters A to D; after one: `'` a variation, `~` upside down in the scale, `<` backwards, `^` and `v` a step up and down, `?` ends open\n" +
	"choices with `|` and a weight with `*3`: `Open -> A A' *3 | A B`; an L-system: `axiom: A`, `iterate: 3`, `A -> A B A'`\n" +
	"left empty, the form chosen below is used"

func melodistKind() Kind {
	return Kind{Name: "melodist", Group: "Generators", Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			m := NewMelodist()
			return &Made{Module: m, Describe: func(string) params.Set { return m.Describe() },
				State:    func() (string, error) { return m.StateText(), nil },
				SetState: m.SetStateText,
				Edit: func(page *panel.Panel, name, _ string) Settings {
					page.Section("Melody")
					melodyui.Add(page, name, m)
					return m
				},
				Notes:  m.SetTargets,
				Status: m.Status,
				Close: func() {
					if m.closed.CompareAndSwap(false, true) {
						close(m.req)
					}
				}}, nil
		}}
}

// --- the editor ---

// melodyPresets are the settings of a few kinds of melody, to start from
var melodyPresetNames = []string{"Music box tune", "Answer, lower", "Glitter", "Slow lead", "Busy line", "Sparse bells"}
var melodyPresets = map[string]map[string]float64{
	"Music box tune": {"form": 4, "contour": 3, "rhythm": 1, "phrase": 2, "grid": 1.5, "density": 0.1, "rest": 0.4, "steps": 0.92, "chord": 0.85, "low": 70, "high": 88, "mutate": 0.12, "dynamics": 0.8, "velocity": 0.42},
	"Answer, lower":  {"form": 4, "contour": 2, "rhythm": 1, "phrase": 2, "grid": 2, "density": 0.1, "rest": 0.55, "steps": 0.92, "chord": 0.85, "low": 60, "high": 79, "mutate": 0.3, "dynamics": 0.8, "velocity": 0.36},
	"Glitter":        {"form": 4, "contour": 3, "rhythm": 3, "phrase": 1, "grid": 0.5, "density": 0.2, "rest": 0.55, "steps": 0.8, "chord": 0.7, "low": 84, "high": 103, "mutate": 0.6, "dynamics": 0.5, "velocity": 0.35},
	"Slow lead":      {"form": 3, "contour": 1, "rhythm": 0, "phrase": 2, "grid": 1, "density": 0.3, "rest": 0.3, "steps": 0.8, "chord": 0.8, "low": 62, "high": 84, "mutate": 0.2, "dynamics": 0.6, "velocity": 0.6},
	"Busy line":      {"form": 3, "contour": 4, "rhythm": 2, "phrase": 1, "grid": 0.25, "density": 0.7, "rest": 0.15, "steps": 0.6, "chord": 0.6, "low": 60, "high": 84, "mutate": 0.4, "dynamics": 0.4, "velocity": 0.7},
	"Sparse bells":   {"form": 4, "contour": 3, "rhythm": 1, "phrase": 3, "grid": 2, "density": 0.08, "rest": 0.65, "steps": 0.9, "chord": 0.9, "low": 76, "high": 96, "mutate": 0.5, "dynamics": 0.7, "velocity": 0.4},
}

// configMap is the settings as numbers by their keys: what the editor shows and what a patch keeps of it
func configMap(c MelodistConfig) map[string]float64 {
	b2f := func(b bool) float64 {
		if b {
			return 1
		}
		return 0
	}
	return map[string]float64{"density": c.Density, "rest": c.Rest, "steps": c.Steps, "chord": c.Chord, "form": float64(c.Form), "contour": float64(c.Contour), "rhythm": float64(c.Rhythm),
		"phrase": float64(c.Phrase), "low": c.Low, "high": c.High, "key": float64(c.Key), "scale": float64(c.Scale), "seed": float64(c.Seed), "mutate": c.Mutate, "freeze": b2f(c.Freeze),
		"counter": b2f(c.Counter), "just": b2f(c.Just), "velocity": c.Velocity, "dynamics": c.Dynamics, "grid": c.Grid, "length": c.Length}
}

// melodistView is what the editor gets
type melodistView struct {
	Config     map[string]float64 `json:"cfg"`
	Grammar    string             `json:"grammar"`
	GrammarErr string             `json:"grammarErr"`
	Sections   []string           `json:"sections"`
	Plays      []string           `json:"plays"` // the sections it plays in (empty: all)
	Section    string             `json:"section"`
	Pass       int64              `json:"pass"`
	Beats      float64            `json:"beats"`
	Notes      [][4]float64       `json:"notes"`
	Counter    [][4]float64       `json:"counter"`
	Chords     []melodyChord      `json:"chords"`
	Pos        float64            `json:"pos"`
	BPS        float64            `json:"bps"`
	Presets    []string           `json:"presets"`
	Keys       []string           `json:"keys"`
	Scales     []string           `json:"scales"`
	Forms      []string           `json:"forms"`
	Contours   []string           `json:"contours"`
	Rhythms    []string           `json:"rhythms"`
	Status     string             `json:"status"`
	Kept       string             `json:"kept"`
}

type melodyChord struct {
	At   float64 `json:"at"`
	Len  float64 `json:"len"`
	Name string  `json:"name"`
}

// View is what the editor gets: the settings, the pass that plays with the chords over it, where it is in it, the sections and the ones it plays in
func (m *Melodist) View() any {
	m.mu.Lock()
	gr := m.gr
	m.mu.Unlock()
	v := melodistView{Config: configMap(m.config()), Grammar: gr, Sections: []string{}, Plays: []string{}, Presets: melodyPresetNames, Keys: Keys, Scales: append([]string{"the harmony's"}, Scales...),
		Forms: MelodyForms, Contours: MelodyContours, Rhythms: MelodyRhythms, Status: m.Status(), Pos: -1, BPS: m.Env().BPM() / 60, Notes: [][4]float64{}, Counter: [][4]float64{}, Chords: []melodyChord{}}
	if e := m.grammarErr.Load(); e != nil {
		v.GrammarErr = *e
	}
	if k := m.kept.Load(); k != nil {
		v.Kept = *k
	}
	bpb := 4.0
	if tr := m.song.Transport(); tr != nil {
		form := tr.Form()
		bpb = form.BeatsPerBar
		for _, s := range form.Sections {
			v.Sections = append(v.Sections, s.Name)
		}
	}
	if pl := m.plays.Load(); pl != nil {
		v.Plays = *pl
	}
	if p := m.last.Load(); p != nil {
		v.Section, v.Pass, v.Beats = p.section, p.n, p.beats
		for _, n := range p.notes {
			v.Notes = append(v.Notes, [4]float64{n.Start, n.Length, float64(n.Key), n.Velocity})
		}
		for _, n := range p.counter {
			v.Counter = append(v.Counter, [4]float64{n.Start, n.Length, float64(n.Key), n.Velocity})
		}
		if h := HarmonyOf(m.Env()); h != nil {
			for _, sp := range h.Spans(p.section, p.beats, bpb) {
				v.Chords = append(v.Chords, melodyChord{At: sp.At, Len: sp.Len, Name: sp.Name})
			}
		}
		if cur := m.cur.Load(); cur != nil && *cur == p.section {
			v.Pos = math.Float64frombits(m.pos.Load())
		}
	}
	return v
}

// Message is what the editor asks: settings to change (by key), the grammar, the sections it plays in, a preset, a button
func (m *Melodist) Message(raw json.RawMessage) {
	var msg struct {
		Set     map[string]float64 `json:"set"`
		Grammar *string            `json:"grammar"`
		Plays   *[]string          `json:"plays"`
		Preset  string             `json:"preset"`
		Action  string             `json:"action"`
	}
	if json.Unmarshal(raw, &msg) != nil {
		return
	}
	apply := func(set map[string]float64) {
		pm := psalm.Message{}
		for k, v := range set {
			pm[k] = v
		}
		m.ReceiveMessage(pm)
	}
	switch {
	case msg.Set != nil:
		apply(msg.Set)
	case msg.Grammar != nil:
		e := ""
		if err := m.SetGrammar(*msg.Grammar); err != nil {
			e = err.Error()
		}
		m.grammarErr.Store(&e)
	case msg.Plays != nil:
		if pl := slices.Clone(*msg.Plays); len(pl) > 0 {
			m.plays.Store(&pl)
		} else {
			m.plays.Store(nil)
		}
	case msg.Preset != "":
		if set, ok := melodyPresets[msg.Preset]; ok {
			apply(set)
		}
	case msg.Action == "again":
		m.DrawAgain()
	case msg.Action == "keep":
		e := "kept in the note editor"
		if err := m.Keep(); err != nil {
			e = err.Error()
		}
		m.kept.Store(&e)
	}
}

// SetPlays sets the sections it plays in (none: all of them); a pass is made for every section all the same, so the tune is there when it comes in
func (m *Melodist) SetPlays(sections ...string) {
	if len(sections) == 0 {
		m.plays.Store(nil)
		return
	}
	pl := slices.Clone(sections)
	m.plays.Store(&pl)
}

// melodistState is what a patch keeps of a melodist: the settings, the grammar and the sections it plays in
type melodistState struct {
	Config  map[string]float64 `json:"cfg"`
	Grammar string             `json:"grammar"`
	Plays   []string           `json:"plays,omitempty"`
}

// StateText is the settings, the grammar and the sections as a text a patch keeps; SetStateText gives them back
func (m *Melodist) StateText() string {
	m.mu.Lock()
	gr := m.gr
	m.mu.Unlock()
	st := melodistState{Config: configMap(m.config()), Grammar: gr}
	if pl := m.plays.Load(); pl != nil {
		st.Plays = *pl
	}
	b, _ := json.Marshal(st)
	return string(b)
}

// SetStateText reads a state kept
func (m *Melodist) SetStateText(text string) error {
	var st melodistState
	if err := json.Unmarshal([]byte(text), &st); err != nil {
		return err
	}
	pm := psalm.Message{}
	for k, v := range st.Config {
		pm[k] = v
	}
	m.ReceiveMessage(pm)
	if err := m.SetGrammar(st.Grammar); err != nil {
		return err
	}
	if len(st.Plays) > 0 {
		pl := slices.Clone(st.Plays)
		m.plays.Store(&pl)
	} else {
		m.plays.Store(nil)
	}
	return nil
}

// Values and Apply are the settings of the editor as a patch and a preset keep them
func (m *Melodist) Values() map[string]float64 { return configMap(m.config()) }
func (m *Melodist) Apply(v map[string]float64) {
	pm := psalm.Message{}
	for k, x := range v {
		pm[k] = x
	}
	m.ReceiveMessage(pm)
}
