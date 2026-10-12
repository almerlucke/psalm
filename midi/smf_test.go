package midi

import (
	"bytes"
	"math"
	"path/filepath"
	"slices"
	"testing"
)

// a file by hand: format 0, 96 ticks a quarter, running status, a tempo change, a note on of
// velocity 0 as the note off
var handMade = []byte{
	'M', 'T', 'h', 'd', 0, 0, 0, 6, 0, 0, 0, 1, 0, 96,
	'M', 'T', 'r', 'k', 0, 0, 0, 40,
	0x00, 0xFF, 0x03, 0x03, 'a', 'b', 'c', // track name
	0x00, 0xFF, 0x51, 0x03, 0x07, 0xA1, 0x20, // 120 bpm
	0x00, 0x90, 60, 100, // note on
	0x60, 60, 0, // a quarter later, running status: note off
	0x00, 64, 90, // running status: note on
	0x00, 0xFF, 0x51, 0x03, 0x0F, 0x42, 0x40, // 60 bpm
	0x81, 0x40, 0x80, 64, 0, // 192 ticks later (a variable length of two bytes), note off
	0x00, 0xFF, 0x2F, 0x00,
}

func TestParseFile(t *testing.T) {
	f, err := ParseFile(handMade)
	if err != nil {
		t.Fatal(err)
	}
	if f.Format != 0 || f.Division != 96 || len(f.Tracks) != 1 {
		t.Fatalf("format %d, division %d, %d tracks", f.Format, f.Division, len(f.Tracks))
	}
	tr := f.Tracks[0]
	if tr.Name() != "abc" {
		t.Errorf("track name %q", tr.Name())
	}
	var notes []string
	for _, e := range tr.Events {
		if !e.IsMeta() {
			notes = append(notes, e.Message.Kind().String())
		}
	}
	if want := []string{"NoteOn", "NoteOff", "NoteOn", "NoteOff"}; !slices.Equal(notes, want) {
		t.Errorf("messages %v, want %v", notes, want)
	}
	if got := tr.Events[len(tr.Events)-2].Tick; got != 96+192 {
		t.Errorf("the last note off at tick %d, want 288", got)
	}
	// a quarter at 120, two at 60
	if d := f.Duration(); math.Abs(d-2.5) > 1e-9 {
		t.Errorf("duration %g s, want 2.5", d)
	}
	if chs := f.Channels(); !slices.Equal(chs, []int{1}) {
		t.Errorf("channels %v", chs)
	}
}

