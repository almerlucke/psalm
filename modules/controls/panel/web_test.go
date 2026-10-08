package panel

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testServer(t *testing.T) (*Panel, *httptest.Server) {
	pn, _ := newPanel()
	pn.Section("Filter")
	pn.Fader("Cutoff", 200, 12000, Exp, Func(func(float64) {}), Default(1000), Unit("Hz"))
	pn.Section("Play")
	pn.Toggle("Hold", Func(func(float64) {}))
	pn.Button("Bang", func() {})
	pn.Select("Preset", []string{"warm", "bright"}, func(int) {})
	srv := httptest.NewServer(pn.Handler())
	t.Cleanup(srv.Close)
	return pn, srv
}

func postJSON(t *testing.T, url string, body string) *http.Response {
	t.Helper()
	res, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestPage(t *testing.T) {
	_, srv := testServer(t)
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("the page should be served, got %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
}

func TestDescribe(t *testing.T) {
	_, srv := testServer(t)
	res, err := http.Get(srv.URL + "/api/panel")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var d description
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		t.Fatal(err)
	}
	if d.Title != "Test" || len(d.Sections) != 2 || d.Sections[0].Name != "Filter" || len(d.Sections[1].Controls) != 3 {
		t.Fatalf("unexpected description %+v", d)
	}
	c := d.Sections[0].Controls[0]
	if c.Kind != FaderKind || c.Curve != Exp || c.Value != 1000 || c.Unit != "Hz" {
		t.Fatalf("unexpected fader %+v", c)
	}
	if opts := d.Sections[1].Controls[2].Options; len(opts) != 2 || opts[1] != "bright" {
		t.Fatalf("the select should list its options, got %v", opts)
	}
}

func TestSetAndPress(t *testing.T) {
	pn, srv := testServer(t)
	res := postJSON(t, srv.URL+"/api/set", `{"name": "Cutoff", "value": 99999}`)
	res.Body.Close()
	if res.StatusCode != 200 || pn.Param("Cutoff").Value() != 12000 {
		t.Fatalf("set should clamp and apply: %d, %v", res.StatusCode, pn.Param("Cutoff").Value())
	}
	if res := postJSON(t, srv.URL+"/api/set", `{"name": "Nope", "value": 1}`); res.StatusCode != 404 {
		t.Fatalf("an unknown control should be 404, got %d", res.StatusCode)
	}
	res = postJSON(t, srv.URL+"/api/press", `{"name": "Bang"}`)
	res.Body.Close()
	if res.StatusCode != 204 || pn.Param("Bang").presses.Load() != 1 {
		t.Fatalf("press should count, got %d", res.StatusCode)
	}
	if res := postJSON(t, srv.URL+"/api/set", `not json`); res.StatusCode != 400 {
		t.Fatalf("a bad request should be 400, got %d", res.StatusCode)
	}
}

func TestSnapshotAPI(t *testing.T) {
	pn, srv := testServer(t)
	res := postJSON(t, srv.URL+"/api/snapshot", `{"Cutoff": 4000, "Hold": 1, "Unknown": 3}`)
	res.Body.Close()
	if pn.Param("Cutoff").Value() != 4000 || pn.Param("Hold").Value() != 1 {
		t.Fatal("posting a snapshot should restore it")
	}
	res, err := http.Get(srv.URL + "/api/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var snap map[string]float64
	_ = json.NewDecoder(res.Body).Decode(&snap)
	if snap["Cutoff"] != 4000 || len(snap) != 3 {
		t.Fatalf("unexpected snapshot %v", snap)
	}
}

func TestEvents(t *testing.T) {
	pn, srv := testServer(t)
	res, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	lines := make(chan string)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				lines <- data
			}
		}
	}()
	next := func() update {
		select {
		case line := <-lines:
			var u update
			if err := json.Unmarshal([]byte(line), &u); err != nil {
				t.Fatal(err)
			}
			return u
		case <-time.After(2 * time.Second):
			t.Fatal("no event")
		}
		return update{}
	}
	if u := next(); u.Values["Cutoff"] != 1000 || len(u.Values) != 3 {
		t.Fatalf("the first event should hold every value, got %+v", u)
	}
	pn.Param("Cutoff").Set(3000)
	if u := next(); u.Values["Cutoff"] != 3000 || len(u.Values) != 1 {
		t.Fatalf("then only what changed, got %+v", u)
	}
}

// freeAddr returns an address with a port nothing listens on
func freeAddr(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// Close ends the event streams and frees the address
func TestClose(t *testing.T) {
	pn, _ := newPanel()
	pn.Toggle("X", nil)
	addr := freeAddr(t)
	served := make(chan error, 1)
	go func() { served <- pn.Serve(addr) }()
	var events *http.Response
	for range 100 {
		var err error
		if events, err = http.Get("http://" + addr + "/api/events"); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if events == nil {
		t.Fatal("the panel didn't serve")
	}
	defer events.Body.Close()

	start := time.Now()
	if err := pn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("close should end the event streams at once, it took %v", took)
	}
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("serve should end without an error after close, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve didn't return after close")
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("the address should be free after close: %v", err)
	}
	ln.Close()
}

func TestServeAddressInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	pn, _ := newPanel()
	err = pn.Serve(ln.Addr().String())
	if err == nil || !strings.Contains(err.Error(), "still running") {
		t.Fatalf("a port in use should say what is likely going on, got %v", err)
	}
}

func TestCloseBeforeServe(t *testing.T) {
	pn, _ := newPanel()
	pn.Close()
	done := make(chan error, 1)
	go func() { done <- pn.Serve(freeAddr(t)) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("serving a closed panel should return at once")
	}
}
