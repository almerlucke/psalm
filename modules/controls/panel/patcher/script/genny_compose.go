package script

import (
	"errors"
	"fmt"
	"strings"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/automaton"
	"github.com/almerlucke/genny/chord"
	"github.com/almerlucke/genny/counterpoint"
	"github.com/almerlucke/genny/evolve"
	"github.com/almerlucke/genny/isorhythm"
	"github.com/almerlucke/genny/line"
	"github.com/almerlucke/genny/melody"
	"github.com/almerlucke/genny/ngram/style"
	"github.com/almerlucke/genny/note"
	"github.com/almerlucke/genny/phase"
	"github.com/almerlucke/genny/rest"
	"github.com/almerlucke/genny/scale"
	"github.com/almerlucke/genny/stochastic"
	"github.com/dop251/goja"
)

func init() {
	for i, m := range [][]float64{scale.Mode1, scale.Mode2, scale.Mode3, scale.Mode4, scale.Mode5, scale.Mode6, scale.Mode7} {
		scales[fmt.Sprintf("messiaen%d", i+1)] = m
	}
}

// events makes note events of an array of objects of a script: { pitch, start, duration = 1, velocity = 0.8 }
func (r *Runtime) events(v goja.Value) []note.Event {
	var out []note.Event
	for _, e := range r.values(v) {
		out = append(out, note.Event{
			Pitch: optNumber(e, "pitch", 60), Velocity: optNumber(e, "velocity", 0.8),
			Start: optNumber(e, "start", 0), Duration: optNumber(e, "duration", 1),
		})
	}
	return out
}

// eventsOut makes an array of objects of note events
func (r *Runtime) eventsOut(es []note.Event) goja.Value {
	a := make([]any, len(es))
	for i, e := range es {
		o := r.vm.NewObject()
		_ = o.Set("pitch", e.Pitch)
		_ = o.Set("velocity", e.Velocity)
		_ = o.Set("start", e.Start)
		_ = o.Set("duration", e.Duration)
		a[i] = o
	}
	return r.vm.NewArray(a...)
}

// scaleArg is the scale of a scale of genny.scale (nil: none given)
func (r *Runtime) scaleArg(v goja.Value, what string) *scale.Scale {
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	if s, ok := opt(v, "scale").Export().(*scale.Scale); ok {
		return s
	}
	panic(r.vm.NewTypeError(what + ": a scale, genny.scale(60, \"major\")"))
}

// guarded runs f; a panic of genny (an argument that can never work) is a TypeError of the script
func (r *Runtime) guarded(name string, f func(goja.FunctionCall) goja.Value) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		defer func() {
			if e := recover(); e != nil {
				switch e.(type) {
				case *goja.Object, *goja.Exception:
					panic(e)
				}
				panic(r.vm.NewTypeError(fmt.Sprintf("%s: %v", name, e)))
			}
		}()
		return f(call)
	}
}

