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
}

// Marimba is a rosewood bar tuned to the fundamental, its fourth and tenth partials, over a
// resonator tube that holds the fundamental
func Marimba() Config {
	return Config{
		Modes:       []Mode{{1, 1, 1.3}, {3.99, 0.35, 1}, {9.2, 0.12, 1}, {15.6, 0.04, 1}},
		Decay:       1.6,
		HighDamping: 0.6,
		Hardness:    0.5,
		Position:    0.2,
		Noise:       0.15,
		Level:       1.1,
	}
}

// Vibraphone is an aluminum bar ringing long, its pedal damping it when let go, and the motor's
// tremolo
func Vibraphone() Config {
	return Config{
		Modes:        []Mode{{1, 1, 1}, {4, 0.3, 0.6}, {10, 0.08, 0.4}},
		Decay:        6,
		HighDamping:  0.3,
		Hardness:     0.4,
		Position:     0.2,
		Release:      0.15,
		Tremolo:      5,
		TremoloDepth: 0.35,
		Noise:        0.05,
		Level:        1.2,
	}
}

// Xylophone is a short, bright bar tuned to the twelfth (the third partial)
func Xylophone() Config {
	return Config{
		Modes:       []Mode{{1, 1, 1}, {3.0, 0.6, 0.8}, {6.0, 0.25, 0.6}, {9.9, 0.1, 0.5}},
		Decay:       0.7,
		HighDamping: 0.5,
		Hardness:    0.8,
		Position:    0.2,
		Noise:       0.3,
		Level:       0.95,
	}
}

// Glockenspiel is a small steel bar, free at both ends: high, clear and long
func Glockenspiel() Config {
	return Config{
		Modes:       []Mode{{1, 1, 1}, {2.756, 0.45, 0.7}, {5.404, 0.25, 0.5}, {8.933, 0.1, 0.4}},
		Decay:       3.5,
		HighDamping: 0.4,
		Hardness:    0.9,
		Position:    0.25,
		Noise:       0.1,
		Level:       0.72,
	}
}

// TubularBells are orchestra chimes: tubes, the note heard from the high modes
func TubularBells() Config {
	return Config{
		Modes:       []Mode{{1, 0.5, 1}, {2.76, 0.8, 1}, {5.40, 1, 0.9}, {8.93, 0.8, 0.8}, {13.34, 0.5, 0.6}, {18.64, 0.3, 0.5}},
		Decay:       7,
		HighDamping: 0.3,
		Hardness:    0.7,
		Position:    0.1,
		Release:     0.6,
		Noise:       0.2,
		Level:       0.44,
	}
}

// ChurchBell is a cast bell: the hum an octave below, the minor third (tierce), the fifth and the
// nominal an octave up, each a beating pair
func ChurchBell() Config {
	return Config{
		Modes: []Mode{{0.5, 0.6, 1.6}, {1, 0.8, 1}, {1.183, 0.6, 0.8}, {1.506, 0.4, 0.7}, {2, 1, 0.6},
			{2.514, 0.4, 0.45}, {2.662, 0.35, 0.4}, {3.011, 0.3, 0.35}, {4.166, 0.2, 0.3}},
		Decay:       9,
		HighDamping: 0.2,
		Hardness:    0.9,
		Position:    0.5,
		Beat:        0.003,
		Noise:       0.2,
		Level:       0.43,
	}
}

// Kalimba is a thumb piano: a steel tine over a box
func Kalimba() Config {
	return Config{
		Modes:       []Mode{{1, 1, 1}, {6.27, 0.25, 0.3}, {17.55, 0.08, 0.2}},
		Decay:       1.8,
		HighDamping: 0.3,
		Hardness:    0.6,
		Position:    0.1,
		Noise:       0.25,
		Level:       1.8,
	}
}

// SingingBowl is a brass bowl struck softly: a few inharmonic modes, beating, ringing very long
func SingingBowl() Config {
	return Config{
		Modes:       []Mode{{1, 1, 1}, {2.71, 0.6, 0.8}, {5.15, 0.3, 0.6}, {8.0, 0.15, 0.5}},
		Decay:       14,
		HighDamping: 0.2,
		Hardness:    0.2,
		Position:    0.4,
		Beat:        0.004,
		Level:       0.86,
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
		Level:       0.84,
	}
}
