package script

import (
	"fmt"
	"slices"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/choose"
	"github.com/almerlucke/genny/constrain"
	"github.com/almerlucke/genny/float/dist"
	chaositer "github.com/almerlucke/genny/float/iter"
	"github.com/almerlucke/genny/float/iter/updaters/chaos"
	"github.com/almerlucke/genny/float/lfo"
	"github.com/almerlucke/genny/float/lissajous"
	"github.com/almerlucke/genny/float/noise"
	"github.com/almerlucke/genny/float/tendency"
	"github.com/almerlucke/genny/grammar"
	"github.com/almerlucke/genny/listtools"
	"github.com/almerlucke/genny/ngram"
	"github.com/almerlucke/genny/sieve"
	"github.com/dop251/goja"
)

// signalsGenny adds the generators of numbers over time and of chance: LFOs, noise, distributions, chaos, tendency masks,
// sieves, rules that constrain, grammars and the tools for lists
func (r *Runtime) signalsGenny(set func(string, func(goja.FunctionCall) goja.Value)) {
	vm := r.vm
	fail := func(format string, args ...any) { panic(vm.NewTypeError(fmt.Sprintf(format, args...))) }
	number := func(call goja.FunctionCall, i int, def float64) float64 {
		if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
			return a.ToFloat()
		}
		return def
	}
	has := func(call goja.FunctionCall, i int) bool {
		a := call.Argument(i)
		return !goja.IsUndefined(a) && !goja.IsNull(a)
	}
	// setg is set, but what genny panics at (an argument that can never work) is a TypeError of the script
	setg := func(name string, f func(goja.FunctionCall) goja.Value) {
		set(name, func(call goja.FunctionCall) (v goja.Value) {
			defer func() {
				if e := recover(); e != nil {
					switch e.(type) {
					case *goja.Object, *goja.Exception:
						panic(e)
					}
					fail("%s: %v", name, e)
				}
			}()
			return f(call)
		})
	}

	// --- modulation ---

	// lfo(shape = "sine", period = 64, { pulseWidth, unipolar }): a wave, one cycle in period values: shape "sine", "triangle",
	// "saw", "square", "sampleHold" or "smoothRandom"; -1 to 1 (0 to 1 when unipolar)
	setg("lfo", func(call goja.FunctionCall) goja.Value {
		shape := option(vm, call, 0, "lfo", map[string]lfo.Shape{"sine": lfo.Sine, "triangle": lfo.Triangle, "saw": lfo.Saw,
			"square": lfo.Square, "sampleHold": lfo.SampleHold, "smoothRandom": lfo.SmoothRandom}, lfo.Sine)
		l := lfo.New(shape, 1, max(1e-6, number(call, 1, 64)))
		if o := call.Argument(2); !goja.IsUndefined(o) {
			if pw := opt(o, "pulseWidth"); !goja.IsUndefined(pw) {
				l.SetPulseWidth(pw.ToFloat())
			}
			l.SetUnipolar(opt(o, "unipolar").ToBoolean())
		}
		return wrapAs[float64](r, l)
	})
	// slew(gen, rise, fall = rise): a number generator that cannot move faster than rise (up) and fall (down) per value
	setg("slew", func(call goja.FunctionCall) goja.Value {
		rise := number(call, 1, 0.1)
		return wrapAs[float64](r, lfo.NewSlew(r.floats(call.Argument(0)), rise, number(call, 2, rise)))
	})
	// noise(step = 0.1, octaves = 1, { unipolar }): smooth noise that drifts: step is how far it moves along the noise per value
	setg("noise", func(call goja.FunctionCall) goja.Value {
		n := noise.New(number(call, 0, 0.1), int(number(call, 1, 1)))
		if o := call.Argument(2); !goja.IsUndefined(o) {
			n.SetUnipolar(opt(o, "unipolar").ToBoolean())
		}
		return wrapAs[float64](r, n)
	})
	// lissajous(a = 3, b = 2, delta = 0, steps = 64): a curve of two numbers, [x, y], one round in steps values
	setg("lissajous", func(call goja.FunctionCall) goja.Value {
		return r.wrap(mapped[[]float64]{Generator: lissajousCopy{lissajous.New(number(call, 0, 3), number(call, 1, 2), number(call, 2, 0), int(number(call, 3, 64)))}, to: r.array})
	})
	// lorenz(sigma = 10, rho = 28, beta = 8/3, dt = 0.01), rossler(a = 0.2, b = 0.2, c = 5.7, dt = 0.05): an attractor, [x, y, z]
	setg("lorenz", func(call goja.FunctionCall) goja.Value {
		it := chaositer.New([]float64{1, 1, 1}, chaos.NewLorenz(number(call, 0, 10), number(call, 1, 28), number(call, 2, 8.0/3), number(call, 3, 0.01)))
		return r.wrap(mapped[[]float64]{Generator: vectors{it}, to: r.array})
	})
	setg("rossler", func(call goja.FunctionCall) goja.Value {
		it := chaositer.New([]float64{1, 1, 1}, chaos.NewRossler(number(call, 0, 0.2), number(call, 1, 0.2), number(call, 2, 5.7), number(call, 3, 0.05)))
		return r.wrap(mapped[[]float64]{Generator: vectors{it}, to: r.array})
	})
	// duffing(), pendulum(), standardMap(k = 1): other attractors, [x, y] (or more)
	setg("duffing", func(goja.FunctionCall) goja.Value {
		return r.wrap(mapped[[]float64]{Generator: vectors{chaositer.New([]float64{0.1, 0.1}, chaos.NewDuffing())}, to: r.array})
	})
	setg("pendulum", func(goja.FunctionCall) goja.Value {
		return r.wrap(mapped[[]float64]{Generator: vectors{chaositer.New([]float64{1, 0}, chaos.NewPendulum())}, to: r.array})
	})
	setg("standardMap", func(call goja.FunctionCall) goja.Value {
		return r.wrap(mapped[[]float64]{Generator: vectors{chaositer.New([]float64{0.5, 0.5}, chaos.NewStandardMap(number(call, 0, 1)))}, to: r.array})
	})

	// --- chance ---

	// gaussian(mean = 0, sd = 1, lo, hi), exponential(mean = 1, lo, hi), beta(a = 0.5, b = 0.5, lo = 0, hi = 1),
	// cauchy(location = 0, scale = 1, lo, hi), weibull(threshold = 0, spread = 1, shape = 1, lo, hi): random numbers that
	// fall by a distribution; with lo and hi they stay between them
	setg("gaussian", func(call goja.FunctionCall) goja.Value {
		g := dist.NewGaussian(number(call, 0, 0), number(call, 1, 1))
		if has(call, 3) {
			g.Clamp(call.Argument(2).ToFloat(), call.Argument(3).ToFloat())
		}
		return wrapAs[float64](r, g)
	})
	setg("exponential", func(call goja.FunctionCall) goja.Value {
		g := dist.NewExponential(number(call, 0, 1))
		if has(call, 2) {
			g.Clamp(call.Argument(1).ToFloat(), call.Argument(2).ToFloat())
		}
		return wrapAs[float64](r, g)
	})
	setg("beta", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, dist.NewBeta(number(call, 0, 0.5), number(call, 1, 0.5)).Scale(number(call, 2, 0), number(call, 3, 1)))
	})
	setg("cauchy", func(call goja.FunctionCall) goja.Value {
		g := dist.NewCauchy(number(call, 0, 0), number(call, 1, 1))
		if has(call, 3) {
			g.Clamp(call.Argument(2).ToFloat(), call.Argument(3).ToFloat())
		}
		return wrapAs[float64](r, g)
	})
	setg("weibull", func(call goja.FunctionCall) goja.Value {
		g := dist.NewWeibull(number(call, 0, 0), number(call, 1, 1), number(call, 2, 1))
		if has(call, 4) {
			g.Clamp(call.Argument(3).ToFloat(), call.Argument(4).ToFloat())
		}
		return wrapAs[float64](r, g)
	})
	// pink(rows = 8): pink noise (-1 to 1); lehmer(seed): a pseudo random sequence that is the same every time
	setg("pink", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, dist.NewPink(int(number(call, 0, 8))))
	})
	setg("lehmer", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, dist.NewLehmer(uint64(max(0, number(call, 0, 1)))))
	})
	// pick(items, distribution = "uniform", ...params): an item at random, by a curve over the list (the first items most with
	// "linear" or "exponential", the middle ones with "gaussian"): "uniform", "linear", "exponential"(rate), "gaussian"(mean,
	// sd), "beta"(a, b), "cauchy"(center, scale), "weibull"(spread, shape), "triangle"(mode), "ratio"(...weights)
	setg("pick", func(call goja.FunctionCall) goja.Value {
		kind := "uniform"
		if has(call, 1) {
			kind = call.Argument(1).String()
		}
		p := func(i int, def float64) float64 { return number(call, 2+i, def) }
		n := float64(len(r.values(call.Argument(0))))
		var d choose.Distribution
		switch kind {
		case "uniform":
			d = choose.Uniform()
		case "linear":
			d = choose.Linear()
		case "exponential":
			d = choose.Exponential(p(0, 3))
		case "gaussian":
			d = choose.Gaussian(p(0, (n-1)/2), p(1, max(1, n/6))) // in items: the middle one, a sixth of the list
		case "beta":
			d = choose.Beta(p(0, 0.5), p(1, 0.5))
		case "cauchy":
			d = choose.Cauchy(p(0, 0.5), p(1, 0.1))
		case "weibull":
			d = choose.Weibull(p(0, 1), p(1, 1))
		case "triangle":
			d = choose.Triangle(p(0, 0.5))
		case "ratio":
			var w []int
			for _, a := range call.Arguments[2:] {
				w = append(w, int(a.ToInteger()))
			}
			d = choose.Ratio(w...)
		default:
			fail("pick: no distribution %q (uniform, linear, exponential, gaussian, beta, cauchy, weibull, triangle, ratio)", kind)
		}
		return r.wrap(choose.NewIndex(r.values(call.Argument(0)), d))
	})
	// shifting(items, every, ...weightSets): an item by weights, the weights changing every `every` values to the next set
	setg("shifting", func(call goja.FunctionCall) goja.Value {
		var sets [][]float64
		for _, a := range call.Arguments[2:] {
			sets = append(sets, r.numbers(a))
		}
		return r.wrap(choose.NewChangingWeights(r.values(call.Argument(0)), int(call.Argument(1).ToInteger()), sets...))
	})

	// --- tendency masks ---

	// tendency(segments, { loop, beta }): numbers between two bounds that move: segments [{ n, from: [lo, hi], to: [lo, hi] }],
	// for n values the lower bound goes from from[0] to to[0], the upper from from[1] to to[1]. beta: [a, b] draws toward the
	// edges (a, b < 1) or the middle (> 1) of the band
	setg("tendency", func(call goja.FunctionCall) goja.Value {
		m := r.mask(call.Argument(0), call.Argument(1))
		if b := r.numbers(opt(call.Argument(1), "beta")); len(b) == 2 {
			return wrapAs[float64](r, tendency.NewBetaValue(m, b[0], b[1]))
		}
		return wrapAs[float64](r, tendency.NewValue(m))
	})
	// tendencyChoice(items, segments, { loop, beta }): items picked by a band that moves over the list (0 to 1)
	setg("tendencyChoice", func(call goja.FunctionCall) goja.Value {
		m := r.mask(call.Argument(1), call.Argument(2))
		items := r.values(call.Argument(0))
		if b := r.numbers(opt(call.Argument(2), "beta")); len(b) == 2 {
			return r.wrap(tendency.NewBetaChoice(m, items, b[0], b[1]))
		}
		return r.wrap(tendency.NewChoice(m, items))
	})
	// spray(values, spread): the values moved at random, up to spread either way (an array)
	setg("spray", func(call goja.FunctionCall) goja.Value {
		return r.array(tendency.Spray(r.numbers(call.Argument(0)), call.Argument(1).ToFloat()))
	})

	// --- sieves ---

	// sieve(mod, start, lo, hi): the members of a residue class from lo to hi, an object: values(), contains(x), union(s),
	// intersection(s), difference(s), complement(), gen(loop = true), gate(length), scale(base)
	setg("sieve", func(call goja.FunctionCall) goja.Value {
		return r.sieveObject(sieve.New(int(call.Argument(0).ToInteger()), int(call.Argument(1).ToInteger()), int(call.Argument(2).ToInteger()), int(call.Argument(3).ToInteger())))
	})
	// sieveOf(lo, hi, values): a sieve of the values between lo and hi
	setg("sieveOf", func(call goja.FunctionCall) goja.Value {
		return r.sieveObject(sieve.Of(int(call.Argument(0).ToInteger()), int(call.Argument(1).ToInteger()), r.integers(call.Argument(2))...))
	})
	// funnel(values, allowed): each value moved to the nearest of the allowed values (an array)
	setg("funnel", func(call goja.FunctionCall) goja.Value {
		return r.array(sieve.Funnel(r.numbers(call.Argument(0)), r.numbers(call.Argument(1))))
	})

	// --- rules that constrain a generator of numbers ---

	// allowIntervals(gen, intervals, { stop, pitchClass }), blockIntervals(gen, intervals): redraw until the step from the
	// last value is (is not) one of the intervals; stop: tries before it gives up (default 100)
	setg("allowIntervals", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, constrain.AllowInterval(r.floats(call.Argument(0)), r.integers(call.Argument(1)), r.constraints(call.Argument(2))...))
	})
	setg("blockIntervals", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, constrain.BlockInterval(r.floats(call.Argument(0)), r.integers(call.Argument(1)), r.constraints(call.Argument(2))...))
	})
	// noRepeats(gen, { stop }): never the same number twice in a row; noDuplicates(gen, n, { stop }): no number twice in a group of n
	setg("noRepeats", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, constrain.RemoveRepetitions(r.floats(call.Argument(0)), r.constraints(call.Argument(1))...))
	})
	setg("noDuplicates", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, constrain.Duplicates(r.floats(call.Argument(0)), int(call.Argument(1).ToInteger()), r.constraints(call.Argument(2))...))
	})
	// intervalWalk(lo, hi, intervals): a melody (whole numbers from lo to hi) that moves by the intervals only (up or down, at random): [1, 2, 4]
	setg("intervalWalk", func(call goja.FunctionCall) goja.Value {
		return wrapAs[int](r, constrain.RandomIntervals(int(call.Argument(0).ToInteger()), int(call.Argument(1).ToInteger()), r.integers(call.Argument(2))...))
	})

	// --- learning, rules and lists ---

	// learn(values, order = 2): a chain that learned the style of an example (numbers) and plays new music in it
	setg("learn", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, ngram.Learn(r.numbers(call.Argument(0)), int(number(call, 1, 2))))
	})
	// grammar(rules, start, depth = 4): a phrase written by rewriting, every cycle a new one: rules { S: [[1, ["A", "S"]], ...] }
	// give for each name its alternatives [weight, symbols]; a symbol { rule: "A" } is another rule, any other is a value
	setg("grammar", func(call goja.FunctionCall) goja.Value {
		g := grammar.New[goja.Value]()
		rules, ok := call.Argument(0).(*goja.Object)
		if !ok {
			fail("grammar: rules, e.g. { S: [[1, [60, { rule: \"S\" }]], [1, [67]]] }")
		}
		for _, name := range rules.Keys() {
			var alts []grammar.Alt[goja.Value]
			for _, alt := range r.values(rules.Get(name)) {
				av := r.values(alt)
				if len(av) != 2 {
					fail("grammar: an alternative of %q is [weight, symbols]", name)
				}
				var syms []grammar.Symbol[goja.Value]
				for _, s := range r.values(av[1]) {
					if ref := opt(s, "rule"); !goja.IsUndefined(ref) {
						syms = append(syms, grammar.Ref[goja.Value](ref.String()))
					} else {
						syms = append(syms, grammar.Term(s))
					}
				}
				alts = append(alts, grammar.Choice(av[0].ToFloat(), syms...))
			}
			g.Rule(name, alts...)
		}
		start := call.Argument(1).String()
		if err := g.Check(start); err != nil {
			fail("grammar: %v", err)
		}
		return r.wrap(grammar.NewPlayer(g, start, int(number(call, 2, 4))))
	})
	// jump(values, chunk, step, leap): a list read in jumps: chunk values, then a leap further on
	setg("jump", func(call goja.FunctionCall) goja.Value {
		return r.wrap(listtools.NewJump(r.values(call.Argument(0)), int(call.Argument(1).ToInteger()), int(call.Argument(2).ToInteger()), int(call.Argument(3).ToInteger())))
	})
	// stutter(gen, n, k): after every n values, the last one k more times
	setg("stutter", func(call goja.FunctionCall) goja.Value {
		return r.wrap(listtools.NewStutter(r.gen(call.Argument(0)), int(call.Argument(1).ToInteger()), int(call.Argument(2).ToInteger())))
	})
}

