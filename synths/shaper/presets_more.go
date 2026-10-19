package shaper

import "github.com/almerlucke/psalm/blocks/modmatrix"

// --- basses ---

// SubSine is a deep sine with a little thump at the start and a touch of odd harmonics, so a small speaker has
// something to play
func SubSine() Config {
	return Config{
		Chains: [NumChains]string{"sin", "sin | softclip 1.8"}, Morph: 0.25, PitchAmount: 7, PitchEnv: env(1, 35, 0, 60),
		Line1: Line{Level: 1}, AmpEnv: env(3, 400, 0.85, 140), Level: 0.575,
	}
}

// KickSub is a kick drum: a sine whose pitch falls three octaves in a flash. Play it low (C1 to C2)
func KickSub() Config {
	return Config{
		Chains: [NumChains]string{"sin | softclip 1.4", "sin | softclip 3"}, Morph: 0, MorphAmount: 0.6, MorphEnv: env(1, 25, 0, 40),
		PitchAmount: 36, PitchEnv: env(1, 55, 0, 40),
		Line1: Line{Level: 1}, AmpEnv: env(1, 330, 0, 60), Level: 0.981,
	}
}

// Reese is two saws a few cents apart, beating slowly, under a low pass that breathes
func Reese() Config {
	return Config{
		Unison: 3, Spread: 12, UniStereo: 0.6, Drift: 0.5, Sub: 0.35,
		Chains: [NumChains]string{"bipolar | softclip 1.15"}, Line1: Line{Level: 1}, Line2: Line{Detune: 21, Level: 1},
		AmpEnv: env(8, 500, 0.9, 200), Width: 0.5, Level: 0.484, LFO1Rate: 0.25,
		Filter: lowpass(650, 0.25, 0.8, 5, 300, 0.7, 200),
		Mods:   slots(modmatrix.Slot{Src: lfo1, Dest: "cutoff", Amount: 0.13}, modmatrix.Slot{Src: lfo1, Dest: "detune2", Amount: 0.15}),
	}
}

// Acid is the squelch: a saw (or a square) into a resonant low pass whose envelope snaps open on every
// note, a harder note opening it further and going square; it plays one note at a time and slides
func Acid() Config {
	return Config{
		Chains: [NumChains]string{"bipolar | softclip 1.1", "pulse 0.5 | bipolar"}, Line1: Line{Level: 1},
		AmpEnv: env(2, 320, 0.8, 90), Level: 1.642, Mono: true, Legato: true, Glide: 70,
		Filter: lowpass(380, 0.62, 3.2, 1, 190, 0, 80),
		Mods:   slots(modmatrix.Slot{Src: vel, Dest: "cutoff", Amount: 0.45}, modmatrix.Slot{Src: vel, Dest: "morph", Amount: 0.6}),
	}
}

// Wobble is a saw and a fold through a low pass that an LFO sweeps up and down, about three times a second
func Wobble() Config {
	return Config{
		Chains: [NumChains]string{"bipolar | softclip 1.2", "sin | mult m1[1.5,4] | mirror -1 1"}, Morph: 0.25, Macros: [4]float64{0.4},
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.6, Detune: 4},
		AmpEnv: env(3, 600, 0.9, 140), Level: 0.7, LFO1Rate: 3, LFO2Rate: 0.4,
		Filter: lowpass(500, 0.35, 0.5, 3, 300, 0.6, 140),
		Mods:   slots(modmatrix.Slot{Src: lfo1, Dest: "cutoff", Amount: 0.3}, modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.4}),
	}
}

// PulseBass is a pulse whose width narrows over the note, through a snapping low pass
func PulseBass() Config {
	return Config{
		Drift:  0.3,
		Chains: [NumChains]string{"pulse m1[0.5,0.12] | bipolar"}, Macros: [4]float64{0.8}, ShapeAmount: -0.7, ShapeEnv: env(2, 380, 0.1, 100),
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.5},
		AmpEnv: env(2, 450, 0.7, 120), Level: 0.625, Filter: lowpass(600, 0.2, 2.2, 1, 240, 0.1, 100),
	}
}

