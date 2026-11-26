package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

// The editor is a web page the plugin serves on this machine (127.0.0.1): the text that plays, checked while it is typed (an error, or
// the notes it makes drawn in a piano roll), saved to the file the plugin reads, with the log and the notes that are held. A plugin
// window would need a toolkit of the host; a page needs a browser, which a musician has. One server per process (all the plugins
// of a host share it), found by the address in the log of the host and of the file next to the spec.

//go:embed editor.html
var editorPage []byte

var editor struct {
	once sync.Once
	addr string
}

// editorURL starts the editor (the first time) and says where it is, "" when it cannot run or is turned off (GENNY_EDITOR=off)
func editorURL() string {
	editor.once.Do(func() {
		if os.Getenv("GENNY_EDITOR") == "off" {
			return
		}
		ln := editorListen()
		if ln == nil {
			return
		}
		editor.addr = ln.Addr().String()
		srv := &http.Server{Handler: editorHandler(editor.addr), ReadHeaderTimeout: 5 * time.Second}
		go func() { _ = srv.Serve(ln) }()
	})
	if editor.addr == "" {
		return ""
	}
	return "http://" + editor.addr + "/"
}

// editorListen takes the port 7870 on this machine (so the address can be kept), else any free one
func editorListen() net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:7870")
	if err != nil {
		if ln, err = net.Listen("tcp", "127.0.0.1:0"); err != nil {
			return nil
		}
	}
	return ln
}

// maxText is the largest file the editor takes
const maxText = 1 << 20

// editorHandler serves the page and its calls. The calls that change something need the header X-Genny (a page of another site cannot
// send one without asking this server first, and it does not answer such questions), and every call must be for this address (a page
// that made its own name point here is not served)
func editorHandler(addr string) http.Handler {
	mux := http.NewServeMux()
	reply := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(v)
	}
	fail := func(w http.ResponseWriter, code int, format string, args ...any) {
		http.Error(w, fmt.Sprintf(format, args...), code)
	}
	instanceOf := func(w http.ResponseWriter, r *http.Request) *instance {
		i, err := strconv.Atoi(r.URL.Query().Get("i"))
		if err != nil || i < 0 || i >= len(instances) || instances[i].Load() == nil {
			fail(w, http.StatusNotFound, "no such plugin")
			return nil
		}
		return instances[i].Load()
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(editorPage)
	})
	// the plugins of the host
	mux.HandleFunc("/api/plugins", func(w http.ResponseWriter, r *http.Request) {
		out := []map[string]any{}
		for i := range instances {
			if in := instances[i].Load(); in != nil {
				in.mu.Lock()
				out = append(out, map[string]any{"i": i, "path": in.path, "file": filepath.Base(in.path), "script": in.isScript})
				in.mu.Unlock()
			}
		}
		reply(w, out)
	})
	// what plays now: the text, the log, the notes held
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		in := instanceOf(w, r)
		if in == nil {
			return
		}
		in.mu.Lock()
		st := map[string]any{"path": in.path, "script": in.isScript, "gen": in.specGen, "log": append([]string(nil), in.logLines...), "error": in.lastErr}
		if r.URL.Query().Get("text") == "1" {
			st["text"] = in.text
		}
		in.mu.Unlock()
		held := []int{}
		for _, h := range in.heldNotes() {
			held = append(held, h.Pitch)
		}
		st["held"] = held
		reply(w, st)
	})
	// the files of the folder the plugin plays from, and the one it plays
	mux.HandleFunc("/api/files", func(w http.ResponseWriter, r *http.Request) {
		in := instanceOf(w, r)
		if in == nil {
			return
		}
		reply(w, map[string]any{"current": filepath.Base(in.pathNow()), "files": in.files()})
	})
	// switch: the plugin plays another file from now on (a new one starts as a copy of what plays)
	mux.HandleFunc("/api/switch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			fail(w, http.StatusMethodNotAllowed, "PUT")
			return
		}
		in := instanceOf(w, r)
		if in == nil {
			return
		}
		if err := in.switchFile(r.URL.Query().Get("name")); err != nil {
			fail(w, http.StatusBadRequest, "%v", err)
			return
		}
		reply(w, map[string]any{"switched": true})
	})
	// check: the text in the body is rendered the way the plugin would (with the seed and the notes held now) without playing it
	mux.HandleFunc("/api/check", func(w http.ResponseWriter, r *http.Request) {
		in := instanceOf(w, r)
		if in == nil {
			return
		}
		text, ok := bodyText(w, r)
		if !ok {
			return
		}
		reply(w, in.check(text, isScriptFile(in.pathNow())))
	})
	// save: the text is written to the file the plugin reads and played from now on
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			fail(w, http.StatusMethodNotAllowed, "PUT")
			return
		}
		in := instanceOf(w, r)
		if in == nil {
			return
		}
		text, ok := bodyText(w, r)
		if !ok {
			return
		}
		if err := in.save(text); err != nil {
			fail(w, http.StatusInternalServerError, "%v", err)
			return
		}
		reply(w, map[string]any{"saved": true})
	})
	return guard(addr, mux)
}

