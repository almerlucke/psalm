package shaper

import (
	"fmt"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	// basses
	"subSine": SubSine, "kickSub": KickSub, "reese": Reese, "acid": Acid, "wobble": Wobble, "pulseBass": PulseBass,
	"fmBass": FMBass, "growl": Growl, "subFold": SubFold,
	// leads
	"supersawLead": SupersawLead, "squareLead": SquareLead, "syncLead": SyncLead, "pluckLead": PluckLead, "fmLead": FMLead,
	"fluteLead": FluteLead, "brassLead": BrassLead, "vowel": Vowel, "reed": Reed,
	// pads
	"warmPad": WarmPad, "stringPad": StringPad, "choirPad": ChoirPad, "airPad": AirPad, "darkPad": DarkPad,
	"evolve": Evolve, "glassPad": GlassPad, "foldPad": FoldPad, "morphPad": MorphPad, "swarm": Swarm, "organ": Organ,
	// keys and plucks
	"pluck": Pluck, "harp": Harp, "kalimba": Kalimba, "marimba": Marimba, "ePiano": EPiano, "clav": Clav, "vibes": Vibes,
	"stab": Stab, "musicBox": MusicBox, "pwm": PWM,
	// bells
	"bellMorph": BellMorph, "glassBell": GlassBell, "churchBell": ChurchBell, "gong": Gong, "metal": Metal, "glass": Glass,
	// drones and textures
	"drone5": Drone5, "subDrone": SubDrone, "crush": Crush,
	// effects
	"laser": Laser, "riser": Riser, "impact": Impact, "zap": Zap,
}

// env is an envelope: attack to 1, decay to a level, release (milliseconds)
func env(attack, decay, sustain, release float64) adsr.Setting {
	return *adsr.NewSetting(1, attack, sustain, decay, 1, release)
}

// lowpass is a steep (24 dB) low pass with an envelope
func lowpass(cutoff, resonance, envAmount, attack, decay, sustain, release float64) voicefilter.Config {
	return voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: cutoff, Resonance: resonance, EnvAmount: envAmount,
		KeyTrack: 0.5, Env: env(attack, decay, sustain, release)}
}

// softLow is a low pass at a fixed cutoff, 24 dB, that takes the edge off the top of a bell or a bright stack of
// partials without moving with the note
func softLow(cutoff float64) voicefilter.Config {
	return voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: cutoff, Resonance: 0.05, EnvAmount: 0, KeyTrack: 0, Env: env(1, 100, 1, 100)}
}

// pm is a chain of phase modulation (FM): a sine at the pitch modulated by a sine at ratio times it, the
// index (a number, or a macro like m1[0,0.5]) in turns: 0.16 is one radian
func pm(ratio int, index string) string {
	return fmt.Sprintf("par(thru, mult %d | sin | mult %s) | sin", ratio, index)
}

// slots fills the first slots of a matrix
func slots(s ...modmatrix.Slot) (m [modmatrix.Slots]modmatrix.Slot) {
	copy(m[:], s)
	return
}

var (
	lfo1 = modmatrix.LFO1
	lfo2 = modmatrix.LFO2
	vel  = modmatrix.Velocity
)

// Glass is a sine that grows overtones as the note starts, folding into a glassy ring
func Glass() Config {
	return Config{
		Chains: [NumChains]string{"sin", "par(sin, mult 3 | sin | mult 0.4)", "sin | mult 3 | mirror -1 1"},
		Morph:  0, MorphAmount: 0.8, MorphEnv: env(2, 900, 0.1, 400),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Detune: 4, Level: 0.3, Morph: -0.2},
		AmpEnv: env(2, 1800, 0.35, 700), Width: 0.4, Level: 0.589,
		VibratoRate: 5, VibratoDepth: 0.05, VibratoDelay: 0.5,
	}
}

// FoldPad is a sine folded back on itself more and more, slowly swelling and fading
func FoldPad() Config {
	return Config{
		Chains: [NumChains]string{"sin", "sin | mult 2 | mirror -1 1", "sin | mult m1[2,6] | mirror -1 1"},
		Morph:  0.35, Macros: [4]float64{0.3},
		Line1: Line{Level: 1}, Line2: Line{Detune: 11, Level: 1, Morph: 0.1},
		AmpEnv: env(900, 600, 0.9, 1400), Width: 0.8, Level: 0.53,
		LFO2Rate: 0.17, LFO1Rate: 0.31,
		Filter: lowpass(3200, 0.15, 0.5, 700, 800, 0.6, 1200),
		Mods: [modmatrix.Slots]modmatrix.Slot{
			{Src: modmatrix.LFO2, Dest: "morph", Amount: 0.35},
			{Src: modmatrix.LFO1, Dest: "macro1", Amount: 0.3},
		},
	}
}

