package patcher_test

import (
	"math"
	"strings"
	"testing"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

func TestParseLanes(t *testing.T) {
	lanes, err := patcher.ParseLanes("Cutoff: 6 0.8 smooth, 0 0.2, 12 0.5 step # a comment\n\nLevel:\nPan: 0 1.5, 4 0 exp")
	if err != nil {
		t.Fatal(err)
	}
	if len(lanes) != 3 || lanes[0].Name != "Cutoff" || len(lanes[0].Points) != 3 || len(lanes[1].Points) != 0 {
		t.Fatalf("lanes %+v", lanes)
	}
	if p := lanes[0].Points; p[0].X != 0 || p[1].X != 6 || p[1].Shape != "smooth" || p[2].Shape != "step" { // sorted by position
		t.Errorf("points %+v", p)
	}
	if lanes[2].Points[0].Y != 1 { // kept in 0 to 1
		t.Errorf("a value of 1.5 kept as %v", lanes[2].Points[0].Y)
	}
	again, err := patcher.ParseLanes(patcher.FormatLanes(lanes))
	if err != nil || patcher.FormatLanes(again) != patcher.FormatLanes(lanes) {
		t.Errorf("not the same written and read: %v\n%s\n%s", err, patcher.FormatLanes(lanes), patcher.FormatLanes(again))
	}
	for _, bad := range []string{"a: 1", "a: x 1", "a: 1 y", "a: -1 0.5", "a: 1 0.5 wobbly", "a: 1 0.5 smooth extra"} {
		if _, err := patcher.ParseLanes(bad); err == nil {
			t.Errorf("%q read", bad)
		}
	}
}

func TestLaneAt(t *testing.T) {
	l, _ := patcher.ParseLanes("a: 2 0.2, 4 0.8, 6 0.2 step, 8 0.6 smooth, 10 0.1, 12 0.4 exp, 14 0.1")
	for _, c := range []struct{ x, want float64 }{
		{0, 0.2}, {2, 0.2}, {3, 0.5}, {4, 0.8}, {5, 0.5}, {6, 0.2}, {7, 0.2}, {8, 0.6}, {9, 0.35}, {10, 0.1}, {11, 0.25},
		{13, 0.2}, {14, 0.1}, {20, 0.1},
	} {
		if got := l[0].At(c.x); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("at %v: %v, want %v", c.x, got, c.want)
		}
	}
	if (patcher.LaneData{}).At(3) != 0 {
		t.Error("a lane without points is not 0")
	}
}

// the lanes follow the bars of the song, each lane has its own mod output, and what the page draws is the lanes with where it is
func TestAutomationFollowsTheSong(t *testing.T) {
	r := newSongRig(t, "A 2, B 2") // a bar: 2 s
	a := r.add("automation")
	if err := r.pt.SetText(a, "lanes", "up: 0 0, 4 1\nstep: 0 0 step, 2 1"); err != nil || r.pt.TextError(a, "lanes") != "" {
		t.Fatal(err, r.pt.TextError(a, "lanes"))
	}
	m := r.pt.Module(a).(*patcher.Automation)
	r.run(3) // a bar and a half in: the second section is not there yet
	if v := m.Value(0); math.Abs(v-0.375) > 0.03 {
		t.Errorf("1.5 bars in the first lane is %v, want 0.375", v)
	}
	if v := m.Value(1); v != 0 {
		t.Errorf("the second lane (a step to 1 at 2 bars) is %v at 1.5 bars", v)
	}
	r.run(2) // 2.5 bars: in B
	if v := m.Value(1); v != 1 {
		t.Errorf("the second lane is %v at 2.5 bars, want 1", v)
	}
	view := m.View()
	if view.Length != 4 || len(view.Sections) != 2 || view.Sections[1].Name != "B" || view.Sections[1].Start != 2 || len(view.Lanes) != 4 || view.Head.Rate < 0.4 || view.Head.Rate > 0.6 {
		t.Errorf("the view %+v", view)
	}
	if view.Head.X < 2.3 || view.Head.X > 2.7 || len(view.Head.Values) != 4 || view.Head.Values[1] != 1 {
		t.Errorf("it is at %+v", view.Head)
	}

	// edited on the page: the lanes are kept as text, the module takes them
	lanes := m.Lanes()
	lanes[2].Points = []patcher.Point{{X: 0, Y: 0.25}}
	lanes[2].Name = "a: lane, with # marks"
	if err := r.pt.SetLanes(a, lanes); err != nil {
		t.Fatal(err)
	}
	r.run(0.2)
	if v := m.Value(2); v != 0.25 {
		t.Errorf("a lane drawn on the page is %v", v)
	}
	if txt := r.pt.Texts(a)[0].Value; !strings.Contains(txt, "a lane with marks: 0 0.25") {
		t.Errorf("the text of the lanes:\n%s", txt)
	}
	if err := r.pt.SetLanes("nowhere", lanes); err == nil {
		t.Error("lanes set on a module that is not there")
	}
}

// every lane is a mod output: a mod cable from the last of four reaches a setting
func TestAutomationModCable(t *testing.T) {
	r := newSongRig(t, "A 2")
	a, clock := r.add("automation"), r.add("clock")
	if _, err := r.pt.Connect(patcher.Cable{Type: patcher.Mod, From: a, Out: 3, To: clock, Key: "division"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pt.Connect(patcher.Cable{Type: patcher.Mod, From: a, Out: 4, To: clock, Key: "division"}); err == nil {
		t.Error("a mod cable from a lane that is not there")
	}
}
