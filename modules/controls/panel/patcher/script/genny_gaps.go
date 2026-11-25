package script

import (
	"fmt"

	"github.com/almerlucke/genny/chord"
	"github.com/almerlucke/genny/fugue"
	gharmony "github.com/almerlucke/genny/harmony"
	"github.com/almerlucke/genny/listtools"
	"github.com/almerlucke/genny/melody"
	"github.com/almerlucke/genny/polyrhythm"
	"github.com/almerlucke/genny/pulse"
	"github.com/almerlucke/genny/row"
	"github.com/almerlucke/genny/sieve"
	"github.com/almerlucke/genny/split"
	"github.com/dop251/goja"
)

// gapsGenny adds what the first bindings left out of genny (found by listing the exported functions the bindings never name): the
// rhythm lists of listtools, rows and permutations, the metric weights of pulse, parts of a value, converting Hz and pitch, and
// the building blocks of a fugue and of harmony on their own
func (r *Runtime) gapsGenny(o *goja.Object, set func(string, func(goja.FunctionCall) goja.Value)) {
	vm := r.vm
	fail := func(format string, args ...any) { panic(vm.NewTypeError(fmt.Sprintf(format, args...))) }
	setg := func(name string, f func(goja.FunctionCall) goja.Value) { set(name, r.guarded(name, f)) }
	number := func(call goja.FunctionCall, i int, def float64) float64 {
		if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
			return a.ToFloat()
		}
		return def
	}

	// --- lists of durations (negative is a rest) ---

	// group(gen, k): every value of gen repeated k times; k is a number or a generator of numbers, taken as a value starts
	setg("group", func(call goja.FunctionCall) goja.Value {
		return r.wrap(listtools.NewGroup(r.gen(call.Argument(0)), r.ints(call.Argument(1))))
	})
	// grow(start, { mode = "geometric", amount = 2, lo = 1, hi = 16, limit = "restart" }): a series from start, each value the last one
	// times (or plus, mode "arithmetic") amount; at the edge of [lo, hi] it stays there ("clip") or goes back to the start ("restart")
	setg("grow", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		mode := option(vm, goja.FunctionCall{Arguments: []goja.Value{opt(c, "mode")}}, 0, "grow mode", map[string]listtools.Mode{"geometric": listtools.Geometric, "arithmetic": listtools.Arithmetic}, listtools.Geometric)
		limit := option(vm, goja.FunctionCall{Arguments: []goja.Value{opt(c, "limit")}}, 0, "grow limit", map[string]listtools.Limit{"clip": listtools.Clip, "restart": listtools.Restart}, listtools.Restart)
		lo, hi := optNumber(c, "lo", 1), optNumber(c, "hi", 16)
		if lo > hi {
			fail("grow: lo is above hi")
		}
		return wrapAs[float64](r, listtools.NewGrow(call.Argument(0).ToFloat(), mode, optNumber(c, "amount", 2), lo, hi, limit))
	})
	// plusMin(gen, p): the values of gen, each made a rest (negative) with chance p
	setg("plusMin", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, listtools.NewPlusMin(r.floats(call.Argument(0)), number(call, 1, 0.5)))
	})
	// positiveNegative(gen, pos, neg): pos values as they are, then neg of them as rests, again and again
	setg("positiveNegative", func(call goja.FunctionCall) goja.Value {
		pos, neg := int(call.Argument(1).ToInteger()), int(call.Argument(2).ToInteger())
		if pos < 0 || neg < 0 || pos+neg == 0 {
			fail("positiveNegative: counts of 0 or more, not both 0")
		}
		return wrapAs[float64](r, listtools.NewPositiveNegative(r.floats(call.Argument(0)), pos, neg))
	})
	// gatherUntil(gen, total): values of gen (durations) until they add up to total, the last one cut to fit: a bar
	setg("gatherUntil", func(call goja.FunctionCall) goja.Value {
		return r.array(listtools.GatherUntil(r.floats(call.Argument(0)), call.Argument(1).ToFloat()))
	})
	// splitEvenly(value, n): value cut into n equal parts
	setg("splitEvenly", func(call goja.FunctionCall) goja.Value {
		n := int(call.Argument(1).ToInteger())
		if n < 1 {
			fail("splitEvenly: at least 1 part")
		}
		return r.array(split.Metric(call.Argument(0).ToFloat(), n))
	})

	// --- rows ---

	// allIntervalRows(n = 12, limit = 10): rows of n pitch classes from 0 in which every interval comes once, at most limit of them
	setg("allIntervalRows", func(call goja.FunctionCall) goja.Value {
		n, limit := int(number(call, 0, 12)), int(number(call, 1, 10))
		if n < 1 || n > 12 {
			fail("allIntervalRows: n from 1 to 12")
		}
		rows := row.AllIntervals(n, max(limit, 1))
		a := make([]any, len(rows))
		for i, rw := range rows {
			a[i] = r.array(intsToFloats(rw))
		}
		return vm.NewArray(a...)
	})
	// permutations(elements, groupLen = all, limit = 100): arrangements of elements in every order, in a fixed order, at most limit
	setg("permutations", func(call goja.FunctionCall) goja.Value {
		els := r.integers(call.Argument(0))
		if len(els) > 8 {
			fail("permutations: at most 8 elements")
		}
		ps := row.Permutations(els, int(min(max(number(call, 2, 100), 1), 5000)), int(number(call, 1, 0)))
		a := make([]any, len(ps))
		for i, p := range ps {
			a[i] = r.array(intsToFloats(p))
		}
		return vm.NewArray(a...)
	})

	// --- the meter ---

	// pulseHierarchy(divisions): the importance of every pulse of a bar divided by divisions ([3, 2, 2] is 3 in 2 in 2): 0 is the least
	setg("pulseHierarchy", func(call goja.FunctionCall) goja.Value {
		return r.array(intsToFloats(pulse.Hierarchy(r.integers(call.Argument(0))...)))
	})
	// pulseVelocity(divisions, lo = 0.4, hi = 1): a velocity for every pulse, the first pulse the strongest
	setg("pulseVelocity", func(call goja.FunctionCall) goja.Value {
		return r.array(pulse.Velocity(pulse.Hierarchy(r.integers(call.Argument(0))...), number(call, 1, 0.4), number(call, 2, 1)))
	})
	// pulseThin(divisions, level): true for the pulses that are at least that important (level 0 keeps all, a higher one is sparser)
	setg("pulseThin", func(call goja.FunctionCall) goja.Value {
		return r.grid(pulse.Thin(pulse.Hierarchy(r.integers(call.Argument(0))...), int(call.Argument(1).ToInteger())))
	})
	// pulseMorph(divisionsA, divisionsB, rate): the importances between two meters of the same number of pulses, rate 0 is the first, 1 the second
	setg("pulseMorph", func(call goja.FunctionCall) goja.Value {
		a, b := pulse.Hierarchy(r.integers(call.Argument(0))...), pulse.Hierarchy(r.integers(call.Argument(1))...)
		if len(a) != len(b) {
			fail("pulseMorph: the meters have %d and %d pulses, they must be the same", len(a), len(b))
		}
		return r.array(intsToFloats(pulse.Interpolate(a, b, number(call, 2, 0.5))))
	})
	// metricWeight(step, barSteps = 16, beatSteps = 4): 3 on the first step of a bar, 2 on the other beats, 1 on half beats, else 0
	setg("metricWeight", func(call goja.FunctionCall) goja.Value {
		bar, beat := int(number(call, 1, 16)), int(number(call, 2, 4))
		if bar < 1 || beat < 1 {
			fail("metricWeight: barSteps and beatSteps are at least 1")
		}
		return vm.ToValue(melody.Weight(int(call.Argument(0).ToInteger()), bar, beat))
	})

	// --- pitch and frequency ---

	// hzToMidi(hz), midiToHz(pitch): 69 is A4 at 440 Hz; pitches have decimals
	setg("hzToMidi", func(call goja.FunctionCall) goja.Value {
		hz := call.Argument(0).ToFloat()
		if !(hz > 0) {
			fail("hzToMidi: a frequency above 0")
		}
		return vm.ToValue(chord.HzToMIDI(hz))
	})
	setg("midiToHz", func(call goja.FunctionCall) goja.Value { return vm.ToValue(chord.MIDIToHz(call.Argument(0).ToFloat())) })

	// --- sieves on lists (the methods of a sieve) are in sieveObject ---

	// --- fugue, in parts ---

	answerKinds := map[string]int{"tonal": int(fugue.Tonal), "real": int(fugue.Real)}
	subjectOut := func(s fugue.Subject) goja.Value {
		obj := vm.NewObject()
		_ = obj.Set("degrees", s.Degrees)
		_ = obj.Set("durations", s.Durations)
		_ = obj.Set("length", s.Length())
		return obj
	}
	subjectIn := func(v goja.Value, what string) fugue.Subject {
		if len(r.values(opt(v, "degrees"))) == 0 {
			fail("%s: a subject is { degrees, durations }", what)
		}
		return fugue.NewSubject(r.integers(opt(v, "degrees")), r.numbers(opt(v, "durations")))
	}
	// fugueAnswer(subject, { scale, answer = "tonal" }): the answer to a subject ({ degrees, durations }), a fifth up
	setg("fugueAnswer", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		key := r.scaleArg(opt(c, "scale"), "fugueAnswer")
		if key == nil {
			fail("fugueAnswer: the key is { scale: genny.scale(62, \"minor\") }")
		}
		kind := fugue.AnswerKind(option(vm, goja.FunctionCall{Arguments: []goja.Value{opt(c, "answer")}}, 0, "fugueAnswer answer", answerKinds, int(fugue.Tonal)))
		return subjectOut(fugue.Answer(subjectIn(call.Argument(0), "fugueAnswer"), key, kind))
	})
	// countersubject(subject, { scale, species = 2, attempts }): a line that goes with a subject (and can swap places with it at the
	// octave), or an error when none is found; uses the random stream
	setg("countersubject", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		key := r.scaleArg(opt(c, "scale"), "countersubject")
		if key == nil {
			fail("countersubject: the key is { scale: genny.scale(62, \"minor\") }")
		}
		cs, err := fugue.Countersubject(subjectIn(call.Argument(0), "countersubject"), key, fugue.Config{Species: int(optNumber(c, "species", 0)), Attempts: int(optNumber(c, "attempts", 0))})
		if err != nil {
			fail("countersubject: %v", err)
		}
		return subjectOut(cs)
	})

	// --- harmony, on its own ---

	hv := o.Get("harmony")
	if h, ok := hv.(*goja.Object); ok {
		hset := func(name string, f func(goja.FunctionCall) goja.Value) {
			_ = h.Set(name, r.guarded("harmony."+name, f))
		}
		keyOut := func(k gharmony.Key) goja.Value {
			obj := vm.NewObject()
			_ = obj.Set("tonic", k.Tonic)
			_ = obj.Set("minor", k.Minor)
			_ = obj.Set("name", k.String())
			return obj
		}
		// dominant(key), relative(key): the key a fifth up, the relative major or minor; keys are as everywhere ("D minor") and give { tonic, minor, name }
		hset("dominant", func(call goja.FunctionCall) goja.Value { return keyOut(r.keyArg(call.Argument(0)).Dominant()) })
		hset("relative", func(call goja.FunctionCall) goja.Value { return keyOut(r.keyArg(call.Argument(0)).Relative()) })
		// check(voicings): the rules the voices of voicings break (the voicings of realize, or { notes: [s, a, t, b], chord }), as texts
		hset("check", func(call goja.FunctionCall) goja.Value {
			var vs []gharmony.Voicing
			for _, x := range r.values(opt(call.Argument(0), "voicings")) {
				notes := r.integers(opt(x, "notes"))
				c, ok := opt(opt(x, "chord"), "_chord").Export().(gharmony.Chord)
				if len(notes) != 4 || !ok {
					fail("harmony.check: voicings is the voicings of harmony.realize")
				}
				vs = append(vs, gharmony.Voicing{Chord: c, Notes: [4]int{notes[0], notes[1], notes[2], notes[3]}})
			}
			var out []any
			for _, v := range gharmony.Check(vs) {
				out = append(out, v.String())
			}
			return vm.NewArray(out...)
		})
	}
}

