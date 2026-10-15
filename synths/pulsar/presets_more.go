package pulsar

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

// more is the second set of presets: pads, leads, clouds, bursts, textures, plucks, basses, drones,
// sweeps and buzzing leads.
func init() {
	for n, f := range morePresets {
		Presets[n] = f
	}
}

var morePresets = map[string]func() Config{
	"ahPad":        AhPadPreset,
	"oohPad":       OohPadPreset,
	"eerieChoir":   EerieChoirPreset,
	"shimmerPad":   ShimmerPadPreset,
	"darkPad":      DarkPadPreset,
	"aeLead":       AeLeadPreset,
	"wahLead":      WahLeadPreset,
	"talkLead":     TalkLeadPreset,
	"sawVowel":     SawVowelPreset,
	"grainCloud":   GrainCloudPreset,
	"shimmerDust":  ShimmerDustPreset,
	"lowCloud":     LowCloudPreset,
	"scatterBells": ScatterBellsPreset,
	"swarmCloud":   SwarmCloudPreset,
	"stutterTick":  StutterTickPreset,
	"gallop":       GallopPreset,
	"clavePulse":   ClavePulsePreset,
	"machineGun":   MachineGunPreset,
	"chopLead":     ChopLeadPreset,
	"fireCrackle":  FireCracklePreset,
	"crickets":     CricketsPreset,
	"cicadas":      CicadasPreset,
	"drizzle":      DrizzlePreset,
	"geigerTicks":  GeigerTicksPreset,
	"pluckPulsar":  PluckPulsarPreset,
	"harpPulsar":   HarpPulsarPreset,
	"kalimba":      KalimbaPreset,
	"squareBass":   SquareBassPreset,
	"reeseBass":    ReeseBassPreset,
	"acidPulsar":   AcidPulsarPreset,
	"deepDrone":    DeepDronePreset,
	"beatingDrone": BeatingDronePreset,
	"ghostDrone":   GhostDronePreset,
	"noiseSweep":   NoiseSweepPreset,
	"windGust":     WindGustPreset,
	"hornet":       HornetPreset,
	"sawLead":      SawLeadPreset,
	"zapLead":      ZapLeadPreset,
}

// env is an envelope of attack, decay, sustain level and release (milliseconds)
func env(attack, decay, sustain, release float64) adsr.Setting {
	return *adsr.NewSetting(1.0, attack, sustain, decay, 1.0, release)
}

// AhPadPreset is a broad "ah" pad: two open formants, a slow drift of the vowel and a wide spread
func AhPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 700
	cfg.Formant2 = 1150
	cfg.Formant2Level = 0.7
	cfg.FormantTracking = 0.25
	cfg.Duty = 0.85
	cfg.FormantLFORate = 0.11
	cfg.FormantLFODepth = 0.25
	cfg.NoteSpread = 0.8
	cfg.VibratoRate = 0.4
	cfg.VibratoDepth = 0.1
	cfg.AmpEnv = env(1100, 600, 0.9, 1600)
	cfg.Level = 0.4374 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// OohPadPreset is a closed, dark "ooh": a low formant with a faint second one, hardly any movement
func OohPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 330
	cfg.Formant2 = 760
	cfg.Formant2Level = 0.25
	cfg.FormantTracking = 0.1
	cfg.Duty = 1
	cfg.Jitter = 0.003
	cfg.FormantLFORate = 0.07
	cfg.FormantLFODepth = 0.08
	cfg.NoteSpread = 0.5
	cfg.VibratoRate = 4.8
	cfg.VibratoDepth = 0.07
	cfg.AmpEnv = env(700, 400, 0.9, 1400)
	cfg.Level = 0.4921 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// EerieChoirPreset is a thin "ee" choir with a little breath: a very low first formant under a high
// second one, and a slow wandering of both
func EerieChoirPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 290
	cfg.Formant2 = 2300
	cfg.Formant2Level = 0.8
	cfg.FormantTracking = 0.2
	cfg.Duty = 0.55
	cfg.Masking = 0.06
	cfg.Jitter = 0.01
	cfg.FormantLFORate = 0.23
	cfg.FormantLFODepth = 0.35
	cfg.NoteSpread = 0.9
	cfg.VibratoRate = 5.8
	cfg.VibratoDepth = 0.18
	cfg.AmpEnv = env(1300, 500, 0.85, 2200)
	cfg.Level = 0.3850 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ShimmerPadPreset is a glassy pad: narrow Gaussian pulses in the highs, the second formant swayed