// FMBass is a phase modulated sine with a lot of modulation at the start that falls away, a sine an octave under
func FMBass() Config {
	return Config{
		Chains: [NumChains]string{pm(1, "m1[0.05,0.38]")}, ShapeAmount: 0.9, ShapeEnv: env(1, 380, 0.1, 100), Macros: [4]float64{0.05},
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.55},
		AmpEnv: env(1, 600, 0.55, 140), Level: 0.769,
	}
}

// Growl is FM with a low modulator folded hard, swelling and falling twice a second
func Growl() Config {
	return Config{
		Chains: [NumChains]string{pm(2, "m1[0.1,0.45]") + " | softclip 1.5"}, Macros: [4]float64{0.4},
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.7, Detune: 6},
		AmpEnv: env(4, 600, 0.9, 140), Level: 0.458, LFO2Rate: 2, Filter: lowpass(1200, 0.2, 0.6, 3, 300, 0.6, 140),
		Mods: slots(modmatrix.Slot{Src: lfo2, Dest: "macro1", Amount: 0.45}, modmatrix.Slot{Src: lfo2, Dest: "cutoff", Amount: 0.1}),
	}
}

// --- leads ---

// SupersawLead is the big trance saw: five copies of two saws a few cents apart, spread wide, through a low pass, a vibrato that
// comes in late
func SupersawLead() Config {
	return Config{
		Unison: 5, Spread: 22, UniStereo: 0.9, Drift: 0.3,
		Chains: [NumChains]string{"bipolar", "bipolar | softclip 1.4"}, Morph: 0.2,
		Line1: Line{Level: 1, Detune: -9}, Line2: Line{Detune: 17, Level: 1, Morph: 0.1}, Width: 1,
		AmpEnv: env(6, 400, 0.85, 350), Level: 0.605, VibratoRate: 5.3, VibratoDepth: 0.07, VibratoDelay: 0.45,
		Filter: lowpass(3200, 0.12, 0.9, 4, 500, 0.5, 350),
		Mods:   slots(modmatrix.Slot{Src: vel, Dest: "cutoff", Amount: 0.15}, modmatrix.Slot{Src: modmatrix.Wheel, Dest: "morph", Amount: 0.5}),
	}
}

// SquareLead is a hollow pulse with its width moving, one note at a time, sliding
func SquareLead() Config {
	return Config{
		Unison: 3, Spread: 10, UniStereo: 0.6, Drift: 0.3,
		Chains: [NumChains]string{"pulse m1[0.5,0.2] | bipolar"}, Macros: [4]float64{0.3},
		Line1: Line{Level: 1}, Line2: Line{Detune: 8, Level: 0.6, Octave: 0},
		AmpEnv: env(5, 300, 0.85, 200), Level: 0.342, Mono: true, Legato: true, Glide: 45, Width: 0.5, LFO1Rate: 0.7,
		VibratoRate: 5.5, VibratoDepth: 0.08, VibratoDelay: 0.3, Filter: lowpass(3000, 0.15, 0.5, 4, 300, 0.6, 200),
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "macro1", Amount: 0.45}),
	}
}

// PluckLead is a bright saw that mellows at once, a filter snapping shut: a sequence lead
func PluckLead() Config {
	return Config{
		Unison: 3, Spread: 12, UniStereo: 0.8, Drift: 0.2,
		Chains: [NumChains]string{"tri | bipolar", "bipolar | softclip 2"}, Morph: 0, MorphAmount: 1, MorphEnv: env(1, 200, 0.05, 150),
		Line1: Line{Level: 1}, Line2: Line{Detune: 7, Level: 0.7}, Width: 0.5,
		AmpEnv: env(1, 400, 0.12, 200), Level: 0.589, Filter: lowpass(900, 0.2, 3, 1, 190, 0.05, 150),
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "cutoff", Amount: 0.2}),
	}
}

