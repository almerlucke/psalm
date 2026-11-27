package main

import (
	"math"
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
func TestTourPlays(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 88 bars")
	}
	tempo = 300 // the fastest the patch goes: the 88 bars take about 70 s
	defer func() { tempo = 60 }()
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
}
