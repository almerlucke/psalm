// Package noteroll is a note editor, a piano roll to draw notes on, and the module that plays
// them: a loop of notes of a length in beats that plays with the tempo of the patch and sends its
// notes to the synths it is given (psalm.Receiver, as a keyboard does). The roll is a plugin of
// the panel (modules/controls/panel): click to put a note, drag its end to make it longer, drag it
// to move it, right click (or Delete) to take it away.
//
//	roll := noteroll.New(p)
//	roll.SetTargets(synth)
//	noteroll.Add(ctl, "Notes", roll)
package noteroll

import (
	_ "embed"
	"encoding/json"
	"maps"
	"math"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/melody"
	"github.com/almerlucke/psalm/tools/rhythm"
)

//go:embed noteroll.js
var script string

//go:embed noteroll.css
var styles string

// Plugin is the plugin (Add uses it)
var Plugin = panel.Plugin{Name: "noteroll", JS: script, CSS: styles}

// Kind is the kind of its controls
const Kind panel.Kind = "noteroll"

// MaxBeats is the longest a loop can be
const MaxBeats = 64

// Note is a note of the roll: where it starts and how long it lasts (in beats), its key (MIDI) and
// velocity (0-1)
type Note struct {
	Start, Length float64
	Key           int
	Velocity      float64
}

// Clip is a loop of notes: the notes (sorted by start) and its length in beats
type Clip struct {
	Notes  []Note
	Length float64
}

// Where tells where the song is at a tick: the section it is in and the tick the section began on (false: the patch has no song)
type Where func(tick psalm.CtrlTick) (section string, began int64, ok bool)

// Section is a section of the song as the editor has it: its name and its length in beats
type Section struct {
	Name  string  `json:"name"`
	Beats float64 `json:"beats"`
}

// ChordSpan is a chord of the harmony over a part of a clip, in beats, with the pitch classes of its tones
type ChordSpan struct {
	At   float64 `json:"at"`
	Len  float64 `json:"len"`
	Name string  `json:"name"`
	Pcs  []int   `json:"pcs"`
}

// Link is what a roll knows of the patch it is in: where the song is (a roll then plays the clip of the section the song is in, from the start of the section), the
// sections (the editor has a clip for each), and the chords of a section for the editor to show (nil: it knows none)
type Link struct {
	Where    Where
	Sections func() []Section
	Chords   func(section string, beats float64) []ChordSpan
	Scale    func() (root int, mode string, ok bool) // the key and scale of the harmony (false: it has none)
}

// Roll is a set of loops of notes, playing with the tempo of the patch: the loop (the clip of no section) goes round from the start of the patch, and the song (when it has
// one and the roll has a clip for the section it is in) plays the clip of its section from the section's start. Its notes and length are safe to change from any
// goroutine (the page, your own code); it plays on the audio thread
type Roll struct {
	*psalm.Module
	clips    atomic.Pointer[map[string]*Clip] // by section; "" is the loop that plays when the song has no clip for the section; replaced, never changed
	view     atomic.Pointer[string]           // the clip the editor shows
	link     atomic.Pointer[Link]
	pos      atomic.Uint64          // where it plays, in beats, float64 bits
	playing  atomic.Pointer[string] // the clip that plays: its section, "" the loop
	targets  []psalm.Receiver       // audio thread
	last     int64                  // the last tick played
	lastClip string                 // the clip it played then

	heard   sync.Mutex // notes the editor asks to hear while it works, played at the next tick
	waiting []Note
	hearing atomic.Int32

	gen     atomic.Pointer[Candidates] // what the editor asked to have made, the last time
	preview atomic.Pointer[sequence]   // notes the editor asks to hear as they are, once
	pvCur   *sequence                  // audio thread: the one playing, from the tick it started on
	pvStart int64
	pvLast  int64
}

// New creates a roll of 8 beats (two bars) without notes; add it to a patch as a control module
func New(p psalm.Patcher) *Roll {
	r := newRoll()
	r.AddTo(p)
	return r
}

func newRoll() *Roll {
	r := &Roll{Module: psalm.NewControl(), last: -1}
	r.SetSelf(r)
	empty := ""
	r.view.Store(&empty)
	r.playing.Store(&empty)
	m := map[string]*Clip{"": {Length: 8}}
	r.clips.Store(&m)
	return r
}

