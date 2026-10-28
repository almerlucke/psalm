package patcher_test

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/chordmodel"
	"github.com/almerlucke/psalm/modules/controls/panel/noteroll"
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
	// a tap on a module chooses it; a double tap: its settings in the side panel (its editor,
	// served by the patcher's mount)
	lfoHead := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, lfo)
	tb.Tap(lfoHead)
	tb.WaitFor(lfoHead+`.parentNode.classList.contains("selected")`, "the LFO chosen")
	if !tb.Eval(`!document.querySelector(".pt-win")`).(bool) {
		t.Error("a tap chooses a module, it doesn't open its side panel")
	}
	doubleTap(tb, lfoHead)
	tb.WaitFor(`!!document.querySelector(".pt-win") && S()?.querySelector(".control")`, "the LFO's settings in the side panel")
	if tb.Eval(`!!document.querySelector("section.mounthost")`).(bool) {
		t.Error("the settings of the module chosen are in the side panel, not in a section below the patch")
	}
	// a module taken away by the Delete key, its cables with it, its side panel closed
	tb.Key("Delete")
	paneltest.Eventually(t, "the LFO gone", func() bool { return len(pt.Nodes()) == 3 })
	tb.WaitFor(`!document.querySelector(".pt-win")`, "its side panel closed")

	// a generator's text in its side panel, in a box as a script's (none on the module): a good
	// pattern taken (Ctrl+Enter, over lines too), a bad one shown with what is wrong with it; its
	// settings below
	clk, err := pt.Add("clock", "", 400, 380)
	if err != nil {
		t.Fatal(err)
	}
	clkNode := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s']")`, clk)
	tb.WaitFor(clkNode, "the clock on the patch")
	if tb.Eval(clkNode + `.querySelector("input, textarea") !== null`).(bool) {
		t.Error("the clock's pattern is in its side panel, not on its module")
	}
	doubleTap(tb, clkNode+`.querySelector(".pt-head")`)
	input := `W().querySelector(".mw-text")`
	tb.WaitFor(`!!document.querySelector(".pt-win") && `+input+`?.value === "x" && S()?.querySelector(".control")`, "the pattern and the clock's settings in its side panel")
	enter := func(text string) {
		tb.Eval(`(() => { const i = ` + input + `; i.focus(); i.value = ` + strconv.Quote(text) + `; i.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	}
	if !tb.Eval(`!!([...W().querySelectorAll(".mw-help li")].some(li => li.textContent.includes("a hit")) && W().querySelector(".mw-help code"))`).(bool) {
		t.Error("what to know to type a pattern is not above its box")
	}
	if !tb.Eval(`W().querySelector(".mw-text").getBoundingClientRect().right <= document.querySelector(".pt-win").getBoundingClientRect().right - 8`).(bool) {
		t.Error("the box of a text runs past the side panel")
	}
	enter("x . x x\nx x . .")
	paneltest.Eventually(t, "the pattern taken", func() bool { tx := pt.Texts(clk); return len(tx) >= 1 && tx[0].Value == "x . x x\nx x . ." })
	if e := pt.TextError(clk, "pattern"); e != "" {
		t.Errorf("a pattern over two lines: %s", e)
	}
	enter("x ?")
	tb.WaitFor(input+`.classList.contains("bad") && W().getElementById("status").textContent.startsWith("error")`, "a bad pattern shown, what is wrong below it")
	tb.Key("Escape")
	tb.WaitFor(`!document.querySelector(".pt-win")`, "the side panel closed")

	// a script, opened in the side panel by a tap (no box of it on the module): an error of it shown
	// on its line, in the side panel and on the module, then what it logs
	scr, err := pt.Add("script", "", 700, 380)
	if err != nil {
		t.Fatal(err)
	}
	node := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s']")`, scr)
	tb.WaitFor(node, "the script on the patch")
	if tb.Eval(node + `.querySelector("textarea") !== null`).(bool) {
		t.Error("a script has no box on its module")
	}
	doubleTap(tb, node+`.querySelector(".pt-head")`)
	box, status := `W().querySelector(".mw-text")`, `W().getElementById("status")`
	tb.WaitFor(`!!document.querySelector(".pt-win") && `+box+`.value.includes("function trigger")`, "the script in the side panel")
	tb.WaitFor(`[...W().querySelectorAll(".mw-help li")].some(li => li.textContent.includes("function trigger(ev)"))`, "what to know to write a script above it")
	run := func(code string) {
		tb.Eval(`(() => { const b = ` + box + `; b.focus(); b.value = ` + strconv.Quote(code) + `; b.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	}
	run("const a = 1\nlet b = (;")
	tb.WaitFor(box+`.classList.contains("bad") && `+status+`.classList.contains("bad") && `+status+`.textContent.includes("line 2")`, "the error on its line in the side panel")
	tb.WaitFor(`[...document.querySelectorAll(".pt-toast")].some(t => t.textContent.includes("line 2"))`, "the error in a toast")
	tb.WaitFor(node+`.classList.contains("failed") && !`+node+`.querySelector(".pt-status")`, "the error not on the module, which has a red edge")
	run("log('hello', [1, 2])")
	// what it logs is in the log of the patch (Log, in the bar), with the time and the module in front, and not on the module
	tb.WaitFor(`!`+status+`.classList.contains("bad") && !`+box+`.classList.contains("bad") && !`+node+`.classList.contains("failed")`, "the error gone")
	tb.Eval(`document.querySelector(".pt-log-btn").click()`)
	tb.WaitFor(`!document.querySelector(".pt-log").hidden && /^\d\d:\d\d:\d\d\.\d{3}  script \w+  hello \[1,2\]$/.test(document.querySelector(".pt-log-text").value)`, "what it logged, in the log of the patch")
	tb.WaitFor(`document.querySelector(".pt-log-btn").textContent === "Log (1)"`, "the count of lines on the button")
	if os.Getenv("SHOT") != "" {
		tb.Screenshot(os.Getenv("SHOT"))
	}

	// kept by name
	tb.Eval(`(() => { const i = document.querySelector(".pt-name"); i.value = "first"; [...document.querySelectorAll(".pt-bar button")].find(b => b.textContent === "Save").click(); })()`)
	paneltest.Eventually(t, "the patch kept", func() bool { _, err := os.Stat(dir + "/first.json"); return err == nil })
	tb.WaitFor(`[...document.querySelectorAll(".pt-patches option")].some(o => o.value === "first")`, "the patch listed")
}

// the workspace: modules chosen together (shift-click, a rectangle) and moved together, a cable
// taken hold of beside its line, a script in the side panel, the patch over the whole screen
func TestPageWorkspace(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	add := func(kind string, x, y float64) string {
		t.Helper()
		id, err := pt.Add(kind, "", x, y)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	lfo, rnd, scr, clock := add("lfo", 60, 40), add("random", 300, 40), add("script", 60, 300), add("clock", 560, 420)
	if _, err := pt.Connect(patcher.Cable{From: clock, To: scr}); err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 5 && document.querySelector("path.pt-cable.trigger")`)
	tb.Size(1300, 900)
	node := func(id string) string { return fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s']")`, id) }
	// the patch as wide as the page: its section takes the whole row
	tb.WaitFor(`Math.abs(document.querySelector(".pt").closest("section").getBoundingClientRect().right - (document.querySelector("main").getBoundingClientRect().right - parseFloat(getComputedStyle(document.querySelector("main")).paddingRight))) < 2`, "the patch as wide as the page")
	at := func(id string) (float64, float64) {
		for _, n := range pt.Nodes() {
			if n.ID == id {
				return n.X, n.Y
			}
		}
		t.Fatalf("no module %s", id)
		return 0, 0
	}

	// two chosen (a tap, a shift-tap), moved together by one of them
	tb.Tap(node(lfo) + `.querySelector(".pt-head")`)
	x, y := tb.Center(node(rnd)+`.querySelector(".pt-title")`, 0.5)
	tb.Mouse("mousePressed", x, y, map[string]any{"modifiers": 8}) // shift
	tb.Mouse("mouseReleased", x, y, map[string]any{"modifiers": 8})
	tb.WaitFor(`document.querySelectorAll(".pt-node.selected").length === 2`, "two modules chosen")
	x, y = tb.Center(node(lfo)+`.querySelector(".pt-title")`, 0.5)
	tb.Mouse("mousePressed", x, y, nil)
	for i := 1; i <= 5; i++ {
		tb.Mouse("mouseMoved", x+float64(i)*20, y+float64(i)*16, nil)
	}
	tb.Mouse("mouseReleased", x+100, y+80, nil)
	paneltest.Eventually(t, "both moved by the drag", func() bool {
		lx, ly := at(lfo)
		rx, ry := at(rnd)
		return lx == 160 && ly == 120 && rx == 400 && ry == 120
	})

	// Esc lets go of them; a rectangle around both chooses them again
	if !tb.Eval(`!document.querySelector(".pt-win")`).(bool) {
		t.Error("taps choose modules, they don't open a side panel")
	}
	tb.Key("Escape")
	tb.WaitFor(`document.querySelectorAll(".pt-node.selected").length === 0`, "nothing chosen")
	rectangle := func() { // dragged around the LFO and the random source, from the empty patch
		r := tb.Eval(`(() => { const a = ` + node(lfo) + `.getBoundingClientRect(), b = ` + node(rnd) + `.getBoundingClientRect();
			return [a.left - 12, a.top - 12, b.right + 12, Math.max(a.bottom, b.bottom) + 12]; })()`).([]any)
		x0, y0, x1, y1 := r[0].(float64), r[1].(float64), r[2].(float64), r[3].(float64)
		tb.Mouse("mousePressed", x0, y0, nil)
		for i := 1; i <= 6; i++ {
			tb.Mouse("mouseMoved", x0+(x1-x0)*float64(i)/6, y0+(y1-y0)*float64(i)/6, nil)
		}
		tb.Mouse("mouseReleased", x1, y1, nil)
	}
	rectangle()
	tb.WaitFor(`[...document.querySelectorAll(".pt-node.selected")].map(n => n.dataset.id).sort().join() === "`+strings.Join(sortedIDs(lfo, rnd), ",")+`"`, "the two in the rectangle chosen")

	// a cable taken hold of 6 pixels beside its line
	pos := tb.Eval(`(() => { const p = document.querySelector("path.pt-cable.trigger"), l = p.getTotalLength(), q = p.getPointAtLength(l / 2), m = p.getScreenCTM();
		return [q.x * m.a + m.e, q.y * m.d + m.f]; })()`).([]any)
	cx, cy := pos[0].(float64), pos[1].(float64)+6
	tb.Mouse("mousePressed", cx, cy, nil)
	tb.Mouse("mouseReleased", cx, cy, nil)
	tb.WaitFor(`document.querySelector(".pt-inspector").textContent.includes("trigger cable")`, "the cable picked beside its line")
	tb.WaitFor(`document.querySelectorAll(".pt-node.selected").length === 0`, "the modules let go of when the cable was picked")
	tb.Key("Delete")
	paneltest.Eventually(t, "the cable taken away", func() bool { return len(pt.Cables()) == 0 })

	// the script in the side panel: typed there, run with Ctrl+Enter, what it logs below; Esc closes
	doubleTap(tb, node(scr)+`.querySelector(".pt-head")`)
	tb.WaitFor(`!!document.querySelector(".pt-win") && W().querySelector(".mw-text").value.includes("function trigger")`, "the script in the side panel")
	tb.Eval(`(() => { const d = W().querySelector(".mw-text"); d.focus(); d.value = "log('wide')"; d.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	tb.WaitFor(`document.querySelector(".pt-log-text").value.endsWith("  wide")`, "what it logs in the log of the patch")
	if tx := pt.Texts(scr); len(tx) != 1 || tx[0].Value != "log('wide')" {
		t.Fatalf("the script typed in the side panel: %v", tx)
	}
	tb.Key("Escape")
	tb.WaitFor(`!document.querySelector(".pt-win")`, "the side panel closed")

	// a window for each module, at the same time: a double tap on a module that has its window brings it to the front (it does not close it), the × closes the one it is on,
	// Esc (in a box, or on the page) the one in front
	head := func(id string) string { return node(id) + `.querySelector(".pt-head")` }
	front := `[...document.querySelectorAll(".pt-win")].sort((a, b) => b.style.zIndex - a.style.zIndex)[0].querySelector(".pt-win-title").textContent`
	doubleTap(tb, head(scr))
	doubleTap(tb, head(lfo))
	tb.WaitFor(`document.querySelectorAll(".pt-win").length === 2 && `+front+` === "lfo"`, "a window for each of the two modules, the LFO's in front")
	doubleTap(tb, head(scr))
	tb.WaitFor(`document.querySelectorAll(".pt-win").length === 2 && `+front+`.startsWith("script")`, "the script's window in front again, not a third window")
	closeFront := `[...document.querySelectorAll(".pt-win")].sort((a, b) => b.style.zIndex - a.style.zIndex)[0].querySelector(".pt-x")`
	tb.Tap(closeFront)
	tb.WaitFor(`document.querySelectorAll(".pt-win").length === 1 && document.querySelector(".pt-win-title").textContent === "lfo"`, "the script's window, the one in front, closed by its ×")
	tb.Tap(closeFront)
	tb.WaitFor(`!document.querySelector(".pt-win")`, "the last window closed by its ×")

	// the patch over the whole screen, and back
	if os.Getenv("SHOT") != "" {
		doubleTap(tb, node(scr)+`.querySelector(".pt-head")`)
		tb.Tap(`document.querySelector(".pt-max-btn")`)
		tb.Screenshot(os.Getenv("SHOT"))
		tb.Tap(`document.querySelector(".pt-win .pt-x")`)
		tb.Tap(`document.querySelector(".pt-max-btn")`)
	}
	tb.Tap(`document.querySelector(".pt-max-btn")`)
	tb.WaitFor(`document.querySelector(".pt").classList.contains("pt-max") && document.querySelector(".pt-area").getBoundingClientRect().height > 750`, "the patch over the whole screen")
	tb.Tap(`document.querySelector(".pt-max-btn")`)
	tb.WaitFor(`!document.querySelector(".pt").classList.contains("pt-max")`, "the patch back in the page")

	// the Delete key: the modules chosen
	rectangle()
	tb.WaitFor(`document.querySelectorAll(".pt-node.selected").length === 2`, "the two chosen again")
	tb.Eval(`window.confirm = () => true`)
	tb.Key("Delete")
	paneltest.Eventually(t, "the two chosen taken away", func() bool { return len(pt.Nodes()) == 3 })
}

func sortedIDs(ids ...string) []string { slices.Sort(ids); return ids }

// doubleTap taps an element twice, quickly (a double click)
func doubleTap(tb *paneltest.Tab, expr string) {
	tb.Eval(windowHelpers)
	x, y := tb.Center(expr, 0.5)
	for _, n := range []int{1, 2} {
		tb.Mouse("mousePressed", x, y, map[string]any{"clickCount": n})
		tb.Mouse("mouseReleased", x, y, map[string]any{"clickCount": n})
	}
	// the window of a module, if it has one to show, is ready: what is in it (module.html) has the module
	tb.WaitFor(`!document.querySelector(".pt-win") || W()?.body?.dataset.ready === "1"`, "the window of the module ready")
}

// windowHelpers: W() is the document in the window of a module (the first), S() the document of its settings frame
const windowHelpers = `window.W = () => document.querySelector(".pt-win-frame")?.contentDocument; window.S = () => W()?.getElementById("frame")?.contentDocument; 0`

// the keys: Ctrl+C copies the modules chosen, Ctrl+V pastes them (chosen then, to move them),
// Ctrl+D duplicates, Ctrl+X cuts
func TestPageCopyPaste(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	clock, _ := pt.Add("clock", "", 60, 40)
	seq, _ := pt.Add("sequence", "", 300, 40)
	if _, err := pt.Connect(patcher.Cable{From: clock, To: seq}); err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 3`)
	tb.Size(1300, 900)
	head := func(id string) string {
		return fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id)
	}
	key := func(k string) {
		tb.Eval(`window.dispatchEvent(new KeyboardEvent("keydown", {key: "` + k + `", ctrlKey: true, bubbles: true}))`)
	}
	chosenIDs := `[...document.querySelectorAll(".pt-node.selected")].map(n => n.dataset.id).sort().join()`

	tb.Tap(head(clock))
	x, y := tb.Center(head(seq), 0.5)
	tb.Mouse("mousePressed", x, y, map[string]any{"modifiers": 8})
	tb.Mouse("mouseReleased", x, y, map[string]any{"modifiers": 8})
	tb.WaitFor(`document.querySelectorAll(".pt-node.selected").length === 2`, "two chosen")
	key("c")
	tb.WaitFor(`document.querySelector(".pt-note").textContent.includes("2 copied")`, "a note of the copy")
	key("v")
	paneltest.Eventually(t, "two pasted", func() bool { return len(pt.Nodes()) == 5 && len(pt.Cables()) == 2 })
	tb.WaitFor(`document.querySelectorAll(".pt-node.selected").length === 2 && !`+chosenIDs+`.split(",").some(id => id === "`+clock+`" || id === "`+seq+`")`, "the modules pasted chosen")
	key("d")
	paneltest.Eventually(t, "two duplicated", func() bool { return len(pt.Nodes()) == 7 && len(pt.Cables()) == 3 })
	tb.WaitFor(`document.querySelectorAll(".pt-node.selected").length === 2`, "the duplicates chosen")
	key("x")
	paneltest.Eventually(t, "the two cut", func() bool { return len(pt.Nodes()) == 5 })
	key("v")
	paneltest.Eventually(t, "the cut pasted", func() bool { return len(pt.Nodes()) == 7 })
}

// a song on the page: where it is in the bar at the top, its sections and their tiles in its side
// panel (a tap asks for a section next)
func TestPageSong(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	if !tb.Eval(`document.querySelector(".pt-songbar").hidden`).(bool) {
		t.Error("the song in the bar at the top without a song")
	}
	song, err := pt.Add("song", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	_ = pt.SetText(song, "sections", "Intro 64, Verse 64")
	tb.WaitFor(`!document.querySelector(".pt-songbar").hidden && document.querySelector(".pt-songbar span").textContent.startsWith("Intro ")`, "the song in the bar at the top")

	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, song))
	frame := `S()`
	tb.WaitFor(`W().querySelector(".mw-text")?.value === "Intro 64, Verse 64" && `+frame+`?.querySelectorAll(".cue").length === 2`, "its sections and their tiles in the side panel")
	tb.Eval(`[...` + frame + `.querySelectorAll(".cue")].find(c => c.textContent.includes("Verse")).click()`)
	tb.WaitFor(`document.querySelector(".pt-songbar span").textContent.includes("→ Verse")`, "the section asked for next in the bar at the top")

	// a gate is a part of the song: its tile in the song's side panel switches it off
	gate, err := pt.Add("gate", "", 300, 40)
	if err != nil {
		t.Fatal(err)
	}
	_ = pt.SetText(gate, "name", "drums")
	tb.WaitFor(`[...`+frame+`.querySelectorAll(".cue")].some(c => c.textContent.includes("drums"))`, "the gate's tile among the song's parts")
	tb.Eval(`[...` + frame + `.querySelectorAll(".cue")].find(c => c.textContent.includes("drums")).click()`)
	paneltest.Eventually(t, "the gate switched off by its tile (kept with the patch)", func() bool { return pt.SettingsOf(gate).Values()["on"] == 0 })
}

