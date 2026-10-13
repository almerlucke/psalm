package patcher_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

func TestMain(m *testing.M) { paneltest.Main(m) }

// on the page: a module added from the palette, a cable drawn by dragging from an output to an
// input, a mod cable to a setting chosen below, a cable and a module taken away, the patch kept
func TestPagePatcher(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t)) // it plays, as in a program
	ctl := panel.New(p, "Patcher")
	dir := t.TempDir()
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(dir))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	syn, err := pt.Add("subtractive", "lead", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1300, 900)

	// add a chorus from the palette
	tb.Eval(`(() => { const s = document.querySelector(".pt-kind"); s.value = "chorus"; s.dispatchEvent(new Event("change")); [...document.querySelectorAll(".pt-bar button")].find(b => b.textContent === "Add").click(); })()`)
	paneltest.Eventually(t, "the chorus added", func() bool { return len(pt.Nodes()) == 3 })
	tb.WaitFor(`document.querySelectorAll(".pt-node").length === 3`, "its module on the page")
	var chorus string
	for _, n := range pt.Nodes() {
		if n.Kind == "chorus" {
			chorus = n.ID
		}
	}
	// move it away from the others by its head
	hx, hy := tb.Center(fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, chorus), 0.5)
	tb.Mouse("mousePressed", hx, hy, nil)
	tb.Mouse("mouseMoved", hx+300, hy+150, nil)
	tb.Mouse("mouseReleased", hx+300, hy+150, nil)
	paneltest.Eventually(t, "the chorus moved", func() bool {
		for _, n := range pt.Nodes() {
			if n.ID == chorus {
				return n.X > 250
			}
		}
		return false
	})

	// a cable: the synth's output to the chorus's input, dragged
	dot := func(id, dir string, i int) (float64, float64) {
		return tb.Center(fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-port[data-dir='%s'][data-i='%d'] .pt-dot")`, id, dir, i), 0.5)
	}
	drag := func(x1, y1, x2, y2 float64) {
		tb.Mouse("mousePressed", x1, y1, nil)
		for i := 1; i <= 5; i++ {
			tb.Mouse("mouseMoved", x1+(x2-x1)*float64(i)/5, y1+(y2-y1)*float64(i)/5, nil)
		}
		tb.Mouse("mouseReleased", x2, y2, nil)
	}
	x1, y1 := dot(syn, "out", 0)
	x2, y2 := dot(chorus, "in", 0)
	drag(x1, y1, x2, y2)
	paneltest.Eventually(t, "the audio cable", func() bool { return len(pt.Cables()) == 1 })
	x1, y1 = dot(chorus, "out", 0)
	x2, y2 = dot("out", "in", 0)
	drag(x1, y1, x2, y2)
	paneltest.Eventually(t, "the cable to the output", func() bool { return len(pt.Cables()) == 2 })
	tb.WaitFor(`document.querySelectorAll("path.pt-cable.audio").length === 2`, "the cables drawn")
	// a second cable from an output that has one already: its port is over the cable's end
	x1, y1 = dot(syn, "out", 0)
	x2, y2 = dot("out", "in", 0)
	drag(x1, y1, x2, y2)
	paneltest.Eventually(t, "a second cable from the same output", func() bool { return len(pt.Cables()) == 3 })
	for _, c := range pt.Cables() {
		if c.From == syn && c.To == "out" {
			pt.Disconnect(c.ID)
		}
	}
	paneltest.Eventually(t, "back to two", func() bool { return len(pt.Cables()) == 2 })

	// an LFO, a mod cable to the synth: its setting chosen below the patch
	lfo, err := pt.Add("lfo", "", 60, 400)
	if err != nil {
		t.Fatal(err)
	}
	tb.WaitFor(fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s']")`, lfo), "the LFO on the page")
	x1, y1 = dot(lfo, "out", 0)
	x2, y2 = dot(syn, "in", 1) // its mod input
	drag(x1, y1, x2, y2)
	tb.WaitFor(`document.querySelector(".pt-inspector .pt-setting")`, "the mod cable's setting to choose")
	tb.Eval(`(() => { const s = document.querySelector(".pt-inspector .pt-setting"); s.value = "cutoff"; s.dispatchEvent(new Event("change")); })()`)
	paneltest.Eventually(t, "the mod cable on the cutoff", func() bool {
		for _, c := range pt.Cables() {
			if c.Type == patcher.Mod && c.Key == "cutoff" {
				return true
			}
		}
		return false
	})
	// a cable taken away: picked, then its button
	tb.Eval(`document.querySelector("path.pt-cable.mod").dispatchEvent(new PointerEvent("pointerdown", {bubbles: true}))`)
	tb.Tap(`[...document.querySelectorAll(".pt-inspector button")].find(b => b.textContent === "Take it away")`)
	paneltest.Eventually(t, "the mod cable gone", func() bool { return len(pt.Cables()) == 2 })
	// a module taken away, its cables with it
	tb.Tap(fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-x")`, lfo))
	paneltest.Eventually(t, "the LFO gone", func() bool { return len(pt.Nodes()) == 3 })

	// a generator's text typed on its module: a good pattern taken, a bad one shown
	clk, err := pt.Add("clock", "", 400, 380)
	if err != nil {
		t.Fatal(err)
	}
	input := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-text input")`, clk)
	tb.WaitFor(input, "the clock's pattern on its module")
	tb.Eval(`(() => { const i = ` + input + `; i.value = "x . x x"; i.dispatchEvent(new Event("change")); })()`)
	paneltest.Eventually(t, "the pattern taken", func() bool { tx := pt.Texts(clk); return len(tx) == 1 && tx[0].Value == "x . x x" })
	tb.Eval(`(() => { const i = ` + input + `; i.value = "x ?"; i.dispatchEvent(new Event("change")); })()`)
	tb.WaitFor(input+`.classList.contains("bad")`, "a bad pattern shown")
	if os.Getenv("SHOT") != "" {
		tb.Screenshot(os.Getenv("SHOT"))
	}

	// kept by name
	tb.Eval(`(() => { const i = document.querySelector(".pt-name"); i.value = "first"; [...document.querySelectorAll(".pt-bar button")].find(b => b.textContent === "Save").click(); })()`)
	paneltest.Eventually(t, "the patch kept", func() bool { _, err := os.Stat(dir + "/first.json"); return err == nil })
	tb.WaitFor(`[...document.querySelectorAll(".pt-patches option")].some(o => o.value === "first")`, "the patch listed")
}
