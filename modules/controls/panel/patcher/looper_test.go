package patcher

import (
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
)

// heard keeps the notes a looper plays: the tick they were asked at, the pitch and the length in seconds
type heard struct {
	*psalm.Module
	notes []heardNote
}

type heardNote struct {
	at       int64
	pitch    float64
	duration float64
}

func (h *heard) ReceiveMessage(msg psalm.Message) {
	if m, ok := msg.Msg("message"); ok {
		pitch, _ := m.FltVal("pitch")
		d, _ := msg.FltVal("duration")
		tick, _ := msg.TickVal("tick")
		h.notes = append(h.notes, heardNote{tick.TickTimestamp, pitch, d})
	}
}
func (h *heard) Receive(v any, _ int) {}

func newLoopRig(t *testing.T) (*Looper, *heard) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	t.Cleanup(func() { psalm.PopConfig() })
	_ = psalm.New(nil)
	l := NewLooper()
	l.Bars = 1 // 960 ticks
	h := &heard{Module: psalm.NewControl()}
	h.SetSelf(h)
	l.SetTargets([]psalm.Receiver{h})
	return l, h
}

func key(l *Looper, on bool, id string, tick int64, pitch float64) {
	msg := psalm.Message{"command": "trigger", "noteOn": id, "tick": psalm.CtrlTick{TickTimestamp: tick}, "message": psalm.Message{"pitch": pitch, "velocity": 0.8}}
	if !on {
		msg = psalm.Message{"command": "trigger", "noteOff": id, "tick": psalm.CtrlTick{TickTimestamp: tick}}
	}
	l.ReceiveMessage(msg)
}

func run(l *Looper, from, to int64) {
	for tk := from; tk <= to; tk++ {
		l.late(psalm.CtrlTick{TickTimestamp: tk})
	}
}

// a layer recorded in a pass plays in the next, on the grid; a new layer goes on top; undo takes it away; the MIDI file has them
func TestLooperRecordsAndPlays(t *testing.T) {
	l, h := newLoopRig(t)
	l.ReceiveMessage(psalm.Message{"record": 1.0})
	run(l, 0, 0) // the pass begins: recording
	if _, rec := l.Layers(); !rec {
		t.Fatal("not recording at the start of the pass")
	}
	run(l, 1, 99)
	key(l, true, "a", 100, 60) // 100 ticks in: on the sixteenth grid (60) it is 120
	run(l, 100, 190)
	key(l, false, "a", 190, 60)                    // 90 ticks long: 120
	if len(h.notes) != 1 || h.notes[0].at != 100 { // what is played goes on to the synths as it is
		t.Errorf("the key was not passed on: %v", h.notes)
	}
	l.ReceiveMessage(psalm.Message{"record": 0.0}) // let go: the layer is done at the end of the pass
	run(l, 191, 959)
	if n, rec := l.Layers(); n != 0 || !rec {
		t.Errorf("before the pass is over: %d layers, recording %v", n, rec)
	}
	run(l, 960, 960)
	if n, rec := l.Layers(); n != 1 || rec {
		t.Fatalf("after the pass: %d layers, recording %v", n, rec)
	}
	h.notes = nil
	run(l, 961, 960+960) // the next pass: the note at 120
	if len(h.notes) != 1 || h.notes[0].at != 960+120 || h.notes[0].pitch != 60 {
		t.Fatalf("played %+v, want one note at tick %d", h.notes, 960+120)
	}
	if want := 120 * 60.0 / 120 / 240; math.Abs(h.notes[0].duration-want) > 1e-9 { // 120 ticks at 120 bpm
		t.Errorf("duration %v, want %v", h.notes[0].duration, want)
	}

	f := l.File(2)
	if len(f.Tracks) != 2 {
		t.Fatalf("%d tracks, want the tempo track and one layer", len(f.Tracks))
	}
	var ons []int64
	for _, e := range f.Tracks[1].Events {
		if !e.IsMeta() && e.Message.Kind() == midi.NoteOn {
			ons = append(ons, e.Tick)
		}
	}
	if len(ons) != 2 || ons[0] != 240 || ons[1] != 1920+240 { // 120 ticks of 240 to the quarter are 240 of 480; the loop is 1920 long
		t.Errorf("note ons at %v", ons)
	}

	l.ReceiveMessage(psalm.Message{"undo": 1.0})
	run(l, 1921, 1921)
	if n, _ := l.Layers(); n != 0 {
		t.Errorf("after undo %d layers", n)
	}
}

