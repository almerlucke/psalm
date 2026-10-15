// Package editor is a plugin of the panel (modules/controls/panel): the controls of any synth or
// effect that describes itself (params), drawn as cards by group: a knob for every value, a switch
// for every switch, a choice for every choice, and its envelopes to drag into shape. Every setting
// is a value of the panel (named "<name> <key>"), so presets keep them, MIDI learn binds them
// (learn mode, tap one), code sets them.
//
//	lead := subtractive.NewPoly(subtractive.LeadPreset(), 12).AddTo(p)
//	editor.Add(ctl, "Lead", lead, subtractive.Describe(subtractive.LeadPreset()))
//	editor.Add(ctl, "Chorus", ch, chorus.Describe(ch.Settings()))
package editor

import (
	_ "embed"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/params"
)

//go:embed editor.js
var script string

//go:embed editor.css
var styles string

// Plugin is the plugin (Add uses it, and the widgets for the envelopes)
var Plugin = panel.Plugin{Name: "editor", JS: script, CSS: styles}

// Kind is the kind of its controls
const Kind panel.Kind = "editor"

// Receiver is what the settings are sent to: a synth, an effect (anything taking messages)
type Receiver interface {
	Receive(value any, index int)
}

// Editor is the controls of a synth or effect
type Editor struct {
	*panel.Param
	parts map[string]*panel.Param // by message key ("cutoff", "ampAttack")
	to    Receiver
	store *Store
	base  atomic.Value // the factory preset chosen last (string)
}

// param is a setting as the page sees it
type param struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Group   string   `json:"group"`
	Kind    string   `json:"kind"` // knob, switch, choice
	Options []string `json:"options,omitempty"`
}

// envelope is an envelope as the page sees it
type envelope struct {
	Prefix string `json:"prefix"`
	Label  string `json:"label"`
	Group  string `json:"group"`
	Stages string `json:"stages"`
}

type data struct {
	Name      string     `json:"name"`
	Groups    []string   `json:"groups"`
	Params    []param    `json:"params"`
	Envelopes []envelope `json:"envelopes"`
}

// the ranges of the envelopes' stages
var stages = map[byte]struct {
	key      string
	min, max float64
	curve    panel.Curve
	unit     string
}{
	'A': {"Attack", 0.5, 5000, panel.Exp, "ms"},
	'D': {"Decay", 1, 10000, panel.Exp, "ms"},
	'S': {"Sustain", 0, 1, panel.Linear, ""},
	'R': {"Release", 1, 20000, panel.Exp, "ms"},
}

// Add adds the controls of a synth or effect described by set (its Describe) to a panel, sending
// every change to it. The settings start where the description has them (and are sent there when
// the patch starts, unless panel.NoDefault is among the options: then the synth keeps its own
// until one is moved). Options apply to every setting
func Add(ctl *panel.Panel, name string, to Receiver, set params.Set, opts ...panel.Option) *Editor {
	ctl.Use(widgets.Plugin)
	ctl.Use(Plugin)
	d := data{Name: set.Name, Groups: set.Groups(), Params: []param{}, Envelopes: []envelope{}}
	for _, p := range set.Params {
		kind := map[params.Kind]string{params.Knob: "knob", params.Switch: "switch", params.Choice: "choice"}[p.Kind]
		label := p.Label
		if label == "" {
			label = p.Key
		}
		d.Params = append(d.Params, param{Key: p.Key, Label: label, Group: p.Group, Kind: kind, Options: p.Options})
	}
	for _, e := range set.Envelopes {
		d.Envelopes = append(d.Envelopes, envelope{Prefix: e.Prefix, Label: e.Label, Group: e.Group, Stages: e.Stages})
	}
	e := &Editor{parts: map[string]*panel.Param{}, to: to}
	e.store = NewStore(func() map[string]float64 {
		v := map[string]float64{}
		for key, c := range e.parts {
			v[key] = c.Value()
		}
		return v
	}, func(settings map[string]float64) {
		for key, v := range settings {
			if c := e.parts[key]; c != nil {
				c.Set(v)
			}
		}
	})
	e.Param = ctl.Custom(Kind, name, nil, panel.Data(d), panel.NoDefault(),
		panel.Live(e.store.state), panel.OnMessage(e.store.message))
	send := func(key string) panel.Target {
		return panel.Func(func(v float64) { to.Receive(psalm.Message{key: v}, 0) })
	}
	with := func(o ...panel.Option) []panel.Option { return append(o, opts...) }
	for _, p := range set.Params {
		curve := panel.Linear
		if p.Exp {
			curve = panel.Exp
		}
		label := p.Label
		if label == "" {
			label = p.Key
		}
		e.parts[p.Key] = e.Part(name+" "+p.Key, p.Min, p.Max, curve, send(p.Key),
			with(panel.Default(p.Value), panel.Step(p.Step), panel.Unit(p.Unit), panel.Label(label))...)
	}
	for _, env := range set.Envelopes {
		values := map[byte]float64{'A': env.Attack, 'D': env.Decay, 'S': env.Sustain, 'R': env.Release}
		for i := range len(env.Stages) {
			st, ok := stages[env.Stages[i]]
			if !ok {
				continue
			}
			key := env.Prefix + st.key
			e.parts[key] = e.Part(name+" "+key, st.min, st.max, st.curve, send(key),
				with(panel.Default(values[env.Stages[i]]), panel.Unit(st.unit), panel.Label(st.key))...)
		}
	}
	return e
}

