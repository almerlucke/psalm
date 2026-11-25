package script

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/almerlucke/genny/fugue"
	gharmony "github.com/almerlucke/genny/harmony"
	"github.com/almerlucke/genny/microtonal"
	"github.com/almerlucke/genny/note"
	"github.com/almerlucke/genny/polyrhythm"
	"github.com/almerlucke/genny/scale"
	"github.com/dop251/goja"
)

// The forms of genny: fugue, functional harmony with four part writing, exact polyrhythm and tunings. Each makes whole lines of
// note events ([{ pitch, start, duration, velocity }], the times in steps), like genny.line and genny.counterpoint do

// formsGenny adds genny.fugue, genny.strettoDistances, genny.harmony, genny.polyrhythm and genny.tuning
func (r *Runtime) formsGenny(o *goja.Object, set func(string, func(goja.FunctionCall) goja.Value)) {
	vm := r.vm
	fail := func(format string, args ...any) { panic(vm.NewTypeError(fmt.Sprintf(format, args...))) }
	setg := func(name string, f func(goja.FunctionCall) goja.Value) { set(name, r.guarded(name, f)) }
	// sub sets a function of an object of its own
	sub := func(obj *goja.Object, prefix, name string, f func(goja.FunctionCall) goja.Value) {
		_ = obj.Set(name, r.guarded(prefix+"."+name, f))
	}
	pick := func(v goja.Value, what string, names map[string]int, def int) int {
		return option(vm, goja.FunctionCall{Arguments: []goja.Value{v}}, 0, what, names, def)
	}
	strs := func(xs []string) goja.Value {
		a := make([]any, len(xs))
		for i, x := range xs {
			a[i] = x
		}
		return vm.NewArray(a...)
	}

	// --- fugue ---

	// the subject: { degrees: [0, 2, 4], durations: [4, 4, 8] } (scale degrees from the tonic, steps), or { pitches, durations }
	// with pitches that are all notes of the key
	subject := func(v goja.Value, key *scale.Scale, what string) fugue.Subject {
		durations := r.numbers(opt(v, "durations"))
		if d := opt(v, "degrees"); !goja.IsUndefined(d) {
			return fugue.NewSubject(r.integers(d), durations)
		}
		if p := opt(v, "pitches"); !goja.IsUndefined(p) {
			var pitches []float64
			for _, x := range r.values(p) {
				q, err := pitchOf(x)
				if err != nil {
					fail("%s: %v", what, err)
				}
				pitches = append(pitches, q)
			}
			return fugue.SubjectFromPitches(key, pitches, durations)
		}
		fail("%s: a subject is { degrees, durations } or { pitches, durations }", what)
		return fugue.Subject{}
	}
	answerKinds := map[string]int{"tonal": int(fugue.Tonal), "real": int(fugue.Real)}
	keyOf := func(c goja.Value, what string) *scale.Scale {
		key := r.scaleArg(opt(c, "scale"), what)
		if key == nil {
			fail("%s: the key is { scale: genny.scale(62, \"minor\") }", what)
		}
		return key
	}
	// fugue(subject, { scale, voices = 3, answer = "tonal", stepsPerBar = 16, beat, centers, order, species = 2, velocity = 0.8, attempts }):
	// the exposition of a fugue, an object to go on with: episode, middle, stretto, close, then voices() (the notes of every voice,
	// the highest first). warning is set when no countersubject was found (the voices then rest)
	setg("fugue", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		key := keyOf(c, "fugue")
		cfg := fugue.Config{
			Answer:      fugue.AnswerKind(pick(opt(c, "answer"), "fugue answer", answerKinds, int(fugue.Tonal))),
			StepsPerBar: optNumber(c, "stepsPerBar", 0), Beat: optNumber(c, "beat", 0),
			Species: int(optNumber(c, "species", 0)), Velocity: optNumber(c, "velocity", 0), Attempts: int(optNumber(c, "attempts", 0)),
		}
		if v := opt(c, "centers"); !goja.IsUndefined(v) {
			cfg.Centers = r.numbers(v)
		}
		if v := opt(c, "order"); !goja.IsUndefined(v) {
			cfg.Order = r.integers(v)
		}
		f, err := fugue.Exposition(subject(call.Argument(0), key, "fugue"), int(optNumber(c, "voices", 3)), key, cfg)
		if f == nil {
			fail("fugue: %v", err)
		}
		return r.fugueObject(f, err)
	})
	// strettoDistances(subject, { scale, answer = "tonal", voices = 2, beat = 4 }): the distances (steps) at which a stretto sounds well
	setg("strettoDistances", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		key := keyOf(c, "strettoDistances")
		kind := fugue.AnswerKind(pick(opt(c, "answer"), "strettoDistances answer", answerKinds, int(fugue.Tonal)))
		return r.array(fugue.StrettoDistances(subject(call.Argument(0), key, "strettoDistances"), key, kind, int(optNumber(c, "voices", 2)), optNumber(c, "beat", 4)))
	})

	// --- functional harmony ---

	h := vm.NewObject()
	cadences := map[string]int{"authentic": int(gharmony.Authentic), "half": int(gharmony.Half), "plagal": int(gharmony.Plagal), "deceptive": int(gharmony.Deceptive)}
	hset := func(name string, f func(goja.FunctionCall) goja.Value) { sub(h, "harmony", name, f) }
	// generator({ key, phraseLength = 4, cadences: { authentic: 1, half: 1 }, seventh = 0, secondary = 0, borrowed = 0 }): chords in phrases
	// that end in cadences: next() is a chord, phrase(n, cadence = "authentic") a phrase of n chords, setKey(key)
	hset("generator", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(0)
		cfg := gharmony.Config{
			Key: r.keyArg(opt(c, "key")), PhraseLength: int(optNumber(c, "phraseLength", 0)),
			SeventhProb: optNumber(c, "seventh", 0), SecondaryProb: optNumber(c, "secondary", 0), BorrowedProb: optNumber(c, "borrowed", 0),
		}
		if cs, ok := opt(c, "cadences").(*goja.Object); ok {
			cfg.Cadences = map[gharmony.Cadence]float64{}
			for _, name := range cs.Keys() {
				id, ok := cadences[name]
				if !ok {
					fail("harmony.generator: no cadence %q (authentic, half, plagal or deceptive)", name)
				}
				cfg.Cadences[gharmony.Cadence(id)] = cs.Get(name).ToFloat()
			}
		}
		g := gharmony.New(cfg)
		obj := vm.NewObject()
		_ = obj.Set("next", func() goja.Value { return r.chordValue(g.Generate()) })
		_ = obj.Set("phrase", func(call goja.FunctionCall) goja.Value {
			cad := gharmony.Authentic
			if a := call.Argument(1); !goja.IsUndefined(a) {
				cad = gharmony.Cadence(pick(a, "harmony phrase cadence", cadences, int(gharmony.Authentic)))
			}
			return r.chords(g.Phrase(int(call.Argument(0).ToInteger()), cad))
		})
		_ = obj.Set("setKey", func(k goja.Value) { g.SetKey(r.keyArg(k)) })
		return obj
	})
	// diatonic(key, degree, seventh = false): the chord on a degree of the key (0 is I, 4 is V)
	hset("diatonic", func(call goja.FunctionCall) goja.Value {
		return r.chordValue(gharmony.Diatonic(r.keyArg(call.Argument(0)), int(call.Argument(1).ToInteger()), call.Argument(2).ToBoolean()))
	})
	// secondaryDominant(key, degree, seventh = false): the dominant of a degree (0 is I), null if the degree has none
	hset("secondaryDominant", func(call goja.FunctionCall) goja.Value {
		if c, ok := gharmony.SecondaryDominant(r.keyArg(call.Argument(0)), int(call.Argument(1).ToInteger()), call.Argument(2).ToBoolean()); ok {
			return r.chordValue(c)
		}
		return goja.Null()
	})
	// borrowed(key, degree): the chord of that degree from the parallel key, null if it is the same
	hset("borrowed", func(call goja.FunctionCall) goja.Value {
		if c, ok := gharmony.Borrowed(r.keyArg(call.Argument(0)), int(call.Argument(1).ToInteger())); ok {
			return r.chordValue(c)
		}
		return goja.Null()
	})
	// modulate(from, to): the chords that lead from one key to another (a pivot, then the dominant seventh of the new key)
	hset("modulate", func(call goja.FunctionCall) goja.Value {
		return r.chords(gharmony.Modulate(r.keyArg(call.Argument(0)), r.keyArg(call.Argument(1))))
	})
	// realize(chords, { variety = 0, ranges: [[lo, hi] x 4] }): the chords in four voices (soprano first), the smoothest voice leading
	// that keeps the rules; { voicings: [{ chord, notes: [s, a, t, b] }], violations: [text] } are what could not be kept
	hset("realize", func(call goja.FunctionCall) goja.Value {
		cs := r.chordList(call.Argument(0))
		c := call.Argument(1)
		cfg := gharmony.VoiceConfig{Variety: optNumber(c, "variety", 0)}
		if rs := opt(c, "ranges"); !goja.IsUndefined(rs) {
			list := r.values(rs)
			if len(list) != 4 {
				fail("harmony.realize: four ranges, [[lo, hi], ...], soprano first")
			}
			for i, x := range list {
				lh := r.integers(x)
				if len(lh) != 2 {
					fail("harmony.realize: a range is [lo, hi]")
				}
				cfg.Ranges[i] = gharmony.Range{Low: lh[0], High: lh[1]}
			}
		} else {
			cfg.Ranges = gharmony.DefaultRanges
		}
		vs, violations := gharmony.Realize(cs, cfg)
		res := vm.NewObject()
		list := make([]any, len(vs))
		for i, v := range vs {
			e := vm.NewObject()
			_ = e.Set("chord", r.chordValue(v.Chord))
			_ = e.Set("notes", []int{v.Notes[0], v.Notes[1], v.Notes[2], v.Notes[3]})
			list[i] = e
		}
		_ = res.Set("voicings", vm.NewArray(list...))
		msgs := make([]string, len(violations))
		for i, v := range violations {
			msgs[i] = v.String()
		}
		_ = res.Set("violations", strs(msgs))
		return res
	})
	// voices(voicings, { length = 16, velocity = 0.7, start = 0 }): the voicings as four lines of notes (soprano first), every chord `length` steps long
	hset("voices", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(1)
		length, vel, start := optNumber(c, "length", 16), optNumber(c, "velocity", 0.7), optNumber(c, "start", 0)
		var lines [4][]note.Event
		for i, v := range r.values(opt(call.Argument(0), "voicings")) {
			notes := r.integers(opt(v, "notes"))
			if len(notes) != 4 {
				fail("harmony.voices: voicings is the voicings of harmony.realize")
			}
			for voice, n := range notes {
				lines[voice] = append(lines[voice], note.Event{Pitch: float64(n), Velocity: vel, Start: start + float64(i)*length, Duration: length})
			}
		}
		a := make([]any, 4)
		for i := range a {
			a[i] = r.eventsOut(lines[i])
		}
		return vm.NewArray(a...)
	})
	_ = o.Set("harmony", h)

	// --- polyrhythm ---

	p := vm.NewObject()
	pset := func(name string, f func(goja.FunctionCall) goja.Value) { sub(p, "polyrhythm", name, f) }
	// layer({ pulses, span = 1 }): pulses equally spaced in span beats; or { every: period }; or { tuplet: [n, m], period = 1 } (n in the time of m
	// periods). Times are beats: a number, "2/3" or [2, 3], exact. And: offset, pattern (booleans), accent (0 to 1), pitch = 60, lo, hi, gate = 0.5,
	// euclid: [steps, pulses, rotation], barlow: { soft, level, divisions: [3, 2] }. events(from, to, stepsPerBeat = 4) gives its notes
	pset("layer", func(call goja.FunctionCall) goja.Value {
		return r.layerObject(r.layerOf(call.Argument(0)))
	})
	// cycle(layers): the beats after which all layers are together again; realign(layers): the beats at which each comes back with the first
	pset("cycle", func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(polyrhythm.Cycle(r.layers(call.Argument(0))...).Float())
	})
	pset("realign", func(call goja.FunctionCall) goja.Value {
		fs := polyrhythm.Realign(r.layers(call.Argument(0))...)
		out := make([]float64, len(fs))
		for i, f := range fs {
			out[i] = f.Float()
		}
		return r.array(out)
	})
	// grid(layers): the finest raster on which all pulses fall: { unit (beats), steps, cells: [[bool]], text }
	pset("grid", func(call goja.FunctionCall) goja.Value {
		g := polyrhythm.NewGrid(r.layers(call.Argument(0))...)
		res := vm.NewObject()
		_ = res.Set("unit", g.Unit.Float())
		_ = res.Set("steps", g.Steps)
		cells := make([]any, len(g.Cells))
		for i, row := range g.Cells {
			cells[i] = r.grid(row)
		}
		_ = res.Set("cells", vm.NewArray(cells...))
		_ = res.Set("text", g.String())
		return res
	})
	// meter(unit, groups): an additive meter, 3+3+2 units: { accents, onsets (beats), length (beats), units, signature: [num, den] or null, layer }
	pset("meter", func(call goja.FunctionCall) goja.Value {
		m := polyrhythm.Additive(r.frac(call.Argument(0)), r.integers(call.Argument(1))...)
		res := vm.NewObject()
		_ = res.Set("accents", r.array(m.Accents()))
		onsets := m.Onsets()
		os := make([]float64, len(onsets))
		for i, f := range onsets {
			os[i] = f.Float()
		}
		_ = res.Set("onsets", r.array(os))
		_ = res.Set("length", m.Length().Float())
		_ = res.Set("units", m.Units())
		if n, d, ok := m.TimeSignature(); ok {
			_ = res.Set("signature", []int{n, d})
		} else {
			_ = res.Set("signature", goja.Null())
		}
		_ = res.Set("layer", r.layerObject(m.Layer()))
		return res
	})
	// modulation(old, new): a metric modulation, old beats of the old tempo are new beats of the new one; ratio(n, m): 120 becomes 120 * m / n.
	// tempo(bpm), factor(), carryEvents(events): the notes with their times as beats of the new tempo
	pset("modulation", func(call goja.FunctionCall) goja.Value {
		return r.modulationObject(polyrhythm.Equate(r.frac(call.Argument(0)), r.frac(call.Argument(1))))
	})
	pset("ratio", func(call goja.FunctionCall) goja.Value {
		return r.modulationObject(polyrhythm.Ratio(int(call.Argument(0).ToInteger()), int(call.Argument(1).ToInteger())))
	})
	_ = o.Set("polyrhythm", p)

	// --- tunings ---

	// tuning({ edo: 31 } or { edo: 13, period: 1901.955 } or { just: [1, 9/8, 5/4, 3/2], period } or { cents: [0, 204, 386, ...], period = 1200 } or
	// { scl: text }, root = 60): the pitches (fractional semitones) of another tuning: pitch(degree), note(degree, octave), snap(p), nearest(p),
	// cent(degree), len(), pitches(lo, hi)
	setg("tuning", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(0)
		root := optNumber(c, "root", 60)
		period := optNumber(c, "period", 1200)
		var t microtonal.Tuning
		switch {
		case !goja.IsUndefined(opt(c, "edo")):
			n := int(optNumber(c, "edo", 12))
			if n < 1 {
				fail("tuning: edo must be at least 1")
			}
			t = microtonal.EqualDivision(root, n, period)
		case !goja.IsUndefined(opt(c, "just")):
			t = microtonal.Just(root, period, r.numbers(opt(c, "just"))...)
		case !goja.IsUndefined(opt(c, "cents")):
			t = microtonal.FromCents(root, period, r.numbers(opt(c, "cents"))...)
		case !goja.IsUndefined(opt(c, "scl")):
			scl, err := scale.ParseScl(strings.NewReader(opt(c, "scl").String()))
			if err != nil {
				fail("tuning: %v", err)
			}
			t = microtonal.FromScl(root, scl)
		default:
			fail("tuning: { edo }, { just }, { cents } or { scl }")
		}
		return r.tuningObject(t)
	})
}

