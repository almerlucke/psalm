// Package widgets is a set of plugins of the panel (modules/controls/panel), controls to play and
// shape synths with from the page:
//
//   - XY: a pad moving two settings at once (a cutoff and a resonance)
//   - Keys: a keyboard on the page, to play a synth without a MIDI keyboard (or the computer's
//     keys: A W S E D F T G Y H U J K)
//   - Pads: a grid of pads for drums and samples (the number keys hit them too)
//   - Envelope: an attack, decay, sustain and release to drag into shape
//   - Scope: the sound of a module, its wave and its spectrum
//
// They all follow the theme of the page. Every synth takes its envelopes by the same messages
// ("ampAttack", ...: see docs/synths.md), so Envelope works on all of them.
//
//	ctl.Section("Synth")
//	widgets.Keys(ctl, "Keys", widgets.Play(lead), widgets.From(48), widgets.Octaves(3))
//	widgets.XY(ctl, "Filter", widgets.Axis{Name: "cutoff", Min: 200, Max: 8000, Curve: panel.Exp, To: panel.Message(lead, "cutoff")},
//		widgets.Axis{Name: "resonance", Max: 0.9, To: panel.Message(lead, "resonance")})
//	widgets.Envelope(ctl, "Amp", lead, "amp", widgets.ADSR{Attack: 5, Decay: 200, Sustain: 0.7, Release: 300})
//	widgets.Pads(ctl, "Drums", widgets.Hit(kit, "kick", "snare", "hat", "clap"))
//	widgets.Scope(ctl, "Out", mix.Master())
package widgets

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

//go:embed widgets.js
var script string

//go:embed widgets.css
var styles string

// Plugin draws the widgets (each of them uses it)
var Plugin = panel.Plugin{Name: "widgets", JS: script, CSS: styles}

// Axis is a value a widget sets: its name on the page, its range and how it moves through it, where
// it goes, where it starts
type Axis struct {
	Name     string
	Min, Max float64
	Curve    panel.Curve // panel.Linear when empty
	To       panel.Target
	Default  float64
	Unit     string
}

func (a Axis) curve() panel.Curve {
	if a.Curve == "" {
		return panel.Linear
	}
	return a.Curve
}

func (a Axis) options() []panel.Option {
	return []panel.Option{panel.Default(a.Default), panel.Unit(a.Unit), panel.Label(a.Name)}
}

// XY adds a pad moving two values at once: x across, y up. It returns the two controls (x is the
// pad, y its part): bind them to MIDI, keep them in presets, set them from code
func XY(ctl *panel.Panel, name string, x, y Axis, opts ...panel.Option) (*panel.Param, *panel.Param) {
	ctl.Use(Plugin)
	xo := append([]panel.Option{panel.Range(x.Min, x.Max, x.curve()), panel.Data(map[string]string{"x": x.Name, "y": y.Name}),
		panel.Default(x.Default), panel.Unit(x.Unit)}, opts...)
	xp := ctl.Custom("xy", name, x.To, xo...)
	yp := xp.Part(name+" y", y.Min, y.Max, y.curve(), y.To, append(y.options(), opts...)...)
	return xp, yp
}

// keys is how a keyboard widget looks
type keys struct {
	From    int `json:"from"`
	Octaves int `json:"octaves"`
}

// KeysOption sets something of the keyboard
type KeysOption func(*keys)

// From is its lowest note (C3, 48, without it)
func From(note int) KeysOption { return func(k *keys) { k.From = note } }

// Octaves is how many octaves it has (2 without it)
func Octaves(n int) KeysOption { return func(k *keys) { k.Octaves = n } }

// Keys adds a keyboard: play gets a note when a key goes down (on, with a velocity 0-1: lower on
// the key is louder, as on a piano) and when it comes up, on the audio thread. Play makes play of
// a synth
func Keys(ctl *panel.Panel, name string, play func(note int, velocity float64, on bool), opts ...KeysOption) *panel.Param {
	ctl.Use(Plugin)
	k := keys{From: 48, Octaves: 2}
	for _, o := range opts {
		o(&k)
	}
	return ctl.Custom("keys", name, nil, panel.Data(k), panel.OnMessage(func(raw json.RawMessage) {
		var m struct {
			Note     int     `json:"note"`
			Velocity float64 `json:"velocity"`
			On       bool    `json:"on"`
		}
		if json.Unmarshal(raw, &m) == nil && m.Note >= 0 && m.Note < 128 {
			play(m.Note, max(0, min(m.Velocity, 1)), m.On)
		}
	}))
}

// Play plays a synth (anything taking trigger messages: a poly synth, the piano): a note on holds
// a note, its note off lets it go
func Play(target psalm.Receiver) func(note int, velocity float64, on bool) {
	return func(note int, velocity float64, on bool) {
		id := fmt.Sprintf("keys-%d", note)
		if !on {
			target.ReceiveMessage(psalm.Message{"command": "trigger", "noteOff": id, "tick": psalm.CtrlTick{}})
			return
		}
		target.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": id, "tick": psalm.CtrlTick{}, "amplitude": velocity,
			"message": psalm.Message{"pitch": float64(note), "velocity": velocity}})
	}
}

// pads is how a grid of pads looks
type pads struct {
	Names   []string `json:"names"`
	Columns int      `json:"columns"`
}

// Hitter is what pads play: the names on the pads, and what a hit on one does (on the audio
// thread, a velocity 0-1)
type Hitter struct {
	Names []string
	Hit   func(pad int, velocity float64)
}

