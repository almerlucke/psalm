// Package midi reads MIDI: the messages (every kind, the channel messages, system exclusive and the
// system common and real time messages), a parser for a MIDI byte stream and inputs that hand the
// events of an audio buffer to the patch, each at the frame it falls on. The keyboard module
// (modules/controls/keyboard) plays synths from an input.
package midi

import "fmt"

// Kind is the kind of a MIDI message
type Kind int

const (
	Invalid Kind = iota

	// channel messages
	NoteOff         // note, velocity
	NoteOn          // note, velocity (a note on with velocity 0 is a NoteOff)
	PolyPressure    // note, pressure (polyphonic aftertouch)
	ControlChange   // controller, value
	ProgramChange   // program
	ChannelPressure // pressure (aftertouch)
	PitchBend       // 14 bit value, 8192 is the center

	// system common
	SysEx        // F0 ... F7
	TimeCode     // MTC quarter frame
	SongPosition // 14 bit position in sixteenths
	SongSelect   // song
	TuneRequest

	// system real time
	Clock // 24 per quarter note
	Start
	Continue
	Stop
	ActiveSensing
	Reset
)

var kindNames = [...]string{"Invalid", "NoteOff", "NoteOn", "PolyPressure", "ControlChange",
	"ProgramChange", "ChannelPressure", "PitchBend", "SysEx", "TimeCode", "SongPosition", "SongSelect",
	"TuneRequest", "Clock", "Start", "Continue", "Stop", "ActiveSensing", "Reset"}

func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("Kind(%d)", int(k))
}

// Controllers with a common meaning
const (
	CCModWheel     = 1
	CCBreath       = 2
	CCFoot         = 4
	CCVolume       = 7
	CCPan          = 10
	CCExpression   = 11
	CCSustain      = 64
	CCPortamento   = 65
	CCSostenuto    = 66
	CCSoft         = 67
	CCAllSoundOff  = 120
	CCResetAll     = 121
	CCAllNotesOff  = 123
	CCOmniOff      = 124
	CCOmniOn       = 125
	CCMonoOn       = 126
	CCPolyOn       = 127
	CCLocalControl = 122
)

// Message is one complete MIDI message, the status byte first
type Message []byte

// Kind returns the kind of the message
func (m Message) Kind() Kind {
	if len(m) == 0 || m[0] < 0x80 {
		return Invalid
	}
	switch m[0] & 0xF0 {
	case 0x80:
		return NoteOff
	case 0x90:
		if len(m) > 2 && m[2] == 0 {
			return NoteOff
		}
		return NoteOn
	case 0xA0:
		return PolyPressure
	case 0xB0:
		return ControlChange
	case 0xC0:
		return ProgramChange
	case 0xD0:
		return ChannelPressure
	case 0xE0:
		return PitchBend
	}
	switch m[0] {
	case 0xF0:
		return SysEx
	case 0xF1:
		return TimeCode
	case 0xF2:
		return SongPosition
	case 0xF3:
		return SongSelect
	case 0xF6:
		return TuneRequest
	case 0xF8:
		return Clock
	case 0xFA:
		return Start
	case 0xFB:
		return Continue
	case 0xFC:
		return Stop
	case 0xFE:
		return ActiveSensing
	case 0xFF:
		return Reset
	}
	return Invalid
}

// IsChannel tells if the message is a channel message (it has a channel)
func (m Message) IsChannel() bool { return len(m) > 0 && m[0] >= 0x80 && m[0] < 0xF0 }

// Channel returns the channel of a channel message (1-16), 0 for a system message
func (m Message) Channel() int {
	if !m.IsChannel() {
		return 0
	}
	return int(m[0]&0x0F) + 1
}

func (m Message) data(i int) int {
	if len(m) > i {
		return int(m[i])
	}
	return 0
}

// Note returns the note of a note or polyphonic pressure message
func (m Message) Note() int { return m.data(1) }

