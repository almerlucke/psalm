// Package keyboard plays synths from MIDI: a control module that reads a midi.Input every buffer
// and turns its events into the messages of psalm, each at the frame it falls on.
package keyboard

import (
	"fmt"
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
)

// Curve is a velocity curve: the velocity (0-1) is raised to it. Every keyboard responds
// differently and every player plays differently: below 1 a soft touch already plays loud, above 1
// it takes more force to get there
type Curve float64

const (
	Soft   Curve = 0.6
	Linear Curve = 1
	Hard   Curve = 1.6
)

// DefaultControls are the controllers sent when Config.Controls is nil: the pedals, as the piano
// takes them
var DefaultControls = map[int]string{
	midi.CCSustain:   "sustain",
	midi.CCSostenuto: "sostenuto",
	midi.CCSoft:      "soft",
}

// Config sets how the keyboard plays
type Config struct {
	Channel   int // the channel listened to (1-16), 0 is all of them (omni)
	Transpose int // semitones

	// Velocity: the curve (0 is Linear), or a map of its own (0-1 to 0-1), or one fixed velocity
	// for every note (above 0)
	Curve         Curve
	VelocityMap   func(v float64) float64
	FixedVelocity float64

	// Dynamics is how much the velocity sets the amplitude of a note: 0 it is always 1 (the synth
	// takes the "velocity" of the note, like the piano and dx), 1 the amplitude is the velocity
	Dynamics float64

	// Controls maps controllers to messages for every voice ({name: value}, the value 0-1); nil is
	// DefaultControls, an empty map sends none
	Controls map[int]string

	// Routes send controllers (and the bend, RouteBend, and the aftertouch, RoutePressure) to any
	// settings, scaled to their ranges: {1: {{Name: "cutoff", Min: 300, Max: 8000, Exp: true}}}
	// sweeps the cutoff with the mod wheel. A routed controller is sent as its routes, not by
	// Controls (nor PitchBend, Pressure)
	Routes map[int][]Route

	// HoldNotes makes the keyboard keep the notes released while the sustain pedal is down (for a
	// synth without a sustain pedal of its own) instead of sending the pedal
	HoldNotes bool

	// The names of the messages for every voice the pitch bend (-1 to 1) and the channel pressure
	// (0-1) are sent as, "" sends none
	PitchBend string
	Pressure  string

	// FollowClock sets the tempo of the patch from a MIDI clock
	FollowClock bool
}

// Keyboard plays synths (any receiver of trigger messages: a poly synth, the piano) from a MIDI
// input. Notes become trigger messages, "noteOn" and "noteOff", the note in "pitch" (transposed)
// and its velocity in "velocity" (after the curve). Controllers, the pitch bend and the pressure
// become messages for every voice as configured; every event, of every kind, also goes to the
// handlers (OnEvent, OnControl, OnProgram) for anything else.
//
// The input is read on the first control tick of every buffer, so a tick has to be shorter than a
// buffer (it is by default)
type Keyboard struct {
	*psalm.Module
	input    midi.Input
	targets  []psalm.Receiver
	cfg      Config
	events   []midi.Event
	buffer   int64
	read     bool
	ids      [16][128]string
	held     [16][128]bool // keys down (as notes sent)
	sounding [16][128]bool // notes on (held, or kept by the pedal)
	pedal    [16]bool      // HoldNotes: the sustain pedal of a channel

	onEvent   []func(midi.Event, psalm.CtrlTick)
	onControl map[int][]func(float64, psalm.CtrlTick)
	onProgram []func(int, psalm.CtrlTick)

	// the clock
	clockAt  int64
	clocks   [24]int64
	clockN   int
	clockSum int64
}

// New returns a keyboard that plays the targets from an input
func New(input midi.Input, cfg Config, targets ...psalm.Receiver) *Keyboard {
	k := &Keyboard{
		Module:    psalm.NewControl(),
		input:     input,
		targets:   targets,
		cfg:       cfg,
		onControl: map[int][]func(float64, psalm.CtrlTick){},
		clockAt:   -1,
	}
	for ch := range k.ids {
		for n := range k.ids[ch] {
			k.ids[ch][n] = fmt.Sprintf("midi-%d-%d", ch+1, n)
		}
	}
	k.SetSelf(k)
	return k
}

// AddTarget adds a synth to play
func (k *Keyboard) AddTarget(r psalm.Receiver) *Keyboard {
	k.targets = append(k.targets, r)
	return k
}

// SetTargets changes the synths played (release the notes first, AllNotesOff)
func (k *Keyboard) SetTargets(targets ...psalm.Receiver) *Keyboard {
	k.targets = targets
	return k
}

// OnEvent calls f for every event (of every kind, the channel messages of the channel listened
// to), before the keyboard handles it
func (k *Keyboard) OnEvent(f func(e midi.Event, tick psalm.CtrlTick)) *Keyboard {
	k.onEvent = append(k.onEvent, f)
	return k
}

