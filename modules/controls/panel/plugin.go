package panel

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// Plugin brings controls of its own to the page: a script that draws them, styles, and files they
// use. The script registers a builder for each kind of control it draws:
//
//	psalm.register("xy", (p, api) => {
//		const el = document.createElement("div");
//		...                                 // p: the control (name, label, min, max, value, data, state)
//		return {
//			el,                             // shown on the page
//			show(value) {},                 // the value changed (moved here, elsewhere or from code)
//			update(state) {},               // the state the Go side gives changed (Push, Live)
//			bindable: true,                 // MIDI learn can bind it (its value)
//		};
//	});
//
// api.send(name, value) sets the value of a control, api.press(name) presses a button,
// api.message(name, data) sends any JSON to the control's OnMessage. api.labelled(p, el, valueEl)
// puts an element under the control's name as the page's own controls are, api.drag, api.position,
// api.valueAt and api.format are the page's own helpers, api.css(name) reads a variable of the
// theme ("accent").
//
// A control of a plugin is added with Custom. The script is served at plugins/<name>/plugin.js,
// the styles at plugins/<name>/plugin.css, Files under plugins/<name>/ (a script can load them
// relative to its own address, import.meta isn't there: use "plugins/<name>/file")
type Plugin struct {
	Name  string
	JS    string
	CSS   string
	Files fs.FS
}

// Use adds a plugin to the page (a plugin used twice is there once)
func (pn *Panel) Use(p Plugin) *Panel {
	if p.Name == "" || strings.ContainsAny(p.Name, "/?#") {
		panic(fmt.Sprintf("panel: a plugin needs a name without / ? # (%q)", p.Name))
	}
	pn.mu.Lock()
	defer pn.mu.Unlock()
	if i := slices.IndexFunc(pn.plugins, func(q Plugin) bool { return q.Name == p.Name }); i >= 0 {
		pn.plugins[i] = p
		return pn
	}
	pn.plugins = append(pn.plugins, p)
	return pn
}

// pluginDesc is a plugin as the page sees it
type pluginDesc struct {
	Name   string `json:"name"`
	Script bool   `json:"script"`
	Style  bool   `json:"style"`
}

func (pn *Panel) pluginList() []Plugin {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return append([]Plugin(nil), pn.plugins...)
}

// servePlugin serves the script, styles and files of a plugin
func (pn *Panel) servePlugin(w http.ResponseWriter, r *http.Request) {
	name, file := r.PathValue("plugin"), r.PathValue("file")
	i := slices.IndexFunc(pn.pluginList(), func(p Plugin) bool { return p.Name == name })
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	p := pn.pluginList()[i]
	switch file {
	case "plugin.js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write([]byte(p.JS))
	case "plugin.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write([]byte(p.CSS))
	default:
		if p.Files == nil {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, p.Files, file)
	}
}

// builtin are the kinds the page draws itself
var builtin = []Kind{FaderKind, KnobKind, ToggleKind, ButtonKind, SelectKind, MeterKind, GridKind, CuesKind, MountKind, PartKind}

// PartKind is the kind of a part of a custom control (Part)
const PartKind Kind = "part"

// Part adds a value to a custom control: a control of its own (a name, a range, a target, kept
// in snapshots and presets, set with api.send on the page) that the control's plugin draws, not
// shown by itself. The second axis of an XY pad, the decay of an envelope. The plugin's builder
// gets them in p.parts, and its part(name, value) is called when one changes
func (par *Param) Part(name string, min, max float64, curve Curve, to Target, opts ...Option) *Param {
	if par.custom == nil || par.panel == nil {
		panic(fmt.Sprintf("panel: %q isn't a custom control on a panel: it can't have parts", par.Name))
	}
	part := newParam(PartKind, name, min, max, curve, to, opts)
	pn := par.panel
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.addLocked(part, false)
	par.parts = append(par.parts, part)
	return part
}

// Parts returns the parts of a custom control
func (par *Param) Parts() []*Param { return append([]*Param(nil), par.parts...) }

