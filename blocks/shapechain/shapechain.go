// Package shapechain is a small language for a chain of shapers (genny's float/shape): a phase
// from 0 to 1 goes in, a wave comes out. A chain is a pipeline of stages joined by |, each stage a
// name and its numbers, and the result is a function of x that a synth reads for every sample:
//
//	sin                          a sine
//	bipolar                      a saw (the phase, 0..1, made -1..1)
//	mult 3 | sin                 a sine of three cycles
//	bipolar | mirror -0.5 0.5    a saw folded back at -0.5 and 0.5
//	mult m1[1,5] | sin           m1 (0..1, moved by an envelope, an LFO...) sets the cycles, 1 to 5
//	par(sin, mult 3 | sin)       the stages in parentheses side by side, their results added
//
// A stage is one of (the numbers in brackets are what it takes when they are left out; they are
// genny's shapers of the same name unless said):
//
//	thru                 x
//	add a [0]            x + a
//	mult m [1]           x * m
//	mod m [1]            the remainder of x / m
//	ripple m [1]         x + the remainder of x / m
//	linear scale shift   x * scale + shift
//	bipolar, unipolar    0..1 to -1..1, -1..1 to 0..1
//	abs, sin, tri        |x|; sin(2πx); a triangle over 0..1
//	mirror lo hi [0 1]   folded back into the range, as a ball bounces
//	pulse w [0.5]        1 below w, else 0
//	quantize q [1]       the nearest multiple of q
//	clip lo hi [-1 1]    cut off at the edges
//	softclip a [2]       an arctangent curve (a above 0, the higher the harder)
//	cheby 1:1 3:0.3      Chebyshev harmonics (n:level, n from 1 to 64): a waveshaper
//	table sine [256]     a wave read over 0..1: sine, cosine, sawtooth, triangle, square (also
//	                     "uni sine"... written uni-sine) or table 0 1 0.5 -1 for your own points
//	pow p [1]            |x|^p with the sign of x           (not in genny)
//	tanh d [1]           tanh(d * x) / tanh(d)              (not in genny)
//	par(chain, chain)    the chains side by side on x, their results added
//
// Where a stage takes a number it can take a macro instead: m1 to m4, a value from 0 to 1 that the
// synth moves while it plays. m1 is 0..1, m1[lo,hi] maps 0..1 to lo..hi (lo can be above hi).
// Spaces, new lines and commas between numbers are the same; # starts a comment to the end of
// the line.
package shapechain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/almerlucke/psalm/blocks/fft"

	"github.com/almerlucke/genny/float/shape"
	"github.com/almerlucke/genny/float/shape/shapers/lookup"
)

// NumMacros is how many macros a chain can use
const NumMacros = 4

// Macros are the values of the macros m1 to m4 (0 to 1)
type Macros [NumMacros]float64

// Chain is a parsed chain. It holds no state: any number of voices may share one
type Chain struct {
	text   string
	stages []stage
	macros [NumMacros]bool

	bwOnce sync.Once
	bw     int
}

// stage is one step of a chain
type stage struct {
	name string
	f    func(x float64, m *Macros) float64
}

// Text is the chain as it was written
func (c *Chain) Text() string { return c.text }

// Uses tells if the chain reads a macro (1 to 4)
func (c *Chain) Uses(macro int) bool { return macro >= 1 && macro <= NumMacros && c.macros[macro-1] }

// Eval is x through the chain, the macros as they are now (nil: all 0.5)
func (c *Chain) Eval(x float64, m *Macros) float64 {
	if m == nil {
		m = &Macros{0.5, 0.5, 0.5, 0.5}
	}
	for i := range c.stages {
		x = c.stages[i].f(x, m)
	}
	return x
}

// Shape is x through the chain with the macros at 0.5: a genny shaper
func (c *Chain) Shape(x float64) float64 { return c.Eval(x, nil) }

// WithMacros is a genny shaper of the chain with the macros as given
func (c *Chain) WithMacros(m Macros) shape.Shaper { return fixed{c: c, m: m} }

type fixed struct {
	c *Chain
	m Macros
}

func (f fixed) Shape(x float64) float64 { return f.c.Eval(x, &f.m) }

var (
	bandwidths     sync.Map // chain text -> its bandwidth
	bandwidthsKept atomic.Int64
)

// Wide is what Bandwidth gives for a chain with a sharp edge (a saw, a pulse, steps): harmonics
// without an end
const Wide = 1 << 20

