package westcoast

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"bongo":   Bongo,
	"pluck":   Pluck,
	"marimba": Marimba,
	"bell":    Bell,
	"bass":    Bass,
	"drone":   Drone,
	"metal":   Metal,
	"wobble":  Wobble,
}

// Bongo is the gate struck with its resonance up: a short sine that rings and dies like a hand drum
func Bongo() Config {
	return Config{Decay: 0.02, Gate: Combo, Resonance: 0.75, Response: 0.12, Color: 0.6, Level: 9}
}

// Pluck is a folded sine struck short: bright at the start, rounding off as the gate closes
func Pluck() Config {
	return Config{Fold: 0.15, EnvToFold: 0.5, Decay: 0.08, Gate: Combo, Resonance: 0.2, Response: 0.25, Level: 1.5}
}

// Marimba is a sine with a little FM an octave and a fifth up, struck soft
func Marimba() Config {
	return Config{ModRatio: 3, FM: 0.2, EnvToFM: 0.8, Decay: 0.05, Gate: Combo, Resonance: 0.4, Response: 0.3,
		Color: 0.7, Level: 1.2}
}

// Bell rings an inharmonic FM, long
func Bell() Config {
	return Config{ModRatio: 3.5, FM: 0.6, EnvToFM: 1.5, Decay: 0.3, Gate: Combo, Response: 1.6, Color: 0.9, Level: 1}
}

// Bass is a folded sine with a sub modulator, held
func Bass() Config {
	return Config{ModRatio: 0.5, FM: 0.4, Fold: 0.3, EnvToFold: 0.3, Attack: 0.004, Decay: 0.25, Sustain: true,
		Gate: Combo, Resonance: 0.1, Response: 0.12, Color: 0.65, Level: 0.25}
}

// Drone is held: a slow triangle folds the wave over and back
func Drone() Config {
	return Config{ModWave: ModTriangle, ModRatio: 1.003, FM: 0.6, Fold: 0.4, Symmetry: 0.2, Attack: 0.8, Decay: 1.5,
		Sustain: true, Gate: Combo, Response: 0.6, Level: 0.37}
}

// Metal is a square modulator at a clashing ratio, folded hard
func Metal() Config {
	return Config{ModWave: ModSquare, ModRatio: 1.414, FM: 1.2, Fold: 0.5, EnvToFold: 0.4, Symmetry: 0.3, Decay: 0.15,
		Gate: Combo, Resonance: 0.3, Response: 0.4, Level: 0.8}
}

// Wobble is amplitude modulated by a saw just off the note, held
func Wobble() Config {
	return Config{ModWave: ModSaw, ModRatio: 0.502, AM: 0.8, FM: 0.3, Fold: 0.25, Attack: 0.01, Decay: 0.3,
		Sustain: true, Gate: Combo, Response: 0.2, Level: 0.33}
}
