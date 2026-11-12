package patcher_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// the shaper's window has the editor of its chains: a rack of cards for the stages, plots that are drawn, the envelopes; a number on a card writes the
// chain, a stage added or taken away changes it, and the text typed makes the cards
func TestPageShapeView(t *testing.T) {
	b, pt, base := newPage(t)
	id, err := pt.Add("shaper", "foldLead", 60, 40)
	if err != nil {
		t.Fatal(err)
	}
	tb := b.Open(t, base, `document.querySelectorAll(".pt-node").length === 2`)
	tb.Size(1400, 1700)
	doubleTap(tb, fmt.Sprintf(`document.querySelector(".pt-node[data-id='%s'] .pt-head")`, id))
	tb.WaitFor(`S()?.querySelector(".shapeview .sv-card")`, "the rack of the chain")
	// the cards are the stages of the chain, each with a plot; the plots are drawn (some pixel is not empty)
	tb.WaitFor(`S().querySelectorAll(".shapeview .sv-card").length === 2`, "a card for sin and one for fold")
	drawn := func(sel string) bool {
		return tb.Eval(fmt.Sprintf(`(() => { const c = S().querySelector(%q); if (!c || !c.width) return false; const d = c.getContext("2d").getImageData(0, 0, c.width, c.height).data; for (let i = 3; i < d.length; i += 4) if (d[i]) return true; return false; })()`, sel)).(bool)
	}
	for _, sel := range []string{".shapeview .wave", ".shapeview .curve", ".shapeview .spec", ".shapeview .envc"} {
		tb.WaitFor(fmt.Sprintf(`true`), "")
		deadline := time.Now().Add(5 * time.Second)
		for !drawn(sel) && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		if !drawn(sel) {
			t.Errorf("nothing drawn in %s", sel)
		}
	}
	// a macro made a constant by its menu, then a number of a card: the chain is written
	tb.Eval(`(() => { const m = S().querySelector(".shapeview .sv-card:nth-child(2) .mc"); m.value = "0"; m.dispatchEvent(new Event("change", {bubbles: true})); return 0; })()`)
	tb.Eval(`(() => { const r = S().querySelector(".shapeview .sv-card:nth-child(2) .sv-arg .num"); r.value = "3"; r.dispatchEvent(new Event("change", {bubbles: true})); return 0; })()`)
	paneltest.Eventually(t, "the chain with the gain 3", func() bool { return strings.Contains(textOf(pt.Texts(id), "chainA"), "fold 3") })
	// a stage added after the picked one
	tb.Eval(`(() => { const s = S().querySelector(".shapeview .sv-add select"); s.value = "softclip"; s.dispatchEvent(new Event("change", {bubbles: true})); return 0; })()`)
	paneltest.Eventually(t, "softclip in the chain", func() bool { return strings.Contains(textOf(pt.Texts(id), "chainA"), "softclip") })
	tb.WaitFor(`S().querySelectorAll(".shapeview .sv-card").length === 3`, "three cards")
	// the text typed makes the cards
	tb.Eval(`(() => { const t = S().querySelector(".shapeview .sv-text"); t.value = "bipolar | tube 2 0.3 | czsaw m1"; t.dispatchEvent(new Event("input", {bubbles: true})); return 0; })()`)
	tb.WaitFor(`S().querySelectorAll(".shapeview .sv-card").length === 3 && S().querySelector(".shapeview .sv-card:nth-child(2) .sv-name").textContent === "tube"`, "the cards of the typed chain")
	paneltest.Eventually(t, "the typed chain in the module", func() bool { return strings.Contains(textOf(pt.Texts(id), "chainA"), "tube 2 0.3") })
	// a bypassed stage is a comment line the chain still reads
	tb.Eval(`(() => { S().querySelector(".shapeview .sv-card:nth-child(2) .sv-x").click(); return 0; })()`)
	paneltest.Eventually(t, "the stage bypassed", func() bool { return strings.Contains(textOf(pt.Texts(id), "chainA"), "#off | tube") })
	if e := pt.TextError(id, "chainA"); e != "" {
		t.Errorf("a bypassed stage should still read: %s", e)
	}
	// the envelope: a fade in sets the attack of the amp envelope
	tb.Eval(`(() => { [...S().querySelectorAll(".shapeview .sv-fades button")].find(b => b.textContent === "Fade in").click(); return 0; })()`)
	paneltest.Eventually(t, "the attack of the fade in", func() bool {
		v := pt.SettingsOf(id).Values()["ampAttack"]
		return v > 1000
	})
	if dir := os.Getenv("SHAPEVIEW_SHOT"); dir != "" {
		time.Sleep(600 * time.Millisecond)
		tb.Eval(`(() => { const w = document.querySelector(".pt-win"); w.style.height = "1640px"; w.style.top = "10px"; return 0; })()`)
		tb.Screenshot(dir)
	}
}

func textOf(texts []patcher.Text, key string) string {
	for _, t := range texts {
		if t.Key == key {
			return t.Value
		}
	}
	return ""
}
