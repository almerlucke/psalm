package subtractive

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"bass":      BassPreset,
	"acidBass":  AcidBassPreset,
	"subBass":   SubBassPreset,
	"lead":      LeadPreset,
	"chipLead":  ChipLeadPreset,
	"pluck":     PluckPreset,
	"pad":       PadPreset,
	"strings":   StringsPreset,
	"brass":     BrassPreset,
	"organ":     OrganPreset,
	"wind":      WindPreset,
	"chipArp":   ChipArpPreset,
	"wobble":    WobblePreset,
	"hollowPad": HollowPadPreset,
	"syncLead":  SyncLeadPreset,
	"ringBell":  RingBellPreset,
	"pwmPad":    PWMPadPreset,
	"zap":       ZapPreset,

	"reeseBass":  ReeseBassPreset,
	"fmBass":     FMBassPreset,
	"squareBass": SquareBassPreset,
	"lfoWobble":  LFOWobblePreset,
	"synthKick":  SynthKickPreset,
	"synthSnare": SynthSnarePreset,
	"synthHat":   SynthHatPreset,
	"synthTom":   SynthTomPreset,
	"flute":      FlutePreset,
	"clarinet":   ClarinetPreset,
	"theremin":   ThereminPreset,
	"ePiano":     EPianoPreset,
	"glassKeys":  GlassKeysPreset,
	"vibes":      VibesPreset,
	"riser":      RiserPreset,
}

// BassPreset is a punchy saw bass with a square sub an octave below
func BassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Octave = -1
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.4
	cfg.Drive = 1.5
	cfg.Cutoff = 250.0
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 3.0
	cfg.KeyTrack = 0.3
	cfg.FilterEnv = *adsr.NewSetting(1.0, 2.0, 0.2, 180.0, 1.0, 120.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.8, 150.0, 1.0, 80.0)
	return cfg
}

// AcidBassPreset is a single squelchy saw with high resonance and a snappy filter envelope
func AcidBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.0
	cfg.Drive = 2.5
	cfg.Cutoff = 220.0
	cfg.Resonance = 0.85
	cfg.FilterEnvAmount = 4.5
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 200.0, 1.0, 80.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.9, 100.0, 1.0, 40.0)
	return cfg
}

// SubBassPreset is a clean, round low end without filter movement
func SubBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.8
	cfg.Osc2Level = 0.15
	cfg.Cutoff = 400.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.0
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 1.0, 10.0, 1.0, 120.0)
	return cfg
}

// LeadPreset is a bright detuned saw lead
func LeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.12
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.45
	cfg.Drive = 1.3
	cfg.Cutoff = 1800.0
	cfg.Resonance = 0.35
	cfg.FilterEnvAmount = 1.5
	cfg.KeyTrack = 0.8
	cfg.FilterEnv = *adsr.NewSetting(1.0, 10.0, 0.5, 300.0, 1.0, 200.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 8.0, 0.8, 200.0, 1.0, 250.0)
	return cfg
}

// ChipLeadPreset is a square plus narrow pulse an octave up, 8-bit style
func ChipLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.Osc2Detune = 12.08
	cfg.PulseWidth = 0.25
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.2
	cfg.Cutoff = 4000.0
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 0.5
	cfg.KeyTrack = 1.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 2.0, 0.6, 150.0, 1.0, 100.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.8, 150.0, 1.0, 120.0)
	return cfg
}

// ChipArpPreset is a thin pulse with a very short envelope for fast arpeggios
func ChipArpPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.125
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.0
	cfg.Cutoff = 2500.0
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 1.0
	cfg.KeyTrack = 1.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.2, 60.0, 1.0, 40.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.5, 60.0, 1.0, 40.0)
	return cfg
}

// PluckPreset is a short saw pluck with a fast closing filter
func PluckPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Detune = 0.05
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.25
	cfg.Cutoff = 300.0
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 4.0
	cfg.KeyTrack = 0.7
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 250.0, 1.0, 150.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 400.0, 1.0, 200.0)
	return cfg
}

