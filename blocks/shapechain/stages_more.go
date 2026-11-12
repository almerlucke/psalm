package shapechain

import (
	"fmt"
	"math"

	"github.com/almerlucke/genny/float/shape"
	"github.com/almerlucke/genny/float/shape/shapers/funnel"
	"github.com/almerlucke/genny/float/shape/shapers/harmonics"
	"github.com/almerlucke/genny/float/shape/shapers/polynomial"
	"github.com/almerlucke/genny/float/shape/shapers/window"
)

// MoreStageNames are the stages that follow genny's shapers of folding, saturating, rectifying, bending, stepping, dead
// zones, phase distortion, windows, harmonics, polynomials and funnels. Those with state in genny (bitcrush, hysteresis, onepole,
// samplehold) are not here: a chain holds none, it is read many times at any place of the wave
var MoreStageNames = []string{"fold", "trifold", "tube", "halfwave", "diode", "curve", "ucurve", "smoothstep", "smootherstep",
	"deadzone", "softdead", "czsaw", "czsquare", "czpulse", "czres", "hann", "hamming", "blackman", "welch", "tukey", "gauss",
	"kaiser", "harm", "poly", "legendre", "funnel"}

// moreStage reads the stages of MoreStageNames; ok false when name is not one of them
func (p *parser) moreStage(t token, name string, mk func(func(float64, *Macros) float64) (stage, error)) (st stage, ok bool, err error) {
	one := func(def float64) (arg, error) {
		a, err := p.numbers(t, def)
		if err != nil {
			return arg{}, err
		}
		return a[0], nil
	}
	ok = true
	switch name {
	case "fold": // sin(g x π/2): every 1 of gain over 1 is a fold
		a, e := one(1)
		if e != nil {
			return stage{}, true, e
		}
		st, err = mk(func(x float64, m *Macros) float64 { return math.Sin(max(0, a.val(m)) * x * math.Pi / 2) })
	case "trifold": // the signal times g, reflected at -1 and 1
		a, e := one(1)
		if e != nil {
			return stage{}, true, e
		}
		st, err = mk(func(x float64, m *Macros) float64 { return foldBack(max(0, a.val(m))*x, -1, 1) })
	case "tube": // an asymmetric tanh: the bias adds even harmonics
		a, e := p.numbers(t, 2, 0.3)
		if e != nil {
			return stage{}, true, e
		}
		st, err = mk(func(x float64, m *Macros) float64 {
			d, b := max(1e-6, a[0].val(m)), max(-0.999, min(a[1].val(m), 0.999))
			off := math.Tanh(d * b)
			n := max(math.Abs(math.Tanh(d*(1+b))-off), math.Abs(math.Tanh(d*(b-1))-off))
			if n == 0 {
				n = 1
			}
			return (math.Tanh(d*(x+b)) - off) / n
		})
	case "halfwave":
		st, err = mk(func(x float64, _ *Macros) float64 { return max(x, 0) })
	case "diode": // nothing until the threshold, a soft knee
		a, e := p.numbers(t, 0, 0.1)
		if e != nil {
			return stage{}, true, e
		}
		st, err = mk(func(x float64, m *Macros) float64 {
			u, k := x-max(0, a[0].val(m)), max(0, a[1].val(m))
			switch {
			case u <= -k:
				return 0
			case u >= k:
				return u
			}
			return (u + k) * (u + k) / (4 * k)
		})
	case "curve", "ucurve": // a power curve with a tension from -1 to 1 (curve: sign kept, -1..1; ucurve: 0..1)
		a, e := one(0)
		if e != nil {
			return stage{}, true, e
		}
		uni := name == "ucurve"
		st, err = mk(func(x float64, m *Macros) float64 {
			tn := max(-0.999, min(a.val(m), 0.999))
			ex := 1 - 2*tn
			if tn > 0 {
				ex = 1 / (1 + 2*tn)
			}
			if uni {
				return math.Pow(max(0, min(x, 1)), ex)
			}
			x = max(-1, min(x, 1))
			return math.Copysign(math.Pow(math.Abs(x), ex), x)
		})
	case "smoothstep", "smootherstep": // an S between two edges
		a, e := p.numbers(t, 0, 1)
		if e != nil {
			return stage{}, true, e
		}
		if a[0].macro == 0 && a[1].macro == 0 && !(a[0].v < a[1].v) {
			return stage{}, true, p.fail(t, "%s: the first edge is below the second", name)
		}
		smoother := name == "smootherstep"
		st, err = mk(func(x float64, m *Macros) float64 {
			e0, e1 := a[0].val(m), a[1].val(m)
			if !(e1 > e0) {
				return 0
			}
			u := max(0, min((x-e0)/(e1-e0), 1))
			if smoother {
				return u * u * u * (u*(6*u-15) + 10)
			}
			return u * u * (3 - 2*u)
		})
	case "deadzone", "softdead": // 0 inside the width (softdead: the rest moves down, no jump)
		a, e := one(0.2)
		if e != nil {
			return stage{}, true, e
		}
		soft := name == "softdead"
		st, err = mk(func(x float64, m *Macros) float64 {
			w, ax := max(0, a.val(m)), math.Abs(x)
			switch {
			case ax <= w:
				return 0
			case soft:
				return math.Copysign(ax-w, x)
			}
			return x
		})
	case "czsaw", "czsquare", "czpulse", "czres": // Casio CZ phase distortion of a phase: the amount from 0 (a cosine) to 1
		a, e := p.numbers(t, 0.5, map[string]float64{"czsquare": 0.5, "czres": 4}[name])
		if e != nil {
			return stage{}, true, e
		}
		kind := name
		st, err = mk(func(x float64, m *Macros) float64 {
			return phaseDist(kind, x, max(0, min(a[0].val(m), 1)), a[1].val(m))
		})
	case "hann":
		st, err = fixedShaper(name, window.Hann().Periodic()), nil
	case "hamming":
		st, err = fixedShaper(name, window.Hamming().Periodic()), nil
	case "blackman":
		st, err = fixedShaper(name, window.Blackman().Periodic()), nil
	case "welch":
		st, err = fixedShaper(name, window.Welch().Periodic()), nil
	case "tukey", "gauss", "kaiser":
		def := map[string]float64{"tukey": 0.5, "gauss": 0.4, "kaiser": 8}[name]
		a, e := p.numbers(t, def)
		if e != nil {
			return stage{}, true, e
		}
		if a[0].macro != 0 {
			return stage{}, true, p.fail(t, "%s: a number, not a macro", name)
		}
		st, err = p.build(t, func() shape.Shaper {
			switch name {
			case "tukey":
				return window.Tukey(a[0].v).Periodic()
			case "gauss":
				return window.Gaussian(a[0].v).Periodic()
			}
			return window.Kaiser(a[0].v).Periodic()
		}, name)
	case "harm": // a sum of sines of x (1:1 2:0.5), scaled so it peaks at 1
		vals, e := p.listPairs(t, name)
		if e != nil {
			return stage{}, true, e
		}
		st, err = p.build(t, func() shape.Shaper { return harmonics.New(vals, nil, true) }, name)
	case "poly": // c0 + c1 x + c2 x² ...
		vals, e := p.listNumbers(t, name)
		if e != nil {
			return stage{}, true, e
		}
		st, err = p.build(t, func() shape.Shaper { return polynomial.New(vals...) }, name)
	case "legendre":
		a, e := p.numbers(t, 3)
		if e != nil {
			return stage{}, true, e
		}
		if a[0].macro != 0 {
			return stage{}, true, p.fail(t, "legendre: a number, not a macro")
		}
		st, err = p.build(t, func() shape.Shaper { return polynomial.NewLegendre(int(a[0].v)) }, name)
	case "funnel": // snaps to the nearest of the values
		vals, e := p.listNumbers(t, name)
		if e != nil {
			return stage{}, true, e
		}
		st, err = p.build(t, func() shape.Shaper { return funnel.New(vals) }, name)
	default:
		return stage{}, false, nil
	}
	return st, ok, err
}

