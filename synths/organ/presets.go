package organ

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"jazz":    Jazz,
	"gospel":  Gospel,
	"full":    Full,
	"rock":    Rock,
	"ballad":  Ballad,
	"flute":   Flute,
	"reed":    Reed,
	"whistle": Whistle,

	"screamer":  Screamer,
	"warble":    Warble,
	"chime":     Chime,
	"growlBass": GrowlBass,
	"seasick":   Seasick,
}

func preset(bars string, f func(*Config)) Config {
	cfg := DefaultConfig()
	cfg.Drawbars = Bars(bars)
	if f != nil {
		f(&cfg)
	}
	return cfg
}

// Jazz is 888000000 with the third percussion, fast: the walking jazz organ
func Jazz() Config {
	return preset("888000000", func(c *Config) {
		c.Percussion, c.PercFast, c.Level = PercThird, true, 1.7
	})
}

// Gospel is 888800000 with the deep chorus
func Gospel() Config {
	return preset("888800000", func(c *Config) { c.Vibrato, c.Level = C3, 1.8 })
}

// Full is every drawbar out, with chorus: the whole organ
func Full() Config {
	return preset("888888888", func(c *Config) { c.Vibrato, c.Level = C3, 1.15 })
}

// Rock is 888000000 driven, with chorus
func Rock() Config {
	return preset("888000000", func(c *Config) {
		c.Vibrato, c.Drive, c.Click, c.Level = C3, 0.6, 0.8, 2.5
	})
}

// Ballad is soft, 806004000: the 16' and 8' with a little 2', with chorus
func Ballad() Config {
	return preset("806004000", func(c *Config) { c.Vibrato, c.Click, c.Level = C2, 0.2, 2.8 })
}

// Flute is the 8' alone with a breath of 4' and vibrato: a soft, round voice
func Flute() Config {
	return preset("008400000", func(c *Config) { c.Vibrato, c.Click, c.Level = V2, 0.1, 2.8 })
}

// Reed is a nasal registration, 16' and the high mixtures
func Reed() Config {
	return preset("800000888", func(c *Config) { c.Vibrato, c.Level = C1, 1.65 })
}

// Whistle is the 16' and the 1', the top whistling over the bottom (the percussion would take the
// 1' away)
func Whistle() Config {
	return preset("800000008", func(c *Config) { c.Vibrato, c.Level = V1, 1.75 })
}

// Screamer is every drawbar out into a preamp driven to its limit, all the click, the deep chorus:
// the organ of a heavy rock band
func Screamer() Config {
	return preset("888888888", func(c *Config) { c.Vibrato, c.Drive, c.Click, c.Level = C3, 1, 1, 2.29 })
}

// Warble runs the scanner at 15 Hz, deep: the vibrato turns into a fluttering, alien warble
func Warble() Config {
	return preset("806040302", func(c *Config) { c.Vibrato, c.VibratoHz, c.Click, c.Level = V3, 15, 0.2, 2.6 })
}

// Chime is only the high drawbars with the third percussion, fast: a glassy, bell like stab
func Chime() Config {
	return preset("000088880", func(c *Config) { c.Percussion, c.PercFast, c.Click, c.Level = PercThird, true, 0.3, 1.28 })
}

// GrowlBass is the sixteen and eight foot, driven, all click and no vibrato: a growling bass organ
func GrowlBass() Config {
	return preset("880000000", func(c *Config) { c.Drive, c.Click, c.Level = 0.8, 1, 2.86 })
}

// Seasick turns the scanner down to 1.5 Hz, deep: everything sways out of tune and back
func Seasick() Config {
	return preset("808000888", func(c *Config) { c.Vibrato, c.VibratoHz, c.Click, c.Level = V3, 1.5, 0.1, 1.13 })
}
