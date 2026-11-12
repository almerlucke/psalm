package shapechain

import (
	"math"
	"strings"

	"github.com/almerlucke/psalm/blocks/fft"
)

// Splitting a chain into its stages and looking at it: what an editor draws.

// SplitStages is the text of a chain cut at its top level pipes (a pipe inside par( ) belongs to the stage), comments left out and
// the stages trimmed; an empty text gives none. Joined with " | " it is the chain again
func SplitStages(text string) []string {
	var out []string
	var cur strings.Builder
	depth := 0
	inComment := false
	flush := func() {
		if s := strings.Join(strings.Fields(cur.String()), " "); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for _, r := range text {
		switch {
		case inComment:
			if r == '\n' {
				inComment = false
				cur.WriteRune(' ')
			}
		case r == '#':
			inComment = true
		case r == '(':
			depth++
			cur.WriteRune(r)
		case r == ')':
			depth = max(0, depth-1)
			cur.WriteRune(r)
		case r == '|' && depth == 0:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// Wave is one cycle of the chain at n points (the phase from 0 to just under 1), macros as given
func (c *Chain) Wave(n int, m Macros) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = clean(c.Eval(float64(i)/float64(n), &m))
	}
	return out
}

// Curve is the chain as a function of an input from lo to hi, at n points (the transfer curve of a shaping stage)
func (c *Chain) Curve(n int, lo, hi float64, m Macros) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = clean(c.Eval(lo+(hi-lo)*float64(i)/float64(n-1), &m))
	}
	return out
}

