package dx

import "fmt"

// NumOperators is the number of operators of a DX voice
const NumOperators = 6

// Algorithm is an operator routing of the 6 operator engine, following the classic 32 algorithm
// chart. Operators are numbered 1-6 in the comments and 0-5 in code. Modulators always have a
// higher number than the operator they modulate, so the voice processes operators from 6 down to 1.
// The feedback loop goes from the output of FeedbackFrom into the input of FeedbackTo (the same
// operator for self feedback, a chain of operators in algorithms 4 and 6)
type Algorithm struct {
	Mods         [][2]int // {modulator, target}
	Carriers     []int
	FeedbackFrom int
	FeedbackTo   int
}

// Check says what is wrong with an algorithm of your own: a modulator has to have a higher number
// than the operator it modulates (the voice runs them from 6 down to 1), there has to be a carrier,
// the feedback goes from an operator into itself or into a higher one (FeedbackFrom <= FeedbackTo)
func (a Algorithm) Check() error {
	in := func(op int) bool { return op >= 0 && op < NumOperators }
	if len(a.Carriers) == 0 {
		return fmt.Errorf("dx: an algorithm without a carrier is silent")
	}
	for _, c := range a.Carriers {
		if !in(c) {
			return fmt.Errorf("dx: carrier %d: operators are 0 to 5", c)
		}
	}
	for _, m := range a.Mods {
		if !in(m[0]) || !in(m[1]) || m[0] <= m[1] {
			return fmt.Errorf("dx: operator %d can't modulate operator %d: a modulator has a higher number", m[0]+1, m[1]+1)
		}
	}
	if !in(a.FeedbackFrom) || !in(a.FeedbackTo) || a.FeedbackFrom > a.FeedbackTo {
		return fmt.Errorf("dx: feedback from operator %d into %d: into itself or a higher one", a.FeedbackFrom+1, a.FeedbackTo+1)
	}
	return nil
}

// Uses says whether an operator is heard: a carrier, or a modulator of one that is (an operator
// that is neither plays nothing)
func (a Algorithm) Uses(op int) bool {
	seen := map[int]bool{}
	var heard func(o int) bool
	heard = func(o int) bool {
		if seen[o] {
			return false
		}
		seen[o] = true
		for _, c := range a.Carriers {
			if c == o {
				return true
			}
		}
		for _, m := range a.Mods {
			if m[0] == o && heard(m[1]) {
				return true
			}
		}
		return false
	}
	return heard(op)
}

// m builds an algorithm from 1 based operator numbers
func m(carriers []int, fbFrom int, fbTo int, mods ...[2]int) Algorithm {
	alg := Algorithm{FeedbackFrom: fbFrom - 1, FeedbackTo: fbTo - 1}
	for _, c := range carriers {
		alg.Carriers = append(alg.Carriers, c-1)
	}
	for _, mod := range mods {
		alg.Mods = append(alg.Mods, [2]int{mod[0] - 1, mod[1] - 1})
	}
	return alg
}

// Algorithms holds the 32 algorithms, index 0 is algorithm 1
var Algorithms = [32]Algorithm{
	m([]int{1, 3}, 6, 6, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 4}, [2]int{6, 5}),            // 1
	m([]int{1, 3}, 2, 2, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 4}, [2]int{6, 5}),            // 2
	m([]int{1, 4}, 6, 6, [2]int{2, 1}, [2]int{3, 2}, [2]int{5, 4}, [2]int{6, 5}),            // 3
	m([]int{1, 4}, 4, 6, [2]int{2, 1}, [2]int{3, 2}, [2]int{5, 4}, [2]int{6, 5}),            // 4
	m([]int{1, 3, 5}, 6, 6, [2]int{2, 1}, [2]int{4, 3}, [2]int{6, 5}),                       // 5
	m([]int{1, 3, 5}, 5, 6, [2]int{2, 1}, [2]int{4, 3}, [2]int{6, 5}),                       // 6
	m([]int{1, 3}, 6, 6, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 3}, [2]int{6, 5}),            // 7
	m([]int{1, 3}, 4, 4, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 3}, [2]int{6, 5}),            // 8
	m([]int{1, 3}, 2, 2, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 3}, [2]int{6, 5}),            // 9
	m([]int{1, 4}, 3, 3, [2]int{2, 1}, [2]int{3, 2}, [2]int{5, 4}, [2]int{6, 4}),            // 10
	m([]int{1, 4}, 6, 6, [2]int{2, 1}, [2]int{3, 2}, [2]int{5, 4}, [2]int{6, 4}),            // 11
	m([]int{1, 3}, 2, 2, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 3}, [2]int{6, 3}),            // 12
	m([]int{1, 3}, 6, 6, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 3}, [2]int{6, 3}),            // 13
	m([]int{1, 3}, 6, 6, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 4}, [2]int{6, 4}),            // 14
	m([]int{1, 3}, 2, 2, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 4}, [2]int{6, 4}),            // 15
	m([]int{1}, 6, 6, [2]int{2, 1}, [2]int{3, 1}, [2]int{4, 3}, [2]int{5, 1}, [2]int{6, 5}), // 16
	m([]int{1}, 2, 2, [2]int{2, 1}, [2]int{3, 1}, [2]int{4, 3}, [2]int{5, 1}, [2]int{6, 5}), // 17
	m([]int{1}, 3, 3, [2]int{2, 1}, [2]int{3, 1}, [2]int{4, 1}, [2]int{5, 4}, [2]int{6, 5}), // 18
	m([]int{1, 4, 5}, 6, 6, [2]int{2, 1}, [2]int{3, 2}, [2]int{6, 4}, [2]int{6, 5}),         // 19
	m([]int{1, 2, 4}, 3, 3, [2]int{3, 1}, [2]int{3, 2}, [2]int{5, 4}, [2]int{6, 4}),         // 20
	m([]int{1, 2, 4, 5}, 3, 3, [2]int{3, 1}, [2]int{3, 2}, [2]int{6, 4}, [2]int{6, 5}),      // 21
	m([]int{1, 3, 4, 5}, 6, 6, [2]int{2, 1}, [2]int{6, 3}, [2]int{6, 4}, [2]int{6, 5}),      // 22
	m([]int{1, 2, 4, 5}, 6, 6, [2]int{3, 2}, [2]int{6, 4}, [2]int{6, 5}),                    // 23
	m([]int{1, 2, 3, 4, 5}, 6, 6, [2]int{6, 3}, [2]int{6, 4}, [2]int{6, 5}),                 // 24
	m([]int{1, 2, 3, 4, 5}, 6, 6, [2]int{6, 4}, [2]int{6, 5}),                               // 25
	m([]int{1, 2, 4}, 6, 6, [2]int{3, 2}, [2]int{5, 4}, [2]int{6, 4}),                       // 26
	m([]int{1, 2, 4}, 3, 3, [2]int{3, 2}, [2]int{5, 4}, [2]int{6, 4}),                       // 27
	m([]int{1, 3, 6}, 5, 5, [2]int{2, 1}, [2]int{4, 3}, [2]int{5, 4}),                       // 28
	m([]int{1, 2, 3, 5}, 6, 6, [2]int{4, 3}, [2]int{6, 5}),                                  // 29
	m([]int{1, 2, 3, 6}, 5, 5, [2]int{4, 3}, [2]int{5, 4}),                                  // 30
	m([]int{1, 2, 3, 4, 5}, 6, 6, [2]int{6, 5}),                                             // 31
	m([]int{1, 2, 3, 4, 5, 6}, 6, 6),                                                        // 32
}
