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
	"github.com/almerlucke/psalm/modules/effects/looper"
)

// toRoot goes to the root of the repository, where the file of the piece is (once: the tests share the directory)
func toRoot(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(midiFile); err == nil {
		return
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
}

// the score has every note of the file: the score holds the notes of the pedal (the pulse follows it): as many as the file has
func TestScoreHasAllNotes(t *testing.T) {
	toRoot(t)
	text, err := score(midiFile)
	if err != nil {
		t.Fatal(err)
	}
	f, err := midi.ReadFile(midiFile)
	if err != nil {
		t.Fatal(err)
	}
	notes := 0
	for _, n := range f.Notes() {
		if roles[n.Channel] == "pedal" {
			notes++
		}
	}
	if notes == 0 {
		t.Fatalf("%d notes in the file for the roles", notes)
	}
	if got := strings.Count(text, "],["); got < notes-1 { // the notes are arrays in lists: a "],[" between two notes
		t.Errorf("the score has about %d notes, the file %d", got, notes)
	}
	if !strings.Contains(text, "const SECTIONS = ") {
		t.Error("no sections in the score")
	}
}

// the piece plays: every bar of every section (at a fast tempo), without a script saying it went wrong and with sound
func TestPiecePlays(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 49 bars")
	}
	t.Run("synths", func(t *testing.T) { playPiece(t, false) })
	t.Run("orchestra", func(t *testing.T) {
		toRoot(t)
		if !hasOrchestra() {
			t.Skip("the orchestra is not in assets")
		}
		playPiece(t, true)
	})
}

func playPiece(t *testing.T, real bool) {
	tempo = 240 // the fastest the patch goes
	defer func() { tempo = 66 }()
	toRoot(t)
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(3)
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	ids, err := buildPatch(pt, true, real)
	if err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	limit := 1.5 // recordings have sharper peaks than the synths (the master limits them)
	if real {
		limit = 2
	}
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
		if peak < 0.02 || peak > limit || math.IsNaN(energy) {
			t.Errorf("section %d (%s): peak %v", i, s.name, peak)
		}
		for _, id := range ids.scripts {
			if st := pt.Status(id); strings.HasPrefix(st, "error") {
				t.Fatalf("section %s, script %s: %s", s.name, id, st)
			}
		}
	}
}

// the live patch has a looper to record on top of the piece: a source mixer (the piece, the sound input), the looper and its cable to the output
func TestLivePatchHasLooper(t *testing.T) {
	toRoot(t)
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(3)
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Audio: p.PatchInput(0)}))
	ids, err := buildPatch(pt, false, false)
	if err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	// the piece as the source, passes recorded: layers
	pt.SettingsOf(ids.sources).Apply(map[string]float64{"level1": 1, "level2": 0})
	pt.SettingsOf(ids.looper).Apply(map[string]float64{"bars": 1, "record": 1})
	loop := pt.Module(ids.looper).(*looper.Looper)
	for range int(5 * 44100 / 256) { // a bar of 4/4 at 66 bpm is 3.6 seconds
		p.ResetRun()
		p.Run()
	}
	pt.SettingsOf(ids.looper).Apply(map[string]float64{"record": 0})
	for range int(8 * 44100 / 256) {
		p.ResetRun()
		p.Run()
	}
	if n, _ := loop.Layers(); n < 1 {
		t.Errorf("%d layers recorded, want at least 1", n)
	}
}
