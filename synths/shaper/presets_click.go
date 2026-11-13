package shaper

import "github.com/almerlucke/psalm/blocks/voicefilter"

// Tick is a small click, a closed hat: a burst of bright noise through a high pass, over in thirty milliseconds. It has no pitch to speak of: any key plays it
func Tick() Config {
	return Config{
		Chains: [NumChains]string{"sin"}, Line1: Line{Level: 0}, Noise: 1, NoiseColor: 1,
		AmpEnv: env(1, 28, 0, 22), Level: 4,
		Filter: voicefilter.Config{Type: voicefilter.HighPass, Cutoff: 6500, Resonance: 0.15, EnvAmount: 0, KeyTrack: 0, Env: env(1, 100, 1, 100)},
	}
}
