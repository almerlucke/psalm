package patcher_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

type helpExample struct {
	title    string
	texts    map[string]string
	settings map[string]float64
}

// helpExamples reads the examples of a help page the way the page does: a block ```example, [name] lines for the texts, [set] for the settings (`name = number`)
func helpExamples(t *testing.T, text string) []helpExample {
	t.Helper()
	var out []helpExample
	set := regexp.MustCompile(`^\s*([A-Za-z0-9_ -]+?)\s*=\s*(-?[0-9.]+)`)
	head := regexp.MustCompile(`^\[([a-z]+)\]\s*$`)
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "```example") {
			continue
		}
		ex := helpExample{title: strings.TrimSpace(strings.TrimPrefix(lines[i], "```example")), texts: map[string]string{}, settings: map[string]float64{}}
		key := ""
		var body []string
		flush := func() {
			if key != "" && key != "set" {
				ex.texts[key] = strings.TrimRight(strings.Join(body, "\n"), "\n")
			}
			body = nil
		}
		for i++; i < len(lines) && !strings.HasPrefix(lines[i], "```"); i++ {
			if m := head.FindStringSubmatch(lines[i]); m != nil {
				flush()
				key = m[1]
				continue
			}
			if key == "set" {
				if m := set.FindStringSubmatch(lines[i]); m != nil {
					v, err := strconv.ParseFloat(m[2], 64)
					if err != nil {
						t.Fatalf("%s: %v", ex.title, err)
					}
					ex.settings[m[1]] = v
				}
			} else if key != "" {
				body = append(body, lines[i])
			}
		}
		flush()
		out = append(out, ex)
	}
	return out
}

// every module of the Song, Generators, Synths, Effects, Modulation, Mix and Sources groups has a help page, and every example of a page loads into a module of its kind: its texts are read and its
// settings are the module's
func TestHelpPages(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Help"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	need := map[string]bool{"clock": true, "sequence": true, "notes": true, "melodist": true}
	for _, k := range catalog.Kinds(catalog.Config{Input: midi.NewQueue()}) {
		if k.Group == "Song" || k.Group == "Generators" || k.Group == "Synths" || k.Group == "Effects" ||
			k.Group == "Modulation" || k.Group == "Mix" || k.Group == "Sources" {
			need[k.Name] = true
		}
	}
	files, _ := filepath.Glob("help/*.md")
	have := map[string]bool{}
	for _, f := range files {
		kind := strings.TrimSuffix(filepath.Base(f), ".md")
		have[kind] = true
		text, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(text), "# "+kind+"\n") {
			t.Errorf("%s: should start with its name as the heading", f)
		}
		for _, ex := range helpExamples(t, string(text)) {
			pt.Clear()
			id, err := pt.Add(kind, "", 0, 0)
			if err != nil {
				t.Errorf("%s: %v", f, err)
				continue
			}
			if err := pt.LoadExample(id, ex.texts, ex.settings); err != nil {
				t.Errorf("%s: %q: %v", f, ex.title, err)
				continue
			}
			for key := range ex.texts {
				if e := pt.TextError(id, key); e != "" {
					t.Errorf("%s: %q: %s: %s", f, ex.title, key, e)
				}
			}
			if len(ex.settings) == 0 {
				continue
			}
			values := pt.SettingsOf(id).Values()
			for k, v := range ex.settings {
				if values[k] != v {
					t.Errorf("%s: %q: %s is %v, the example says %v", f, ex.title, k, values[k], v)
				}
			}
		}
	}
	for kind := range need {
		if !have[kind] {
			t.Errorf("no help page for %s (help/%s.md)", kind, kind)
		}
	}
}

// an example the module cannot take is an error and loads nothing
func TestLoadExampleErrors(t *testing.T) {
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Help"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	id, _ := pt.Add("voice", "", 0, 0)
	if err := pt.LoadExample(id, map[string]string{"pattern": "x . x ."}, map[string]float64{"nonsense": 1}); err == nil {
		t.Error("a setting the module does not have should be an error")
	}
	if err := pt.LoadExample(id, map[string]string{"nonsense": "x"}, nil); err == nil {
		t.Error("a text the module does not have should be an error")
	}
	if err := pt.LoadExample("nope", nil, nil); err == nil {
		t.Error("no such module")
	}
	if got := pt.SettingsOf(id).Values()["division"]; got != 3 {
		t.Errorf("nothing should have been loaded, division is %v", got)
	}
}

// the ? of a module opens its help in a window of its own (markdown: headings, tables, code, examples); the Load button of an example puts its texts and settings in the module the
// help was opened from; a link goes to the help of another module; the help closes by its x and when it is opened again it is one window
func TestPageHelp(t *testing.T) {
	b, pt, base := newPage(t)
	v, err := pt.Add("voice", "", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, base, `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1300, 900)
	tb.Tap(fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-help")`, v))
	tb.WaitFor(`document.querySelectorAll(".pt-help-window").length === 1 && document.querySelector(".pt-help-window h2")`, "the help of the voice")
	if !tb.Eval(`document.querySelector(".pt-help-window .pt-win-title").textContent === "help · voice" && document.querySelectorAll(".pt-help-window .pt-md-example").length >= 3 && !!document.querySelector(".pt-help-window table")`).(bool) {
		t.Error("the help is not drawn: title, examples and tables")
	}
	// the Load button of the first example
	tb.Tap(`document.querySelector(".pt-help-window .pt-md-example button")`)
	paneltest.Eventually(t, "the example loaded into the voice", func() bool {
		tx := pt.Texts(v)
		return len(tx) >= 1 && strings.HasPrefix(tx[0].Value, "x . . . x . . . x") && pt.SettingsOf(v).Values()["pitch"] == 36
	})
	// a link goes to the help of another module, a second ? to the same one is not a second window
	tb.Tap(fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-help")`, v))
	tb.Eval(`[...document.querySelectorAll(".pt-help-window a")].find(a => a.title === "the help of clock").scrollIntoView()`)
	tb.Tap(`[...document.querySelectorAll(".pt-help-window a")].find(a => a.title === "the help of clock")`)
	tb.WaitFor(`document.querySelectorAll(".pt-help-window").length === 2`, "the help of the clock from a link")
	tb.Tap(`document.querySelector(".pt-help-window .pt-x")`)
	tb.Tap(`document.querySelector(".pt-help-window .pt-x")`)
	tb.WaitFor(`document.querySelectorAll(".pt-help-window").length === 0`, "the helps closed")
	// the Load button is off when the module is gone
	tb.Tap(fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-help")`, v))
	tb.WaitFor(`!!document.querySelector(".pt-help-window .pt-md-example button")`, "the help again")
	if err := pt.Remove(v); err != nil {
		t.Fatal(err)
	}
	tb.WaitFor(`document.querySelector(".pt-help-window .pt-md-example button").disabled`, "the Load button off when the module is gone")
}
