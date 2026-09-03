package timer

import (
	"github.com/almerlucke/genny/constant"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
)

func WithTicks(ticks float64) *Timer {
	return New(constant.New(ticks))
}

func WithSteps(s []*steps.Step) *Timer {
	return New(steps.New(sequence.NewLoop[*steps.Step](s...)))
}
