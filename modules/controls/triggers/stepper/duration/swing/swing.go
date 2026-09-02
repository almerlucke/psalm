package swing

import (
	"math"
	"math/rand"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/psalm"
)

var _shuffleMultiplier = 0.7

type Step struct {
	Skip        bool    `json:"skip"`
	Shuffle     float64 `json:"shuffle"` // 0.0 - 1.0 -> (0.5 + shuffle * 0.35) * milliPerNote * 2.0  50% - 85%  --- 50, 54, 58, 62, 66, 70, 74, 78, 82, 86
	ShuffleRand float64 `json:"shuffleRand"`
	SkipChance  float64 `json:"skipChance"` // 0% - 90% chance of skipping
	Multiply    float64 `json:"multiply"`
	BurstChance float64 `json:"burstChance"`
	NumBurst    int     `json:"numBurst"`
}

func (s *Step) shuffleDelay(ticks float64) float64 {
	shuffleRandBandwidth := math.Min(s.Shuffle, 1.0-s.Shuffle)
	shuffleAmount := s.Shuffle + (rand.Float64()*2.0-1.0)*math.Min(s.ShuffleRand, shuffleRandBandwidth)

	return _shuffleMultiplier * shuffleAmount * ticks
}

func (s *Step) Burst() bool {
	return rand.Float64() < s.BurstChance
}

func (s *Step) SkipStep() bool {
	return s.Skip || rand.Float64() < s.SkipChance
}

/*
[]*swing.Step{
		{}, {Skip: true}, {Skip: true}, {}, {Skip: true}, {Skip: true}, {Skip: true}, {Skip: true},
		{Skip: true}, {Skip: true}, {Skip: true}, {Skip: true}, {Skip: true}, {Skip: true}, {Skip: true}, {Skip: true},
	})


*/

func QuickSteps(args ...any) []*Step {
	steps := make([]*Step, len(args))

	for index, arg := range args {
		var step *Step

		switch v := arg.(type) {
		case int:
			step = &Step{
				Skip: v == 0,
			}
		case bool:
			step = &Step{
				Skip: !v,
			}
		case *Step:
			step = v
		case float32:
			var skip bool
			if v < 0.0 {
				skip = true
				v = -v
			}
			step = &Step{
				Multiply: float64(v),
				Skip:     skip,
			}
		case float64:
			var skip bool
			if v < 0.0 {
				skip = true
				v = -v
			}
			step = &Step{
				Multiply: v,
				Skip:     skip,
			}
		default:
			step = &Step{
				Skip: true,
			}
		}

		steps[index] = step
	}

	return steps
}

func Multiply(steps []*Step, multiply float64) []*Step {
	if multiply > 0 {
		for _, step := range steps {
			if step.Multiply > 0.0 {
				step.Multiply *= multiply
			} else {
				step.Multiply = multiply
			}
		}
	}
	return steps
}

type Swing struct {
	steps             genny.Generator[*Step]
	remainingDuration float64
	delayed           bool
	burstCount        int
	burstMode         bool
	burstDuration     float64
}

func New(steps genny.Generator[*Step]) *Swing {
	return &Swing{
		steps: steps,
	}
}

func (sw *Swing) Generate() float64 {
	if sw.delayed {
		sw.delayed = false
		return sw.remainingDuration
	}

	if sw.burstMode {
		sw.burstCount--
		if sw.burstCount == 0 {
			sw.burstMode = false
		}

		return sw.burstDuration
	}

	step := sw.steps.Generate()

	multiply := 1.0

	if step.Multiply > 0.0 {
		multiply = step.Multiply
	}

	ticks := float64(psalm.PPQ()) * multiply

	if step.SkipStep() {
		return -ticks
	}

	if step.Burst() {
		sw.burstMode = true
		sw.burstCount = step.NumBurst
		sw.burstDuration = ticks / float64(step.NumBurst)
		return sw.Generate()
	}

	delay := step.shuffleDelay(ticks)
	if delay > 0.0 {
		sw.remainingDuration = ticks - delay
		sw.delayed = true
		return -delay
	}

	return ticks
}

func (sw *Swing) Done() bool {
	return sw.steps.Done()
}

func (sw *Swing) Reset() {
	sw.steps.Reset()
	sw.delayed = false
	sw.burstMode = false
}

func (sw *Swing) Continuous() bool {
	return sw.steps.Continuous()
}