// tuningObject is a tuning for a script
func (r *Runtime) tuningObject(t microtonal.Tuning) goja.Value {
	o := r.vm.NewObject()
	_ = o.Set("len", t.Len)
	_ = o.Set("root", t.Root)
	_ = o.Set("pitch", func(degree int) float64 { return t.Pitch(degree) })
	_ = o.Set("note", func(degree, octave int) float64 { return t.Note(degree, octave) })
	_ = o.Set("snap", func(p float64) float64 { return t.Snap(p) })
	_ = o.Set("nearest", func(p float64) int { return t.Nearest(p) })
	_ = o.Set("cent", func(degree int) float64 { return t.Cent(degree) })
	_ = o.Set("pitches", func(lo, hi float64) goja.Value {
		var out []float64
		for oct := -10; oct <= 10 && len(out) < 4096; oct++ {
			for d := range t.Len() {
				if p := t.Note(d, oct); p >= lo && p <= hi {
					out = append(out, p)
				}
			}
		}
		return r.array(out)
	})
	return o
}

// fugueObject is the fugue to go on with
func (r *Runtime) fugueObject(f *fugue.Fugue, warning error) goja.Value {
	vm := r.vm
	o := vm.NewObject()
	set := func(name string, fn func(goja.FunctionCall) goja.Value) {
		_ = o.Set(name, r.guarded("fugue."+name, fn))
	}
	kinds := map[string]int{"subject": int(fugue.KindSubject), "answer": int(fugue.KindAnswer)}
	if warning != nil {
		_ = o.Set("warning", warning.Error())
	}
	_ = o.Set("hasCounter", len(f.Counter.Degrees) > 0)
	// episode({ voices: [0, 1], repeats = 3, step = -1, shift = 0, fragment }): a sequence from the head of the subject; gives the shift to aim a middle entry at
	set("episode", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(0)
		cfg := fugue.EpisodeConfig{Repeats: int(optNumber(c, "repeats", 0)), Step: int(optNumber(c, "step", 0)), Shift: int(optNumber(c, "shift", 0)), Fragment: optNumber(c, "fragment", 0)}
		if v := opt(c, "voices"); !goja.IsUndefined(v) {
			cfg.Voices = r.integers(v)
		}
		return vm.ToValue(f.Episode(cfg))
	})
	// middle(voice, "subject" or "answer", shift, withCounter = false): one entry in a new key area (2 the relative major, 4 the dominant, 3 the subdominant)
	set("middle", func(call goja.FunctionCall) goja.Value {
		kind := fugue.Kind(option(vm, call, 1, "fugue middle", kinds, int(fugue.KindSubject)))
		f.Middle(int(call.Argument(0).ToInteger()), kind, int(call.Argument(2).ToInteger()), call.Argument(3).ToBoolean())
		return goja.Undefined()
	})
	// stretto(distance, voices = the order of the exposition): overlapping entries; false (and nothing added) if the distance does not work
	set("stretto", func(call goja.FunctionCall) goja.Value {
		var voices []int
		if v := call.Argument(1); !goja.IsUndefined(v) && !goja.IsNull(v) {
			voices = r.integers(v)
		}
		if err := f.Stretto(call.Argument(0).ToFloat(), voices); err != nil {
			if errors.Is(err, fugue.ErrStretto) {
				return vm.ToValue(false)
			}
			panic(vm.NewTypeError("stretto: " + err.Error()))
		}
		return vm.ToValue(true)
	})
	// close({ bars = 4, leadingTone = false }): the ending, a pedal on the dominant and then the tonic
	set("close", func(call goja.FunctionCall) goja.Value {
		c := call.Argument(0)
		f.Close(fugue.CloseConfig{Bars: int(optNumber(c, "bars", 0)), LeadingTone: opt(c, "leadingTone").ToBoolean()})
		return goja.Undefined()
	})
	// voices(): the notes of every voice, the highest first; end(): where it ends (steps); check(): what the rules find wrong (texts)
	set("voices", func(goja.FunctionCall) goja.Value {
		a := make([]any, len(f.Voices))
		for i, v := range f.Voices {
			a[i] = r.eventsOut(v)
		}
		return vm.NewArray(a...)
	})
	set("end", func(goja.FunctionCall) goja.Value { return vm.ToValue(f.End()) })
	set("check", func(goja.FunctionCall) goja.Value {
		var out []string
		for _, p := range f.Check() {
			out = append(out, p.String())
		}
		a := make([]any, len(out))
		for i, s := range out {
			a[i] = s
		}
		return vm.NewArray(a...)
	})
	// sections(): [{ name, start, end }] (exposition, episode, middle entry, stretto, close); entries(): [{ voice, start, kind, shift, section }]
	set("sections", func(goja.FunctionCall) goja.Value {
		a := make([]any, len(f.Sections))
		for i, s := range f.Sections {
			e := vm.NewObject()
			_ = e.Set("name", s.Name)
			_ = e.Set("start", s.Start)
			_ = e.Set("end", s.End)
			a[i] = e
		}
		return vm.NewArray(a...)
	})
	set("entries", func(goja.FunctionCall) goja.Value {
		a := make([]any, len(f.Entries))
		for i, s := range f.Entries {
			e := vm.NewObject()
			_ = e.Set("voice", s.Voice)
			_ = e.Set("start", s.Start)
			_ = e.Set("kind", s.Kind.String())
			_ = e.Set("shift", s.Shift)
			_ = e.Set("section", s.Section)
			a[i] = e
		}
		return vm.NewArray(a...)
	})
	return o
}

