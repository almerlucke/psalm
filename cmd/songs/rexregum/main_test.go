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

// the score has every note of the file: the parts of the roles hold as many notes as the channels of the file they are made from
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
		if _, ok := roles[n.Channel]; ok {
			notes++
		}
	}
	if notes == 0 {
		t.Fatalf("%d notes in the file for the roles", notes)
	}
	if got := strings.Count(text, "],["); got < notes-len(roles) { // the notes are arrays in lists: a "],[" between two notes
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
