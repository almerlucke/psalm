package score

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func TestPitch(t *testing.T) {
	for name, want := range map[string]float64{"C4": 60, "A4": 69, "F#3": 54, "Bb2": 46, "62": 62, "C-1": 0, "B#3": 60, "Ebb4": 62} {
		got, err := Pitch(name)
		if err != nil || got != want {
			t.Errorf("%s: got %v (%v), want %v", name, got, err, want)
		}
	}
	for _, bad := range []string{"H4", "C", "Cx4", ""} {
		if _, err := Pitch(bad); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}

func TestParse(t *testing.T) {
	notes := MustParse("A3:1 [D4 F4 A4]:2 | r:1 E4:.5@.6 C5:1_3 G4 -:.5")
	want := []Note{
		{Pitches: []float64{57}, Beats: 1},
		{Pitches: []float64{62, 65, 69}, Beats: 2},
		{Beats: 1},
		{Pitches: []float64{64}, Beats: 0.5, Velocity: 0.6},
		{Pitches: []float64{72}, Beats: 1, Hold: 3},
		{Pitches: []float64{67}, Beats: 1}, // the length of the note before
		{Beats: 0.5},
	}
	if len(notes) != len(want) {
		t.Fatalf("%d notes, want %d: %+v", len(notes), len(want), notes)
	}
	for i := range want {
		g, w := notes[i], want[i]
		if g.Beats != w.Beats || g.Hold != w.Hold || g.Velocity != w.Velocity || len(g.Pitches) != len(w.Pitches) {
			t.Fatalf("note %d: got %+v, want %+v", i, g, w)
		}
		for j := range w.Pitches {
			if g.Pitches[j] != w.Pitches[j] {
				t.Fatalf("note %d: got %+v, want %+v", i, g, w)
			}
		}
	}
	for _, bad := range []string{"A3:x", "Q4:1", "A3:1@v", "[A3 Z9]:1", "A3:1_x"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}

func TestHelpers(t *testing.T) {
	ns := Join(Notes(1, 60, 0, 64), Chords(2, []float64{60, 64}))
	if Beats(ns) != 5 || !ns[1].IsRest() || len(ns[3].Pitches) != 2 {
		t.Fatalf("unexpected notes %+v", ns)
	}
	up := Transpose(ns, 12)
	if up[0].Pitches[0] != 72 || ns[0].Pitches[0] != 60 {
		t.Fatal("transpose should copy, not change the notes")
	}
	if Beats(Repeat(ns, 3)) != 15 {
		t.Fatal("repeat")
	}
	loud := With(ns, func(n *Note) { n.Velocity = 1 })
	if loud[0].Velocity != 1 || ns[0].Velocity != 0 {
		t.Fatal("with should copy")
	}
}

// recorder records the triggers it receives with the tick they arrive at
type recorder struct {
	*psalm.Module
	got []psalm.Message
}

func newRecorder() *recorder {
	r := &recorder{Module: psalm.NewControl()}
	r.SetSelf(r)
	return r
}

func (r *recorder) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		r.got = append(r.got, msg)
	}
}

// run drives a sequencer for ticks
func run(s *Sequencer, ticks int) {
	for t := 1; t <= ticks; t++ {
		s.CtrlTick(psalm.CtrlTick{TickTimestamp: int64(t), SampleOffset: t % 7})
	}
}

func tickOf(msg psalm.Message) int64 {
	return msg["tick"].(psalm.CtrlTick).TickTimestamp - 1
}

func TestSequencer(t *testing.T) {
	ppq := int64(psalm.PPQ())
	beatSec := 60.0 / psalm.BPM()
	notes := []Note{
		N(60, 1),
		Rest(1),
		{Pitches: []float64{64, 67}, Beats: 0.5, Velocity: 0.6, Params: map[string]any{"position": 0.3}},
		{Pitches: []float64{72}, Beats: 1.5, Hold: 3, Amp: 0.5},
	}
	seq := NewSequencer(notes, Config{Amp: 0.8, Legato: 0.5})
	rec := newRecorder()
	rec.ValueIn(seq)
	run(seq, int(4*ppq)+50)

	if len(rec.got) != 4 {
		t.Fatalf("expected 4 triggers (a chord is two), got %d", len(rec.got))
	}
	type want struct {
		tick  int64
		pitch float64
		dur   float64
		amp   float64
	}
	for i, w := range []want{
		{0, 60, 0.5 * beatSec, 0.8},
		{2 * ppq, 64, 0.25 * beatSec, 0.8},
		{2 * ppq, 67, 0.25 * beatSec, 0.8},
		{ppq * 5 / 2, 72, 3 * beatSec, 0.4},
	} {
		m := rec.got[i]
		inner := m["message"].(psalm.Message)
		if tickOf(m) != w.tick || inner["pitch"] != w.pitch || math.Abs(m["duration"].(float64)-w.dur) > 1e-9 ||
			math.Abs(m["amplitude"].(float64)-w.amp) > 1e-9 || m["command"] != "trigger" {
			t.Fatalf("trigger %d: got %v, want %+v", i, m, w)
		}
	}
	chord := rec.got[1]["message"].(psalm.Message)
	if chord["velocity"] != 0.6 || chord["position"] != 0.3 {
		t.Fatalf("expected the velocity and settings of the note, got %v", chord)
	}
	if _, ok := rec.got[0]["message"].(psalm.Message)["velocity"]; ok {
		t.Fatal("no velocity should be sent without one")
	}
	if rec.got[0]["tick"].(psalm.CtrlTick).SampleOffset != 1 {
		t.Fatal("the tick (and its sample offset) should be passed on for sample accurate timing")
	}
}

func TestSequencerLoopsAndGain(t *testing.T) {
	ppq := int64(psalm.PPQ())
	seq := NewSequencer(Notes(1, 60, 62), Config{Loop: true, Length: 4, Velocity: 0.7, Key: "frequency"})
	rec := newRecorder()
	rec.ValueIn(seq)
	run(seq, int(4*ppq))
	seq.SetGain(0.5)
	seq.CtrlTick(psalm.CtrlTick{TickTimestamp: 4*ppq + 1}) // the next loop starts

	if len(rec.got) != 3 {
		t.Fatalf("expected two notes and the first note of the next loop, got %d", len(rec.got))
	}
	last := rec.got[2]
	if tickOf(last) != 4*ppq || last["amplitude"] != 0.5 || last["message"].(psalm.Message)["frequency"] != 60.0 ||
		last["message"].(psalm.Message)["velocity"] != 0.7 {
		t.Fatalf("unexpected loop trigger %v", last)
	}
}

func TestSequencerWithoutLoopStops(t *testing.T) {
	seq := NewSequencer(Notes(1, 60), Config{})
	rec := newRecorder()
	rec.ValueIn(seq)
	run(seq, psalm.PPQ()*5)
	if len(rec.got) != 1 {
		t.Fatalf("expected one note, got %d", len(rec.got))
	}
}
