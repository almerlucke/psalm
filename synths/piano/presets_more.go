package piano

// morePresets are the palette of pianos by instrument character, see the constructors below
var morePresets = map[string]func() Config{
	// concert grands
	"steinway":    SteinwayPreset,
	"bosendorfer": BosendorferPreset,
	"yamaha":      YamahaPreset,
	"fazioli":     FazioliPreset,
	// recording styles
	"closeMic":      CloseMicPreset,
	"roomAmbient":   RoomAmbientPreset,
	"popCompressed": PopCompressedPreset,
	"jazzMellow":    JazzMellowPreset,
	"jazzDry":       JazzDryPreset,
	"filmWide":      FilmWidePreset,
	// uprights
	"schoolUpright": SchoolUprightPreset,
	"tack":          TackPreset,
	"barPiano":      BarPianoPreset,
	"jangle":        JanglePreset,
	"ragtime":       RagtimePreset,
	// historic
	"fortepiano":     FortepianoPreset,
	"cristofori":     CristoforiPreset,
	"squarePiano":    SquarePianoPreset,
	"harpsichordish": HarpsichordishPreset,
	// bells and boxes
	"musicBox":    MusicBoxPreset,
	"celesta":     CelestaPreset,
	"glassHammer": GlassHammerPreset,
	// prepared and lo-fi
	"muffled":   MuffledPreset,
	"satie":     SatiePreset,
	"tapePiano": TapePianoPreset,
	// tunings, stereo, bridge, noise, pedals
	"stretchedWide": StretchedWidePreset,
	"equalTempered": EqualTemperedPreset,
	"widePiano":     WidePianoPreset,
	"monoPiano":     MonoPianoPreset,
	"bridgeRich":    BridgeRichPreset,
	"bridgeDry":     BridgeDryPreset,
	"silentAction":  SilentActionPreset,
	"noisyAction":   NoisyActionPreset,
	"pedalWash":     PedalWashPreset,
	"unaCorda":      UnaCordaPreset,
}

func init() {
	for n, f := range morePresets {
		Presets[n] = f
	}
}

// SteinwayPreset is the New York concert grand: brilliant, singing treble, a strong, clear bass
// with some bark, hammers voiced hard enough to cut through an orchestra
func SteinwayPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.62
	cfg.StrikePosition = 0.122
	cfg.Soundboard = 0.7
	cfg.Bark = 0.65
	cfg.Treble = 1.5
	cfg.Damping = 1.1
	cfg.Position = Hall
	cfg.Level = 16.1824 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// BosendorferPreset is the Vienna grand: dark and warm, a long ringing bass (extra strings and a
// resonant spruce case), soft hammers, a smooth rounded treble
func BosendorferPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.3
	cfg.StrikePosition = 0.135
	cfg.Soundboard = 0.9
	cfg.Body = 0.9
	cfg.Resonance = 0.85
	cfg.Damping = 1.9
	cfg.Decay = 1.3
	cfg.Treble = -3
	cfg.Bark = 0.3
	cfg.Position = Hall
	cfg.Level = 16.4204 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// YamahaPreset is the Japanese concert grand: clear, even and focused, little bark, every key the
// same as its neighbour, a clean sustain
func YamahaPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.5
	cfg.Variation = 0.15
	cfg.Bark = 0.2
	cfg.Unison = 0.5
	cfg.Soundboard = 0.55
	cfg.Damping = 1.0
	cfg.Mechanics = 0.3
	cfg.Treble = 0.5
	cfg.Position = Studio
	cfg.Level = 15.0069 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// FazioliPreset is the Italian grand: brilliant, glassy, a very long ring, the top octaves shining
// above everything, a long bass with a lot of resonance
func FazioliPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.72
	cfg.StrikePosition = 0.115
	cfg.Decay = 1.6
	cfg.Damping = 0.7
	cfg.Stiffness = 0.8
	cfg.Treble = 3
	cfg.Resonance = 0.75
	cfg.Soundboard = 0.65
	cfg.Position = Studio
	cfg.Level = 15.2339 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// CloseMicPreset is the grand with the microphones inches from the strings: tight, dry, every
// detail of the hammer and the damper
func CloseMicPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.55
	cfg.Position = Studio
	cfg.Soundboard = 0.4
	cfg.Body = 0.4
	cfg.Mechanics = 0.9
	cfg.KeyBed = 0.8
	cfg.Decay = 0.75
	cfg.DamperTime = 0.12
	cfg.Resonance = 0.3
	cfg.Variation = 0.9
	cfg.Level = 15.6975 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// RoomAmbientPreset is the grand in a big room: the hall around it, everything ringing along, the
