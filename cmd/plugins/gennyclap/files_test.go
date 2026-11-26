package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// each instance plays a file of its own, that starts as what earlier versions played for all
func TestOwnFiles(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("GENNY_SPEC", "")
	dir := filepath.Join(config, "psalm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := `function render(p) { return [{pitch: 61, start: 0, duration: 1}] }`
	if err := os.WriteFile(filepath.Join(dir, "genny.js"), []byte(shared), 0o644); err != nil {
		t.Fatal(err)
	}
	a, b := newInstance(), newInstance()
	a.loadFile()
	b.loadFile()
	if a.path == b.path || a.path == "" || filepath.Dir(a.path) != dir || !strings.HasPrefix(filepath.Base(a.path), "genny-") {
		t.Fatalf("%q %q", a.path, b.path)
	}
	if a.text != shared || !a.isScript || filepath.Ext(a.path) != ".js" {
		t.Fatalf("a starts as %q (%s)", a.text, a.path)
	}
	// an edit of the file of one does not reach the other
	if err := a.save("function render(p) { return [] }"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(b.path); string(data) != shared {
		t.Fatalf("b's file: %q", data)
	}
	// and the instance keeps its file when genny.js changes
	if err := os.WriteFile(filepath.Join(dir, "genny.js"), []byte("// other"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := b.path
	b.loadFile()
	if b.path != path || b.text != shared {
		t.Fatalf("b moved: %q %q", b.path, b.text)
	}

	// a project saved with a text and a file that is still there plays that file; one whose file is gone makes a new file of the text
	doc := a.saveState()
	c := newInstance()
	c.loadState(doc)
	c.loadFile()
	if c.path != a.path || c.text != "function render(p) { return [] }" {
		t.Fatalf("c: %q %q", c.path, c.text)
	}
	if err := os.Remove(a.path); err != nil {
		t.Fatal(err)
	}
	d := newInstance()
	d.loadState(doc)
	d.loadFile()
	if d.path == a.path || d.path == "" || d.text != "function render(p) { return [] }" {
		t.Fatalf("d: %q %q", d.path, d.text)
	}
	if data, _ := os.ReadFile(d.path); string(data) != d.text {
		t.Fatalf("d's file: %q", data)
	}

	// switch to a file, a new one starts as a copy of what plays; names are checked
	if err := b.switchFile("mine.js"); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(b.path) != "mine.js" || b.text != shared {
		t.Fatalf("%q %q", b.path, b.text)
	}
	if !slices.Contains(b.files(), "mine.js") || slices.Contains(b.files(), "genny.js.log") {
		t.Fatalf("%v", b.files())
	}
	for _, bad := range []string{"../x.js", "a/b.js", "x.txt", "x", ".js", "x.js.log", ""} {
		if b.switchFile(bad) == nil {
			t.Errorf("%q is taken as a name", bad)
		}
	}
	// the project remembers it
	e := newInstance()
	e.loadState(b.saveState())
	e.loadFile()
	if e.path != b.path {
		t.Fatalf("%q", e.path)
	}
	// without any file of earlier versions a new file is the demo
	empty := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", empty)
	f := newInstance()
	f.loadFile()
	if f.text != defaultSpec || filepath.Ext(f.path) != ".json" {
		t.Fatalf("%q %s", f.text[:20], f.path)
	}
}
