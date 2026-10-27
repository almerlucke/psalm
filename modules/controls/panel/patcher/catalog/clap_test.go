package catalog

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/clap"
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
	if _, err := fx.Make("Psalm Test Gain"); err != nil {
		t.Error(err)
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
	if set.Params[0].Key != "preset" || len(set.Params[0].Options) < 1000 {
		t.Errorf("Surge XT lists %d presets (its data is needed: see docs/clap.md)", len(set.Params[0].Options))
	} else {
		want := slices.Index(set.Params[0].Options, "Plucks / Taped Plucks")
		synth.Module.ReceiveMessage(psalm.Message{"preset": float64(want)})
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
