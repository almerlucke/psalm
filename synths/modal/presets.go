package modal

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"marimba":      Marimba,
	"vibraphone":   Vibraphone,
	"xylophone":    Xylophone,
	"glockenspiel": Glockenspiel,
	"tubularBells": TubularBells,
	"churchBell":   ChurchBell,
	"kalimba":      Kalimba,
	"singingBowl":  SingingBowl,
	"woodBlock":    WoodBlock,

	"alienBell":  AlienBell,
	"gamelan":    Gamelan,
	"crystal":    Crystal,
	"steelPan":   SteelPan,
	"handbell":   Handbell,
	"gong":       Gong,
	"cymbal":     Cymbal,
	"celesta":    Celesta,
	"metalPlate": MetalPlate,
	"spaceTine":  SpaceTine,

	"wood":  Wood,
	"metal": Metal,
	"glass": Glass,
}

// Marimba is a rosewood bar undercut and tuned to the fundamental, its fourth and its tenth partial
// (1 : 4 : 10), over a resonator tube that holds the fundamental; struck near the middle, a little off
// the node of the second mode
func Marimba() Config {
	return Config{
		Instrument: InstMarimbaBar,
		Decay:      1.6,
		Resonator:  0.6,
		Hardness:   0.5,
		Position:   0.4,
		Contact:    0.4,
		Mass:       0.3,
		Noise:      0.15,
		Level:      0.506,
	}
}

// Vibraphone is an aluminum bar tuned 1 : 4 : 10 ringing long over its resonator tubes, a pedal damping
// it when let go, and the motor's fans swinging the tubes: the tremolo
func Vibraphone() Config {
	return Config{
		Instrument:   InstVibraphoneBar,
		Decay:        6,
		Resonator:    0.7,
		Hardness:     0.4,
		Position:     0.4,
		Contact:      0.3,
		Release:      0.15,
		Tremolo:      5,
		TremoloDepth: 0.6,
		Noise:        0.05,
		Level:        0.405,
	}
}

// Xylophone is a short, bright bar tuned to the twelfth and the double octave plus a fifth (1 : 3 : 6)
func Xylophone() Config {
	return Config{
		Instrument: InstXylophoneBar,
		Decay:      0.7,
		Hardness:   0.8,
		Position:   0.4,
		Contact:    0.4,
		Noise:      0.3,
		Level:      0.806,
	}
}

// Glockenspiel is a small steel bar, free at both ends: the partials of a uniform bar, high, clear and long
func Glockenspiel() Config {
	return Config{
		Instrument: InstGlockenspielBar,
		Decay:      3.5,
		Hardness:   0.9,
		Position:   0.35,
		Contact:    0.3,
		Noise:      0.1,
		Level:      0.613,
	}
}

// TubularBells are orchestra chimes: a tube struck on its cap, the modes 4 to 8 at 2 : 3 : 4 : 5 : 6 of a
// strike tone that does not sound itself, the lower modes under it
func TubularBells() Config {
	return Config{
		Instrument: InstTubularBell,
		Decay:      7,
		Hardness:   0.7,
		Position:   0.04,
		Contact:    0.3,
		Release:    0.6,
		Noise:      0.2,
		Level:      0.316,
	}
}

// ChurchBell is a cast bell: the hum an octave below, the minor third (tierce), the fifth and the
// nominal an octave up, each a degenerate pair that beats, struck at the sound bow
func ChurchBell() Config {
	return Config{
		Instrument: InstChurchBell,
		Decay:      9,
		Hardness:   0.9,
		Position:   0.9,
		Spread:     1,
		Contact:    0.3,
		Noise:      0.2,
		Level:      0.267,
	}
}

// Kalimba is a thumb piano: a steel tine clamped at one end (1 : 6.27 : 17.5), plucked near the tip
func Kalimba() Config {
	return Config{
		Instrument: InstTineBar,
		Decay:      1.8,
		Hardness:   0.6,
		Position:   0.9,
		Contact:    0.3,
		Noise:      0.25,
		Level:      0.523,
	}
}

// SingingBowl is a brass bowl struck softly at the rim: the bending modes of a shell, each a pair that
// beats, ringing very long
func SingingBowl() Config {
	return Config{
		Instrument: InstSingingBowl,
		Decay:      14,
		Hardness:   0.2,
		Position:   0.9,
		Spread:     1,
		Mass:       0.5,
		Level:      0.733,
	}
}

// WoodBlock is a hollow block of wood: a short knock with a pitch
func WoodBlock() Config {
	return Config{
		Modes:       []Mode{{1, 1, 1}, {1.58, 0.5, 0.8}, {2.47, 0.3, 0.6}, {3.9, 0.15, 0.5}},
		Decay:       0.12,
		HighDamping: 0.5,
		Hardness:    0.8,
		Position:    0.3,
		Noise:       0.4,
		Level:       0.881,
	}
}

// AlienBell rings at ratios no bell has, each mode a beating pair: a bell from somewhere else
func AlienBell() Config {
	return Config{
		Modes: []Mode{{1, 1, 1}, {1.37, 0.7, 0.9}, {2.13, 0.6, 0.8}, {2.91, 0.5, 0.7}, {4.07, 0.35, 0.5},
			{5.33, 0.25, 0.4}, {7.11, 0.15, 0.3}},
		Decay: 6, HighDamping: 0.2, Hardness: 0.85, Position: 0.37, Beat: 0.006, Noise: 0.1, Level: 0.457,
	}
}

