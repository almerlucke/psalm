package main

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/noteroll"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// the patch builds without a text that cannot be read, has a melody and a counter-line for the verse and the chorus, and sounds in every section
func TestEverythingSounds(t *testing.T) {
	tempo := 240.0
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(3)
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	ids, err := buildPatch(pt)
	if err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	for _, n := range pt.Nodes() {
		for _, tx := range pt.Texts(n.ID) {
			if e := pt.TextError(n.ID, tx.Key); e != "" {
				t.Errorf("%s %s: %s", n.Kind, tx.Key, e)
			}
		}
	}
	for _, id := range []string{ids.roll, ids.counter} {
		r := pt.Module(id).(*noteroll.Roll)
		for _, s := range []string{"Verse", "Chorus"} {
			if c, ok := r.Clip(s); !ok || len(c.Notes) < 4 {
				t.Errorf("%s: the clip of the %s has %d notes", id, s, len(c.Notes))
			}
		}
	}
	if len(ids.drums) != 5 {
		t.Errorf("%d drum clocks", len(ids.drums))
	}
	bar := int(4.0 * 60 / tempo * 44100 / 256)
	for _, s := range sections {
		var peak float64
		for range s.bars * bar {
			p.ResetRun()
			p.Run()
			for _, v := range pt.Out().Output(0).Channels[0] {
				peak = math.Max(peak, math.Abs(float64(v)))
			}
		}
		if peak < 0.01 || peak > 2 || math.IsNaN(peak) {
			t.Errorf("section %s: peak %v", s.name, peak)
		}
	}
}
