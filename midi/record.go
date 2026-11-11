package midi

import (
	"math"
	"sync"
)

// Recorder is an input that records another: it passes the events on as they are and, while it
// records, keeps the channel messages with the time they played (the frame they fell on). File
// turns what it kept into a MIDI file. Every method can be called from any goroutine
type Recorder struct {
	input     Input
	mu        sync.Mutex
	recording bool
	frames    int64 // frames read while recording
	rate      float64
	taken     []recorded
}

type recorded struct {
	seconds float64
	msg     Message
}

// Record returns a recorder of an input, not recording yet
func Record(input Input) *Recorder { return &Recorder{input: input} }

// Read implements Input
func (r *Recorder) Read(events []Event, frames int, sampleRate float64) []Event {
	start := len(events)
	events = r.input.Read(events, frames, sampleRate)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.recording {
		return events
	}
	r.rate = sampleRate
	for _, e := range events[start:] {
		if e.Message.IsChannel() {
			r.taken = append(r.taken, recorded{float64(r.frames+int64(e.Offset)) / sampleRate, append(Message(nil), e.Message...)})
		}
	}
	r.frames += int64(frames)
	return events
}

// Start starts recording anew (what was recorded before is let go)
func (r *Recorder) Start() {
	r.mu.Lock()
	r.recording, r.frames, r.taken = true, 0, nil
	r.mu.Unlock()
}

// Stop stops recording; what was recorded stays for File
func (r *Recorder) Stop() {
	r.mu.Lock()
	r.recording = false
	r.mu.Unlock()
}

// Recording says whether it records
func (r *Recorder) Recording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recording
}

// Len returns the messages recorded
func (r *Recorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.taken)
}

// File returns what was recorded as a MIDI file of one track at a tempo (quarter notes a minute,
// 0 is 120), 480 ticks a quarter note. The silence before the first note is left out (the
// controllers set before it come at the start); notes still held at the end are let go there
func (r *Recorder) File(bpm float64) *File {
	if bpm <= 0 {
		bpm = 120
	}
	r.mu.Lock()
	taken := append([]recorded(nil), r.taken...)
	end := float64(r.frames) / max(1, r.rate)
	r.mu.Unlock()

	first := end
	for _, t := range taken {
		if t.msg.Kind() == NoteOn {
			first = t.seconds
			break
		}
	}
	const division = 480
	tick := func(seconds float64) int64 {
		return int64(math.Round(max(0, seconds-first) * bpm / 60 * division))
	}
	var tr Track
	tr.AddMeta(0, MetaTempo, TempoData(bpm))
	tr.AddMeta(0, MetaTimeSignature, TimeSignatureData(4, 4))
	var held [16][128]bool
	for _, t := range taken {
		tr.Events = append(tr.Events, TrackEvent{Tick: tick(t.seconds), Message: t.msg}) // in order already
		ch := t.msg.Channel() - 1
		switch t.msg.Kind() {
		case NoteOn:
			held[ch][t.msg.Note()] = true
		case NoteOff:
			held[ch][t.msg.Note()] = false
		}
	}
	last := tick(end)
	if n := len(tr.Events); n > 0 {
		last = max(last, tr.Events[n-1].Tick)
	}
	for ch := range held {
		for note, on := range held[ch] {
			if on {
				tr.Events = append(tr.Events, TrackEvent{Tick: last, Message: NoteOffMsg(ch+1, note, 0)})
			}
		}
	}
	tr.Events = append(tr.Events, TrackEvent{Tick: last, Meta: MetaEndOfTrack})
	return &File{Format: 0, Division: division, Tracks: []Track{tr}}
}