// by the second LFO of the matrix so it glitters
func ShimmerPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 2600
	cfg.Formant2 = 3900
	cfg.Formant2Level = 0.6
	cfg.FormantTracking = 0.4
	cfg.Duty = 0.7
	cfg.Window = Gaussian
	cfg.Jitter = 0.01
	cfg.NoteSpread = 1
	cfg.LFO2Rate = 0.37
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "formant2", Amount: 0.2}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "formant", Amount: -0.1}
	cfg.AmpEnv = env(1500, 800, 0.9, 2500)
	cfg.Level = 0.3118 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DarkPadPreset is a brooding pad: low formants through a ladder filter that opens slowly at every note
func DarkPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 420
	cfg.Formant2 = 1000
	cfg.Formant2Level = 0.6
	cfg.FormantTracking = 0.6
	cfg.Duty = 0.6
	cfg.Window = Tapered
	cfg.NoteSpread = 0.7
	cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 380, Resonance: 0.3, EnvAmount: 2.2, KeyTrack: 0.6,
		Env: env(1800, 1500, 0.4, 1500)}
	cfg.AmpEnv = env(900, 500, 0.9, 1800)
	cfg.Level = 0.9010 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AeLeadPreset is a lead that sings "a-e-a": the formants swing slowly between the vowels
func AeLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 650
	cfg.Formant2 = 1700
	cfg.Formant2Level = 0.6
	cfg.FormantTracking = 0.3
	cfg.Duty = 0.5
	cfg.FormantLFORate = 1.3
	cfg.FormantLFODepth = 0.55
	cfg.VibratoRate = 5.6
	cfg.VibratoDepth = 0.2
	cfg.Glide = 60
	cfg.AmpEnv = env(25, 100, 0.85, 220)
	cfg.Level = 0.4239 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WahLeadPreset is a lead played with the mod wheel and the touch: the wheel opens the formant, a
// hard note opens it more, the saw pulses give the wah something to bite into
func WahLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 450
	cfg.Formant2 = 1400
	cfg.Formant2Level = 0.4
	cfg.FormantTracking = 0.5
	cfg.Duty = 0.45
	cfg.Pulsaret2 = Saw
	cfg.ShapeMix = 0.4
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "formant", Amount: 0.35}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "formant", Amount: 0.25}
	cfg.Mods[2] = modmatrix.Slot{Src: modmatrix.FilterEnv, Dest: "formant", Amount: 0.2}
	cfg.FormantEnvAmount = 0
	cfg.FormantEnv = env(1, 220, 0.3, 200)
	cfg.VibratoRate = 5.2
	cfg.VibratoDepth = 0.12
	cfg.Glide = 30
	cfg.AmpEnv = env(8, 100, 0.85, 160)
	cfg.Level = 0.4191 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// TalkLeadPreset is a talking synth: a fast swing of the formants, a vowel every few hundred
// milliseconds, the pitch gliding between the notes
func TalkLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 500
	cfg.Formant2 = 1900
	cfg.Formant2Level = 0.7
	cfg.FormantTracking = 0.15
	cfg.Duty = 0.4
	cfg.Window = Tapered
	cfg.FormantLFORate = 3.2
	cfg.FormantLFODepth = 0.9
	cfg.Glide = 90
	cfg.AmpEnv = env(5, 100, 0.9, 120)
	cfg.Level = 0.4052 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SawVowelPreset is a vowel lead from saw pulsarets: all the harmonics inside the formant, a low
// pass keeping the top down, a random touch of formant at every note
func SawVowelPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 800
	cfg.Formant2 = 1200
	cfg.Formant2Level = 0.5
	cfg.FormantTracking = 0.4
	cfg.Duty = 0.75
	cfg.Pulsaret2 = Saw
	cfg.ShapeMix = 0.75
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Random, Dest: "formant", Amount: 0.1}
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.12
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 3500, Resonance: 0.15, KeyTrack: 0.5, Env: env(1, 100, 1, 100)}
	cfg.AmpEnv = env(20, 150, 0.85, 200)
	cfg.Level = 0.4484 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GrainCloudPreset is a cloud of grains for notes around 40 Hz: soft Gaussian pulses, half of them