// FMLead is an electric-organ-bright phase modulated lead: the harder the key the brighter, the wheel adds more
func FMLead() Config {
	return Config{
		Drift:  0.3,
		Chains: [NumChains]string{pm(2, "m1[0.05,0.5]")}, Macros: [4]float64{0.15},
		Line1: Line{Level: 1}, Line2: Line{Detune: 5, Level: 0.5, Octave: 0}, Width: 0.5,
		AmpEnv: env(5, 300, 0.85, 220), Level: 0.49, VibratoRate: 5.4, VibratoDepth: 0.08, VibratoDelay: 0.25,
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "macro1", Amount: 0.65}, modmatrix.Slot{Src: modmatrix.Wheel, Dest: "macro1", Amount: 0.35}),
	}
}

// FluteLead is a nearly pure tone with a breath of the second and third harmonic, slow to speak, with a vibrato
func FluteLead() Config {
	return Config{
		Drift:  0.35,
		Chains: [NumChains]string{"par(sin, mult 2 | sin | mult 0.1, mult 3 | sin | mult 0.04)", "par(sin, mult 2 | sin | mult 0.22, mult 3 | sin | mult 0.1)"},
		Line1:  Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.07, Detune: 3}, Width: 0.2, Mono: true, Legato: true, Glide: 40,
		AmpEnv: env(70, 300, 0.9, 220), Level: 0.474, VibratoRate: 5.1, VibratoDepth: 0.14, VibratoDelay: 0.35,
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "morph", Amount: 0.7}),
	}
}

// BrassLead swells in with its brightness: phase modulation that opens over the first tenth of a second
func BrassLead() Config {
	return Config{
		Unison: 3, Spread: 8, UniStereo: 0.5, Drift: 0.3,
		Chains: [NumChains]string{pm(1, "m1[0.05,0.4]")}, Macros: [4]float64{0.1}, ShapeAmount: 0.8, ShapeEnv: env(90, 400, 0.55, 250),
		Line1: Line{Level: 1}, Line2: Line{Detune: 6, Level: 0.5}, Width: 0.4,
		AmpEnv: env(45, 300, 0.85, 220), Level: 0.478, VibratoRate: 5.2, VibratoDepth: 0.06, VibratoDelay: 0.4,
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "macro1", Amount: 0.25}),
	}
}

// --- pads ---

// WarmPad is detuned saws (three copies each, drifting) behind a low pass that breathes, wide and slow, with a little sub
func WarmPad() Config {
	return Config{
		Unison: 3, Spread: 14, UniStereo: 1, Drift: 0.5, Sub: 0.2,
		Chains: [NumChains]string{"bipolar | softclip 1.1"}, Line1: Line{Level: 1, Detune: -7}, Line2: Line{Detune: 13, Level: 1}, Width: 1,
		AmpEnv: env(1400, 900, 0.9, 1800), Level: 0.529, LFO1Rate: 0.12, Filter: lowpass(1100, 0.12, 0.7, 900, 1000, 0.7, 1500),
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "cutoff", Amount: 0.1}),
	}
}

// StringPad is saw and pulse together, the morph between them drifting
func StringPad() Config {
	return Config{
		Unison: 3, Spread: 12, UniStereo: 0.9, Drift: 0.5,
		Chains: [NumChains]string{"bipolar | softclip 1.2", "pulse 0.4 | bipolar"}, Morph: 0.4,
		Line1: Line{Level: 1, Detune: -6}, Line2: Line{Detune: 11, Level: 1, Morph: 0.15}, Width: 0.9,
		AmpEnv: env(700, 900, 0.9, 1400), Level: 0.578, LFO2Rate: 0.18, VibratoRate: 5.2, VibratoDepth: 0.04, VibratoDelay: 0.8,
		Filter: lowpass(2300, 0.1, 0.6, 600, 900, 0.7, 1400), Mods: slots(modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.35}),
	}
}

