package modal

import "github.com/almerlucke/psalm/blocks/modmatrix"

func init() {
	for name, f := range morePresets {
		Presets[name] = f
	}
}

// morePresets are the registrations of presets_more.go
var morePresets = map[string]func() Config{
	"softMarimba":     SoftMarimba,
	"hardMarimba":     HardMarimba,
	"marimbaEdge":     MarimbaEdge,
	"bowedVibraphone": BowedVibraphone,
	"vibeMotorFast":   VibeMotorFast,
	"vibeDamped":      VibeDamped,
	"xyloWarm":        XyloWarm,
	"toyXylophone":    ToyXylophone,
	"musicBox":        MusicBox,
	"kalimbaSoft":     KalimbaSoft,
	"bellMarimba":     BellMarimba,
	"vibeBell":        VibeBell,
	"handbellSoft":    HandbellSoft,
	"handbellChoir":   HandbellChoir,
	"templeBell":      TempleBell,
	"carillon":        Carillon,
	"gamelanSaron":    GamelanSaron,
	"gamelanGender":   GamelanGender,
	"gongAgeng":       GongAgeng,
	"tamTam":          TamTam,
	"gongSoft":        GongSoft,
	"bowedGong":       BowedGong,
	"tunedBowlHigh":   TunedBowlHigh,
	"tunedBowlDeep":   TunedBowlDeep,
	"bowedBowl":       BowedBowl,
	"glassHarmonica":  GlassHarmonica,
	"bottleBow":       BottleBow,
	"bottleTap":       BottleTap,
	"steelPanSoft":    SteelPanSoft,
	"steelPanBright":  SteelPanBright,
	"tubularSoft":     TubularSoft,
	"doorChime":       DoorChime,
	"triangle":        Triangle,
	"anvil":           Anvil,
	"crashCymbal":     CrashCymbal,
	"rideCymbal":      RideCymbal,
	"hihatClosed":     HihatClosed,
	"hihatOpen":       HihatOpen,
	"brushedCymbal":   BrushedCymbal,
	"claves":          Claves,
	"slitDrum":        SlitDrum,
	"thunderSheet":    ThunderSheet,
	"wheelVibe":       WheelVibe,
	"keyedBell":       KeyedBell,
	"wobbleBowl":      WobbleBowl,
}

// SoftMarimba is a marimba played with soft yarn mallets, heavy and slow: round, almost without attack
func SoftMarimba() Config {
	return Config{Instrument: InstMarimbaBar, Decay: 2.2, Resonator: 0.8, Hardness: 0.12, Position: 0.45,
		Contact: 0.5, Mass: 0.7, Noise: 0.05, Level: 0.3965}
}

// HardMarimba is a marimba struck with hard rubber, near the end of the bar: clacking, bright, short
func HardMarimba() Config {
	return Config{Instrument: InstMarimbaBar, Decay: 1.1, Resonator: 0.4, HighDamping: 0.3, Hardness: 0.95, Position: 0.18,
		Contact: 0.2, Mass: 0.05, Noise: 0.45, Level: 1.727}
}

// MarimbaEdge is a marimba struck almost at its end, where the fundamental hardly sounds: thin and woody
func MarimbaEdge() Config {
	return Config{Instrument: InstMarimbaBar, Decay: 0.9, Resonator: 0.2, Hardness: 0.7, Position: 0.06,
		Contact: 0.3, Noise: 0.35, Undercut: 0.5, Level: 0.5159}
}

// BowedVibraphone is a vibraphone bar drawn with a bow: the note swells in and rings as long as it is held
func BowedVibraphone() Config {
	return Config{Instrument: InstVibraphoneBar, Excitation: Bow, Decay: 5, Resonator: 0.7, Release: 1.2,
		Position: 0.4, Tremolo: 4.5, TremoloDepth: 0.5, Level: 0.3812}
}

