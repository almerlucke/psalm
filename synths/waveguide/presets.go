package waveguide

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"guitar":      Guitar,
	"steel":       Steel,
	"muted":       Muted,
	"harp":        Harp,
	"bass":        Bass,
	"pizzicato":   Pizzicato,
	"violin":      Violin,
	"cello":       Cello,
	"slowStrings": SlowStrings,
	"ponticello":  Ponticello,
	"flautando":   Flautando,
}

func plucked(decay, damping, brightness, pick, release float64, body Body, mix, level float64) Config {
	return Config{Model: Pluck, Decay: decay, Damping: damping, Brightness: brightness, PickPosition: pick,
		Release: release, Body: body, BodyMix: mix, Level: level}
}

func bowed(pressure, position, attack, release, vibRate, vibDepth, vibDelay float64, body Body, mix, level float64) Config {
	return Config{Model: Bow, Pressure: pressure, BowPosition: position, BowAttack: attack, BowRelease: release,
		VibratoRate: vibRate, VibratoDepth: vibDepth, VibratoDelay: vibDelay, Body: body, BodyMix: mix, Level: level}
}

// Guitar is a nylon string guitar, plucked with the fingers
func Guitar() Config { return plucked(3, 0.35, 0.45, 0.17, 0.25, GuitarBody, 0.45, 1.6) }

// Steel is a steel string guitar played with a pick: bright and long
func Steel() Config { return plucked(5, 0.12, 0.8, 0.1, 0.3, GuitarBody, 0.35, 1.45) }

// Muted is a palm muted guitar: a short, dull pluck
func Muted() Config { return plucked(0.25, 0.6, 0.6, 0.12, 0.05, GuitarBody, 0.4, 1.5) }

// Harp is plucked near the middle and rings on after the note
func Harp() Config { return plucked(5, 0.3, 0.35, 0.33, 0, HarpBody, 0.4, 1.5) }

// Bass is an electric bass, fingered: round, no body
func Bass() Config { return plucked(2.5, 0.5, 0.3, 0.2, 0.12, NoBody, 0, 1) }

// Pizzicato is a violin plucked: short, through its body
func Pizzicato() Config { return plucked(0.5, 0.4, 0.4, 0.25, 0.1, ViolinBody, 0.6, 1.6) }

// Violin is bowed, with a vibrato that comes in after the start
func Violin() Config { return bowed(0.55, 0.127, 0.08, 0.15, 5.5, 0.12, 0.25, ViolinBody, 0.6, 1.1) }

// Cello is bowed lower on the string, slower, through the large body
func Cello() Config { return bowed(0.6, 0.15, 0.12, 0.25, 5, 0.12, 0.3, CelloBody, 0.6, 0.85) }

// SlowStrings swell in and fade out slowly
func SlowStrings() Config { return bowed(0.5, 0.13, 0.6, 0.9, 5, 0.1, 0.4, ViolinBody, 0.6, 1.2) }

// Ponticello is bowed close to the bridge: glassy, with the high partials
func Ponticello() Config { return bowed(0.6, 0.08, 0.1, 0.2, 0, 0, 0, ViolinBody, 0.5, 1) }

// Flautando is bowed lightly over the fingerboard: soft and airy, like a flute
func Flautando() Config { return bowed(0.25, 0.4, 0.15, 0.2, 5, 0.08, 0.3, ViolinBody, 0.5, 1.2) }
