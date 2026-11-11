package catalog

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// testVST3 compiles the test plugins of modules/vst3 into a bundle in a directory
func testVST3(t *testing.T) string {
	t.Helper()
	cxx, err := exec.LookPath("c++")
	if err != nil {
		t.Skip("no C++ compiler")
	}
	dir := t.TempDir()
	root := "../../../../vst3"
	so := filepath.Join(dir, "test.vst3", "Contents", "x86_64-linux")
	if err := os.MkdirAll(so, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(cxx, "-std=c++17", "-shared", "-fPIC", "-I", filepath.Join(root, "include"), "-I", root, "-o", filepath.Join(so, "test.so"),
		filepath.Join(root, "testdata/testplugin.cpp"), filepath.Join(root, "funknown_cpp.cpp"), filepath.Join(root, "iids.cpp")).CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return dir
}

// the kinds of the VST3 plugins in a directory: an instrument with its three outputs, an effect with a side chain and a latency, their settings described
func TestVST3Kinds(t *testing.T) {
	dir := testVST3(t)
	byName := map[string]patcher.Kind{}
	for _, k := range vst3Kinds(dir) {
		byName[k.Name] = k
	}
	synth, fx := byName["vst3"], byName["vst3 fx"]
	if !slices.Equal(synth.Presets, []string{"Psalm VST3 Sine"}) || !slices.Equal(fx.Presets, []string{"Psalm VST3 Gain"}) || synth.Group != "Synths" || fx.Group != "Effects" {
		t.Fatalf("kinds %+v %+v", synth, fx)
	}
	made, err := synth.Make("Psalm VST3 Sine")
	if err != nil {
		t.Fatal(err)
	}
	defer made.Close()
	if set := made.Describe(""); len(set.Params) != 2 || set.Params[0].Key != "preset" || set.Params[1].Key != "gain" {
		t.Errorf("settings %+v", set)
	}
	if len(made.Outs) != 3 || made.Outs[0].Name != "out" || made.Open != nil || made.Status() != "" {
		t.Errorf("an instrument with three outputs: %v, a window %v, says %q", made.Outs, made.Open != nil, made.Status())
	}
	madeFx, err := fx.Make("Psalm VST3 Gain")
	if err != nil {
		t.Fatal(err)
	}
	defer madeFx.Close()
	if len(madeFx.Ins) != 2 || !strings.Contains(madeFx.Status(), "latency 64") {
		t.Errorf("the effect has the ports %v and says %q", madeFx.Ins, madeFx.Status())
	}
	if _, err := synth.Make("nothing"); err == nil {
		t.Error("a plugin that is not there")
	}
}

// a VST3 plugin in a patch that is kept: its state comes back, and after a program is chosen the editor's knob follows the plugin
func TestVST3StateInPatch(t *testing.T) {
	dir := testVST3(t)
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	newPatcher := func(keep string) (*patcher.Patcher, *psalm.Psalm) {
		p := psalm.New(nil)
		pt := patcher.New(p, panel.New(p, "Test"), "Patch", vst3Kinds(dir), patcher.Keep(keep))
		p.Out(pt.Out())
		return pt, p
	}
	var pt2dbg func() patcher.Settings
	until := func(p *psalm.Psalm, what string, done func() bool) {
		t.Helper()
		for deadline := time.Now().Add(10 * time.Second); !done() && time.Now().Before(deadline); time.Sleep(3 * time.Millisecond) {
			p.ResetRun()
			p.Run()
		}
		if !done() {
			t.Fatalf("%s (the knob is at %v, the preset at %v)", what, pt2dbg().Values()["gain"], pt2dbg().Values()["preset"])
		}
	}
	keep := t.TempDir()
	pt, p1 := newPatcher(keep)
	id, err := pt.Add("vst3", "Psalm VST3 Sine", 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	pt2dbg = func() patcher.Settings { return pt.SettingsOf(id) }
	if _, err := pt.Connect(patcher.Cable{Type: patcher.Audio, From: id, To: "out"}); err != nil {
		t.Fatal(err)
	}
	gain := func(pt *patcher.Patcher, id string) float64 { return pt.SettingsOf(id).Values()["gain"] }
	pt.SettingsOf(id).Apply(map[string]float64{"preset": 3}) // the program "Loud" (none, Quiet, Middle, Loud): a gain of 0.9
	until(p1, "the knob does not follow the program", func() bool { return math.Abs(gain(pt, id)-0.9) < 1e-6 })
	if err := pt.Save("with state"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(keep, "with state.json"))
	if !strings.Contains(string(data), `"state"`) {
		t.Errorf("no state in the patch kept: %s", data)
	}
	pt2, p2 := newPatcher(keep)
	if err := pt2.Load("with state"); err != nil {
		t.Fatal(err)
	}
	var id2 string
	for _, n := range pt2.Nodes() {
		if n.Kind == "vst3" {
			id2 = n.ID
		}
	}
	pt2dbg = func() patcher.Settings { return pt2.SettingsOf(id2) }
	until(p2, "after loading the patch the knob is not back", func() bool { return math.Abs(gain(pt2, id2)-0.9) < 1e-6 })
	// and it stays: what the editor is told (the preset: none) is not a preset to load over the state
	for deadline := time.Now().Add(1500 * time.Millisecond); time.Now().Before(deadline); time.Sleep(3 * time.Millisecond) {
		p2.ResetRun()
		p2.Run()
	}
	if g := gain(pt2, id2); math.Abs(g-0.9) > 1e-6 {
		t.Errorf("the knob went to %v after the state was given back, want it to stay at 0.9", g)
	}
}

// real plugins, when VST3_TEST_DIR names a directory with the VST3 builds of Surge XT and Surge XT Effects: the synth into the effect, a chord through both
func TestVST3RealPlugins(t *testing.T) {
	dir := os.Getenv("VST3_TEST_DIR")
	if dir == "" {
		t.Skip("VST3_TEST_DIR is not set")
	}
	byName := map[string]patcher.Kind{}
	for _, k := range vst3Kinds(dir) {
		byName[k.Name] = k
	}
	synth, err := byName["vst3"].Make("Surge XT")
	if err != nil {
		t.Fatal(err)
	}
	fx, err := byName["vst3 fx"].Make("Surge XT Effects")
	if err != nil {
		t.Fatal(err)
	}
	synth.Module.Connect(0, fx.Module, 0)
	for i, pitch := range []float64{60, 64, 67} {
		synth.Module.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": string(rune('a' + i)), "amplitude": 0.7, "message": psalm.Message{"pitch": pitch}, "tick": psalm.CtrlTick{}})
	}
	var dry, wet float64
	for range 400 {
		fx.Module.ResetRun()
		synth.Module.ResetRun()
		fx.Module.Run()
		for _, s := range fx.Module.Output(0).Channels[0] {
			wet = max(wet, math.Abs(float64(s)))
		}
		for _, s := range synth.Module.Output(0).Channels[0] {
			dry = max(dry, math.Abs(float64(s)))
		}
	}
	t.Logf("Surge XT %d settings, Surge XT Effects %d; the synth peaks at %.3f, through the effect %.3f", len(synth.Describe("").Params), len(fx.Describe("").Params), dry, wet)
	if dry < 0.05 || wet == 0 || math.IsNaN(wet) {
		t.Errorf("silent: %v %v", dry, wet)
	}
	_ = psalm.CurrentEnv
}
