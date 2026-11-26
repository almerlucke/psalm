package script

import (
	"fmt"
	"math"
	"strconv"
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
//	note(pitch, velocity = 0.8, steps = 1, delay = 0) // a pitch (48, "C3") or a chord ([48, 51, 55]); delay steps later (0.1: a tenth of a step), to play off the grid
//	set(key, value)  or  set({key: value})  // settings of the synths it plays, by their keys (morph, macro1, chainA...)
//	send(port, v)                           // "a" or "b"
//	fire(delay = 0, count = 1)              // a trigger on its trigger output: later (in steps, 0.5 halfway the next), or count of them in a step (a ratchet)
//	clock(division = "1/16", swing = 0.5)   // its own clock, on the tempo of the patch: trigger(ev) on its steps without a clock cabled; clock(false) stops it
//	mod(v)                                  // 0-1, to its mod output
//	log(...)                                // a line in the log of the patch (the bar at the top), with the time and the name of this module
//	input(port)                             // the last value on "a" or "b"
//	chord("Dm7")                           // close(lowest), bass(lowest), tones(lo, hi), contains(p), voiceLead(prev)
//	harmony()                              // the chord of the harmony module now, as chord() gives one (null: none)
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
	journal atomic.Pointer[patcher.Journal] // the log of the patch (nil: not in a patch)
	name    atomic.Pointer[string]          // what its lines in the log start with
	modVal  atomic.Uint64                   // the bits of mod's value

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
	fires          []fired        // triggers to come, from fire(delay, count)
	later          []delayed      // notes to come, from note(pitch, velocity, steps, delay)
	own            *patcher.Clock // its own clock, when it has asked for one (clock(division))
	ownOn          bool
	lastTick       int64
	sets           []psalm.Message // settings to send to the synths at the end of the tick
	targets        []psalm.Receiver
}

// fired is a trigger to send when its tick comes
type fired struct {
	due float64 // the tick (a fraction: between two)
	ev  patcher.Event
}

// delayed is a note to play when its tick comes
type delayed struct {
	due int64
	queued
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
	s := &Script{Module: psalm.NewModule(nil, nil), rt: rt, own: patcher.NewClock()}
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

// Attach is given the log of the patch and the name of the module (a patcher.Made.Attach): its log() goes there
func (s *Script) Attach(j *patcher.Journal, name string) {
	s.name.Store(&name)
	s.journal.Store(j)
}

// Status is what it says: its error, or what it logged last (alone)
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
	s.fires, s.later, s.ownOn = s.fires[:0], s.later[:0], false
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
	res, err := s.rt.call(budget(s.Live(), StartBudget), f, s.api, s.rt.shared, s.rt.genny)
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
	if tr := s.song.Transport(); tr != nil {
		beats := tr.Form().BeatsPerBar
		_ = o.Set("beats", beats) // beats to the bar, and where in the bar the song is (a beat of a quarter note)
		_ = o.Set("beat", math.Mod(pos.Beat, beats))
	}
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
	if _, err := s.rt.call(budget(s.Live(), CallBudget), fn, args...); err != nil {
		s.fail(err)
		s.stopped = strings.HasPrefix(err.Error(), "it took too long")
	}
}

// CtrlTick starts a new script, and follows the song
func (s *Script) CtrlTick(tick psalm.CtrlTick) {
	s.start()
	if tick.TickTimestamp < s.lastTick { // the patch started over: its samples are counted from 0 again, and the notes of the script are timed by the tick it has now
		s.fires, s.later = s.fires[:0], s.later[:0]
		s.tick = tick
	}
	if tick.SampleTimestamp > s.tick.SampleTimestamp {
		s.tick = tick
	}
	s.lastTick = tick.TickTimestamp
	s.follow(tick)
	if s.ownOn { // its own clock: a trigger on its steps
		s.own.Play(tick, s.triggered)
	}
	if len(s.fires) > 0 { // the triggers fire(delay) asked for, when they are due
		keep := s.fires[:0]
		for _, f := range s.fires {
			if f.due <= float64(tick.TickTimestamp)+1e-9 {
				f.ev.Tick = tick
				s.Send(f.ev, 2)
			} else {
				keep = append(keep, f)
			}
		}
		s.fires = keep
	}
}

