package fm

import "github.com/almerlucke/genny/float/envelopes/adsr"

// More presets, covering the instrument families of General MIDI. They are original FM designs
// using the classic building blocks: a 1:1 modulator gives all harmonics (strings, brass), 1:2 only
// odd harmonics (clarinet, square), a high ratio modulator with a fast decay gives the attack of a
// hammer, pick or mallet, inharmonic ratios give bells and metal, and feedback gives buzz and breath

func init() {
	for name, preset := range gmPresets {
		Presets[name] = preset
	}
}

var gmPresets = map[string]func() Config{
	// pianos and keys
	"grandPiano":  GrandPianoPreset,
	"honkyTonk":   HonkyTonkPreset,
	"harpsichord": HarpsichordPreset,
	"clavinet":    ClavinetPreset,

	// chromatic percussion
	"celesta":      CelestaPreset,
	"glockenspiel": GlockenspielPreset,
	"musicBox":     MusicBoxPreset,
	"xylophone":    XylophonePreset,
	"tubularBells": TubularBellsPreset,

	// organs and accordion
	"percussiveOrgan": PercussiveOrganPreset,
	"churchOrgan":     ChurchOrganPreset,
	"accordion":       AccordionPreset,

	// guitars
	"nylonGuitar":     NylonGuitarPreset,
	"steelGuitar":     SteelGuitarPreset,
	"overdriveGuitar": OverdriveGuitarPreset,

	// basses
	"uprightBass":  UprightBassPreset,
	"slapBass":     SlapBassPreset,
	"fretlessBass": FretlessBassPreset,

	// strings
	"strings":   StringsPreset,
	"pizzicato": PizzicatoPreset,
	"harp":      HarpPreset,

	// brass
	"trumpet":    TrumpetPreset,
	"frenchHorn": FrenchHornPreset,

	// reeds and pipes
	"saxophone": SaxophonePreset,
	"clarinet":  ClarinetPreset,
	"oboe":      OboePreset,
	"flute":     FlutePreset,
	"panFlute":  PanFlutePreset,

	// synth leads and pads
	"squareLead": SquareLeadPreset,
	"sawLead":    SawLeadPreset,
	"warmPad":    WarmPadPreset,
	"sweepPad":   SweepPadPreset,

	// ethnic
	"sitar":     SitarPreset,
	"kalimba":   KalimbaPreset,
	"steelDrum": SteelDrumPreset,
	"koto":      KotoPreset,

	// percussion
	"timpani":   TimpaniPreset,
	"woodblock": WoodblockPreset,
	"taiko":     TaikoPreset,
}

// decay is a percussive envelope: instant attack, decay to silence
func decay(decayMs float64, releaseMs float64) *adsr.Setting {
	return adsr.NewSetting(1.0, 1.0, 0.0, decayMs, 1.0, releaseMs)
}

// sustain is an envelope that holds a level while the note lasts
func sustain(attackMs float64, level float64, decayMs float64, releaseMs float64) *adsr.Setting {
	return adsr.NewSetting(1.0, attackMs, level, decayMs, 1.0, releaseMs)
}

// --- pianos and keys ---

// GrandPianoPreset: a body pair plus an octave partial, with a short hammer transient on top
func GrandPianoPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4 // 4>3, 2>1
	cfg.Ops[0] = op(1, 1.0, 0.3, decay(3500, 500))
	cfg.Ops[1] = op(1, 0.45, 0.8, decay(900, 300))
	cfg.Ops[2] = op(2, 0.3, 0.3, decay(1800, 400))
	cfg.Ops[3] = op(7, 0.15, 0.9, decay(40, 30))
	cfg.Ops[2].Detune = 2.0
	return cfg
}

// HonkyTonkPreset: the grand piano with a second, detuned string
func HonkyTonkPreset() Config {
	cfg := GrandPianoPreset()
	cfg.Ops[2].Ratio = 1
	cfg.Ops[2].Level = 0.8
	cfg.Ops[2].Detune = 14.0
	return cfg
}

