package westcoast

import "github.com/almerlucke/psalm/blocks/modmatrix"

// more is the second set of presets: plucks, gongs, percussion, bells, drones, pads, wobbles, basses,
// leads, cycling arps, effects and vowels
func init() {
	for n, f := range morePresets {
		Presets[n] = f
	}
}

var morePresets = map[string]func() Config{
	"stringPluck":  StringPluck,
	"kotoPluck":    KotoPluck,
	"rubberPluck":  RubberPluck,
	"amPluck":      AMPluck,
	"gongDeep":     GongDeep,
	"templeGong":   TempleGong,
	"tamTam":       TamTam,
	"conga":        Conga,
	"tom":          Tom,
	"woodBlock":    WoodBlock,
	"metalHat":     MetalHat,
	"glocken":      Glocken,
	"tubularBell":  TubularBell,
	"fmBell":       FMBell,
	"dxChime":      DXChime,
	"ringBell":     RingBell,
	"subDrone":     SubDrone,
	"choirDrone":   ChoirDrone,
	"shimmerDrone": ShimmerDrone,
	"foldPad":      FoldPad,
	"glassPad":     GlassPad,
	"breathingPad": BreathingPad,
	"wubWobble":    WubWobble,
	"dubWobble":    DubWobble,
	"subPluck":     SubPluck,
	"foldBass":     FoldBass,
	"fmBass":       FMBass,
	"reeseWest":    ReeseWest,
	"sineLead":     SineLead,
	"foldLead":     FoldLead,
	"ringLead":     RingLead,
	"cycleArp":     CycleArp,
	"bubbleArp":    BubbleArp,
	"pulseSeq":     PulseSeq,
	"swellArp":     SwellArp,
	"randomArp":    RandomArp,
	"alienPing":    AlienPing,
	"tractorBeam":  TractorBeam,
	"radioChatter": RadioChatter,
	"voxFold":      VoxFold,
	"ohFold":       OhFold,
	"eeFold":       EeFold,
}

// StringPluck is a struck string: a light FM an octave up, the lamp hit at once and let go over
// a hundred milliseconds, a faint ring
func StringPluck() Config {
	return Config{ModRatio: 2, FM: 0.3, EnvToFM: 1.2, Decay: 0.12, Gate: Combo, Resonance: 0.15, Response: 0.5, Color: 0.8,
		Strike: 0.6, StrikeDecay: 0.08, KeyTrack: 0.8, Level: 1.0793}
}

// KotoPluck is a bright, plinking pluck with a high gate resonance: a pitched click and a short ring
func KotoPluck() Config {
	return Config{ModRatio: 1.5, FM: 0.15, EnvToFM: 2, Fold: 0.1, EnvToFold: 0.4, Decay: 0.06, Gate: Combo, Resonance: 0.4,
		Response: 0.6, Color: 1, Strike: 1, StrikeDecay: 0.15, KeyTrack: 1, Level: 0.7196}
}

// RubberPluck is a boing: the modulator an octave down and the index falling fast, so the pitch
// seems to sag as the string goes slack
func RubberPluck() Config {
	return Config{ModRatio: 0.5, EnvToFM: 4, Decay: 0.05, Gate: Combo, Resonance: 0.5, Response: 0.3, Color: 0.9,
		Strike: 0.7, StrikeDecay: 0.05, KeyTrack: 0.5, Level: 1.2492}
}

// AMPluck is a hollow pluck: amplitude modulation by a modulator just above the note, which makes it
// ring like a metal tine
func AMPluck() Config {
	return Config{ModRatio: 2.01, AM: 0.7, FM: 0.1, Decay: 0.1, Gate: Combo, Resonance: 0.3, Response: 0.7, Color: 0.9,
		Strike: 0.5, StrikeDecay: 0.1, Level: 1.4531}
}

// GongDeep is a big gong: heavy FM at a fifth and a second oscillator a hair off, the lamp struck hard
// and fading over seconds
func GongDeep() Config {
	return Config{ModRatio: 1.5, FM: 1, EnvToFM: 2, Fold: 0.3, EnvToFold: 0.4, Osc2Mix: 0.5, Osc2Detune: 0.3, Osc2ModRatio: 1.01,
		Decay: 0.9, Gate: Combo, Resonance: 0.15, Response: 2.2, Color: 0.8, Strike: 1, StrikeDecay: 1.5, Level: 0.5290}
}

