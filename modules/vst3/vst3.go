// Package vst3 hosts VST3 plugins (https://steinbergmedia.github.io/vst3_dev_portal): an instrument or an effect from a .vst3 bundle becomes a module of a patch. An
// instrument takes the notes of the patcher (the pitch of a note between the keys goes to the plugin as the tuning of the note, in cents), an effect takes the audio of
// its input and a side chain; the parameters of the plugin are described as the settings of any synth (params.Set) and move by messages, by the slug of their name:
// {"filter cutoff": 0.4}. The plugin's own window (X11) opens too. It is the sibling of package clap.
//
// Plugins are searched for in $VST3_PATH (colon separated), $PSALM_VST3_PATH, ~/.vst3, /usr/lib/vst3 and /usr/local/lib/vst3.
package vst3

/*
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/include -I${SRCDIR}
#cgo LDFLAGS: -ldl -lX11 -lpthread -lstdc++
#include <stdlib.h>
#include "glue.h"
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// Descriptor is a plugin found in a .vst3 bundle
type Descriptor struct {
	Path       string // the bundle
	CID        string // the class id, 32 hex digits
	Name       string
	Vendor     string
	Categories string // "Instrument|Synth", "Fx|Reverb"
}

// Instrument is whether it plays notes, Effect whether it processes audio
func (d Descriptor) Instrument() bool { return strings.Contains(d.Categories, "Instrument") }
func (d Descriptor) Effect() bool     { return strings.Contains(d.Categories, "Fx") }

var (
	libMu sync.Mutex
	libs  = map[string]*C.v3_lib{} // a bundle is opened once and stays open
)

func openMain(path string) (*C.v3_lib, error) {
	libMu.Lock()
	defer libMu.Unlock()
	if l, ok := libs[path]; ok {
		return l, nil
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var errbuf [256]C.char
	l := C.v3_open(cpath, &errbuf[0], 256)
	if l == nil {
		return nil, fmt.Errorf("%s: %s", path, C.GoString(&errbuf[0]))
	}
	libs[path] = l
	return l, nil
}

func open(path string) (lib *C.v3_lib, err error) {
	mainDo(func() { lib, err = openMain(path) })
	return
}

// SearchPath is where plugins are looked for
func SearchPath() []string {
	var dirs []string
	for _, env := range []string{"VST3_PATH", "PSALM_VST3_PATH"} {
		for _, d := range filepath.SplitList(os.Getenv(env)) {
			if d != "" {
				dirs = append(dirs, d)
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".vst3"))
	}
	return append(dirs, "/usr/lib/vst3", "/usr/local/lib/vst3")
}

// Scan lists the plugins of the .vst3 bundles in the directories (the search path when there are none), searched in their subdirectories too. A bundle that will not
// load is skipped
func Scan(dirs ...string) []Descriptor {
	if len(dirs) == 0 {
		dirs = SearchPath()
	}
	var out []Descriptor
	seen := map[string]bool{}
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, e os.DirEntry, err error) error {
			if err != nil || !e.IsDir() || path == dir || filepath.Ext(path) != ".vst3" {
				return nil
			}
			if !seen[path] {
				seen[path] = true
				out = append(out, describe(path)...)
			}
			return filepath.SkipDir // a bundle has nothing but its own files inside
		})
	}
	return out
}

func describeMain(path string) []Descriptor {
	lib, err := openMain(path)
	if err != nil {
		return nil
	}
	var out []Descriptor
	var cid, name, vendor, cats [512]C.char
	for i := 0; i < int(C.v3_class_count(lib)); i++ {
		if C.v3_class(lib, C.int(i), &cid[0], &name[0], &vendor[0], &cats[0], 512) == 0 {
			continue
		}
		out = append(out, Descriptor{Path: path, CID: C.GoString(&cid[0]), Name: C.GoString(&name[0]), Vendor: C.GoString(&vendor[0]), Categories: C.GoString(&cats[0])})
	}
	return out
}

func describe(path string) (out []Descriptor) {
	mainDo(func() { out = describeMain(path) })
	return
}

// param is a parameter of the plugin
type param struct {
	id         uint32
	key, label string
	group      string
	def        float64
	steps      int
}

// mainWords are what a plugin with many parameters is played with: the first parameter whose name has the word is in the group "Main" at the top (a plugin with forty
// parameters or fewer needs no such group)
var mainWords = []string{"volume", "gain", "cutoff", "resonance", "attack", "decay", "sustain", "release", "drive", "mix", "reverb", "delay", "width"}

// note is a note that sounds: the id the plugin knows it by, its key, and for a note of a duration the samples it has left
type note struct {
	id      int32
	key     int
	samples int // above 0 for a note of a duration
}

// Plugin is a VST3 plugin as a module: an instrument (it takes notes) or an effect (it takes the audio on its input)
type Plugin struct {
	*psalm.Module
	desc      Descriptor
	inst      *C.v3_inst
	nIn       int // audio input buses: the first is played; the second (when there is one) is the side chain
	nOut      int // audio output buses, each an output of the module
	params    []param
	keys      map[string]int
	byID      map[uint32]int
	held      map[string]note
	timed     []note
	nextID    int32
	frames    int
	tempo     atomic.Uint64
	beats     atomic.Uint64
	closed    atomic.Bool
	syncFn    atomic.Pointer[func(map[string]float64)]
	nameFn    atomic.Pointer[func(map[string]string)]
	loadMu    sync.Mutex
	metaMu    sync.RWMutex
	changeMu  sync.Mutex
	changes   map[string]float64
	flushing  bool
	telling   atomic.Bool
	presets   []Preset
	preset    atomic.Int64 // the one loaded last, and one: 0 is none
	progParam uint32
	progSteps int
}

func newMain(d Descriptor) (*Plugin, error) {
	lib, err := openMain(d.Path)
	if err != nil {
		return nil, err
	}
	cfg := psalm.CurrentEnv().Config()
	cid := C.CString(d.CID)
	defer C.free(unsafe.Pointer(cid))
	var errbuf [256]C.char
	inst := C.v3_new(lib, cid, C.double(cfg.SampleRate), C.uint32_t(cfg.BufferSize), &errbuf[0], 256)
	if inst == nil {
		return nil, fmt.Errorf("%s: %s", d.Name, C.GoString(&errbuf[0]))
	}
	var nin, nout C.int
	C.v3_bus_counts(inst, &nin, &nout)
	p := &Plugin{desc: d, inst: inst, nIn: int(nin), nOut: min(max(int(nout), 1), 16), held: map[string]note{}, keys: map[string]int{}, byID: map[uint32]int{}, frames: cfg.BufferSize}
	ins := []int{psalm.Stereo}
	if p.nIn > 1 {
		ins = append(ins, psalm.Stereo) // the side chain
	}
	outs := make([]int, p.nOut)
	for i := range outs {
		outs[i] = psalm.Stereo
	}
	p.Module = psalm.NewModule(ins, outs)
	p.SetSelf(p)
	p.readParams()
	p.readPrograms()
	p.presets = append(p.presets, filePresets(d)...)
	return p, nil
}

// New makes the plugin: active, at the sample rate and buffer size of the patch
func New(d Descriptor) (p *Plugin, err error) {
	mainDo(func() { p, err = newMain(d) })
	return
}

// slug is a key of a message from a name: lower case, words apart by a space
func slug(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

func (p *Plugin) readParams() {
	var title, units [256]C.char
	for i := 0; i < int(C.v3_param_count(p.inst)); i++ {
		var id C.uint32_t
		var steps, flags, unit C.int
		var def C.double
		if C.v3_param_info(p.inst, C.int(i), &id, &title[0], &units[0], 256, &steps, &def, &flags, &unit) == 0 {
			continue
		}
		const readonly, hidden, program, bypassFlag = 1 << 1, 1 << 4, 1 << 15, 1 << 16
		if flags&(readonly|hidden|program) != 0 {
			continue
		}
		label := strings.TrimSpace(C.GoString(&title[0]))
		if label == "" {
			label = fmt.Sprintf("parameter %d", id)
		}
		key := slug(label)
		if _, dup := p.keys[key]; dup {
			key = fmt.Sprintf("%s %d", key, id)
		}
		group := ""
		if unit != 0 {
			var gn [256]C.char
			if C.v3_unit_name(p.inst, unit, &gn[0], 256) != 0 {
				group = C.GoString(&gn[0])
			}
		}
		p.keys[key] = len(p.params)
		p.byID[uint32(id)] = len(p.params)
		p.params = append(p.params, param{id: uint32(id), key: key, label: label, group: group, def: float64(def), steps: int(steps)})
	}
	p.pickMain()
}

func (p *Plugin) pickMain() {
	if len(p.params) <= 40 {
		return
	}
	var main, rest []param
	taken := map[int]bool{}
	for _, w := range mainWords {
		pick := -1
		for i, q := range p.params {
			name := strings.ToLower(q.group + " " + q.label)
			if taken[i] || !strings.Contains(strings.ToLower(q.label), w) || strings.Contains(name, "lfo") || strings.Contains(name, "macro") {
				continue
			}
			if pick < 0 {
				pick = i
			}
			if strings.Contains(name, "filter") || strings.Contains(name, "amp") || strings.Contains(name, "master") || strings.Contains(name, "global") {
				pick = i // the one that is the sound's own: its filter, its amplifier, the master
				break
			}
		}
		if pick >= 0 {
			taken[pick] = true
			q := p.params[pick]
			q.group = "Main"
			main = append(main, q)
		}
	}
	for i, q := range p.params {
		if !taken[i] {
			rest = append(rest, q)
		}
	}
	p.params = append(main, rest...)
	for i, q := range p.params {
		p.keys[q.key] = i
		p.byID[q.id] = i
	}
}

// Descriptor is the plugin as it was found
func (p *Plugin) Descriptor() Descriptor { return p.desc }

func (p *Plugin) relabelMain() int {
	if p.inst == nil {
		return 0
	}
	changed := 0
	var title, units [256]C.char
	for i := 0; i < int(C.v3_param_count(p.inst)); i++ {
		var id C.uint32_t
		var steps, flags, unit C.int
		var def C.double
		if C.v3_param_info(p.inst, C.int(i), &id, &title[0], &units[0], 256, &steps, &def, &flags, &unit) == 0 {
			continue
		}
		j, ok := p.byID[uint32(id)]
		if !ok {
			continue
		}
		if label := strings.TrimSpace(C.GoString(&title[0])); label != "" {
			p.metaMu.Lock()
			if p.params[j].label != label {
				p.params[j].label = label
				changed++
			}
			p.metaMu.Unlock()
		}
	}
	return changed
}

// Relabel asks the plugin for the titles of its parameters again (some plugins rename them when a setting changes) and returns how many changed; the keys stay what
// they were
func (p *Plugin) Relabel() (n int) {
	mainDo(func() { n = p.relabelMain() })
	return
}

func (p *Plugin) describeMain() params.Set {
	set := params.Set{Name: p.desc.Name}
	p.metaMu.RLock()
	defer p.metaMu.RUnlock()
	if len(p.presets) > 0 {
		names := make([]string, len(p.presets)+1)
		names[0] = "—" // no preset: the plugin has the sound it was given some other way (a state, its own window); the others are the presets, one number higher than Loaded
		for i, pr := range p.presets {
			names[i+1] = pr.label()
		}
		set.Params = append(set.Params, params.Param{Key: presetKey, Label: "preset", Group: "Preset", Kind: params.Choice, Options: names, Min: 0, Max: float64(len(names) - 1), Step: 1, Value: float64(p.Loaded() + 1)})
	}
	for _, q := range p.params {
		v := float64(C.v3_param_value(p.inst, C.uint32_t(q.id)))
		group := q.group
		if group == "" {
			group = p.desc.Name
		}
		var pr params.Param
		switch {
		case q.steps == 1:
			pr = params.S(group, q.key, v >= 0.5, params.Label(q.label))
		case q.steps > 1:
			pr = params.K(group, q.key, 0, 1, v, params.Step(1/float64(q.steps)), params.Label(q.label))
		default:
			pr = params.K(group, q.key, 0, 1, v, params.Label(q.label))
		}
		set.Params = append(set.Params, pr)
	}
	return set
}

// Describe is the settings of the plugin as a synth describes its own: a knob for every parameter, where it is now
func (p *Plugin) Describe() (set params.Set) {
	mainDo(func() { set = p.describeMain() })
	return
}

func (p *Plugin) textMain(key string, value float64) string {
	i, ok := p.keys[key]
	if !ok || p.inst == nil {
		return ""
	}
	var buf [256]C.char
	if C.v3_param_text(p.inst, C.uint32_t(p.params[i].id), C.double(value), &buf[0], 256) == 0 {
		return ""
	}
	return C.GoString(&buf[0])
}

// Text is what the plugin shows for a value of the parameter with this key ("440 Hz", "Saw"), empty when it shows none
func (p *Plugin) Text(key string, value float64) (s string) {
	mainDo(func() { s = p.textMain(key, value) })
	return
}

func (p *Plugin) valuesMain() map[string]float64 {
	out := make(map[string]float64, len(p.params))
	if p.inst == nil {
		return out
	}
	for _, q := range p.params {
		out[q.key] = float64(C.v3_param_value(p.inst, C.uint32_t(q.id)))
	}
	if len(p.presets) > 0 {
		out[presetKey] = float64(p.Loaded() + 1) // 0 is none
	}
	return out
}

// Values are the settings of the plugin as it has them now, by key
func (p *Plugin) Values() (v map[string]float64) {
	mainDo(func() { v = p.valuesMain() })
	return
}

// Keys are the keys of the messages that move the parameters
func (p *Plugin) Keys() []string {
	out := make([]string, len(p.params))
	for i, q := range p.params {
		out[i] = q.key
	}
	return out
}

// Bind gives the plugin a function that sets the settings of its editor and one that changes their labels: after a state is loaded, or a knob turned in the plugin's
// window, the plugin tells where all its settings are now, so the knobs follow
func (p *Plugin) Bind(set func(values map[string]float64), relabel func(labels map[string]string)) {
	p.syncFn.Store(&set)
	p.nameFn.Store(&relabel)
}

// tellEditor waits until the plugin has settled on what it loaded and then tells the editor the settings it has
func (p *Plugin) tellEditor() {
	p.tellEditorWith(false)
}

func (p *Plugin) tellEditorWith(titles bool) {
	fn := p.syncFn.Load()
	if fn == nil {
		return
	}
	last := p.Values()
	for i := 0; i < 60; i++ { // up to three seconds
		time.Sleep(50 * time.Millisecond)
		now := p.Values()
		if mapsEqual(now, last) && i > 1 {
			break
		}
		last = now
	}
	if titles && p.Relabel() > 0 {
		if names := p.nameFn.Load(); names != nil {
			labels := make(map[string]string, len(p.params))
			p.metaMu.RLock()
			for _, q := range p.params {
				labels[q.key] = q.label
			}
			p.metaMu.RUnlock()
			(*names)(labels)
		}
	}
	(*fn)(last)
}

func mapsEqual(a, b map[string]float64) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// collectChanges notes what the plugin did to its own parameters in the block (a knob turned in its window) and has the editor told a moment later
func (p *Plugin) collectChanges() {
	var ids [64]C.uint32_t
	var values [64]C.double
	n := int(C.v3_take_changes(p.inst, &ids[0], &values[0], 64))
	if n == 0 || p.syncFn.Load() == nil {
		return
	}
	p.changeMu.Lock()
	if p.changes == nil {
		p.changes = map[string]float64{}
	}
	for i := 0; i < n; i++ {
		if j, ok := p.byID[uint32(ids[i])]; ok {
			p.changes[p.params[j].key] = float64(values[i])
		}
	}
	start := !p.flushing && len(p.changes) > 0
	if start {
		p.flushing = true
	}
	p.changeMu.Unlock()
	if start {
		time.AfterFunc(60*time.Millisecond, func() {
			p.changeMu.Lock()
			pending := p.changes
			p.changes, p.flushing = nil, false
			p.changeMu.Unlock()
			if fn := p.syncFn.Load(); fn != nil && len(pending) > 0 {
				(*fn)(pending)
			}
		})
	}
}

// stateMagic starts a state: the component's state, then the controller's, each with its length
const stateMagic = "V3S1"

func (p *Plugin) saveStateMain() ([]byte, error) {
	if p.inst == nil {
		return nil, errors.New("the plugin is gone")
	}
	var comp, ctrl *C.uint8_t
	var cl, tl C.size_t
	if C.v3_state_save(p.inst, &comp, &cl, &ctrl, &tl) == 0 {
		return nil, fmt.Errorf("%s keeps no state", p.desc.Name)
	}
	defer C.v3_free_bytes(unsafe.Pointer(comp))
	if ctrl != nil {
		defer C.v3_free_bytes(unsafe.Pointer(ctrl))
	}
	out := []byte(stateMagic)
	out = binary.BigEndian.AppendUint32(out, uint32(cl))
	out = append(out, C.GoBytes(unsafe.Pointer(comp), C.int(cl))...)
	out = binary.BigEndian.AppendUint32(out, uint32(tl))
	if tl > 0 {
		out = append(out, C.GoBytes(unsafe.Pointer(ctrl), C.int(tl))...)
	}
	return out, nil
}

// SaveState is what the plugin keeps of its sound (the state of its component and of its controller), to give back with LoadState
func (p *Plugin) SaveState() (st []byte, err error) {
	mainDo(func() { st, err = p.saveStateMain() })
	return
}

func (p *Plugin) loadStateMain(state []byte) error {
	p.loadMu.Lock()
	defer p.loadMu.Unlock()
	if p.inst == nil || p.closed.Load() {
		return errors.New("the plugin is gone")
	}
	if len(state) < 12 || string(state[:4]) != stateMagic {
		return errors.New("not a state of a VST3 plugin")
	}
	cl := int(binary.BigEndian.Uint32(state[4:]))
	if 8+cl+4 > len(state) {
		return errors.New("a state cut short")
	}
	comp := state[8 : 8+cl]
	tl := int(binary.BigEndian.Uint32(state[8+cl:]))
	if 12+cl+tl > len(state) {
		return errors.New("a state cut short")
	}
	ctrl := state[12+cl : 12+cl+tl]
	p.loadMu.Unlock()
	err := p.loadStateParts(comp, ctrl)
	p.loadMu.Lock()
	if err != nil {
		return err
	}
	go p.tellEditor()
	return nil
}

// LoadState gives the plugin a state back
func (p *Plugin) LoadState(state []byte) (err error) {
	mainDo(func() { err = p.loadStateMain(state) })
	return
}

// Crashed is the signal that crashed the plugin (11 for a fault, 6 for an abort), 0 while it is well: a plugin that crashed in a call we made makes no more sound and
// answers nothing, and the program goes on
func (p *Plugin) Crashed() int {
	if p.inst == nil {
		return 0
	}
	return int(C.v3_dead(p.inst))
}

func (p *Plugin) latencyMain() int {
	if p.inst == nil {
		return 0
	}
	return int(C.v3_latency(p.inst))
}

// Latency is how many samples the plugin says it delays the sound
func (p *Plugin) Latency() (n int) {
	mainDo(func() { n = p.latencyMain() })
	return
}

func (p *Plugin) portNamesMain() (ins, outs []string) {
	if p.inst == nil {
		return nil, nil
	}
	var ch C.int
	var name [256]C.char
	for i := 0; i < p.nIn && i < 2; i++ {
		if C.v3_bus_info(p.inst, 1, C.int(i), &ch, &name[0], 256) != 0 {
			ins = append(ins, C.GoString(&name[0]))
		}
	}
	for i := 0; i < p.nOut; i++ {
		if C.v3_bus_info(p.inst, 0, C.int(i), &ch, &name[0], 256) != 0 {
			outs = append(outs, C.GoString(&name[0]))
		}
	}
	return ins, outs
}

// PortNames are the names the plugin gives its audio buses: the inputs (the first is played; a second is the side chain) and the outputs
func (p *Plugin) PortNames() (ins, outs []string) {
	mainDo(func() { ins, outs = p.portNamesMain() })
	return
}

func (p *Plugin) hasWindowMain() bool { return p.inst != nil && C.v3_gui_available(p.inst) != 0 }

// HasWindow is whether the plugin has a window of its own (X11) that can be shown
func (p *Plugin) HasWindow() (b bool) {
	mainDo(func() { b = p.hasWindowMain() })
	return
}

func (p *Plugin) showWindowMain() error {
	if p.inst == nil || p.closed.Load() {
		return errors.New("the plugin is gone")
	}
	if C.v3_gui_show(p.inst) == 0 {
		return fmt.Errorf("%s has no window to show here", p.desc.Name)
	}
	return nil
}

// ShowWindow opens the window of the plugin (made and run by the main thread of the plugins); HideWindow closes it
func (p *Plugin) ShowWindow() (err error) {
	mainDo(func() { err = p.showWindowMain() })
	return
}

func (p *Plugin) hideWindowMain() {
	if p.inst != nil {
		C.v3_gui_hide(p.inst)
	}
}

func (p *Plugin) HideWindow() { mainDo(p.hideWindowMain) }

// WindowOpen is whether the window is showing
func (p *Plugin) WindowOpen() bool { return p.inst != nil && C.v3_gui_visible(p.inst) != 0 }

// Close lets the plugin go: the audio thread frees it at its next block (or Free does, when it doesn't run)
func (p *Plugin) Close() { p.closed.Store(true) }

func (p *Plugin) freeMain() {
	if p.inst != nil {
		C.v3_free(p.inst)
		p.inst = nil
	}
}

// Free lets the plugin go at once, for a plugin that is not running on the audio thread
func (p *Plugin) Free() { mainDo(p.freeMain) }

func (p *Plugin) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
	}
}

// pitchOf is the pitch of a note message: its key and how far the pitch is above it in cents
func pitchOf(msg psalm.Message) (key int, cents float64) {
	pitch := 60.0
	msg.IfFltVal("pitch", func(f float64) { pitch = f })
	msg.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			pitch = 69 + 12*math.Log2(f/440)
		}
	})
	key = int(math.Round(pitch))
	return min(127, max(0, key)), (pitch - float64(key)) * 100
}

// ReceiveMessage plays notes ("command": "trigger", as a synth gets them) and moves parameters (a message of their keys)
func (p *Plugin) ReceiveMessage(msg psalm.Message) {
	if p.inst == nil || p.closed.Load() {
		return
	}
	if command, ok := msg.StrVal("command"); ok {
		if command != "trigger" {
			return
		}
		tick := msg.MustTickVal("tick")
		at := uint32(min(max(tick.SampleOffset, 0), p.frames-1))
		if id, ok := msg.StrVal("noteOff"); ok {
			if n, ok := p.held[id]; ok {
				C.v3_note(p.inst, 0, C.uint32_t(at), C.int(n.key), 0, C.int32_t(n.id), 0)
				delete(p.held, id)
			}
			return
		}
		n := p.start(msg, at)
		if id, ok := msg.StrVal("noteOn"); ok {
			p.held[id] = n
		} else {
			n.samples = int(msg.MustFltVal("duration")*p.Config().SampleRate) + int(at)
			p.timed = append(p.timed, n)
		}
		return
	}
	msg.IfFltVal(presetKey, func(f float64) { // loaded off the audio thread: it reads files
		if i := int(math.Round(f)) - 1; len(p.presets) > 0 && i >= 0 && i != p.Loaded() { // the editor tells the plugin the preset it has (0: none): nothing to do then
			go func() { _ = p.LoadPreset(i) }()
		}
	})
	for key, i := range p.keys {
		msg.IfFltVal(key, func(f float64) {
			C.v3_param(p.inst, C.uint32_t(p.params[i].id), C.double(min(1, max(0, f))), 0)
		})
	}
}

func (p *Plugin) start(msg psalm.Message, at uint32) note {
	m, _ := msg.Msg("message")
	key, cents := pitchOf(m)
	p.nextID++
	n := note{id: p.nextID, key: key}
	amp := min(1, max(0, msg.MustFltVal("amplitude")))
	C.v3_note(p.inst, 1, C.uint32_t(at), C.int(key), C.double(amp), C.int32_t(n.id), C.double(cents))
	return n
}

// AllNotesOff lets go of every note
func (p *Plugin) AllNotesOff() {
	for id, n := range p.held {
		C.v3_note(p.inst, 0, 0, C.int(n.key), 0, C.int32_t(n.id), 0)
		delete(p.held, id)
	}
	for _, n := range p.timed {
		C.v3_note(p.inst, 0, 0, C.int(n.key), 0, C.int32_t(n.id), 0)
	}
	p.timed = p.timed[:0]
}

// CtrlTick tells the plugin where the patch is: the tempo and the position in beats
func (p *Plugin) CtrlTick(tick psalm.CtrlTick) {
	p.tempo.Store(math.Float64bits(tick.BPM))
	if tick.PPQ > 0 {
		p.beats.Store(math.Float64bits(float64(tick.TickTimestamp) / float64(tick.PPQ)))
	}
}

// Synthesize runs the plugin over a block
func (p *Plugin) Synthesize() {
	n := len(p.Output(0).Channels[0])
	for o := 0; o < p.nOut; o++ {
		p.Output(o).Channels.Clear()
	}
	if p.closed.Load() {
		p.Free()
	}
	if p.inst == nil || C.v3_dead(p.inst) != 0 {
		return
	}
	// the notes of a duration let go when their time is up
	for i := 0; i < len(p.timed); {
		if p.timed[i].samples <= n {
			t := p.timed[i]
			C.v3_note(p.inst, 0, C.uint32_t(min(max(t.samples, 0), n-1)), C.int(t.key), 0, C.int32_t(t.id), 0)
			p.timed = append(p.timed[:i], p.timed[i+1:]...)
			continue
		}
		p.timed[i].samples -= n
		i++
	}
	if tempo := math.Float64frombits(p.tempo.Load()); tempo > 0 {
		C.v3_transport(p.inst, C.double(tempo), C.double(math.Float64frombits(p.beats.Load())), 1)
	}
	var in0, in1, sc0, sc1 *C.float
	if p.nIn > 0 {
		in := p.Input(0).Channels
		in0 = (*C.float)(unsafe.Pointer(&in[0][0]))
		in1 = in0
		if len(in) > 1 {
			in1 = (*C.float)(unsafe.Pointer(&in[1][0]))
		}
	}
	if p.nIn > 1 && p.NumInputs() > 1 {
		sc := p.Input(1).Channels
		sc0, sc1 = (*C.float)(unsafe.Pointer(&sc[0][0])), (*C.float)(unsafe.Pointer(&sc[1][0]))
	}
	C.v3_process(p.inst, in0, in1, sc0, sc1, C.uint32_t(n))
	if C.v3_dead(p.inst) != 0 {
		return // it crashed in this block: what it left in its buffers is not sound
	}
	for o := 0; o < p.nOut; o++ {
		out := p.Output(o).Channels
		for ch := 0; ch < 2 && ch < len(out); ch++ {
			if src := C.v3_out(p.inst, C.int(o), C.int(ch)); src != nil {
				copy(out[ch], unsafe.Slice((*float32)(unsafe.Pointer(src)), n))
			}
		}
	}
	p.collectChanges()
	if flags := int(C.v3_take_restart(p.inst)); flags&(4|16) != 0 && p.telling.CompareAndSwap(false, true) { // the values or the titles of the parameters changed
		go func() {
			defer p.telling.Store(false)
			p.tellEditorWith(flags&16 != 0)
		}()
	}
}

var _ psalm.Modular = (*Plugin)(nil)
