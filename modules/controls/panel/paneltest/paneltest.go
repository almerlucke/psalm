// Package paneltest drives pages of the panel (modules/controls/panel) and of its plugins in
// headless Chrome for tests, over the DevTools protocol on a pair of pipes
// (--remote-debugging-pipe: Chrome reads commands on its fd 3 and writes on its fd 4, every message
// JSON ending in a 0 byte). A test is skipped with -short, and when there is no Chrome (PSALM_CHROME
// names one that is not on the path). Any exception or console error of a page fails its test.
//
//	func TestMain(m *testing.M) { paneltest.Main(m) }
//
//	func TestPage(t *testing.T) {
//		b := paneltest.Start(t)
//		tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".knob")`)
//		tb.Tap(`document.querySelector("button")`)
//		paneltest.Eventually(t, "the button pressed", func() bool { return pressed.Load() })
//	}
package paneltest

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// Browser is a headless Chrome, shared by the tests of a package
type Browser struct {
	cmd    *exec.Cmd
	in     *os.File // to Chrome
	mu     sync.Mutex
	nextID int
	wait   map[int]chan cdpMessage
	events chan cdpMessage
	done   chan struct{}
}

type cdpMessage struct {
	ID        int             `json:"id"`
	Method    string          `json:"method"`
	SessionID string          `json:"sessionId"`
	Params    json.RawMessage `json:"params"`
	Result    json.RawMessage `json:"result"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error"`
}

var (
	sharedBrowser *Browser
	browserErr    error
	browserOnce   sync.Once
)

// chromePath returns the Chrome to use, "" for none
func chromePath() string {
	if p := os.Getenv("PSALM_CHROME"); p != "" {
		return p
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// Start returns the browser of the tests, started the first time, or skips the test
func Start(t *testing.T) *Browser {
	t.Helper()
	if testing.Short() {
		t.Skip("page tests drive a browser: not with -short")
	}
	path := chromePath()
	if path == "" {
		t.Skip("no Chrome for the page tests (set PSALM_CHROME)")
	}
	browserOnce.Do(func() { sharedBrowser, browserErr = launch(path) })
	if browserErr != nil {
		t.Fatalf("starting Chrome: %v", browserErr)
	}
	return sharedBrowser
}

func launch(path string) (*Browser, error) {
	dir, err := os.MkdirTemp("", "psalm-chrome-")
	if err != nil {
		return nil, err
	}
	toChromeR, toChromeW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	fromChromeR, fromChromeW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(path, "--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check",
		"--remote-debugging-pipe", "--user-data-dir="+dir, "--window-size=1000,900", "about:blank")
	cmd.ExtraFiles = []*os.File{toChromeR, fromChromeW} // fd 3 and fd 4
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	toChromeR.Close()
	fromChromeW.Close()
	b := &Browser{cmd: cmd, in: toChromeW, wait: map[int]chan cdpMessage{}, events: make(chan cdpMessage, 256), done: make(chan struct{})}
	go b.read(fromChromeR, dir)
	return b, nil
}

// read reads the messages of Chrome: answers go to whoever waits for them, events to the events
func (b *Browser) read(r *os.File, dir string) {
	defer close(b.done)
	defer os.RemoveAll(dir)
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, 64<<20)
	sc.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexByte(data, 0); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF && len(data) > 0 {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	for sc.Scan() {
		var m cdpMessage
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		if m.ID != 0 {
			b.mu.Lock()
			ch := b.wait[m.ID]
			delete(b.wait, m.ID)
			b.mu.Unlock()
			if ch != nil {
				ch <- m
			}
			continue
		}
		select {
		case b.events <- m:
		default: // nobody listens: dropped
		}
	}
}

// call sends a command (to a page when session is set) and returns its result
func (b *Browser) call(session, method string, params any) (json.RawMessage, error) {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	ch := make(chan cdpMessage, 1)
	b.wait[id] = ch
	msg := map[string]any{"id": id, "method": method, "params": params}
	if session != "" {
		msg["sessionId"] = session
	}
	data, _ := json.Marshal(msg)
	_, err := b.in.Write(append(data, 0))
	b.mu.Unlock()
	if err != nil {
		return nil, err
	}
	select {
	case m := <-ch:
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, m.Error.Message)
		}
		return m.Result, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("%s: no answer", method)
	}
}

// Tab is a page open in the browser
type Tab struct {
	t       *testing.T
	b       *Browser
	target  string
	session string
	mu      sync.Mutex
	errors  []string // exceptions and console errors of the page
	stop    chan struct{}
}

