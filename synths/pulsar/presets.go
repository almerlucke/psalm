package pulsar

import "github.com/almerlucke/genny/float/envelopes/adsr"

// Presets maps preset names to their constructors
var Presets = map[string]func() Config{
	"vowelLead":  VowelLeadPreset,
	"buzzBass":   BuzzBassPreset,
	"glass":      GlassPreset,
	"formantPad": FormantPadPreset,
	"laser":      LaserPreset,
	"rhythm":     RhythmPreset,
	"crackle":    CracklePreset,
	"choir":      ChoirPreset,
	"strings":    StringsPreset,
	"brass":      BrassPreset,
	"mallet":     MalletPreset,
	"subBass":    SubBassPreset,
	"wobbleBass": WobbleBassPreset,
	"insects":    InsectsPreset,
	"raindrops":  RaindropsPreset,
	"motor":      MotorPreset,
}

// VowelLeadPreset is a singing lead: two formants like an "ah", a slow drift of the formants,
// vibrato and glide
func VowelLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 700
	cfg.Formant2 = 1150
	cfg.Formant2Level = 0.6
	cfg.FormantTracking = 0.2
	cfg.Duty = 0.6
	cfg.FormantLFORate = 0.3
	cfg.FormantLFODepth = 0.3
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.15
	cfg.Glide = 40
	cfg.AmpEnv = *adsr.NewSetting(1.0, 15.0, 0.85, 200.0, 1.0, 250.0)
	cfg.Level = 0.4413 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BuzzBassPreset is a buzzy bass: tapered pulses (rectangular with smooth edges, so no aliasing)
// with a formant that follows the pitch and snaps down from two octaves up at every note, like a
// plucked filter
func BuzzBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 400
	cfg.FormantTracking = 1
	cfg.Duty = 0.35
	cfg.Window = Tapered
	cfg.FormantEnvAmount = 2
	cfg.FormantEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 180.0, 1.0, 100.0)
	cfg.Glide = 15
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.7, 250.0, 1.0, 80.0)
	cfg.Level = 0.6505 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GlassPreset is a bright, glassy key: narrow Gaussian pulses with two high formants that follow
// the pitch, a struck attack and a long decay
func GlassPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 2400
	cfg.Formant2 = 3900
	cfg.Formant2Level = 0.5
	cfg.FormantTracking = 1
	cfg.Duty = 0.8
	cfg.Window = Gaussian
	cfg.FormantEnvAmount = 0.5
	cfg.FormantEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 400.0, 1.0, 300.0)
	cfg.NoteSpread = 0.3
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.0, 1800.0, 1.0, 600.0)
	cfg.Level = 0.3449 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FormantPadPreset is a slow, wide pad whose formants sweep slowly through vowel colours
func FormantPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 600
	cfg.Formant2 = 1500
	cfg.Formant2Level = 0.5
	cfg.FormantTracking = 0.3
	cfg.Duty = 0.9
	cfg.FormantLFORate = 0.08
	cfg.FormantLFODepth = 0.8
	cfg.NoteSpread = 0.6
	cfg.VibratoRate = 0.3
	cfg.VibratoDepth = 0.08
	cfg.AmpEnv = *adsr.NewSetting(1.0, 900.0, 0.9, 500.0, 1.0, 1800.0)
	cfg.Level = 0.4209 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// LaserPreset is a falling chirp: the formant sweeps down four octaves at every note
func LaserPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 350
	cfg.Duty = 0.4
	cfg.FormantEnvAmount = 4
	cfg.FormantEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 220.0, 1.0, 100.0)
	cfg.NoteSpread = 0.4
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 400.0, 1.0, 100.0)
	cfg.Level = 0.4045 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RhythmPreset is meant for very low notes (a few Hz, send a frequency): every pulse is a short
// percussive tick, so the pitch becomes a tempo and the formant the colour of the tick
func RhythmPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1200
	cfg.Formant2 = 2900
	cfg.Formant2Level = 0.4
	cfg.Duty = 0.12
	cfg.Window = Percussive
	cfg.Jitter = 0.03
	cfg.StereoSpread = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 1.0, 10.0, 1.0, 200.0)
	cfg.Level = 0.3101 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// CracklePreset is a stochastic texture for low notes (30-100 Hz): most pulses are masked and the
// period jitters, so the regular buzz breaks up into crackles scattered over the stereo field
func CracklePreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 3000
	cfg.Duty = 0.15
	cfg.Window = Percussive
	cfg.Masking = 0.7
	cfg.Jitter = 0.7
	cfg.StereoSpread = 1
	cfg.FormantLFORate = 0.2
	cfg.FormantLFODepth = 0.7
	cfg.AmpEnv = *adsr.NewSetting(1.0, 300.0, 1.0, 10.0, 1.0, 1000.0)
	cfg.Level = 0.3281 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ChoirPreset is a soft "ooh" choir pad: low formants, a little jitter for breath, slow vibrato and
