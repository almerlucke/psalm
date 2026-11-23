package waveguide

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

// morePresets are the instruments of the families, one by one: the plucked strings of the world,
// the bowed ones, the drums and the pipes. They join Presets when the package starts.
var morePresets = map[string]func() Config{
	// plucked: guitars
	"nylonClassical":  NylonClassical,
	"ownGuitar":       OwnGuitar,
	"steelAcoustic":   SteelAcoustic,
	"twelveString":    TwelveString,
	"palmMuted":       PalmMuted,
	"jazzArchtop":     JazzArchtop,
	"electricPick":    ElectricPick,
	"resonator":       Resonator,
	"pinchHarmonic":   PinchHarmonic,
	"twelfthHarmonic": TwelfthHarmonic,
	// plucked: basses
	"slapBass":       SlapBass,
	"fretlessBass":   FretlessBass,
	"pickBass":       PickBass,
	"uprightWalking": UprightWalking,
	// plucked: harps, zithers and lutes
	"harpHarmonics":   HarpHarmonics,
	"celticHarp":      CelticHarp,
	"banjo":           Banjo,
	"mandolin":        Mandolin,
	"mandolinTremolo": MandolinTremolo,
	"ukulele":         Ukulele,
	"sitar":           Sitar,
	"koto":            Koto,
	"oud":             Oud,
	"lute":            Lute,
	"dulcimer":        Dulcimer,
	"zither":          Zither,
	"harpsichord":     Harpsichord,
	"clavinet":        Clavinet,
	"pianoString":     PianoString,
	"snapPizzicato":   SnapPizzicato,
	"colLegno":        ColLegno,

	// bowed
	"violinSolo":     ViolinSolo,
	"violinSection":  ViolinSection,
	"viola":          Viola,
	"contrabass":     Contrabass,
	"legatoViolin":   LegatoViolin,
	"staccatoViolin": StaccatoViolin,
	"spiccato":       Spiccato,
	"tremoloStrings": TremoloStrings,
	"hurdyGurdy":     HurdyGurdy,
	"erhu":           Erhu,
	"rebab":          Rebab,
	"bowedGlass":     BowedGlass,
	"bowedPsaltery":  BowedPsaltery,
	"celloDrone":     CelloDrone,
	"singingSaw":     SingingSaw,

	// membranes
	"kickDrum":   KickDrum,
	"tomHigh":    TomHigh,
	"tomLow":     TomLow,
	"congaOpen":  CongaOpen,
	"congaSlap":  CongaSlap,
	"congaBass":  CongaBass,
	"bongo":      Bongo,
	"djembe":     Djembe,
	"bayan":      Bayan,
	"dayan":      Dayan,
	"bodhran":    Bodhran,
	"taiko":      Taiko,
	"snareTight": SnareTight,
	"snareFat":   SnareFat,
	"snareBrush": SnareBrush,
	"timbale":    Timbale,
	"cajon":      Cajon,
	"tambourine": Tambourine,

	// pipes
	"altoFlute":      AltoFlute,
	"piccolo":        Piccolo,
	"shakuhachi":     Shakuhachi,
	"ocarina":        Ocarina,
	"recorder":       Recorder,
	"bassClarinet":   BassClarinet,
	"chalumeau":      Chalumeau,
	"breathyFlute":   BreathyFlute,
	"overblownFlute": OverblownFlute,
	"whistle":        Whistle,
	"bansuri":        Bansuri,
	"tinWhistle":     TinWhistle,
}

func init() {
	for name, f := range morePresets {
		Presets[name] = f
	}
}

// lvl sets the output level of a preset (measured, see synths/levels)
func lvl(c Config, level float64) Config {
	c.Level = level
	return c
}

// mod puts a source to a destination in a slot of the matrix
func mod(c *Config, slot int, src modmatrix.Source, dest string, amount float64) {
	c.Mods[slot] = modmatrix.Slot{Src: src, Dest: dest, Amount: amount}
}

// ---- plucked: guitars

