package editor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/params"
)

// receiver keeps the settings it gets
type receiver struct{ got map[string]float64 }

func (r *receiver) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		for k, x := range m {
			r.got[k] = psalm.FltVal[float64](x)
		}
	}
}

func TestEditor(t *testing.T) {
	set := params.Set{Name: "test", Params: []params.Param{
		params.K("Filter", "cutoff", 20, 18000, 800, params.Exp(), params.Unit("Hz")),
		params.S("Filter", "sync", true),
		params.C("Osc", "wave", []string{"sine", "saw"}, 1),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", 5, 200, 0.7, 300),
		{Prefix: "fold", Label: "fold", Group: "Envelopes", Stages: "AR", Attack: 10, Release: 50},
	}}
	ctl := panel.New(psalm.New(nil), "Test")
	r := &receiver{got: map[string]float64{}}
	e := Add(ctl, "Synth", r, set)
	tick := func() { ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: 1}) }
	tick()
	// the settings start where the description has them, sent to the synth
	for k, want := range map[string]float64{"cutoff": 800, "sync": 1, "wave": 1, "ampAttack": 5, "ampSustain": 0.7, "foldRelease": 50} {
		if r.got[k] != want {
			t.Errorf("%s: got %v, want %v", k, r.got[k], want)
		}
	}
	if _, ok := r.got["foldDecay"]; ok {
		t.Errorf("a stage the envelope doesn't have was sent")
	}
	// moved on the page
	srv := httptest.NewServer(ctl.Handler())
	defer srv.Close()
	if res, err := http.Post(srv.URL+"/api/set", "application/json", strings.NewReader(`{"name": "Synth cutoff", "value": 2000}`)); err != nil || res.StatusCode != 200 {
		t.Fatal(err)
	}
	tick()
	if r.got["cutoff"] != 2000 || e.Setting("cutoff").Value() != 2000 {
		t.Errorf("cutoff: %v", r.got["cutoff"])
	}
	// the description on the page: the groups, the settings, the envelopes
	var d struct {
		Plugins  []struct{ Name string }
		Sections []struct{ Controls []struct{ Data data } }
	}
	res, _ := http.Get(srv.URL + "/api/panel")
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		t.Fatal(err)
	}
	dd := d.Sections[0].Controls[0].Data
	if strings.Join(dd.Groups, ",") != "Filter,Osc,Envelopes" || len(dd.Params) != 3 || dd.Params[1].Kind != "switch" || len(dd.Envelopes) != 2 {
		t.Errorf("data %+v", dd)
	}
	if len(d.Plugins) != 2 {
		t.Errorf("plugins %v: the widgets (envelopes) and the editor", d.Plugins)
	}
	// loading another setting of it
	other := set
	other.Params = append([]params.Param(nil), set.Params...)
	other.Params[0].Value = 300
	e.Load(other)
	tick()
	if r.got["cutoff"] != 300 {
		t.Errorf("loaded: cutoff %v", r.got["cutoff"])
	}
}

func TestPresets(t *testing.T) {
	set := params.Set{Name: "test", Params: []params.Param{
		params.K("Filter", "cutoff", 20, 18000, 800, params.Exp()),
		params.K("Filter", "resonance", 0, 1, 0.2),
	}, Envelopes: []params.Envelope{params.E("Envelopes", "amp", "amp", 5, 200, 0.7, 300)}}
	factory := map[string]float64{"dark": 300, "bright": 9000}
	describe := func(name string) params.Set {
		s := set
		s.Params = append([]params.Param(nil), set.Params...)
		s.Params[0].Value = factory[name]
		return s
	}
	ctl := panel.New(psalm.New(nil), "Test")
	r := &receiver{got: map[string]float64{}}
	e := Add(ctl, "Synth", r, set).Factory([]string{"bright", "dark"}, describe)
	dir := t.TempDir()
	if err := e.Keep(dir); err != nil {
		t.Fatal(err)
	}
	tick := func() { ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: 1}) }
	srv := httptest.NewServer(ctl.Handler())
	defer srv.Close()
	message := func(body string) {
		t.Helper()
		if res, err := http.Post(srv.URL+"/api/message", "application/json", strings.NewReader(`{"name": "Synth", "data": `+body+`}`)); err != nil || res.StatusCode != 200 {
			t.Fatal(err)
		}
		tick()
	}
	tick()

	// your own: the settings as they are, saved, loaded back after a change
	e.Setting("cutoff").Set(1234)
	e.Setting("ampRelease").Set(2000)
	tick()
	message(`{"save": "my lead"}`)
	waitFor(t, func() bool { return len(e.Presets().Mine()) == 1 })
	e.Setting("cutoff").Set(500)
	e.Setting("ampRelease").Set(10)
	tick()
	message(`{"load": "my lead", "mine": true}`)
	tick()
	if r.got["cutoff"] != 1234 || r.got["ampRelease"] != 2000 {
		t.Errorf("my preset loaded back: cutoff %v, release %v", r.got["cutoff"], r.got["ampRelease"])
	}
	// a factory one
	message(`{"load": "dark"}`)
	tick()
	if r.got["cutoff"] != 300 {
		t.Errorf("the factory preset: cutoff %v", r.got["cutoff"])
	}
	st := e.Presets().state().(state)
	if st.Current != "dark" || !st.Keeps || len(st.Factory) != 2 || len(st.Mine) != 1 {
		t.Errorf("state %+v", st)
	}
	// a name that can't be a file
	message(`{"save": "../escape"}`)
	waitFor(t, func() bool { return e.Presets().state().(state).Note != "" })
	// kept on disk: another editor reads them
	other := Add(panel.New(psalm.New(nil), "Other"), "Synth", r, set)
	if err := other.Keep(dir); err != nil || len(other.Presets().Mine()) != 1 {
		t.Fatalf("read back: %v %v", err, other.Presets().Mine())
	}
	message(`{"delete": "my lead"}`)
	waitFor(t, func() bool { return len(e.Presets().Mine()) == 0 })
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	for range 200 {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("it didn't happen")
}
