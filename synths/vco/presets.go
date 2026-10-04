package vco

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
