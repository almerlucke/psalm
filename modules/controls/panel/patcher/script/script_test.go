package script_test

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

// played keeps the notes a synth gets: their pitch, the sample they start on, how long they are
type played struct {
	*psalm.Module
	pitches, seconds []float64
	at               []int64
}

func (r *played) ReceiveMessage(m psalm.Message) {
	if m["command"] != "trigger" {
		return
	}
	r.pitches = append(r.pitches, m.MustMsg("message").MustFltVal("pitch"))
	r.at = append(r.at, m.MustTickVal("tick").SampleTimestamp)
	r.seconds = append(r.seconds, m.MustFltVal("duration"))
}

// rig is a patch of 120 bpm with a clock, scripts and a synth that records
type rig struct {
	t   *testing.T
	p   *psalm.Psalm
	pt  *patcher.Patcher
	rec *played
}

func newRig(t *testing.T) *rig {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	t.Cleanup(func() { psalm.PopConfig() })
	p := psalm.New(nil)
	rec := &played{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	kinds := append(patcher.Generators(), script.Kind(script.NewRuntime()), patcher.Kind{Name: "rec", Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}},
		Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: rec}, nil }})
	return &rig{t: t, p: p, pt: patcher.New(p, panel.New(p, "Test"), "Patch", kinds), rec: rec}
}

func (r *rig) add(kind, preset string) string {
	r.t.Helper()
	id, err := r.pt.Add(kind, preset, 0, 0)
	if err != nil {
		r.t.Fatal(err)
	}
	return id
}

func (r *rig) connect(c patcher.Cable) {
	r.t.Helper()
	if _, err := r.pt.Connect(c); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) code(id, code string) {
	r.t.Helper()
	if err := r.pt.SetText(id, "code", code); err != nil {
		r.t.Fatal(err)
	}
}

// run runs the patch for some seconds
func (r *rig) run(seconds float64) {
	for range int(seconds * 44100 / 256) {
		r.p.ResetRun()
		r.p.Run()
	}
}

func (r *rig) status(id string) string { return r.pt.Status(id) }

// a clock of sixteenths into a script into a synth: the script's notes, on the clock's steps
func TestScriptPlays(t *testing.T) {
	r := newRig(t)
	clock, s, syn := r.add("clock", ""), r.add("script", ""), r.add("rec", "")
	r.code(s, `
const minor = scale("C minor")
const notes = genny.sequence(["C3", "E3", "G3", "B3"].map(pitch))
function trigger(ev) {
  note(minor.snap(notes.next()), 0.8, 2)
  if (ev.step === 1) note(["C2", 43])
}`)
	r.connect(patcher.Cable{From: clock, To: s})
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
	r.run(2)
	if st := r.status(s); st != "" {
		t.Fatalf("status %q", st)
	}
	if len(r.rec.pitches) < 16 {
		t.Fatalf("%d notes in two seconds of sixteenths", len(r.rec.pitches))
	}
	if want := []float64{48, 51, 36, 43, 55, 58, 48}; !slices.Equal(r.rec.pitches[:7], want) {
		t.Errorf("pitches %v, want %v", r.rec.pitches[:7], want)
	}
	if d := r.rec.at[4] - r.rec.at[0]; math.Abs(float64(d)-2*5512.5) > 1 {
		t.Errorf("two steps are %d samples, want 11025", d)
	}
	if math.Abs(r.rec.seconds[0]-0.25) > 1e-9 || math.Abs(r.rec.seconds[2]-0.125) > 1e-9 {
		t.Errorf("lengths %v, want two steps (0.25 s) and one (0.125 s)", r.rec.seconds[:3])
	}
}

// every preset starts and plays without an error
func TestPresets(t *testing.T) {
	for _, preset := range script.Presets {
		t.Run(preset, func(t *testing.T) {
			r := newRig(t)
			clock, s, syn := r.add("clock", ""), r.add("script", preset), r.add("rec", "")
			r.connect(patcher.Cable{From: clock, To: s})
			r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
			r.run(1)
			if st := r.status(s); strings.HasPrefix(st, "error") {
				t.Fatal(st)
			}
			if preset != "values" && len(r.rec.pitches) < 7 {
				t.Fatalf("%d notes", len(r.rec.pitches))
			}
		})
	}
}

