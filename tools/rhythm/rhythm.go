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

// Markov learns a rhythm from examples and makes another like it: the chance of a hit or a rest after the last two steps of the examples (steps written x and ., other characters
// left out), for n steps from a seed. The examples are what it has seen; with fewer than three steps it has nothing to learn from
func Markov(example string, n int, seed uint64) ([]bool, error) {
	var ex []bool
	for _, r := range example {
		switch r {
		case 'x', 'X', 'o', 'O', '1':
			ex = append(ex, true)
		case '.', '-', '_', '0':
			ex = append(ex, false)
		}
	}
	if len(ex) < 3 || n < 1 || n > 128 {
		return nil, fmt.Errorf("M(x . x x . x . .;16;1): at least 3 steps to learn from, 1 to 128 to make")
	}
	// counts[a][b][next]: after the steps a, b (the example goes round)
	var counts [2][2][2]float64
	for i := range ex {
		a, b, c := ex[i], ex[(i+1)%len(ex)], ex[(i+2)%len(ex)]
		counts[b2i(a)][b2i(b)][b2i(c)]++
	}
	state := seed*2654435761 + 12345
	next := func() float64 { // a splitmix step
		state += 0x9e3779b97f4a7c15
		z := state
		z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
		z = (z ^ z>>27) * 0x94d049bb133111eb
		return float64((z^z>>31)>>11) / (1 << 53)
	}
	a, b := ex[0], ex[1]
	out := make([]bool, 0, n)
	out = append(out, a)
	if n > 1 {
		out = append(out, b)
	}
	for len(out) < n {
		c := counts[b2i(a)][b2i(b)]
		on := false
		if t := c[0] + c[1]; t > 0 {
			on = next() < c[1]/t
		} else {
			on = next() < 0.5
		}
		out = append(out, on)
		a, b = b, on
	}
	return out[:n], nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
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
// CA(30,16,4,11) a cellular automaton (rule, cells, the generation, a seed), M(x . x x . x . .;16;3) a rhythm like the example (a Markov chain on its steps; the steps to make, a seed). Text without them is as it was
func Expand(text string) (string, error) { return ExpandPass(text, 0) }

// IsLive says the text has a function that changes from one pass through the pattern to the next: an argument `+1` (a rotation or a generation that goes on by that much every
// pass: E(5,16,+1), CA(30,16,+1)) or `/4` on the seed of a Markov rhythm (a new one every 4 passes: M(x . x x;16;3/4))
func IsLive(text string) bool {
	for i := 0; i < len(text); i++ {
		for _, n := range []string{"CA(", "E(", "M("} {
			if strings.HasPrefix(text[i:], n) && (i == 0 || !isLetter(text[i-1])) {
				if end := strings.IndexByte(text[i:], ')'); end > 0 && strings.ContainsAny(text[i:i+end], "+/") {
					return true
				}
			}
		}
	}
	return false
}

// ExpandPass is Expand for a pass through the pattern (counting from 0): the functions that go on every pass give another rhythm each time
func ExpandPass(text string, pass int64) (string, error) {
	var out strings.Builder
	for i := 0; i < len(text); {
		name := ""
		for _, n := range []string{"CA(", "E(", "A(", "M("} {
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
		r, err := call(strings.TrimSuffix(name, "("), args, pass)
		if err != nil {
			return "", err
		}
		out.WriteString(Steps(r))
		i += end + 1
	}
	return out.String(), nil
}

// intArg reads a whole number that may go on with the passes: 3, +1 (one more every pass, from 0), 2+1 (from 2), 3/4 (3, one more every 4 passes)
func intArg(s string, pass int64) (int, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "+"); i >= 0 {
		a, b := s[:i], s[i+1:]
		base, step := 0, 1
		var err error
		if a != "" {
			if base, err = strconv.Atoi(a); err != nil {
				return 0, err
			}
		}
		if b != "" {
			if step, err = strconv.Atoi(b); err != nil {
				return 0, err
			}
		}
		return base + step*int(pass), nil
	}
	if a, b, ok := strings.Cut(s, "/"); ok {
		base, err := strconv.Atoi(a)
		every, err2 := strconv.Atoi(b)
		if err != nil || err2 != nil || every < 1 {
			return 0, fmt.Errorf("not a number")
		}
		return base + int(pass)/every, nil
	}
	return strconv.Atoi(s)
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

func call(name string, args []string, pass int64) ([]bool, error) {
	nums := func(min, max int) ([]int, error) {
		if len(args) < min || len(args) > max {
			return nil, fmt.Errorf("%s(): %d to %d numbers", name, min, max)
		}
		out := make([]int, len(args))
		for i, a := range args {
			n, err := intArg(a, pass)
			if err != nil {
				return nil, fmt.Errorf("%s(%s): %q is not a whole number (or 2+1: 2 and one more every pass)", name, strings.Join(args, ","), a)
			}
			out[i] = n
		}
		return out, nil
	}
	switch name {
	case "M": // the arguments are split by commas above: M(x . x x;16;1) has semicolons, so it is one argument
		parts := strings.Split(strings.Join(args, ","), ";")
		n, seed := 16, 1
		if len(parts) > 1 {
			v, err := intArg(parts[1], 0)
			if err != nil {
				return nil, fmt.Errorf("M(): %q is not a number of steps", parts[1])
			}
			n = v
		}
		if len(parts) > 2 {
			v, err := intArg(parts[2], pass)
			if err != nil {
				return nil, fmt.Errorf("M(): %q is not a seed", parts[2])
			}
			seed = v
		}
		return Markov(parts[0], n, uint64(seed))
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
	Name  string    `json:"name"`
	Shift []float64 `json:"shift"`
	Vel   []float64 `json:"vel"`
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