// left out, the period jittering, every grain in its own place; the formant drifts
func GrainCloudPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1800
	cfg.Duty = 0.2
	cfg.Window = Gaussian
	cfg.Masking = 0.5
	cfg.Jitter = 0.8
	cfg.StereoSpread = 1
	cfg.FormantLFORate = 0.17
	cfg.FormantLFODepth = 0.9
	cfg.AmpEnv = env(900, 300, 1, 1500)
	cfg.Level = 0.2922 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ShimmerDustPreset is fine dust for notes around 60 Hz: tiny high grains in two bands, sparse
// and wide
func ShimmerDustPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 5200
	cfg.Formant2 = 7600
	cfg.Formant2Level = 0.6
	cfg.Duty = 0.12
	cfg.Window = Gaussian
	cfg.Masking = 0.65
	cfg.Jitter = 0.5
	cfg.StereoSpread = 1
	cfg.LFO2Rate = 0.21
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "masking", Amount: 0.2}
	cfg.AmpEnv = env(1200, 300, 1, 1800)
	cfg.Level = 0.2550 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// LowCloudPreset is a rumbling cloud for notes around 25 Hz: long soft grains in the lows that
// overlap and drift in and out
func LowCloudPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 300
	cfg.Formant2 = 520
	cfg.Formant2Level = 0.5
	cfg.Duty = 0.6
	cfg.Masking = 0.3
	cfg.Jitter = 0.9
	cfg.StereoSpread = 0.9
	cfg.FormantLFORate = 0.09
	cfg.FormantLFODepth = 0.6
	cfg.AmpEnv = env(1500, 300, 1, 2000)
	cfg.Level = 0.3321 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ScatterBellsPreset is bells thrown about for notes around 20 Hz: sparse struck pulses in the
// highs, each at a place and a pitch of its own, the formant wandering
func ScatterBellsPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 3200
	cfg.Formant2 = 4800
	cfg.Formant2Level = 0.5
	cfg.Duty = 0.45
	cfg.Window = Percussive
	cfg.Masking = 0.75
	cfg.Jitter = 1
	cfg.StereoSpread = 1
	cfg.FormantLFORate = 0.6
	cfg.FormantLFODepth = 1.2
	cfg.AmpEnv = env(400, 300, 1, 1500)
	cfg.Level = 0.2295 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SwarmCloudPreset is a swarm for notes around 80 Hz: buzzing mid grains, the pulse rate swaying
// by whole semitones so the swarm swells and thins
func SwarmCloudPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1200
	cfg.Formant2 = 2100
	cfg.Formant2Level = 0.5
	cfg.Duty = 0.3
	cfg.Window = Tapered
	cfg.Masking = 0.4
	cfg.Jitter = 0.6
	cfg.StereoSpread = 0.8
	cfg.VibratoRate = 0.45
	cfg.VibratoDepth = 4
	cfg.AmpEnv = env(1000, 300, 1, 1500)
	cfg.Level = 0.3283 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// StutterTickPreset is a stuttering hi-hat for notes around 12 Hz: three ticks and two rests, over and over
func StutterTickPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 5200
	cfg.Formant2 = 7400
	cfg.Formant2Level = 0.6
	cfg.Duty = 0.1
	cfg.Window = Percussive
	cfg.Burst = 3
	cfg.BurstRest = 2
	cfg.Jitter = 0.02
	cfg.StereoSpread = 0.4
	cfg.AmpEnv = env(1, 10, 1, 150)
	cfg.Level = 0.3085 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GallopPreset is a low gallop for notes around 10 Hz: three dull thuds and a rest
func GallopPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 110
	cfg.Formant2 = 260
	cfg.Formant2Level = 0.4
	cfg.Duty = 0.25
	cfg.Window = Percussive
	cfg.Burst = 3
	cfg.BurstRest = 1
	cfg.AmpEnv = env(1, 10, 1, 200)
	cfg.Level = 0.3705 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ClavePulsePreset is a clave pattern for notes around 8 Hz: two woody knocks and three rests,
// a five-step feel
func ClavePulsePreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 2400
	cfg.Formant2 = 3700
	cfg.Formant2Level = 0.5
	cfg.Duty = 0.2
	cfg.Window = Percussive
	cfg.Burst = 2
	cfg.BurstRest = 3
	cfg.StereoSpread = 0.3
	cfg.AmpEnv = env(1, 10, 1, 200)
	cfg.Level = 0.2858 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MachineGunPreset is a burst of fire for notes around 18 Hz: four hits with a falling, snapping