// OnControl calls f for a controller, with its value (0-1)
func (k *Keyboard) OnControl(controller int, f func(value float64, tick psalm.CtrlTick)) *Keyboard {
	k.onControl[controller] = append(k.onControl[controller], f)
	return k
}

// OnProgram calls f for a program change
func (k *Keyboard) OnProgram(f func(program int, tick psalm.CtrlTick)) *Keyboard {
	k.onProgram = append(k.onProgram, f)
	return k
}

// Settings returns the settings of the keyboard
func (k *Keyboard) Settings() Config { return k.cfg }

// SetConfig changes the settings
func (k *Keyboard) SetConfig(cfg Config) { k.cfg = cfg }

// SetRoutes sets the routes of a source (a controller, RouteBend or RoutePressure); none removes
// them, the source goes back to Controls (PitchBend, Pressure)
func (k *Keyboard) SetRoutes(source int, routes ...Route) *Keyboard {
	if len(routes) == 0 {
		delete(k.cfg.Routes, source)
		return k
	}
	if k.cfg.Routes == nil {
		k.cfg.Routes = map[int][]Route{}
	}
	k.cfg.Routes[source] = routes
	return k
}

// ReceiveMessage changes the settings: "channel", "transpose", "curve", "fixedVelocity",
// "dynamics", and "route" (a route as text, see ParseRoute: it replaces the routes of its source)
// and "unroute" (a source: "1", "bend", "pressure")
func (k *Keyboard) ReceiveMessage(msg psalm.Message) {
	msg.IfStrVal("route", func(text string) {
		if source, r, err := ParseRoute(text); err == nil {
			k.SetRoutes(source, r)
		}
	})
	msg.IfStrVal("unroute", func(text string) {
		if source, _, err := ParseRoute(text + ":x:0:1"); err == nil {
			k.SetRoutes(source)
		}
	})
	msg.IfIntVal("channel", func(v int) { k.cfg.Channel = v })
	msg.IfIntVal("transpose", func(v int) { k.cfg.Transpose = v })
	msg.IfFltVal("curve", func(v float64) { k.cfg.Curve = Curve(v) })
	msg.IfFltVal("fixedVelocity", func(v float64) { k.cfg.FixedVelocity = v })
	msg.IfFltVal("dynamics", func(v float64) { k.cfg.Dynamics = v })
}

// Receive takes a message (settings)
func (k *Keyboard) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		k.ReceiveMessage(msg)
	}
}

// Velocity returns the velocity of a note (a MIDI velocity, 1-127) after the curve, 0-1
func (k *Keyboard) Velocity(velocity int) float64 {
	if k.cfg.FixedVelocity > 0 {
		return math.Min(k.cfg.FixedVelocity, 1)
	}
	v := math.Max(0, math.Min(float64(velocity)/127, 1))
	if k.cfg.VelocityMap != nil {
		return math.Max(0, math.Min(k.cfg.VelocityMap(v), 1))
	}
	c := float64(k.cfg.Curve)
	if c <= 0 {
		c = 1
	}
	return math.Pow(v, c)
}

// CtrlTick reads the input on the first tick of a buffer and handles its events
func (k *Keyboard) CtrlTick(tick psalm.CtrlTick) {
	buffer := tick.SampleTimestamp - int64(tick.SampleOffset)
	if k.read && buffer == k.buffer {
		return
	}
	k.read, k.buffer = true, buffer
	cfg := k.Config()
	k.events = k.input.Read(k.events[:0], cfg.BufferSize, cfg.SampleRate)
	for _, e := range k.events {
		t := tick
		t.SampleOffset = e.Offset
		t.SampleTimestamp = buffer + int64(e.Offset)
		k.handle(e, t)
	}
	clear(k.events)
}

func (k *Keyboard) send(msg psalm.Message) {
	for _, t := range k.targets {
		t.ReceiveMessage(msg)
	}
}

func (k *Keyboard) handle(e midi.Event, tick psalm.CtrlTick) {
	m := e.Message
	if m.IsChannel() && k.cfg.Channel > 0 && m.Channel() != k.cfg.Channel {
		return
	}
	for _, f := range k.onEvent {
		f(e, tick)
	}
	ch := max(0, m.Channel()-1)
	switch m.Kind() {
	case midi.NoteOn:
		k.noteOn(ch, m.Note(), m.Velocity(), tick)
	case midi.NoteOff:
		k.noteOff(ch, m.Note(), tick)
	case midi.ControlChange:
		k.control(ch, m.Controller(), m.Value(), tick)
	case midi.ProgramChange:
		for _, f := range k.onProgram {
			f(m.Program(), tick)
		}
	case midi.PitchBend:
		if k.routed(RouteBend, (m.BendAmount()+1)/2) {
			break
		}
		if k.cfg.PitchBend != "" {
			k.send(psalm.Message{k.cfg.PitchBend: m.BendAmount()})
		}
	case midi.ChannelPressure:
		if k.routed(RoutePressure, float64(m.Pressure())/127) {
			break
		}
		if k.cfg.Pressure != "" {
			k.send(psalm.Message{k.cfg.Pressure: float64(m.Pressure()) / 127})
		}
	case midi.Clock:
		k.clock(tick)
	case midi.Start, midi.Stop:
		k.clockAt, k.clockN, k.clockSum = -1, 0, 0
	case midi.Reset:
		k.AllNotesOff(tick)
	}
}

