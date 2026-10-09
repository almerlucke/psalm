package panel

import (
	"bufio"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
)

func TestBindCC(t *testing.T) {
	pn, _ := newPanel()
	cutoff := pn.Fader("Cutoff", 100, 6400, Exp, nil).BindCC(0, 74)
	pn.ControlChange(1, 74, 127)
	if cutoff.Value() != 6400 {
		t.Fatalf("CC 127 should be the top of the range, got %v", cutoff.Value())
	}
	pn.ControlChange(3, 74, 0)
	if cutoff.Value() != 100 {
		t.Fatalf("any channel should move a control bound to channel 0, got %v", cutoff.Value())
	}
	pn.ControlChange(1, 75, 127)
	if cutoff.Value() != 100 {
		t.Fatal("another controller should not move it")
	}
	pn.ControlChange(1, 74, 64)
	if v := cutoff.Value(); math.Abs(v-100*math.Pow(64, 64.0/127)) > 1e-9 {
		t.Fatalf("an exponential control should follow the controller by its curve, got %v", v)
	}

	level := pn.Fader("Level", 0, 1, Linear, nil).BindCC(2, 7)
	pn.ControlChange(1, 7, 127)
	if level.Value() != 0 {
		t.Fatal("a control bound to channel 2 should not follow channel 1")
	}
	pn.ControlChange(2, 7, 127)
	if level.Value() != 1 {
		t.Fatal("it should follow channel 2")
	}
	if ch, ctl, ok := level.CC(); !ok || ch != 2 || ctl != 7 {
		t.Fatalf("unexpected binding %d %d %v", ch, ctl, ok)
	}
}

func TestLearn(t *testing.T) {
	pn, _ := newPanel()
	a := pn.Knob("A", 0, 1, Linear, nil)
	b := pn.Knob("B", 0, 1, Linear, nil)
	a.Learn()
	if pn.Learning() != a {
		t.Fatal("A should be learning")
	}
	pn.ControlChange(1, 20, 127)
	if _, ctl, ok := a.CC(); !ok || ctl != 20 || pn.Learning() != nil {
		t.Fatal("the control change should bind to A and end the learning")
	}
	if a.Value() != 1 {
		t.Fatal("the control change that binds should also move the control")
	}
	b.Learn()
	pn.ControlChange(1, 20, 0) // the same controller: B takes it from A
	if _, _, ok := a.CC(); ok {
		t.Fatal("a controller moves one control: A should have lost it")
	}
	if b.Value() != 0 || a.Value() != 1 {
		t.Fatal("only B should follow now")
	}
	b.Unbind()
	if _, _, ok := b.CC(); ok {
		t.Fatal("unbind should take the controller")
	}
}

func TestMIDIKinds(t *testing.T) {
	pn, _ := newPanel()
	hold := pn.Toggle("Hold", nil).BindCC(0, 64)
	presses := 0
	bang := pn.Button("Bang", func() {}).BindCC(0, 80)
	sel := pn.Select("Preset", []string{"a", "b", "c", "d"}, func(int) {}).BindCC(0, 81)

	pn.ControlChange(1, 64, 100)
	if hold.Value() != 1 {
		t.Fatal("a toggle should be on from 64")
	}
	pn.ControlChange(1, 64, 10)
	if hold.Value() != 0 {
		t.Fatal("and off below")
	}
	for _, v := range []int{127, 127, 0, 127} { // pressed, held, released, pressed
		pn.ControlChange(1, 80, v)
	}
	presses = int(bang.presses.Load())
	if presses != 2 {
		t.Fatalf("a button should be pressed once per press of the controller, got %d", presses)
	}
	pn.ControlChange(1, 81, 85)
	if sel.Value() != 2 {
		t.Fatalf("a select should pick an option over the travel, got %v", sel.Value())
	}
}

func TestPickup(t *testing.T) {
	pn, _ := newPanel()
	f := pn.Fader("F", 0, 1, Linear, nil, Pickup()).BindCC(0, 1)
	f.Set(0.5) // moved elsewhere: the controller is somewhere else
	pn.ControlChange(1, 1, 13)
	pn.ControlChange(1, 1, 30)
	if f.Value() != 0.5 {
		t.Fatalf("with pickup the control should wait for the controller, got %v", f.Value())
	}
	pn.ControlChange(1, 1, 80) // past 0.5: caught
	if math.Abs(f.Value()-80.0/127) > 1e-9 {
		t.Fatalf("once the controller passed the value it should follow, got %v", f.Value())
	}
	pn.ControlChange(1, 1, 20)
	if math.Abs(f.Value()-20.0/127) > 1e-9 {
		t.Fatal("and keep following")
	}
}