// composeGenny adds the generators and the tools that make whole lines: lines of note events and what is done to them,
// counterpoint, a melody over chords, isorhythm, phasing, stochastic music, Game of Life, an evolving phrase and a learned style
func (r *Runtime) composeGenny(o *goja.Object, set func(string, func(goja.FunctionCall) goja.Value)) {
	vm := r.vm
	fail := func(format string, args ...any) { panic(vm.NewTypeError(fmt.Sprintf(format, args...))) }
	number := func(call goja.FunctionCall, i int, def float64) float64 {
		if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
			return a.ToFloat()
		}
		return def
	}
	setg := func(name string, f func(goja.FunctionCall) goja.Value) { set(name, r.guarded(name, f)) }

	// --- scales ---

	// messiaen(root, mode): one of Messiaen's modes of limited transposition (1 to 7), a scale like genny.scale
	setg("messiaen", func(call goja.FunctionCall) goja.Value {
		m := int(call.Argument(1).ToInteger())
		steps, ok := scales[fmt.Sprintf("messiaen%d", m)]
		if !ok {
			fail("messiaen: mode 1 to 7")
		}
		return r.gennyScale(scale.New(call.Argument(0).ToFloat(), steps))
	})
	// transpositions(steps): how many different transpositions a set of steps has (a name or an array): 2 for the whole tone scale
	setg("transpositions", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(scale.Transpositions(r.steps(call.Argument(0), "transpositions")))
	})
	// sclScale(root, text): a scale from a Scala .scl file (its text), or one of "12tet", "just5", "meantone", "pythagorean",
	// "bohlenPierce"; the pitches are fractional semitones
	setg("sclScale", func(call goja.FunctionCall) goja.Value {
		text := call.Argument(1).String()
		known := map[string]string{"12tet": scale.SclTET12, "just5": scale.SclJust5, "meantone": scale.SclMeantone,
			"pythagorean": scale.SclPythagorean, "bohlenPierce": scale.SclBohlenPierce}
		if t, ok := known[text]; ok {
			text = t
		}
		scl, err := scale.ParseScl(strings.NewReader(text))
		if err != nil {
			fail("sclScale: %v", err)
		}
		return r.gennyScale(scl.Scale(call.Argument(0).ToFloat()))
	})

	// --- lines of note events: [{ pitch, start, duration = 1, velocity = 0.8 }] with the start and duration in steps ---

	ln := vm.NewObject()
	lset := func(name string, f func(goja.FunctionCall) goja.Value) { _ = ln.Set(name, r.guarded("line."+name, f)) }
	ev := func(call goja.FunctionCall) []note.Event { return r.events(call.Argument(0)) }
	out := r.eventsOut
	// transpose(events, semitones), stretch(events, factor), augment/diminish(events, factor): moved, slower or faster
	lset("transpose", func(c goja.FunctionCall) goja.Value { return out(line.Transpose(ev(c), c.Argument(1).ToFloat())) })
	lset("stretch", func(c goja.FunctionCall) goja.Value { return out(line.Stretch(ev(c), c.Argument(1).ToFloat())) })
	lset("augment", func(c goja.FunctionCall) goja.Value { return out(line.Augment(ev(c), c.Argument(1).ToFloat())) })
	lset("diminish", func(c goja.FunctionCall) goja.Value { return out(line.Diminish(ev(c), c.Argument(1).ToFloat())) })
	// invert(events, axis, scale): mirrored around the pitch axis; with a scale (genny.scale) the mirror is diatonic
	lset("invert", func(c goja.FunctionCall) goja.Value {
		return out(line.Invert(ev(c), c.Argument(1).ToFloat(), r.scaleArg(c.Argument(2), "line.invert")))
	})
	// retrograde(events): backwards in time
	lset("retrograde", func(c goja.FunctionCall) goja.Value { return out(line.Retrograde(ev(c))) })
	// shift(events, steps): later (or earlier, negative)
	lset("shift", func(c goja.FunctionCall) goja.Value { return out(line.Shift(ev(c), c.Argument(1).ToFloat())) })
	// merge(...lines): together, in order of start
	lset("merge", func(c goja.FunctionCall) goja.Value {
		var ls [][]note.Event
		for _, a := range c.Arguments {
			ls = append(ls, r.events(a))
		}
		return out(line.Merge(ls...))
	})
	// slice(events, from, to): the notes that start from `from` up to `to`, moved so `from` is 0
	lset("slice", func(c goja.FunctionCall) goja.Value {
		return out(line.Slice(ev(c), c.Argument(1).ToFloat(), c.Argument(2).ToFloat()))
	})
	// loop(events, n, length): n times one after the other; each cycle is `length` steps long
	lset("loop", func(c goja.FunctionCall) goja.Value {
		return out(line.Loop(ev(c), int(c.Argument(1).ToInteger()), c.Argument(2).ToFloat()))
	})
	// quantize(events, scale): every pitch to the nearest of a scale
	lset("quantize", func(c goja.FunctionCall) goja.Value {
		s := r.scaleArg(c.Argument(1), "line.quantize")
		if s == nil {
			fail("line.quantize: a scale")
		}
		return out(line.Quantize(ev(c), s))
	})
	// canon(events, delay, interval, voices): the line again, `delay` steps later and `interval` semitones higher, for each of
	// `voices`: an array of lines
	lset("canon", func(c goja.FunctionCall) goja.Value {
		lines := line.Canon(ev(c), c.Argument(1).ToFloat(), c.Argument(2).ToFloat(), int(c.Argument(3).ToInteger()))
		a := make([]any, len(lines))
		for i, l := range lines {
			a[i] = out(l)
		}
		return vm.NewArray(a...)
	})
	// thin(events, gate): the notes where a gate generator says true, one value for each note
	lset("thin", func(c goja.FunctionCall) goja.Value { return out(line.Thin(ev(c), r.bools(c.Argument(1)))) })
	_ = o.Set("line", ln)

	// --- counterpoint and melody ---

	// counterpoint(cantus, { side = "above", scale, maxLeap = 9, maxDistance = 19, low, high, budget, second = false }): a line
	// of notes against a cantus firmus (an array of pitches, whole notes), by the rules of first species counterpoint (or
	// second: two notes to each of the cantus). Gives { line, solved }: solved is false when no line follows every rule; the
	// line is then the best it found
	setg("counterpoint", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		cfg := counterpoint.Config{
			Side:        option(vm, goja.FunctionCall{Arguments: []goja.Value{opt(c, "side")}}, 0, "counterpoint side", map[string]counterpoint.Side{"above": counterpoint.Above, "below": counterpoint.Below}, counterpoint.Above),
			Scale:       r.scaleArg(opt(c, "scale"), "counterpoint"),
			MaxLeap:     int(optNumber(c, "maxLeap", 0)),
			Low:         int(optNumber(c, "low", 0)),
			High:        int(optNumber(c, "high", 0)),
			Budget:      int(optNumber(c, "budget", 0)),
			MaxDistance: int(optNumber(c, "maxDistance", 0)),
		}
		gen := counterpoint.Generate
		if opt(c, "second").ToBoolean() {
			gen = counterpoint.GenerateSecond
		}
		cp, err := gen(r.integers(call.Argument(0)), cfg)
		if err != nil && !errors.Is(err, counterpoint.ErrNoSolution) {
			fail("counterpoint: %v", err)
		}
		res := vm.NewObject()
		_ = res.Set("line", r.array(intsToFloats(cp)))
		_ = res.Set("solved", err == nil)
		return res
	})
	// checkCounterpoint(cantus, line, second = false): the rules a line breaks, as texts ("note 3: parallel fifths: ...")
	setg("checkCounterpoint", func(call goja.FunctionCall) goja.Value {
		check := counterpoint.Check
		if flag(call, 2, false) {
			check = counterpoint.CheckSecond
		}
		var texts []any
		for _, v := range check(r.integers(call.Argument(0)), r.integers(call.Argument(1))) {
			texts = append(texts, v.String())
		}
		return vm.NewArray(texts...)
	})
	// toLine(pitches, length = 1): pitches as note events one after the other
	setg("toLine", func(call goja.FunctionCall) goja.Value {
		return out(counterpoint.Line(r.integers(call.Argument(0)), number(call, 1, 1)))
	})
	// chordTones(chords, bars, { scale, barSteps = 16, beatSteps = 4, maxLeap = 3, low = 60, high = 84 }): a melody over chords, one chord
	// to a bar (an array of chords, each an array of pitches, or a generator of them): chord tones on the strong beats, scale
	// tones between, as note events
	setg("chordTones", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(2)
		cfg := melody.Config{
			Scale:    r.scaleArg(opt(c, "scale"), "chordTones"),
			BarSteps: int(optNumber(c, "barSteps", 0)), BeatSteps: int(optNumber(c, "beatSteps", 0)), MaxLeap: int(optNumber(c, "maxLeap", 0)),
			Low: optNumber(c, "low", 0), High: optNumber(c, "high", 0),
		}
		if cfg.Scale == nil {
			fail("chordTones: the options need a scale")
		}
		bars := int(call.Argument(1).ToInteger())
		if g, ok := call.Argument(0).Export().(*Gen); ok {
			return out(melody.ChordTones(from[chord.Chord]{g.g, func(v goja.Value) chord.Chord { return chord.Chord(r.numbers(v)) }}, bars, cfg))
		}
		var list []chord.Chord
		for _, ch := range r.values(call.Argument(0)) {
			list = append(list, chord.Chord(r.numbers(ch)))
		}
		return out(melody.ChordTonesList(list, bars, cfg))
	})

	// --- isorhythm and phasing ---

	// isorhythm(talea, color, length): a rhythm (the talea, durations in steps, 0 is a rest) against a row of pitches (the
	// color) of another length, for `length` steps: note events; the two come back together after isorhythmPeriod notes
	setg("isorhythm", func(call goja.FunctionCall) goja.Value {
		return out(isorhythm.Events(r.numbers(call.Argument(0)), r.numbers(call.Argument(1)), call.Argument(2).ToFloat()))
	})
	setg("isorhythmPeriod", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(isorhythm.Period(r.numbers(call.Argument(0)), r.numbers(call.Argument(1))))
	})
	// isorhythmSteps(talea, color): the same on a grid of steps, as two generators to use together: { pitch, length } (pitch is
	// null on the steps a note sounds on, length the length of the note that starts there)
	setg("isorhythmSteps", func(call goja.FunctionCall) goja.Value {
		st := isorhythm.New(r.numbers(call.Argument(0)), r.numbers(call.Argument(1)))
		res := vm.NewObject()
		_ = res.Set("pitch", r.wrap(mapped[rest.Optional[float64]]{Generator: st.Pitch(), to: func(p rest.Optional[float64]) goja.Value {
			if p.Played() {
				return vm.ToValue(p.Value)
			}
			return goja.Null()
		}}))
		_ = res.Set("length", wrapAs[float64](r, genny.Generator[float64](st.Length())))
		return res
	})
	// phase(pattern, cycles, drift): Reich's phasing: two voices play the pattern (note events), the second a little faster
	// (by `drift` steps a cycle), for `cycles` cycles. Gives { first, second }
	setg("phase", func(call goja.FunctionCall) goja.Value {
		a, b := phase.Events(r.events(call.Argument(0)), int(call.Argument(1).ToInteger()), call.Argument(2).ToFloat())
		res := vm.NewObject()
		_ = res.Set("first", out(a))
		_ = res.Set("second", out(b))
		return res
	})
	// phaseLength(pattern), phaseCycles(pattern, drift): the length of the pattern; the cycles after which the voices are together again
	setg("phaseLength", func(call goja.FunctionCall) goja.Value { return vm.ToValue(phase.Length(r.events(call.Argument(0)))) })
	setg("phaseCycles", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(phase.Cycles(r.events(call.Argument(0)), call.Argument(1).ToFloat()))
	})

	// --- stochastic music ---

	// poisson(rate, length): the times of events that come at random, `rate` to a step on average, over `length` steps
	setg("poisson", func(call goja.FunctionCall) goja.Value {
		return r.array(stochastic.Poisson(call.Argument(0).ToFloat(), call.Argument(1).ToFloat()))
	})
	// poissonInt(mean): a whole number by Poisson's law
	setg("poissonInt", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(stochastic.PoissonInt(call.Argument(0).ToFloat()))
	})
	// density(gen): a gate with a chance of true for every step: the numbers of a generator, 0 (never) to 1 (always)
	setg("density", func(call goja.FunctionCall) goja.Value {
		return wrapAs[bool](r, stochastic.NewDensity(r.floats(call.Argument(0))))
	})
	// cloud({ density, pitch, velocity = 0.8, duration = 1, scale }, steps): a cloud of notes, as events: in each step the number
	// of notes is Poisson around the density (a number, a function of the step or a generator), onsets uniform in the step.
	// pitch is a generator or [lo, hi]
	setg("cloud", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(0)
		cfg := stochastic.Config{Scale: r.scaleArg(opt(c, "scale"), "cloud")}
		d := opt(c, "density")
		if f, ok := goja.AssertFunction(d); ok {
			cfg.Density = func(step int) float64 { return r.must(f(goja.Undefined(), vm.ToValue(step))).ToFloat() }
		} else if g, ok := d.Export().(*Gen); ok {
			cfg.Density = stochastic.FromGenerator(r.floats(vm.ToValue(g)))
		} else {
			v := d.ToFloat()
			cfg.Density = func(int) float64 { return v }
		}
		p := opt(c, "pitch")
		if isArray(p) && len(r.values(p)) == 2 {
			b := r.numbers(p)
			cfg.Pitch = stochastic.Uniform(b[0], b[1])
		} else if !goja.IsUndefined(p) {
			cfg.Pitch = r.floats(p)
		}
		if v := opt(c, "velocity"); !goja.IsUndefined(v) {
			cfg.Velocity = r.floats(v)
		}
		if v := opt(c, "duration"); !goja.IsUndefined(v) {
			cfg.Duration = r.floats(v)
		}
		return out(stochastic.NewCloud(cfg).Render(int(call.Argument(1).ToInteger())))
	})
	// freeStochastic(length, density, meanDuration, pitchMean, pitchSpread, scale): a section of "free stochastic music" (Xenakis),
	// as events: Poisson onsets, exponential durations, Gaussian pitches
	setg("freeStochastic", func(call goja.FunctionCall) goja.Value {
		return out(stochastic.Free(call.Argument(0).ToFloat(), call.Argument(1).ToFloat(), call.Argument(2).ToFloat(),
			call.Argument(3).ToFloat(), call.Argument(4).ToFloat(), r.scaleArg(call.Argument(5), "freeStochastic")))
	})

	// --- Game of Life ---

	// life(width, height, rule = "B3/S23", density = 0, { pattern, x, y }): a grid of cells that are born and die, on a torus. rule
	// "B3/S23" (Conway), "B36/S23" (HighLife), "B3678/S34678" (DayAndNight), "B2/S" (Seeds) or your own. density
	// (0 to 1) fills it at random. pattern is one of "glider", "blinker", "block", "rPentomino", "toad", "lwss", or an array of rows
	// ("#" alive) put at x, y
	setg("life", func(call goja.FunctionCall) goja.Value {
		rule := "B3/S23"
		if a := call.Argument(2); !goja.IsUndefined(a) && !goja.IsNull(a) {
			rule = a.String()
		}
		w, h := int(call.Argument(0).ToInteger()), int(call.Argument(1).ToInteger())
		var l *automaton.Life
		if d := number(call, 3, 0); d > 0 {
			l = automaton.NewLifeRandom(w, h, rule, d)
		} else {
			l = automaton.NewLife(w, h, rule)
		}
		if p := opt(call.Argument(4), "pattern"); !goja.IsUndefined(p) {
			pat := lifePatterns[p.String()]
			if isArray(p) {
				for _, row := range r.values(p) {
					pat = append(pat, row.String())
				}
			} else if pat == nil {
				fail("life: no pattern %q (glider, blinker, block, rPentomino, toad, lwss, or an array of rows)", p.String())
			}
			l.Place(pat, int(optNumber(call.Argument(4), "x", 0)), int(optNumber(call.Argument(4), "y", 0)))
		}
		return r.lifeObject(l)
	})

	// --- an evolving phrase ---

	// evolve({ population = 30, length = 8, min, max, fitness, tournament, crossover, mutation, elitism, start }, perCycle = 1): a
	// genetic algorithm for phrases of whole numbers (notes, or degrees of a scale). fitness rates a phrase: a function
	// (phrase => number, higher is better), or a list of [kind, weight, argument]: "smoothness", "contour" (target, numbers 0 to 1),
	// "arch", "inScale" (a scale), "intervalVariety", "target" (a phrase). Gives { player, step(), run(n), best(), phrase() }: player is a
	// generator of the notes of the best phrase that evolves `perCycle` generations after every phrase
	setg("evolve", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(0)
		cfg := evolve.Config{
			Population: int(optNumber(c, "population", 30)), Length: int(optNumber(c, "length", 8)),
			Min: int(optNumber(c, "min", 48)), Max: int(optNumber(c, "max", 72)),
			Tournament: int(optNumber(c, "tournament", 0)), CrossoverRate: optNumber(c, "crossover", 0),
			MutationRate: optNumber(c, "mutation", 0.1), Elitism: int(optNumber(c, "elitism", 1)),
		}
		for _, p := range r.values(opt(c, "start")) {
			cfg.Start = append(cfg.Start, r.integers(p))
		}
		cfg.Fitness = r.fitness(opt(c, "fitness"), cfg.Length)
		ga := evolve.New(cfg)
		per := int(number(call, 1, 1))
		pl := evolve.NewPlayer(ga, per)
		res := vm.NewObject()
		_ = res.Set("player", wrapAs[int](r, genny.Generator[int](pl)))
		_ = res.Set("step", ga.Step)
		_ = res.Set("run", func(n int) goja.Value { return r.array(intsToFloats(ga.Run(n).Phrase)) })
		_ = res.Set("best", func() goja.Value { return r.array(intsToFloats(ga.Best().Phrase)) })
		_ = res.Set("generation", ga.Generation)
		_ = res.Set("phrase", func() goja.Value { return r.array(intsToFloats(pl.Phrase())) })
		return res
	})

	// --- a learned style ---

	// learnStyle(events, order = 2, { intervals, ratios }): new music in the style of an example (events): pitches and durations
	// together. intervals learns the steps between pitches (the style is the same in any key), ratios the ratios of durations (the
	// same at any speed). Gives { next(), phrase(n), fill(length), events(n, start, velocity) , transpose(semitones), scale(factor) }
	setg("learnStyle", func(call goja.FunctionCall) goja.Value {
		var opts []style.Option
		if c := call.Argument(2); !goja.IsUndefined(c) {
			if opt(c, "intervals").ToBoolean() {
				opts = append(opts, style.Intervals())
			}
			if opt(c, "ratios").ToBoolean() {
				opts = append(opts, style.Ratios())
			}
		}
		st := style.Learn(r.events(call.Argument(0)), int(number(call, 1, 2)), opts...)
		return r.styleObject(st)
	})
}

