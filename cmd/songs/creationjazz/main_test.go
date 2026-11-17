package main

import (
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// the piece plays: every bar of every day (at a fast tempo), without a script saying it went wrong, and with sound; the cables of the recordings reach their
// levels and the void's reaches its cutoff; every day writes its verse to the log of the patch
func TestPatchPlays(t *testing.T) { playsPatch(t, false) }

// the same with the instruments of the orchestra where it is
func TestPatchPlaysWithOrchestra(t *testing.T) {
	if _, err := os.Stat("../../../" + orchestra); err != nil {
		t.Skip("the Virtual Playing Orchestra is not in assets")
	}
	playsPatch(t, true)
}

func playsPatch(t *testing.T, real bool) {
	if testing.Short() {
		t.Skip("plays 156 bars")
	}
	// the sounds are found from the root of the repository, where the song is run from
	t.Chdir("../../..")
	tempo = 300 // the fastest the patch goes
	defer func() { tempo = bpm }()
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(3)
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	ids, err := buildPatch(pt, options{just: true, a4: 440, seed: 1, human: true, real: real, once: true})
	if err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())

	for _, name := range []string{"animals", "gulls", "sea and birds", "children", "bells", "marimba", "kalimba", "upright bass", "piano", "vibes", "flute", "steel pan", "darbuka", "congas", "bongos", "claves", "shakers"} {
		if _, ok := ids.synths[name]; !ok {
			t.Errorf("no %s in the combo", name)
		}
	}
	var keys []string
	for _, c := range pt.Cables() {
		if c.Type == patcher.Mod {
			keys = append(keys, c.Key)
		}
	}
	slices.Sort(keys)
	want := []string{"cutoff"}
	for _, name := range []string{"waves"} {
		want = append(want, "level"+strconv.Itoa(ids.index[name]+1))
	}
	slices.Sort(want)
	if !slices.Equal(keys, want) {
		t.Errorf("mod cables to %v, want %v", keys, want)
	}

	mixer := pt.Module(ids.mixer).(*patcher.Mixer) // what the cables of the scripts moved
	cutoff := func() float64 { return pt.SettingsOf(ids.synths["void"]).Values()["cutoff"] }
	bar := int(beats * 60 / tempo * 44100 / 256) // buffers in a bar
	for _, d := range days {
		var peak, energy float64
		for range d.bars * bar {
			p.ResetRun()
			p.Run()
			for _, v := range pt.Out().Output(0).Channels[0] {
				peak = max(peak, math.Abs(float64(v)))
				energy += float64(v) * float64(v)
			}
		}
		if peak < 0.01 || peak > 1.5 || math.IsNaN(energy) {
			t.Errorf("%s: peak %v", d.name, peak)
		}
		switch d.name {
		case days[0].name: // the darkness has lifted: the cutoff is the drone's own; the waters are at -18 dB
			if c := cutoff(); math.Abs(c-1500) > 100 {
				t.Errorf("after the intro the cutoff is %v, want about 1500", c)
			}
			if l := mixer.Level[ids.index["waves"]]; math.Abs(l-math.Pow(10, -18.0/20)) > 0.05 {
				t.Errorf("after the intro the waves are at %v, want about %v", l, math.Pow(10, -18.0/20))
			}
		}
		for _, id := range ids.scripts {
			if st := pt.Status(id); strings.HasPrefix(st, "error") {
				t.Fatalf("%s, script %s: %s", d.name, id, st)
			}
		}
	}
	log := pt.Journal().Text()
	for _, d := range days {
		if !strings.Contains(log, d.name) {
			t.Errorf("%s is not in the log", d.name)
		}
	}
	if !strings.Contains(log, "And God said, Let there be light") || !strings.Contains(log, "having made peace through the blood of his cross") {
		t.Errorf("the verses are not in the log: %q", log)
	}
}