// HarpsichordPreset: a bright, plucked tone with a quick release
func HarpsichordPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 2 // (4 + 3>2)>1
	cfg.Ops[0] = op(1, 1.0, 0.1, decay(1500, 150))
	cfg.Ops[1] = op(3, 0.4, 0.3, decay(900, 150))
	cfg.Ops[2] = op(1, 0.3, 0.2, decay(600, 100))
	cfg.Ops[3] = op(5, 0.25, 0.3, decay(300, 80))
	cfg.Feedback = 0.2
	return cfg
}

// ClavinetPreset: a funky, buzzy plucked key
func ClavinetPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0 // stack
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(700, 60))
	cfg.Ops[1] = op(1, 0.6, 0.8, sustain(1, 0.2, 400, 60))
	cfg.Ops[2] = op(3, 0.3, 0.4, decay(200, 40))
	cfg.Ops[3] = op(1, 0.2, 0.3, decay(150, 40))
	cfg.Feedback = 0.4
	return cfg
}

// --- chromatic percussion ---

// CelestaPreset: a soft, sparkling bell keyboard
func CelestaPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(1400, 500))
	cfg.Ops[1] = op(4, 0.3, 0.6, decay(300, 200))
	cfg.Ops[2] = op(4, 0.25, 0.2, decay(900, 300))
	cfg.Ops[3] = op(9.5, 0.15, 0.5, decay(150, 100))
	return cfg
}

// GlockenspielPreset: bright metal bars with inharmonic partials
func GlockenspielPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(2500, 800))
	cfg.Ops[1] = op(3.5, 0.4, 0.5, decay(800, 300))
	cfg.Ops[2] = op(2.76, 0.3, 0.2, decay(1500, 500))
	cfg.Ops[3] = op(5.4, 0.25, 0.5, decay(400, 200))
	return cfg
}

// MusicBoxPreset: tiny plucked comb teeth
func MusicBoxPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 5 // 4>(3, 2, 1)
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(1800, 400))
	cfg.Ops[1] = op(2, 0.4, 0.2, decay(900, 300))
	cfg.Ops[2] = op(4, 0.2, 0.2, decay(500, 200))
	cfg.Ops[3] = op(7, 0.2, 0.6, decay(120, 80))
	return cfg
}

// XylophonePreset: short, dry wooden bars
func XylophonePreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(350, 100))
	cfg.Ops[1] = op(3, 0.35, 0.7, decay(40, 30))
	cfg.Ops[2] = op(3.93, 0.25, 0.2, decay(150, 60))
	cfg.Ops[3] = op(10, 0.2, 0.6, decay(20, 20))
	return cfg
}

// TubularBellsPreset: long, deep chimes
func TubularBellsPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 1 // (4+3)>2>1
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(6000, 2500))
	cfg.Ops[1] = op(3.5, 0.7, 0.5, decay(3000, 1500))
	cfg.Ops[2] = op(5.6, 0.3, 0.3, decay(1500, 800))
	cfg.Ops[3] = op(2.4, 0.3, 0.3, decay(2000, 1000))
	return cfg
}

// --- organs and accordion ---

// PercussiveOrganPreset: an additive organ with a decaying third harmonic click on every note
func PercussiveOrganPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 7 // all carriers
	cfg.Ops[0] = op(1, 1.0, 0, sustain(3, 1.0, 10, 60))
	cfg.Ops[1] = op(0.5, 0.6, 0, sustain(3, 1.0, 10, 60))
	cfg.Ops[2] = op(2, 0.5, 0, sustain(3, 1.0, 10, 60))
	cfg.Ops[3] = op(3, 0.7, 0, decay(250, 60))
	cfg.Level = 1.5
	return cfg
}

