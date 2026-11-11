package patcher

import (
	"slices"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// ArpModes are the orders an arpeggiator plays the notes held in
var ArpModes = []string{"up", "down", "up & down", "down & up", "as played", "random", "chord"}

// arpNote is a note held: its pitch, velocity and when it was pressed
type arpNote struct {
	id       string
	pitch    float64
	velocity float64
	until    int64 // for a note of a duration: the sample it ends (0: held until its key is let go)
}

// Arp is an arpeggiator: the notes it gets (from a keyboard, a note editor, another module) are
// held, and on every trigger of its clock input it plays the next of them, to the synths its notes
// output goes to. Settings: "mode" (an index of ArpModes), "octaves" (1-4: it climbs that many),
// "gate" (the part of a step a note lasts), "hold" (the notes stay after the keys are let go, until
// new ones are played), "transpose" (semitones)
type Arp struct {
	*psalm.Module
	Mode, Octaves int
	Gate          float64
	Hold          bool
	Transpose     float64

	held    []arpNote
	latched bool // hold: the held notes are to be replaced by the next ones played
	pos     int
	pending []Event
	targets []psalm.Receiver
	now     int64 // the latest sample timestamp seen
}

// NewArp returns an arpeggiator going up over one octave, a note taking 0.8 of a step
func NewArp() *Arp {
	a := &Arp{Module: psalm.NewModule(nil, nil), Octaves: 1, Gate: 0.8}
	a.SetSelf(a)
	return a
}

// SetTargets sets the synths it plays (on the audio thread)
func (a *Arp) SetTargets(targets []psalm.Receiver) { a.targets = targets }

// ReceiveMessage takes the notes of the trigger messages of a keyboard or a note source (held
// notes), passes the others (the bend, the wheel) on to its synths, and takes its settings from a
// message without a command
func (a *Arp) ReceiveMessage(msg psalm.Message) {
	cmd, ok := msg.StrVal("command")
	if !ok {
		msg.IfIntVal("mode", func(m int) { a.Mode = max(0, min(m, len(ArpModes)-1)) })
		msg.IfIntVal("octaves", func(o int) { a.Octaves = max(1, min(o, 4)) })
		msg.IfFltVal("gate", func(f float64) { a.Gate = max(0.05, min(f, 1)) })
		msg.IfFltVal("hold", func(f float64) {
			a.Hold = f > 0.5
			if !a.Hold {
				a.latched = false
			}
		})
		msg.IfFltVal("transpose", func(f float64) { a.Transpose = f })
		return
	}
	if cmd != "trigger" {
		for _, t := range a.targets {
			t.ReceiveMessage(msg)
		}
		return
	}
	if id, ok := msg.StrVal("noteOff"); ok {
		a.release(id)
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
	n := arpNote{pitch: pitch, velocity: vel}
	if id, ok := msg.StrVal("noteOn"); ok {
		n.id = id
	} else if d, ok := msg.FltVal("duration"); ok {
		tick := msg.MustTickVal("tick")
		n.until = tick.SampleTimestamp + int64(d*a.Config().SampleRate)
	}
	if a.latched && a.Hold { // new keys after all were let go: the old notes go
		a.held, a.latched = a.held[:0], false
	}
	if len(a.held) == 0 {
		a.pos = 0
	}
	a.held = append(a.held, n)
}

// release lets go of the note of a key (with hold, the notes stay until the next key)
func (a *Arp) release(id string) {
	i := slices.IndexFunc(a.held, func(n arpNote) bool { return n.id == id })
	if i < 0 {
		return
	}
	if a.Hold {
		held := 0
		for _, n := range a.held {
			if n.id != "" {
				held++
			}
		}
		a.held[i].id = "" // no longer a key; stays as a note
		if held == 1 {
			a.latched = true
		}
		return
	}
	a.held = slices.Delete(a.held, i, i+1)
}

// Receive takes a trigger (input 1: the clock, played at the end of the tick) or a message
func (a *Arp) Receive(v any, index int) {
	if m, ok := psalm.MsgVal(v); ok {
		a.ReceiveMessage(m)
		return
	}
	if ev, ok := v.(Event); ok {
		a.pending = append(a.pending, ev)
	}
}

// pool is the notes to step through now: the held ones over the octaves, in the order of the mode
func (a *Arp) pool() []arpNote {
	var pool []arpNote
	for o := range a.Octaves {
		for _, n := range a.held {
			n.pitch += 12 * float64(o)
			pool = append(pool, n)
		}
	}
	if a.Mode != 4 && a.Mode != 5 {
		slices.SortStableFunc(pool, func(x, y arpNote) int {
			switch {
			case x.pitch < y.pitch:
				return -1
			case x.pitch > y.pitch:
				return 1
			}
			return 0
		})
	}
	return pool
}

// next returns the notes to play at a step
func (a *Arp) next(pool []arpNote) []arpNote {
	n := len(pool)
	switch a.Mode {
	case 5:
		return pool[a.Rand().IntN(n):][:1]
	case 6:
		return pool
	}
	var i int
	switch a.Mode {
	case 1:
		i = n - 1 - a.pos%n
	case 2, 3:
		cycle := max(1, 2*n-2)
		i = a.pos % cycle
		if i >= n {
			i = cycle - i
		}
		if a.Mode == 3 {
			i = n - 1 - i
		}
	default:
		i = a.pos % n
	}
	return pool[i : i+1]
}

// late plays the triggers of the tick
func (a *Arp) late(tick psalm.CtrlTick) {
	a.now = tick.SampleTimestamp
	// notes of a duration that are over
	a.held = slices.DeleteFunc(a.held, func(n arpNote) bool { return n.until > 0 && n.until <= a.now })
	if len(a.pending) == 0 {
		return
	}
	for _, ev := range a.pending {
		if len(a.held) == 0 {
			continue
		}
		pool := a.pool()
		play := a.next(pool)
		seconds := max(0.005, a.Gate*ev.Seconds)
		for _, n := range play {
			msg := psalm.Message{"command": "trigger", "tick": ev.Tick, "duration": seconds, "amplitude": n.velocity,
				"message": psalm.Message{"pitch": n.pitch + a.Transpose, "velocity": n.velocity}}
			for _, t := range a.targets {
				t.ReceiveMessage(msg)
			}
		}
		a.pos++
	}
	a.pending = a.pending[:0]
}

// Describe describes the arpeggiator's settings
func (a *Arp) Describe() params.Set {
	return params.Set{Name: "arp", Params: []params.Param{
		params.C("Arp", "mode", ArpModes, a.Mode),
		params.K("Arp", "octaves", 1, 4, float64(a.Octaves), params.Step(1)),
		params.K("Arp", "gate", 0.05, 1, a.Gate, params.Label("gate (of a step)")),
		params.S("Arp", "hold", a.Hold),
		params.K("Arp", "transpose", -36, 36, a.Transpose, params.Step(1), params.Unit("st")),
	}}
}

// ArpKind is the arpeggiator: notes in, a clock in, notes out
func ArpKind() Kind {
	return Kind{Name: "arp", Group: "Generators",
		Ins:  []Port{{Name: "notes", Type: Notes}, {Name: "clock", Type: Trigger, Index: 1}},
		Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			a := NewArp()
			return &Made{Module: a, Describe: func(string) params.Set { return a.Describe() },
				Notes: a.SetTargets, Late: a.late}, nil
		}}
}