// lissajousCopy gives a copy of the curve (the generator reuses its slice)
type lissajousCopy struct{ l *lissajous.Lissajous }

func (c lissajousCopy) Generate() []float64 { return slices.Clone(c.l.Generate()) }
func (lissajousCopy) Continuous() bool      { return true }
func (lissajousCopy) Done() bool            { return false }
func (c lissajousCopy) Reset()              { c.l.Reset() }

// vectors makes the values of an iterator copies
type vectors struct{ it *chaositer.Iterator }

func (v vectors) Generate() []float64 { return slices.Clone(v.it.Generate()) }
func (vectors) Continuous() bool      { return true }
func (vectors) Done() bool            { return false }
func (v vectors) Reset()              { v.it.Reset() }

// mask makes the tendency mask of segments of a script
func (r *Runtime) mask(segments, options goja.Value) *tendency.Mask {
	var segs []tendency.Segment
	for _, s := range r.values(segments) {
		from, to := r.numbers(opt(s, "from")), r.numbers(opt(s, "to"))
		if len(from) != 2 {
			panic(r.vm.NewTypeError("tendency: every segment has from: [lo, hi]"))
		}
		if len(to) != 2 {
			to = from
		}
		segs = append(segs, tendency.Segment{N: int(optNumber(s, "n", 1)), A1: from[0], A2: from[1], Z1: to[0], Z2: to[1]})
	}
	if len(segs) == 0 {
		panic(r.vm.NewTypeError("tendency: it needs segments, [{ n, from: [lo, hi], to: [lo, hi] }]"))
	}
	return tendency.NewMask(opt(options, "loop").ToBoolean(), segs...)
}