// attack a little blurred
func RoomAmbientPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.35
	cfg.Position = Hall
	cfg.Soundboard = 0.95
	cfg.Body = 0.85
	cfg.Decay = 1.4
	cfg.Resonance = 1
	cfg.Bridge = 0.9
	cfg.Mechanics = 0.2
	cfg.DamperTime = 0.6
	cfg.Level = 16.9337 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// PopCompressedPreset is the piano of a pop record: bright, even in level whatever is played,
// short and punchy, with no room
func PopCompressedPreset() Config {
	cfg := BrightPreset()
	cfg.Hardness = 0.7
	cfg.VelocitySens = 0.2
	cfg.Decay = 0.6
	cfg.Damping = 1.2
	cfg.Soundboard = 0.4
	cfg.Variation = 0.1
	cfg.Mechanics = 0.15
	cfg.Treble = 2
	cfg.Position = Studio
	cfg.Level = 14.5881 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// JazzMellowPreset is a trio piano: round, soft hammers, a warm midrange and a relaxed treble
func JazzMellowPreset() Config {
	cfg := WarmPreset()
	cfg.Hardness = 0.2
	cfg.Damping = 1.2
	cfg.Treble = -7
	cfg.Bark = 0.2
	cfg.Decay = 0.9
	cfg.VelocitySens = 0.8
	cfg.Position = Player
	cfg.Level = 17.4031 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// JazzDryPreset is a dry club piano: a thick, woody midrange, very little ring, the dampers
// quick, no air around it
func JazzDryPreset() Config {
	cfg := GrandPreset()
	cfg.Hardness = 0.38
	cfg.Decay = 0.5
	cfg.Damping = 2.2
	cfg.DamperTime = 0.1
	cfg.Resonance = 0.2
	cfg.Body = 0.85
	cfg.Soundboard = 0.7
	cfg.Treble = -4
	cfg.Position = Player
	cfg.Level = 15.4391 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// FilmWidePreset is the cinema piano: soft hammers, the longest rooms, the keys spread over the
// whole stereo image
func FilmWidePreset() Config {
	cfg := CinematicPreset()
	cfg.Hardness = 0.12
	cfg.Width = 1
	cfg.Decay = 1.5
	cfg.Damping = 2.6
	cfg.DamperTime = 0.7
	cfg.Bridge = 0.85
	cfg.Treble = -2
	cfg.Level = 21.0418 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// SchoolUprightPreset is the upright of a music school: worn but kept, a little dull, a boxy
// cabinet, pleasantly out of tune
func SchoolUprightPreset() Config {
	cfg := UprightPreset()
	cfg.Hardness = 0.45
	cfg.Unison = 2.4
	cfg.Treble = -3
	cfg.Soundboard = 0.9
	cfg.Bark = 0.8
	cfg.Mechanics = 0.7
	cfg.Variation = 0.8
	cfg.Position = Player
	cfg.Level = 18.9657 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// TackPreset is the tack piano: thumbtacks in the hammers, a hard metallic plink, short and
// bright, the strings in tune with each other
func TackPreset() Config {
	cfg := UprightPreset()
	cfg.Hardness = 0.97
	cfg.Unison = 1.2
	cfg.Decay = 0.45
	cfg.Damping = 0.7
	cfg.StrikePosition = 0.1
	cfg.Treble = 2
	cfg.Soundboard = 0.5
	cfg.Level = 12.7406 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// BarPianoPreset is a barroom upright that nobody has tuned for years: strings of a key several cents
// apart, a worn action, dull and thin together
func BarPianoPreset() Config {
	cfg := HonkyTonkPreset()
	cfg.Hardness = 0.55
	cfg.Unison = 7
	cfg.Variation = 1
	cfg.Mechanics = 1
	cfg.Decay = 0.45
	cfg.Damping = 1.8
	cfg.Stretch = 0.2
	cfg.Treble = -2
	cfg.Level = 16.0226 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// JanglePreset is a jangle piano: strings tuned a wide, even distance apart, a wide chorus shimmer
// on a bright hammer
func JanglePreset() Config {
	cfg := HonkyTonkPreset()
	cfg.Hardness = 0.85
	cfg.Unison = 20
	cfg.Variation = 0.3
	cfg.Decay = 0.8
	cfg.Damping = 0.9
	cfg.Width = 0.9
	cfg.Level = 15.6423 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// RagtimePreset is the ragtime upright: honky, with tacks on the bass hammers only felt in
// the attack, brisk, dry, in the middle of the room
func RagtimePreset() Config {
	cfg := HonkyTonkPreset()
	cfg.Hardness = 0.88
	cfg.Unison = 4.5
	cfg.Decay = 0.5
	cfg.Damping = 1.0
	cfg.DamperTime = 0.1
	cfg.StrikePosition = 0.115
	cfg.Mechanics = 0.8
	cfg.Variation = 0.8
	cfg.Level = 12.0680 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// FortepianoPreset is the Mozart fortepiano: leather covered hammers, thin light strings with
// little ring, a nasal, clear sound and a fast damping
func FortepianoPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 0.78
	cfg.Strings = 2
	cfg.Stiffness = 0.6
	cfg.Decay = 0.45
	cfg.Damping = 2.2
	cfg.DamperTime = 0.08
	cfg.Soundboard = 0.4
	cfg.Body = 0.45
	cfg.StrikePosition = 0.09
	cfg.Stretch = 0.2
	cfg.Wound = 0.1
	cfg.Bark = 0.1
	cfg.Bridge = 0.3
	cfg.Resonance = 0.3
	cfg.Position = Player
	cfg.Level = 17.7782 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// CristoforiPreset is the first piano: hard thin hammers, one string for each key in the
// top, a clipped, plucked, almost harpsichord like attack
func CristoforiPreset() Config {
	cfg := FortepianoPreset()
	cfg.Hardness = 0.93
	cfg.Strings = 2
	cfg.Decay = 0.35
	cfg.Damping = 1.2
	cfg.StrikePosition = 0.075
	cfg.Soundboard = 0.3
	cfg.VelocitySens = 0.5
	cfg.Stretch = 0
	cfg.Level = 17.1677 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// SquarePianoPreset is the square piano of the early 19th century: a small box, a single or
// double string, a soft thin hammer, a tinkling, guitar like tone
func SquarePianoPreset() Config {
	cfg := FortepianoPreset()
	cfg.Hardness = 0.6
	cfg.Strings = 1
	cfg.Stiffness = 1.2
	cfg.Decay = 0.5
	cfg.Damping = 1.6
	cfg.Soundboard = 0.7
	cfg.Body = 0.3
	cfg.Treble = -4
	cfg.Width = 0.3
	cfg.Level = 23.2332 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// HarpsichordishPreset is a piano playing like a harpsichord: a plucked, bright, clipped sound,
// the dampers dropping at once, all notes the same loudness
func HarpsichordishPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 1
	cfg.Strings = 2
	cfg.Stiffness = 0.15
	cfg.StrikePosition = 0.18
	cfg.StrikeRegister = 0
	cfg.Decay = 0.55
	cfg.Damping = 0.4
	cfg.DamperTime = 0.04
	cfg.VelocitySens = 0.1
	cfg.Soundboard = 0.25
	cfg.Body = 0.3
	cfg.Unison = 2
	cfg.Stretch = 0
	cfg.Bridge = 0.2
	cfg.Resonance = 0.2
	cfg.Mechanics = 0.2
	cfg.Treble = -3
	cfg.Level = 14.0015 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// MusicBoxPreset is a music box played on piano strings: one thin string, an extremely hard hit
// in the middle, hollow, no body and no pedal
func MusicBoxPreset() Config {
	cfg := ToyPreset()
	cfg.Hardness = 1
	cfg.Decay = 0.9
	cfg.Damping = 0.3
	cfg.Soundboard = 0.1
	cfg.Body = 0.15
	cfg.Stiffness = 0.05
	cfg.Treble = -4
	cfg.DamperTime = 0.5
	cfg.Mechanics = 0
	cfg.Width = 0.6
	cfg.Level = 14.0042 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// CelestaPreset is a celesta: hard hammers on short strings, the soft chime of bars with a
// fast fade, a little mellow
func CelestaPreset() Config {
	cfg := ToyPreset()
	cfg.Hardness = 0.75
	cfg.StrikePosition = 0.3
	cfg.Strings = 2
	cfg.Stiffness = 0.7
	cfg.Decay = 0.7
	cfg.Damping = 1.8
	cfg.Soundboard = 0.5
	cfg.Body = 0.4
	cfg.Treble = -6
	cfg.Width = 0.5
	cfg.Mechanics = 0.1
	cfg.Level = 16.3270 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// GlassHammerPreset is a piano struck by glass: the hardest hammers on the thinnest stiffened
// strings, a glassy, inharmonic shimmer
func GlassHammerPreset() Config {
	cfg := DefaultConfig()
	cfg.Hardness = 1
	cfg.Strings = 1
	cfg.Stiffness = 3
	cfg.Decay = 1.2
	cfg.Damping = 0.5
	cfg.StrikePosition = 0.4
	cfg.StrikeRegister = 0
	cfg.Soundboard = 0.3
	cfg.Treble = -8
	cfg.Wound = 0
	cfg.Bark = 0
	cfg.Precursor = 0
	cfg.Level = 19.0709 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// MuffledPreset is a piano with a blanket over the strings: the highs gone at once, a thud
// with a pitch
func MuffledPreset() Config {
	cfg := FeltPreset()
	cfg.Hardness = 0
	cfg.Damping = 4
	cfg.Decay = 0.4
	cfg.DamperTime = 0.1
	cfg.Soundboard = 0.7
	cfg.Treble = -6
	cfg.Resonance = 0.1
	cfg.Level = 21.9665 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// SatiePreset is the intimate piano of quiet piano music: soft, a close microphone, the
// mechanics breathing, the strings ringing a little
func SatiePreset() Config {
	cfg := FeltPreset()
	cfg.Hardness = 0.15
	cfg.Damping = 1.8
	cfg.Decay = 1.0
	cfg.Mechanics = 1
	cfg.KeyBed = 0.8
	cfg.Variation = 0.9
	cfg.Soundboard = 0.8
	cfg.Width = 0.5
	cfg.Level = 19.2236 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// TapePianoPreset is a piano that has been through an old tape: detuned strings, a muffled top,
// every key a little different
func TapePianoPreset() Config {
	cfg := VintagePreset()
	cfg.Hardness = 0.3
	cfg.Unison = 6
	cfg.Variation = 1
	cfg.Treble = -9
	cfg.Damping = 2.8
	cfg.Decay = 0.7
	cfg.Stretch = 0.3
	cfg.Mechanics = 0.5
	cfg.Level = 21.9063 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// StretchedWidePreset is a grand tuned wider than a tuner would: the bass flat and the treble
// sharp against equal temperament
func StretchedWidePreset() Config {
	cfg := GrandPreset()
	cfg.Stretch = 1.9
	cfg.Level = 16.1211 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// EqualTemperedPreset is the grand tuned to equal temperament with no stretch and no
// inharmonicity: the partials are exact, the sound a little pure
func EqualTemperedPreset() Config {
	cfg := GrandPreset()
	cfg.Stretch = 0
	cfg.Stiffness = 0.1
	cfg.Level = 18.1998 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// WidePianoPreset is the grand with the keys spread over the whole stereo image, as the player
// hears bass and treble
func WidePianoPreset() Config {
	cfg := GrandPreset()
	cfg.Width = 1
	cfg.Position = Player
	cfg.Level = 16.3301 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// MonoPianoPreset is a piano all in the middle: the keys in one place, the way a small
// speaker plays it
func MonoPianoPreset() Config {
	cfg := GrandPreset()
	cfg.Width = 0
	cfg.Position = Hall
	cfg.Level = 17.3816 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// BridgeRichPreset is a piano whose strings are strongly coupled by the bridge: chords bloom and
// the notes lose their energy to each other and to the board
func BridgeRichPreset() Config {
	cfg := GrandPreset()
	cfg.Bridge = 1
	cfg.Wound = 1
	cfg.Precursor = 1
	cfg.Bark = 0.9
	cfg.Soundboard = 0.85
	cfg.Level = 16.8591 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// BridgeDryPreset is a piano whose keys do not know of each other: plain strings all over, no
// bridge coupling, no precursor
func BridgeDryPreset() Config {
	cfg := GrandPreset()
	cfg.Bridge = 0
	cfg.Wound = 0
	cfg.Precursor = 0
	cfg.Bark = 0.1
	cfg.Soundboard = 0.4
	cfg.Level = 14.7605 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// SilentActionPreset is a grand without a trace of the machine: no keys, dampers, pedals
func SilentActionPreset() Config {
	cfg := GrandPreset()
	cfg.Mechanics = 0
	cfg.KeyBed = 0
	cfg.Level = 16.1485 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// NoisyActionPreset is a grand with the machine in front: thuds of the keys on the keybed, the
// dampers, the pedal
func NoisyActionPreset() Config {
	cfg := GrandPreset()
	cfg.Mechanics = 1
	cfg.KeyBed = 1
	cfg.Position = Studio
	cfg.Level = 16.0097 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// PedalWashPreset is the sustain pedal always down: the keys ring on a long time after they
// are released, every string answers
func PedalWashPreset() Config {
	cfg := GrandPreset()
	cfg.DamperTime = 2
	cfg.Resonance = 1
	cfg.Decay = 1.8
	cfg.Bridge = 0.9
	cfg.Damping = 1.5
	cfg.Position = Hall
	cfg.Level = 17.4913 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}

// UnaCordaPreset is the soft pedal always down: two strings of three, a softer part of the
// felt, veiled and gentle
func UnaCordaPreset() Config {
	cfg := GrandPreset()
	cfg.Strings = 2
	cfg.Hardness = 0.25
	cfg.Treble = -3
	cfg.VelocitySens = 0.8
	cfg.Level = 21.6699 // calibrated: a -1 dB peak at amplitude 1
	return cfg
}
