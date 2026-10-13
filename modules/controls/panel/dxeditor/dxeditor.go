// Package dxeditor is a plugin of the panel (modules/controls/panel): an editor of the six operator
// FM synth (synths/dx) on the page, as the front panel of a DX7 would be if it had a screen. The
// algorithm as a diagram (click through the 32, or make one of your own: which operator
// modulates which, the carriers, the feedback loop), each operator (its frequency, ratio or fixed,
// level, detune, velocity, rate scaling, amplitude modulation, its envelope to drag and its
// keyboard scaling), the LFO and the pitch envelope, and a bank of patches to load from.
//
// Every setting is a value of the panel (named "<name> op3Coarse", as the dx messages are), so
// presets keep them, MIDI learn can bind them, code can set them.
//
//	synth := dx.NewPoly(dx.Presets["bellPad"](), 12, 2).AddTo(p)
//	ctl.Section("DX")
//	dxeditor.Add(ctl, "DX", synth, dx.Presets["bellPad"](), dxeditor.Bank(patches))
package dxeditor

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/synths/dx"
)

//go:embed dxeditor.js
var script string

//go:embed dxeditor.css
var styles string

// Plugin is the plugin (Add uses it)
var Plugin = panel.Plugin{Name: "dxeditor", JS: script, CSS: styles}

// Kind is the kind of its control
const Kind panel.Kind = "dx7"

// Editor edits the patch of a dx synth
type Editor struct {
	*panel.Param // the algorithm, 1-32 (the settings are its parts)
	synth        *poly.Poly[*dx.Voice]
	patch        dx.Patch // as edited, on the audio thread
	dirty        bool
	bank         []dx.Patch
	parts        map[string]*panel.Param // by setting name ("op3Coarse", "custom", "mod42")
	mu           sync.Mutex
	name         string // of the patch loaded last
}

func (e *Editor) patchName() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.name
}

// Option sets something of the editor
type Option func(*Editor)

// Bank gives the editor patches to load from (a SysEx bank: dx.LoadSysEx)
func Bank(patches []dx.Patch) Option { return func(e *Editor) { e.bank = patches } }

// algorithm is an algorithm as the page draws it, operators from 1
type algorithm struct {
	Carriers []int    `json:"carriers"`
	Mods     [][2]int `json:"mods"`
	Feedback [2]int   `json:"feedback"`
}

func describe(a dx.Algorithm) algorithm {
	d := algorithm{Mods: [][2]int{}, Feedback: [2]int{a.FeedbackFrom + 1, a.FeedbackTo + 1}}
	for _, c := range a.Carriers {
		d.Carriers = append(d.Carriers, c+1)
	}
	for _, m := range a.Mods {
		d.Mods = append(d.Mods, [2]int{m[0] + 1, m[1] + 1})
	}
	return d
}

// data is what the page gets once
type data struct {
	Algorithms []algorithm `json:"algorithms"`
	Bank       []string    `json:"bank"`
}

// state is what it gets as it changes
type state struct {
	Name string `json:"name"`
}

