package script

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/and"
	"github.com/almerlucke/genny/arpeggio"
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/combine"
	"github.com/almerlucke/genny/constant"
	"github.com/almerlucke/genny/continuous"
	"github.com/almerlucke/genny/float/rand"
	"github.com/almerlucke/genny/float/rwalk"
	"github.com/almerlucke/genny/or"
	"github.com/almerlucke/genny/repeat"
	"github.com/almerlucke/genny/sequence"
	"github.com/dop251/goja"
)

// Gen is a genny generator in a script: gen.next(), gen.done(), gen.reset(), gen.take(n). Any of
// the generators of genny.* takes others: a Gen, an array (a sequence of it, looping), a function
// (called for every value), a function* (its values, started over on a reset) or a value (always
// it)
type Gen struct {
	g  genny.Generator[goja.Value]
	vm *goja.Runtime
	r  *Runtime
}

// Generator is the generator in Go (genny), to use it from Go
func (g *Gen) Generator() genny.Generator[goja.Value] { return g.g }

// Next is the next value
func (g *Gen) Next() goja.Value { return g.g.Generate() }

// Done tells if it has ended (one that does not loop)
func (g *Gen) Done() bool { return g.g.Done() }

// Reset starts it over
func (g *Gen) Reset() { g.g.Reset() }

// Take is its next n values
func (g *Gen) Take(n int) []goja.Value {
	out := make([]goja.Value, 0, max(0, min(n, 4096)))
	for range cap(out) {
		out = append(out, g.g.Generate())
	}
	return out
}

// mapped is a generator of another type, its values made values of the script
type mapped[T any] struct {
	genny.Generator[T]
	to func(T) goja.Value
}

func (m mapped[T]) Generate() goja.Value { return m.to(m.Generator.Generate()) }

// fn calls a function for every value
type fn struct {
	r *Runtime
	f goja.Callable
}

func (f fn) Generate() goja.Value { return f.r.must(f.f(goja.Undefined())) }
func (fn) Continuous() bool       { return true }
func (fn) Done() bool             { return false }
func (fn) Reset()                 {}

// iter takes the values of an iterator: of a function* (made again on a reset) or of any object
// with next(). It is a value ahead: done right after its last value (as genny's are), not a call
// later
type iter struct {
	r     *Runtime
	start func() *goja.Object // nil: it cannot start over
	it    *goja.Object
	next  goja.Callable
	ahead goja.Value // the value it gives next
	done  bool       // no value ahead: it has ended
	last  goja.Value
}

func newIter(r *Runtime, start func() *goja.Object, it *goja.Object) *iter {
	i := &iter{r: r, start: start, last: goja.Undefined()}
	i.set(it)
	return i
}

func (i *iter) set(it *goja.Object) {
	i.it = it
	i.next, _ = goja.AssertFunction(it.Get("next"))
	if i.next == nil {
		panic(i.r.vm.NewTypeError("not an iterator: it has no next()"))
	}
	i.fetch()
}

// fetch takes the value ahead
func (i *iter) fetch() {
	i.done = true
	o, ok := i.r.must(i.next(i.it)).(*goja.Object)
	if !ok {
		return
	}
	if d := o.Get("done"); d != nil && d.ToBoolean() {
		return
	}
	i.done, i.ahead = false, o.Get("value")
	if i.ahead == nil {
		i.ahead = goja.Undefined()
	}
}

func (i *iter) Generate() goja.Value {
	if i.done {
		return i.last
	}
	i.last = i.ahead
	i.fetch()
	return i.last
}

func (*iter) Continuous() bool { return false }
func (i *iter) Done() bool     { return i.done }
func (i *iter) Reset() {
	if i.start != nil {
		i.set(i.start())
	}
}

// must returns the value of a call into a script, or throws its error on (from a Go function
// called by the script, up through it). One stopped (too long) is let be: the script stops on
func (r *Runtime) must(v goja.Value, err error) goja.Value {
	var ex *goja.Exception
	var ie *goja.InterruptedError
	switch {
	case err == nil:
		return v
	case errors.As(err, &ie):
		return goja.Undefined()
	case errors.As(err, &ex):
		panic(ex)
	default:
		panic(r.vm.NewGoError(err))
	}
}