// ChoirPad is mouth shapes moving slowly, a few people singing
func ChoirPad() Config {
	return Config{
		Unison: 3, Spread: 10, UniStereo: 0.8, Drift: 0.5,
		Chains: [NumChains]string{
			"par(sin | mult 0.8, mult 2 | sin | mult 0.2, mult 4 | sin | mult 0.8, mult 5 | sin | mult 0.2)",
			"par(sin | mult 0.7, mult 3 | sin | mult 0.2, mult 6 | sin | mult 0.5, mult 9 | sin | mult 0.5)",
			"par(sin, mult 2 | sin | mult 0.9, mult 3 | sin | mult 0.3, mult 8 | sin | mult 0.12)",
		},
		Morph: 0.3, Line1: Line{Level: 1, Detune: -5}, Line2: Line{Detune: 9, Level: 0.9, Morph: 0.12}, Width: 0.8,
		AmpEnv: env(900, 700, 0.9, 1600), Level: 0.338, LFO2Rate: 0.09, VibratoRate: 5.3, VibratoDepth: 0.08, VibratoDelay: 0.5,
		Mods: slots(modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.45}),
	}
}

// AirPad is high, soft and moving, sines and a fold that comes and goes, with a breath of bright noise on top
func AirPad() Config {
	return Config{
		Unison: 3, Spread: 16, UniStereo: 1, Drift: 0.5, Noise: 0.01, NoiseColor: 0.3,
		Chains: [NumChains]string{"par(sin, mult 2 | sin | mult 0.3)", "sin | mult m1[2,4] | mirror -1 1"}, Morph: 0.3, Macros: [4]float64{0.3},
		Line1: Line{Level: 1}, Line2: Line{Detune: 8, Level: 0.8, Octave: 0}, Width: 1,
		AmpEnv: env(1800, 900, 0.9, 2200), Level: 0.379, LFO1Rate: 0.2, LFO2Rate: 0.13,
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "macro1", Amount: 0.4}, modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.35}),
	}
}

// DarkPad is a low pulse behind a closed filter that opens and shuts very slowly
func DarkPad() Config {
	return Config{
		Unison: 3, Spread: 10, UniStereo: 0.8, Drift: 0.5, Sub: 0.25,
		Chains: [NumChains]string{"pulse 0.5 | bipolar"}, Line1: Line{Level: 1}, Line2: Line{Octave: -1, Detune: 9, Level: 0.9}, Width: 0.8,
		AmpEnv: env(1600, 900, 0.9, 2400), Level: 0.364, LFO1Rate: 0.07, Filter: lowpass(520, 0.22, 0.5, 1200, 1200, 0.7, 2000),
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "cutoff", Amount: 0.2}),
	}
}

// Evolve goes through four waves by itself on two slow LFOs that never line up: sine, a fold, a saw, a
// pulse; no two bars sound the same
func Evolve() Config {
	return Config{
		Unison: 3, Spread: 12, UniStereo: 1, Drift: 0.5,
		Chains: [NumChains]string{"sin", "sin | mult m1[1.5,4] | mirror -1 1", "bipolar | softclip 1.2", "pulse m1[0.5,0.2] | bipolar"}, Morph: 0.5, Macros: [4]float64{0.5},
		Line1: Line{Level: 1, Detune: -6}, Line2: Line{Detune: 10, Level: 0.9, Morph: 0.2}, Width: 1,
		AmpEnv: env(1500, 900, 0.9, 2500), Level: 0.418, LFO1Rate: 0.05, LFO2Rate: 0.031,
		Filter: lowpass(2000, 0.15, 0.5, 900, 900, 0.7, 2000),
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "morph", Amount: 0.55}, modmatrix.Slot{Src: lfo2, Dest: "macro1", Amount: 0.5},
			modmatrix.Slot{Src: lfo2, Dest: "cutoff", Amount: 0.1}),
	}
}

