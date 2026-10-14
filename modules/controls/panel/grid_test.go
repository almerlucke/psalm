package panel

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testFields = []Field{
	{Name: "velocity", Min: 0, Max: 1, Default: 0},
	{Name: "roll", Min: 1, Max: 8, Default: 1, Step: 1},
}

func TestGridValues(t *testing.T) {
	pn, _ := newPanel()
	g := pn.Grid("Drums", []string{"kick", "snare"}, 16, 2, testFields...)
	vel, roll := g.Field("velocity"), g.Field("roll")
	if g.Value(1, 1, 15, roll) != 1 {
		t.Fatal("cells should start at their defaults")
	}
	g.Set(1, g.Row("snare"), 3, vel, 0.8)
	g.Set(1, 1, 3, roll, 3.6)
	g.Set(0, 0, 0, roll, 99)
	if g.Value(1, 1, 3, vel) != 0.8 || g.Value(1, 1, 3, roll) != 4 || g.Value(0, 0, 0, roll) != 8 {
		t.Fatal("set should clamp and round to the field")
	}
	g.Set(5, 0, 0, vel, 1) // out of the grid: nothing
	g.Clear(1)
	if g.Value(1, 1, 3, vel) != 0 {
		t.Fatal("clear should set a pattern back to the defaults")
	}
	g.SetPattern(7)
	if g.Pattern() != 1 {
		t.Fatalf("the pattern should stay in range, got %d", g.Pattern())
	}
	defer func() {
		if recover() == nil {
			t.Fatal("a grid named like a control should panic")
		}
	}()
	pn.Toggle("Drums", nil)
}

func TestGridSnapshot(t *testing.T) {
	pn, _ := newPanel()
	g := pn.Grid("Drums", []string{"kick", "snare"}, 8, 2, testFields...)
	g.Set(0, 0, 0, 0, 0.9)
	g.Set(1, 1, 7, 1, 3)
	g.SetPattern(1)
	snap := pn.Snapshot()
	if snap["Drums/A/kick/1/velocity"] != 0.9 || snap["Drums/B/snare/8/roll"] != 3 || snap["Drums"] != 1 || len(snap) != 3 {
		t.Fatalf("the snapshot should hold the pattern and the cells away from their defaults: %v", snap)
	}
	g.Clear(0)
	g.Clear(1)
	g.Set(0, 1, 1, 0, 0.5) // not in the snapshot: back to its default on restore
	pn.Restore(snap)
	if g.Value(0, 0, 0, 0) != 0.9 || g.Value(1, 1, 7, 1) != 3 || g.Value(0, 1, 1, 0) != 0 || g.Pattern() != 1 {
		t.Fatal("restore should bring the grid back exactly")
	}
}

