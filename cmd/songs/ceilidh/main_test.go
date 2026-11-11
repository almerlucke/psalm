package main

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// the piece plays: every bar of every section (at a fast tempo), without a script saying it went wrong
// (a chain a script sets that can't be read is an error of the script, so is a chord) and with sound
func TestPiecePlays(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 144 bars")
	}
	tempo = 300 // the fastest the patch goes: the 144 bars take about two minutes
	defer func() { tempo = 150 }()
	toRoot(t)
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(3)
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	ids, err := buildPatch(pt, true)
	if err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	bar := int(3 * 60 / tempo * 44100 / 256) // buffers in a bar (three beats)
	for i, s := range sections {
		var peak, energy float64
		for range s.bars * bar {
			p.ResetRun()
			p.Run()
			for _, v := range pt.Out().Output(0).Channels[0] {
				peak = max(peak, math.Abs(float64(v)))
				energy += float64(v) * float64(v)
			}
		}
		if peak < 0.02 || peak > 1.5 || math.IsNaN(energy) {
			t.Errorf("section %d (%s): peak %v", i, s.name, peak)
		}
		for _, id := range ids.scripts {
			if st := pt.Status(id); strings.HasPrefix(st, "error") {
				t.Fatalf("section %s, script %s: %s", s.name, id, st)
			}
		}
	}
}

// played live the patch also has the microphone, with a recorder (and the choice of the device); it builds
func TestLivePatchHasMicrophone(t *testing.T) {
	toRoot(t)
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Audio: p.PatchInput(0)}))
	if _, err := buildPatch(pt, false); err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, n := range pt.Nodes() {
		kinds[n.Kind] = true
	}
	if !kinds["input"] || !kinds["recorder"] {
		t.Errorf("kinds in the patch: %v", kinds)
	}
}

// toRoot goes to the root of the repository, where the files of the piece are (once: the tests share the directory)
func toRoot(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(jigs[0]); err == nil {
		return
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
}