// Open opens a page in a new tab and waits for it to build (until ready is true, a JS expression);
// the tab closes at the end of the test, failing it on any error of the page
func (b *Browser) Open(t *testing.T, url, ready string) *Tab {
	t.Helper()
	res, err := b.call("", "Target.createTarget", map[string]any{"url": "about:blank"})
	if err != nil {
		t.Fatal(err)
	}
	var created struct{ TargetID string }
	_ = json.Unmarshal(res, &created)
	res, err = b.call("", "Target.attachToTarget", map[string]any{"targetId": created.TargetID, "flatten": true})
	if err != nil {
		t.Fatal(err)
	}
	var attached struct{ SessionID string }
	_ = json.Unmarshal(res, &attached)
	tb := &Tab{t: t, b: b, target: created.TargetID, session: attached.SessionID, stop: make(chan struct{})}
	go tb.collect()
	tb.Call("Runtime.enable", nil)
	tb.Call("Page.enable", nil)
	tb.Call("Page.navigate", map[string]any{"url": url})
	// the errors of the page are told at the end, also when it never builds
	t.Cleanup(func() {
		close(tb.stop)
		_, _ = b.call("", "Target.closeTarget", map[string]any{"targetId": tb.target})
		tb.mu.Lock()
		defer tb.mu.Unlock()
		for _, e := range tb.errors {
			t.Errorf("the page: %s", e)
		}
	})
	tb.WaitFor(ready, "the page to build")
	return tb
}

// collect keeps the errors of the page (events of other tabs are left for them)
func (tb *Tab) collect() {
	for {
		select {
		case <-tb.stop:
			return
		case m := <-tb.b.events:
			if m.SessionID != tb.session {
				select {
				case tb.b.events <- m: // another tab's
				default:
				}
				time.Sleep(time.Millisecond)
				continue
			}
			switch m.Method {
			case "Runtime.exceptionThrown":
				var p struct {
					ExceptionDetails struct {
						Text      string
						Exception struct{ Description string }
					}
				}
				_ = json.Unmarshal(m.Params, &p)
				tb.addError(p.ExceptionDetails.Exception.Description + p.ExceptionDetails.Text)
			case "Runtime.consoleAPICalled":
				var p struct {
					Type string
					Args []struct {
						Value       any
						Description string
					}
				}
				_ = json.Unmarshal(m.Params, &p)
				if p.Type == "error" {
					var parts []string
					for _, a := range p.Args {
						parts = append(parts, fmt.Sprint(a.Value, a.Description))
					}
					tb.addError("console: " + strings.Join(parts, " "))
				}
			}
		}
	}
}

func (tb *Tab) addError(e string) {
	tb.mu.Lock()
	tb.errors = append(tb.errors, e)
	tb.mu.Unlock()
}

// Call sends a DevTools command to the page and returns its result
func (tb *Tab) Call(method string, params any) json.RawMessage {
	tb.t.Helper()
	if params == nil {
		params = map[string]any{}
	}
	res, err := tb.b.call(tb.session, method, params)
	if err != nil {
		tb.t.Fatal(err)
	}
	return res
}

// Eval returns the value of a JS expression in the page
func (tb *Tab) Eval(expr string) any {
	tb.t.Helper()
	res := tb.Call("Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true})
	var r struct {
		Result struct {
			Value any
		}
		ExceptionDetails *struct {
			Text      string
			Exception struct{ Description string }
		}
	}
	_ = json.Unmarshal(res, &r)
	if r.ExceptionDetails != nil {
		tb.t.Fatalf("evaluating %s: %s %s", expr, r.ExceptionDetails.Text, r.ExceptionDetails.Exception.Description)
	}
	return r.Result.Value
}

// WaitFor waits until a JS expression is true in the page
func (tb *Tab) WaitFor(expr, what string) {
	tb.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v, _ := tb.Eval("!!(" + expr + ")").(bool); v {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	tb.t.Fatalf("waited for %s: %s", what, expr)
}

// Eventually waits until a Go condition holds (the server side of a tap)
func Eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("waited for %s", what)
}

// Tap clicks the middle of the element a JS expression gives, as a mouse would
func (tb *Tab) Tap(expr string) {
	tb.t.Helper()
	pos, ok := tb.Eval("(() => { const e = " + expr + "; if (!e) return null; const r = e.getBoundingClientRect(); return [r.x + r.width / 2, r.y + r.height / 2]; })()").([]any)
	if !ok || len(pos) != 2 {
		tb.t.Fatalf("nothing to tap: %s", expr)
	}
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		tb.Call("Input.dispatchMouseEvent", map[string]any{"type": typ, "x": pos[0], "y": pos[1], "button": "left", "clickCount": 1})
	}
}

