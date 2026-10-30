package main

import (
	"encoding/json"
	"math"
	"os"
	"slices"
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

// the piece builds, has no text that can't be read, sounds in every section without clipping, and does not repeat: the bars of the rhythms are not the same bar again,
// and the chord model draws new chords when the song leaves a section
func TestPieceNeverRepeats(t *testing.T) {
	if err := os.Chdir("../../.."); err != nil { // the recordings are found from the root of the repository
		t.Fatal(err)
	}
	tempo := 300.0 // the fastest the patch goes
	psalm.PushConfig(psalm.Config{BPM: tempo, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	psalm.Seed(5)
	p := psalm.New(nil)
	taps := map[string]*tap{}
	kinds := catalog.Kinds(catalog.Config{Input: midi.NewQueue()})
	for _, n := range []string{"drums kick", "drums hats", "glitch ticks", "chops part", "slices strokes", "modem chirps", "groove kick", "groove hats", "bass part", "metal hits", "timp part"} {
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
		t.Logf("%-9s peak %.2f, level %.1f dB", s.name, sectionPeak, 10*math.Log10(energy/float64(s.bars*bar*256)+1e-12))
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
	}{{"drums kick", "Fracture", 8}, {"drums kick", "Ridge", 10}, {"drums hats", "Fracture", 8}, {"drums hats", "Ridge", 8}, {"glitch ticks", "Ridge", 15}, {"chops part", "Ridge", 10}, {"slices strokes", "Ridge", 10}} {
		if got := distinct(taps[c.tap], c.section); got < c.min {
			t.Errorf("%s in the %s: only %d different bars", c.tap, c.section, got)
		}
	}
	// the groove builds: more hits a bar as the sections go (the Static to the Fracture to the Machine)
	perBar := func(tp *tap, name string) float64 {
		from, to := section(name)
		c := 0
		for _, at := range tp.at {
			if float64(at) >= from && float64(at) < to {
				c++
			}
		}
		for _, sec := range sections {
			if sec.name == name {
				return float64(c) / float64(sec.bars)
			}
		}
		return 0
	}
	for _, n := range []string{"groove kick", "groove hats"} {
		a, b, c := perBar(taps[n], "Static"), perBar(taps[n], "Fracture"), perBar(taps[n], "Ridge")
		t.Logf("%s: %.1f hits a bar in the Static, %.1f in the Fracture, %.1f in the Ridge", n, a, b, c)
		if !(a <= b) || b < 0.5 { // the groove comes in with the Fracture
			t.Errorf("%s does not come in: %.2f, %.2f, %.2f hits a bar", n, a, b, c)
		}
	}
	// the Bare is the Rift without the tune (its drums and bass play), the Collapse has no drums and no bass either
	if perBar(taps["bass part"], "Bare") < 1 || perBar(taps["drums hats"], "Bare") < 1 {
		t.Error("the Bare has no bass or no drums")
	}
	for _, n := range []string{"drums kick", "drums hats", "groove kick", "groove hats", "bass part", "metal hits", "timp part"} {
		if got := perBar(taps[n], "Collapse"); got != 0 {
			t.Errorf("%s plays %.2f hits a bar in the Collapse, which has no drums, no bass, no anvil and no timpani", n, got)
		}
	}
	// the Skeleton, after the Bare, is the drums, a rim, the bass and the glitch and the metal that are left: not the tune, the groove, the synth, the samplers and the timpani
	for _, n := range []string{"drums kick", "drums hats", "bass part", "glitch ticks"} {
		if perBar(taps[n], "Skeleton") <= 0 {
			t.Errorf("%s does not play in the Skeleton", n)
		}
	}
	for _, n := range []string{"timp part", "groove kick", "groove hats", "chops part", "slices strokes", "modem chirps"} {
		if got := perBar(taps[n], "Skeleton"); got != 0 {
			t.Errorf("%s plays %.2f hits a bar in the Skeleton, which keeps the drums, the bass, the glitch and the metal", n, got)
		}
	}
	for i, id := range ids.toybox {
		if n := pt.Module(id).(*patcher.Melodist).Passes(); n < 2 {
			t.Errorf("the toy box layer %d made %d passes", i, n)
		}
		var v struct {
			Plays []string `json:"plays"`
		}
		raw, _ := json.Marshal(pt.Module(id).(*patcher.Melodist).View())
		_ = json.Unmarshal(raw, &v)
		if len(v.Plays) == 0 || slices.Contains(v.Plays, "Collapse") || slices.Contains(v.Plays, "Bare") || slices.Contains(v.Plays, "Skeleton") {
			t.Errorf("the toy box layer %d plays in %v: not in the Bare, the Skeleton and the Collapse", i, v.Plays)
		}

	}
	// the chords of the sections that have none of their own were drawn again as the song left the sections: the harmony has another line than it began with
	got := ""
	for _, x := range pt.Texts(ids.harmony) {
		if x.Key == "chords" {
			got = x.Value
		}
	}
	if got == harmonyText {
		t.Error("the chord model wrote nothing to the harmony")
	}
}
