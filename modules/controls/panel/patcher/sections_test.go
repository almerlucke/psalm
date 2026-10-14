package patcher_test

import (
	"math"
	"slices"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

func TestParseCurve(t *testing.T) {
	for _, c := range []struct {
		text string
		at   [3]float64 // at 0, 0.5, 1
	}{
		{"0.5", [3]float64{0.5, 0.5, 0.5}},
		{"0 → 1", [3]float64{0, 0.5, 1}},
		{"0 -> 1 smooth", [3]float64{0, 0.5, 1}},
		{"100 → 400 exp", [3]float64{100, 200, 400}},
	} {
		cv, err := patcher.ParseCurve(c.text)
		if err != nil {
			t.Errorf("%q: %v", c.text, err)
			continue
		}
		for i, x := range []float64{0, 0.5, 1} {
			if math.Abs(cv.At(x)-c.at[i]) > 1e-9 {
				t.Errorf("%q at %v: %v, want %v", c.text, x, cv.At(x), c.at[i])
			}
		}
	}
	for _, bad := range []string{"", "a", "0 → ", "0 to 1", "0 → 1 wobbly", "0 → 1 exp"} {
		if _, err := patcher.ParseCurve(bad); err == nil {
			t.Errorf("%q read", bad)
		}
	}
}

func TestParsePart(t *testing.T) {
	p, err := patcher.ParsePart("C3 - Eb3+G3 . [C4 D4]*2 60?25")
	if err != nil {
		t.Fatal(err)
	}
	want := []patcher.Hit{
		{At: 0, Len: 2, Chance: 1, Pitches: []float64{48}},
		{At: 2, Len: 1, Chance: 1, Pitches: []float64{51, 55}},
		{At: 4, Len: 1, Chance: 1, Pitches: []float64{60}},
		{At: 5, Len: 1, Chance: 1, Pitches: []float64{62}},
		{At: 6, Len: 1, Chance: 0.25, Pitches: []float64{60}},
	}
	ok := p.Steps == 7 && len(p.Hits) == len(want)
	for i := 0; ok && i < len(want); i++ {
		h, w := p.Hits[i], want[i]
		ok = math.Abs(h.At-w.At) < 1e-9 && math.Abs(h.Len-w.Len) < 1e-9 && h.Chance == w.Chance && slices.Equal(h.Pitches, w.Pitches)
	}
	if !ok {
		t.Errorf("%v steps %v, want 7 steps %v", p.Steps, p.Hits, want)
	}
	if p, _ := patcher.ParsePart(". - C3"); len(p.Hits) != 1 || p.Hits[0].Len != 1 {
		t.Errorf("a tie after a rest holds nothing: %v", p.Hits)
	}
	for _, bad := range []string{"", "H9", "[C3", "C3]", "[]", "C3*", "C3?200", "C3!"} {
		if _, err := patcher.ParsePart(bad); err == nil {
			t.Errorf("%q read", bad)
		}
	}
}

// dc plays a constant 1: a sound to hear a gate let through
type dc struct{ *psalm.Module }

func (d *dc) Synthesize() {
	for _, ch := range d.Output(0).Channels {
		for i := range ch {
			ch[i] = 1
		}
	}
}

// a lane through the sections: its curve per section, as the song goes
func TestLane(t *testing.T) {
	r := newSongRig(t, "A 1, B 1") // a bar: 2 s
	lane := r.add("lane")
	if err := r.pt.SetText(lane, "curves", "A: 0 → 1\nB: 0.5"); err != nil || r.pt.TextError(lane, "curves") != "" {
		t.Fatal(err, r.pt.TextError(lane, "curves"))
	}
	l := r.pt.Module(lane).(*patcher.Lane)
	r.run(1) // half way through A
	if v := l.Value(); math.Abs(v-0.5) > 0.02 {
		t.Errorf("half way through A at %v, want 0.5", v)
	}
	r.run(0.98)
	if v := l.Value(); v < 0.97 {
		t.Errorf("at the end of A at %v, want 1", v)
	}
	r.run(1)
	if v := l.Value(); v != 0.5 {
		t.Errorf("in B at %v, want 0.5", v)
	}
}

// a gate lets triggers, notes and sound through in its sections, when it is on; a note let go always
func TestGate(t *testing.T) {
	r := newSongRig(t, "A 1, B 1")
	clock, gate, hear := r.add("clock"), r.add("gate"), r.add("hear")
	r.pt.SettingsOf(clock).Apply(map[string]float64{"division": 0}) // quarters
	if err := r.pt.SetText(gate, "sections", "B"); err != nil || r.pt.TextError(gate, "sections") != "" {
		t.Fatal(err, r.pt.TextError(gate, "sections"))
	}
	r.connect(patcher.Cable{From: clock, To: gate, In: 2})
	r.connect(patcher.Cable{From: gate, Out: 2, To: hear, In: 1})
	r.run(3.9)                                // A, B
	inBar := func(at []int64) (out []int64) { // the bar a trigger falls in
		for _, a := range at {
			out = append(out, a/88200)
		}
		return
	}
	if got := inBar(r.hear.at[1]); len(got) < 3 || slices.ContainsFunc(got, func(b int64) bool { return b != 1 }) {
		t.Errorf("triggers through the gate in bars %v, want only in B (bar 1)", got)
	}

	// switched off by its tile (the song's parts): nothing through
	g := r.pt.Module(gate).(*patcher.Gate)
	if gs := patcher.SongOf(g.Env()).Gates(); len(gs) != 1 || gs[0].Name() != "part" {
		t.Fatalf("the song's parts %v", gs)
	}
	r.pt.SettingsOf(gate).Apply(map[string]float64{"on": 0})
	r.hear.at = map[int][]int64{}
	r.run(4)
	if n := len(r.hear.at[1]); n != 0 {
		t.Errorf("%d triggers through a gate switched off", n)
	}
	if v := r.pt.SettingsOf(gate).Values()["on"]; v != 0 {
		t.Errorf("the gate off kept as %v", v)
	}

	// a note let go goes through a closed gate, a note played doesn't
	got := &played{Module: psalm.NewControl()}
	g.SetTargets([]psalm.Receiver{got})
	g.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "k", "tick": psalm.CtrlTick{}, "amplitude": 1.0, "message": psalm.Message{"pitch": 60.0}})
	off := &offs{}
	g.SetTargets([]psalm.Receiver{off})
	g.ReceiveMessage(psalm.Message{"command": "trigger", "noteOff": "k", "tick": psalm.CtrlTick{}})
	if len(got.pitches) != 0 || off.n != 1 {
		t.Errorf("closed: %d notes through (want 0), %d let go (want 1)", len(got.pitches), off.n)
	}
}

