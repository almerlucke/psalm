package dx

import "math"

// The engine maps the classic 0-99 parameters to sound with these approximations of the hardware
// behavior. They are kept in one place so they are easy to tune

// levelToAmp converts a level (0-99, about 0.75 dB per step) to a linear amplitude, 99 is 1
func levelToAmp(level float64) float64 {
	if level <= 0.0 {
		return 0.0
	}
	return math.Pow(10.0, -(99.0-level)*0.0375)
}

// rateToSpeed converts an envelope rate (0-99) to level units per sample: a full 0-99 sweep takes
// about 38 seconds at rate 0 and about 1 millisecond at rate 99
func rateToSpeed(rate float64, sr float64) float64 {
	rate = max(0.0, min(rate, 99.0))
	seconds := 38.0 * math.Pow(2.0, -rate/6.5)
	return 99.0 / (seconds * sr)
}

// pitchLevelToSemitones converts a pitch envelope level (0-99, 50 is no shift) to semitones,
// gentle around the middle and up to about 4 octaves at the extremes
func pitchLevelToSemitones(level float64) float64 {
	x := (level - 50.0) / 50.0
	return 12.0*x + 36.0*x*x*x
}

var lfoSpeedTable = [][2]float64{{0, 0.06}, {10, 1.0}, {20, 2.5}, {35, 5.4}, {50, 8.0}, {70, 15.0}, {99, 47.0}}

// lfoSpeedToHz converts an LFO speed (0-99) to Hz by interpolating a table of approximate values
func lfoSpeedToHz(speed int) float64 {
	s := float64(max(0, min(speed, 99)))
	for i := 1; i < len(lfoSpeedTable); i++ {
		a, b := lfoSpeedTable[i-1], lfoSpeedTable[i]
		if s <= b[0] {
			t := (s - a[0]) / (b[0] - a[0])
			return a[1] * math.Pow(b[1]/a[1], t)
		}
	}
	return lfoSpeedTable[len(lfoSpeedTable)-1][1]
}

// lfoDelayToSeconds converts an LFO delay (0-99) to seconds
func lfoDelayToSeconds(delay int) float64 {
	d := float64(max(0, min(delay, 99))) / 99.0
	return 5.0 * d * d
}

// pitchModSemitones is the maximum LFO pitch modulation per pitch mod sensitivity (0-7)
var pitchModSemitones = [8]float64{0, 0.08, 0.15, 0.25, 0.4, 0.7, 1.3, 7.0}

// ampModDepth is the LFO amplitude modulation depth per amp mod sensitivity (0-3)
var ampModDepth = [4]float64{0, 0.25, 0.5, 1.0}

// feedbackAmount converts feedback (0-7) to a phase modulation amount in cycles
func feedbackAmount(fb int) float64 {
	if fb <= 0 {
		return 0.0
	}
	return math.Pow(2.0, float64(min(fb, 7)-7)) * 0.6
}

// modIndex is the modulation index in cycles of a modulator at full level (about 12.6 radians)
const modIndex = 2.0

// keyboardScaling returns the level offset (in level units) for a note, from the break point,
// depths and curves of an operator
func keyboardScaling(op *Operator, note float64) float64 {
	dist := note - float64(op.BreakPoint+21) // break point 39 is C3 (MIDI 60)
	depth, curve := op.RightDepth, op.RightCurve
	if dist < 0 {
		depth, curve, dist = op.LeftDepth, op.LeftCurve, -dist
	}

	octaves := dist / 12.0
	var amount float64
	switch curve {
	case 1, 2: // exponential
		amount = float64(depth) * (math.Pow(2.0, octaves) - 1.0) / 15.0
	default: // linear
		amount = float64(depth) * octaves / 4.0
	}

	if curve <= 1 { // -LIN, -EXP
		return -amount
	}
	return amount
}

// attackSpeedup makes rising envelope stages faster than falling ones at the same rate, fastest
// while the level is still low (8 times at level 0, easing to 2 times at the top). A full attack
// takes about a quarter of the time of the same decay, and the sound becomes audible quickly
// instead of rising through the inaudible low levels first, like the original envelopes
func attackSpeedup(level float64) float64 {
	return 2.0 * (4.0 - 3.0*level/99.0)
}

// envelope is a 4 rate / 4 level envelope: from the current level it moves to level 1, 2 and 3 at
// rates 1, 2 and 3, holds level 3 while the note is held and moves to level 4 at rate 4 on release.
// Levels move in level units (decibels), so decays sound natural, rising stages are faster (see
// attackSpeedup)
type envelope struct {
	level  float64
	stage  int // 0-2 moving to level 1-3, 3 sustain, 4 release, 5 idle
	speeds [4]float64
	levels [4]float64
}

const (
	stageSustain = 3
	stageRelease = 4
	stageIdle    = 5
)

func (e *envelope) keyOn(rates [4]int, levels [4]int, rateAdd float64, sr float64) {
	for i := range 4 {
		e.speeds[i] = rateToSpeed(float64(rates[i])+rateAdd, sr)
		e.levels[i] = float64(levels[i])
	}
	if e.stage == stageIdle {
		e.level = e.levels[3]
	}
	e.stage = 0
}

func (e *envelope) keyOff() {
	if e.stage < stageRelease {
		e.stage = stageRelease
	}
}

func (e *envelope) idle() bool {
	return e.stage == stageIdle
}

func (e *envelope) reset() {
	e.stage = stageIdle
	e.level = 0.0
}

// next advances the envelope by one sample and returns the level
func (e *envelope) next() float64 {
	var target, speed float64

	switch e.stage {
	case 0, 1, 2:
		target, speed = e.levels[e.stage], e.speeds[e.stage]
	case stageRelease:
		target, speed = e.levels[3], e.speeds[3]
	default:
		return e.level
	}

	if e.level < target {
		e.level = min(target, e.level+speed*attackSpeedup(e.level))
	} else {
		e.level = max(target, e.level-speed)
	}

	if e.level == target {
		e.stage++ // 2 -> sustain, release -> idle
	}

	return e.level
}