// NewRoll creates a roll that isn't in a patch yet (the patcher adds it)
func NewRoll() *Roll { return newRoll() }

// SetTargets sets the synths it plays; call it on the audio thread (psalm.Psalm.Do)
func (r *Roll) SetTargets(targets ...psalm.Receiver) { r.targets = targets }

// Link tells the roll about the song and the chords of the patch (the patcher does)
func (r *Roll) Link(l Link) { r.link.Store(&l) }

// makeClip puts notes in order, within the loop, and made to fit its limits
func makeClip(notes []Note, length float64) *Clip {
	length = math.Max(1, math.Min(math.Round(length*1000)/1000, MaxBeats))
	ns := make([]Note, 0, len(notes))
	for _, n := range notes {
		n.Key = max(0, min(n.Key, 127))
		n.Velocity = math.Max(0, math.Min(n.Velocity, 1))
		n.Start = math.Max(0, math.Round(n.Start*1000)/1000)
		n.Length = math.Max(0.01, math.Round(n.Length*1000)/1000)
		if n.Start >= length {
			continue
		}
		ns = append(ns, n)
	}
	slices.SortStableFunc(ns, func(a, b Note) int {
		if a.Start != b.Start {
			if a.Start < b.Start {
				return -1
			}
			return 1
		}
		return a.Key - b.Key
	})
	return &Clip{Notes: ns, Length: length}
}

// with replaces the clips by a changed copy
func (r *Roll) with(change func(m map[string]*Clip)) {
	for {
		old := r.clips.Load()
		m := maps.Clone(*old)
		change(m)
		if r.clips.CompareAndSwap(old, &m) {
			return
		}
	}
}

// Set sets the notes and the length of the loop in beats (1 to MaxBeats): the loop of no section. Notes are put in order, within the loop, and made to fit its limits
func (r *Roll) Set(notes []Note, length float64) { r.SetClip("", notes, length) }

// SetClip sets the clip of a section ("" is the loop)
func (r *Roll) SetClip(section string, notes []Note, length float64) {
	c := makeClip(notes, length)
	r.with(func(m map[string]*Clip) { m[section] = c })
}

// DropClip takes the clip of a section away (the loop can't go: it is emptied)
func (r *Roll) DropClip(section string) {
	r.with(func(m map[string]*Clip) {
		if section == "" {
			m[""] = &Clip{Length: m[""].Length}
		} else {
			delete(m, section)
		}
	})
}

// CopyClip puts a copy of a clip in other sections
func (r *Roll) CopyClip(from string, to ...string) {
	src, ok := (*r.clips.Load())[from]
	if !ok {
		return
	}
	r.with(func(m map[string]*Clip) {
		for _, name := range to {
			if name != from {
				m[name] = &Clip{Notes: slices.Clone(src.Notes), Length: src.Length}
			}
		}
	})
}

// Clip returns the clip of a section ("" the loop); false when the section has none
func (r *Roll) Clip(section string) (Clip, bool) {
	c, ok := (*r.clips.Load())[section]
	if !ok {
		return Clip{}, false
	}
	return Clip{Notes: slices.Clone(c.Notes), Length: c.Length}, true
}

// Clips returns the names of the sections that have a clip (and "" for the loop, always first)
func (r *Roll) Clips() []string {
	names := slices.Sorted(maps.Keys(*r.clips.Load()))
	return names
}

// Notes returns the notes of the loop, in order of start
func (r *Roll) Notes() []Note { c, _ := r.Clip(""); return c.Notes }

// Length returns the length of the loop in beats
func (r *Roll) Length() float64 { c, _ := r.Clip(""); return c.Length }

// Position returns where the clip that plays is, in beats
func (r *Roll) Position() float64 { return math.Float64frombits(r.pos.Load()) }

// Playing returns the clip that plays: the section it belongs to, "" for the loop
func (r *Roll) Playing() string { return *r.playing.Load() }

// sequence is notes to play once, from when the editor asks: beats from the start
type sequence struct {
	Notes  []Note
	Length float64
}

