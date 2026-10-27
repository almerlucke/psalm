package patcher_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/effects/gain"
)

func newPage(t *testing.T) (*paneltest.Browser, *patcher.Patcher, string) {
	t.Helper()
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(t.TempDir()))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	return b, pt, paneltest.Serve(t, ctl.Handler())
}

// a double click on a module opens its window inside the page (a frame of module.html, no window of the browser: it has no address bar): its texts, its settings, what is
// wrong with what is typed; it follows the patch; it is closed by its x or Esc and when its module goes; one for each module, a double click on the module again brings its
// window to the front
func TestPageModuleWindow(t *testing.T) {
	b, pt, base := newPage(t)
	clk, err := pt.Add("clock", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, base, `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1300, 900)
	head := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, clk)
	doubleTap(tb, head)
	tb.WaitFor(`document.querySelectorAll(".pt-win").length === 1 && document.querySelector(".pt-win-title").textContent === "clock"`, "the window of the clock")
	tb.WaitFor(`W().querySelector(".mw-text").value === "x"`, "the pattern of the clock in its window")
	tb.WaitFor(`S()?.querySelector(".control")`, "the settings of the clock in its window")
	if tb.Eval(`!!document.querySelector(".pt-win iframe").src.match(/^about:|^$/)`).(bool) {
		t.Error("the window is not a page of the server")
	}
	// it is a frame in the page: inside the viewport, over the patch
	if !tb.Eval(`(() => { const r = document.querySelector(".pt-win").getBoundingClientRect(); return r.left >= 0 && r.top >= 0 && r.right <= innerWidth && r.bottom <= innerHeight && r.width > 300 && r.height > 200; })()`).(bool) {
		t.Error("the window is not on the page")
	}

	// moved by its title, the size by its corner: kept, per kind of module, for the next window
	before := tb.Eval(`(() => { const r = document.querySelector(".pt-win").getBoundingClientRect(); return [r.left, r.top, r.width, r.height]; })()`).([]any)
	x, y := tb.Center(`document.querySelector(".pt-win-title")`, 0.5)
	tb.Mouse("mousePressed", x, y, nil)
	for i := 1; i <= 4; i++ {
		tb.Mouse("mouseMoved", x-40*float64(i), y+20*float64(i), nil)
	}
	tb.Mouse("mouseReleased", x-160, y+80, nil)
	tb.WaitFor(fmt.Sprintf(`Math.abs(document.querySelector(".pt-win").getBoundingClientRect().left - %v) > 100`, before[0]), "the window moved by its title")
	gx, gy := tb.Center(`document.querySelector(".pt-win-grip")`, 0.5)
	tb.Mouse("mousePressed", gx, gy, nil)
	for i := 1; i <= 4; i++ {
		tb.Mouse("mouseMoved", gx-30*float64(i), gy-20*float64(i), nil)
	}
	tb.Mouse("mouseReleased", gx-120, gy-80, nil)
	tb.WaitFor(fmt.Sprintf(`document.querySelector(".pt-win").getBoundingClientRect().width < %v - 50`, before[2]), "the window made smaller by its corner")
	kept := tb.Eval(`(() => { const r = document.querySelector(".pt-win").getBoundingClientRect(); return JSON.stringify({x: Math.round(r.left), y: Math.round(r.top), w: Math.round(r.width), h: Math.round(r.height)}) === localStorage.getItem("psalm.patcher.window.Patch.clock"); })()`).(bool)
	if !kept {
		t.Error("the place and size of the window are not kept")
	}
	// closed by its x, a window again at the place it was
	tb.Tap(`document.querySelector(".pt-win .pt-x")`)
	tb.WaitFor(`!document.querySelector(".pt-win")`, "the window closed by its ×")
	doubleTap(tb, head)
	tb.WaitFor(fmt.Sprintf(`Math.abs(document.querySelector(".pt-win").getBoundingClientRect().width - %v) < 3`, tb.Eval(`JSON.parse(localStorage.getItem("psalm.patcher.window.Patch.clock")).w`)), "the next window opens as big as the last was")

	// what is typed is taken (Ctrl+Enter), a bad pattern is told in the window (and in a toast, which is over the corner of the page: after the moving above)
	tb.Eval(`(() => { const i = W().querySelector(".mw-text"); i.focus(); i.value = "x . x x"; i.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	paneltest.Eventually(t, "the pattern typed in the window taken", func() bool { tx := pt.Texts(clk); return len(tx) == 1 && tx[0].Value == "x . x x" })
	tb.Eval(`(() => { const i = W().querySelector(".mw-text"); i.value = "x ?"; i.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	tb.WaitFor(`W().querySelector(".mw-text").classList.contains("bad") && W().getElementById("status").textContent.startsWith("error")`, "a bad pattern shown in the window")

	// Esc in a box closes it; the module taken away closes it
	tb.Key("Escape")
	tb.WaitFor(`!document.querySelector(".pt-win")`, "the window closed by Esc")
	doubleTap(tb, head)
	tb.WaitFor(`!!document.querySelector(".pt-win")`, "the window again")
	if err := pt.Remove(clk); err != nil {
		t.Fatal(err)
	}
	tb.WaitFor(`!document.querySelector(".pt-win")`, "the window of a module that is gone closed")
}

// a module with a window of its own (a plugin's): a double click opens that, not the window of settings; with Alt held it is the settings
func TestPageNativeWindow(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	var opened atomic.Int32
	kinds := []patcher.Kind{{Name: "native", Group: "Test", Outs: []patcher.Port{{Name: "out", Type: patcher.Audio}}, Make: func(string) (*patcher.Made, error) {
		return &patcher.Made{Module: gain.New(1), Texts: []patcher.Text{{Key: "note", Label: "note", Lines: 1, Value: "hello"}}, SetText: func(string, string) error { return nil },
			Open: func() error { opened.Add(1); return nil }}, nil
	}}}
	pt := patcher.New(p, ctl, "Patch", kinds)
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("native", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1300, 900)
	head := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id)
	doubleTap(tb, head)
	paneltest.Eventually(t, "the window of the module opened", func() bool { return opened.Load() == 1 })
	if tb.Eval(`document.querySelectorAll(".pt-win").length`).(float64) != 0 {
		t.Error("a window of settings opens with it")
	}
	tb.Eval(`(() => { const h = ` + head + `; h.dispatchEvent(new MouseEvent("dblclick", {altKey: true, bubbles: true})); })()`)
	tb.WaitFor(`document.querySelectorAll(".pt-win").length === 1`, "the settings window opened by Alt and a double click")
	if opened.Load() != 1 {
		t.Errorf("the window of the module opened %d times", opened.Load())
	}
	_ = strings.TrimSpace
}