// SyncLead is a saw hard synced to a higher pitch that sweeps down as the note goes on
func SyncLead() Config {
	return Config{
		Chains: [NumChains]string{"mult m1[1,6] | mod 1 | bipolar"},
		Macros: [4]float64{0}, ShapeAmount: 1, ShapeEnv: env(3, 500, 0.15, 300),
		Line1: Line{Level: 1}, Line2: Line{Detune: 7, Level: 0.6},
		AmpEnv: env(4, 400, 0.8, 250), Width: 0.3, Level: 0.643,
		VibratoRate: 5.5, VibratoDepth: 0.1, VibratoDelay: 0.3,
		Filter: lowpass(5200, 0.1, 0.4, 3, 500, 0.5, 250),
		Mods:   [modmatrix.Slots]modmatrix.Slot{{Src: modmatrix.Velocity, Dest: "macro1", Amount: 0.3}},
	}
}

// PWM is a pulse whose width moves on an LFO, two lines apart
func PWM() Config {
	return Config{
		Chains: [NumChains]string{"pulse m1[0.4,0.06] | bipolar"},
		Macros: [4]float64{0.4},
		Line1:  Line{Level: 1, Detune: -9}, Line2: Line{Detune: 12, Level: 1, Octave: 1},
		AmpEnv: env(15, 500, 0.8, 350), Width: 0.9, Level: 0.339,
		LFO1Rate: 0.7, LFO2Rate: 0.53,
		Filter: lowpass(2600, 0.18, 0.3, 40, 500, 0.6, 350),
		Mods: [modmatrix.Slots]modmatrix.Slot{
			{Src: modmatrix.LFO1, Dest: "macro1", Amount: 0.4},
			{Src: modmatrix.LFO2, Dest: "detune2", Amount: 0.1},
		},
	}
}

// Reed goes from a soft triangle to a bright squeezed saw with how hard the key is struck
func Reed() Config {
	return Config{
		Chains: [NumChains]string{"tri | bipolar | softclip 1.5", "bipolar | softclip 3 | mult 0.9"},
		Morph:  0.1, Line1: Line{Level: 1}, Line2: Line{Octave: -1, Detune: 5, Level: 0.4},
		AmpEnv: env(25, 300, 0.85, 200), Width: 0.25, Level: 0.574,
		VibratoRate: 5, VibratoDepth: 0.07, VibratoDelay: 0.4,
		Filter: lowpass(4200, 0.1, 0.5, 20, 400, 0.6, 200),
		Mods: [modmatrix.Slots]modmatrix.Slot{
			{Src: modmatrix.Velocity, Dest: "morph", Amount: 0.9},
			{Src: modmatrix.Wheel, Dest: "morph", Amount: 0.4},
		},
	}
}

// MorphPad goes through four waves, sine, triangle, saw, square, and back, on a slow LFO
func MorphPad() Config {
	return Config{
		Chains: [NumChains]string{"sin", "tri | bipolar", "bipolar", "pulse 0.5 | bipolar"},
		Morph:  0.5, Line1: Line{Level: 1}, Line2: Line{Detune: 9, Level: 0.8, Morph: 0.15},
		AmpEnv: env(1100, 700, 0.9, 1600), Width: 0.9, Level: 0.426, LFO2Rate: 0.11, LFO1Rate: 0.07,
		Filter: lowpass(2400, 0.12, 0.7, 900, 800, 0.7, 1400),
		Mods: [modmatrix.Slots]modmatrix.Slot{
			{Src: modmatrix.LFO2, Dest: "morph", Amount: 0.5},
			{Src: modmatrix.LFO1, Dest: "cutoff", Amount: 0.25},
		},
	}
}

// Metal stacks whole number partials, brightening and fading: a struck pipe
func Metal() Config {
	return Config{
		Chains: [NumChains]string{
			"par(sin, mult 3 | sin | mult 0.2)",
			"par(sin, mult 3 | sin | mult 0.6, mult 7 | sin | mult 0.4)",
			"par(sin | mult 0.6, mult 5 | sin | mult 0.5, mult 9 | sin | mult 0.5, mult 13 | sin | mult 0.3)",
		},
		Morph: 0.2, MorphAmount: 0.7, MorphEnv: env(1, 1400, 0, 500),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Detune: 3, Level: 0.25},
		AmpEnv: env(1, 2600, 0, 900), Width: 0.5, Level: 0.483,
		Filter: softLow(4500),
	}
}

// Organ is drawbars as a sum of sines, the upper ones joining in with the morph
func Organ() Config {
	return Config{
		Chains: [NumChains]string{
			"par(sin, mult 2 | sin | mult 0.5)",
			"par(sin, mult 2 | sin | mult 0.5, mult 3 | sin | mult 0.4, mult 4 | sin | mult 0.3)",
			"par(sin, mult 2 | sin | mult 0.8, mult 3 | sin | mult 0.7, mult 4 | sin | mult 0.6, mult 6 | sin | mult 0.5, mult 8 | sin | mult 0.4)",
		},
		Morph: 0.4, Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.5},
		AmpEnv: env(6, 100, 1, 90), Width: 0.2, Level: 0.306,
		VibratoRate: 6.3, VibratoDepth: 0.04, VibratoDelay: 0.1,
		Mods: [modmatrix.Slots]modmatrix.Slot{{Src: modmatrix.Wheel, Dest: "morph", Amount: 0.6}},
	}
}

