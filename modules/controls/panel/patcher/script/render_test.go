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
