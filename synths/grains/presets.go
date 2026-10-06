package grains

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/sndfile"
)

// Presets are starting points for different granular instruments. They take the sample to play
// and its root note (the MIDI pitch of the recording), the character comes from the settings, so
// every preset works with any sample

// Presets maps preset names to their constructors
var Presets = map[string]func(sample sndfile.SoundFiler[float32], rootNote float64) Config{
	"choirPad":    ChoirPad,
	"frozenDrone": FrozenDrone,
	"glassKeys":   GlassKeys,
	"texture":     Texture,
	"stutter":     Stutter,
	"shimmer":     Shimmer,
	"pluck":       Pluck,
}

func preset(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := DefaultConfig()
	cfg.Sample = sample
	cfg.RootNote = rootNote
	return cfg
}

// ChoirPad is a slow, wide pad: long overlapping grains drifting slowly through the sample
func ChoirPad(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := preset(sample, rootNote)
	cfg.Position = 0.4
	cfg.Scan = 0.05
	cfg.Spray = 0.15
	cfg.Density = 40
	cfg.GrainSize = 250
	cfg.PitchSpread = 0.08
	cfg.StereoSpread = 0.9
	cfg.Jitter = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 800.0, 0.9, 500.0, 1.0, 2000.0)
	cfg.Level = 3.151 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FrozenDrone holds one moment of the sample forever (no scan), a living, breathing drone
func FrozenDrone(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := preset(sample, rootNote)
	cfg.Position = 0.5
	cfg.Scan = 0
	cfg.Spray = 0.02
	cfg.Density = 50
	cfg.GrainSize = 300
	cfg.PitchSpread = 0.05
	cfg.StereoSpread = 0.8
	cfg.Jitter = 0.6
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1500.0, 0.9, 500.0, 1.0, 2500.0)
	cfg.Level = 3.176 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GlassKeys is a playable keyboard: dense short grains and a plucked envelope
func GlassKeys(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := preset(sample, rootNote)
	cfg.Position = 0.2
	cfg.Scan = 0.5
	cfg.Spray = 0.01
	cfg.Density = 60
	cfg.GrainSize = 60
	cfg.Shape = 0.3
	cfg.StereoSpread = 0.4
	cfg.Jitter = 0.2
	cfg.VelocityDensity = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.2, 600.0, 1.0, 400.0)
	cfg.Level = 5.283 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Texture is a restless cloud: wide spray, pitch smear, reversed grains and irregular timing
func Texture(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := preset(sample, rootNote)
	cfg.Position = 0.3
	cfg.Scan = 0.3
	cfg.Spray = 0.5
	cfg.Density = 25
	cfg.GrainSize = 150
	cfg.PitchSpread = 0.3
	cfg.StereoSpread = 1.0
	cfg.Reverse = 0.3
	cfg.Jitter = 0.8
	cfg.AmpEnv = *adsr.NewSetting(1.0, 600.0, 0.9, 500.0, 1.0, 1500.0)
	cfg.Level = 2.295 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Stutter is rhythmic: regular, percussive grains repeating small pieces of the sample
func Stutter(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := preset(sample, rootNote)
	cfg.Position = 0.3
	cfg.Scan = 1
	cfg.Spray = 0
	cfg.Density = 12
	cfg.GrainSize = 70
	cfg.Shape = 1
	cfg.StereoSpread = 0.6
	cfg.Jitter = 0
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.9, 100.0, 1.0, 200.0)
	cfg.Level = 1.167 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Shimmer is a bright, sparkling pad: part of the grains play an octave up
func Shimmer(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := ChoirPad(sample, rootNote)
	cfg.Octave = 0.35
	cfg.GrainSize = 200
	cfg.Density = 50
	cfg.Level = 2.883 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Pluck is a short, plucked tone from a single burst of grains
func Pluck(sample sndfile.SoundFiler[float32], rootNote float64) Config {
	cfg := preset(sample, rootNote)
	cfg.Position = 0.1
	cfg.Scan = 1
	cfg.Spray = 0.005
	cfg.Density = 80
	cfg.GrainSize = 40
	cfg.Shape = 0.6
	cfg.StereoSpread = 0.3
	cfg.Jitter = 0.3
	cfg.VelocityDensity = 0.6
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.0, 350.0, 1.0, 150.0)
	cfg.Level = 12.48 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
