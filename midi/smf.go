package midi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"
)

// Meta event types of a MIDI file
const (
	MetaSequence      = 0x00
	MetaText          = 0x01
	MetaCopyright     = 0x02
	MetaTrackName     = 0x03
	MetaInstrument    = 0x04
	MetaLyric         = 0x05
	MetaMarker        = 0x06
	MetaCuePoint      = 0x07
	MetaChannelPrefix = 0x20
	MetaPort          = 0x21
	MetaEndOfTrack    = 0x2F
	MetaTempo         = 0x51
	MetaSMPTEOffset   = 0x54
	MetaTimeSignature = 0x58
	MetaKeySignature  = 0x59
)

// File is a Standard MIDI File: tracks of events at ticks. Its time is in ticks of a quarter note
// (Division ticks to a quarter, the tempo events say how long a quarter lasts) or, rarely, in
// ticks of SMPTE frames (FPS frames a second, TicksPerFrame ticks to a frame, Division 0)
type File struct {
	Format        int // 0 one track, 1 tracks played together, 2 tracks each a sequence of its own
	Division      int // ticks per quarter note
	FPS           int // SMPTE time: frames per second (24, 25, 29 for 29.97, 30)
	TicksPerFrame int
	Tracks        []Track
}

// Track is the events of a track, in order of their ticks
type Track struct {
	Events []TrackEvent
}

// TrackEvent is a message at a tick: a channel message or a system exclusive message (Message),
// or a meta event (no Message, Meta its type, Data what it holds)
type TrackEvent struct {
	Tick    int64 // from the start of the file
	Message Message
	Meta    byte
	Data    []byte
}

// IsMeta says whether the event is a meta event
func (e TrackEvent) IsMeta() bool { return len(e.Message) == 0 }

// Tempo returns the tempo of a tempo event in quarter notes a minute
func (e TrackEvent) Tempo() (bpm float64, ok bool) {
	if !e.IsMeta() || e.Meta != MetaTempo || len(e.Data) < 3 {
		return 0, false
	}
	us := int(e.Data[0])<<16 | int(e.Data[1])<<8 | int(e.Data[2])
	if us == 0 {
		return 0, false
	}
	return 60e6 / float64(us), true
}

// TimeSignature returns the time signature of a time signature event (6/8: 6, 8)
func (e TrackEvent) TimeSignature() (beats, unit int, ok bool) {
	if !e.IsMeta() || e.Meta != MetaTimeSignature || len(e.Data) < 2 {
		return 0, 0, false
	}
	return int(e.Data[0]), 1 << min(e.Data[1], 16), true
}

// Text returns the text of a text event (a track name, a lyric, a marker...), without the zero
// bytes some programs end it with
func (e TrackEvent) Text() (string, bool) {
	if !e.IsMeta() || e.Meta < MetaText || e.Meta > 0x0F {
		return "", false
	}
	return strings.TrimRight(string(e.Data), "\x00"), true
}

// Add adds a message at a tick (copied), after the events at that tick
func (t *Track) Add(tick int64, msg Message) {
	t.insert(TrackEvent{Tick: tick, Message: append(Message(nil), msg...)})
}

// AddMeta adds a meta event at a tick
func (t *Track) AddMeta(tick int64, meta byte, data []byte) {
	t.insert(TrackEvent{Tick: tick, Meta: meta, Data: append([]byte{}, data...)})
}

func (t *Track) insert(e TrackEvent) {
	i := len(t.Events)
	for i > 0 && t.Events[i-1].Tick > e.Tick {
		i--
	}
	t.Events = slices.Insert(t.Events, i, e)
}

// Name returns the name of the track (its first track name event)
func (t Track) Name() string {
	for _, e := range t.Events {
		if e.IsMeta() && e.Meta == MetaTrackName {
			name, _ := e.Text()
			return name
		}
	}
	return ""
}

// TempoData is the data of a tempo event, a tempo in quarter notes a minute
func TempoData(bpm float64) []byte {
	us := int(math.Round(60e6 / bpm))
	us = max(1, min(us, 1<<24-1))
	return []byte{byte(us >> 16), byte(us >> 8), byte(us)}
}

// TimeSignatureData is the data of a time signature event (6/8: 6, 8), a click every quarter
func TimeSignatureData(beats, unit int) []byte {
	return []byte{byte(beats), byte(bits(unit)), 24, 8}
}

func bits(n int) int {
	b := 0
	for n > 1 {
		n >>= 1
		b++
	}
	return b
}

