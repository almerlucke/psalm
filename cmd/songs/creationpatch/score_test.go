package main

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/dop251/goja"
)

// the notes of the parts in the scripts are those of the song of Go (cmd/songs/creation): its compiled parts are kept in testdata/original_parts.json (the pitches
// in the tuning of equal temperament, one note after the other, a rest with no pitches)
type origNote struct {
	P     []float64 `json:"p"`
	Beats float64   `json:"beats"`
	Hold  float64   `json:"hold"`
	Vel   float64   `json:"vel"`
	Amp   float64   `json:"amp"`
}

type event struct {
	Beat    float64   `json:"beat"`
	Pitches []float64 `json:"pitches"`
	Hold    float64   `json:"hold"`
	Vel     float64   `json:"vel"`
	Amp     float64   `json:"amp"`
}

// scriptNames are the scripts of the parts by the name of the part
var scriptNames = map[string]string{
	"void": "void", "spirit": "spirit", "the word": "word", "violins": "violins", "celli": "celli", "pizzicato": "pizzicato", "harp": "harp", "timpani": "timpani",
	"flute": "flute", "oboe": "oboe", "clarinet": "clarinet", "horns": "horns", "horn": "horn", "celesta": "celesta", "bell pad": "bellpad", "bells": "bells",
	"the deep": "deep", "piano": "piano", "soprano": "soprano", "choir": "choir", "organ": "organ",
}

// legatos are those the song of Go played its parts with
var legatos = map[string]float64{"violins": 1, "celli": 1, "pizzicato": 0.5, "harp": 1.6, "timpani": 1, "flute": 0.95, "oboe": 0.95, "clarinet": 0.95, "horns": 0.95,
	"horn": 0.95, "celesta": 0.9, "bell pad": 1, "bells": 1, "the deep": 1, "piano": 1, "soprano": 0.95, "choir": 1, "organ": 1, "spirit": 1, "the word": 1}

// runtime loads the scripts as the patcher does (a function of the api, shared and genny) with stubs for what a part plays with
func runtime(t *testing.T, just bool) (*goja.Runtime, *goja.Object) {
	t.Helper()
	return runtimeWith(t, just, false)
}

// runtimeWith is runtime with the parts playing like players or not
func runtimeWith(t *testing.T, just, human bool) (*goja.Runtime, *goja.Object) {
	t.Helper()
	vm := goja.New()
	shared := vm.NewObject()
	prelude, err := preludeJS(just, 440, 1, human)
	if err != nil {
		t.Fatal(err)
	}
	noop := func(goja.FunctionCall) goja.Value { return goja.Undefined() }
	api := vm.NewObject()
	for _, name := range []string{"note", "set", "send", "fire", "clock", "mod", "log", "input", "scale", "chord", "bpm"} {
		_ = api.Set(name, noop)
	}
	_ = api.Set("pitch", func(name string) float64 {
		p, err := score.Pitch(name)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return p
	})
	song := vm.NewObject()
	_ = song.Set("bar", 0)
	_ = song.Set("beat", 0)
	_ = api.Set("song", song)
	load := func(code string) {
		const header = `(function(api, shared, genny) { const {note, set, send, fire, clock, mod, log, input, pitch, scale, chord, bpm, song} = api; `
		v, err := vm.RunString(header + code + "\n})")
		if err != nil {
			t.Fatal(err)
		}
		fn, _ := goja.AssertFunction(v)
		if _, err := fn(goja.Undefined(), api, shared, vm.NewObject()); err != nil {
			t.Fatal(err)
		}
	}
	load(prelude + script("score"))
	for _, s := range scriptNames {
		load(script(s))
	}
	return vm, shared
}

