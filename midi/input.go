package midi

import (
	"sync"
	"time"
)

// Event is a message in an audio buffer: Offset is the frame it falls on
type Event struct {
	Message Message
	Offset  int
}

// Input hands the events of an audio buffer over. Read is called once for every buffer, before
// it is rendered, with the frames in the buffer; it appends the events that play in it, in
// order, every offset within [0, frames)
type Input interface {
	Read(events []Event, frames int, sampleRate float64) []Event
}

// Queue is an input that takes messages from anywhere (another goroutine, a device, a network
// connection, generated input). A message pushed by time keeps its timing: it is placed in the
// buffer that plays it a constant latency after it arrived, at the frame that latency puts it on.
//
// Audio callbacks don't come evenly: a sound server often asks for a few buffers right after
// each other and then waits. The queue follows when the buffers are read against how much audio
// they hold, and keeps the latency just long enough to cover that (a buffer when the callbacks
// are even); a message can wait for a later buffer
type Queue struct {
	mu      sync.Mutex
	pending []timed
	now     func() time.Time
	origin  time.Time // the time of the first read
	pos     int64     // frames read
	lags    [512]float64
	lagN    int
	last    int64 // the frame of the last message placed by time
}

type timed struct {
	msg    Message
	at     time.Time
	frame  int64 // where it plays, -1 until placed
	offset int   // a frame offset in the next buffer, -1 when placed by time
}

// lateness above which the timing of the callbacks is taken to have changed (a dropout)
const maxJitter = 0.1

// NewQueue returns an empty queue
func NewQueue() *Queue {
	return &Queue{now: time.Now}
}

// Push adds a message, placed by the time it arrives. The message is copied
func (q *Queue) Push(msg Message) {
	q.PushAt(msg, q.now())
}

// PushAt adds a message that arrived at a time
func (q *Queue) PushAt(msg Message, at time.Time) {
	q.mu.Lock()
	q.pending = append(q.pending, timed{msg: append(Message(nil), msg...), at: at, frame: -1, offset: -1})
	q.mu.Unlock()
}

// PushOffset adds a message at a frame offset of the next buffer read
func (q *Queue) PushOffset(msg Message, offset int) {
	q.mu.Lock()
	q.pending = append(q.pending, timed{msg: append(Message(nil), msg...), frame: -1, offset: max(0, offset)})
	q.mu.Unlock()
}

// Read implements Input
func (q *Queue) Read(events []Event, frames int, sampleRate float64) []Event {
	now := q.now()
	q.mu.Lock()
	defer q.mu.Unlock()

	// how late this buffer is read for the audio before it; the earliest read of the last few
	// seconds is the reference
	if q.origin.IsZero() {
		q.origin = now
	}
	lag := now.Sub(q.origin).Seconds() - float64(q.pos)/sampleRate
	earliest := lag
	for _, l := range q.lags[:min(q.lagN, len(q.lags))] {
		earliest = min(earliest, l)
	}
	if lag-earliest > maxJitter {
		q.lagN, earliest = 0, lag // a dropout: start over
	}
	q.lags[q.lagN%len(q.lags)] = lag
	q.lagN++

	// a message plays a buffer (and a millisecond) after the earliest read could have taken it
	latency := float64(frames) + 0.001*sampleRate
	start, end := q.pos, q.pos+int64(frames)
	q.pos = end
	n, prev := 0, 0
	for i := range q.pending {
		t := &q.pending[i]
		offset := t.offset
		if offset < 0 {
			if t.frame < 0 {
				at := t.at.Sub(q.origin).Seconds() - earliest
				t.frame = max(q.last, int64(at*sampleRate+latency)) // in order
				q.last = t.frame
			}
			if t.frame >= end {
				break // the rest waits for a later buffer
			}
			offset = int(max(0, t.frame-start))
		}
		prev = max(prev, min(offset, frames-1))
		events = append(events, Event{Message: t.msg, Offset: prev})
		n++
	}
	rest := copy(q.pending, q.pending[n:])
	clear(q.pending[rest:])
	q.pending = q.pending[:rest]
	return events
}

// Inputs merges inputs into one (all keyboards, a device and generated input)
type Inputs []Input

// Read implements Input, the events of all inputs ordered by their offsets
func (in Inputs) Read(events []Event, frames int, sampleRate float64) []Event {
	from := len(events)
	for _, i := range in {
		events = i.Read(events, frames, sampleRate)
	}
	// insertion sort keeps the order of every input (the buffers are short)
	merged := events[from:]
	for i := 1; i < len(merged); i++ {
		for j := i; j > 0 && merged[j].Offset < merged[j-1].Offset; j-- {
			merged[j], merged[j-1] = merged[j-1], merged[j]
		}
	}
	return events
}
