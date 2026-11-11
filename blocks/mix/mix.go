// Package mix has the building blocks effects share to blend and change their settings: an equal
// power dry/wet crossfade and a smoother that glides a setting to a new value, so automating it
// doesn't step or click
package mix

import (
	"math"

	"github.com/almerlucke/psalm/blocks/sine"
)

// EqualPower returns the dry and wet gains of an equal power crossfade at mix (0 dry, 1 wet). A
// wet signal that doesn't correlate with the dry one (a reverb, echoes, chorus voices, grains)
// adds in power, so the crossfade keeps the loudness at every mix: halfway is as loud as dry, where
// a linear crossfade dips 3 dB. Not for effects that make their sound by cancelling against the
// dry signal (a flanger, a phaser), they keep a linear crossfade
func EqualPower(mix float64) (dry, wet float64) {
	mix = max(0, min(mix, 1))
	return sine.CosCycle(0.25 * mix), sine.Cycle(0.25 * mix)
}

// LinearToEqualPower returns the mix of an equal power crossfade with the same balance between
// dry and wet as a linear crossfade at mix, and the gain (at most 1) that makes it as loud: the
// linear crossfade equals the equal power one at the returned mix, times gain
func LinearToEqualPower(mix float64) (equalMix, gain float64) {
	mix = max(0, min(mix, 1))
	gain = math.Hypot(1-mix, mix)
	return math.Atan2(mix, 1-mix) * 2 / math.Pi, gain
}

// SmoothMs is how fast settings glide by default: fast enough to follow automation, slow enough
// not to click
const SmoothMs = 30.0

// Smoother glides a setting to its target with a one pole lowpass
type Smoother struct {
	Value float64 // the current value
	coef  float64
}

// NewSmoother returns a smoother that starts at value and covers about two thirds of a change in
// ms milliseconds
func NewSmoother(value, ms, sampleRate float64) Smoother {
	return Smoother{Value: value, coef: 1 - math.Exp(-1/(ms*0.001*sampleRate))}
}

// Next moves the value a sample towards target and returns it
func (s *Smoother) Next(target float64) float64 {
	s.Value += (target - s.Value) * s.coef
	return s.Value
}
