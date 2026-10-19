package script

import (
	"fmt"
	"math"
	"slices"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/flatten"
	chaositer "github.com/almerlucke/genny/float/iter"
	"github.com/almerlucke/genny/float/iter/updaters/chaos"
	"github.com/almerlucke/genny/float/ramp"
	"github.com/almerlucke/genny/markov"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/transform"
	"github.com/almerlucke/genny/walk"
	"github.com/dop251/goja"
)

// Map is a generator of the values of this one through a function: gen.map(v => v + 12)
func (g *Gen) Map(f goja.Callable) *Gen {
	fn := func(v goja.Value) goja.Value { return g.r.must(f(goja.Undefined(), v)) }
	return &Gen{g: transform.New(g.g, fn), vm: g.vm, r: g.r}
}

// choice picks one of its values, by weight, every time
type choice struct {
	values []goja.Value
	sums   []float64 // the weights added up
}

func (c *choice) Generate() goja.Value {
	x := genny.Float64() * c.sums[len(c.sums)-1]
	i, _ := slices.BinarySearchFunc(c.sums, x, func(s, x float64) int {
		if s <= x {
			return -1
		}
		return 1
	})
	return c.values[min(i, len(c.values)-1)]
}
func (*choice) Continuous() bool { return true }
func (*choice) Done() bool       { return false }
func (*choice) Reset()           {}

// firstOf makes a generator of the first value of a vector generator
type firstOf struct{ g *chaositer.Iterator }

func (f firstOf) Generate() float64 { return f.g.Generate()[0] }
func (firstOf) Continuous() bool    { return true }
func (firstOf) Done() bool          { return false }
func (f firstOf) Reset()            { f.g.Reset() }