// formant, four rests
func MachineGunPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 900
	cfg.Duty = 0.35
	cfg.Window = Percussive
	cfg.Burst = 4
	cfg.BurstRest = 4
	cfg.Masking = 0.1
	cfg.Jitter = 0.08
	cfg.FormantEnvAmount = 1.2
	cfg.FormantEnv = env(1, 150, 0.3, 200)
	cfg.StereoSpread = 0.5
	cfg.AmpEnv = env(1, 10, 1, 300)
	cfg.Level = 0.3226 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ChopLeadPreset is a pitched lead gated by its own bursts: six pulses on, two off, so the tone
// flutters at a quarter of the pitch
func ChopLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1100
	cfg.Formant2 = 2400
	cfg.Formant2Level = 0.4
	cfg.FormantTracking = 0.6
	cfg.Duty = 0.5
	cfg.Window = Tapered
	cfg.Burst = 6
	cfg.BurstRest = 2
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.1
	cfg.AmpEnv = env(10, 100, 0.85, 150)
	cfg.Level = 0.4760 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FireCracklePreset is a fire for notes around 45 Hz: most pulses gone, the rest in bright pops
func FireCracklePreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1600
	cfg.Formant2 = 280
	cfg.Formant2Level = 0.8
	cfg.Duty = 0.2
	cfg.Window = Percussive
	cfg.Masking = 0.88
	cfg.Jitter = 1
	cfg.StereoSpread = 1
	cfg.FormantLFORate = 0.9
	cfg.FormantLFODepth = 1
	cfg.AmpEnv = env(300, 300, 1, 1000)
	cfg.Level = 0.2260 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// CricketsPreset is crickets for notes around 40 Hz: short chirps of three pulses in a high
// band with a rest between, the chirp rate wavering
func CricketsPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 4300
	cfg.Duty = 0.25
	cfg.Window = Gaussian
	cfg.Burst = 3
	cfg.BurstRest = 5
	cfg.Jitter = 0.05
	cfg.StereoSpread = 0.7
	cfg.VibratoRate = 0.8
	cfg.VibratoDepth = 1.5
	cfg.AmpEnv = env(600, 300, 1, 1200)
	cfg.Level = 0.3033 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// CicadasPreset is a tree full of cicadas for a note: a bright buzz pulsing in and out, the
// level swaying with the second LFO of the matrix
func CicadasPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 6400
	cfg.Formant2 = 5300
	cfg.Formant2Level = 0.7
	cfg.FormantTracking = 0.2
	cfg.Duty = 0.45
	cfg.Window = Tapered
	cfg.Masking = 0.08
	cfg.Jitter = 0.3
	cfg.NoteSpread = 1
	cfg.LFO2Rate = 0.31
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "level", Amount: -0.2}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "formant", Amount: 0.05}
	cfg.AmpEnv = env(800, 300, 1, 1500)
	cfg.Level = 0.2116 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DrizzlePreset is light rain for notes around 15 Hz: soft drops, half missing, the pitch of
// every drop a little different
func DrizzlePreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 3800
	cfg.Formant2 = 1500
	cfg.Formant2Level = 0.3
	cfg.Duty = 0.25
	cfg.Window = Percussive
	cfg.Masking = 0.5
	cfg.Jitter = 1
	cfg.StereoSpread = 1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "formant", Amount: 0.3}
	cfg.FormantLFORate = 7.3
	cfg.AmpEnv = env(600, 300, 1, 1500)
	cfg.Level = 0.2667 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GeigerTicksPreset is a counter for notes around 30 Hz: very short clicks, nine in ten gone
func GeigerTicksPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 6800
	cfg.Duty = 0.04
	cfg.Window = Rectangle
	cfg.Masking = 0.92
	cfg.Jitter = 1
	cfg.StereoSpread = 0.6
	cfg.AmpEnv = env(1, 10, 1, 200)
	cfg.Level = 0.3118 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PluckPulsarPreset is a tonal pluck: a struck pulse whose formant falls fast from high, like a