// TempleGong is the inharmonic partials of a bar: a modulator at 2.76 times the note and a second
// oscillator a tritone off, a long shimmering ring
func TempleGong() Config {
	return Config{ModRatio: 2.76, FM: 0.8, EnvToFM: 1, Osc2Mix: 0.6, Osc2Detune: -6.8, Osc2ModRatio: 1.41, Decay: 1.2,
		Gate: Combo, Response: 2.2, Color: 0.9, Strike: 0.8, StrikeDecay: 1.2, Symmetry: 0.1, Level: 0.7930}
}

// TamTam is a wash of metal: square modulators at clashing ratios on two oscillators, folded hard,
// the gate slow to let go
func TamTam() Config {
	return Config{ModWave: ModSquare, ModRatio: 1.414, FM: 2, Fold: 0.5, EnvToFold: 0.4, Osc2Mix: 1, Osc2Detune: 1.7,
		Osc2ModWave: ModSaw, Osc2ModRatio: 1.7, Decay: 1, Gate: Combo, Resonance: 0.1, Response: 2.5, Color: 1, Strike: 0.8,
		StrikeDecay: 2, Level: 0.5743}
}

// Conga is a hand drum, a little less ringing than the bongo: a slap of FM on the strike
func Conga() Config {
	return Config{ModRatio: 1.5, FM: 0.1, EnvToFM: 0.8, Decay: 0.03, Gate: Combo, Resonance: 0.6, Response: 0.2, Color: 0.5,
		Strike: 0.5, StrikeDecay: 0.04, KeyTrack: 0.3, Level: 6.6373}
}

// Tom is a low tom: a dull, fat thump with the modulator below the note, dying fast
func Tom() Config {
	return Config{ModRatio: 0.5, FM: 0.2, EnvToFM: 1, Decay: 0.04, Gate: Combo, Resonance: 0.4, Response: 0.35, Color: 0.4,
		Strike: 1, StrikeDecay: 0.1, KeyTrack: 0.2, Level: 1.5408}
}

// WoodBlock is a hard click that rings for a moment in the highs: an inharmonic ratio, the gate
// resonating and gone in a few milliseconds
func WoodBlock() Config {
	return Config{ModRatio: 2.4, FM: 0.5, EnvToFM: 1, Decay: 0.015, Gate: Combo, Resonance: 0.95, Response: 0.08, Color: 1,
		Strike: 0.5, StrikeDecay: 0.02, KeyTrack: 1, Level: 2.9100}
}

// MetalHat is a hi-hat of two square oscillators at clashing ratios, folded to noise and gone at once
func MetalHat() Config {
	return Config{ModWave: ModSquare, ModRatio: 1.414, FM: 3, Fold: 0.8, Osc2Mix: 1, Osc2Detune: 6.2, Osc2ModWave: ModSquare,
		Osc2ModRatio: 1.5, Decay: 0.03, Gate: Combo, Response: 0.2, Color: 1, Strike: 1, StrikeDecay: 0.06, KeyTrack: 0.2, Level: 4.0742}
}

// Glocken is a glockenspiel: a plain high FM and a short, bright ring
func Glocken() Config {
	return Config{ModRatio: 3.5, FM: 0.3, EnvToFM: 1, Decay: 0.25, Gate: Combo, Response: 1.2, Color: 1, Strike: 0.5,
		StrikeDecay: 0.3, KeyTrack: 1, Level: 1.1078}
}

// TubularBell is a church bell: a second oscillator an octave up with its own inharmonic modulator,
// a slow strike and a long gate
func TubularBell() Config {
	return Config{ModRatio: 1.4, FM: 0.8, EnvToFM: 1, Osc2Mix: 0.5, Osc2Detune: 12.1, Osc2ModRatio: 2.76, Decay: 0.6,
		Gate: Combo, Response: 1.4, Color: 0.9, Strike: 0.8, StrikeDecay: 0.8, Level: 1.2234}
}

// FMBell is a classic FM bell: a one to 3.5 ratio with the index starting high and falling away
func FMBell() Config {
	return Config{ModRatio: 3.5, FM: 0.2, EnvToFM: 3, Decay: 0.4, Gate: Combo, Resonance: 0.1, Response: 1.2, Color: 1,
		Strike: 0.6, StrikeDecay: 0.5, Level: 0.8740}
}

// DXChime is a glassy chime: a ratio of 5.19, the index high, almost no body
func DXChime() Config {
	return Config{ModRatio: 5.19, FM: 1, EnvToFM: 1.5, Decay: 0.5, Gate: Combo, Resonance: 0.3, Response: 1.1, Color: 1,
		KeyTrack: 1, Level: 0.6844}
}

