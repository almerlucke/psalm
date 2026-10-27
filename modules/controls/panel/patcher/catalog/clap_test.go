package catalog

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/clap"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// the kinds of the plugins in a directory: an instrument that plays the notes it is sent, an effect, their settings described, and the plugin let go again
func TestClapKinds(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("no C compiler")
	}
	dir := t.TempDir()
	root := "../../../../clap"
	if out, err := exec.Command(cc, "-shared", "-fPIC", "-I", filepath.Join(root, "include"), "-o", filepath.Join(dir, "test.clap"), filepath.Join(root, "testdata/testplugin.c"), "-lm").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	byName := map[string]patcher.Kind{}
	for _, k := range clapKinds(dir) {
		byName[k.Name] = k
	}
	synth, fx := byName["clap"], byName["clap fx"]
	if !slices.Equal(synth.Presets, []string{"Psalm Test Sine"}) || !slices.Equal(fx.Presets, []string{"Psalm Test Gain"}) || synth.Group != "Synths" || fx.Group != "Effects" {
		t.Fatalf("kinds %+v %+v", synth, fx)
	}
	made, err := synth.Make("Psalm Test Sine")
	if err != nil {
		t.Fatal(err)
	}
	if set := made.Describe(""); len(set.Params) != 2 || set.Params[0].Key != "preset" || set.Params[1].Key != "output gain" { // the plugin's own presets first
		t.Errorf("settings %+v", set)
	}
	m := made.Module
	m.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "a", "amplitude": 1.0, "message": psalm.Message{"pitch": 69.0}, "tick": psalm.CtrlTick{}})
	m.ResetRun()
	m.Synthesize()
	peak := 0.0
	for _, s := range m.Output(0).Channels[0] {
		peak = math.Max(peak, math.Abs(float64(s)))
	}
	if peak < 0.4 {
		t.Errorf("the note does not sound: %v", peak)
	}
	made.Close()
	m.ResetRun()
	m.Synthesize() // lets go of it
	if m.Output(0).Channels[0][10] != 0 {
		t.Error("sounds after it is let go")
	}
	if len(made.Outs) != 3 || made.Outs[0].Name != "out" {
		t.Errorf("an instrument with three audio outputs has the ports %v", made.Outs)
	}
	if made.Status() != "" {
		t.Errorf("status %q of a plugin with no latency", made.Status())
	}
	madeFx, err := fx.Make("Psalm Test Gain")
	if err != nil {
		t.Fatal(err)
	}
	if len(madeFx.Ins) != 2 || madeFx.Ins[1].Name != "sidechain" && madeFx.Ins[1].Name != "main" || !strings.Contains(madeFx.Status(), "latency 64") {
		t.Errorf("the effect has the ports %v and says %q", madeFx.Ins, madeFx.Status())
	}
	if _, err := synth.Make("nothing"); err == nil {
		t.Error("a plugin that is not there")
	}
}

// real plugins, when CLAP_TEST_DIR names a directory with Surge XT and Surge XT Effects: the synth into the effect, a chord through both
func TestClapRealPlugins(t *testing.T) {
	dir := os.Getenv("CLAP_TEST_DIR")
	if dir == "" {
		t.Skip("CLAP_TEST_DIR is not set")
	}
	byName := map[string]patcher.Kind{}
	for _, k := range clapKinds(dir) {
		byName[k.Name] = k
	}
	synth, err := byName["clap"].Make("Surge XT")
	if err != nil {
		t.Fatal(err)
	}
	fx, err := byName["clap fx"].Make("Surge XT Effects")
	if err != nil {
		t.Fatal(err)
	}
	synth.Module.Connect(0, fx.Module, 0)
	for i, pitch := range []float64{60, 64, 67.02} {
		synth.Module.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": fmt.Sprint(i), "amplitude": 0.7, "message": psalm.Message{"pitch": pitch}, "tick": psalm.CtrlTick{}})
	}
	var peak, dry float64
	for range 400 {
		fx.Module.ResetRun()
		synth.Module.ResetRun()
		fx.Module.Run()
		for _, s := range fx.Module.Output(0).Channels[0] {
			peak = math.Max(peak, math.Abs(float64(s)))
		}
		for _, s := range synth.Module.Output(0).Channels[0] {
			dry = math.Max(dry, math.Abs(float64(s)))
		}
	}
	// its own presets, chosen by number as the editor does
	set := synth.Describe("")
	var presets []string
	for _, p := range set.Params {
		if p.Key == "preset" {
			presets = p.Options
		}
	}
	if len(presets) < 1000 {
		t.Errorf("Surge XT lists %d presets (its data is needed: see docs/clap.md)", len(presets))
	} else {
		want := slices.Index(presets, "Plucks / Taped Plucks") - 1 // the list has "none" first
		synth.Module.ReceiveMessage(psalm.Message{"preset": float64(want + 1)})
		pl := synth.Module.(*clap.Plugin)
		deadline := time.Now().Add(3 * time.Second)
		for pl.Loaded() != want && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if pl.Loaded() != want || want < 0 {
			t.Errorf("preset %d (Plucks / Taped Plucks) not loaded: %d", want, pl.Loaded())
		}
	}
	t.Logf("Surge XT %d settings, Surge XT Effects %d; the synth peaks at %.3f, through the effect %.3f", len(synth.Describe("").Params), len(fx.Describe("").Params), dry, peak)
	if dry < 0.05 || peak == 0 || math.IsNaN(peak) {
		t.Errorf("silent: %v %v", dry, peak)
	}
}

