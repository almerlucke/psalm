package main

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/dop251/goja"
)

type event struct {
	Beat    float64   `json:"beat"`
	Pitches []float64 `json:"pitches"`
	Hold    float64   `json:"hold"`
	Vel     float64   `json:"vel"`
	Amp     float64   `json:"amp"`
}

// scriptNames are the scripts of the written parts (the others are played as the song goes) by the name of the part
var scriptNames = map[string]string{"void": "void", "the word": "word", "flute": "flute", "steel pan": "steelpan"}

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

// the tune of the island: the same every time, on the hits of the day, breathing every fourth bar, and led by the voices that follow it
func TestIslandPhrase(t *testing.T) {
	vm, shared := runtime(t, true)
	phrase, _ := goja.AssertFunction(shared.Get("phrase"))
	get := func(dayNumber, bar int) (end int, hits []int) {
		day := shared.Get("days").ToObject(vm).Get(fmt.Sprint(dayNumber - 1))
		v, err := phrase(goja.Undefined(), day, vm.ToValue(bar))
		if err != nil {
			t.Fatal(err)
		}
		o := v.ToObject(vm)
		hs := o.Get("hits").ToObject(vm)
		for i := 0; i < int(hs.Get("length").ToInteger()); i++ {
			h := hs.Get(fmt.Sprint(i)).ToObject(vm)
			hits = append(hits, int(h.Get("s").ToInteger()))
			if p := h.Get("pos").ToInteger(); p < 0 || p > 5 {
				t.Errorf("tone %d out of the chord", p)
			}
		}
		return int(o.Get("end").ToInteger()), hits
	}
	for _, c := range []struct{ day, hits int }{{3, 7}, {5, 5}, {7, 3}} {
		e1, h1 := get(c.day, 5)
		e2, h2 := get(c.day, 5)
		if e1 != e2 || !reflect.DeepEqual(h1, h2) || len(h1) != c.hits {
			t.Errorf("day %d: %v %v, want %d hits the same every time", c.day, h1, h2, c.hits)
		}
		if _, h := get(c.day, 3); len(h) != 0 {
			t.Errorf("day %d: the tune does not breathe in its fourth bar: %v", c.day, h)
		}
	}
}
