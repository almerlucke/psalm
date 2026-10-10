package panel

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// The page tests drive the panel's page in headless Chrome, over the DevTools protocol on a pair of
// pipes (--remote-debugging-pipe: Chrome reads commands on its fd 3 and writes on its fd 4, every
// message JSON ending in a 0 byte). They are skipped with -short, and when there is no Chrome
// (PSALM_CHROME names one that is not on the path)

// browser is a headless Chrome
type browser struct {
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
	sharedBrowser *browser
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

// startBrowser returns the browser of the tests, started the first time, or skips the test
func startBrowser(t *testing.T) *browser {
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

func launch(path string) (*browser, error) {
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
	b := &browser{cmd: cmd, in: toChromeW, wait: map[int]chan cdpMessage{}, events: make(chan cdpMessage, 256), done: make(chan struct{})}
	go b.read(fromChromeR, dir)
	return b, nil
}

// read reads the messages of Chrome: answers go to whoever waits for them, events to the events
func (b *browser) read(r *os.File, dir string) {
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
func (b *browser) call(session, method string, params any) (json.RawMessage, error) {
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

// tab is a page open in the browser
type tab struct {
	t       *testing.T
	b       *browser
	target  string
	session string
	mu      sync.Mutex
	errors  []string // exceptions and console errors of the page
	stop    chan struct{}
}

// open opens a page in a new tab and waits for it to build (until ready is true, a JS expression);
// the tab closes at the end of the test, failing it on any error of the page
func (b *browser) open(t *testing.T, url, ready string) *tab {
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
	tb := &tab{t: t, b: b, target: created.TargetID, session: attached.SessionID, stop: make(chan struct{})}
	go tb.collect()
	tb.call("Runtime.enable", nil)
	tb.call("Page.enable", nil)
	tb.call("Page.navigate", map[string]any{"url": url})
	tb.waitFor(ready, "the page to build")
	t.Cleanup(func() {
		close(tb.stop)
		_, _ = b.call("", "Target.closeTarget", map[string]any{"targetId": tb.target})
		tb.mu.Lock()
		defer tb.mu.Unlock()
		for _, e := range tb.errors {
			t.Errorf("the page: %s", e)
		}
	})
	return tb
}

// collect keeps the errors of the page (events of other tabs are left for them)
func (tb *tab) collect() {
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

func (tb *tab) addError(e string) {
	tb.mu.Lock()
	tb.errors = append(tb.errors, e)
	tb.mu.Unlock()
}

func (tb *tab) call(method string, params any) json.RawMessage {
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

// eval returns the value of a JS expression in the page
func (tb *tab) eval(expr string) any {
	tb.t.Helper()
	res := tb.call("Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true})
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

// waitFor waits until a JS expression is true in the page
func (tb *tab) waitFor(expr, what string) {
	tb.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v, _ := tb.eval("!!(" + expr + ")").(bool); v {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	tb.t.Fatalf("waited for %s: %s", what, expr)
}

// eventually waits until a Go condition holds (the server side of a tap)
func eventually(t *testing.T, what string, ok func() bool) {
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

// tap clicks the middle of the element a JS expression gives, as a mouse would
func (tb *tab) tap(expr string) {
	tb.t.Helper()
	pos, ok := tb.eval("(() => { const e = " + expr + "; if (!e) return null; const r = e.getBoundingClientRect(); return [r.x + r.width / 2, r.y + r.height / 2]; })()").([]any)
	if !ok || len(pos) != 2 {
		tb.t.Fatalf("nothing to tap: %s", expr)
	}
	for _, typ := range []string{"mousePressed", "mouseReleased"} {
		tb.call("Input.dispatchMouseEvent", map[string]any{"type": typ, "x": pos[0], "y": pos[1], "button": "left", "clickCount": 1})
	}
}

// key presses a key on the element focused
func (tb *tab) key(key string) {
	tb.t.Helper()
	codes := map[string]int{"ArrowUp": 38, "ArrowDown": 40, " ": 32, "Escape": 27}
	for _, typ := range []string{"keyDown", "keyUp"} {
		tb.call("Input.dispatchKeyEvent", map[string]any{"type": typ, "key": key, "windowsVirtualKeyCode": codes[key]})
	}
}

// TestMain stops the browser after the tests
func TestMain(m *testing.M) {
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