// the drum machine: its grid in the side panel, the sequencer playing it with the patch
func TestPageDrumMachine(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Drums: "../../../../assets/sounds/drums"}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("drummachine", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	frame := `S()`
	tb.WaitFor(`!!document.querySelector(".pt-win") && `+frame+`?.querySelectorAll(".cell").length === 7*32`, "its grid in the side panel")
	shown := `[...` + frame + `.querySelectorAll(".cell")].filter(c => !c.hidden && c.offsetParent).length`
	tb.WaitFor(shown+` === 7*16 && `+frame+`.querySelectorAll(".gridhead > button").length === 16`, "sixteen patterns of sixteen steps")
	tb.Eval(frame + `.querySelector("input.stepsin").value = "24"; ` + frame + `.querySelector("input.stepsin").dispatchEvent(new Event("change"))`)
	tb.WaitFor(shown+` === 7*24`, "the first pattern of 24 steps")
	tb.Eval(`[...` + frame + `.querySelectorAll(".gridhead > button")][1].click()`)
	tb.WaitFor(shown+` === 7*16`, "the second pattern still of 16")
	tb.WaitFor(frame+`.querySelectorAll(".gridhead > button.off").length === 12 && `+frame+`.querySelector(".stepslab input[type=checkbox]").checked`, "the first four patterns in the rotation")
	tb.WaitFor(`[...`+frame+`.querySelectorAll("select")].filter(s => s.options.length > 200).length === 7`, "a sample to choose for every row")
}

