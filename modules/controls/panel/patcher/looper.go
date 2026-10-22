package patcher

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/params"
)

// LooperQuantize are the grids a looper can snap what is played to, a part of a quarter note (0: as it was played)
var LooperQuantize = []string{"off", "1/4", "1/8", "1/16", "1/32"}

var looperGrid = []float64{0, 1, 0.5, 0.25, 0.125}

// loopNote is a note of a loop: where it begins in the loop and how long it is, in ticks
type loopNote struct {
	start, length int64
	pitch         float64
	velocity      float64
}

// heldNote is a key pressed while recording: where in the loop (and at which tick) it began
type heldNote struct {
	start, tick int64
	pitch       float64
	velocity    float64
}

// Looper is a live looper for notes. Notes come in (from a keyboard, a note editor, another module) and go on to the synths its output goes to as they
// are. The loop is a number of bars (4/4) on the grid of the patch: it begins at the beginning of the bars of the song and follows the tempo, so what is
// played is always on the beat. Recording begins at the start of the next pass of the loop and ends at the end of a pass, and what was recorded is a layer;
// the layers play together, on every pass, and a new one can be recorded on top of them as long as you like. Settings: "bars" (1-16), "quantize" (an index of
// LooperQuantize: the start and the length of the notes recorded are put on that grid), "record" (a layer is recorded from the next pass; off, it ends with the
// pass), "play" (the layers play), "undo" (takes the last layer away), "clear" (all layers), "export" (writes the layers to a MIDI file, the loop repeated
// "repeats" times) - the last three are pressed and let go by themselves
type Looper struct {
	*psalm.Module
	Bars, Quantize, Repeats int
	Record, Play            bool
	Undo, Clear, Export     bool

	layers    [][]loopNote
	cur       []loopNote
	recording bool
	held      map[string]heldNote
	targets   []psalm.Receiver
	pos, prev int64
	started   bool
	dir       string
	log       func(string)
}

// NewLooper returns a looper of two bars, quantizing to sixteenths
func NewLooper() *Looper {
	l := &Looper{Module: psalm.NewModule(nil, nil), Bars: 2, Quantize: 3, Repeats: 4, Play: true, held: map[string]heldNote{}, dir: "recordings"}
	l.SetSelf(l)
	return l
}

// SetTargets sets the synths it plays (on the audio thread)
func (l *Looper) SetTargets(targets []psalm.Receiver) { l.targets = targets }

// OnEvent sets where the looper tells what it does (the log of the patch)
func (l *Looper) OnEvent(f func(string)) { l.log = f }

func (l *Looper) say(format string, args ...any) {
	if l.log != nil {
		l.log(fmt.Sprintf(format, args...))
	}
}

// length is the ticks of the loop
func (l *Looper) length() int64 { return int64(max(1, l.Bars)) * 4 * int64(l.Config().PPQ) }

// grid is the ticks of the grid notes are put on, 0 for none
func (l *Looper) grid() int64 {
	return int64(math.Round(looperGrid[max(0, min(l.Quantize, len(looperGrid)-1))] * float64(l.Config().PPQ)))
}

// ReceiveMessage takes the notes of the trigger messages (recording them when a layer is being recorded) and passes every message on; a message without a command is
// the settings
func (l *Looper) ReceiveMessage(msg psalm.Message) {
	cmd, ok := msg.StrVal("command")
	if !ok {
		msg.IfIntVal("bars", func(b int) { l.Bars = max(1, min(b, 16)) })
		msg.IfIntVal("quantize", func(q int) { l.Quantize = max(0, min(q, len(LooperQuantize)-1)) })
		msg.IfIntVal("repeats", func(r int) { l.Repeats = max(1, min(r, 64)) })
		msg.IfFltVal("record", func(v float64) { l.Record = v >= 0.5 })
		msg.IfFltVal("play", func(v float64) { l.Play = v >= 0.5 })
		msg.IfFltVal("undo", func(v float64) { l.Undo = v >= 0.5 })
		msg.IfFltVal("clear", func(v float64) { l.Clear = v >= 0.5 })
		msg.IfFltVal("export", func(v float64) { l.Export = v >= 0.5 })
		return
	}
	if cmd == "trigger" && l.recording {
		l.take(msg)
	}
	for _, t := range l.targets {
		t.ReceiveMessage(msg)
	}
}

