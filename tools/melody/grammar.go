package melody

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

// A grammar writes the form of a melody as rules that rewrite themselves, so a form is found, not chosen from a list. The text has a rule a line, `S -> A A' B A`, with
// alternatives (`|`) that can have a weight (`*3`):
//
//	S    -> Open Close
//	Open -> A A' *3 | A B
//	Close -> A~ | B A'
//
// The phrases are the letters A to D (a phrase of its own the first time, the same again after that), with what happens to them after the letter: ' a variation, ~ turned upside
// down in the scale, < backwards, ^ a step higher in the scale, v a step lower (^^ two), ? ending open (on the third or the fifth, a question). Any other word is a rule: the first
// rule is the start (S if there is a rule of that name). It starts at S and rewrites every word of a rule until only phrases are left.
//
// An L-system rewrites in parallel: a line `axiom: A` and `iterate: 3` make the text start as that and every symbol with a rule is rewritten at once, three times
// (`A -> A B A'`, `B -> A`: A becomes A B A', then A B A' A A B A' ... the same pattern at every size), the phrases that are left are the form.
type Token struct {
	Letter byte   // A to D
	Ops    string // ' ~ < ^ v ?
}

type rule struct {
	alts    [][]string
	weights []float64
}

const maxTokens = 256

// ParseForm reads a grammar or an L-system; false when it has a mistake (the message says where)
func ParseForm(text string) error {
	_, err := Expand(text, rand.New(rand.NewPCG(1, 2)), 8)
	return err
}

// Expand writes the phrases of a grammar or L-system, at most n (it needs as many as there are phrases in the melody; a shorter form goes round)
func Expand(text string, rng *rand.Rand, n int) ([]Token, error) {
	rules := map[string]*rule{}
	var order []string
	axiom, iterate := "", 0
	for ln, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "axiom:"):
			axiom = strings.TrimSpace(strings.TrimPrefix(line, "axiom:"))
			continue
		case strings.HasPrefix(line, "iterate:"):
			v, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "iterate:")))
			if err != nil || v < 0 || v > 8 {
				return nil, fmt.Errorf("line %d: iterate: 0 to 8", ln+1)
			}
			iterate = v
			continue
		}
		name, body, ok := strings.Cut(line, "->")
		name = strings.TrimSpace(name)
		if !ok || name == "" || strings.ContainsAny(name, " \t") {
			return nil, fmt.Errorf("line %d: a rule is `Name -> phrases`, like S -> A A' B A", ln+1)
		}
		r := rules[name]
		if r == nil {
			r = &rule{}
			rules[name] = r
			order = append(order, name)
		}
		for _, alt := range strings.Split(body, "|") {
			fs := strings.Fields(alt)
			w := 1.0
			if k := len(fs); k > 0 && strings.HasPrefix(fs[k-1], "*") {
				v, err := strconv.ParseFloat(fs[k-1][1:], 64)
				if err != nil || v <= 0 {
					return nil, fmt.Errorf("line %d: %s: a weight is *2", ln+1, fs[k-1])
				}
				w, fs = v, fs[:k-1]
			}
			if len(fs) == 0 {
				return nil, fmt.Errorf("line %d: an empty choice in %s", ln+1, name)
			}
			r.alts, r.weights = append(r.alts, fs), append(r.weights, w)
		}
	}
	if len(rules) == 0 && axiom == "" {
		return nil, fmt.Errorf("no rules: S -> A A' B A")
	}
	var seq []string
	if axiom != "" || iterate > 0 { // an L-system
		if axiom == "" {
			axiom = order[0]
		}
		seq = strings.Fields(axiom)
		for range iterate {
			var next []string
			for _, sym := range seq {
				if r := rules[sym]; r != nil {
					next = append(next, r.pick(rng)...)
				} else {
					next = append(next, sym)
				}
				if len(next) > maxTokens {
					break
				}
			}
			seq = next
			if len(seq) > maxTokens {
				seq = seq[:maxTokens]
			}
		}
	} else {
		start := order[0]
		if rules["S"] != nil {
			start = "S"
		}
		seq = []string{start}
		for depth := 0; depth < 12; depth++ {
			var next []string
			changed := false
			for _, sym := range seq {
				if r := rules[sym]; r != nil && !isPhrase(sym) {
					next = append(next, r.pick(rng)...)
					changed = true
				} else {
					next = append(next, sym)
				}
			}
			seq = next
			if len(seq) > maxTokens {
				seq = seq[:maxTokens]
			}
			if !changed {
				break
			}
		}
	}
	var out []Token
	for _, sym := range seq {
		if !isPhrase(sym) {
			if rules[sym] != nil {
				return nil, fmt.Errorf("%s never ends: a rule that rewrites itself without a phrase to stop at", sym)
			}
			return nil, fmt.Errorf("%q: a phrase is a letter A to D with ' ~ < ^ v ? after it, any other word needs a rule", sym)
		}
		out = append(out, Token{Letter: sym[0], Ops: sym[1:]})
		if len(out) >= n {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the grammar makes no phrases")
	}
	return out, nil
}

// isPhrase says a word is a phrase: a letter A to D and operators
func isPhrase(s string) bool {
	if s == "" || s[0] < 'A' || s[0] > 'D' {
		return false
	}
	return strings.Trim(s[1:], "'~<^v?") == ""
}

func (r *rule) pick(rng *rand.Rand) []string {
	total := 0.0
	for _, w := range r.weights {
		total += w
	}
	x := rng.Float64() * total
	for i, w := range r.weights {
		if x < w {
			return r.alts[i]
		}
		x -= w
	}
	return r.alts[len(r.alts)-1]
}