// ReadFile reads a MIDI file (a .mid, or an .rmi: a MIDI file in a RIFF)
func ReadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := ParseFile(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

// ErrNotMIDIFile is returned for data that isn't a MIDI file
var ErrNotMIDIFile = errors.New("not a MIDI file")

// ParseFile reads a MIDI file from its bytes. It takes files as they come: a track cut short ends
// where the file does, a missing end of track is fine, chunks of other kinds are skipped
func ParseFile(data []byte) (*File, error) {
	// an RMID: the MIDI file is the data chunk of a RIFF
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "RMID" {
		for c := data[12:]; len(c) >= 8; {
			n := int(binary.LittleEndian.Uint32(c[4:]))
			if string(c[:4]) == "data" {
				data = c[8:min(len(c), 8+n)]
				break
			}
			c = c[min(len(c), 8+n+n&1):]
		}
	}
	if len(data) < 14 || string(data[:4]) != "MThd" {
		return nil, ErrNotMIDIFile
	}
	n := int(binary.BigEndian.Uint32(data[4:]))
	if n < 6 || 8+n > len(data) {
		return nil, fmt.Errorf("%w: the header is cut short", ErrNotMIDIFile)
	}
	f := &File{Format: int(binary.BigEndian.Uint16(data[8:]))}
	tracks := int(binary.BigEndian.Uint16(data[10:]))
	division := binary.BigEndian.Uint16(data[12:])
	if division&0x8000 != 0 {
		f.FPS = int(-int8(division >> 8))
		f.TicksPerFrame = int(division & 0xFF)
		if f.FPS <= 0 || f.TicksPerFrame == 0 {
			return nil, fmt.Errorf("SMPTE time of %d frames, %d ticks a frame", f.FPS, f.TicksPerFrame)
		}
	} else {
		f.Division = int(division)
		if f.Division == 0 {
			return nil, errors.New("0 ticks a quarter note")
		}
	}
	rest := data[8+n:]
	for len(rest) >= 8 && len(f.Tracks) < tracks {
		id, n := string(rest[:4]), int(binary.BigEndian.Uint32(rest[4:]))
		chunk := rest[8:min(len(rest), 8+n)]
		rest = rest[8+len(chunk):]
		if id != "MTrk" {
			continue
		}
		t, err := parseTrack(chunk)
		if err != nil {
			return nil, fmt.Errorf("track %d: %w", len(f.Tracks)+1, err)
		}
		f.Tracks = append(f.Tracks, t)
	}
	return f, nil
}

// errShort is a track cut off in the middle of an event
var errShort = errors.New("cut short")

func parseTrack(data []byte) (Track, error) {
	var t Track
	var tick int64
	var running byte
	i := 0
	varLen := func() (int, error) {
		v := 0
		for range 4 {
			if i >= len(data) {
				return 0, errShort
			}
			b := data[i]
			i++
			v = v<<7 | int(b&0x7F)
			if b < 0x80 {
				return v, nil
			}
		}
		return 0, errors.New("a number longer than four bytes")
	}
	for i < len(data) {
		delta, err := varLen()
		if err != nil {
			break // a delta cut off at the end: the track ends
		}
		tick += int64(delta)
		if i >= len(data) {
			break
		}
		status := data[i]
		switch {
		case status == 0xFF:
			// a meta event
			if i+1 >= len(data) {
				return t, nil
			}
			meta := data[i+1]
			i += 2
			n, err := varLen()
			if err != nil || i+n > len(data) {
				return t, nil
			}
			body := data[i : i+n]
			i += n
			if meta == MetaEndOfTrack {
				t.Events = append(t.Events, TrackEvent{Tick: tick, Meta: meta})
				return t, nil
			}
			t.Events = append(t.Events, TrackEvent{Tick: tick, Meta: meta, Data: append([]byte{}, body...)})
			running = 0
		case status == 0xF0 || status == 0xF7:
			// system exclusive (F0, its length, the bytes after F0) or an escape (F7, its length,
			// any bytes): the escapes are left out
			i++
			n, err := varLen()
			if err != nil || i+n > len(data) {
				return t, nil
			}
			if status == 0xF0 {
				t.Events = append(t.Events, TrackEvent{Tick: tick, Message: append(Message{0xF0}, data[i:i+n]...)})
			}
			i += n
			running = 0
		default:
			if status >= 0x80 {
				running = status
				i++
			} else if running == 0 {
				return t, fmt.Errorf("data byte %#x without a status at tick %d", status, tick)
			}
			if running >= 0xF0 {
				return t, fmt.Errorf("status %#x in a track", running)
			}
			need := dataBytes(running)
			if i+need > len(data) {
				return t, nil
			}
			msg := make(Message, 1+need)
			msg[0] = running
			copy(msg[1:], data[i:i+need])
			i += need
			t.Events = append(t.Events, TrackEvent{Tick: tick, Message: msg})
		}
	}
	return t, nil
}

// Write writes the file as a Standard MIDI File (every track ends with an end of track)
func (f *File) Write(w io.Writer) error {
	var b bytes.Buffer
	b.WriteString("MThd")
	b.Write([]byte{0, 0, 0, 6})
	format := f.Format
	if len(f.Tracks) > 1 && format == 0 {
		format = 1
	}
	binary.Write(&b, binary.BigEndian, uint16(format))
	binary.Write(&b, binary.BigEndian, uint16(len(f.Tracks)))
	if f.Division > 0 {
		binary.Write(&b, binary.BigEndian, uint16(f.Division))
	} else if f.FPS > 0 && f.TicksPerFrame > 0 {
		b.Write([]byte{byte(-int8(f.FPS)), byte(f.TicksPerFrame)})
	} else {
		return errors.New("no division: set Division (ticks a quarter note) or FPS and TicksPerFrame")
	}
	for _, t := range f.Tracks {
		events := slices.Clone(t.Events)
		slices.SortStableFunc(events, func(a, b TrackEvent) int { return int(a.Tick - b.Tick) })
		var body []byte
		var tick int64
		ended := false
		for _, e := range events {
			if ended {
				break
			}
			if !e.IsMeta() && e.Message[0] != 0xF0 && !e.Message.IsChannel() {
				continue // real time and system common messages have no place in a file
			}
			body = putVarLen(body, int(max(0, e.Tick-tick)))
			tick = max(tick, e.Tick)
			switch {
			case e.IsMeta():
				body = append(body, 0xFF, e.Meta)
				body = putVarLen(body, len(e.Data))
				body = append(body, e.Data...)
				ended = e.Meta == MetaEndOfTrack
			case e.Message[0] == 0xF0:
				body = append(body, 0xF0)
				body = putVarLen(body, len(e.Message)-1)
				body = append(body, e.Message[1:]...)
			default:
				body = append(body, e.Message...)
			}
		}
		if !ended {
			body = append(body, 0, 0xFF, MetaEndOfTrack, 0)
		}
		b.WriteString("MTrk")
		binary.Write(&b, binary.BigEndian, uint32(len(body)))
		b.Write(body)
	}
	_, err := w.Write(b.Bytes())
	return err
}

// WriteFile writes the file to a path
func (f *File) WriteFile(path string) error {
	var b bytes.Buffer
	if err := f.Write(&b); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

func putVarLen(b []byte, v int) []byte {
	var tmp [4]byte
	n := 0
	for {
		tmp[n] = byte(v & 0x7F)
		n++
		v >>= 7
		if v == 0 || n == 4 {
			break
		}
	}
	for j := n - 1; j >= 0; j-- {
		if j > 0 {
			b = append(b, tmp[j]|0x80)
		} else {
			b = append(b, tmp[j])
		}
	}
	return b
}

// tempoChange is where the tempo changes: the tick, the seconds to it, the seconds of a tick from it
type tempoChange struct {
	tick    int64
	seconds float64
	perTick float64
}

// tempoMap returns the tempo changes of the file (of every track: in a file of format 1 they are
// in the first, but not always), from 120 quarter notes a minute until the first
func (f *File) tempoMap() []tempoChange {
	if f.Division <= 0 {
		perTick := 1 / float64(max(1, f.FPS)*max(1, f.TicksPerFrame))
		if f.FPS == 29 {
			perTick = 1 / (29.97 * float64(f.TicksPerFrame))
		}
		return []tempoChange{{perTick: perTick}}
	}
	type change struct {
		tick int64
		bpm  float64
	}
	var changes []change
	for i, t := range f.Tracks {
		if f.Format == 2 && i > 0 {
			break
		}
		for _, e := range t.Events {
			if bpm, ok := e.Tempo(); ok {
				changes = append(changes, change{e.Tick, bpm})
			}
		}
	}
	slices.SortStableFunc(changes, func(a, b change) int { return int(a.tick - b.tick) })
	perQuarter := func(bpm float64) float64 { return 60 / bpm / float64(f.Division) }
	m := []tempoChange{{perTick: perQuarter(120)}}
	for _, c := range changes {
		last := &m[len(m)-1]
		if c.tick == last.tick {
			last.perTick = perQuarter(c.bpm)
			continue
		}
		m = append(m, tempoChange{
			tick:    c.tick,
			seconds: last.seconds + float64(c.tick-last.tick)*last.perTick,
			perTick: perQuarter(c.bpm),
		})
	}
	return m
}

// secondsAt returns the seconds to a tick in a tempo map
func secondsAt(m []tempoChange, tick int64) float64 {
	i, _ := slices.BinarySearchFunc(m, tick, func(c tempoChange, t int64) int {
		if c.tick <= t {
			return -1
		}
		return 1
	})
	c := m[max(0, i-1)]
	return c.seconds + float64(tick-c.tick)*c.perTick
}

// Seconds returns the time of a tick from the start, following the tempo
func (f *File) Seconds(tick int64) float64 { return secondsAt(f.tempoMap(), tick) }

// Ticks returns the length of the file in ticks (to its last event)
func (f *File) Ticks() int64 {
	var end int64
	for _, t := range f.Tracks {
		if n := len(t.Events); n > 0 {
			end = max(end, t.Events[n-1].Tick)
		}
	}
	return end
}

// Duration returns the length of the file in seconds
func (f *File) Duration() float64 { return f.Seconds(f.Ticks()) }

// Channels returns the channels the file plays on (1-16), in order
func (f *File) Channels() []int {
	var used [16]bool
	for _, t := range f.Tracks {
		for _, e := range t.Events {
			if e.Message.IsChannel() {
				used[e.Message.Channel()-1] = true
			}
		}
	}
	var chs []int
	for ch, u := range used {
		if u {
			chs = append(chs, ch+1)
		}
	}
	return chs
}

// Programs returns the programs a channel plays (1-16): the ones its program changes choose, in
// the order they come, the one before the first program change (0, a piano) first when a note
// plays before it
func (f *File) Programs(channel int) []int {
	type at struct {
		tick    int64
		program int // -1 a note
	}
	var all []at
	for _, t := range f.Tracks {
		for _, e := range t.Events {
			if !e.Message.IsChannel() || e.Message.Channel() != channel {
				continue
			}
			switch e.Message.Kind() {
			case ProgramChange:
				all = append(all, at{e.Tick, e.Message.Program()})
			case NoteOn:
				all = append(all, at{e.Tick, -1})
			}
		}
	}
	slices.SortStableFunc(all, func(a, b at) int {
		if a.tick != b.tick {
			return int(a.tick - b.tick)
		}
		return b.program - a.program // a program change before a note at the same tick
	})
	var programs []int
	current := -1
	for _, a := range all {
		switch {
		case a.program >= 0:
			current = a.program
		case current < 0:
			current = 0
		default:
			continue
		}
		if !slices.Contains(programs, current) {
			programs = append(programs, current)
		}
	}
	return programs
}

// Note is a note of a file: when it starts and ends (seconds, following the tempo), its key,
// velocity, channel (1-16) and track
type Note struct {
	Start, End float64
	Key        int
	Velocity   int
	Channel    int
	Track      int
}

// Notes returns the notes of the file in order of their start: every note on with the note off
// that ends it (a note never let go ends at the end of the file)
func (f *File) Notes() []Note {
	tempo := f.tempoMap()
	end := secondsAt(tempo, f.Ticks())
	var notes []Note
	for i, t := range f.Tracks {
		var open [16][128][]int // the notes started and not ended, first in first out
		for _, e := range t.Events {
			m := e.Message
			if !m.IsChannel() {
				continue
			}
			ch, key := m.Channel()-1, m.Note()
			switch m.Kind() {
			case NoteOn:
				open[ch][key] = append(open[ch][key], len(notes))
				notes = append(notes, Note{Start: secondsAt(tempo, e.Tick), End: -1, Key: key, Velocity: m.Velocity(), Channel: ch + 1, Track: i})
			case NoteOff:
				if q := open[ch][key]; len(q) > 0 {
					notes[q[0]].End = secondsAt(tempo, e.Tick)
					open[ch][key] = q[1:]
				}
			}
		}
	}
	for i := range notes {
		if notes[i].End < 0 {
			notes[i].End = end
		}
	}
	slices.SortStableFunc(notes, func(a, b Note) int {
		switch {
		case a.Start < b.Start:
			return -1
		case a.Start > b.Start:
			return 1
		}
		return 0
	})
	return notes
}
