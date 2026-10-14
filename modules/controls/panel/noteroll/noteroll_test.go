package noteroll

import (
	"testing"

	"github.com/almerlucke/psalm"
)

// rec keeps the notes it is given: the pitch and the tick, and how long it lasts
type rec struct {
	pitches []float64
	ticks   []int64
	secs    []float64
	vels    []float64
	at      *int64
}

func (r *rec) Receive(any, int) {}
func (r *rec) ReceiveMessage(m psalm.Message) {
	inner, _ := psalm.MsgVal(m["message"])
	r.pitches = append(r.pitches, psalm.FltVal[float64](inner["pitch"]))
	r.ticks = append(r.ticks, *r.at)
	r.secs = append(r.secs, psalm.FltVal[float64](m["duration"]))
	r.vels = append(r.vels, psalm.FltVal[float64](m["amplitude"]))
}

func rig(t *testing.T) (*Roll, *rec, func(beats float64)) {
	t.Helper()
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	t.Cleanup(func() { psalm.PopConfig() })
	p := psalm.New(nil)
	r := New(p)
	var tick int64
	rc := &rec{at: &tick}
	r.SetTargets(rc)
	return r, rc, func(beats float64) {
		end := tick + int64(beats*240)
		for ; tick < end; tick++ {
			r.CtrlTick(psalm.CtrlTick{TickTimestamp: tick})
		}
	}
}

// the notes play where they are written, the loop comes round, a note lasts its length in seconds
func TestRollPlays(t *testing.T) {
	r, rc, run := rig(t)
	r.Set([]Note{{Start: 1, Length: 0.5, Key: 62, Velocity: 0.5}, {Start: 0, Length: 1, Key: 60, Velocity: 1}, {Start: 9, Length: 1, Key: 70, Velocity: 1}}, 4)
	run(8) // two loops of 4 beats; the note at 9 is outside the loop
	if len(rc.pitches) != 4 || rc.pitches[0] != 60 || rc.pitches[1] != 62 || rc.pitches[2] != 60 {
		t.Fatalf("notes %v", rc.pitches)
	}
	if rc.ticks[0] != 0 || rc.ticks[1] != 240 || rc.ticks[2] != 960 || rc.ticks[3] != 960+240 {
		t.Fatalf("ticks %v", rc.ticks)
	}
	if rc.secs[0] < 0.499 || rc.secs[0] > 0.501 || rc.secs[1] < 0.249 || rc.secs[1] > 0.251 { // at 120 bpm a beat is half a second
		t.Fatalf("durations %v", rc.secs)
	}
	if rc.vels[1] != 0.5 {
		t.Fatalf("velocities %v", rc.vels)
	}
}

// the notes and the length are kept as numbers and come back
func TestRollValues(t *testing.T) {
	r, _, _ := rig(t)
	r.Set([]Note{{Start: 0.25, Length: 0.75, Key: 64, Velocity: 0.8}, {Start: 2, Length: 1, Key: 55, Velocity: 0.3}}, 6)
	v := r.Values()
	other := NewRoll()
	other.Apply(v)
	got := other.Notes()
	if other.Length() != 6 || len(got) != 2 || got[0] != (Note{0.25, 0.75, 64, 0.8}) || got[1].Key != 55 {
		t.Fatalf("applied %v, length %v", got, other.Length())
	}
	other.Apply(map[string]float64{}) // nothing to apply: as it is
	if len(other.Notes()) != 2 {
		t.Fatal("an empty apply should leave it")
	}
	r.Set(nil, 0.2) // at least a beat, notes only inside the loop
	if r.Length() != 1 {
		t.Fatalf("length %v", r.Length())
	}
}
