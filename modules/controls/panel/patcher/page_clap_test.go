package patcher_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// the demo patch with Surge XT on the page: choosing the plugin shows its 775 settings in the side panel (CLAP_TEST_DIR names the directory with Surge XT)
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
	doubleTap(tb, head)
	tb.WaitFor(`!document.querySelector(".pt-drawer").hidden && document.querySelector(".pt-drawer-frame").contentDocument?.querySelector(".control")`, "the side panel of Surge XT")
	t.Logf("the side panel opened after %v", time.Since(start))
	tb.WaitFor(`document.querySelector(".pt-drawer-frame").contentDocument?.querySelectorAll(".control").length > 500`, "the settings of Surge XT in the side panel")
	t.Logf("its settings after %v", time.Since(start))
}
