package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	textA = `function render(p) { return [{pitch: 61, start: 0, duration: 1}] }`
	textB = `function render(p) { return [{pitch: 62, start: 0, duration: 1}] }`
)

// a plugin without a file is untitled: it plays the demo, or the text of the project, and writes no file
func TestUntitled(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("GENNY_SPEC", "")
	in := newInstance()
	in.loadFile()
	if in.path != "" || in.text != defaultSpec || in.isScript {
		t.Fatalf("%q %q", in.path, in.text[:min(20, len(in.text))])
	}
	if entries, _ := os.ReadDir(config); len(entries) != 0 {
		t.Fatalf("files were written: %v", entries)
	}

	// saving keeps the text in the project only
	if err := in.save(textA, true); err != nil {
		t.Fatal(err)
	}
	if in.text != textA || !in.isScript || in.path != "" {
		t.Fatalf("%q %v %q", in.text, in.isScript, in.path)
	}
	if entries, _ := os.ReadDir(config); len(entries) != 0 {
		t.Fatalf("files were written: %v", entries)
	}
	// and the project brings it back, on another machine as well
	doc := in.saveState()
	b := newInstance()
	b.loadState(doc)
	b.loadFile()
	if b.text != textA || !b.isScript || b.path != "" {
		t.Fatalf("%q %v %q", b.text, b.isScript, b.path)
	}
}

// save as makes a file and the plugin plays it from then on, several plugins can open it, a file is not written over by mistake
func TestSaveAsAndOpen(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("GENNY_SPEC", "")
	dir := filepath.Join(config, "psalm")
	a, b := newInstance(), newInstance()
	a.loadFile()
	b.loadFile()

	if err := a.saveAs("tune.js", textA, false); err != nil {
		t.Fatal(err)
	}
	if a.path != filepath.Join(dir, "tune.js") || a.text != textA || !a.isScript {
		t.Fatalf("%q %q %v", a.path, a.text, a.isScript)
	}
	if data, _ := os.ReadFile(a.path); string(data) != textA {
		t.Fatalf("%q", data)
	}
	if err := b.saveAs("tune.js", textB, false); !errors.Is(err, errExists) {
		t.Fatalf("a file was written over: %v", err)
	}
	if data, _ := os.ReadFile(a.path); string(data) != textA {
		t.Fatalf("%q", data)
	}
	for _, bad := range []string{"../x.js", "a/b.js", "x.txt", "x", ".js", "x.js.log", ""} {
		if a.saveAs(bad, "x", true) == nil || b.openFile(bad) == nil {
			t.Errorf("%q is taken as a name", bad)
		}
	}
	if err := b.openFile("nothing.js"); err == nil {
		t.Error("opened a file that is not there")
	}

	// b plays the same file; a save of one writes the file, the other plays it when it looks again
	if err := b.openFile("tune.js"); err != nil {
		t.Fatal(err)
	}
	if b.text != textA || b.path != a.path {
		t.Fatalf("%q %q", b.text, b.path)
	}
	if err := a.save(textB, true); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(a.path, future, future)
	if !b.loadFile() || b.text != textB {
		t.Fatalf("b did not follow: %q", b.text)
	}
	if a.loadFile() {
		t.Fatal("a read its own save as a change")
	}
	if !slices.Contains(b.files(), "tune.js") || slices.Contains(b.files(), "tune.js.log") {
		t.Fatalf("%v", b.files())
	}

	// the project remembers the file; when the file is gone the copy of the project plays
	doc := a.saveState()
	c := newInstance()
	c.loadState(doc)
	c.loadFile()
	if c.path != a.path || c.text != textB {
		t.Fatalf("%q %q", c.path, c.text)
	}
	if err := os.Remove(a.path); err != nil {
		t.Fatal(err)
	}
	d := newInstance()
	d.loadState(doc)
	d.loadFile()
	if d.path != "" || d.text != textB || d.lostPath != a.path || !d.isScript {
		t.Fatalf("%q %q %q", d.path, d.text, d.lostPath)
	}
}

// a plugin that plays a file follows the file when it is changed on disk
func TestRunFollowsItsFile(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("GENNY_SPEC", "")
	in := newInstance()
	in.stop, in.done = make(chan struct{}), make(chan struct{})
	go in.run()
	defer func() { close(in.stop); <-in.done }()
	wait(t, "the demo", func() bool { return in.store.Load() != nil })
	if entries, _ := os.ReadDir(config); len(entries) != 0 {
		t.Fatalf("an untitled plugin wrote files: %v", entries)
	}
	if err := in.saveAs("mine.js", textA, false); err != nil {
		t.Fatal(err)
	}
	wait(t, "the script", func() bool {
		st := in.store.Load()
		return st != nil && len(st.lines[0].events) == 1 && st.lines[0].events[0].pitch == 61
	})
	path := filepath.Join(config, "psalm", "mine.js")
	if err := os.WriteFile(path, []byte(textB), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(3 * time.Second)
	_ = os.Chtimes(path, future, future)
	wait(t, "the file changed on disk", func() bool {
		st := in.store.Load()
		return len(st.lines[0].events) == 1 && st.lines[0].events[0].pitch == 62
	})
	if data, _ := os.ReadFile(path + ".log"); !strings.Contains(string(data), "1 notes in 4 beats") {
		t.Fatalf("log:\n%s", data)
	}
}
