package vocal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

func TestParseWord(t *testing.T) {
	w, err := ParseWord("ta Ke-sho a")
	want := []Syllable{{0, T, 0}, {1, K, 0}, {3, Sh, 0}, {0, NoConsonant, 0}}
	if err != nil || len(w) != len(want) {
		t.Fatalf("%v %v", w, err)
	}
	for i := range want {
		if w[i] != want[i] {
			t.Errorf("syllable %d: %v, want %v", i, w[i], want[i])
		}
	}
	for _, bad := range []string{"t", "xa", "taa", "sh"} {
		if _, err := ParseWord(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
}

func finite(t *testing.T, x ...[]float64) {
	t.Helper()
	for _, s := range x {
		for _, v := range s {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatal("NaN or Inf")
			}
		}
	}
}

func TestWordStepsPerNote(t *testing.T) {
	cfg := steady(Solo())
	cfg.Word, _ = ParseWord("ta ke so")
	v := New(cfg)
	var got []int
	for range 5 {
		v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
		got = append(got, v.sylNow)
		run(v, 0.05)
	}
	for i, w := range []int{0, 1, 2, 0, 1} {
		if got[i] != w {
			t.Fatalf("syllables %v", got)
		}
	}
	// the message sets the next one
	v.ReceiveMessage(psalm.Message{"syllable": 2})
	v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
	if v.sylNow != 2 || v.consNow != S {
		t.Errorf("syllable %d consonant %d", v.sylNow, v.consNow)
	}
	l, r := run(v, 0.3)
	finite(t, l, r)
}

func TestWordSharedInPoly(t *testing.T) {
	cfg := steady(Solo())
	cfg.Word, _ = ParseWord("ta ke so")
	p := NewPoly(cfg, 3)
	var vs []*Voice
	p.CallVoices(func(v *Voice) { vs = append(vs, v) })
	for i, want := range []int{0, 1, 2, 0} {
		v := vs[i%3]
		v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
		if v.sylNow != want {
			t.Errorf("note %d sings syllable %d, want %d", i, v.sylNow, want)
		}
	}
	p.ReceiveMessage(psalm.Message{"wordStep": 1})
	v := vs[2]
	v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
	if v.sylNow != 1 {
		t.Errorf("wordStep: syllable %d", v.sylNow)
	}
}

func TestWordVowelsSung(t *testing.T) {
	cfg := steady(Solo())
	cfg.Range, cfg.VowelGlide = Tenor, 0
	cfg.Word, _ = ParseWord("a i")
	v := New(cfg)
	f0 := hz(48)
	var ratio [2]float64
	for i := range 2 {
		v.NoteOn(1, psalm.Message{"pitch": 48.0}, psalm.CtrlTick{}, v.Config())
		x, _ := run(v, 0.6)
		y := x[int(0.3*sr()):]
		ratio[i] = harmonicsIn(y, f0, 550, 800) / harmonicsIn(y, f0, 200, 400)
		v.NoteOff()
		run(v, 1)
	}
	if ratio[0] < 3*ratio[1] {
		t.Errorf("a and i sound alike: %g, %g", ratio[0], ratio[1])
	}
}

func TestWordClock(t *testing.T) {
	cfg := steady(Solo())
	cfg.Word, _ = ParseWord("ta ke so")
	cfg.WordSync = 0.25 // a sixteenth
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
	seen := map[int]bool{}
	for range int(2 * sr() / float64(v.Config().BufferSize)) {
		v.Synthesize()
		seen[v.sylNow] = true
	}
	if len(seen) != 3 {
		t.Errorf("a held note passes %v", seen)
	}
}

func TestFormantShift(t *testing.T) {
	cfg := steady(Solo())
	cfg.Range, cfg.VowelGlide = Tenor, 0
	f0 := hz(48)
	band := func(shift float64) float64 {
		c := cfg
		c.FormantShift = shift
		x, _ := sung(c, psalm.Message{"pitch": 48.0}, 0.6)
		finite(t, x)
		y := x[int(0.3*sr()):]
		return harmonicsIn(y, f0, 850, 1100) / harmonicsIn(y, f0, 500, 750)
	}
	if lo, hi := band(1), band(1.5); hi < 3*lo {
		t.Errorf("a child's first formant doesn't move up: %g vs %g", lo, hi)
	}
	if z, o := band(0), band(1); math.Abs(z-o) > 0.3*o {
		t.Errorf("0 is 1: %g vs %g", z, o)
	}
}

func TestWordWithMatrix(t *testing.T) {
	cfg := steady(Solo())
	cfg.Word, _ = ParseWord("sha ti fe")
	cfg.WordSync = 0.5
	cfg.Ext[0] = 1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "vowel", Amount: 0.5}
	a, b := sung(cfg, psalm.Message{"pitch": 55.0}, 0.8)
	finite(t, a, b)
	c, _ := sung(cfg, psalm.Message{"pitch": 55.0}, 0.8)
	if rms(a) == 0 || math.Abs(rms(a)-rms(c)) > 0.2*rms(a) {
		t.Errorf("silent or unsteady: %g, %g", rms(a), rms(c))
	}
}