// Key presses a key on the element focused
func (tb *Tab) Key(key string) {
	tb.t.Helper()
	codes := map[string]int{"ArrowUp": 38, "ArrowDown": 40, " ": 32, "Escape": 27}
	for _, typ := range []string{"keyDown", "keyUp"} {
		tb.Call("Input.dispatchKeyEvent", map[string]any{"type": typ, "key": key, "windowsVirtualKeyCode": codes[key]})
	}
}

// Main runs the tests of a package and stops the browser after them: call it from TestMain
func Main(m *testing.M) {
	code := m.Run()
	if b := sharedBrowser; b != nil {
		exited := make(chan struct{})
		go func() { _ = b.cmd.Wait(); close(exited) }()
		_, _ = b.call("", "Browser.close", nil) // Chrome closes itself and its helpers
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = b.cmd.Process.Kill()
			<-exited
		}
		<-b.done
	}
	os.Exit(code)
}

// Serve serves a handler (a panel's) for a test and returns its address: the server closes at the
// end of the test, the open pages' event streams with it
func Serve(t *testing.T, h http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	return srv.URL + "/"
}

// Mouse sends a mouse event at a point of the page: "mousePressed", "mouseMoved", "mouseReleased",
// "mouseWheel" (extra: "deltaY"...); the left button is down unless extra says otherwise
func (tb *Tab) Mouse(kind string, x, y float64, extra map[string]any) {
	tb.t.Helper()
	m := map[string]any{"type": kind, "x": x, "y": y, "button": "left", "buttons": 1, "clickCount": 1}
	if kind == "mouseReleased" {
		m["buttons"] = 0
	}
	for k, v := range extra {
		m[k] = v
	}
	tb.Call("Input.dispatchMouseEvent", m)
}

// Center returns the middle of the element a JS expression gives (x, y in the page's viewport), at
// a part of its height (0.5 the middle)
func (tb *Tab) Center(expr string, height float64) (float64, float64) {
	tb.t.Helper()
	pos, ok := tb.Eval(fmt.Sprintf("(() => { const e = %s; if (!e) return null; const r = e.getBoundingClientRect(); return [r.x + r.width / 2, r.y + r.height * %g]; })()", expr, height)).([]any)
	if !ok || len(pos) != 2 {
		tb.t.Fatalf("no element: %s", expr)
	}
	return pos[0].(float64), pos[1].(float64)
}

// Size sets the size of the page's viewport
func (tb *Tab) Size(width, height int) {
	tb.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": width, "height": height, "deviceScaleFactor": 1, "mobile": false})
}

// Screenshot writes a picture of the whole page as a PNG file (for looking at a test by eye)
func (tb *Tab) Screenshot(path string) {
	tb.t.Helper()
	res := tb.Call("Page.captureScreenshot", map[string]any{"format": "png", "captureBeyondViewport": true})
	var r struct{ Data string }
	_ = json.Unmarshal(res, &r)
	png, err := base64.StdEncoding.DecodeString(r.Data)
	if err == nil {
		err = os.WriteFile(path, png, 0o644)
	}
	if err != nil {
		tb.t.Fatal(err)
	}
}

// Engine is an audio engine for page tests: it asks for a buffer every millisecond on a goroutine
// of its own, as a sound card would, so a patch plays while a test drives its page, and changes
// made with Psalm.Do wait for the gap between two buffers as they do in a program:
//
//	p := psalm.New(paneltest.NewEngine(t))
//	_ = p.AudioStart()
type Engine struct {
	cb   func(in, out [][]float32)
	stop chan struct{}
	done chan struct{}
	mu   sync.Mutex
}

// NewEngine returns an engine at 44.1 kHz in buffers of 256, stopped at the end of the test
func NewEngine(t *testing.T) *Engine {
	e := &Engine{}
	t.Cleanup(func() { _ = e.Stop() })
	return e
}

func (e *Engine) GetSampleRate() float64                   { return 44100 }
func (e *Engine) GetBufferSize() int                       { return 256 }
func (e *Engine) SetCallback(cb func(in, out [][]float32)) { e.cb = cb }
func (e *Engine) Init() error                              { return nil }
func (e *Engine) Close() error                             { return e.Stop() }

// Start asks for buffers until Stop
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stop != nil {
		return nil
	}
	e.stop, e.done = make(chan struct{}), make(chan struct{})
	in := [][]float32{make([]float32, 256), make([]float32, 256)}
	out := [][]float32{make([]float32, 256), make([]float32, 256)}
	go func(stop, done chan struct{}) {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			e.cb(in, out)
			time.Sleep(time.Millisecond)
		}
	}(e.stop, e.done)
	return nil
}

// Stop stops asking for buffers
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stop == nil {
		return nil
	}
	close(e.stop)
	<-e.done
	e.stop = nil
	return nil
}