// VibeMotorFast is a vibraphone with the motor at full speed, a deep fast tremolo
func VibeMotorFast() Config {
	return Config{Instrument: InstVibraphoneBar, Decay: 7, Resonator: 0.9, Hardness: 0.35, Position: 0.4,
		Mass: 0.4, Release: 0.3, Tremolo: 9, TremoloDepth: 1, Noise: 0.03, Level: 0.2857}
}

// VibeDamped is a vibraphone with the pedal down hard, hard mallets: a short, dry, bright note
func VibeDamped() Config {
	return Config{Instrument: InstVibraphoneBar, Decay: 1.4, Resonator: 0.1, Release: 0.08, Hardness: 0.9,
		Position: 0.3, Contact: 0.1, Noise: 0.2, Level: 0.6958}
}

// XyloWarm is a rosewood xylophone with medium mallets, a little undercut: mellow, a short ring
func XyloWarm() Config {
	return Config{Instrument: InstXylophoneBar, Decay: 1.2, Hardness: 0.55, Position: 0.45, Mass: 0.3,
		Undercut: 0.6, Resonator: 0.3, Noise: 0.1, Level: 0.5441}
}

// ToyXylophone is a child's xylophone: hard, thin, clicking
func ToyXylophone() Config {
	return Config{Instrument: InstXylophoneBar, Decay: 0.35, HighDamping: 0.4, Hardness: 1, Position: 0.15,
		Contact: 0.1, Noise: 0.6, Inharmonicity: 0.2, Level: 1.322}
}

// MusicBox is a comb of steel tines plucked by a pin: high, thin, ringing for a few seconds
func MusicBox() Config {
	return Config{Instrument: InstTineBar, Decay: 4, Hardness: 0.85, Position: 0.99, Contact: 0.1,
		Inharmonicity: 0.5, Noise: 0.08, Beat: 0.004, Level: 0.3947}
}

// KalimbaSoft is a thumb piano played with the pad of the thumb, near the base of the tine: dull and woody
func KalimbaSoft() Config {
	return Config{Instrument: InstTineBar, Decay: 1.2, HighDamping: 0.6, Hardness: 0.2, Position: 0.55,
		Mass: 0.5, Noise: 0.3, Level: 1.464}
}

// BellMarimba is a mallet bell hybrid: the marimba bar's 1 : 4 : 10 with the partials of a bar bell
// above, struck with a medium mallet
func BellMarimba() Config {
	return Config{
		Modes: []Mode{{1, 1, 1}, {4, 0.7, 0.7}, {10, 0.4, 0.5}, {2.76, 0.5, 1.4}, {5.4, 0.35, 1.2}, {8.93, 0.25, 1},
			{13.3, 0.15, 0.8}},
		Decay: 3, HighDamping: 0.15, Hardness: 0.6, Position: 0.35, Beat: 0.004, Noise: 0.15, Level: 0.5146}
}

// VibeBell is a vibraphone struck like a bell: long modes in a stretched stack, the tremolo slow
func VibeBell() Config {
	return Config{
		Modes: []Mode{{1, 1, 1}, {3.9, 0.8, 1}, {9.8, 0.5, 0.8}, {17.2, 0.3, 0.6}},
		Decay: 8, HighDamping: 0.1, Hardness: 0.7, Position: 0.3, Tremolo: 3, TremoloDepth: 0.4, Beat: 0.003,
		Inharmonicity: 0.15, Noise: 0.05, Level: 0.6118}
}

// HandbellSoft is a handbell with a soft clapper
func HandbellSoft() Config {
	return Config{Instrument: InstHandbell, Decay: 5, Hardness: 0.35, Position: 0.9, Spread: 1.5, Mass: 0.5,
		Noise: 0.05, Level: 0.5395}
}

// HandbellChoir is handbells with the pairs split wide, ringing together: a shimmering, beating chime
func HandbellChoir() Config {
	return Config{Instrument: InstHandbell, Decay: 10, HighDamping: 0.4, Hardness: 0.2, Position: 0.3, Spread: 4,
		Tremolo: 3, TremoloDepth: 0.8, Mass: 0.8, Noise: 0.1, Level: 1.656}
}

