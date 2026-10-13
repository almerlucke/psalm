package pianoroll

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
)

func TestMain(m *testing.M) { paneltest.Main(m) }

// the notes drawn, a click on the roll moves the player there
func TestPageRoll(t *testing.T) {
	b := paneltest.Start(t)
	f := &midi.File{Division: 480}
	var tr midi.Track
	tr.AddMeta(0, midi.MetaTempo, midi.TempoData(60))
	for i := range 16 {
		tr.Add(int64(i*480), midi.NoteOnMsg(1, 60+i%5, 100))
		tr.Add(int64(i*480+400), midi.NoteOffMsg(1, 60+i%5, 0))
	}
	f.Tracks = []midi.Track{tr}
	player := midi.NewPlayer(f)
	ctl := panel.New(psalm.New(nil), "Test")
	Add(ctl, "Notes", player, Seconds(8))
	tick := func() { ctl.CtrlTick(psalm.CtrlTick{SampleTimestamp: 1}) }
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".pianoroll canvas")`)
	tb.Size(1200, 600)
	tb.WaitFor(`(() => { const c = document.querySelector(".pianoroll canvas"); if (!c.width) return false; const d = c.getContext("2d").getImageData(0, 0, c.width, c.height).data; let n = 0; for (let i = 0; i < d.length; i += 4) if (d[i + 3] > 0 && Math.max(d[i], d[i + 1], d[i + 2]) - Math.min(d[i], d[i + 1], d[i + 2]) > 60) n++; return n > 200; })()`, "the notes drawn in colour")
	// a click halfway to the right edge from the playhead (at 30%): about 8 * 0.35 = 2.8 s
	x, y := tb.Center(`document.querySelector(".pianoroll canvas")`, 0.5)
	w := tb.Eval(`document.querySelector(".pianoroll canvas").getBoundingClientRect().width`).(float64)
	tb.Mouse("mousePressed", x+w*0.15, y, nil)
	tb.Mouse("mouseReleased", x+w*0.15, y, nil)
	paneltest.Eventually(t, "the player to jump there", func() bool { tick(); p := player.Position(); return p > 2 && p < 3.5 })
}
