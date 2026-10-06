package steps

import (
	"math"
	"math/rand/v2"

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

func (s *Step) shuffleDelay(ticks float64, rng *rand.Rand) float64 {
	shuffleRandBandwidth := math.Min(s.Shuffle, 1.0-s.Shuffle)
	shuffleAmount := s.Shuffle + (rng.Float64()*2.0-1.0)*math.Min(s.ShuffleRand, shuffleRandBandwidth)

	return _shuffleMultiplier * shuffleAmount * ticks
}

func (s *Step) Burst(rng *rand.Rand) bool {
	return rng.Float64() < s.BurstChance
}

func (s *Step) SkipStep(rng *rand.Rand) bool {
	return s.Skip || rng.Float64() < s.SkipChance
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

func Multiply(s []*Step, multiply float64) []*Step {
	if multiply > 0 {
		for _, step := range s {
			if step.Multiply > 0.0 {
				step.Multiply *= multiply
			} else {
				step.Multiply = multiply
			}
		}
	}
	return s
}

type Steps struct {
	steps             genny.Generator[*Step]
	rng               *rand.Rand // a stream of the patch being built
	ppq               int
	remainingDuration float64
	delayed           bool
	burstCount        int
	burstMode         bool
	burstDuration     float64
}

func New(s genny.Generator[*Step]) *Steps {
	return &Steps{
		steps: s,
		rng:   psalm.CurrentEnv().NewRand(),
		ppq:   psalm.PPQ(),
	}
}

func (s *Steps) Generate() float64 {
	if s.delayed {
		s.delayed = false
		return s.remainingDuration
	}

	if s.burstMode {
		s.burstCount--
		if s.burstCount == 0 {
			s.burstMode = false
		}

		return s.burstDuration
	}

	step := s.steps.Generate()

	multiply := 1.0

	if step.Multiply > 0.0 {
		multiply = step.Multiply
	}

	ticks := float64(s.ppq) * multiply

	if step.SkipStep(s.rng) {
		return -ticks
	}

	if step.Burst(s.rng) {
		s.burstMode = true
		s.burstCount = step.NumBurst
		s.burstDuration = ticks / float64(step.NumBurst)
		return s.Generate()
	}

	delay := step.shuffleDelay(ticks, s.rng)
	if delay > 0.0 {
		s.remainingDuration = ticks - delay
		s.delayed = true
		return -delay
	}

	return ticks
}

func (s *Steps) Done() bool {
	return s.steps.Done()
}

func (s *Steps) Reset() {
	s.steps.Reset()
	s.delayed = false
	s.burstMode = false
}

func (s *Steps) Continuous() bool {
	return s.steps.Continuous()
}
