package dxeditor

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/synths/dx"
)

func TestMain(m *testing.M) { paneltest.Main(m) }

// the algorithm stepped on the page, one of your own made from it (a modulation added in the grid),
// a patch of the bank loaded: the synth plays each
func TestPageEditor(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(nil)
	start := dx.Presets["bellPad"]()
	synth := dx.NewPoly(start, 2, 2)
	synth.AddTo(p)
	ctl := panel.New(p, "Test")
	other := dx.Presets["woodBass"]()
	other.Name = "WOOD"
	Add(ctl, "DX", synth, start, Bank([]dx.Patch{start, other}))
	var ts int64
	tick := func() {
		ts += int64(synth.Config().BufferSize)
		ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: ts})
	}
	voice := func() dx.Patch {
		var v dx.Patch
		synth.CallVoices(func(x *dx.Voice) { v = x.Patch() })
		return v
	}
	tick()
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".dx-op-row").length === 7 && document.querySelector(".dx-alg .dx-op")`)
	tb.Size(1200, 1000)

	alg := start.Algorithm
	tb.Tap(`[...document.querySelectorAll(".dx-step")].find(b => b.textContent === "▶")`)
	paneltest.Eventually(t, "the next algorithm", func() bool { tick(); return voice().Algorithm == (alg+1)%32 })
	if n := tb.Eval(`document.querySelectorAll(".dx-alg .dx-op").length`); n != 6.0 {
		t.Errorf("the diagram has %v operators", n)
	}

	// my own: from the algorithm shown, and 6 into 1 added in the grid
	tb.Tap(`[...document.querySelectorAll(".dx button")].find(b => b.textContent === "my own")`)
	tb.WaitFor(`document.querySelector(".dx.custom .dx-routing")`, "the routing grid")
	tb.Tap(`[...document.querySelectorAll(".dx-cell")].find(c => c.title === "operator 6 modulates operator 1")`)
	paneltest.Eventually(t, "my own algorithm with 6 into 1", func() bool {
		tick()
		r := voice().Routing
		if r == nil {
			return false
		}
		for _, m := range r.Mods {
			if m == [2]int{5, 0} {
				return true
			}
		}
		return false
	})

	// a patch of the bank
	tb.Eval(`(() => { const s = document.querySelector(".dx-head select.dx-bank"); s.value = "1"; s.dispatchEvent(new Event("change")); })()`)
	paneltest.Eventually(t, "the bank's patch", func() bool {
		tick()
		v := voice()
		return v.Algorithm == other.Algorithm && v.Ops[0].Coarse == other.Ops[0].Coarse && v.Routing == nil
	})
	tb.WaitFor(`document.querySelector(".dx-name").textContent === "WOOD"`, "its name")
}