// Hit plays the sounds of a kit by name (a drumsynth, the drums of a sample kit): a pad for each
func Hit(kit psalm.Receiver, sounds ...string) Hitter {
	return Hitter{Names: sounds, Hit: func(pad int, velocity float64) {
		kit.ReceiveMessage(psalm.Message{"command": "trigger", "duration": 0.0, "tick": psalm.CtrlTick{}, "amplitude": 1.0,
			"message": psalm.Message{"sound": sounds[pad], "velocity": velocity}})
	}}
}

// Pads adds a grid of pads, four across (or columns): a hit is louder in the middle of a pad. The
// number keys 1 to 9 hit the first nine
func Pads(ctl *panel.Panel, name string, h Hitter, columns ...int) *panel.Param {
	ctl.Use(Plugin)
	p := pads{Names: h.Names, Columns: 4}
	if len(columns) > 0 && columns[0] > 0 {
		p.Columns = columns[0]
	}
	return ctl.Custom("pads", name, nil, panel.Data(p), panel.OnMessage(func(raw json.RawMessage) {
		var m struct {
			Pad      int     `json:"pad"`
			Velocity float64 `json:"velocity"`
		}
		if json.Unmarshal(raw, &m) == nil && m.Pad >= 0 && m.Pad < len(h.Names) {
			h.Hit(m.Pad, max(0, min(m.Velocity, 1)))
		}
	}))
}

// ADSR is an envelope: the attack, decay and release in milliseconds, the sustain a level 0-1
type ADSR struct {
	Attack, Decay, Sustain, Release float64
}

// Envelope adds an envelope to drag into shape, sent to a synth as "<name>Attack", "<name>Decay",
// "<name>Sustain" and "<name>Release" ("amp" its loudness, "filter" its filter: the envelopes
// every synth takes, see docs/synths.md). It starts at def; options (panel.NoDefault: the synth
// keeps its own envelope until it is moved) apply to its four values. It returns the attack, with
// the others as its parts
func Envelope(ctl *panel.Panel, name string, synth psalm.Receiver, envelope string, def ADSR, opts ...panel.Option) *panel.Param {
	ctl.Use(Plugin)
	to := func(stage string) panel.Target {
		key := envelope + stage
		return panel.Func(func(v float64) { synth.ReceiveMessage(psalm.Message{key: v}) })
	}
	with := func(o ...panel.Option) []panel.Option { return append(o, opts...) }
	a := ctl.Custom("envelope", name, to("Attack"), with(panel.Range(0.5, 5000, panel.Exp), panel.Default(def.Attack), panel.Unit("ms"))...)
	a.Part(name+" decay", 1, 5000, panel.Exp, to("Decay"), with(panel.Default(def.Decay), panel.Unit("ms"))...)
	a.Part(name+" sustain", 0, 1, panel.Linear, to("Sustain"), with(panel.Default(def.Sustain))...)
	a.Part(name+" release", 1, 10000, panel.Exp, to("Release"), with(panel.Default(def.Release), panel.Unit("ms"))...)
	return a
}

// scopeSize is the samples a scope shows (and its spectrum is of)
const scopeSize = 1024

// tap keeps the last samples of a module's output (both channels mixed), written on the audio
// thread, read by the page
type tap struct {
	module psalm.Modular
	mu     sync.Mutex
	ring   [2 * scopeSize]float32
	pos    int
	sr     float64
}

// take reads the module's last buffer (on the audio thread, never waiting)
func (t *tap) take() {
	out := t.module.Output(0)
	if out == nil || len(out.Channels) == 0 || !t.mu.TryLock() {
		return
	}
	defer t.mu.Unlock()
	t.sr = t.module.Config().SampleRate
	chs := out.Channels
	for i := range chs[0] {
		s := float32(0)
		for _, ch := range chs {
			s += ch[i]
		}
		t.ring[t.pos] = s / float32(len(chs))
		t.pos = (t.pos + 1) % len(t.ring)
	}
}

// scope is what the page gets: the sample rate and the last samples
type scope struct {
	SampleRate float64   `json:"sr"`
	Samples    []float64 `json:"samples"`
}

func (t *tap) state() any {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := scope{SampleRate: t.sr, Samples: make([]float64, scopeSize)}
	start := (t.pos - scopeSize + len(t.ring)) % len(t.ring)
	for i := range s.Samples {
		s.Samples[i] = math.Round(float64(t.ring[(start+i)%len(t.ring)])*1e4) / 1e4
	}
	return s
}

// Scope adds a view of the sound of a module (a synth, a mixer strip, the master): its wave,
// steady on its rising zero crossings, and its spectrum. A click switches between the wave, the
// spectrum and both
func Scope(ctl *panel.Panel, name string, m psalm.Modular) *panel.Param {
	ctl.Use(Plugin)
	t := &tap{module: m}
	return ctl.Custom("scope", name, nil, panel.OnBuffer(t.take), panel.Live(t.state))
}

// Readout adds a line of text the program writes: f is asked for it every time the page is updated (on the page's goroutine, so keep it quick), for
// the time of a song at the top of the page (patcher.SongClock). With Top it is small, beside the volume
func Readout(ctl *panel.Panel, name string, f func() string) *panel.Param {
	ctl.Use(Plugin)
	return ctl.Custom("readout", name, nil, panel.Live(func() any { return f() }))
}
