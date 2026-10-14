package patcher_test

import (
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// the log: what is sent to it, shown on the module, and sent on
func TestLogModule(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	clock, _ := pt.Add("clock", "", 0, 0)
	seq, _ := pt.Add("sequence", "", 0, 0)
	trigLog, _ := pt.Add("log", "", 0, 0)
	valLog, _ := pt.Add("log", "", 0, 0)
	_ = pt.SetText(seq, "values", "60 [48 52]")
	_ = pt.SetText(valLog, "label", "seq")
	for _, c := range []patcher.Cable{{From: clock, To: seq}, {From: clock, To: trigLog, In: 1}, {From: seq, To: valLog, In: 0}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	for range 44100 / 256 {
		p.ResetRun()
		p.Run()
	}
	if s := pt.Status(trigLog); !strings.Contains(s, "trigger step") {
		t.Errorf("the log should show the triggers: %q", s)
	}
	s := pt.Status(valLog)
	if !strings.HasPrefix(s, "seq  ") || !strings.Contains(s, "  60") || !strings.Contains(s, "[48 52]") {
		t.Errorf("the log should show the values with its label: %q", s)
	}
	if n := strings.Count(s, "\n") + 1; n > 12 {
		t.Errorf("no more than 12 lines: %d", n)
	}
}
