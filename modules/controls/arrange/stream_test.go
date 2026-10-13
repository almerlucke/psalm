package arrange

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
)

// runStream drives a streamer for beats and returns the triggers
func runStream(f *Form, sc StreamScore, beats float64) []psalm.Message {
	p := psalm.New(nil)
	rec := &recorder{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	s := f.Stream(p, rec, "test", sc)
	ppq := int64(psalm.PPQ())
	for t := int64(0); t < int64(beats*float64(ppq)); t++ {
		s.CtrlTick(psalm.CtrlTick{TickTimestamp: t})
	}
	return rec.triggers
}

func beatAt(m psalm.Message) float64 {
	return float64(m["tick"].(psalm.CtrlTick).TickTimestamp) / float64(psalm.PPQ())
}

func pitch(m psalm.Message) any { return m["message"].(psalm.Message)["pitch"] }

func TestStreamSteps(t *testing.T) {
	f := NewForm(Section{Name: "a", Bars: 2})
	got := runStream(f, func(s Section) *Stream {
		return &Stream{
			Rhythm:  sequence.NewLoop(1.0, -1.0, 0.5, 0.5),
			Message: template.Template{"pitch": sequence.NewLoop(60.0, 62.0, 64.0)},
			Amp:     0.5,
			Legato:  0.5,
		}
	}, 6)
	// steps: a note of 1, a rest of 1, two notes of 0.5, then again from beat 3
	wantBeats := []float64{0, 2, 2.5, 3, 5, 5.5}
	wantPitches := []float64{60, 62, 64, 60, 62, 64}
	if len(got) != len(wantBeats) {
		t.Fatalf("expected %d notes, got %d", len(wantBeats), len(got))
	}
	beatSec := 60 / psalm.BPM()
	for i := range got {
		if beatAt(got[i]) != wantBeats[i] || pitch(got[i]) != wantPitches[i] {
			t.Fatalf("note %d: beat %v pitch %v, want %v %v", i, beatAt(got[i]), pitch(got[i]), wantBeats[i], wantPitches[i])
		}
	}
	if d := got[0]["duration"].(float64); math.Abs(d-0.5*beatSec) > 1e-9 || got[0]["amplitude"] != 0.5 {
		t.Fatalf("first note duration %v amplitude %v", d, got[0]["amplitude"])
	}
}

func TestStreamChord(t *testing.T) {
	f := NewForm(Section{Name: "a", Bars: 1})
	got := runStream(f, func(s Section) *Stream {
		return &Stream{
			Rhythm:  sequence.NewLoop(4.0),
			Message: template.Template{"pitch": sequence.NewLoop([]float64{60, 64, 67})},
		}
	}, 1)
	if len(got) != 3 || pitch(got[0]) != 60.0 || pitch(got[2]) != 67.0 || beatAt(got[2]) != 0 {
		t.Fatalf("expected a chord of three notes, got %v", got)
	}
}

func TestStreamsPerSection(t *testing.T) {
	f := NewForm(Section{Name: "a", Bars: 1}, Section{Name: "quiet", Bars: 1}, Section{Name: "b", Bars: 1}, Section{Name: "a2", Bars: 1})
	keep := &Stream{Rhythm: sequence.NewLoop(1.0), Message: template.Template{"pitch": sequence.NewLoop(60.0, 61.0, 62.0, 63.0, 64.0, 65.0, 66.0, 67.0)}}
	got := runStream(f, func(s Section) *Stream {
		switch s.Name {
		case "quiet":
			return nil
		case "b":
			return &Stream{Rhythm: sequence.NewLoop(2.0), Message: template.Template{"pitch": 80.0}}
		}
		return keep // the same stream in a and a2: it goes on where it was
	}, 16)
	var pitches []any
	for _, m := range got {
		pitches = append(pitches, pitch(m))
	}
	want := []any{60.0, 61.0, 62.0, 63.0, 80.0, 80.0, 64.0, 65.0, 66.0, 67.0}
	if len(pitches) != len(want) {
		t.Fatalf("got %v, want %v", pitches, want)
	}
	for i := range want {
		if pitches[i] != want[i] {
			t.Fatalf("got %v, want %v", pitches, want)
		}
	}
	if beatAt(got[4]) != 8 || beatAt(got[6]) != 12 {
		t.Fatalf("expected b to start at beat 8 and a2 at 12, got %v and %v", beatAt(got[4]), beatAt(got[6]))
	}
}

func TestSectionParams(t *testing.T) {
	f := NewForm(
		Section{Name: "dark", Bars: 1, Params: map[string]any{"cutoff": 400.0}},
		Section{Name: "bright", Bars: 1, Params: map[string]any{"cutoff": 4000.0}, Dynamics: 0.5},
	)
	stream := &Stream{
		Rhythm:  sequence.NewLoop(4.0),
		Message: template.Template{"pitch": 60.0, "cutoff": template.NewParameter("cutoff", 1000.0)},
	}
	got := runStream(f, func(s Section) *Stream { return stream }, 8)
	if len(got) != 2 {
		t.Fatalf("expected two notes, got %d", len(got))
	}
	if c := got[0]["message"].(psalm.Message)["cutoff"]; c != 400.0 {
		t.Fatalf("dark cutoff %v", c)
	}
	if c := got[1]["message"].(psalm.Message)["cutoff"]; c != 4000.0 {
		t.Fatalf("bright cutoff %v", c)
	}
	if a := got[1]["amplitude"]; a != 0.5 {
		t.Fatalf("the section dynamics should scale the amplitude, got %v", a)
	}
}

func TestSendParams(t *testing.T) {
	f := NewForm(
		Section{Name: "a", Bars: 1, Params: map[string]any{"mix": 0.2}},
		Section{Name: "b", Bars: 1, Params: map[string]any{"mix": 0.8, "freeze": 1.0}},
	)
	a := NewArrangement(f)
	rec := &recorder{Module: psalm.NewControl()}
	a.SendParams(rec)
	for t := int64(0); t < int64(8*psalm.PPQ()); t++ {
		a.CtrlTick(psalm.CtrlTick{TickTimestamp: t})
	}
	if len(rec.triggers) != 2 || rec.triggers[0]["mix"] != 0.2 || rec.triggers[1]["freeze"] != 1.0 {
		t.Fatalf("expected the parameters of both sections, got %v", rec.triggers)
	}
}

func TestStreamEndsWithTheSong(t *testing.T) {
	f := NewForm(Section{Name: "a", Bars: 1})
	f.Loop = false
	got := runStream(f, func(s Section) *Stream {
		return &Stream{Rhythm: sequence.NewLoop(1.0), Message: template.Template{"pitch": 60.0}}
	}, 8)
	if len(got) != 4 {
		t.Fatalf("expected the 4 beats of the song, got %d notes", len(got))
	}
}

// valueRecorder records plain values per input
type valueRecorder struct {
	*psalm.Module
	values map[int][]any
}

func (r *valueRecorder) Receive(value any, index int) {
	r.values[index] = append(r.values[index], value)
}

func TestStreamMessagesAndValues(t *testing.T) {
	f := NewForm(Section{Name: "a", Bars: 1, Params: map[string]any{"mix": 0.7}})
	p := psalm.New(nil)
	rec := &valueRecorder{Module: psalm.NewControl(), values: map[int][]any{}}
	rec.SetSelf(rec)

	// messages as they are, to input 0, a slice splits them
	f.Stream(p, rec, "settings", func(s Section) *Stream {
		return &Stream{
			Output:  Messages,
			Rhythm:  sequence.NewLoop(2.0),
			Message: template.Template{"grainSize": sequence.NewLoop(50.0, 100.0), "mix": template.NewParameter("mix", 0.0), "bang": true},
		}
	})
	// values to input 3
	st := f.NewStream(p, "density", func(s Section) *Stream {
		return &Stream{Output: Values, Rhythm: sequence.NewLoop(1.0), Message: template.Template{"value": sequence.NewLoop(10.0, 20.0, 30.0)}}
	})
	rec.ValueIn(st, 0, 3)

	for _, m := range p.Modules() {
		if sm, ok := m.(*Streamer); ok {
			for tk := int64(0); tk < int64(4*psalm.PPQ()); tk++ {
				sm.CtrlTick(psalm.CtrlTick{TickTimestamp: tk})
			}
		}
	}

	msgs := rec.values[0]
	if len(msgs) != 2 {
		t.Fatalf("expected two messages, got %v", msgs)
	}
	first := msgs[0].(psalm.Message)
	if first["grainSize"] != 50.0 || first["mix"] != 0.7 || first["bang"] != true || first["command"] != nil {
		t.Fatalf("expected the template message as it is, got %v", first)
	}
	vals := rec.values[3]
	if len(vals) != 4 || vals[0] != 10.0 || vals[2] != 30.0 {
		t.Fatalf("expected plain values on input 3, got %v", vals)
	}
}

// fader is a module muted with a fade, like a mixer strip
type fader struct {
	*psalm.Module
	mutes []bool
}

func (f *fader) SetMute(m bool) { f.mutes = append(f.mutes, m) }

func TestPlay(t *testing.T) {
	f := NewForm(Section{Name: "intro", Bars: 1}, Section{Name: "verse", Bars: 1}, Section{Name: "chorus", Bars: 1})
	a := NewArrangement(f)
	plain := psalm.NewControl()
	strip := &fader{Module: psalm.NewControl()}
	var plainMuted []bool
	a.Play(plain, "verse", "chorus").Play(strip, "chorus")
	a.OnSection(func(pos Position) { plainMuted = append(plainMuted, plain.Muted()) })
	for tk := int64(0); tk < int64(12*psalm.PPQ()); tk++ {
		a.CtrlTick(psalm.CtrlTick{TickTimestamp: tk})
	}
	if want := []bool{true, false, false}; !slices.Equal(plainMuted, want) {
		t.Fatalf("plain module muted per section %v, want %v", plainMuted, want)
	}
	if want := []bool{true, true, false}; !slices.Equal(strip.mutes, want) {
		t.Fatalf("strip muted per section %v, want %v (with its own fade)", strip.mutes, want)
	}

	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "chrous") {
			t.Fatalf("expected a panic naming the unknown section, got %v", r)
		}
	}()
	a.Play(plain, "chrous")
}
