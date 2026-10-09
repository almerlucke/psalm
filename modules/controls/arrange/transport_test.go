package arrange

import (
	"slices"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
)

const testPPQ = 4 // a bar of 4 beats is 16 ticks

// same compares positions
func same(a, b Position) bool {
	return a.Tick == b.Tick && a.Bar == b.Bar && a.Section.Index == b.Section.Index && a.Progress == b.Progress && a.Loop == b.Loop && a.Ended == b.Ended
}

// bars plays a number of bars on a transport and returns the section of every bar (- after the
// end), with a function called at the start of every bar to change things on the way
func bars(tr *Transport, tick *int64, n int, at func(bar int)) []string {
	var got []string
	for bar := range n {
		if at != nil {
			at(bar)
		}
		for k := range 16 {
			pos := tr.At(*tick, testPPQ)
			*tick++
			if k == 0 {
				if pos.Ended {
					got = append(got, "-")
				} else {
					got = append(got, pos.Section.Name)
				}
			}
		}
	}
	return got
}

func abc(loop bool) *Form {
	f := NewForm(Section{Name: "A", Bars: 2}, Section{Name: "B", Bars: 1}, Section{Name: "C", Bars: 1})
	f.Loop = loop
	return f
}

func names(s string) []string { return strings.Fields(s) }

func TestTransportAsWritten(t *testing.T) {
	f := abc(true)
	tr := f.Transport()
	for tick := int64(0); tick < 16*4*3; tick++ {
		got, want := tr.At(tick, testPPQ), f.PositionAt(tick, testPPQ)
		if !same(got, want) {
			t.Fatalf("tick %d: the transport should play the form as written: %+v, not %+v", tick, got, want)
		}
	}
	f = abc(false)
	tr = f.Transport()
	for tick := int64(0); tick < 16*6; tick++ {
		if got, want := tr.At(tick, testPPQ), f.PositionAt(tick, testPPQ); !same(got, want) {
			t.Fatalf("tick %d: a song that doesn't loop should end as written: %+v, not %+v", tick, got, want)
		}
	}
}

func TestTransportSkip(t *testing.T) {
	f := abc(true)
	tr := f.Transport().SetEnabled("B", false)
	var tick int64
	if got := bars(tr, &tick, 7, nil); !slices.Equal(got, names("A A C A A C A")) {
		t.Fatalf("a section switched off should be skipped: %v", got)
	}
	if len(tr.Order()) != 2 || tr.Enabled("B") {
		t.Fatal("the order should leave B out")
	}
	tr.SetEnabled("B", true).SetEnabled("A", false)
	tick = 0 // the patch starts again
	if got := bars(tr, &tick, 5, nil); !slices.Equal(got, names("B C B C B")) {
		t.Fatalf("the song should start with the first section that is on: %v", got)
	}
}

func TestTransportQueue(t *testing.T) {
	tr := abc(true).Transport()
	var tick int64
	got := bars(tr, &tick, 6, func(bar int) {
		if bar == 1 {
			tr.Queue("C") // in the second bar of A: at its end
		}
	})
	if !slices.Equal(got, names("A A C A A B")) || tr.Queued() != "" {
		t.Fatalf("a queued section should start when the section playing ends: %v", got)
	}

	tr = abc(true).Transport().SetJump(AtBar)
	tick = 0
	got = bars(tr, &tick, 4, func(bar int) {
		if bar == 1 {
			tr.Queue("C")
		}
	})
	if !slices.Equal(got, names("A C A A")) {
		t.Fatalf("with AtBar a queued section should start at the next bar: %v", got)
	}

	tr = abc(true).Transport().SetJump(AtBar)
	tick = 0
	var at []Position
	for range 16 * 3 {
		if tick == 20 {
			tr.Queue("A") // the start of the same section again
		}
		at = append(at, tr.At(tick, testPPQ))
		tick++
	}
	if at[31].Loop != 0 || at[32].Loop != 1 || at[32].Bar != 0 || at[32].Tick != 0 {
		t.Fatal("going back to a section should count as a loop and start at its first bar")
	}
}

