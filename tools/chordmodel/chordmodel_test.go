package chordmodel

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/almerlucke/psalm/tools/harmony"
)

func rng(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed*7+1)) }

func bars(slots []Slot) float64 {
	sum := 0.0
	for _, s := range slots {
		sum += s.Bars
	}
	return sum
}

// the presets are models of numerals that are all chords of their key, with rows of weights, and they give progressions of the length asked
func TestPresets(t *testing.T) {
	for _, name := range PresetNames() {
		m, ok := Preset(name)
		if !ok {
			t.Fatal(name)
		}
		scale := harmony.MustScale("D", "major")
		for _, s := range m.States {
			if _, err := scale.Roman(s); err != nil {
				t.Errorf("%s: %s is no chord: %v", name, s, err)
			}
		}
		for i, row := range m.W {
			sum := 0.0
			for _, x := range row {
				sum += x
			}
			if sum <= 0 {
				t.Errorf("%s: the chord %s leads nowhere", name, m.States[i])
			}
		}
		for _, n := range []float64{4, 8, 12.5} {
			slots := m.Fill(n, rng(3))
			if math.Abs(bars(slots)-n) > 1e-9 {
				t.Errorf("%s: %v bars asked, %v given", name, n, bars(slots))
			}
		}
	}
	if _, ok := Preset("polka"); ok {
		t.Error("a preset that does not exist")
	}
}

// the chords follow the chances of the rows: after V in the pop model the I comes about half the time
func TestFollowsTheChances(t *testing.T) {
	m, _ := Preset("pop")
	r := rng(5)
	v, i := m.Index("V"), m.Index("I")
	got, n := 0, 0
	for range 20000 {
		if m.pick(r, v, -1, -1) == i {
			got++
		}
		n++
	}
	if f := float64(got) / float64(n); math.Abs(f-0.5) > 0.02 {
		t.Errorf("I after V %.3f of the times, want 0.5", f)
	}
	// every chord can begin; the start chances are those of Start
	starts := map[int]int{}
	for range 5000 {
		starts[m.Fill(1, r)[0].State]++
	}
	if f := float64(starts[i]) / 5000; math.Abs(f-0.8) > 0.03 {
		t.Errorf("I begins %.3f of the progressions, want 0.8", f)
	}
}

// a locked chord stays, the ones round it fit it: before a locked I a V is likelier than a ii... and the end chord ends
func TestLocksAndTheEnd(t *testing.T) {
	m, _ := Preset("pop")
	r := rng(9)
	slots := m.Fill(8, r)
	slots[3] = Slot{State: m.Index("vi"), Bars: slots[3].Bars, Lock: true}
	locked := slots[3]
	for range 200 {
		m.Reroll(slots, r)
		if slots[3] != locked {
			t.Fatalf("the locked chord moved: %v", slots[3])
		}
	}
	// the chord before a locked I is mostly V (V -> I is the likeliest): compare with the chances without the lock
	before := 0
	slots = []Slot{{State: m.Index("IV"), Bars: 1, Lock: true}, {Bars: 1}, {State: m.Index("I"), Bars: 1, Lock: true}}
	for range 4000 {
		m.Reroll(slots, r)
		if slots[1].State == m.Index("V") {
			before++
		}
	}
	if f := float64(before) / 4000; f < 0.5 {
		t.Errorf("V between IV and I %.3f of the times, want the likeliest by far", f)
	}
	m.End = m.Index("I")
	for range 100 {
		s := m.Fill(8, r)
		if last := s[len(s)-1]; last.State != m.Index("I") {
			t.Fatalf("a progression that ends on %s", m.States[last.State])
		}
	}
	// the chord before the end leads to it: V or IV or vii° far more often than ii
	count := map[string]int{}
	for range 3000 {
		s := m.Fill(4, r)
		if len(s) >= 2 {
			count[m.States[s[len(s)-2].State]]++
		}
	}
	if count["V"] < 3*count["ii"] {
		t.Errorf("before the end: %v, want V often and ii rarely", count)
	}
}

