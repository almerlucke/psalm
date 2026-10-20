package shaper

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

// --- beyond: sounds only this synth makes ---
//
// The shaper builds the wave itself and lets anything in the matrix reach into it, so a sound can do what an oscillator
// and a filter cannot: travel through four different waves in one held note, be another instrument on every key
// or at every strength of touch, draw itself anew with every note, ring with partials that belong to no pitch, and play a
// drum kit's thunder from the same sine as the bass.

var (
	key    = modmatrix.Key
	random = modmatrix.Random
	ampEnv = modmatrix.AmpEnv
)

// Rainbow is a pad that travels through four colours in one held chord: a sine, a bend of two sines, a folded wave and at
// last the partials 1, 3, 7 and 12 shining together. The travel is the morph envelope (six seconds up, then it settles
// between the colours) and a slow wander on top, so no two chords turn the same way
func Rainbow() Config {
	return Config{
		Chains: [NumChains]string{
			"sin",
			pm(1, "m1[0.08,0.4]"),
			"sin | mult m2[1.2,3] | mirror -0.7 0.7 | sin",
			"par(sin, mult 3 | sin | mult 0.5, mult 7 | sin | mult 0.33, mult 12 | sin | mult 0.2)",
		},
		Morph: 0.05, MorphAmount: 0.9, MorphEnv: env(6000, 9000, 0.4, 2500), Macros: [4]float64{0.4, 0.5},
		Unison: 3, Spread: 8, UniStereo: 0.8, Drift: 0.4, Sub: 0.15,
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Detune: 5, Level: 0.25},
		AmpEnv: env(1100, 800, 0.9, 2200), Width: 0.7, Level: 0.55, LFO2Rate: 0.083,
		Filter: softLow(3600),
		Mods:   slots(modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.12}),
	}
}

// CrystalSea is a pad that does not repeat: two slow waves, one of twelve seconds and one of seven, move the morph and a
// macro, a new value for every note moves another, and the copies of the unison drift. The waves never come back to the
// same place, so the sea is never the same twice
func CrystalSea() Config {
	return Config{
		Chains: [NumChains]string{
			"par(sin, mult 2 | sin | mult 0.3)",
			"par(sin, mult 3 | sin | mult m1[0.1,0.7], mult 5 | sin | mult m2[0.5,0.05])",
			"sin | mult m3[1.5,2.5] | mirror -0.8 0.8 | sin",
		},
		Morph: 0.4, Macros: [4]float64{0.5, 0.5, 0.5},
		Unison: 3, Spread: 14, UniStereo: 1, Drift: 0.7,
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Detune: 7, Level: 0.3, Morph: 0.2},
		AmpEnv: env(1600, 900, 0.9, 3200), Width: 0.9, Level: 0.5, LFO1Rate: 1.0 / 12, LFO2Rate: 1.0 / 7,
		Filter: softLow(4200),
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "morph", Amount: 0.45}, modmatrix.Slot{Src: lfo2, Dest: "macro1", Amount: 0.5},
			modmatrix.Slot{Src: random, Dest: "macro2", Amount: 0.4}, modmatrix.Slot{Src: lfo1, Dest: "macro3", Amount: -0.4},
			modmatrix.Slot{Src: lfo2, Dest: "width", Amount: 0.2}),
	}
}

// FourBeasts is four instruments on the keys: the notes pull the morph, so the low notes are a lion (a dark, folded FM
// growl), those above it an ox (a hollow square), the middle a man (a voice with its overtones) and the high ones an eagle
// (a bright ring of partials). Between two notes the instrument is a blend, and a run up the keyboard turns one into the next
func FourBeasts() Config {
	return Config{
		Chains: [NumChains]string{
			pm(2, "0.3") + " | softclip 1.8",
			"pulse 0.5 | bipolar | softclip 1.3",
			"cheby 1:1 2:0.45 3:0.6 4:0.3 5:0.5",
			"par(sin, mult 5 | sin | mult 0.4, mult 8 | sin | mult 0.25)",
		},
		Morph: 0.5, Unison: 2, Spread: 6, Drift: 0.3,
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.3},
		AmpEnv: env(25, 400, 0.75, 350), Level: 0.6, Width: 0.3,
		VibratoRate: 5, VibratoDepth: 0.06, VibratoDelay: 0.4,
		Filter: keyFilter(voicefilter.Ladder12, 2600, 1),
		Mods:   slots(modmatrix.Slot{Src: key, Dest: "morph", Amount: 1}),
	}
}

