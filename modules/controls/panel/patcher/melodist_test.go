package patcher_test

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/noteroll"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// melodistPlays runs a melodist (with the settings the function gives it) for some seconds at 120 bpm in a patch with the harmony and the song of a rig, and gives the notes it played
// and the samples they started on
func melodistRun(t *testing.T, seconds float64, setup func(m *patcher.Melodist)) (*harmonyRig, *patcher.Melodist, *played) {
	t.Helper()
	r := newHarmonyRig(t, "A 2, B 2", "A: Dm7 G7\nB: Cmaj7 Am7")
	id := r.add("melodist")
	m := r.pt.Module(id).(*patcher.Melodist)
	setup(m)
	r.connect(patcher.Cable{From: id, To: r.add("rec")})
	for range 80 { // the first passes are made before the song begins: it plays from its first beat
		if m.Passes() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	r.run(seconds)
	return r, m, r.rec
}

func TestMelodistPlaysTheSections(t *testing.T) {
	r, m, rec := melodistRun(t, 8.4, func(m *patcher.Melodist) {
		m.ReceiveMessage(psalm.Message{"scale": float64(0), "form": float64(1), "phrase": float64(2), "density": 0.6, "low": float64(62), "high": float64(86)})
	})
	_ = r
	if len(rec.pitches) < 8 {
		t.Fatalf("%d notes in two sections", len(rec.pitches))
	}
	// the harmony says C major (key 0, scale major): every note in it, in the range; some in each section (4 s of A, then 4 s of B)
	inC := map[int]bool{0: true, 2: true, 4: true, 5: true, 7: true, 9: true, 11: true}
	a, b := 0, 0
	for i, p := range rec.pitches {
		if !inC[int(p)%12] || p < 62 || p > 86 {
			t.Errorf("note %v is not in C major or the range", p)
		}
		if rec.at[i] < 4*44100 {
			a++
		} else {
			b++
		}
	}
	if a < 3 || b < 3 {
		t.Errorf("%d notes in A, %d in B", a, b)
	}
	if m.Status() == "" {
		t.Error("the tile says nothing")
	}
}

// the second time through is the first, changed a little (mutate), the same (0) or new (1); frozen it stays
func TestMelodistMutatesBetweenPasses(t *testing.T) {
	pass := func(mutate float64, freeze bool) [][]float64 {
		psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
		defer psalm.PopConfig()
		p := psalm.New(nil)
		rec := &played{Module: psalm.NewControl()}
		rec.SetSelf(rec)
		kinds := append(patcher.Generators(), patcher.Kind{Name: "rec", Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}},
			Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: rec}, nil }})
		pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
		id, _ := pt.Add("melodist", "", 0, 0)
		rc, _ := pt.Add("rec", "", 0, 0)
		m := pt.Module(id).(*patcher.Melodist)
		m.ReceiveMessage(psalm.Message{"length": float64(8), "mutate": mutate, "freeze": freeze, "density": 0.6, "seed": float64(3)})
		if _, err := pt.Connect(patcher.Cable{From: id, To: rc}); err != nil {
			t.Fatal(err)
		}
		p.Out(pt.Out())
		run := func(sec float64) {
			for range int(sec * 44100 / 256) {
				p.ResetRun()
				p.Run()
			}
		}
		run(0.01)
		for i := 0; i < 200 && m.Passes() < 2; i++ {
			time.Sleep(5 * time.Millisecond)
		}
		rec.pitches, rec.at = nil, nil
		p.Restart()
		run(12.1) // 3 passes of 8 beats (4 s)
		out := make([][]float64, 3)
		for i, pitch := range rec.pitches {
			k := int(rec.at[i] / (4 * 44100))
			if k < 3 {
				out[k] = append(out[k], pitch)
			}
		}
		return out
	}
	same := pass(0, false)
	if len(same[0]) < 4 || !slices.Equal(same[0], same[1]) || !slices.Equal(same[1], same[2]) {
		t.Errorf("mutate 0: %v", same)
	}
	moved := pass(0.4, false)
	if len(moved[1]) == 0 || slices.Equal(moved[0], moved[1]) {
		t.Errorf("mutate 0.4 changes nothing: %v", moved)
	}
	common := 0
	for _, a := range moved[0] {
		if slices.Contains(moved[1], a) {
			common++
		}
	}
	if float64(common)/float64(len(moved[0])) < 0.5 {
		t.Errorf("mutate 0.4 is another melody: %v and %v", moved[0], moved[1])
	}
	fresh := pass(1, false)
	if slices.Equal(fresh[0], fresh[1]) {
		t.Errorf("mutate 1 repeats: %v", fresh)
	}
	frozen := pass(1, true)
	if !slices.Equal(frozen[0], frozen[1]) || !slices.Equal(frozen[1], frozen[2]) {
		t.Errorf("frozen changes: %v", frozen)
	}
	_ = math.Pi
}

// keep writes what plays into the clip of its section in a roll
func TestMelodistKeepsInARoll(t *testing.T) {
	r, m, _ := melodistRun(t, 2.5, func(m *patcher.Melodist) {})
	roll := r.add("noteroll")
	rl := r.pt.Module(roll).(*noteroll.Roll)
	var id string
	for _, n := range r.pt.Nodes() {
		if n.Kind == "melodist" {
			id = n.ID
		}
	}
	if err := r.pt.Act(id, "keep"); err != nil {
		t.Fatal(err)
	}
	c, ok := rl.Clip("A")
	if !ok || len(c.Notes) < 3 || c.Length != 8 {
		t.Errorf("the clip of A: %+v %v", c, ok)
	}
	_ = m
	if err := r.pt.Act(id, "nothing"); err == nil {
		t.Error("a button that is not there")
	}
}