// keyArg is a key of a script: "D minor", "C" (major) or { tonic: "D4", minor: true }
func (r *Runtime) keyArg(v goja.Value) gharmony.Key {
	fail := func(format string, args ...any) { panic(r.vm.NewTypeError(fmt.Sprintf(format, args...))) }
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		fail("a key: \"D minor\", or { tonic: \"D4\", minor: true }")
	}
	pitch := func(x goja.Value) int {
		if s, ok := x.Export().(string); ok && !strings.ContainsAny(s, "0123456789") {
			x = r.vm.ToValue(s + "4")
		}
		p, err := pitchOf(x)
		if err != nil {
			fail("key: %v", err)
		}
		return int(math.Round(p))
	}
	if s, ok := v.Export().(string); ok {
		fields := strings.Fields(s)
		if len(fields) == 0 || len(fields) > 2 {
			fail("key: %q is a note and a mode, \"D minor\"", s)
		}
		k := gharmony.Key{Tonic: pitch(r.vm.ToValue(fields[0]))}
		if len(fields) == 2 {
			switch strings.ToLower(fields[1]) {
			case "minor", "min", "m":
				k.Minor = true
			case "major", "maj":
			default:
				fail("key: the mode is major or minor, not %q", fields[1])
			}
		}
		return k
	}
	return gharmony.Key{Tonic: pitch(opt(v, "tonic")), Minor: opt(v, "minor").ToBoolean()}
}