// Bandwidth is the highest harmonic of the wave the chain makes that has a say: above it the harmonics
// together are 60 dB down on the wave (a sine is 1, a few partials the highest of them, a fold
// some tens), or Wide when they go on past what is looked at (512). The wave is looked at with
// each macro it uses at 0, half and 1 (all together). A synth reads from it how often a sample
// the chain must be read to keep what lies above the audible band from folding back
func (c *Chain) Bandwidth() int {
	c.bwOnce.Do(func() {
		// a chain of the same text has been looked at already (a script sets the same chains again and
		// again): its answer is kept, so the audio thread does the analysis once for a text
		key := c.text
		if v, ok := bandwidths.Load(key); ok {
			c.bw = v.(int)
			return
		}
		defer func() {
			if bandwidthsKept.Add(1) < 4096 {
				bandwidths.Store(key, c.bw)
			}
		}()
		const n = 1024
		f := fft.NewReal(n)
		x, re, im := make([]float64, n), make([]float64, n/2+1), make([]float64, n/2+1)
		grid := []float64{0.5}
		for _, used := range c.macros {
			if used {
				grid = []float64{0, 0.5, 1}
				break
			}
		}
		for _, v := range grid {
			m := Macros{v, v, v, v}
			for i := range x {
				x[i] = c.Eval(float64(i)/n, &m)
				if math.IsNaN(x[i]) || math.IsInf(x[i], 0) {
					x[i] = 0
				}
			}
			f.Forward(x, re, im)
			power := make([]float64, n/2+1)
			total := 0.0
			for k := 1; k <= n/2; k++ { // the harmonics (not the level)
				power[k] = re[k]*re[k] + im[k]*im[k]
				total += power[k]
			}
			if total < 1e-20 {
				continue
			}
			tail := 0.0
			h := n / 2
			for ; h > 1; h-- { // down from the top while what is above is below -60 dB
				if tail += power[h]; tail > total*1e-6 {
					break
				}
			}
			bw := h
			if h >= n/4 { // still something at a quarter of the sample rate of the look: it goes on
				bw = Wide
			}
			c.bw = max(c.bw, bw)
		}
		c.bw = max(c.bw, 1)
	})
	return c.bw
}

// --- reading ---

type token struct {
	s   string
	col int // 1-based
}

func tokenize(text string) []token {
	var out []token
	rs := []rune(text)
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '#':
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
		case unicode.IsSpace(r):
			i++
		case strings.ContainsRune("|(),[]", r):
			out = append(out, token{string(r), i + 1})
			i++
		default:
			j := i
			for j < len(rs) && !unicode.IsSpace(rs[j]) && !strings.ContainsRune("|(),[]#", rs[j]) {
				j++
			}
			out = append(out, token{string(rs[i:j]), i + 1})
			i = j
		}
	}
	return out
}

type parser struct {
	toks   []token
	i      int
	macros [NumMacros]bool
	depth  int
}

// Error is a chain that can't be read: where, and why
type Error struct {
	Col int
	Msg string
}

func (e *Error) Error() string {
	if e.Col == 0 {
		return e.Msg
	}
	return fmt.Sprintf("column %d: %s", e.Col, e.Msg)
}

func (p *parser) fail(t token, format string, args ...any) error {
	return &Error{Col: t.col, Msg: fmt.Sprintf(format, args...)}
}

func (p *parser) peek() (token, bool) {
	if p.i < len(p.toks) {
		return p.toks[p.i], true
	}
	return token{}, false
}

// Parse reads a chain
func Parse(text string) (*Chain, error) {
	p := &parser{toks: tokenize(text)}
	stages, err := p.chain()
	if err != nil {
		return nil, err
	}
	if t, ok := p.peek(); ok {
		return nil, p.fail(t, "%q: a stage or | was expected", t.s)
	}
	return &Chain{text: strings.TrimSpace(text), stages: stages, macros: p.macros}, nil
}

// MustParse reads a chain that is known to be right
func MustParse(text string) *Chain {
	c, err := Parse(text)
	if err != nil {
		panic(fmt.Sprintf("shapechain %q: %v", text, err))
	}
	return c
}

// chain reads stages joined by |, up to the end or a ) or ,
func (p *parser) chain() ([]stage, error) {
	var stages []stage
	for {
		s, err := p.stage()
		if err != nil {
			return nil, err
		}
		stages = append(stages, s)
		if t, ok := p.peek(); ok && t.s == "|" {
			p.i++
			continue
		}
		return stages, nil
	}
}

