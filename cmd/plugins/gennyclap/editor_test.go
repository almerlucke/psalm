package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func init() {
	os.Setenv("GENNY_EDITOR", "off") // the tests that run a plugin do not start the server
	// and none writes in the folder of the plugin of the person who runs them
	dir, _ := os.MkdirTemp("", "gennyclap-config")
	os.Setenv("XDG_CONFIG_HOME", dir)
}

const host = "127.0.0.1:7870"

func call(t *testing.T, h http.Handler, method, url, body string, header bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	req.Host = host
	if header {
		req.Header.Set("X-Genny", "1")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("%v: %s", err, w.Body.String())
	}
	return m
}

func TestEditor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genny.js")
	t.Setenv("GENNY_SPEC", path)
	in := newInstance()
	in.handle = 5
	instances[5].Store(in)
	defer instances[5].Store(nil)
	in.loadFile()
	h := editorHandler(host)

	// the page and the plugins
	if w := call(t, h, "GET", "/", "", false); w.Code != 200 || !strings.Contains(w.Body.String(), "<title>Genny editor</title>") {
		t.Fatalf("page: %d", w.Code)
	}
	var list []map[string]any
	if err := json.Unmarshal(call(t, h, "GET", "/api/plugins", "", false).Body.Bytes(), &list); err != nil || len(list) != 1 || list[0]["path"] != path {
		t.Fatalf("plugins: %v %v", list, err)
	}

	// the state: the text that plays
	st := decode(t, call(t, h, "GET", "/api/state?i=5&text=1", "", false))
	if st["script"] != true || !strings.Contains(st["text"].(string), "function render") {
		t.Fatalf("state: %v", st)
	}

	// a check draws the notes, or says what is wrong; nothing plays
	gen := in.specGen
	ok := decode(t, call(t, h, "POST", "/api/check?i=5", `function render(p) { return [{pitch: 60, start: 0, duration: 2}, {pitch: 64, start: 2, duration: 2}] }`, true))
	if ok["ok"] != true || ok["count"] != 2.0 || ok["length"] != 4.0 {
		t.Fatalf("check: %v", ok)
	}
	bad := decode(t, call(t, h, "POST", "/api/check?i=5", `function render(p) { return 1 +`, true))
	if bad["ok"] != false || bad["error"] == "" {
		t.Fatalf("bad check: %v", bad)
	}
	if in.specGen != gen {
		t.Fatal("a check changed what plays")
	}

	// a save writes the file and is played at once
	text := `function render(p) { return [{pitch: 72, start: 0, duration: 1}] }`
	if w := call(t, h, "PUT", "/api/save?i=5&script=1", text, true); w.Code != 200 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	if data, _ := os.ReadFile(path); string(data) != text {
		t.Fatalf("file: %q", data)
	}
	if in.specGen == gen || in.text != text {
		t.Fatal("the saved text does not play")
	}
	if in.loadFile() {
		t.Fatal("the saved file was read again as if somebody else had changed it")
	}
	if _, err := os.Stat(path + ".new"); err == nil {
		t.Fatal("the temporary file is left")
	}

	// a spec file is checked as a spec
	specPath := filepath.Join(dir, "other.json")
	t.Setenv("GENNY_SPEC", specPath)
	in.loadFile()
	if r := decode(t, call(t, h, "POST", "/api/check?i=5", `not json`, true)); r["ok"] != false {
		t.Fatalf("bad spec: %v", r)
	}
	if r := decode(t, call(t, h, "POST", "/api/check?i=5", defaultSpec, true)); r["ok"] != true {
		t.Fatalf("default spec: %v", r)
	}

	// the files: save as, open
	files := decode(t, call(t, h, "GET", "/api/files?i=5", "", false))
	if files["current"] != "other.json" {
		t.Fatalf("files: %v", files)
	}
	if w := call(t, h, "PUT", "/api/saveas?i=5&name=second.js", textA, true); w.Code != 200 {
		t.Fatalf("save as: %d %s", w.Code, w.Body.String())
	}
	if filepath.Base(in.pathNow()) != "second.js" || in.text != textA || !in.isScript {
		t.Fatalf("plays %q %q", in.pathNow(), in.text)
	}
	if w := call(t, h, "PUT", "/api/saveas?i=5&name=second.js", textB, true); w.Code != 409 {
		t.Fatalf("over a file: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/saveas?i=5&name=second.js&overwrite=1", textB, true); w.Code != 200 {
		t.Fatalf("overwrite: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/open?i=5&name=other.json", "", true); w.Code != 400 {
		t.Fatalf("a file that is not there: %d", w.Code) // other.json was never written in this test
	}
	if w := call(t, h, "PUT", "/api/saveas?i=5&name=../evil.js", "x", true); w.Code != 400 {
		t.Fatalf("a bad name: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/open?i=5&name=x.js", "", false); w.Code != 403 {
		t.Fatalf("without the header: %d", w.Code)
	}
	if tpl := decode(t, call(t, h, "GET", "/api/template?kind=js", "", false)); !strings.Contains(tpl["text"].(string), "render") {
		t.Fatalf("template: %v", tpl)
	}

	// no such plugin
	if w := call(t, h, "GET", "/api/state?i=9", "", false); w.Code != 404 {
		t.Fatalf("missing plugin: %d", w.Code)
	}
}

// a page of another site cannot change anything: it cannot send the header, and a name that points here has another host
func TestEditorGuard(t *testing.T) {
	t.Setenv("GENNY_SPEC", filepath.Join(t.TempDir(), "genny.js"))
	in := newInstance()
	instances[6].Store(in)
	defer instances[6].Store(nil)
	in.loadFile()
	h := editorHandler(host)
	if w := call(t, h, "PUT", "/api/save?i=6", "x", false); w.Code != 403 {
		t.Fatalf("without the header: %d", w.Code)
	}
	if w := call(t, h, "POST", "/api/check?i=6", "x", false); w.Code != 403 {
		t.Fatalf("check without the header: %d", w.Code)
	}
	req := httptest.NewRequest("GET", "/api/state?i=6", nil)
	req.Host = "evil.example:7870"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("another host: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/save?i=6", strings.Repeat("x", maxText+1), true); w.Code != 413 {
		t.Fatalf("too long: %d", w.Code)
	}
	if w := call(t, h, "GET", "/api/save?i=6", "", false); w.Code != 405 && w.Code != 400 {
		t.Fatalf("GET save: %d", w.Code)
	}
}

// the editor starts on this machine only
func TestEditorURL(t *testing.T) {
	ln := editorListen()
	if ln == nil {
		t.Fatal("no port")
	}
	defer ln.Close()
	if a := ln.Addr().String(); !strings.HasPrefix(a, "127.0.0.1:") {
		t.Fatalf("%q", a)
	}
}

// an untitled plugin: checked and saved as the page says (script=1), no file is written
func TestEditorUntitled(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("GENNY_SPEC", "")
	in := newInstance()
	instances[7].Store(in)
	defer instances[7].Store(nil)
	in.loadFile()
	h := editorHandler(host)
	if r := decode(t, call(t, h, "POST", "/api/check?i=7&script=1", textA, true)); r["ok"] != true || r["count"] != 1.0 {
		t.Fatalf("script: %v", r)
	}
	if r := decode(t, call(t, h, "POST", "/api/check?i=7&script=0", textA, true)); r["ok"] != false {
		t.Fatalf("the same text as a spec: %v", r)
	}
	if w := call(t, h, "PUT", "/api/save?i=7&script=1", textA, true); w.Code != 200 || !in.isScript || in.text != textA {
		t.Fatalf("save: %d %q", w.Code, in.text)
	}
	st := decode(t, call(t, h, "GET", "/api/state?i=7", "", false))
	if st["linked"] != false || st["file"] != "untitled" {
		t.Fatalf("state: %v", st)
	}
	if entries, _ := os.ReadDir(config); len(entries) != 0 {
		t.Fatalf("files: %v", entries)
	}
}

// a plugin has a name: the one it was given, else its track, else its file; the project keeps the one it was given
func TestEditorNames(t *testing.T) {
	t.Setenv("GENNY_SPEC", filepath.Join(t.TempDir(), "tune.js"))
	in := newInstance()
	in.handle = 3
	instances[3].Store(in)
	defer instances[3].Store(nil)
	in.loadFile()
	h := editorHandler(host)
	if n := in.title(); n != "tune.js" {
		t.Fatalf("by its file: %q", n)
	}
	in.track = "Bass"
	if n := in.title(); n != "Bass" {
		t.Fatalf("by its track: %q", n)
	}
	if w := call(t, h, "PUT", "/api/rename?i=3", "  Lead pad ", true); w.Code != 200 {
		t.Fatalf("rename: %d", w.Code)
	}
	st := decode(t, call(t, h, "GET", "/api/state?i=3", "", false))
	if st["name"] != "Lead pad" || st["custom"] != true || st["track"] != "Bass" {
		t.Fatalf("state: %v", st)
	}
	b := newInstance()
	b.loadState(in.saveState())
	if b.name != "Lead pad" {
		t.Fatalf("the project lost the name: %q", b.name)
	}
	if w := call(t, h, "PUT", "/api/rename?i=3", strings.Repeat("x", 61), true); w.Code != 400 {
		t.Fatalf("a long name: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/rename?i=3", "two\nlines", true); w.Code != 400 {
		t.Fatalf("two lines: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/rename?i=3", "x", false); w.Code != 403 {
		t.Fatalf("without the header: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/rename?i=3", "", true); w.Code != 200 || in.title() != "Bass" {
		t.Fatalf("clearing the name: %d %q", w.Code, in.title())
	}
	in.track, in.path = "", ""
	if n := in.title(); n != "Genny 4" {
		t.Fatalf("by its place: %q", n)
	}
}