// PadPreset is a wide, slow detuned saw pad
func PadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.15
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.NoiseLevel = 0.01
	cfg.Cutoff = 900.0
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 1.0
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1200.0, 0.6, 1500.0, 1.0, 2000.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 800.0, 0.9, 1000.0, 1.0, 1800.0)
	return cfg
}

// HollowPadPreset is a soft pulse pad with a darker, hollow character
func HollowPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.Osc2Waveform = blosc.Triangle
	cfg.PulseWidth = 0.35
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.35
	cfg.Cutoff = 700.0
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 0.8
	cfg.KeyTrack = 0.6
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1500.0, 0.5, 1500.0, 1.0, 2000.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1000.0, 0.85, 1000.0, 1.0, 2500.0)
	return cfg
}

// StringsPreset is a string ensemble style saw sound with a gentle swell
func StringsPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.08
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.45
	cfg.Cutoff = 2200.0
	cfg.Resonance = 0.05
	cfg.FilterEnvAmount = 0.3
	cfg.KeyTrack = 0.8
	cfg.FilterEnv = *adsr.NewSetting(1.0, 300.0, 0.8, 500.0, 1.0, 600.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 250.0, 0.9, 400.0, 1.0, 700.0)
	return cfg
}

// BrassPreset is a saw brass with the classic delayed filter swell
func BrassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.06
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.45
	cfg.Drive = 1.4
	cfg.Cutoff = 500.0
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 2.5
	cfg.KeyTrack = 0.6
	cfg.FilterEnv = *adsr.NewSetting(1.0, 80.0, 0.6, 400.0, 1.0, 200.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 40.0, 0.85, 300.0, 1.0, 200.0)
	return cfg
}

// OrganPreset is a square plus an octave triangle with an instant, steady envelope
func OrganPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.35
	cfg.Osc2Level = 0.4
	cfg.Cutoff = 3000.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3.0, 1.0, 10.0, 1.0, 40.0)
	return cfg
}

// WobblePreset is a resonant bass meant to have its cutoff driven by an LFO (input 1)
func WobblePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Octave = -1
	cfg.Osc2Detune = 0.03
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.4
	cfg.Drive = 2.0
	cfg.Cutoff = 400.0
	cfg.Resonance = 0.6
	cfg.FilterEnvAmount = 0.5
	cfg.KeyTrack = 0.2
	cfg.FilterEnv = *adsr.NewSetting(1.0, 5.0, 0.8, 100.0, 1.0, 100.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3.0, 0.9, 100.0, 1.0, 80.0)
	return cfg
}

// WindPreset is filtered noise through a resonant bandpass, the pitch sets the band
func WindPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Level = 0.0
	cfg.Osc2Level = 0.0
	cfg.NoiseLevel = 0.8
	cfg.FilterMode = BandPass
	cfg.Cutoff = 500.0
	cfg.Resonance = 0.8
	cfg.FilterEnvAmount = 1.0
	cfg.KeyTrack = 1.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1500.0, 0.3, 2000.0, 1.0, 2000.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1000.0, 0.8, 1000.0, 1.0, 2000.0)
	return cfg
}

// SyncLeadPreset is a hard synced lead, the filter envelope sweeps osc 2 for the classic sync scream
func SyncLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc1Level = 0.2
	cfg.Osc2Level = 0.5
	cfg.Osc2Detune = 7.0
	cfg.Sync = true
	cfg.Cutoff = 2500.0
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 1.0
	cfg.KeyTrack = 0.8
	cfg.FilterEnv = *adsr.NewSetting(1.0, 5.0, 0.4, 400.0, 1.0, 300.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.8, 200.0, 1.0, 250.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.5, Delay: 300.0, Pitch: 0.25}
	return cfg
}

// RingBellPreset is a metallic bell from ring modulating two inharmonic sines
func RingBellPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc1Level = 0.2
	cfg.Osc2Level = 0.0
	cfg.Osc2Detune = 16.3 // inharmonic ratio for a bell-like spectrum
	cfg.RingModLevel = 0.7
	cfg.Cutoff = 5000.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 1500.0, 1.0, 800.0)
	return cfg
}