func TestWriteReadsBack(t *testing.T) {
	f := &File{Division: 480}
	var conductor, piano, drums Track
	conductor.AddMeta(0, MetaTempo, TempoData(90))
	conductor.AddMeta(0, MetaTimeSignature, TimeSignatureData(6, 8))
	piano.AddMeta(0, MetaTrackName, []byte("piano"))
	piano.Add(0, ProgramChangeMsg(1, 4))
	piano.Add(0, NoteOnMsg(1, 60, 80))
	piano.Add(100000, NoteOffMsg(1, 60, 0)) // a long delta
	piano.Add(50, PitchBendMsg(1, 12000))   // added out of order
	piano.Add(60, Message{0xF0, 0x7E, 0x7F, 0x09, 0x01, 0xF7})
	piano.Add(70, Message{0xF8}) // a clock: no place in a file, left out
	drums.Add(240, NoteOnMsg(10, 36, 127))
	drums.Add(250, NoteOffMsg(10, 36, 0))
	f.Tracks = []Track{conductor, piano, drums}

	var b bytes.Buffer
	if err := f.Write(&b); err != nil {
		t.Fatal(err)
	}
	g, err := ParseFile(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if g.Format != 1 || g.Division != 480 || len(g.Tracks) != 3 {
		t.Fatalf("format %d, division %d, %d tracks", g.Format, g.Division, len(g.Tracks))
	}
	if bpm, _ := g.Tracks[0].Events[0].Tempo(); math.Abs(bpm-90) > 1e-3 {
		t.Errorf("tempo %g", bpm)
	}
	if b, u, ok := g.Tracks[0].Events[1].TimeSignature(); !ok || b != 6 || u != 8 {
		t.Errorf("time signature %d/%d", b, u)
	}
	type at struct {
		tick int64
		msg  string
	}
	var got []at
	for _, e := range g.Tracks[1].Events {
		if !e.IsMeta() {
			got = append(got, at{e.Tick, e.Message.String()})
		}
	}
	want := []at{
		{0, ProgramChangeMsg(1, 4).String()},
		{0, NoteOnMsg(1, 60, 80).String()},
		{50, PitchBendMsg(1, 12000).String()},
		{60, Message{0xF0, 0x7E, 0x7F, 0x09, 0x01, 0xF7}.String()},
		{100000, NoteOffMsg(1, 60, 0).String()},
	}
	if !slices.Equal(got, want) {
		t.Errorf("piano track\n got %v\nwant %v", got, want)
	}
	if g.Tracks[1].Name() != "piano" {
		t.Errorf("name %q", g.Tracks[1].Name())
	}
	if p := g.Programs(1); !slices.Equal(p, []int{4}) {
		t.Errorf("programs of channel 1: %v", p)
	}
	if p := g.Programs(10); !slices.Equal(p, []int{0}) {
		t.Errorf("programs of channel 10: %v", p)
	}
	// a file and back
	path := filepath.Join(t.TempDir(), "x.mid")
	if err := g.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	if h, err := ReadFile(path); err != nil || len(h.Tracks) != 3 {
		t.Errorf("read back from a file: %v", err)
	}
}

func TestParseOddFiles(t *testing.T) {
	if _, err := ParseFile([]byte("hello")); err == nil {
		t.Errorf("not a MIDI file parsed")
	}
	// cut short in the middle of a note: the notes before stay
	f, err := ParseFile(handMade[:len(handMade)-20])
	if err != nil {
		t.Fatal(err)
	}
	if n := len(f.Tracks[0].Events); n < 3 {
		t.Errorf("%d events left of a file cut short", n)
	}
	// in a RIFF
	riff := append([]byte("RIFF\x00\x00\x00\x00RMIDdata"), byte(len(handMade)), 0, 0, 0)
	riff = append(riff, handMade...)
	if f, err := ParseFile(riff); err != nil || len(f.Tracks) != 1 {
		t.Errorf("an RMID: %v", err)
	}
	// SMPTE time: 25 frames a second, 40 ticks a frame: a millisecond a tick
	smpte := slices.Clone(handMade)
	smpte[12], smpte[13] = byte(256-25), 40
	f, err = ParseFile(smpte)
	if err != nil {
		t.Fatal(err)
	}
	if d := f.Duration(); math.Abs(d-0.288) > 1e-9 {
		t.Errorf("SMPTE duration %g, want 0.288", d)
	}
}

// song is two channels: at 120 bpm (480 ticks a quarter) a quarter note on channel 1, then on
// channel 2 a bend and a note on the second beat, the tempo halved on the third beat and a note
// after it
func song() *File {
	f := &File{Division: 480}
	var a, b Track
	a.AddMeta(0, MetaTempo, TempoData(120))
	a.Add(0, ProgramChangeMsg(1, 5))
	a.Add(0, NoteOnMsg(1, 60, 100))
	a.Add(480, NoteOffMsg(1, 60, 0))
	a.AddMeta(960, MetaTempo, TempoData(60))
	a.Add(1440, NoteOnMsg(1, 62, 100))
	a.Add(1920, NoteOffMsg(1, 62, 0))
	b.Add(400, ControlChangeMsg(2, CCModWheel, 90))
	b.Add(450, PitchBendMsg(2, 10000))
	b.Add(480, NoteOnMsg(2, 48, 70))
	b.Add(1900, NoteOffMsg(2, 48, 0))
	f.Tracks = []Track{a, b}
	return f
}

// collect reads a player in buffers, every event with the frame it falls on from the start
type timedMsg struct {
	frame int64
	msg   string
}

func collect(in Input, buffers, frames int, sr float64) []timedMsg {
	var all []timedMsg
	var events []Event
	for i := range buffers {
		events = in.Read(events[:0], frames, sr)
		for _, e := range events {
			if e.Offset < 0 || e.Offset >= frames {
				panic("an offset outside the buffer")
			}
			all = append(all, timedMsg{int64(i*frames + e.Offset), e.Message.String()})
		}
	}
	return all
}

func frameOf(all []timedMsg, msg Message) int64 {
	for _, m := range all {
		if m.msg == msg.String() {
			return m.frame
		}
	}
	return -1
}

func TestPlayerTiming(t *testing.T) {
	const sr = 48000.0
	f := song()
	p := NewPlayer(f)
	if d := p.Duration(); math.Abs(d-3) > 1e-9 {
		t.Fatalf("duration %g, want 3 (two beats at 120, two at 60)", d)
	}
	p.Play()
	all := collect(p, 300, 512, sr)
	for _, c := range []struct {
		msg     Message
		seconds float64
	}{
		{NoteOnMsg(1, 60, 100), 0},
		{NoteOffMsg(1, 60, 0), 0.5},
		{NoteOnMsg(2, 48, 70), 0.5},
		{PitchBendMsg(2, 10000), 450.0 / 960},
		{NoteOnMsg(1, 62, 100), 2}, // a beat at 60 after the change at 1 s
		{NoteOffMsg(1, 62, 0), 3},
	} {
		want := int64(c.seconds * sr)
		if got := frameOf(all, c.msg); got != want {
			t.Errorf("%s at frame %d, want %d", c.msg, got, want)
		}
	}
	if p.Playing() {
		t.Errorf("still playing after the end")
	}
	// the end lets go of everything: the pedal up and all notes off on both channels
	for _, ch := range []int{1, 2} {
		if frameOf(all, ControlChangeMsg(ch, CCAllNotesOff, 0)) < 0 {
			t.Errorf("no all notes off on channel %d at the end", ch)
		}
	}

	// twice as fast, everything at half the time
	p = NewPlayer(f)
	p.SetSpeed(2)
	p.Play()
	all = collect(p, 300, 512, sr)
	if got, want := frameOf(all, NoteOnMsg(1, 62, 100)), int64(1*sr); got != want {
		t.Errorf("at double speed the second note at %d, want %d", got, want)
	}
}

func TestPlayerSeekChases(t *testing.T) {
	p := NewPlayer(song())
	p.Seek(1.2) // after the bend, the wheel and the program, in the middle of channel 2's note
	p.Play()
	var events []Event
	events = p.Read(events, 256, 48000)
	var got []string
	for _, e := range events {
		got = append(got, e.Message.String())
	}
	for _, want := range []Message{
		ControlChangeMsg(1, CCResetAll, 0),
		ProgramChangeMsg(1, 5),
		ControlChangeMsg(2, CCResetAll, 0),
		ControlChangeMsg(2, CCModWheel, 90),
		PitchBendMsg(2, 10000),
	} {
		if !slices.Contains(got, want.String()) {
			t.Errorf("seeking doesn't set %s: %v", want, got)
		}
	}
	// the note of channel 2 started before the seek: it doesn't sound, its note off is left out
	all := collect(p, 200, 512, 48000)
	if frameOf(all, NoteOffMsg(2, 48, 0)) >= 0 {
		t.Errorf("a note off for a note that never played")
	}
	if frameOf(all, NoteOnMsg(1, 62, 100)) < 0 {
		t.Errorf("the note after the seek doesn't play")
	}
}

func TestPlayerLoopsAndStops(t *testing.T) {
	const sr, frames = 48000.0, 480
	p := NewPlayer(song())
	p.SetLoop(true)
	p.Play()
	all := collect(p, int(7*sr/frames), frames, sr) // 7 s: two and a bit times through
	n := 0
	var at []int64
	for _, m := range all {
		if m.msg == NoteOnMsg(1, 60, 100).String() {
			n++
			at = append(at, m.frame)
		}
	}
	if n != 3 {
		t.Fatalf("the first note played %d times in 7 s, want 3", n)
	}
	if d := at[1] - at[0]; math.Abs(float64(d)-3*sr) > 2 {
		t.Errorf("the loop is %d frames, want %g", d, 3*sr)
	}
	if !p.Playing() {
		t.Errorf("a loop stopped")
	}

	// stopping lets go of the notes sounding
	p = NewPlayer(song())
	p.Play()
	collect(p, 70, 512, sr) // into the second note of channel 2
	p.Stop()
	var events []Event
	events = p.Read(events, 512, sr)
	var got []string
	for _, e := range events {
		got = append(got, e.Message.String())
	}
	if !slices.Contains(got, NoteOffMsg(2, 48, 0).String()) {
		t.Errorf("stopping doesn't let the note go: %v", got)
	}
	if events = p.Read(events[:0], 512, sr); len(events) != 0 {
		t.Errorf("a stopped player plays %d events", len(events))
	}
	// play on from there
	pos := p.Position()
	p.Play()
	collect(p, 10, 512, sr)
	if p.Position() <= pos {
		t.Errorf("it doesn't play on from %g", pos)
	}
}

func TestPlayerMutes(t *testing.T) {
	p := NewPlayer(song())
	p.MuteChannel(2, true)
	p.Play()
	all := collect(p, 300, 512, 48000)
	if frameOf(all, NoteOnMsg(2, 48, 70)) >= 0 {
		t.Errorf("a muted channel plays a note")
	}
	if frameOf(all, PitchBendMsg(2, 10000)) < 0 {
		t.Errorf("a muted channel loses its bend")
	}
	p = NewPlayer(song())
	p.MuteTrack(0, true)
	p.Play()
	all = collect(p, 300, 512, 48000)
	if frameOf(all, NoteOnMsg(1, 60, 100)) >= 0 || frameOf(all, NoteOnMsg(2, 48, 70)) < 0 {
		t.Errorf("muting track 0 mutes the wrong notes")
	}
}

func TestTee(t *testing.T) {
	p := NewPlayer(song())
	p.Play()
	ins := Tee(p, 3)
	for range 300 {
		var counts []int
		for _, in := range ins {
			counts = append(counts, len(in.Read(nil, 512, 48000)))
		}
		if counts[0] != counts[1] || counts[1] != counts[2] {
			t.Fatalf("the readers get %v events", counts)
		}
	}
	if p.Playing() {
		t.Errorf("three readers didn't play it through at the speed of one: at %g s", p.Position())
	}
}