// GlassPad is soft phase modulation, the glassy sheen of an old digital pad
func GlassPad() Config {
	return Config{
		Unison: 3, Spread: 8, UniStereo: 0.9, Drift: 0.4,
		Chains: [NumChains]string{pm(3, "m1[0.02,0.25]"), pm(1, "m1[0.02,0.3]")}, Morph: 0.3, Macros: [4]float64{0.4},
		Line1: Line{Level: 1, Detune: -4}, Line2: Line{Detune: 7, Level: 0.9}, Width: 0.9,
		AmpEnv: env(1000, 900, 0.9, 1800), Level: 0.381, LFO1Rate: 0.14, LFO2Rate: 0.08,
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "macro1", Amount: 0.45}, modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.4}),
	}
}

// --- keys and plucks ---

// Harp is a plucked string: all the harmonics at the start, the high ones dying first
func Harp() Config {
	return Config{
		Chains:      [NumChains]string{"sin", "par(sin, mult 2 | sin | mult 0.5, mult 3 | sin | mult 0.3, mult 4 | sin | mult 0.2, mult 5 | sin | mult 0.1)"},
		MorphAmount: 1, MorphEnv: env(1, 500, 0, 200), Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.15, Detune: 2},
		AmpEnv: env(1, 1500, 0.02, 400), Width: 0.4, Level: 0.51,
	}
}

// Kalimba is a thumb piano: a quick ping on top of the note that falls away
func Kalimba() Config {
	return Config{
		Chains: [NumChains]string{pm(4, "m1[0,0.2]")}, Macros: [4]float64{0.15}, ShapeAmount: 0.8, ShapeEnv: env(1, 220, 0, 100),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.2, Detune: 3},
		AmpEnv: env(1, 1100, 0.02, 250), Width: 0.4, Level: 0.721,
		Filter: softLow(6000),
	}
}

// Marimba is wood: a sine with a hard partial at four times its pitch that dies in a moment
func Marimba() Config {
	return Config{
		Chains: [NumChains]string{"sin", "par(sin, mult 4 | sin | mult 0.8)"}, MorphAmount: 1, MorphEnv: env(1, 90, 0, 80),
		Line1: Line{Level: 1}, AmpEnv: env(1, 600, 0.02, 150), Width: 0.3, Level: 0.476,
	}
}

// EPiano is a tine piano: phase modulation, brighter the harder the key, the bright part dying first, a
// second, softer layer ringing a little out of tune
func EPiano() Config {
	return Config{
		Chains: [NumChains]string{pm(1, "m1[0.03,0.3]"), pm(7, "m1[0,0.12]")}, Macros: [4]float64{0.12}, ShapeAmount: 0.6, ShapeEnv: env(1, 700, 0.05, 200),
		Line1: Line{Level: 1}, Line2: Line{Detune: 4, Level: 0.45, Morph: 0.7}, Width: 0.5,
		AmpEnv: env(2, 2200, 0.12, 400), Level: 0.685,
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "macro1", Amount: 0.5}, modmatrix.Slot{Src: modmatrix.Key, Dest: "macro1", Amount: -0.2}),
	}
}

// Clav is a clavinet: a narrow pulse and a filter that snaps shut in a few milliseconds
func Clav() Config {
	return Config{
		Chains: [NumChains]string{"pulse 0.22 | bipolar"}, Line1: Line{Level: 1}, Line2: Line{Detune: 3, Level: 0.4, Octave: 1},
		AmpEnv: env(1, 450, 0.02, 90), Level: 0.531, Width: 0.3, Filter: lowpass(1200, 0.18, 3.4, 1, 90, 0.02, 60),
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "cutoff", Amount: 0.25}),
	}
}

