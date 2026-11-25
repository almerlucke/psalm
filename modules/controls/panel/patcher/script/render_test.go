package script

import (
	"slices"
	"strings"
	"testing"
)

const renderMelody = `
function render(p) {
	const key = scale("D minor")
	const walk = genny.rand(0, 7)
	const notes = []
	for (let i = 0; i < 8; i++) {
		notes.push({pitch: key.pitch(1 + Math.floor(walk.next()), 3), start: i, duration: 0.5, velocity: 0.7, channel: i % 2})
	}
	return notes
}`

func TestRenderNotes(t *testing.T) {
	p, err := NewRuntime().Render(renderMelody, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Notes) != 8 || p.StepsPerBeat != 4 {
		t.Fatalf("%d notes, %v steps a beat", len(p.Notes), p.StepsPerBeat)
	}
	if p.Steps != 16 { // the end of the notes (7.5 steps), up to a bar of 4 beats of 4 steps
		t.Fatalf("length %v steps, want 16", p.Steps)
	}
	if p.Notes[1].Channel != 1 || p.Notes[1].Start != 1 {
		t.Fatalf("note 1: %+v", p.Notes[1])
	}
}

// the same seed makes the same piece, another one a different piece
func TestRenderSeed(t *testing.T) {
	render := func(seed uint64) []RenderedNote {
		p, err := NewRuntime().Render(renderMelody, seed)
		if err != nil {
			t.Fatal(err)
		}
		return p.Notes
	}
	a, b, c := render(7), render(7), render(8)
	if !slices.Equal(a, b) {
		t.Fatal("the same seed made another piece")
	}
	if slices.Equal(a, c) {
		t.Fatal("another seed made the same piece")
	}
}

func TestRenderObjectResult(t *testing.T) {
	p, err := NewRuntime().Render(`function render() { return {stepsPerBeat: 2, length: 6, notes: [{pitch: "C4", start: 0}, {pitch: 62, start: 3, duration: 2}]} }`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if p.StepsPerBeat != 2 || p.Steps != 6 || len(p.Notes) != 2 || p.Notes[0].Pitch != 60 {
		t.Fatalf("%+v", p)
	}
}

func TestRenderLog(t *testing.T) {
	p, err := NewRuntime().Render(`function render() { log("hi", [1, 2]); return [] }`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Log) != 1 || p.Log[0] != "hi [1,2]" {
		t.Fatalf("log %q", p.Log)
	}
}

// a fugue of genny.fugue is a piece: one channel for each voice
func TestRenderFugue(t *testing.T) {
	code := `
function render(p) {
	const key = genny.scale(62, "minor")
	const f = genny.fugue({degrees: [0, 2, 4, 2, 0, -1, 0], durations: [4, 2, 2, 4, 2, 1, 1]}, {scale: key, voices: 3})
	f.episode({repeats: 2})
	f.middle(0, "answer", 4, true)
	f.close({bars: 2})
	const notes = []
	f.voices().forEach((voice, i) => voice.forEach(n => notes.push({...n, channel: i})))
	return {length: Math.ceil(f.end() / 16) * 16, notes}
}`
	p, err := NewRuntime().Render(code, 3)
	if err != nil {
		t.Fatal(err)
	}
	channels := map[int]bool{}
	for _, n := range p.Notes {
		channels[n.Channel] = true
	}
	if len(channels) != 3 || len(p.Notes) < 30 || p.Steps < 64 {
		t.Fatalf("%d notes on %d channels, %v steps", len(p.Notes), len(channels), p.Steps)
	}
}

// what a script logged before it failed is still given back
func TestRenderLogWithError(t *testing.T) {
	p, err := NewRuntime().Render(`function render() { log("before"); throw new Error("boom") }`, 1)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error %v", err)
	}
	if len(p.Log) != 1 || p.Log[0] != "before" {
		t.Fatalf("log %q", p.Log)
	}
}

// the notes held down are given to render, nothing held gives empty lists
func TestRenderHeld(t *testing.T) {
	code := `function render(p) { return {length: 16, notes: p.held.map((h, i) => ({pitch: h.pitch + 12, start: i * 2, duration: 2, velocity: h.velocity}))} }`
	p, err := NewRuntime().RenderHeld(code, 1, []HeldNote{{60, 0.5}, {64, 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Notes) != 2 || p.Notes[1].Pitch != 76 || p.Notes[0].Velocity != 0.5 || p.Notes[1].Start != 2 {
		t.Fatalf("%+v", p.Notes)
	}
	p, err = NewRuntime().RenderHeld(`function render(p) { return p.pitches.length === 0 && p.held.length === 0 ? [] : [{pitch: 1}] }`, 1, nil)
	if err != nil || len(p.Notes) != 0 {
		t.Fatalf("%v %v", p.Notes, err)
	}
}

// p.chord says what the notes held are
func TestRenderChord(t *testing.T) {
	code := `function render(p) { return p.chord ? [{pitch: 60, start: 0, duration: 1, channel: p.chord.inversion}] : [] }`
	p, err := NewRuntime().RenderHeld(code, 1, []HeldNote{{64, 1}, {67, 1}, {72, 1}})
	if err != nil || len(p.Notes) != 1 || p.Notes[0].Channel != 1 {
		t.Fatalf("%+v %v", p.Notes, err)
	}
	p, err = NewRuntime().RenderHeld(code, 1, []HeldNote{{64, 1}})
	if err != nil || len(p.Notes) != 0 {
		t.Fatalf("%+v %v", p.Notes, err)
	}
}

func TestRenderErrors(t *testing.T) {
	for _, c := range []struct{ code, want string }{
		{`var x = ;`, "line 1"},
		{`var x = 1`, "no function render"},
		{`function render() { return 5 }`, "must return"},
		{`function render() { return [{pitch: 200}] }`, "outside 0..127"},
		{`function render() { return [{pitch: 60, channel: 16}] }`, "channel"},
		{`function render() { return [{pitch: 60, start: -1}] }`, "start"},
		{`function render() { throw new Error("boom") }`, "boom"},
		{`function render() { return genny.nothing() }`, ""},
	} {
		_, err := NewRuntime().Render(c.code, 1)
		if err == nil {
			t.Errorf("%s: no error", c.code)
		} else if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %q, want %q in it", c.code, err, c.want)
		}
	}
}
