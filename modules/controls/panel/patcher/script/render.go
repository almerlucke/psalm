package script

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/note"
	"github.com/dop251/goja"
)

// A script can also be rendered: it makes a whole piece at once, off the audio thread, instead of playing a note
// for each trigger. This is how a plugin (cmd/plugins/gennyclap) uses scripts. The script has the same genny.*,
// scale, chord and pitch as one in a patch, and defines render():
//
//	function render(p) {                          // p.seed, p.stepsPerBeat
//	    const key = scale("D minor")
//	    return [ {pitch: key.pitch(1, 3), start: 0, duration: 1, velocity: 0.8, channel: 0}, ... ]
//	}
//
// render returns the notes (times in steps), or {stepsPerBeat, length, notes} when it wants another step or a length
// (in steps) for the loop. The random numbers of genny are seeded with the seed of the call just before the script
// starts, so the same seed gives the same piece

// RenderedNote is a note of a rendered piece: times in steps
type RenderedNote struct {
	note.Event
	Channel int // 0..15
}

// Piece is what a script renders
type Piece struct {
	Notes        []RenderedNote
	StepsPerBeat float64
	// Steps is the length of the piece (a loop) in steps: what the script says, else the notes' end rounded up to a bar
	Steps float64
	// Log is what the script logged with log()
	Log []string
}

// Limits keep a script from asking for unbounded output
const (
	maxRenderNotes = 500000
	maxRenderSteps = 1e6
)

const (
	renderHeader = `(function(api, shared, genny) { const {scale, chord, pitch, log, seed} = api; const console = {log: log}; `
	renderFooter = "\n;return typeof render === \"function\" ? render : undefined;\n})"
)

// RenderScript compiles a script for Render: its errors, with their lines
func RenderScript(name, code string) (*goja.Program, error) {
	p, err := goja.Compile(name, renderHeader+code+renderFooter, false)
	if err != nil {
		return nil, cleanError(err)
	}
	return p, nil
}

// Render renders a script (see above) for a seed. It is used on one goroutine, and the random stream of genny is
// seeded here: callers that render on several goroutines must take turns
func (r *Runtime) Render(code string, seed uint64) (Piece, error) {
	prog, err := RenderScript("script", code)
	if err != nil {
		return Piece{}, err
	}
	var logs []string
	vm := r.vm
	api := vm.NewObject()
	set := func(name string, f func(call goja.FunctionCall) goja.Value) { _ = api.Set(name, f) }
	set("pitch", func(call goja.FunctionCall) goja.Value {
		p, err := pitchOf(call.Argument(0))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(p)
	})
	set("scale", func(call goja.FunctionCall) goja.Value {
		o, err := r.scaleObject(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return o
	})
	set("chord", func(call goja.FunctionCall) goja.Value {
		o, err := r.chordObject(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return o
	})
	set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = show(a)
		}
		if len(logs) < 100 {
			logs = append(logs, strings.Join(parts, " "))
		}
		return goja.Undefined()
	})
	_ = api.Set("seed", seed)

	genny.Seed(seed)
	wrapper, err := vm.RunProgram(prog)
	if err != nil {
		return Piece{}, cleanError(err)
	}
	f, _ := goja.AssertFunction(wrapper)
	res, err := r.call(RenderBudget, f, api, r.shared, r.genny)
	if err != nil {
		return Piece{}, err
	}
	render, ok := goja.AssertFunction(res)
	if !ok {
		return Piece{}, errors.New("the script has no function render()")
	}
	arg := vm.NewObject()
	_ = arg.Set("seed", seed)
	_ = arg.Set("stepsPerBeat", 4)
	out, err := r.call(RenderBudget, render, arg)
	if err != nil {
		return Piece{}, err
	}
	piece, err := r.piece(out)
	piece.Log = logs
	return piece, err
}

// piece reads what render returned
func (r *Runtime) piece(v goja.Value) (Piece, error) {
	p := Piece{StepsPerBeat: 4}
	var list goja.Value = v
	if o, ok := v.(*goja.Object); ok && o.ClassName() == "Object" {
		if spb := optNumber(o, "stepsPerBeat", 4); spb > 0 && spb <= 960 {
			p.StepsPerBeat = spb
		} else {
			return p, fmt.Errorf("stepsPerBeat must be in (0, 960], got %v", spb)
		}
		p.Steps = optNumber(o, "length", 0)
		list = o.Get("notes")
	}
	if a, ok := list.(*goja.Object); !ok || a == nil || a.ClassName() != "Array" {
		return p, errors.New("render() must return an array of notes, or {notes: [...]}")
	}
	var end float64
	for i, e := range r.values(list) {
		if len(p.Notes) >= maxRenderNotes {
			return p, fmt.Errorf("more than %d notes", maxRenderNotes)
		}
		pitch, start := optNumber(e, "pitch", math.NaN()), optNumber(e, "start", 0)
		if v := optValue(e, "pitch"); v != nil {
			if q, err := pitchOf(v); err == nil {
				pitch = q
			}
		}
		if math.IsNaN(pitch) || pitch < 0 || pitch > 127.99 {
			return p, fmt.Errorf("note %d: the pitch %v is outside 0..127", i, pitch)
		}
		if !(start >= 0 && start < maxRenderSteps) {
			return p, fmt.Errorf("note %d: the start %v is outside 0..%g steps", i, start, maxRenderSteps)
		}
		dur := min(max(optNumber(e, "duration", 1), 1e-3), maxRenderSteps)
		ch := int(optNumber(e, "channel", 0))
		if ch < 0 || ch > 15 {
			return p, fmt.Errorf("note %d: the channel %d is outside 0..15", i, ch)
		}
		vel := min(max(optNumber(e, "velocity", 0.8), 0), 1)
		p.Notes = append(p.Notes, RenderedNote{note.Event{Pitch: pitch, Velocity: vel, Start: start, Duration: dur}, ch})
		end = max(end, start+dur)
	}
	if p.Steps <= 0 {
		bar := 4 * p.StepsPerBeat
		p.Steps = max(math.Ceil(end/bar), 1) * bar
	}
	if p.Steps > maxRenderSteps {
		return p, fmt.Errorf("the piece is longer than %g steps", maxRenderSteps)
	}
	return p, nil
}

// optValue is the field of an object of a script (nil if it is not there)
func optValue(v goja.Value, name string) goja.Value {
	if o, ok := v.(*goja.Object); ok {
		if x := o.Get(name); x != nil && !goja.IsUndefined(x) && !goja.IsNull(x) {
			return x
		}
	}
	return nil
}