// ChurchOrganPreset: a full, slowly speaking pipe organ
func ChurchOrganPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 7
	cfg.Ops[0] = op(1, 1.0, 0, sustain(60, 1.0, 10, 400))
	cfg.Ops[1] = op(2, 0.8, 0, sustain(70, 1.0, 10, 400))
	cfg.Ops[2] = op(4, 0.5, 0, sustain(80, 1.0, 10, 400))
	cfg.Ops[3] = op(0.5, 0.8, 0, sustain(60, 1.0, 10, 400))
	cfg.Feedback = 0.05
	cfg.Level = 1.4
	return cfg
}

// AccordionPreset: two slightly detuned buzzing reeds
func AccordionPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4 // 4>3, 2>1
	cfg.Ops[0] = op(1, 1.0, 0, sustain(30, 0.9, 100, 120))
	cfg.Ops[1] = op(1, 0.45, 0.3, sustain(30, 0.9, 100, 120))
	cfg.Ops[2] = op(1, 0.8, 0, sustain(30, 0.9, 100, 120))
	cfg.Ops[3] = op(1, 0.4, 0.3, sustain(30, 0.9, 100, 120))
	cfg.Ops[2].Detune = 8.0
	return cfg
}

// --- guitars ---

// NylonGuitarPreset: a soft, round plucked string
func NylonGuitarPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 2
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(1500, 200))
	cfg.Ops[1] = op(1, 0.3, 0.6, decay(300, 150))
	cfg.Ops[2] = op(3, 0.2, 0.3, decay(200, 100))
	cfg.Ops[3] = op(2, 0.15, 0.4, decay(250, 100))
	return cfg
}

// SteelGuitarPreset: a brighter, ringing steel string
func SteelGuitarPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 2
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(2000, 250))
	cfg.Ops[1] = op(1, 0.5, 0.7, sustain(1, 0.05, 500, 200))
	cfg.Ops[2] = op(4, 0.25, 0.3, decay(250, 100))
	cfg.Ops[3] = op(1, 0.3, 0.4, decay(300, 100))
	cfg.Feedback = 0.25
	return cfg
}

// OverdriveGuitarPreset: a sustained, gritty lead guitar from heavy feedback on a stack
func OverdriveGuitarPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0.1, sustain(3, 0.7, 800, 200))
	cfg.Ops[1] = op(1, 0.7, 0.6, sustain(3, 0.6, 600, 200))
	cfg.Ops[2] = op(2, 0.4, 0.4, sustain(3, 0.5, 500, 200))
	cfg.Ops[3] = op(1, 0.4, 0.3, sustain(3, 0.7, 500, 200))
	cfg.Feedback = 0.8
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.15
	cfg.VibratoDelay = 500.0
	return cfg
}

// --- basses ---

// UprightBassPreset: a plucked acoustic bass, round with a soft finger attack
func UprightBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 2
	cfg.Ops[0] = op(1, 1.0, 0.2, adsr.NewSetting(1.0, 5.0, 0.0, 1200.0, 1.0, 150.0))
	cfg.Ops[1] = op(1, 0.3, 0.5, adsr.NewSetting(1.0, 5.0, 0.0, 250.0, 1.0, 100.0))
	cfg.Ops[2] = op(2, 0.15, 0.3, decay(150, 80))
	cfg.Ops[3] = op(1, 0.1, 0.3, decay(60, 40))
	return cfg
}

// SlapBassPreset: a snappy, bright bass with a percussive pop
func SlapBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0.2, sustain(1, 0.5, 300, 100))
	cfg.Ops[1] = op(1, 0.7, 0.8, sustain(1, 0.15, 120, 80))
	cfg.Ops[2] = op(3, 0.4, 0.5, decay(40, 30))
	cfg.Ops[3] = op(1, 0.3, 0.4, decay(30, 30))
	cfg.Feedback = 0.5
	return cfg
}

