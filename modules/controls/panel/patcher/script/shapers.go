package script

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/float/shape"
	"github.com/almerlucke/genny/float/shape/shapers/add"
	"github.com/almerlucke/genny/float/shape/shapers/chebyshev"
	"github.com/almerlucke/genny/float/shape/shapers/clip"
	"github.com/almerlucke/genny/float/shape/shapers/emulations/hardsync"
	"github.com/almerlucke/genny/float/shape/shapers/emulations/jp8000"
	"github.com/almerlucke/genny/float/shape/shapers/emulations/pwmod"
	"github.com/almerlucke/genny/float/shape/shapers/emulations/softsync"
	"github.com/almerlucke/genny/float/shape/shapers/emulations/supersaw"
	"github.com/almerlucke/genny/float/shape/shapers/emulations/voyager"
	"github.com/almerlucke/genny/float/shape/shapers/function"
	"github.com/almerlucke/genny/float/shape/shapers/interp"
	"github.com/almerlucke/genny/float/shape/shapers/linear"
	"github.com/almerlucke/genny/float/shape/shapers/lookup"
	"github.com/almerlucke/genny/float/shape/shapers/mirror"
	"github.com/almerlucke/genny/float/shape/shapers/mixer"
	"github.com/almerlucke/genny/float/shape/shapers/mod"
	"github.com/almerlucke/genny/float/shape/shapers/mult"
	"github.com/almerlucke/genny/float/shape/shapers/parallel"
	"github.com/almerlucke/genny/float/shape/shapers/pulse"
	"github.com/almerlucke/genny/float/shape/shapers/quantize"
	"github.com/almerlucke/genny/float/shape/shapers/ripple"
	"github.com/almerlucke/genny/float/shape/shapers/route"
	"github.com/almerlucke/genny/float/shape/shapers/series"
	"github.com/almerlucke/genny/float/shape/shapers/thru"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/dop251/goja"
)

// Shaper is a shaper of genny in a script: a function of a number to a number, to put after a
// generator (gen.shape(s)) or to call (s.shape(x), x a number, a note name or a chord). Where a
// script wants a shaper it can give a function too
type Shaper struct {
	s shape.Shaper
	r *Runtime
}

// Shape shapes a number, a note name, or every pitch of a chord
func (s *Shaper) Shape(v goja.Value) goja.Value { return s.r.shaped(s.s, v) }

// Shape is a generator of the values of this one through a shaper: gen.shape(shaper)
func (g *Gen) Shape(s goja.Value) *Gen {
	return &Gen{g: shapedGen{g: g.g, s: g.r.shaper(s), r: g.r}, vm: g.vm, r: g.r}
}

// shapedGen shapes the values of a generator
type shapedGen struct {
	g genny.Generator[goja.Value]
	s shape.Shaper
	r *Runtime
}

func (s shapedGen) Generate() goja.Value { return s.r.shaped(s.s, s.g.Generate()) }
func (s shapedGen) Continuous() bool     { return s.g.Continuous() }
func (s shapedGen) Done() bool           { return s.g.Done() }
func (s shapedGen) Reset()               { s.g.Reset() }

// shaped is a value through a shaper: a number or a note name, or the pitches of an array one by one
func (r *Runtime) shaped(s shape.Shaper, v goja.Value) goja.Value {
	one := func(x goja.Value) goja.Value {
		p, err := pitchOf(x)
		if err != nil {
			panic(r.vm.NewTypeError(err.Error()))
		}
		y := s.Shape(p)
		if math.IsNaN(y) || math.IsInf(y, 0) {
			panic(r.vm.NewTypeError(fmt.Sprintf("shape: %v gave %v", p, y)))
		}
		return r.vm.ToValue(y)
	}
	if o, ok := v.(*goja.Object); ok && o.ClassName() == "Array" {
		vals := r.values(v)
		out := make([]any, len(vals))
		for i, x := range vals {
			out[i] = one(x)
		}
		return r.vm.NewArray(out...)
	}
	return one(v)
}

// fnShaper is a function of the script as a shaper
type fnShaper struct {
	r *Runtime
	f goja.Callable
}

func (f fnShaper) Shape(x float64) float64 {
	return f.r.must(f.f(goja.Undefined(), f.r.vm.ToValue(x))).ToFloat()
}

// shaper makes a shaper of a value of a script: a Shaper, or a function
func (r *Runtime) shaper(v goja.Value) shape.Shaper {
	if v != nil {
		if s, ok := v.Export().(*Shaper); ok {
			return s.s
		}
		if f, ok := goja.AssertFunction(v); ok {
			return fnShaper{r: r, f: f}
		}
	}
	panic(r.vm.NewTypeError("a shaper (from genny.shapers) or a function of a number"))
}

// safeTable is a table that never reads beyond its end: 0 to 1 over it (a table of a wave is a
// cycle, its end is its start again)
type safeTable struct {
	t    lookup.Table
	open bool // the values of a script: the end is the last value, not the start again
}

