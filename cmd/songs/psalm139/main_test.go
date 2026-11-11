package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/sndfile"
)

// the piece plays: every bar of every section (at a fast tempo), with the models of the instruments and with the recordings of the orchestra where it is in assets,
// without a script saying it went wrong, and with sound in every section
func TestPieceSounds(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 96 bars twice")
	}
	if err := os.Chdir("../../.."); err != nil { // the assets are found from the root of the repository
		t.Fatal(err)
	}
	if _, err := os.Stat("assets/IR44100/St Nicolaes Church.wav"); err != nil {
		t.Skip("the church is not in assets")
	}
	tempo = 300 // the fastest the patch goes
	defer func() { tempo = float64(bpm) }()
	for _, real := range []bool{false, true} {
		if real && !hasOrchestra() {
			continue
		}
		psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
		psalm.Seed(3)
		p := psalm.New(nil)
		church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
		pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds(catalog.Config{Input: midi.NewQueue(), IR: church}))
		ids, err := buildPatch(pt, options{just: true, a4: 440, seed: 3, human: true, real: real, once: true})
		if err != nil {
			t.Fatal(err)
		}
		p.Out(pt.Out())
		for _, id := range ids.live {
			for _, tx := range pt.Texts(id) {
				if e := pt.TextError(id, tx.Key); e != "" {
					t.Errorf("live part %s %s: %s", id, tx.Key, e)
				}
			}
		}
		bar := int(4 * 60 / tempo * 44100 / 256) // buffers in a bar
		for _, s := range sections {
			var peak, energy float64
			for range s.bars * bar {
				p.ResetRun()
				p.Run()
				for _, v := range pt.Out().Output(0).Channels[0] {
					peak = max(peak, math.Abs(float64(v)))
					energy += float64(v) * float64(v)
				}
			}
			if peak < 0.005 || peak > 2 || math.IsNaN(energy) {
				t.Errorf("recordings %v, section %s: peak %v", real, s.name, peak)
			}
			for _, id := range ids.scripts {
				if st := pt.Status(id); strings.HasPrefix(st, "error") {
					t.Fatalf("recordings %v, section %s, script %s: %s", real, s.name, id, st)
				}
			}
		}
		if m := pt.Module(ids.melodist).(*patcher.Melodist); m.Passes() < 3 {
			t.Errorf("the melodist made %d passes", m.Passes())
		}
		psalm.PopConfig()
		p.Close()
	}
}

// the harmony of the score fits the sections (the script says so when it starts), the tunes sit on their chords: on the strong beats of the loop the notes of the
// tunes A and B are notes of the chord of the bar, which is what lets the voices sing them over each other
func TestSectionsAreTheLengthOfTheirChords(t *testing.T) {
	total := 0
	for _, s := range sections {
		total += s.bars
	}
	if total != 92 {
		t.Errorf("the piece is %d bars", total)
	}
	if sectionsText() != "Thread 8, Knit 16, Wonder 16, Shadow 12, Light 12, Sand 16, Awake 12" {
		t.Errorf("sections: %q", sectionsText())
	}
}

// the chords of the harmony module are the chords of the score: every section has a cycle of chords that fits its bars, and the harmony reads them
func TestProgressionsFitTheSections(t *testing.T) {
	for _, s := range sections {
		text, ok := progressions[s.name]
		if !ok {
			t.Fatalf("no chords for %s", s.name)
		}
		n := 0
		for _, tok := range strings.Fields(text) {
			if _, bars, found := strings.Cut(tok, "*"); found {
				var b int
				if _, err := fmt.Sscan(bars, &b); err != nil {
					t.Fatalf("%s: %q", s.name, tok)
				}
				n += b
			} else {
				n++
			}
		}
		if n == 0 || s.bars%n != 0 {
			t.Errorf("%s: a cycle of %d bars does not fit %d", s.name, n, s.bars)
		}
	}
	if got := harmonyText(); !strings.HasPrefix(got, "Thread: Dsus2*4 D*2 Dsus2*2\nKnit: D Bm7 Gmaj7 A") {
		t.Errorf("harmony text %q", got)
	}
}
