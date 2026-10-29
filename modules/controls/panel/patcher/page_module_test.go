package patcher_test

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	paneltest.Eventually(t, "the pattern typed in the window taken", func() bool { tx := pt.Texts(clk); return len(tx) >= 1 && tx[0].Value == "x . x x" })
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

// the lanes of a clock as bars in its window: a lane started with +, a bar dragged to a value, a bar made a gap, the lane shortened; the text follows, and the bars follow the text
func TestPageStepLanes(t *testing.T) {
	b, pt, base := newPage(t)
	clk, err := pt.Add("clock", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, base, `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, clk))
	tb.WaitFor(`W().querySelectorAll(".mw-lane").length === 5`, "a row for each of the five lanes")
	lane := func(name string) string { return `W().querySelector(".mw-lane[data-lane='` + name + `']")` }
	lanesText := func() string {
		for _, x := range pt.Texts(clk) {
			if x.Key == "lanes" {
				return x.Value
			}
		}
		return ""
	}
	// + twice: a lane of two steps at the default
	tb.Eval(lane("prob") + `.querySelectorAll("button")[1].click()`)
	tb.Eval(lane("prob") + `.querySelectorAll("button")[1].click()`)
	paneltest.Eventually(t, "a lane of two steps", func() bool { return lanesText() == "prob: 100 100" })
	tb.WaitFor(lane("prob")+`.querySelectorAll(".mw-lane-bar").length === 2`, "two bars")
	// the second bar dragged to the middle of its height: about 50
	pos := func(step int, frac float64) (float64, float64) {
		r := tb.Eval(fmt.Sprintf(`(() => { const f = document.querySelector(".pt-win-frame").getBoundingClientRect(), b = %s.querySelectorAll(".mw-lane-bar")[%d].getBoundingClientRect(); return [f.left + b.left + b.width / 2, f.top + b.top, b.height]; })()`, lane("prob"), step)).([]any)
		return r[0].(float64), r[1].(float64) + r[2].(float64)*frac
	}
	x, y := pos(1, 0.5)
	tb.Mouse("mousePressed", x, y, nil)
	tb.Mouse("mouseReleased", x, y, nil)
	paneltest.Eventually(t, "the bar dragged to half", func() bool { return lanesText() == "prob: 100 50" })
	// a gap with Alt
	x, y = pos(0, 0.5)
	tb.Mouse("mousePressed", x, y, map[string]any{"modifiers": 1})
	tb.Mouse("mouseReleased", x, y, map[string]any{"modifiers": 1})
	paneltest.Eventually(t, "a gap", func() bool { return lanesText() == "prob: . 50" })
	// typed: the bars follow
	tb.Eval(`(() => { const i = W().querySelectorAll(".mw-text")[1]; i.value = "vel: 100 60 80\nprob: 50"; i.dispatchEvent(new Event("input", {bubbles: true})); })()`)
	tb.WaitFor(lane("vel")+`.querySelectorAll(".mw-lane-bar").length === 3 && `+lane("prob")+`.querySelectorAll(".mw-lane-bar").length === 1`, "the bars of the lanes typed")
	// shortened to nothing takes the lane away
	tb.Eval(lane("prob") + `.querySelectorAll("button")[0].click()`)
	paneltest.Eventually(t, "the lane taken away", func() bool { return lanesText() == "vel: 100 60 80" })
}

// an engine that can choose its input device
type inputChoice struct {
	*psalm.Module
	now    atomic.Pointer[string]
	chosen chan string
}

func (f *inputChoice) InputDevices() []string {
	return []string{"default", "headset (pulse)", "built-in (pulse)"}
}
func (f *inputChoice) InputDevice() string { return *f.now.Load() }
func (f *inputChoice) SetInputDevice(n string) error {
	f.now.Store(&n)
	f.chosen <- n
	return nil
}

// the device of the sound input is chosen in the window of the input module: the engine listens to it
func TestPageInputDevice(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	src := &inputChoice{Module: psalm.NewModule(nil, []int{psalm.Stereo}), chosen: make(chan string, 4)}
	src.SetSelf(src)
	def := "default"
	src.now.Store(&def)
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Audio: src}))
	id, err := pt.Add("input", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1300, 800)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	device := `[...S().querySelectorAll(".control")].find(c => c.textContent.startsWith("device") && !c.querySelector(".control") && c.querySelector("select"))?.querySelector("select")`
	tb.WaitFor(`!!`+device, "the device in the settings of the input")
	tb.Eval(`(() => { const s = ` + device + `; s.value = "2"; s.dispatchEvent(new (S().defaultView.Event)("change", {bubbles: true})); })()`)
	select {
	case got := <-src.chosen:
		if got != "built-in (pulse)" {
			t.Errorf("the engine was told %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the engine was not told")
	}
}

// the editor of a melodist: what plays, the sections as chips, a preset, a slider, the keyboard of the range, freeze
func TestPageMelodistEditor(t *testing.T) {
	b, pt, base := newPage(t)
	song, _ := pt.Add("song", "", 60, 300)
	if err := pt.SetText(song, "sections", "Verse 8, Chorus 8"); err != nil {
		t.Fatal(err)
	}
	harm, _ := pt.Add("harmony", "", 60, 400)
	if err := pt.SetText(harm, "chords", "Verse: Dm7 G7 Cmaj7 Am7\nChorus: F C Dm Bb"); err != nil {
		t.Fatal(err)
	}
	id, err := pt.Add("melodist", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	m := pt.Module(id).(*patcher.Melodist)
	tb := b.Open(t, base, `document.querySelectorAll(".pt-node").length >= 4`)
	tb.Size(1300, 1000)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	root := `S().querySelector(".melodist .md")`
	tb.WaitFor(`!!`+root+` && `+root+`.querySelectorAll(".md-chip").length === 3 && /notes/.test(`+root+`.querySelector(".md-status").textContent)`, "the editor, with the chips of the sections and what plays")
	click := func(sel string) {
		tb.Eval(`(() => { ` + root + `.querySelector(` + strconv.Quote(sel) + `).click(); })()`)
	}
	// a preset
	tb.Eval(`(() => { const s = ` + root + `.querySelector(".md-bar select"); s.value = "Music box tune"; s.dispatchEvent(new (S().defaultView.Event)("change")); })()`)
	paneltest.Eventually(t, "a preset taken", func() bool { v := m.Values(); return v["grid"] == 1.5 && v["contour"] == 3 && v["low"] == 70 })
	// a slider: the movement to leaps
	tb.Eval(`(() => { const r = [...` + root + `.querySelectorAll(".md-row")].find(r => r.textContent.startsWith("Movement")).querySelector("input"); r.value = "0.8"; r.dispatchEvent(new (S().defaultView.Event)("input")); })()`)
	paneltest.Eventually(t, "the slider taken", func() bool { return math.Abs(m.Values()["steps"]-0.2) < 0.02 })
	// the sections: out of the Verse
	click(".md-chip:nth-child(2)") // Verse off
	paneltest.Eventually(t, "the verse taken out of where it plays", func() bool {
		var st struct {
			Plays []string `json:"plays"`
		}
		raw, _ := json.Marshal(m.View())
		_ = json.Unmarshal(raw, &st)
		return len(st.Plays) == 1 && st.Plays[0] == "Chorus"
	})
	// freeze
	click(".md-bar button:nth-child(3)")
	paneltest.Eventually(t, "frozen", func() bool { return m.Values()["freeze"] == 1 })
	// what a patch keeps: back from its state
	st := m.StateText()
	n := patcher.NewMelodist()
	if err := n.SetStateText(st); err != nil || n.Values()["grid"] != 1.5 || n.Values()["freeze"] != 1 {
		t.Errorf("the state kept: %v %v", err, n.Values())
	}
}

// the grid of the sections: a row for every pattern text, a column for every section; a cell edits, varies, clears and copies the line of its section
func TestPageSectionGrid(t *testing.T) {
	b, pt, base := newPage(t)
	song, _ := pt.Add("song", "", 60, 300)
	if err := pt.SetText(song, "sections", "Verse 8, Chorus 8"); err != nil {
		t.Fatal(err)
	}
	clk, _ := pt.Add("clock", "", 60, 40)
	if err := pt.SetText(clk, "pattern", "Verse: x . x x\nChorus: E(5,16,+)"); err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, base, `document.querySelectorAll(".pt-node").length >= 2`)
	tb.Size(1300, 1000)
	tb.Eval(`document.querySelector(".pt-grid-btn").click()`)
	tb.WaitFor(`document.querySelectorAll(".pt-grid-cell").length === 2 && document.querySelectorAll(".pt-grid-col").length === 2`, "a row with a cell for each section")
	cell := func(i int) string {
		return `document.querySelectorAll(".pt-grid-cell")[` + strconv.Itoa(i) + `]`
	}
	text := func(id, key string) string {
		for _, x := range pt.Texts(id) {
			if x.Key == key {
				return x.Value
			}
		}
		return ""
	}
	// edit the verse
	tb.Eval(cell(0) + `.click()`)
	tb.WaitFor(`!!document.querySelector(".pt-grid-text")`, "the editor of the cell")
	tb.Eval(`(() => { const a = document.querySelector(".pt-grid-text"); a.value = "x x x x"; [...document.querySelectorAll(".pt-grid-edit-row button")].find(b => b.textContent === "Save").click(); })()`)
	paneltest.Eventually(t, "the verse saved", func() bool { return text(clk, "pattern") == "Verse: x x x x\nChorus: E(5,16,+)" })
	// vary the chorus: the rotation of the Euclidean pattern moves
	tb.Eval(cell(1) + `.click()`)
	tb.WaitFor(`/E\(5/.test(document.querySelector(".pt-grid-text")?.value || "")`, "the chorus in the editor")
	tb.Eval(`[...document.querySelectorAll(".pt-grid-edit-row button")].find(b => b.textContent === "Vary").click()`)
	paneltest.Eventually(t, "the chorus varied", func() bool {
		s := text(clk, "pattern")
		return strings.HasPrefix(s, "Verse: x x x x\nChorus: E(5,16,") && !strings.Contains(s, "E(5,16,+)")
	})
	// copy the verse to the chorus
	tb.Eval(cell(0) + `.click()`)
	tb.WaitFor(`document.querySelector(".pt-grid-text")?.value === "x x x x"`, "the verse in the editor")
	tb.Eval(`(() => { const s = document.querySelector(".pt-grid-edit-row select"); s.value = "Chorus"; s.dispatchEvent(new Event("change")); })()`)
	paneltest.Eventually(t, "the verse copied", func() bool { return text(clk, "pattern") == "Verse: x x x x\nChorus: x x x x" })
	// clear the chorus
	tb.Eval(cell(1) + `.click()`)
	tb.WaitFor(`document.querySelector(".pt-grid-edit-title")?.textContent.endsWith("Chorus")`, "the chorus in the editor")
	tb.Eval(`[...document.querySelectorAll(".pt-grid-edit-row button")].find(b => b.textContent === "Clear").click()`)
	paneltest.Eventually(t, "the chorus cleared", func() bool { return text(clk, "pattern") == "Verse: x x x x" })
}
