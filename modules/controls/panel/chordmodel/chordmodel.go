// Package chordmodel is the editor of a Markov model of chord progressions (tools/chordmodel) and the module that holds it: a matrix of how likely each chord is to follow
// another, to drag with the mouse; a progression drawn from it, a bar or two a chord, whose chords can be locked while the others are drawn again, set by hand, or heard
// (it plays them through the synths it is cabled to); and a button that writes the progression into the harmony of the patch. The editor is a plugin of the panel
// (modules/controls/panel).
package chordmodel

import (
	_ "embed"
	"encoding/json"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	cm "github.com/almerlucke/psalm/tools/chordmodel"
	"github.com/almerlucke/psalm/tools/harmony"
)

//go:embed chordmodel.js
var script string

//go:embed chordmodel.css
var styles string

// Plugin is the plugin (Add uses it)
var Plugin = panel.Plugin{Name: "chordmodel", JS: script, CSS: styles}

// Kind is the kind of its controls
const Kind panel.Kind = "chordmodel"

// Keys and Scales are the keys the chords are put in, and the scales the numerals are read in
var (
	Keys   = []string{"C", "Db", "D", "Eb", "E", "F", "F#", "G", "Ab", "A", "Bb", "B"}
	Scales = []string{"major", "minor", "dorian", "mixolydian", "lydian", "phrygian", "harmonic minor"}
)

// Section is a section of the song, for the editor to choose where the chords are written
type Section struct {
	Name string `json:"name"`
	Bars int    `json:"bars"`
}

// Link is what the module knows of the patch around it: the sections of the song, and a way to write chords into the harmony of the patch (a section's line, "" the whole)
type Link struct {
	Sections func() []Section
	Bake     func(section, chords string) error
	// Where is the section of the song at a tick (false: no song, or it is over); Live uses it to see the song leave a section
	Where func(tick psalm.CtrlTick) (section string, ok bool)
}

// State is everything the editor edits
type State struct {
	Model  *cm.Model `json:"model"`
	Key    int       `json:"key"`
	Scale  int       `json:"scale"`
	Bars   float64   `json:"bars"`   // how many bars a new progression has
	Seed   uint64    `json:"seed"`   // the seed of the next one
	Strip  []cm.Slot `json:"strip"`  // the progression
	Target string    `json:"target"` // the section the chords are written to
	As     string    `json:"as"`     // "symbols" or "numerals"
	Rate   float64   `json:"rate"`   // how much a progression taught counts
	Live   bool      `json:"live"`   // draw the others again (the chords not locked) and write them to the harmony when the song leaves the section it writes to (when the song has none: every Every bars)
	Every  float64   `json:"every"`  // the bars between two new progressions without a song (0: the length of the progression)
	Preset string    `json:"preset"`
}

// Module is the model: a control module that plays its progression when asked to
type Module struct {
	*psalm.Module
	mu      sync.Mutex // the state is changed by one message at a time
	st      atomic.Pointer[State]
	link    atomic.Pointer[Link]
	rev     atomic.Int64
	err     atomic.Pointer[string]
	playing atomic.Bool
	pos     atomic.Uint64 // the bar the preview is at, float64 bits
	targets []psalm.Receiver

	// on the audio thread: the preview
	started bool
	origin  int64
	last    int
	liveIn  string // live: the section the song was in at the last tick, and whether it was in one
	liveHas bool
	liveBar int64
	liveSet bool
	redraws chan struct{} // live: a progression is wanted; a goroutine makes it (never on the audio thread)
	worker  sync.Once
	drawn   atomic.Int64              // how many progressions live has drawn
	hear    atomic.Pointer[[]float64] // a chord asked for to hear (once), played at the next tick
	prev    []float64                 // the voicing of the chord before, for the voice leading
}

// New returns a module with the model of the pop preset in C major and a progression of eight bars
func New() *Module {
	m := &Module{Module: psalm.NewControl()}
	m.SetSelf(m)
	model, _ := cm.Preset("pop")
	st := &State{Model: model, Key: 0, Scale: 0, Bars: 8, Seed: 1, As: "symbols", Rate: 0.25, Preset: "pop"}
	st.Strip = model.Fill(st.Bars, rand.New(rand.NewPCG(1, 2)))
	st.Seed = 2
	m.st.Store(st)
	e := ""
	m.err.Store(&e)
	return m
}

// SetLink tells the module about the patch (the patcher does)
func (m *Module) SetLink(l Link) { m.link.Store(&l) }

// SetTargets sets the synths it plays; call it on the audio thread (psalm.Psalm.Do)
func (m *Module) SetTargets(targets ...psalm.Receiver) { m.targets = targets }

