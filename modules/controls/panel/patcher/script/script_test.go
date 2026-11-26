package script_test

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

// played keeps the notes a synth gets: their pitch, the sample they start on, how long they are
type played struct {
	*psalm.Module
	pitches, seconds []float64
	at               []int64
	settings         []psalm.Message
	order            []string
}

func (r *played) ReceiveMessage(m psalm.Message) {
	if m["command"] != "trigger" {
		r.settings = append(r.settings, m)
		r.order = append(r.order, "set")
		return
	}
	r.order = append(r.order, "note")
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

// logged is the text of the last line of the log of the patch (after its time and the module that wrote it)
func (r *rig) logged() string {
	lines := strings.Split(r.pt.Journal().Text(), "\n")
	if parts := strings.SplitN(lines[len(lines)-1], "  ", 3); len(parts) == 3 {
		return parts[2]
	}
	return ""
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
			if preset != "values" && len(r.rec.pitches) < 4 {
				t.Fatalf("%d notes", len(r.rec.pitches))
			}
		})
	}
}

// errors: of compiling (its line, the script before it plays on), of running (on the module), of
// a script that goes on too long (stopped; the rest plays on)
func TestScriptErrors(t *testing.T) {
	// the rig renders, which has a long budget: the test shortens it to stop the scripts that never end quickly
	defer func(b time.Duration) { script.RenderBudget = b }(script.RenderBudget)
	script.RenderBudget = script.CallBudget
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
	st := r.logged()
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

// set() sends settings to the synths the notes go to, before the notes of the tick
func TestScriptSet(t *testing.T) {
	r := newRig(t)
	clock, s, syn := r.add("clock", ""), r.add("script", ""), r.add("rec", "")
	r.code(s, `
function trigger(ev) {
  if (ev.step === 0) set("morph", 0.25)
  if (ev.step === 1) set({cutoff: 900, chainB: "mult 3 | sin", on: true, steps: ev.step})
  note(48)
}`)
	r.connect(patcher.Cable{From: clock, To: s})
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
	r.run(0.4)
	if st := r.status(s); strings.HasPrefix(st, "error") {
		t.Fatal(st)
	}
	if len(r.rec.settings) < 2 {
		t.Fatalf("%d settings", len(r.rec.settings))
	}
	if m := r.rec.settings[0]; m["morph"] != 0.25 || len(m) != 1 {
		t.Errorf("first %v", m)
	}
	if c, ok := r.rec.settings[1]["chainB"].(*shapechain.Chain); !ok || c.Text() != "mult 3 | sin" {
		t.Errorf("a chain is sent read: %T", r.rec.settings[1]["chainB"])
	}
	if m := r.rec.settings[1]; m["cutoff"] != 900.0 || m["on"] != 1.0 || m["steps"] != 1.0 {
		t.Errorf("second %v", m)
	}
	if r.rec.order[0] != "set" || r.rec.order[1] != "note" {
		t.Errorf("the settings come before the notes of their tick: %v", r.rec.order[:4])
	}
	// a list of numbers goes as it is (a granular synth's positionSeq)
	r.code(s, `const g = genny.rwalk(0.5, 0, 1, 0.1)
function trigger() { set({positionSeq: [0.1, 2, 3], panSeq: g.take(4), scale: "minor"}) }`)
	r.run(0.2)
	if st := r.status(s); strings.HasPrefix(st, "error") {
		t.Errorf("a list of numbers: %s", st)
	}
	if l, ok := r.rec.settings[len(r.rec.settings)-1]["positionSeq"].([]float64); !ok || len(l) != 3 || l[0] != 0.1 || l[1] != 2 {
		t.Errorf("a list is sent as numbers: %v", r.rec.settings[len(r.rec.settings)-1]["positionSeq"])
	}
	if l, ok := r.rec.settings[len(r.rec.settings)-1]["panSeq"].([]float64); !ok || len(l) != 4 {
		t.Errorf("the take of a generator is a list too: %v", r.rec.settings[len(r.rec.settings)-1]["panSeq"])
	}
	for _, code := range []string{`function trigger() { set("chainA", "wobble 3") }`, `function trigger() { set("x", [1, "a"]) }`, `function trigger() { set("x", {a: 1}) }`, `function trigger() { set("x") }`, `function trigger() { set() }`} {
		r.code(s, code)
		r.run(0.2)
		if st := r.status(s); !strings.HasPrefix(st, "error") {
			t.Errorf("%s: status %q, want an error", code, st)
		}
	}
}

// chord() gives a chord of a symbol, its voicings and its tones in a range
func TestScriptChord(t *testing.T) {
	r := newRig(t)
	clock, s, syn := r.add("clock", ""), r.add("script", ""), r.add("rec", "")
	r.code(s, `
const c = chord("Dm7")
const g = chord("Bb/D")
let said = false
function trigger(ev) {
  if (said) return
  said = true
  log(JSON.stringify([c.close(50), c.bass(40), c.tones(60, 70), c.contains(62), c.contains(61), g.bass(40), c.voiceLead([57, 60, 64, 67]), c.name, c.root]))
  note(c.close(50), 0.5, 1)
}`)
	r.connect(patcher.Cable{From: clock, To: s})
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
	r.run(0.3)
	// Dm7 from D3 (50): D F A C; its bass at or above 40 is D (50); the tones from 60 to 70: C D F A; D is in
	// it, C# is not; Bb/D has D in the bass; voice led from A C E G: A C D F
	want := `[[50,53,57,60],50,[60,62,65,69],true,false,50,[57,60,62,65],"Dm7",2]`
	if st := r.logged(); st != want {
		t.Fatalf("%s, want %s", st, want)
	}
	for _, bad := range []string{`chord("Hm")`, `chord("")`, `chord("Dm7wow")`} {
		r.code(s, "function trigger() { "+bad+" }")
		r.run(0.2)
		if st := r.status(s); !strings.HasPrefix(st, "error") {
			t.Errorf("%s: status %q, want an error", bad, st)
		}
	}
}

// a patch that renders has no deadline: a call that takes longer than the budget of one that plays (a busy machine, a
// pause of the garbage collector) is not stopped
func TestScriptRenderHasNoDeadline(t *testing.T) {
	r := newRig(t)
	clock, s, syn := r.add("clock", ""), r.add("script", ""), r.add("rec", "")
	r.connect(patcher.Cable{From: clock, To: s})
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
	r.code(s, "function trigger(ev) { const t = Date.now(); while (Date.now() - t < 3 * 5) {} note(60) }") // three times the budget
	r.run(0.3)
	if st := r.status(s); st != "" {
		t.Errorf("a slow call of a script in a render: status %q", st)
	}
	if len(r.rec.pitches) == 0 {
		t.Error("the slow script did not play")
	}
}

// a script keeps its own clock (clock("1/8")): trigger(ev) on its steps, with no clock cabled
func TestScriptOwnClock(t *testing.T) {
	r := newRig(t)
	s, syn := r.add("script", ""), r.add("rec", "")
	r.code(s, `
clock("1/8")
function trigger(ev) { note(60, 0.8, 1) }`)
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
	r.run(2.1)
	// 120 bpm: an eighth is a quarter of a second, 11025 samples
	if n := len(r.rec.at); n < 8 || n > 9 {
		t.Fatalf("%d notes in two seconds, want 8", n)
	}
	for i := 1; i < len(r.rec.at); i++ {
		if d := r.rec.at[i] - r.rec.at[i-1]; math.Abs(float64(d)-11025) > 2 {
			t.Errorf("note %d is %d samples after the one before, want 11025", i, d)
		}
	}
	for _, bad := range []string{`clock("1/7")`, `clock("1/8", 0.6); clock(false); function trigger() { note(60) }`} {
		r.code(s, bad)
		r.run(0.1)
	}
	if st := r.status(s); strings.HasPrefix(st, "error") {
		t.Errorf("clock(false) after a clock: %q", st)
	}
	r.code(s, `clock("1/7")`)
	r.run(0.1)
	if st := r.status(s); !strings.Contains(st, "division") {
		t.Errorf("a division that is no division: status %q", st)
	}
}

// fire(delay, count): a trigger later, or several in a step
func TestScriptFireLaterAndRatchet(t *testing.T) {
	r := newRig(t)
	a, b, syn := r.add("script", ""), r.add("script", ""), r.add("rec", "")
	r.code(a, `
clock("1/4")
function trigger(ev) { note(60, 0.8, 1); if (ev.step % 2 === 0) fire(0.5); else fire(0, 3) }`)
	r.code(b, `function trigger(ev) { note(72, 0.8, 1) }`)
	r.connect(patcher.Cable{From: a, Out: 3, To: b, In: 0})
	r.connect(patcher.Cable{From: a, To: syn, Type: patcher.Notes})
	r.connect(patcher.Cable{From: b, To: syn, Type: patcher.Notes})
	r.run(2.1)
	var base, extra []int64
	for i, p := range r.rec.pitches {
		if p == 60 {
			base = append(base, r.rec.at[i])
		} else {
			extra = append(extra, r.rec.at[i])
		}
	}
	// a quarter is 22050 samples: an even step fires half a quarter later (11025), an odd one three times, a third of a quarter apart (7350)
	if len(base) < 4 || len(extra) < 2+3 {
		t.Fatalf("%d notes of the script, %d of the triggers", len(base), len(extra))
	}
	near := func(a, b int64) bool { return math.Abs(float64(a-b)) <= 2 }
	if !near(extra[0]-base[0], 11025) {
		t.Errorf("fire(0.5): %d samples after the step, want 11025", extra[0]-base[0])
	}
	if !near(extra[1], base[1]) || !near(extra[2]-extra[1], 7350) || !near(extra[3]-extra[2], 7350) {
		t.Errorf("fire(0, 3): at %v for the step at %d, want a third of a step apart", extra[1:4], base[1])
	}
}

// note(pitch, velocity, steps, delay): a note later in the step, off the grid by a tick at most
func TestScriptNoteDelay(t *testing.T) {
	r := newRig(t)
	s, syn := r.add("script", ""), r.add("rec", "")
	r.code(s, `
clock("1/4")
function trigger(ev) { note(60, 0.8, 1); note(72, 0.8, 1, 0.5); note(84, 0.8, 1, -1) }`)
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
	r.run(2.1)
	var base, late []int64
	for i, p := range r.rec.pitches {
		switch p {
		case 60:
			base = append(base, r.rec.at[i])
		case 72:
			late = append(late, r.rec.at[i])
		}
	}
	if len(base) < 4 || len(late) < 3 {
		t.Fatalf("%d notes on the step, %d delayed", len(base), len(late))
	}
	// a quarter is 22050 samples, half of it 11025; a tick is 92 samples
	for i := range late {
		if d := late[i] - base[i]; math.Abs(float64(d)-11025) > 92 {
			t.Errorf("delay 0.5: %d samples after the step, want 11025", d)
		}
	}
	if len(r.rec.pitches) < len(base)*3-1 { // a negative delay is no delay: the note plays on the step
		t.Errorf("%d notes, want three a step", len(r.rec.pitches))
	}
}