// sieveMethods adds to a sieve of a script what works on lists: select(values), filter(gen, length), interpret(pattern, kind)
func (r *Runtime) sieveMethods(o *goja.Object, s sieve.Sieve) {
	vm := r.vm
	_ = o.Set("select", r.guarded("sieve.select", func(call goja.FunctionCall) goja.Value {
		vals := r.values(call.Argument(0))
		a := make([]any, 0, len(vals))
		for _, v := range sieve.Select(vals, s) {
			a = append(a, v)
		}
		return vm.NewArray(a...)
	}))
	_ = o.Set("filter", r.guarded("sieve.filter", func(call goja.FunctionCall) goja.Value {
		length := int(call.Argument(1).ToInteger())
		if length < 1 {
			panic(vm.NewTypeError("sieve.filter: a length of at least 1 step"))
		}
		return r.wrap(restToNull{sieve.NewFilter(r.gen(call.Argument(0)), s, length)})
	}))
	_ = o.Set("interpret", r.guarded("sieve.interpret", func(call goja.FunctionCall) goja.Value {
		kind := int(call.Argument(1).ToInteger())
		if kind < 1 || kind > 4 {
			panic(vm.NewTypeError("sieve.interpret: kind 1 to 4"))
		}
		return r.array(sieve.Interpret(r.numbers(call.Argument(0)), s, kind))
	}))
}

// modulationMethods adds to a modulation of a script what combines it: then(other), inverse(), carry(beats)
func (r *Runtime) modulationMethods(o *goja.Object, m polyrhythm.Modulation) {
	vm := r.vm
	_ = o.Set("_mod", vm.ToValue(m))
	_ = o.Set("inverse", func() goja.Value { return r.modulationObject(m.Inverse()) })
	_ = o.Set("carry", func(beats goja.Value) float64 { return m.Carry(r.frac(beats)).Float() })
	_ = o.Set("then", r.guarded("modulation.then", func(call goja.FunctionCall) goja.Value {
		n, ok := opt(call.Argument(0), "_mod").Export().(polyrhythm.Modulation)
		if !ok {
			panic(vm.NewTypeError("modulation.then: another modulation"))
		}
		return r.modulationObject(m.Then(n))
	}))
}
