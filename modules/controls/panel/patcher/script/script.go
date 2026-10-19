package script

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/shapechain"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/dop251/goja"
)

// Script is a module of JavaScript. Its top level runs when it starts (to make its generators);
// then trigger(ev) on every trigger ({step, seconds}: the step of the clock, how long it is) and
// value(v, port) on every value on input "a" or "b". It plays notes (note), sends values (send),
// triggers (fire) and a value for mod cables (mod):
//
//	note(pitch, velocity = 0.8, steps = 1)  // a pitch (48, "C3") or a chord ([48, 51, 55])
//	set(key, value)  or  set({key: value})  // settings of the synths it plays, by their keys (morph, macro1, chainA...)
//	send(port, v)                           // "a" or "b"
//	fire()                                  // a trigger on its trigger output
//	mod(v)                                  // 0-1, to its mod output
//	log(...)                                // under the module
//	input(port)                             // the last value on "a" or "b"
//	chord("Dm7")                           // close(lowest), bass(lowest), tones(lo, hi), contains(p), voiceLead(prev)
//	pitch("C3"), scale("C minor"), bpm()    // scale: snap(p), pitch(degree, octave), notes(lo, hi), contains(p), step(p, n)
//	genny.*, shared                         // generators (see Gen), what every script of the patch sees
//	song.section, song.bar, song.progress  // where the song of the patch is (a song module); section(name, index) when one starts
//
// Inputs: trigger, a, b; outputs: notes, a, b, trigger, mod
type Script struct {
	*psalm.Module
	rt      *Runtime
	pending atomic.Pointer[goja.Program] // a script compiled, to start
	status  atomic.Pointer[string]
	modVal  atomic.Uint64 // the bits of mod's value

	// on the audio thread
	api            *goja.Object
	trigger, value goja.Callable
	section        goja.Callable // called when a section of the song starts
	song           *patcher.Song
	songObj        *goja.Object // song in the script
	passing        [2]int       // the section of the song passing
	inSong         bool
	stopped        bool // too long: not called again, until a new script
	ev             patcher.Event
	tick           psalm.CtrlTick
	inputs         [2]goja.Value
	notes          []queued
	sets           []psalm.Message // settings to send to the synths at the end of the tick
	targets        []psalm.Receiver
}

// queued is a note to play at the end of the tick
type queued struct {
	tick     psalm.CtrlTick
	pitches  []float64
	velocity float64
	seconds  float64
}

// New returns a script of the runtime of the patch
func New(rt *Runtime) *Script {
	s := &Script{Module: psalm.NewModule(nil, nil), rt: rt}
	s.SetSelf(s)
	s.song = patcher.SongOf(s.Env())
	s.setStatus("")
	return s
}

// SetCode compiles a script (off the audio thread) and has it start (on the audio thread, at the
// next tick); its errors, the script as it was going on
func (s *Script) SetCode(code string) error {
	p, err := Compile("script", code)
	if err != nil {
		s.setStatus("error: " + err.Error())
		return err
	}
	s.pending.Store(p)
	return nil
}

// Status is what it says: its error, or what it logged last
func (s *Script) Status() string { return *s.status.Load() }

func (s *Script) setStatus(text string) { s.status.Store(&text) }

// fail shows an error of the script
func (s *Script) fail(err error) { s.setStatus("error: " + err.Error()) }

// Mod is the value of its mod output
func (s *Script) Mod() float64 { return math.Float64frombits(s.modVal.Load()) }

// SetTargets sets the synths it plays (on the audio thread)
func (s *Script) SetTargets(targets []psalm.Receiver) { s.targets = targets }

// start starts a script compiled (on the audio thread)
func (s *Script) start() {
	p := s.pending.Swap(nil)
	if p == nil {
		return
	}
	s.trigger, s.value, s.section, s.stopped, s.inSong = nil, nil, nil, false, false
	s.setStatus("")
	if s.api == nil {
		s.api = s.newAPI()
	}
	wrapper, err := s.rt.vm.RunProgram(p)
	if err != nil {
		s.fail(cleanError(err))
		return
	}
	f, _ := goja.AssertFunction(wrapper)
	res, err := s.rt.call(StartBudget, f, s.api, s.rt.shared, s.rt.genny)
	if err != nil {
		s.fail(err)
		return
	}
	o := res.ToObject(s.rt.vm)
	s.trigger, _ = goja.AssertFunction(o.Get("trigger"))
	s.value, _ = goja.AssertFunction(o.Get("value"))
	s.section, _ = goja.AssertFunction(o.Get("section"))
}