// Custom adds a control a plugin draws (Use the plugin too): a kind of its own, a value like a
// fader's (0 to 1 unless Range says otherwise) bound to a target (nil for none), and what the
// options give it: Data (anything the plugin needs to draw it, sent once), Live or Push (a state
// that changes, sent when it does), OnMessage (what the plugin sends back). Its value is in
// snapshots and presets like any other
func (pn *Panel) Custom(kind Kind, name string, to Target, opts ...Option) *Param {
	if kind == "" || slices.Contains(builtin, kind) {
		panic(fmt.Sprintf("panel: %q is not a kind of its own for a custom control", kind))
	}
	par := newParam(kind, name, 0, 1, Linear, to, opts)
	if par.custom == nil {
		par.custom = &custom{}
	}
	return pn.add(par)
}

// custom is what a control of a plugin has besides its value
type custom struct {
	data      json.RawMessage
	live      func() any
	onMessage func(data json.RawMessage)
	onBuffer  func()

	mu       sync.Mutex
	state    json.RawMessage // pushed
	version  atomic.Int64
	messages []json.RawMessage // from the page, for the audio thread
}

func (par *Param) customPart() *custom {
	if par.custom == nil {
		par.custom = &custom{}
	}
	return par.custom
}

// Range sets the range of a custom control's value (a fader's range: min, max and a curve)
func Range(min, max float64, curve Curve) Option {
	if curve == Exp && (min <= 0 || max <= 0) {
		panic(fmt.Sprintf("panel: an exponential range has to be above 0 (%v to %v)", min, max))
	}
	return func(par *Param) { par.Min, par.Max, par.Curve = min, max, curve }
}

// Data gives a custom control what its plugin needs to draw it (anything that turns into JSON),
// sent to the page once with the control (p.data)
func Data(v any) Option {
	return func(par *Param) {
		b, err := json.Marshal(v)
		if err != nil {
			panic(fmt.Sprintf("panel: the data of %q: %v", par.Name, err))
		}
		par.customPart().data = b
	}
}

// Live gives a custom control a state its plugin shows (anything that turns into JSON): f is asked
// for it every time the page is updated (on the page's goroutine, twenty times a second while a
// page is open, so keep it quick and safe to call from there), and the plugin's update gets it
// when it changed
func Live(f func() any) Option {
	return func(par *Param) { par.customPart().live = f }
}

// OnMessage is called with what the plugin of a custom control sends (api.message), on the audio
// thread like the other controls' changes
func OnMessage(f func(data json.RawMessage)) Option {
	return func(par *Param) { par.customPart().onMessage = f }
}

// OnBuffer is called on the audio thread at the start of every buffer, for a custom control that
// follows the sound (a scope reading the output of a module, as the meters do). Keep it quick
func OnBuffer(f func()) Option {
	return func(par *Param) { par.customPart().onBuffer = f }
}

// Push gives a custom control a new state for its plugin (anything that turns into JSON); the
// pages get it with their next update. From any goroutine
func (par *Param) Push(state any) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	c := par.customPart()
	c.mu.Lock()
	c.state = b
	c.mu.Unlock()
	c.version.Add(1)
	return nil
}

// stateNow returns the state of a custom control: the live one, else the one pushed
func (c *custom) stateNow() json.RawMessage {
	if c.live != nil {
		b, err := json.Marshal(c.live())
		if err == nil {
			return b
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// receive keeps a message from the page for the audio thread
func (c *custom) receive(data json.RawMessage) {
	if c.onMessage == nil {
		return
	}
	c.mu.Lock()
	c.messages = append(c.messages, append(json.RawMessage(nil), data...))
	c.mu.Unlock()
}

// deliver hands the messages from the page to OnMessage (on the audio thread, never waiting)
func (c *custom) deliver() {
	if c.onMessage == nil || !c.mu.TryLock() {
		return
	}
	msgs := c.messages
	c.messages = nil
	c.mu.Unlock()
	for _, m := range msgs {
		c.onMessage(m)
	}
}