// Candidates are melodies made for the editor: the id of the request they answer, and each as notes of the clip ([start, length, key, velocity])
type Candidates struct {
	ID    int            `json:"id"`
	Notes [][][4]float64 `json:"notes"`
	Err   string         `json:"err,omitempty"` // what is wrong with the grammar
}

// GenParams are what the editor asks a melody to be (tools/melody): zero is the default; Mode "" is the scale of the harmony of the patch, when it has one
type GenParams struct {
	Root     int     `json:"root"`
	Mode     string  `json:"mode"`
	Low      int     `json:"low"`
	High     int     `json:"high"`
	Density  float64 `json:"density"`
	Rest     float64 `json:"rest"`
	Steps    float64 `json:"steps"`
	Chord    float64 `json:"chord"`
	Rhythm   string  `json:"rhythm"`
	Contour  string  `json:"contour"`
	Form     string  `json:"form"`
	Grammar  string  `json:"grammar"` // rules that write the form (tools/melody Expand); used when Form is "grammar"
	Phrase   float64 `json:"phrase"`
	Grid     float64 `json:"grid"`
	Velocity float64 `json:"velocity"`
}

// Generate makes melodies for a stretch of the clip shown (from beat from to beat to), count of them from a seed on, with the chords of the harmony under them; they are what the
// editor gets next (Candidates)
func (r *Roll) Generate(id int, from, to float64, count int, seed uint64, gp GenParams) {
	view := *r.view.Load()
	c, ok := r.Clip(view)
	if !ok {
		c.Length = r.Length()
	}
	if gp.Form == "grammar" {
		if err := melody.ParseForm(gp.Grammar); err != nil {
			r.gen.Store(&Candidates{ID: id, Err: err.Error()})
			return
		}
	}
	notes := r.melodies(view, c.Length, from, to, count, seed, gp)
	if notes == nil {
		return
	}
	r.gen.Store(&Candidates{ID: id, Notes: notes})
}

// GenerateSections makes a melody for the clip of every section of the song (a clip the section has is replaced), from the same seed: the sections share the way the melody is made
// and follow the chords of their own; it returns how many clips it made
func (r *Roll) GenerateSections(seed uint64, gp GenParams) int {
	l := r.link.Load()
	if l == nil || l.Sections == nil {
		return 0
	}
	n := 0
	for _, s := range l.Sections() {
		out := r.melodies(s.Name, s.Beats, 0, s.Beats, 1, seed, gp)
		if len(out) == 0 {
			continue
		}
		notes := make([]Note, 0, len(out[0]))
		for _, m := range out[0] {
			notes = append(notes, Note{Start: m[0], Length: m[1], Key: int(m[2]), Velocity: m[3]})
		}
		r.SetClip(s.Name, notes, s.Beats)
		n++
	}
	return n
}

// melodies makes count melodies for a stretch of a clip of a section (nil when it can't)
func (r *Roll) melodies(section string, length, from, to float64, count int, seed uint64, gp GenParams) [][][4]float64 {
	l := r.link.Load()
	from, to = math.Max(0, from), math.Min(to, length)
	if to-from < 0.25 {
		return nil
	}
	scale, err := r.scaleFor(gp, l)
	if err != nil {
		return nil
	}
	mp := melody.Params{Scale: scale, Beats: to - from, Low: gp.Low, High: gp.High, Density: gp.Density, Rest: gp.Rest, Steps: gp.Steps, Chord: gp.Chord,
		Rhythm: gp.Rhythm, Contour: gp.Contour, Form: gp.Form, Grammar: grammarOf(gp), Phrase: gp.Phrase, Grid: gp.Grid, Velocity: gp.Velocity}
	if l != nil && l.Chords != nil {
		for _, sp := range l.Chords(section, length) {
			at, end := math.Max(sp.At, from), math.Min(sp.At+sp.Len, to)
			if end-at > 1e-9 {
				root := 0
				if len(sp.Pcs) > 0 {
					root = sp.Pcs[0]
				}
				mp.Chords = append(mp.Chords, melody.Span{At: at - from, Len: end - at, Pcs: sp.Pcs, Root: root})
			}
		}
	}
	var out [][][4]float64
	for i := range max(1, min(count, 12)) {
		mp.Seed = seed + uint64(i)
		var notes [][4]float64
		for _, n := range melody.Generate(mp) {
			notes = append(notes, [4]float64{n.Start + from, n.Length, float64(n.Key), math.Round(n.Velocity*100) / 100})
		}
		out = append(out, notes)
	}
	return out
}

