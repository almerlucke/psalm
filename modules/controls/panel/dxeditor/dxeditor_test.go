package dxeditor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/synths/dx"
)

func TestEditor(t *testing.T) {
	p := psalm.New(nil)
	start := dx.Presets["bellPad"]()
	synth := dx.NewPoly(start, 2, 2)
	synth.AddTo(p)
	ctl := panel.New(p, "Test")
	other := dx.Presets["woodBass"]()
	other.Name = "WOOD"
	e := Add(ctl, "DX", synth, start, Bank([]dx.Patch{start, other}))
	srv := httptest.NewServer(ctl.Handler())
	defer srv.Close()
	var ts int64
	tick := func() {
		for range 2 { // a buffer: the values reach the editor, the next applies them
			ts += int64(synth.Config().BufferSize)
			ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: ts})
		}
	}
	set := func(name string, v string) {
		res, err := http.Post(srv.URL+"/api/set", "application/json", strings.NewReader(`{"name": "`+name+`", "value": `+v+`}`))
		if err != nil || res.StatusCode != 200 {
			t.Fatalf("set %s: %v %v", name, err, res.Status)
		}
	}
	voice := func() dx.Patch {
		var got dx.Patch
		synth.CallVoices(func(v *dx.Voice) { got = v.Patch() })
		return got
	}
	tick()
	if v := voice(); v.Algorithm != start.Algorithm || v.Ops[2].Coarse != start.Ops[2].Coarse {
		t.Errorf("the synth should play the patch it started with")
	}

	// a setting of an operator, the algorithm
	set("DX op3Coarse", "7")
	set("DX", "17")
	tick()
	if v := voice(); v.Ops[2].Coarse != 7 || v.Algorithm != 16 || v.Routing != nil {
		t.Errorf("edited: coarse %d, algorithm %d", v.Ops[2].Coarse, v.Algorithm+1)
	}

	// an algorithm of your own: 6 into 1, 5 into 1, carriers 1 and 2, feedback on 5
	for _, c := range []struct{ name, v string }{{"DX carrier1", "1"}, {"DX carrier2", "1"}, {"DX carrier3", "0"},
		{"DX carrier4", "0"}, {"DX carrier5", "0"}, {"DX carrier6", "0"}, {"DX fbFrom", "5"}, {"DX fbTo", "5"}} {
		set(c.name, c.v)
	}
	for from := 2; from <= 6; from++ {
		for to := 1; to < from; to++ {
			on := "0"
			if to == 1 && (from == 6 || from == 5) {
				on = "1"
			}
			set("DX mod"+string(rune('0'+from))+string(rune('0'+to)), on)
		}
	}
	set("DX custom", "1")
	tick()
	r := voice().Routing
	if r == nil || len(r.Carriers) != 2 || len(r.Mods) != 2 || r.FeedbackFrom != 4 {
		t.Fatalf("my own algorithm: %+v", r)
	}
	set("DX custom", "0")
	tick()
	if voice().Routing != nil {
		t.Errorf("custom off: the algorithm again")
	}

	// loading a patch of the bank moves every setting to it
	set("DX program", "1")
	tick()
	tick()
	if v := voice(); v.Ops[0].Coarse != other.Ops[0].Coarse || v.Algorithm != other.Algorithm || v.Ops[1].EGRates != other.Ops[1].EGRates {
		t.Errorf("loaded: %+v", v.Ops[0])
	}
	if e.Setting("op1Coarse").Value() != float64(other.Ops[0].Coarse) || e.patchName() != "WOOD" {
		t.Errorf("the controls didn't follow the patch loaded")
	}
}
