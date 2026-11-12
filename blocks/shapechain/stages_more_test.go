package shapechain

import (
	"math"
	"testing"

	"github.com/almerlucke/genny/float/shape"
	"github.com/almerlucke/genny/float/shape/shapers/curve"
	"github.com/almerlucke/genny/float/shape/shapers/deadzone"
	"github.com/almerlucke/genny/float/shape/shapers/funnel"
	"github.com/almerlucke/genny/float/shape/shapers/harmonics"
	"github.com/almerlucke/genny/float/shape/shapers/phasedist"
	"github.com/almerlucke/genny/float/shape/shapers/polynomial"
	"github.com/almerlucke/genny/float/shape/shapers/rectify"
	"github.com/almerlucke/genny/float/shape/shapers/saturate"
	"github.com/almerlucke/genny/float/shape/shapers/smoothstep"
	"github.com/almerlucke/genny/float/shape/shapers/wavefold"
	"github.com/almerlucke/genny/float/shape/shapers/window"
)

// the stages that follow genny's newer shapers give what genny's shapers give
func TestMoreAgainstGenny(t *testing.T) {
	for _, c := range []struct {
		text string
		g    shape.Shaper
	}{
		{"fold 2.5", wavefold.NewSine(2.5)},
		{"trifold 3", wavefold.NewTriangle(3)},
		{"tube 3 0.4", saturate.NewTube(3, 0.4)},
		{"halfwave", rectify.New(rectify.HalfWave)},
		{"diode 0.1 0.2", rectify.NewDiode(0.1, 0.2)},
		{"curve 0.5", curve.NewBipolar(0.5)},
		{"curve -0.3", curve.NewBipolar(-0.3)},
		{"ucurve 0.6", curve.NewUnipolar(0.6)},
		{"smoothstep 0.2 0.8", smoothstep.New(smoothstep.Smoothstep, 0.2, 0.8)},
		{"smootherstep 0.2 0.8", smoothstep.New(smoothstep.Smootherstep, 0.2, 0.8)},
		{"deadzone 0.3", deadzone.NewDeadZone(0.3, deadzone.Jump)},
		{"softdead 0.3", deadzone.NewDeadZone(0.3, deadzone.Continuous)},
		{"czsaw 0.7", phasedist.NewSaw(0.7)},
		{"czsquare 0.6 0.3", phasedist.NewSquareWidth(0.6, 0.3)},
		{"czpulse 0.8", phasedist.NewPulse(0.8)},
		{"czres 0.5 6", phasedist.NewResonant(0.5, 6)},
		{"hann", window.Hann().Periodic()},
		{"kaiser 6", window.Kaiser(6).Periodic()},
		{"tukey 0.3", window.Tukey(0.3).Periodic()},
		{"harm 1:1 3:0.4", harmonics.New([]float64{1, 0, 0.4}, nil, true)},
		{"poly 0 1 0 -0.3", polynomial.New(0, 1, 0, -0.3)},
		{"legendre 4", polynomial.NewLegendre(4)},
		{"funnel -1 -0.25 0.5 1", funnel.New([]float64{-1, -0.25, 0.5, 1})},
	} {
		ch, err := Parse(c.text)
		if err != nil {
			t.Errorf("%q: %v", c.text, err)
			continue
		}
		for i := -20; i <= 20; i++ {
			x := float64(i) / 20
			if got, want := ch.Shape(x), c.g.Shape(x); math.Abs(got-want) > 1e-9 {
				t.Errorf("%q at %v: %v, genny %v", c.text, x, got, want)
				break
			}
		}
	}
}

// macros move the numbers of the new stages, and numbers out of range are errors
func TestMoreMacrosAndErrors(t *testing.T) {
	ch := MustParse("bipolar | fold m1[1,4]")
	a, b := ch.Eval(0.3, &Macros{0}), ch.Eval(0.3, &Macros{1})
	if a == b {
		t.Error("m1 does not move fold")
	}
	for _, bad := range []string{"legendre 40", "gauss 0", "poly", "harm", "smoothstep 1 0", "funnel", "tukey m1"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
}
