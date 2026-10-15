package catalog_test

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// the drum machine: a kind of the patch, its grid kept with the patch and loaded again
func TestDrumMachine(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	kinds := func() []patcher.Kind {
		return catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Drums: "../../../../../assets/sounds/drums"})
	}
	dir := t.TempDir()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", kinds(), patcher.Keep(dir))
	id, err := pt.Add("drummachine", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	v := pt.SettingsOf(id).Values()
	if v["Beat/0/0/0/0"] == 0 && len(v) < 10 {
		t.Fatalf("no beat in its settings: %v", v)
	}
	if err := pt.Save("beat"); err != nil {
		t.Fatal(err)
	}
	p2 := psalm.New(nil)
	pt2 := patcher.New(p2, panel.New(p2, "B"), "Patch", kinds(), patcher.Keep(dir))
	if err := pt2.Load("beat"); err != nil {
		t.Fatal(err)
	}
	if got := pt2.SettingsOf(id).Values(); len(got) != len(v) {
		t.Errorf("loaded %d settings, saved %d", len(got), len(v))
	}
}

// the sequencer plays with the patch: the kit is heard
func TestDrumMachinePlays(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Drums: "../../../../../assets/sounds/drums"}))
	p.Out(pt.Out())
	id, err := pt.Add("drummachine", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pt.Connect(patcher.Cable{From: id, To: "out"}); err != nil {
		t.Fatal(err)
	}
	var peak float32
	for range 44100 * 2 / 256 {
		p.ResetRun()
		p.Run()
		for _, s := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, s, -s)
		}
	}
	if peak < 0.01 {
		t.Errorf("silent, peak %v", peak)
	}
}

// the sample of a row is chosen from the folder, and kept with the patch by the sample, not its place in the list
func TestDrumMachineSamples(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Drums: "../../../../../assets/sounds/drums"}))
	id, err := pt.Add("drummachine", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	v := pt.SettingsOf(id).Values()
	kick, snare := v["kick sample id"], v["snare sample id"]
	if kick == 0 || snare == 0 || kick == snare {
		t.Fatalf("the samples of the rows should be in the settings: %v %v", kick, snare)
	}
	if _, ok := v["kick sample"]; ok {
		t.Error("a sample is kept by its id, not its place in the list")
	}
	pt.SettingsOf(id).Apply(map[string]float64{"snare sample id": kick})
	if got := pt.SettingsOf(id).Values()["snare sample id"]; got != kick {
		t.Errorf("the snare should have the kick's sample now: %v, want %v", got, kick)
	}
	pt.SettingsOf(id).Apply(map[string]float64{"snare sample id": 12345}) // gone: it stays
	if got := pt.SettingsOf(id).Values()["snare sample id"]; got != kick {
		t.Errorf("a sample that is gone should leave the one chosen: %v", got)
	}
}

// the pattern is asked for from the patch: by a value (a sequence, a song's section), or the next one by a trigger
func TestDrumMachinePatternInput(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Drums: "../../../../../assets/sounds/drums"}))
	p.Out(pt.Out())
	drum, _ := pt.Add("drummachine", "", 0, 0)
	clock, _ := pt.Add("clock", "", 0, 0)
	seq, _ := pt.Add("sequence", "", 0, 0)
	if err := pt.SetText(seq, "values", "3"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: clock, To: seq}, {From: seq, To: drum, In: 0}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	run := func(seconds float64) {
		for range int(seconds * 44100 / 256) {
			p.ResetRun()
			p.Run()
		}
	}
	run(1)
	if got := pt.SettingsOf(drum).Values()["Beat"]; got != 3 {
		t.Errorf("the sequence asked for pattern D (3): %v", got)
	}
	pt.Clear()
	drum, _ = pt.Add("drummachine", "", 0, 0)
	clock, _ = pt.Add("clock", "", 0, 0)
	if _, err := pt.Connect(patcher.Cable{From: clock, To: drum, In: 1}); err != nil {
		t.Fatal(err)
	}
	run(0.3)
	if got := pt.SettingsOf(drum).Values()["Beat"]; got < 1 {
		t.Errorf("every trigger should ask for the next pattern: %v", got)
	}
	if got := pt.SettingsOf(drum).Values()["Beat"]; got > 3 {
		t.Errorf("only A to D are in the rotation to start with: %v", got)
	}
	if err := pt.Remove(drum); err != nil { // its sequencer goes with it
		t.Fatal(err)
	}
	run(0.2)
}