// the log: not on the module, but in a box of its side panel, below the label
func TestPageLog(t *testing.T) {
	b := paneltest.Start(t)
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	clock, _ := pt.Add("clock", "", 60, 40)
	lg, _ := pt.Add("log", "", 400, 40)
	if _, err := pt.Connect(patcher.Cable{From: clock, To: lg, In: 1}); err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 3`)
	tb.Size(1300, 900)
	if !tb.Eval(fmt.Sprintf(`!document.querySelector(".pt-node[data-id='%s'] .pt-status")`, lg)).(bool) {
		t.Error("the log is not on the module")
	}
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, lg))
	tb.WaitFor(`W().querySelector(".mw-text[readonly]")?.value.includes("trigger step")`, "the log in a box of the side panel")
	if !tb.Eval(`W().querySelector(".mw-text[readonly]").readOnly && W().querySelector(".mw-text:not([readonly])").getBoundingClientRect().bottom <= W().querySelector(".mw-text[readonly]").getBoundingClientRect().top`).(bool) {
		t.Error("the label above the log, the log read only")
	}
}

// the note editor: a click puts a note (a drag makes it longer), a drag moves it, a right click takes it away
func TestPageNoteRoll(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("noteroll", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	frame := `S()`
	canvas := frame + `.querySelector(".noteroll canvas")`
	tb.WaitFor(`!!`+canvas+` && `+canvas+`.clientHeight > 300 && parseInt(W().getElementById("frame").style.height) > 400`, "the roll in the side panel, at its size")
	// where a beat and a key are on the canvas (in the page): 8 beats across, the keys from C3 up, under the ruler (34 px), over the velocity lane (70 px and a gap of 6), rows of about 13 px
	at := func(beat, key float64) (float64, float64) {
		r := tb.Eval(fmt.Sprintf(`(() => { const g = document.querySelector(".pt-win-frame").getBoundingClientRect(), f0 = W().getElementById("frame").getBoundingClientRect(), f = {left: g.left + f0.left, top: g.top + f0.top}, c = %s, r = c.getBoundingClientRect();
			const pxb = (r.width - 52) / 8, areaH = r.height - 34 - 70 - 6, rows = Math.max(4, Math.floor(areaH / 13)), rowH = areaH / rows;
			return [f.left + r.left + 52 + %v * pxb, f.top + r.top + 34 + (rows - 1 - (%v - 48)) * rowH + rowH / 2]; })()`, canvas, beat, key)).([]any)
		return r[0].(float64), r[1].(float64)
	}
	tb.Eval(`S().querySelector('.nr-tool[data-tool="draw"]').click()`) // the draw tool: a click puts a note
	values := func() map[string]float64 { return pt.SettingsOf(id).Values() }
	x0, y0 := at(1.1, 60)
	x1, _ := at(2.4, 60)
	tb.Mouse("mousePressed", x0, y0, nil)
	tb.Mouse("mouseMoved", x1, y0, nil)
	tb.Mouse("mouseReleased", x1, y0, nil)
	paneltest.Eventually(t, "a note put, dragged longer", func() bool {
		v := values()
		return v["n/0/key"] == 60 && v["n/0/start"] == 1 && v["n/0/length"] >= 1.25 && v["n/1/key"] == 0
	})
	// moved up and later
	xa, ya := at(1.5, 60)
	xb, yb := at(3.1, 64)
	tb.Mouse("mousePressed", xa, ya, nil)
	tb.Mouse("mouseMoved", xb, yb, nil)
	tb.Mouse("mouseReleased", xb, yb, nil)
	paneltest.Eventually(t, "the note moved", func() bool {
		v := values()
		return v["n/0/key"] == 64 && v["n/0/start"] >= 2.5
	})
	// a right click takes it away
	xc, yc := at(3.4, 64)
	tb.Mouse("mousePressed", xc, yc, map[string]any{"button": "right", "buttons": 2, "clickCount": 1})
	tb.Mouse("mouseReleased", xc, yc, map[string]any{"button": "right", "clickCount": 1})
	paneltest.Eventually(t, "the note taken away", func() bool { _, ok := values()["n/0/key"]; return !ok })
}

// the note editor edits like a sequencer's: a rectangle selects, copy and paste at the cursor, undo and redo, duplicate, a region of the ruler, the velocity lane, quantize
func TestPageNoteRollEditing(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("noteroll", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	// four notes: C D E F, a beat each, in a loop of 8
	pt.SettingsOf(id).Apply(map[string]float64{"length": 8,
		"n/0/start": 0, "n/0/length": 1, "n/0/key": 60, "n/0/velocity": 0.8,
		"n/1/start": 1, "n/1/length": 1, "n/1/key": 62, "n/1/velocity": 0.8,
		"n/2/start": 2, "n/2/length": 1, "n/2/key": 64, "n/2/velocity": 0.8,
		"n/3/start": 3, "n/3/length": 1, "n/3/key": 65, "n/3/velocity": 0.8})
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	canvas := `S().querySelector(".noteroll canvas")`
	tb.WaitFor(`!!`+canvas+` && `+canvas+`.clientHeight > 300 && parseInt(W().getElementById("frame").style.height) > 400`, "the roll in the side panel, at its size")
	// the geometry of the canvas, as the page has it: 8 beats across, the keys from the middle of the notes (60 and 65: from 48 or so), a ruler of 34 px, a velocity lane of 70 px
	geom := func() (left, top, pxb, rowH, rows, lo float64) {
		r := tb.Eval(fmt.Sprintf(`(() => { const g = document.querySelector(".pt-win-frame").getBoundingClientRect(), f0 = W().getElementById("frame").getBoundingClientRect(), c = %s, r = c.getBoundingClientRect();
			const areaH = r.height - 34 - 70 - 6, rows = Math.max(4, Math.floor(areaH / 13));
			return [g.left + f0.left + r.left, g.top + f0.top + r.top, (r.width - 52) / 8, areaH / rows, rows, r.height]; })()`, canvas)).([]any)
		return r[0].(float64), r[1].(float64), r[2].(float64), r[3].(float64), r[4].(float64), r[5].(float64)
	}
	// lo is where the notes are: half way between the lowest and the highest, less half the rows
	loKey := func() float64 {
		_, _, _, _, rows, _ := geom()
		return math.Floor((60+65)/2) - math.Floor(rows/2) // the first state put the keys there
	}
	note := func(beat, key float64) (float64, float64) {
		left, top, pxb, rowH, rows, _ := geom()
		return left + 52 + beat*pxb, top + 34 + (rows-1-(key-loKey()))*rowH + rowH/2
	}
	ruler := func(beat float64) (float64, float64) {
		left, top, pxb, _, _, _ := geom()
		return left + 52 + beat*pxb, top + 10
	}
	vel := func(beat, level float64) (float64, float64) { // the velocity lane: 70 px high at the bottom, bars of up to 62
		left, top, pxb, _, _, h := geom()
		return left + 52 + beat*pxb + 1, top + h - 2 - level*62
	}
	press := func(x, y float64) { tb.Mouse("mousePressed", x, y, nil) }
	moveTo := func(x, y float64) { tb.Mouse("mouseMoved", x, y, nil) }
	release := func(x, y float64) { tb.Mouse("mouseReleased", x, y, nil) }
	drag := func(x0, y0, x1, y1 float64) {
		press(x0, y0)
		moveTo((x0+x1)/2, (y0+y1)/2)
		moveTo(x1, y1)
		release(x1, y1)
	}
	keyDown := func(k string, mods ...string) {
		ctrl, shift := slices.Contains(mods, "ctrl"), slices.Contains(mods, "shift")
		tb.Eval(fmt.Sprintf(`%s.dispatchEvent(new (S().defaultView.KeyboardEvent)("keydown", {key: %q, ctrlKey: %v, shiftKey: %v, bubbles: true, cancelable: true}))`, canvas, k, ctrl, shift))
	}
	starts := func() []float64 {
		v := pt.SettingsOf(id).Values()
		var out []float64
		for i := 0; ; i++ {
			s, ok := v[fmt.Sprintf("n/%d/start", i)]
			if !ok {
				return out
			}
			out = append(out, s)
		}
	}
	paneltest.Eventually(t, "the four notes", func() bool { return len(starts()) == 4 })

	// a rectangle around the first two notes selects them; copy; the cursor in the ruler at beat 4; paste
	x0, y0 := note(0.02, 67)
	x1, y1 := note(1.9, 59)
	drag(x0, y0, x1, y1)
	tb.Eval(canvas + `.focus()`)
	keyDown("c", "ctrl")
	rx, ry := ruler(4)
	press(rx, ry)
	release(rx, ry)
	keyDown("v", "ctrl")
	paneltest.Eventually(t, "the two notes pasted at the cursor", func() bool {
		s := starts()
		return len(s) == 6 && slices.Contains(s, 4) && slices.Contains(s, 5)
	})
	// undo takes them away, redo puts them back
	keyDown("z", "ctrl")
	paneltest.Eventually(t, "the paste undone", func() bool { return len(starts()) == 4 })
	keyDown("z", "ctrl", "shift")
	paneltest.Eventually(t, "the paste redone", func() bool { return len(starts()) == 6 })

	// a rectangle around the pasted notes: ctrl+D copies them after themselves (two beats on), the arrows move the copies up a semitone and a beat later
	px0, py0 := note(4.02, 67)
	px1, py1 := note(5.9, 59)
	drag(px0, py0, px1, py1)
	keyDown("d", "ctrl")
	paneltest.Eventually(t, "the notes duplicated after themselves", func() bool { s := starts(); return len(s) == 8 && slices.Contains(s, 6) && slices.Contains(s, 7) })
	keyDown("ArrowUp")
	keyDown("ArrowRight", "shift")
	paneltest.Eventually(t, "the copies moved", func() bool {
		v := pt.SettingsOf(id).Values()
		moved := 0
		for i := range 8 {
			if v[fmt.Sprintf("n/%d/key", i)] == 61 || v[fmt.Sprintf("n/%d/key", i)] == 63 {
				moved++
			}
		}
		return moved == 2
	})

	// a drag in the velocity lane sets the velocity of the first note
	vx, vy := vel(0, 0.25)
	drag(vx, vy, vx, vy)
	paneltest.Eventually(t, "the velocity of the first note", func() bool { v := pt.SettingsOf(id).Values(); return v["n/0/velocity"] < 0.4 })

	// a region of the ruler, duplicated: the notes in beats 0 to 2 again in 2 to 4
	before := len(starts())
	ax, ay := ruler(0)
	bx, by := ruler(2)
	drag(ax, ay, bx, by)
	tb.Eval(canvas + `.focus()`)
	keyDown("d", "ctrl")
	paneltest.Eventually(t, "the region duplicated", func() bool { return len(starts()) > before })
}

// the note editor has a clip for every section of the song: the one chosen is edited, the chords of the harmony show over it, a clip is copied to other sections
func TestPageNoteRollClips(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	song, _ := pt.Add("song", "", 60, 200)
	if err := pt.SetText(song, "sections", "Verse 2, Chorus 2, Bridge 1"); err != nil {
		t.Fatal(err)
	}
	harm, _ := pt.Add("harmony", "", 60, 300)
	if err := pt.SetText(harm, "chords", "Verse: Dm7 G7\nChorus: C Am"); err != nil {
		t.Fatal(err)
	}
	id, err := pt.Add("noteroll", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	roll := pt.Module(id).(*noteroll.Roll)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	canvas := `S().querySelector(".noteroll canvas")`
	tb.WaitFor(`!!`+canvas+` && `+canvas+`.clientHeight > 300 && `+canvas+`.rollState().sections === 3`, "the roll, with the sections of the song")
	// the clip of the chorus: eight beats (two bars), the chords of the chorus over it
	tb.Eval(`(() => { const s = S().querySelector(".nr-view select"); s.value = "Chorus"; s.dispatchEvent(new (S().defaultView.Event)("change")); })()`)
	tb.WaitFor(`JSON.stringify(`+canvas+`.rollState().view) === '"Chorus"' && `+canvas+`.rollState().chords === 2`, "the chorus, with two chords")
	// a note drawn there is in the clip of the chorus, not in the loop
	tb.Eval(`S().querySelector('.nr-tool[data-tool="draw"]').click()`)
	left := tb.Eval(fmt.Sprintf(`(() => { const g = document.querySelector(".pt-win-frame").getBoundingClientRect(), f0 = W().getElementById("frame").getBoundingClientRect(), r = %s.getBoundingClientRect();
		return [g.left + f0.left + r.left, g.top + f0.top + r.top, r.width, r.height]; })()`, canvas)).([]any)
	cl, ct, cw, chh := left[0].(float64), left[1].(float64), left[2].(float64), left[3].(float64)
	areaH := chh - 34 - 70 - 6
	x, y := cl+52+(cw-52)/8*1.1, ct+34+areaH/2 // the first beat, half way up
	tb.Mouse("mousePressed", x, y, nil)
	tb.Mouse("mouseReleased", x, y, nil)
	paneltest.Eventually(t, "a note in the clip of the chorus", func() bool {
		c, ok := roll.Clip("Chorus")
		return ok && len(c.Notes) == 1 && c.Length == 8 && len(roll.Notes()) == 0
	})
	// copied to the verse and the bridge
	tb.Eval(`(() => { const s = S().querySelector(".nr-view").parentNode.querySelectorAll("select")[1]; s.value = "*"; s.dispatchEvent(new (S().defaultView.Event)("change")); })()`)
	paneltest.Eventually(t, "the clip copied to the other sections", func() bool {
		v, ok1 := roll.Clip("Verse")
		br, ok2 := roll.Clip("Bridge")
		return ok1 && ok2 && len(v.Notes) == 1 && len(br.Notes) == 1
	})
	// a clip taken away
	tb.WaitFor(`!S().querySelector("button[title^='take the clip']").disabled`, "the button that takes the clip away")
	tb.Eval(`S().querySelector("button[title^='take the clip']").click()`)
	paneltest.Eventually(t, "the clip of the chorus taken away", func() bool { _, ok := roll.Clip("Chorus"); return !ok })
}

// the chord model: the matrix and the progression on the page, a chord locked, another chosen, a new progression drawn, the progression written into the harmony of the song
func TestPageChordModel(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	song, _ := pt.Add("song", "", 60, 200)
	if err := pt.SetText(song, "sections", "Verse 4, Chorus 8"); err != nil {
		t.Fatal(err)
	}
	harm, _ := pt.Add("harmony", "", 60, 300)
	id, err := pt.Add("chordmodel", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	m := pt.Module(id).(*chordmodel.Module)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 1100)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	tb.WaitFor(`!!S() && S().querySelectorAll(".cm-card").length > 2 && S().querySelectorAll(".cm-cell").length >= 56`, "the progression and the matrix")
	want := len(m.State().Strip)
	if n := tb.Eval(`S().querySelectorAll(".cm-card").length`).(float64); int(n) != want {
		t.Errorf("%v cards for %d chords", n, want)
	}
	// lock the first chord
	tb.Eval(`S().querySelector(".cm-lock").click()`)
	paneltest.Eventually(t, "the first chord locked", func() bool { return m.State().Strip[0].Lock })
	first := m.State().Strip[0].State
	tb.Eval(`[...S().querySelectorAll("button")].find(b => b.textContent.startsWith("Draw the others")).click()`)
	paneltest.Eventually(t, "the others drawn again, the locked chord kept", func() bool { s := m.State(); return s.Seed > 2 && s.Strip[0].State == first && s.Strip[0].Lock })
	// another chord chosen from the list of a card: it is locked
	tb.Eval(`(() => { const s = S().querySelectorAll(".cm-card-chord")[2]; s.value = "3"; s.dispatchEvent(new (S().defaultView.Event)("change")); })()`)
	paneltest.Eventually(t, "a chord chosen", func() bool { s := m.State().Strip[2]; return s.State == 3 && s.Lock })
	// a cell of the matrix dragged up: the chord after I is likelier to be ii (the cells: a corner and 7 heads, then the start row, then a head and the cells of I)
	before := m.State().Model.W[0][1]
	cell := `S().querySelectorAll(".cm-cell")[7 + 1]`
	tb.Eval(cell + `.scrollIntoView({block: "center"})`)
	pos := tb.Eval(fmt.Sprintf(`(() => { const g = document.querySelector(".pt-win-frame").getBoundingClientRect(), f0 = W().getElementById("frame").getBoundingClientRect(), c = %s.getBoundingClientRect();
		return [g.left + f0.left + c.left + c.width / 2, g.top + f0.top + c.top + c.height / 2]; })()`, cell)).([]any)
	x, y := pos[0].(float64), pos[1].(float64)
	tb.Mouse("mousePressed", x, y, nil)
	tb.Mouse("mouseMoved", x, y-30, nil)
	tb.Mouse("mouseReleased", x, y-30, nil)
	paneltest.Eventually(t, "a cell of the matrix dragged up", func() bool { return m.State().Model.W[0][1] > before+3 })
	// the progression written to the harmony, into the line of the chorus
	tb.Eval(`(() => { const s = [...S().querySelectorAll("select")].find(s => [...s.options].some(o => o.value === "Chorus")); s.value = "Chorus"; s.dispatchEvent(new (S().defaultView.Event)("change")); })()`)
	paneltest.Eventually(t, "the chorus chosen", func() bool { return m.State().Target == "Chorus" })
	tb.Eval(`[...S().querySelectorAll("button")].find(b => b.textContent.startsWith("Write to")).click()`)
	paneltest.Eventually(t, "the chords in the harmony", func() bool {
		for _, tx := range pt.Texts(harm) {
			if tx.Key == "chords" {
				st := m.State()
				return strings.HasPrefix(tx.Value, "Dm7 G7 Cmaj7 Fmaj7") && strings.Contains(tx.Value, "Chorus: "+st.Text())
			}
		}
		return false
	})
}

// the arpeggiator is on the page: a module with its notes in, clock in and notes out, and its settings in the side panel
func TestPageArp(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("arp", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	node := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s']")`, id)
	tb.WaitFor(node+`.querySelectorAll(".pt-port").length >= 3`, "its ports: notes and clock in, notes out")
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	frame := `S()`
	tb.WaitFor(`!!`+frame+` && !!`+frame+`.body && `+frame+`.body.innerText.includes("octaves")`, "its settings in the side panel")
}

