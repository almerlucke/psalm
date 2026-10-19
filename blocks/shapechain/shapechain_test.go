package shapechain

import (
	"math"
	"strings"
	"testing"

	"github.com/almerlucke/genny/float/shape"
	"github.com/almerlucke/genny/float/shape/shapers/add"
	"github.com/almerlucke/genny/float/shape/shapers/chebyshev"
	"github.com/almerlucke/genny/float/shape/shapers/clip"
	"github.com/almerlucke/genny/float/shape/shapers/function"
	"github.com/almerlucke/genny/float/shape/shapers/linear"
	"github.com/almerlucke/genny/float/shape/shapers/lookup"
	"github.com/almerlucke/genny/float/shape/shapers/mirror"
	"github.com/almerlucke/genny/float/shape/shapers/mod"
	"github.com/almerlucke/genny/float/shape/shapers/mult"
	"github.com/almerlucke/genny/float/shape/shapers/parallel"
	"github.com/almerlucke/genny/float/shape/shapers/pulse"
	"github.com/almerlucke/genny/float/shape/shapers/quantize"
	"github.com/almerlucke/genny/float/shape/shapers/ripple"
	"github.com/almerlucke/genny/float/shape/shapers/series"
)

// every stage that genny has gives what genny's shaper gives
func TestAgainstGenny(t *testing.T) {
	for _, c := range []struct {
		text string
		g    shape.Shaper
	}{
		{"add 0.25", add.New(0.25)},
		{"mult 3", mult.New(3)},
		{"mod 0.3", mod.New(0.3)},
		{"ripple 0.4", ripple.New(0.4)},
		{"linear 2 -1", linear.New(2, -1)},
		{"bipolar", linear.NewBipolar()},
		{"unipolar", linear.NewUnipolar()},
		{"abs", function.NewAbs()},
		{"sin", function.NewSin()},
		{"tri", function.NewTri()},
		{"mirror -0.5 0.5", mirror.New(-0.5, 0.5)},
		{"pulse 0.3", pulse.New(0.3)},
		{"quantize 0.25", quantize.New(0.25)},
		{"clip -0.5 0.7", clip.New(clip.Hard, -0.5, 0.7)},
		{"softclip 3", clip.New(clip.Soft, 3)},
		{"cheby 1:1 3:0.3 5:-0.2", chebyshev.New(map[int]float64{1: 1, 3: 0.3, 5: -0.2})},
		{"table sine 64", lookup.NewSineTable(64)},
		{"table sawtooth", lookup.NewSawtoothTable(256)},
		{"table uni-triangle 32", lookup.NewUniTriangleTable(32)},
		{"par(mult 2, add 1)", parallel.New(mult.New(2), add.New(1))},
		{"mult 2 | add 1 | sin", series.New(mult.New(2), add.New(1), function.NewSin())},
		{"par(mult 2 | sin, bipolar), add 0", nil}, // below
	} {
		if c.g == nil {
			continue
		}
		ch, err := Parse(c.text)
		if err != nil {
			t.Errorf("%q: %v", c.text, err)
			continue
		}
		for x := -0.3; x < 1.4; x += 0.0173 {
			// genny's tables read beyond their ends: only 0..1 for them
			if strings.HasPrefix(c.text, "table") && (x < 0 || x >= 1) {
				continue
			}
			if want, got := c.g.Shape(x), ch.Shape(x); math.Abs(want-got) > 1e-6 {
				t.Errorf("%q at %v: %v, genny %v", c.text, x, got, want)
				break
			}
		}
	}
}

func TestMacros(t *testing.T) {
	c := MustParse("mult m1[1,5] | add m2")
	if !c.Uses(1) || !c.Uses(2) || c.Uses(3) {
		t.Errorf("uses %v", c.macros)
	}
	if got := c.Eval(2, &Macros{0, 0, 0, 0}); got != 2 {
		t.Errorf("m1=0, m2=0: %v", got)
	}
	if got := c.Eval(2, &Macros{1, 0.5, 0, 0}); got != 10.5 {
		t.Errorf("m1=1, m2=0.5: %v", got)
	}
	if got := MustParse("mult m1[5,1]").Eval(1, &Macros{1}); got != 1 {
		t.Errorf("a range upside down: %v", got)
	}
	if got := MustParse("mod m1").Eval(0.7, &Macros{0}); got != 0.7 {
		t.Errorf("mod by a macro at 0 is x: %v", got)
	}
}