// FretlessBassPreset: a smooth, singing bass that glides between notes
func FretlessBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0.1, sustain(10, 0.8, 400, 150))
	cfg.Ops[1] = op(1, 0.25, 0.5, sustain(20, 0.4, 500, 150))
	cfg.Ops[2] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Ops[3] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Glide = 60.0
	return cfg
}

// --- strings ---

// StringsPreset: a string section from three detuned saw like carriers sharing one modulator
func StringsPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 5 // 4>(3, 2, 1)
	cfg.Ops[0] = op(1, 1.0, 0, sustain(250, 0.9, 500, 600))
	cfg.Ops[1] = op(1, 0.9, 0, sustain(250, 0.9, 500, 600))
	cfg.Ops[2] = op(1, 0.8, 0, sustain(250, 0.9, 500, 600))
	cfg.Ops[3] = op(1, 0.25, 0.3, sustain(300, 0.8, 500, 600))
	cfg.Ops[1].Detune = 7.0
	cfg.Ops[2].Detune = -7.0
	cfg.VibratoRate = 5.2
	cfg.VibratoDepth = 0.08
	cfg.VibratoDelay = 400.0
	return cfg
}

// PizzicatoPreset: plucked strings
func PizzicatoPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(400, 100))
	cfg.Ops[1] = op(1, 0.3, 0.7, decay(80, 50))
	cfg.Ops[2] = op(2, 0.4, 0.2, decay(250, 80))
	cfg.Ops[3] = op(1, 0.2, 0.5, decay(60, 40))
	return cfg
}

// HarpPreset: long ringing plucked strings
func HarpPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(2500, 500))
	cfg.Ops[1] = op(2, 0.2, 0.6, decay(200, 150))
	cfg.Ops[2] = op(2, 0.3, 0.2, decay(1200, 400))
	cfg.Ops[3] = op(3, 0.15, 0.5, decay(150, 100))
	return cfg
}

// --- brass ---

// TrumpetPreset: a bright brass tone that opens up as it speaks, with delayed vibrato
func TrumpetPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 1 // (4+3)>2>1
	cfg.Ops[0] = op(1, 1.0, 0.2, sustain(30, 0.85, 200, 150))
	cfg.Ops[1] = op(1, 0.55, 0.7, sustain(60, 0.75, 300, 150))
	cfg.Ops[2] = op(2, 0.15, 0.4, sustain(80, 0.6, 300, 150))
	cfg.Ops[3] = op(1, 0.2, 0.4, sustain(50, 0.6, 300, 150))
	cfg.Feedback = 0.35
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.12
	cfg.VibratoDelay = 350.0
	return cfg
}

// FrenchHornPreset: a mellow, round brass tone
func FrenchHornPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0.2, sustain(70, 0.9, 200, 250))
	cfg.Ops[1] = op(1, 0.3, 0.6, sustain(120, 0.7, 400, 250))
	cfg.Ops[2] = op(1, 0.1, 0.3, sustain(150, 0.6, 400, 250))
	cfg.Ops[3] = op(1, 0.05, 0.2, sustain(150, 0.6, 400, 250))
	cfg.Feedback = 0.1
	cfg.VibratoRate = 4.8
	cfg.VibratoDepth = 0.06
	cfg.VibratoDelay = 500.0
	return cfg
}

// --- reeds and pipes ---

// SaxophonePreset: a reedy, breathy sax with vibrato
func SaxophonePreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 1
	cfg.Ops[0] = op(1, 1.0, 0.2, sustain(40, 0.85, 200, 150))
	cfg.Ops[1] = op(1, 0.5, 0.7, sustain(50, 0.7, 300, 150))
	cfg.Ops[2] = op(3, 0.12, 0.4, sustain(60, 0.6, 300, 150))
	cfg.Ops[3] = op(1, 0.1, 0.3, sustain(40, 0.6, 300, 150))
	cfg.Feedback = 0.5
	cfg.VibratoRate = 5.0
	cfg.VibratoDepth = 0.15
	cfg.VibratoDelay = 300.0
	return cfg
}