// the notes spread over the stereo field
func ChoirPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 450
	cfg.Formant2 = 900
	cfg.Formant2Level = 0.5
	cfg.FormantTracking = 0.15
	cfg.Duty = 0.7
	cfg.Jitter = 0.004
	cfg.NoteSpread = 0.7
	cfg.FormantLFORate = 0.15
	cfg.FormantLFODepth = 0.15
	cfg.VibratoRate = 4.5
	cfg.VibratoDepth = 0.12
	cfg.AmpEnv = *adsr.NewSetting(1.0, 600.0, 0.9, 400.0, 1.0, 1200.0)
	cfg.Level = 0.4429 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// StringsPreset is a bowed string section: short smooth pulses with a bright formant that partly
// follows the pitch, a slow bow and vibrato
func StringsPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1300
	cfg.Formant2 = 2600
	cfg.Formant2Level = 0.3
	cfg.FormantTracking = 0.7
	cfg.Duty = 0.3
	cfg.Jitter = 0.002
	cfg.NoteSpread = 0.4
	cfg.VibratoRate = 5.2
	cfg.VibratoDepth = 0.1
	cfg.AmpEnv = *adsr.NewSetting(1.0, 250.0, 0.85, 300.0, 1.0, 600.0)
	cfg.Level = 0.3257 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BrassPreset is a brassy lead: the formant opens up during the attack and settles, like a horn
// that gets brighter the harder it is blown
func BrassPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 500
	cfg.Formant2 = 1300
	cfg.Formant2Level = 0.4
	cfg.FormantTracking = 0.5
	cfg.Duty = 0.45
	cfg.FormantEnvAmount = 1.5
	cfg.FormantEnv = *adsr.NewSetting(1.0, 80.0, 0.4, 300.0, 1.0, 200.0)
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.06
	cfg.Glide = 20
	cfg.AmpEnv = *adsr.NewSetting(1.0, 40.0, 0.85, 200.0, 1.0, 150.0)
	cfg.Level = 0.4071 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MalletPreset is a struck wooden mallet: percussive pulses with a formant at about the fourth
// harmonic, a bright strike and a quick decay
func MalletPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 1050
	cfg.FormantTracking = 1
	cfg.Duty = 0.6
	cfg.Window = Percussive
	cfg.FormantEnvAmount = 1
	cfg.FormantEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 60.0, 1.0, 50.0)
	cfg.NoteSpread = 0.2
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 600.0, 1.0, 200.0)
	cfg.Level = 0.4307 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SubBassPreset is a deep, round bass: full pulses holding one cycle of a formant at the pitch
// itself (almost a sine), with a touch of the octave
func SubBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = middleC
	cfg.Formant2 = 2 * middleC
	cfg.Formant2Level = 0.3
	cfg.FormantTracking = 1
	cfg.Duty = 1
	cfg.Glide = 30
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.9, 200.0, 1.0, 120.0)
	cfg.Level = 0.4594 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WobbleBassPreset is a talking bass: a fast LFO swings the formant over one and a half octaves,
// like a wah opening and closing. Tapered pulses keep it buzzy without the aliasing of hard edges
func WobbleBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 500
	cfg.FormantTracking = 1
	cfg.Duty = 0.4
	cfg.Window = Tapered
	cfg.FormantLFORate = 3
	cfg.FormantLFODepth = 1.5
	cfg.Glide = 25
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3.0, 0.9, 200.0, 1.0, 100.0)
	cfg.Level = 0.4668 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// InsectsPreset is a summer night of insects for low notes (30-60 Hz): tiny high pulses, partly
// masked, with a fast wide vibrato that makes the pulse rate chirp
func InsectsPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 6000
	cfg.Formant2 = 4500
	cfg.Formant2Level = 0.5
	cfg.Duty = 0.08
	cfg.Window = Gaussian
	cfg.Masking = 0.3
	cfg.Jitter = 0.2
	cfg.StereoSpread = 0.8
	cfg.VibratoRate = 7
	cfg.VibratoDepth = 4
	cfg.AmpEnv = *adsr.NewSetting(1.0, 500.0, 1.0, 10.0, 1.0, 1000.0)
	cfg.Level = 0.2324 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RaindropsPreset is rain on a window for very low notes (8-20 Hz): irregular percussive drops,
// each one at a different pitch and place
func RaindropsPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 2500
	cfg.Duty = 0.3
	cfg.Window = Percussive
	cfg.Masking = 0.4
	cfg.Jitter = 1
	cfg.StereoSpread = 1
	cfg.FormantLFORate = 3.7
	cfg.FormantLFODepth = 1
	cfg.AmpEnv = *adsr.NewSetting(1.0, 800.0, 1.0, 10.0, 1.0, 1500.0)
	cfg.Level = 0.3 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MotorPreset is a running engine for low notes (15-40 Hz): rough rectangular pulses with an
// uneven period, raise the frequency to rev it up
func MotorPreset() Config {
	cfg := DefaultConfig()
	cfg.Formant = 180
	cfg.Formant2 = 700
	cfg.Formant2Level = 0.4
	cfg.Duty = 0.6
	cfg.Window = Rectangle
	cfg.Jitter = 0.15
	cfg.Glide = 400
	cfg.AmpEnv = *adsr.NewSetting(1.0, 200.0, 1.0, 10.0, 1.0, 500.0)
	cfg.Level = 0.4357 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