func TestSyntax(t *testing.T) {
	for _, text := range []string{
		"sin", "mult 3|sin", "mult 3 | sin", "bipolar # a saw\n | mirror -0.5, 0.5",
		"par(sin, mult 3 | sin)", "par(mult 2, sin)", "par(par(sin, sin), tri) | mult 0.5",
		"mult m1", "mult m3[0 , 8]", "cheby 1:1, 3:0.5", "table 0 1 0.5 -1", "table 0, 1", "clip", "add", "pow 2 | tanh 3",
	} {
		if _, err := Parse(text); err != nil {
			t.Errorf("%q: %v", text, err)
		}
	}
	for text, want := range map[string]string{
		"":             "a stage was expected",
		"sin |":        "a stage was expected",
		"wobble":       `"wobble" is not a stage`,
		"mod 0":        "not 0",
		"quantize 0":   "not 0",
		"mirror 1 0":   "bottom is below the top",
		"par(sin":      "a ) is missing",
		"par sin":      "in parentheses",
		"sin )":        "a stage or |",
		"cheby":        "the harmonics",
		"cheby 0:1":    "a harmonic from 1 to 64",
		"table zigzag": "not a wave",
		"table 1":      "2 or more",
		"mult m1[1,2":  "a ] is missing",
		"mult m1[1":    "high end of the range",
		"sin\n | x":    "column",
		"mult 2 3":     "a stage or |",
	} {
		_, err := Parse(text)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want an error of %q", text, err, want)
		}
	}
}

func TestMirrorFar(t *testing.T) {
	// genny's mirror loops once per fold; this one folds at once
	for _, x := range []float64{1e9, -1e9, 12345.678} {
		if y := foldBack(x, 0, 1); y < 0 || y > 1 {
			t.Errorf("%v folds to %v", x, y)
		}
	}
	if got := foldBack(1.25, 0, 1); math.Abs(got-0.75) > 1e-12 {
		t.Errorf("1.25 folds to %v", got)
	}
}

// the table sine is the sine, over a few cycles and far away, and at the edges
func TestSinTurns(t *testing.T) {
	worst := 0.0
	for x := -7.3; x < 9; x += 0.000731 {
		worst = math.Max(worst, math.Abs(sinTurns(x)-math.Sin(2*math.Pi*x)))
	}
	if worst > 4e-7 {
		t.Errorf("the table sine is %v from the sine", worst)
	}
	for _, x := range []float64{0, 1, -1, 0.9999999999, 1e12 + 0.25} {
		if y := sinTurns(x); math.IsNaN(y) || math.Abs(y) > 1 {
			t.Errorf("sinTurns(%v) = %v", x, y)
		}
	}
}

func TestBandwidth(t *testing.T) {
	for _, c := range []struct {
		text     string
		min, max int
	}{
		{"sin", 1, 1},
		{"par(sin, mult 3 | sin | mult 0.4)", 3, 3},
		{"par(sin, mult 7 | sin | mult 0.4, mult 13 | sin | mult 0.1)", 13, 13},
		{"sin | mult 3 | mirror -1 1", 8, 300},          // a fold: corners, a hundred or so harmonics
		{"sin | mult m1[1,7] | mirror -1 1", 100, Wide}, // folded seven times over at the top of the macro
		{"bipolar", Wide, Wide},
		{"pulse 0.5 | bipolar", Wide, Wide},
		{"bipolar | quantize 0.1", Wide, Wide},
		{"mult m1[1,6] | mod 1 | bipolar", Wide, Wide},
		{"tri | bipolar | softclip 1.5", 3, 200},
	} {
		got := MustParse(c.text).Bandwidth()
		if got < c.min || got > c.max {
			t.Errorf("%q: bandwidth %d, want %d to %d", c.text, got, c.min, c.max)
		}
	}
}
