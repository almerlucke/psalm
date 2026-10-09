package panel

import (
	"encoding/json"
	"github.com/almerlucke/psalm"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

// cueList is a cue list for the tests
type cueList struct {
	mu    sync.Mutex
	state CueState
}

func newCueList() *cueList {
	return &cueList{state: CueState{Playing: 0, Queued: -1, Enabled: []bool{true, true, true}}}
}

func (c *cueList) CueNames() []string { return []string{"Intro", "Verse", "Chorus"} }
func (c *cueList) CueState() CueState {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := c.state
	st.Enabled = slices.Clone(st.Enabled)
	return st
}
func (c *cueList) QueueCue(i int) { c.mu.Lock(); c.state.Queued = i; c.mu.Unlock() }
func (c *cueList) EnableCue(i int, on bool) {
	c.mu.Lock()
	c.state.Enabled[i] = on
	c.mu.Unlock()
}

func TestCues(t *testing.T) {
	pn, _ := newPanel()
	c := newCueList()
	pn.Section("Song")
	cs := pn.Cues("Song", c)
	srv := httptest.NewServer(pn.Handler())
	defer srv.Close()

	res, _ := http.Get(srv.URL + "/api/panel")
	var d description
	_ = json.NewDecoder(res.Body).Decode(&d)
	res.Body.Close()
	ctl := d.Sections[0].Controls[0]
	if ctl.Kind != CuesKind || ctl.Cues == nil || len(ctl.Cues.Names) != 3 || ctl.Cues.State.Queued != -1 {
		t.Fatalf("the cue list should be described: %+v", ctl)
	}

	for _, body := range []string{`{"name": "Song", "queue": 2}`, `{"name": "Song", "cue": 1, "on": false}`} {
		if res := postJSON(t, srv.URL+"/api/cues", body); res.StatusCode != 204 {
			t.Fatalf("%s: %d", body, res.StatusCode)
		}
	}
	if st := cs.State(); st.Queued != 2 || st.Enabled[1] {
		t.Fatalf("the page should queue and switch cues: %+v", st)
	}
	if res := postJSON(t, srv.URL+"/api/cues", `{"name": "Nope", "queue": 1}`); res.StatusCode != 404 {
		t.Fatal("an unknown cue list should be 404")
	}
	cs.Queue(7) // out of the list: nothing
	if cs.State().Queued != 2 {
		t.Fatal("a cue out of the list should be left out")
	}

	snap := pn.Snapshot()
	if snap["Song/Verse"] != 0 || snap["Song/Intro"] != 1 {
		t.Fatalf("the snapshot should hold which cues are on: %v", snap)
	}
	cs.Enable(1, true)
	cs.Enable(0, false)
	pn.Restore(snap)
	if st := cs.State(); !st.Enabled[0] || st.Enabled[1] {
		t.Fatalf("restore should switch the cues back: %+v", st)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("a cue list named like a control should panic")
		}
	}()
	pn.Toggle("Song", nil)
}