// ClarinetPreset: a hollow woodwind, a 1:2 modulator gives mostly odd harmonics
func ClarinetPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0.1, sustain(40, 0.9, 100, 150))
	cfg.Ops[1] = op(2, 0.35, 0.5, sustain(40, 0.8, 200, 150))
	cfg.Ops[2] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Ops[3] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.VibratoRate = 4.8
	cfg.VibratoDepth = 0.05
	cfg.VibratoDelay = 500.0
	return cfg
}

// OboePreset: a nasal double reed
func OboePreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.1, sustain(30, 0.9, 100, 120))
	cfg.Ops[1] = op(1, 0.55, 0.5, sustain(30, 0.8, 200, 120))
	cfg.Ops[2] = op(2, 0.3, 0.1, sustain(30, 0.9, 100, 120))
	cfg.Ops[3] = op(3, 0.25, 0.4, sustain(30, 0.8, 200, 120))
	cfg.VibratoRate = 5.0
	cfg.VibratoDepth = 0.1
	cfg.VibratoDelay = 300.0
	return cfg
}

// FlutePreset: a nearly pure tone with breath: strong feedback turns operator 4 into noise,
// which roughens operator 3
func FlutePreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4 // 4>3, 2>1
	cfg.Ops[0] = op(1, 1.0, 0.1, sustain(60, 0.9, 200, 180))
	cfg.Ops[1] = op(1, 0.08, 0.5, sustain(80, 0.6, 300, 180))
	cfg.Ops[2] = op(2, 0.15, 0.1, sustain(60, 0.9, 200, 180))
	cfg.Ops[3] = op(1, 0.4, 0.3, sustain(30, 0.15, 200, 150))
	cfg.Feedback = 0.9
	cfg.VibratoRate = 5.0
	cfg.VibratoDepth = 0.1
	cfg.VibratoDelay = 400.0
	return cfg
}

// PanFlutePreset: a breathier flute with a chiff at the start of every note
func PanFlutePreset() Config {
	cfg := FlutePreset()
	cfg.Ops[3] = op(1, 0.7, 0.4, sustain(5, 0.1, 120, 150))
	cfg.Ops[1].Level = 0.04
	cfg.VibratoDepth = 0.06
	return cfg
}

// --- synth leads and pads ---

// SquareLeadPreset: a hollow, square like lead (1:2 modulator) with glide
func SquareLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0.1, sustain(5, 0.9, 100, 150))
	cfg.Ops[1] = op(2, 0.45, 0.4, sustain(5, 0.9, 100, 150))
	cfg.Ops[2] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Ops[3] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Glide = 20.0
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.15
	cfg.VibratoDelay = 300.0
	return cfg
}

// SawLeadPreset: operator 4 with strong feedback is a saw like oscillator, operator 1 adds body
func SawLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 7 // all carriers
	cfg.Ops[0] = op(1, 0.3, 0, sustain(5, 0.9, 100, 150))
	cfg.Ops[1] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Ops[2] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Ops[3] = op(1, 1.0, 0, sustain(5, 0.9, 100, 150))
	cfg.Feedback = 0.75
	cfg.Glide = 25.0
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.15
	cfg.VibratoDelay = 300.0
	cfg.Level = 2.0
	return cfg
}

// WarmPadPreset: a soft, slowly blooming pad
func WarmPadPreset() Config {
	cfg := StringsPreset()
	for i := range 3 {
		cfg.Ops[i].Env = *sustain(1200, 0.9, 1000, 2000)
	}
	cfg.Ops[3] = op(2, 0.15, 0, sustain(2000, 0.7, 1500, 2000))
	cfg.VibratoDepth = 0.05
	return cfg
}

