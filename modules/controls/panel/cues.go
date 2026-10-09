package panel

import (
	"fmt"
)

// CuesKind is the kind of a cue list on the page
const CuesKind Kind = "cues"

// Cuer is what a cue list shows and plays: named cues (the sections of a song, scenes, patterns),
// one of them playing, one asked for next, each one on or off. A song transport (arrange) is one.
// The methods are called from the page's goroutines
type Cuer interface {
	CueNames() []string
	CueState() CueState
	QueueCue(i int)           // asks for a cue to play next, -1 takes the request back
	EnableCue(i int, on bool) // switches a cue on or off
}

// CueState is the state of a cue list
type CueState struct {
	Playing  int     `json:"playing"`  // the cue playing, -1 none
	Progress float64 `json:"progress"` // through it, 0-1
	Queued   int     `json:"queued"`   // the cue asked for next, -1 none
	Enabled  []bool  `json:"enabled"`  // per cue whether it is on
}

// Cues is a cue list on the panel
type Cues struct {
	Name    string
	Section string
	Names   []string
	tiles   []*Param
	cuer    Cuer
}

// Cues adds a cue list: a tile per cue showing which one plays (and how far), which one is asked
// for next and which ones are off. Tap a tile to ask for it next (tap it again to take that back),
// tap its switch to turn it on or off. Which cues are on is kept in snapshots and presets.
//
// Every tile is a button too, named "<list>/<cue>" ("Song/Chorus"): pressing it does what a tap
// does. Bind it to a pad or a key of a MIDI controller with MIDI learn on the page (tap the tile),
// or in code: ctl.Param("Song/Chorus").BindNote(0, 36), or Tile(i)
func (pn *Panel) Cues(name string, c Cuer) *Cues {
	cs := &Cues{Name: name, Names: c.CueNames(), cuer: c}
	pn.mu.Lock()
	defer pn.mu.Unlock()
	if pn.taken(name) {
		panic(fmt.Sprintf("panel: a control named %q twice", name))
	}
	for i, cue := range cs.Names {
		tile := newParam(ButtonKind, name+"/"+cue, 0, 1, Linear, nil, nil)
		tile.press = func() { cs.Toggle(i) }
		cs.tiles = append(cs.tiles, pn.addLocked(tile, false))
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

// State returns the state of the cue list
func (cs *Cues) State() CueState { return cs.cuer.CueState() }

// Queue asks for a cue to play next, -1 takes the request back
func (cs *Cues) Queue(i int) {
	if i >= -1 && i < len(cs.Names) {
		cs.cuer.QueueCue(i)
	}
}

// Toggle asks for a cue to play next, or takes the request back when it was asked for already
func (cs *Cues) Toggle(i int) {
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
	if i >= 0 && i < len(cs.Names) {
		cs.cuer.EnableCue(i, on)
	}
}

// snapshot writes per cue whether it is on, as "list/cue"
func (cs *Cues) snapshot(values map[string]float64) {
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
	for i, name := range cs.Names {
		if v, ok := values[cs.Name+"/"+name]; ok {
			cs.Enable(i, v >= 0.5)
		}
	}
}