// RingBell is a ring-modulated bell: amplitude modulation by a modulator at one and a half, the
// partials at the sum and difference
func RingBell() Config {
	return Config{ModRatio: 1.5, AM: 1, FM: 0.15, Decay: 0.4, Gate: Combo, Response: 1, Color: 0.9, Strike: 0.7,
		StrikeDecay: 0.4, Level: 1.3826}
}

// SubDrone is a held low drone: a sine with a sub modulator, a trace of fold, a slow fade in
func SubDrone() Config {
	return Config{ModRatio: 0.5, FM: 0.3, Fold: 0.1, Attack: 1.5, Decay: 2, Sustain: true, Gate: Combo, Response: 0.8, Level: 0.4284}
}

// ChoirDrone is two complex oscillators 0.12 semitones apart, which beat, under a triangle modulator
func ChoirDrone() Config {
	return Config{ModWave: ModTriangle, ModRatio: 1, FM: 0.5, Fold: 0.2, Osc2Mix: 1, Osc2Detune: 0.12, Osc2ModWave: ModTriangle,
		Attack: 1.8, Decay: 2.5, Sustain: true, Gate: Combo, Response: 0.9, Level: 0.6029}
}

// ShimmerDrone is a high shimmer over a drone: a second oscillator a fifth up, the FM breathing
func ShimmerDrone() Config {
	return Config{ModRatio: 4, FM: 0.3, Osc2Mix: 0.6, Osc2Detune: 7.02, Osc2ModRatio: 2, LFORate: 0.15, LFOFM: 0.4, Attack: 2,
		Decay: 3, Sustain: true, Gate: Combo, Response: 1, Color: 1, Level: 0.5910}
}

// FoldPad is a pad of two detuned oscillators folded over and back by a slow LFO
func FoldPad() Config {
	return Config{ModWave: ModTriangle, ModRatio: 1.003, FM: 0.2, Fold: 0.3, Osc2Mix: 0.8, Osc2Detune: 0.15, LFORate: 0.12,
		LFOFold: 0.3, Attack: 1.2, Decay: 2, Sustain: true, Gate: Combo, Response: 0.7, Level: 0.4060}
}

// GlassPad is a bright pad whose FM sways with the second LFO of the matrix, glassy at the top
func GlassPad() Config {
	cfg := Config{ModRatio: 2, FM: 0.25, Fold: 0.1, LFO2Rate: 0.2, Attack: 1.5, Decay: 2.5, Sustain: true, Gate: Combo,
		Response: 0.9, Color: 1, Level: 0.3962}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "fm", Amount: 0.12}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "modRatio", Amount: 0.03}
	return cfg
}

// BreathingPad lets the function generator cycle slowly as an LFO: every six seconds it swells,
// opening the low pass gate and the fold, and falls again
func BreathingPad() Config {
	return Config{ModWave: ModTriangle, ModRatio: 1.002, FM: 0.2, Fold: 0.15, EnvToFold: 0.4, EnvToFM: 0.8, Osc2Mix: 0.7,
		Osc2Detune: -0.1, Cycle: true, CycleRate: 0.17, Attack: 2.5, Decay: 3, Gate: LowPass, Response: 0.8, Color: 1, Level: 0.3366}
}

// WubWobble is a wobble bass: the fold and the FM swung by an LFO in sixteenths, the mod wheel
// making it faster
func WubWobble() Config {
	cfg := Config{ModRatio: 0.5, FM: 0.5, Fold: 0.3, Symmetry: 0.3, LFOSync: 0.25, LFOFold: 0.4, LFOFM: 0.6, Attack: 0.005,
		Decay: 0.2, Sustain: true, Gate: Combo, Resonance: 0.3, Response: 0.1, Color: 0.7, Level: 0.2186}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "lfoRate", Amount: 0.3}
	return cfg
}

// DubWobble is a slow, deep wobble: the second LFO, synced to the beat, moves the fold and the
// symmetry together
func DubWobble() Config {
	cfg := Config{ModWave: ModTriangle, ModRatio: 0.5, FM: 0.4, Fold: 0.25, LFO2Sync: 1, Attack: 0.01, Decay: 0.3, Sustain: true,
		Gate: Combo, Resonance: 0.15, Response: 0.15, Color: 0.6, Level: 0.2685}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "fold", Amount: 0.35}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "symmetry", Amount: 0.4}
	return cfg
}

