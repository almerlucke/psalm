package waveguide

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"guitar":      Guitar,
	"steel":       Steel,
	"muted":       Muted,
	"harp":        Harp,
	"bass":        Bass,
	"uprightBass": UprightBass,
	"harmonics":   Harmonics,
	"pizzicato":   Pizzicato,
	"violin":      Violin,
	"cello":       Cello,
	"slowStrings": SlowStrings,
	"ponticello":  Ponticello,
	"flautando":   Flautando,
	"rosin":       Rosin,

	"glassHarp": GlassHarp,
	"wire":      Wire,
	"eBow":      EBow,
	"ghostBow":  GhostBow,
	"scrape":    Scrape,

	"tom":       Tom,
	"floorTom":  FloorTom,
	"conga":     Conga,
	"tabla":     Tabla,
	"frameDrum": FrameDrum,
	"snareDrum": SnareDrum,

	"flute":    Flute,
	"clarinet": Clarinet,
	"panpipe":  Panpipe,
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
func Guitar() Config {
	c := plucked(3, 0.35, 0.45, 0.17, 0.25, GuitarBody, 0.45, 1.209)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 3e-5, 0.07, 0.1 // nylon, a fingertip
	c.Polarization, c.BodyCoupling, c.FretNoise = 0.3, 0.5, 0.15
	return c
}

// Steel is a steel string guitar played with a pick: bright and long
func Steel() Config {
	c := plucked(5, 0.12, 0.8, 0.1, 0.3, GuitarBody, 0.35, 1.078)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.2e-4, 0.025, 0.9 // steel, a pick
	c.Polarization, c.BodyCoupling, c.TensionMod, c.FretNoise = 0.4, 0.4, 0.5, 0.1
	return c
}

// Muted is a palm muted guitar: a short, dull pluck
func Muted() Config {
	c := plucked(0.25, 0.6, 0.6, 0.12, 0.05, GuitarBody, 0.4, 1.432)
	c.Inharmonicity, c.PluckWidth, c.Plectrum, c.FretNoise = 1e-4, 0.03, 0.6, 0.1
	return c
}

// Harp is plucked near the middle and rings on after the note
func Harp() Config {
	c := plucked(5, 0.3, 0.35, 0.33, 0, HarpBody, 0.4, 1.37)
	c.Inharmonicity, c.PluckWidth = 2e-5, 0.05 // gut, a fingertip
	c.Polarization, c.BodyCoupling = 0.2, 0.3
	return c
}

// Bass is an electric bass, fingered: round, no body, a wound string that is stiff and a bit
// of tension modulation when it is dug into
func Bass() Config {
	c := plucked(2.5, 0.5, 0.3, 0.2, 0.12, NoBody, 0, 0.875)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 3e-4, 0.08, 0.1
	c.Polarization, c.TensionMod, c.FretNoise = 0.5, 0.3, 0.1
	return c
}

// UprightBass is a double bass plucked with the finger: the long thick gut or steel string, the
// large body that gives way under it, the string slapping the fingerboard a little
func UprightBass() Config {
	c := plucked(1.6, 0.45, 0.3, 0.2, 0.2, BassBody, 0.55, 1.429)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 2e-4, 0.09, 0
	c.Polarization, c.BodyCoupling, c.TensionMod, c.FretBuzz, c.FretNoise = 0.35, 0.7, 0.4, 0.15, 0.2
	return c
}

// Harmonics are the natural harmonics of a guitar string, a finger laid on its middle and
// the string plucked beside it: a bell an octave above the open string
func Harmonics() Config {
	c := plucked(4, 0.15, 0.7, 0.14, 0.2, GuitarBody, 0.35, 1.19)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1e-4, 0.03, 0.6
	c.Harmonic, c.Polarization, c.BodyCoupling = 0.5, 0.3, 0.3
	return c
}

// Pizzicato is a violin plucked: short, through its body
func Pizzicato() Config {
	c := plucked(0.5, 0.4, 0.4, 0.25, 0.1, ViolinBody, 0.6, 1.6)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 6e-5, 0.04, 0.4
	c.BodyCoupling, c.TensionMod, c.FretBuzz = 0.5, 0.5, 0.1
	return c
}

// rubbed makes a bowed string a friction curve (BowModel 1): a bow of a weight (BowForce, 0-1: from
// 0.06, a light bow, to 0.16 a heavy one that holds the string at the lower notes), the speed
// following the velocity and the weight following the speed. The grain of the rosin (BowNoise) is
// left to Rosin: it makes a bow that holds on one string slip to another on the next
func rubbed(c Config, force float64) Config {
	c.BowModel, c.BowForce = 1, force
	return c
}

// Violin is bowed, with a vibrato that comes in after the start
func Violin() Config {
	return rubbed(bowed(0.55, 0.127, 0.08, 0.15, 5.5, 0.12, 0.25, ViolinBody, 0.6, 1.315), 0.14)
}

// Cello is bowed lower on the string, slower, through the large body
func Cello() Config {
	return rubbed(bowed(0.6, 0.15, 0.12, 0.25, 5, 0.12, 0.3, CelloBody, 0.6, 1.348), 0.1)
}

// SlowStrings swell in and fade out slowly
func SlowStrings() Config {
	return rubbed(bowed(0.5, 0.13, 0.6, 0.9, 5, 0.1, 0.4, ViolinBody, 0.6, 1.576), 0.12)
}

