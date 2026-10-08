package panel

import (
	"encoding/json"
	"os"
)

// Snapshot returns the value of every fader, knob, toggle and select by name, and the cells of the
// grids (as "grid/pattern/row/step/field", the cells at their defaults left out): the state of the
// panel, to keep and restore (a preset of the patch)
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