// TrumpetVoice is a voice as of a trumpet: touched softly it is a hollow breath, and the harder it is played the more the
// wave bends and folds into brass, the swell of the morph taking longer to come than the note (the bwaah). It is the strength
// of the touch that changes the instrument, not only how loud it is
func TrumpetVoice() Config {
	return Config{
		Chains: [NumChains]string{"sin", pm(1, "m1[0.05,0.3]"), "bipolar | softclip 1.5"},
		Morph:  0, MorphAmount: 0.2, MorphEnv: env(140, 300, 0.6, 250), Macros: [4]float64{0.4},
		Unison: 2, Spread: 7, Drift: 0.2,
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.2},
		AmpEnv: env(60, 400, 0.8, 300), Level: 0.6, Width: 0.3,
		VibratoRate: 5.2, VibratoDepth: 0.08, VibratoDelay: 0.6,
		Filter: lowpass(1200, 0.1, 0.8, 90, 300, 0.7, 250),
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "morph", Amount: 0.8}, modmatrix.Slot{Src: vel, Dest: "macro1", Amount: 0.5},
			modmatrix.Slot{Src: vel, Dest: "cutoff", Amount: 0.5}),
	}
}

// SevenLamps is a bell of fire: a sine bends the phase of another at a ratio of seven to three, so its partials fall
// where no harmonic series puts them, the bend dies as the note rings and flickers like a flame (a quick wobble, a
// different one every note)
func SevenLamps() Config {
	return Config{
		Chains: [NumChains]string{"sin"},
		Line1:  Line{Level: 1, Octave: 1, Detune: 265}, Line2: Line{Level: 1}, Width: 0.4, Drift: 0.2,
		XMod: 0.03, XModSilent: true, AmpEnv: env(1, 2800, 0, 1800), Level: 1.1, LFO1Rate: 6.5,
		Mods: slots(modmatrix.Slot{Src: ampEnv, Dest: "xmod", Amount: 0.26}, modmatrix.Slot{Src: lfo1, Dest: "xmod", Amount: 0.02},
			modmatrix.Slot{Src: random, Dest: "detune2", Amount: 0.06}),
	}
}

// CrownsCast is a clatter of gold: two lines a seventh apart and a bit multiplied, then a bright ring of the sums and the
// differences of all their partials, dying in a second as a handful of crowns does on a stone floor
func CrownsCast() Config {
	return Config{
		Chains: [NumChains]string{"par(sin, mult 2 | sin | mult 0.5)"},
		Line1:  Line{Level: 1, Octave: 1, Detune: 983}, Line2: Line{Level: 1, Detune: 7}, XMod: 0.8, XModType: 1, XModSilent: true,
		Width: 0.8, Drift: 0.5, AmpEnv: env(1, 900, 0, 500), Level: 2.3, Filter: softLow(5200),
		Mods: slots(modmatrix.Slot{Src: random, Dest: "detune2", Amount: 0.3}),
	}
}

// Thunder is the voice of the throne in the low register: a sine that falls two octaves, a little fold and the rumble that
// comes with it. Played at C1 it is a kick, a fourth or an octave and a half higher a tom, and a roll of the hardest is a
// storm
func Thunder() Config {
	return Config{
		Chains:      [NumChains]string{"sin | softclip 2"},
		PitchAmount: 24, PitchEnv: env(1, 110, 0, 60), Noise: 0.1, NoiseColor: -0.8,
		Line1: Line{Level: 1}, AmpEnv: env(1, 700, 0, 160), Level: 1.1, Filter: softLow(900),
	}
}

