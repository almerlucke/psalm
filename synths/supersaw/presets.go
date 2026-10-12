package supersaw

import "github.com/almerlucke/genny/float/envelopes/adsr"

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"trancePad":  TrancePad,
	"lead":       Lead,
	"pluck":      Pluck,
	"hoover":     Hoover,
	"stab":       Stab,
	"unisonBass": UnisonBass,

	"hypersaw":   Hypersaw,
	"wobbleBass": WobbleBass,
	"breathing":  Breathing,
	"swarm":      Swarm,
	"glideLead":  GlideLead,
}

// env is an envelope: attack to 1, decay to a level, release (milliseconds)
func env(attack, decay, sustain, release float64) adsr.Setting {
	return *adsr.NewSetting(1, attack, sustain, decay, 1, release)
}

// TrancePad is wide and slow, the filter opening with the swell
func TrancePad() Config {
	return Config{Detune: 0.5, Mix: 0.7, Stereo: 1, HighPass: true, Cutoff: 1800, Resonance: 0.1, FilterEnvAmount: 1.5,
		KeyTrack: 0.5, FilterEnv: env(1200, 2000, 0.4, 1500), AmpEnv: env(600, 800, 0.9, 1500), Level: 0.72}
}

// Lead is bright and fast, with a little glide
func Lead() Config {
	return Config{Detune: 0.3, Mix: 0.6, Stereo: 0.6, HighPass: true, Cutoff: 3500, Resonance: 0.15, FilterEnvAmount: 1,
		KeyTrack: 0.6, Velocity: 1, FilterEnv: env(5, 400, 0.6, 300), AmpEnv: env(5, 200, 0.9, 250), Glide: 0.04, Level: 0.82}
}

// Pluck is short: the filter snaps shut
func Pluck() Config {
	return Config{Detune: 0.35, Mix: 0.6, Stereo: 0.8, HighPass: true, Cutoff: 500, Resonance: 0.2, FilterEnvAmount: 4,
		KeyTrack: 0.5, Velocity: 1, FilterEnv: env(2, 220, 0, 200), AmpEnv: env(2, 450, 0, 200), Level: 1.05}
}

// Hoover is the rave hoover: detuned wide, a sub, and a pitch that glides
func Hoover() Config {
	return Config{Detune: 0.8, Mix: 0.9, Stereo: 1, Sub: 0.6, Cutoff: 2500, Resonance: 0.1, FilterEnvAmount: 0.5,
		FilterEnv: env(50, 600, 0.6, 400), AmpEnv: env(20, 300, 0.9, 400), Glide: 0.15, Level: 0.48}
}

// Stab is a chord stab: open, short, punchy
func Stab() Config {
	return Config{Detune: 0.4, Mix: 0.65, Stereo: 0.9, HighPass: true, Cutoff: 1200, Resonance: 0.15,
		FilterEnvAmount: 3, Velocity: 1, FilterEnv: env(2, 150, 0.2, 150), AmpEnv: env(2, 250, 0.4, 150), Level: 1.15}
}

// UnisonBass is a narrow supersaw with a sub, low and round
func UnisonBass() Config {
	return Config{Detune: 0.15, Mix: 0.45, Stereo: 0.3, Sub: 0.7, Cutoff: 350, Resonance: 0.25, FilterEnvAmount: 2.5,
		KeyTrack: 0.3, Velocity: 0.8, FilterEnv: env(2, 300, 0.2, 150), AmpEnv: env(2, 400, 0.8, 120), Level: 0.41}
}

// Hypersaw is sixteen saws, as wide and detuned as they go: a wall of sound
func Hypersaw() Config {
	return Config{Saws: 16, Detune: 0.9, Mix: 1, Stereo: 1, HighPass: true, Cutoff: 7000, Resonance: 0.05,
		FilterEnvAmount: 0.5, FilterEnv: env(800, 1500, 0.6, 1200), AmpEnv: env(250, 600, 0.9, 1400), Level: 1.01}
}

// WobbleBass wobbles its filter in eighth notes, with the tempo: a dubstep bass
func WobbleBass() Config {
	return Config{Detune: 0.2, Mix: 0.5, Stereo: 0.4, Sub: 0.8, Cutoff: 280, Resonance: 0.55, KeyTrack: 0.3,
		LFOSync: 0.5, LFOCutoff: 2.6, FilterEnv: env(2, 200, 0, 100), AmpEnv: env(3, 300, 0.95, 120), Glide: 0.03, Level: 0.253}
}

// Breathing is a pad whose detune swells and settles slowly: the saws drift apart and come back together
func Breathing() Config {
	return Config{Detune: 0.15, Mix: 0.75, Stereo: 1, HighPass: true, Cutoff: 2600, Resonance: 0.1, KeyTrack: 0.4,
		LFORate: 0.12, LFODetune: 0.75, LFOCutoff: 0.4, FilterEnv: env(1500, 2000, 0.6, 1500),
		AmpEnv: env(900, 1000, 0.9, 2000), Level: 0.7}
}

// Swarm rises: the filter opens over four seconds through a resonant peak, the saws wide apart
func Swarm() Config {
	return Config{Detune: 0.7, Mix: 0.85, Stereo: 1, HighPass: true, Cutoff: 150, Resonance: 0.45, FilterEnvAmount: 6,
		FilterEnv: env(4000, 3000, 0.7, 2000), AmpEnv: env(1500, 1000, 0.9, 2000), LFORate: 5.5, LFOCutoff: 0.25, Level: 0.58}
}

// GlideLead slides between its notes like a theremin, bright and detuned
func GlideLead() Config {
	return Config{Detune: 0.45, Mix: 0.7, Stereo: 0.7, HighPass: true, Cutoff: 4500, Resonance: 0.2, FilterEnvAmount: 1,
		KeyTrack: 0.5, Velocity: 1, FilterEnv: env(10, 500, 0.6, 300), AmpEnv: env(10, 200, 0.9, 300), Glide: 0.25,
		LFORate: 5, LFOCutoff: 0.3, Level: 0.765}
}