// Vibes is a vibraphone: a sine with a bright partial that fades, and a tremolo
func Vibes() Config {
	return Config{
		Chains: [NumChains]string{"sin", "par(sin, mult 4 | sin | mult 0.2, mult 10 | sin | mult 0.04)"}, MorphAmount: 0.8, MorphEnv: env(1, 300, 0.05, 200),
		Line1: Line{Level: 1}, Line2: Line{Detune: 3, Level: 0.5}, AmpEnv: env(1, 2400, 0.1, 500), Width: 0.5, Level: 0.481, LFO2Rate: 4.6,
		Mods:   slots(modmatrix.Slot{Src: lfo2, Dest: "level", Amount: 0.14}),
		Filter: softLow(6000),
	}
}

// Stab is the chord stab of house: detuned saws, a filter that snaps shut, a short tail
func Stab() Config {
	return Config{
		Unison: 3, Spread: 14, UniStereo: 0.8,
		Chains: [NumChains]string{"bipolar | softclip 1.2"}, Line1: Line{Level: 1, Detune: -8}, Line2: Line{Detune: 12, Level: 1}, Width: 0.9,
		AmpEnv: env(2, 300, 0.12, 140), Level: 0.625, Filter: lowpass(700, 0.2, 2.8, 1, 260, 0.05, 120),
		Mods: slots(modmatrix.Slot{Src: vel, Dest: "cutoff", Amount: 0.2}),
	}
}

// MusicBox is a comb of metal: a sine and a partial at five times its pitch, tinkling
func MusicBox() Config {
	return Config{
		Chains: [NumChains]string{"par(sin, mult 5 | sin | mult 0.25)", "sin"}, MorphAmount: 0.7, MorphEnv: env(1, 800, 0, 300),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.12, Detune: -3}, AmpEnv: env(1, 1800, 0, 500), Width: 0.4, Level: 0.725,
		Filter: softLow(6000),
	}
}

// --- bells ---

// GlassBell is phase modulation at three times the pitch, falling to a pure tone: a struck glass
func GlassBell() Config {
	return Config{
		Chains: [NumChains]string{pm(3, "m1[0,0.35]")}, Macros: [4]float64{0.02}, ShapeAmount: 0.9, ShapeEnv: env(1, 900, 0, 300),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.18, Detune: 6}, AmpEnv: env(1, 2600, 0, 800), Width: 0.5, Level: 0.712,
		Mods:   slots(modmatrix.Slot{Src: vel, Dest: "macro1", Amount: 0.2}),
		Filter: softLow(5000),
	}
}

// ChurchBell is partials that die one after the other: the high ones in a moment, the low ring on
func ChurchBell() Config {
	return Config{
		Chains: [NumChains]string{
			"sin",
			"par(sin, mult 2 | sin | mult 0.9, mult 3 | sin | mult 0.5)",
			"par(sin, mult 2 | sin | mult 0.9, mult 3 | sin | mult 0.7, mult 4 | sin | mult 0.5, mult 5 | sin | mult 0.4, mult 6 | sin | mult 0.3)",
		},
		Morph: 0, MorphAmount: 1, MorphEnv: env(1, 2600, 0.02, 1500), Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.35, Detune: 2},
		AmpEnv: env(1, 5200, 0, 2200), Width: 0.6, Level: 0.328,
		Filter: softLow(5500),
	}
}

// Gong is a swell of a very bright, very low sound that settles over many seconds
func Gong() Config {
	return Config{
		Chains: [NumChains]string{pm(2, "m1[0.02,0.3]"), pm(3, "m1[0.02,0.3]")}, Morph: 0.2, Macros: [4]float64{0.12}, ShapeAmount: 0.8, ShapeEnv: env(300, 2500, 0.1, 1500),
		Line1: Line{Level: 1}, Line2: Line{Detune: 12, Level: 0.7, Morph: 0.6}, AmpEnv: env(60, 6000, 0, 3000), Width: 0.8, Level: 0.636, LFO2Rate: 0.7,
		Mods:   slots(modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.2}),
		Filter: softLow(5000),
	}
}

// --- drones and effects ---