// Velocity returns the velocity of a note message (0-127)
func (m Message) Velocity() int { return m.data(2) }

// Controller returns the controller of a control change
func (m Message) Controller() int { return m.data(1) }

// Value returns the value of a control change (0-127)
func (m Message) Value() int { return m.data(2) }

// Program returns the program of a program change (0-127)
func (m Message) Program() int { return m.data(1) }

// Pressure returns the pressure of a channel or polyphonic pressure message (0-127)
func (m Message) Pressure() int {
	if m.Kind() == PolyPressure {
		return m.data(2)
	}
	return m.data(1)
}

// Bend returns the value of a pitch bend (0-16383, 8192 is the center)
func (m Message) Bend() int { return m.data(1) | m.data(2)<<7 }

// BendAmount returns the pitch bend from -1 to 1 (0 is the center)
func (m Message) BendAmount() float64 {
	b := m.Bend() - 8192
	if b < 0 {
		return float64(b) / 8192
	}
	return float64(b) / 8191
}

// SongPosition returns the position of a song position message in sixteenths
func (m Message) SongPosition() int { return m.data(1) | m.data(2)<<7 }

func (m Message) String() string {
	k := m.Kind()
	switch k {
	case NoteOn, NoteOff:
		return fmt.Sprintf("%v ch %d note %d velocity %d", k, m.Channel(), m.Note(), m.Velocity())
	case PolyPressure:
		return fmt.Sprintf("%v ch %d note %d pressure %d", k, m.Channel(), m.Note(), m.Pressure())
	case ControlChange:
		return fmt.Sprintf("%v ch %d controller %d value %d", k, m.Channel(), m.Controller(), m.Value())
	case ProgramChange:
		return fmt.Sprintf("%v ch %d program %d", k, m.Channel(), m.Program())
	case ChannelPressure:
		return fmt.Sprintf("%v ch %d pressure %d", k, m.Channel(), m.Pressure())
	case PitchBend:
		return fmt.Sprintf("%v ch %d bend %d", k, m.Channel(), m.Bend())
	case SysEx:
		return fmt.Sprintf("%v % X", k, []byte(m))
	}
	return k.String()
}

// constructors, for tests, generated input and output

func channelMsg(status byte, channel int, data ...int) Message {
	m := Message{status | byte(max(0, min(channel-1, 15)))}
	for _, d := range data {
		m = append(m, byte(max(0, min(d, 127))))
	}
	return m
}

// NoteOnMsg returns a note on (channel 1-16)
func NoteOnMsg(channel, note, velocity int) Message {
	return channelMsg(0x90, channel, note, velocity)
}

// NoteOffMsg returns a note off
func NoteOffMsg(channel, note, velocity int) Message {
	return channelMsg(0x80, channel, note, velocity)
}

// ControlChangeMsg returns a control change
func ControlChangeMsg(channel, controller, value int) Message {
	return channelMsg(0xB0, channel, controller, value)
}

// ProgramChangeMsg returns a program change
func ProgramChangeMsg(channel, program int) Message {
	return channelMsg(0xC0, channel, program)
}

// ChannelPressureMsg returns a channel pressure (aftertouch)
func ChannelPressureMsg(channel, pressure int) Message {
	return channelMsg(0xD0, channel, pressure)
}

// PolyPressureMsg returns a polyphonic pressure
func PolyPressureMsg(channel, note, pressure int) Message {
	return channelMsg(0xA0, channel, note, pressure)
}

// PitchBendMsg returns a pitch bend (0-16383, 8192 is the center)
func PitchBendMsg(channel, bend int) Message {
	bend = max(0, min(bend, 16383))
	return channelMsg(0xE0, channel, bend&0x7F, bend>>7)
}

// Real time messages
var (
	ClockMsg    = Message{0xF8}
	StartMsg    = Message{0xFA}
	ContinueMsg = Message{0xFB}
	StopMsg     = Message{0xFC}
)