// errors: of compiling (its line, the script before it plays on), of running (on the module), of
// a script that goes on too long (stopped; the rest plays on)
func TestScriptErrors(t *testing.T) {
	r := newRig(t)
	clock, s, syn := r.add("clock", ""), r.add("script", ""), r.add("rec", "")
	r.connect(patcher.Cable{From: clock, To: s})
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})

	r.code(s, "function trigger(ev) {\n  note(48\n}")
	if err := r.pt.TextError(s, "code"); !strings.HasPrefix(err, "line 3") {
		t.Fatalf("compiling: %q, want an error on line 3", err)
	}
	r.run(0.5)
	if len(r.rec.pitches) == 0 {
		t.Fatal("the script before it stopped playing")
	}

	r.code(s, "function trigger(ev) {\n  note(nope)\n}")
	r.run(0.2)
	if st := r.status(s); !strings.Contains(st, "nope") || !strings.Contains(st, "line 2") {
		t.Errorf("status %q, want the error of nope, on line 2", st)
	}

	r.code(s, `note("H9")`)
	r.run(0.1)
	if st := r.status(s); !strings.HasPrefix(st, "error") {
		t.Errorf("status %q, want the error of a note name", st)
	}

	// too long: in the script, in a generator of it (a function* that genny calls)
	for _, code := range []string{
		"function trigger(ev) { while (true) {} }",
		"const g = genny.from(function* () { while (true) {} })\nfunction trigger(ev) { note(g.next()) }",
		"while (true) {}",
	} {
		r.code(s, code)
		start := time.Now()
		r.run(0.5)
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("%q: half a second took %v", code, d)
		}
		if st := r.status(s); !strings.Contains(st, "too long") {
			t.Errorf("%q: status %q, want it stopped", code, st)
		}
	}

	r.code(s, "function trigger(ev) { note(60) }")
	n := len(r.rec.pitches)
	r.run(0.5)
	if r.status(s) != "" || len(r.rec.pitches) == n || r.rec.pitches[len(r.rec.pitches)-1] != 60 {
		t.Errorf("after a stop a new script does not play: status %q", r.status(s))
	}
}

// scripts talk: by values and triggers on cables, by shared; a script plays a notes module
func TestScriptsTogether(t *testing.T) {
	r := newRig(t)
	clock, a, b, notes, syn := r.add("clock", ""), r.add("script", ""), r.add("script", ""), r.add("notes", ""), r.add("rec", "")
	r.code(a, `
shared.count = 0
function trigger(ev) {
  shared.count++
  send("a", ev.step % 2 ? "G3" : [48, 52])
  send("b", ev.step)
  if (ev.step % 4 === 0) fire()
  mod(0.25)
}`)
	r.code(b, `
let fired = 0
function value(v, port) { if (port === "b") log("b", v, "a", input("a"), "count", shared.count, "fired", fired) }
function trigger() { fired++ }`)
	r.connect(patcher.Cable{From: clock, To: a})
	r.connect(patcher.Cable{From: a, Out: 1, To: b, In: 1})             // a → a
	r.connect(patcher.Cable{From: a, Out: 2, To: b, In: 2})             // b → b
	r.connect(patcher.Cable{From: a, Out: 3, To: b, In: 0})             // trigger
	r.connect(patcher.Cable{From: a, Out: 1, To: notes, In: 1})         // a → the pitch of notes
	r.connect(patcher.Cable{From: a, Out: 3, To: notes, In: 0})         // the trigger of notes
	r.connect(patcher.Cable{From: notes, To: syn, Type: patcher.Notes}) // notes
	r.run(1)
	st := r.status(b)
	var step, count, fired int
	var at string
	if _, err := fmt.Sscanf(st, "b %d a %s count %d fired %d", &step, &at, &count, &fired); err != nil {
		t.Fatalf("status %q: %v", st, err)
	}
	if count != step+1 || fired != (step+3)/4 { // the fire of a step comes after its values (b logs)
		t.Errorf("status %q: count %d, fired %d at step %d", st, count, fired, step)
	}
	if !strings.Contains(st, "[48,52]") && !strings.Contains(st, "55") {
		t.Errorf("status %q: input a is not the value sent", st)
	}
	if len(r.rec.pitches) < 4 || r.rec.pitches[0] != 48 || r.rec.pitches[1] != 52 {
		t.Errorf("notes played %v: want the chord of step 0 first", r.rec.pitches)
	}
}