// follow moves song on to where the song is at a tick, and calls section when one starts
func (s *Script) follow(tick psalm.CtrlTick) {
	if s.songObj == nil {
		return
	}
	pos, ok := s.song.At(tick, s.Config().PPQ)
	o := s.songObj
	if !ok || pos.Ended {
		_ = o.Set("playing", false)
		_ = o.Set("section", "")
		_ = o.Set("index", -1)
		s.inSong = false
		return
	}
	_ = o.Set("playing", true)
	_ = o.Set("section", pos.Section.Name)
	_ = o.Set("index", pos.Section.Index)
	_ = o.Set("bar", pos.SectionBar())
	_ = o.Set("bars", pos.Section.Bars)
	_ = o.Set("progress", pos.Progress)
	_ = o.Set("loop", pos.Loop)
	if key := [2]int{pos.Loop, pos.Section.Index}; !s.inSong || key != s.passing {
		s.inSong, s.passing = true, key
		_ = o.Set("sections", s.song.Transport().Sections())
		s.run(s.section, s.rt.vm.ToValue(pos.Section.Name), s.rt.vm.ToValue(pos.Section.Index))
	}
}

// Restart starts the song again with the patch (Psalm.Restart): its first section is called again
func (s *Script) Restart() { s.inSong = false }

// run calls a function of the script, within its budget
func (s *Script) run(fn goja.Callable, args ...goja.Value) {
	if fn == nil || s.stopped {
		return
	}
	if _, err := s.rt.call(CallBudget, fn, args...); err != nil {
		s.fail(err)
		s.stopped = strings.HasPrefix(err.Error(), "it took too long")
	}
}

// CtrlTick starts a new script, and follows the song
func (s *Script) CtrlTick(tick psalm.CtrlTick) {
	s.start()
	if tick.SampleTimestamp > s.tick.SampleTimestamp {
		s.tick = tick
	}
	s.follow(tick)
}

// Receive takes a trigger (input 0) and values (inputs 1 and 2: "a" and "b")
func (s *Script) Receive(v any, index int) {
	s.start()
	vm := s.rt.vm
	switch index {
	case 0:
		ev, _ := v.(patcher.Event)
		s.ev = ev
		if ev.Tick.SampleTimestamp > s.tick.SampleTimestamp {
			s.tick = ev.Tick
		}
		s.follow(ev.Tick) // the section it falls in, before the trigger
		o := vm.NewObject()
		_ = o.Set("step", ev.Step)
		_ = o.Set("seconds", ev.Seconds)
		s.run(s.trigger, o)
	case 1, 2:
		jv := vm.ToValue(v)
		s.inputs[index-1] = jv
		s.run(s.value, jv, vm.ToValue(string(rune('a'+index-1))))
	}
}

// late plays the notes of the tick
func (s *Script) late(psalm.CtrlTick) {
	for _, m := range s.sets { // the settings first: the notes of the tick sound with them
		for _, t := range s.targets {
			t.ReceiveMessage(m)
		}
	}
	s.sets = s.sets[:0]
	for _, n := range s.notes {
		for _, pitch := range n.pitches {
			msg := psalm.Message{"command": "trigger", "tick": n.tick, "duration": n.seconds, "amplitude": n.velocity,
				"message": psalm.Message{"pitch": pitch, "velocity": n.velocity}}
			for _, t := range s.targets {
				t.ReceiveMessage(msg)
			}
		}
	}
	s.notes = s.notes[:0]
}

// port is the index of an output or input of a script's name ("a", "b", or 0, 1)
func (s *Script) port(v goja.Value) int {
	switch v.String() {
	case "a", "0":
		return 0
	case "b", "1":
		return 1
	}
	panic(s.rt.vm.NewTypeError(fmt.Sprintf("no port %q: it is \"a\" or \"b\"", v.String())))
}

// pitches are the pitches of a value of a script: a number, a note name, or an array of them
func (s *Script) pitches(v goja.Value) []float64 {
	vals := s.rt.values(v)
	out := make([]float64, 0, len(vals))
	for _, x := range vals {
		p, err := pitchOf(x)
		if err != nil {
			panic(s.rt.vm.NewGoError(err))
		}
		out = append(out, p)
	}
	return out
}

// goValue is a value of a script for other modules: a number, a chord ([]float64), or what it is
func (s *Script) goValue(v goja.Value) any {
	if o, ok := v.(*goja.Object); ok && o.ClassName() == "Array" {
		return s.pitches(v)
	}
	switch x := v.Export().(type) {
	case string:
		if p, err := pitchOf(v); err == nil {
			return p
		}
		return x
	case int64:
		return float64(x)
	}
	return v.Export()
}

