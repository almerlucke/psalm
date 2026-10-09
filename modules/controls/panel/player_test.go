package panel

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type fakePlayer struct {
	running atomic.Bool
	fail    bool
}

func (f *fakePlayer) AudioStart() error {
	if f.fail {
		return errors.New("no sound card")
	}
	f.running.Store(true)
	return nil
}
func (f *fakePlayer) AudioStop() error { f.running.Store(false); return nil }
func (f *fakePlayer) Running() bool    { return f.running.Load() }

// restartable is a player that goes back to the start
type restartable struct {
	fakePlayer
	restarts int
}

func (r *restartable) Restart() { r.restarts++ }

func TestPlayer(t *testing.T) {
	pn, _ := newPanel()
	if pn.Player() != nil {
		t.Fatal("a patch that renders offline has no player")
	}
	pn.Meter("Master", nil)
	srv := httptest.NewServer(pn.Handler())
	defer srv.Close()
	if res := postJSON(t, srv.URL+"/api/audio", `{"running": true}`); res.StatusCode != 404 {
		t.Fatalf("without a player: 404, got %d", res.StatusCode)
	}

	pl := &fakePlayer{}
	pl.running.Store(true)
	pn.SetPlayer(pl)
	if d := pn.describe(); !d.Player || !d.Running {
		t.Fatal("the page should know it can stop the sound")
	}

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
	next := func() update {
		for {
			select {
			case u := <-lines:
				if u.Audio != nil {
					return u
				}
			case <-time.After(2 * time.Second):
				t.Fatal("no audio update")
			}
		}
	}
	if u := next(); !u.Audio.Running {
		t.Fatal("the first update should say it plays")
	}

	res := postJSON(t, srv.URL+"/api/audio", `{"running": false}`)
	var got map[string]bool
	_ = json.NewDecoder(res.Body).Decode(&got)
	res.Body.Close()
	if pl.Running() || got["running"] {
		t.Fatal("the page should stop the sound")
	}
	if u := next(); u.Audio.Running || u.Meters["Master"] != [2]float64{-120, -120} {
		t.Fatalf("the page should hear it stopped, the meters silent: %+v", u)
	}

	pl.fail = true
	if res := postJSON(t, srv.URL+"/api/audio", `{"running": true}`); res.StatusCode != 500 {
		t.Fatalf("an engine that fails to start should say so, got %d", res.StatusCode)
	}
	if d := pn.describe(); d.Restart {
		t.Fatal("a player without Restart has no restart button")
	}
	if res := postJSON(t, srv.URL+"/api/audio", `{"restart": true}`); res.StatusCode != 404 {
		t.Fatalf("restarting a player that can't should be 404, got %d", res.StatusCode)
	}
	rp := &restartable{}
	pn.SetPlayer(rp)
	if !pn.describe().Restart {
		t.Fatal("a player with Restart should have the button")
	}
	if res := postJSON(t, srv.URL+"/api/audio", `{"restart": true}`); res.StatusCode != 200 || rp.restarts != 1 || rp.Running() {
		t.Fatal("restart should go back to the start, and leave the sound as it was (stopped)")
	}
	pn.SetPlayer(nil)
	if pn.describe().Player {
		t.Fatal("SetPlayer(nil) should take the button away")
	}
}