// TempleBell is a big bronze bell struck by a log: a very long hum under the strike, the pairs beating slowly
func TempleBell() Config {
	return Config{Instrument: InstChurchBell, Decay: 14, HighDamping: 0.2, Hardness: 0.25, Position: 0.55,
		Spread: 0.4, Mass: 1, Noise: 0.3, Level: 0.735}
}

// Carillon is the small bells of a tower: the church bell tuned high, hard and quick to die
func Carillon() Config {
	return Config{Instrument: InstChurchBell, Decay: 3.5, HighDamping: 0.5, Hardness: 1, Position: 0.95,
		Spread: 2, Contact: 0.6, Noise: 0.1, Level: 0.2521}
}

// GamelanSaron is a bronze bar with a hard beater, the pairs tuned to beat: bright and shimmering
func GamelanSaron() Config {
	return Config{
		Modes: []Mode{{1, 1, 1}, {2.7, 0.6, 0.6}, {5.2, 0.4, 0.4}, {8.6, 0.25, 0.3}, {12.9, 0.12, 0.2}},
		Decay: 3, HighDamping: 0.25, Hardness: 0.5, Position: 0.6, Beat: 0.03, Noise: 0.2, Level: 0.6959}
}

// GamelanGender is a gender: thin keys over resonators, a soft disc beater, a low, humming bloom
func GamelanGender() Config {
	return Config{
		Modes: []Mode{{1, 1, 1}, {2, 0.35, 0.8}, {3.9, 0.3, 0.5}, {6.1, 0.12, 0.3}},
		Decay: 6, HighDamping: 0.3, Hardness: 0.25, Position: 0.45, Mass: 0.6, Beat: 0.012, Resonator: 0.9,
		Noise: 0.05, Level: 0.7005}
}

// GongAgeng is a great gong ageng: huge, low, a soft beater and a long bloom
func GongAgeng() Config {
	return Config{Instrument: InstGong, Decay: 14, HighDamping: 0.7, Hardness: 0.15, Position: 0.08, Contact: 0.8, Spread: 1.5,
		Mass: 1, Nonlinearity: 0.35, Noise: 0.05, Level: 0.6806}
}

// TamTam is an orchestral tam-tam hit hard: it roars up from the low modes into a wash
func TamTam() Config {
	return Config{Instrument: InstGong, Decay: 8, HighDamping: 0.3, Hardness: 1, Position: 0.95, Spread: 4, Mass: 0.05,
		Contact: 1, Nonlinearity: 1, Noise: 0.25, Level: 0.08474}
}

// GongSoft is a small gong tapped with a soft mallet: a low gong note and little else
func GongSoft() Config {
	return Config{Instrument: InstGong, Decay: 8, HighDamping: 1, Hardness: 0.05, Position: 0.15, Mass: 0.2,
		Nonlinearity: 0.1, Noise: 0.02, Level: 0.5425}
}

// BowedGong is a gong drawn with a bow on its edge: a slow swell of metal
func BowedGong() Config {
	return Config{Instrument: InstGong, Excitation: Bow, Decay: 8, HighDamping: 0.4, Position: 0.9, Spread: 1,
		Release: 2, Level: 0.3755}
}

// TunedBowlHigh is a small brass bowl struck sharply on the rim: clear and high, ringing a long time
func TunedBowlHigh() Config {
	return Config{Instrument: InstSingingBowl, Decay: 9, Hardness: 0.7, Position: 0.95, Spread: 0.5, Contact: 0.4,
		Noise: 0.1, Level: 0.4227}
}

// TunedBowlDeep is a big bowl hit with a padded striker on its side: low, warm, slowly beating
func TunedBowlDeep() Config {
	return Config{Instrument: InstSingingBowl, Decay: 14, HighDamping: 0.3, Hardness: 0.05, Position: 0.6, Spread: 2,
		Mass: 1, Noise: 0.02, Level: 2.001}
}

