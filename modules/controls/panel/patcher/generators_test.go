package patcher_test

import (
	"math"
	"slices"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// played keeps the notes a synth gets: their pitch and the sample they start on
type played struct {
	*psalm.Module
	pitches []float64
	at      []int64
}

func (r *played) ReceiveMessage(m psalm.Message) {
	if m["command"] != "trigger" {
		return
	}
	tick := m.MustTickVal("tick")
	r.pitches = append(r.pitches, m.MustMsg("message").MustFltVal("pitch"))
	r.at = append(r.at, tick.SampleTimestamp)
}

func (r *played) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		r.ReceiveMessage(m)
	}
}

// a clock of sixteenths at 120 bpm through a sequence, a quantizer and notes into a synth that
// records: the notes in order, quantized, on their steps
func TestGenerators(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	rec := &played{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	kinds := append(patcher.Generators(), patcher.Kind{Name: "rec", Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}},
		Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: rec}, nil }})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	must := func(id string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	clock := must(pt.Add("clock", "", 0, 0))
	seq := must(pt.Add("sequence", "", 0, 0))
	q := must(pt.Add("quantize", "", 0, 0))
	note := must(pt.Add("notes", "", 0, 0))
	syn := must(pt.Add("rec", "", 0, 0))
	if err := pt.SetText(seq, "values", "C3 E3 G3 B3"); err != nil { // E and B leave C minor: Eb, Bb
		t.Fatal(err)
	}
	must(pt.Connect(patcher.Cable{From: clock, To: seq}))               // trigger
	must(pt.Connect(patcher.Cable{From: seq, Out: 0, To: q}))           // value
	must(pt.Connect(patcher.Cable{From: q, To: note, In: 1}))           // pitch
	must(pt.Connect(patcher.Cable{From: seq, Out: 1, To: note, In: 0})) // the trigger after the value
	must(pt.Connect(patcher.Cable{From: note, To: syn}))                // notes

	for range int(2 * 44100 / 256) { // two seconds: 16 sixteenths
		p.ResetRun()
		p.Run()
	}
	if len(rec.pitches) < 15 || len(rec.pitches) > 17 {
		t.Fatalf("%d notes in two seconds of sixteenths", len(rec.pitches))
	}
	if want := []float64{48, 51, 55, 58, 48, 51}; !slices.Equal(rec.pitches[:6], want) {
		t.Errorf("pitches %v, want %v (quantized to C minor)", rec.pitches[:6], want)
	}
	for i := 1; i < len(rec.at); i++ {
		if d := rec.at[i] - rec.at[i-1]; math.Abs(float64(d)-5512.5) > 1 {
			t.Errorf("notes %d and %d are %d samples apart, a sixteenth is 5512.5", i-1, i, d)
			break
		}
	}

	// a pattern with rests, a chord
	rec.pitches, rec.at = nil, nil
	_ = pt.SetText(clock, "pattern", "x . x x")
	_ = pt.SetText(seq, "values", "[C3 Eb3 G3] Bb3")
	for range int(1 * 44100 / 256) { // eight sixteenths: six hits, every other one a chord
		p.ResetRun()
		p.Run()
	}
	hits := map[int64]int{}
	for _, at := range rec.at {
		hits[at]++
	}
	if len(hits) < 5 || len(hits) > 7 {
		t.Errorf("%d hits in eight steps of x . x x", len(hits))
	}
	chords := 0
	for _, n := range hits {
		if n == 3 {
			chords++
		}
	}
	if chords < 2 {
		t.Errorf("no chords played: %v", rec.pitches)
	}
	if err := pt.SetText(clock, "pattern", "x ? x"); err != nil {
		t.Fatal(err)
	}
	if e := pt.TextError(clock, "pattern"); e == "" {
		t.Errorf("a bad pattern isn't reported")
	}
}

// the texts of the modules are kept with the patch
func TestGeneratorsKept(t *testing.T) {
	p := psalm.New(nil)
	dir := t.TempDir()
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", patcher.Generators(), patcher.Keep(dir))
	clock, _ := pt.Add("clock", "", 0, 0)
	seq, _ := pt.Add("sequence", "", 0, 0)
	_ = pt.SetText(clock, "pattern", "x . . x")
	_ = pt.SetText(seq, "values", "60 [62 65] 67")
	if err := pt.Save("gen"); err != nil {
		t.Fatal(err)
	}
	pt.Clear()
	if err := pt.Load("gen"); err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, n := range pt.Nodes() {
		for _, tx := range pt.Texts(n.ID) {
			texts[n.Kind+"/"+tx.Key] = tx.Value
		}
	}
	if texts["clock/pattern"] != "x . . x" || texts["sequence/values"] != "60 [62 65] 67" {
		t.Errorf("texts loaded %v", texts)
	}
}