// State returns a copy of the state
func (m *Module) State() State {
	st := *m.st.Load()
	st.Model = st.Model.Clone()
	st.Strip = slices.Clone(st.Strip)
	return st
}

// scale is the scale the numerals are read in
func (s *State) scale() harmony.Scale {
	sc, err := harmony.NewScale(Keys[max(0, min(s.Key, len(Keys)-1))], Scales[max(0, min(s.Scale, len(Scales)-1))])
	if err != nil {
		return harmony.MustScale("C", "major")
	}
	return sc
}

// Text is the progression as the harmony wants it (symbols in the key, or numerals)
func (s *State) Text() string { return s.Model.Text(s.Strip, s.scale(), s.As != "numerals") }

// change makes a changed copy of the state and keeps it
func (m *Module) change(f func(st *State)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.st.Load()
	st := *old
	st.Model = old.Model.Clone()
	st.Strip = slices.Clone(old.Strip)
	f(&st)
	// the strip names states of the model: keep them inside it
	for i := range st.Strip {
		st.Strip[i].State = max(0, min(st.Strip[i].State, len(st.Model.States)-1))
	}
	m.st.Store(&st)
	m.rev.Add(1)
}

func (m *Module) fail(err error) {
	text := ""
	if err != nil {
		text = err.Error()
	}
	m.err.Store(&text)
}

// Generate draws a new progression (the chords that are locked stay when keep is set and the length is the same)
func (m *Module) Generate(seed uint64) {
	m.change(func(st *State) {
		r := rand.New(rand.NewPCG(seed, seed*7+3))
		st.Strip = st.Model.Fill(st.Bars, r)
		st.Seed = seed + 1
	})
}

// Reroll draws the chords that are not locked again
func (m *Module) Reroll(seed uint64) {
	m.change(func(st *State) {
		if len(st.Strip) == 0 {
			st.Strip = st.Model.Fill(st.Bars, rand.New(rand.NewPCG(seed, seed*7+3)))
		} else {
			st.Model.Reroll(st.Strip, rand.New(rand.NewPCG(seed, seed*7+3)))
		}
		st.Seed = seed + 1
	})
}

// SetStrip sets the progression
func (m *Module) SetStrip(slots []cm.Slot) {
	m.change(func(st *State) { st.Strip = slices.Clone(slots) })
}

// SetKey sets the key (an index of Keys) and the scale (an index of Scales) the chords are put in
func (m *Module) SetKey(key, scale int) {
	m.change(func(st *State) { st.Key, st.Scale = max(0, min(key, len(Keys)-1)), max(0, min(scale, len(Scales)-1)) })
}

// SetTarget sets the section the progression is written to (Bake)
func (m *Module) SetTarget(section string) { m.change(func(st *State) { st.Target = section }) }

// Preset replaces the model by a preset (the progression is drawn again)
func (m *Module) Preset(name string) bool {
	model, ok := cm.Preset(name)
	if !ok {
		return false
	}
	m.change(func(st *State) {
		st.Model, st.Preset = model, name
		st.Strip = model.Fill(st.Bars, rand.New(rand.NewPCG(st.Seed, st.Seed*7+3)))
		st.Seed++
	})
	return true
}

// Learn teaches the model progressions typed as numerals
func (m *Module) Learn(text string) {
	var err error
	m.change(func(st *State) { _, err = st.Model.LearnText(text, st.Rate); st.Preset = "" })
	m.fail(err)
}

// Bake writes the progression into the harmony of the patch
func (m *Module) Bake() {
	st := m.st.Load()
	l := m.link.Load()
	if l == nil || l.Bake == nil {
		m.fail(errNoLink)
		return
	}
	m.fail(l.Bake(st.Target, st.Text()))
}

// live asks for a new progression when the song leaves the section the chords are written to (every section when they are written to the whole harmony), and without a song every
// so many bars: the progression is drawn again (the chords locked stay) and written into the harmony, so the next time through the section has other chords
func (m *Module) live(tick psalm.CtrlTick) {
	st := m.st.Load()
	if !st.Live {
		m.liveSet = false
		return
	}
	l := m.link.Load()
	section, has := "", false
	if l != nil && l.Where != nil {
		section, has = l.Where(tick)
	}
	want := false
	if has || m.liveHas {
		if m.liveSet && (m.liveHas && (!has || section != m.liveIn)) && (st.Target == "" || st.Target == m.liveIn) {
			want = true // the song left the section
		}
		m.liveIn, m.liveHas, m.liveSet = section, has, true
	} else {
		total := st.Every
		if total <= 0 {
			for _, s := range st.Strip {
				total += s.Bars
			}
		}
		if total <= 0 {
			return
		}
		bar := int64(float64(tick.TickTimestamp) / float64(m.Config().PPQ) / 4 / total)
		if m.liveSet && bar != m.liveBar {
			want = true
		}
		m.liveBar, m.liveSet = bar, true
	}
	if !want {
		return
	}
	m.worker.Do(func() {
		m.redraws = make(chan struct{}, 1)
		go func() {
			for range m.redraws {
				m.redraw()
			}
		}()
	})
	select {
	case m.redraws <- struct{}{}:
	default:
	}
}

