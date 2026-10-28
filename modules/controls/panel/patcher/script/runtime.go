// Package script is the Script module of the patcher (modules/controls/panel/patcher): JavaScript
// (goja) that makes music in a patch. A script gets a trigger (and values on its inputs), and plays
// notes, sends values and triggers; it builds generators with genny (genny.sequence, bucket,
// arpeggio, repeat, and, or, rand, rwalk, from a function or a function*), quantizes with scales,
// and keeps what every script of the patch can see in shared.
//
// Every script of a patch runs in one runtime, each in a scope of its own, all on the audio thread.
// A script is compiled off the audio thread when it is typed; every call into it has a time budget:
// one that runs too long is stopped, its error shown on the module, the rest of the patch plays on.
//
//	rt := script.NewRuntime()
//	kinds = append(kinds, script.Kind(rt))
package script

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sync"
	"time"

	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/dop251/goja"
)

// Budgets are how long a script may take: to start (its top level, making its generators) and on
// every trigger or value
var (
	StartBudget = 50 * time.Millisecond
	CallBudget  = 5 * time.Millisecond
	// RenderBudget is what a call may take when the patch renders and does not play: there is no deadline, so the budget
	// only stops a script that never ends (a busy machine or a pause of the garbage collector must not stop one)
	RenderBudget = 20 * time.Second
)

// budget is the time a call of a script may take: the budget given when the patch plays, RenderBudget when it renders
func budget(live bool, b time.Duration) time.Duration {
	if live {
		return b
	}
	return max(b, RenderBudget)
}

// Runtime is the JavaScript of a patch: every script in it, each in a scope of its own; a shared
// object they can all see. It is used on one goroutine (the audio thread), but for Compile
type Runtime struct {
	vm     *goja.Runtime
	shared *goja.Object
	genny  *goja.Object

	// the watchdog: it stops a call that goes on beyond its budget (the call it was set for only)
	timer  *time.Timer
	mu     sync.Mutex
	armed  uint64 // the call the timer is for
	budget time.Duration
}

// NewRuntime returns a runtime with genny, scales and an empty shared object
func NewRuntime() *Runtime {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.UncapFieldNameMapper())
	r := &Runtime{vm: vm, shared: vm.NewObject()}
	r.timer = time.AfterFunc(time.Hour, r.stop)
	r.timer.Stop()
	r.genny = r.gennyObject()
	return r
}

// header and footer wrap a script in a function of its own: its scope, the api handed in; the
// header is on the script's first line, so the lines of errors are the script's
const (
	header = `(function(api, shared, genny) { const {note, set, send, fire, clock, mod, log, input, pitch, scale, chord, harmony, bpm, song} = api; const console = {log: log}; `
	footer = "\n;return {trigger: typeof trigger === \"function\" ? trigger : undefined, value: typeof value === \"function\" ? value : undefined, section: typeof section === \"function\" ? section : undefined};\n})"
)

// Compile compiles a script (from any goroutine): its errors, with their lines
func Compile(name, code string) (*goja.Program, error) {
	p, err := goja.Compile(name, header+code+footer, false)
	if err != nil {
		return nil, cleanError(err)
	}
	return p, nil
}

var compileError = regexp.MustCompile(`^(?:SyntaxError: )?\S+: Line (\d+):\d+ (.*?)(?: \(and \d+ more errors?\))?$`)

// cleanError makes an error of goja read as the script's: its message, its line in front
func cleanError(err error) error {
	msg := err.Error()
	var ex *goja.Exception
	if errors.As(err, &ex) {
		msg = ex.Value().String()
		for _, f := range ex.Stack() {
			if f.SrcName() == "script" {
				return fmt.Errorf("line %d: %s", f.Position().Line, msg)
			}
		}
	}
	if m := compileError.FindStringSubmatch(msg); m != nil { // "SyntaxError: script: Line 3:1 Unexpected token } (and 3 more errors)"
		return errors.New("line " + m[1] + ": " + m[2])
	}
	return errors.New(msg)
}

// stop stops the call that is going on beyond its budget (the watchdog)
func (r *Runtime) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.armed%2 == 1 { // a call is going on
		r.vm.Interrupt(fmt.Sprintf("it took too long: stopped after %v", r.budget))
	}
}

// call calls a function of a script within a budget (on the audio thread)
func (r *Runtime) call(budget time.Duration, fn goja.Callable, args ...goja.Value) (goja.Value, error) {
	r.mu.Lock()
	r.armed++ // odd: going on
	r.budget = budget
	r.mu.Unlock()
	r.timer.Reset(budget)
	v, err := fn(goja.Undefined(), args...)
	r.mu.Lock()
	r.armed++ // even: no more stopping it
	r.mu.Unlock()
	r.timer.Stop()
	r.vm.ClearInterrupt() // a stop that came as the call ended
	if err != nil {
		var ie *goja.InterruptedError
		if errors.As(err, &ie) {
			return nil, fmt.Errorf("%v", ie.Value())
		}
		return nil, cleanError(err)
	}
	return v, nil
}

// pitchOf returns the pitch of a value of a script: a number, or a note name ("C3")
func pitchOf(v goja.Value) (float64, error) {
	if s, ok := v.Export().(string); ok {
		return score.Pitch(s)
	}
	return v.ToFloat(), nil
}

// scaleObject is a scale for a script: snap(p), pitch(degree, octave), notes(low, high), contains(p)
func (r *Runtime) scaleObject(key string) (*goja.Object, error) {
	sc, err := harmony.Key(key)
	if err != nil {
		return nil, err
	}
	o := r.vm.NewObject()
	_ = o.Set("snap", func(p float64) float64 { return sc.Snap(p) })
	_ = o.Set("pitch", func(degree, octave int) float64 { return sc.Pitch(degree, octave) })
	_ = o.Set("notes", func(low, high float64) []float64 { return sc.Pitches(low, high) })
	_ = o.Set("contains", func(p float64) bool { return sc.Contains(p) })
	_ = o.Set("step", func(p float64, n int) float64 { return sc.Step(p, n) })
	return o, nil
}

// chordObject is a chord for a script: close(lowest) the chord in root position from the root at or
// above lowest, bass(lowest) its bass note, tones(low, high) every chord tone between two pitches,
// contains(p), voiceLead(prev) the chord for the same number of voices as an array of pitches, moving
// as little as it can; name and root (a pitch class)
func (r *Runtime) chordObject(symbol string) (*goja.Object, error) {
	c, err := harmony.ParseChord(symbol)
	if err != nil {
		return nil, err
	}
	return r.chordObjectOf(c), nil
}

// chordObjectOf is chordObject for a chord
func (r *Runtime) chordObjectOf(c harmony.Chord) *goja.Object {
	o := r.vm.NewObject()
	_ = o.Set("name", c.Name)
	_ = o.Set("root", c.Root)
	_ = o.Set("close", func(lowest float64) []float64 { return c.Close(lowest) })
	_ = o.Set("bass", func(lowest float64) float64 { return c.BassPitch(lowest) })
	_ = o.Set("contains", func(p float64) bool { return c.Contains(p) })
	_ = o.Set("tones", func(low, high float64) []float64 {
		out := []float64{}
		for p := math.Ceil(low); p <= high; p++ {
			if c.Contains(p) {
				out = append(out, p)
			}
		}
		return out
	})
	_ = o.Set("voiceLead", func(prev []float64) []float64 { return harmony.VoiceLead(prev, c) })
	return o
}
