package widgets

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
)

func TestMain(m *testing.M) { paneltest.Main(m) }

// notes keeps what a keyboard plays
type notes struct {
	mu   sync.Mutex
	held map[int]bool
	ons  int
	vel  float64 // of the last note on
}

func (n *notes) play(note int, velocity float64, on bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.held[note] = on
	if on {
		n.ons++
		n.vel = velocity
	}
}

func (n *notes) velocity() float64 {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.vel
}

func (n *notes) sounding() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	c := 0
	for _, on := range n.held {
		if on {
			c++
		}
	}
	return c
}

// a key goes down and plays; every way of leaving it lets go: up, up away from the keys, the page
// scrolled under it, the page hidden; sliding over the keys plays them
func TestPageKeys(t *testing.T) {
	b := paneltest.Start(t)
	pn := panel.New(psalm.New(nil), "Keys")
	n := &notes{held: map[int]bool{}}
	Keys(pn, "Keys", n.play)
	pn.Section("Below") // a page long enough to scroll
	for i := range 12 {
		pn.Fader(fmt.Sprint("f", i), 0, 1, panel.Linear, nil)
	}
	tick := func() { pn.CtrlTick(psalm.CtrlTick{SampleTimestamp: 1}) }
	sounding := func(want int) func() bool { return func() bool { tick(); return n.sounding() == want } }
	tb := b.Open(t, paneltest.Serve(t, pn.Handler()), `document.querySelector(".keys .white")`)
	tb.Size(900, 500)
	key := func(note int) (float64, float64) {
		return tb.Center(fmt.Sprintf(`document.querySelector(".keys [data-note='%d']")`, note), 0.8)
	}

	x, y := key(52)
	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "a key to play", sounding(1))
	tb.Mouse("mouseReleased", x, y, nil)
	paneltest.Eventually(t, "letting go of it", sounding(0))

	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "a key to play", sounding(1))
	tb.Mouse("mouseWheel", x, y, map[string]any{"deltaX": 0, "deltaY": 300})
	paneltest.Eventually(t, "the page scrolling under it to let go", sounding(0))
	tb.Mouse("mouseReleased", x, y, nil)
	tb.Eval(`window.scrollTo(0, 0)`)
	tb.WaitFor(`window.scrollY === 0`, "the page back at the top") // before a key's place is taken

	x, y = key(55)
	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "a key to play", sounding(1))
	tb.Mouse("mouseMoved", x, y+400, nil)
	tb.Mouse("mouseReleased", x, y+400, nil)
	paneltest.Eventually(t, "letting go away from the keys", sounding(0))

	before := n.ons
	x, y = key(48)
	x2, _ := key(57)
	tb.Mouse("mousePressed", x, y, nil)
	for i := 1; i <= 12; i++ {
		tb.Mouse("mouseMoved", x+(x2-x)*float64(i)/12, y, nil)
	}
	tb.Mouse("mouseReleased", x2, y, nil)
	paneltest.Eventually(t, "the glissando to let go at its end", sounding(0))
	if played := n.ons - before; played < 5 {
		t.Errorf("sliding over six white keys played %d", played)
	}

	x, y = key(50)
	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "a key to play", sounding(1))
	tb.Eval(`Object.defineProperty(document, "hidden", {value: true, configurable: true}); document.dispatchEvent(new Event("visibilitychange")); true`)
	paneltest.Eventually(t, "the page hidden to let go", sounding(0))
	tb.Mouse("mouseReleased", x, y, nil)
}