func TestTransportRepeat(t *testing.T) {
	tr := abc(true).Transport()
	var tick int64
	got := bars(tr, &tick, 7, func(bar int) {
		switch bar {
		case 3:
			tr.SetRepeat(true) // at the end of B: B again and again
		case 5:
			tr.SetRepeat(false)
		}
	})
	if !slices.Equal(got, names("A A B B B C A")) {
		t.Fatalf("a repeated section should play until it is let go: %v", got)
	}
	tr = abc(true).Transport().SetRepeat(true)
	tick = 0
	if got := bars(tr, &tick, 4, func(bar int) {
		if bar == 1 {
			tr.Queue("C")
		}
	}); !slices.Equal(got, names("A A C C")) {
		t.Fatalf("a queued section should end the repeat of the section before: %v", got)
	}
}

func TestTransportEnd(t *testing.T) {
	tr := abc(false).Transport()
	var tick int64
	got := bars(tr, &tick, 6, func(bar int) {
		if bar == 5 {
			tr.Queue("B") // after the end: at once
		}
	})
	if !slices.Equal(got, names("A A B C - B")) {
		t.Fatalf("a song that doesn't loop should end, and start again with a queued section: %v", got)
	}
	if s, _ := tr.Playing(); s != "B" {
		t.Fatalf("playing should be B, got %q", s)
	}
	tr = abc(true).Transport()
	for _, s := range tr.Sections() {
		tr.SetEnabled(s, false)
	}
	tick = 0
	if got := bars(tr, &tick, 2, nil); !slices.Equal(got, names("- -")) {
		t.Fatalf("with every section off nothing plays: %v", got)
	}
	if s, _ := tr.Playing(); s != "" {
		t.Fatal("nothing should be playing")
	}
}

func TestTransportPanics(t *testing.T) {
	defer func() {
		if r, _ := recover().(string); !strings.Contains(r, "Bridge") {
			t.Fatalf("an unknown section should panic naming it, got %v", r)
		}
	}()
	abc(true).Transport().Queue("Bridge")
}