// Gamelan is a bronze metallophone; its pairs beat a few times a second, the shimmer (ombak) of a
// gamelan tuned in pairs
func Gamelan() Config {
	return Config{
		Modes: []Mode{{1, 1, 1}, {2.39, 0.45, 0.6}, {4.37, 0.25, 0.4}, {6.85, 0.12, 0.3}},
		Decay: 4.5, HighDamping: 0.3, Hardness: 0.75, Position: 0.3, Beat: 0.012, Noise: 0.15, Level: 0.862,
	}
}

// Crystal is a glassy cluster of high, long modes, struck hard, shimmering with a gentle tremolo
func Crystal() Config {
	return Config{
		Modes: []Mode{{1, 1, 1}, {2.02, 0.7, 1}, {3.03, 0.5, 0.9}, {4.21, 0.4, 0.8}, {5.43, 0.3, 0.7},
			{6.82, 0.25, 0.6}, {8.37, 0.15, 0.5}},
		Decay: 7, HighDamping: 0.05, Hardness: 1, Position: 0.13, Tremolo: 6.5, TremoloDepth: 0.2, Beat: 0.002,
		Level: 0.550,
	}
}

// SteelPan rings in the tuned harmonics of a steel drum's note area (fundamental, octave, twelfth),
// shaped like a drum head, a little out of tune with itself
func SteelPan() Config {
	return Config{
		Instrument: InstSteelPan,
		Decay:      2,
		Hardness:   0.55,
		Position:   0.3,
		Beat:       0.003,
		Noise:      0.2,
		Level:      0.507,
	}
}

// MetalPlate is a sheet of steel: a circular plate with a free edge, its Bessel modes split in pairs, a
// hard hit, a long industrial ring
func MetalPlate() Config {
	return Config{
		Instrument: InstPlate,
		Decay:      5,
		Hardness:   0.95,
		Position:   0.6,
		Spread:     1,
		Contact:    0.3,
		Noise:      0.4,
		Level:      0.207,
	}
}

// SpaceTine is a thumb piano's tine ringing for ages, wobbling in a fast tremolo
func SpaceTine() Config {
	return Config{
		Instrument:   InstTineBar,
		Decay:        9,
		HighDamping:  0.2,
		Hardness:     0.7,
		Position:     0.9,
		Tremolo:      11,
		TremoloDepth: 0.5,
		Noise:        0.2,
		Level:        0.519,
	}
}

// Wood is a plain bar of hardwood: a dull, short knock with a uniform bar's modes, no stretch
func Wood() Config {
	return Config{
		Instrument:  InstBar,
		Decay:       0.5,
		HighDamping: 0.9,
		Hardness:    0.45,
		Position:    0.4,
		Noise:       0.3,
		Level:       0.831,
	}
}

// Metal is a stiff steel bar: long, bright, its upper modes stretched sharp
func Metal() Config {
	return Config{
		Instrument:    InstBar,
		Decay:         5,
		HighDamping:   0.1,
		Hardness:      0.85,
		Position:      0.4,
		Inharmonicity: 0.45,
		Beat:          0.002,
		Noise:         0.15,
		Level:         0.609,
	}
}

// Glass is a glass bar or bowl rim: pure, high, a little stretched and shimmering
func Glass() Config {
	return Config{
		Modes:         []Mode{{1, 1, 1}, {2.32, 0.55, 0.9}, {4.25, 0.35, 0.8}, {6.63, 0.2, 0.6}},
		Decay:         4,
		HighDamping:   0.1,
		Hardness:      0.95,
		Position:      0.3,
		Inharmonicity: 0.2,
		Beat:          0.004,
		Noise:         0.05,
		Level:         0.722,
	}
}

// Handbell is a thin brass bell, tuned close to the harmonic series, ringing a few seconds
func Handbell() Config {
	return Config{
		Instrument: InstHandbell,
		Decay:      3,
		Hardness:   0.85,
		Position:   0.9,
		Spread:     1,
		Contact:    0.3,
		Noise:      0.15,
		Level:      0.399,
	}
}

// Gong is a tam-tam: the modes of a disc as the zeros of the Bessel derivatives, struck near the centre
func Gong() Config {
	return Config{
		Instrument: InstGong,
		Decay:      9,
		Hardness:   0.4,
		Position:   0.35,
		Spread:     1,
		Mass:       0.8,
		Noise:      0.1,
		Level:      0.436,
	}
}

// Cymbal is a dense wash of plate modes that dies from the top
func Cymbal() Config {
	return Config{
		Instrument: InstCymbal,
		Decay:      2.5,
		Hardness:   1,
		Position:   0.7,
		Spread:     1,
		Contact:    0.5,
		Noise:      0.3,
		Level:      0.107,
	}
}

// Celesta is a soft steel bar struck by a felt hammer: a glockenspiel's modes, mellow
func Celesta() Config {
	return Config{
		Instrument: InstGlockenspielBar,
		Decay:      2.5,
		Hardness:   0.45,
		Position:   0.35,
		Contact:    0.4,
		Mass:       0.2,
		Noise:      0.05,
		Level:      0.719,
	}
}
