package script

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/automaton"
	"github.com/almerlucke/genny/chord"
	"github.com/almerlucke/genny/clock"
	"github.com/almerlucke/genny/euclid"
	"github.com/almerlucke/genny/groove"
	"github.com/almerlucke/genny/lsystem"
	"github.com/almerlucke/genny/mutate"
	"github.com/almerlucke/genny/numseq"
	"github.com/almerlucke/genny/progression"
	"github.com/almerlucke/genny/pulse"
	"github.com/almerlucke/genny/ratchet"
	"github.com/almerlucke/genny/rest"
	"github.com/almerlucke/genny/row"
	"github.com/almerlucke/genny/scale"
	"github.com/almerlucke/genny/section"
	"github.com/almerlucke/genny/split"
	"github.com/dop251/goja"
)

// from makes a generator of Go values out of a generator of a script: its values through f
type from[T any] struct {
	g genny.Generator[goja.Value]
	f func(goja.Value) T
}

func (c from[T]) Generate() T      { return c.f(c.g.Generate()) }
func (c from[T]) Continuous() bool { return c.g.Continuous() }
func (c from[T]) Done() bool       { return c.g.Done() }
func (c from[T]) Reset()           { c.g.Reset() }
func (r *Runtime) floats(v goja.Value) genny.Generator[float64] {
	return from[float64]{r.gen(v), func(x goja.Value) float64 { return x.ToFloat() }}
}
func (r *Runtime) ints(v goja.Value) genny.Generator[int] {
	return from[int]{r.gen(v), func(x goja.Value) int { return int(x.ToInteger()) }}
}
func (r *Runtime) bools(v goja.Value) genny.Generator[bool] {
	return from[bool]{r.gen(v), func(x goja.Value) bool { return x.ToBoolean() }}
}

// numbers is the numbers of an array of a script
func (r *Runtime) numbers(v goja.Value) []float64 {
	vs := r.values(v)
	out := make([]float64, len(vs))
	for i, x := range vs {
		out[i] = x.ToFloat()
	}
	return out
}

// integers is the whole numbers of an array of a script
func (r *Runtime) integers(v goja.Value) []int {
	vs := r.values(v)
	out := make([]int, len(vs))
	for i, x := range vs {
		out[i] = int(x.ToInteger())
	}
	return out
}

// array makes an array of a script of numbers
func (r *Runtime) array(xs []float64) goja.Value {
	a := make([]any, len(xs))
	for i, x := range xs {
		a[i] = x
	}
	return r.vm.NewArray(a...)
}

// opt is the field of an object of options (undefined if there is none)
func opt(v goja.Value, key string) goja.Value {
	if o, ok := v.(*goja.Object); ok && o != nil {
		if x := o.Get(key); x != nil {
			return x
		}
	}
	return goja.Undefined()
}

// optNumber is a number of an object of options
func optNumber(v goja.Value, key string, def float64) float64 {
	if x := opt(v, key); !goja.IsUndefined(x) && !goja.IsNull(x) {
		return x.ToFloat()
	}
	return def
}

// scales are the scales a script can name
var scales = map[string][]float64{
	"major": scale.Major, "minor": scale.Minor, "harmonicMinor": scale.HarmonicMinor, "melodicMinor": scale.MelodicMinor,
	"dorian": scale.Dorian, "phrygian": scale.Phrygian, "lydian": scale.Lydian, "mixolydian": scale.Mixolydian,
	"locrian": scale.Locrian, "majorPentatonic": scale.MajorPentatonic, "minorPentatonic": scale.MinorPentatonic,
	"blues": scale.Blues, "wholeTone": scale.WholeTone, "diminished": scale.Diminished, "chromatic": scale.Chromatic,
}