func compiled(t *testing.T, vm *goja.Runtime, shared *goja.Object, part, day string) []event {
	t.Helper()
	compile, _ := goja.AssertFunction(shared.Get("compile"))
	v, err := compile(goja.Undefined(), vm.ToValue(part), vm.ToValue(day))
	if err != nil {
		t.Fatalf("%s in %s: %v", part, day, err)
	}
	var out []event
	b, _ := json.Marshal(v.Export())
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPartsAreThoseOfTheSong(t *testing.T) {
	raw, err := os.ReadFile("testdata/original_parts.json")
	if err != nil {
		t.Fatal(err)
	}
	var orig map[string][]origNote
	if err := json.Unmarshal(raw, &orig); err != nil {
		t.Fatal(err)
	}
	vm, shared := runtime(t, false)
	for part := range scriptNames {
		if part == "void" {
			continue // not in the fixture (the song of Go does not check it against the chords)
		}
		var want []event
		beat := 0.0
		for _, n := range orig[part] {
			if len(n.P) > 0 {
				hold := n.Hold
				if hold == 0 {
					hold = n.Beats * legatos[part]
				}
				want = append(want, event{Beat: beat, Pitches: n.P, Hold: hold, Vel: n.Vel, Amp: max(n.Amp, 0)})
			}
			beat += n.Beats
		}
		var got []event
		offset := 0.0
		for _, d := range days {
			for _, e := range compiled(t, vm, shared, part, d.name) {
				e.Beat += offset
				got = append(got, e)
			}
			offset += float64(d.bars * beats)
		}
		if len(got) != len(want) {
			t.Errorf("%s: %d notes, the song of Go has %d", part, len(got), len(want))
			continue
		}
		for i := range want {
			w, g := want[i], got[i]
			amp := func(a float64) float64 {
				if a == 0 {
					return 1
				}
				return a
			}
			if math.Abs(w.Beat-g.Beat) > 1e-6 || math.Abs(w.Hold-g.Hold) > 1e-6 || math.Abs(w.Vel-g.Vel) > 1e-6 || math.Abs(amp(w.Amp)-amp(g.Amp)) > 1e-6 || len(w.Pitches) != len(g.Pitches) {
				t.Errorf("%s note %d: %+v, the song of Go has %+v", part, i, g, w)
				break
			}
			for k := range w.Pitches {
				if math.Abs(w.Pitches[k]-g.Pitches[k]) > 1e-9 {
					t.Errorf("%s note %d: pitches %v, the song of Go has %v", part, i, g.Pitches, w.Pitches)
					break
				}
			}
		}
	}
}

// every part fits its days: its notes last exactly the bars of each day (compile says when not), also the void's
func TestPartsLastTheirDays(t *testing.T) {
	vm, shared := runtime(t, true)
	for part := range scriptNames {
		for _, d := range days {
			compiled(t, vm, shared, part, d.name)
		}
	}
}

// the tuning: just intonation moves the notes a few cents to the pure ratios of E flat, equal temperament leaves them
func TestTuning(t *testing.T) {
	for _, c := range []struct {
		just        bool
		pitch, want float64
	}{{false, 63, 63}, {true, 63, 63}, {true, 67, 67 - 0.1369}, {true, 70, 70 + 0.0196}, {true, 55, 55 - 0.1369}, {true, 64, 64 + 0.1173}} {
		vm, shared := runtime(t, c.just)
		tune, _ := goja.AssertFunction(shared.Get("tune"))
		v, err := tune(goja.Undefined(), vm.ToValue(c.pitch))
		if err != nil {
			t.Fatal(err)
		}
		if got := v.ToFloat(); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("just %v: %v is %v, want %v", c.just, c.pitch, got, c.want)
		}
	}
}

