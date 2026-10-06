package piano

// Presets maps preset names to their constructors
var Presets = map[string]func() Config{
	"grand":     GrandPreset,
	"bright":    BrightPreset,
	"upright":   UprightPreset,
	"felt":      FeltPreset,
	"honkyTonk": HonkyTonkPreset,
	"toy":       ToyPreset,
	"concert":   ConcertPreset,
	"warm":      WarmPreset,
	"baby":      BabyPreset,
	"vintage":   VintagePreset,
	"cinematic": CinematicPreset,
}

// GrandPreset is a concert grand as it is usually recorded, from close microphones: warm hammers,
// a lot of soundboard and wood, strings that beat a little, an instrument with a character
func GrandPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 0.42
	cfg.Soundboard = 0.75
	cfg.Body = 0.7
	cfg.Damping = 1.4
	cfg.Unison = 1.5
	cfg.Variation = 0.7
	cfg.Resonance = 0.6
	cfg.Mechanics = 0.6
	cfg.Position = Studio
	cfg.Level = 18.2470 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
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
	cfg.Level = 15.4649 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
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
	cfg.Stretch = 0.7
	cfg.Level = 19.3681 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FeltPreset is a felt piano: a strip of felt between hammers and strings, soft, warm and close
// (heard from close microphones), the sound of quiet piano music and ambient
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
	cfg.Position = Studio // felt pianos are recorded close
	cfg.Level = 26.0961   // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
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
	cfg.Stretch = 0.6
	cfg.Level = 18.4821 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
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
	cfg.Stretch = 0
	cfg.Level = 27.2504 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ConcertPreset is the concert grand from the audience: the lid throwing the sound out into the
// hall, hammers voiced a little brighter to carry, the dampers felt less
func ConcertPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.55
	cfg.Damping = 1.1
	cfg.Mechanics = 0.3
	cfg.Position = Hall
	cfg.Level = 19.7358 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WarmPreset is a grand in a jazz club: close, round hammers, a rich body, the strings ringing
// along freely
func WarmPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.3
	cfg.Soundboard = 0.85
	cfg.Body = 0.85
	cfg.Damping = 1.8
	cfg.Resonance = 0.8
	cfg.Bark = 0.35
	cfg.Level = 18.6066 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BabyPreset is a small grand: shorter, stiffer strings that ring less long, a smaller, boxier
// soundboard, a narrower keyboard
func BabyPreset() Config {
	cfg := GrandPreset()
	cfg.Stiffness = 1.4
	cfg.Decay = 0.7
	cfg.Damping = 1.6
	cfg.Bark = 0.7
	cfg.Soundboard = 0.85
	cfg.Width = 0.45
	cfg.Stretch = 0.9
	cfg.Position = Player
	cfg.Level = 18.9641 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// VintagePreset is an old upright: worn hammers gone hard and bright, strings drifting apart, a
// noisy action, every key a little different
func VintagePreset() Config {
	cfg := UprightPreset()
	cfg.Hardness = 0.75
	cfg.Unison = 3.5
	cfg.Variation = 1
	cfg.Mechanics = 0.9
	cfg.Decay = 0.5
	cfg.Damping = 1.9
	cfg.Stretch = 0.5
	cfg.Position = Studio
	cfg.Level = 18.5793 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// CinematicPreset is the slow piano of film scores: soft, dark hammers, heard from the hall, the
// strings ringing along a lot
func CinematicPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.2
	cfg.Damping = 2.2
	cfg.Soundboard = 0.9
	cfg.Body = 0.8
	cfg.Resonance = 0.9
	cfg.VelocitySens = 0.6
	cfg.Mechanics = 0.4
	cfg.Position = Hall
	cfg.Level = 24.1592 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
