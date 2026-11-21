// Package guitartone is the tone given to the guitar after the measured body (an impulse response,
// convolved): what the guitarbody example plays through, and what cmd/tools/guitarcompare -chain measures
// and fits against the real guitar
package guitartone

import "github.com/almerlucke/psalm/modules/effects/eq"

// ConvolverGain is the gain of the convolver: the impulse response is normalized to the energy of one, but
// a plucked string drives the few bass modes of this one hard
const ConvolverGain = 0.17

// Tone is the EQ after the convolver. The microphones that measured the body were close to it: they heard
// too much of its low modes and too little of the top
func Tone() eq.Config {
	c := eq.DefaultConfig()
	c.Mid1 = eq.Band{Freq: 190, Gain: -9, Q: 1.6} // the boom of the low strings, narrow: the high strings have their fundamentals above it
	c.Mid2 = eq.Band{Freq: 450, Gain: 6, Q: 0.9}  // the warmth of the fundamentals of the high strings
	c.High = eq.Band{Freq: 3000, Gain: 11}
	return c
}
