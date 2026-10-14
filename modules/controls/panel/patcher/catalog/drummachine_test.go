package catalog_test

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
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
