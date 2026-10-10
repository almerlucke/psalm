package panel

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
)

// serve serves a panel for a page test
func serve(t *testing.T, pn *Panel) string {
	t.Helper()
	srv := httptest.NewServer(pn.Handler())
	t.Cleanup(srv.Close)
	return srv.URL + "/"
}

// built is true once the page built its sections
const built = `document.querySelectorAll("main section").length > 0`

// every kind of control on one page: it builds without an error
func TestPageBuilds(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	pn.Section("Controls")
	pn.Fader("Cutoff", 200, 12000, Exp, nil, Default(1000), Unit("Hz"))
	pn.Knob("Resonance", 0, 1, Linear, nil)
	pn.Toggle("Mute", nil)
	pn.Button("Bang", func() {})
	pn.Select("Wave", []string{"sine", "saw"}, func(int) {})
	pn.Meter("Out", nil)
	pn.Section("Beat")
	pn.Grid("Drums", []string{"kick", "snare"}, 8, 2, testFields...)
	pn.Section("Song")
	pn.Cues("Songs", newCueList())
	pn.Cues("Sections", &changing{cueList: *newCueList(), names: []string{"Intro", "Verse"}}, CueSlots(4))
	pn.Section("Mounted")
	pn.Mount("Song panel", &mounter{})
	_ = pn.Presets(t.TempDir())
	pl := &restartable{}
	pl.running.Store(true)
	pn.SetPlayer(pl)

	tb := b.open(t, serve(t, pn), built)
	tb.waitFor(`document.querySelectorAll(".cue").length === 5`, "the tiles of both cue lists")
	for _, sel := range []string{".fader", ".knob", ".meter", ".cell", ".mountbox", "#presets:not([hidden])", "#run:not([hidden])", "#restart:not([hidden])"} {
		if n, _ := tb.eval(`document.querySelectorAll("` + sel + `").length`).(float64); n == 0 {
			t.Errorf("the page should have %s", sel)
		}
	}
	if title := tb.eval(`document.title`); title != "Test" {
		t.Errorf("the title should be the panel's, got %v", title)
	}
}

// moving controls on the page moves them on the panel
func TestPageControls(t *testing.T) {
	b := startBrowser(t)
	pn, tk := newPanel()
	level := pn.Fader("Level", 0, 1, Linear, nil, Default(0.5))
	mute := pn.Toggle("Mute", nil)
	var presses atomic.Int32
	pn.Button("Bang", func() { presses.Add(1) })
	wave := pn.Select("Wave", []string{"sine", "saw", "square"}, func(int) {})
	tb := b.open(t, serve(t, pn), built)

	control := func(name string) string {
		return `[...document.querySelectorAll(".control")].find(c => c.querySelector(".name") && c.querySelector(".name").textContent === "` + name + `")`
	}
	tb.eval(control("Level") + `.querySelector(".fader").focus()`)
	tb.key("ArrowUp")
	eventually(t, "the fader to move up", func() bool { return level.Value() > 0.5 })

	tb.tap(`document.querySelector('button[title="Mute"]')`) // toggles and buttons are a button named by its title
	eventually(t, "the toggle to switch on", func() bool { return mute.Value() == 1 })

	tb.tap(`document.querySelector('button[title="Bang"]')`)
	eventually(t, "the button to be pressed", func() bool { tk.tick(1); return presses.Load() == 1 })

	tb.eval(`(() => { const s = ` + control("Wave") + `.querySelector("select"); s.value = "2"; s.dispatchEvent(new Event("change")); })()`)
	eventually(t, "the select to choose", func() bool { return wave.Value() == 2 })

	// and the other way: a change in the patch shows on the page
	level.Set(0.1)
	tb.waitFor(control("Level")+`.querySelector(".value").textContent === "0.10"`, "the page to follow the panel")
}

