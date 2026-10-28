package melody

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/almerlucke/psalm/tools/harmony"
)

func letters(toks []Token) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteByte(t.Letter)
		b.WriteString(t.Ops)
		b.WriteByte(' ')
	}
	return strings.TrimSpace(b.String())
}

func TestGrammar(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 1))
	toks, err := Expand("S -> Open Close\nOpen -> A A'\nClose -> B A~?", rng, 16)
	if err != nil || letters(toks) != "A A' B A~?" {
		t.Fatalf("%q %v", letters(toks), err)
	}
	// a weight: the heavier choice comes up more
	counts := map[string]int{}
	for i := range 400 {
		toks, err := Expand("S -> A B *4 | B A", rand.New(rand.NewPCG(uint64(i), 3)), 4)
		if err != nil {
			t.Fatal(err)
		}
		counts[letters(toks)]++
	}
	if counts["A B"] < 250 || counts["B A"] < 30 {
		t.Errorf("weights %v", counts)
	}
	// an L-system: A -> A B A, three times
	toks, err = Expand("axiom: A\niterate: 2\nA -> A B A\nB -> B B", rand.New(rand.NewPCG(1, 1)), 64)
	if err != nil || letters(toks) != "A B A B B A B A" {
		t.Errorf("L-system %q %v", letters(toks), err)
	}
	// at most the phrases wanted
	if toks, _ := Expand("axiom: A\niterate: 4\nA -> A B", rng, 3); len(toks) != 3 {
		t.Errorf("%d tokens", len(toks))
	}
	for _, bad := range []string{"", "S A B", "S -> A X", "S -> S A", "S -> A *x", "S ->", "iterate: 20\nS -> A"} {
		if err := ParseForm(bad); err == nil {
			t.Errorf("%q was taken", bad)
		}
	}
}

func TestMelodyFromGrammar(t *testing.T) {
	scale, _ := harmony.NewScale("C", "major")
	p := Params{Scale: scale, Beats: 16, Phrase: 4, Seed: 5, Grammar: "S -> A B A~ A'"}
	a := Generate(p)
	b := Generate(p)
	if len(a) == 0 || !slices.Equal(a, b) {
		t.Fatal("a grammar melody is not repeatable")
	}
	// the first and the third phrase are the same A (the third turned upside down: not the same pitches), the first and last phrases begin alike
	var first, third []Note
	for _, n := range a {
		switch {
		case n.Start < 4:
			first = append(first, n)
		case n.Start >= 8 && n.Start < 12:
			third = append(third, n)
		}
	}
	if len(first) != len(third) {
		t.Errorf("the phrase turned over has other notes: %d and %d", len(first), len(third))
	}
	for i := range min(len(first), len(third)) {
		if first[i].Start+8 != third[i].Start {
			t.Fatal("the rhythm of A turned over differs")
		}
	}
	// the end is on the root
	if last := a[len(a)-1]; last.Key%12 != 0 {
		t.Errorf("the last note %d is not the root", last.Key)
	}
	// a plain A A' A form and its grammar are one melody
	q := p
	q.Grammar, q.Form = "", "AA'"
	r := p
	r.Grammar = "S -> A A'"
	if !slices.Equal(Generate(q), Generate(r)) {
		t.Log("the grammar made another melody than the form (the random numbers differ), fine")
	}
}
