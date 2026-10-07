package keyboard

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
)

type recorder struct{ msgs []psalm.Message }

func (r *recorder) Receive(value any, _ int) {
	if m, ok := psalm.MsgVal(value); ok {
		r.ReceiveMessage(m)
	}
}
func (r *recorder) ReceiveMessage(msg psalm.Message) { r.msgs = append(r.msgs, msg) }

// setup returns a patch with a keyboard playing a recorder from a queue
func setup(cfg Config) (*psalm.Psalm, *midi.Queue, *Keyboard, *recorder) {
	p := psalm.New(nil)
	q := midi.NewQueue()
	rec := &recorder{}
	k := New(q, cfg, rec)
	k.AddTo(p)
	return p, q, k, rec
}

func run(p *psalm.Psalm) {
	p.ResetRun()
	p.Run()
}

func TestNotes(t *testing.T) {
	p, q, _, rec := setup(Config{Transpose: 12, Dynamics: 1})
	q.PushOffset(midi.NoteOnMsg(1, 60, 127), 10)
	q.PushOffset(midi.NoteOnMsg(2, 64, 64), 20)
	q.PushOffset(midi.NoteOffMsg(1, 60, 0), 30)
	run(p)
	if len(rec.msgs) != 3 {
		t.Fatalf("got %v", rec.msgs)
	}
	on := rec.msgs[0]
	note := on.MustMsg("message")
	if on["noteOn"] != "midi-1-60" || note.MustFltVal("pitch") != 72 || note.MustFltVal("velocity") != 1 ||
		on.MustFltVal("amplitude") != 1 || on.MustTickVal("tick").SampleOffset != 10 {
		t.Fatalf("note on %v", on)
	}
	if v := rec.msgs[1].MustFltVal("amplitude"); math.Abs(v-64.0/127) > 1e-9 {
		t.Fatalf("amplitude %v", v)
	}
	if off := rec.msgs[2]; off["noteOff"] != "midi-1-60" || off.MustTickVal("tick").SampleOffset != 30 {
		t.Fatalf("note off %v", off)
	}
	// a buffer later the input is read again
	q.PushOffset(midi.NoteOnMsg(1, 62, 100), 0)
	run(p)
	if len(rec.msgs) != 4 {
		t.Fatalf("second buffer: %v", rec.msgs)
	}
}

func TestVelocityCurve(t *testing.T) {
	k := New(midi.NewQueue(), Config{})
	if v := k.Velocity(127); v != 1 {
		t.Fatal(v)
	}
	lin := k.Velocity(64)
	k.cfg.Curve = Soft
	soft := k.Velocity(64)
	k.cfg.Curve = Hard
	hard := k.Velocity(64)
	if !(soft > lin && lin > hard) {
		t.Fatalf("soft %v, linear %v, hard %v", soft, lin, hard)
	}
	k.cfg.FixedVelocity = 0.7
	if k.Velocity(10) != 0.7 {
		t.Fatal("fixed")
	}
	k.cfg.FixedVelocity = 0
	k.cfg.VelocityMap = func(v float64) float64 { return 1 - v }
	if k.Velocity(127) != 0 {
		t.Fatal("map")
	}
}

func TestControls(t *testing.T) {
	p, q, k, rec := setup(Config{Channel: 2, PitchBend: "bend", Pressure: "pressure"})
	var program, mod = -1, -1.0
	var kinds []midi.Kind
	k.OnProgram(func(pr int, _ psalm.CtrlTick) { program = pr })
	k.OnControl(midi.CCModWheel, func(v float64, _ psalm.CtrlTick) { mod = v })
	k.OnEvent(func(e midi.Event, _ psalm.CtrlTick) { kinds = append(kinds, e.Message.Kind()) })
	q.PushOffset(midi.NoteOnMsg(1, 60, 100), 0) // another channel
	q.PushOffset(midi.ControlChangeMsg(2, midi.CCSustain, 127), 1)
	q.PushOffset(midi.ControlChangeMsg(2, midi.CCModWheel, 127), 2)
	q.PushOffset(midi.ProgramChangeMsg(2, 7), 3)
	q.PushOffset(midi.PitchBendMsg(2, 0), 4)
	q.PushOffset(midi.ChannelPressureMsg(2, 127), 5)
	q.PushOffset(midi.StartMsg, 6)
	run(p)
	if program != 7 || mod != 1 || len(kinds) != 6 {
		t.Fatalf("program %v, mod %v, kinds %v", program, mod, kinds)
	}
	want := []psalm.Message{{"sustain": 1.0}, {"bend": -1.0}, {"pressure": 1.0}}
	if len(rec.msgs) != len(want) {
		t.Fatalf("got %v", rec.msgs)
	}
	for i, w := range want {
		for key, v := range w {
			if rec.msgs[i][key] != v {
				t.Fatalf("got %v, want %v", rec.msgs[i], w)
			}
		}
	}
}

func TestHoldNotes(t *testing.T) {
	p, q, _, rec := setup(Config{HoldNotes: true})
	q.PushOffset(midi.NoteOnMsg(1, 60, 100), 0)
	q.PushOffset(midi.ControlChangeMsg(1, midi.CCSustain, 127), 1)
	q.PushOffset(midi.NoteOffMsg(1, 60, 0), 2)
	q.PushOffset(midi.NoteOnMsg(1, 62, 100), 3)
	run(p)
	if len(rec.msgs) != 2 {
		t.Fatalf("the pedal keeps the note: %v", rec.msgs)
	}
	q.PushOffset(midi.ControlChangeMsg(1, midi.CCSustain, 0), 0)
	run(p)
	if len(rec.msgs) != 3 || rec.msgs[2]["noteOff"] != "midi-1-60" {
		t.Fatalf("the pedal up releases the released note only: %v", rec.msgs)
	}
	q.PushOffset(midi.ControlChangeMsg(1, midi.CCAllNotesOff, 0), 0)
	run(p)
	if len(rec.msgs) != 4 || rec.msgs[3]["noteOff"] != "midi-1-62" {
		t.Fatalf("all notes off: %v", rec.msgs)
	}
}

func TestClock(t *testing.T) {
	p, q, k, _ := setup(Config{FollowClock: true})
	sr := psalm.SampleRate()
	bs := psalm.BufferSize()
	// 100 BPM: a clock every 60/100/24 seconds
	every := 60 / 100.0 / 24 * sr
	next := 0.0
	for b := 0; b < 200; b++ {
		for ; next < float64((b+1)*bs); next += every {
			q.PushOffset(midi.ClockMsg, int(next)-b*bs)
		}
		run(p)
	}
	if bpm := k.Env().BPM(); math.Abs(bpm-100) > 0.5 {
		t.Fatalf("tempo %v (clock %v)", bpm, k.Tempo())
	}
}
