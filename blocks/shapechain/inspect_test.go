package shapechain

import (
	"math"
	"reflect"
	"strconv"
	"testing"
)

func TestSplitStages(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"sin", []string{"sin"}},
		{"bipolar | fold  m1[1, 6]\n | softclip 2", []string{"bipolar", "fold m1[1, 6]", "softclip 2"}},
		{"par(sin, mult 3 | sin) | tanh 2", []string{"par(sin, mult 3 | sin)", "tanh 2"}},
		{"sin # a comment | not a stage\n| fold 2", []string{"sin", "fold 2"}},
	} {
		if got := SplitStages(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: %q, want %q", c.in, got, c.want)
		}
	}
}

// the stages cut apart read as the chain does: every stage of the catalog, with its numbers as they start, is a chain
func TestCatalogStagesRead(t *testing.T) {
	for _, s := range Catalog {
		text := s.Example
		if text == "" {
			text = s.Name
			for _, p := range s.Params {
				text += " " + trimNum(p.Def)
			}
		}
		if _, err := Parse(text); err != nil {
			t.Errorf("%s: %q: %v", s.Name, text, err)
		}
		for _, p := range s.Params {
			if !(p.Lo < p.Hi) || p.Def < p.Lo || p.Def > p.Hi {
				t.Errorf("%s %s: default %v outside %v..%v", s.Name, p.Name, p.Def, p.Lo, p.Hi)
			}
		}
	}
}

func trimNum(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

func TestSpectrumOfSine(t *testing.T) {
	w := MustParse("sin").Wave(256, Macros{})
	sp := Spectrum(w, 8)
	if sp[0] != 1 {
		t.Errorf("the first harmonic is %v", sp[0])
	}
	for i := 1; i < len(sp); i++ {
		if sp[i] > 1e-6 {
			t.Errorf("harmonic %d of a sine is %v", i+1, sp[i])
		}
	}
	saw := Spectrum(MustParse("bipolar").Wave(256, Macros{}), 4)
	if math.Abs(saw[1]-0.5) > 0.01 {
		t.Errorf("the second harmonic of a saw is %v, want 0.5", saw[1])
	}
}
