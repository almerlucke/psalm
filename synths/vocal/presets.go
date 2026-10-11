package vocal

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"solo":    Solo,
	"choir":   Choir,
	"ooh":     Ooh,
	"aah":     Aah,
	"breathy": Breathy,
	"monks":   Monks,
	"angels":  Angels,
	"robot":   Robot,
}

// Solo is one singer with a natural vibrato
func Solo() Config {
	return Config{Vowel: 0, VowelGlide: 0.15, Brightness: 0.5, Breath: 0.05, Attack: 0.08, Release: 0.3,
		VibratoRate: 5.5, VibratoDepth: 0.25, VibratoDelay: 0.3, Singers: 1, Level: 1.3}
}

// Choir is a section of eight singing "a", spread wide
func Choir() Config {
	return Config{Vowel: 0, VowelGlide: 0.3, Brightness: 0.4, Breath: 0.08, Attack: 0.25, Release: 0.6,
		VibratoRate: 5.2, VibratoDepth: 0.15, VibratoDelay: 0.2, Singers: 8, Detune: 18, Spread: 0.8, Level: 1.4}
}

// Ooh is a soft choir on "u"
func Ooh() Config {
	cfg := Choir()
	cfg.Vowel, cfg.Brightness, cfg.Attack, cfg.Release = 4, 0.2, 0.4, 0.8
	cfg.Level = 3.1
	return cfg
}

// Aah is a bright, full choir on "a"
func Aah() Config {
	cfg := Choir()
	cfg.Brightness, cfg.Singers, cfg.Detune = 0.65, 12, 22
	cfg.Level = 1.25
	return cfg
}

// Breathy is a close, airy solo voice on "o"
func Breathy() Config {
	cfg := Solo()
	cfg.Vowel, cfg.Brightness, cfg.Breath, cfg.VibratoDepth = 3, 0.15, 0.35, 0.15
	cfg.Level = 2.25
	return cfg
}

// Monks are low men's voices on "o", without vibrato
func Monks() Config {
	return Config{Range: Bass, Vowel: 3, VowelGlide: 0.4, Brightness: 0.35, Breath: 0.05, Attack: 0.3, Release: 0.8,
		Singers: 6, Detune: 12, Spread: 0.6, Level: 2.2}
}

// Angels are high voices on "i" morphing slowly, wide and airy
func Angels() Config {
	return Config{Range: Soprano, Vowel: 2, VowelGlide: 0.8, Brightness: 0.3, Breath: 0.15, Attack: 0.6, Release: 1.2,
		VibratoRate: 5, VibratoDepth: 0.12, VibratoDelay: 0.4, Singers: 8, Detune: 14, Spread: 1, Level: 1}
}

// Robot is a steady, bright voice with no vibrato and no drift: a vocoder's voice
func Robot() Config {
	return Config{Range: Tenor, Vowel: 1, VowelGlide: 0.05, Brightness: 0.9, Attack: 0.01, Release: 0.1,
		Singers: 1, Steady: true, Level: 1.2}
}