// Counter makes a second voice for the notes of a stretch (the melody: start, length, key, velocity, in beats from the start of the clip), with the chords of the harmony under
// it, and in the scale; what the editor gets next (Candidates, one melody) is the voice, under the melody. every is a note every so many beats (0: one under every note of the
// melody) and motion how much it moves against the melody (0 to 1)
func (r *Roll) Counter(id int, from, to float64, melodyNotes [][4]float64, every, motion float64, gp GenParams) {
	view := *r.view.Load()
	l := r.link.Load()
	c, ok := r.Clip(view)
	if !ok {
		c.Length = r.Length()
	}
	scale, err := r.scaleFor(gp, l)
	if err != nil || len(melodyNotes) == 0 {
		return
	}
	lo, hi := 127, 0
	var m []melody.Note
	for _, n := range melodyNotes {
		k := int(math.Round(n[2]))
		lo, hi = min(lo, k), max(hi, k)
		m = append(m, melody.Note{Start: n[0] - from, Length: n[1], Key: k, Velocity: n[3]})
	}
	slices.SortStableFunc(m, func(a, b melody.Note) int { return int(math.Copysign(1, a.Start-b.Start)) })
	cp := melody.CounterParams{Scale: scale, Low: max(24, lo-17), High: hi - 3, Every: every, Motion: motion, Velocity: gp.Velocity * 0.85}
	cp.High = max(cp.High, cp.Low+7)
	if l != nil && l.Chords != nil {
		for _, sp := range l.Chords(view, c.Length) {
			at, end := math.Max(sp.At, from), math.Min(sp.At+sp.Len, to)
			if end-at > 1e-9 {
				root := 0
				if len(sp.Pcs) > 0 {
					root = sp.Pcs[0]
				}
				cp.Chords = append(cp.Chords, melody.Span{At: at - from, Len: end - at, Pcs: sp.Pcs, Root: root})
			}
		}
	}
	var notes [][4]float64
	for _, n := range melody.Counter(m, cp) {
		notes = append(notes, [4]float64{n.Start + from, n.Length, float64(n.Key), math.Round(n.Velocity*100) / 100})
	}
	r.gen.Store(&Candidates{ID: id, Notes: [][][4]float64{notes}})
}

// grammarOf is the grammar a melody is asked for: the text when the form is a grammar
func grammarOf(gp GenParams) string {
	if gp.Form == "grammar" {
		return gp.Grammar
	}
	return ""
}

// scaleFor is the scale a melody is in: the one asked for, or the harmony's
func (r *Roll) scaleFor(gp GenParams, l *Link) (harmony.Scale, error) {
	if gp.Mode != "" {
		return harmony.NewScale(harmony.MustScale("C", "major").Name(((gp.Root%12)+12)%12), gp.Mode)
	}
	if l != nil && l.Scale != nil {
		if root, mode, ok := l.Scale(); ok {
			return harmony.NewScale(harmony.MustScale("C", "major").Name(((root%12)+12)%12), mode)
		}
	}
	return harmony.NewScale("C", "major")
}

// Preview plays notes once, as they are (a melody to hear before it is used): beats from the start, from the next control tick
func (r *Roll) Preview(notes []Note, length float64) {
	r.preview.Store(&sequence{Notes: slices.Clone(notes), Length: length})
}

// Audition plays a note at once, for the editor to let you hear what you put in or move (from any goroutine): at the next control tick, for a length in beats
func (r *Roll) Audition(key int, velocity, beats float64) {
	r.heard.Lock()
	if len(r.waiting) < 16 {
		r.waiting = append(r.waiting, Note{Key: max(0, min(key, 127)), Velocity: math.Max(0, math.Min(velocity, 1)), Length: math.Max(0.05, math.Min(beats, 8))})
		r.hearing.Store(int32(len(r.waiting)))
	}
	r.heard.Unlock()
}