// off the grid it stays as played
func TestLooperQuantizeOff(t *testing.T) {
	l, _ := newLoopRig(t)
	l.ReceiveMessage(psalm.Message{"quantize": 0, "record": 1.0})
	run(l, 0, 0)
	key(l, true, "a", 101, 62)
	key(l, false, "a", 191, 62)
	if len(l.cur) != 1 || l.cur[0].start != 101 || l.cur[0].length != 90 {
		t.Errorf("recorded %+v, want a note at 101, 90 long", l.cur)
	}
}

func TestLooperDescribes(t *testing.T) {
	l, _ := newLoopRig(t)
	if got := len(l.Describe().Params); got != 10 {
		t.Errorf("%d settings", got)
	}
}

// a layer can be taken away by itself (by the number the page has for it), and the page's view follows
func TestLooperDropsALayer(t *testing.T) {
	l, _ := newLoopRig(t)
	l.ReceiveMessage(psalm.Message{"record": 1.0})
	for pass := range int64(3) { // three passes, a note in each, pitches 60, 62, 64
		base := pass * 960
		run(l, base, base+49)
		key(l, true, "k", base+50, 60+float64(pass)*2)
		run(l, base+50, base+99)
		key(l, false, "k", base+100, 60+float64(pass)*2)
		run(l, base+100, base+959)
	}
	run(l, 2880, 2880)
	l.ReceiveMessage(psalm.Message{"record": 0.0})
	v := l.view.Load()
	if v == nil || len(v.Layers) != 3 {
		t.Fatalf("view %+v, want three layers", v)
	}
	mid := v.Layers[1].ID
	l.Drop(mid)
	run(l, 2881, 2881)
	v = l.view.Load()
	if len(v.Layers) != 2 || v.Layers[0].Notes[0][2] != 60 || v.Layers[1].Notes[0][2] != 64 {
		t.Fatalf("after taking the middle away: %+v", v.Layers)
	}
	if n, _ := l.Layers(); n != 2 {
		t.Errorf("%d layers", n)
	}
	l.Drop(mid) // already gone: nothing
	run(l, 2882, 2882)
	if n, _ := l.Layers(); n != 2 {
		t.Errorf("%d layers after asking again", n)
	}
	if f := l.File(1); len(f.Tracks) != 3 {
		t.Errorf("%d tracks, want the tempo track and two layers", len(f.Tracks))
	}
	if s, ok := l.state().(looperState); !ok || s.Bars != 1 || len(s.Layers) != 2 {
		t.Errorf("state %+v", s)
	}
}

// all the layers in one script: one clock and one table with the notes of both, and the files follow the setting
func TestLooperScriptAll(t *testing.T) {
	l, _ := newLoopRig(t)
	l.layers = [][]loopNote{{{start: 0, length: 60, pitch: 60, velocity: 0.8}}, {{start: 150, length: 30, pitch: 64.5, velocity: 0.5}, {start: 90, length: 60, pitch: 67, velocity: 1}}}
	l.ids = []int{0, 1}
	s := l.ScriptAll()
	for _, want := range []string{"All 2 layers", `clock("1/32")`, "const LOOP = 32", "// layer 1, bar 1", "// layer 2, bar 1", `[0, 2, "C4", 0.8]`, `[5, 1, 64.5, 0.5]`, `[3, 2, "G4", 1]`} {
		if !strings.Contains(s, want) {
			t.Errorf("no %q in\n%s", want, s)
		}
	}
	if strings.Index(s, `[3, 2, "G4"`) > strings.Index(s, `[5, 1, 64.5`) {
		t.Error("the notes of a layer are not in order")
	}
	dir := t.TempDir()
	l.dir = dir
	l.ScriptFiles = 1
	l.exportScripts()
	l.ScriptFiles = 0
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m, _ := filepath.Glob(filepath.Join(dir, "loop-*.js")); len(m) == 1 && !strings.Contains(m[0], "layer") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("no single file written")
}
