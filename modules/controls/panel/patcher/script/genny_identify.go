package script

import (
	"math"
	"slices"
	"strconv"

	"github.com/almerlucke/genny/chord"
	gharmony "github.com/almerlucke/genny/harmony"
	"github.com/dop251/goja"
)

// quality is a kind of chord: the intervals above the root as genny's chord package has them, and how it is written
type quality struct {
	name, symbol string
	ivs          []float64
}

// qualities in the order they are tried: a set of notes that fits two of them (C6 and Am7) is the first, unless the lowest note
// decides (it is tried as the root first)
var qualities = []quality{
	{"major", "", chord.Major}, {"minor", "m", chord.Minor}, {"diminished", "dim", chord.Diminished}, {"augmented", "aug", chord.Augmented},
	{"sus2", "sus2", chord.Sus2}, {"sus4", "sus4", chord.Sus4}, {"power", "5", []float64{0, 7}},
	{"dominant7", "7", chord.Dominant7}, {"major7", "maj7", chord.Major7}, {"minor7", "m7", chord.Minor7},
	{"minorMajor7", "mMaj7", chord.MinorMajor7}, {"halfDiminished", "m7b5", chord.HalfDiminished}, {"diminished7", "dim7", chord.Diminished7},
	{"major6", "6", chord.Major6}, {"minor6", "m6", chord.Minor6}, {"add9", "add9", chord.Add9},
	{"dominant9", "9", chord.Dominant9}, {"major9", "maj9", chord.Major9}, {"minor9", "m9", chord.Minor9},
}

var pitchNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

func pc(x int) int { return ((x % 12) + 12) % 12 }

// found is a chord that was recognised in a set of pitches
type found struct {
	root, bass, inversion int
	q                     quality
	pitchClasses          []int // the distinct pitch classes, root first
}

func (f found) name() string { return pitchNames[f.root] + " " + f.q.name }
func (f found) symbol() string {
	s := pitchNames[f.root] + f.q.symbol
	if f.bass != f.root {
		s += "/" + pitchNames[f.bass]
	}
	return s
}

// identify names the chord that the pitches make: the lowest pitch is tried as the root first, so a chord in root position is
// named from its bass and an inversion (the bass is another chord tone) is named from the root it leans on
func identify(pitches []int) (found, bool) {
	if len(pitches) == 0 {
		return found{}, false
	}
	lowest := slices.Min(pitches)
	var set []int
	for _, p := range pitches {
		if !slices.Contains(set, pc(p)) {
			set = append(set, pc(p))
		}
	}
	if len(set) < 2 {
		return found{}, false
	}
	// the roots to try: the bass, then the others from the lowest up
	sorted := slices.Clone(pitches)
	slices.Sort(sorted)
	var roots []int
	for _, p := range sorted {
		if !slices.Contains(roots, pc(p)) {
			roots = append(roots, pc(p))
		}
	}
	for _, root := range roots {
		for _, q := range qualities {
			if len(q.ivs) != len(set) {
				continue
			}
			ok := true
			var cls []int
			for _, iv := range q.ivs {
				c := pc(root + int(iv))
				cls = append(cls, c)
				if !slices.Contains(set, c) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			bass := pc(lowest)
			return found{root: root, bass: bass, inversion: slices.Index(cls, bass), q: q, pitchClasses: cls}, true
		}
	}
	return found{}, false
}

// pitchList reads the pitches of a script's array: numbers, or notes with a pitch (p.held)
func (r *Runtime) pitchList(v goja.Value, who string) []int {
	var out []int
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		panic(r.vm.NewTypeError(who + ": an array of pitches"))
	}
	arr := v.ToObject(r.vm)
	n := int(arr.Get("length").ToInteger())
	for i := 0; i < n; i++ {
		item := arr.Get(strconv.Itoa(i))
		if o, ok := item.(*goja.Object); ok {
			if p := o.Get("pitch"); p != nil && !goja.IsUndefined(p) {
				item = p
			}
		}
		var f float64
		switch n := item.Export().(type) {
		case int64:
			f = float64(n)
		case float64:
			f = n
		default:
			panic(r.vm.NewTypeError(who + ": pitches are numbers (or notes with a pitch)"))
		}
		if math.IsNaN(f) || f < -1000 || f > 1000 {
			panic(r.vm.NewTypeError(who + ": a pitch is out of range"))
		}
		out = append(out, int(math.Round(f)))
	}
	return out
}

