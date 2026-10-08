package panel

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Presets keeps setups of the panel in a folder, one JSON file each (the snapshots: every control
// and every grid), and lets the page save, load and delete them by name. The folder is made when
// it isn't there
func (pn *Panel) Presets(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	pn.presetMu.Lock()
	pn.presetDir = dir
	pn.presetMu.Unlock()
	return nil
}

// presetName is what a preset may be called: letters, digits, spaces, - _ and ., so a name can't
// reach outside the folder
var presetName = regexp.MustCompile(`^[\pL\pN _.-]{1,64}$`)

// presetFile returns the file of a preset, or an error for a name that isn't allowed or a panel
// without presets
func (pn *Panel) presetFile(name string) (string, error) {
	pn.presetMu.Lock()
	dir := pn.presetDir
	pn.presetMu.Unlock()
	if dir == "" {
		return "", errors.New("panel: no presets (Presets sets their folder)")
	}
	name = strings.TrimSpace(name)
	if !presetName.MatchString(name) || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("panel: %q can't be the name of a preset: use letters, digits, spaces, - _ and .", name)
	}
	return filepath.Join(dir, name+".json"), nil
}

// PresetNames returns the names of the presets, sorted
func (pn *Panel) PresetNames() ([]string, error) {
	pn.presetMu.Lock()
	dir := pn.presetDir
	pn.presetMu.Unlock()
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".json"); ok && !e.IsDir() {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// SavePreset saves the setup of the panel under a name (an existing preset of that name is
// replaced)
func (pn *Panel) SavePreset(name string) error {
	file, err := pn.presetFile(name)
	if err != nil {
		return err
	}
	return pn.Save(file)
}

// LoadPreset sets the panel to a saved setup
func (pn *Panel) LoadPreset(name string) error {
	file, err := pn.presetFile(name)
	if err != nil {
		return err
	}
	return pn.Load(file)
}

// DeletePreset deletes a saved setup
func (pn *Panel) DeletePreset(name string) error {
	file, err := pn.presetFile(name)
	if err != nil {
		return err
	}
	return os.Remove(file)
}

// presetsEnabled tells whether the panel keeps presets
func (pn *Panel) presetsEnabled() bool {
	pn.presetMu.Lock()
	defer pn.presetMu.Unlock()
	return pn.presetDir != ""
}

// presetState is what a panel keeps for its presets
type presetState struct {
	presetMu  sync.Mutex
	presetDir string
}
