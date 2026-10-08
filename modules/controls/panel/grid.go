package panel

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
)

// GridKind is the kind of a grid on the page
const GridKind Kind = "grid"

// Field is a value every cell of a grid has: the velocity of a step, its chance, its roll ...
type Field struct {
	Name    string   `json:"name"`
	Min     float64  `json:"min"`
	Max     float64  `json:"max"`
	Default float64  `json:"default"`
	Step    float64  `json:"step"`              // rounds the value (1 for whole numbers)
	Unit    string   `json:"unit,omitempty"`    //
	Options []string `json:"options,omitempty"` // names of the whole values from Min, for a choice
}

func (f Field) clamp(v float64) float64 {
	if math.IsNaN(v) {
		v = f.Default
	}
	v = math.Max(f.Min, math.Min(v, f.Max))
	if f.Step > 0 {
		v = f.Min + math.Round((v-f.Min)/f.Step)*f.Step
		v = math.Max(f.Min, math.Min(v, f.Max))
	}
	return v
}

// Grid is a control of rows and steps, in patterns: every cell holds the values of the fields (a
// step sequencer's steps: velocity, chance, roll ...). The page draws it as a grid to tap and edit.
// Values, the pattern and the playheads are safe from any goroutine: the page, the sequencer
// playing the grid on the audio thread and your own code
type Grid struct {
	Name     string
	Section  string
	Rows     []string
	Steps    int
	Patterns int
	Fields   []Field

	cells    []atomic.Uint64 // float64 bits, [pattern][row][step][field]
	pattern  atomic.Int32    // the pattern asked for (shown and edited on the page)
	playing  atomic.Int32    // the pattern playing (the sequencer switches at its own moment)
	playhead []atomic.Int32  // per row the step playing, -1 none
	lengths  []atomic.Int32  // per row how many steps it plays (rows of their own length loop apart)
	version  atomic.Int64    // counts the changes of the cells and the pattern, for the page
}

// Grid adds a grid of rows and steps, with a number of patterns and the fields of its cells
func (pn *Panel) Grid(name string, rows []string, steps, patterns int, fields ...Field) *Grid {
	if len(rows) == 0 || steps <= 0 || len(fields) == 0 {
		panic(fmt.Sprintf("panel: grid %q needs rows, steps and fields", name))
	}
	patterns = max(1, patterns)
	g := &Grid{Name: name, Rows: append([]string(nil), rows...), Steps: steps, Patterns: patterns, Fields: append([]Field(nil), fields...)}
	g.cells = make([]atomic.Uint64, patterns*len(rows)*steps*len(fields))
	for i := range g.cells {
		g.cells[i].Store(math.Float64bits(g.Fields[i%len(fields)].Default))
	}
	g.playhead = make([]atomic.Int32, len(rows))
	g.lengths = make([]atomic.Int32, len(rows))
	for i := range g.playhead {
		g.playhead[i].Store(-1)
		g.lengths[i].Store(int32(steps))
	}

	pn.mu.Lock()
	defer pn.mu.Unlock()
	if _, ok := pn.byName[name]; ok || pn.grids[name] != nil {
		panic(fmt.Sprintf("panel: a control named %q twice", name))
	}
	if pn.section == "" {
		pn.section = pn.title
		pn.sections = append(pn.sections, pn.section)
	}
	g.Section = pn.section
	if pn.grids == nil {
		pn.grids = map[string]*Grid{}
	}
	pn.grids[name] = g
	pn.order = append(pn.order, item{grid: g})
	return g
}

// GridByName returns a grid by name, nil when the panel has none of that name
func (pn *Panel) GridByName(name string) *Grid {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return pn.grids[name]
}

// Field returns the index of a field by name, -1 when the grid has none of that name
func (g *Grid) Field(name string) int {
	for i, f := range g.Fields {
		if f.Name == name {
			return i
		}
	}
	return -1
}

// Row returns the index of a row by name, -1 when the grid has none of that name
func (g *Grid) Row(name string) int {
	for i, r := range g.Rows {
		if r == name {
			return i
		}
	}
	return -1
}

func (g *Grid) index(pattern, row, step, field int) int {
	if pattern < 0 || pattern >= g.Patterns || row < 0 || row >= len(g.Rows) || step < 0 || step >= g.Steps || field < 0 || field >= len(g.Fields) {
		return -1
	}
	return ((pattern*len(g.Rows)+row)*g.Steps+step)*len(g.Fields) + field
}

// Value returns a field of a cell
func (g *Grid) Value(pattern, row, step, field int) float64 {
	i := g.index(pattern, row, step, field)
	if i < 0 {
		return 0
	}
	return math.Float64frombits(g.cells[i].Load())
}

