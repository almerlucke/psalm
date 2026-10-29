package main

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// tap keeps the samples the notes sent to it fall on
type tap struct {
	*psalm.Module
	at []int64
}

func (t *tap) ReceiveMessage(m psalm.Message) {
	if m["command"] == "trigger" {
		t.at = append(t.at, m.MustTickVal("tick").SampleTimestamp)
	}
}
func (t *tap) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		t.ReceiveMessage(m)
	}
}

// the piece builds, has no text that can't be read, sounds in every section without clipping, and does not repeat: the bars of the rhythms are not the same bar again, the melodist
// makes a new pass every time through, and the chord model draws new chords when the song leaves a section
func TestPieceNeverRepeats(t *testing.T) {
	tempo := 300.0 // the fastest the patch goes
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(5)
	p := psalm.New(nil)
	taps := map[string]*tap{}
	kinds := catalog.Kinds(catalog.Config{Input: midi.NewQueue()})
	for _, n := range []string{"drums kick", "drums hats", "glitch ticks"} {
		tp := &tap{Module: psalm.NewControl()}
		tp.SetSelf(tp)
		taps[n] = tp
		kinds = append(kinds, patcher.Kind{Name: "tap " + n, Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}}, Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: tp}, nil }})
	}
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	ids, err := buildPatch(pt, 5)
	if err != nil {
		t.Fatal(err)
	}
	for n, tp := range taps {
		id, _ := pt.Add("tap "+n, "", 0, 0)
		if _, err := pt.Connect(patcher.Cable{From: ids.notes[n], To: id}); err != nil {
			t.Fatal(err)
		}
		_ = tp
	}
	pt.Tempo().Set(tempo)
	p.Out(pt.Out())
	for _, n := range pt.Nodes() {
		for _, tx := range pt.Texts(n.ID) {
			if e := pt.TextError(n.ID, tx.Key); e != "" {
				t.Errorf("%s %s: %s", n.Kind, tx.Key, e)
			}
		}
	}
	bar := int(4 * 60 / tempo * 44100 / 256) // buffers in a bar
	var peak float64
	for _, s := range sections {
		var sectionPeak, energy float64
		for range s.bars * bar {
			p.ResetRun()
			p.Run()
			for _, v := range pt.Out().Output(0).Channels[0] {
				a := math.Abs(float64(v))
				sectionPeak = math.Max(sectionPeak, a)
				energy += float64(v) * float64(v)
			}
		}
		if sectionPeak < 0.005 || math.IsNaN(energy) {
			t.Errorf("section %s: peak %v", s.name, sectionPeak)
		}
		peak = math.Max(peak, sectionPeak)
	}
	if peak > 1.0 {
		t.Errorf("the piece peaks at %.2f: it clips", peak)
	}
	t.Logf("peak %.2f", peak)

	// the bars of a rhythm in a section, by the steps they have hits on: not one bar again and again
	barSamples := 4 * 60 / tempo * 44100
	section := func(name string) (from, to float64) {
		bars := 0
		for _, s := range sections {
			if s.name == name {
				return float64(bars) * barSamples, float64(bars+s.bars) * barSamples
			}
			bars += s.bars
		}
		return 0, 0
	}
	distinct := func(tp *tap, name string) int {
		from, to := section(name)
		seen := map[string]bool{}
		for b := from; b < to; b += barSamples {
			key := ""
			for _, at := range tp.at {
				if float64(at) >= b && float64(at) < b+barSamples {
					key += string(rune('a' + int(math.Round((float64(at)-b)/barSamples*32))))
				}
			}
			seen[key] = true
		}
		return len(seen)
	}
	for _, c := range []struct {
		tap, section string
		min          int
	}{{"drums kick", "Fracture", 8}, {"drums kick", "Machine", 10}, {"drums hats", "Fracture", 8}, {"drums hats", "Machine", 8}, {"glitch ticks", "Machine", 15}} {
		if got := distinct(taps[c.tap], c.section); got < c.min {
			t.Errorf("%s in the %s: only %d different bars", c.tap, c.section, got)
		}
	}
	if n := pt.Module(ids.melodist).(*patcher.Melodist).Passes(); n < 3 {
		t.Errorf("the melodist made %d passes", n)
	}
	// the chords of the sections that have none of their own were drawn again as the song left the sections: the harmony has another line than it began with
	if got := pt.Texts(ids.harmony)[0].Value; got == harmonyText {
		t.Error("the chord model wrote nothing to the harmony")
	}
}
