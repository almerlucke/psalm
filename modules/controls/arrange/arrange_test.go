package arrange

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/score"
)

func testForm() *Form {
	return NewForm(
		Section{Name: "Intro", Bars: 2},
		Section{Name: "Verse", Bars: 4, Dynamics: 0.5},
		Section{Name: "Outro", Bars: 2},
	)
}

func TestForm(t *testing.T) {
	f := testForm()
	if f.Bars() != 8 || f.Beats() != 32 {
		t.Fatalf("expected 8 bars, got %d", f.Bars())
	}
	v, ok := f.Section("Verse")
	if !ok || v.StartBar != 2 || v.Index != 1 || v.Dynamics != 0.5 {
		t.Fatalf("unexpected verse %+v", v)
	}
	if intro, _ := f.Section("Intro"); intro.Dynamics != 1 {
		t.Fatal("dynamics should default to 1")
	}
	if s := f.Schedule(); !strings.Contains(s, "Verse") || !strings.Contains(s, "(loops)") {
		t.Fatalf("unexpected schedule:\n%s", s)
	}
}

func TestCompile(t *testing.T) {
	f := testForm()
	notes := f.Compile("lead", func(s Section) []score.Note {
		if s.Name == "Intro" {
			return nil
		}
		return PerBar(s, func(bar int) []score.Note { return score.Notes(2, 60+float64(bar), 0) })
	})
	if score.Beats(notes) != f.Beats() {
		t.Fatalf("the part should last the song, %v beats", score.Beats(notes))
	}
	if !notes[0].IsRest() || notes[0].Beats != 8 {
		t.Fatal("the intro should be a rest")
	}
	if notes[1].Amp != 0.5 {
		t.Fatalf("the verse dynamics should scale the notes, amp %v", notes[1].Amp)
	}
	last := notes[len(notes)-2]
	if last.Amp != 0 || last.Pitches[0] != 61 {
		t.Fatalf("the outro plays at its own level, got %+v", last)
	}
}

func TestCompilePanicsOnWrongLength(t *testing.T) {
	defer func() {
		r := recover()
		msg, _ := r.(string)
		if !strings.Contains(msg, "bass") || !strings.Contains(msg, "Verse") || !strings.Contains(msg, "expected 16") {
			t.Fatalf("expected a panic naming the part and section, got %v", r)
		}
	}()
	testForm().Compile("bass", func(s Section) []score.Note {
		if s.Name == "Verse" {
			return score.Notes(1, 40, 40) // 2 beats instead of 16
		}
		return nil
	})
}

func TestPosition(t *testing.T) {
	a := NewArrangement(testForm())
	ppq := int64(psalm.PPQ())
	pos := a.PositionAt(4*4*ppq + 2*ppq) // bar 4, two beats in: the verse, halfway its third bar
	if pos.Bar != 4 || pos.Section.Name != "Verse" || pos.SectionBar() != 2 || math.Abs(pos.Progress-(10.0/16)) > 1e-9 {
		t.Fatalf("unexpected position %+v", pos)
	}
	pos = a.PositionAt(32*ppq + ppq) // the second time around
	if pos.Loop != 1 || pos.Bar != 0 || pos.Section.Name != "Intro" {
		t.Fatalf("unexpected position after a loop %+v", pos)
	}
}

func TestArrangement(t *testing.T) {
	f := testForm()
	a := NewArrangement(f)
	ppq := int64(psalm.PPQ())

	var values []float64
	var valueTicks []int64
	var tick int64
	a.AutomateFunc(Lane{Default: 100, Sections: map[string]Curve{"Verse": Exp(200, 3200)}}, func(v float64) {
		values = append(values, v)
		valueTicks = append(valueTicks, tick)
	})
	var sections []string
	bars := 0
	a.OnSection(func(pos Position) { sections = append(sections, pos.Section.Name) })
	a.OnBar(func(pos Position) { bars++ })

	for tick = 0; tick < 2*32*ppq; tick++ { // the song twice
		a.CtrlTick(psalm.CtrlTick{TickTimestamp: tick})
	}

	if got := strings.Join(sections, " "); got != "Intro Verse Outro Intro Verse Outro" {
		t.Fatalf("sections %q", got)
	}
	if bars != 16 {
		t.Fatalf("expected 16 bars, got %d", bars)
	}

	// the lane: 100 in the intro, from 200 up to 3200 through the verse, 100 again in the outro
	valueAt := func(tk int64) float64 {
		v := math.NaN()
		for i, vt := range valueTicks {
			if vt <= tk {
				v = values[i]
			}
		}
		return v
	}
	verseStart, verseEnd := 8*ppq, 24*ppq
	if v := valueAt(verseStart - 1); v != 100 {
		t.Fatalf("intro value %v", v)
	}
	if v := valueAt(verseStart); v != 200 {
		t.Fatalf("verse start value %v", v)
	}
	if v := valueAt(verseStart + 8*ppq); math.Abs(v-800) > 1 { // halfway, two octaves up
		t.Fatalf("verse halfway value %v", v)
	}
	if v := valueAt(verseEnd - 1); v < 3000 {
		t.Fatalf("verse end value %v", v)
	}
	if v := valueAt(verseEnd); v != 100 {
		t.Fatalf("outro value %v", v)
	}
	for i := 1; i < len(values); i++ {
		if values[i] == values[i-1] {
			t.Fatal("values should only be sent when they change")
		}
	}
}