// Receive takes a trigger (input 0) and values (inputs 1 and 2: "a" and "b")
func (s *Script) Receive(v any, index int) {
	s.start()
	vm := s.rt.vm
	switch index {
	case 0:
		ev, _ := v.(patcher.Event)
		s.triggered(ev)
	case 1, 2:
		jv := vm.ToValue(v)
		s.inputs[index-1] = jv
		s.run(s.value, jv, vm.ToValue(string(rune('a'+index-1))))
	}
}

// triggered is a trigger of the clock (or of its own): trigger(ev) is called
func (s *Script) triggered(ev patcher.Event) {
	s.start()
	s.ev = ev
	if ev.Tick.SampleTimestamp > s.tick.SampleTimestamp {
		s.tick = ev.Tick
	}
	s.follow(ev.Tick) // the section it falls in, before the trigger
	o := s.rt.vm.NewObject()
	_ = o.Set("step", ev.Step)
	_ = o.Set("seconds", ev.Seconds)
	s.run(s.trigger, o)
}

// late plays the notes of the tick
func (s *Script) late(tick psalm.CtrlTick) {
	for _, m := range s.sets { // the settings first: the notes of the tick sound with them
		for _, t := range s.targets {
			t.ReceiveMessage(m)
		}
	}
	s.sets = s.sets[:0]
	s.play(s.notes)
	s.notes = s.notes[:0]
	if len(s.later) > 0 { // the notes note(.., delay) asked for, when they are due: they sound at this tick
		keep := s.later[:0]
		for _, d := range s.later {
			if d.due <= tick.TickTimestamp {
				d.tick = tick
				s.play([]queued{d.queued})
			} else {
				keep = append(keep, d)
			}
		}
		s.later = keep
	}
}

