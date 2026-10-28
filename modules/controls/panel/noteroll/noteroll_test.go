package noteroll

import (
	"slices"
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

// the editor asks for melodies: for a stretch of the clip, in the scale and over the chords the link gives, the notes inside the stretch
func TestRollGenerate(t *testing.T) {
	r, _, _ := rig(t)
	r.Link(Link{
		Scale: func() (int, string, bool) { return 2, "minor", true }, // D minor
		Chords: func(string, float64) []ChordSpan {
			return []ChordSpan{{At: 0, Len: 4, Name: "Dm", Pcs: []int{2, 5, 9}}, {At: 4, Len: 4, Name: "Bb", Pcs: []int{10, 2, 5}}}
		},
	})
	r.Generate(7, 2, 6, 3, 10, GenParams{Low: 57, High: 81, Chord: 1, Form: "AA'", Phrase: 2})
	g := r.gen.Load()
	if g == nil || g.ID != 7 || len(g.Notes) != 3 {
		t.Fatalf("candidates %+v", g)
	}
	scale := map[int]bool{2: true, 4: true, 5: true, 7: true, 9: true, 10: true, 0: true} // D minor
	for _, melody := range g.Notes {
		if len(melody) == 0 {
			t.Fatal("an empty melody")
		}
		for _, n := range melody {
			if n[0] < 2 || n[0] >= 6 || n[0]+n[1] > 6+1e-6 || n[2] < 57 || n[2] > 81 || !scale[int(n[2])%12] {
				t.Fatalf("note %v is outside the stretch 2 to 6, the range or the scale", n)
			}
		}
	}
	if fmt := g.Notes[0]; len(fmt) == len(g.Notes[1]) && fmt[0] == g.Notes[1][0] && fmt[len(fmt)-1] == g.Notes[1][len(g.Notes[1])-1] {
		t.Error("two seeds, one melody")
	}
	// a stretch that is too short makes nothing new
	r.Generate(8, 3, 3.1, 1, 1, GenParams{})
	if r.gen.Load().ID != 7 {
		t.Error("a stretch of a tenth of a beat made a melody")
	}
	// the page gets the scale of the harmony
	if st := r.state(); st.Scale == nil || st.Scale.Root != 2 || !slices.Equal(st.Scale.Pcs, []int{2, 4, 5, 7, 9, 10, 0}) || st.Gen == nil || st.Gen.ID != 7 {
		t.Errorf("state: %+v %+v", st.Scale, st.Gen)
	}
}

// notes the editor asks to hear play once, on their beats, from the next tick
func TestRollPreview(t *testing.T) {
	r, rc, run := rig(t)
	r.Preview([]Note{{Start: 0, Length: 1, Key: 60, Velocity: 1}, {Start: 1, Length: 1, Key: 64, Velocity: 1}, {Start: 2.5, Length: 1, Key: 67, Velocity: 1}}, 4)
	run(8)
	if !slices.Equal(rc.pitches, []float64{60, 64, 67}) {
		t.Fatalf("pitches %v", rc.pitches)
	}
	if d := rc.ticks[1] - rc.ticks[0]; d != 240 {
		t.Errorf("the second note %d ticks after the first, want a beat (240)", d)
	}
	if d := rc.ticks[2] - rc.ticks[0]; d != 600 {
		t.Errorf("the third note %d ticks after the first, want 2.5 beats", d)
	}
	run(8) // not again
	if len(rc.pitches) != 3 {
		t.Errorf("played again: %v", rc.pitches)
	}
}

// a melody for every section: a clip each, over the chords of its own, the same seed giving the same
func TestRollGenerateSections(t *testing.T) {
	r, _, _ := rig(t)
	r.Link(Link{
		Sections: func() []Section { return []Section{{"Verse", 8}, {"Chorus", 4}} },
		Scale:    func() (int, string, bool) { return 0, "major", true },
		Chords: func(section string, _ float64) []ChordSpan {
			if section == "Verse" {
				return []ChordSpan{{At: 0, Len: 8, Name: "C", Pcs: []int{0, 4, 7}}}
			}
			return []ChordSpan{{At: 0, Len: 4, Name: "F", Pcs: []int{5, 9, 0}}}
		},
	})
	if n := r.GenerateSections(3, GenParams{Low: 60, High: 84, Chord: 1}); n != 2 {
		t.Fatalf("%d clips", n)
	}
	v, ok1 := r.Clip("Verse")
	c, ok2 := r.Clip("Chorus")
	if !ok1 || !ok2 || v.Length != 8 || c.Length != 4 || len(v.Notes) == 0 || len(c.Notes) == 0 {
		t.Fatalf("clips %+v %+v", v, c)
	}
	for _, n := range c.Notes {
		if n.Start+n.Length > 4+1e-6 {
			t.Fatalf("note %+v over the end of the chorus", n)
		}
	}
	if r.GenerateSections(3, GenParams{}) != 2 {
		t.Error("again")
	}
	v2, _ := r.Clip("Verse")
	if len(v2.Notes) == 0 {
		t.Error("an empty clip")
	}
}

// a grammar writes the form of the melody; one with a mistake is told back
func TestRollGenerateGrammar(t *testing.T) {
	r, _, _ := rig(t)
	r.Link(Link{Scale: func() (int, string, bool) { return 0, "major", true }})
	r.Generate(1, 0, 16, 1, 5, GenParams{Form: "grammar", Grammar: "S -> A A' B A", Phrase: 4})
	g := r.gen.Load()
	if g == nil || g.ID != 1 || g.Err != "" || len(g.Notes) != 1 || len(g.Notes[0]) < 8 {
		t.Fatalf("%+v", g)
	}
	r.Generate(2, 0, 16, 1, 5, GenParams{Form: "grammar", Grammar: "S -> A X"})
	if g := r.gen.Load(); g.ID != 2 || g.Err == "" || len(g.Notes) != 0 {
		t.Fatalf("a bad grammar: %+v", g)
	}
}