// Ponticello is bowed close to the bridge: glassy, with the high partials
func Ponticello() Config {
	return rubbed(bowed(0.6, 0.08, 0.1, 0.2, 0, 0, 0, ViolinBody, 0.5, 1.98), 0.12)
}

// Rosin is a violin bowed by the physics of the thing: a friction curve with a memory, a bow
// that has a weight and a speed, and the grain of the rosin. The bow doesn't turn round (BowReverse
// is there to set: it dips a held note every stroke, and the highest notes lurch by it)
func Rosin() Config {
	c := bowed(0.55, 0.127, 0.08, 0.15, 5.5, 0.1, 0.25, ViolinBody, 0.6, 0.933) // the pressure is that of the table bow, for the highest notes
	c.BowModel, c.BowForce, c.BowSpeed, c.BowNoise, c.BowReverse = 1, 0.3, 0.5, 0.3, 0
	return c
}

// Flautando is bowed lightly over the fingerboard: soft and airy, like a flute
func Flautando() Config {
	return rubbed(bowed(0.25, 0.4, 0.15, 0.2, 5, 0.08, 0.3, ViolinBody, 0.5, 1.126), 0.12)
}

// GlassHarp is a string with no losses to speak of, plucked near its end: it shimmers for many
// seconds, its overtones dying no faster than the note
func GlassHarp() Config {
	c := plucked(14, 0, 0.9, 0.06, 0, HarpBody, 0.2, 1.12)
	c.Inharmonicity = 2e-4 // a hint of bell
	return c
}

// Wire is a bright steel wire picked right at its end, no body: thin, jangling, metallic
func Wire() Config {
	c := plucked(6, 0.02, 1, 0.015, 0.4, NoBody, 0, 1.03)
	c.Inharmonicity, c.Plectrum, c.TensionMod, c.FretBuzz = 5e-4, 0.5, 0.6, 0.25
	return c
}

// EBow is a string kept singing by a steady bow near its end, swelling in, no vibrato, no body: the
// endless sustain of an electric guitar under an EBow
func EBow() Config { return bowed(0.5, 0.1, 0.8, 0.4, 0, 0, 0, NoBody, 0, 0.618) }

// GhostBow is a cello bowed so lightly the note floats, swelling in, a slow wide vibrato
func GhostBow() Config {
	return rubbed(bowed(0.18, 0.35, 1.2, 1.5, 3, 0.35, 0.2, CelloBody, 0.5, 1.327), 0.14)
}

// Scrape is a bow pressed hard close to the bridge: gritty, metallic, on the edge of breaking
func Scrape() Config { return bowed(0.95, 0.1, 0.05, 0.1, 0, 0, 0, NoBody, 0, 0.556) }

func drum(decay, damping, strike, hardness, tension, snare, release float64, level float64) Config {
	return Config{Model: Membrane, Decay: decay, Damping: damping, Strike: strike, Hardness: hardness,
		Tension: tension, Snare: snare, Release: release, Level: level}
}

func blown(reed bool, breath, noise, embouchure, damping, attack, release, vibRate, vibDepth, vibDelay float64, level float64) Config {
	return Config{Model: Pipe, Reed: reed, Breath: breath, Noise: noise, Embouchure: embouchure, Damping: damping,
		BowAttack: attack, BowRelease: release, VibratoRate: vibRate, VibratoDepth: vibDepth, VibratoDelay: vibDelay, Level: level}
}

// Tom is a rack tom struck a little off the middle: a round tone with a bit of thump as the skin tightens
func Tom() Config { return drum(0.9, 0.45, 0.35, 0.5, 0.5, 0, 0, 0.6977) }

// FloorTom is a large tom: long, dark, the pitch falling back after the hit
func FloorTom() Config { return drum(1.4, 0.35, 0.4, 0.4, 0.7, 0, 0, 0.7815) }

// Conga is hit near the rim with the hand: bright and short, and a hand on the skin stops it
func Conga() Config { return drum(0.5, 0.6, 0.7, 0.75, 0.3, 0, 0.15, 0.8472) }

// Tabla is a small skin with a strong pitch bend after the strike and rings clearly
func Tabla() Config { return drum(0.7, 0.25, 0.25, 0.6, 0.9, 0, 0.2, 0.6519) }

// FrameDrum is a soft mallet on a wide, loose skin: a warm, dull boom
func FrameDrum() Config { return drum(1.2, 0.7, 0.5, 0.25, 0.1, 0, 0.3, 0.9018) }

// SnareDrum is a skin with the wires under it buzzing along, hit with a hard stick
func SnareDrum() Config { return drum(0.35, 0.5, 0.4, 0.8, 0.15, 0.7, 0, 0.5394) }

// Flute is blown over the edge: airy, with a vibrato that comes in after the start
func Flute() Config { return blown(false, 0.72, 0.25, 0.4, 0.6, 0.06, 0.12, 5, 0.15, 0.3, 0.146) }

// Clarinet is a single reed on a closed tube: hollow, the odd partials
func Clarinet() Config { return blown(true, 0.8, 0.1, 0.4, 0.5, 0.04, 0.08, 4.5, 0.06, 0.4, 0.42) }

// Panpipe is a flute blown hard and breathy, a closed tube each, no vibrato
func Panpipe() Config { return blown(false, 0.7, 0.45, 0.5, 0.7, 0.08, 0.15, 0, 0, 0, 0.158) }
