package panel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGridCopy(t *testing.T) {
	pn, _ := newPanel()
	g := pn.Grid("Drums", []string{"kick", "snare"}, 8, 3, testFields...)
	g.Set(1, 0, 2, 0, 0.7)
	g.Set(1, 1, 5, 1, 4)
	g.Set(2, 0, 0, 0, 1) // overwritten by the copy
	g.Copy(1, 2)
	if g.Value(2, 0, 2, 0) != 0.7 || g.Value(2, 1, 5, 1) != 4 || g.Value(2, 0, 0, 0) != 0 {
		t.Fatal("copy should make the pattern the same as the other")
	}
	g.Set(1, 0, 2, 0, 0.1)
	if g.Value(2, 0, 2, 0) != 0.7 {
		t.Fatal("a copy should stand on its own")
	}
	g.Copy(0, 9) // out of the grid: nothing
}

func TestPresets(t *testing.T) {
	pn, _ := newPanel()
	cutoff := pn.Fader("Cutoff", 200, 12000, Exp, nil, Default(1000))
	g := pn.Grid("Drums", []string{"kick"}, 4, 2, testFields...)
	if err := pn.SavePreset("x"); err == nil {
		t.Fatal("a panel without a preset folder can't save presets")
	}
	dir := filepath.Join(t.TempDir(), "presets")
	if err := pn.Presets(dir); err != nil {
		t.Fatal(err)
	}
	cutoff.Set(3000)
	g.Set(0, 0, 1, 0, 0.9)
	if err := pn.SavePreset(" Groove 1 "); err != nil {
		t.Fatal(err)
	}
	cutoff.Set(500)
	g.Clear(0)
	if err := pn.SavePreset("empty"); err != nil {
		t.Fatal(err)
	}
	if names, _ := pn.PresetNames(); !slices.Equal(names, []string{"Groove 1", "empty"}) {
		t.Fatalf("the presets should be listed by name, got %v", names)
	}
	if err := pn.LoadPreset("Groove 1"); err != nil {
		t.Fatal(err)
	}
	if cutoff.Value() != 3000 || g.Value(0, 0, 1, 0) != 0.9 {
		t.Fatal("loading a preset should bring the controls and the grids back")
	}
	for _, bad := range []string{"", "../x", "a/b", ".hidden", `c:\x`} {
		if err := pn.SavePreset(bad); err == nil {
			t.Fatalf("%q should not be a preset name", bad)
		}
	}
	if err := pn.DeletePreset("empty"); err != nil {
		t.Fatal(err)
	}
	if err := pn.LoadPreset("empty"); !os.IsNotExist(err) {
		t.Fatalf("a deleted preset is gone, got %v", err)
	}
}

func TestPresetsWeb(t *testing.T) {
	pn, _ := newPanel()
	cutoff := pn.Fader("Cutoff", 200, 12000, Exp, nil, Default(1000))
	srv := httptest.NewServer(pn.Handler())
	defer srv.Close()
	list := func(res *http.Response) (r struct {
		Enabled bool     `json:"enabled"`
		Names   []string `json:"names"`
	}) {
		t.Helper()
		defer res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("status %d", res.StatusCode)
		}
		_ = json.NewDecoder(res.Body).Decode(&r)
		return r
	}
	res, _ := http.Get(srv.URL + "/api/presets")
	if r := list(res); r.Enabled {
		t.Fatal("presets should be off until Presets")
	}
	_ = pn.Presets(t.TempDir())
	var d description
	res, _ = http.Get(srv.URL + "/api/panel")
	_ = json.NewDecoder(res.Body).Decode(&d)
	res.Body.Close()
	if !d.Presets {
		t.Fatal("the page should know presets are on")
	}

	cutoff.Set(4000)
	if r := list(postJSON(t, srv.URL+"/api/presets", `{"action": "save", "name": "Bright"}`)); !slices.Equal(r.Names, []string{"Bright"}) {
		t.Fatalf("save should answer with the names, got %v", r.Names)
	}
	cutoff.Set(300)
	list(postJSON(t, srv.URL+"/api/presets", `{"action": "load", "name": "Bright"}`))
	if cutoff.Value() != 4000 {
		t.Fatal("load should set the panel")
	}
	for body, status := range map[string]int{
		`{"action": "load", "name": "Nope"}`:    404,
		`{"action": "save", "name": "../x"}`:    400,
		`{"action": "dance", "name": "Bright"}`: 400,
	} {
		res := postJSON(t, srv.URL+"/api/presets", body)
		res.Body.Close()
		if res.StatusCode != status {
			t.Fatalf("%s: expected %d, got %d", body, status, res.StatusCode)
		}
	}
	if r := list(postJSON(t, srv.URL+"/api/presets", `{"action": "delete", "name": "Bright"}`)); len(r.Names) != 0 {
		t.Fatal("delete should remove the preset")
	}
	res, _ = http.Get(srv.URL + "/api/snapshot?download")
	res.Body.Close()
	if cd := res.Header.Get("Content-Disposition"); cd != `attachment; filename="Test.json"` {
		t.Fatalf("a download should be a file, got %q", cd)
	}
}
