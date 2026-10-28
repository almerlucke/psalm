package rhythm

import (
	"slices"
	"strings"
	"testing"
)

func count(r []bool) (n int) {
	for _, b := range r {
		if b {
			n++
		}
	}
	return
}

func TestEuclid(t *testing.T) {
	for _, c := range []struct{ k, n int }{{3, 8}, {5, 8}, {5, 16}, {7, 12}, {1, 4}, {0, 4}, {4, 4}} {
		if got := count(Euclid(c.k, c.n, 0)); got != c.k {
			t.Errorf("E(%d,%d) has %d hits", c.k, c.n, got)
		}
	}
	// the evenest: the gaps between hits differ by at most one
	r := Euclid(5, 16, 0)
	var gaps []int
	last := -1
	for i, b := range r {
		if b {
			if last >= 0 {
				gaps = append(gaps, i-last)
			}
			last = i
		}
	}
	gaps = append(gaps, 16-last+slices.Index(r, true))
	if slices.Max(gaps)-slices.Min(gaps) > 1 {
		t.Errorf("uneven gaps %v", gaps)
	}
	// 3 over 8 is the tresillo (x . . x . . x .), in some rotation
	tres := Steps(Euclid(3, 8, 0))
	ok := false
	for rot := range 8 {
		if Steps(Rotate(Euclid(3, 8, 0), rot)) == "x . . x . . x ." {
			ok = true
		}
	}
	if !ok {
		t.Errorf("3 over 8 is %s", tres)
	}
	if Steps(Rotate(Euclid(3, 8, 0), 1)) == Steps(Rotate(Euclid(3, 8, 0), 0)) {
		t.Error("a rotation did not turn it")
	}
	if !slices.Equal(Rotate(Euclid(5, 16, 3), -3), Euclid(5, 16, 0)) {
		t.Error("rotating back")
	}
}

func TestAdditive(t *testing.T) {
	r, err := Additive("3+3+2")
	if err != nil || Steps(r) != "x . . x . . x ." {
		t.Fatalf("%v %v", Steps(r), err)
	}
	if _, err := Additive("3+x"); err == nil {
		t.Error("a bad group was taken")
	}
}

func TestCellular(t *testing.T) {
	// rule 90 from one cell is the Sierpinski triangle: symmetric
	r := Cellular(90, 15, 3, 0)
	for i := range r {
		if r[i] != r[len(r)-1-i] {
			t.Fatalf("rule 90 is not symmetric: %s", Steps(r))
		}
	}
	if count(Cellular(30, 16, 5, 0)) == 0 || slices.Equal(Cellular(30, 16, 5, 0), Cellular(30, 16, 6, 0)) {
		t.Error("rule 30 does not change")
	}
	if !slices.Equal(Cellular(30, 16, 5, 7), Cellular(30, 16, 5, 7)) {
		t.Error("not repeatable")
	}
}

func TestExpand(t *testing.T) {
	got, err := Expand("E(3,8) | A(2+2) | CA(90,8,0)")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "x . . x . . x .") || !strings.Contains(got, "| x . x . |") {
		t.Errorf("%q", got)
	}
	if got, _ := Expand("x . x?50 [x x]"); got != "x . x?50 [x x]" {
		t.Errorf("text without functions changed: %q", got)
	}
	for _, bad := range []string{"E(9,4)", "E(3)", "E(a,4)", "A(3+q)", "E(3,8"} {
		if _, err := Expand(bad); err == nil {
			t.Errorf("%q was taken", bad)
		}
	}
}

func TestCond(t *testing.T) {
	c, err := ParseCond("2/4")
	if err != nil {
		t.Fatal(err)
	}
	var on []int64
	for p := range int64(8) {
		if c.Holds(p, false) {
			on = append(on, p)
		}
	}
	if !slices.Equal(on, []int64{1, 5}) {
		t.Errorf("2/4 plays on %v", on)
	}
	n, _ := ParseCond("!2/4")
	if n.Holds(1, false) || !n.Holds(0, false) {
		t.Error("!2/4")
	}
	f, _ := ParseCond("fill")
	nf, _ := ParseCond("!fill")
	if f.Holds(0, false) || !f.Holds(0, true) || nf.Holds(0, true) || !nf.Holds(0, false) {
		t.Error("fill")
	}
	fi, _ := ParseCond("first")
	if !fi.Holds(0, false) || fi.Holds(1, false) {
		t.Error("first")
	}
	for _, bad := range []string{"5/4", "1/1", "x", "2/", "fil"} {
		if _, err := ParseCond(bad); err == nil {
			t.Errorf("%q was taken", bad)
		}
	}
}

func TestLaneAndGroove(t *testing.T) {
	l, err := ParseLane("100 . 50")
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := l.At(3); !ok || v != 100 {
		t.Error("a lane goes round")
	}
	if _, ok := l.At(1); ok {
		t.Error("a gap has no value")
	}
	if _, err := ParseLane("1 x"); err == nil {
		t.Error("a bad value was taken")
	}
	for _, g := range Grooves {
		for s := range int64(32) {
			sh, v := g.At(s)
			if sh < -0.5 || sh > 0.5 || v <= 0 || v > 1.3 {
				t.Errorf("%s step %d: %v %v", g.Name, s, sh, v)
			}
		}
	}
	if sh, _ := Grooves[3].At(1); sh <= 0 {
		t.Error("mpc 62 does not push the offbeat late")
	}
}
