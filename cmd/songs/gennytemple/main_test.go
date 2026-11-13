package main

import (
	"math"
	"os"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// toRoot goes to the root of the repository, where the file of the piece is
func toRoot(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(midiFile); err == nil {
		return
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
}

// every channel of the file that has notes is played by a part, and no part plays a channel the file has not
func TestPartsPlayTheChannelsOfTheFile(t *testing.T) {
	toRoot(t)
	f, err := midi.ReadFile(midiFile)
	if err != nil {
		t.Fatal(err)
	}
	in := map[int]bool{}
	for _, n := range f.Notes() {
		in[n.Channel] = true
	}
	played := map[int]bool{}
	for _, p := range parts {
		for _, ch := range p.channels {
			if !in[ch] {
				t.Errorf("%s plays channel %d, which has no notes", p.name, ch)
			}
			played[ch] = true
		}
	}
	for ch := range in {
		if !played[ch] {
			t.Errorf("channel %d has notes and no part", ch)
		}
	}
}

// the piece plays, the first minute at least (it plays as it is: a faster tempo would change the file's time): it sounds and does not clip
func TestPiecePlays(t *testing.T) {
	if testing.Short() {
		t.Skip("plays a minute")
	}
	toRoot(t)
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(3)
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	if _, err := buildPatch(pt); err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	var peak, energy float64
	for range int(60 * 44100 / 256) {
		p.ResetRun()
		p.Run()
		for _, v := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(v)))
			energy += float64(v) * float64(v)
		}
	}
	if peak < 0.02 || peak > 1.5 || math.IsNaN(energy) {
		t.Errorf("peak %v", peak)
	}
}