// Add adds an editor of a dx synth playing a patch (the patch the editor starts from) to a panel
// (and the plugin to its page)
func Add(ctl *panel.Panel, name string, synth *poly.Poly[*dx.Voice], patch dx.Patch, opts ...Option) *Editor {
	ctl.Use(Plugin)
	e := &Editor{synth: synth, patch: patch, parts: map[string]*panel.Param{}, name: patch.Name}
	for _, o := range opts {
		o(e)
	}
	d := data{Bank: []string{}}
	for _, a := range dx.Algorithms {
		d.Algorithms = append(d.Algorithms, describe(a))
	}
	for _, p := range e.bank {
		d.Bank = append(d.Bank, p.Name)
	}
	set := func(key string) panel.Target {
		return panel.Func(func(v float64) {
			e.patch.Set(key, v)
			e.dirty = true
		})
	}
	e.Param = ctl.Custom(Kind, name, set("algorithm"), panel.Range(1, 32, panel.Linear), panel.Step(1),
		panel.Default(float64(patch.Algorithm+1)), panel.Data(d),
		panel.OnBuffer(e.apply), panel.Live(func() any { return state{Name: e.patchName()} }))
	part := func(key string, max float64, def float64, to panel.Target) {
		e.parts[key] = e.Part(fmt.Sprintf("%s %s", name, key), 0, max, panel.Linear, to, panel.Step(1), panel.Default(def))
	}
	for _, s := range dx.Settings {
		v, _ := patch.Get(s.Name)
		part(s.Name, float64(s.Max), v, set(s.Name))
	}

	// an algorithm of your own: on (custom), the carriers, each modulation a higher operator can
	// make into a lower one, the feedback loop; the editor plays them when custom is on
	routing := patch.Routed()
	mark := func(key string) panel.Target { return panel.Func(func(float64) { e.dirty = true }) }
	on := 0.0
	if patch.Routing != nil {
		on = 1
	}
	part("custom", 1, on, mark("custom"))
	for op := 1; op <= dx.NumOperators; op++ {
		c := 0.0
		for _, carrier := range routing.Carriers {
			if carrier == op-1 {
				c = 1
			}
		}
		part(fmt.Sprintf("carrier%d", op), 1, c, mark(""))
	}
	for from := 2; from <= dx.NumOperators; from++ {
		for to := 1; to < from; to++ {
			m := 0.0
			for _, mod := range routing.Mods {
				if mod == [2]int{from - 1, to - 1} {
					m = 1
				}
			}
			part(fmt.Sprintf("mod%d%d", from, to), 1, m, mark(""))
		}
	}
	e.parts["fbFrom"] = e.Part(name+" fbFrom", 1, 6, panel.Linear, mark(""), panel.Step(1), panel.Default(float64(routing.FeedbackFrom+1)))
	e.parts["fbTo"] = e.Part(name+" fbTo", 1, 6, panel.Linear, mark(""), panel.Step(1), panel.Default(float64(routing.FeedbackTo+1)))

	if len(e.bank) > 0 {
		e.parts["program"] = e.Part(name+" program", 0, float64(len(e.bank)-1), panel.Linear,
			panel.Func(func(v float64) { e.Load(e.bank[int(v)]) }), panel.Step(1), panel.NoDefault())
	}
	return e
}

// routing returns the algorithm of your own as the parts have it, nil when custom is off (or it
// can't play)
func (e *Editor) routing() *dx.Algorithm {
	if e.parts["custom"].Value() < 0.5 {
		return nil
	}
	a := dx.Algorithm{FeedbackFrom: int(e.parts["fbFrom"].Value()) - 1, FeedbackTo: int(e.parts["fbTo"].Value()) - 1}
	for op := 1; op <= dx.NumOperators; op++ {
		if e.parts[fmt.Sprintf("carrier%d", op)].Value() > 0.5 {
			a.Carriers = append(a.Carriers, op-1)
		}
	}
	for from := 2; from <= dx.NumOperators; from++ {
		for to := 1; to < from; to++ {
			if e.parts[fmt.Sprintf("mod%d%d", from, to)].Value() > 0.5 {
				a.Mods = append(a.Mods, [2]int{from - 1, to - 1})
			}
		}
	}
	if a.FeedbackFrom > a.FeedbackTo {
		a.FeedbackFrom, a.FeedbackTo = a.FeedbackTo, a.FeedbackFrom
	}
	if a.Check() != nil {
		return nil
	}
	return &a
}

// apply gives the synth the patch edited (on the audio thread, once a buffer when it changed)
func (e *Editor) apply() {
	if !e.dirty {
		return
	}
	e.dirty = false
	e.patch.Routing = e.routing()
	dx.SetPatch(e.synth, e.patch)
}

// Load sets the editor (and the synth) to a patch: every setting moves to it. From any goroutine
func (e *Editor) Load(p dx.Patch) {
	e.mu.Lock()
	e.name = p.Name
	e.mu.Unlock()
	e.Set(float64(p.Algorithm + 1))
	for _, s := range dx.Settings {
		v, _ := p.Get(s.Name)
		e.parts[s.Name].Set(v)
	}
	e.parts["custom"].Set(0)
	if p.Routing != nil {
		r := describe(*p.Routing)
		e.parts["custom"].Set(1)
		for op := 1; op <= dx.NumOperators; op++ {
			e.parts[fmt.Sprintf("carrier%d", op)].Set(0)
		}
		for _, c := range r.Carriers {
			e.parts[fmt.Sprintf("carrier%d", c)].Set(1)
		}
		for from := 2; from <= dx.NumOperators; from++ {
			for to := 1; to < from; to++ {
				e.parts[fmt.Sprintf("mod%d%d", from, to)].Set(0)
			}
		}
		for _, m := range r.Mods {
			e.parts[fmt.Sprintf("mod%d%d", m[0], m[1])].Set(1)
		}
		e.parts["fbFrom"].Set(float64(r.Feedback[0]))
		e.parts["fbTo"].Set(float64(r.Feedback[1]))
	}
}

// Setting returns the control of a setting ("op3Coarse", "feedback", "custom", "mod42", "carrier1",
// "fbFrom", "program"), nil for none
func (e *Editor) Setting(name string) *panel.Param { return e.parts[name] }
