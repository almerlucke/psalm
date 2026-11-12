package shaper

import "github.com/almerlucke/psalm/blocks/modmatrix"

// CZResonant is the Casio CZ resonant wave: a falling saw times a cosine of higher pitch (czres), whose resonance rises as the
// note opens (m1) and returns as it dies, so the key sounds like a filter sweep that no filter makes
func CZResonant() Config {
	return Config{
		Chains: [NumChains]string{"czres m1[0.15,1] m2[3,9]"},
		Macros: [4]float64{0.2, 0.5}, ShapeAmount: 0.8, ShapeEnv: env(5, 700, 0.3, 400),
		Line1: Line{Level: 1}, Line2: Line{Detune: 6, Level: 0.5},
		AmpEnv: env(4, 500, 0.7, 300), Width: 0.4, Level: 0.86,
		Filter: softLow(6000),
		Mods:   [modmatrix.Slots]modmatrix.Slot{{Src: modmatrix.Velocity, Dest: "macro1", Amount: 0.3}},
	}
}

// TubeOrgan is a sine and its octave and fifth driven through an asymmetric tube (tube), whose bias leans the wave so the even
// harmonics come in with the touch
func TubeOrgan() Config {
	return Config{
		Chains: [NumChains]string{"par(sin, mult 2 | sin | mult 0.5, mult 3 | sin | mult 0.3) | tube m1[1,5] m2[0.05,0.5]"},
		Macros: [4]float64{0.35, 0.4},
		Line1:  Line{Level: 1}, Line2: Line{Octave: -1, Detune: 4, Level: 0.35},
		Unison: 2, Spread: 6, UniStereo: 0.5,
		AmpEnv: env(12, 300, 0.85, 220), Width: 0.4, Level: 0.5,
		Filter: softLow(4800),
		Mods:   [modmatrix.Slots]modmatrix.Slot{{Src: modmatrix.Velocity, Dest: "macro1", Amount: 0.4}},
	}
}