// hear plays the notes the editor asked to hear
func (r *Roll) hear(tick psalm.CtrlTick) {
	r.heard.Lock()
	ns := r.waiting
	r.waiting = nil
	r.hearing.Store(0)
	r.heard.Unlock()
	second := r.Env().TickDuration() * float64(r.Config().PPQ)
	for _, n := range ns {
		msg := psalm.Message{"command": "trigger", "tick": tick, "duration": math.Max(0.05, n.Length*second), "amplitude": n.Velocity,
			"message": psalm.Message{"pitch": float64(n.Key), "velocity": n.Velocity}}
		for _, target := range r.targets {
			target.ReceiveMessage(msg)
		}
	}
}

// CtrlTick plays the notes that start on the tick: the clip of the section the song is in (from the start of the section), else the loop, which comes round every Length beats of the
// patch, wherever the tempo is
func (r *Roll) CtrlTick(tick psalm.CtrlTick) {
	if r.hearing.Load() > 0 {
		r.hear(tick)
	}
	r.playPreview(tick)
	ppq := float64(r.Config().PPQ)
	clips := *r.clips.Load()
	clip, name, base := clips[""], "", int64(0)
	if l := r.link.Load(); l != nil && l.Where != nil {
		if section, began, ok := l.Where(tick); ok {
			if c, has := clips[section]; has {
				clip, name, base = c, section, began
			}
		}
	}
	if name != *r.playing.Load() {
		r.playing.Store(&name)
	}
	loop := int64(math.Round(clip.Length * ppq))
	if loop < 1 {
		return
	}
	t := tick.TickTimestamp
	r.pos.Store(math.Float64bits(float64(mod(t-base, loop)) / ppq))
	from := r.last + 1
	if r.last < 0 || t < from || t-from >= loop || name != r.lastClip { // the first tick, a jump back, a long pause, another clip: only this one
		from = t
	}
	r.last, r.lastClip = t, name
	second := r.Env().TickDuration() * ppq // seconds a beat
	for tt := from; tt <= t; tt++ {
		at := mod(tt-base, loop)
		for _, n := range clip.Notes {
			start := int64(math.Round(n.Start * ppq))
			if start > at {
				break
			}
			if start != at {
				continue
			}
			msg := psalm.Message{"command": "trigger", "tick": tick, "duration": math.Max(0.005, n.Length*second), "amplitude": n.Velocity,
				"message": psalm.Message{"pitch": float64(n.Key), "velocity": n.Velocity}}
			for _, target := range r.targets {
				target.ReceiveMessage(msg)
			}
		}
	}
}

// playPreview plays the notes the editor asked to hear, from the tick it saw them
func (r *Roll) playPreview(tick psalm.CtrlTick) {
	if p := r.preview.Swap(nil); p != nil {
		r.pvCur, r.pvStart, r.pvLast = p, tick.TickTimestamp, -1
	}
	if r.pvCur == nil {
		return
	}
	ppq := float64(r.Config().PPQ)
	rel := tick.TickTimestamp - r.pvStart
	second := r.Env().TickDuration() * ppq
	for _, n := range r.pvCur.Notes {
		t := int64(math.Round(n.Start * ppq))
		if t <= r.pvLast || t > rel {
			continue
		}
		msg := psalm.Message{"command": "trigger", "tick": tick, "duration": math.Max(0.005, n.Length*second), "amplitude": n.Velocity,
			"message": psalm.Message{"pitch": float64(n.Key), "velocity": n.Velocity}}
		for _, target := range r.targets {
			target.ReceiveMessage(msg)
		}
	}
	r.pvLast = rel
	if float64(rel) > r.pvCur.Length*ppq+ppq {
		r.pvCur = nil
	}
}

func mod(a, b int64) int64 { return ((a % b) + b) % b }