// offs counts notes let go
type offs struct{ n int }

func (o *offs) ReceiveMessage(m psalm.Message) {
	if _, ok := m["noteOff"]; ok {
		o.n++
	}
}
func (o *offs) Receive(any, int) {}

// the sound through a gate: there in its sections, silent in the others
func TestGateSound(t *testing.T) {
	r := newSongRig(t, "A 1, B 1")
	gate := r.add("gate")
	_ = r.pt.SetText(gate, "sections", "A")
	g := r.pt.Module(gate).(*patcher.Gate)
	src := &dc{Module: psalm.NewModule(nil, []int{psalm.Stereo})}
	src.SetSelf(src)
	r.p.Do(func() { src.AddTo(r.p); src.Connect(0, g, 0) })
	r.connect(patcher.Cable{From: gate, To: "out"})
	level := func() float64 { return float64(r.pt.Out().Output(0).Channels[0][255]) }
	r.run(1)
	if v := level(); math.Abs(v-1) > 1e-6 {
		t.Errorf("in A: %v, want 1", v)
	}
	r.run(1.5)
	if v := level(); v != 0 {
		t.Errorf("in B: %v, want 0", v)
	}
}

// a part plays its notes: held, chords, per section
func TestPart(t *testing.T) {
	r := newSongRig(t, "A 1, B 1")
	part, syn := r.add("part"), r.add("rec")
	pp := r.pt.Module(part).(*patcher.Part) // before it plays: its settings would come with its first tick
	pp.Division, pp.Length = 0, 1           // quarters, legato
	if err := r.pt.SetText(part, "notes", "C3 - Eb3+G3 .\nB: 60*4"); err != nil || r.pt.TextError(part, "notes") != "" {
		t.Fatal(err, r.pt.TextError(part, "notes"))
	}
	r.connect(patcher.Cable{From: part, To: syn})
	r.run(3.9) // A, B
	if want := []float64{48, 51, 55, 60}; !slices.Equal(r.rec.pitches, want) {
		t.Fatalf("pitches %v, want %v", r.rec.pitches, want)
	}
	if want := []float64{1, 0.5, 0.5, 2}; !slices.EqualFunc(r.rec.seconds, want, func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }) {
		t.Errorf("lengths %v, want %v (C3 held two quarters, a chord a quarter, B's a bar)", r.rec.seconds, want)
	}
	if d := r.rec.at[1] - r.rec.at[0]; math.Abs(float64(d)-44100) > 92 {
		t.Errorf("the chord %d samples after C3, want two quarters (44100)", d)
	}
}
