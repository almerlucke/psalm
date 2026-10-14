// Package patcher is a plugin of the panel (modules/controls/panel): a modular patch made on the
// page. Add modules from a palette (any kind given: synths, effects, a keyboard, LFOs, a mixer),
// move them around, and draw cables between their ports:
//
//   - audio, from an output to an input (the sound),
//   - notes, from a keyboard to the synths it plays,
//   - modulation, from an LFO, a random source or the mod wheel to any setting of a module (what
//     it describes of itself, params), over a part of its range.
//
// Double-tap a module for its side panel: its settings (its editor) and its script; click a cable to take
// it away, Delete takes the modules chosen away. Patches are kept by name in a folder: the modules, where they are,
// the cables and every setting. Every change is made between two buffers while it plays.
//
//	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(), patcher.Keep("patches"))
//	mix.Channel("patch", pt.Out(), mixer.StripConfig{})
package patcher

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/modules/effects/gain"
	"github.com/almerlucke/psalm/params"
)

//go:embed patcher.js
var script string

//go:embed patcher.css
var styles string

// Plugin is the plugin (New uses it)
var Plugin = panel.Plugin{Name: "patcher", JS: script, CSS: styles}

// PortType is what goes through a port: sound, notes or modulation
type PortType string

const (
	Audio   PortType = "audio"
	Notes   PortType = "notes"
	Mod     PortType = "mod"
	Trigger PortType = "trigger" // a moment (an Event), on a value connection
	Value   PortType = "value"   // a number (or a chord: []float64), on a value connection
)

// Event is what a trigger cable carries: the control tick it falls on (its sample offset in the
// buffer), the step of the clock that sent it and how long a step lasts
type Event struct {
	Tick    psalm.CtrlTick
	Step    int64
	Seconds float64
}

// Text is a setting of text, typed on the module itself (a pattern, a list of values)
type Text struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
	Lines int    `json:"lines,omitempty"` // more than one: a box of that many lines (a script)
}

// Port is a port of a kind of module: its name on the page, what goes through it, its index
// (the module's input or output, for audio; the notes or mod output it is)
type Port struct {
	Name  string   `json:"name"`
	Type  PortType `json:"type"`
	Index int      `json:"-"`
}

// Kind is a kind of module the patch can have
type Kind struct {
	Name    string
	Group   string   // where it is in the palette ("Synths", "Effects", "Sources", "Modulation", "Mix")
	Presets []string // to choose from when it is added (none: it has none)
	Ins     []Port
	Outs    []Port
	// Make makes one at a preset (off the audio thread)
	Make func(preset string) (*Made, error)
}

// Made is a module made: what the patch adds, and what the patcher asks of it
type Made struct {
	Module psalm.Modular   // its audio ports are the module's inputs and outputs
	Extra  []psalm.Modular // more for the patch (a keyboard's own modules)
	// Describe describes its settings at a preset (nil: it has none to edit or modulate); the
	// presets of the kind are its factory presets in its editor
	Describe func(preset string) params.Set
	// Edit makes an editor of its own on its page instead (the DX7's); it returns the settings
	Edit func(page *panel.Panel, name, preset string) Settings
	// Notes plays the targets of its notes output (a keyboard), on the audio thread
	Notes func(targets []psalm.Receiver)
	// Mod returns what a mod output is now, 0-1 (asked on the audio thread, once a buffer)
	Mod func(out int) func() float64
	// Close lets go of what it holds when it is taken away (a MIDI device)
	Close func()
	// Ins and Outs are its ports when they aren't its kind's (a mixer of as many channels as its
	// preset says); nil: the kind's
	Ins, Outs []Port
	// Texts are its settings of text, as they start; SetText changes one (off the audio thread: it
	// parses there, the module takes what it made on the audio thread); an error leaves it as it was
	Texts   []Text
	SetText func(key, value string) error
	// Late is called on every control tick after every module had its own (a Notes module plays the
	// triggers of the tick there, when the values feeding it are up to date)
	Late func(tick psalm.CtrlTick)
	// Status is a line shown on the module (a script's error or what it logged; "" nothing), asked
	// when the page is updated
	Status func() string
}

// Settings are the settings of a module's editor: what a patch saves of it
type Settings interface {
	Values() map[string]float64
	Apply(values map[string]float64)
}

