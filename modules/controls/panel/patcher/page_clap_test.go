package patcher_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/clap"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/vst3"
)

// the demo patch with Surge XT on the page: choosing the plugin shows its 775 settings in the side panel (CLAP_TEST_DIR names the directory with Surge XT and its data: see docs/clap.md)
func TestPageClap(t *testing.T) {
	dir := os.Getenv("CLAP_TEST_DIR")
	if dir == "" {
		t.Skip("CLAP_TEST_DIR is not set")
	}
	t.Setenv("CLAP_PATH", dir)
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep("../../../../patches"))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	if err := pt.Load("clap demo"); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 8`)
	t.Logf("the patch on the page after %v", time.Since(start))
	tb.Size(1300, 900)
	var surge string
	for _, n := range pt.Nodes() {
		if n.Kind == "clap" {
			surge = n.ID
		}
	}
	start = time.Now()
	head := fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, surge)
	// Alt held: the settings of ours, in a window inside the page (a plain double click opens the window of the plugin)
	tb.Eval(windowHelpers)
	tb.Eval(head + `.dispatchEvent(new MouseEvent("dblclick", {altKey: true, bubbles: true}))`)
	tb.WaitFor(`!!document.querySelector(".pt-win") && S()?.querySelector(".control")`, "the window of the settings of Surge XT")
	t.Logf("the window opened after %v", time.Since(start))
	tb.WaitFor(`S()?.querySelectorAll(".control").length > 500`, "the settings of Surge XT in its window")
	frame := `S()`
	if shut := tb.Eval(`[0, 1].some(i => ` + frame + `.querySelectorAll(".ed-group")[i].classList.contains("closed"))`).(bool); shut {
		t.Error("the preset and the main knobs are not both open")
	}
	if first := tb.Eval(frame + `.querySelector(".ed-title").textContent`).(string); first != "Preset (1)" {
		t.Errorf("the first group is %q", first)
	}
	if options := tb.Eval(frame + `.querySelector(".ed-group select").options.length`).(float64); options < 1000 {
		t.Errorf("%v presets to choose from, want the thousands Surge XT has", options)
	}
	t.Logf("its settings after %v", time.Since(start))
}

// a double click on a plugin opens the plugin's own window (opt in: it puts a window on the screen)
func TestPageClapNativeWindow(t *testing.T) {
	dir := os.Getenv("CLAP_TEST_DIR")
	if dir == "" || os.Getenv("CLAP_TEST_WINDOW") == "" {
		t.Skip("CLAP_TEST_DIR and CLAP_TEST_WINDOW are not set")
	}
	t.Setenv("CLAP_PATH", dir)
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("clap", "OB-Xf", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	pl := pt.Module(id).(*clap.Plugin)
	t.Cleanup(pl.HideWindow)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 2`)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	paneltest.Eventually(t, "the window of the plugin opened by a double click", func() bool { return pl.WindowOpen() })
	if tb.Eval(`document.querySelectorAll(".pt-win").length`).(float64) != 0 {
		t.Error("a window of settings opens besides the window of the plugin")
	}
}

// a double click on a VST3 plugin opens the plugin's own window (opt in: it puts a window on the screen; VST3_TEST_DIR names the directory with the VST3 build of OB-Xf)
func TestPageVST3NativeWindow(t *testing.T) {
	dir := os.Getenv("VST3_TEST_DIR")
	if dir == "" || os.Getenv("CLAP_TEST_WINDOW") == "" {
		t.Skip("VST3_TEST_DIR and CLAP_TEST_WINDOW are not set")
	}
	t.Setenv("VST3_PATH", dir)
	b := paneltest.Start(t)
	p := psalm.New(paneltest.NewEngine(t))
	ctl := panel.New(p, "Patcher")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	id, err := pt.Add("vst3", "OB-Xf", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	pl := pt.Module(id).(*vst3.Plugin)
	t.Cleanup(pl.HideWindow)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelectorAll(".pt-node").length === 2`)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	paneltest.Eventually(t, "the window of the plugin opened by a double click", func() bool { return pl.WindowOpen() })
	if tb.Eval(`document.querySelectorAll(".pt-win").length`).(float64) != 0 {
		t.Error("a window of settings opens besides the window of the plugin")
	}
}
