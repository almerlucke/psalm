package dxeditor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/almerlucke/psalm/synths/dx"
)

// SysExFolder gives the editor a folder of .syx files: the page lists them, choosing one makes its
// voices the bank (the program list), and the page saves the patch (a single voice dump) or the
// bank (a 32 voice dump) into it under a name. Off without it
func SysExFolder(dir string) Option { return func(e *Editor) { e.sysexDir = dir } }

// sysexState is what the page gets of the folder
type sysexState struct {
	Keeps bool     `json:"keeps"`
	Files []string `json:"files"`
	File  string   `json:"file"` // the file the bank is from, "" for the built in bank
	Bank  []string `json:"bank"` // names of the bank as it is now
	Note  string   `json:"note"`
	Error bool     `json:"error"`
}

// SysExFile checks the name of a file in the folder and returns it with its .syx suffix: letters,
// digits, spaces, - _ . and (), no path, no leading dot
func SysExFile(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len(name) > 4 && strings.EqualFold(name[len(name)-4:], ".syx") {
		name = strings.TrimSpace(name[:len(name)-4])
	}
	if name == "" || len(name) > 64 || strings.HasPrefix(name, ".") || strings.Contains(name, "..") {
		return "", fmt.Errorf("%q can't be the name of a SysEx file", name)
	}
	for _, r := range name {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(" -_.()", r)) {
			return "", fmt.Errorf("%q can't be the name of a SysEx file: use letters, digits, spaces, - _ . and ()", name)
		}
	}
	return name + ".syx", nil
}

// SysExFiles returns the .syx files in the folder, sorted (nil without a folder)
func (e *Editor) SysExFiles() []string {
	if e.sysexDir == "" {
		return nil
	}
	entries, _ := os.ReadDir(e.sysexDir)
	var names []string
	for _, f := range entries {
		if f.Type().IsRegular() && strings.EqualFold(filepath.Ext(f.Name()), ".syx") {
			names = append(names, f.Name())
		}
	}
	slices.Sort(names)
	return names
}

// LoadSysEx makes the voices of a file of the folder ("" for the built in ones) the bank
func (e *Editor) LoadSysEx(file string) error {
	if file == "" {
		e.setBank("", e.builtin)
		return nil
	}
	if e.sysexDir == "" {
		return errors.New("no folder for SysEx files")
	}
	name, err := SysExFile(file)
	if err != nil {
		return err
	}
	if !slices.Contains(e.SysExFiles(), name) {
		return fmt.Errorf("%s isn't in the folder", name)
	}
	patches, err := dx.LoadSysEx(filepath.Join(e.sysexDir, name))
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	e.setBank(name, patches)
	return nil
}

func (e *Editor) setBank(file string, patches []dx.Patch) {
	e.mu.Lock()
	e.bank, e.file = patches, file
	e.mu.Unlock()
}

// SaveSysEx writes the patch as it is (what "bank" is false) or the bank as it is into the folder
// under a name; a file there is only replaced when overwrite is true (os.ErrExist otherwise).
// It returns the name of the file
func (e *Editor) SaveSysEx(name string, bank, overwrite bool) (string, error) {
	if e.sysexDir == "" {
		return "", errors.New("no folder for SysEx files")
	}
	file, err := SysExFile(name)
	if err != nil {
		return "", err
	}
	var data []byte
	if bank {
		e.mu.Lock()
		patches := slices.Clone(e.bank)
		e.mu.Unlock()
		if len(patches) == 0 {
			return "", errors.New("no bank to save")
		}
		data = dx.EncodeBank(patches)
	} else {
		data = dx.EncodeSingle(e.current())
	}
	if err := os.MkdirAll(e.sysexDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(e.sysexDir, file)
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("%s is there already: %w", file, os.ErrExist)
	}
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return "", err
	}
	return file, f.Close()
}

// current is the patch as edited now: the one loaded last, with the settings as the page has them
func (e *Editor) current() dx.Patch {
	e.mu.Lock()
	p := e.base
	p.Name = e.name
	e.mu.Unlock()
	p.Algorithm = int(e.Value()) - 1
	for _, s := range dx.Settings {
		if v, _ := p.Get(s.Name); v != e.parts[s.Name].Value() {
			p.Set(s.Name, e.parts[s.Name].Value())
		}
	}
	p.Routing = e.routing()
	return p
}

// sysexMessage handles what the page sends about the folder
func (e *Editor) sysexMessage(raw json.RawMessage) {
	var m struct {
		Refresh   bool    `json:"refresh"`
		Load      *string `json:"load"`
		Save      string  `json:"save"`
		Bank      bool    `json:"bank"`
		Overwrite bool    `json:"overwrite"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	note, failed := "", error(nil)
	switch {
	case m.Load != nil:
		if failed = e.LoadSysEx(*m.Load); failed == nil {
			e.mu.Lock()
			n := len(e.bank)
			e.mu.Unlock()
			note = fmt.Sprintf("%d patches", n)
		}
	case m.Save != "":
		var file string
		if file, failed = e.SaveSysEx(m.Save, m.Bank, m.Overwrite); failed == nil {
			note = "saved " + file
		}
	}
	e.mu.Lock()
	e.files = e.SysExFiles()
	e.note, e.noteErr = note, failed != nil
	if failed != nil {
		e.note = failed.Error()
	}
	e.mu.Unlock()
}

func (e *Editor) sysexState() *sysexState {
	if e.sysexDir == "" {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	s := &sysexState{Keeps: true, Files: e.files, File: e.file, Note: e.note, Error: e.noteErr, Bank: []string{}}
	if s.Files == nil {
		s.Files = []string{}
	}
	for _, p := range e.bank {
		s.Bank = append(s.Bank, p.Name)
	}
	return s
}

// queueSysEx hands a message of the page to the goroutine that works on the files (dropped when
// it is far behind)
func (e *Editor) queueSysEx(raw json.RawMessage) {
	if e.jobs == nil {
		return
	}
	select {
	case e.jobs <- slices.Clone(raw):
	default:
	}
}
