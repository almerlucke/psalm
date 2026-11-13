package catalog_test

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// the MIDI player plays the notes of a file to the synth it is cabled to once play is set (in either order of file and setting), a file typed in swaps
// while it plays, and its controller outputs can be cabled
func TestMidiPlayer(t *testing.T) {
	// the file has channel 1 only: the output of channel 1 and the output of all play it, the output of channel 2 plays nothing
	for out, plays := range map[int]bool{0: true, 1: true, 2: false} {
		if peak := midiPlayerPeak(t, out); (peak > 0.01) != plays {
			t.Errorf("output %d: peak %v, plays %v", out, peak, plays)
		}
	}
}

func midiPlayerPeak(t *testing.T, out int) float64 {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	f := &midi.File{Format: 1, Division: 480, Tracks: []midi.Track{{}}}
	f.Tracks[0].Add(0, midi.ControlChangeMsg(1, 74, 100))
	f.Tracks[0].Add(0, midi.NoteOnMsg(1, 60, 100))
	f.Tracks[0].Add(480, midi.NoteOffMsg(1, 60, 0))
	f.Tracks[0].Add(960, midi.NoteOnMsg(1, 64, 100))
	f.Tracks[0].Add(1440, midi.NoteOffMsg(1, 64, 0))
	path := filepath.Join(t.TempDir(), "t.mid")
	if err := f.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	p := psalm.New(nil)
	psalm.Seed(1)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	pl, err := pt.Add("midiplayer", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	syn, err := pt.Add("subtractive", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: pl, Out: out, To: syn}, {From: syn, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pt.Connect(patcher.Cable{Type: patcher.Mod, From: pl, Out: 23, To: syn, In: 1, Key: "cutoff", Lo: 0, Hi: 1}); err != nil {
		t.Fatalf("a controller output can't be cabled: %v", err)
	}
	pt.SettingsOf(pl).Apply(map[string]float64{"play": 1}) // before the file
	pt.SetText(pl, "file", "/no/such/file.mid")
	if pt.TextError(pl, "file") == "" {
		t.Error("a missing file is not an error")
	}
	pt.SetText(pl, "controllers", "1 2 3 4 5 300")
	if pt.TextError(pl, "controllers") == "" {
		t.Error("a controller of 300 is accepted")
	}
	pt.SetText(pl, "file", path)
	if e := pt.TextError(pl, "file"); e != "" {
		t.Fatal(e)
	}
	var peak float64
	for range 258 {
		p.ResetRun()
		p.Run()
		for _, v := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(v)))
		}
	}
	return peak
}