// the looper's side panel shows the bars of the loop and its layers, each with a button that takes it away
func TestPageLooper(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("looper", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	pt.SettingsOf(id).Apply(map[string]float64{"bars": 3})
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	frame := `S()`
	tb.WaitFor(frame+`.querySelectorAll(".looper .lp-bar").length === 3`, "a cell for every bar of the loop")
	tb.WaitFor(`/bar \d, beat \d/.test(`+frame+`.querySelector(".looper .lp-status").textContent)`, "the bar and beat it is at")

	// notes from a clock go through it: a layer is recorded and has its row and button; the button takes it away
	clock, err := pt.Add("clock", "", 60, 200)
	if err != nil {
		t.Fatal(err)
	}
	notes, err := pt.Add("notes", "", 260, 200)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: clock, To: notes}, {From: notes, To: id}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	pt.SettingsOf(id).Apply(map[string]float64{"bars": 1, "record": 1})
	tb.WaitFor(frame+`.querySelector(".looper .lp-status").textContent.startsWith("recording layer")`, "the page to say it records")
	pt.SettingsOf(id).Apply(map[string]float64{"record": 0})
	tb.WaitFor(frame+`.querySelectorAll(".looper .lp-layer button").length === 1`, "a row for the layer recorded")
	tb.Eval(frame + `.querySelector(".looper .lp-layer button").click()`)
	tb.WaitFor(frame+`.querySelectorAll(".looper .lp-layer button").length === 0`, "the layer taken away")

	// a switch that lets go by itself is let go on the page too: lit when clicked, and out again
	find := `[...` + frame + `.querySelectorAll("button")].find(b => b.textContent === "export to MIDI")`
	tb.Eval(find + `.click()`)
	tb.WaitFor(find+`.classList.contains("on")`, "the export switch lit")
	tb.WaitFor(`!`+find+`.classList.contains("on")`, "the export switch let go")
}