// NylonClassical is a classical guitar played with the fingertips of the right hand: round and
// warm, the place of the pluck changing a little from note to note and the harder touch brighter
func NylonClassical() Config {
	c := plucked(2.6, 0.4, 0.4, 0.16, 0.2, GuitarBody, 0.5, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 2.5e-5, 0.08, 0.05
	c.Polarization, c.BodyCoupling, c.FretNoise = 0.35, 0.6, 0.12
	mod(&c, 0, modmatrix.Random, "pickPosition", 0.6)
	mod(&c, 1, modmatrix.Velocity, "brightness", 0.5)
	return lvl(c, 1.21)
}

// OwnGuitar is Guitar with strings that die as those of a real nylon guitar do, a recording of which it is
// fitted to (the reference of cmd/tools/guitarcompare): the bass strings ring long and dark, the trebles
// shorter and brighter, keeping their top partials (their damping goes below 0 from about G3 up). The fit is the Track; Decay and Damping hold at A3 and turn the whole of it
//
// Its body is that of the guitar too: taps on it, as an impulse response (OwnGuitarBody), half of the sound next
// to the resonators the other half is. A plain Voice has the resonators only; NewPoly adds the measured body
func OwnGuitar() Config {
	c := fitted(ownTrack)
	c.Measured = OwnGuitarBody()
	return c
}

// fitted is Guitar with a track, its Decay and Damping those of the track at its reference pitch
func fitted(track *KeyTrack) Config {
	c := Guitar()
	c.Track = track
	if len(track.Decay) > 0 {
		ref := float64(track.Ref)
		c.Decay, c.Damping = track.DecayAt(ref), track.DampingAt(ref)
	}
	return c
}

// SteelAcoustic is a steel string flat top played with the fingers (fingerpicking): the bright ring
// of the wire without the click of a pick
func SteelAcoustic() Config {
	c := plucked(4.2, 0.18, 0.5, 0.13, 0.25, GuitarBody, 0.4, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1e-4, 0.04, 0.3
	c.Polarization, c.BodyCoupling, c.FretNoise = 0.45, 0.5, 0.1
	mod(&c, 0, modmatrix.Velocity, "brightness", 0.4)
	return lvl(c, 1.079)
}

// TwelveString is a guitar with its strings in pairs and octaves: the two of a pair never quite
// in unison (beats), the octave strings sounding along
func TwelveString() Config {
	c := plucked(4.5, 0.15, 0.7, 0.11, 0.3, GuitarBody, 0.4, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1e-4, 0.03, 0.7
	c.Polarization, c.BodyCoupling, c.Sympathetic = 0.9, 0.4, 0.45
	return lvl(c, 1.012)
}

// PalmMuted is an electric guitar chugged with the heel of the hand on the strings at the bridge:
// a thick, short thud with a pick attack
func PalmMuted() Config {
	c := plucked(0.16, 0.8, 0.55, 0.09, 0.03, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum, c.FretNoise = 1.2e-4, 0.02, 0.9, 0.15
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 2200}
	return lvl(c, 1.148)
}

// JazzArchtop is a hollow body guitar with flatwound strings, played with the thumb: dark, round
// and smooth, little sustain
func JazzArchtop() Config {
	c := plucked(1.8, 0.62, 0.22, 0.2, 0.15, GuitarBody, 0.35, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 2.5e-4, 0.07, 0.35
	c.Polarization, c.BodyCoupling = 0.3, 0.45
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 3000}
	return lvl(c, 1.166)
}

// ElectricPick is a solid body electric guitar through a pickup: picked, long and clear, with a
// little twang on a hard note; the pickup is a low pass, the touch opens it
func ElectricPick() Config {
	c := plucked(3.5, 0.22, 0.45, 0.12, 0.12, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.1e-4, 0.025, 0.6
	c.Polarization, c.TensionMod, c.FretNoise = 0.3, 0.6, 0.1
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 3500}
	mod(&c, 0, modmatrix.Velocity, "brightness", 0.4)
	return lvl(c, 0.9663)
}

// Resonator is a dobro: a steel guitar with a metal cone in its body, picked and slid. A tinny, nasal
// ring, and the notes glide into each other on a mono voice
func Resonator() Config {
	c := plucked(3, 0.2, 0.72, 0.09, 0.2, GuitarBody, 0.45, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.2e-4, 0.03, 0.55
	c.Polarization, c.BodyCoupling, c.Slide = 0.4, 0.8, 0.12
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 3000, Resonance: 0.3}
	return lvl(c, 1.133)
}

// PinchHarmonic is a pick that brushes the string with the thumb: an artificial harmonic squealing
// two octaves above the string's note, on a node a quarter of the string away
func PinchHarmonic() Config {
	c := plucked(3.5, 0.12, 0.8, 0.12, 0.2, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.1e-4, 0.02, 0.95
	c.Harmonic, c.TensionMod = 0.25, 0.3
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 5000}
	return lvl(c, 1.233)
}

// TwelfthHarmonic is the harmonic at the seventh fret of a nylon string: a soft bell a twelfth above
// the open string, a finger laid on a third of its length
func TwelfthHarmonic() Config {
	c := plucked(3, 0.25, 0.5, 0.12, 0.2, GuitarBody, 0.3, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 4e-5, 0.05, 0.1
	c.Harmonic, c.Polarization, c.BodyCoupling = 1.0/3, 0.25, 0.3
	return lvl(c, 1.069)
}

// ---- plucked: basses

// SlapBass is an electric bass slapped with the thumb and popped by the finger: bright, with a
// click, the string slapping the frets
func SlapBass() Config {
	c := plucked(1.3, 0.35, 0.9, 0.12, 0.08, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 2.5e-4, 0.03, 0.95
	c.TensionMod, c.FretBuzz, c.FretNoise, c.Polarization = 0.85, 0.5, 0.4, 0.3
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 4500}
	return lvl(c, 1.059)
}

// FretlessBass is a bass without frets: the string sings against the fingerboard, the "mwah" of a
// buzz that comes with the note, and a slide from note to note on a mono voice
func FretlessBass() Config {
	c := plucked(3.2, 0.42, 0.28, 0.2, 0.18, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 2.5e-4, 0.09, 0.1
	c.Polarization, c.FretBuzz, c.FretPosition, c.Slide = 0.4, 0.3, 0.9, 0.08
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 1800}
	return lvl(c, 1.002)
}

// PickBass is a bass played with a plectrum: a hard attack and a clear, even sustain
func PickBass() Config {
	c := plucked(1.4, 0.25, 0.6, 0.08, 0.1, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 3e-4, 0.02, 1
	c.Polarization, c.TensionMod, c.FretNoise = 0.5, 0.3, 0.35
	return lvl(c, 0.8512)
}

// UprightWalking is an upright bass in a walking line: the finger pulls the thick gut string,
// a woody note that fades before the next
func UprightWalking() Config {
	c := plucked(1.1, 0.55, 0.38, 0.17, 0.12, BassBody, 0.65, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 2e-4, 0.1, 0
	c.Polarization, c.BodyCoupling, c.FretBuzz, c.FretNoise = 0.3, 0.8, 0.08, 0.25
	return lvl(c, 1.58)
}

// ---- plucked: harps, zithers and lutes

// HarpHarmonics are the harmonics of a harp: a string stopped in the middle by the palm and plucked,
// the octave ringing like a small bell
func HarpHarmonics() Config {
	c := plucked(3.5, 0.2, 0.5, 0.12, 0, HarpBody, 0.4, 1)
	c.Inharmonicity, c.PluckWidth = 2e-5, 0.04
	c.Harmonic, c.Polarization, c.BodyCoupling = 0.5, 0.2, 0.3
	return lvl(c, 1.289)
}

// CelticHarp is a small harp with brass strings plucked with the nails: bright, bell like,
// ringing on
func CelticHarp() Config {
	c := plucked(3.5, 0.12, 0.6, 0.22, 0, HarpBody, 0.45, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.6e-4, 0.025, 0.25
	c.Polarization, c.BodyCoupling = 0.25, 0.4
	mod(&c, 0, modmatrix.Random, "pickPosition", 0.6)
	return lvl(c, 1.421)
}

// Banjo is a steel string over a drum head: short and bright, with a metallic ring and the pluck
// right at the bridge
func Banjo() Config {
	c := plucked(0.75, 0.12, 0.85, 0.07, 0.1, ViolinBody, 0.45, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.3e-4, 0.015, 0.7
	c.Polarization, c.FretNoise = 0.3, 0.1
	c.Filter = voicefilter.Config{Type: voicefilter.HighPass, Cutoff: 300}
	return lvl(c, 1.565)
}

// Mandolin is plucked with a plectrum on pairs of steel strings that are a little out of tune with
// each other: bright, quickly fading, shimmering
func Mandolin() Config {
	c := plucked(0.9, 0.2, 0.25, 0.1, 0.1, ViolinBody, 0.35, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.4e-4, 0.015, 0.95
	c.Polarization, c.BodyCoupling, c.FretNoise = 0.95, 0.4, 0.1
	mod(&c, 0, modmatrix.Velocity, "brightness", 0.4)
	return lvl(c, 1.094)
}

// MandolinTremolo is the tremolo of the mandolin: the plectrum going back and forth so fast that the
// notes join, a string kept sounding, 12 times a second
func MandolinTremolo() Config {
	c := plucked(2.2, 0.25, 0.78, 0.1, 0.12, ViolinBody, 0.35, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.4e-4, 0.015, 0.9
	c.Polarization, c.BodyCoupling = 0.95, 0.4
	c.LFO2Rate = 12
	mod(&c, 0, modmatrix.LFO2, "level", -0.45)
	return lvl(c, 1.129)
}

// Ukulele is a small nylon guitar strummed by the thumb: soft, plunky, quickly gone
func Ukulele() Config {
	c := plucked(1.1, 0.5, 0.42, 0.14, 0.1, GuitarBody, 0.4, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 3e-5, 0.06, 0.2
	c.Polarization, c.BodyCoupling, c.FretNoise = 0.3, 0.45, 0.08
	c.Filter = voicefilter.Config{Type: voicefilter.HighPass, Cutoff: 150}
	return lvl(c, 1.495)
}

// Sitar has a flat bridge that the string is lifted from and falls back on: the string buzzes
// against it with every swing (the jawari), a plectrum on the main string, the drone strings
// sounding along
func Sitar() Config {
	c := plucked(5.5, 0.2, 0.6, 0.1, 0.2, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 5e-5, 0.03, 0.6
	c.FretBuzz, c.FretPosition, c.Sympathetic, c.TensionMod = 0.85, 0.06, 0.55, 0.4
	c.Polarization = 0.3
	return lvl(c, 0.8861)
}

// Koto is a long zither of silk strings on movable bridges, plucked with a plectrum on the thumb
// and bent after the pluck by pressing the string behind the bridge
func Koto() Config {
	c := plucked(2.6, 0.22, 0.68, 0.1, 0.1, HarpBody, 0.4, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 8e-5, 0.025, 0.8
	c.TensionMod, c.Polarization, c.BodyCoupling = 0.8, 0.3, 0.35
	return lvl(c, 1.433)
}

// Oud is the fretless lute of the Middle East: paired gut strings plucked with a long plectrum
// (risha), a deep round bowl
func Oud() Config {
	c := plucked(1.6, 0.42, 0.52, 0.15, 0.18, GuitarBody, 0.55, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 4e-5, 0.045, 0.6
	c.Polarization, c.BodyCoupling, c.FretNoise = 0.7, 0.75, 0.1
	return lvl(c, 1.222)
}

// Lute is a renaissance lute: courses of gut strings plucked with the fingers, the soft, woody sound
// of the back of its large bowl
func Lute() Config {
	c := plucked(1.9, 0.38, 0.42, 0.16, 0.2, GuitarBody, 0.5, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 6e-5, 0.06, 0.12
	c.Polarization, c.BodyCoupling, c.Sympathetic = 0.65, 0.6, 0.15
	return lvl(c, 1.16)
}

// Dulcimer is a hammered dulcimer: courses of wires struck with light hammers, every note shimmering
// and the open strings sounding along
func Dulcimer() Config {
	c := plucked(3.8, 0.16, 0.55, 0.125, 0, HarpBody, 0.5, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.8e-4, 0.015, 0.2
	c.Polarization, c.BodyCoupling, c.Sympathetic = 0.9, 0.3, 0.3
	return lvl(c, 1.426)
}

// Zither is a flat box of strings plucked with a ring on the thumb, ringing into each other
func Zither() Config {
	c := plucked(4.2, 0.22, 0.5, 0.12, 0, HarpBody, 0.45, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1e-4, 0.03, 0.4
	c.Polarization, c.BodyCoupling, c.Sympathetic = 0.55, 0.4, 0.4
	return lvl(c, 1.42)
}

// Harpsichord has a quill in a jack that plucks the string as the key goes down and a damper that
// stops it as it goes up: bright, the same loudness whatever the touch
func Harpsichord() Config {
	c := plucked(2, 0.14, 0.82, 0.1, 0.03, HarpBody, 0.4, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.2e-4, 0.012, 1
	c.BodyCoupling, c.FretNoise = 0.4, 0.2
	return lvl(c, 1.505)
}

// Clavinet is a clavichord for funk: a string pressed onto an anvil and picked up electrically,
// a bright, stinging, tight note
func Clavinet() Config {
	c := plucked(0.9, 0.3, 0.78, 0.05, 0.04, NoBody, 0, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1e-4, 0.015, 0.9
	c.TensionMod, c.FretBuzz, c.FretPosition = 0.3, 0.25, 0.1
	c.Filter = voicefilter.Config{Type: voicefilter.BandPass, Cutoff: 2200, Resonance: 0.2}
	return lvl(c, 2.786)
}

// PianoString is a piano string alone, struck by a felt hammer an eighth of the way along: stiff
// partials a little sharp, a long ring, the soundboard under it
func PianoString() Config {
	c := plucked(6, 0.28, 0.45, 0.125, 0.1, HarpBody, 0.5, 1)
	c.Inharmonicity, c.PluckWidth = 4e-4, 0.12
	c.Polarization, c.BodyCoupling = 0.45, 0.6
	return lvl(c, 1.39)
}

// SnapPizzicato is a Bartok pizzicato on a cello: the string pulled up and let go so that it snaps
// against the fingerboard
func SnapPizzicato() Config {
	c := plucked(0.9, 0.35, 0.7, 0.2, 0.08, CelloBody, 0.55, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 1.5e-4, 0.05, 0.4
	c.TensionMod, c.FretBuzz, c.FretPosition, c.FretNoise = 1, 0.9, 0.93, 0.3
	return lvl(c, 1.302)
}

// ColLegno is the wood of the bow struck on a violin string: a dry, short, clicking tone, hardly
// any ring
func ColLegno() Config {
	c := plucked(0.22, 0.65, 0.9, 0.12, 0.05, ViolinBody, 0.5, 1)
	c.Inharmonicity, c.PluckWidth, c.Plectrum = 8e-5, 0.01, 0.9
	c.FretNoise, c.BodyCoupling = 0.5, 0.3
	return lvl(c, 1.448)
}

// ---- bowed

// ViolinSolo is a solo violin with a vibrato the wheel deepens and a harder bow with the velocity
func ViolinSolo() Config {
	c := rubbed(bowed(0.55, 0.12, 0.07, 0.15, 6, 0.2, 0.2, ViolinBody, 0.6, 1), 0.14)
	c.BowNoise = 0.1
	mod(&c, 0, modmatrix.Wheel, "vibratoDepth", 0.3)
	mod(&c, 1, modmatrix.Velocity, "bowForce", 0.1)
	return lvl(c, 1.244)
}

// ViolinSection is a section of violins played together: a slower start, a smaller and later
// vibrato, the grain of the rosin of many bows
func ViolinSection() Config {
	c := rubbed(bowed(0.55, 0.13, 0.28, 0.35, 5.2, 0.1, 0.45, ViolinBody, 0.75, 1), 0.12)
	c.BowNoise = 0.3
	return lvl(c, 1.623)
}

// Viola is a viola: the string longer and thicker, a darker, reedier bow, the vibrato slower
func Viola() Config {
	c := rubbed(bowed(0.58, 0.15, 0.1, 0.2, 5, 0.15, 0.25, ViolinBody, 0.5, 1), 0.12)
	c.BowNoise = 0.15
	return lvl(c, 1.165)
}

// Contrabass is a double bass bowed: slow to speak, big and woody, a heavy bow
func Contrabass() Config {
	c := rubbed(bowed(0.65, 0.17, 0.22, 0.35, 4.5, 0.08, 0.5, BassBody, 0.7, 1), 0.16)
	c.BowNoise = 0.1
	return lvl(c, 1.737)
}

// LegatoViolin is a violin that changes the note without a new bow: on a mono voice the next note
// glides in
func LegatoViolin() Config {
	c := rubbed(bowed(0.55, 0.125, 0.2, 0.25, 5.5, 0.15, 0.15, ViolinBody, 0.6, 1), 0.14)
	c.Slide = 0.12
	return lvl(c, 1.151)
}

// StaccatoViolin is the bow stopped on the string after each note: it speaks at once and stops short
func StaccatoViolin() Config {
	c := rubbed(bowed(0.6, 0.11, 0.02, 0.05, 0, 0, 0, ViolinBody, 0.55, 1), 0.16)
	c.BowNoise = 0.2
	return lvl(c, 1.162)
}

// Spiccato is a bow that bounces: a sharp speech, a dry rub, gone as soon as it comes
func Spiccato() Config {
	c := rubbed(bowed(0.6, 0.1, 0.008, 0.025, 0, 0, 0, ViolinBody, 0.5, 1), 0.18)
	c.BowNoise = 0.35
	return lvl(c, 1.251)
}

// TremoloStrings are the bow moved back and forth so fast that the sound shivers: the loudness
// swinging 10 times a second
func TremoloStrings() Config {
	c := rubbed(bowed(0.5, 0.11, 0.12, 0.2, 0, 0, 0, ViolinBody, 0.65, 1), 0.14)
	c.BowNoise = 0.25
	c.LFO2Rate = 10
	mod(&c, 0, modmatrix.LFO2, "level", -0.4)
	return lvl(c, 1.143)
}

// HurdyGurdy is a rosined wheel turned against the strings: a steady, nasal drone that rattles, a
// buzzing bridge, the drone strings sounding along
func HurdyGurdy() Config {
	c := bowed(0.8, 0.06, 0.05, 0.08, 0, 0, 0, ViolinBody, 0.5, 1)
	c.Sympathetic = 0.4
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 4000}
	return lvl(c, 0.755)
}

// Erhu is the Chinese two string fiddle: a thin, nasal, vocal tone with a vibrato that comes in late
func Erhu() Config {
	c := bowed(0.5, 0.09, 0.1, 0.2, 5.5, 0.28, 0.25, ViolinBody, 0.7, 1)
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 3500}
	mod(&c, 0, modmatrix.Wheel, "vibratoDepth", 0.3)
	return lvl(c, 1.264)
}

// Rebab is a bowed lute with a skin for a belly: a rough, breathy and nasal fiddle
func Rebab() Config {
	c := bowed(0.62, 0.12, 0.12, 0.25, 4.8, 0.15, 0.3, HarpBody, 0.55, 1)
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 2600}
	return lvl(c, 1.337)
}

// BowedGlass is a string that sings like a rubbed glass: a light bow far from the bridge, a slow
// swell and a slow fade, a shimmer of strings tuned to its octaves
func BowedGlass() Config {
	c := bowed(0.3, 0.22, 0.6, 0.9, 0, 0, 0, NoBody, 0, 1)
	c.Sympathetic = 0.3
	return lvl(c, 1.122)
}

// BowedPsaltery is a psaltery bowed on the edge: a small triangle of strings, thin and sweet
func BowedPsaltery() Config {
	c := bowed(0.4, 0.2, 0.09, 0.25, 0, 0, 0, HarpBody, 0.5, 1)
	return lvl(c, 1.738)
}

// CelloDrone is the open string of a cello bowed with a slow, steady stroke: a long, deep drone
func CelloDrone() Config {
	c := rubbed(bowed(0.6, 0.13, 0.5, 1, 0, 0, 0, CelloBody, 0.6, 1), 0.12)
	c.BowNoise = 0.1
	return lvl(c, 0.754)
}

// SingingSaw is a saw bowed on its blade: a thin, wavering tone that slides from note to note
func SingingSaw() Config {
	c := bowed(0.35, 0.3, 0.3, 0.4, 5.5, 0.5, 0.1, NoBody, 0, 1)
	c.Slide = 0.25
	c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 3500}
	return lvl(c, 1.492)
}

// ---- membranes

// KickDrum is a large drum struck hard: the pitch drops fast after the hit, a short, deep thump
func KickDrum() Config { return lvl(drum(0.5, 0.75, 0.15, 0.5, 1, 0, 0.05, 1), 0.7227) }

// TomHigh is a small rack tom: a short, round and ringing tone
func TomHigh() Config { return lvl(drum(0.55, 0.4, 0.4, 0.65, 0.35, 0, 0, 1), 0.7628) }

// TomLow is a mid tom, a stick on a loose head: long and dark
func TomLow() Config { return lvl(drum(0.7, 0.2, 0.55, 0.75, 0.3, 0, 0, 1), 0.7588) }

// CongaOpen is the open tone of a conga: the hand bounces off the rim, a clear ring
func CongaOpen() Config { return lvl(drum(0.55, 0.45, 0.8, 0.7, 0.15, 0, 0.2, 1), 0.8685) }

// CongaSlap is the slap: the fingers whipped across the head, bright and short
func CongaSlap() Config { return lvl(drum(0.2, 0.8, 1, 1, 0.1, 0, 0.05, 1), 1.763) }

// CongaBass is the bass tone: the palm in the middle of the head, a dull thud
func CongaBass() Config { return lvl(drum(0.65, 0.6, 0.15, 0.3, 0.45, 0, 0.1, 1), 0.8351) }

// Bongo is a pair of small drums on the knees: high, dry and tight
func Bongo() Config { return lvl(drum(0.28, 0.6, 0.65, 0.85, 0.2, 0, 0.08, 1), 0.9383) }

// Djembe is a goblet drum struck by the hand: a deep bass tone with a bright crack
func Djembe() Config { return lvl(drum(0.6, 0.3, 0.55, 0.7, 0.6, 0, 0.12, 1), 0.7673) }

// Bayan is the left drum of the tabla: a large head that bends far upwards under the heel of the hand
func Bayan() Config { return lvl(drum(1.2, 0.4, 0.15, 0.25, 1, 0, 0.3, 1), 1.067) }

// Dayan is the right drum of the tabla: high, with the black spot of the head that gives a clear,
// ringing tone
func Dayan() Config { return lvl(drum(1, 0.12, 0.55, 0.85, 0.1, 0, 0.3, 1), 0.7909) }

// Bodhran is the Irish goatskin frame drum struck by a tipper: a hollow, dry knock
func Bodhran() Config { return lvl(drum(0.35, 0.65, 0.3, 0.5, 0.2, 0, 0.1, 1), 0.7446) }

// Taiko is a huge Japanese drum with a thick skin struck by heavy sticks: it booms
func Taiko() Config { return lvl(drum(2.2, 0.5, 0.25, 0.35, 0.55, 0, 0, 1), 0.7711) }

// SnareTight is a small snare tuned high, rimmed and hit hard: a crack
func SnareTight() Config { return lvl(drum(0.18, 0.65, 0.45, 0.95, 0.2, 0.9, 0, 1), 0.5745) }

// SnareFat is a deep snare with loose wires: a thick, long backbeat
func SnareFat() Config { return lvl(drum(0.9, 0.3, 0.3, 0.5, 0.6, 0.6, 0, 1), 0.5722) }

// SnareBrush is a snare swept with brushes: a soft hit and a hush of the wires
func SnareBrush() Config { return lvl(drum(1.2, 0.97, 0.95, 0, 0, 1, 0.15, 1), 1.001) }

// Timbale is a shallow metal-shelled drum hit with a stick: tight, high and ringing
func Timbale() Config { return lvl(drum(0.4, 0.3, 0.8, 0.95, 0.15, 0.15, 0.1, 1), 0.8883) }

// Cajon is a wooden box sat on and hit by the hand: a bass thud with a snare-like slap
func Cajon() Config { return lvl(drum(0.15, 0.8, 0.25, 0.45, 0.45, 0.15, 0.05, 1), 0.8526) }

// Tambourine is a frame with a head and jingles: a short slap and a shimmer of metal
func Tambourine() Config { return lvl(drum(1.0, 0.3, 1, 0.3, 0.1, 1, 0.1, 1), 0.5412) }

// ---- pipes

// AltoFlute is a flute in G, a fourth lower: wide, dark and airy, soft to start
func AltoFlute() Config {
	c := blown(false, 0.66, 0.2, 0.3, 0.65, 0.12, 0.18, 4.5, 0.1, 0.4, 1)
	mod(&c, 0, modmatrix.Wheel, "breath", 0.3)
	return lvl(c, 0.173)
}

// Piccolo is the small flute that plays an octave over the flute: bright, thin, piercing
func Piccolo() Config {
	return lvl(blown(false, 0.65, 0.3, 0.6, 0.1, 0.04, 0.08, 5.5, 0.1, 0.3, 1), 0.1743)
}

// Shakuhachi is the Japanese bamboo flute with no reed or lip plate: very much air, a slow head
// shake for a vibrato and a note that swells
func Shakuhachi() Config {
	c := blown(false, 0.72, 0.85, 0.6, 0.8, 0.2, 0.25, 3, 0.3, 0.5, 1)
	mod(&c, 0, modmatrix.Wheel, "breath", 0.3)
	return lvl(c, 0.158)
}

// Ocarina is a vessel flute of clay: a pure, hooting tone with hardly any overtones
func Ocarina() Config {
	return lvl(blown(false, 0.6, 0.08, 0.3, 0.95, 0.06, 0.1, 5, 0.12, 0.3, 1), 0.3391)
}

// Recorder is a block flute: the air is led onto the edge by a duct so that it speaks at once
// and the breath hardly changes the tone
func Recorder() Config {
	c := blown(false, 0.6, 0.12, 0.35, 0.8, 0.03, 0.06, 0, 0, 0, 1)
	mod(&c, 0, modmatrix.Velocity, "breath", 0.2)
	return lvl(c, 0.172)
}

// BassClarinet is a clarinet an octave below: a deep, dark and woody reed
func BassClarinet() Config {
	return lvl(blown(true, 0.7, 0.12, 0.3, 0.8, 0.09, 0.14, 4, 0.05, 0.5, 1), 0.474)
}

// Chalumeau is the low register of the clarinet: hollow, rich and woody, a little rough
func Chalumeau() Config {
	return lvl(blown(true, 0.7, 0.1, 0.25, 0.4, 0.05, 0.1, 0, 0, 0, 1), 0.448)
}

// BreathyFlute is a flute blown with more air than tone: the sound of a whisper with a pitch to it
func BreathyFlute() Config {
	return lvl(blown(false, 0.55, 0.88, 0.45, 0.6, 0.1, 0.18, 4, 0.1, 0.4, 1), 0.287)
}

// OverblownFlute is a flute pushed to the edge of the next register: loud, edgy and unstable
func OverblownFlute() Config {
	return lvl(blown(false, 1, 0.45, 0.85, 0.5, 0.03, 0.08, 0, 0, 0, 1), 0.090)
}

// Whistle is a pure whistled tone with a wide vibrato
func Whistle() Config {
	c := blown(false, 0.62, 0.05, 0.25, 0.97, 0.05, 0.1, 6, 0.3, 0.2, 1)
	mod(&c, 0, modmatrix.Velocity, "breath", 0.2)
	return lvl(c, 0.193)
}

// Bansuri is the Indian bamboo flute: warm and breathy, with a slow swell
func Bansuri() Config {
	c := blown(false, 0.74, 0.3, 0.45, 0.45, 0.1, 0.2, 0, 0, 0, 1)
	mod(&c, 0, modmatrix.Wheel, "breath", 0.3)
	return lvl(c, 0.138)
}

// TinWhistle is the penny whistle of Irish music: a bright and reedy, slightly raw tone
func TinWhistle() Config {
	return lvl(blown(false, 0.66, 0.12, 0.2, 0.4, 0.02, 0.05, 0, 0, 0, 1), 0.166)
}
