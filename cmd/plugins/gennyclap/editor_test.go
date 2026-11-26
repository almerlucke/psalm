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
	if w := call(t, h, "PUT", "/api/save?i=5", text, true); w.Code != 200 {
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

	// the files, and switching to another (a new one is a copy of what plays)
	files := decode(t, call(t, h, "GET", "/api/files?i=5", "", false))
	if files["current"] != "other.json" {
		t.Fatalf("files: %v", files)
	}
	if w := call(t, h, "PUT", "/api/switch?i=5&name=second.js", "", true); w.Code != 200 {
		t.Fatalf("switch: %d %s", w.Code, w.Body.String())
	}
	if filepath.Base(in.pathNow()) != "second.js" {
		t.Fatalf("plays %q", in.pathNow())
	}
	if w := call(t, h, "PUT", "/api/switch?i=5&name=../evil.js", "", true); w.Code != 400 {
		t.Fatalf("a bad name: %d", w.Code)
	}
	if w := call(t, h, "PUT", "/api/switch?i=5&name=x.js", "", false); w.Code != 403 {
		t.Fatalf("without the header: %d", w.Code)
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