// values is the values of an array of a script
func (r *Runtime) values(v goja.Value) []goja.Value {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	o, ok := v.(*goja.Object)
	if !ok || o.ClassName() != "Array" {
		return []goja.Value{v}
	}
	n := int(o.Get("length").ToInteger())
	out := make([]goja.Value, n)
	for i := range out {
		out[i] = o.Get(strconv.Itoa(i))
	}
	return out
}

// gen makes a generator of a value of a script: a Gen, an array, a function, a function*, an
// iterator or a value
func (r *Runtime) gen(v goja.Value) genny.Generator[goja.Value] {
	if v == nil {
		return constant.New(goja.Undefined())
	}
	if g, ok := v.Export().(*Gen); ok {
		return g.g
	}
	o, ok := v.(*goja.Object)
	if !ok {
		return constant.New(v)
	}
	if o.ClassName() == "Array" {
		return sequence.NewLoop(r.values(v)...)
	}
	if f, ok := goja.AssertFunction(v); ok {
		if isGeneratorFunction(o) {
			start := func() *goja.Object { return r.must(f(goja.Undefined())).ToObject(r.vm) }
			return newIter(r, start, start())
		}
		return fn{r: r, f: f}
	}
	if _, ok := goja.AssertFunction(o.Get("next")); ok {
		return newIter(r, nil, o)
	}
	return constant.New(v)
}

// isGeneratorFunction tells a function* from a function
func isGeneratorFunction(o *goja.Object) bool {
	c, ok := o.Get("constructor").(*goja.Object)
	return ok && c.Get("name") != nil && c.Get("name").String() == "GeneratorFunction"
}

// gens makes generators of the values of a script (arguments, or the one array of them)
func (r *Runtime) gens(args []goja.Value) []genny.Generator[goja.Value] {
	if len(args) == 1 {
		if o, ok := args[0].(*goja.Object); ok && o.ClassName() == "Array" {
			args = r.values(args[0])
		}
	}
	if len(args) == 0 {
		panic(r.vm.NewTypeError("it needs at least one generator"))
	}
	gs := make([]genny.Generator[goja.Value], len(args))
	for i, a := range args {
		gs[i] = r.gen(a)
	}
	return gs
}

// wrap makes a generator a Gen of the script
func (r *Runtime) wrap(g genny.Generator[goja.Value]) goja.Value {
	return r.vm.ToValue(&Gen{g: g, vm: r.vm, r: r})
}

// option is an option of a string (by its name), def if it is not given
func option[T any](vm *goja.Runtime, call goja.FunctionCall, i int, what string, names map[string]T, def T) T {
	a := call.Argument(i)
	if goja.IsUndefined(a) || goja.IsNull(a) {
		return def
	}
	if v, ok := names[a.String()]; ok {
		return v
	}
	keys := slices.Sorted(maps.Keys(names))
	panic(vm.NewTypeError(fmt.Sprintf("%s: no %q (it is one of %s)", what, a.String(), strings.Join(keys, ", "))))
}

// flag is an option that is true or false (by default def)
func flag(call goja.FunctionCall, i int, def bool) bool {
	a := call.Argument(i)
	if goja.IsUndefined(a) || goja.IsNull(a) {
		return def
	}
	return a.ToBoolean()
}

