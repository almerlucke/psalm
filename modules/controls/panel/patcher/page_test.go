package patcher_test

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
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
	// a tap on a module chooses it; a double tap: its settings in the side panel (its editor,
	// served by the patcher's mount)
	lfoHead := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, lfo)
	tb.Tap(lfoHead)
	tb.WaitFor(lfoHead+`.parentNode.classList.contains("selected")`, "the LFO chosen")
	if !tb.Eval(`document.querySelector(".pt-drawer").hidden`).(bool) {
		t.Error("a tap chooses a module, it doesn't open its side panel")
	}
	doubleTap(tb, lfoHead)
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden && document.querySelector(".pt-drawer-frame").contentDocument?.querySelector(".control")`, "the LFO's settings in the side panel")
	if tb.Eval(`!!document.querySelector("section.mounthost")`).(bool) {
		t.Error("the settings of the module chosen are in the side panel, not in a section below the patch")
	}
	// a module taken away by the Delete key, its cables with it, its side panel closed
	tb.Key("Delete")
	paneltest.Eventually(t, "the LFO gone", func() bool { return len(pt.Nodes()) == 3 })
	tb.WaitFor(`document.querySelector(".pt-drawer").hidden`, "its side panel closed")

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
	input := `document.querySelector(".pt-drawer-text")`
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden && `+input+`?.value === "x" && document.querySelector(".pt-drawer-frame").contentDocument?.querySelector(".control")`, "the pattern and the clock's settings in its side panel")
	enter := func(text string) {
		tb.Eval(`(() => { const i = ` + input + `; i.focus(); i.value = ` + strconv.Quote(text) + `; i.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	}
	if !tb.Eval(`!!([...document.querySelectorAll(".pt-drawer .pt-help li")].some(li => li.textContent.includes("a hit")) && document.querySelector(".pt-help code"))`).(bool) {
		t.Error("what to know to type a pattern is not above its box")
	}
	if !tb.Eval(`document.querySelector(".pt-drawer-text").getBoundingClientRect().right <= document.querySelector(".pt-drawer").getBoundingClientRect().right - 8`).(bool) {
		t.Error("the box of a text runs past the side panel")
	}
	enter("x . x x\nx x . .")
	paneltest.Eventually(t, "the pattern taken", func() bool { tx := pt.Texts(clk); return len(tx) == 1 && tx[0].Value == "x . x x\nx x . ." })
	if e := pt.TextError(clk, "pattern"); e != "" {
		t.Errorf("a pattern over two lines: %s", e)
	}
	enter("x ?")
	tb.WaitFor(input+`.classList.contains("bad") && document.querySelector(".pt-drawer .pt-status").textContent.startsWith("error")`, "a bad pattern shown, what is wrong below it")
	tb.Key("Escape")
	tb.WaitFor(`document.querySelector(".pt-drawer").hidden`, "the side panel closed")

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
	box, status := `document.querySelector(".pt-drawer-text")`, `document.querySelector(".pt-drawer .pt-status")`
	onNode := node + `.querySelector(".pt-status")`
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden && `+box+`.value.includes("function trigger")`, "the script in the side panel")
	tb.WaitFor(`[...document.querySelectorAll(".pt-drawer .pt-help li")].some(li => li.textContent.includes("function trigger(ev)"))`, "what to know to write a script above it")
	run := func(code string) {
		tb.Eval(`(() => { const b = ` + box + `; b.focus(); b.value = ` + strconv.Quote(code) + `; b.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	}
	run("const a = 1\nlet b = (;")
	tb.WaitFor(box+`.classList.contains("bad") && `+status+`.classList.contains("bad") && `+status+`.textContent.includes("line 2") && `+onNode+`?.textContent.includes("line 2")`, "the error on its line")
	run("log('hello', [1, 2])")
	tb.WaitFor(status+`.textContent === 'hello [1,2]' && !`+status+`.classList.contains("bad") && !`+box+`.classList.contains("bad") && `+onNode+`.textContent === 'hello [1,2]'`, "what it logged")
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
	if !tb.Eval(`document.querySelector(".pt-drawer").hidden`).(bool) {
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
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden && document.querySelector(".pt-drawer-text").value.includes("function trigger")`, "the script in the side panel")
	tb.Eval(`(() => { const d = document.querySelector(".pt-drawer-text"); d.focus(); d.value = "log('wide')"; d.dispatchEvent(new KeyboardEvent("keydown", {key: "Enter", ctrlKey: true, bubbles: true})); })()`)
	tb.WaitFor(`document.querySelector(".pt-drawer .pt-status").textContent === "wide"`, "what it logs in the side panel")
	if tx := pt.Texts(scr); len(tx) != 1 || tx[0].Value != "log('wide')" {
		t.Fatalf("the script typed in the side panel: %v", tx)
	}
	tb.Key("Escape")
	tb.WaitFor(`document.querySelector(".pt-drawer").hidden`, "the side panel closed")

	// the side panel open: a tap on another module shows that one, a double tap on it closes it
	doubleTap(tb, node(scr)+`.querySelector(".pt-head")`)
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden`, "the script's side panel again")
	tb.Tap(node(lfo) + `.querySelector(".pt-head")`)
	tb.WaitFor(`document.querySelector(".pt-drawer-title").textContent === "lfo" && document.querySelector(".pt-drawer-code").hidden`, "the LFO in the side panel, after a tap")
	doubleTap(tb, node(lfo)+`.querySelector(".pt-head")`)
	tb.WaitFor(`document.querySelector(".pt-drawer").hidden`, "the side panel closed by a double tap")
	doubleTap(tb, node(lfo)+`.querySelector(".pt-head")`)
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden`, "the LFO's side panel by a double tap")
	// a double tap on another module, the panel open: its first tap brings the panel, which stays
	doubleTap(tb, node(scr)+`.querySelector(".pt-head")`)
	if open := tb.Eval(`new Promise(r => setTimeout(() => r(!document.querySelector(".pt-drawer").hidden && document.querySelector(".pt-drawer-title").textContent.startsWith("script")), 150))`).(bool); !open {
		t.Error("a double tap on another module, the panel open, should show that module, not close the panel")
	}
	tb.Tap(`document.querySelector(".pt-drawer .pt-x")`)
	tb.WaitFor(`document.querySelector(".pt-drawer").hidden`, "the side panel closed by its ×")

	// the patch over the whole screen, and back
	if os.Getenv("SHOT") != "" {
		doubleTap(tb, node(scr)+`.querySelector(".pt-head")`)
		tb.Tap(`document.querySelector(".pt-max-btn")`)
		tb.Screenshot(os.Getenv("SHOT"))
		tb.Tap(`document.querySelector(".pt-drawer .pt-x")`)
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
	x, y := tb.Center(expr, 0.5)
	for _, n := range []int{1, 2} {
		tb.Mouse("mousePressed", x, y, map[string]any{"clickCount": n})
		tb.Mouse("mouseReleased", x, y, map[string]any{"clickCount": n})
	}
}

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
	frame := `document.querySelector(".pt-drawer-frame").contentDocument`
	tb.WaitFor(`document.querySelector(".pt-drawer-text")?.value === "Intro 64, Verse 64" && `+frame+`?.querySelectorAll(".cue").length === 2`, "its sections and their tiles in the side panel")
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
	frame := `document.querySelector(".pt-drawer-frame").contentDocument`
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden && `+frame+`?.querySelectorAll(".cell").length === 7*32`, "its grid in the side panel")
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
	tb.WaitFor(`document.querySelector(".pt-drawer-log")?.value.includes("trigger step")`, "the log in a box of the side panel")
	if !tb.Eval(`document.querySelector(".pt-drawer-log").readOnly && document.querySelector(".pt-drawer-text:not(.pt-drawer-log)").getBoundingClientRect().bottom <= document.querySelector(".pt-drawer-log").getBoundingClientRect().top`).(bool) {
		t.Error("the label above the log, the log read only")
	}
}