// state is what the page gets as the roll plays: the notes of the clip it shows as [start, length, key, velocity], and where it plays
type state struct {
	Notes    [][4]float64    `json:"notes"`
	Length   float64         `json:"length"`
	Pos      float64         `json:"pos"` // where the clip shown plays, -1 when another plays
	BPS      float64         `json:"bps"` // beats a second
	View     string          `json:"view"`
	Playing  string          `json:"playing"`
	Clips    []string        `json:"clips"`
	Sections []Section       `json:"sections"`
	Chords   []ChordSpan     `json:"chords"`
	Scale    *ScaleInfo      `json:"scale"`
	Gen      *Candidates     `json:"gen"`
	Grooves  []rhythm.Groove `json:"grooves"`
}

// ScaleInfo is the key and scale of the harmony, for the editor to light the keys of the scale
type ScaleInfo struct {
	Root int    `json:"root"`
	Mode string `json:"mode"`
	Pcs  []int  `json:"pcs"`
}

func (r *Roll) state() state {
	view := *r.view.Load()
	l := r.link.Load()
	st := state{Notes: [][4]float64{}, BPS: r.Env().BPM() / 60, View: view, Playing: r.Playing(), Clips: r.Clips(), Sections: []Section{}, Chords: []ChordSpan{}, Grooves: rhythm.Grooves}
	if l != nil && l.Sections != nil {
		st.Sections = l.Sections()
	}
	c, ok := r.Clip(view)
	if !ok { // no clip yet: as long as the section, else as the loop
		c.Length = r.Length()
		for _, s := range st.Sections {
			if s.Name == view && s.Beats > 0 {
				c.Length = math.Min(s.Beats, MaxBeats)
			}
		}
	}
	st.Length = c.Length
	for _, n := range c.Notes {
		st.Notes = append(st.Notes, [4]float64{n.Start, n.Length, float64(n.Key), math.Round(n.Velocity*100) / 100})
	}
	if st.Playing == view {
		st.Pos = math.Round(r.Position()*1000) / 1000
	} else {
		st.Pos = -1
	}
	if l != nil && l.Chords != nil {
		st.Chords = l.Chords(view, c.Length)
	}
	if l != nil && l.Scale != nil {
		if root, mode, ok := l.Scale(); ok {
			if sc, err := harmony.NewScale(harmony.MustScale("C", "major").Name(((root%12)+12)%12), mode); err == nil {
				info := &ScaleInfo{Root: sc.Root, Mode: mode}
				for _, step := range sc.Steps {
					info.Pcs = append(info.Pcs, (sc.Root+step)%12)
				}
				st.Scale = info
			}
		}
	}
	st.Gen = r.gen.Load()
	return st
}