// a tap on a tile asks for its cue, a tap again takes it back, the dot switches it off
func TestPageCues(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	c := newCueList()
	pn.Cues("Songs", c)
	tb := b.open(t, serve(t, pn), built)
	tile := func(i string) string { return `document.querySelectorAll(".cue")[` + i + `]` }

	tb.tap(tile("2"))
	eventually(t, "the tile to ask for its cue", func() bool { return c.CueState().Queued == 2 })
	tb.waitFor(tile("2")+`.classList.contains("queued")`, "the tile to show it is next")
	tb.tap(tile("2"))
	eventually(t, "a second tap to take it back", func() bool { return c.CueState().Queued == -1 })

	tb.tap(tile("1") + `.querySelector(".switch")`)
	eventually(t, "the dot to switch the cue off", func() bool { return !c.CueState().Enabled[1] })
	tb.waitFor(tile("1")+`.classList.contains("off")`, "the tile to show it is off")
	if q := c.CueState().Queued; q != -1 {
		t.Fatalf("the dot should not ask for the cue, queued %d", q)
	}
}

// sometimes is a cue list whose cues come and go (the sections of the song a playlist plays, none
// between songs): its state has no on and off list without cues
type sometimes struct {
	mu     sync.Mutex
	names  []string
	queued int
}

func (s *sometimes) CueNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.names...)
}

func (s *sometimes) CueState() CueState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := CueState{Playing: -1, Queued: s.queued}
	if len(s.names) > 0 {
		st.Playing = 0
		st.Enabled = make([]bool, len(s.names))
		for i := range st.Enabled {
			st.Enabled[i] = true
		}
	}
	return st
}

func (s *sometimes) QueueCue(i int)      { s.mu.Lock(); s.queued = i; s.mu.Unlock() }
func (s *sometimes) EnableCue(int, bool) {}
func (s *sometimes) set(names ...string) { s.mu.Lock(); s.names = names; s.queued = -1; s.mu.Unlock() }
func (s *sometimes) queuedNow() (q int)  { s.mu.Lock(); q = s.queued; s.mu.Unlock(); return }
func (s *sometimes) String() string      { return strings.Join(s.CueNames(), ",") }
func (s *sometimes) has(name string) string {
	return `[...document.querySelectorAll(".cue")].some(t => t.title === "` + name + `")`
}

// the tiles follow cues that change, also when there are none for a while (between two songs: the
// page once crashed there)
func TestPageCueSlots(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	s := &sometimes{names: []string{"Intro", "Verse", "End"}, queued: -1}
	pn.Cues("Sections", s, CueSlots(8))
	tb := b.open(t, serve(t, pn), built)
	tb.waitFor(s.has("Verse"), "the first cues")

	s.set() // between songs
	tb.waitFor(`document.querySelectorAll(".cue").length === 0`, "no tiles without cues")
	s.set("A", "B", "Break", "Out")
	tb.waitFor(s.has("Break"), "the tiles of the next song")
	tb.tap(`document.querySelectorAll(".cue")[3]`)
	eventually(t, "the fourth tile to ask for the fourth cue", func() bool { return s.queuedNow() == 3 })
}

// a mount shows the panel mounted in a frame of its height, another one when it changes
func TestPageMount(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	m := &mounter{}
	pn.Section("Song")
	pn.Mount("Song", m)
	morning := New(psalm.New(nil), "Morning")
	morning.Fader("Piano", -40, 6, Linear, nil)
	m.p.Store(morning)
	var loads atomic.Int32 // the mounted panel's page loaded
	h := pn.Handler()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mount/Song/" {
			loads.Add(1)
		}
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	tb := b.open(t, srv.URL+"/", built)

	frame := `document.querySelector(".mountbox iframe")`
	tb.waitFor(frame+`.offsetHeight > 100`, "the frame to take the height of the mounted panel")
	tb.waitFor(`document.querySelector("section.mounthost h2").textContent === "Morning"`, "the heading to be the panel's title")
	tb.waitFor(frame+`.contentDocument && `+frame+`.contentDocument.querySelector(".fader")`, "the mounted panel's fader in the frame")
	time.Sleep(300 * time.Millisecond) // a few updates of the page
	if n := loads.Load(); n != 1 {
		t.Fatalf("the mounted panel should load once, not on every update: %d times", n)
	}

	groove := New(psalm.New(nil), "Groove")
	tempo := groove.Fader("Tempo", 90, 120, Linear, nil, Default(104))
	m.p.Store(groove)
	tb.waitFor(`document.querySelector("section.mounthost h2").textContent === "Groove"`, "the next panel's title")
	inFrame := frame + `.contentDocument`
	tb.waitFor(inFrame+` && `+inFrame+`.title === "Groove" && `+inFrame+`.querySelector(".fader") && `+frame+`.offsetHeight > 100`, "the next panel in the frame")
	// a control in the frame moves the mounted panel
	tb.eval(`(() => { const f = ` + inFrame + `.querySelector(".fader"); f.focus(); f.dispatchEvent(new KeyboardEvent("keydown", {key: "ArrowUp", bubbles: true})); })()`)
	eventually(t, "the mounted panel's fader to move", func() bool { return tempo.Value() > 104 })

	m.p.Store(nil)
	tb.waitFor(`!document.querySelector(".mountbox .none").hidden`, "nothing to show without a panel mounted")
}