// the XY pad moves both values, the pads hit, the envelope drags its stages, the scope draws
func TestPageWidgets(t *testing.T) {
	b := paneltest.Start(t)
	pn := panel.New(psalm.New(nil), "Widgets")
	x, y := XY(pn, "Pad", Axis{Name: "cutoff", Min: 100, Max: 10000, Curve: panel.Exp, Default: 1000}, Axis{Name: "res", Max: 1, Default: 0.5})
	var mu sync.Mutex
	var hits []int
	Pads(pn, "Pads", Hitter{Names: []string{"kick", "snare", "hat", "clap"}, Hit: func(pad int, _ float64) { mu.Lock(); hits = append(hits, pad); mu.Unlock() }})
	synth := &recorder{got: map[string]float64{}}
	env := Envelope(pn, "Amp", synth, "amp", ADSR{Attack: 10, Decay: 200, Sustain: 0.5, Release: 300})
	src := psalm.NewModule(nil, []int{psalm.Stereo})
	for i := range src.Output(0).Channels[0] {
		v := float32(math.Sin(2 * math.Pi * 440 * float64(i) / 44100))
		src.Output(0).Channels[0][i], src.Output(0).Channels[1][i] = v, v
	}
	Scope(pn, "Scope", src)
	tick := func() { pn.CtrlTick(psalm.CtrlTick{SampleTimestamp: int64(pn.Config().BufferSize) + 1}) }
	tick()
	tb := b.Open(t, paneltest.Serve(t, pn.Handler()), `document.querySelector(".xypad") && document.querySelector("canvas.envelope")`)
	tb.Size(1000, 900)

	// the XY pad: the top right corner is the top of both ranges
	px, py := tb.Center(`document.querySelector(".xypad")`, 0.5)
	tb.Mouse("mousePressed", px, py, nil)
	tb.Mouse("mouseMoved", px+300, py-300, nil)
	tb.Mouse("mouseReleased", px+300, py-300, nil)
	paneltest.Eventually(t, "the pad to move both", func() bool { return x.Value() == 10000 && y.Value() == 1 })

	// the pads: a tap on the third
	tb.Tap(`document.querySelectorAll(".pads .pad")[2]`)
	paneltest.Eventually(t, "the pad hit", func() bool { tick(); mu.Lock(); defer mu.Unlock(); return len(hits) == 1 && hits[0] == 2 })

	// the envelope: press near its bottom right (the release point is the nearest handle there) and
	// drag to the right: a longer release
	before := env.Parts()[2].Value()
	r := tb.Eval(`(() => { const r = document.querySelector("canvas.envelope").getBoundingClientRect(); return [r.left, r.top, r.width, r.height]; })()`).([]any)
	left, top, w, h := r[0].(float64), r[1].(float64), r[2].(float64), r[3].(float64)
	tb.Mouse("mousePressed", left+w*0.8, top+h-10, nil)
	tb.Mouse("mouseMoved", left+w-2, top+h-10, nil)
	tb.Mouse("mouseReleased", left+w-2, top+h-10, nil)
	paneltest.Eventually(t, "the release to grow", func() bool { return env.Parts()[2].Value() > before })
	tick()
	if synth.get("ampRelease") <= 300 {
		t.Errorf("the synth's release: %v", synth.get("ampRelease"))
	}

	// the scope: a wave drawn (pixels other than the background in its canvas)
	for range 6 {
		tick()
	}
	tb.WaitFor(`(() => { const c = document.querySelector("canvas.scope"); const d = c.getContext("2d").getImageData(0, 0, c.width, c.height).data; const bg = [d[0], d[1], d[2]]; for (let i = 0; i < d.length; i += 4 * 7) if (Math.abs(d[i] - bg[0]) + Math.abs(d[i + 1] - bg[1]) + Math.abs(d[i + 2] - bg[2]) > 60) return true; return false; })()`, "the scope to draw the wave")
}

// recorder keeps the last value of every key it gets
type recorder struct {
	mu  sync.Mutex
	got map[string]float64
}

func (r *recorder) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		r.ReceiveMessage(m)
	}
}

func (r *recorder) ReceiveMessage(m psalm.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, x := range m {
		r.got[k] = psalm.FltVal[float64](x)
	}
}

func (r *recorder) get(k string) float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.got[k]
}