// SubFold is a deep sine folded into a growl by the filter's envelope, which here is the shape's
func SubFold() Config {
	return Config{
		Chains: [NumChains]string{"sin", "sin | mult m1[1.5,5] | mirror -1 1 | softclip 1.3"},
		Morph:  0.5, Macros: [4]float64{0.4}, ShapeAmount: 0.6, ShapeEnv: env(2, 300, 0.1, 120),
		MorphAmount: 0.4, MorphEnv: env(2, 250, 0.2, 120),
		Line1: Line{Level: 1}, Line2: Line{Octave: -1, Level: 0.8},
		AmpEnv: env(3, 600, 0.75, 120), Level: 1.237,
		Filter: lowpass(900, 0.25, 1.5, 2, 300, 0.2, 150),
	}
}

// Crush is a saw cut into steps, fewer of them as the macro goes up
func Crush() Config {
	return Config{
		Chains: [NumChains]string{"bipolar | quantize m1[0.5,0.04]"},
		Macros: [4]float64{0.5}, ShapeAmount: -0.5, ShapeEnv: env(2, 700, 0.5, 200),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.35, Detune: 2},
		AmpEnv: env(3, 350, 0.7, 180), Width: 0.3, Level: 0.812,
		Filter: lowpass(4800, 0.1, 0.3, 3, 300, 0.5, 180),
	}
}

// Pluck is a bright start that falls to a mellow tone at once
func Pluck() Config {
	return Config{
		Chains: [NumChains]string{"sin | mult 1.4 | softclip 1.5", "bipolar | softclip 2", "par(bipolar, mult 3 | bipolar | mult 0.5) | softclip 1.5"},
		Morph:  0, MorphAmount: 1, MorphEnv: env(1, 260, 0.05, 200),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Detune: 6, Level: 0.3},
		AmpEnv: env(1, 1100, 0.04, 260), Width: 0.4, Level: 0.383,
		Mods: [modmatrix.Slots]modmatrix.Slot{{Src: modmatrix.Velocity, Dest: "morph", Amount: 0.4}},
	}
}

// Vowel moves between mouth shapes made of a few partials, no filter needed
func Vowel() Config {
	return Config{
		Chains: [NumChains]string{
			"par(sin | mult 1, mult 2 | sin | mult 0.9, mult 3 | sin | mult 0.3, mult 8 | sin | mult 0.12)",
			"par(sin | mult 0.8, mult 2 | sin | mult 0.2, mult 4 | sin | mult 0.8, mult 5 | sin | mult 0.2)",
			"par(sin | mult 0.7, mult 3 | sin | mult 0.2, mult 6 | sin | mult 0.5, mult 9 | sin | mult 0.5)",
		},
		Morph: 0, Line1: Line{Level: 1}, Line2: Line{Detune: 6, Level: 0.6, Morph: 0.05},
		AmpEnv: env(60, 400, 0.85, 300), Width: 0.4, Level: 0.301, LFO1Rate: 0.4,
		VibratoRate: 5.2, VibratoDepth: 0.08, VibratoDelay: 0.35,
		Mods: [modmatrix.Slots]modmatrix.Slot{
			{Src: modmatrix.LFO1, Dest: "morph", Amount: 0.5},
			{Src: modmatrix.Key, Dest: "morph", Amount: 0.2},
		},
	}
}

// BellMorph strikes a bell and lets it settle into a plain sine
func BellMorph() Config {
	return Config{
		Chains: [NumChains]string{"sin", "par(sin, mult 2 | sin | mult 0.6, mult 5 | sin | mult 0.5)", "par(sin, mult 3 | sin | mult 0.9, mult 4 | sin | mult 0.6, mult 7 | sin | mult 0.5)"},
		Morph:  0, MorphAmount: 1, MorphEnv: env(1, 1600, 0.05, 600),
		Line1: Line{Level: 1}, Line2: Line{Octave: 1, Level: 0.2, Detune: -3},
		AmpEnv: env(1, 3000, 0.0, 1100), Width: 0.5, Level: 0.351,
		Mods:   [modmatrix.Slots]modmatrix.Slot{{Src: modmatrix.Velocity, Dest: "morph", Amount: 0.3}},
		Filter: softLow(5000),
	}
}

// Swarm is a saw folded and detuned wide, restless on two LFOs
func Swarm() Config {
	return Config{
		Chains: [NumChains]string{"bipolar | mult m1[1,2.5] | mirror -1 1", "bipolar | mult m1[1.5,4] | mirror -0.8 0.8 | mult 1.2"},
		Morph:  0.4, Macros: [4]float64{0.3},
		Line1: Line{Level: 1}, Line2: Line{Detune: 13, Level: 1, Morph: 0.2},
		AmpEnv: env(500, 600, 0.85, 1200), Width: 1, Level: 0.659, LFO1Rate: 0.37, LFO2Rate: 0.23,
		Filter: lowpass(3000, 0.18, 0.6, 500, 700, 0.6, 1100),
		Mods: [modmatrix.Slots]modmatrix.Slot{
			{Src: modmatrix.LFO1, Dest: "macro1", Amount: 0.5},
			{Src: modmatrix.LFO2, Dest: "morph", Amount: 0.4},
		},
	}
}