// arg is a number of a stage: a constant, or a macro mapped to a range
type arg struct {
	v      float64
	macro  int // 1 to 4, 0 a constant
	lo, hi float64
}

func (a arg) val(m *Macros) float64 {
	if a.macro == 0 {
		return a.v
	}
	return a.lo + (a.hi-a.lo)*m[a.macro-1]
}

// number reads a constant or a macro; ok false when the next token is neither
func (p *parser) number() (arg, bool, error) {
	t, ok := p.peek()
	if !ok {
		return arg{}, false, nil
	}
	if n := macroNumber(t.s); n > 0 {
		p.i++
		p.macros[n-1] = true
		a := arg{macro: n, lo: 0, hi: 1}
		if b, ok := p.peek(); ok && b.s == "[" {
			p.i++
			lo, err := p.constant("the low end of the range")
			if err != nil {
				return a, false, err
			}
			if c, ok := p.peek(); ok && c.s == "," {
				p.i++
			}
			hi, err := p.constant("the high end of the range")
			if err != nil {
				return a, false, err
			}
			if c, ok := p.peek(); !ok || c.s != "]" {
				return a, false, p.fail(t, "%s[lo,hi]: a ] is missing", t.s)
			}
			p.i++
			a.lo, a.hi = lo, hi
		}
		return a, true, nil
	}
	v, err := strconv.ParseFloat(t.s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return arg{}, false, nil
	}
	p.i++
	return arg{v: v}, true, nil
}

func (p *parser) constant(what string) (float64, error) {
	t, ok := p.peek()
	if !ok {
		return 0, &Error{Msg: what + " is missing"}
	}
	v, err := strconv.ParseFloat(t.s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, p.fail(t, "%q: %s is a number", t.s, what)
	}
	p.i++
	return v, nil
}

// skipComma steps over a comma between two numbers; a comma before anything else belongs to a par(
func (p *parser) skipComma(follows func(token) bool) {
	if t, ok := p.peek(); ok && t.s == "," && p.i+1 < len(p.toks) && follows(p.toks[p.i+1]) {
		p.i++
	}
}

func isNumber(t token) bool {
	if macroNumber(t.s) > 0 {
		return true
	}
	v, err := strconv.ParseFloat(t.s, 64)
	return err == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}

// macroNumber is 1 to 4 for m1 to m4, else 0
func macroNumber(s string) int {
	if len(s) == 2 && s[0] == 'm' && s[1] >= '1' && s[1] <= '0'+NumMacros {
		return int(s[1] - '0')
	}
	return 0
}

// numbers reads the numbers after a stage's name: at least min, at most len(defs); the ones left
// out are the defaults
func (p *parser) numbers(name token, defs ...float64) ([]arg, error) {
	var out []arg
	for len(out) < len(defs) {
		a, ok, err := p.number()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		out = append(out, a)
		p.skipComma(isNumber)
	}
	for i := len(out); i < len(defs); i++ {
		out = append(out, arg{v: defs[i]})
	}
	return out, nil
}

