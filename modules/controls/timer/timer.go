package timer

import (
	"math"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/psalm"
)

// Timer bangs after durations in ticks, from a generator of durations (a negative one is a rest).
// It follows the clock of its patch: a timer that misses ticks (out of the patch for a while, not
// processed) catches up silently and goes on where it would have been, and a timer added while the
// patch plays starts where it would be had it run from the start. FromNow starts it at its first
// tick instead
type Timer struct {
	*psalm.Module
	ticks    float64
	counter  float64
	bang     bool
	rest     bool
	ticksGen genny.Generator[float64]
	last     int64 // the tick (TickTimestamp) of the last control tick, 0 for none yet
	fromNow  bool  // starts at its first tick, not aligned to the clock
}

func New(ticksGen genny.Generator[float64]) *Timer {
	s := &Timer{
		Module:   psalm.NewControl(),
		ticksGen: ticksGen,
	}

	s.SetSelf(s)
	s.setNextDuration(true)

	return s
}

// FromNow makes the timer start at the first control tick it gets, wherever the clock of the patch
// is (its first duration starts there), instead of where it would be had it run from the start of
// the clock. A timer added to a patch before it plays starts at the start either way
func (t *Timer) FromNow() *Timer {
	t.fromNow = true
	return t
}

// setNextDuration takes the next duration; send tells the loop's end (output 2), not while catching up
func (t *Timer) setNextDuration(send bool) {
	if t.ticksGen.Done() {
		t.ticksGen.Reset()
		if send {
			t.Send(psalm.Bang, 2)
		}
	}
	ticks := t.ticksGen.Generate()
	if ticks < 0 {
		t.ticks = -ticks
	} else {
		t.ticks = ticks
		t.bang = true
	}
}

// Restart starts the timer again from the start of its durations (psalm.Restarter, from
// Psalm.Restart): the first duration starts with the next tick, as when the patch started
func (t *Timer) Restart() {
	t.counter, t.bang, t.last = 0, false, 0
	t.ticksGen.Reset()
	t.setNextDuration(true)
}

// skip goes through a number of ticks without banging: the durations go by as if it had played
// them (a bang due on the tick after the last one skipped is kept)
func (t *Timer) skip(n int64) {
	for n > 0 && t.ticks > 0 {
		t.bang = false
		// the ticks that go by before one crosses the end of the duration
		free := int64(math.Ceil(t.ticks-t.counter)) - 1
		if free >= n {
			t.counter += float64(n)
			return
		}
		if free > 0 {
			t.counter += float64(free)
			n -= free
		}
		t.counter += 1.0 // the tick crossing it
		n--
		for t.counter >= t.ticks {
			t.counter -= t.ticks
			t.setNextDuration(false)
		}
	}
}

// follow keeps the timer with the clock of its patch at a tick: it catches up on ticks it missed, a
// clock going back (a restart) starts it again; false for a tick it had already
func (t *Timer) follow(now int64) bool {
	switch {
	case t.last > 0 && now == t.last:
		return false
	case t.last > 0 && now < t.last: // the clock went back
		t.Restart()
		t.skip(now - 1)
	case t.last == 0 && !t.fromNow: // the first tick: where it would be had it run from the start
		t.skip(now - 1)
	case t.last > 0:
		t.skip(now - t.last - 1) // the ticks it missed
	}
	t.last = now
	return true
}

func (t *Timer) CtrlTick(tick psalm.CtrlTick) {
	if !t.follow(tick.TickTimestamp) {
		return
	}
	if t.bang {
		dur := t.ticks * t.Env().TickDuration()
		t.bang = false
		t.Send(dur, 1)
		t.Send(psalm.Message{
			"bang":     true,
			"tick":     tick,
			"sender":   t.ID(),
			"duration": dur,
		}, 0)
	}

	t.counter += 1.0

	for t.counter >= t.ticks {
		t.counter -= t.ticks
		t.setNextDuration(true)
	}
}