// play sends notes to the synths
func (s *Script) play(notes []queued) {
	for _, n := range notes {
		for _, pitch := range n.pitches {
			msg := psalm.Message{"command": "trigger", "tick": n.tick, "duration": n.seconds, "amplitude": n.velocity,
				"message": psalm.Message{"pitch": pitch, "velocity": n.velocity}}
			for _, t := range s.targets {
				t.ReceiveMessage(msg)
			}
		}
	}
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
		q := queued{tick: s.tick, pitches: s.pitches(call.Argument(0)),
			velocity: max(0, min(number(call, 1, 0.8), 1)),
			seconds:  max(0.005, number(call, 2, 1)*step)}
		// delay: later, in steps; the tick it falls on is the nearest (a tick is a few milliseconds)
		if delay := number(call, 3, 0); delay > 0 {
			ticks := delay * step * s.BPM() * float64(s.Config().PPQ) / 60
			if due := s.tick.TickTimestamp + int64(math.Round(ticks)); due > s.tick.TickTimestamp && len(s.later) < 4096 {
				s.later = append(s.later, delayed{due: due, queued: q})
				return goja.Undefined()
			}
		}
		s.notes = append(s.notes, q)
		return goja.Undefined()
	})
	// set(key, value) or set({key: value, ...}): settings of the synths the notes go to, by their
	// keys as on their pages (morph, macro1, cutoff, ampAttack...); a number, a string (a shaper
	// synth's chains: set("chainB", "mult 3 | sin")) or true/false
	set("set", func(call goja.FunctionCall) goja.Value {
		msg := psalm.Message{}
		add := func(key string, v goja.Value) {
			exported := v.Export()
			// the list a generator's take() gives is a list of values of the script, not of Go values
			if o, ok := v.(*goja.Object); ok && o.ClassName() == "Array" {
				n := int(o.Get("length").ToInteger())
				list := make([]any, 0, min(n, 4097))
				for i := 0; i < n && i <= 4096; i++ {
					list = append(list, o.Get(strconv.Itoa(i)).Export())
				}
				exported = list
			}
			switch x := exported.(type) {
			case float64:
				msg[key] = x
			case int64:
				msg[key] = float64(x)
			case string:
				// a chain of the shaper synth is read here: a chain that can't be read is an error of
				// the script (with its place), and the synth gets it read
				if (len(key) == 6 && key[:5] == "chain" && key[5] >= 'A' && key[5] <= 'D' || len(key) == 7 && key[:6] == "chain2" && key[6] >= 'A' && key[6] <= 'D') && strings.TrimSpace(x) != "" {
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
			case []any: // a list of numbers: values for the grains of a granular synth one after the other (positionSeq), a scale, a stack
				list := make([]float64, 0, len(x))
				for _, e := range x {
					switch n := e.(type) {
					case float64:
						list = append(list, n)
					case int64:
						list = append(list, float64(n))
					default:
						panic(vm.NewTypeError(fmt.Sprintf("set: %q is a list of numbers", key)))
					}
				}
				if len(list) > 4096 {
					panic(vm.NewTypeError(fmt.Sprintf("set: %q has more than 4096 numbers", key)))
				}
				msg[key] = list
			default:
				panic(vm.NewTypeError(fmt.Sprintf("set: %q is a number, a string, a list of numbers or true/false", key)))
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
	// fire(delay = 0, count = 1): a trigger on the trigger output, carrying the event being handled; delay steps later (0.5: halfway the next step), count
	// of them (a ratchet) spread over one step, the first after the delay
	set("fire", func(call goja.FunctionCall) goja.Value {
		delay, count := max(0, number(call, 0, 0)), int(max(1, min(number(call, 1, 1), 64)))
		step := s.ev.Seconds
		if step <= 0 { // no trigger yet: a sixteenth
			step = 15 / s.BPM()
		}
		stepTicks := step * s.BPM() * float64(s.Config().PPQ) / 60
		for i := range count {
			ev := s.ev
			ev.Seconds = step / float64(count)
			if at := delay + float64(i)/float64(count); at <= 0 {
				ev.Tick = s.tick
				s.Send(ev, 2)
			} else if len(s.fires) < 4096 {
				s.fires = append(s.fires, fired{due: float64(s.tick.TickTimestamp) + at*stepTicks, ev: ev})
			}
		}
		return goja.Undefined()
	})
	// clock(division = "1/16", swing = 0.5): the script keeps its own clock on the tempo of the patch, trigger(ev) on its steps, as if a clock were
	// cabled to it; clock(false) stops it. A division is 1/4, 1/8, 1/8T, 1/16, 1/16T or 1/32
	set("clock", func(call goja.FunctionCall) goja.Value {
		if on, ok := call.Argument(0).Export().(bool); ok && !on {
			s.ownOn = false
			return goja.Undefined()
		}
		division := "1/16"
		if a := call.Argument(0); !goja.IsUndefined(a) && !goja.IsNull(a) {
			division = a.String()
		}
		if err := s.own.Set(division, number(call, 1, 0.5)); err != nil {
			panic(vm.NewTypeError("clock: " + err.Error()))
		}
		s.ownOn = true
		return goja.Undefined()
	})
	set("mod", func(call goja.FunctionCall) goja.Value {
		s.modVal.Store(math.Float64bits(max(0, min(number(call, 0, 0), 1))))
		return goja.Undefined()
	})
	set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, a := range call.Arguments {
			parts[i] = show(a)
		}
		text := strings.Join(parts, " ")
		if j, name := s.journal.Load(), s.name.Load(); j != nil && name != nil {
			j.Add(*name, text)
		} else {
			s.setStatus(text) // alone, without a patch
		}
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
	// harmony(): the chord the harmony module has at this moment, as chord("Dm7") is (null: no harmony, or no chord)
	set("harmony", func(goja.FunctionCall) goja.Value {
		if h := patcher.HarmonyOf(s.Env()); h != nil {
			if c, ok := h.ChordAt(s.tick); ok {
				return s.rt.chordObjectOf(c)
			}
		}
		return goja.Null()
	})
	set("bpm", func(goja.FunctionCall) goja.Value { return vm.ToValue(s.BPM()) })
	s.songObj = vm.NewObject()
	for k, v := range map[string]any{"playing": false, "section": "", "index": -1, "bar": 0, "bars": 0, "progress": 0.0, "loop": 0, "beats": 4.0, "beat": 0.0, "sections": []string{}} {
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