func (p *parser) stage() (stage, error) {
	t, ok := p.peek()
	if !ok {
		return stage{}, &Error{Msg: "a stage was expected (sin, mult 3...)"}
	}
	name := t.s
	p.i++
	if name == "par" {
		return p.par(t)
	}
	if name == "cheby" {
		return p.cheby(t)
	}
	if name == "table" {
		return p.table(t)
	}
	one := func(def float64) (arg, error) {
		a, err := p.numbers(t, def)
		if err != nil {
			return arg{}, err
		}
		return a[0], nil
	}
	nonzero := func(a arg) error {
		if a.macro == 0 && a.v == 0 {
			return p.fail(t, "%s: not 0", name)
		}
		return nil
	}
	mk := func(f func(x float64, m *Macros) float64) (stage, error) { return stage{name: name, f: f}, nil }
	switch name {
	case "thru":
		return mk(func(x float64, _ *Macros) float64 { return x })
	case "add":
		a, err := one(0)
		if err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 { return x + a.val(m) })
	case "mult":
		a, err := one(1)
		if err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 { return x * a.val(m) })
	case "mod", "ripple":
		a, err := one(1)
		if err != nil {
			return stage{}, err
		}
		if err := nonzero(a); err != nil {
			return stage{}, err
		}
		if name == "mod" {
			return mk(func(x float64, m *Macros) float64 {
				d := a.val(m)
				if d == 0 {
					return x
				}
				return math.Mod(x, d)
			})
		}
		return mk(func(x float64, m *Macros) float64 {
			d := a.val(m)
			if d == 0 {
				return x
			}
			return x + math.Mod(x, d)
		})
	case "linear":
		a, err := p.numbers(t, 1, 0)
		if err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 { return x*a[0].val(m) + a[1].val(m) })
	case "bipolar":
		return mk(func(x float64, _ *Macros) float64 { return x*2 - 1 })
	case "unipolar":
		return mk(func(x float64, _ *Macros) float64 { return x*0.5 + 0.5 })
	case "abs":
		return mk(func(x float64, _ *Macros) float64 { return math.Abs(x) })
	case "sin":
		return mk(func(x float64, _ *Macros) float64 { return sinTurns(x) })
	case "tri":
		return mk(func(x float64, _ *Macros) float64 {
			if x < 0.5 {
				return 2 * x
			}
			return 2 - 2*x
		})
	case "mirror":
		a, err := p.numbers(t, 0, 1)
		if err != nil {
			return stage{}, err
		}
		if a[0].macro == 0 && a[1].macro == 0 && !(a[0].v < a[1].v) {
			return stage{}, p.fail(t, "mirror: the bottom is below the top")
		}
		return mk(func(x float64, m *Macros) float64 { return foldBack(x, a[0].val(m), a[1].val(m)) })
	case "pulse":
		a, err := one(0.5)
		if err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 {
			if x < a.val(m) {
				return 1
			}
			return 0
		})
	case "quantize":
		a, err := one(1)
		if err != nil {
			return stage{}, err
		}
		if err := nonzero(a); err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 {
			q := a.val(m)
			if q == 0 {
				return x
			}
			return math.Round(x/q) * q
		})
	case "clip":
		a, err := p.numbers(t, -1, 1)
		if err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 {
			lo, hi := a[0].val(m), a[1].val(m)
			if lo > hi {
				lo, hi = hi, lo
			}
			return max(lo, min(x, hi))
		})
	case "softclip":
		a, err := one(2)
		if err != nil {
			return stage{}, err
		}
		if err := nonzero(a); err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 {
			k := a.val(m)
			if k == 0 {
				return x
			}
			return math.Atan(k*x) / math.Atan(k)
		})
	case "pow":
		a, err := one(1)
		if err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 { return math.Copysign(math.Pow(math.Abs(x), a.val(m)), x) })
	case "tanh":
		a, err := one(1)
		if err != nil {
			return stage{}, err
		}
		if err := nonzero(a); err != nil {
			return stage{}, err
		}
		return mk(func(x float64, m *Macros) float64 {
			d := a.val(m)
			if d == 0 {
				return x
			}
			return math.Tanh(d*x) / math.Tanh(d)
		})
	}
	if st, ok, err := p.moreStage(t, name, mk); ok {
		return st, err
	}
	return stage{}, p.fail(t, "%q is not a stage (%s)", name, strings.Join(append(append([]string{}, StageNames...), MoreStageNames...), ", "))
}

// SinTurns is sin(2πx) from a table: see sinTurns
func SinTurns(x float64) float64 { return sinTurns(x) }

// sinTurns is sin(2πx), read from a table of 4096 points with a straight line between them: within
// 3e-7 of math.Sin (130 dB down), several times faster, and the most used stage of a chain
func sinTurns(x float64) float64 {
	f := (x - math.Floor(x)) * sineSize
	i := int(f)
	a := sineTable[i]
	return a + (sineTable[i+1]-a)*(f-float64(i))
}

const sineSize = 4096

var sineTable = func() (t [sineSize + 2]float64) {
	for i := range t {
		t[i] = math.Sin(2 * math.Pi * float64(i) / sineSize)
	}
	return
}()

// foldBack folds x back into [bottom, top] as genny's mirror does, without looping a long time on
// a far value
func foldBack(x, bottom, top float64) float64 {
	if bottom > top {
		bottom, top = top, bottom
	}
	span := top - bottom
	if span <= 0 {
		return bottom
	}
	period := 2 * span
	y := x - bottom
	y -= period * math.Floor(y/period) // math.Mod is slow, and this is a fold per sample
	if y > span {
		y = period - y
	}
	return bottom + y
}

// StageNames are the stages, for messages
var StageNames = []string{"thru", "add", "mult", "mod", "ripple", "linear", "bipolar", "unipolar", "abs", "sin", "tri",
	"mirror", "pulse", "quantize", "clip", "softclip", "cheby", "table", "pow", "tanh", "par"}

