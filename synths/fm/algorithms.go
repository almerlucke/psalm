package fm

// NumOperators is the number of FM operators of a voice
const NumOperators = 4

// Algorithm describes how the operators are connected: which operators modulate which, and which
// are carriers (heard). Operators are numbered 1-4 in the descriptions and 0-3 in code. A higher
// operator only modulates lower ones, so the voice can process them from 4 down to 1. Operator 4
// can modulate itself (feedback)
type Algorithm struct {
	Name     string
	Mods     [][2]int // {modulator, target} pairs, 0 based
	Carriers []int    // 0 based
}

// Algorithms are the classic 4 operator routings
var Algorithms = []Algorithm{
	{Name: "4>3>2>1 (stack)", Mods: [][2]int{{3, 2}, {2, 1}, {1, 0}}, Carriers: []int{0}},
	{Name: "(4+3)>2>1", Mods: [][2]int{{3, 1}, {2, 1}, {1, 0}}, Carriers: []int{0}},
	{Name: "(4 + 3>2)>1", Mods: [][2]int{{2, 1}, {1, 0}, {3, 0}}, Carriers: []int{0}},
	{Name: "(4>3 + 2)>1", Mods: [][2]int{{3, 2}, {2, 0}, {1, 0}}, Carriers: []int{0}},
	{Name: "4>3, 2>1 (two pairs)", Mods: [][2]int{{3, 2}, {1, 0}}, Carriers: []int{0, 2}},
	{Name: "4>(3, 2, 1)", Mods: [][2]int{{3, 2}, {3, 1}, {3, 0}}, Carriers: []int{0, 1, 2}},
	{Name: "4>3, 2, 1", Mods: [][2]int{{3, 2}}, Carriers: []int{0, 1, 2}},
	{Name: "4, 3, 2, 1 (additive)", Mods: nil, Carriers: []int{0, 1, 2, 3}},
}
