package panel

import (
	"fmt"
)

// CuesKind is the kind of a cue list on the page
const CuesKind Kind = "cues"

// Cuer is what a cue list shows and plays: named cues (the sections of a song, scenes, patterns),
// one of them playing, one asked for next, each one on or off. A song transport (arrange) is one.
// The methods are called from the page's goroutines. The cues can change while it plays (the
// sections of the song a playlist plays) in a cue list with slots (CueSlots)
type Cuer interface {
	CueNames() []string
	CueState() CueState
	QueueCue(i int)           // asks for a cue to play next, -1 takes the request back
	EnableCue(i int, on bool) // switches a cue on or off
}

// CueState is the state of a cue list
type CueState struct {
	Playing  int      `json:"playing"`          // the cue playing, -1 none
	Progress float64  `json:"progress"`         // through it, 0-1
	Queued   int      `json:"queued"`           // the cue asked for next, -1 none
	Enabled  []bool   `json:"enabled"`          // per cue whether it is on
	Names    []string `json:"names,omitempty"`  // the cues now (a cue list with slots, whose cues change)
	Active   []bool   `json:"active,omitempty"` // per cue whether it sounds now (the parts playing in a section)
}

// CueOption is an option of a cue list
type CueOption func(*Cues)

// CueSlots makes a cue list whose cues can change while it plays (the sections of the song a
// playlist is playing): the page shows the cues there are now, and the buttons of the tiles are
// named by position, "<list>/1" to "<list>/<n>", so a MIDI pad bound to the second tile plays the
// second cue whatever it is. Which cues are on is not in snapshots (they are other cues every time)
func CueSlots(n int) CueOption { return func(cs *Cues) { cs.Slots = n } }

// CueToggles makes a cue list of switches (the parts of a song): a tap on a tile switches its cue on
// or off (no cue is asked for next), the cues that sound now (CueState.Active) light up
func CueToggles() CueOption { return func(cs *Cues) { cs.Toggles = true } }

// Cues is a cue list on the panel
type Cues struct {
	Name    string
	Section string
	Names   []string // the cues when it was made
	Slots   int      // tiles named by position, for cues that change (CueSlots); 0: named by cue
	Toggles bool     // a tap switches a cue on or off (CueToggles)
	tiles   []*Param
	cuer    Cuer
}

// Cues adds a cue list: a tile per cue showing which one plays (and how far), which one is asked
// for next and which ones are off. Tap a tile to ask for it next (tap it again to take that back),
// tap its switch to turn it on or off. Which cues are on is kept in snapshots and presets.
//
// Every tile is a button too, named "<list>/<cue>" ("Song/Chorus"): pressing it does what a tap
// does. Bind it to a pad or a key of a MIDI controller with MIDI learn on the page (tap the tile),
// or in code: ctl.Param("Song/Chorus").BindNote(0, 36), or Tile(i). With CueSlots the cues can
// change while it plays, and the buttons are named by position
func (pn *Panel) Cues(name string, c Cuer, opts ...CueOption) *Cues {
	cs := &Cues{Name: name, Names: c.CueNames(), cuer: c}
	for _, o := range opts {
		o(cs)
	}
	pn.mu.Lock()
	defer pn.mu.Unlock()
	if pn.taken(name) {
		panic(fmt.Sprintf("panel: a control named %q twice", name))
	}
	tiles := cs.Names
	if cs.Slots > 0 {
		tiles = make([]string, cs.Slots)
		for i := range tiles {
			tiles[i] = fmt.Sprint(i + 1)
		}
	}
	for i, tile := range tiles {
		par := newParam(ButtonKind, name+"/"+tile, 0, 1, Linear, nil, nil)
		par.press = func() { cs.Toggle(i) }
		cs.tiles = append(cs.tiles, pn.addLocked(par, false))
	}
	if pn.section == "" {
		pn.section = pn.title
		pn.sections = append(pn.sections, pn.section)
	}
	cs.Section = pn.section
	if pn.cues == nil {
		pn.cues = map[string]*Cues{}
	}
	pn.cues[name] = cs
	pn.order = append(pn.order, item{cues: cs})
	return cs
}

// CuesByName returns a cue list by name, nil when the panel has none of that name
func (pn *Panel) CuesByName(name string) *Cues {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return pn.cues[name]
}

// State returns the state of the cue list (with the cues there are now, for a cue list with slots)
func (cs *Cues) State() CueState {
	st := cs.cuer.CueState()
	if cs.Slots > 0 {
		st.Names = cs.cuer.CueNames()
	}
	return st
}

// count returns how many cues there are now
func (cs *Cues) count() int {
	if cs.Slots > 0 {
		return len(cs.cuer.CueNames())
	}
	return len(cs.Names)
}

// Queue asks for a cue to play next, -1 takes the request back
func (cs *Cues) Queue(i int) {
	if i >= -1 && i < cs.count() {
		cs.cuer.QueueCue(i)
	}
}

// Toggle asks for a cue to play next, or takes the request back when it was asked for already
func (cs *Cues) Toggle(i int) {
	if i >= cs.count() {
		return
	}
	if cs.Toggles { // switches: on or off
		st := cs.State()
		cs.Enable(i, !(i < len(st.Enabled) && st.Enabled[i]))
		return
	}
	if cs.State().Queued == i {
		cs.Queue(-1)
	} else {
		cs.Queue(i)
	}
}

// Tile returns the button of a cue's tile, to bind it to MIDI (BindNote, BindCC)
func (cs *Cues) Tile(i int) *Param { return cs.tiles[i] }

// Enable switches a cue on or off
func (cs *Cues) Enable(i int, on bool) {
	if i >= 0 && i < cs.count() {
		cs.cuer.EnableCue(i, on)
	}
}

// snapshot writes per cue whether it is on, as "list/cue" (not for cues that change)
func (cs *Cues) snapshot(values map[string]float64) {
	if cs.Slots > 0 {
		return
	}
	for i, on := range cs.State().Enabled {
		v := 0.0
		if on {
			v = 1
		}
		values[cs.Name+"/"+cs.Names[i]] = v
	}
}

// restore switches the cues the snapshot has on or off
func (cs *Cues) restore(values map[string]float64) {
	if cs.Slots > 0 {
		return
	}
	for i, name := range cs.Names {
		if v, ok := values[cs.Name+"/"+name]; ok {
			cs.Enable(i, v >= 0.5)
		}
	}
}
