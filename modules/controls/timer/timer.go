package timer

import (
	"github.com/almerlucke/psalm"
)

type Timer struct {
	*psalm.Module
	previousTimestamp int64
	tickCntr          int64
	ticksPerBeat      int64
	firstBang         bool
}

func New(ticksPerBeat int, firstBang bool) *Timer {
	t := &Timer{
		Module:       psalm.NewControl(),
		ticksPerBeat: int64(ticksPerBeat),
		firstBang:    firstBang,
	}

	t.SetSelf(t)

	return t
}

func (t *Timer) CtrlTick(tick psalm.CtrlTick) {
	if tick.TickTimestamp == 1 && t.firstBang {
		t.Send(psalm.Bang, 0)
		return
	}

	t.tickCntr++
	for t.tickCntr > t.ticksPerBeat {
		t.tickCntr -= t.ticksPerBeat
		t.Send(psalm.Bang, 0)
	}
}