// string losing its brightness
func PluckPulsarPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1200
	cfg.FormantTracking = 1
	cfg.Duty = 0.7
	cfg.Window = Percussive
	cfg.FormantEnvAmount = 2.5
	cfg.FormantEnv = env(1, 1, 0, 120)
	cfg.AmpEnv = env(1, 700, 0, 200)
	cfg.Level = 0.4672 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HarpPulsarPreset is a harp: narrow Gaussian pulses with a second formant an octave and a half up,
// ringing long and the notes spread over the stereo field
func HarpPulsarPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1500
	cfg.Formant2 = 2250
	cfg.Formant2Level = 0.35
	cfg.FormantTracking = 1
	cfg.Duty = 0.9
	cfg.Window = Gaussian
	cfg.FormantEnvAmount = 1.2
	cfg.FormantEnv = env(1, 1, 0, 350)
	cfg.NoteSpread = 0.6
	cfg.AmpEnv = env(1, 2200, 0, 400)
	cfg.Level = 0.4074 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// KalimbaPreset is a thumb piano: soft triangle pulsarets, a woody tine with a short ring
func KalimbaPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 2300
	cfg.FormantTracking = 0.5
	cfg.Duty = 0.55
	cfg.Window = Gaussian
	cfg.Pulsaret2 = Triangle
	cfg.ShapeMix = 0.9
	cfg.FormantEnvAmount = 0.6
	cfg.FormantEnv = env(1, 1, 0, 80)
	cfg.NoteSpread = 0.3
	cfg.AmpEnv = env(1, 1100, 0, 250)
	cfg.Level = 0.3654 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SquareBassPreset is a hollow square bass: odd harmonics inside a formant that follows the pitch,
// a low pass that snaps
func SquareBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 300
	cfg.FormantTracking = 1
	cfg.Duty = 1
	cfg.Window = Rectangle
	cfg.Pulsaret2 = Square
	cfg.ShapeMix = 1
	cfg.Glide = 20
	cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 350, Resonance: 0.25, EnvAmount: 2.5, KeyTrack: 0.8,
		Env: env(1, 220, 0.2, 150)}
	cfg.AmpEnv = env(3, 200, 0.9, 120)
	cfg.Level = 0.6307 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ReeseBassPreset is a thick, moving bass: a saw in a formant a little slow of the pitch,
// the vibrato wide and slow so the harmonics beat
func ReeseBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 220
	cfg.FormantTracking = 1
	cfg.Duty = 0.9
	cfg.Pulsaret2 = Saw
	cfg.ShapeMix = 1
	cfg.NoteSpread = 0.2
	cfg.VibratoRate = 0.35
	cfg.VibratoDepth = 0.35
	cfg.Jitter = 0.01
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 900, Resonance: 0.3, KeyTrack: 0.7, Env: env(1, 100, 1, 100)}
	cfg.AmpEnv = env(5, 200, 0.9, 200)
	cfg.Level = 0.7042 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AcidPulsarPreset is an acid line: saw pulses through a resonant ladder snapping shut at every
// note, the notes sliding into each other
func AcidPulsarPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 250
	cfg.FormantTracking = 1
	cfg.Duty = 1
	cfg.Pulsaret2 = Saw
	cfg.ShapeMix = 1
	cfg.Glide = 70
	cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 280, Resonance: 0.72, EnvAmount: 3.2, KeyTrack: 0.8,
		Env: env(1, 180, 0.05, 100)}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.3}
	cfg.AmpEnv = env(2, 150, 0.9, 80)
	cfg.Level = 0.9558 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DeepDronePreset is a drone in the cellar: a formant far under the pitch, the notes spread, a
// very slow swell of the formant
func DeepDronePreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 130
	cfg.Formant2 = 195
	cfg.Formant2Level = 0.6
	cfg.FormantTracking = 0.7
	cfg.Duty = 1
	cfg.Jitter = 0.005
	cfg.FormantLFORate = 0.05
	cfg.FormantLFODepth = 0.5
	cfg.NoteSpread = 0.5
	cfg.AmpEnv = env(3000, 500, 1, 3000)
	cfg.Level = 0.9937 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BeatingDronePreset is a drone of two formants a hair apart, which beat against each other
// slowly; the vibrato is slow and shallow
func BeatingDronePreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 500
	cfg.Formant2 = 506
	cfg.Formant2Level = 1
	cfg.FormantTracking = 1
	cfg.Duty = 1
	cfg.NoteSpread = 0.6
	cfg.VibratoRate = 0.2
	cfg.VibratoDepth = 0.08
	cfg.AmpEnv = env(2000, 500, 1, 3000)
	cfg.Level = 0.5260 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GhostDronePreset is a hollow drone with the breath of a ghost: some pulses gone, a band pass
