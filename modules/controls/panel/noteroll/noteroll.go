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
	"math"
	"slices"
	"strconv"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
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

// Roll is a loop of notes, playing with the tempo of the patch. Its notes and length are safe to
// change from any goroutine (the page, your own code); it plays on the audio thread
type Roll struct {
	*psalm.Module
	notes   atomic.Pointer[[]Note] // sorted by start
	length  atomic.Uint64          // beats, float64 bits
	pos     atomic.Uint64          // where it plays, in beats, float64 bits
	targets []psalm.Receiver       // audio thread
	last    int64                  // the last tick played
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
	r.Set(nil, 8)
	return r
}

// NewRoll creates a roll that isn't in a patch yet (the patcher adds it)
func NewRoll() *Roll { return newRoll() }

// SetTargets sets the synths it plays; call it on the audio thread (psalm.Psalm.Do)
func (r *Roll) SetTargets(targets ...psalm.Receiver) { r.targets = targets }

// Set sets the notes and the length of the loop in beats (1 to MaxBeats). Notes are put in order,
// within the loop, and made to fit its limits
func (r *Roll) Set(notes []Note, length float64) {
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
	r.notes.Store(&ns)
	r.length.Store(math.Float64bits(length))
}

// Notes returns the notes, in order of start
func (r *Roll) Notes() []Note { return slices.Clone(*r.notes.Load()) }

// Length returns the length of the loop in beats
func (r *Roll) Length() float64 { return math.Float64frombits(r.length.Load()) }

// Position returns where the loop plays, in beats
func (r *Roll) Position() float64 { return math.Float64frombits(r.pos.Load()) }

// CtrlTick plays the notes that start on the tick: the loop comes round every Length beats of the
// patch, wherever the tempo is
func (r *Roll) CtrlTick(tick psalm.CtrlTick) {
	ppq := float64(r.Config().PPQ)
	loop := int64(math.Round(r.Length() * ppq))
	if loop < 1 {
		return
	}
	t := tick.TickTimestamp
	r.pos.Store(math.Float64bits(float64(t%loop) / ppq))
	from := r.last + 1
	if r.last < 0 || t < from || t-from >= loop { // the first tick, a jump back, a long pause: only this one
		from = t
	}
	r.last = t
	notes := *r.notes.Load()
	beat := r.Env().TickDuration() * ppq // seconds
	for tt := from; tt <= t; tt++ {
		at := tt % loop
		for _, n := range notes {
			start := int64(math.Round(n.Start * ppq))
			if start > at {
				break
			}
			if start != at {
				continue
			}
			msg := psalm.Message{"command": "trigger", "tick": tick, "duration": math.Max(0.005, n.Length*beat), "amplitude": n.Velocity,
				"message": psalm.Message{"pitch": float64(n.Key), "velocity": n.Velocity}}
			for _, target := range r.targets {
				target.ReceiveMessage(msg)
			}
		}
	}
}

// state is what the page gets as the roll plays: the notes as [start, length, key, velocity]
type state struct {
	Notes  [][4]float64 `json:"notes"`
	Length float64      `json:"length"`
	Pos    float64      `json:"pos"`
	BPS    float64      `json:"bps"` // beats a second
}

func (r *Roll) state() state {
	st := state{Notes: [][4]float64{}, Length: r.Length(), Pos: math.Round(r.Position()*1000) / 1000, BPS: r.Env().BPM() / 60}
	for _, n := range *r.notes.Load() {
		st.Notes = append(st.Notes, [4]float64{n.Start, n.Length, float64(n.Key), math.Round(n.Velocity*100) / 100})
	}
	return st
}

// Add adds the editor of a roll to a panel (and the plugin to its page)
func Add(ctl *panel.Panel, name string, roll *Roll) *panel.Param {
	ctl.Use(Plugin)
	return ctl.Custom(Kind, name, nil, panel.NoDefault(),
		panel.Live(func() any { return roll.state() }),
		panel.OnMessage(func(raw json.RawMessage) {
			var m struct {
				Set *struct {
					Notes  [][4]float64 `json:"notes"`
					Length float64      `json:"length"`
				} `json:"set"`
			}
			if json.Unmarshal(raw, &m) != nil || m.Set == nil {
				return
			}
			notes := make([]Note, 0, len(m.Set.Notes))
			for _, n := range m.Set.Notes {
				notes = append(notes, Note{Start: n[0], Length: n[1], Key: int(math.Round(n[2])), Velocity: n[3]})
			}
			roll.Set(notes, m.Set.Length)
		}))
}

// Values returns the notes and length as numbers to keep in a patch ("length", and "n/<i>/start",
// "n/<i>/length", "n/<i>/key", "n/<i>/velocity" for every note): Apply reads them back
func (r *Roll) Values() map[string]float64 {
	v := map[string]float64{"length": r.Length()}
	for i, n := range *r.notes.Load() {
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
