package editor

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/params"
)

func TestMain(m *testing.M) { paneltest.Main(m) }

// the cards of a description, every kind of control, the envelopes; moving them reaches the synth;
// a preset of your own saved and loaded back on the page
func TestPageEditor(t *testing.T) {
	b := paneltest.Start(t)
	set := params.Set{Name: "test", Params: []params.Param{
		params.K("Filter", "cutoff", 20, 18000, 800, params.Exp(), params.Unit("Hz")),
		params.S("Filter", "sync", false),
		params.C("Osc", "wave", []string{"sine", "saw", "square"}, 0),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", 5, 200, 0.7, 300),
		{Prefix: "fold", Label: "fold", Group: "Envelopes", Stages: "AR", Attack: 10, Release: 50},
	}}
	ctl := panel.New(psalm.New(nil), "Test")
	r := &receiver{got: map[string]float64{}}
	e := Add(ctl, "Synth", r, set).Factory([]string{"init"}, func(string) params.Set { return set })
	if err := e.Keep(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	tick := func() { ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: 1}) }
	tick()
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".ed-group").length === 3`)
	tb.Size(1100, 900)

	if n := tb.Eval(`document.querySelectorAll(".editor canvas.envelope").length`); n != 1.0 {
		t.Errorf("the ADSR envelope as a graph: %v", n)
	}
	if n := tb.Eval(`document.querySelectorAll(".editor .ed-env-knobs .knob").length`); n != 2.0 {
		t.Errorf("the AR envelope as two knobs: %v", n)
	}
	// a knob: up with the arrow keys
	tb.Eval(`[...document.querySelectorAll(".editor .control")].find(c => c.querySelector(".name") && c.querySelector(".name").textContent === "cutoff").querySelector(".knob").focus()`)
	for range 5 {
		tb.Key("ArrowUp")
	}
	paneltest.Eventually(t, "the cutoff to go up", func() bool { tick(); return r.got["cutoff"] > 800 })
	// a switch, a choice
	tb.Tap(`[...document.querySelectorAll(".editor button")].find(b => b.textContent === "sync")`)
	tb.Eval(`(() => { const s = [...document.querySelectorAll(".editor .ed-group select")][0]; s.value = "2"; s.dispatchEvent(new Event("change")); })()`)
	paneltest.Eventually(t, "the switch and the choice", func() bool { tick(); return r.got["sync"] == 1 && r.got["wave"] == 2 })

	// a preset of your own: saved under a name on the page, listed, loaded back after a change
	tb.Eval(`(() => { const i = document.querySelector(".ed-presets input"); i.value = "mine"; [...document.querySelectorAll(".ed-presets button")].find(b => b.textContent === "Save").click(); })()`)
	paneltest.Eventually(t, "my preset saved", func() bool { tick(); return len(e.Presets().Mine()) == 1 }) // messages reach it on a tick
	tb.WaitFor(`[...document.querySelectorAll(".ed-presets option")].some(o => o.value === "m:mine")`, "my preset listed")
	saved := e.Setting("cutoff").Value()
	e.Setting("cutoff").Set(100)
	tick()
	tb.Eval(`(() => { const s = document.querySelector(".ed-presets select"); s.value = "m:mine"; s.dispatchEvent(new Event("change")); })()`)
	paneltest.Eventually(t, "my preset loaded back", func() bool { tick(); return r.got["cutoff"] == saved })
	tb.WaitFor(`!document.querySelectorAll(".ed-presets button")[1].disabled`, "delete enabled for one of mine")
}

// a setting says what it does when the pointer rests on it, a group has a line under its title, an envelope a hint too
func TestPageEditorHints(t *testing.T) {
	b := paneltest.Start(t)
	set := params.Set{Name: "test", Params: []params.Param{
		params.K("Filter", "cutoff", 20, 18000, 800, params.Exp(), params.Hint("where the sound gets dull")),
		params.C("Filter", "wave", []string{"sine", "saw"}, 0, params.Hint("the wave")),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "Volume", Group: "Filter", Stages: "ADSR", Attack: 5, Decay: 200, Sustain: 0.7, Release: 300, Hint: "how a note goes"},
	}, GroupHints: map[string]string{"Filter": "what the sound is made of"}}
	ctl := panel.New(psalm.New(nil), "Test")
	Add(ctl, "Synth", &receiver{got: map[string]float64{}}, set)
	ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: 1})
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".ed-group").length === 1`)
	tb.Size(1100, 700)
	if got := tb.Eval(`document.querySelector(".ed-group .ed-hint").textContent`); got != "what the sound is made of" {
		t.Errorf("the line under the group: %v", got)
	}
	for _, want := range []string{"where the sound gets dull", "the wave", "how a note goes"} {
		if n := tb.Eval(`[...document.querySelectorAll(".editor [title]")].filter(e => e.title === ` + `"` + want + `"` + `).length`); n != 1.0 {
			t.Errorf("hint %q is on %v elements", want, n)
		}
	}
}
