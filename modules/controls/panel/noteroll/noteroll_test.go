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

// a roll keeps a clip for every section: they are set, copied, taken away, and kept as a text with the loop
func TestRollClips(t *testing.T) {
	r, _, _ := rig(t)
	r.Set([]Note{{Start: 0, Length: 1, Key: 60, Velocity: 1}}, 4)
	r.SetClip("Verse", []Note{{Start: 1, Length: 1, Key: 62, Velocity: 0.5}, {Start: 0, Length: 2, Key: 64, Velocity: 0.7}}, 16)
	if got := r.Clips(); len(got) != 2 || got[0] != "" || got[1] != "Verse" {
		t.Fatalf("clips %v", got)
	}
	if c, ok := r.Clip("Verse"); !ok || c.Length != 16 || len(c.Notes) != 2 || c.Notes[0].Key != 64 {
		t.Fatalf("the clip of the verse %v %v", c, ok)
	}
	if _, ok := r.Clip("Chorus"); ok {
		t.Fatal("a clip that was never made")
	}
	r.CopyClip("Verse", "Chorus", "Bridge")
	if c, ok := r.Clip("Bridge"); !ok || len(c.Notes) != 2 {
		t.Fatalf("the copy %v %v", c, ok)
	}

	other := NewRoll()
	if err := other.SetStateText(r.StateText()); err != nil {
		t.Fatal(err)
	}
	if got := other.Clips(); len(got) != 4 {
		t.Errorf("clips kept: %v", got)
	}
	if c, _ := other.Clip("Chorus"); c.Length != 16 || c.Notes[1].Key != 62 || c.Notes[1].Velocity != 0.5 {
		t.Errorf("the clip kept: %v", c)
	}
	if other.Length() != 4 || len(other.Notes()) != 1 {
		t.Errorf("the loop kept: %v %v", other.Length(), other.Notes())
	}

	r.DropClip("Chorus")
	if _, ok := r.Clip("Chorus"); ok {
		t.Error("a clip taken away is still there")
	}
	r.DropClip("")
	if len(r.Notes()) != 0 || r.Length() != 4 {
		t.Errorf("the loop emptied: %v, length %v", r.Notes(), r.Length())
	}
	if err := other.SetStateText("not json"); err == nil {
		t.Error("a bad state is not reported")
	}
}

// with a song a roll plays the clip of the section from the section's start, the loop for a section without one
func TestRollClipPlaysPerSection(t *testing.T) {
	r, rc, _ := rig(t)
	r.Set([]Note{{Start: 0, Length: 1, Key: 48, Velocity: 1}}, 4) // the loop: a C every 4 beats
	r.SetClip("B", []Note{{Start: 1, Length: 1, Key: 72, Velocity: 1}}, 2)
	// the song: A for 4 beats, B for 4 beats (began on tick 960), A again (from 1920)
	r.Link(Link{Where: func(tick psalm.CtrlTick) (string, int64, bool) {
		beat := tick.TickTimestamp / 240
		switch {
		case beat < 4:
			return "A", 0, true
		case beat < 8:
			return "B", 960, true
		}
		return "A", 1920, true
	}})
	var tick int64
	run := func(beats float64) {
		end := tick + int64(beats*240)
		for ; tick < end; tick++ {
			*rc.at = tick
			r.CtrlTick(psalm.CtrlTick{TickTimestamp: tick})
		}
	}
	run(12)
	// A (the loop): 48 at tick 0; B: its clip of 2 beats goes round, the note on its second beat: 72 at 960+240 and 960+720; A: the loop from 1920: 48 at 1920 (and 2400 is 12 beats: out)
	var want []float64
	var ticks []int64
	for i, p := range rc.pitches {
		_ = i
		want = append(want, p)
		ticks = append(ticks, rc.ticks[i])
	}
	if len(want) != 4 || want[0] != 48 || want[1] != 72 || want[2] != 72 || want[3] != 48 {
		t.Fatalf("notes %v at %v", want, ticks)
	}
	if ticks[1] != 960+240 || ticks[2] != 960+240+480 || ticks[3] != 1920 {
		t.Errorf("ticks %v, want the clip of B from the start of B (960): 1200 and 1680, then A from 1920", ticks)
	}
	if r.Playing() != "" {
		t.Errorf("playing %q: A has no clip, the loop plays", r.Playing())
	}
}