// chordValue is a chord for a script: { name, numeral, function, root, intervals, pitchClasses, of, borrowed, seventh }; it can be given back
func (r *Runtime) chordValue(c gharmony.Chord) goja.Value {
	o := r.vm.NewObject()
	_ = o.Set("name", c.Name())
	_ = o.Set("numeral", c.Numeral)
	_ = o.Set("function", c.Function.String())
	_ = o.Set("root", c.Root)
	_ = o.Set("intervals", c.Intervals)
	_ = o.Set("pitchClasses", c.PitchClasses())
	_ = o.Set("of", c.Of)
	_ = o.Set("borrowed", c.Borrowed)
	_ = o.Set("seventh", c.IsSeventh())
	_ = o.Set("key", c.Key.String())
	_ = o.Set("_chord", r.vm.ToValue(c))
	return o
}

func (r *Runtime) chords(cs []gharmony.Chord) goja.Value {
	a := make([]any, len(cs))
	for i, c := range cs {
		a[i] = r.chordValue(c)
	}
	return r.vm.NewArray(a...)
}

// chordList is the chords of an array of a script made by harmony
func (r *Runtime) chordList(v goja.Value) []gharmony.Chord {
	var out []gharmony.Chord
	for _, x := range r.values(v) {
		c, ok := opt(x, "_chord").Export().(gharmony.Chord)
		if !ok {
			panic(r.vm.NewTypeError("harmony.realize: chords are those of genny.harmony (diatonic, generator, modulate...)"))
		}
		out = append(out, c)
	}
	return out
}