// chords are the chords a script can name
var chords = map[string][]float64{
	"major": chord.Major, "minor": chord.Minor, "diminished": chord.Diminished, "augmented": chord.Augmented,
	"sus2": chord.Sus2, "sus4": chord.Sus4, "major6": chord.Major6, "minor6": chord.Minor6, "major7": chord.Major7,
	"minor7": chord.Minor7, "dominant7": chord.Dominant7, "minorMajor7": chord.MinorMajor7,
	"halfDiminished": chord.HalfDiminished, "diminished7": chord.Diminished7, "add9": chord.Add9,
	"major9": chord.Major9, "minor9": chord.Minor9, "dominant9": chord.Dominant9,
}

// progressions are the tables of chord moves a script can name
var progressions = map[string]progression.Table{
	"pop": progression.Pop, "classical": progression.Classical, "jazz": progression.Jazz, "blues": progression.Blues,
}

// steps is the steps of a scale of a script: a name or an array of semitones
func (r *Runtime) steps(v goja.Value, what string) []float64 {
	if o, ok := v.(*goja.Object); ok && o.ClassName() == "Array" {
		return r.numbers(v)
	}
	if s, ok := scales[v.String()]; ok {
		return s
	}
	names := make([]string, 0, len(scales))
	for k := range scales {
		names = append(names, k)
	}
	slices.Sort(names)
	panic(r.vm.NewTypeError(fmt.Sprintf("%s: no scale %q (an array of semitones, or one of %s)", what, v.String(), strings.Join(names, ", "))))
}

// gennyScale is a scale in a script: scale.degree(i), scale.quantize(x), scale.degrees(gen), scale.walk(...)
func (r *Runtime) gennyScale(s *scale.Scale) goja.Value {
	vm := r.vm
	o := vm.NewObject()
	_ = o.Set("len", s.Len)
	_ = o.Set("root", s.Root)
	_ = o.Set("scale", vm.ToValue(s)) // for diatonic
	// degree(i): the pitch of a degree of the scale (0 is the root, negative below it)
	_ = o.Set("degree", func(i int) float64 { return s.Degree(i) })
	// quantize(x): the nearest pitch of the scale
	_ = o.Set("quantize", func(x float64) float64 { return s.Quantize(x) })
	// degreeOf(x): the degree nearest to a pitch (the other way round of degree)
	_ = o.Set("degreeOf", func(x float64) int { return s.DegreeOf(x) })
	// degrees(gen): a generator of the pitches of the degrees a generator gives
	_ = o.Set("degrees", func(call goja.FunctionCall) goja.Value {
		return wrapAs(r, genny.Generator[float64](scale.NewDegrees(s, r.ints(call.Argument(0)))))
	})
	// walk(start = 0, lo = -7, hi = 7, maxStep = 2, leap = 0.1, maxLeap = 4): a walk over the degrees, now and then a leap
	_ = o.Set("walk", func(call goja.FunctionCall) goja.Value {
		n := func(i int, def float64) float64 {
			if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
				return a.ToFloat()
			}
			return def
		}
		return wrapAs(r, genny.Generator[float64](scale.NewWalk(s, int(n(0, 0)), int(n(1, -7)), int(n(2, 7)), int(n(3, 2)), n(4, 0.1), int(n(5, 4)))))
	})
	return o
}

// wrapAs makes a generator of Go values a Gen of the script (numbers, booleans, texts)
func wrapAs[T any](r *Runtime, g genny.Generator[T]) goja.Value {
	return r.wrap(mapped[T]{Generator: g, to: func(v T) goja.Value { return r.vm.ToValue(v) }})
}

// wrapList makes a generator of lists of numbers a Gen of the script
func wrapList(r *Runtime, g genny.Generator[[]float64]) goja.Value {
	return r.wrap(mapped[[]float64]{Generator: g, to: r.array})
}

// grid is a pattern of a script: an array of booleans
func (r *Runtime) grid(b []bool) goja.Value {
	a := make([]any, len(b))
	for i, x := range b {
		a[i] = x
	}
	return r.vm.NewArray(a...)
}