// redraw draws the progression again and writes it to the harmony (off the audio thread)
func (m *Module) redraw() {
	m.Reroll(m.st.Load().Seed)
	m.drawn.Add(1)
	m.Bake()
}

// Drawn is how many progressions live has drawn
func (m *Module) Drawn() int64 { return m.drawn.Load() }

// Hear plays one chord of the progression once, through the synths the notes output goes to (the bass under it)
func (m *Module) Hear(i int) {
	st := m.st.Load()
	if i < 0 || i >= len(st.Strip) {
		return
	}
	c, err := st.scale().Roman(st.Model.States[st.Strip[i].State])
	if err != nil {
		return
	}
	pitches := append([]float64{c.BassPitch(36)}, c.Close(55)...)
	m.hear.Store(&pitches)
}

// SetLive switches live on or off, and how many bars a progression lasts when the patch has no song (0: the length of the progression)
func (m *Module) SetLive(on bool, every float64) {
	m.change(func(st *State) { st.Live, st.Every = on, max(0, min(every, 128)) })
}

// Preview starts or stops playing the progression through the synths the notes output goes to
func (m *Module) Preview(on bool) { m.playing.Store(on) }

type linkError string

func (e linkError) Error() string { return string(e) }

const errNoLink = linkError("this module is not in a patch with a harmony module to write to")

// CtrlTick plays the progression when the preview is on: a chord for its bars, voiced in root position from C3, the progression looping
func (m *Module) CtrlTick(tick psalm.CtrlTick) {
	m.live(tick)
	if p := m.hear.Swap(nil); p != nil { // a chord to hear
		second := m.Env().TickDuration() * float64(m.Config().PPQ)
		for _, pitch := range *p {
			msg := psalm.Message{"command": "trigger", "tick": tick, "duration": 1.6 * second, "amplitude": 0.6, "message": psalm.Message{"pitch": pitch, "velocity": 0.6}}
			for _, t := range m.targets {
				t.ReceiveMessage(msg)
			}
		}
	}
	if !m.playing.Load() {
		m.started = false
		return
	}
	st := m.st.Load()
	total := 0.0
	for _, s := range st.Strip {
		total += s.Bars
	}
	if total <= 0 {
		return
	}
	ppq := float64(m.Config().PPQ)
	if !m.started {
		m.started, m.origin, m.last, m.prev = true, tick.TickTimestamp, -1, nil
	}
	const beatsPerBar = 4.0
	bars := float64(tick.TickTimestamp-m.origin) / ppq / beatsPerBar
	x := math.Mod(bars, total)
	m.pos.Store(math.Float64bits(x))
	i, at := 0, 0.0
	for ; i < len(st.Strip)-1 && x >= at+st.Strip[i].Bars-1e-9; i++ {
		at += st.Strip[i].Bars
	}
	loop := int(bars / total)
	key := loop*1000 + i
	if key == m.last {
		return
	}
	m.last = key
	c, err := st.scale().Roman(st.Model.States[st.Strip[i].State])
	if err != nil {
		return
	}
	left := (at + st.Strip[i].Bars - x) * beatsPerBar // beats of the chord still to go
	voicing := c.Close(55)
	if m.prev != nil {
		voicing = harmony.VoiceLead(m.prev, c)
	}
	m.prev = voicing
	pitches := append([]float64{c.BassPitch(36)}, voicing...)
	for j, p := range pitches {
		vel := 0.6
		if j == 0 {
			vel = 0.7
		}
		msg := psalm.Message{"command": "trigger", "tick": tick, "duration": max(0.05, left*m.Env().TickDuration()*ppq*0.95), "amplitude": vel,
			"message": psalm.Message{"pitch": p, "velocity": vel}}
		for _, t := range m.targets {
			t.ReceiveMessage(msg)
		}
	}
}

// view is what the page gets: the state, the names of the chords, the text, the sections, how far the preview is
type view struct {
	State
	Names    []string  `json:"names"` // the chord of every state of the model, in the key
	Text     string    `json:"text"`
	Keys     []string  `json:"keys"`
	Scales   []string  `json:"scales"`
	Presets  []string  `json:"presets"`
	Sections []Section `json:"sections"`
	Playing  bool      `json:"playing"`
	Pos      float64   `json:"pos"` // bars into the progression while it plays
	Rev      int64     `json:"rev"`
	Error    string    `json:"error"`
}