// Set sets a field of a cell (clamped to the field's range and step)
func (g *Grid) Set(pattern, row, step, field int, v float64) {
	i := g.index(pattern, row, step, field)
	if i < 0 {
		return
	}
	g.cells[i].Store(math.Float64bits(g.Fields[field].clamp(v)))
	g.version.Add(1)
}

// Clear sets every cell of a pattern back to the defaults of the fields
func (g *Grid) Clear(pattern int) {
	for row := range g.Rows {
		for step := range g.Steps {
			for f, field := range g.Fields {
				g.Set(pattern, row, step, f, field.Default)
			}
		}
	}
}

// Copy copies every cell of a pattern to another
func (g *Grid) Copy(from, to int) {
	if from == to || from < 0 || from >= g.Patterns || to < 0 || to >= g.Patterns {
		return
	}
	for row := range g.Rows {
		for step := range g.Steps {
			for f := range g.Fields {
				g.Set(to, row, step, f, g.Value(from, row, step, f))
			}
		}
	}
}

// Pattern returns the pattern asked for (on the page, or with SetPattern)
func (g *Grid) Pattern() int { return int(g.pattern.Load()) }

// SetPattern asks for a pattern; a sequencer switches to it at its own moment (the end of a loop)
func (g *Grid) SetPattern(pattern int) {
	g.pattern.Store(int32(max(0, min(pattern, g.Patterns-1))))
	g.version.Add(1)
}

// Playing returns the pattern playing, SetPlaying is what a sequencer tells
func (g *Grid) Playing() int           { return int(g.playing.Load()) }
func (g *Grid) SetPlaying(pattern int) { g.playing.Store(int32(pattern)) }

// SetPlayhead tells the page which step of a row plays (-1 none): a sequencer calls it
func (g *Grid) SetPlayhead(row, step int) {
	if row >= 0 && row < len(g.playhead) {
		g.playhead[row].Store(int32(step))
	}
}

// SetLength tells how many steps a row plays (the page dims the others): a sequencer calls it
func (g *Grid) SetLength(row, steps int) {
	if row >= 0 && row < len(g.lengths) {
		g.lengths[row].Store(int32(max(1, min(steps, g.Steps))))
	}
}

// Lengths returns how many steps every row plays
func (g *Grid) Lengths() []int {
	ls := make([]int, len(g.lengths))
	for i := range g.lengths {
		ls[i] = int(g.lengths[i].Load())
	}
	return ls
}

// Playheads returns the step playing of every row
func (g *Grid) Playheads() []int {
	ps := make([]int, len(g.playhead))
	for i := range g.playhead {
		ps[i] = int(g.playhead[i].Load())
	}
	return ps
}

// values returns all cells, in the order of index
func (g *Grid) values() []float64 {
	vs := make([]float64, len(g.cells))
	for i := range g.cells {
		vs[i] = math.Float64frombits(g.cells[i].Load())
	}
	return vs
}

// patternName is how a pattern is called: A, B, C ...
func patternName(p int) string {
	if p < 26 {
		return string(rune('A' + p))
	}
	return strconv.Itoa(p + 1)
}

// snapshot adds the grid to a snapshot: its pattern, and every cell that isn't at its default, as
// "name/pattern/row/step/field"
func (g *Grid) snapshot(values map[string]float64) {
	values[g.Name] = float64(g.Pattern())
	for p := range g.Patterns {
		for r, row := range g.Rows {
			for s := range g.Steps {
				for f, field := range g.Fields {
					if v := g.Value(p, r, s, f); v != field.Default {
						values[fmt.Sprintf("%s/%s/%s/%d/%s", g.Name, patternName(p), row, s+1, field.Name)] = v
					}
				}
			}
		}
	}
}

// restore takes the grid from a snapshot when the snapshot has it: the cells it leaves out go back
// to their defaults
func (g *Grid) restore(values map[string]float64) {
	pattern, ok := values[g.Name]
	if !ok {
		return
	}
	for p := range g.Patterns {
		g.Clear(p)
	}
	g.SetPattern(int(pattern))
	prefix := g.Name + "/"
	for key, v := range values {
		rest, ok := strings.CutPrefix(key, prefix)
		if !ok {
			continue
		}
		parts := strings.Split(rest, "/")
		if len(parts) != 4 {
			continue
		}
		p := -1
		for i := range g.Patterns {
			if patternName(i) == parts[0] {
				p = i
			}
		}
		step, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		g.Set(p, g.Row(parts[1]), step-1, g.Field(parts[3]), v)
	}
}
