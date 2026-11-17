package patcher_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// a recorder keeps a file in the folder recordings when the record switch goes off, also one that only taps a voice (nothing is cabled on from it)
func TestRecorderKeepsFile(t *testing.T) {
	t.Chdir(t.TempDir())
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	s, _ := pt.Add("subtractive", "", 0, 0)
	r, err := pt.Add("recorder", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: s, To: r}, {From: s, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	run := func(n int) {
		for range n {
			p.ResetRun()
			p.Run()
		}
	}
	pt.SettingsOf(r).Apply(map[string]float64{"record": 1})
	run(50)
	pt.SettingsOf(r).Apply(map[string]float64{"record": 0})
	run(5)
	time.Sleep(300 * time.Millisecond)
	files, _ := filepath.Glob("recordings/*.wav")
	if len(files) != 1 {
		ents, _ := os.ReadDir(".")
		t.Fatalf("files %v (%v)", files, ents)
	}
	if fi, _ := os.Stat(files[0]); fi.Size() < 1000 {
		t.Errorf("file of %d bytes", fi.Size())
	}
}