// lifePatterns are the known patterns of the game of life
var lifePatterns = map[string]automaton.Pattern{
	"glider": automaton.Glider, "blinker": automaton.Blinker, "block": automaton.Block,
	"rPentomino": automaton.RPentomino, "toad": automaton.Toad, "lwss": automaton.LWSS,
}

// fitness makes the fitness of a genetic algorithm of a script: a function of a phrase, or a list of [kind, weight, argument]
func (r *Runtime) fitness(v goja.Value, length int) evolve.Fitness {
	if f, ok := goja.AssertFunction(v); ok {
		return func(phrase []int) float64 {
			return r.must(f(goja.Undefined(), r.array(intsToFloats(phrase)))).ToFloat()
		}
	}
	var parts []evolve.Part
	for _, p := range r.values(v) {
		spec := r.values(p)
		if len(spec) == 0 {
			continue
		}
		weight, arg := 1.0, goja.Undefined()
		if len(spec) > 1 {
			weight = spec[1].ToFloat()
		}
		if len(spec) > 2 {
			arg = spec[2]
		}
		var f evolve.Fitness
		switch kind := spec[0].String(); kind {
		case "smoothness":
			f = evolve.Smoothness()
		case "intervalVariety":
			f = evolve.IntervalVariety()
		case "arch":
			f = evolve.Contour(evolve.Arch(length))
		case "contour":
			f = evolve.Contour(r.numbers(arg))
		case "target":
			f = evolve.Target(r.integers(arg))
		case "inScale":
			s := r.scaleArg(arg, "evolve inScale")
			if s == nil {
				panic(r.vm.NewTypeError("evolve: inScale needs a scale"))
			}
			f = evolve.InScale(s)
		default:
			panic(r.vm.NewTypeError(fmt.Sprintf("evolve: no fitness %q (smoothness, intervalVariety, arch, contour, target, inScale, or a function)", kind)))
		}
		parts = append(parts, evolve.Weight(f, weight))
	}
	if len(parts) == 0 {
		panic(r.vm.NewTypeError("evolve: it needs a fitness, a function or a list [[\"smoothness\", 1]]"))
	}
	return evolve.Weighted(parts...)
}

