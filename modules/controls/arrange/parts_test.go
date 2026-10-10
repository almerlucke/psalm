package arrange

import (
	"slices"
	"testing"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
)

func TestParts(t *testing.T) {
	f := testForm() // Intro 2, Verse 4, Outro 2 bars
	p := psalm.New(nil)
	keys, lead, glass := &recorder{Module: psalm.NewControl()}, &recorder{Module: psalm.NewControl()}, &recorder{Module: psalm.NewControl()}
	for _, r := range []*recorder{keys, lead, glass} {
		r.SetSelf(r)
		r.AddTo(p)
	}
	f.Part(p, keys, "keys", score.Config{}, func(s Section) []score.Note { // a note every bar, everywhere
		return PerBar(s, func(int) []score.Note { return score.Notes(4, 60) })
	})
	f.Part(p, lead, "lead", score.Config{}, Only("Verse", func(s Section) []score.Note {
		return PerBar(s, func(int) []score.Note { return score.Notes(4, 72) })
	}))
	f.Stream(p, glass, "glass", func(s Section) *Stream {
		if s.Name != "Outro" {
			return nil
		}
		return &Stream{Rhythm: sequence.NewLoop(4.0), Message: template.Template{"pitch": 84.0}}
	})
	a := NewArrangement(f)
	a.AddTo(p)
	if !slices.Equal(f.Parts(), []string{"keys", "lead", "glass"}) {
		t.Fatalf("the parts in the order they were added: %v", f.Parts())
	}
	bars := func(n float64) {
		for range int(n * 4 * 60 / psalm.BPM() * psalm.SampleRate() / float64(psalm.BufferSize())) {
			p.ResetRun()
			p.Run()
		}
	}
	bars(1)
	if !f.PartPlaying("keys") || f.PartPlaying("lead") || f.PartPlaying("glass") {
		t.Fatal("in the intro only the keys play")
	}
	f.SetPartEnabled("keys", false)
	bars(2.5) // into the verse
	if !f.PartPlaying("lead") || f.PartEnabled("keys") {
		t.Fatal("in the verse the lead plays; the keys stay off")
	}
	keysBefore := len(keys.triggers)
	bars(2)
	if len(keys.triggers) != keysBefore {
		t.Fatal("a part switched off plays no notes")
	}
	f.SetPartEnabled("keys", true)
	f.SetPartEnabled("glass", false) // before the outro
	bars(1)
	if len(keys.triggers) == keysBefore {
		t.Fatal("switched on again it plays, in time")
	}
	bars(1) // into the outro
	if len(glass.triggers) != 0 || !f.PartPlaying("glass") {
		t.Fatalf("a stream switched off sends nothing, but it plays in the outro: %d", len(glass.triggers))
	}

	// on a panel: tiles that switch, the parts playing lit, kept in presets
	ctl := panel.New(p, "Test")
	f.PartControls(ctl, "Parts")
	cs := ctl.CuesByName("Parts")
	if !cs.Toggles || !slices.Equal(cs.Names, f.Parts()) {
		t.Fatal("the parts as a cue list of switches")
	}
	st := cs.State()
	if !slices.Equal(st.Enabled, []bool{true, true, false}) || !slices.Equal(st.Active, []bool{true, false, true}) {
		t.Fatalf("on: keys and lead; playing in the outro: keys and glass: %+v", st)
	}
	ctl.Param("Parts/lead").Press()
	ctl.CtrlTick(psalm.CtrlTick{TickTimestamp: 1})
	if f.PartEnabled("lead") {
		t.Fatal("the tile's button should switch the part")
	}
	snap := ctl.Snapshot()
	if snap["Parts/lead"] != 0 || snap["Parts/keys"] != 1 {
		t.Fatalf("the parts that are on should be in the snapshot: %v", snap)
	}
}
