package arrange

import (
	"fmt"

	"github.com/almerlucke/psalm/modules/controls/panel"
)

// part is a part of the form, written (Part) or a stream (Stream), that can be switched on and off
// while the song plays
type part struct {
	name    string
	enable  func(on bool)
	enabled func() bool
	active  func() bool // it plays in the section playing
}

// addPart keeps a part of the form
func (f *Form) addPart(p *part) {
	f.partsMu.Lock()
	defer f.partsMu.Unlock()
	f.parts = append(f.parts, p)
}

// partList returns the parts
func (f *Form) partList() []*part {
	f.partsMu.Lock()
	defer f.partsMu.Unlock()
	return append([]*part(nil), f.parts...)
}

// part returns a part by name, panicking on a name the form doesn't have
func (f *Form) part(name string) *part {
	for _, p := range f.partList() {
		if p.name == name {
			return p
		}
	}
	panic(fmt.Sprintf("arrange: the form has no part %q", name))
}

// Parts returns the names of the parts of the form (Part, Stream), in the order they were added
func (f *Form) Parts() []string {
	var names []string
	for _, p := range f.partList() {
		names = append(names, p.name)
	}
	return names
}

// SetPartEnabled switches a part on or off while the song plays (from any goroutine): a part that is
// off starts no new notes (what sounds rings out) and keeps its place, so it comes back in time. It
// stays off, through the sections, until it is switched on. It panics on a name the form doesn't have
func (f *Form) SetPartEnabled(name string, on bool) *Form {
	f.part(name).enable(on)
	return f
}

// PartEnabled tells whether a part is on
func (f *Form) PartEnabled(name string) bool { return f.part(name).enabled() }

// PartPlaying tells whether a part plays in the section playing (it has notes there, or a stream)
func (f *Form) PartPlaying(name string) bool { return f.part(name).active() }

// PartControls puts the parts on a panel: a tile per part, tap it to switch the part off or on; the
// parts that play in the section playing light up. Which parts are on is kept in presets. Every tile
// is a button too, "<name>/<part>", for a pad of a MIDI controller
func (f *Form) PartControls(ctl *panel.Panel, name string) *Form {
	ctl.Cues(name, PartCues(func() *Form { return f }), panel.CueToggles())
	return f
}

// PartCues shows the parts of a form as a cue list of toggles (panel.Cuer); form gives the form, nil
// for none (a playlist gives the form of the song playing)
func PartCues(form func() *Form) panel.Cuer { return partCuer{form} }

// partCuer is the parts of a form as a cue list: every cue on or off, the ones playing active
type partCuer struct{ form func() *Form }

func (c partCuer) parts() []*part {
	if f := c.form(); f != nil {
		return f.partList()
	}
	return nil
}

func (c partCuer) CueNames() []string {
	var names []string
	for _, p := range c.parts() {
		names = append(names, p.name)
	}
	return names
}

func (c partCuer) CueState() panel.CueState {
	st := panel.CueState{Playing: -1, Queued: -1}
	for _, p := range c.parts() {
		st.Enabled = append(st.Enabled, p.enabled())
		st.Active = append(st.Active, p.active())
	}
	return st
}

func (c partCuer) QueueCue(int) {} // parts are switched, not asked for

func (c partCuer) EnableCue(i int, on bool) {
	if ps := c.parts(); i >= 0 && i < len(ps) {
		ps[i].enable(on)
	}
}