// the feel: the notes of a chord strummed after each other and every part a little late; the accents of the pizzicato; a seed gives the same every time
func TestFeel(t *testing.T) {
	type played struct {
		pitches        []float64
		v, hold, delay float64
	}
	run := func(human bool, part string, beats ...float64) []played {
		var out []played
		vm, shared := runtimeWith(t, false, human)
		reg, _ := goja.AssertFunction(shared.Get("register"))
		mk, err := vm.RunString(`shared => day => [shared.chord([63, 70, 75], 1, { hold: 1 }), shared.chord([63, 70, 75], 1, { hold: 1 }), shared.chord([63, 70, 75], 1, { hold: 1 }), shared.N([], day.bars * shared.beats - 3)]`)
		if err != nil {
			t.Fatal(err)
		}
		mkf, _ := goja.AssertFunction(mk)
		score, err := mkf(goja.Undefined(), shared)
		if err != nil {
			t.Fatal(err)
		}
		pl, err := reg(goja.Undefined(), vm.ToValue(part), vm.ToValue(map[string]any{"legato": 1}), score)
		if err != nil {
			t.Fatal(err)
		}
		po := pl.ToObject(vm)
		sec, _ := goja.AssertFunction(po.Get("section"))
		trig, _ := goja.AssertFunction(po.Get("trigger"))
		if _, err := sec(po, vm.ToValue(days[1].name)); err != nil {
			t.Fatal(err)
		}
		note := vm.ToValue(func(c goja.FunctionCall) goja.Value {
			var ps []float64
			if arr, ok := c.Argument(0).Export().([]any); ok {
				for _, p := range arr {
					ps = append(ps, toFloat(p))
				}
			} else {
				ps = []float64{c.Argument(0).ToFloat()}
			}
			delay := 0.0
			if d := c.Argument(3); !goja.IsUndefined(d) {
				delay = d.ToFloat()
			}
			out = append(out, played{ps, c.Argument(1).ToFloat(), c.Argument(2).ToFloat(), delay})
			return goja.Undefined()
		})
		for _, b := range beats {
			song := vm.NewObject()
			_ = song.Set("bar", int(b)/beats3)
			_ = song.Set("beat", math.Mod(b, beats3))
			ev := vm.NewObject()
			_ = ev.Set("seconds", 0.125)
			if _, err := trig(po, song, note, ev); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}

	// bells: a chord of three strummed 40 ms apart (a third and a bit of a step of 125 ms), all waiting 32 ms
	got := run(true, "bells", 0)
	if len(got) != 3 {
		t.Fatalf("%d notes of a chord of three", len(got))
	}
	for i, g := range got {
		if want := (32 + 40*float64(i)) / 1000 / 0.125; math.Abs(g.delay-want) > 1e-6 {
			t.Errorf("note %d is %v steps late, want %v", i, g.delay, want)
		}
	}
	// off: on the grid
	if g := run(false, "bells", 0); len(g) != 1 || g[0].delay != 0 || len(g[0].pitches) != 3 {
		t.Errorf("not on the grid: %+v", g)
	}
	// the violins start a chord not quite together, and the same every time
	a, b := run(true, "violins", 0, 1, 2), run(true, "violins", 0, 1, 2)
	if len(a) != 9 || !reflect.DeepEqual(a, b) {
		t.Errorf("not the same every time: %v %v", a, b)
	}
	same := true
	for _, g := range a[1:3] {
		same = same && g.delay == a[0].delay
	}
	if same {
		t.Error("the notes of a chord start together")
	}
	for _, g := range a {
		if g.delay < 0 || g.delay*0.125*1000 > 32+8+12+1e-9 {
			t.Errorf("%v ms late", g.delay*125)
		}
	}
	// the pizzicato accents: the first beat of the bar is the loudest, the second the softest (1, 0.8, 0.85); a note on the grid when the parts do not play like players
	pz := run(false, "pizzicato", 0, 1, 2)
	if len(pz) != 3 || !(pz[0].v > pz[2].v && pz[2].v > pz[1].v) || math.Abs(pz[1].v/pz[0].v-0.8) > 1e-9 || math.Abs(pz[2].v/pz[0].v-0.85) > 1e-9 {
		t.Errorf("accents %+v", pz)
	}
}

const beats3 = 3

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	}
	return math.NaN()
}
