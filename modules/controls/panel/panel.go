// Package panel controls a patch while it plays: faders, knobs, toggles, buttons and selects,
// declared in code next to the patch and bound to the settings of its modules, and meters showing
// the levels of modules. The panel is a control module in the patch: changes made anywhere (the web
// page Serve gives, your own code on any goroutine) reach the modules on the audio thread, between
// two control ticks, and the audio thread never waits for them.
//
//	ctl := panel.New(p, "Synth")
//	ctl.Fader("Cutoff", 200, 12000, panel.Exp, panel.Message(filter, "cutoff"), panel.Default(2000), panel.Unit("Hz"))
//	ctl.Toggle("Sustain", panel.Message(piano, "sustain"))
//	ctl.Button("Bells", bells.Bang)
//	ctl.Meter("Master", mix.Master())
//	go ctl.Serve("localhost:8087")
package panel

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
)

// Kind is the kind of a control
type Kind string

const (
	FaderKind  Kind = "fader"
	KnobKind   Kind = "knob"
	ToggleKind Kind = "toggle"
	ButtonKind Kind = "button"
	SelectKind Kind = "select"
	MeterKind  Kind = "meter"
)

// Curve is how a fader or knob moves through its range
type Curve string

const (
	// Linear moves in even steps: levels in dB, mix amounts, pan
	Linear Curve = "linear"
	// Exp moves in even steps per octave (the minimum above 0): frequencies, times
	Exp Curve = "exp"
)

// Panel is a set of controls bound to the modules of a patch, and the control module that applies
// their changes. Create it with New, add controls, then Serve it (or change controls from code with
// Param.Set)
type Panel struct {
	*psalm.Module
	title    string
	mu       sync.Mutex // guards the list of controls and sections while they are declared
	params   []*Param
	byName   map[string]*Param
	sections []string // in order; every control belongs to the last section declared before it
	section  string
	buffer   int64 // the buffer the last tick was in, for the meters
	midiState

	srvMu   sync.Mutex // guards the servers
	servers []server
	closed  bool

	grids     map[string]*Grid
	cues      map[string]*Cues
	mounts    map[string]*Mount
	mountsNow atomic.Pointer[[]*Mount] // the mounts in order, for the audio thread
	order     []item                   // the controls, grids and cue lists in the order they were declared, for the page

	player  Player   // started and stopped from the page, nil for none
	theme   *Theme   // the panel's own theme, nil for the default
	plugins []Plugin // the plugins of the page
	id      int64    // every panel its own (a mount shows which one it shows)
	presetState
}

// item is a control, a grid or a cue list, in the order of the panel
type item struct {
	param *Param
	grid  *Grid
	cues  *Cues
	mount *Mount
}

// taken tells whether a control, grid or cue list has a name (with the lock held)
func (pn *Panel) taken(name string) bool {
	_, ok := pn.byName[name]
	return ok || pn.grids[name] != nil || pn.cues[name] != nil || pn.mounts[name] != nil
}

// cueLists returns the cue lists in the order they were declared
func (pn *Panel) cueLists() []*Cues {
	var cs []*Cues
	for _, it := range pn.items() {
		if it.cues != nil {
			cs = append(cs, it.cues)
		}
	}
	return cs
}

// items returns the controls and grids in order
func (pn *Panel) items() []item {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return append([]item(nil), pn.order...)
}

// Grids returns the grids in the order they were declared
func (pn *Panel) Grids() []*Grid {
	var gs []*Grid
	for _, it := range pn.items() {
		if it.grid != nil {
			gs = append(gs, it.grid)
		}
	}
	return gs
}

// New creates a panel with a title and adds it to the patch
func New(p psalm.Patcher, title string) *Panel {
	pn := &Panel{Module: psalm.NewControl(), title: title, byName: map[string]*Param{}, buffer: -1, player: playerOf(p), id: panelIDs.Add(1)}
	pn.SetSelf(pn)
	pn.AddTo(p)
	return pn
}

// Section starts a group of controls: the controls declared after it are shown together under its
// name
func (pn *Panel) Section(name string) *Panel {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.section = name
	pn.sections = append(pn.sections, name)
	return pn
}