// Factory gives the editor the presets of the synth (or effect) to choose from: their names, and
// how one is described (its Describe of the preset's config)
func (e *Editor) Factory(names []string, describe func(name string) params.Set) *Editor {
	e.store.SetFactory(names, func(name string) { e.Load(describe(name)) })
	return e
}

// FactoryApply is Factory for a synth that is more than its settings: choosing a preset moves the
// controls to its description without sending it (the synth is not told setting by setting), and
// calls apply with its name to make the synth that preset (poly.Swap: build one, swap its voices in).
// A preset is made of more than a description says (the model of a string, a list of modes, a
// word), which messages of settings leave as the preset before had it. The controls end up showing
// the described values, and the synth is the one made at the preset. apply is called from the
// goroutine of the load (the audio thread when the page chose it): it must not wait
func (e *Editor) FactoryApply(names []string, describe func(name string) params.Set, apply func(name string)) *Editor {
	e.store.SetFactory(names, func(name string) {
		e.Place(describe(name))
		e.base.Store(name)
		if apply != nil {
			apply(name)
		}
	})
	return e
}

// Base returns the factory preset chosen last with FactoryApply ("" for none): what the synth is
// made at, the settings kept apart from it are the ones moved since
func (e *Editor) Base() string {
	s, _ := e.base.Load().(string)
	return s
}

// Keep keeps presets of your own in a folder (made when it isn't there): the page saves the
// settings under a name, loads them back, deletes them
func (e *Editor) Keep(dir string) error { return e.store.keep(dir) }

// Presets returns the presets of the editor, to save, load and delete them from code
func (e *Editor) Presets() *Store { return e.store }

// Values returns every setting as it is, by message key (what a preset of your own keeps)
func (e *Editor) Values() map[string]float64 { return e.store.values() }

// Apply moves the settings to values by message key (from any goroutine); keys it doesn't have are
// left out
func (e *Editor) Apply(values map[string]float64) { e.store.apply(values) }

// Setting returns the control of a setting by its message key ("cutoff", "ampRelease"), nil for none
func (e *Editor) Setting(key string) *panel.Param { return e.parts[key] }

// Load moves every setting to where another description has them (a preset of the synth): the
// synth gets them as they move. From any goroutine
func (e *Editor) Load(set params.Set) { e.move(set, (*panel.Param).Set) }

// Place is Load without sending: the controls show the description, the synth is left as it is
func (e *Editor) Place(set params.Set) { e.move(set, (*panel.Param).Place) }

func (e *Editor) move(set params.Set, set1 func(*panel.Param, float64)) {
	for _, p := range set.Params {
		if c := e.parts[p.Key]; c != nil {
			set1(c, p.Value)
		}
	}
	for _, env := range set.Envelopes {
		values := map[byte]float64{'A': env.Attack, 'D': env.Decay, 'S': env.Sustain, 'R': env.Release}
		for i := range len(env.Stages) {
			if st, ok := stages[env.Stages[i]]; ok {
				if c := e.parts[env.Prefix+st.key]; c != nil {
					set1(c, values[env.Stages[i]])
				}
			}
		}
	}
}