// Receive takes a message
func (l *Looper) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		l.ReceiveMessage(m)
	}
}

// take records a note on, a note off, or a note of a duration
func (l *Looper) take(msg psalm.Message) {
	tick, ok := msg.TickVal("tick")
	if !ok {
		return
	}
	loop := l.length()
	if id, ok := msg.StrVal("noteOff"); ok {
		h, ok := l.held[id]
		if !ok {
			return
		}
		delete(l.held, id)
		l.add(h.start, tick.TickTimestamp-h.tick, h.pitch, h.velocity)
		return
	}
	m, ok := msg.Msg("message")
	if !ok {
		return
	}
	pitch, ok := m.FltVal("pitch")
	if !ok {
		return
	}
	vel, ok := m.FltVal("velocity")
	if !ok {
		vel, _ = msg.FltVal("amplitude")
	}
	start := tick.TickTimestamp % loop
	if id, ok := msg.StrVal("noteOn"); ok {
		l.held[id] = heldNote{start: start, tick: tick.TickTimestamp, pitch: pitch, velocity: vel}
		return
	}
	if d, ok := msg.FltVal("duration"); ok { // a note of a length (a note generator, a script): the seconds in ticks
		l.add(start, int64(math.Round(d*l.BPM()/60*float64(l.Config().PPQ))), pitch, vel)
	}
}

// add puts a note in the layer being recorded, on the grid
func (l *Looper) add(start, length int64, pitch, velocity float64) {
	loop := l.length()
	if g := l.grid(); g > 0 {
		start = int64(math.Round(float64(start)/float64(g))) * g
		length = max(g, int64(math.Round(float64(length)/float64(g)))*g)
	}
	start %= loop
	length = max(1, min(length, loop))
	l.cur = append(l.cur, loopNote{start: start, length: length, pitch: pitch, velocity: velocity})
}

// late moves the loop on one tick: at the start of a pass a layer that was being recorded is done and the next one begins, and the notes that begin now play
func (l *Looper) late(tick psalm.CtrlTick) {
	loop := l.length()
	l.pos = tick.TickTimestamp % loop
	wrapped := !l.started || l.pos < l.prev
	l.prev, l.started = l.pos, true

	if l.Undo {
		l.Undo = false
		if n := len(l.layers); n > 0 {
			l.layers = l.layers[:n-1]
			l.say("undo: %d layers left", len(l.layers))
		}
	}
	if l.Clear {
		l.Clear, l.layers, l.cur = false, nil, nil
		l.say("cleared")
	}
	if l.Export {
		l.Export = false
		l.export()
	}
	if wrapped {
		if l.recording { // the pass is over: what was recorded is a layer
			for id, h := range l.held { // keys still down are let go at the end
				l.add(h.start, loop-h.start, h.pitch, h.velocity)
				delete(l.held, id)
			}
			if len(l.cur) > 0 {
				l.layers = append(l.layers, l.cur)
				l.say("layer %d: %d notes", len(l.layers), len(l.cur))
			}
			l.cur, l.recording = nil, false
		}
		if l.Record {
			l.recording = true
			l.say("recording layer %d", len(l.layers)+1)
		}
	}
	if !l.Play && !l.recording {
		return
	}
	secondsPerTick := 60 / l.BPM() / float64(l.Config().PPQ)
	for _, layer := range l.layers {
		for _, n := range layer {
			if n.start != l.pos {
				continue
			}
			msg := psalm.Message{"command": "trigger", "tick": tick, "duration": math.Max(0.005, float64(n.length)*secondsPerTick), "amplitude": n.velocity,
				"message": psalm.Message{"pitch": n.pitch, "velocity": n.velocity}}
			for _, t := range l.targets {
				t.ReceiveMessage(msg)
			}
		}
	}
}