// Param returns a control by name, nil when the panel has none of that name
func (pn *Panel) Param(name string) *Param {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return pn.byName[name]
}

// Params returns the controls in the order they were declared
func (pn *Panel) Params() []*Param {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return append([]*Param(nil), pn.params...)
}

func (pn *Panel) add(par *Param) *Param {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return pn.addLocked(par, true)
}

// addLocked adds a control with the lock held; a control that isn't shown is part of another one
// on the page (the tiles of a cue list), and can still be found by name and bound to MIDI
func (pn *Panel) addLocked(par *Param, shown bool) *Param {
	if pn.taken(par.Name) {
		panic(fmt.Sprintf("panel: a control named %q twice", par.Name))
	}
	if pn.section == "" {
		pn.section = pn.title
		pn.sections = append(pn.sections, pn.section)
	}
	par.Section = pn.section
	par.panel = pn
	pn.params = append(pn.params, par)
	if shown {
		pn.order = append(pn.order, item{param: par})
	}
	pn.byName[par.Name] = par
	return par
}

// Fader adds a vertical fader for a range, moving by a curve, bound to a target
func (pn *Panel) Fader(name string, min, max float64, curve Curve, to Target, opts ...Option) *Param {
	return pn.add(newParam(FaderKind, name, min, max, curve, to, opts))
}

// Knob adds a knob, like a fader in a smaller space
func (pn *Panel) Knob(name string, min, max float64, curve Curve, to Target, opts ...Option) *Param {
	return pn.add(newParam(KnobKind, name, min, max, curve, to, opts))
}

// Toggle adds a switch that sends 1 when on and 0 when off (a pedal, a mute, a freeze)
func (pn *Panel) Toggle(name string, to Target, opts ...Option) *Param {
	return pn.add(newParam(ToggleKind, name, 0, 1, Linear, to, append([]Option{Step(1)}, opts...)))
}

// Button adds a button that calls f (on the audio thread) every time it is pressed: start a
// sample, strike a chord, move to the next scene
func (pn *Panel) Button(name string, f func(), opts ...Option) *Param {
	par := newParam(ButtonKind, name, 0, 1, Linear, nil, opts)
	par.press = f
	return pn.add(par)
}

// Select adds a choice between options; f gets the index of the one chosen (on the audio thread)
func (pn *Panel) Select(name string, options []string, f func(i int), opts ...Option) *Param {
	if len(options) == 0 {
		panic(fmt.Sprintf("panel: select %q without options", name))
	}
	par := newParam(SelectKind, name, 0, float64(len(options)-1), Linear,
		Func(func(v float64) { f(int(math.Round(v))) }), append([]Option{Step(1)}, opts...))
	par.Options = append([]string(nil), options...)
	return pn.add(par)
}

// Meter adds a level meter of the output of a module: a synth, a mixer strip, the master
func (pn *Panel) Meter(name string, m psalm.Modular) *Param {
	par := newParam(MeterKind, name, 0, 0, Linear, nil, nil)
	par.meter = &meter{module: m, peakDB: -120}
	par.meter.peak.Store(math.Float64bits(-120))
	par.meter.rms.Store(math.Float64bits(-120))
	return pn.add(par)
}

// CtrlTick applies the changes of the controls, and measures the meters at the start of every
// buffer
func (pn *Panel) CtrlTick(tick psalm.CtrlTick) {
	cfg := pn.Config()
	dt := pn.Env().TickDuration()
	buffer := tick.SampleTimestamp / int64(max(1, cfg.BufferSize))
	newBuffer := buffer != pn.buffer
	pn.buffer = buffer

	for _, par := range pn.params {
		if par.meter != nil {
			if newBuffer {
				par.meter.measure(cfg.SampleRate, cfg.BufferSize)
			}
			continue
		}
		if par.custom != nil {
			if newBuffer && par.custom.onBuffer != nil {
				par.custom.onBuffer()
			}
			par.custom.deliver()
		}
		par.tick(dt)
	}
}

