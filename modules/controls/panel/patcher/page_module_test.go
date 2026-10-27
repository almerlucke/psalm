package patcher_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// a double click on a module opens its window (window.open of module.html: the module, the folder of the page and the theme in the address), the same window again
// is brought to the front; the window shows the texts of the module and its settings, takes what is typed, follows the patch and says when its module is gone
func TestPageModuleWindow(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(t.TempDir()))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	clk, err := pt.Add("clock", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	base := paneltest.Serve(t, ctl.Handler())
	tb := b.Open(t, base, `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1300, 900)

	// what window.open is asked: recorded, the window it gives is one that is still open
	tb.Eval(`window.__recording = true; window.__opened = []; window.__focused = 0; window.open = (url, name, features) => { window.__opened.push([url, name, features]); return {closed: false, focus() { window.__focused++; }}; }`)
	head := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, clk)
	doubleTap(tb, head)
	tb.WaitFor(`window.__opened.length === 1`, "a window opened by the double click")
	if !tb.Eval(`document.querySelector(".pt-drawer").hidden`).(bool) {
		t.Error("the side panel opens besides the window")
	}
	opened := tb.Eval(`window.__opened[0][0]`).(string)
	u, err := url.Parse(opened)
	if err != nil || !strings.HasSuffix(u.Path, "/plugins/patcher/module.html") || u.Query().Get("id") != clk || u.Query().Get("patcher") != "Patch" || u.Query().Get("mount") != "Patch module" {
		t.Fatalf("the window is asked for at %q", opened)
	}
	if !strings.Contains(tb.Eval(`window.__opened[0][2]`).(string), "popup") {
		t.Error("the window is not a window of its own (popup)")
	}
	doubleTap(tb, head)
	tb.WaitFor(`window.__focused === 1`, "the window brought to the front")
	if n := tb.Eval(`window.__opened.length`).(float64); n != 1 {
		t.Errorf("%v windows for one module", n)
	}

	// the window itself: its own tab here
	win := b.Open(t, opened, `document.getElementById("title").textContent === "clock" && document.querySelector(".mw-text")`)
	win.Size(700, 800)
	win.WaitFor(`document.querySelector(".mw-text").value === "x"`, "the pattern of the clock in its window")
	win.WaitFor(`document.getElementById("frame").contentDocument?.querySelector(".control")`, "the settings of the clock in its window")
	win.WaitFor(`document.getElementById("live").classList.contains("on")`, "the window connected to the patch")
	win.Eval(`(() => { const i = document.querySelector(".mw-text"); i.focus(); i.value = "x . x x"; i.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	paneltest.Eventually(t, "the pattern typed in the window taken", func() bool { tx := pt.Texts(clk); return len(tx) == 1 && tx[0].Value == "x . x x" })
	// a bad pattern: what is wrong, in the window
	win.Eval(`(() => { const i = document.querySelector(".mw-text"); i.value = "x ?"; i.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	win.WaitFor(`document.querySelector(".mw-text").classList.contains("bad") && document.getElementById("status").textContent.startsWith("error")`, "a bad pattern shown in the window")
	// a setting moved in the window reaches the module
	// the module taken away: the window says so
	if err := pt.Remove(clk); err != nil {
		t.Fatal(err)
	}
	win.WaitFor(`!document.getElementById("gone").hidden`, "the window says its module is gone")
}