// phaseDist is the phase distortion of genny's phasedist, with the amount d (0-1) and the width of a square or the harmonic of a resonance
func phaseDist(kind string, x, d, v float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		x = 0
	}
	ph := x - math.Floor(x)
	e := 1 - 0.99*d
	var g float64
	switch kind {
	case "czsaw":
		b := 0.5 * e
		if ph < b {
			g = 0.5 * ph / b
		} else {
			g = 0.5 + 0.5*(ph-b)/(1-b)
		}
	case "czsquare":
		w := max(0.01, min(v, 0.99))
		if ph < w {
			g = 0.5 * math.Min(ph/(w*e), 1)
		} else {
			g = 0.5 + 0.5*math.Min((ph-w)/((1-w)*e), 1)
		}
	case "czpulse":
		g = math.Min(ph/e, 1)
	default: // czres
		h := 1 + d*(max(1, v)-1)
		return (1 - d*ph) * math.Cos(2*math.Pi*h*ph)
	}
	return math.Cos(2 * math.Pi * g)
}

// fixedShaper is a stage of a genny shaper that has no numbers to move
func fixedShaper(name string, s shape.Shaper) stage {
	return stage{name: name, f: func(x float64, _ *Macros) float64 { return s.Shape(x) }}
}

// build makes the shaper, a panic of genny (a number out of range) being an error of the chain
func (p *parser) build(t token, mk func() shape.Shaper, name string) (st stage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = p.fail(t, "%s: %v", name, r)
		}
	}()
	s := mk()
	return fixedShaper(name, s), nil
}

// listNumbers reads the numbers after the name (at least one)
func (p *parser) listNumbers(t token, name string) ([]float64, error) {
	var vals []float64
	for {
		c, ok := p.peek()
		if !ok || !isNumber(c) || macroNumber(c.s) > 0 {
			break
		}
		v, err := p.constant("a number")
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)
		p.skipComma(func(t token) bool { return isNumber(t) && macroNumber(t.s) == 0 })
	}
	if len(vals) == 0 {
		return nil, p.fail(t, "%s: numbers are expected", name)
	}
	return vals, nil
}

// listPairs reads harmonics (1:1 2:0.5) into the amplitudes of harmonics 1 to the highest
func (p *parser) listPairs(t token, name string) ([]float64, error) {
	var amps []float64
	for {
		b, ok := p.peek()
		if !ok {
			break
		}
		var n int
		var v float64
		if c, err := fmt.Sscanf(b.s, "%d:%g", &n, &v); c != 2 || err != nil {
			break
		}
		if n < 1 || n > 64 || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, p.fail(b, "%q: a harmonic from 1 to 64 and its level (3:0.3)", b.s)
		}
		p.i++
		for len(amps) < n {
			amps = append(amps, 0)
		}
		amps[n-1] = v
		p.skipComma(func(t token) bool { var n int; var v float64; c, _ := fmt.Sscanf(t.s, "%d:%g", &n, &v); return c == 2 })
	}
	if len(amps) == 0 {
		return nil, p.fail(t, "%s: the harmonics and their levels (%s 1:1 3:0.3)", name, name)
	}
	return amps, nil
}
