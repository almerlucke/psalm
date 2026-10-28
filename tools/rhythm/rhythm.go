// Package rhythm has the techniques that make rhythms: euclidean rhythms and their rotations, additive rhythms (3+3+2), cellular automata, conditions on a hit (every second
// time through, only in a fill), lanes of a value a step with a length of their own (polymeter), and grooves (the timing and loudness of the steps of a bar)
package rhythm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Euclid spreads k hits as evenly as it can over n steps (Bjorklund), turned rot steps to the right (a negative number to the left)
func Euclid(k, n, rot int) []bool {
	if n <= 0 {
		return nil
	}
	k = max(0, min(k, n))
	out := make([]bool, n)
	for i := range n { // the hit i of k falls on the step floor(i*n/k) of the rhythm that starts on a hit
		if k > 0 && (i*k)%n < k {
			out[i] = true
		}
	}
	return Rotate(out, rot)
}

// Rotate turns a rhythm n steps to the right (to the left when negative)
func Rotate(r []bool, n int) []bool {
	l := len(r)
	if l == 0 {
		return r
	}
	out := make([]bool, l)
	for i, b := range r {
		out[((i+n)%l+l)%l] = b
	}
	return out
}

// Additive makes a rhythm of groups of steps ("3+3+2"): a hit on the first step of every group
func Additive(groups string) ([]bool, error) {
	var out []bool
	for _, g := range strings.Split(groups, "+") {
		n, err := strconv.Atoi(strings.TrimSpace(g))
		if err != nil || n < 1 || n > 32 {
			return nil, fmt.Errorf("%q: a group of 1 to 32 steps (3+3+2)", g)
		}
		out = append(out, true)
		for range n - 1 {
			out = append(out, false)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no groups: 3+3+2")
	}
	return out, nil
}

// Cellular runs an elementary cellular automaton (a rule 0 to 255, rule 30 or 90 for instance) on a ring of width cells for a number of generations, and gives a row of it: the
// rhythm of generation gen. The first row is a single cell on, or the cells of seed (a number: its bits)
func Cellular(rule, width, gen int, seed uint64) []bool {
	if width < 1 {
		return nil
	}
	row := make([]bool, width)
	if seed == 0 {
		row[width/2] = true
	} else {
		for i := range width {
			row[i] = seed>>(uint(i)%64)&1 == 1
		}
	}
	for range max(0, gen) {
		next := make([]bool, width)
		for i := range width {
			l, c, r := row[(i+width-1)%width], row[i], row[(i+1)%width]
			idx := 0
			for _, b := range []bool{l, c, r} {
				idx <<= 1
				if b {
					idx |= 1
				}
			}
			next[i] = rule>>uint(idx)&1 == 1
		}
		row = next
	}
	return row
}

// Steps writes a rhythm as the steps of a pattern ("x . . x .")
func Steps(r []bool) string {
	var b strings.Builder
	for i, on := range r {
		if i > 0 {
			b.WriteByte(' ')
		}
		if on {
			b.WriteByte('x')
		} else {
			b.WriteByte('.')
		}
	}
	return b.String()
}

// Expand replaces the rhythm functions in the text of a pattern by their steps: E(5,16) E(5,16,2) a euclidean rhythm (turned 2), A(3+3+2) an additive rhythm, CA(30,16) CA(30,16,4)
// CA(30,16,4,11) a cellular automaton (rule, cells, the generation, a seed). Text without them is as it was
func Expand(text string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(text); {
		name := ""
		for _, n := range []string{"CA(", "E(", "A("} {
			if strings.HasPrefix(text[i:], n) && (i == 0 || !isLetter(text[i-1])) {
				name = n
				break
			}
		}
		if name == "" {
			out.WriteByte(text[i])
			i++
			continue
		}
		end := strings.IndexByte(text[i:], ')')
		if end < 0 {
			return "", fmt.Errorf("%s without its )", name)
		}
		args := strings.Split(text[i+len(name):i+end], ",")
		r, err := call(strings.TrimSuffix(name, "("), args)
		if err != nil {
			return "", err
		}
		out.WriteString(Steps(r))
		i += end + 1
	}
	return out.String(), nil
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

func call(name string, args []string) ([]bool, error) {
	nums := func(min, max int) ([]int, error) {
		if len(args) < min || len(args) > max {
			return nil, fmt.Errorf("%s(): %d to %d numbers", name, min, max)
		}
		out := make([]int, len(args))
		for i, a := range args {
			n, err := strconv.Atoi(strings.TrimSpace(a))
			if err != nil {
				return nil, fmt.Errorf("%s(%s): %q is not a whole number", name, strings.Join(args, ","), a)
			}
			out[i] = n
		}
		return out, nil
	}
	switch name {
	case "A":
		return Additive(strings.Join(args, ","))
	case "E":
		n, err := nums(2, 3)
		if err != nil {
			return nil, err
		}
		if n[1] < 1 || n[1] > 64 || n[0] < 0 || n[0] > n[1] {
			return nil, fmt.Errorf("E(%d,%d): hits 0 to the steps, steps 1 to 64", n[0], n[1])
		}
		rot := 0
		if len(n) == 3 {
			rot = n[2]
		}
		return Euclid(n[0], n[1], rot), nil
	default: // CA
		n, err := nums(2, 4)
		if err != nil {
			return nil, err
		}
		if n[0] < 0 || n[0] > 255 || n[1] < 1 || n[1] > 64 {
			return nil, fmt.Errorf("CA(%d,%d): a rule 0 to 255, 1 to 64 cells", n[0], n[1])
		}
		gen, seed := n[1]/2, 0
		if len(n) > 2 {
			gen = n[2]
		}
		if len(n) > 3 {
			seed = n[3]
		}
		return Cellular(n[0], n[1], gen, uint64(seed)), nil
	}
}

// Cond is a condition on a hit: it plays only when it holds. Written `2/4` (the second time through in every four), `fill` (in the last bar of a section), `!fill`, `first` (the
// first time through), `!first`; `!2/4` is the other three
type Cond struct {
	Nth, Of int  // Of > 0: the Nth of every Of passes
	Fill    int  // 1 only in a fill, -1 never in one
	First   int  // 1 only on the first pass, -1 never on it
	Not     bool // for Nth/Of: all but that one
}

// ParseCond reads a condition
func ParseCond(s string) (Cond, error) {
	var c Cond
	t := s
	not := strings.HasPrefix(t, "!")
	t = strings.TrimPrefix(t, "!")
	switch t {
	case "fill":
		c.Fill = 1
		if not {
			c.Fill = -1
		}
		return c, nil
	case "first":
		c.First = 1
		if not {
			c.First = -1
		}
		return c, nil
	}
	a, b, ok := strings.Cut(t, "/")
	n, err1 := strconv.Atoi(a)
	of, err2 := strconv.Atoi(b)
	if !ok || err1 != nil || err2 != nil || of < 2 || of > 64 || n < 1 || n > of {
		return c, fmt.Errorf("%q: a condition is 2/4 (the second of every four), !2/4, fill, !fill, first or !first", s)
	}
	return Cond{Nth: n, Of: of, Not: not}, nil
}

// Holds says whether the hit plays on a pass (counting from 0) of the pattern, in a fill or not
func (c Cond) Holds(pass int64, fill bool) bool {
	if c.Of > 0 {
		is := int(((pass%int64(c.Of))+int64(c.Of)))%c.Of == c.Nth-1
		if is == c.Not {
			return false
		}
	}
	if c.Fill == 1 && !fill || c.Fill == -1 && fill {
		return false
	}
	if c.First == 1 && pass != 0 || c.First == -1 && pass == 0 {
		return false
	}
	return true
}

// Zero says there is no condition
func (c Cond) Zero() bool { return c == Cond{} }

// Lane is a value a step, going round on a length of its own (a lane of 5 against a pattern of 16 steps drifts: polymeter). A value of `.` is a gap (the default)
type Lane struct {
	Vals []float64
	Gap  []bool
}

// ParseLane reads the values of a lane: numbers, `.` for the default
func ParseLane(text string) (Lane, error) {
	var l Lane
	for _, f := range strings.Fields(text) {
		if f == "." {
			l.Vals, l.Gap = append(l.Vals, 0), append(l.Gap, true)
			continue
		}
		v, err := strconv.ParseFloat(f, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return l, fmt.Errorf("%q: a number (or . for the default)", f)
		}
		l.Vals, l.Gap = append(l.Vals, v), append(l.Gap, false)
	}
	if len(l.Vals) > 256 {
		return l, fmt.Errorf("a lane of at most 256 steps")
	}
	return l, nil
}

// At is the value of the lane at a step (it goes round); false when the lane has none there
func (l Lane) At(step int64) (float64, bool) {
	n := int64(len(l.Vals))
	if n == 0 {
		return 0, false
	}
	i := ((step % n) + n) % n
	return l.Vals[i], !l.Gap[i]
}

// Groove is the feel of the steps of a bar: how much each is late (in steps, -0.5 to 0.5) and how loud (a factor), going round on its length
type Groove struct {
	Name  string
	Shift []float64
	Vel   []float64
}

// At is the groove at a step
func (g Groove) At(step int64) (shift, vel float64) {
	n := int64(len(g.Shift))
	if n == 0 {
		return 0, 1
	}
	i := ((step % n) + n) % n
	return g.Shift[i], g.Vel[i]
}

// Grooves are the grooves there are, for the steps of a sixteenth division: straight is none
var Grooves = []Groove{
	{Name: "straight"},
	{Name: "mpc 54", Shift: rep16(0, 0.08), Vel: rep16(1.0, 0.92)},
	{Name: "mpc 58", Shift: rep16(0, 0.16), Vel: rep16(1.0, 0.9)},
	{Name: "mpc 62", Shift: rep16(0, 0.24), Vel: rep16(1.0, 0.88)},
	{Name: "shuffle", Shift: rep16(0, 0.33), Vel: rep16(1.0, 0.8)},
	{Name: "laid back", Shift: seq16(0.06, 0.02, 0.1, 0.04), Vel: seq16(1, 0.85, 0.95, 0.85)},
	{Name: "push", Shift: seq16(-0.06, -0.02, -0.04, -0.01), Vel: seq16(1, 0.9, 0.95, 0.9)},
	{Name: "dilla", Shift: seq16(0.12, -0.05, 0.2, 0.0, -0.08, 0.14, 0.04, -0.1), Vel: seq16(1, 0.8, 0.9, 0.75, 0.95, 0.8, 0.85, 0.7)},
	{Name: "backbeat", Shift: seq16(0), Vel: seq16(1, 0.7, 0.85, 0.7, 1.1, 0.7, 0.85, 0.7)},
}

// GrooveNames are the names of the grooves
func GrooveNames() []string {
	out := make([]string, len(Grooves))
	for i, g := range Grooves {
		out[i] = g.Name
	}
	return out
}

func rep16(a, b float64) []float64 { return seq16(a, b) }

// seq16 repeats the values to fill sixteen steps
func seq16(v ...float64) []float64 {
	out := make([]float64, 16)
	for i := range out {
		out[i] = v[i%len(v)]
	}
	return out
}