func TestCueTilesMIDI(t *testing.T) {
	pn, _ := newPanel()
	c := newCueList()
	cs := pn.Cues("Song", c)
	chorus := pn.Param("Song/Chorus")
	if chorus == nil || chorus != cs.Tile(2) || chorus.Kind != ButtonKind {
		t.Fatal("every tile should be a button named list/cue")
	}
	if len(pn.describe().Sections[0].Controls) != 1 {
		t.Fatal("the tiles should not show as controls of their own")
	}
	press := func() {
		pn.CtrlTick(psalm.CtrlTick{TickTimestamp: 1})
	}

	chorus.BindNote(0, 36)
	pn.NoteOn(10, 36, 100)
	press()
	if c.CueState().Queued != 2 {
		t.Fatal("a note bound to a tile should ask for its cue")
	}
	pn.NoteOn(10, 36, 0) // a note off: nothing
	pn.NoteOn(10, 36, 90)
	press()
	if c.CueState().Queued != -1 {
		t.Fatal("pressing the tile of the cue asked for should take the request back")
	}

	// learning: a note goes to a tile, a fader waits for a controller
	verse := pn.Param("Song/Verse")
	cutoff := pn.Fader("Cutoff", 0, 1, Linear, nil)
	cutoff.Learn()
	pn.NoteOn(1, 40, 100)
	if _, ok := cutoff.Binding(); ok || pn.Learning() != cutoff {
		t.Fatal("a fader should not learn a note")
	}
	verse.Learn()
	pn.NoteOn(1, 36, 100) // the note of the chorus: taken from it
	if b, ok := verse.Binding(); !ok || b != (Binding{Channel: 1, Controller: 36, Note: true}) {
		t.Fatalf("the tile should learn the note: %+v", b)
	}
	if _, ok := chorus.Binding(); ok {
		t.Fatal("a note should move one control: the chorus loses it")
	}
	if _, _, ok := verse.CC(); ok {
		t.Fatal("CC should not report a note")
	}

	// a controller works too, and the bindings keep the notes
	chorus.BindCC(0, 20)
	pn.ControlChange(1, 20, 127)
	press()
	if c.CueState().Queued != 2 {
		t.Fatal("a controller bound to a tile should ask for its cue")
	}
	bs := pn.Bindings()
	pn.SetBindings(nil)
	pn.SetBindings(bs)
	if b, _ := verse.Binding(); !b.Note {
		t.Fatal("the bindings should keep the notes")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("binding a note to a fader should panic")
		}
	}()
	cutoff.BindNote(0, 1)
}

func TestToggleNote(t *testing.T) {
	pn, tk := newPanel()
	mute := pn.Toggle("Mute", nil)
	mute.BindNote(0, 50)
	pn.NoteOn(1, 50, 64)
	tk.tick(1)
	if mute.Value() != 1 {
		t.Fatal("a note should switch a toggle on")
	}
	pn.NoteOn(1, 50, 64)
	tk.tick(1)
	if mute.Value() != 0 {
		t.Fatal("and off again")
	}
}

// changing is a cue list whose cues change
type changing struct {
	cueList
	names []string
}

func (c *changing) CueNames() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.names)
}

func TestCueSlots(t *testing.T) {
	pn, _ := newPanel()
	c := &changing{cueList: *newCueList(), names: []string{"Intro", "Verse", "Chorus"}}
	cs := pn.Cues("Sections", c, CueSlots(4))
	if pn.Param("Sections/1") == nil || pn.Param("Sections/4") == nil || pn.Param("Sections/Intro") != nil {
		t.Fatal("the tiles of a list with slots should be named by position")
	}
	if st := cs.State(); !slices.Equal(st.Names, c.names) {
		t.Fatalf("the state should hold the cues there are now: %v", st.Names)
	}
	if d := pn.describe(); d.Sections[0].Controls[0].Cues.Slots != 4 {
		t.Fatal("the page should know the cues change")
	}
	c.mu.Lock()
	c.names = []string{"A", "B"}
	c.mu.Unlock()
	cs.Queue(2) // there are two now
	if c.CueState().Queued != -1 {
		t.Fatal("a cue past the cues there are now should be left out")
	}
	pn.Param("Sections/2").Press()
	pn.CtrlTick(psalm.CtrlTick{TickTimestamp: 1})
	if c.CueState().Queued != 1 {
		t.Fatal("the second tile's button should ask for the second cue, whatever it is")
	}
	pn.Param("Sections/4").Press()
	pn.CtrlTick(psalm.CtrlTick{TickTimestamp: 2})
	if c.CueState().Queued != 1 {
		t.Fatal("a slot without a cue does nothing")
	}
	if snap := pn.Snapshot(); len(snap) != 0 {
		t.Fatalf("cues that change are not in snapshots: %v", snap)
	}
}
