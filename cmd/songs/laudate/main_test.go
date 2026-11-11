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

// the piece plays: every bar of every section (at a fast tempo), without a script saying it went wrong, and with sound
func TestTourPlays(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 66 bars")
	}
	// the sounds are found from the root of the repository, where the song is run from
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	tempo = 300 // the fastest the patch goes
	defer func() { tempo = nominalTempo }()
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
	bar := int(4 * 60 / tempo * 44100 / 256) // buffers in a bar
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
	// each section writes its verse of the psalm to the log of the patch
	if log := pt.Journal().Text(); !strings.Contains(log, "Psalm 150:1 Praise ye the LORD") || !strings.Contains(log, "Psalm 150:5") || !strings.Contains(log, "hath breath") {
		t.Errorf("the verses are not in the log: %q", log)
	}
}