// BowedBowl is a singing bowl sounded by rubbing the rim: it swells and sings as long as it is held
func BowedBowl() Config {
	return Config{Instrument: InstSingingBowl, Excitation: Bow, Decay: 10, Position: 0.95, Spread: 1,
		Release: 3, Tremolo: 0.7, TremoloDepth: 0.3, Level: 0.892}
}

// GlassHarmonica is wet glass bowls rubbed with a finger: pure, high, floating
func GlassHarmonica() Config {
	return Config{
		Modes:      []Mode{{1, 1, 1}, {2.32, 0.3, 0.9}, {4.25, 0.12, 0.8}, {6.63, 0.05, 0.6}},
		Excitation: Bow, Decay: 4, HighDamping: 0.1, Position: 0.3, Beat: 0.003, Inharmonicity: 0.1, Release: 1.5,
		Level: 0.9067}
}

// BottleBow is a bottle sounded by a bow on the glass: the resonance of the body swells and trembles
func BottleBow() Config {
	return Config{Instrument: InstBottle, Excitation: Bow, Decay: 2.5, Release: 0.6, Tremolo: 5.5,
		TremoloDepth: 0.15, Level: 0.8248}
}

// BottleTap is a bottle tapped with a knuckle, soft: a dull, hollow plonk
func BottleTap() Config {
	return Config{Instrument: InstBottle, Decay: 0.4, HighDamping: 0.6, Hardness: 0.15, Mass: 0.5, Noise: 0.5,
		Level: 1.27}
}

// SteelPanSoft is a steel pan played with rubber-tipped sticks, rolling in a slow shimmer
func SteelPanSoft() Config {
	return Config{Instrument: InstSteelPan, Decay: 3.5, Hardness: 0.25, Position: 0.4, Mass: 0.5, Beat: 0.006,
		Tremolo: 6, TremoloDepth: 0.2, Noise: 0.08, Level: 0.8005}
}

// SteelPanBright is a steel pan hit hard near the rim of the note: bright and short
func SteelPanBright() Config {
	return Config{Instrument: InstSteelPan, Decay: 0.8, HighDamping: 0.8, Hardness: 1, Position: 0.6,
		Contact: 0.1, Noise: 0.35, Beat: 0.002, Level: 0.6865}
}

// TubularSoft is chimes struck with a leather hammer: the strike tone mellow, ringing and ringing
func TubularSoft() Config {
	return Config{Instrument: InstTubularBell, Decay: 10, Hardness: 0.2, Position: 0.12, Mass: 0.6, Release: 1.5,
		Noise: 0.08, Level: 0.9043}
}

// DoorChime is a short tube chime: hard struck, a quick two second ring
func DoorChime() Config {
	return Config{Instrument: InstTubularBell, Decay: 2, HighDamping: 0.4, Hardness: 1, Position: 0.04,
		Contact: 0.1, Noise: 0.4, Release: 0.3, Level: 0.4038}
}

// Triangle is a steel triangle, bright and inharmonic, shimmering at the top
func Triangle() Config {
	return Config{
		Modes: []Mode{{1, 0.6, 1}, {2.71, 1, 1.1}, {4.82, 0.9, 0.9}, {7.6, 0.8, 0.8}, {10.9, 0.6, 0.6}, {14.7, 0.4, 0.5}},
		Decay: 4, HighDamping: 0.05, Hardness: 1, Position: 0.33, Beat: 0.01, Noise: 0.1, Level: 0.4582}
}

// Anvil is a steel plate hit with a hammer: a hard, ringing clang
func Anvil() Config {
	return Config{Instrument: InstPlate, Decay: 2.5, HighDamping: 0.3, Hardness: 1, Position: 0.3, Spread: 2,
		Contact: 0.1, Mass: 0.5, Noise: 0.5, Level: 0.3007}
}