// musicGenny adds the generators for rhythm, pitch and form: gates, rests, grooves, scales, chords and rows, phrases that
// change and rules that write them
func (r *Runtime) musicGenny(o *goja.Object, set func(string, func(goja.FunctionCall) goja.Value)) {
	vm := r.vm
	fail := func(format string, args ...any) { panic(vm.NewTypeError(fmt.Sprintf(format, args...))) }
	number := func(call goja.FunctionCall, i int, def float64) float64 {
		if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
			return a.ToFloat()
		}
		return def
	}
	guard := func(name string, f func(goja.FunctionCall) goja.Value) func(goja.FunctionCall) goja.Value {
		// a constructor of genny panics on what can never work: the script sees a TypeError saying which function
		return func(call goja.FunctionCall) (v goja.Value) {
			defer func() {
				if e := recover(); e != nil {
					if _, ok := e.(*goja.Object); ok {
						panic(e)
					}
					if ex, ok := e.(*goja.Exception); ok {
						panic(ex)
					}
					fail("%s: %v", name, e)
				}
			}()
			return f(call)
		}
	}
	setg := func(name string, f func(goja.FunctionCall) goja.Value) { set(name, guard(name, f)) }

	// --- rhythm ---

	// euclidPattern(steps, pulses, rotation = 0): the pattern of true and false of a euclidean rhythm, as an array
	setg("euclidPattern", func(call goja.FunctionCall) goja.Value {
		return r.grid(euclid.Pattern(int(call.Argument(0).ToInteger()), int(call.Argument(1).ToInteger()), int(number(call, 2, 0))))
	})
	// rests(gen, p): the values of a generator, each played with chance p, else null (a rest)
	setg("rests", func(call goja.FunctionCall) goja.Value {
		g := rest.NewProbability(r.gen(call.Argument(0)), number(call, 1, 0.5))
		return r.wrap(restToNull{g})
	})
	// gated(gen, gate): the values of a generator where a generator of true and false says true, else null
	setg("gated", func(call goja.FunctionCall) goja.Value {
		return r.wrap(restToNull{rest.NewGated(r.gen(call.Argument(0)), r.bools(call.Argument(1)))})
	})
	// swing(amount = 0.2): the shift of the timing of a step (every second one is late), as a fraction of a step
	setg("swing", func(call goja.FunctionCall) goja.Value {
		s := groove.NewSwing(number(call, 0, 0.2))
		if h := number(call, 1, 0); h > 0 {
			s.Humanize(h)
		}
		return wrapAs[float64](r, s)
	})
	// humanize(amount = 0.05): a random shift of the timing, up to amount either way
	setg("humanize", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, groove.NewHumanize(number(call, 0, 0.05)))
	})
	// jitter(gen, amount): the numbers of a generator, each moved at random up to amount either way
	setg("jitter", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, groove.Jitter(r.floats(call.Argument(0)), number(call, 1, 0.05)))
	})
	// accent(groups, { downbeat, group, other }): velocities by the meter: [2, 2, 3] is 7/8 as 2+2+3
	setg("accent", func(call goja.FunctionCall) goja.Value {
		a := groove.NewAccent(r.integers(call.Argument(0))...)
		if l := call.Argument(1); !goja.IsUndefined(l) {
			a.Levels(optNumber(l, "downbeat", 1), optNumber(l, "group", 0.8), optNumber(l, "other", 0.55))
		}
		return wrapAs[float64](r, a)
	})
	// divider(n, offset = 0): true on every nth step
	setg("divider", func(call goja.FunctionCall) goja.Value {
		return wrapAs[bool](r, clock.NewDivider(int(call.Argument(0).ToInteger()), int(number(call, 1, 0))))
	})
	// poly(lengths): cycles of different lengths run together: every step an array of true and false, one for each
	setg("poly", func(call goja.FunctionCall) goja.Value {
		return r.wrap(mapped[[]bool]{Generator: clock.NewPoly(r.integers(call.Argument(0))...), to: func(b []bool) goja.Value { return r.grid(b) }})
	})
	// ratchet(p = 0.3, max = 4): the starts of a roll inside a step (fractions of it, [0] is no roll)
	setg("ratchet", func(call goja.FunctionCall) goja.Value {
		return wrapList(r, ratchet.NewRandom(number(call, 0, 0.3), int(number(call, 1, 4))))
	})
	// ratchetPattern(counts): the same, with the counts in order: [1, 1, 2, 1, 3]
	setg("ratchetPattern", func(call goja.FunctionCall) goja.Value {
		return wrapList(r, ratchet.NewPattern(r.integers(call.Argument(0))...))
	})
	// automaton(rule = 30, width = 16, random = false): the rows of a cellular automaton, arrays of true and false
	setg("automaton", func(call goja.FunctionCall) goja.Value {
		return r.wrap(mapped[[]bool]{Generator: r.automaton(call), to: func(b []bool) goja.Value { return r.grid(b) }})
	})
	// automatonGate(rule = 30, width = 16, random = false): a gate that reads the rows of an automaton one after the other
	setg("automatonGate", func(call goja.FunctionCall) goja.Value {
		return wrapAs[bool](r, automaton.NewGate(r.automaton(call)))
	})
	// pulseGate(divisions, level): a gate by the metric weight of the pulses: divisions [3, 2, 2] is a bar of 3 in 2 in 2;
	// level (a number or a generator) thins it out: 0 plays every pulse, the higher the sparser (6 pulses in this bar, 4 or more only the first)
	setg("pulseGate", func(call goja.FunctionCall) goja.Value {
		ind := pulse.Hierarchy(r.integers(call.Argument(0))...)
		if l := call.Argument(1); isNumber(l) {
			return wrapAs[bool](r, pulse.NewConstGate(ind, int(l.ToInteger())))
		}
		return wrapAs[bool](r, pulse.NewGate(ind, r.floats(call.Argument(1))))
	})
	// split(value, lower, prob = 0.5, branches = 2): a duration cut into parts (an array): split again, each part, with chance
	// prob, until it is lower than lower
	setg("split", func(call goja.FunctionCall) goja.Value {
		return r.array(split.Split(call.Argument(0).ToFloat(), call.Argument(1).ToFloat(), number(call, 2, 0.5), int(number(call, 3, 2))))
	})
	// splitting(value, lower, prob = 0.5, branches = 2): a generator of the parts, cut again every time they have been played
	setg("splitting", func(call goja.FunctionCall) goja.Value {
		return wrapAs[float64](r, split.NewGen(call.Argument(0).ToFloat(), call.Argument(1).ToFloat(), number(call, 2, 0.5), int(number(call, 3, 2))))
	})
	// section(parts, loop = true): the parts one after the other, each for steps values: [{ gen, steps, restart }]
	setg("section", func(call goja.FunctionCall) goja.Value {
		var parts []section.Part[goja.Value]
		for _, p := range r.values(call.Argument(0)) {
			parts = append(parts, section.Part[goja.Value]{Gen: r.gen(opt(p, "gen")), Steps: int(optNumber(p, "steps", 0)), Restart: opt(p, "restart").ToBoolean()})
		}
		if len(parts) == 0 {
			fail("section: it needs parts, [{ gen, steps }]")
		}
		if flag(call, 1, true) {
			return r.wrap(section.NewLoop(parts...))
		}
		return r.wrap(section.New(parts...))
	})

	// --- pitch and harmony ---

	// scale(root, steps = "major", period = 12): a scale; steps is a name or an array of semitones above the root
	setg("scale", func(call goja.FunctionCall) goja.Value {
		steps := r.steps(call.Argument(1), "scale")
		if goja.IsUndefined(call.Argument(1)) {
			steps = scale.Major
		}
		if p := number(call, 2, 12); p != 12 {
			return r.gennyScale(scale.NewPeriod(call.Argument(0).ToFloat(), steps, p))
		}
		return r.gennyScale(scale.New(call.Argument(0).ToFloat(), steps))
	})
	// equalScale(root, divisions): a scale of equal steps of the octave (19 steps, 24 for quarter tones)
	setg("equalScale", func(call goja.FunctionCall) goja.Value {
		return r.gennyScale(scale.Equal(call.Argument(0).ToFloat(), int(call.Argument(1).ToInteger())))
	})
	// justScale(root, ratios): a scale from ratios, e.g. [1, 9/8, 5/4, 4/3, 3/2]
	setg("justScale", func(call goja.FunctionCall) goja.Value {
		return r.gennyScale(scale.FromRatios(call.Argument(0).ToFloat(), r.numbers(call.Argument(1))...))
	})
	// chord(root, kind = "major"): the pitches of a chord (an array); kind is a name or an array of intervals
	setg("chord", func(call goja.FunctionCall) goja.Value {
		var iv []float64
		switch k := call.Argument(1); {
		case goja.IsUndefined(k):
			iv = chord.Major
		case isArray(k):
			iv = r.numbers(k)
		default:
			var ok bool
			if iv, ok = chords[k.String()]; !ok {
				fail("chord: no chord %q", k.String())
			}
		}
		return r.array(chord.New(call.Argument(0).ToFloat(), iv))
	})
	// diatonic(scale, degree, size = 3): the chord on a degree of a scale (a scale of genny.scale): stacked thirds of the scale
	setg("diatonic", func(call goja.FunctionCall) goja.Value {
		var sc *scale.Scale
		if o, ok := call.Argument(0).(*goja.Object); ok {
			sc, _ = opt(o, "scale").Export().(*scale.Scale)
		}
		if sc == nil {
			fail("diatonic: the first argument is a scale, genny.scale(60, \"major\")")
		}
		return r.array(chord.Diatonic(sc, int(call.Argument(1).ToInteger()), int(number(call, 2, 3))))
	})
	// invert(chord, n), open(chord), transpose(chord, semitones): another chord of the pitches
	setg("invert", func(call goja.FunctionCall) goja.Value {
		return r.array(chord.Chord(r.numbers(call.Argument(0))).Invert(int(call.Argument(1).ToInteger())))
	})
	setg("open", func(call goja.FunctionCall) goja.Value {
		return r.array(chord.Chord(r.numbers(call.Argument(0))).Open())
	})
	setg("transpose", func(call goja.FunctionCall) goja.Value {
		return r.array(chord.Chord(r.numbers(call.Argument(0))).Transpose(call.Argument(1).ToFloat()))
	})
	// harmonic(hz, harmonics, distortion = 1, sub = false): the chord of partials of a note (pitches, not Hz)
	setg("harmonic", func(call goja.FunctionCall) goja.Value {
		return r.array(chord.Harmonic(call.Argument(0).ToFloat(), r.integers(call.Argument(1)), number(call, 2, 1), flag(call, 3, false)))
	})
	// fmChord(carrierHz, modHz, orders = 3), ringChord(aHz, bHz): the sidebands of two tones as a chord
	setg("fmChord", func(call goja.FunctionCall) goja.Value {
		return r.array(chord.FM(call.Argument(0).ToFloat(), call.Argument(1).ToFloat(), int(number(call, 2, 3))))
	})
	setg("ringChord", func(call goja.FunctionCall) goja.Value {
		return r.array(chord.RingMod(call.Argument(0).ToFloat(), call.Argument(1).ToFloat()))
	})
	// voiceLeader(low, high): turns chords into the voicing that moves the least: leader.next(chord)
	setg("voiceLeader", func(call goja.FunctionCall) goja.Value {
		v := chord.NewVoiceLeader(call.Argument(0).ToFloat(), call.Argument(1).ToFloat())
		lo := vm.NewObject()
		_ = lo.Set("next", func(c goja.Value) goja.Value { return r.array(v.Transform(r.numbers(c))) })
		_ = lo.Set("reset", v.Reset)
		return lo
	})
	// progression(kind = "pop", start = 0): the degrees of chords, one after the other; kind is "pop", "classical", "jazz" or
	// "blues", or a table { 0: [[3, 3], [4, 1]], ... }: from each degree, [to, weight] moves
	setg("progression", func(call goja.FunctionCall) goja.Value {
		table, ok := progression.Table{}, true
		switch k := call.Argument(0); {
		case goja.IsUndefined(k):
			table = progression.Pop
		case isArray(k):
			fail("progression: a name or a table { degree: [[to, weight], ...] }")
		default:
			if o, isObj := k.(*goja.Object); isObj && o.ClassName() == "Object" {
				for _, key := range o.Keys() {
					var d int
					if _, err := fmt.Sscan(key, &d); err != nil {
						fail("progression: %q is no degree", key)
					}
					for _, m := range r.values(o.Get(key)) {
						mv := r.numbers(m)
						if len(mv) != 2 {
							fail("progression: a move is [to, weight]")
						}
						table[d] = append(table[d], progression.Move{To: int(mv[0]), Weight: mv[1]})
					}
				}
			} else if table, ok = progressions[k.String()]; !ok {
				fail("progression: no table %q (pop, classical, jazz, blues, or your own)", k.String())
			}
		}
		return wrapAs[int](r, progression.New(table, int(number(call, 1, 0))))
	})
	// row(n = 12): a random row of n pitch classes (an array), each once
	setg("row", func(call goja.FunctionCall) goja.Value { return r.array(row.Random(int(number(call, 0, 12)))) })
	// retrograde(row), inversion(row), retroInversion(row), transposeRow(row, semitones): the forms of a row
	setg("retrograde", func(call goja.FunctionCall) goja.Value { return r.array(row.Retrograde(r.numbers(call.Argument(0)))) })
	setg("inversion", func(call goja.FunctionCall) goja.Value { return r.array(row.Inversion(r.numbers(call.Argument(0)))) })
	setg("retroInversion", func(call goja.FunctionCall) goja.Value {
		return r.array(row.RetrogradeInversion(r.numbers(call.Argument(0))))
	})
	setg("transposeRow", func(call goja.FunctionCall) goja.Value {
		return r.array(row.Transpose(r.numbers(call.Argument(0)), call.Argument(1).ToFloat()))
	})
	// serial(row, forms): the forms of a row one after the other, looping: forms are [op, transpose], op "prime", "retro",
	// "invert" or "retroInvert"
	setg("serial", func(call goja.FunctionCall) goja.Value {
		ops := map[string]row.Op{"prime": row.Prime, "retro": row.Retro, "invert": row.Invert, "retroInvert": row.RetroInvert}
		var forms []row.Form
		for _, f := range r.values(call.Argument(1)) {
			fv := r.values(f)
			if len(fv) != 2 {
				fail("serial: a form is [op, transpose]")
			}
			op, ok := ops[fv[0].String()]
			if !ok {
				fail("serial: no form %q (prime, retro, invert, retroInvert)", fv[0].String())
			}
			forms = append(forms, row.Form{Op: op, Transpose: fv[1].ToFloat()})
		}
		return wrapAs[float64](r, genny.Generator[float64](row.NewSerial(r.numbers(call.Argument(0)), forms...)))
	})
	// matrix(row): the twelve tone matrix of a row of pitch classes (an array of arrays)
	setg("matrix", func(call goja.FunctionCall) goja.Value {
		var rows []any
		for _, line := range row.Matrix(r.integers(call.Argument(0))) {
			rows = append(rows, r.array(intsToFloats(line)))
		}
		return vm.NewArray(rows...)
	})

	// --- variation and form ---

	// mutate(values, { replace, swap, rotate, reverse }): a phrase that changes a little every time it has been played: the
	// chance (0 to 1) of each change
	setg("mutate", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		cfg := mutate.Config[goja.Value]{
			ReplaceProb: optNumber(c, "replace", 0), SwapProb: optNumber(c, "swap", 0),
			RotateProb: optNumber(c, "rotate", 0), ReverseProb: optNumber(c, "reverse", 0),
		}
		if f, ok := goja.AssertFunction(opt(c, "with")); ok { // with: a function that makes the value of a replaced note
			cfg.Replace = func() goja.Value { return r.must(f(goja.Undefined())) }
		}
		return r.wrap(mutate.New(r.values(call.Argument(0)), cfg))
	})
	// lsystem(axiom, rules, iterations, symbols, loop = true): a phrase from rewriting: rules { A: "AB", B: "A" }, symbols the
	// value of each letter { A: 60, B: 67 }
	setg("lsystem", func(call goja.FunctionCall) goja.Value {
		rules, symbols := map[rune]string{}, map[rune]goja.Value{}
		if ro, ok := call.Argument(1).(*goja.Object); ok {
			for _, k := range ro.Keys() {
				rules[[]rune(k)[0]] = ro.Get(k).String()
			}
		}
		so, ok := call.Argument(3).(*goja.Object)
		if !ok {
			fail("lsystem: symbols, e.g. { A: 60, B: 67 }")
		}
		for _, k := range so.Keys() {
			symbols[[]rune(k)[0]] = so.Get(k)
		}
		axiom, n := call.Argument(0).String(), int(call.Argument(2).ToInteger())
		if flag(call, 4, true) {
			return r.wrap(lsystem.NewLoop(axiom, rules, n, symbols))
		}
		return r.wrap(lsystem.New(axiom, rules, n, symbols))
	})
	// fibonacci(mod = 12), collatz(start, loop = true), thueMorse(), primes(): the number sequences
	setg("fibonacci", func(call goja.FunctionCall) goja.Value {
		return wrapAs[int](r, numseq.NewFibonacci(int(number(call, 0, 12))))
	})
	setg("collatz", func(call goja.FunctionCall) goja.Value {
		if flag(call, 1, true) {
			return wrapAs[int](r, genny.Generator[int](numseq.NewCollatzLoop(int(call.Argument(0).ToInteger()))))
		}
		return wrapAs[int](r, genny.Generator[int](numseq.NewCollatz(int(call.Argument(0).ToInteger()))))
	})
	setg("thueMorse", func(goja.FunctionCall) goja.Value { return wrapAs[int](r, numseq.NewThueMorse()) })
	setg("primes", func(goja.FunctionCall) goja.Value { return wrapAs[int](r, numseq.NewPrimes()) })
	_ = o
}