// SubPluck is a deep plucked bass: nearly a sine, a thump of FM at the strike
func SubPluck() Config {
	return Config{ModRatio: 0.5, FM: 0.1, EnvToFM: 1.5, Decay: 0.6, Gate: Combo, Resonance: 0.1, Response: 0.4, Color: 0.5,
		Strike: 0.8, StrikeDecay: 0.25, KeyTrack: 0.3, Level: 0.9327}
}

// FoldBass is a held bass folded asymmetrically: the harder the key, the brighter
func FoldBass() Config {
	cfg := Config{Fold: 0.4, Symmetry: 0.4, EnvToFold: 0.3, Attack: 0.003, Decay: 0.3, Sustain: true, Gate: Combo, Resonance: 0.2,
		Response: 0.12, Color: 0.7, Level: 0.1750}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "fold", Amount: 0.25}
	return cfg
}

// FMBass is a growling FM bass: a one to one ratio with the index kicked up at the start of every note
func FMBass() Config {
	return Config{ModRatio: 1, FM: 1.2, EnvToFM: 2, Fold: 0.15, Attack: 0.002, Decay: 0.25, Sustain: true, Gate: Combo,
		Response: 0.1, Color: 0.65, Level: 0.7714}
}

// ReeseWest is a reese: a saw modulator on two oscillators a little apart, their beating folded
func ReeseWest() Config {
	return Config{ModWave: ModSaw, ModRatio: 0.501, FM: 0.4, Fold: 0.3, Osc2Mix: 1, Osc2Detune: 0.12, Osc2ModWave: ModSaw,
		Attack: 0.01, Decay: 0.3, Sustain: true, Gate: Combo, Response: 0.12, Color: 0.65, Level: 0.3994}
}

// SineLead is a clean lead: a sine, a trace of FM and fold, the wheel adding FM, vibrato from the
// second LFO
func SineLead() Config {
	cfg := Config{FM: 0.1, Fold: 0.12, LFO2Rate: 5.5, Attack: 0.01, Decay: 0.2, Sustain: true, Gate: Combo, Response: 0.1,
		Color: 0.9, Level: 0.3540}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "fm", Amount: 0.15}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "pitch", Amount: 0.008}
	return cfg
}

// FoldLead is a screaming lead: the wave folded hard, brighter at the start and with the velocity
func FoldLead() Config {
	cfg := Config{Fold: 0.5, Symmetry: 0.2, EnvToFold: 0.3, Attack: 0.005, Decay: 0.25, Sustain: true, Gate: Combo, Response: 0.1,
		Color: 1, Level: 0.2586}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "fold", Amount: 0.2}
	return cfg
}

// RingLead is a metallic lead: full amplitude modulation at a ratio of 1.5, the carrier sliding
// in and out of the sidebands
func RingLead() Config {
	return Config{ModRatio: 1.5, AM: 1, FM: 0.2, Fold: 0.1, Attack: 0.01, Decay: 0.2, Sustain: true, Gate: Combo,
		Response: 0.1, Color: 1, Level: 0.3854}
}

// CycleArp turns the function generator into a clock: six times a second it retriggers, so a held
// note plays itself as bright plucks
func CycleArp() Config {
	return Config{ModRatio: 2, FM: 0.2, EnvToFM: 2, Fold: 0.1, EnvToFold: 0.3, Cycle: true, CycleRate: 6, Attack: 0.003,
		Decay: 0.12, Gate: Combo, Resonance: 0.3, Response: 0.15, Color: 1, Level: 0.5970}
}

// BubbleArp is bubbles: a modulator an octave up with a high index, resonant, four to the second
func BubbleArp() Config {
	return Config{ModRatio: 2, EnvToFM: 3, Cycle: true, CycleRate: 4, Attack: 0.004, Decay: 0.08, Gate: Combo, Resonance: 0.7,
		Response: 0.1, Color: 1, KeyTrack: 1, Level: 0.8148}
}

// PulseSeq is a hard sequencer tick at eight a second: a square modulator and a deep fold
func PulseSeq() Config {
	return Config{ModWave: ModSquare, ModRatio: 1, FM: 0.3, Fold: 0.35, EnvToFold: 0.4, Cycle: true, CycleRate: 8, Attack: 0.001,
		Decay: 0.05, Gate: Combo, Resonance: 0.4, Response: 0.08, Color: 0.8, Level: 1.2210}
}

