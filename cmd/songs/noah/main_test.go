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

// the piece plays: every bar of every section (at a fast tempo), without a script saying it went wrong and with sound. The recording is not in the repository (it is in
// local/), so without it the test is skipped
func TestPiecePlays(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 96 bars")
	}
	toRoot(t)
	if _, err := os.Stat(soundPath); err != nil {
		t.Skip("no recording:", err)
	}
	tempo = 240
	defer func() { tempo = 60 }()
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 48000, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(3)
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	ids, err := buildPatch(pt, true, 3)
	if err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	bar := int(4 * 60 / tempo * 48000 / 256) // buffers in a bar
	for i, s := range sections {
		var peak, energy float64
		for range s.bars * bar {
			p.ResetRun()
			p.Run()
			for _, v := range pt.Out().Output(0).Channels[0] {
				peak = math.Max(peak, math.Abs(float64(v)))
				energy += float64(v) * float64(v)
			}
		}
		if peak < 0.001 || peak > 1.5 || math.IsNaN(energy) {
			t.Errorf("section %d (%s): peak %v", i, s.name, peak)
		}
		for _, id := range ids.scripts {
			if st := pt.Status(id); strings.HasPrefix(st, "error") {
				t.Fatalf("section %s, script %s: %s", s.name, id, st)
			}
		}
	}
}

// the sections are as long as the recording, 6:24
func TestLengthIsTheRecording(t *testing.T) {
	if got := tourSeconds(); got != 384 {
		t.Errorf("%v seconds, want 384", got)
	}
}

// toRoot goes to the root of the repository, where local/ is
func toRoot(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("go.mod"); err == nil {
		return
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
}