// Lightning is a crack of light: noise through a band that follows the key, so a low note is the snare of a drum kit
// (a body under the crack) and a high one a hat, a tick of a few milliseconds
func Lightning() Config {
	return Config{
		Chains: [NumChains]string{"sin | softclip 2"}, Noise: 0.8, NoiseColor: 0.2,
		PitchAmount: 5, PitchEnv: env(1, 30, 0, 20), Line1: Line{Level: 0.5},
		AmpEnv: env(1, 150, 0, 60), Level: 4,
		Filter: keyFilter(voicefilter.BandPass, 3000, 0.8),
	}
}

// LionBass is the Lion that has prevailed: touched softly a clean sine purr, and the harder the touch the more the
// wave folds, until it roars
func LionBass() Config {
	return Config{
		Chains: [NumChains]string{"sin | softclip 1.2", "sin | mult m1[1.5,3.5] | mirror -0.9 0.9", pm(2, "0.35") + " | softclip 2.2"},
		Morph:  0, Macros: [4]float64{0.6}, Sub: 0.3, Drift: 0.2,
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.4},
		AmpEnv: env(6, 500, 0.85, 200), Level: 0.75, Filter: lowpass(500, 0.25, 1.2, 4, 300, 0.4, 160),
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "morph", Amount: 0.9}, modmatrix.Slot{Src: vel, Dest: "cutoff", Amount: 0.5}),
	}
}

// ThroneDrone is a drone of four partials, the first, the third, the seventh and the twelfth, whose levels breathe against
// one another on two slow waves (twelve seconds and seven), so the harmonics drift apart and together and the colour of
// the one held note changes through a minute
func ThroneDrone() Config {
	return Config{
		Chains: [NumChains]string{"par(sin, mult 3 | sin | mult m1[0.05,0.6], mult 7 | sin | mult m2[0.05,0.5], mult 12 | sin | mult m3[0.02,0.4])"},
		Macros: [4]float64{0.5, 0.5, 0.5}, Unison: 2, Spread: 6, UniStereo: 0.6, Drift: 0.5,
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.6},
		AmpEnv: env(2600, 500, 1, 4200), Level: 0.32, LFO1Rate: 1.0 / 12, LFO2Rate: 1.0 / 7,
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "macro1", Amount: 0.5}, modmatrix.Slot{Src: lfo2, Dest: "macro2", Amount: 0.5},
			modmatrix.Slot{Src: lfo1, Dest: "macro3", Amount: -0.5}, modmatrix.Slot{Src: lfo2, Dest: "macro1", Amount: -0.3}),
	}
}

// ManyEyes is a lead full of eyes: it draws itself anew with every note, so no two notes of a line are the same instrument.
// The morph, the fold and the bend of the phase all take a new value from the note, and then settle
func ManyEyes() Config {
	return Config{
		Chains: [NumChains]string{pm(1, "m1[0.05,0.5]"), "sin | mult m2[1,4] | mirror -0.8 0.8 | sin", "par(sin, mult 2 | sin | mult 0.5, mult 5 | sin | mult m3[0.1,0.6])"},
		Morph:  0.5, Macros: [4]float64{0.5, 0.5, 0.5}, Unison: 2, Spread: 7, Drift: 0.3,
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Detune: 3, Level: 0.25},
		AmpEnv: env(30, 400, 0.8, 380), Level: 0.6, Width: 0.5,
		VibratoRate: 4.8, VibratoDepth: 0.06, VibratoDelay: 0.5,
		Filter: softLow(3800),
		Mods: slots(modmatrix.Slot{Src: random, Dest: "morph", Amount: 0.7}, modmatrix.Slot{Src: random, Dest: "macro1", Amount: 0.5},
			modmatrix.Slot{Src: random, Dest: "macro2", Amount: -0.5}, modmatrix.Slot{Src: random, Dest: "macro3", Amount: 0.4}),
	}
}

// keyFilter is a filter at a fixed cutoff (at middle C) that follows the note by keyTrack
func keyFilter(t voicefilter.Type, cutoff, keyTrack float64) voicefilter.Config {
	return voicefilter.Config{Type: t, Cutoff: cutoff, Resonance: 0.15, KeyTrack: keyTrack, Env: env(1, 100, 1, 100)}
}