// frac is a time in beats of a script, exactly: an integer, "2/3", [2, 3], or a number that is a ratio of small whole numbers
func (r *Runtime) frac(v goja.Value) polyrhythm.Frac {
	fail := func(format string, args ...any) { panic(r.vm.NewTypeError(fmt.Sprintf(format, args...))) }
	switch x := v.Export().(type) {
	case string:
		n, d, ok := strings.Cut(x, "/")
		num, err1 := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		den := int64(1)
		var err2 error
		if ok {
			den, err2 = strconv.ParseInt(strings.TrimSpace(d), 10, 64)
		}
		if err1 != nil || err2 != nil || den == 0 {
			fail("%q is not a fraction, \"2/3\"", x)
		}
		return polyrhythm.F(num, den)
	case []any:
		if len(x) != 2 {
			fail("a fraction is [numerator, denominator]")
		}
		n, d := r.vm.ToValue(x[0]).ToInteger(), r.vm.ToValue(x[1]).ToInteger()
		if d == 0 {
			fail("a fraction has no zero denominator")
		}
		return polyrhythm.F(n, d)
	}
	f := v.ToFloat()
	if math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > 1e9 {
		fail("a time in beats: a number, \"2/3\" or [2, 3]")
	}
	if f == math.Trunc(f) {
		return polyrhythm.Int(int64(f))
	}
	// the ratio of small whole numbers nearest to it (continued fractions)
	h0, h1, k0, k1 := int64(0), int64(1), int64(1), int64(0)
	x := f
	for range 20 {
		a := math.Floor(x)
		h0, h1 = h1, int64(a)*h1+h0
		k0, k1 = k1, int64(a)*k1+k0
		if k1 > 10000 || math.Abs(f-float64(h1)/float64(k1)) < 1e-9 {
			break
		}
		x = 1 / (x - a)
	}
	return polyrhythm.F(h1, k1)
}