func (m *Module) view() view {
	st := m.State()
	v := view{State: st, Text: st.Text(), Keys: Keys, Scales: Scales, Presets: cm.PresetNames(), Sections: []Section{}, Playing: m.playing.Load(),
		Pos: math.Float64frombits(m.pos.Load()), Rev: m.rev.Load(), Error: *m.err.Load()}
	sc := st.scale()
	for i := range st.Model.States {
		v.Names = append(v.Names, st.Model.ChordName(i, sc))
	}
	if l := m.link.Load(); l != nil && l.Sections != nil {
		v.Sections = l.Sections()
	}
	return v
}

// Add adds the editor of a model to a panel (and the plugin to its page)
func Add(ctl *panel.Panel, name string, m *Module) *panel.Param {
	ctl.Use(Plugin)
	return ctl.Custom(Kind, name, nil, panel.NoDefault(),
		panel.Live(func() any { return m.view() }),
		panel.OnMessage(func(raw json.RawMessage) { m.message(raw) }))
}

// message is what the page asks: an edit of the model or the progression, a new progression, a preset, teaching, writing to the harmony
func (m *Module) message(raw json.RawMessage) {
	var msg struct {
		Set *struct {
			Model  *cm.Model  `json:"model"`
			Key    *int       `json:"key"`
			Scale  *int       `json:"scale"`
			Bars   *float64   `json:"bars"`
			Strip  *[]cm.Slot `json:"strip"`
			Target *string    `json:"target"`
			As     *string    `json:"as"`
			Rate   *float64   `json:"rate"`
			Live   *bool      `json:"live"`
			Every  *float64   `json:"every"`
		} `json:"set"`
		Generate *uint64 `json:"generate"`
		Reroll   *uint64 `json:"reroll"`
		Preset   *string `json:"preset"`
		Learn    *string `json:"learn"`
		Bake     bool    `json:"bake"`
		Preview  *bool   `json:"preview"`
		Hear     *int    `json:"hear"`
	}
	if json.Unmarshal(raw, &msg) != nil {
		return
	}
	switch {
	case msg.Set != nil:
		s := msg.Set
		m.change(func(st *State) {
			if s.Model != nil && len(s.Model.States) > 1 && len(s.Model.W) == len(s.Model.States) && len(s.Model.Start) == len(s.Model.States) {
				st.Model = s.Model
				st.Preset = ""
			}
			if s.Key != nil {
				st.Key = max(0, min(*s.Key, len(Keys)-1))
			}
			if s.Scale != nil {
				st.Scale = max(0, min(*s.Scale, len(Scales)-1))
			}
			if s.Bars != nil {
				st.Bars = max(0.5, min(*s.Bars, 128))
			}
			if s.Strip != nil {
				st.Strip = *s.Strip
			}
			if s.Target != nil {
				st.Target = *s.Target
			}
			if s.As != nil && (*s.As == "symbols" || *s.As == "numerals") {
				st.As = *s.As
			}
			if s.Rate != nil {
				st.Rate = max(0.01, min(*s.Rate, 2))
			}
			if s.Live != nil {
				st.Live = *s.Live
			}
			if s.Every != nil {
				st.Every = max(0, min(*s.Every, 128))
			}
		})
	case msg.Generate != nil:
		m.Generate(*msg.Generate)
	case msg.Reroll != nil:
		m.Reroll(*msg.Reroll)
	case msg.Preset != nil:
		m.Preset(*msg.Preset)
	case msg.Learn != nil:
		m.Learn(strings.TrimSpace(*msg.Learn))
	case msg.Bake:
		m.Bake()
	case msg.Hear != nil:
		m.Hear(*msg.Hear)
	case msg.Preview != nil:
		m.Preview(*msg.Preview)
	}
}

// StateText is the state as a text a patch keeps; SetStateText gives it back
func (m *Module) StateText() string {
	b, _ := json.Marshal(m.State())
	return string(b)
}

// SetStateText reads a state kept
func (m *Module) SetStateText(text string) error {
	var st State
	if err := json.Unmarshal([]byte(text), &st); err != nil {
		return err
	}
	if st.Model == nil || len(st.Model.States) < 2 || len(st.Model.W) != len(st.Model.States) || len(st.Model.Start) != len(st.Model.States) {
		return errBadState
	}
	if st.As == "" {
		st.As = "symbols"
	}
	m.mu.Lock()
	m.st.Store(&st)
	m.rev.Add(1)
	m.mu.Unlock()
	return nil
}

const errBadState = linkError("a chord model that does not make sense")