// recorder counts the triggers it receives
type recorder struct {
	*psalm.Module
	triggers []psalm.Message
}

func (r *recorder) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		r.triggers = append(r.triggers, msg)
	}
}

func TestPart(t *testing.T) {
	f := testForm()
	p := psalm.New(nil)
	rec := &recorder{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.AddTo(p)
	f.Part(p, rec, "pad", score.Config{Amp: 0.4}, func(s Section) []score.Note {
		if s.Name != "Verse" {
			return nil
		}
		return PerBar(s, func(bar int) []score.Note { return []score.Note{score.Chord([]float64{60, 64, 67}, 4)} })
	})
	for range int(2 * f.Beats() * 60 / psalm.BPM() * psalm.SampleRate() / float64(psalm.BufferSize())) {
		p.ResetRun()
		p.Run()
	}
	if len(rec.triggers) != 2*4*3 {
		t.Fatalf("expected 4 chords of 3 in each of two loops, got %d triggers", len(rec.triggers))
	}
	if amp := rec.triggers[0]["amplitude"].(float64); math.Abs(amp-0.2) > 1e-9 {
		t.Fatalf("expected the part amplitude times the verse dynamics, got %v", amp)
	}
}

func TestTempo(t *testing.T) {
	base := psalm.CurrentConfig()
	base.BPM = 100
	psalm.PushConfig(base)
	defer psalm.PopConfig()

	f := testForm()
	a := NewArrangement(f).Tempo(Lane{Sections: map[string]Curve{"Verse": Linear(100, 60)}})
	ppq := int64(psalm.PPQ())

	bpmAt := map[int64]float64{}
	for loop := range 3 {
		for tick := int64(0); tick < 32*ppq; tick++ {
			t0 := int64(loop)*32*ppq + tick
			a.CtrlTick(psalm.CtrlTick{TickTimestamp: t0})
			if loop == 0 {
				bpmAt[tick] = psalm.BPM()
			}
		}
	}

	if bpmAt[4*ppq] != 100 {
		t.Fatalf("intro tempo %v", bpmAt[4*ppq])
	}
	if got := bpmAt[16*ppq]; math.Abs(got-80) > 0.1 { // halfway the verse
		t.Fatalf("expected 80 BPM halfway the verse, got %v", got)
	}
	if got := bpmAt[16*ppq+ppq]; math.Abs(got-77.5) > 0.1 { // a beat later, between whole BPM values
		t.Fatalf("expected 77.5 BPM a beat later, got %v", got)
	}
	if got := bpmAt[24*ppq-1]; got > 60.1 {
		t.Fatalf("expected about 60 BPM at the end of the verse, got %v", got)
	}
	if got := bpmAt[24*ppq]; got != 100 {
		t.Fatalf("expected the base tempo back in the outro, got %v", got)
	}

	// three loops later only the base config is on top: the arrangement popped its own configs
	if psalm.BPM() != 100 {
		t.Fatalf("expected the base tempo after the song, got %v", psalm.BPM())
	}
	if popped := psalm.PopConfig(); popped.BPM != 100 || psalm.BPM() == 100 {
		t.Fatal("the arrangement left configs on the stack")
	}
	psalm.PushConfig(base) // for the deferred pop

	// the schedule follows the tempo: the outro starts later than at a steady 100 BPM
	steady := 24 * 60.0 / 100
	sched := a.Schedule()
	var outro float64
	for _, line := range strings.Split(sched, "\n") {
		if strings.Contains(line, "Outro") {
			_, _ = fmt.Sscanf(strings.TrimSpace(line), "%fs", &outro)
		}
	}
	// the verse lasts 16 beats at an average of about 80 BPM: 16*60/80 (roughly, the mean of 1/bpm)
	if outro < steady+1.5 || outro > steady+3.5 {
		t.Fatalf("outro starts at %.1fs, expected about %.1fs\n%s", outro, steady+2.4, sched)
	}
}

// In a patch the tempo lane really changes the speed: at half the tempo half the notes play
func TestTempoInPatch(t *testing.T) {
	count := func(slow bool) int {
		base := psalm.CurrentConfig()
		base.BPM = 120
		psalm.PushConfig(base)
		defer psalm.PopConfig()

		f := NewForm(Section{Name: "All", Bars: 4})
		p := psalm.New(nil)
		rec := &recorder{Module: psalm.NewControl()}
		rec.SetSelf(rec)
		rec.AddTo(p)
		f.Part(p, rec, "beat", score.Config{}, func(s Section) []score.Note {
			return score.Repeat(score.Notes(1, 60), 16)
		})
		a := NewArrangement(f)
		if slow {
			a.Tempo(Lane{Default: 60})
		}
		a.AddTo(p)
		for range int(4 * psalm.SampleRate() / float64(psalm.BufferSize())) { // 4 seconds
			p.ResetRun()
			p.Run()
		}
		return len(rec.triggers)
	}
	fast, slow := count(false), count(true)
	if fast < 7 || fast > 9 || slow < 3 || slow > 5 {
		t.Fatalf("expected about 8 beats in 4 seconds at 120 BPM and 4 at 60, got %d and %d", fast, slow)
	}
}

func TestDuration(t *testing.T) {
	base := psalm.CurrentConfig()
	base.BPM = 120
	psalm.PushConfig(base)
	defer psalm.PopConfig()
	a := NewArrangement(testForm()) // 8 bars of 4 beats
	if d := a.Duration(); math.Abs(d-16) > 0.01 {
		t.Fatalf("8 bars at 120 BPM last 16 s, got %v", d)
	}
	a.Tempo(Lane{Sections: map[string]Curve{"Outro": Const(60)}}) // the last 2 bars at half speed
	if d := a.Duration(); math.Abs(d-20) > 0.01 {
		t.Fatalf("with a slow outro 20 s, got %v", d)
	}
}

// Sections are a lane by themselves (0 elsewhere), Else gives the other sections a value
func TestSections(t *testing.T) {
	plain := Sections{"Verse": Const(2)}.lane()
	other := Sections{"Verse": Const(2)}.Else(5)
	verse, _ := testForm().Section("Verse")
	outro, _ := testForm().Section("Outro")
	if plain.value(Position{Section: verse}) != 2 || plain.value(Position{Section: outro}) != 0 {
		t.Fatal("sections should play their curves and 0 elsewhere")
	}
	if other.value(Position{Section: outro}) != 5 {
		t.Fatal("else should set the other sections")
	}

	// the arrangement takes both
	a := NewArrangement(testForm())
	var got []float64
	a.AutomateFunc(Sections{"Verse": Const(1)}, func(v float64) { got = append(got, v) })
	ppq := int64(psalm.PPQ())
	for tick := int64(0); tick < 32*ppq; tick++ {
		a.CtrlTick(psalm.CtrlTick{TickTimestamp: tick})
	}
	if fmt.Sprint(got) != "[0 1 0]" {
		t.Fatalf("expected 0 1 0 through the song, got %v", got)
	}
}

// A span places curves at bars
func TestSpan(t *testing.T) {
	f := testForm()
	verse := f.Span("Verse") // 4 bars
	at := func(c Curve, bar float64) float64 { return c(bar / float64(verse)) }

	open := verse.Between(1, 2, Linear(0, 10))
	if at(open, 0.5) != 0 || at(open, 1.5) != 5 || at(open, 3) != 10 {
		t.Fatalf("between: %v %v %v", at(open, 0.5), at(open, 1.5), at(open, 3))
	}
	if at(verse.From(2, Linear(0, 10)), 3) != 5 {
		t.Fatal("from should run to the end of the section")
	}
	if at(verse.Last(1, Linear(0, 10)), 3.5) != 5 {
		t.Fatal("last should run over the last bars")
	}

	fade := verse.Fade(-6, 1, 3) // half a bar of fade at each edge
	if at(fade, 0.5) != Silent || at(fade, 2) != -6 || at(fade, 3.5) != Silent {
		t.Fatalf("fade: silent outside, the level inside: %v %v %v", at(fade, 0.5), at(fade, 2), at(fade, 3.5))
	}
	if v := at(fade, 1.25); math.Abs(v-(-6-6.02)) > 0.01 { // halfway the fade in: half the amplitude
		t.Fatalf("fade in should be even in amplitude, %v dB", v)
	}

	carry := verse.Fade(-6, -1, 5) // carried on from the section before and into the one after
	if at(carry, 0) != -6 || at(carry, 4) != -6 {
		t.Fatalf("a fade beyond the section should not fade at its edges: %v %v", at(carry, 0), at(carry, 4))
	}

	steps := verse.Steps(0, 0.5, 1)
	if at(steps, 0.5) != 0 || at(steps, 1.5) != 0.5 || at(steps, 3.5) != 1 {
		t.Fatal("steps should give every bar its value, the last one holding on")
	}
	on := verse.On(1, 2.5)
	if at(on, 0.9) != 0 || at(on, 1) != 1 || at(on, 2.4) != 1 || at(on, 2.5) != 0 {
		t.Fatal("on should be 1 from bar to bar only")
	}

	pedal := verse.Pedal(1, 3)
	for _, c := range []struct {
		bar  float64
		down float64
	}{{0.5, 0}, {1.01, 0}, {1.5, 1}, {2.02, 0}, {2.9, 1}, {3.5, 0}} {
		if at(pedal, c.bar) != c.down {
			t.Fatalf("pedal at bar %v: %v", c.bar, at(pedal, c.bar))
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("a span of a section the form doesn't have should panic")
		}
	}()
	f.Span("Bridge")
}

// At happens at the start of its bar, every time the song gets there
func TestAt(t *testing.T) {
	a := NewArrangement(testForm())
	ppq := int64(psalm.PPQ())
	var ticks []int64
	var tick int64
	a.At("Verse", 2, func() { ticks = append(ticks, tick) })
	for tick = 0; tick < 2*32*ppq; tick++ {
		a.CtrlTick(psalm.CtrlTick{TickTimestamp: tick})
	}
	verseBar2 := (2 + 2) * 4 * ppq // the intro is 2 bars
	if len(ticks) != 2 || ticks[0] != verseBar2 || ticks[1] != 32*ppq+verseBar2 {
		t.Fatalf("expected the third bar of the verse in both loops, got %v", ticks)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("a bar the section doesn't reach should panic")
		}
	}()
	a.At("Verse", 4, func() {})
}

