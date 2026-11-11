package catalog_test

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

// the streamer's sounds are a text in its side panel; a text that can't be read leaves the sounds as they were, and a note plays the file from the disk
func TestStreamerSounds(t *testing.T) {
	if _, err := os.Stat(tabla); err != nil {
		t.Skip("no tabla loop in assets")
	}
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	roll, _ := pt.Add("noteroll", "", 0, 0)
	syn, err := pt.Add("streamer", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: roll, To: syn}, {From: syn, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 1, "n/0/key": 36, "n/0/velocity": 0.9})
	if err := pt.SetText(syn, "sounds", "36 "+tabla+" name=tabla"); err != nil {
		t.Fatal(err)
	}
	if st := pt.Status(syn); !strings.Contains(st, "from disk") || !strings.Contains(st, "tabla") {
		t.Errorf("status %q", st)
	}
	_ = pt.SetText(syn, "sounds", "36 nothere.wav")
	if pt.TextError(syn, "sounds") == "" || !strings.HasPrefix(pt.Status(syn), "error") {
		t.Errorf("a missing file is no error: %q", pt.Status(syn))
	}
	var peak float64
	for range 260 {
		p.ResetRun()
		p.Run()
		for _, s := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(s)))
		}
	}
	if peak < 0.05 {
		t.Errorf("the streamer plays the file: peak %g", peak)
	}
}