// layerOf is the layer a script describes (see layer)
func (r *Runtime) layerOf(c goja.Value) polyrhythm.Layer {
	fail := func(format string, args ...any) { panic(r.vm.NewTypeError(fmt.Sprintf(format, args...))) }
	var l polyrhythm.Layer
	switch {
	case !goja.IsUndefined(opt(c, "every")):
		l = polyrhythm.Every(r.frac(opt(c, "every")))
	case !goja.IsUndefined(opt(c, "tuplet")):
		nm := r.integers(opt(c, "tuplet"))
		if len(nm) != 2 {
			fail("layer: tuplet is [n, m]")
		}
		period := polyrhythm.Int(1)
		if v := opt(c, "period"); !goja.IsUndefined(v) {
			period = r.frac(v)
		}
		l = polyrhythm.Tuplet(nm[0], nm[1], period)
	default:
		span := polyrhythm.Int(1)
		if v := opt(c, "span"); !goja.IsUndefined(v) {
			span = r.frac(v)
		}
		l = polyrhythm.NewLayer(int(optNumber(c, "pulses", 0)), span)
	}
	if v := opt(c, "name"); !goja.IsUndefined(v) {
		l.Name = v.String()
	}
	if v := opt(c, "offset"); !goja.IsUndefined(v) {
		l.Offset = r.frac(v)
	}
	if v := opt(c, "pattern"); !goja.IsUndefined(v) {
		for _, b := range r.values(v) {
			l.Pattern = append(l.Pattern, b.ToBoolean())
		}
	}
	if v := opt(c, "accent"); !goja.IsUndefined(v) {
		l.Accent = r.numbers(v)
	}
	// the velocities of a layer are MIDI velocities (60 and 110 by default); a script has them from 0 to 1 as everywhere else
	l.Pitch, l.Gate = optNumber(c, "pitch", 60), optNumber(c, "gate", 0)
	l.Lo, l.Hi = 127*optNumber(c, "lo", 60.0/127), 127*optNumber(c, "hi", 110.0/127)
	if v := opt(c, "euclid"); !goja.IsUndefined(v) {
		e := r.integers(v)
		if len(e) < 2 {
			fail("layer: euclid is [steps, pulses, rotation = 0]")
		}
		e = append(e, 0)
		l = l.WithEuclid(e[0], e[1], e[2])
	}
	if v := opt(c, "barlow"); !goja.IsUndefined(v) {
		l = l.WithBarlow(optNumber(v, "soft", 0.5), int(optNumber(v, "level", 1)), r.integers(opt(v, "divisions"))...)
	}
	return l
}

