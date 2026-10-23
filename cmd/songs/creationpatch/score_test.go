package main

import (
	"encoding/json"
	"math"
	"os"
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
	vm := goja.New()
	shared := vm.NewObject()
	prelude, err := preludeJS(just, 440)
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