func TestGridWeb(t *testing.T) {
	pn, _ := newPanel()
	pn.Section("Beat")
	g := pn.Grid("Drums", []string{"kick", "snare"}, 4, 2, testFields...)
	pn.Toggle("Fill", nil)
	srv := httptest.NewServer(pn.Handler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/panel")
	if err != nil {
		t.Fatal(err)
	}
	var d description
	_ = json.NewDecoder(res.Body).Decode(&d)
	res.Body.Close()
	c := d.Sections[0].Controls[0]
	if c.Kind != GridKind || c.Grid == nil || c.Grid.Steps != 4 || len(c.Grid.Cells) != 2*2*4*2 || d.Sections[0].Controls[1].Name != "Fill" {
		t.Fatalf("the grid should be described in its place: %+v", d.Sections[0])
	}

	res = postJSON(t, srv.URL+"/api/grid", `{"name": "Drums", "pattern": 1, "set": [[1, 0, 2, 0, 0.7], [1, 1, 3, 1, 2]]}`)
	res.Body.Close()
	if res.StatusCode != 204 || g.Value(1, 0, 2, 0) != 0.7 || g.Value(1, 1, 3, 1) != 2 || g.Pattern() != 1 {
		t.Fatalf("a batch should set the cells and the pattern, got %d", res.StatusCode)
	}
	if res := postJSON(t, srv.URL+"/api/grid", `{"name": "Nope"}`); res.StatusCode != 404 {
		t.Fatalf("an unknown grid should be 404, got %d", res.StatusCode)
	}

	events, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer events.Body.Close()
	lines := make(chan update)
	go func() {
		sc := bufio.NewScanner(events.Body)
		sc.Buffer(nil, 1<<20)
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				var u update
				_ = json.Unmarshal([]byte(data), &u)
				lines <- u
			}
		}
	}()
	next := func() *gridUpdate {
		for {
			select {
			case u := <-lines:
				if gu := u.Grids["Drums"]; gu != nil {
					return gu
				}
			case <-time.After(2 * time.Second):
				t.Fatal("no grid update")
			}
		}
	}
	if gu := next(); len(gu.Cells) != 32 || gu.Pattern != 1 {
		t.Fatalf("the first update should hold the cells, got %+v", gu)
	}
	g.SetPlayhead(0, 2)
	if gu := next(); gu.Cells != nil || gu.Playheads[0] != 2 {
		t.Fatalf("a playhead moving should be sent without the cells, got %+v", gu)
	}
	g.Set(0, 0, 0, 0, 1)
	if gu := next(); gu.Cells == nil || gu.Cells[0] != 1 {
		t.Fatal("a changed cell should send the cells")
	}
}

// the steps of a pattern are kept in a snapshot (when they aren't the default), and come back
func TestGridPatternSteps(t *testing.T) {
	pn, _ := newPanel()
	g := pn.Grid("Drums", []string{"kick"}, 8, 3, testFields...)
	g.DefaultSteps(4)
	g.SetPatternSteps(1, 7)
	g.SetPatternSteps(2, 99) // no more than the grid has
	if g.PatternSteps(0) != 4 || g.PatternSteps(1) != 7 || g.PatternSteps(2) != 8 {
		t.Fatalf("steps %d %d %d", g.PatternSteps(0), g.PatternSteps(1), g.PatternSteps(2))
	}
	snap := pn.Snapshot()
	if snap["Drums/B/steps"] != 7 || snap["Drums/C/steps"] != 8 || snap["Drums/A/steps"] != 0 {
		t.Fatalf("the snapshot should hold the steps away from the default: %v", snap)
	}
	g.SetPatternSteps(0, 2)
	g.SetPatternSteps(1, 3)
	pn.Restore(snap)
	if g.PatternSteps(0) != 4 || g.PatternSteps(1) != 7 || g.PatternSteps(2) != 8 {
		t.Fatalf("restored steps %d %d %d", g.PatternSteps(0), g.PatternSteps(1), g.PatternSteps(2))
	}
}

// patterns out of the rotation are skipped by Next, and kept in a snapshot
func TestGridRotation(t *testing.T) {
	pn, _ := newPanel()
	g := pn.Grid("Drums", []string{"kick"}, 4, 5, testFields...)
	g.SetActive(1, false)
	g.SetActive(2, false)
	if g.Next(0) != 3 || g.Next(3) != 4 || g.Next(4) != 0 || g.Next(1) != 3 {
		t.Fatalf("next: %d %d %d %d", g.Next(0), g.Next(3), g.Next(4), g.Next(1))
	}
	snap := pn.Snapshot()
	if snap["Drums/B/off"] != 1 || snap["Drums/C/off"] != 1 || len(snap) != 3 {
		t.Fatalf("the snapshot should hold the patterns out of the rotation: %v", snap)
	}
	g.SetActive(1, true)
	g.SetActive(4, false)
	pn.Restore(snap)
	if g.Active(1) || g.Active(2) || !g.Active(4) || !g.Active(0) {
		t.Fatal("restore should bring the rotation back")
	}
	g.SetActive(0, false)
	g.SetActive(3, false)
	g.SetActive(4, false)
	g.SetActive(1, false)
	if g.Next(2) != 2 {
		t.Fatal("with none active the pattern stays")
	}
}