// CrashCymbal is a crash: struck hard, the wash blooming and fading over four seconds
func CrashCymbal() Config {
	return Config{Instrument: InstCymbal, Decay: 6, HighDamping: 0, Hardness: 1, Position: 0.8, Spread: 3,
		Contact: 0.3, Nonlinearity: 1, Noise: 0.4, Level: 0.1267}
}

// RideCymbal is a ride struck on the bow with a stick: a ping over a soft, steady wash
func RideCymbal() Config {
	return Config{Instrument: InstCymbal, Decay: 3, HighDamping: 0.2, Hardness: 0.8, Position: 0.25, Spread: 1,
		Contact: 0.8, Nonlinearity: 0.25, Noise: 0.2, Level: 0.1225}
}

// HihatClosed is a pair of cymbals pressed together: a tick, a breath of hiss
func HihatClosed() Config {
	return Config{Instrument: InstCymbal, Decay: 0.12, HighDamping: 0.8, Hardness: 1, Position: 0.5, Spread: 3,
		Contact: 0.1, Noise: 0.7, Level: 0.2299}
}

// HihatOpen is the same pair loose: a sizzle that rings a second
func HihatOpen() Config {
	return Config{Instrument: InstCymbal, Decay: 0.9, HighDamping: 0.6, Hardness: 1, Position: 0.5, Spread: 3,
		Contact: 0.1, Nonlinearity: 0.3, Noise: 0.6, Level: 0.1071}
}

// BrushedCymbal is a cymbal swept with a brush: a burst of noise into the modes, a soft swish
func BrushedCymbal() Config {
	return Config{Instrument: InstCymbal, Excitation: Burst, Decay: 1.8, HighDamping: 0.3, Hardness: 0.2,
		Position: 0.6, Spread: 2, Level: 0.3405}
}

// Claves are two sticks of hardwood knocked together: a bright, dry tock
func Claves() Config {
	return Config{Instrument: InstBar, Decay: 0.18, HighDamping: 1.2, Hardness: 1, Position: 0.22, Contact: 0.2,
		Inharmonicity: 0.1, Noise: 0.2, Level: 1.436}
}

// SlitDrum is a hollow log struck with a padded stick: a deep, round knock
func SlitDrum() Config {
	return Config{Instrument: InstTempleBlock, Decay: 0.7, HighDamping: 0.5, Hardness: 0.2, Position: 0.5,
		Mass: 0.8, Noise: 0.15, Level: 1}
}

// ThunderSheet is a sheet of steel shaken: a burst of noise into a plate, rumbling
func ThunderSheet() Config {
	return Config{Instrument: InstPlate, Excitation: Burst, Decay: 4, HighDamping: 0.1, Hardness: 0.1,
		Position: 0.5, Spread: 3, Nonlinearity: 0.6, Level: 0.4309}
}

// WheelVibe is a vibraphone whose motor the mod wheel turns on: the wheel brings in the tremolo and hardens the mallet
func WheelVibe() Config {
	c := Vibraphone()
	c.Decay, c.Hardness, c.Position, c.Tremolo, c.TremoloDepth = 9, 0.5, 0.2, 6.5, 0
	c.Level = 1.54
	c.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "tremoloDepth", Amount: 1}
	c.Mods[1] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "hardness", Amount: 0.4}
	return c
}

// KeyedBell is a tubular bell whose ring is shorter the higher the note, the key moving where it is struck
func KeyedBell() Config {
	c := TubularBells()
	c.Position, c.Decay, c.Hardness = 0.1, 4, 0.3
	c.Level = 0.3534
	c.Mods[0] = modmatrix.Slot{Src: modmatrix.Key, Dest: "decay", Amount: -1}
	c.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "position", Amount: 0.6}
	c.Mods[2] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "noise", Amount: 0.5}
	return c
}

// WobbleBowl is a singing bowl whose pitch the slow LFO sways, a few cents: a bowl on a rocking boat
func WobbleBowl() Config {
	c := SingingBowl()
	c.Decay, c.LFORate = 12, 0.4
	c.Level = 0.7675
	c.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "pitch", Amount: 0.06}
	return c
}