// SwellArp is a slow cycling swell, two and a half a second: each note rises and falls like a bowed note
func SwellArp() Config {
	return Config{ModWave: ModTriangle, ModRatio: 2, FM: 0.1, EnvToFM: 1, Fold: 0.1, EnvToFold: 0.3, Cycle: true, CycleRate: 2.5,
		Attack: 0.15, Decay: 0.2, Gate: Combo, Response: 0.25, Color: 1, Level: 0.6684}
}

// RandomArp is a cycling pluck whose modulator ratio is picked at random for every note, so each
// held note gets its own timbre
func RandomArp() Config {
	cfg := Config{ModRatio: 2, FM: 0.2, EnvToFM: 2, Cycle: true, CycleRate: 5, Attack: 0.003, Decay: 0.15, Gate: Combo,
		Resonance: 0.4, Response: 0.2, Color: 1, Level: 0.3777}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Random, Dest: "modRatio", Amount: 0.5}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Random, Dest: "fold", Amount: 0.3}
	return cfg
}

// AlienPing is an alien call: an odd modulator ratio with a high index, a second oscillator a tritone
// off, ringing out
func AlienPing() Config {
	return Config{ModRatio: 5.3, EnvToFM: 5, Osc2Mix: 0.5, Osc2Detune: 5, Osc2ModRatio: 1.3, Decay: 0.3, Gate: Combo, Resonance: 0.8,
		Response: 0.5, Color: 1, Strike: 0.6, StrikeDecay: 0.2, Level: 0.4474}
}

// TractorBeam is a held beam: a saw modulator, FM and fold swung by a fast LFO
func TractorBeam() Config {
	return Config{ModWave: ModSaw, ModRatio: 2, FM: 1.5, Fold: 0.3, LFORate: 7, LFOFM: 1, LFOFold: 0.3, Attack: 0.5, Decay: 1,
		Sustain: true, Gate: Combo, Response: 0.4, Color: 1, Level: 0.3218}
}

// RadioChatter is a burst of squawks: a rounded square folded hard, the FM swung by the second
// LFO at 13 Hz, the function generator cycling at eleven a second
func RadioChatter() Config {
	cfg := Config{ModWave: ModSquare, ModRatio: 3.01, FM: 0.5, Fold: 0.6, LFO2Rate: 13, Cycle: true, CycleRate: 11, Attack: 0.002,
		Decay: 0.07, Gate: Combo, Resonance: 0.5, Response: 0.08, Color: 1, Level: 0.4782}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "fm", Amount: 0.2}
	return cfg
}

// VoxFold is a vowel by folding: a square modulator at twice the note and a deep, lopsided fold put
// a broad hump in the overtones, a slow LFO and vibrato make it sing
func VoxFold() Config {
	cfg := Config{ModWave: ModSquare, ModRatio: 2, FM: 0.6, Fold: 0.55, Symmetry: 0.3, Osc2Mix: 0.4, Osc2ModRatio: 3, LFORate: 0.4,
		LFOFM: 0.5, LFO2Rate: 5.5, Attack: 0.08, Decay: 0.3, Sustain: true, Gate: Combo, Response: 0.15, Color: 0.9, Level: 0.3163}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "pitch", Amount: 0.01}
	return cfg
}

// OhFold is a rounder vowel: a soft fold of a sine with a modulator at the same pitch, little index
func OhFold() Config {
	cfg := Config{ModWave: ModTriangle, ModRatio: 1, FM: 0.25, Fold: 0.3, Symmetry: 0.15, LFORate: 0.3, LFOFold: 0.1, LFO2Rate: 5,
		Attack: 0.12, Decay: 0.3, Sustain: true, Gate: Combo, Response: 0.2, Color: 0.6, KeyTrack: 0.3, Level: 0.4462}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "pitch", Amount: 0.008}
	return cfg
}

// EeFold is a thin vowel: a bright modulator, a heavy fold with the symmetry moved by the wheel
func EeFold() Config {
	cfg := Config{ModWave: ModSaw, ModRatio: 3, FM: 0.5, Fold: 0.65, Symmetry: 0.1, Osc2Mix: 0.3, Osc2Detune: 12, LFORate: 0.5, LFOFM: 0.3,
		LFO2Rate: 6, Attack: 0.06, Decay: 0.3, Sustain: true, Gate: Combo, Response: 0.15, Color: 1, Level: 0.3201}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "symmetry", Amount: 0.5}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "pitch", Amount: 0.008}
	return cfg
}
