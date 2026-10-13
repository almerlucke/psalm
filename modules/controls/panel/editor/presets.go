package editor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// Store keeps presets of a synth or an effect: the factory ones (by name, described by the
// synth) and your own, saved in a folder as a JSON file each (the settings by message key, so a
// preset stays good when the editor is named otherwise). An editor has one (Factory, Keep); the
// page shows its presets above the controls: choose one, save the settings under a name, delete
// one of your own
type Store struct {
	mu      sync.Mutex
	dir     string
	factory []string
	mine    map[string]map[string]float64
	current string
	note    string // the last thing that went wrong, for the page

	values  func() map[string]float64 // the settings now (from any goroutine)
	apply   func(map[string]float64)  // sets them (on the audio thread)
	fromKey func(name string)         // loads a factory preset (on the audio thread)
}

// NewStore makes a store of presets for a plugin of its own (the editor has one): values gives
// the settings as they are (by key), apply sets them (from any goroutine: setting the controls).
// Keep gives it a folder; the plugin shows State on its page and hands Message what the page
// sends ({"load": name, "mine": true}, {"save": name}, {"delete": name})
func NewStore(values func() map[string]float64, apply func(map[string]float64)) *Store {
	return &Store{mine: map[string]map[string]float64{}, values: values, apply: apply}
}

// Keep keeps presets of your own in a folder (made when it isn't there) and reads the ones there
func (s *Store) Keep(dir string) error { return s.keep(dir) }

// SetFactory gives the store presets to choose from that aren't files: their names, and how one
// is loaded (setting the controls)
func (s *Store) SetFactory(names []string, load func(name string)) {
	s.mu.Lock()
	s.factory, s.fromKey = append([]string(nil), names...), load
	s.mu.Unlock()
}

// State is what a page shows of the presets: their names, the one chosen last, what went wrong
func (s *Store) State() any { return s.state() }

// Message handles what a page sends about the presets (on the audio thread: files are written on a
// goroutine of their own)
func (s *Store) Message(raw json.RawMessage) { s.message(raw) }

// file is a preset of your own on disk
type file struct {
	Settings map[string]float64 `json:"settings"`
}

// presetName is what a preset may be called: letters, digits, spaces, - _ and ., so a name can't
// reach outside the folder
var presetName = regexp.MustCompile(`^[\pL\pN _.-]{1,64}$`)

func (s *Store) path(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !presetName.MatchString(name) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("%q can't be the name of a preset: use letters, digits, spaces, - _ and .", name)
	}
	if s.dir == "" {
		return "", errors.New("no folder for presets of your own")
	}
	return filepath.Join(s.dir, name+".json"), nil
}

// keep reads the presets of your own in a folder (made when it isn't there)
func (s *Store) keep(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	mine := map[string]map[string]float64{}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		var f file
		if err := json.Unmarshal(data, &f); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
		mine[name] = f.Settings
	}
	s.mu.Lock()
	s.dir, s.mine = dir, mine
	s.mu.Unlock()
	return nil
}

// Save saves the settings as they are under a name (a preset of your own of that name is
// replaced). From any goroutine
func (s *Store) Save(name string) error {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	path, err := s.path(name)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	settings := s.values()
	data, err := json.MarshalIndent(file{Settings: settings}, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	s.mu.Lock()
	s.mine[name] = settings
	s.current = name
	s.mu.Unlock()
	return nil
}

// Delete deletes a preset of your own. From any goroutine
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	path, err := s.path(name)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.mine, strings.TrimSpace(name))
	if s.current == name {
		s.current = ""
	}
	s.mu.Unlock()
	return nil
}

// Load sets the settings to a preset, of your own (mine) or a factory one: they move to it, the
// synth gets them on the next control tick. From any goroutine
func (s *Store) Load(name string, mine bool) error {
	s.mu.Lock()
	settings, ok := s.mine[name]
	isFactory := slices.Contains(s.factory, name)
	s.mu.Unlock()
	switch {
	case mine && ok:
		s.apply(settings)
	case !mine && isFactory && s.fromKey != nil:
		s.fromKey(name)
	default:
		return fmt.Errorf("no preset %q", name)
	}
	s.mu.Lock()
	s.current = name
	s.mu.Unlock()
	return nil
}

// Mine returns the names of the presets of your own, sorted
func (s *Store) Mine() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.mine))
	for n := range s.mine {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// state is what the page shows of the presets
type state struct {
	Factory []string `json:"factory"`
	Mine    []string `json:"mine"`
	Keeps   bool     `json:"keeps"` // presets of your own can be saved
	Current string   `json:"current"`
	Note    string   `json:"note"`
}

func (s *Store) state() any {
	mine := s.Mine()
	s.mu.Lock()
	defer s.mu.Unlock()
	return state{Factory: s.factory, Mine: mine, Keeps: s.dir != "", Current: s.current, Note: s.note}
}

// message handles what the page asks (on the audio thread): load a preset, save, delete (on a
// goroutine of their own: files aren't written on the audio thread)
func (s *Store) message(raw json.RawMessage) {
	var m struct {
		Load   *string `json:"load"`
		Mine   bool    `json:"mine"`
		Save   *string `json:"save"`
		Delete *string `json:"delete"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	report := func(err error) {
		s.mu.Lock()
		s.note = ""
		if err != nil {
			s.note = err.Error()
		}
		s.mu.Unlock()
	}
	switch {
	case m.Load != nil:
		report(s.Load(*m.Load, m.Mine))
	case m.Save != nil:
		name := *m.Save
		go func() { report(s.Save(name)) }()
	case m.Delete != nil:
		name := *m.Delete
		go func() { report(s.Delete(name)) }()
	}
}
