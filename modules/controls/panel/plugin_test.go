package panel

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

// a plugin drawing a "dial": it shows its value and state, sends a value and a message
const dialJS = `psalm.register("dial", (p, api) => {
	const box = document.createElement("div");
	box.className = "dial";
	box.dataset.data = JSON.stringify(p.data);
	return {
		el: api.labelled(p, box),
		show(v) { box.dataset.value = v; },
		update(st) { box.dataset.state = JSON.stringify(st); },
		bindable: true,
	};
});`

func TestCustomControl(t *testing.T) {
	pn, tk := newPanel()
	rec := newRecorder()
	var got []string
	pn.Use(Plugin{Name: "dials", JS: dialJS})
	dial := pn.Custom("dial", "Dial", Message(rec, "cutoff"), Range(0, 10, Linear), Default(3), Data(map[string]int{"rings": 4}),
		OnMessage(func(data json.RawMessage) { got = append(got, string(data)) }))
	if err := dial.Push(map[string]string{"mood": "calm"}); err != nil {
		t.Fatal(err)
	}
	tk.tick(1)
	if v := rec.last(); v != 3 {
		t.Errorf("the default should reach the target, got %v", v)
	}
	// the description carries the data and the state
	d := pn.describe()
	c := d.Sections[0].Controls[0]
	if c.Kind != "dial" || string(c.Data) != `{"rings":4}` || string(c.State) != `{"mood":"calm"}` || c.Max != 10 {
		t.Errorf("described as %+v", c)
	}
	if len(d.Plugins) != 1 || d.Plugins[0].Name != "dials" || !d.Plugins[0].Script || d.Plugins[0].Style {
		t.Errorf("plugins %+v", d.Plugins)
	}
	// a message from the page reaches OnMessage on the next tick
	srv := httptest.NewServer(pn.Handler())
	defer srv.Close()
	res, err := http.Post(srv.URL+"/api/message", "application/json", strings.NewReader(`{"name": "Dial", "data": {"turn": 2}}`))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("message: %v %v", err, res.Status)
	}
	if len(got) != 0 {
		t.Errorf("a message should wait for the audio thread")
	}
	tk.tick(1)
	if len(got) != 1 || got[0] != `{"turn": 2}` {
		t.Errorf("messages %v", got)
	}
	// the script is served
	res, _ = http.Get(srv.URL + "/plugins/dials/plugin.js")
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), `psalm.register("dial"`) || !strings.Contains(res.Header.Get("Content-Type"), "javascript") {
		t.Errorf("plugin.js: %s", res.Header.Get("Content-Type"))
	}
	if res, _ := http.Get(srv.URL + "/plugins/nothing/plugin.js"); res.StatusCode != 404 {
		t.Errorf("an unknown plugin: %v", res.Status)
	}
	// a kind of the page's own can't be custom
	defer func() {
		if recover() == nil {
			t.Errorf("a custom fader should panic")
		}
	}()
	pn.Custom(FaderKind, "Fake", nil)
}

func TestThemeStylesheet(t *testing.T) {
	css := Theme{Name: "x", Light: map[string]string{"accent": "red", "--bg": "white"}, CSS: ".fader{}"}.stylesheet()
	for _, want := range []string{`:root:not([data-mode="light"])`, "--accent: red;", "--bg: white;", ".fader{}"} {
		if !strings.Contains(css, want) {
			t.Errorf("the stylesheet lacks %q:\n%s", want, css)
		}
	}
	if strings.Contains(css, "@media") {
		t.Errorf("a theme without dark has no dark rules")
	}
	css = Themes["contrast"].stylesheet()
	if !strings.Contains(css, "@media (prefers-color-scheme: dark)") || !strings.Contains(css, `:root[data-mode="dark"]`) {
		t.Errorf("a theme with dark: %s", css)
	}
	pn, _ := newPanel()
	pn.SetTheme(Theme{Name: "mine", Light: map[string]string{"accent": "#123456"}})
	d := pn.describe()
	if d.Theme != "mine" || d.Themes[0] != "default" || !strings.Contains(strings.Join(d.Themes, ","), "mine") {
		t.Errorf("themes %v, theme %q", d.Themes, d.Theme)
	}
	srv := httptest.NewServer(pn.Handler())
	defer srv.Close()
	res, _ := http.Get(srv.URL + "/theme.css")
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "#123456") {
		t.Errorf("the panel's own theme: %s", body)
	}
	res, _ = http.Get(srv.URL + "/theme.css?name=amber")
	body, _ = io.ReadAll(res.Body)
	if !strings.Contains(string(body), "#ffb000") {
		t.Errorf("a theme by name: %s", body)
	}
}