// moreGenny adds the generators of genny that are not a plain list: weighted choices, chains,
// walks, chaos, ramps, rhythms
func (r *Runtime) moreGenny(set func(string, func(goja.FunctionCall) goja.Value), float func(genny.Generator[float64]) goja.Value) {
	vm := r.vm
	fail := func(format string, args ...any) { panic(vm.NewTypeError(fmt.Sprintf(format, args...))) }
	number := func(call goja.FunctionCall, i int, def float64) float64 {
		if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
			return a.ToFloat()
		}
		return def
	}

	// choice(values, weights): any one every time, the more weight the more often
	set("choice", func(call goja.FunctionCall) goja.Value {
		vals := r.values(call.Argument(0))
		if len(vals) == 0 {
			fail("choice: it needs values")
		}
		weights := make([]float64, len(vals))
		for i := range weights {
			weights[i] = 1
		}
		if w := call.Argument(1); !goja.IsUndefined(w) && !goja.IsNull(w) {
			ws := r.values(w)
			if len(ws) != len(vals) {
				fail("choice: %d weights for %d values", len(ws), len(vals))
			}
			for i, x := range ws {
				if weights[i] = x.ToFloat(); weights[i] < 0 || math.IsNaN(weights[i]) {
					fail("choice: a weight is 0 or more")
				}
			}
		}
		sums, total := make([]float64, len(weights)), 0.0
		for i, w := range weights {
			total += w
			sums[i] = total
		}
		if total <= 0 {
			fail("choice: the weights are all 0")
		}
		return r.wrap(&choice{values: vals, sums: sums})
	})

	// markov(table, start): a chain of states. table is {state: {next: weight, ...}, ...}; every
	// state is a value ("C3", 60 as "60": note names and numbers work in note()), start the first
	set("markov", func(call goja.FunctionCall) goja.Value {
		t, ok := call.Argument(0).(*goja.Object)
		if !ok {
			fail("markov: a table of {state: {next: weight}}")
		}
		keys := t.Keys()
		if len(keys) == 0 {
			fail("markov: no states")
		}
		states := map[string]*markov.ProbabilityState[goja.Value]{}
		for _, k := range keys {
			states[k] = markov.NewProbabilityState(vm.ToValue(k))
		}
		for _, k := range keys {
			row, ok := t.Get(k).(*goja.Object)
			if !ok || len(row.Keys()) == 0 {
				fail("markov: %q needs {next: weight, ...}", k)
			}
			var args []any
			total := 0.0
			for _, n := range row.Keys() {
				to, ok := states[n]
				if !ok {
					fail("markov: %q goes to %q, which is not a state", k, n)
				}
				w := row.Get(n).ToFloat()
				if w < 0 || math.IsNaN(w) {
					fail("markov: a weight is 0 or more (%q to %q)", k, n)
				}
				total += w
				args = append(args, to, w)
			}
			if total <= 0 {
				fail("markov: %q has weights of 0 only", k)
			}
			states[k].SetProbabilities(args...)
		}
		start := keys[0]
		if a := call.Argument(1); !goja.IsUndefined(a) && !goja.IsNull(a) {
			start = a.String()
			if _, ok := states[start]; !ok {
				fail("markov: no state %q to start in", start)
			}
		}
		return r.wrap(markov.New[goja.Value](states[start], 1))
	})

	// walk(dims, values): a random walk through a grid, a step along one dimension at a time;
	// values is a list laid out first dimension first, dims e.g. [4, 3]
	set("walk", func(call goja.FunctionCall) goja.Value {
		dims := []int{}
		for _, d := range r.values(call.Argument(0)) {
			if n := int(d.ToInteger()); n >= 2 {
				dims = append(dims, n)
			} else {
				fail("walk: every dimension has 2 values or more")
			}
		}
		if len(dims) == 0 {
			fail("walk: dimensions, e.g. [4, 3]")
		}
		vals := r.values(call.Argument(1))
		n := 1
		for _, d := range dims {
			n *= d
		}
		if len(vals) != n {
			fail("walk: %d values for a grid of %d", len(vals), n)
		}
		return r.wrap(walk.New[goja.Value](walk.NewMatrix(dims, vals)))
	})

	// logistic(a = 3.9, x0 = 0.5): the logistic map, 0 to 1: a above 3.57 is chaos
	set("logistic", func(call goja.FunctionCall) goja.Value {
		return float(firstOf{chaositer.New([]float64{number(call, 1, 0.5)}, chaos.NewVerhulst(number(call, 0, 3.9)))})
	})
	// henon(a = 1.4, b = 0.3): the x of the Hénon map, about -1.3 to 1.3
	set("henon", func(call goja.FunctionCall) goja.Value {
		return float(firstOf{chaositer.New([]float64{0, 0}, chaos.NewHenon(number(call, 0, 1.4), number(call, 1, 0.3)))})
	})

	// ramp(steps, from, to, curve = 1, loop = false): steps values from one to the other, along
	// a curve (1 straight, above 1 slow to start); then it ends, or loops
	set("ramp", func(call goja.FunctionCall) goja.Value {
		steps := int(number(call, 0, 0))
		if steps < 2 {
			fail("ramp: 2 steps or more")
		}
		curve := number(call, 3, 1)
		if flag(call, 4, false) {
			return float(ramp.NewForever(steps, number(call, 1, 0), number(call, 2, 1), curve).MakeContinuous(true))
		}
		return float(ramp.New(steps, number(call, 1, 0), number(call, 2, 1), curve))
	})

	// euclid(hits, steps, rotate = 0): true on hits spread evenly over steps (3 of 8: x . . x . . x .),
	// false between, looping
	set("euclid", func(call goja.FunctionCall) goja.Value {
		hits, steps := int(number(call, 0, 0)), int(number(call, 1, 0))
		if steps < 1 || steps > 1024 || hits < 0 || hits > steps {
			fail("euclid: hits from 0 to steps, steps from 1 to 1024")
		}
		rot := int(number(call, 2, 0))
		vals := make([]goja.Value, steps)
		for i := range vals {
			j := ((i+rot)%steps + steps) % steps
			vals[i] = vm.ToValue(j*hits%steps < hits)
		}
		return r.wrap(sequence.NewLoop(vals...))
	})

	// flatten(gen): a generator of arrays (chords, bars) as their values one by one
	set("flatten", func(call goja.FunctionCall) goja.Value {
		g := r.gen(call.Argument(0))
		arrays := mapped2{g: g, vm: vm, r: r}
		return r.wrap(flatten.New[goja.Value](arrays))
	})

	// shape(gen, shaper): a generator through a shaper (also gen.shape(s)); r.shapersObject has them
	set("shape", func(call goja.FunctionCall) goja.Value {
		return r.wrap((&Gen{g: r.gen(call.Argument(0)), vm: vm, r: r}).Shape(call.Argument(1)).g)
	})
	// map(gen, f): the values of a generator through a function (also gen.map(f))
	set("map", func(call goja.FunctionCall) goja.Value {
		f, ok := goja.AssertFunction(call.Argument(1))
		if !ok {
			fail("map: a function")
		}
		return r.wrap((&Gen{g: r.gen(call.Argument(0)), vm: vm, r: r}).Map(f).g)
	})
}

// mapped2 is a generator of arrays of values (of a generator of arrays of a script)
type mapped2 struct {
	g  genny.Generator[goja.Value]
	vm *goja.Runtime
	r  *Runtime
}

func (m mapped2) Generate() []goja.Value {
	return m.r.values(m.g.Generate())
}
func (m mapped2) Continuous() bool { return m.g.Continuous() }
func (m mapped2) Done() bool       { return m.g.Done() }
func (m mapped2) Reset()           { m.g.Reset() }