// File returns the layers as a MIDI file at the tempo of the patch: a track for every layer, the loop repeated `repeats` times, 480 ticks a quarter note
func (l *Looper) File(repeats int) *midi.File {
	const division = 480
	scale := float64(division) / float64(l.Config().PPQ)
	loop := int64(float64(l.length()) * scale)
	f := &midi.File{Format: 1, Division: division}
	var head midi.Track
	head.AddMeta(0, midi.MetaTempo, midi.TempoData(l.BPM()))
	head.AddMeta(0, midi.MetaTimeSignature, midi.TimeSignatureData(4, 4))
	head.Events = append(head.Events, midi.TrackEvent{Tick: loop * int64(repeats), Meta: midi.MetaEndOfTrack})
	f.Tracks = append(f.Tracks, head)
	for _, layer := range l.layers {
		var tr midi.Track
		for r := range int64(repeats) {
			for _, n := range layer {
				at := r*loop + int64(math.Round(float64(n.start)*scale))
				end := at + max(1, int64(math.Round(float64(n.length)*scale)))
				vel := max(1, min(127, int(math.Round(n.velocity*127))))
				tr.Add(at, midi.NoteOnMsg(1, int(n.pitch), vel))
				tr.Add(end, midi.NoteOffMsg(1, int(n.pitch), 0))
			}
		}
		tr.Events = append(tr.Events, midi.TrackEvent{Tick: loop * int64(repeats), Meta: midi.MetaEndOfTrack})
		f.Tracks = append(f.Tracks, tr)
	}
	return f
}

// export writes the layers to a file in the folder `recordings`, named after the time (from a goroutine of its own)
func (l *Looper) export() {
	if len(l.layers) == 0 {
		l.say("nothing to export: no layers yet")
		return
	}
	f := l.File(l.Repeats)
	path := filepath.Join(l.dir, "loop-"+time.Now().Format("20060102-150405")+".mid")
	say := l.say
	layers, repeats := len(l.layers), l.Repeats
	go func() {
		err := os.MkdirAll(l.dir, 0o755)
		if err == nil {
			err = f.WriteFile(path)
		}
		if err != nil {
			say("error: can't write %s: %v", path, err)
			return
		}
		say("saved %s (%d layers, the loop %d times)", path, layers, repeats)
	}()
}

// Layers says how many layers there are and whether one is being recorded
func (l *Looper) Layers() (int, bool) { return len(l.layers), l.recording }

// Describe describes the looper's settings
func (l *Looper) Describe() params.Set {
	return params.Set{Name: "looper", Params: []params.Param{
		params.K("Loop", "bars", 1, 16, float64(l.Bars), params.Step(1)),
		params.C("Loop", "quantize", LooperQuantize, l.Quantize),
		params.S("Loop", "record", l.Record),
		params.S("Loop", "play", l.Play),
		params.S("Layers", "undo", l.Undo),
		params.S("Layers", "clear", l.Clear),
		params.S("Export", "export", l.Export, params.Label("export to MIDI")),
		params.K("Export", "repeats", 1, 64, float64(l.Repeats), params.Step(1), params.Label("loop repeated")),
	}}
}

// LooperKind is the looper: notes in, notes out
func LooperKind() Kind {
	return Kind{Name: "looper", Group: "Generators",
		Ins:  []Port{{Name: "notes", Type: Notes}},
		Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			l := NewLooper()
			return &Made{Module: l, Describe: func(string) params.Set { return l.Describe() },
				Notes: l.SetTargets, Late: l.late,
				Attach: func(j *Journal, name string) { l.OnEvent(func(s string) { j.Add(name, s) }) }}, nil
		}}
}