// PWMPadPreset is a lush pulse width modulated pad with slow vibrato
func PWMPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.PulseWidth = 0.5
	cfg.SubLevel = 0.2
	cfg.Cutoff = 1200.0
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 0.8
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1000.0, 0.6, 1500.0, 1.0, 2000.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 700.0, 0.9, 1000.0, 1.0, 1800.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 0.4, PulseWidth: 0.35}
	cfg.LFO2 = LFOConfig{Waveform: LFOSine, Rate: 4.5, Delay: 600.0, Pitch: 0.1, Cutoff: 0.2}
	return cfg
}

// ZapPreset is a laser zap / percussive blip using a fast filter envelope pitch drop
func ZapPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.0
	cfg.FilterEnvToPitch = 36.0
	cfg.Cutoff = 3000.0
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 1.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 120.0, 1.0, 50.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 250.0, 1.0, 80.0)
	return cfg
}

// ReeseBassPreset is a dark bass of two heavily detuned saws with a slowly moving filter
func ReeseBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.25
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.45
	cfg.SubLevel = 0.3
	cfg.Drive = 1.8
	cfg.Cutoff = 350.0
	cfg.Resonance = 0.25
	cfg.FilterEnvAmount = 1.0
	cfg.KeyTrack = 0.3
	cfg.FilterEnv = *adsr.NewSetting(1.0, 20.0, 0.5, 400.0, 1.0, 200.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.9, 200.0, 1.0, 150.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 0.3, Cutoff: 1.2}
	return cfg
}

// FMBassPreset is a growly bass from a sine phase modulated by a saw
func FMBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.0
	cfg.Osc2Level = 0.6
	cfg.FMAmount = 3.0
	cfg.SubLevel = 0.3
	cfg.Cutoff = 900.0
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 2.0
	cfg.KeyTrack = 0.4
	cfg.FilterEnv = *adsr.NewSetting(1.0, 2.0, 0.2, 200.0, 1.0, 100.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.8, 150.0, 1.0, 100.0)
	return cfg
}

// SquareBassPreset is a hollow, slightly rubbery square bass in the style of classic monosynths
func SquareBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.4
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.0
	cfg.SubLevel = 0.4
	cfg.Cutoff = 300.0
	cfg.Resonance = 0.4
	cfg.FilterEnvAmount = 2.5
	cfg.KeyTrack = 0.3
	cfg.Glide = 30.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 2.0, 0.2, 250.0, 1.0, 120.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.85, 150.0, 1.0, 100.0)
	return cfg
}

// LFOWobblePreset is a wobble bass with the cutoff driven by the internal LFO
func LFOWobblePreset() Config {
	cfg := WobblePreset()
	cfg.Cutoff = 250.0
	cfg.Resonance = 0.65
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 4.0, KeySync: true, Cutoff: 2.5}
	return cfg
}

// SynthKickPreset is an analog style kick: a sine with a fast falling pitch, play around MIDI 36
func SynthKickPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc1Level = 0.9
	cfg.Osc2Level = 0.0
	cfg.FilterEnvToPitch = 36.0
	cfg.Drive = 1.5
	cfg.Cutoff = 3000.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 45.0, 1.0, 20.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 350.0, 1.0, 60.0)
	return cfg
}

// SynthSnarePreset mixes a pitched triangle body with a burst of noise, play around MIDI 50
func SynthSnarePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Triangle
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.0
	cfg.NoiseLevel = 0.7
	cfg.FilterEnvToPitch = 12.0
	cfg.Cutoff = 3500.0
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 1.0
	cfg.KeyTrack = 0.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 60.0, 1.0, 30.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 180.0, 1.0, 50.0)
	return cfg
}

// SynthHatPreset is highpassed noise with a very short envelope
func SynthHatPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Level = 0.0
	cfg.Osc2Level = 0.0
	cfg.NoiseLevel = 0.8
	cfg.FilterMode = HighPass
	cfg.Cutoff = 7000.0
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.0
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 50.0, 1.0, 30.0)
	return cfg
}