// StateText is the clips as a text the patch keeps (the clips of the sections as well as the loop)
func (r *Roll) StateText() string {
	type clip struct {
		Length float64      `json:"length"`
		Notes  [][4]float64 `json:"notes"`
	}
	out := map[string]clip{}
	for name, c := range *r.clips.Load() {
		cl := clip{Length: c.Length, Notes: [][4]float64{}}
		for _, n := range c.Notes {
			cl.Notes = append(cl.Notes, [4]float64{n.Start, n.Length, float64(n.Key), n.Velocity})
		}
		out[name] = cl
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// SetStateText gives the clips back from StateText
func (r *Roll) SetStateText(text string) error {
	var in map[string]struct {
		Length float64      `json:"length"`
		Notes  [][4]float64 `json:"notes"`
	}
	if err := json.Unmarshal([]byte(text), &in); err != nil {
		return err
	}
	m := map[string]*Clip{}
	for name, c := range in {
		notes := make([]Note, 0, len(c.Notes))
		for _, n := range c.Notes {
			notes = append(notes, Note{Start: n[0], Length: n[1], Key: int(math.Round(n[2])), Velocity: n[3]})
		}
		m[name] = makeClip(notes, c.Length)
	}
	if _, ok := m[""]; !ok {
		m[""] = &Clip{Length: 8}
	}
	r.clips.Store(&m)
	return nil
}

// Add adds the editor of a roll to a panel (and the plugin to its page)
func Add(ctl *panel.Panel, name string, roll *Roll) *panel.Param {
	ctl.Use(Plugin)
	return ctl.Custom(Kind, name, nil, panel.NoDefault(),
		panel.Live(func() any { return roll.state() }),
		panel.OnMessage(func(raw json.RawMessage) {
			var m struct {
				Set *struct {
					Clip   *string      `json:"clip"`
					Notes  [][4]float64 `json:"notes"`
					Length float64      `json:"length"`
				} `json:"set"`
				View *string `json:"view"`
				Copy *struct {
					From string   `json:"from"`
					To   []string `json:"to"`
				} `json:"copy"`
				Drop *string `json:"drop"`
				Play *struct {
					Key int     `json:"key"`
					Vel float64 `json:"vel"`
				} `json:"play"`
				Gen *struct {
					ID     int       `json:"id"`
					From   float64   `json:"from"`
					To     float64   `json:"to"`
					Count  int       `json:"count"`
					Seed   uint64    `json:"seed"`
					Params GenParams `json:"params"`
				} `json:"gen"`
				Counter *struct {
					ID     int          `json:"id"`
					From   float64      `json:"from"`
					To     float64      `json:"to"`
					Notes  [][4]float64 `json:"notes"`
					Every  float64      `json:"every"`
					Motion float64      `json:"motion"`
					Params GenParams    `json:"params"`
				} `json:"counter"`
				GenAll *struct {
					Seed   uint64    `json:"seed"`
					Params GenParams `json:"params"`
				} `json:"genall"`
				Preview *struct {
					Notes  [][4]float64 `json:"notes"`
					Length float64      `json:"length"`
				} `json:"preview"`
			}
			if json.Unmarshal(raw, &m) != nil {
				return
			}
			switch {
			case m.View != nil:
				roll.view.Store(m.View)
			case m.Copy != nil:
				roll.CopyClip(m.Copy.From, m.Copy.To...)
			case m.Drop != nil:
				roll.DropClip(*m.Drop)
			case m.Play != nil:
				roll.Audition(m.Play.Key, m.Play.Vel, 0.5)
			case m.Gen != nil:
				roll.Generate(m.Gen.ID, m.Gen.From, m.Gen.To, m.Gen.Count, m.Gen.Seed, m.Gen.Params)
			case m.Counter != nil:
				roll.Counter(m.Counter.ID, m.Counter.From, m.Counter.To, m.Counter.Notes, m.Counter.Every, m.Counter.Motion, m.Counter.Params)
			case m.GenAll != nil:
				roll.GenerateSections(m.GenAll.Seed, m.GenAll.Params)
			case m.Preview != nil:
				notes := make([]Note, 0, len(m.Preview.Notes))
				for _, n := range m.Preview.Notes {
					notes = append(notes, Note{Start: n[0], Length: n[1], Key: int(math.Round(n[2])), Velocity: n[3]})
				}
				roll.Preview(notes, m.Preview.Length)
			case m.Set != nil:
				notes := make([]Note, 0, len(m.Set.Notes))
				for _, n := range m.Set.Notes {
					notes = append(notes, Note{Start: n[0], Length: n[1], Key: int(math.Round(n[2])), Velocity: n[3]})
				}
				name := *roll.view.Load()
				if m.Set.Clip != nil {
					name = *m.Set.Clip
				}
				roll.SetClip(name, notes, m.Set.Length)
			}
		}))
}

// Values returns the notes and length as numbers to keep in a patch ("length", and "n/<i>/start",
// "n/<i>/length", "n/<i>/key", "n/<i>/velocity" for every note): Apply reads them back
func (r *Roll) Values() map[string]float64 {
	v := map[string]float64{"length": r.Length()}
	for i, n := range r.Notes() {
		p := "n/" + strconv.Itoa(i) + "/"
		v[p+"start"], v[p+"length"], v[p+"key"], v[p+"velocity"] = n.Start, n.Length, float64(n.Key), n.Velocity
	}
	return v
}

// Apply takes the notes and length back from Values; a roll without "length" in them is left as it is
func (r *Roll) Apply(v map[string]float64) {
	length, ok := v["length"]
	if !ok {
		return
	}
	var notes []Note
	for i := 0; ; i++ {
		p := "n/" + strconv.Itoa(i) + "/"
		start, ok := v[p+"start"]
		if !ok {
			break
		}
		notes = append(notes, Note{Start: start, Length: v[p+"length"], Key: int(v[p+"key"]), Velocity: v[p+"velocity"]})
	}
	r.Set(notes, length)
}
