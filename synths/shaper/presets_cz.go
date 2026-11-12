package shaper

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

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

// stage is a sound that plays one stage of the chain language with m1 swept over the note by the shape envelope (from to to): the
// presets that show the stages of genny's newer shapers
func stage(chain string, from, to float64, amp, shape adsr.Setting, level float64, cutoff float64) Config {
	return Config{
		Chains: [NumChains]string{chain},
		Macros: [4]float64{from}, ShapeAmount: to - from, ShapeEnv: shape,
		Line1: Line{Level: 1}, Line2: Line{Detune: 5, Level: 0.4},
		AmpEnv: amp, Width: 0.4, Level: level,
		Filter: softLow(cutoff),
	}
}

// FoldLead is a sine folded more and more as the note opens: the harmonics climb like a west coast synth's
func FoldLead() Config {
	return stage("sin | fold m1[1,6]", 0, 1, env(8, 500, 0.75, 350), env(5, 900, 0.4, 500), 0.6, 5500)
}

// TriFoldPad is a sine reflected at the edges by a gain that rises slowly: a pad that grows teeth
func TriFoldPad() Config {
	c := stage("sin | trifold m1[1,5]", 0, 1, env(600, 900, 0.85, 1400), env(2500, 1500, 0.6, 1500), 0.6, 4500)
	c.Unison, c.Spread, c.UniStereo = 2, 7, 0.6
	return c
}

// TubeBass is a sine driven harder into an asymmetric tube as the note starts, then settling: a round bass with even harmonics
func TubeBass() Config {
	return stage("sin | tube m1[1,8] 0.4", 0, 1, env(4, 400, 0.85, 200), env(3, 350, 0.3, 200), 0.6, 3200)
}

// SoftDeadLead is a sine with the middle cut out (a dead zone widening over the note): a thin, reedy lead
func SoftDeadLead() Config {
	return stage("sin | softdead m1[0,0.8] | mult 1.8", 0, 1, env(10, 400, 0.8, 300), env(6, 1000, 0.5, 400), 0.5, 5500)
}

// CurveBass is a saw bent by a tension from one side to the other, a bass that is hollow and then full
func CurveBass() Config {
	return stage("bipolar | curve m1[-0.9,0.9]", 0, 1, env(4, 400, 0.8, 200), env(4, 600, 0.3, 300), 0.6, 3000)
}

// StepLead is a triangle pressed to a square by a smootherstep whose edge closes in: a lead that goes from soft to bright
func StepLead() Config {
	return stage("tri | smootherstep 0 m1[1,0.15] | bipolar", 0, 1, env(10, 400, 0.8, 300), env(5, 900, 0.4, 400), 0.45, 5500)
}

// CZSawLead is a Casio CZ saw whose distortion grows during the note: a cosine that becomes a saw
func CZSawLead() Config {
	c := stage("czsaw m1[0.2,1] | tanh 1.6", 0, 1, env(8, 450, 0.8, 300), env(5, 800, 0.5, 400), 0.46, 3800)
	c.Line2 = Line{Octave: -1, Detune: 11, Level: 0.5}
	c.VibratoRate, c.VibratoDepth, c.VibratoDelay = 5.2, 0.15, 0.4
	return c
}

// CZSquareBass is the CZ square: a cosine turned into a pulse with fast edges
func CZSquareBass() Config {
	return stage("czsquare m1[0,0.95]", 0, 1, env(4, 400, 0.85, 200), env(3, 500, 0.4, 250), 0.45, 3500)
}

// CZPulseArp is the CZ pulse: a cosine squeezed into the start of the cycle, plucked
func CZPulseArp() Config {
	return stage("czpulse m1", 0, 1, env(3, 250, 0.1, 250), env(3, 300, 0.1, 200), 0.6, 5500)
}

// HarmPad is a sum of sines (1, 3, 5 and 8) as a waveshaper, so a pitch comes out with those partials and their sum tones, held
func HarmPad() Config {
	c := stage("bipolar | harm 1:1 3:0.6 5:0.4 8:0.3", 0, 0, env(700, 900, 0.85, 1500), env(1, 1, 1, 1), 0.7, 4800)
	c.Unison, c.Spread, c.UniStereo, c.Drift = 3, 9, 0.7, 0.3
	return c
}

// PolyLead is a saw through a cubic (1.6x - 0.6x^3): a rounded saw, soft at the peaks
func PolyLead() Config {
	return stage("bipolar | poly 0 1.6 0 -0.6", 0, 0, env(8, 400, 0.8, 300), env(1, 1, 1, 1), 0.46, 5500)
}

// LegendrePad is the Legendre polynomial of order five as a waveshaper: a strange, even stack of harmonics
func LegendrePad() Config {
	c := stage("bipolar | legendre 5", 0, 0, env(600, 800, 0.85, 1400), env(1, 1, 1, 1), 1, 4500)
	c.Unison, c.Spread, c.UniStereo = 2, 8, 0.6
	return c
}

// FunnelArp is a saw snapped to six levels: a staircase, a plucked sound with the grain of a low bit depth
func FunnelArp() Config {
	return stage("bipolar | funnel -1 -0.6 -0.2 0.2 0.6 1", 0, 0, env(3, 300, 0.1, 250), env(1, 1, 1, 1), 0.7, 5000)
}

// DuoLead is two different oscillators: line 1 a sine folded as the note opens, line 2 (chains of its own) a CZ saw an octave up and a few cents
// off, whose distortion rises with it; the two together are fuller than one wave twice
func DuoLead() Config {
	c := stage("sin | fold m1[1,4]", 0, 1, env(8, 500, 0.75, 350), env(5, 900, 0.4, 500), 0.5, 5200)
	c.Chains2[0] = "czsaw m1[0.15,0.9]"
	c.Line2 = Line{Octave: 1, Detune: 7, Level: 0.5}
	c.Width = 0.6
	c.Unison, c.Spread, c.UniStereo = 2, 6, 0.5
	return c
}