// the note editor: its notes play a synth of the patch, and are kept with the patch
func TestNoteRoll(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	dir := t.TempDir()
	mk := func() (*psalm.Psalm, *patcher.Patcher) {
		p := psalm.New(nil)
		pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(dir))
		p.Out(pt.Out())
		return p, pt
	}
	p, pt := mk()
	roll, _ := pt.Add("noteroll", "", 0, 0)
	synth, _ := pt.Add("subtractive", "", 0, 0)
	for _, c := range []patcher.Cable{{From: roll, To: synth}, {From: synth, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 1, "n/0/key": 60, "n/0/velocity": 0.9})
	var peak float32
	for range 44100 * 2 / 256 {
		p.ResetRun()
		p.Run()
		for _, s := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, s, -s)
		}
	}
	if peak < 0.01 {
		t.Errorf("the roll should play the synth: peak %v", peak)
	}
	if err := pt.Save("roll"); err != nil {
		t.Fatal(err)
	}
	_, pt2 := mk()
	if err := pt2.Load("roll"); err != nil {
		t.Fatal(err)
	}
	if v := pt2.SettingsOf(roll).Values(); v["length"] != 4 || v["n/0/key"] != 60 || v["n/0/velocity"] != 0.9 {
		t.Errorf("the notes should come back with the patch: %v", v)
	}
}

// a synth's modulation matrix is among its settings: a slot joins an Ext source to its level, the
// source is a setting too (a mod cable of the patch moves it)
func TestSynthMatrixInPatch(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	peak := func(ext float64, cable bool) float32 {
		p := psalm.New(nil)
		pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
		p.Out(pt.Out())
		roll, _ := pt.Add("noteroll", "", 0, 0)
		synth, _ := pt.Add("subtractive", "", 0, 0)
		for _, c := range []patcher.Cable{{From: roll, To: synth}, {From: synth, To: "out"}} {
			if _, err := pt.Connect(c); err != nil {
				t.Fatal(err)
			}
		}
		pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 2, "n/0/key": 60, "n/0/velocity": 0.9})
		level := 0
		for i, d := range subtractive.Dests {
			if d.Key == "level" {
				level = i + 1
			}
		}
		pt.SettingsOf(synth).Apply(map[string]float64{"mod1Src": float64(modmatrix.Ext1), "mod1Dest": float64(level), "mod1Amount": -1, "ext1": ext, "level": 1})
		if cable { // an LFO cabled to the source: the page can move it like any setting
			lfo, _ := pt.Add("lfo", "", 0, 0)
			if _, err := pt.Connect(patcher.Cable{From: lfo, To: synth, In: 1, Key: "ext1"}); err != nil {
				t.Fatal(err)
			}
		}
		var pk float32
		for i := range 44100 * 2 / 256 {
			p.ResetRun()
			p.Run()
			if i < 44100/10/256 { // the source glides to its value in the first milliseconds
				continue
			}
			for _, s := range pt.Out().Output(0).Channels[0] {
				pk = max(pk, s, -s)
			}
		}
		return pk
	}
	if open, shut := peak(0, false), peak(1, false); open < 0.01 || shut > open*0.05 {
		t.Errorf("peak %v with the source at 0, %v at 1: the slot should silence the synth", open, shut)
	}
	if peak(0, true) < 0.001 {
		t.Error("the synth with a mod cable to ext1 doesn't play")
	}
}