// newAPI is what a script can do (made on the audio thread, on the runtime)
func (s *Script) newAPI() *goja.Object {
	vm := s.rt.vm
	api := vm.NewObject()
	set := func(name string, f func(call goja.FunctionCall) goja.Value) { _ = api.Set(name, f) }
	number := func(call goja.FunctionCall, i int, def float64) float64 {
		if a := call.Argument(i); !goja.IsUndefined(a) && !goja.IsNull(a) {
			return a.ToFloat()
		}
		return def
	}
	set("note", func(call goja.FunctionCall) goja.Value {
		step := s.ev.Seconds
		if step <= 0 { // no trigger yet: a sixteenth
			step = 15 / s.BPM()
		}
		s.notes = append(s.notes, queued{tick: s.tick, pitches: s.pitches(call.Argument(0)),
			velocity: math.Max(0, math.Min(number(call, 1, 0.8), 1)),
			seconds:  math.Max(0.005, number(call, 2, 1)*step)})
		return goja.Undefined()
	})
	// set(key, value) or set({key: value, ...}): settings of the synths the notes go to, by their
	// keys as on their pages (morph, macro1, cutoff, ampAttack...); a number, a string (a shaper
	// synth's chains: set("chainB", "mult 3 | sin")) or true/false
	set("set", func(call goja.FunctionCall) goja.Value {
		msg := psalm.Message{}
		add := func(key string, v goja.Value) {
			switch x := v.Export().(type) {
			case float64:
				msg[key] = x
			case int64:
				msg[key] = float64(x)
			case string:
				// a chain of the shaper synth is read here: a chain that can't be read is an error of
				// the script (with its place), and the synth gets it read
				if len(key) == 6 && key[:5] == "chain" && key[5] >= 'A' && key[5] <= 'D' && strings.TrimSpace(x) != "" {
					c, err := shapechain.Parse(x)
					if err != nil {
						panic(vm.NewTypeError(fmt.Sprintf("set %s: %v", key, err)))
					}
					msg[key] = c
				} else {
					msg[key] = x
				}
			case bool:
				msg[key] = map[bool]float64{true: 1, false: 0}[x]
			default:
				panic(vm.NewTypeError(fmt.Sprintf("set: %q is a number, a string or true/false", key)))
			}
		}
		if o, ok := call.Argument(0).(*goja.Object); ok && o.ClassName() == "Object" {
			for _, k := range o.Keys() {
				add(k, o.Get(k))
			}
		} else if k := call.Argument(0); !goja.IsUndefined(k) && len(call.Arguments) >= 2 {
			add(k.String(), call.Argument(1))
		} else {
			panic(vm.NewTypeError("set(key, value) or set({key: value})"))
		}
		if len(msg) > 0 && len(s.sets) < 256 {
			s.sets = append(s.sets, msg)
		}
		return goja.Undefined()
	})
	set("send", func(call goja.FunctionCall) goja.Value {
		s.Send(s.goValue(call.Argument(1)), s.port(call.Argument(0)))
		return goja.Undefined()
	})
	set("fire", func(goja.FunctionCall) goja.Value {
		ev := s.ev
		ev.Tick = s.tick
		s.Send(ev, 2)
		return goja.Undefined()
	})
	set("mod", func(call goja.FunctionCall) goja.Value {
		s.modVal.Store(math.Float64bits(math.Max(0, math.Min(number(call, 0, 0), 1))))
		return goja.Undefined()
	})
	set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = show(a)
		}
		s.setStatus(strings.Join(parts, " "))
		return goja.Undefined()
	})
	set("input", func(call goja.FunctionCall) goja.Value {
		if v := s.inputs[s.port(call.Argument(0))]; v != nil {
			return v
		}
		return goja.Undefined()
	})
	set("pitch", func(call goja.FunctionCall) goja.Value {
		p, err := pitchOf(call.Argument(0))
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(p)
	})
	set("scale", func(call goja.FunctionCall) goja.Value {
		o, err := s.rt.scaleObject(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return o
	})
	set("chord", func(call goja.FunctionCall) goja.Value {
		o, err := s.rt.chordObject(call.Argument(0).String())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return o
	})
	set("bpm", func(goja.FunctionCall) goja.Value { return vm.ToValue(s.BPM()) })
	s.songObj = vm.NewObject()
	for k, v := range map[string]any{"playing": false, "section": "", "index": -1, "bar": 0, "bars": 0, "progress": 0.0, "loop": 0, "sections": []string{}} {
		_ = s.songObj.Set(k, v)
	}
	_ = api.Set("song", s.songObj)
	return api
}

// show is a value of a script as log shows it: arrays and objects as JSON
func show(v goja.Value) string {
	if o, ok := v.(*goja.Object); ok {
		if _, fn := goja.AssertFunction(v); !fn {
			if b, err := o.MarshalJSON(); err == nil {
				return string(b)
			}
		}
	}
	return v.String()
}