// SweepPadPreset: the modulator rises slowly, so every chord brightens over a few seconds
func SweepPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0, sustain(1500, 0.9, 500, 2500))
	cfg.Ops[1] = op(1, 0.6, 0, sustain(4000, 1.0, 100, 2500))
	cfg.Ops[2] = op(2, 0.15, 0, sustain(3000, 1.0, 100, 2500))
	cfg.Ops[3] = op(1, 0.0, 0, sustain(10, 1.0, 10, 100))
	cfg.Ops[1].Detune = 4.0
	return cfg
}

// --- ethnic ---

// SitarPreset: a plucked string with the characteristic buzzing bridge (feedback and a high partial)
func SitarPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 1
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(2500, 500))
	cfg.Ops[1] = op(1, 0.5, 0.6, decay(1500, 400))
	cfg.Ops[2] = op(9.97, 0.2, 0.3, decay(800, 300))
	cfg.Ops[3] = op(1, 0.3, 0.3, decay(1200, 400))
	cfg.Feedback = 0.6
	return cfg
}

// KalimbaPreset: a thumb piano, a soft tine with a little metallic click
func KalimbaPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(700, 200))
	cfg.Ops[1] = op(6, 0.15, 0.6, decay(60, 40))
	cfg.Ops[2] = op(3, 0.2, 0.2, decay(300, 100))
	cfg.Ops[3] = op(1, 0.1, 0.3, decay(100, 60))
	return cfg
}

// SteelDrumPreset: a bright, ringing steel pan
func SteelDrumPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(900, 250))
	cfg.Ops[1] = op(2, 0.35, 0.6, decay(250, 150))
	cfg.Ops[2] = op(3, 0.3, 0.2, decay(600, 200))
	cfg.Ops[3] = op(1, 0.2, 0.5, decay(150, 100))
	return cfg
}

// KotoPreset: a plucked zither with a sharp attack
func KotoPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 2
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(1200, 200))
	cfg.Ops[1] = op(3, 0.35, 0.6, decay(180, 100))
	cfg.Ops[2] = op(1, 0.25, 0.4, decay(150, 80))
	cfg.Ops[3] = op(1, 0.25, 0.4, decay(100, 60))
	cfg.Feedback = 0.3
	return cfg
}

// --- percussion ---

// TimpaniPreset: a tuned kettle drum with inharmonic overtones
func TimpaniPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.3, adsr.NewSetting(1.0, 2.0, 0.0, 1500.0, 1.0, 500.0))
	cfg.Ops[1] = op(1.5, 0.4, 0.6, adsr.NewSetting(1.0, 2.0, 0.0, 200.0, 1.0, 100.0))
	cfg.Ops[2] = op(1.68, 0.4, 0.3, adsr.NewSetting(1.0, 2.0, 0.0, 800.0, 1.0, 300.0))
	cfg.Ops[3] = op(2.3, 0.3, 0.5, decay(100, 80))
	return cfg
}

// WoodblockPreset: a short, hollow knock
func WoodblockPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, decay(80, 40))
	cfg.Ops[1] = op(2.53, 0.4, 0.5, decay(30, 20))
	cfg.Ops[2] = op(3.7, 0.4, 0.2, decay(60, 40))
	cfg.Ops[3] = op(1, 0.2, 0.5, decay(20, 20))
	return cfg
}

// TaikoPreset: a big, low drum: a noisy burst from operator 4 (full feedback) for the skin hit
func TaikoPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.3, adsr.NewSetting(1.0, 2.0, 0.0, 900.0, 1.0, 300.0))
	cfg.Ops[1] = op(1.4, 0.5, 0.6, decay(80, 50))
	cfg.Ops[2] = op(2.1, 0.2, 0.3, adsr.NewSetting(1.0, 2.0, 0.0, 400.0, 1.0, 150.0))
	cfg.Ops[3] = op(1, 0.3, 0.5, decay(50, 30))
	cfg.Feedback = 0.95
	return cfg
}
