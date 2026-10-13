package widgets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

// receiver keeps the messages it gets
type receiver struct {
	*psalm.Module
	msgs []psalm.Message
}

func newReceiver() *receiver {
	r := &receiver{Module: psalm.NewModule(nil, []int{psalm.Stereo})}
	r.SetSelf(r)
	return r
}
func (r *receiver) ReceiveMessage(msg psalm.Message) { r.msgs = append(r.msgs, msg) }
func (r *receiver) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		r.ReceiveMessage(m)
	}
}

func post(t *testing.T, url, body string) {
	t.Helper()
	res, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("%s %s: %v %v", url, body, err, res.Status)
	}
}

func TestWidgets(t *testing.T) {
	ctl := panel.New(psalm.New(nil), "Test")
	synth, kit := newReceiver(), newReceiver()
	x, y := XY(ctl, "Filter", Axis{Name: "cutoff", Min: 100, Max: 8000, Curve: panel.Exp, To: panel.Message(synth, "cutoff"), Default: 1000},
		Axis{Name: "resonance", Max: 1, To: panel.Message(synth, "resonance"), Default: 0.3})
	Keys(ctl, "Keys", Play(synth))
	Pads(ctl, "Pads", Hit(kit, "kick", "snare"))
	env := Envelope(ctl, "Amp", synth, "amp", ADSR{Attack: 10, Decay: 200, Sustain: 0.5, Release: 300}, panel.NoDefault())
	Scope(ctl, "Scope", kit)
	srv := httptest.NewServer(ctl.Handler())
	defer srv.Close()
	tick := int64(0)
	run := func() { tick += 512; ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: tick}) }

	// the description: the kinds, the y axis and the envelope's stages as parts, one plugin
	var d struct {
		Plugins  []struct{ Name string }
		Sections []struct {
			Controls []struct {
				Kind  string
				Name  string
				Parts []struct{ Name string }
			}
		}
	}
	res, _ := http.Get(srv.URL + "/api/panel")
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, c := range d.Sections[0].Controls {
		kinds = append(kinds, c.Kind)
	}
	if strings.Join(kinds, " ") != "xy keys pads envelope scope" || len(d.Plugins) != 1 {
		t.Fatalf("kinds %v, plugins %v", kinds, d.Plugins)
	}
	if p := d.Sections[0].Controls[0].Parts; len(p) != 1 || p[0].Name != "Filter y" {
		t.Errorf("the xy's parts %v", p)
	}
	if p := d.Sections[0].Controls[3].Parts; len(p) != 3 {
		t.Errorf("the envelope's parts %v", p)
	}

	// the defaults go out, but not the envelope's (NoDefault)
	run()
	sent := map[string]float64{}
	for _, m := range synth.msgs {
		for k, v := range m {
			if f, ok := v.(float64); ok {
				sent[k] = f
			}
		}
	}
	if sent["cutoff"] != 1000 || sent["resonance"] != 0.3 {
		t.Errorf("the xy's defaults: %v", sent)
	}
	if _, ok := sent["ampAttack"]; ok {
		t.Errorf("an envelope with NoDefault sent its default")
	}
	synth.msgs = nil

	// moving: the y part by its name, the envelope's sustain
	post(t, srv.URL+"/api/set", `{"name": "Filter y", "value": 0.8}`)
	post(t, srv.URL+"/api/set", `{"name": "Amp sustain", "value": 0.25}`)
	run()
	if y.Value() != 0.8 || x.Value() != 1000 || env.Parts()[1].Value() != 0.25 {
		t.Errorf("values: y %g, x %g, sustain %g", y.Value(), x.Value(), env.Parts()[1].Value())
	}
	got := map[string]float64{}
	for _, m := range synth.msgs {
		for k, v := range m {
			if f, ok := v.(float64); ok {
				got[k] = f
			}
		}
	}
	if got["resonance"] != 0.8 || got["ampSustain"] != 0.25 {
		t.Errorf("sent %v", got)
	}
	synth.msgs = nil

	// the keys play and let go, the pads hit
	post(t, srv.URL+"/api/message", `{"name": "Keys", "data": {"note": 64, "velocity": 0.5, "on": true}}`)
	post(t, srv.URL+"/api/message", `{"name": "Keys", "data": {"note": 64, "on": false}}`)
	post(t, srv.URL+"/api/message", `{"name": "Pads", "data": {"pad": 1, "velocity": 0.9}}`)
	run()
	if len(synth.msgs) != 2 || synth.msgs[0]["noteOn"] != "keys-64" || synth.msgs[0].MustMsg("message").MustFltVal("pitch") != 64 || synth.msgs[1]["noteOff"] != "keys-64" {
		t.Errorf("the keys sent %v", synth.msgs)
	}
	if len(kit.msgs) != 1 || kit.msgs[0].MustMsg("message")["sound"] != "snare" {
		t.Errorf("the pads sent %v", kit.msgs)
	}
}

func TestScopeTap(t *testing.T) {
	m := newReceiver()
	out := m.Output(0)
	for i := range out.Channels[0] {
		out.Channels[0][i] = 0.5
		out.Channels[1][i] = -0.1
	}
	tp := &tap{module: m}
	for range 8 {
		tp.take()
	}
	st := tp.state().(scope)
	if len(st.Samples) != scopeSize || st.Samples[scopeSize-1] != 0.2 || st.SampleRate == 0 {
		t.Errorf("the scope %d samples, the last %g, at %g Hz", len(st.Samples), st.Samples[scopeSize-1], st.SampleRate)
	}
}
