package stepper

import (
	"github.com/almerlucke/genny"
	"github.com/almerlucke/psalm"
)

type Stepper struct {
	*psalm.Module
	ticks    float64
	counter  float64
	bang     bool
	rest     bool
	ticksGen genny.Generator[float64]
}

func NewStepper(ticksGen genny.Generator[float64]) *Stepper {
	s := &Stepper{
		Module:   psalm.NewControl(),
		ticksGen: ticksGen,
	}

	s.SetSelf(s)
	s.setNextDuration()

	return s
}

func (s *Stepper) setNextDuration() {
	if s.ticksGen.Done() {
		s.ticksGen.Reset()
		s.Send(psalm.Bang, 2)

	}
	ticks := s.ticksGen.Generate()
	if ticks < 0 {
		s.ticks = -ticks
	} else {
		s.ticks = ticks
		s.bang = true
	}
}

func (s *Stepper) CtrlTick(tick psalm.CtrlTick) {
	if s.bang {
		dur := s.ticks * psalm.TickDuration()
		s.bang = false
		s.Send(dur, 1)
		s.Send(psalm.Message{
			"bang":     true,
			"tick":     tick,
			"sender":   s.ID(),
			"duration": dur,
		}, 0)
	}

	s.counter += 1.0

	for s.counter >= s.ticks {
		s.counter -= s.ticks
		s.setNextDuration()
	}
}
