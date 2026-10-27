package vst3

/*
#include <stdlib.h>
#include "glue.h"
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unsafe"
)

// presetKey is the key of the setting that loads a preset of the plugin: the number of the preset
const presetKey = "preset"

// Preset is a sound a plugin has: a program of its own (the list it tells of), or a .vstpreset file made for it
type Preset struct {
	Name     string
	Category string // the vendor and plugin folder of a file, "program" for a program
	program  int    // the number of a program, -1 for a file
	path     string
}

func (pr Preset) label() string {
	if pr.Category != "" {
		return pr.Category + " / " + pr.Name
	}
	return pr.Name
}

// PresetDirs are where the files of presets are looked for: presets/<vendor>/<plugin>/<name>.vstpreset in them. $VST3_PRESET_PATH comes first
func PresetDirs() []string {
	dirs := filepath.SplitList(os.Getenv("VST3_PRESET_PATH"))
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".vst3", "presets"))
	}
	return append(dirs, "/usr/share/vst3/presets", "/usr/local/share/vst3/presets")
}

// vstpreset is a file of a preset: "VST3", a version, the class id as text, the offset of the list of chunks; the chunks "Comp" (the state of the component) and "Cont" (that of
// the controller) are what a plugin is given back
type vstpreset struct {
	classID    string
	comp, cont []byte
}

func readVstpreset(path string, chunks bool) (*vstpreset, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	head := make([]byte, 48)
	if _, err := f.ReadAt(head, 0); err != nil || string(head[:4]) != "VST3" {
		return nil, errors.New("not a preset")
	}
	vp := &vstpreset{classID: strings.ToUpper(strings.TrimRight(string(head[8:40]), "\x00"))}
	if !chunks {
		return vp, nil
	}
	listAt := int64(binary.LittleEndian.Uint64(head[40:48]))
	list := make([]byte, 8)
	if _, err := f.ReadAt(list, listAt); err != nil || string(list[:4]) != "List" {
		return nil, errors.New("no list of chunks")
	}
	n := int(binary.LittleEndian.Uint32(list[4:8]))
	entry := make([]byte, 20*n)
	if _, err := f.ReadAt(entry, listAt+8); err != nil {
		return nil, err
	}
	for i := 0; i < n; i++ {
		e := entry[20*i:]
		off, size := int64(binary.LittleEndian.Uint64(e[4:12])), int64(binary.LittleEndian.Uint64(e[12:20]))
		if size < 0 || size > 1<<28 {
			return nil, errors.New("a chunk too big")
		}
		data := make([]byte, size)
		if _, err := f.ReadAt(data, off); err != nil {
			return nil, err
		}
		switch string(e[:4]) {
		case "Comp":
			vp.comp = data
		case "Cont":
			vp.cont = data
		}
	}
	return vp, nil
}

var (
	presetMu    sync.Mutex
	presetFiles = map[string][]Preset{} // by class id and the places searched
)

// filePresets are the .vstpreset files made for the class of a plugin
func filePresets(d Descriptor) []Preset {
	presetMu.Lock()
	defer presetMu.Unlock()
	key := d.CID + "|" + strings.Join(PresetDirs(), ":") // found once for these places
	if ps, ok := presetFiles[key]; ok {
		return ps
	}
	var out []Preset
	for _, dir := range PresetDirs() {
		_ = filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() || !strings.HasSuffix(path, ".vstpreset") {
				return nil
			}
			if vp, err := readVstpreset(path, false); err == nil && vp.classID == strings.ToUpper(d.CID) {
				rel, _ := filepath.Rel(dir, filepath.Dir(path))
				out = append(out, Preset{Name: strings.TrimSuffix(filepath.Base(path), ".vstpreset"), Category: filepath.ToSlash(rel), program: -1, path: path})
			}
			return nil
		})
	}
	slices.SortFunc(out, func(a, b Preset) int { return strings.Compare(strings.ToLower(a.label()), strings.ToLower(b.label())) })
	presetFiles[key] = out
	return out
}

// readPrograms asks the plugin for its programs (on the main thread, at the start)
func (p *Plugin) readPrograms() {
	var param C.uint32_t
	var steps C.int
	n := int(C.v3_programs(p.inst, &param, &steps))
	if n == 0 {
		return
	}
	p.progParam, p.progSteps = uint32(param), int(steps)
	var name [256]C.char
	for i := 0; i < n; i++ {
		if C.v3_program_name(p.inst, C.int(i), &name[0], 256) == 0 {
			continue
		}
		p.presets = append(p.presets, Preset{Name: strings.TrimSpace(C.GoString(&name[0])), Category: "program", program: i})
	}
}

// Presets are the sounds the plugin has: its programs, then the files made for it
func (p *Plugin) Presets() []Preset { return p.presets }

// Loaded is the number of the preset loaded last, -1 when none was
func (p *Plugin) Loaded() int { return int(p.preset.Load()) - 1 }

func (p *Plugin) loadPresetMain(i int) error {
	if i < 0 || i >= len(p.presets) {
		return fmt.Errorf("no preset %d", i)
	}
	p.loadMu.Lock()
	if p.inst == nil || p.closed.Load() {
		p.loadMu.Unlock()
		return errors.New("the plugin is gone")
	}
	pr := p.presets[i]
	var err error
	if pr.program >= 0 { // a program: the parameter that selects it is set
		v := 0.0
		if p.progSteps > 0 {
			v = math.Min(1, float64(pr.program)/float64(p.progSteps))
		}
		C.v3_param(p.inst, C.uint32_t(p.progParam), C.double(v), 0)
		C.v3_param_set_now(p.inst, C.uint32_t(p.progParam), C.double(v))
	} else {
		var vp *vstpreset
		if vp, err = readVstpreset(pr.path, true); err == nil {
			p.loadMu.Unlock()
			err = p.loadStateParts(vp.comp, vp.cont)
			p.loadMu.Lock()
		}
	}
	p.loadMu.Unlock()
	if err != nil {
		return fmt.Errorf("%s: %w", pr.Name, err)
	}
	p.preset.Store(int64(i + 1))
	go p.tellEditor()
	return nil
}

// LoadPreset loads the preset with this number into the plugin
func (p *Plugin) LoadPreset(i int) (err error) {
	mainDo(func() { err = p.loadPresetMain(i) })
	return
}

// loadStateParts gives the plugin the state of its component and of its controller
func (p *Plugin) loadStateParts(comp, ctrl []byte) error {
	p.loadMu.Lock()
	defer p.loadMu.Unlock()
	if p.inst == nil || p.closed.Load() {
		return errors.New("the plugin is gone")
	}
	var cp, tp *C.uint8_t
	if len(comp) > 0 {
		cp = (*C.uint8_t)(unsafe.Pointer(&comp[0]))
	}
	if len(ctrl) > 0 {
		tp = (*C.uint8_t)(unsafe.Pointer(&ctrl[0]))
	}
	if C.v3_state_load(p.inst, cp, C.size_t(len(comp)), tp, C.size_t(len(ctrl))) == 0 {
		return fmt.Errorf("%s would not take it", p.desc.Name)
	}
	return nil
}
