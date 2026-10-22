package patcher

import (
	"math"
	"testing"

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
	if got := len(l.Describe().Params); got != 8 {
		t.Errorf("%d settings", got)
	}
}