// the page loads the plugin, draws its control, follows its value and state, and its messages and
// values come back
func TestPagePlugin(t *testing.T) {
	b := startBrowser(t)
	pn, tk := newPanel()
	var messages atomic.Int32
	pn.Use(Plugin{Name: "dials", JS: dialJS, CSS: ".dial { width: 33px; height: 20px; }",
		Files: fstest.MapFS{"ring.svg": {Data: []byte("<svg/>")}}})
	dial := pn.Custom("dial", "Dial", nil, Data([]int{1, 2}), OnMessage(func(json.RawMessage) { messages.Add(1) }))
	pn.Custom("nobody", "Orphan", nil) // no plugin draws it
	tb := b.Open(t, serve(t, pn), built)
	tb.WaitFor(`document.querySelector(".dial")`, "the plugin's control")
	if w := tb.Eval(`document.querySelector(".dial").getBoundingClientRect().width`); w != 33.0 {
		t.Errorf("the plugin's styles: width %v", w)
	}
	if d := tb.Eval(`document.querySelector(".dial").dataset.data`); d != "[1,2]" {
		t.Errorf("the data: %v", d)
	}
	if k := tb.Eval(`document.querySelector(".dial").closest("[data-kind]").dataset.kind`); k != "dial" {
		t.Errorf("data-kind %v", k)
	}
	tb.WaitFor(`document.querySelector(".missing")`, "a control no plugin draws")
	_ = dial.Push(map[string]int{"n": 7})
	tb.WaitFor(`document.querySelector(".dial").dataset.state === '{"n":7}'`, "the state pushed")
	dial.Set(0.25)
	tb.WaitFor(`document.querySelector(".dial").dataset.value === "0.25"`, "the value set from code")
	tb.Eval(`psalm.send("Dial", 0.75)`)
	eventually(t, "the value sent from the page", func() bool { return dial.Value() == 0.75 })
	tb.Eval(`psalm.message("Dial", {hi: 1})`)
	eventually(t, "the message", func() bool { tk.tick(1); return messages.Load() == 1 })
	if s := tb.Eval(`fetch("plugins/dials/ring.svg").then(r => r.text())`); s != "<svg/>" {
		t.Errorf("a file of the plugin: %v", s)
	}
}

// the theme chosen on the page sets its look, and is remembered
func TestPageTheme(t *testing.T) {
	b := startBrowser(t)
	pn, _ := newPanel()
	pn.SetTheme(Theme{Name: "mine", Light: map[string]string{"accent": "rgb(1, 2, 3)"}})
	pn.Fader("Level", 0, 1, Linear, nil)
	tb := b.Open(t, serve(t, pn), built)
	tb.Eval(`localStorage.clear()`)
	tb.WaitFor(`psalm.css("accent") === "rgb(1, 2, 3)"`, "the panel's own theme")
	tb.Eval(`(() => { const s = document.getElementById("themes"); s.value = "amber"; s.onchange(); })()`)
	tb.WaitFor(`psalm.css("accent") === "#ffb000"`, "the theme chosen")
	if v := tb.Eval(`localStorage.getItem("psalm.theme")`); v != "amber" {
		t.Errorf("the theme isn't remembered: %v", v)
	}
	tb.Eval(`(() => { const s = document.getElementById("mode"); s.value = "dark"; s.onchange(); })()`)
	if m := tb.Eval(`document.documentElement.dataset.mode`); m != "dark" {
		t.Errorf("mode %v", m)
	}
	tb.Eval(`localStorage.clear()`)
}