// a tap on a step switches it on
func TestPageGrid(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	g := pn.Grid("Drums", []string{"kick", "snare"}, 8, 2, testFields...)
	tb := b.open(t, serve(t, pn), built)
	tb.tap(`document.querySelectorAll(".cell")[10]`) // the snare's third step
	eventually(t, "the step to switch on", func() bool { return g.Value(0, 1, 2, 0) > 0 })
}

// MIDI learn: the button arms the mode, a tap on a control arms it, Escape leaves the mode
func TestPageLearn(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	cutoff := pn.Fader("Cutoff", 0, 1, Linear, nil)
	pn.Cues("Songs", newCueList())
	tb := b.open(t, serve(t, pn), built)
	tb.tap(`document.getElementById("learn")`)
	tb.waitFor(`document.body.classList.contains("learn")`, "learn mode")
	tb.tap(`document.querySelector(".fader")`)
	eventually(t, "the fader to be armed", func() bool { return pn.Learning() == cutoff })
	tb.tap(`document.querySelectorAll(".cue")[1]`)
	eventually(t, "a tile to be armed (its button)", func() bool { return pn.Learning() == pn.Param("Songs/Verse") })
	tb.key("Escape")
	eventually(t, "Escape to stop learning", func() bool { return pn.Learning() == nil })
}

// Start/Stop and Restart in the header
func TestPagePlayer(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	pn.Fader("Level", 0, 1, Linear, nil)
	pl := &restartable{}
	pl.running.Store(true)
	pn.SetPlayer(pl)
	tb := b.open(t, serve(t, pn), built)
	tb.tap(`document.getElementById("run")`)
	eventually(t, "Stop to stop the sound", func() bool { return !pl.Running() })
	tb.waitFor(`document.getElementById("run").textContent.includes("Start")`, "the button to offer Start")
	tb.tap(`document.getElementById("restart")`)
	eventually(t, "Restart to go back to the start", func() bool { return pl.restarts.Load() == 1 })
}

// presets: save a setup by name, change it, load it again
func TestPagePresets(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	level := pn.Fader("Level", 0, 1, Linear, nil)
	dir := t.TempDir()
	_ = pn.Presets(dir)
	tb := b.open(t, serve(t, pn), built)
	level.Set(0.75)
	tb.eval(`document.querySelector("#presets input").value = "Loud"`)
	tb.tap(`[...document.querySelectorAll("#presets button")].find(b => b.textContent === "Save")`)
	eventually(t, "the preset to be saved", func() bool {
		_, err := os.Stat(filepath.Join(dir, "Loud.json"))
		return err == nil
	})
	level.Set(0.2)
	tb.waitFor(`[...document.querySelectorAll("#presets select option")].some(o => o.value === "Loud")`, "the preset in the list")
	tb.eval(`document.querySelector("#presets select").value = "Loud"`)
	tb.tap(`[...document.querySelectorAll("#presets button")].find(b => b.textContent === "Load")`)
	eventually(t, "the preset to be loaded", func() bool { return level.Value() == 0.75 })
}