// node is a module of the patch
type node struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	Preset string  `json:"preset,omitempty"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`

	kind     *Kind
	made     *Made
	page     *panel.Panel
	settings Settings
	set      params.Set        // its settings described (for modulation: their ranges)
	texts    map[string]string // its settings of text as they are
	textErr  map[string]string // what was wrong with one typed (it kept the one before)
}

// Cable is a cable between two ports: the output Out of From, the input In of To (for a mod
// cable: the setting Key of To, moved from Lo to Hi of its range, 0-1)
type Cable struct {
	ID   string   `json:"id"`
	Type PortType `json:"type"`
	From string   `json:"from"`
	Out  int      `json:"out"`
	To   string   `json:"to"`
	In   int      `json:"in"`
	Key  string   `json:"key,omitempty"`
	Lo   float64  `json:"lo,omitempty"`
	Hi   float64  `json:"hi,omitempty"`
}

// Patcher is a patch made on the page
type Patcher struct {
	*panel.Param
	p     *psalm.Psalm
	ctl   *panel.Panel
	theme panel.Theme
	kinds []Kind
	out   *gain.Gain // the output: what the patch plays

	mu       sync.Mutex // the patch as the page sees it (the audio thread has its own copies)
	nodes    []*node
	cables   []*Cable
	next     int
	selected string
	note     string
	dir      string // patches kept here
	presets  string // presets of your own of the modules' editors, a folder for every kind
	current  string // the patch loaded or saved last

	selectedPage atomic.Pointer[panel.Panel]
	mods         atomic.Pointer[[]*modConn] // read on the audio thread
	work         chan json.RawMessage
	late         *lateStep // kept last in the patch
}

// modConn is a mod cable on the audio thread
type modConn struct {
	value  func() float64
	target psalm.Modular
	key    string
	at     func(v float64) float64 // the setting at a value 0-1
	last   float64
}

// Option sets something of the patcher
type Option func(*Patcher)

// Keep keeps patches in a folder (made when it isn't there)
func Keep(dir string) Option { return func(pt *Patcher) { pt.dir = dir } }

// Presets keeps the presets of your own of the modules' editors in a folder, a folder for every
// kind (the same as the editors example's: editors-presets)
func Presets(dir string) Option { return func(pt *Patcher) { pt.presets = dir } }

// Theme gives the modules' pages a theme
func Theme(t panel.Theme) Option { return func(pt *Patcher) { pt.theme = t } }

// kindDesc is a kind as the page sees it
type kindDesc struct {
	Name    string   `json:"name"`
	Group   string   `json:"group"`
	Presets []string `json:"presets"`
	Ins     []Port   `json:"ins"`
	Outs    []Port   `json:"outs"`
}

// New adds a patcher to a panel (with its plugin): the patch, a module's settings (its editor, its
// script) in a side panel when it is chosen; its output Out
func New(p *psalm.Psalm, ctl *panel.Panel, name string, kinds []Kind, opts ...Option) *Patcher {
	ctl.Use(Plugin)
	pt := &Patcher{p: p, ctl: ctl, kinds: kinds, out: gain.New(0), work: make(chan json.RawMessage, 256), theme: panel.Themes["default"]}
	for _, o := range opts {
		o(pt)
	}
	pt.out.AddTo(p)
	empty := []*modConn{}
	pt.mods.Store(&empty)
	(&engine{Module: psalm.NewControl(), pt: pt}).init().AddTo(p)
	pt.late = &lateStep{Module: psalm.NewControl()}
	pt.late.SetSelf(pt.late)
	pt.late.AddTo(p)
	if pt.dir != "" {
		_ = os.MkdirAll(pt.dir, 0o755)
	}
	var d struct {
		Kinds []kindDesc `json:"kinds"`
		Mount string     `json:"mount"` // where the page of the module chosen is served (mount/<it>/)
	}
	d.Mount = name + " module"
	for _, k := range kinds {
		d.Kinds = append(d.Kinds, kindDesc{Name: k.Name, Group: k.Group, Presets: k.Presets, Ins: k.Ins, Outs: k.Outs})
	}
	// the output, always there
	pt.nodes = append(pt.nodes, &node{ID: "out", Kind: "output", X: 900, Y: 220,
		kind: &Kind{Name: "output", Ins: []Port{{Name: "in", Type: Audio}}}, made: &Made{Module: pt.out}})
	pt.Param = ctl.Custom("patcher", name, nil, panel.Data(d), panel.NoDefault(),
		panel.Live(pt.state), panel.OnMessage(func(raw json.RawMessage) {
			select {
			case pt.work <- append(json.RawMessage(nil), raw...):
			default: // too much at once: dropped
			}
		}))
	ctl.Mount(d.Mount, selection{pt}).Hide() // the settings of the module chosen: in the side panel
	go pt.worker()
	return pt
}

