package panel

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
)

type mounter struct{ p atomic.Pointer[Panel] }

func (m *mounter) Mounted() *Panel { return m.p.Load() }

func TestMount(t *testing.T) {
	host, _ := newPanel()
	m := &mounter{}
	host.Section("Song")
	host.Mount("Song", m)
	srv := httptest.NewServer(host.Handler())
	defer srv.Close()

	if res, _ := http.Get(srv.URL + "/mount/Song/api/panel"); res.StatusCode != 404 {
		t.Fatalf("nothing mounted: 404, got %d", res.StatusCode)
	}
	song := New(psalm.New(nil), "Morning")
	cutoff := song.Fader("Cutoff", 0, 1, Linear, nil)
	m.p.Store(song)

	var d description
	res, _ := http.Get(srv.URL + "/mount/Song/api/panel")
	_ = json.NewDecoder(res.Body).Decode(&d)
	res.Body.Close()
	if d.Title != "Morning" || d.Sections[0].Controls[0].Name != "Cutoff" {
		t.Fatalf("the mounted panel should be served under the mount: %+v", d)
	}
	if res := postJSON(t, srv.URL+"/mount/Song/api/set", `{"name": "Cutoff", "value": 0.7}`); res.StatusCode != 200 || cutoff.Value() != 0.7 {
		t.Fatalf("the mounted panel's controls should be set through it: %d %v", res.StatusCode, cutoff.Value())
	}
	if res, _ := http.Get(srv.URL + "/mount/Song/"); res.StatusCode != 200 {
		t.Fatal("the mounted panel's page should be there")
	}
	if c := host.describe().Sections[0].Controls[0]; c.Kind != MountKind || c.Mount.Title != "Morning" {
		t.Fatalf("the host should describe the mount: %+v", c)
	}

	// MIDI reaches the panel mounted
	cutoff.BindCC(0, 20)
	host.ControlChange(1, 20, 0)
	if cutoff.Value() != 0 {
		t.Fatalf("a control change should reach the mounted panel: %v", cutoff.Value())
	}

	// the page hears when another panel is mounted
	events, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer events.Body.Close()
	lines := make(chan update)
	go func() {
		sc := bufio.NewScanner(events.Body)
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				var u update
				_ = json.Unmarshal([]byte(data), &u)
				lines <- u
			}
		}
	}()
	next := func() *mountDesc {
		for {
			select {
			case u := <-lines:
				if md := u.Mounts["Song"]; md != nil {
					return md
				}
			case <-time.After(2 * time.Second):
				t.Fatal("no mount update")
			}
		}
	}
	first := next()
	if first.Title != "Morning" {
		t.Fatalf("the first update should name the panel mounted: %+v", first)
	}
	evening := New(psalm.New(nil), "Evening")
	m.p.Store(evening)
	if md := next(); md.Title != "Evening" || md.ID == first.ID {
		t.Fatalf("another panel mounted should be sent, with a new version: %+v", md)
	}
	m.p.Store(nil)
	if md := next(); md.Title != "" {
		t.Fatal("nothing mounted should be sent too")
	}
}