// SynthTomPreset is a pitched sine/triangle tom with a short pitch drop, pitch sets the drum size
func SynthTomPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.6
	cfg.Osc2Level = 0.2
	cfg.NoiseLevel = 0.05
	cfg.FilterEnvToPitch = 7.0
	cfg.Cutoff = 2500.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 120.0, 1.0, 40.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 400.0, 1.0, 80.0)
	return cfg
}

// FlutePreset is a soft triangle with breath noise and a delayed vibrato
func FlutePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Triangle
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.1
	cfg.NoiseLevel = 0.06
	cfg.NoiseType = PinkNoise
	cfg.Cutoff = 2000.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.5
	cfg.KeyTrack = 1.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 80.0, 0.7, 300.0, 1.0, 200.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 70.0, 0.85, 200.0, 1.0, 180.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.0, Delay: 350.0, KeySync: true, Pitch: 0.15, Amp: 0.1}
	return cfg
}

// ClarinetPreset is a woody, hollow square with a gentle attack
func ClarinetPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.0
	cfg.NoiseLevel = 0.02
	cfg.NoiseType = PinkNoise
	cfg.Cutoff = 1100.0
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 0.6
	cfg.KeyTrack = 0.9
	cfg.FilterEnv = *adsr.NewSetting(1.0, 60.0, 0.6, 300.0, 1.0, 150.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 50.0, 0.85, 200.0, 1.0, 150.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 4.5, Delay: 500.0, KeySync: true, Pitch: 0.08}
	return cfg
}

// ThereminPreset is a pure sine with long glide and a wide vibrato, best played on a single voice
func ThereminPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.6
	cfg.Osc2Level = 0.1
	cfg.Glide = 180.0
	cfg.Cutoff = 4000.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 150.0, 0.9, 200.0, 1.0, 400.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 6.0, Delay: 200.0, Pitch: 0.35}
	return cfg
}

// EPianoPreset is an electric piano like tone from a gently FM'd sine pair with soft tremolo
func EPianoPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Detune = 0.02
	cfg.Osc1Level = 0.3
	cfg.Osc2Level = 0.4
	cfg.FMAmount = 1.2
	cfg.Cutoff = 2500.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 1.0
	cfg.KeyTrack = 0.7
	cfg.FilterEnv = *adsr.NewSetting(1.0, 2.0, 0.3, 600.0, 1.0, 300.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3.0, 0.4, 1500.0, 1.0, 500.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 4.0, Amp: 0.25}
	return cfg
}

// GlassKeysPreset is a bright, glassy bell keyboard from FM with an octave up modulator
func GlassKeysPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.0
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.25
	cfg.FMAmount = 2.5
	cfg.Cutoff = 6000.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 1200.0, 1.0, 600.0)
	return cfg
}

// VibesPreset is a mallet tone with the classic motor driven tremolo
func VibesPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Detune = 24.0 // a faint high partial for the mallet strike
	cfg.Osc1Level = 0.6
	cfg.Osc2Level = 0.08
	cfg.Cutoff = 5000.0
	cfg.Resonance = 0.0
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.0, 1800.0, 1.0, 800.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.5, Amp: 0.5}
	return cfg
}

// RiserPreset is a noise and saw sweep effect that rises in pitch and brightness over a few seconds
func RiserPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.2
	cfg.Osc1Level = 0.25
	cfg.Osc2Level = 0.25
	cfg.NoiseLevel = 0.5
	cfg.FilterMode = BandPass
	cfg.Cutoff = 300.0
	cfg.Resonance = 0.6
	cfg.FilterEnvAmount = 4.0
	cfg.FilterEnvToPitch = 12.0
	cfg.KeyTrack = 0.0
	cfg.FilterEnv = *adsr.NewSetting(1.0, 4000.0, 1.0, 10.0, 1.0, 300.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3500.0, 1.0, 10.0, 1.0, 300.0)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 6.0, Amp: 0.3}
	return cfg
}
