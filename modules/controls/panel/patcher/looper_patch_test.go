package patcher_test

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// notes of a clock through a looper: one pass is recorded, and plays in the passes after it, along with what comes in
func TestLooperInAPatch(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	rec := &played{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	kinds := append(patcher.Generators(), patcher.Kind{Name: "rec", Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}},
		Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: rec}, nil }})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	add := func(kind string) string {
		t.Helper()
		id, err := pt.Add(kind, "", 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	connect := func(c patcher.Cable) {
		t.Helper()
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	clock, note, loop, syn := add("clock"), add("notes"), add("looper"), add("rec")
	if err := pt.SetText(clock, "pattern", "x . . . . . . ."); err != nil { // a note every half bar... of sixteenths: 8 steps, a bar is 16
		t.Fatal(err)
	}
	connect(patcher.Cable{From: clock, To: note})
	connect(patcher.Cable{From: note, To: loop})
	connect(patcher.Cable{From: loop, To: syn})
	set := pt.SettingsOf(loop)
	set.Apply(map[string]float64{"bars": 1, "record": 1})
	run := func(seconds float64) {
		for range int(seconds * 44100 / 256) {
			p.ResetRun()
			p.Run()
		}
	}
	run(2) // the first bar (2 s at 120 bpm): recorded
	set.Apply(map[string]float64{"record": 0})
	run(0.1) // the pass ends, the layer is done
	before := len(rec.pitches)
	run(4) // two passes more
	if before < 3 {
		t.Fatalf("%d notes in the first pass", before)
	}
	// the clock goes on, so the notes come in as before (2 a pass, a note every 8 sixteenths); the loop plays the 2 recorded as well
	if got, want := len(rec.pitches)-before, 4*2; got < want-2 || got > want+2 {
		t.Errorf("%d notes in two more passes, want about %d (the clock's and the loop's)", got, want)
	}
	if n, recording := patcher.LooperLayers(pt, loop); n != 1 || recording {
		t.Errorf("%d layers, recording %v, want 1 and not recording", n, recording)
	}
}