func (k *Keyboard) noteOn(ch, note, velocity int, tick psalm.CtrlTick) {
	if k.sounding[ch][note] {
		// struck again: the note sounding stops, the piano strikes its key again
		k.release(ch, note, tick)
	}
	v := k.Velocity(velocity)
	d := math.Max(0, math.Min(k.cfg.Dynamics, 1))
	k.held[ch][note], k.sounding[ch][note] = true, true
	k.send(psalm.Message{
		"command":   "trigger",
		"noteOn":    k.ids[ch][note],
		"tick":      tick,
		"amplitude": 1 - d + d*v,
		"message":   psalm.Message{"pitch": float64(note + k.cfg.Transpose), "velocity": v},
	})
}

func (k *Keyboard) noteOff(ch, note int, tick psalm.CtrlTick) {
	if !k.held[ch][note] {
		return
	}
	k.held[ch][note] = false
	if k.cfg.HoldNotes && k.pedal[ch] {
		return // the pedal keeps it
	}
	k.release(ch, note, tick)
}

func (k *Keyboard) release(ch, note int, tick psalm.CtrlTick) {
	k.sounding[ch][note] = false
	k.send(psalm.Message{"command": "trigger", "noteOff": k.ids[ch][note], "tick": tick})
}

// AllNotesOff releases every note
func (k *Keyboard) AllNotesOff(tick psalm.CtrlTick) {
	for ch := range k.sounding {
		k.pedal[ch] = false
		for note, on := range k.sounding[ch] {
			k.held[ch][note] = false
			if on {
				k.release(ch, note, tick)
			}
		}
	}
}

func (k *Keyboard) control(ch, controller, value int, tick psalm.CtrlTick) {
	v := float64(value) / 127
	for _, f := range k.onControl[controller] {
		f(v, tick)
	}
	switch controller {
	case midi.CCAllNotesOff, midi.CCAllSoundOff:
		for note, on := range k.sounding[ch] {
			k.held[ch][note] = false
			if on {
				k.release(ch, note, tick)
			}
		}
		k.pedal[ch] = false
		return
	case midi.CCResetAll:
		// the controllers back to rest: the pedals up, no bend, no pressure, the routes at rest
		for c := range k.controls() {
			k.control(ch, c, 0, tick)
		}
		for c := range k.cfg.Routes {
			switch c {
			case RouteBend:
				k.routed(c, 0.5)
			case RoutePressure:
				k.routed(c, 0)
			default:
				if _, ok := k.controls()[c]; !ok {
					k.control(ch, c, 0, tick)
				}
			}
		}
		if _, ok := k.controls()[midi.CCSustain]; !ok && k.cfg.HoldNotes {
			k.control(ch, midi.CCSustain, 0, tick)
		}
		if k.cfg.PitchBend != "" {
			k.send(psalm.Message{k.cfg.PitchBend: 0.0})
		}
		if k.cfg.Pressure != "" {
			k.send(psalm.Message{k.cfg.Pressure: 0.0})
		}
		return
	}
	if controller == midi.CCSustain && k.cfg.HoldNotes {
		down := value >= 64
		if k.pedal[ch] && !down {
			for note, on := range k.sounding[ch] {
				if on && !k.held[ch][note] {
					k.release(ch, note, tick)
				}
			}
		}
		k.pedal[ch] = down
		return
	}
	if k.routed(controller, v) {
		return
	}
	if name, ok := k.controls()[controller]; ok {
		k.send(psalm.Message{name: v})
	}
}

func (k *Keyboard) controls() map[int]string {
	if k.cfg.Controls == nil {
		return DefaultControls
	}
	return k.cfg.Controls
}

// clock follows a MIDI clock: the tempo from the average of the last beat (24 clocks)
func (k *Keyboard) clock(tick psalm.CtrlTick) {
	at := tick.SampleTimestamp
	if k.clockAt >= 0 {
		d := at - k.clockAt
		i := k.clockN % len(k.clocks)
		if k.clockN >= len(k.clocks) {
			k.clockSum -= k.clocks[i]
		}
		k.clocks[i] = d
		k.clockSum += d
		k.clockN++
	}
	k.clockAt = at
	if !k.cfg.FollowClock || k.clockN < len(k.clocks) || k.clockSum <= 0 {
		return
	}
	bpm := 60 * k.Config().SampleRate / float64(k.clockSum)
	bpm = math.Round(bpm*10) / 10
	if env := k.Env(); math.Abs(env.BPM()-bpm) >= 0.1 {
		env.SetBPM(bpm)
	}
}

// Tempo returns the tempo of the MIDI clock (0 before a beat of it came in)
func (k *Keyboard) Tempo() float64 {
	if k.clockN < len(k.clocks) || k.clockSum <= 0 {
		return 0
	}
	return 60 * k.Config().SampleRate / float64(k.clockSum)
}