func TestSaveBindings(t *testing.T) {
	pn, _ := newPanel()
	pn.Fader("A", 0, 1, Linear, nil).BindCC(1, 10)
	b := pn.Fader("B", 0, 1, Linear, nil)
	path := filepath.Join(t.TempDir(), "midi.json")
	if err := pn.SaveBindings(path); err != nil {
		t.Fatal(err)
	}
	pn.Param("A").Unbind()
	b.BindCC(0, 11)
	if err := pn.LoadBindings(path); err != nil {
		t.Fatal(err)
	}
	if bs := pn.Bindings(); len(bs) != 1 || bs["A"] != (Binding{Channel: 1, Controller: 10}) {
		t.Fatalf("loading should restore the bindings exactly, got %v", bs)
	}
}

// A keyboard's control changes reach the panel through the patch
func TestMIDIFromKeyboard(t *testing.T) {
	p := psalm.New(nil)
	q := midi.NewQueue()
	kb := keyboard.New(q, keyboard.Config{})
	kb.AddTo(p)
	pn := New(p, "Test")
	cutoff := pn.Fader("Cutoff", 0, 1, Linear, nil).BindCC(0, 74)
	pn.MIDI(kb)
	q.PushOffset(midi.ControlChangeMsg(1, 74, 127), 0)
	p.ResetRun()
	p.Run()
	if cutoff.Value() != 1 {
		t.Fatalf("the control change should move the control, got %v", cutoff.Value())
	}
}

func TestLearnAPI(t *testing.T) {
	pn, srv := testServer(t)
	res := postJSON(t, srv.URL+"/api/learn", `{"name": "Cutoff"}`)
	res.Body.Close()
	if res.StatusCode != 204 || pn.Learning() != pn.Param("Cutoff") {
		t.Fatalf("learn should arm the control, got %d", res.StatusCode)
	}
	if res := postJSON(t, srv.URL+"/api/learn", `{"name": "Nope"}`); res.StatusCode != 404 {
		t.Fatalf("an unknown control should be 404, got %d", res.StatusCode)
	}

	events, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer events.Body.Close()
	lines := make(chan update)
	go func() {
		sc := bufio.NewScanner(events.Body)
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				var u update
				_ = json.Unmarshal([]byte(data), &u)
				lines <- u
			}
		}
	}()
	next := func() update {
		select {
		case u := <-lines:
			return u
		case <-time.After(2 * time.Second):
			t.Fatal("no event")
		}
		return update{}
	}
	if u := next(); u.MIDI == nil || u.MIDI.Learning != "Cutoff" {
		t.Fatalf("the page should hear what is learning, got %+v", u.MIDI)
	}
	pn.ControlChange(1, 74, 64)
	if u := next(); u.MIDI == nil || u.MIDI.Bindings["Cutoff"] != (Binding{Channel: 1, Controller: 74}) || u.MIDI.Learning != "" {
		t.Fatalf("the page should hear the new binding, got %+v", u.MIDI)
	}
	res = postJSON(t, srv.URL+"/api/unbind", `{"name": "Cutoff"}`)
	res.Body.Close()
	if _, _, ok := pn.Param("Cutoff").CC(); ok || res.StatusCode != 204 {
		t.Fatal("unbind should take the controller")
	}
	res = postJSON(t, srv.URL+"/api/learn", `{"name": ""}`)
	res.Body.Close()
	if pn.Learning() != nil {
		t.Fatal("an empty name should stop learning")
	}
}

// KeepBindings loads the bindings and saves them as they change
func TestKeepBindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "midi.json")
	pn, _ := newPanel()
	a := pn.Fader("A", 0, 1, Linear, nil)
	if err := pn.KeepBindings(path); err != nil {
		t.Fatalf("a file that isn't there yet is fine: %v", err)
	}
	a.BindCC(1, 12)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(path); err == nil && strings.Contains(string(data), `"controller": 12`) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	other, _ := newPanel()
	b := other.Fader("A", 0, 1, Linear, nil)
	if err := other.KeepBindings(path); err != nil {
		t.Fatal(err)
	}
	if _, ctl, ok := b.CC(); !ok || ctl != 12 {
		t.Fatal("a new panel should load the bindings saved by the first")
	}
}
