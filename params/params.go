// Package params describes the settings of a synth or an effect: every message it takes to change
// a setting, with its range, how it moves through it, its unit, the group it belongs to and where
// it is now; and its envelopes (the messages of blocks/envelope). A synth or an effect describes
// itself with a Describe function (subtractive.Describe(cfg), chorus.Describe(cfg)), and anything
// can be built on that: the editor of the panel (modules/controls/panel/editor) draws a control
// for each, a sequencer could pick settings to step through, a randomiser what to move.
package params

// Kind is what a setting is
type Kind int

const (
	Knob   Kind = iota // a value in a range
	Switch             // on or off: 1 or 0
	Choice             // one of Options: its index
)

// Param is a setting: the message key that changes it and what it takes
type Param struct {
	Key      string // the message key ("cutoff")
	Label    string // shown, short ("cutoff" when empty: the key)
	Group    string // what it belongs to ("Filter")
	Kind     Kind
	Min, Max float64
	Exp      bool    // moves evenly through octaves (frequencies, times): Min above 0
	Step     float64 // 1 for whole numbers, 0 any value
	Unit     string  // "Hz", "ms", "dB", "st"
	Options  []string
	Value    float64 // where it is now
	Hint     string  // what it does, in a sentence: shown when the pointer rests on it
}

// Envelope is an envelope set by the messages of blocks/envelope: "<Prefix>Attack", ...
type Envelope struct {
	Prefix string // "amp", "filter"
	Label  string // "Amp envelope"
	Group  string
	Stages string // which of A, D, S and R it has ("ADSR", "AR" for an attack and a release)
	Hint   string // what it does, in a sentence
	// where it is now: milliseconds, the sustain a level 0-1
	Attack, Decay, Sustain, Release float64
}

// Set is what a synth or an effect describes of itself
type Set struct {
	Name      string
	Params    []Param
	Envelopes []Envelope
	// GroupHints say what a group of settings is, by its name: a line under its title
	GroupHints map[string]string
}

// Option sets something of a setting
type Option func(*Param)

// Exp makes a setting move evenly through octaves (frequencies, times)
func Exp() Option { return func(p *Param) { p.Exp = true } }

// Unit gives a setting a unit
func Unit(u string) Option { return func(p *Param) { p.Unit = u } }

// Step rounds a setting to steps of a size (1 for whole numbers)
func Step(s float64) Option { return func(p *Param) { p.Step = s } }

// Hint says what a setting does (shown when the pointer rests on it)
func Hint(h string) Option { return func(p *Param) { p.Hint = h } }

// Label shows a setting by another name than its key
func Label(l string) Option { return func(p *Param) { p.Label = l } }

// K is a value in a range
func K(group, key string, min, max, value float64, opts ...Option) Param {
	p := Param{Key: key, Group: group, Kind: Knob, Min: min, Max: max, Value: value}
	for _, o := range opts {
		o(&p)
	}
	return p
}

// S is a switch, on or off
func S(group, key string, on bool, opts ...Option) Param {
	p := Param{Key: key, Group: group, Kind: Switch, Max: 1, Step: 1}
	if on {
		p.Value = 1
	}
	for _, o := range opts {
		o(&p)
	}
	return p
}

// C is a choice between options, by index
func C(group, key string, options []string, value int, opts ...Option) Param {
	p := Param{Key: key, Group: group, Kind: Choice, Max: float64(len(options) - 1), Step: 1, Options: options, Value: float64(value)}
	for _, o := range opts {
		o(&p)
	}
	return p
}

// E is an envelope with all four stages, its durations in milliseconds
func E(group, label, prefix string, attack, decay, sustain, release float64) Envelope {
	return Envelope{Prefix: prefix, Label: label, Group: group, Stages: "ADSR", Attack: attack, Decay: decay, Sustain: sustain, Release: release}
}

// Find returns a setting by its key
func (s Set) Find(key string) (Param, bool) {
	for _, p := range s.Params {
		if p.Key == key {
			return p, true
		}
	}
	return Param{}, false
}

// Groups returns the groups of the settings and envelopes in the order they first come
func (s Set) Groups() []string {
	var gs []string
	add := func(g string) {
		for _, h := range gs {
			if h == g {
				return
			}
		}
		gs = append(gs, g)
	}
	for _, p := range s.Params {
		add(p.Group)
	}
	for _, e := range s.Envelopes {
		add(e.Group)
	}
	return gs
}

// Clamp keeps a value in a setting's range and on its step
func (p Param) Clamp(v float64) float64 {
	if p.Step > 0 {
		v = p.Min + float64(int64((v-p.Min)/p.Step+0.5))*p.Step
	}
	return max(p.Min, min(v, p.Max))
}
