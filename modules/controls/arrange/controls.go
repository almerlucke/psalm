package arrange

import (
	"github.com/almerlucke/psalm/modules/controls/panel"
)

// Controls puts the transport on a panel: a tile per section (tap one to play it next, tap its
// switch to turn it on or off; the section playing shows how far it is), a switch to repeat the
// section playing and a choice of when a section asked for starts (at the end of the section
// playing, or at the next bar). The sections that are on, the repeat and the jump are kept in
// presets. Name is the name of the tiles; the switch and the choice are "<name> repeat" and
// "<name> jump" (shown as Repeat and Jump at). Every tile is a button too, "<name>/<section>": bind
// it to a pad of a MIDI controller (MIDI learn on the page, or ctl.Param("Song/Chorus").BindNote)
func (t *Transport) Controls(ctl *panel.Panel, name string) *Transport {
	ctl.Cues(name, cuer{t})
	ctl.Toggle(name+" repeat", panel.Func(func(v float64) { t.SetRepeat(v >= 0.5) }),
		panel.Label("Repeat"), panel.Default(boolValue(t.Repeat())))
	ctl.Select(name+" jump", []string{"section end", "next bar"}, func(i int) { t.SetJump(JumpAt(i)) },
		panel.Label("Jump at"), panel.Default(float64(t.Jump())))
	return t
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// cuer shows a transport as a cue list on a panel
type cuer struct{ t *Transport }

func (c cuer) CueNames() []string { return c.t.Sections() }

func (c cuer) CueState() panel.CueState {
	st := panel.CueState{Playing: int(c.t.playing.Load()), Queued: int(c.t.queued.Load())}
	if st.Playing >= 0 {
		_, st.Progress = c.t.Playing()
	}
	st.Enabled = make([]bool, len(c.t.off))
	for i := range c.t.off {
		st.Enabled[i] = !c.t.off[i].Load()
	}
	return st
}

func (c cuer) QueueCue(i int) { c.t.queued.Store(int32(i)) }

func (c cuer) EnableCue(i int, on bool) { c.t.off[i].Store(!on) }