// guard lets a call through when it is for this address and, when it changes something, has the header
func guard(addr string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != addr {
			http.Error(w, "wrong host", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Header.Get("X-Genny") != "1" {
			http.Error(w, "missing X-Genny header", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bodyText(w http.ResponseWriter, r *http.Request) (string, bool) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "PUT or POST", http.StatusMethodNotAllowed)
		return "", false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxText+1))
	if err != nil || len(data) > maxText {
		http.Error(w, "the text is too long", http.StatusRequestEntityTooLarge)
		return "", false
	}
	return string(data), true
}

func (in *instance) pathNow() string {
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.path
}

// check renders text the way the plugin would and describes what it makes: { ok, error, length, notes: [[start, end, pitch,
// velocity, channel]] (beats), log }. The runtime is a new one, so a check never disturbs the one that plays
func (in *instance) check(text string, isScript bool) map[string]any {
	seed := uint64(in.get(pSeed))
	var l *lineData
	var logs []string
	var err error
	if isScript {
		renderMu.Lock()
		var piece script.Piece
		piece, err = script.NewRuntime().RenderHeld(text, seed, in.heldNotes())
		renderMu.Unlock()
		logs = piece.Log
		if err == nil {
			l = buildPiece(piece)
		}
	} else {
		l, err = renderSpec(text, seed)
	}
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error(), "log": logs}
	}
	const most = 4000
	notes := make([][5]float64, 0, min(len(l.events), most))
	for _, e := range l.events {
		if len(notes) == most {
			break
		}
		notes = append(notes, [5]float64{e.start, e.end, e.pitch, e.vel, float64(e.channel)})
	}
	return map[string]any{"ok": true, "length": l.length, "count": len(l.events), "notes": notes, "log": logs}
}

// save writes the text to the file the plugin reads (written beside it and moved over it, so a half-written file is never read) and
// plays it from now on
func (in *instance) save(text string) error {
	in.mu.Lock()
	defer in.mu.Unlock()
	if in.path == "" {
		return fmt.Errorf("there is no file to save to")
	}
	if err := os.MkdirAll(filepath.Dir(in.path), 0o755); err != nil {
		return err
	}
	tmp := in.path + ".new"
	if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, in.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if fi, err := os.Stat(in.path); err == nil {
		in.modTime = fi.ModTime()
	}
	if text != in.text || isScriptFile(in.path) != in.isScript {
		in.text, in.isScript, in.specGen = text, isScriptFile(in.path), in.specGen+1
	}
	in.fromSave = false
	return nil
}

// editorLine is what the log says about the editor, once for each plugin
func editorLine() string {
	if u := editorURL(); u != "" {
		return "editor: " + strings.TrimSuffix(u, "/")
	}
	return ""
}