// that sweeps slowly by itself
func GhostDronePreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 900
	cfg.Formant2 = 1370
	cfg.Formant2Level = 0.5
	cfg.FormantTracking = 0.3
	cfg.Duty = 0.65
	cfg.Masking = 0.2
	cfg.Jitter = 0.06
	cfg.NoteSpread = 1
	cfg.FormantLFORate = 0.13
	cfg.FormantLFODepth = 0.6
	cfg.LFO2Rate = 0.07
	cfg.Filter = voicefilter.Config{Type: voicefilter.BandPass, Cutoff: 1200, Resonance: 0.55, KeyTrack: 0.4, Env: env(1, 100, 1, 100)}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "cutoff", Amount: 0.2}
	cfg.AmpEnv = env(2500, 500, 1, 3500)
	cfg.Level = 0.5144 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// NoiseSweepPreset is a riser: a pulse train torn to noise by a full jitter, the formant climbing
// four octaves over three seconds
func NoiseSweepPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 200
	cfg.FormantTracking = 0
	cfg.Duty = 0.9
	cfg.Window = Tapered
	cfg.Jitter = 1
	cfg.Masking = 0.25
	cfg.StereoSpread = 1
	cfg.FormantEnvAmount = 4.3
	cfg.FormantEnv = *adsr.NewSetting(1.0, 3500.0, 1.0, 500.0, 1.0, 300.0)
	cfg.AmpEnv = env(2500, 500, 1, 500)
	cfg.Level = 0.9164 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WindGustPreset is wind: noisy pulses through a resonant band pass, the formant and the pass
// swaying about in slow, unrelated cycles
func WindGustPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 700
	cfg.FormantTracking = 0
	cfg.Duty = 0.8
	cfg.Jitter = 1
	cfg.Masking = 0.2
	cfg.StereoSpread = 1
	cfg.FormantLFORate = 0.19
	cfg.FormantLFODepth = 1
	cfg.LFO2Rate = 0.11
	cfg.Filter = voicefilter.Config{Type: voicefilter.BandPass, Cutoff: 900, Resonance: 0.6, Env: env(1, 100, 1, 100)}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "cutoff", Amount: 0.45}
	cfg.AmpEnv = env(1500, 500, 1, 2500)
	cfg.Level = 1.0643 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HornetPreset is a buzzing lead: rectangular pulses in a bright formant that follows the notes,
// a quick vibrato like wings
func HornetPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1500
	cfg.Formant2 = 2900
	cfg.Formant2Level = 0.4
	cfg.FormantTracking = 0.6
	cfg.Duty = 0.5
	cfg.Window = Rectangle
	cfg.Jitter = 0.02
	cfg.VibratoRate = 11
	cfg.VibratoDepth = 0.2
	cfg.Glide = 25
	cfg.AmpEnv = env(8, 100, 0.9, 120)
	cfg.Level = 0.3641 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SawLeadPreset is a classic saw lead out of pulses: all the harmonics, a formant opening at the
// start, a touch of slide
func SawLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 500
	cfg.FormantTracking = 1
	cfg.Duty = 1
	cfg.Pulsaret2 = Saw
	cfg.ShapeMix = 1
	cfg.FormantEnvAmount = 0.8
	cfg.FormantEnv = env(1, 250, 0.3, 200)
	cfg.VibratoRate = 5.3
	cfg.VibratoDepth = 0.12
	cfg.Glide = 35
	cfg.AmpEnv = env(5, 100, 0.9, 150)
	cfg.Level = 0.4074 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ZapLeadPreset is a lead with a zap in it: tapered pulses and a formant swung fast and wide by
// its LFO, pitch and wheel sending it further
func ZapLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 900
	cfg.Formant2 = 2200
	cfg.Formant2Level = 0.5
	cfg.FormantTracking = 0.5
	cfg.Duty = 0.4
	cfg.Window = Tapered
	cfg.FormantLFORate = 9
	cfg.FormantLFODepth = 1.1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "lfoRate", Amount: 0.3}
	cfg.AmpEnv = env(4, 100, 0.85, 120)
	cfg.Level = 0.4037 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
