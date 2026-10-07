package midi

import (
	"bytes"
	"testing"
	"time"
)

func parse(data ...byte) []Message {
	var p Parser
	var out []Message
	p.Feed(data, func(m Message) { out = append(out, append(Message(nil), m...)) })
	return out
}

func TestParser(t *testing.T) {
	got := parse(
		0x90, 60, 100, // note on
		62, 90, // running status
		0xF8,        // a clock in between
		64, 0xFA, 0, // running status, a start inside the message: a note on with velocity 0
		0xB1, 64, 127, // sustain on channel 2
		0xF0, 0x7E, 0x01, 0xF7, // system exclusive
		0xE0, 0x00, 0x40, // pitch bend at the center
		0xC3, 5, // program change on channel 4
		0xF2, 0x10, 0x01, // song position
		0xD0, 33, 0xFE, // pressure, active sensing
	)
	want := []struct {
		kind Kind
		data []byte
	}{
		{NoteOn, []byte{0x90, 60, 100}},
		{NoteOn, []byte{0x90, 62, 90}},
		{Clock, []byte{0xF8}},
		{Start, []byte{0xFA}},
		{NoteOff, []byte{0x90, 64, 0}},
		{ControlChange, []byte{0xB1, 64, 127}},
		{SysEx, []byte{0xF0, 0x7E, 0x01, 0xF7}},
		{PitchBend, []byte{0xE0, 0x00, 0x40}},
		{ProgramChange, []byte{0xC3, 5}},
		{SongPosition, []byte{0xF2, 0x10, 0x01}},
		{ChannelPressure, []byte{0xD0, 33}},
		{ActiveSensing, []byte{0xFE}},
	}
	if len(got) != len(want) {
		t.Fatalf("%d messages, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind() != w.kind || !bytes.Equal(got[i], w.data) {
			t.Errorf("message %d: got %v % X, want %v % X", i, got[i].Kind(), []byte(got[i]), w.kind, w.data)
		}
	}
	if m := got[5]; m.Channel() != 2 || m.Controller() != CCSustain || m.Value() != 127 {
		t.Errorf("control change: %v", m)
	}
	if b := got[7].BendAmount(); b != 0 {
		t.Errorf("bend at the center: %v", b)
	}
	if got[8].Channel() != 4 || got[8].Program() != 5 || got[9].SongPosition() != 0x90 {
		t.Errorf("program %v, position %v", got[8], got[9].SongPosition())
	}
}

func TestParserSplit(t *testing.T) {
	// a message over two reads, and data without a status ignored
	var p Parser
	var out []Message
	emit := func(m Message) { out = append(out, append(Message(nil), m...)) }
	p.Feed([]byte{10, 20, 0x80}, emit)
	p.Feed([]byte{60}, emit)
	p.Feed([]byte{0, 61, 0}, emit)
	if len(out) != 2 || out[0].Kind() != NoteOff || out[1].Note() != 61 {
		t.Fatalf("got %v", out)
	}
}

func TestConstructors(t *testing.T) {
	for _, c := range []struct {
		m    Message
		kind Kind
	}{
		{NoteOnMsg(1, 60, 100), NoteOn},
		{NoteOffMsg(16, 60, 0), NoteOff},
		{ControlChangeMsg(3, 1, 64), ControlChange},
		{ProgramChangeMsg(1, 9), ProgramChange},
		{ChannelPressureMsg(1, 9), ChannelPressure},
		{PolyPressureMsg(1, 60, 9), PolyPressure},
		{PitchBendMsg(1, 16383), PitchBend},
	} {
		if got := parse(c.m...); len(got) != 1 || got[0].Kind() != c.kind {
			t.Errorf("%v: parsed as %v", c.m, got)
		}
	}
	if NoteOffMsg(16, 60, 0).Channel() != 16 || PitchBendMsg(1, 16383).BendAmount() != 1 || PitchBendMsg(1, 0).BendAmount() != -1 {
		t.Error("channel or bend")
	}
}

func TestQueue(t *testing.T) {
	q := NewQueue()
	now := time.Unix(100, 0)
	q.now = func() time.Time { return now }
	sr := 1000.0
	if ev := q.Read(nil, 100, sr); len(ev) != 0 {
		t.Fatal("empty queue")
	}
	// even callbacks: events at 20 and 70 ms play a buffer and a millisecond later
	q.PushAt(NoteOnMsg(1, 60, 100), now.Add(20*time.Millisecond))
	q.PushAt(NoteOffMsg(1, 60, 0), now.Add(70*time.Millisecond))
	q.PushOffset(ControlChangeMsg(1, 64, 127), 90)
	now = now.Add(100 * time.Millisecond)
	ev := q.Read(nil, 100, sr)
	if len(ev) != 3 || ev[0].Offset != 21 || ev[1].Offset != 71 || ev[2].Offset != 90 {
		t.Fatalf("got %+v", ev)
	}
}

func TestQueueBursts(t *testing.T) {
	// the callbacks come four at a time, every four buffers: the events keep their spacing
	q := NewQueue()
	start := time.Unix(100, 0)
	now := start
	q.now = func() time.Time { return now }
	sr, frames := 1000.0, 10
	var pushed []time.Time
	frame := int64(0)
	var got []int64
	for burst := 0; burst < 20; burst++ {
		now = start.Add(time.Duration(burst*4*frames) * time.Millisecond)
		for range 4 {
			for _, e := range q.Read(nil, frames, sr) {
				got = append(got, frame+int64(e.Offset))
			}
			frame += int64(frames)
			now = now.Add(100 * time.Microsecond)
		}
		// events between the bursts, 7 ms apart
		for at := burst * 4 * frames; at < (burst+1)*4*frames; at += 7 {
			if at > burst*4*frames {
				pushed = append(pushed, start.Add(time.Duration(at)*time.Millisecond))
				q.PushAt(NoteOnMsg(1, 60, 100), pushed[len(pushed)-1])
			}
		}
	}
	if len(got) < len(pushed)-8 {
		t.Fatalf("%d of %d events", len(got), len(pushed))
	}
	for i := range got {
		want := float64(pushed[i].Sub(start).Milliseconds())
		if lat := float64(got[i]) - want; lat < 1 || lat > float64(4*frames)+2 || i > 0 && got[i]-got[i-1] != int64(want-float64(pushed[i-1].Sub(start).Milliseconds())) {
			t.Fatalf("event %d at frame %d, pushed at %v ms (latency %v): %v", i, got[i], want, lat, got)
		}
	}
}

type fixed []Event

func (f fixed) Read(events []Event, _ int, _ float64) []Event { return append(events, f...) }

func TestInputs(t *testing.T) {
	in := Inputs{fixed{{Offset: 5}, {Offset: 50}}, fixed{{Offset: 1}, {Offset: 30}}}
	ev := in.Read(nil, 64, 44100)
	for i, o := range []int{1, 5, 30, 50} {
		if ev[i].Offset != o {
			t.Fatalf("got %+v", ev)
		}
	}
}