// Top shows the control in the bar at the top of the page, beside Stop and Restart, and small: a
// fader or a knob as a short slider with its value (a tap on the value types one), a meter as a
// bar; the control of a plugin gets "top" to draw itself small. Its section loses it (a section left
// with no controls isn't shown). Call it before the page is served:
//
//	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(out, "volume"), panel.Unit("dB")).Top()
//	ctl.Meter("Out", out).Top()
func (par *Param) Top() *Param {
	par.top = true
	return par
}

// Option sets something of a control
type Option func(par *Param)

// Default is the value a control starts at (and goes back to on a double click); without it a
// control starts at its minimum. The panel sends it to the target when the patch starts
func Default(v float64) Option { return func(par *Param) { par.Default = v } }

// NoDefault keeps the default to the control: it isn't sent to the target when the patch starts,
// the target keeps its own setting until the control is moved (an envelope over synths that each
// have their own)
func NoDefault() Option {
	return func(par *Param) { par.noDefault = true }
}

// Label is shown on the page instead of the name: names have to be unique ("kick mute"), a label
// can be short and repeat ("M")
func Label(text string) Option { return func(par *Param) { par.Label = text } }

// Unit is shown after the value: "Hz", "dB", "ms"
func Unit(unit string) Option { return func(par *Param) { par.Unit = unit } }

// Step rounds the value to steps of a size (1 for whole numbers)
func Step(step float64) Option { return func(par *Param) { par.Step = step } }

// Smooth glides to a new value over seconds instead of jumping, so a quick move doesn't step
// audibly (a cutoff, a level). The glide moves evenly through the fader's travel: through the
// octaves for an exponential fader
func Smooth(seconds float64) Option { return func(par *Param) { par.Smooth = seconds } }

// Param is a control of a panel: its range and its value. Set and Value are safe from any
// goroutine
type Param struct {
	Kind     Kind
	Name     string
	Section  string
	Min, Max float64
	Curve    Curve
	Step     float64
	Unit     string
	Label    string // shown on the page instead of the name (names are unique, labels can repeat); SetLabel changes it
	Default  float64
	Smooth   float64  // seconds
	Options  []string // of a select

	noDefault bool // the default isn't sent when the patch starts
	top       bool // shown in the bar at the top of the page (Top)

	target Target
	press  func()
	custom *custom  // a control a plugin draws
	parts  []*Param // its parts (Part)
	meter  *meter
	panel  *Panel

	// MIDI: the controller bound (see packCC), soft takeover
	cc      atomic.Int32
	pickup  bool
	caught  atomic.Bool
	lastPos float64 // the controller's last position, on the MIDI goroutine

	relabel atomic.Pointer[string] // the label changed by SetLabel
	value   atomic.Uint64          // the value asked for (float64 bits)
	dirty   atomic.Bool
	placed  atomic.Bool // placed (Place): the glide goes there, nothing is sent
	presses atomic.Int64

	// the audio thread's side
	pos     float64 // where the glide is, 0-1
	sent    bool
	handled int64 // presses handled
}

func newParam(kind Kind, name string, min, max float64, curve Curve, to Target, opts []Option) *Param {
	if curve == Exp && (min <= 0 || max <= 0) {
		panic(fmt.Sprintf("panel: %q is exponential, its range has to be above 0 (%v to %v)", name, min, max))
	}
	par := &Param{Kind: kind, Name: name, Min: min, Max: max, Curve: curve, target: to, Default: min}
	for _, o := range opts {
		o(par)
	}
	par.Default = par.clamp(par.Default)
	par.value.Store(math.Float64bits(par.Default))
	par.dirty.Store(kind != ButtonKind && kind != MeterKind && !par.noDefault) // the default goes out when the patch starts
	return par
}

// SetLabel changes the label the page shows (from any goroutine): the next time the page asks for the panel, a plugin that renames its settings when another is changed
func (par *Param) SetLabel(text string) { par.relabel.Store(&text) }

// Shown is the label the page shows: the one SetLabel gave, else the label
func (par *Param) Shown() string {
	if l := par.relabel.Load(); l != nil {
		return *l
	}
	return par.Label
}