// lifeObject is a game of life in a script
func (r *Runtime) lifeObject(l *automaton.Life) goja.Value {
	vm := r.vm
	o := vm.NewObject()
	// step(n = 1): the next generation (n of them)
	_ = o.Set("step", func(call goja.FunctionCall) goja.Value {
		n := 1
		if a := call.Argument(0); !goja.IsUndefined(a) {
			n = int(a.ToInteger())
		}
		for range min(n, 100000) {
			l.Step()
		}
		return goja.Undefined()
	})
	_ = o.Set("alive", func(x, y int) bool { return l.Alive(x, y) })
	_ = o.Set("set", func(x, y int, alive bool) { l.Set(x, y, alive) })
	_ = o.Set("randomize", func(d float64) { l.Randomize(d) })
	_ = o.Set("population", l.Population)
	_ = o.Set("generation", l.Generation)
	_ = o.Set("width", l.Width)
	_ = o.Set("height", l.Height)
	_ = o.Set("row", func(y int) goja.Value { return r.grid(l.Row(y)) })
	_ = o.Set("column", func(x int) goja.Value { return r.grid(l.Column(x)) })
	_ = o.Set("toString", l.String)
	// rowGate(y), columnGate(x): a gate that reads the cells of a row (a column) one after the other, the grid taking a step after each pass
	_ = o.Set("rowGate", func(y int) goja.Value { return wrapAs[bool](r, automaton.NewRowGate(l, y)) })
	_ = o.Set("columnGate", func(x int) goja.Value { return wrapAs[bool](r, automaton.NewColumnGate(l, x)) })
	// pitches(y, scale, degrees): the pitches of the live cells of a row (the degrees of a scale, one for each cell), null where dead
	_ = o.Set("pitches", func(call goja.FunctionCall) goja.Value {
		s := r.scaleArg(call.Argument(1), "life.pitches")
		if s == nil {
			panic(vm.NewTypeError("life.pitches: a scale"))
		}
		p := automaton.NewLifePitches(l, int(call.Argument(0).ToInteger()), s, r.integers(call.Argument(2))...)
		return r.wrap(mapped[rest.Optional[float64]]{Generator: p, to: func(v rest.Optional[float64]) goja.Value {
			if v.Played() {
				return vm.ToValue(v.Value)
			}
			return goja.Null()
		}})
	})
	// density(): a number generator, the part of the cells that is alive (0 to 1)
	_ = o.Set("density", func() goja.Value { return wrapAs[float64](r, automaton.NewLifeDensity(l)) })
	return o
}

