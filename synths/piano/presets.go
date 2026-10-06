package piano

// Presets maps preset names to their constructors
var Presets = map[string]func() Config{
	"grand":     GrandPreset,
	"bright":    BrightPreset,
	"upright":   UprightPreset,
	"felt":      FeltPreset,
	"honkyTonk": HonkyTonkPreset,
	"toy":       ToyPreset,
}

// GrandPreset is a concert grand: round hammers, long sustain, a wide keyboard
func GrandPreset() Config {
	cfg := DefaultConfig()
	cfg.Level = 16.5626 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BrightPreset is a grand voiced for pop and rock: hard hammers that cut through a band, a
// little shorter
func BrightPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 0.8
	cfg.Soundboard = 0.5
	cfg.Decay = 0.85
	cfg.Damping = 0.8
	cfg.DamperTime = 0.2
	cfg.Level = 13.8375 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// UprightPreset is an upright: shorter, stiffer strings that ring less long, a boxier
// soundboard, a little out of tune
func UprightPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 0.6
	cfg.StrikePosition = 0.11
	cfg.Soundboard = 0.8
	cfg.Stiffness = 1.8
	cfg.Decay = 0.55
	cfg.Damping = 1.6
	cfg.Unison = 1.8
	cfg.Width = 0.35
	cfg.Level = 18.2635 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FeltPreset is a felt piano: a strip of felt between hammers and strings, soft, warm and close,
// the sound of quiet piano music and ambient
func FeltPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 0.05
	cfg.StrikePosition = 0.14
	cfg.Soundboard = 0.9
	cfg.Decay = 0.8
	cfg.Damping = 2.5
	cfg.DamperTime = 0.4
	cfg.VelocitySens = 0.5
	cfg.Width = 0.7
	cfg.Level = 22.5738 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HonkyTonkPreset is a saloon upright: the strings of every key far out of tune with each other,
// bright and short
func HonkyTonkPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 0.7
	cfg.Stiffness = 1.5
	cfg.Decay = 0.6
	cfg.Damping = 1.3
	cfg.Unison = 12
	cfg.Width = 0.3
	cfg.Level = 16.7217 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ToyPreset is a toy piano: one very stiff, short string per key (more like a struck rod), hard
// hammers, a tiny body
func ToyPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 0.9
	cfg.StrikePosition = 0.2
	cfg.Soundboard = 0.3
	cfg.Stiffness = 8
	cfg.Decay = 0.2
	cfg.Damping = 0.5
	cfg.Strings = 1
	cfg.DamperTime = 0.1
	cfg.Width = 0.2
	cfg.Level = 24.7193 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
