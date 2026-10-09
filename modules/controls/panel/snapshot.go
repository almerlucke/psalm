package panel

import (
	"encoding/json"
	"os"
	"strings"
)

// Snapshot returns the value of every fader, knob, toggle and select by name, and the cells of the
// grids (as "grid/pattern/row/step/field", the cells at their defaults left out) and which cues of
// the cue lists are on (as "list/cue", 1 or 0), and what the mounts keep (as "mount/..."): the state
// of the panel, to keep and restore (a preset of the patch)
func (pn *Panel) Snapshot() map[string]float64 {
	values := map[string]float64{}
	for _, par := range pn.Params() {
		if par.Kind != ButtonKind && par.Kind != MeterKind {
			values[par.Name] = par.Value()
		}
	}
	for _, g := range pn.Grids() {
		g.snapshot(values)
	}
	for _, cs := range pn.cueLists() {
		cs.snapshot(values)
	}
	for _, mt := range pn.mountList() {
		if k, ok := mt.m.(MountKeeper); ok {
			for key, v := range k.SnapshotMount() {
				values[mt.Name+"/"+key] = v
			}
		}
	}
	return values
}

// Restore sets the controls to the values of a snapshot; names the panel doesn't have are left out
func (pn *Panel) Restore(values map[string]float64) {
	for name, v := range values {
		if par := pn.Param(name); par != nil {
			par.Set(v)
		}
	}
	for _, g := range pn.Grids() {
		g.restore(values)
	}
	for _, cs := range pn.cueLists() {
		cs.restore(values)
	}
	for _, mt := range pn.mountList() {
		k, ok := mt.m.(MountKeeper)
		if !ok {
			continue
		}
		mine := map[string]float64{}
		for key, v := range values {
			if rest, ok := strings.CutPrefix(key, mt.Name+"/"); ok {
				mine[rest] = v
			}
		}
		if len(mine) > 0 {
			k.RestoreMount(mine)
		}
	}
}

// Save writes a snapshot to a file as JSON
func (pn *Panel) Save(path string) error {
	data, err := json.MarshalIndent(pn.Snapshot(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load restores a snapshot from a file written by Save
func (pn *Panel) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var values map[string]float64
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	pn.Restore(values)
	return nil
}