func (r *Runtime) layerObject(l polyrhythm.Layer) goja.Value {
	o := r.vm.NewObject()
	_ = o.Set("_layer", r.vm.ToValue(l))
	_ = o.Set("period", l.Period().Float())
	_ = o.Set("onset", func(k int) float64 { return l.Onset(k).Float() })
	_ = o.Set("sounds", func(k int) bool { return l.Sounds(k) })
	_ = o.Set("events", r.guarded("layer.events", func(call goja.FunctionCall) goja.Value {
		spb := 4.0
		if a := call.Argument(2); !goja.IsUndefined(a) {
			spb = a.ToFloat()
		}
		if !(spb > 0) {
			panic(r.vm.NewTypeError("layer.events: stepsPerBeat must be above 0"))
		}
		events := l.Events(r.frac(call.Argument(0)), r.frac(call.Argument(1)), spb)
		for i := range events {
			events[i].Velocity = min(max(events[i].Velocity/127, 0), 1)
		}
		return r.eventsOut(events)
	}))
	return o
}

// layers are the layers of an array of a script, made by polyrhythm.layer
func (r *Runtime) layers(v goja.Value) []polyrhythm.Layer {
	var out []polyrhythm.Layer
	for _, x := range r.values(v) {
		l, ok := opt(x, "_layer").Export().(polyrhythm.Layer)
		if !ok {
			panic(r.vm.NewTypeError("polyrhythm: layers are those of genny.polyrhythm.layer"))
		}
		out = append(out, l)
	}
	return out
}

func (r *Runtime) modulationObject(m polyrhythm.Modulation) goja.Value {
	o := r.vm.NewObject()
	_ = o.Set("tempo", func(bpm goja.Value) float64 { return m.Tempo(r.frac(bpm)).Float() })
	_ = o.Set("factor", func() float64 { return m.Factor().Float() })
	_ = o.Set("carryEvents", func(events goja.Value) goja.Value { return r.eventsOut(m.CarryEvents(r.events(events))) })
	return o
}
