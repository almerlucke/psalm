package timer

import (
	"github.com/almerlucke/genny"
	"github.com/almerlucke/psalm"
)

type Timer struct {
	*psalm.Module
	ticks    float64
	counter  float64
	bang     bool
	rest     bool
	ticksGen genny.Generator[float64]
}

func New(ticksGen genny.Generator[float64]) *Timer {
	s := &Timer{
		Module:   psalm.NewControl(),
		ticksGen: ticksGen,
	}

	s.SetSelf(s)
	s.setNextDuration()

	return s
}

func (t *Timer) setNextDuration() {
	if t.ticksGen.Done() {
		t.ticksGen.Reset()
		t.Send(psalm.Bang, 2)

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
	t.counter, t.bang = 0, false
	t.ticksGen.Reset()
	t.setNextDuration()
}

func (t *Timer) CtrlTick(tick psalm.CtrlTick) {
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
		t.setNextDuration()
	}
}