// Drone5 is a note and its fifth, folded softly, breathing: a drone to sit under a piece
func Drone5() Config {
	return Config{
		Drift:  0.6,
		Chains: [NumChains]string{"sin | mult m1[1,2.2] | mirror -1 1", "par(sin, mult 2 | sin | mult 0.5, mult 3 | sin | mult 0.3)"}, Morph: 0.3, Macros: [4]float64{0.3},
		Line1: Line{Level: 1}, Line2: Line{Detune: 702, Level: 0.8, Morph: 0.4, Octave: -1}, Width: 1,
		AmpEnv: env(3000, 900, 0.95, 4000), Level: 0.36, LFO1Rate: 0.06, LFO2Rate: 0.043,
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "macro1", Amount: 0.4}, modmatrix.Slot{Src: lfo2, Dest: "morph", Amount: 0.4}),
	}
}

// SubDrone is a very low sine folded slowly into a growl and back
func SubDrone() Config {
	return Config{
		Chains: [NumChains]string{"sin | mult m1[1,3] | mirror -1 1 | softclip 1.2"}, Macros: [4]float64{0.2},
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.2, Detune: 5}, AmpEnv: env(2500, 900, 0.95, 3000), Level: 0.377, LFO1Rate: 0.08, Width: 0.5,
		Mods: slots(modmatrix.Slot{Src: lfo1, Dest: "macro1", Amount: 0.5}),
	}
}

// Laser falls from far above the note in a moment, a hard synced saw
func Laser() Config {
	return Config{
		Chains: [NumChains]string{"mult m1[1,5] | mod 1 | bipolar"}, Macros: [4]float64{0.2}, ShapeAmount: 0.8, ShapeEnv: env(1, 200, 0, 60),
		PitchAmount: 48, PitchEnv: env(1, 150, 0, 60), Line1: Line{Level: 1},
		AmpEnv: env(1, 300, 0, 80), Level: 1.44, Filter: lowpass(1500, 0.2, 0.6, 1, 150, 0.2, 60),
	}
}

// Riser climbs for a long note: pitch, brightness and fold all going up, for the bars before a change
func Riser() Config {
	return Config{
		Chains: [NumChains]string{"sin | mult m1[1,6] | mirror -1 1", "bipolar | softclip 1.5"}, Macros: [4]float64{0}, ShapeAmount: 1, ShapeEnv: env(6000, 500, 1, 400),
		MorphAmount: 0.8, MorphEnv: env(6000, 500, 1, 400), PitchAmount: 14, PitchEnv: env(6000, 500, 1, 400),
		Line1: Line{Level: 1}, Line2: Line{Detune: 14, Level: 0.8}, Width: 0.9,
		AmpEnv: env(5000, 400, 1, 600), Level: 0.654, Filter: lowpass(500, 0.25, 3.5, 6000, 500, 1, 400),
	}
}

// Impact is a low hit: a fold falling from high to low and a sine thump under it, ringing out
func Impact() Config {
	return Config{
		Chains: [NumChains]string{"sin | mult m1[1,5] | mirror -1 1 | softclip 1.3"}, Macros: [4]float64{0.1}, ShapeAmount: 0.9, ShapeEnv: env(1, 700, 0, 400),
		PitchAmount: 18, PitchEnv: env(1, 420, 0, 200), Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.8},
		AmpEnv: env(1, 2800, 0, 900), Level: 0.75, Width: 0.5, Filter: lowpass(1800, 0.2, 1, 1, 600, 0.1, 300),
	}
}

// Zap is a short fall in pitch of an FM tone: a game sound
func Zap() Config {
	return Config{
		Chains: [NumChains]string{pm(3, "m1[0,0.5]")}, Macros: [4]float64{0.4}, ShapeAmount: -0.3, ShapeEnv: env(1, 160, 0, 60),
		PitchAmount: 24, PitchEnv: env(1, 120, 0, 50), Line1: Line{Level: 1}, AmpEnv: env(1, 260, 0, 80), Level: 0.823, Width: 0.3,
	}
}