// a plugin in a patch that is kept: its state comes back (the sound is not the settings of the knobs of the time), and after a preset the editor's knobs follow the plugin
func TestClapStateInPatch(t *testing.T) {
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("no C compiler")
	}
	dir := t.TempDir()
	root := "../../../../clap"
	if out, err := exec.Command(cc, "-shared", "-fPIC", "-I", filepath.Join(root, "include"), "-o", filepath.Join(dir, "test.clap"), filepath.Join(root, "testdata/testplugin.c"), "-lm").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	newPatcher := func(keep string) (*patcher.Patcher, *psalm.Psalm) {
		p := psalm.New(nil)
		pt := patcher.New(p, panel.New(p, "Test"), "Patch", clapKinds(dir), patcher.Keep(keep))
		p.Out(pt.Out())
		return pt, p
	}
	keep := t.TempDir()
	var dbg func() any
	// the patch runs while we wait: the controls of the editor tick and send what they were set to
	until := func(p *psalm.Psalm, what string, done func() bool) {
		t.Helper()
		for deadline := time.Now().Add(10 * time.Second); !done() && time.Now().Before(deadline); time.Sleep(3 * time.Millisecond) {
			p.ResetRun()
			p.Run()
		}
		if !done() {
			t.Fatalf("%s (%v)", what, dbg())
		}
	}
	pt, p1 := newPatcher(keep)
	id, err := pt.Add("clap", "Psalm Test Sine", 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pt.Connect(patcher.Cable{Type: patcher.Audio, From: id, To: "out"}); err != nil { // so that the plugin is run
		t.Fatal(err)
	}
	node := func(pt *patcher.Patcher, id string) patcher.Settings {
		s := pt.SettingsOf(id)
		if s == nil {
			t.Fatal("no settings")
		}
		return s
	}
	// the preset "Loud" (gain 1): the knob of the editor follows
	pt.SettingsOf(id).Apply(map[string]float64{"preset": 2}) // "none", "Quiet", "Loud"
	until(p1, "the knob does not follow the preset", func() bool { return node(pt, id).Values()["output gain"] == 1 })
	// the plugin renames the knob when it is turned up (it tells the host): the editor shows the new label
	label := func() string {
		return node(pt, id).(interface{ Setting(string) *panel.Param }).Setting("output gain").Shown()
	}
	if l := label(); l != "Loudness" { // at 1 after the preset
		until(p1, "the knob is not renamed after the preset: "+label(), func() bool { return label() == "Loudness" })
	}
	// a knob turned in the plugin's own window (here: a note of the key 127): the knob of the editor follows
	pl := pt.Module(id).(*clap.Plugin)
	pl.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "gui", "amplitude": 1.0, "message": psalm.Message{"pitch": 127.0}, "tick": psalm.CtrlTick{}})
	until(p1, "the knob does not follow what the plugin did itself", func() bool { return math.Abs(node(pt, id).Values()["output gain"]-0.123) < 1e-6 })
	pl.ReceiveMessage(psalm.Message{"command": "trigger", "noteOff": "gui", "tick": psalm.CtrlTick{}})
	pt.SettingsOf(id).Apply(map[string]float64{"output gain": 1})
	until(p1, "the knob does not go back to 1", func() bool { return node(pt, id).Values()["output gain"] == 1 })
	if err := pt.Save("with state"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(keep, "with state.json"))
	if !strings.Contains(string(data), `"state"`) {
		t.Errorf("no state in the patch kept: %s", data)
	}
	// another patcher loads it: the plugin has gain 1 again (a plugin made new has 0.5)
	pt2, p2 := newPatcher(keep)
	if err := pt2.Load("with state"); err != nil {
		t.Fatal(err)
	}
	var id2 string
	for _, n := range pt2.Nodes() {
		if n.Kind == "clap" {
			id2 = n.ID
		}
	}
	dbg = func() any { return node(pt2, id2).Values() }
	until(p2, "after loading the patch the knob is not at 1", func() bool { return node(pt2, id2).Values()["output gain"] == 1 })
	// and it stays: what the editor is told (the preset: none) is not a preset to load over the state
	for deadline := time.Now().Add(1500 * time.Millisecond); time.Now().Before(deadline); time.Sleep(3 * time.Millisecond) {
		p2.ResetRun()
		p2.Run()
	}
	if g := node(pt2, id2).Values()["output gain"]; g != 1 {
		t.Errorf("the knob went to %v after the state was given back, want it to stay at 1", g)
	}
}

// the plugin's own window opens (what a double click on its module does) (opt in: it puts a window on the screen; CLAP_TEST_DIR names the directory with OB-Xf)
func TestClapWindow(t *testing.T) {
	dir := os.Getenv("CLAP_TEST_DIR")
	if dir == "" || os.Getenv("CLAP_TEST_WINDOW") == "" {
		t.Skip("CLAP_TEST_DIR and CLAP_TEST_WINDOW are not set")
	}
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	byName := map[string]patcher.Kind{}
	for _, k := range clapKinds(dir) {
		byName[k.Name] = k
	}
	made, err := byName["clap"].Make("OB-Xf")
	if err != nil {
		t.Fatal(err)
	}
	pl := made.Module.(*clap.Plugin)
	defer pl.Free()
	if made.Open == nil {
		t.Fatal("a plugin with a window has it to open")
	}
	if err := made.Open(); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); !pl.WindowOpen() && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
	}
	if !pl.WindowOpen() {
		t.Fatal("the window did not open")
	}
	made.Module.ReceiveMessage(psalm.Message{"window": 0.0})
	for deadline := time.Now().Add(5 * time.Second); pl.WindowOpen() && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
	}
	if pl.WindowOpen() {
		t.Error("the window did not close")
	}
}
