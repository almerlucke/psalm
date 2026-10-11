package sampler

import "math"

const (
	stageDelay = iota
	stageAttack
	stageHold
	stageDecay
	stageSustain
	stageRelease
	stageDone
)

// envelope runs an Envelope: a delay, a linear attack, a hold, an exponential decay to the
// sustain, and an exponential release that ends 80 dB down
type envelope struct {
	stage   int
	level   float64
	count   int // samples left in the stage
	sr      float64
	set     *Envelope
	inc     float64 // the attack per sample
	coef    float64 // the decay or release per sample
	sustain float64
}

// -60 dB over a time: the decay and release
var fall60 = math.Log(1e-3)

func (e *envelope) start(set *Envelope, sr float64) {
	*e = envelope{set: set, sr: sr, sustain: math.Max(0, math.Min(set.Sustain, 1))}
	e.enter(stageDelay)
}

func (e *envelope) samples(seconds float64) int {
	return max(0, int(seconds*e.sr))
}

// enter goes to a stage, skipping the stages that take no time
func (e *envelope) enter(stage int) {
	for {
		e.stage = stage
		switch stage {
		case stageDelay:
			e.count = e.samples(e.set.Delay)
		case stageAttack:
			e.count = e.samples(e.set.Attack)
			if e.count > 0 {
				e.inc = (1 - e.level) / float64(e.count)
			} else {
				e.level = 1
			}
		case stageHold:
			e.count = e.samples(e.set.Hold)
		case stageDecay:
			e.count = e.samples(e.set.Decay)
			if e.count > 0 {
				e.coef = math.Exp(fall60 / float64(e.count))
			} else {
				e.level = e.sustain
			}
		case stageSustain:
			if e.sustain <= 0 {
				e.level, e.stage = 0, stageDone
			}
			return
		default:
			return
		}
		if e.count > 0 {
			return
		}
		stage++
	}
}

// next returns the level of the next sample
func (e *envelope) next() float64 {
	switch e.stage {
	case stageDone:
		return 0
	case stageSustain:
		return e.level
	case stageAttack:
		e.level += e.inc
	case stageDecay:
		e.level = e.sustain + (e.level-e.sustain)*e.coef
	case stageRelease:
		e.level *= e.coef
	}
	e.count--
	l := e.level
	if e.count <= 0 {
		switch e.stage {
		case stageDecay:
			e.level = e.sustain
		case stageRelease:
			e.level, e.stage = 0, stageDone
			return l
		}
		e.enter(e.stage + 1)
	}
	return l
}

// release starts the release, from the level it is at
func (e *envelope) release() {
	e.releaseIn(e.set.Release)
}

// releaseIn fades out in seconds, 80 dB down
func (e *envelope) releaseIn(seconds float64) {
	if e.stage >= stageRelease {
		if e.stage == stageRelease && e.samples(seconds) < e.count { // a faster release wins
			e.count = max(1, e.samples(seconds))
			e.coef = math.Exp(math.Log(1e-4) / float64(e.count))
		}
		return
	}
	e.stage = stageRelease
	e.count = max(1, e.samples(seconds))
	e.coef = math.Exp(math.Log(1e-4) / float64(e.count))
}