// styleObject is a learned style in a script
func (r *Runtime) styleObject(st *style.Style) goja.Value {
	vm := r.vm
	o := vm.NewObject()
	noteOut := func(n style.Note) goja.Value {
		x := vm.NewObject()
		_ = x.Set("pitch", n.Pitch)
		_ = x.Set("duration", n.Duration)
		return x
	}
	notes := func(ns []style.Note) goja.Value {
		a := make([]any, len(ns))
		for i, n := range ns {
			a[i] = noteOut(n)
		}
		return vm.NewArray(a...)
	}
	_ = o.Set("next", func() goja.Value { return noteOut(st.Generate()) })
	_ = o.Set("phrase", func(n int) goja.Value { return notes(st.Phrase(min(n, 4096))) })
	_ = o.Set("fill", func(length float64) goja.Value { return notes(st.Fill(length)) })
	_ = o.Set("events", func(n int, start, velocity float64) goja.Value {
		return r.eventsOut(style.Events(st.Phrase(min(n, 4096)), start, velocity))
	})
	_ = o.Set("reset", st.Reset)
	_ = o.Set("transpose", func(semitones float64) goja.Value { return r.styleObject(st.Transpose(semitones)) })
	_ = o.Set("scale", func(factor float64) goja.Value { return r.styleObject(st.Scale(factor)) })
	return o
}