func (s safeTable) Shape(x float64) float64 {
	if s.open && x >= 1 {
		return s.t[len(s.t)-1]
	}
	return s.t.Shape(math.Max(0, math.Min(x, math.Nextafter(1, 0))))
}

// tables are the waves a table of shapers can be: a shaper reading one over 0 to 1
var tables = map[string]func(int) lookup.Table{
	"sine": lookup.NewSineTable, "uni sine": lookup.NewUniSineTable,
	"cosine": lookup.NewCosineTable, "uni cosine": lookup.NewUniCosineTable,
	"sawtooth": lookup.NewSawtoothTable, "uni sawtooth": lookup.NewUniSawtoothTable,
	"triangle": lookup.NewTriangleTable, "uni triangle": lookup.NewUniTriangleTable,
	"square": lookup.NewSquareTable, "uni square": lookup.NewUniSquareTable,
}

// shapersObject is genny.shapers: the shapers of genny
func (r *Runtime) shapersObject() *goja.Object {
	vm := r.vm
	o := vm.NewObject()
	fail := func(format string, args ...any) { panic(vm.NewTypeError(fmt.Sprintf(format, args...))) }
	num := func(call goja.FunctionCall, i int, def float64) float64 {
		if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
			return a.ToFloat()
		}
		return def
	}
	nonzero := func(what string, x float64) float64 {
		if x == 0 || math.IsNaN(x) {
			fail("%s: not 0", what)
		}
		return x
	}
	wrap := func(s shape.Shaper) goja.Value { return vm.ToValue(&Shaper{s: s, r: r}) }
	set := func(name string, f func(call goja.FunctionCall) shape.Shaper) {
		_ = o.Set(name, func(call goja.FunctionCall) goja.Value { return wrap(f(call)) })
	}
	list := func(call goja.FunctionCall, from int, what string) []shape.Shaper {
		args := call.Arguments[min(from, len(call.Arguments)):]
		if len(args) == 1 {
			if a, ok := args[0].(*goja.Object); ok && a.ClassName() == "Array" {
				args = r.values(args[0])
			}
		}
		if len(args) == 0 {
			fail("%s: it needs shapers", what)
		}
		out := make([]shape.Shaper, len(args))
		for i, a := range args {
			out[i] = r.shaper(a)
		}
		return out
	}

	set("add", func(c goja.FunctionCall) shape.Shaper { return add.New(num(c, 0, 0)) })
	set("mult", func(c goja.FunctionCall) shape.Shaper { return mult.New(num(c, 0, 1)) })
	set("mod", func(c goja.FunctionCall) shape.Shaper { return mod.New(nonzero("mod", num(c, 0, 1))) })
	set("ripple", func(c goja.FunctionCall) shape.Shaper { return ripple.New(nonzero("ripple", num(c, 0, 1))) })
	set("linear", func(c goja.FunctionCall) shape.Shaper { return linear.New(num(c, 0, 1), num(c, 1, 0)) })
	set("bipolar", func(goja.FunctionCall) shape.Shaper { return linear.NewBipolar() })
	set("unipolar", func(goja.FunctionCall) shape.Shaper { return linear.NewUnipolar() })
	set("abs", func(goja.FunctionCall) shape.Shaper { return function.NewAbs() })
	set("sin", func(goja.FunctionCall) shape.Shaper { return function.NewSin() })
	set("tri", func(goja.FunctionCall) shape.Shaper { return function.NewTri() })
	set("thru", func(goja.FunctionCall) shape.Shaper { return thru.New() })
	set("mirror", func(c goja.FunctionCall) shape.Shaper {
		lo, hi := num(c, 0, 0), num(c, 1, 1)
		if !(lo < hi) {
			fail("mirror: bottom below top")
		}
		return mirror.New(lo, hi)
	})
	set("pulse", func(c goja.FunctionCall) shape.Shaper { return pulse.New(num(c, 0, 0.5)) })
	set("quantize", func(c goja.FunctionCall) shape.Shaper { return quantize.New(nonzero("quantize", num(c, 0, 1))) })
	set("chebyshev", func(c goja.FunctionCall) shape.Shaper {
		h, ok := c.Argument(0).(*goja.Object)
		if !ok {
			fail("chebyshev: the levels of the harmonics, {1: 1, 3: 0.3}")
		}
		levels := map[int]float64{}
		for _, k := range h.Keys() {
			var n int
			if _, err := fmt.Sscanf(k, "%d", &n); err != nil || n < 1 || n > 64 {
				fail("chebyshev: %q is not a harmonic from 1 to 64", k)
			}
			levels[n] = h.Get(k).ToFloat()
		}
		return chebyshev.New(levels)
	})
	// clip("hard", low, high) or clip("soft", amount)
	set("clip", func(c goja.FunctionCall) shape.Shaper {
		switch m := c.Argument(0).String(); m {
		case "hard", "undefined":
			return clip.New(clip.Hard, num(c, 1, -1), num(c, 2, 1))
		case "soft":
			return clip.New(clip.Soft, nonzero("clip", num(c, 1, 2)))
		default:
			fail("clip: no %q (it is hard or soft)", m)
			return nil
		}
	})
	set("fn", func(c goja.FunctionCall) shape.Shaper { return r.shaper(c.Argument(0)) })
	// chain(text, macros): a chain written as text, as the shaper synth reads it (see
	// blocks/shapechain: "mult 3 | sin", "bipolar | mirror -0.5 0.5"); the macros m1 to m4 (0 to 1)
	// are 0.5 or the array given
	set("chain", func(c goja.FunctionCall) shape.Shaper {
		ch, err := shapechain.Parse(c.Argument(0).String())
		if err != nil {
			fail("chain: %v", err)
		}
		m := shapechain.Macros{0.5, 0.5, 0.5, 0.5}
		if a := c.Argument(1); !goja.IsUndefined(a) && !goja.IsNull(a) {
			for i, x := range r.values(a) {
				if i < len(m) {
					m[i] = math.Max(0, math.Min(x.ToFloat(), 1))
				}
			}
		}
		return ch.WithMacros(m)
	})
	// table(wave, size = 256) or table([values]): read over 0 to 1, between its values
	set("table", func(c goja.FunctionCall) shape.Shaper {
		a := c.Argument(0)
		if ao, ok := a.(*goja.Object); ok && ao.ClassName() == "Array" {
			vals := r.values(a)
			if len(vals) < 2 || len(vals) > 65536 {
				fail("table: 2 values or more")
			}
			t := make(lookup.Table, 0, len(vals)+1)
			for _, v := range vals {
				t = append(t, v.ToFloat())
			}
			return safeTable{t: t, open: true}
		}
		name := a.String()
		maker, ok := tables[name]
		if !ok {
			names := tableNames()
			fail("table: no %q (it is one of %s)", name, strings.Join(names, ", "))
		}
		n := int(num(c, 1, 256))
		if n < 2 || n > 65536 {
			fail("table: 2 to 65536 values")
		}
		return safeTable{t: maker(n)}
	})
	set("series", func(c goja.FunctionCall) shape.Shaper { return series.New(list(c, 0, "series")...) })
	set("parallel", func(c goja.FunctionCall) shape.Shaper { return parallel.New(list(c, 0, "parallel")...) })
	// mixer(shapers, levels): the shapers added, every one at a level (1 to start)
	set("mixer", func(c goja.FunctionCall) shape.Shaper {
		ss := list(goja.FunctionCall{Arguments: c.Arguments[:min(1, len(c.Arguments))]}, 0, "mixer")
		levels := make([]float64, len(ss))
		for i := range levels {
			levels[i] = 1
		}
		if l := c.Argument(1); !goja.IsUndefined(l) {
			ls := r.values(l)
			if len(ls) != len(ss) {
				fail("mixer: %d levels for %d shapers", len(ls), len(ss))
			}
			for i, x := range ls {
				levels[i] = x.ToFloat()
			}
		}
		return mixer.NewWithLevels(ss, levels)
	})
	// interp(index, ...shapers): between two neighbours, index 0 (the first) to 1 (the last)
	set("interp", func(c goja.FunctionCall) shape.Shaper {
		ip := interp.New(list(c, 1, "interp")...)
		ip.Index = math.Max(0, math.Min(num(c, 0, 0), 1))
		return ip
	})
	// route(index, ...shapers): the shaper at an index
	set("route", func(c goja.FunctionCall) shape.Shaper {
		ss := list(c, 1, "route")
		i := int(num(c, 0, 0))
		if i < 0 || i >= len(ss) {
			fail("route: an index from 0 to %d", len(ss)-1)
		}
		return route.New(i, ss...)
	})
	// the waves of synths, over a phase 0 to 1 (the emulations of genny)
	set("supersaw", func(c goja.FunctionCall) shape.Shaper {
		return supersaw.New(num(c, 0, 1.5), num(c, 1, 0.25), num(c, 2, 0.88))
	})
	set("hardsync", func(c goja.FunctionCall) shape.Shaper { return hardsync.New(num(c, 0, 2)) })
	set("softsync", func(c goja.FunctionCall) shape.Shaper { return softsync.NewTriangle(num(c, 0, 2)) })
	set("trimod", func(c goja.FunctionCall) shape.Shaper { return jp8000.NewTriMod(num(c, 0, 0.5)) })
	set("voyager", func(goja.FunctionCall) shape.Shaper { return voyager.NewSawtooth() })
	set("pwmod", func(goja.FunctionCall) shape.Shaper { return pwmod.New() })
	return o
}

// tableNames are the names of the tables
func tableNames() []string {
	names := make([]string, 0, len(tables))
	for n := range tables {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}