// Value returns the value of the control (the last one asked for, a glide may still be on its way)
func (par *Param) Value() float64 { return math.Float64frombits(par.value.Load()) }

// Set asks for a value (clamped to the range, rounded to the step); it reaches the target on the
// next control tick. A controller with pickup has to reach the new value before it moves the
// control again
func (par *Param) Set(v float64) {
	par.set(v)
	par.placed.Store(false) // a value asked for after a placement is sent: the placement not yet seen by a tick must not drop it
	par.caught.Store(false)
}

// Place puts the control at a value without sending it: the page shows it, the target is left as it
// is (it was made at that value some other way), and a value asked for but not yet sent is dropped.
// From any goroutine
func (par *Param) Place(v float64) {
	if par.Kind == ButtonKind || par.Kind == MeterKind {
		return
	}
	par.value.Store(math.Float64bits(par.clamp(v)))
	par.placed.Store(true)
	par.dirty.Store(false)
	par.caught.Store(false)
}

func (par *Param) set(v float64) {
	if par.Kind == ButtonKind || par.Kind == MeterKind {
		return
	}
	par.value.Store(math.Float64bits(par.clamp(v)))
	par.dirty.Store(true)
}

// Press presses a button; its function is called on the next control tick
func (par *Param) Press() {
	if par.Kind == ButtonKind {
		par.presses.Add(1)
	}
}

// Levels returns the peak and RMS level (dB) of a meter: the peak falls back slowly, as on a mixer
func (par *Param) Levels() (peak, rms float64) {
	if par.meter == nil {
		return math.Inf(-1), math.Inf(-1)
	}
	return par.meter.levels()
}

// clamp keeps a value in the range and on the step
func (par *Param) clamp(v float64) float64 {
	if math.IsNaN(v) {
		v = par.Min
	}
	lo, hi := math.Min(par.Min, par.Max), math.Max(par.Min, par.Max)
	v = math.Max(lo, math.Min(v, hi))
	if par.Step > 0 {
		v = par.Min + math.Round((v-par.Min)/par.Step)*par.Step
		v = math.Max(lo, math.Min(v, hi))
	}
	return v
}

// Position returns where a value is along the travel of the control, 0-1
func (par *Param) Position(v float64) float64 {
	if par.Max == par.Min {
		return 0
	}
	if par.Curve == Exp {
		return math.Log(v/par.Min) / math.Log(par.Max/par.Min)
	}
	return (v - par.Min) / (par.Max - par.Min)
}

// At returns the value at a position along the travel of the control, 0-1
func (par *Param) At(pos float64) float64 {
	pos = math.Max(0, math.Min(pos, 1))
	if par.Curve == Exp {
		return par.Min * math.Pow(par.Max/par.Min, pos)
	}
	return par.Min + (par.Max-par.Min)*pos
}

// tick applies the control on the audio thread: a press, a new value, a step of a glide
func (par *Param) tick(dt float64) {
	if par.Kind == ButtonKind {
		if n := par.presses.Load(); n != par.handled {
			for ; par.handled < n; par.handled++ {
				if par.press != nil {
					par.press()
				}
			}
		}
		return
	}
	if par.target == nil {
		par.dirty.Store(false)
		return
	}
	if par.placed.Swap(false) {
		par.pos = par.Position(par.Value())
		par.dirty.Store(false)
		return
	}
	glide := par.Smooth > 0 && par.Kind != ToggleKind && par.Kind != SelectKind
	if !par.dirty.Load() && !(glide && par.sent && par.pos != par.Position(par.Value())) {
		return
	}
	par.dirty.Store(false)
	want := par.Value()
	if !glide || !par.sent {
		par.pos = par.Position(want)
		par.sent = true
		par.target.Set(want)
		return
	}
	goal := par.Position(want)
	par.pos += (goal - par.pos) * (1 - math.Exp(-dt/par.Smooth))
	if math.Abs(goal-par.pos) < 1e-4 {
		par.pos = goal
		par.target.Set(want)
		return
	}
	par.target.Set(par.At(par.pos))
}
