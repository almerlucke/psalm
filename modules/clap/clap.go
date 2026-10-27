// Package clap hosts CLAP plugins (https://cleveraudio.org): an instrument or an effect from a .clap file becomes a module of a patch. An instrument takes the
// notes of the patcher (the pitch of a note between the keys goes to the plugin as the tuning of the note, so just intonation sounds as tuned), an effect takes
// the audio of its input; the parameters of the plugin are described as the settings of any synth (params.Set) and move by messages, by the slug of their name:
// {"filter cutoff": 0.4}. The plugin has no window of its own here.
//
// Plugins are searched for in $CLAP_PATH (colon separated), $PSALM_CLAP_PATH, ~/.clap, /usr/lib/clap and /usr/local/lib/clap.
package clap

/*
#cgo CFLAGS: -I${SRCDIR}/include -std=gnu11
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include "glue.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// Descriptor is a plugin found in a .clap file
type Descriptor struct {
	Path     string
	ID       string
	Name     string
	Vendor   string
	Features []string
}

// Instrument is whether it plays notes, Effect whether it processes audio
func (d Descriptor) Instrument() bool { return slices.Contains(d.Features, "instrument") }
func (d Descriptor) Effect() bool     { return slices.Contains(d.Features, "audio-effect") }

var (
	libMu sync.Mutex
	libs  = map[string]*C.pc_lib{} // a file is opened once and stays open
)

func open(path string) (*C.pc_lib, error) {
	libMu.Lock()
	defer libMu.Unlock()
	if l, ok := libs[path]; ok {
		return l, nil
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var errbuf [256]C.char
	l := C.pc_open(cpath, &errbuf[0], 256)
	if l == nil {
		return nil, fmt.Errorf("%s: %s", path, C.GoString(&errbuf[0]))
	}
	libs[path] = l
	return l, nil
}

// SearchPath is where plugins are looked for
func SearchPath() []string {
	var dirs []string
	for _, env := range []string{"CLAP_PATH", "PSALM_CLAP_PATH"} {
		for _, d := range filepath.SplitList(os.Getenv(env)) {
			if d != "" {
				dirs = append(dirs, d)
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".clap"))
	}
	return append(dirs, "/usr/lib/clap", "/usr/local/lib/clap")
}

// Scan lists the plugins of the .clap files in the directories (the search path when there are none), searched in their subdirectories too. A file that will not
// load is skipped
func Scan(dirs ...string) []Descriptor {
	if len(dirs) == 0 {
		dirs = SearchPath()
	}
	var out []Descriptor
	seen := map[string]bool{}
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, e os.DirEntry, err error) error {
			if err != nil || e.IsDir() || filepath.Ext(path) != ".clap" || seen[path] {
				return nil
			}
			seen[path] = true
			out = append(out, describe(path)...)
			return nil
		})
	}
	return out
}

func describe(path string) []Descriptor {
	lib, err := open(path)
	if err != nil {
		return nil
	}
	var out []Descriptor
	var id, name, vendor, features [512]C.char
	for i := 0; i < int(C.pc_count(lib)); i++ {
		if C.pc_desc(lib, C.int(i), &id[0], &name[0], &vendor[0], &features[0], 512) == 0 {
			continue
		}
		out = append(out, Descriptor{Path: path, ID: C.GoString(&id[0]), Name: C.GoString(&name[0]), Vendor: C.GoString(&vendor[0]), Features: strings.Split(C.GoString(&features[0]), "\x1f")})
	}
	return out
}

// Preset is a preset the plugin's library tells of (its preset discovery): a file it knows how to read, or one that is inside the plugin
type Preset struct {
	Name        string
	Creator     string
	Features    []string // what kind of sound it is ("bass", "pad"), as the plugin's authors say
	Description string
	Category    string // the kind of sound: the first of its features, else the folder a file is in
	Factory     bool   // it comes with the plugin; the others are the user's
	kind        uint32
	location    string
	loadKey     string
	plugin      string // the plugin it is for: empty for any in the library
}

var (
	presetMu sync.Mutex
	presets  = map[string][]Preset{} // by file: the presets of a library are found once
)

// PresetsOf are the presets the library of a descriptor has for its plugin, in the order the library lists them
func PresetsOf(d Descriptor) []Preset {
	lib, err := open(d.Path)
	if err != nil {
		return nil
	}
	presetMu.Lock()
	all, done := presets[d.Path]
	if !done {
		n := int(C.pc_scan_presets(lib))
		for i := 0; i < n; i++ {
			var c C.pc_preset
			if C.pc_preset_get(lib, C.int(i), &c) == 0 {
				continue
			}
			all = append(all, Preset{Name: C.GoString(&c.name[0]), Creator: C.GoString(&c.creator[0]), Description: C.GoString(&c.description[0]),
				Features: splitFeatures(C.GoString(&c.features[0])), Factory: c.flags&1 != 0, kind: uint32(c.kind), location: C.GoString(&c.location[0]),
				loadKey: C.GoString(&c.loadkey[0]), plugin: C.GoString(&c.plugin[0])})
			pr := &all[len(all)-1]
			if len(pr.Features) > 0 {
				pr.Category = pr.Features[0]
			} else if pr.kind == 0 && pr.location != "" {
				pr.Category = filepath.Base(filepath.Dir(pr.location))
			}
		}
		presets[d.Path] = all
	}
	presetMu.Unlock()
	var mine []Preset
	for _, pr := range all {
		if pr.plugin == "" || pr.plugin == d.ID {
			mine = append(mine, pr)
		}
	}
	slices.SortStableFunc(mine, func(a, b Preset) int { return strings.Compare(strings.ToLower(a.label()), strings.ToLower(b.label())) })
	return mine
}

func splitFeatures(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\x1f")
}

// param is a parameter of the plugin
type param struct {
	id            uint32
	key, label    string
	group         string
	min, max, def float64
	stepped       bool
}

// note is a note that sounds: the id the plugin knows it by, its key, and for a note of a duration the samples it has left
type note struct {
	id      int32
	key     int
	samples int // above 0 for a note of a duration
}

// Plugin is a CLAP plugin as a module: an instrument (it takes notes) or an effect (it takes the audio on its input)
type Plugin struct {
	*psalm.Module
	desc    Descriptor
	inst    *C.pc_inst
	inCh    int
	params  []param
	keys    map[string]int // the key of a message to the index of the parameter
	held    map[string]note
	timed   []note
	nextID  int32
	frames  int
	closed  atomic.Bool // asked to let go: the audio thread does it at its next block
	presets []Preset
	preset  atomic.Int64 // the one loaded last, and one: 0 is none
	loadMu  sync.Mutex
}

// New makes the plugin: active, at the sample rate and buffer size of the patch
func New(d Descriptor) (*Plugin, error) {
	lib, err := open(d.Path)
	if err != nil {
		return nil, err
	}
	cfg := psalm.CurrentEnv().Config()
	cid := C.CString(d.ID)
	defer C.free(unsafe.Pointer(cid))
	var errbuf [256]C.char
	inst := C.pi_new(lib, cid, C.double(cfg.SampleRate), C.uint32_t(cfg.BufferSize), &errbuf[0], 256)
	if inst == nil {
		return nil, fmt.Errorf("%s: %s", d.Name, C.GoString(&errbuf[0]))
	}
	var in, out C.int
	C.pi_ports(inst, &in, &out)
	p := &Plugin{desc: d, inst: inst, inCh: int(in), held: map[string]note{}, keys: map[string]int{}, frames: cfg.BufferSize, presets: PresetsOf(d)}
	p.Module = psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo})
	p.SetSelf(p)
	p.readParams()
	return p, nil
}

// slug is a key of a message from a name: lower case, words apart by a space
func slug(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

func (p *Plugin) readParams() {
	var name, module [256]C.char
	for i := 0; i < int(C.pi_param_count(p.inst)); i++ {
		var id, flags C.uint32_t
		var mn, mx, df C.double
		if C.pi_param_info(p.inst, C.int(i), &id, &name[0], &module[0], 256, &mn, &mx, &df, &flags) == 0 {
			continue
		}
		const stepped, hidden, readonly = 1 << 0, 1 << 2, 1 << 3 // CLAP_PARAM_IS_STEPPED, _IS_HIDDEN, _IS_READONLY
		if flags&(hidden|readonly) != 0 || math.IsNaN(float64(mn)) || float64(mx) <= float64(mn) {
			continue
		}
		label := C.GoString(&name[0])
		group := strings.ReplaceAll(strings.Trim(C.GoString(&module[0]), "/"), "/", " / ")
		key := slug(strings.TrimSpace(group + " " + label))
		if _, dup := p.keys[key]; dup {
			key = fmt.Sprintf("%s %d", key, id)
		}
		p.keys[key] = len(p.params)
		p.params = append(p.params, param{id: uint32(id), key: key, label: label, group: group, min: float64(mn), max: float64(mx), def: float64(df), stepped: flags&stepped != 0})
	}
	p.pickMain()
}

// mainWords are what a plugin with many parameters is played with: the first parameter whose name has the word is in the group "Main" at the top (a plugin with
// forty parameters or fewer needs no such group)
var mainWords = []string{"volume", "gain", "cutoff", "resonance", "attack", "decay", "sustain", "release", "drive", "mix", "reverb", "delay", "width"}

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
	}
}

// Descriptor is the plugin as it was found
func (p *Plugin) Descriptor() Descriptor { return p.desc }

// Describe is the settings of the plugin as a synth describes its own: a knob for every parameter, where it is now
func (p *Plugin) Describe() params.Set {
	set := params.Set{Name: p.desc.Name}
	if len(p.presets) > 0 {
		names := make([]string, len(p.presets))
		for i, pr := range p.presets {
			names[i] = pr.label()
		}
		set.Params = append(set.Params, params.Param{Key: presetKey, Label: "preset", Group: "Preset", Kind: params.Choice, Options: names, Min: 0, Max: float64(len(names) - 1), Step: 1, Value: float64(max(p.Loaded(), 0))})
	}
	for _, q := range p.params {
		group := q.group
		if group == "" {
			group = p.desc.Name
		}
		v := float64(C.pi_param_value(p.inst, C.uint32_t(q.id)))
		var pr params.Param
		switch {
		case q.stepped && q.max-q.min == 1:
			pr = params.S(group, q.key, v >= (q.min+q.max)/2, params.Label(q.label))
		case q.stepped:
			pr = params.K(group, q.key, q.min, q.max, v, params.Step(1), params.Label(q.label))
		default:
			pr = params.K(group, q.key, q.min, q.max, v, params.Label(q.label))
		}
		if pr.Kind == params.Switch {
			pr.Min, pr.Max = q.min, q.max
		}
		set.Params = append(set.Params, pr)
	}
	return set
}

// Text is what the plugin shows for a value of the parameter with this key ("Classic", "440 Hz"), empty when it shows none
func (p *Plugin) Text(key string, value float64) string {
	i, ok := p.keys[key]
	if !ok || p.inst == nil {
		return ""
	}
	var buf [256]C.char
	if C.pi_param_text(p.inst, C.uint32_t(p.params[i].id), C.double(value), &buf[0], 256) == 0 {
		return ""
	}
	return C.GoString(&buf[0])
}

// Value is where the plugin has the parameter with this key now (0 to 1)
func (p *Plugin) Value(key string) float64 {
	i, ok := p.keys[key]
	if !ok || p.inst == nil {
		return 0
	}
	return float64(C.pi_param_value(p.inst, C.uint32_t(p.params[i].id)))
}

// presetKey is the key of the setting that loads a preset of the plugin: the number of the preset
const presetKey = "preset"

// label is what a preset is called in a list: the kind of sound it is, then its name
func (pr Preset) label() string {
	if pr.Category != "" {
		return pr.Category + " / " + pr.Name
	}
	return pr.Name
}

// Presets are the presets the plugin has, as it lists them
func (p *Plugin) Presets() []Preset { return p.presets }

// LoadPreset loads the preset with this number into the plugin. The plugin says when it is done: a long preset loads on its own thread, so this is quick
func (p *Plugin) LoadPreset(i int) error {
	if i < 0 || i >= len(p.presets) {
		return fmt.Errorf("no preset %d", i)
	}
	p.loadMu.Lock()
	defer p.loadMu.Unlock()
	if p.inst == nil || p.closed.Load() {
		return errors.New("the plugin is gone")
	}
	pr := p.presets[i]
	loc, key := C.CString(pr.location), C.CString(pr.loadKey)
	defer C.free(unsafe.Pointer(loc))
	defer C.free(unsafe.Pointer(key))
	if C.pi_load_preset(p.inst, C.uint32_t(pr.kind), loc, key) == 0 {
		return fmt.Errorf("%s: the plugin would not load it", pr.Name)
	}
	p.preset.Store(int64(i + 1))
	return nil
}

// Loaded is the number of the preset loaded last, -1 when none was
func (p *Plugin) Loaded() int { return int(p.preset.Load()) - 1 }

// Keys are the keys of the messages that move the parameters
func (p *Plugin) Keys() []string {
	out := make([]string, len(p.params))
	for i, q := range p.params {
		out[i] = q.key
	}
	return out
}

// Close lets the plugin go: the audio thread frees it at its next block (or Free does, when it doesn't run)
func (p *Plugin) Close() { p.closed.Store(true) }

// Free lets the plugin go at once, for a plugin that is not running on the audio thread
func (p *Plugin) Free() {
	if p.inst != nil {
		C.pi_free(p.inst)
		p.inst = nil
	}
}

func (p *Plugin) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
	}
}

// pitchOf is the pitch of a note message: its key and how far the pitch is above it in semitones (between -0.5 and 0.5)
func pitchOf(msg psalm.Message) (key int, tuning float64) {
	pitch := 60.0
	msg.IfFltVal("pitch", func(f float64) { pitch = f })
	msg.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			pitch = 69 + 12*math.Log2(f/440)
		}
	})
	key = int(math.Round(pitch))
	return min(127, max(0, key)), pitch - float64(key)
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
				C.pi_note(p.inst, 0, C.uint32_t(at), C.int(n.key), 0, C.int32_t(n.id), 0)
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
		if len(p.presets) > 0 {
			go func() { _ = p.LoadPreset(int(math.Round(f))) }()
		}
	})
	for key, i := range p.keys {
		msg.IfFltVal(key, func(f float64) {
			q := p.params[i]
			C.pi_param(p.inst, C.uint32_t(q.id), C.double(min(q.max, max(q.min, f))), 0)
		})
	}
}

func (p *Plugin) start(msg psalm.Message, at uint32) note {
	m, _ := msg.Msg("message")
	key, tuning := pitchOf(m)
	p.nextID++
	n := note{id: p.nextID, key: key}
	amp := min(1, max(0, msg.MustFltVal("amplitude")))
	C.pi_note(p.inst, 1, C.uint32_t(at), C.int(key), C.double(amp), C.int32_t(n.id), C.double(tuning))
	return n
}

// AllNotesOff lets go of every note
func (p *Plugin) AllNotesOff() {
	for id, n := range p.held {
		C.pi_note(p.inst, 0, 0, C.int(n.key), 0, C.int32_t(n.id), 0)
		delete(p.held, id)
	}
	for _, n := range p.timed {
		C.pi_note(p.inst, 0, 0, C.int(n.key), 0, C.int32_t(n.id), 0)
	}
	p.timed = p.timed[:0]
}

// Synthesize runs the plugin over a block
func (p *Plugin) Synthesize() {
	out := p.Output(0).Channels
	n := len(out[0])
	out.Clear()
	if p.closed.Load() {
		p.Free()
	}
	if p.inst == nil {
		return
	}
	// the notes of a duration let go when their time is up
	for i := 0; i < len(p.timed); {
		if p.timed[i].samples <= n {
			t := p.timed[i]
			C.pi_note(p.inst, 0, C.uint32_t(min(max(t.samples, 0), n-1)), C.int(t.key), 0, C.int32_t(t.id), 0)
			p.timed = append(p.timed[:i], p.timed[i+1:]...)
			continue
		}
		p.timed[i].samples -= n
		i++
	}
	var in0, in1 *C.float
	if p.inCh > 0 {
		in := p.Input(0).Channels
		in0 = (*C.float)(unsafe.Pointer(&in[0][0]))
		in1 = in0
		if p.inCh > 1 && len(in) > 1 {
			in1 = (*C.float)(unsafe.Pointer(&in[1][0]))
		}
	}
	C.pi_process(p.inst, in0, in1, (*C.float)(unsafe.Pointer(&out[0][0])), (*C.float)(unsafe.Pointer(&out[1][0])), C.uint32_t(n))
}

var _ psalm.Modular = (*Plugin)(nil)
