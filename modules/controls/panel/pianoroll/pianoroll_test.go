package pianoroll

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

func TestPianoRoll(t *testing.T) {
	f := &midi.File{Division: 480}
	var tr midi.Track
	tr.AddMeta(0, midi.MetaTempo, midi.TempoData(60))
	tr.Add(0, midi.NoteOnMsg(1, 60, 100))
	tr.Add(480, midi.NoteOffMsg(1, 60, 0))
	tr.Add(960, midi.NoteOnMsg(10, 36, 90))
	tr.Add(1000, midi.NoteOffMsg(10, 36, 0))
	f.Tracks = []midi.Track{tr}
	player := midi.NewPlayer(f)
	player.MuteChannel(10, true)

	ctl := panel.New(psalm.New(nil), "Test")
	roll := Add(ctl, "Notes", player, Seconds(4))
	srv := httptest.NewServer(ctl.Handler())
	defer srv.Close()

	var d struct {
		Plugins  []struct{ Name string }
		Sections []struct {
			Controls []struct {
				Kind  string
				Data  data
				State state
			}
		}
	}
	res, _ := http.Get(srv.URL + "/api/panel")
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		t.Fatal(err)
	}
	c := d.Sections[0].Controls[0]
	if len(d.Plugins) != 1 || d.Plugins[0].Name != "pianoroll" || c.Kind != "pianoroll" {
		t.Fatalf("described as %+v", d)
	}
	if len(c.Data.Notes) != 2 || c.Data.Notes[0] != [5]float64{0, 1, 60, 1, 100} || c.Data.Seconds != 4 || c.Data.Duration != 2+40.0/480 {
		t.Errorf("data %+v", c.Data)
	}
	if len(c.State.Muted) != 1 || c.State.Muted[0] != 10 {
		t.Errorf("state %+v", c.State)
	}
	// a click seeks the player (on the audio thread: the panel's tick)
	res, err := http.Post(srv.URL+"/api/message", "application/json", strings.NewReader(`{"name": "Notes", "data": {"seek": 1.5}}`))
	if err != nil || res.StatusCode != 200 {
		t.Fatal(err, res.Status)
	}
	ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: 1})
	if pos := player.Position(); pos != 1.5 {
		t.Errorf("seeked to %g, want 1.5", pos)
	}
	_ = roll
}