// foundValue is a recognised chord for a script
func (r *Runtime) foundValue(f found) *goja.Object {
	o := r.vm.NewObject()
	_ = o.Set("name", f.name())
	_ = o.Set("symbol", f.symbol())
	_ = o.Set("quality", f.q.name)
	_ = o.Set("root", f.root)
	_ = o.Set("rootName", pitchNames[f.root])
	_ = o.Set("bass", f.bass)
	_ = o.Set("inversion", f.inversion)
	_ = o.Set("intervals", f.q.ivs)
	_ = o.Set("pitchClasses", f.pitchClasses)
	return o
}

// chordOf is p.chord: what the pitches are, or null
func (r *Runtime) chordOf(pitches []int) goja.Value {
	if f, ok := identify(pitches); ok {
		return r.foundValue(f)
	}
	return goja.Null()
}

// analyze finds the pitches among the chords of a key: diatonic, borrowed from the parallel key, or the dominant of a degree
func analyze(f found, k gharmony.Key) (c gharmony.Chord, how string, degree int, ok bool) {
	same := func(c gharmony.Chord) bool {
		a, b := c.PitchClasses(), f.pitchClasses
		if len(a) != len(b) {
			return false
		}
		for _, x := range a {
			if !slices.Contains(b, x) {
				return false
			}
		}
		return true
	}
	seventh := len(f.pitchClasses) == 4
	for d := 0; d < 7; d++ {
		if c := gharmony.Diatonic(k, d, seventh); same(c) {
			return c, "diatonic", d, true
		}
	}
	for d := 0; d < 7; d++ {
		if c, ok := gharmony.Borrowed(k, d); ok && same(c) {
			return c, "borrowed", d, true
		}
	}
	for d := 0; d < 7; d++ {
		if c, ok := gharmony.SecondaryDominant(k, d, seventh); ok && same(c) {
			return c, "secondary", d, true
		}
	}
	return gharmony.Chord{}, "", -1, false
}

// identifyGenny adds identify and analyze to genny
func (r *Runtime) identifyGenny(set func(string, func(goja.FunctionCall) goja.Value)) {
	// identify(pitches): the chord those pitches make, { name: "C major", symbol: "Cmaj7/E", quality, root (pitch class), rootName,
	// bass (pitch class), inversion (0 is root position), intervals, pitchClasses }, or null when they are no chord it knows. The
	// pitches are numbers or notes with a pitch, so p.held and p.pitches both do
	set("identify", r.guarded("identify", func(call goja.FunctionCall) goja.Value {
		return r.chordOf(r.pitchList(call.Argument(0), "identify"))
	}))
	// analyze(pitches, key = "C major"): identify, and what the chord is in the key: { numeral: "V7", function: "T" | "S" | "D",
	// degree (0 is I), kind: "diatonic" | "borrowed" | "secondary" } or no numeral at all when the key does not have it
	set("analyze", r.guarded("analyze", func(call goja.FunctionCall) goja.Value {
		f, ok := identify(r.pitchList(call.Argument(0), "analyze"))
		if !ok {
			return goja.Null()
		}
		o := r.foundValue(f)
		key := gharmony.Key{}
		if k := call.Argument(1); !goja.IsUndefined(k) && !goja.IsNull(k) {
			key = r.keyArg(k)
		}
		if c, how, d, ok := analyze(f, key); ok {
			_ = o.Set("numeral", c.Numeral)
			_ = o.Set("function", c.Function.String())
			_ = o.Set("kind", how)
			_ = o.Set("degree", d) // of the chord itself, or of the degree it borrows or points at
			_ = o.Set("of", c.Of)
		}
		return o
	}))
}