// Out is what the patch plays: connect it to a mixer, or the patch's output
func (pt *Patcher) Out() psalm.Modular { return pt.out }

// selection is the mount of the page of the module chosen
type selection struct{ pt *Patcher }

func (s selection) Mounted() *panel.Panel { return s.pt.selectedPage.Load() }

// worker does what the page asks, in order (off the audio thread: making modules, reading files)
func (pt *Patcher) worker() {
	for raw := range pt.work {
		if err := pt.handle(raw); err != nil {
			pt.mu.Lock()
			pt.note = err.Error()
			pt.mu.Unlock()
		}
	}
}

// handle does one thing the page asks
func (pt *Patcher) handle(raw json.RawMessage) error {
	var m struct {
		Add    string  `json:"add"`
		Preset string  `json:"preset"`
		Move   string  `json:"move"`
		Remove string  `json:"remove"`
		Select *string `json:"select"`
		Cable  *Cable  `json:"connect"`
		Unplug string  `json:"disconnect"`
		Range  *Cable  `json:"range"`
		Text   *struct {
			ID, Key, Value string
		} `json:"text"`
		Save   *string     `json:"save"`
		Load   *string     `json:"load"`
		Delete *string     `json:"delete"`
		Clear  bool        `json:"clear"`
		Pos    *[2]float64 `json:"pos"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	pt.mu.Lock()
	pt.note = ""
	pt.mu.Unlock()
	switch {
	case m.Add != "":
		x, y := 100.0, 100.0
		if m.Pos != nil {
			x, y = m.Pos[0], m.Pos[1]
		}
		_, err := pt.Add(m.Add, m.Preset, x, y)
		return err
	case m.Move != "" && m.Pos != nil:
		pt.Move(m.Move, m.Pos[0], m.Pos[1])
	case m.Remove != "":
		return pt.Remove(m.Remove)
	case m.Select != nil:
		pt.Select(*m.Select)
	case m.Cable != nil:
		_, err := pt.Connect(*m.Cable)
		return err
	case m.Unplug != "":
		pt.Disconnect(m.Unplug)
	case m.Range != nil:
		return pt.SetRange(m.Range.ID, m.Range.Key, m.Range.Lo, m.Range.Hi)
	case m.Text != nil:
		return pt.SetText(m.Text.ID, m.Text.Key, m.Text.Value)
	case m.Save != nil:
		return pt.Save(*m.Save)
	case m.Load != nil:
		return pt.Load(*m.Load)
	case m.Delete != nil:
		return pt.DeletePatch(*m.Delete)
	case m.Clear:
		pt.Clear()
	}
	return nil
}

// later is a patch that keeps what is added to it, to add it to the real one between two buffers
type later struct {
	psalm.Patcher
	added []psalm.Modular
}

func (l *later) Add(m psalm.Modular) psalm.Patcher {
	l.added = append(l.added, m)
	return l
}

// kind returns a kind by name
func (pt *Patcher) kind(name string) *Kind {
	for i := range pt.kinds {
		if pt.kinds[i].Name == name {
			return &pt.kinds[i]
		}
	}
	return nil
}

// build makes a module of a kind and its page (off the audio thread), not in the patch yet
func (pt *Patcher) build(id, kindName, preset string) (*node, []psalm.Modular, error) {
	k := pt.kind(kindName)
	if k == nil {
		return nil, nil, fmt.Errorf("no kind of module %q", kindName)
	}
	if preset == "" && len(k.Presets) > 0 {
		preset = k.Presets[0]
	}
	made, err := k.Make(preset)
	if err != nil {
		return nil, nil, err
	}
	n := &node{ID: id, Kind: kindName, Preset: preset, kind: k, made: made, texts: map[string]string{}, textErr: map[string]string{}}
	for _, t := range made.Texts {
		n.texts[t.Key] = t.Value
	}
	add := append([]psalm.Modular{made.Module}, made.Extra...)
	if made.Describe != nil || made.Edit != nil {
		l := &later{}
		title := kindName
		if preset != "" {
			title += " · " + preset
		}
		n.page = panel.New(l, title)
		n.page.SetTheme(pt.theme)
		if made.Describe != nil {
			n.set = made.Describe(preset) // what a mod cable can move
		}
		if made.Edit != nil {
			n.settings = made.Edit(n.page, kindName, preset)
		} else {
			ed := editor.Add(n.page, kindName, made.Module, n.set, panel.NoDefault())
			if len(k.Presets) > 0 {
				ed.Factory(k.Presets, made.Describe)
			}
			if pt.presets != "" {
				_ = ed.Keep(filepath.Join(pt.presets, kindName))
			}
			n.settings = ed
		}
		add = append(add, l.added...)
	}
	return n, add, nil
}

// Add adds a module of a kind at a preset (the kind's first without), at a place on the patch; it
// returns its id. From any goroutine
func (pt *Patcher) Add(kindName, preset string, x, y float64) (string, error) {
	pt.mu.Lock()
	pt.next++
	id := "m" + strconv.Itoa(pt.next)
	pt.mu.Unlock()
	n, add, err := pt.build(id, kindName, preset)
	if err != nil {
		return "", err
	}
	n.X, n.Y = x, y
	pt.do(func() {
		for _, m := range add {
			pt.p.Add(m)
		}
		pt.p.Remove(pt.late) // the late step last again
		pt.p.Add(pt.late)
		if n.made.Late != nil {
			pt.late.add(n.ID, n.made.Late)
		}
	})
	pt.mu.Lock()
	pt.nodes = append(pt.nodes, n)
	pt.mu.Unlock()
	return id, nil
}

// do makes a change to the patch between two buffers, and waits for it
func (pt *Patcher) do(f func()) {
	done := make(chan struct{})
	pt.p.Do(func() { f(); close(done) })
	<-done
}

// node returns a module by id (with the lock held)
func (pt *Patcher) node(id string) *node {
	for _, n := range pt.nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

// Move moves a module on the page
func (pt *Patcher) Move(id string, x, y float64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if n := pt.node(id); n != nil {
		n.X, n.Y = x, y
	}
}

// Select chooses the module whose settings the side panel shows (its page, served by the
// patcher's hidden mount; "" none)
func (pt *Patcher) Select(id string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.selected = ""
	pt.selectedPage.Store(nil)
	if n := pt.node(id); n != nil {
		pt.selected = id
		pt.selectedPage.Store(n.page)
	}
}

// Remove takes a module and its cables away (the output stays). From any goroutine
func (pt *Patcher) Remove(id string) error {
	if id == "out" {
		return errors.New("the output stays")
	}
	pt.mu.Lock()
	n := pt.node(id)
	var cables []string
	for _, c := range pt.cables {
		if c.From == id || c.To == id {
			cables = append(cables, c.ID)
		}
	}
	pt.mu.Unlock()
	if n == nil {
		return nil
	}
	for _, c := range cables {
		pt.Disconnect(c)
	}
	pt.do(func() {
		pt.late.remove(n.ID)
		for _, m := range append([]psalm.Modular{n.made.Module}, n.made.Extra...) {
			pt.p.Remove(m)
		}
		if n.page != nil {
			pt.p.Remove(n.page)
		}
	})
	if n.made.Close != nil {
		n.made.Close()
	}
	pt.mu.Lock()
	pt.nodes = slices.DeleteFunc(pt.nodes, func(o *node) bool { return o == n })
	if pt.selected == id {
		pt.selected = ""
		pt.selectedPage.Store(nil)
	}
	pt.mu.Unlock()
	return nil
}

// inPorts and outPorts are the ports of a module: its own, else its kind's
func (n *node) inPorts() []Port {
	if n.made != nil && n.made.Ins != nil {
		return n.made.Ins
	}
	return n.kind.Ins
}

func (n *node) outPorts() []Port {
	if n.made != nil && n.made.Outs != nil {
		return n.made.Outs
	}
	return n.kind.Outs
}

// ports returns the inputs of a module as the page sees them: its kind's, and a mod input when it
// has settings to modulate
func (n *node) ins() []Port {
	ins := slices.Clone(n.inPorts())
	if len(n.set.Params) > 0 {
		ins = append(ins, Port{Name: "mod", Type: Mod, Index: -1})
	}
	return ins
}

// Connect draws a cable (its id given back); an audio or notes cable joins ports of the same type,
// a mod cable an output of a modulation source to a setting (Key) of a module. From any goroutine
func (pt *Patcher) Connect(c Cable) (string, error) {
	pt.mu.Lock()
	from, to := pt.node(c.From), pt.node(c.To)
	if from == nil || to == nil {
		pt.mu.Unlock()
		return "", errors.New("no such module")
	}
	if c.Out < 0 || c.Out >= len(from.outPorts()) {
		pt.mu.Unlock()
		return "", errors.New("no such output")
	}
	out := from.outPorts()[c.Out]
	ins := to.ins()
	if c.In < 0 || c.In >= len(ins) {
		pt.mu.Unlock()
		return "", errors.New("no such input")
	}
	in := ins[c.In]
	if out.Type != in.Type {
		pt.mu.Unlock()
		return "", fmt.Errorf("%s doesn't go into %s", out.Type, in.Type)
	}
	if from == to && out.Type != Mod {
		pt.mu.Unlock()
		return "", errors.New("a module can't play into itself")
	}
	for _, o := range pt.cables {
		if o.From == c.From && o.Out == c.Out && o.To == c.To && o.In == c.In && o.Key == c.Key && out.Type != Mod {
			pt.mu.Unlock()
			return o.ID, nil // there already
		}
	}
	c.Type = out.Type
	if c.Type == Mod {
		p, ok := to.set.Find(c.Key)
		if c.Key == "" || !ok {
			// the first setting by default (the page asks which)
			if len(to.set.Params) == 0 {
				pt.mu.Unlock()
				return "", errors.New("nothing to modulate")
			}
			p = to.set.Params[0]
			c.Key = p.Key
		}
		if c.Lo == 0 && c.Hi == 0 {
			c.Lo, c.Hi = 0, 1
		}
	}
	pt.next++
	c.ID = "c" + strconv.Itoa(pt.next)
	cc := c
	pt.cables = append(pt.cables, &cc)
	pt.mu.Unlock()
	pt.wire(&cc, true)
	return cc.ID, nil
}

// wire makes or takes away what a cable does in the patch
func (pt *Patcher) wire(c *Cable, on bool) {
	pt.mu.Lock()
	from, to := pt.node(c.From), pt.node(c.To)
	pt.mu.Unlock()
	if from == nil || to == nil {
		return
	}
	switch c.Type {
	case Audio:
		outIdx, inIdx := from.outPorts()[c.Out].Index, to.inPorts()[c.In].Index
		pt.do(func() {
			if on {
				from.made.Module.Connect(outIdx, to.made.Module, inIdx)
			} else {
				from.made.Module.Disconnect(outIdx, to.made.Module, inIdx)
			}
		})
	case Notes:
		pt.renote(from)
	case Trigger, Value:
		outIdx, inIdx := from.outPorts()[c.Out].Index, to.inPorts()[c.In].Index
		pt.do(func() {
			if on {
				from.made.Module.ValueConnect(outIdx, to.made.Module, inIdx)
			} else {
				from.made.Module.ValueDisconnect(outIdx, to.made.Module, inIdx)
			}
		})
	case Mod:
		pt.remod()
	}
}

// renote gives a notes source the synths its cables go to
func (pt *Patcher) renote(from *node) {
	if from.made.Notes == nil {
		return
	}
	pt.mu.Lock()
	var targets []psalm.Receiver
	for _, c := range pt.cables {
		if c.Type == Notes && c.From == from.ID {
			if to := pt.node(c.To); to != nil {
				targets = append(targets, to.made.Module)
			}
		}
	}
	pt.mu.Unlock()
	pt.do(func() { from.made.Notes(targets) })
}

// remod gives the audio thread the mod cables as they are
func (pt *Patcher) remod() {
	pt.mu.Lock()
	var conns []*modConn
	for _, c := range pt.cables {
		if c.Type != Mod {
			continue
		}
		from, to := pt.node(c.From), pt.node(c.To)
		if from == nil || to == nil || from.made.Mod == nil {
			continue
		}
		p, ok := to.set.Find(c.Key)
		if !ok {
			continue
		}
		lo, hi := c.Lo, c.Hi
		at := func(v float64) float64 {
			x := lo + (hi-lo)*math.Max(0, math.Min(v, 1)) // along the travel
			if p.Exp && p.Min > 0 {
				return p.Clamp(p.Min * math.Pow(p.Max/p.Min, x))
			}
			return p.Clamp(p.Min + (p.Max-p.Min)*x)
		}
		conns = append(conns, &modConn{value: from.made.Mod(from.outPorts()[c.Out].Index), target: to.made.Module, key: c.Key, at: at, last: math.NaN()})
	}
	pt.mu.Unlock()
	pt.mods.Store(&conns)
}

// Disconnect takes a cable away. From any goroutine
func (pt *Patcher) Disconnect(id string) {
	pt.mu.Lock()
	var c *Cable
	for _, o := range pt.cables {
		if o.ID == id {
			c = o
		}
	}
	if c == nil {
		pt.mu.Unlock()
		return
	}
	pt.cables = slices.DeleteFunc(pt.cables, func(o *Cable) bool { return o == c })
	pt.mu.Unlock()
	pt.wire(c, false)
}

// SetRange sets the setting a mod cable moves and the part of its range (0-1, low to high)
func (pt *Patcher) SetRange(id, key string, lo, hi float64) error {
	pt.mu.Lock()
	var c *Cable
	for _, o := range pt.cables {
		if o.ID == id && o.Type == Mod {
			c = o
		}
	}
	if c == nil {
		pt.mu.Unlock()
		return errors.New("no such mod cable")
	}
	if to := pt.node(c.To); to != nil && key != "" {
		if _, ok := to.set.Find(key); !ok {
			pt.mu.Unlock()
			return fmt.Errorf("no setting %q", key)
		}
		c.Key = key
	}
	c.Lo, c.Hi = math.Max(0, math.Min(lo, 1)), math.Max(0, math.Min(hi, 1))
	pt.mu.Unlock()
	pt.remod()
	return nil
}

// SetText changes a setting of text of a module (a pattern, a list of values); what can't be read
// is kept to show on the module, the module going on with the one before
func (pt *Patcher) SetText(id, key, value string) error {
	pt.mu.Lock()
	n := pt.node(id)
	pt.mu.Unlock()
	if n == nil || n.made.SetText == nil {
		return errors.New("no such setting")
	}
	err := n.made.SetText(key, value)
	pt.mu.Lock()
	defer pt.mu.Unlock()
	n.texts[key] = value
	if err != nil {
		n.textErr[key] = err.Error()
	} else {
		delete(n.textErr, key)
	}
	return nil
}

// Texts returns the settings of text of a module, as they are
func (pt *Patcher) Texts(id string) []Text {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	n := pt.node(id)
	if n == nil {
		return nil
	}
	var ts []Text
	for _, t := range n.made.Texts {
		ts = append(ts, Text{Key: t.Key, Label: t.Label, Value: n.texts[t.Key], Lines: t.Lines})
	}
	return ts
}

// TextError returns what was wrong with the text last typed into a setting ("" nothing)
func (pt *Patcher) TextError(id, key string) string {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if n := pt.node(id); n != nil {
		return n.textErr[key]
	}
	return ""
}

// Status returns what a module says of itself (a script: its error, what it logged; "" nothing)
func (pt *Patcher) Status(id string) string {
	pt.mu.Lock()
	n := pt.node(id)
	pt.mu.Unlock()
	if n == nil || n.made.Status == nil {
		return ""
	}
	return n.made.Status()
}

// Clear takes every module away (the output stays)
func (pt *Patcher) Clear() {
	pt.mu.Lock()
	var ids []string
	for _, n := range pt.nodes {
		if n.ID != "out" {
			ids = append(ids, n.ID)
		}
	}
	pt.mu.Unlock()
	for _, id := range ids {
		_ = pt.Remove(id)
	}
}

// --- the page ---

// nodeState is a module as the page sees it
type nodeState struct {
	ID       string            `json:"id"`
	Kind     string            `json:"kind"`
	Preset   string            `json:"preset,omitempty"`
	X        float64           `json:"x"`
	Y        float64           `json:"y"`
	Ins      []Port            `json:"ins"`
	Outs     []Port            `json:"outs"`
	Settings []string          `json:"settings,omitempty"` // what a mod cable can move: key, label
	Labels   []string          `json:"labels,omitempty"`
	Texts    []Text            `json:"texts,omitempty"`  // its settings of text, as they are
	Errors   map[string]string `json:"errors,omitempty"` // what was wrong with one typed
	Status   string            `json:"status,omitempty"`
	Page     bool              `json:"page,omitempty"` // it has settings to show (its editor)
}

type state struct {
	Nodes    []nodeState `json:"nodes"`
	Cables   []*Cable    `json:"cables"`
	Selected string      `json:"selected"`
	Patches  []string    `json:"patches"`
	Current  string      `json:"current"`
	Keeps    bool        `json:"keeps"`
	Note     string      `json:"note"`
}

func (pt *Patcher) state() any {
	patches := pt.Patches()
	pt.mu.Lock()
	defer pt.mu.Unlock()
	st := state{Cables: []*Cable{}, Selected: pt.selected, Patches: patches, Current: pt.current, Keeps: pt.dir != "", Note: pt.note}
	for _, c := range pt.cables {
		cc := *c
		st.Cables = append(st.Cables, &cc)
	}
	for _, n := range pt.nodes {
		ns := nodeState{ID: n.ID, Kind: n.Kind, Preset: n.Preset, X: n.X, Y: n.Y, Ins: n.ins(), Outs: n.outPorts()}
		if ns.Ins == nil { // lists, also empty ones, for the page
			ns.Ins = []Port{}
		}
		if ns.Outs == nil {
			ns.Outs = []Port{}
		}
		for _, t := range n.made.Texts {
			ns.Texts = append(ns.Texts, Text{Key: t.Key, Label: t.Label, Value: n.texts[t.Key], Lines: t.Lines})
		}
		if len(n.textErr) > 0 {
			ns.Errors = maps.Clone(n.textErr)
		}
		ns.Page = n.page != nil
		if n.made.Status != nil {
			ns.Status = n.made.Status()
		}
		for _, p := range n.set.Params {
			label := p.Label
			if label == "" {
				label = p.Key
			}
			ns.Settings = append(ns.Settings, p.Key)
			ns.Labels = append(ns.Labels, p.Group+" · "+label)
		}
		st.Nodes = append(st.Nodes, ns)
	}
	return st
}

// --- patches ---

// patchName is what a patch may be called
var patchName = regexp.MustCompile(`^[\pL\pN _.-]{1,64}$`)

// patchFile is a patch on disk
type patchFile struct {
	Nodes []struct {
		node
		Settings map[string]float64 `json:"settings,omitempty"`
		Texts    map[string]string  `json:"texts,omitempty"`
	} `json:"nodes"`
	Cables []*Cable `json:"cables"`
}

func (pt *Patcher) file(name string) (string, error) {
	name = strings.TrimSpace(name)
	if pt.dir == "" {
		return "", errors.New("no folder for patches")
	}
	if !patchName.MatchString(name) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("%q can't be the name of a patch: use letters, digits, spaces, - _ and .", name)
	}
	return filepath.Join(pt.dir, name+".json"), nil
}

// Patches returns the names of the patches kept, sorted
func (pt *Patcher) Patches() []string {
	if pt.dir == "" {
		return nil
	}
	entries, err := os.ReadDir(pt.dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if n, ok := strings.CutSuffix(e.Name(), ".json"); ok && !e.IsDir() {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	return names
}

// Save keeps the patch under a name: its modules, where they are, its cables, every setting
func (pt *Patcher) Save(name string) error {
	path, err := pt.file(name)
	if err != nil {
		return err
	}
	var f patchFile
	pt.mu.Lock()
	for _, n := range pt.nodes {
		e := struct {
			node
			Settings map[string]float64 `json:"settings,omitempty"`
			Texts    map[string]string  `json:"texts,omitempty"`
		}{node: *n}
		if n.settings != nil {
			e.Settings = n.settings.Values()
		}
		if len(n.texts) > 0 {
			e.Texts = maps.Clone(n.texts)
		}
		f.Nodes = append(f.Nodes, e)
	}
	f.Cables = pt.cables
	data, err := json.MarshalIndent(f, "", "\t")
	pt.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	pt.mu.Lock()
	pt.current = strings.TrimSpace(name)
	pt.mu.Unlock()
	return nil
}

// Load makes the patch kept under a name (the modules there now go)
func (pt *Patcher) Load(name string) error {
	path, err := pt.file(name)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f patchFile
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	pt.Clear()
	ids := map[string]string{"out": "out"} // the ids in the file, the ids now
	var failed []string
	for _, n := range f.Nodes {
		if n.ID == "out" {
			pt.Move("out", n.X, n.Y)
			continue
		}
		id, err := pt.Add(n.Kind, n.Preset, n.X, n.Y)
		if err != nil {
			failed = append(failed, n.Kind+": "+err.Error())
			continue
		}
		ids[n.ID] = id
		for key, value := range n.Texts {
			_ = pt.SetText(id, key, value)
		}
		if len(n.Settings) > 0 {
			pt.mu.Lock()
			made := pt.node(id)
			pt.mu.Unlock()
			if made != nil && made.settings != nil {
				made.settings.Apply(n.Settings)
			}
		}
	}
	for _, c := range f.Cables {
		from, ok1 := ids[c.From]
		to, ok2 := ids[c.To]
		if !ok1 || !ok2 {
			continue
		}
		cc := *c
		cc.From, cc.To = from, to
		if _, err := pt.Connect(cc); err != nil {
			failed = append(failed, err.Error())
		}
	}
	pt.mu.Lock()
	pt.current = strings.TrimSpace(name)
	pt.mu.Unlock()
	if len(failed) > 0 {
		return errors.New("loaded, but: " + strings.Join(failed, "; "))
	}
	return nil
}

// DeletePatch deletes a patch kept
func (pt *Patcher) DeletePatch(name string) error {
	path, err := pt.file(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// --- the late step: after every module's control tick ---

// lateStep calls the late functions of the modules (a Notes module playing the triggers of the
// tick) at the end of every control tick: it is kept the last module of the patch
type lateStep struct {
	*psalm.Module
	funcs atomic.Pointer[[]lateFunc]
}

type lateFunc struct {
	id string
	f  func(psalm.CtrlTick)
}

// add and remove change the late functions (on the audio thread, between buffers)
func (l *lateStep) add(id string, f func(psalm.CtrlTick)) {
	var fs []lateFunc
	if old := l.funcs.Load(); old != nil {
		fs = slices.Clone(*old)
	}
	fs = append(fs, lateFunc{id, f})
	l.funcs.Store(&fs)
}

func (l *lateStep) remove(id string) {
	old := l.funcs.Load()
	if old == nil {
		return
	}
	fs := slices.DeleteFunc(slices.Clone(*old), func(f lateFunc) bool { return f.id == id })
	l.funcs.Store(&fs)
}

func (l *lateStep) CtrlTick(tick psalm.CtrlTick) {
	if fs := l.funcs.Load(); fs != nil {
		for _, f := range *fs {
			f.f(tick)
		}
	}
}

// --- modulation on the audio thread ---

// engine sends the mod cables' values to their settings once a buffer
type engine struct {
	*psalm.Module
	pt     *Patcher
	buffer int64
}

func (e *engine) init() *engine {
	e.buffer = -1
	e.SetSelf(e)
	return e
}

func (e *engine) CtrlTick(tick psalm.CtrlTick) {
	buffer := tick.SampleTimestamp / int64(max(1, e.Config().BufferSize))
	if buffer == e.buffer {
		return
	}
	e.buffer = buffer
	for _, c := range *e.pt.mods.Load() {
		v := c.at(c.value())
		if v == c.last {
			continue
		}
		c.last = v
		c.target.Receive(psalm.Message{c.key: v}, 0)
	}
}

// NodeInfo is a module of the patch: its id, kind, preset and place
type NodeInfo struct {
	ID, Kind, Preset string
	X, Y             float64
}

// Nodes returns the modules of the patch (the output first)
func (pt *Patcher) Nodes() []NodeInfo {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	var ns []NodeInfo
	for _, n := range pt.nodes {
		ns = append(ns, NodeInfo{ID: n.ID, Kind: n.Kind, Preset: n.Preset, X: n.X, Y: n.Y})
	}
	return ns
}

// Cables returns the cables of the patch
func (pt *Patcher) Cables() []Cable {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	var cs []Cable
	for _, c := range pt.cables {
		cs = append(cs, *c)
	}
	return cs
}

// Module returns the module of a node (nil for none)
func (pt *Patcher) Module(id string) psalm.Modular {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if n := pt.node(id); n != nil {
		return n.made.Module
	}
	return nil
}

// SettingsOf returns the settings of a module (its editor), nil for one without
func (pt *Patcher) SettingsOf(id string) Settings {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if n := pt.node(id); n != nil && n.settings != nil {
		return n.settings
	}
	return nil
}