func clean(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// Spectrum is the levels of the first k harmonics of a wave of a power of two points, each 0 to 1 against the loudest (the wave's mean is not one)
func Spectrum(wave []float64, k int) []float64 {
	n := len(wave)
	if n < 8 || n&(n-1) != 0 {
		return make([]float64, k)
	}
	f := fft.NewReal(n)
	re, im := make([]float64, n/2+1), make([]float64, n/2+1)
	f.Forward(wave, re, im)
	out := make([]float64, k)
	top := 0.0
	for h := 1; h <= k && h <= n/2; h++ {
		out[h-1] = math.Hypot(re[h], im[h])
		top = max(top, out[h-1])
	}
	if top > 0 {
		for i := range out {
			out[i] /= top
		}
	}
	return out
}

// Param is a number of a stage, as an editor shows it: its name, what it is when left out, and the range of its knob
type Param struct {
	Name string  `json:"name"`
	Def  float64 `json:"def"`
	Lo   float64 `json:"lo"`
	Hi   float64 `json:"hi"`
}

// StageInfo describes a stage for an editor. Kind is "" (numbers: Params), "list" (a list of numbers or harmonics written as text), "word" (a word, then
// numbers: table) or "par" (chains in parentheses)
type StageInfo struct {
	Name   string  `json:"name"`
	Group  string  `json:"group"`
	Help   string  `json:"help"`
	Kind   string  `json:"kind,omitempty"`
	Params []Param `json:"params,omitempty"`
	// Example is the text of a stage to add (for a list or a word: what to start with)
	Example string `json:"example,omitempty"`
}

func p(name string, def, lo, hi float64) Param { return Param{name, def, lo, hi} }

// Catalog is every stage, in the order an editor lists them
var Catalog = []StageInfo{
	{Name: "sin", Group: "Sources", Help: "a sine of the phase"},
	{Name: "bipolar", Group: "Sources", Help: "the phase as a saw, -1 to 1"},
	{Name: "unipolar", Group: "Sources", Help: "-1..1 made 0..1"},
	{Name: "tri", Group: "Sources", Help: "a triangle over 0..1"},
	{Name: "pulse", Group: "Sources", Help: "1 below the width, else 0", Params: []Param{p("width", 0.5, 0.01, 0.99)}},
	{Name: "table", Group: "Sources", Help: "a wave read over 0..1: sine, cosine, sawtooth, triangle, square, or points", Kind: "word", Example: "table sine 256"},
	{Name: "czsaw", Group: "Phase distortion", Help: "Casio CZ saw: the phase bent, then a cosine", Params: []Param{p("amount", 0.5, 0, 1)}},
	{Name: "czsquare", Group: "Phase distortion", Help: "Casio CZ square", Params: []Param{p("amount", 0.5, 0, 1), p("width", 0.5, 0.05, 0.95)}},
	{Name: "czpulse", Group: "Phase distortion", Help: "Casio CZ pulse", Params: []Param{p("amount", 0.5, 0, 1)}},
	{Name: "czres", Group: "Phase distortion", Help: "Casio CZ resonance: a falling saw times a higher cosine", Params: []Param{p("amount", 0.5, 0, 1), p("harmonic", 4, 1, 16)}},

	{Name: "add", Group: "Math", Help: "x + a", Params: []Param{p("a", 0, -1, 1)}},
	{Name: "mult", Group: "Math", Help: "x * m (mult 3 | sin is three cycles)", Params: []Param{p("m", 1, 0, 8)}},
	{Name: "mod", Group: "Math", Help: "the remainder of x / m", Params: []Param{p("m", 1, 0.1, 4)}},
	{Name: "ripple", Group: "Math", Help: "x plus its remainder of x / m", Params: []Param{p("m", 1, 0.1, 4)}},
	{Name: "linear", Group: "Math", Help: "x * scale + shift", Params: []Param{p("scale", 1, -4, 4), p("shift", 0, -2, 2)}},
	{Name: "abs", Group: "Math", Help: "the absolute value"},
	{Name: "pow", Group: "Math", Help: "|x|^p with the sign of x", Params: []Param{p("p", 1, 0.2, 5)}},

	{Name: "fold", Group: "Shapers", Help: "a sine wave folder: each 1 of gain above 1 is a fold", Params: []Param{p("gain", 1, 0, 8)}},
	{Name: "trifold", Group: "Shapers", Help: "reflected at -1 and 1", Params: []Param{p("gain", 1, 0, 8)}},
	{Name: "tube", Group: "Shapers", Help: "an asymmetric tanh: the bias adds even harmonics", Params: []Param{p("drive", 2, 0.2, 10), p("bias", 0.3, -0.95, 0.95)}},
	{Name: "softclip", Group: "Shapers", Help: "an arctangent curve", Params: []Param{p("amount", 2, 0.2, 12)}},
	{Name: "tanh", Group: "Shapers", Help: "tanh(d x) / tanh(d)", Params: []Param{p("drive", 1, 0.2, 10)}},
	{Name: "clip", Group: "Shapers", Help: "cut off at the edges", Params: []Param{p("lo", -1, -2, 0), p("hi", 1, 0, 2)}},
	{Name: "mirror", Group: "Shapers", Help: "folded back into the range, as a ball bounces", Params: []Param{p("lo", 0, -2, 1), p("hi", 1, -1, 2)}},
	{Name: "curve", Group: "Shapers", Help: "a power curve, sign kept", Params: []Param{p("tension", 0, -0.95, 0.95)}},
	{Name: "ucurve", Group: "Shapers", Help: "a power curve over 0..1", Params: []Param{p("tension", 0, -0.95, 0.95)}},
	{Name: "smoothstep", Group: "Shapers", Help: "an S between two edges", Params: []Param{p("edge0", 0, -1, 1), p("edge1", 1, -1, 2)}},
	{Name: "smootherstep", Group: "Shapers", Help: "a smoother S between two edges", Params: []Param{p("edge0", 0, -1, 1), p("edge1", 1, -1, 2)}},
	{Name: "deadzone", Group: "Shapers", Help: "0 while |x| is under the width", Params: []Param{p("width", 0.2, 0, 1)}},
	{Name: "softdead", Group: "Shapers", Help: "a dead zone without a jump", Params: []Param{p("width", 0.2, 0, 1)}},
	{Name: "halfwave", Group: "Shapers", Help: "the negative half cut off"},
	{Name: "diode", Group: "Shapers", Help: "nothing under the threshold, a soft knee", Params: []Param{p("threshold", 0, 0, 1), p("knee", 0.1, 0, 1)}},
	{Name: "quantize", Group: "Shapers", Help: "the nearest multiple of q", Params: []Param{p("q", 0.25, 0.02, 1)}},
	{Name: "funnel", Group: "Shapers", Help: "snaps to the nearest of the values", Kind: "list", Example: "funnel -1 -0.5 0 0.5 1"},

	{Name: "cheby", Group: "Harmonics", Help: "Chebyshev harmonics (n:level)", Kind: "list", Example: "cheby 1:1 3:0.3"},
	{Name: "harm", Group: "Harmonics", Help: "a sum of sines of x (n:level)", Kind: "list", Example: "harm 1:1 3:0.4"},
	{Name: "poly", Group: "Harmonics", Help: "c0 + c1 x + c2 x^2...", Kind: "list", Example: "poly 0 1.5 0 -0.5"},
	{Name: "legendre", Group: "Harmonics", Help: "a Legendre polynomial of order n", Params: []Param{p("order", 3, 0, 20)}},

	{Name: "hann", Group: "Windows", Help: "a raised cosine bump over 0..1"},
	{Name: "hamming", Group: "Windows", Help: "a Hamming window"},
	{Name: "blackman", Group: "Windows", Help: "a Blackman window"},
	{Name: "welch", Group: "Windows", Help: "a parabola bump"},
	{Name: "tukey", Group: "Windows", Help: "flat middle, cosine ends", Params: []Param{p("alpha", 0.5, 0, 1)}},
	{Name: "gauss", Group: "Windows", Help: "a Gaussian bump", Params: []Param{p("sigma", 0.4, 0.05, 1)}},
	{Name: "kaiser", Group: "Windows", Help: "a Kaiser window", Params: []Param{p("beta", 8, 0, 20)}},

	{Name: "par", Group: "Combine", Help: "chains side by side on x, their results added", Kind: "par", Example: "par(sin, mult 3 | sin | mult 0.4)"},
}