// constraints are the options of a rule that constrains
func (r *Runtime) constraints(options goja.Value) []constrain.Option {
	var out []constrain.Option
	if n := optNumber(options, "stop", 0); n > 0 {
		out = append(out, constrain.Stop(int(n)))
	}
	if opt(options, "pitchClass").ToBoolean() {
		out = append(out, constrain.PitchClass())
	}
	return out
}

// sieveObject is a sieve in a script
func (r *Runtime) sieveObject(s sieve.Sieve) goja.Value {
	vm := r.vm
	o := vm.NewObject()
	other := func(v goja.Value) sieve.Sieve {
		if p, ok := opt(v, "sieve").Export().(sieve.Sieve); ok {
			return p
		}
		panic(vm.NewTypeError("sieve: another sieve"))
	}
	_ = o.Set("sieve", vm.ToValue(s))
	_ = o.Set("values", func() goja.Value { return r.array(intsToFloats(s.Values())) })
	_ = o.Set("contains", func(x int) bool { return s.Contains(x) })
	_ = o.Set("union", func(v goja.Value) goja.Value { return r.sieveObject(s.Union(other(v))) })
	_ = o.Set("intersection", func(v goja.Value) goja.Value { return r.sieveObject(s.Intersection(other(v))) })
	_ = o.Set("difference", func(v goja.Value) goja.Value { return r.sieveObject(s.Difference(other(v))) })
	_ = o.Set("complement", func() goja.Value { return r.sieveObject(s.Complement()) })
	_ = o.Set("scale", func(base float64) goja.Value { return r.array(sieve.Scale(s, base)) })
	_ = o.Set("gen", func(call goja.FunctionCall) goja.Value {
		return wrapAs[int](r, sieve.NewGen(s, flag(call, 0, true)))
	})
	_ = o.Set("gate", func(length int) goja.Value { return wrapAs[bool](r, sieve.NewGate(s, length)) })
	return o
}

// vectors of an iterator are generators of lists
var _ genny.Generator[[]float64] = vectors{}