// the lengths of the chords follow their weights (here only 2 and 1/2 bars) and the last is cut to fit
func TestLengths(t *testing.T) {
	m, _ := Preset("pop")
	m.Lengths = []Length{{0.5, 1}, {1, 0}, {2, 1}, {4, 0}}
	r := rng(2)
	for range 50 {
		slots := m.Fill(5, r)
		for _, s := range slots {
			if s.Bars != 0.5 && s.Bars != 2 && s.Bars != 1 && s.Bars != 1.5 { // the last may be cut to what is left
				t.Fatalf("a chord of %v bars: %v", s.Bars, slots)
			}
		}
		if math.Abs(bars(slots)-5) > 1e-9 {
			t.Fatalf("%v bars", bars(slots))
		}
	}
}

// learning makes what was seen likelier, adds the chords it did not have, and keeps pairs that Memory trusts
func TestLearn(t *testing.T) {
	m := New("I", "IV", "V")
	n, err := m.LearnText("I IV V I\nI IV V I\nI bVII IV", 0.5)
	if err != nil || n != 3 {
		t.Fatal(n, err)
	}
	if m.Index("bVII") != 3 || len(m.W) != 4 || len(m.W[3]) != 4 || len(m.Start) != 4 {
		t.Fatalf("the new chord is not in the model: %v", m.States)
	}
	i, iv, v := m.Index("I"), m.Index("IV"), m.Index("V")
	if p := m.Next(iv, -1); p[v] < 0.4 {
		t.Errorf("V after IV %.2f, it was seen twice", p[v])
	}
	if p := m.Next(v, -1); p[i] < 0.5 {
		t.Errorf("I after V %.2f", p[i])
	}
	// after I then IV it was V twice (and never anything else): with Memory the pair decides
	m.Memory = 1
	if p := m.Next(iv, i); p[v] < 0.99 {
		t.Errorf("V after I IV with memory: %.2f", p[v])
	}
	m.Memory = 0
	if p := m.Next(iv, i); p[v] > 0.99 {
		t.Errorf("without memory the pair should not decide: %.2f", p[v])
	}
	if _, err := m.LearnText("I xyz", 0.5); err == nil {
		t.Error("a chord that is no numeral is not reported")
	}
	m.RemoveState(m.Index("bVII"))
	if len(m.States) != 3 || len(m.W) != 3 || m.Pairs != nil {
		t.Errorf("the chord taken out: %v", m.States)
	}
}

// the progression as text for the harmony module, in a key
func TestText(t *testing.T) {
	m, _ := Preset("pop")
	slots := []Slot{{State: m.Index("I"), Bars: 1}, {State: m.Index("V"), Bars: 2}, {State: m.Index("vi"), Bars: 0.5}, {State: m.Index("IV"), Bars: 1}}
	d := harmony.MustScale("D", "major")
	if got := m.Text(slots, d, true); got != "D A*2 Bm*0.5 G" {
		t.Errorf("symbols %q", got)
	}
	if got := m.Text(slots, d, false); got != "I V*2 vi*0.5 IV" {
		t.Errorf("numerals %q", got)
	}
	j, _ := Preset("jazz")
	if got := j.ChordName(j.Index("V7"), d); !strings.HasPrefix(got, "A7") {
		t.Errorf("jazz V7 in D: %q", got)
	}
}

// a model is kept as JSON and comes back the same
func TestJSON(t *testing.T) {
	m, _ := Preset("minor")
	m.Memory, m.End = 0.4, m.Index("i")
	_, _ = m.LearnText("i VI VII i", 0.3)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var back Model
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(back.States, m.States) || back.Memory != 0.4 || back.End != m.End || len(back.Pairs) != len(m.Pairs) || back.W[0][5] != m.W[0][5] {
		t.Errorf("model after JSON: %+v", back)
	}
	// the same seed gives the same progression
	a, c := m.Fill(8, rng(4)), back.Fill(8, rng(4))
	if !slices.Equal(a, c) {
		t.Errorf("%v against %v", a, c)
	}
}