// the parts, the streams and the arrangement all follow the transport
func TestTransportInPatch(t *testing.T) {
	f := testForm() // Intro 2, Verse 4, Outro 2 bars
	p := psalm.New(nil)
	rec := &recorder{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.AddTo(p)
	f.Part(p, rec, "lead", score.Config{}, func(s Section) []score.Note {
		pitch := map[string]float64{"Intro": 60, "Verse": 62, "Outro": 64}[s.Name]
		return PerBar(s, func(int) []score.Note { return score.Notes(4, pitch) })
	})
	var sections []string
	a := NewArrangement(f).OnSection(func(pos Position) { sections = append(sections, pos.Section.Name) })
	a.AddTo(p)
	a.Transport().SetEnabled("Verse", false)

	barSeconds := 4 * 60 / psalm.BPM()
	if d := a.Duration(); d < 4*barSeconds-0.01 || d > 4*barSeconds+0.01 {
		t.Fatalf("the duration should leave the verse out: %v, not %v", d, 4*barSeconds)
	}
	if s := a.Schedule(); !strings.Contains(s, "(off)") {
		t.Fatalf("the schedule should show the verse off:\n%s", s)
	}
	for range int(4 * barSeconds * psalm.SampleRate() / float64(psalm.BufferSize())) {
		p.ResetRun()
		p.Run()
	}
	var pitches []float64
	for _, m := range rec.triggers {
		pitches = append(pitches, m["message"].(psalm.Message)["pitch"].(float64))
	}
	if !slices.Equal(pitches, []float64{60, 60, 64, 64}) || !slices.Equal(sections, []string{"Intro", "Outro"}) {
		t.Fatalf("the part and the arrangement should skip the verse: %v %v", pitches, sections)
	}
}

func TestTransportStartQueued(t *testing.T) {
	tr := abc(true).Transport().Queue("B")
	var tick int64
	if got := bars(tr, &tick, 3, nil); !slices.Equal(got, names("B C A")) {
		t.Fatalf("a section queued before the start should start the song: %v", got)
	}
}

func TestTransportControls(t *testing.T) {
	p := psalm.New(nil)
	ctl := panel.New(p, "Test")
	tr := abc(true).Transport().Controls(ctl, "Song")
	cs := ctl.CuesByName("Song")
	if cs == nil || !slices.Equal(cs.Names, names("A B C")) || ctl.Param("Song repeat") == nil || ctl.Param("Song jump") == nil || ctl.Param("Song/B") == nil {
		t.Fatal("the controls should add the sections, the repeat and the jump")
	}
	cs.Queue(2)
	cs.Enable(1, false)
	if tr.Queued() != "C" || tr.Enabled("B") {
		t.Fatal("the tiles should queue and switch the sections")
	}
	ctl.Param("Song repeat").Set(1)
	ctl.Param("Song jump").Set(1)
	ctl.CtrlTick(psalm.CtrlTick{TickTimestamp: 1})
	if !tr.Repeat() || tr.Jump() != AtBar {
		t.Fatal("the switch and the choice should set the repeat and the jump")
	}
	tr.At(0, testPPQ) // queued before the start: the song starts with it
	if st := cs.State(); st.Playing != 2 || st.Queued != -1 || st.Enabled[1] {
		t.Fatalf("the tiles should show the transport: %+v", st)
	}
}

func TestRestartSong(t *testing.T) {
	f := testForm() // Intro 2, Verse 4, Outro 2 bars
	p := psalm.New(nil)
	rec := &recorder{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.AddTo(p)
	f.Part(p, rec, "lead", score.Config{}, func(s Section) []score.Note {
		pitch := map[string]float64{"Intro": 60, "Verse": 62, "Outro": 64}[s.Name]
		return PerBar(s, func(int) []score.Note { return score.Notes(4, pitch) })
	})
	var sections []string
	a := NewArrangement(f).OnSection(func(pos Position) { sections = append(sections, pos.Section.Name) })
	a.AddTo(p)
	streamed := 0
	f.NewStream(p, "stream", func(s Section) *Stream {
		streamed++
		return nil
	})

	bars := func(n float64) {
		for range int(n * 4 * 60 / psalm.BPM() * psalm.SampleRate() / float64(psalm.BufferSize())) {
			p.ResetRun()
			p.Run()
		}
	}
	bars(3.5) // into the verse
	a.Transport().Queue("Outro")
	p.Restart()
	rec.triggers = nil
	bars(1)
	var pitches []float64
	for _, m := range rec.triggers {
		pitches = append(pitches, m["message"].(psalm.Message)["pitch"].(float64))
	}
	if !slices.Equal(sections, names("Intro Verse Intro")) || !slices.Equal(pitches, []float64{60}) {
		t.Fatalf("a restart should play the song from the intro: %v, pitches %v", sections, pitches)
	}
	if a.Transport().Queued() != "" || streamed != 3 {
		t.Fatalf("a restart should forget the section asked for and ask the streams again (%d)", streamed)
	}
}

func TestTransportEnded(t *testing.T) {
	tr := abc(false).Transport()
	if tr.Form() == nil || tr.Ended() {
		t.Fatal("a transport that didn't play yet has not ended")
	}
	tr.SetEnabled("B", false)
	var tick int64
	bars(tr, &tick, 3, nil) // A A C
	if tr.Ended() {
		t.Fatal("the last section playing: not ended yet")
	}
	bars(tr, &tick, 1, nil)
	if !tr.Ended() {
		t.Fatal("after the last section that is on the song ended")
	}
	tr.restart()
	if tr.Ended() {
		t.Fatal("a restart starts the song again")
	}
}