// restToNull makes the generator of a rest a generator of values: null where it rests
type restToNull struct {
	g genny.Generator[rest.Optional[goja.Value]]
}

func (c restToNull) Generate() goja.Value {
	if v := c.g.Generate(); v.Played() {
		return v.Value
	}
	return goja.Null()
}
func (c restToNull) Continuous() bool { return c.g.Continuous() }
func (c restToNull) Done() bool       { return c.g.Done() }
func (c restToNull) Reset()           { c.g.Reset() }

// automaton makes the automaton the arguments of a script say
func (r *Runtime) automaton(call goja.FunctionCall) *automaton.Automaton {
	rule, width := uint8(30), 16
	if a := call.Argument(0); !goja.IsUndefined(a) {
		rule = uint8(a.ToInteger())
	}
	if a := call.Argument(1); !goja.IsUndefined(a) {
		width = int(a.ToInteger())
	}
	if flag(call, 2, false) {
		return automaton.NewRandom(rule, width)
	}
	return automaton.New(rule, width)
}

func isArray(v goja.Value) bool {
	o, ok := v.(*goja.Object)
	return ok && o.ClassName() == "Array"
}

func isNumber(v goja.Value) bool {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return false
	}
	_, ok := v.Export().(*Gen)
	if ok {
		return false
	}
	switch v.Export().(type) {
	case int64, float64, int:
		return !math.IsNaN(v.ToFloat())
	}
	return false
}

func intsToFloats(xs []int) []float64 {
	out := make([]float64, len(xs))
	for i, x := range xs {
		out[i] = float64(x)
	}
	return out
}