// A bar without notes is a rest of a bar
func TestPerBarRests(t *testing.T) {
	f := testForm()
	f.BeatsPerBar = 3
	notes := f.Compile("lead", func(s Section) []score.Note {
		return PerBar(s, func(bar int) []score.Note {
			if bar%2 == 1 {
				return nil
			}
			return score.Notes(3, 60)
		})
	})
	if score.Beats(notes) != f.Beats() {
		t.Fatalf("the part should last the song, %v of %v beats", score.Beats(notes), f.Beats())
	}
	if !notes[1].IsRest() || notes[1].Beats != 3 {
		t.Fatalf("the second bar should be a rest of a bar, got %+v", notes[1])
	}

	// a note lasting several bars: the bars it covers add nothing
	held := f.Compile("pad", func(s Section) []score.Note {
		return PerBar(s, func(bar int) []score.Note {
			if bar == 0 {
				return score.Notes(6, 60) // two bars
			}
			if bar == 1 {
				return nil
			}
			return score.Notes(3, 62)
		})
	})
	if score.Beats(held) != f.Beats() {
		t.Fatalf("a bar a longer note covers should add nothing, the part lasts %v beats", score.Beats(held))
	}
	for _, n := range held {
		if n.IsRest() {
			t.Fatalf("no bar should rest, got %+v", n)
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("a rest without a bar length should panic")
		}
	}()
	PerBar(Section{Bars: 1}, func(int) []score.Note { return nil })
}

// Only plays a score in its section
func TestOnly(t *testing.T) {
	f := testForm()
	notes := f.Compile("lead", Only("Verse", func(s Section) []score.Note { return score.Notes(16, 60) }))
	if len(notes) != 3 || !notes[0].IsRest() || notes[1].Pitches[0] != 60 || !notes[2].IsRest() {
		t.Fatalf("only the verse should play, got %+v", notes)
	}
}

// Bars writes one section bar by bar
func TestBars(t *testing.T) {
	f := testForm()
	notes := f.Compile("lead", Bars("Verse", func(s Section, bar int) []score.Note {
		if s.Name != "Verse" {
			t.Fatalf("bars of the %s", s.Name)
		}
		if bar == 1 {
			return nil
		}
		return score.Notes(4, 60+float64(bar))
	}))
	if score.Beats(notes) != f.Beats() {
		t.Fatalf("the part should last the song, %v beats", score.Beats(notes))
	}
	if !notes[0].IsRest() || notes[1].Pitches[0] != 60 || !notes[2].IsRest() || notes[3].Pitches[0] != 62 {
		t.Fatalf("the intro rests, the verse plays its bars with a rest in the second: %+v", notes[:4])
	}
}