// the looper for sound has the same page: the bars, and a row for every layer recorded, each with a button that takes it away
func TestPageAudioLooper(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("audiolooper", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pt.Connect(patcher.Cable{From: id, To: "out"}); err != nil { // only what is pulled by the output runs
		t.Fatal(err)
	}
	pt.SettingsOf(id).Apply(map[string]float64{"bars": 1})
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	frame := `S()`
	tb.WaitFor(frame+`.querySelectorAll(".looper .lp-bar").length === 1`, "a cell for the bar of the loop")
	pt.SettingsOf(id).Apply(map[string]float64{"record": 1})
	tb.WaitFor(frame+`.querySelector(".looper .lp-status").textContent.startsWith("recording layer")`, "the page to say it records")
	pt.SettingsOf(id).Apply(map[string]float64{"record": 0})
	tb.WaitFor(frame+`.querySelectorAll(".looper .lp-layer button").length === 1`, "a row for the layer recorded")
	tb.Eval(frame + `.querySelector(".looper .lp-layer button").click()`)
	tb.WaitFor(frame+`.querySelectorAll(".looper .lp-layer button").length === 0`, "the layer taken away")
	// a click on a switch that lets go by itself lights it, and it goes out again
	find := `[...` + frame + `.querySelectorAll("button")].find(b => b.textContent === "export to WAV")`
	tb.Eval(find + `.click()`)
	tb.WaitFor(find+`.classList.contains("on")`, "the export switch lit")
	tb.WaitFor(`!`+find+`.classList.contains("on")`, "the export switch let go")
}

// the lanes of an automation module over the whole width at the bottom of the patch: a click on a row adds a point, a drag moves it, a double-click gives it
// the next shape, Delete takes it away (and not the modules chosen), a name is kept; the song is on the ruler and the heads move with it
func TestPageLanes(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	song, err := pt.Add("song", "", 60, 200)
	if err != nil {
		t.Fatal(err)
	}
	if err := pt.SetText(song, "sections", "Intro 4, Verse 4, Chorus 8"); err != nil {
		t.Fatal(err)
	}
	id, err := pt.Add("automation", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	a := pt.Module(id).(*patcher.Automation)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length >= 2`)
	tb.Size(1300, 900)
	tb.Tap(`document.querySelector(".pt-lanes-btn")`)
	rows := `document.querySelectorAll(".pt-lanes .pt-lane-row:not(.ruler)")`
	tb.WaitFor(`!document.querySelector(".pt-lanes").hidden && `+rows+`.length === 4`, "a row for every lane")
	tb.WaitFor(`document.querySelector(".pt-lanes").getBoundingClientRect().width > 1000`, "the lanes over the whole width")
	canvas := func(i int) string { return fmt.Sprintf(`%s[%d].querySelector("canvas")`, rows, i) }

	// the song is on the ruler: the length of the lanes is its 16 bars; a head moves with it
	tb.WaitFor(`!`+rows+`[0].querySelector(".pt-lane-head").hidden`, "the head of the lane")

	// a click in the middle of the first lane: a point at bar 8, the middle of the value
	x, y := tb.Center(canvas(0), 0.5)
	tb.Mouse("mousePressed", x, y, nil)
	tb.Mouse("mouseReleased", x, y, nil)
	paneltest.Eventually(t, "a point added", func() bool { return len(a.Lanes()[0].Points) == 1 })
	if pt0 := a.Lanes()[0].Points[0]; pt0.X != 8 || pt0.Y < 0.4 || pt0.Y > 0.6 {
		t.Errorf("the point is at %+v, want bar 8 and about 0.5", pt0)
	}

	// dragged a quarter of the width on, and up
	r := tb.Eval(canvas(0) + `.getBoundingClientRect().width`).(float64)
	_, top := tb.Center(canvas(0), 0.1)
	tb.Mouse("mousePressed", x, y, nil)
	tb.Mouse("mouseMoved", x+r/4, top, nil)
	tb.Mouse("mouseReleased", x+r/4, top, nil)
	paneltest.Eventually(t, "the point moved", func() bool { l := a.Lanes()[0].Points; return len(l) == 1 && l[0].X == 12 && l[0].Y > 0.8 })

	// a second point on the first lane makes a line; a double-click on it gives it a shape
	tb.Mouse("mousePressed", x-r/4, y, nil)
	tb.Mouse("mouseReleased", x-r/4, y, nil)
	paneltest.Eventually(t, "a second point", func() bool { return len(a.Lanes()[0].Points) == 2 })
	tb.Eval(fmt.Sprintf(`%s.dispatchEvent(new MouseEvent("dblclick", {clientX: %g, clientY: %g, bubbles: true}))`, canvas(0), x-r/4, y))
	paneltest.Eventually(t, "the next shape", func() bool { return a.Lanes()[0].Points[0].Shape == "smooth" })
	if v := a.Value(0); v < 0 || v > 1 {
		t.Errorf("a value of %v", v)
	}

	// Delete takes the point chosen away, and the module of the patch stays (the patch would have taken it)
	tb.Key("Delete")
	paneltest.Eventually(t, "the point taken away", func() bool { return len(a.Lanes()[0].Points) == 1 })
	if len(pt.Nodes()) != 3 {
		t.Errorf("%d modules after Delete on the lanes, want the 3 there were", len(pt.Nodes()))
	}

	// a name typed is kept, the text of the lanes has what was drawn
	tb.Eval(`(() => { const i = document.querySelector(".pt-lanes .pt-lane-name"); i.value = "Cutoff"; i.dispatchEvent(new Event("change")); })()`)
	paneltest.Eventually(t, "the name kept", func() bool { return a.Lanes()[0].Name == "Cutoff" })
	if txt := pt.Texts(id)[0].Value; !strings.HasPrefix(txt, "Cutoff: 12 ") {
		t.Errorf("the text of the lanes:\n%s", txt)
	}

	// what is typed in the module's box shows in the lanes
	if err := pt.SetText(id, "lanes", "Wide: 0 0, 16 1\nTwo: 4 1"); err != nil {
		t.Fatal(err)
	}
	tb.WaitFor(`document.querySelector(".pt-lanes .pt-lane-name").value === "Wide"`, "the lanes typed on the page")

	// closed again
	tb.Tap(`document.querySelector(".pt-lanes-btn")`)
	tb.WaitFor(`document.querySelector(".pt-lanes").hidden`, "the lanes closed")
}

// a lane armed on the page records the setting its cable moves while the song plays: the button lights, the move is written into the lane and drawn
func TestPageLaneRecord(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	song, _ := pt.Add("song", "", 60, 200)
	if err := pt.SetText(song, "sections", "A 8"); err != nil {
		t.Fatal(err)
	}
	id, _ := pt.Add("automation", "", 60, 40)
	fx, err := pt.Add("chorus", "", 400, 40)
	if err != nil {
		t.Fatal(err)
	}
	key := "mix" // 0 to 1
	cable := patcher.Cable{Type: patcher.Mod, From: id, Out: 0, To: fx, In: 1, Key: key}
	if _, err := pt.Connect(cable); err != nil {
		t.Fatal(err)
	}
	a := pt.Module(id).(*patcher.Automation)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length >= 3`)
	tb.Size(1300, 900)
	tb.Tap(`document.querySelector(".pt-lanes-btn")`)
	rec := `document.querySelectorAll(".pt-lanes .pt-lane-row:not(.ruler)")[0].querySelector(".pt-lane-rec")`
	tb.WaitFor(fmt.Sprintf(`document.querySelector(".pt-lanes .pt-lane-target").textContent === "→ %s · %s"`, fx, key), "the lane says where it goes")
	tb.Tap(rec)
	tb.WaitFor(rec+`.classList.contains("armed")`, "the lane armed")
	if a.Armed() != 0 {
		t.Fatal("not armed")
	}
	// the setting played: from low to high over a second
	for i := 0; i <= 10; i++ {
		pt.SettingsOf(fx).Apply(map[string]float64{key: float64(i) / 10 * 0.9})
		time.Sleep(100 * time.Millisecond)
	}
	tb.WaitFor(`document.querySelector(".pt-lanes").querySelector(".pt-lane-rec.armed")`, "still armed")
	paneltest.Eventually(t, "a move written", func() bool { return len(a.Lanes()[0].Points) >= 2 })
	tb.Tap(rec) // off
	tb.WaitFor(`!`+rec+`.classList.contains("armed")`, "the lane let go")
	if a.Armed() != -1 {
		t.Error("still armed")
	}
	if got := a.Lanes()[0].Points; len(got) < 2 || got[0].X == got[len(got)-1].X || got[len(got)-1].Y < got[0].Y+0.6 { // played from 0 to 0.9 of the range
		t.Errorf("the points written: %+v", got)
	}
}

// the melody panel: a melody generated for the clip, six variations to choose from, a mutation of the notes selected
func TestPageNoteRollMelody(t *testing.T) {
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("noteroll", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	roll := pt.Module(id).(*noteroll.Roll)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pt-node")`)
	tb.Size(1300, 900)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	tb.WaitFor(`!!S().querySelector(".nr-gen")`, "the melody panel")
	tb.Eval(`S().querySelector(".nr-gen").open = true`)
	btn := func(title string) string {
		return fmt.Sprintf(`S().querySelector(".nr-gen button[title^='%s']")`, title)
	}
	tb.Eval(btn("make a melody") + `.click()`)
	paneltest.Eventually(t, "a melody in the clip", func() bool { return len(roll.Notes()) > 3 })
	// six to choose from; one used replaces the notes
	tb.Eval(btn("make six") + `.click()`)
	tb.WaitFor(`S().querySelectorAll(".nr-cand").length === 6`, "six melodies")
	tb.Eval(`S().querySelector(".nr-cand button[title='put it in the clip']").click()`)
	paneltest.Eventually(t, "the candidate in the clip", func() bool { return len(roll.Notes()) > 3 })
	// reversed: the same notes, the first one now where the last was
	first := roll.Notes()
	time.Sleep(600 * time.Millisecond)
	first = roll.Notes()
	tb.Eval(btn("play the notes") + `.click()`)
	paneltest.Eventually(t, "the notes reversed", func() bool {
		now := roll.Notes()
		return len(now) == len(first) && fmt.Sprint(now) != fmt.Sprint(first)
	})
	// a groove puts the sixteenths a little late: some note off the grid; one taken from them is a groove of its own
	tb.WaitFor(`[...S().querySelectorAll(".nr-gen select option")].some(o => o.textContent === "mpc 62")`, "the grooves")
	tb.Eval(`(() => { const s = [...S().querySelectorAll(".nr-gen select")].find(s => [...s.options].some(o => o.textContent === "mpc 62")); s.value = [...s.options].find(o => o.textContent === "mpc 62").value; })()`)
	tb.Eval(btn("put the groove") + `.click()`)
	paneltest.Eventually(t, "notes off the grid", func() bool {
		for _, n := range roll.Notes() {
			if math.Abs(n.Start*4-math.Round(n.Start*4)) > 0.05 {
				return true
			}
		}
		return false
	})
	tb.Eval(btn("take the feel") + `.click()`)
	tb.WaitFor(`[...S().querySelectorAll(".nr-gen select option")].some(o => o.textContent === "mine 1")`, "a groove taken from the notes")
	// a counter-line adds a voice
	n0 := len(roll.Notes())
	tb.Eval(btn("add a second voice") + `.click()`)
	paneltest.Eventually(t, "a second voice", func() bool { return len(roll.Notes()) > n0 })

}