// par reads par(chain, chain, ...)
func (p *parser) par(t token) (stage, error) {
	if b, ok := p.peek(); !ok || b.s != "(" {
		return stage{}, p.fail(t, "par(chain, chain): the chains go in parentheses")
	}
	p.i++
	if p.depth++; p.depth > 8 {
		return stage{}, p.fail(t, "par: chains in chains up to 8 deep")
	}
	var chains [][]stage
	for {
		st, err := p.chain()
		if err != nil {
			return stage{}, err
		}
		chains = append(chains, st)
		b, ok := p.peek()
		if !ok {
			return stage{}, p.fail(t, "par(: a ) is missing")
		}
		p.i++
		if b.s == ")" {
			break
		}
		if b.s != "," {
			return stage{}, p.fail(b, "%q: , or ) was expected", b.s)
		}
	}
	p.depth--
	return stage{name: "par", f: func(x float64, m *Macros) float64 {
		sum := 0.0
		for _, c := range chains {
			y := x
			for i := range c {
				y = c[i].f(y, m)
			}
			sum += y
		}
		return sum
	}}, nil
}

// cheby reads the harmonics: 1:1 3:0.3
func (p *parser) cheby(t token) (stage, error) {
	levels := map[int]float64{}
	max := 0
	for {
		b, ok := p.peek()
		if !ok {
			break
		}
		k, v, found := strings.Cut(b.s, ":")
		if !found {
			break
		}
		n, err := strconv.Atoi(k)
		lv, err2 := strconv.ParseFloat(v, 64)
		if err != nil || err2 != nil || n < 1 || n > 64 || math.IsNaN(lv) || math.IsInf(lv, 0) {
			return stage{}, p.fail(b, "%q: a harmonic from 1 to 64 and its level (3:0.3)", b.s)
		}
		p.i++
		levels[n] = lv
		if n > max {
			max = n
		}
		p.skipComma(func(t token) bool { return strings.Contains(t.s, ":") })
	}
	if len(levels) == 0 {
		return stage{}, p.fail(t, "cheby: the harmonics and their levels (cheby 1:1 3:0.3)")
	}
	coef := make([]float64, max+1)
	for n, v := range levels {
		coef[n] = v
	}
	return stage{name: "cheby", f: func(x float64, _ *Macros) float64 {
		t0, t1, mix := 1.0, x, 0.0
		for h := 1; h <= max; h++ {
			mix += coef[h] * t1
			t0, t1 = t1, 2*x*t1-t0
		}
		return mix
	}}, nil
}

var waves = map[string]func(int) lookup.Table{
	"sine": lookup.NewSineTable, "uni-sine": lookup.NewUniSineTable,
	"cosine": lookup.NewCosineTable, "uni-cosine": lookup.NewUniCosineTable,
	"sawtooth": lookup.NewSawtoothTable, "uni-sawtooth": lookup.NewUniSawtoothTable,
	"triangle": lookup.NewTriangleTable, "uni-triangle": lookup.NewUniTriangleTable,
	"square": lookup.NewSquareTable, "uni-square": lookup.NewUniSquareTable,
}

// table reads table <wave> [n], or table v1 v2 ...
func (p *parser) table(t token) (stage, error) {
	b, ok := p.peek()
	if !ok {
		return stage{}, p.fail(t, "table: a wave (sine...) or points (table 0 1 0.5)")
	}
	var tab lookup.Table
	periodic := false
	if mk, isWave := waves[b.s]; isWave {
		p.i++
		n := 256
		if c, ok := p.peek(); ok {
			if v, err := strconv.Atoi(c.s); err == nil {
				if v < 2 || v > 65536 {
					return stage{}, p.fail(c, "table: 2 to 65536 points")
				}
				n = v
				p.i++
			}
		}
		tab, periodic = mk(n), true
	} else {
		var vals []float64
		for {
			c, ok := p.peek()
			if !ok {
				break
			}
			v, err := strconv.ParseFloat(c.s, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				break
			}
			p.i++
			vals = append(vals, v)
			p.skipComma(isNumber)
		}
		if len(vals) < 2 {
			return stage{}, p.fail(b, "table: %q is not a wave (sine, cosine, sawtooth, triangle, square, uni-...), and points need 2 or more", b.s)
		}
		tab = lookup.Table(vals)
	}
	last := len(tab) - 1
	return stage{name: "table", f: func(x float64, _ *Macros) float64 {
		if !periodic && x >= 1 {
			return tab[last]
		}
		return tab.Shape(max(0, min(x, math.Nextafter(1, 0))))
	}}, nil
}