// the keys in a sidebar, upright: played there (louder further along a key), hidden to a tab and
// back from it, kept so through a reload, back on the page
func TestPageKeysSidebar(t *testing.T) {
	b := paneltest.Start(t)
	pn := panel.New(psalm.New(nil), "Keys")
	n := &notes{held: map[int]bool{}}
	Keys(pn, "Keys", n.play)
	tick := func() { pn.CtrlTick(psalm.CtrlTick{}) }
	sounding := func(want int) func() bool { return func() bool { tick(); return n.sounding() == want } }
	tb := b.Open(t, paneltest.Serve(t, pn.Handler()), `document.querySelector(".keys .white")`)
	tb.Size(900, 700)
	tb.Eval(`localStorage.clear()`)

	tb.Tap(`document.querySelector(".keys-dock")`)
	tb.WaitFor(`(() => { const s = document.querySelector(".keys-side"); return s && !s.hidden && s.contains(document.querySelector(".keys.upright"))
		&& parseFloat(getComputedStyle(document.body).paddingRight) === s.offsetWidth; })()`, "the keys upright in the sidebar, the page making room")
	key := `document.querySelector(".keys-side [data-note='52']")`
	near := func(part float64) (float64, float64) {
		pos := tb.Eval(`(() => { const r = ` + key + `.getBoundingClientRect(); return [r.left + r.width * ` + fmt.Sprint(part) + `, r.top + r.height / 2]; })()`).([]any)
		return pos[0].(float64), pos[1].(float64)
	}
	x, y := near(0.9)
	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "a key played in the sidebar", sounding(1))
	tb.Mouse("mouseReleased", x, y, nil)
	paneltest.Eventually(t, "let go", sounding(0))
	loud := n.velocity()
	x, y = near(0.75)
	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "played again", sounding(1))
	tb.Mouse("mouseReleased", x, y, nil)
	paneltest.Eventually(t, "let go", sounding(0))
	if soft := n.velocity(); !(loud > soft) {
		t.Errorf("upright, further along the key is louder: %.2f at 90%%, %.2f at 75%%", loud, soft)
	}

	// hidden to a tab, back from it
	tb.Tap(`document.querySelector(".keys-side-head button")`)
	tb.WaitFor(`document.querySelector(".keys-side").hidden && !document.querySelector(".keys-tab").hidden`, "the sidebar hidden, its tab there")
	tb.Tap(`document.querySelector(".keys-tab")`)
	tb.WaitFor(`!document.querySelector(".keys-side").hidden`, "the sidebar back")

	// kept through a reload
	tb.Eval(`location.reload()`)
	tb.WaitFor(`(() => { const s = document.querySelector(".keys-side"); return s && !s.hidden && s.querySelector(".keys.upright"); })()`, "the sidebar after a reload")

	// back on the page
	tb.Tap(`document.querySelectorAll(".keys-side-head button")[1]`)
	tb.WaitFor(`!document.querySelector(".keys-side") && !document.querySelector(".keys.upright") && document.querySelector(".keys-wrap .keys")
		&& parseFloat(getComputedStyle(document.body).paddingRight) === 0`, "the keys back on the page")
}

// the keys at the top of the page: a button showing them in the sidebar (hidden at first), played
// there, hidden again; they don't go back onto the page
func TestPageKeysTop(t *testing.T) {
	b := paneltest.Start(t)
	pn := panel.New(psalm.New(nil), "Keys")
	n := &notes{held: map[int]bool{}}
	Keys(pn, "Keys", n.play).Top()
	pn.Section("Other")
	pn.Knob("Drive", 0, 1, panel.Linear, nil)
	sounding := func(want int) func() bool {
		return func() bool { pn.CtrlTick(psalm.CtrlTick{}); return n.sounding() == want }
	}
	tb := b.Open(t, paneltest.Serve(t, pn.Handler()), `document.querySelector("#top .keys-toggle")`)
	tb.Size(900, 700)
	tb.Eval(`localStorage.clear()`)
	tb.WaitFor(`!document.querySelector(".keys-side") || document.querySelector(".keys-side").hidden`, "the sidebar hidden at first")
	tb.Tap(`document.querySelector("#top .keys-toggle")`)
	tb.WaitFor(`!document.querySelector(".keys-side").hidden && document.querySelector(".keys-side .keys.upright") && document.querySelector("#top .keys-toggle").classList.contains("on")`, "the keys in the sidebar")
	if tb.Eval(`[...document.querySelectorAll(".keys-side-head button")].some(b => !b.hidden && b.textContent === "×")`).(bool) {
		t.Error("keys at the top stay in the sidebar: no × to put them back on the page")
	}
	x, y := tb.Center(`document.querySelector(".keys-side [data-note='52']")`, 0.5)
	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "a key played", sounding(1))
	tb.Mouse("mouseReleased", x, y, nil)
	paneltest.Eventually(t, "let go", sounding(0))
	tb.Tap(`document.querySelector("#top .keys-toggle")`)
	tb.WaitFor(`document.querySelector(".keys-side").hidden && !document.querySelector("#top .keys-toggle").classList.contains("on")`, "the sidebar hidden by the button")
	tb.Eval(`localStorage.clear()`)
}
