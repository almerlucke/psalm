package patcher_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// a cable lights when a note goes through it
func TestNotesCableLights(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	q := midi.NewQueue()
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: q}))
	p.Out(pt.Out())
	k, _ := pt.Add("keyboard", "", 0, 0)
	s, _ := pt.Add("subtractive", "", 0, 0)
	if _, err := pt.Connect(patcher.Cable{From: k, To: s}); err != nil {
		t.Fatal(err)
	}
	run := func() { p.ResetRun(); p.Run() }
	run()
	if st := patcher.StateJSON(pt); !strings.Contains(st, `"lit":[]`) {
		t.Fatalf("lit before a note: %s", st)
	}
	q.Push(midi.NoteOnMsg(1, 60, 100))
	run()
	run()
	if st := patcher.StateJSON(pt); strings.Contains(st, `"lit":[]`) {
		t.Fatalf("not lit after a note: %s", st)
	}
}

// the cables of a midiplayer's channels light too
func TestMidiPlayerCableLights(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	f := &midi.File{Format: 1, Division: 480, Tracks: []midi.Track{{}}}
	f.Tracks[0].Add(480, midi.NoteOnMsg(1, 60, 100))
	f.Tracks[0].Add(960, midi.NoteOffMsg(1, 60, 0))
	path := filepath.Join(t.TempDir(), "t.mid")
	if err := f.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	pl, _ := pt.Add("midiplayer", "", 0, 0)
	s, _ := pt.Add("subtractive", "", 0, 0)
	if _, err := pt.Connect(patcher.Cable{From: pl, Out: 1, To: s}); err != nil {
		t.Fatal(err)
	}
	pt.SetText(pl, "file", path)
	pt.SettingsOf(pl).Apply(map[string]float64{"play": 1})
	seen := false
	for range 200 {
		p.ResetRun()
		p.Run()
		if st := patcher.StateJSON(pt); !strings.Contains(st, `"lit":[]`) {
			seen = true
			break
		}
	}
	if !seen {
		t.Error("the cable of the midiplayer never lights")
	}
}
