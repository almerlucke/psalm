package melody

import (
	"slices"
	"testing"

	"github.com/almerlucke/psalm/tools/harmony"
)

func TestCounterLine(t *testing.T) {
	scale, _ := harmony.NewScale("C", "major")
	chords := []Span{{At: 0, Len: 4, Pcs: []int{0, 4, 7}, Root: 0}, {At: 4, Len: 4, Pcs: []int{5, 9, 0}, Root: 5}, {At: 8, Len: 4, Pcs: []int{7, 11, 2}, Root: 7}, {At: 12, Len: 4, Pcs: []int{0, 4, 7}, Root: 0}}
	good, total, contrary, similar := 0, 0, 0, 0
	for seed := uint64(1); seed <= 20; seed++ {
		m := Generate(Params{Scale: scale, Beats: 16, Low: 64, High: 84, Chords: chords, Form: "AA'", Phrase: 8, Seed: seed})
		c := Counter(m, CounterParams{Scale: scale, Chords: chords, Low: 48, High: 70, Motion: 1})
		if len(c) != len(m) {
			t.Fatalf("%d notes under %d", len(c), len(m))
		}
		again := Counter(m, CounterParams{Scale: scale, Chords: chords, Low: 48, High: 70, Motion: 1})
		if !slices.Equal(c, again) {
			t.Fatal("not repeatable")
		}
		for i := range c {
			if c[i].Key < 48 || c[i].Key > 70 || c[i].Key >= m[i].Key {
				t.Fatalf("seed %d note %d: %d under %d, out of its range or not under the melody", seed, i, c[i].Key, m[i].Key)
			}
			iv := ((m[i].Key-c[i].Key)%12 + 12) % 12
			total++
			if slices.Contains([]int{0, 3, 4, 7, 8, 9}, iv) {
				good++
			}
			if i == 0 {
				continue
			}
			ia := ((m[i-1].Key-c[i-1].Key)%12 + 12) % 12
			dm, dc := m[i].Key-m[i-1].Key, c[i].Key-c[i-1].Key
			if (ia == 0 || ia == 7) && ia == iv && dm != 0 && dc != 0 {
				t.Fatalf("seed %d note %d: parallel perfect intervals (%d)", seed, i, iv)
			}
			switch {
			case dm > 0 && dc < 0 || dm < 0 && dc > 0:
				contrary++
			case dm != 0 && dc != 0:
				similar++
			}
		}
	}
	if float64(good)/float64(total) < 0.8 {
		t.Errorf("only %d of %d intervals are consonant", good, total)
	}
	if contrary <= similar {
		t.Errorf("contrary motion %d is not more than similar %d", contrary, similar)
	}
	// a note a bar under the melody
	m := Generate(Params{Scale: scale, Beats: 16, Chords: chords, Seed: 3})
	c := Counter(m, CounterParams{Scale: scale, Chords: chords, Every: 4})
	if len(c) != 4 || c[1].Start != 4 {
		t.Errorf("a note every four beats: %+v", c)
	}
	if Counter(nil, CounterParams{Scale: scale}) != nil {
		t.Error("a counter-line to nothing")
	}
}