// gennyObject is genny in a script
func (r *Runtime) gennyObject() *goja.Object {
	vm := r.vm
	o := vm.NewObject()
	set := func(name string, f func(call goja.FunctionCall) goja.Value) { _ = o.Set(name, f) }
	float := func(g genny.Generator[float64]) goja.Value {
		return r.wrap(mapped[float64]{Generator: g, to: func(f float64) goja.Value { return vm.ToValue(f) }})
	}

	// sequence(values, loop = true): the values in order
	set("sequence", func(call goja.FunctionCall) goja.Value {
		vals := r.values(call.Argument(0))
		if flag(call, 1, true) {
			return r.wrap(sequence.NewLoop(vals...))
		}
		return r.wrap(sequence.New(vals...))
	})
	// bucket(values, mode = "shuffle", loop = true): shuffled, every one once before again; or
	// "random", any one every time
	set("bucket", func(call goja.FunctionCall) goja.Value {
		vals := r.values(call.Argument(0))
		mode := option(vm, call, 1, "bucket", map[string]bucket.Mode{"shuffle": bucket.Indexed, "random": bucket.Random}, bucket.Indexed)
		if flag(call, 2, true) {
			return r.wrap(bucket.NewLoop(vals...).SetMode(mode))
		}
		return r.wrap(bucket.New(vals...).SetMode(mode))
	})
	// arpeggio(values, mode = "up", mirror = "none", reverse = false): "up", "converge",
	// "alternate", "random"; mirrored "none", "exclusive", "inclusive"
	set("arpeggio", func(call goja.FunctionCall) goja.Value {
		vals := r.values(call.Argument(0))
		mode := option(vm, call, 1, "arpeggio", map[string]arpeggio.Mode{"up": arpeggio.Up, "converge": arpeggio.Converge, "alternate": arpeggio.Alternate, "random": arpeggio.Random}, arpeggio.Up)
		mirror := option(vm, call, 2, "arpeggio mirror", map[string]arpeggio.MirrorMode{"none": arpeggio.None, "exclusive": arpeggio.Exclusive, "inclusive": arpeggio.Inclusive}, arpeggio.None)
		return r.wrap(arpeggio.NewLoop(vals, mode, mirror, flag(call, 3, false)))
	})
	// repeat(gen, n[, n2]): n values of a generator that loops, or one that ends played n times
	// over (n from n to n2 at random, every time it starts); then it ends
	set("repeat", func(call goja.FunctionCall) goja.Value {
		g := r.gen(call.Argument(0))
		n := int(call.Argument(1).ToInteger())
		if n2 := call.Argument(2); !goja.IsUndefined(n2) {
			return r.wrap(repeat.NewRand(g, max(1, n), max(1, int(n2.ToInteger()))))
		}
		return r.wrap(repeat.New(g, max(1, n)))
	})
	// and(...gens): one after the other (each to its end), looping
	set("and", func(call goja.FunctionCall) goja.Value { return r.wrap(and.NewLoop(r.gens(call.Arguments)...)) })
	// or(mode, ...gens): one of them every time ("random", or "shuffle": each once before again)
	set("or", func(call goja.FunctionCall) goja.Value {
		mode := option(vm, call, 0, "or", map[string]or.Mode{"random": or.Random, "shuffle": or.Indexed}, or.Random)
		return r.wrap(or.NewLoop(mode, r.gens(call.Arguments[1:])...))
	})
	// combine(...gens): a value of each, together (an array)
	set("combine", func(call goja.FunctionCall) goja.Value {
		return r.wrap(mapped[[]goja.Value]{Generator: combine.New(r.gens(call.Arguments)...), to: func(vs []goja.Value) goja.Value {
			return vm.NewArray(func() []any {
				a := make([]any, len(vs))
				for i, v := range vs {
					a[i] = v
				}
				return a
			}()...)
		}})
	})
	// continuous(gen): one that ends started over every time
	set("continuous", func(call goja.FunctionCall) goja.Value { return r.wrap(continuous.New(r.gen(call.Argument(0)))) })
	// constant(v): always v
	set("constant", func(call goja.FunctionCall) goja.Value { return r.wrap(constant.New(call.Argument(0))) })
	// rand(lo = 0, hi = 1): a random number
	set("rand", func(call goja.FunctionCall) goja.Value {
		lo, hi := 0.0, 1.0
		if a := call.Argument(0); !goja.IsUndefined(a) {
			lo = a.ToFloat()
		}
		if a := call.Argument(1); !goja.IsUndefined(a) {
			hi = a.ToFloat()
		}
		return float(rand.New(lo, hi))
	})
	// rwalk(origin, min, max, step = 0.1, mode = "bounce"): a random walk, steps of up to a part of
	// the range; at the edges "clip", "bounce" or "jump" (back to the origin)
	set("rwalk", func(call goja.FunctionCall) goja.Value {
		step := 0.1
		if a := call.Argument(3); !goja.IsUndefined(a) {
			step = a.ToFloat()
		}
		mode := option(vm, call, 4, "rwalk", map[string]rwalk.Mode{"clip": rwalk.Clip, "bounce": rwalk.Bounce, "jump": rwalk.Jump}, rwalk.Bounce)
		return float(rwalk.New(call.Argument(0).ToFloat(), call.Argument(1).ToFloat(), call.Argument(2).ToFloat(), step, mode))
	})
	r.moreGenny(set, float)
	// from(v): a generator of anything (a function, a function*, an iterator, an array)
	set("from", func(call goja.FunctionCall) goja.Value { return r.wrap(r.gen(call.Argument(0))) })
	_ = o.Set("shapers", r.shapersObject())
	return o
}
