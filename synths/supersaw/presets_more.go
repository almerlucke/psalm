package supersaw

import "github.com/almerlucke/psalm/blocks/modmatrix"

// mod is a slot of the matrix
func mod(src modmatrix.Source, dest string, amount float64) modmatrix.Slot {
	return modmatrix.Slot{Src: src, Dest: dest, Amount: amount}
}

// StringMachine is the ensemble strings: a narrow supersaw through the three delay chorus, a slow swell
func StringMachine() Config {
	return Config{Detune: 0.12, Mix: 0.5, Stereo: 0.9, HighPass: true, Cutoff: 3000, Resonance: 0.05, FilterEnvAmount: 0.3,
		KeyTrack: 0.7, FilterEnv: env(300, 500, 0.8, 600), AmpEnv: env(500, 600, 0.95, 900), Ensemble: 0.85, EnsembleRate: 0.6, Level: 1.25}
}

// ChoirWash is a vast, hazy wash: the cutoff low and slow, the ensemble wide and slow, the lows cut at 350 Hz
func ChoirWash() Config {
	return Config{Detune: 0.25, Mix: 0.6, Stereo: 1, HPCutoff: 350, Cutoff: 1400, Resonance: 0.3, FilterEnvAmount: 1.2,
		KeyTrack: 0.6, FilterEnv: env(1500, 2000, 0.5, 2000), AmpEnv: env(900, 800, 0.9, 2200), Ensemble: 1, EnsembleRate: 0.35,
		LFORate: 0.2, LFOCutoff: 0.3, Level: 1.05}
}

// EpicPad is sixteen saws wide open, the wheel spreading them further and the velocity opening the filter
func EpicPad() Config {
	return Config{Saws: 16, Detune: 0.55, Mix: 0.8, Stereo: 1, HighPass: true, Cutoff: 2200, Resonance: 0.1, FilterEnvAmount: 2,
		KeyTrack: 0.6, FilterEnv: env(1800, 2500, 0.5, 2500), AmpEnv: env(1000, 800, 0.9, 2500), Ensemble: 0.4,
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Wheel, "detune", 0.3), mod(modmatrix.Velocity, "cutoff", 0.1)}, Level: 1.107}
}

// DarkSwarm is a low, heavy swarm: a sub under a dark filter that opens and closes with a very slow LFO
func DarkSwarm() Config {
	return Config{Detune: 0.6, Mix: 0.9, Stereo: 1, Sub: 0.5, Cutoff: 300, Resonance: 0.35, FilterEnvAmount: 3,
		KeyTrack: 0.4, FilterEnv: env(3000, 3000, 0.5, 3000), AmpEnv: env(1500, 800, 0.9, 3000), LFORate: 0.08, LFOCutoff: 0.8, Level: 0.5114}
}

// ShimmerPad is a high, glittering pad: the lows cut at 900 Hz, an ensemble that breathes with LFO 2
func ShimmerPad() Config {
	return Config{Detune: 0.35, Mix: 0.4, Stereo: 1, HPCutoff: 900, Cutoff: 9000, Resonance: 0.1, KeyTrack: 0.8,
		FilterEnv: env(800, 1000, 0.8, 1500), AmpEnv: env(900, 800, 0.9, 2500), Ensemble: 0.6, EnsembleRate: 0.9,
		LFO2Rate: 0.15, LFO2Shape: 1, Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.LFO2, "ensemble", 0.4)}, Level: 1.412}
}

// AnthemLead is a stadium trance lead: a glide, a vibrato from LFO 2 and the wheel pulling the saws apart
func AnthemLead() Config {
	return Config{Detune: 0.4, Mix: 0.75, Stereo: 0.8, HighPass: true, Cutoff: 3000, Resonance: 0.15, FilterEnvAmount: 1.5,
		KeyTrack: 0.7, Velocity: 1.2, FilterEnv: env(5, 450, 0.55, 300), AmpEnv: env(5, 200, 0.9, 300), Glide: 0.06,
		Ensemble: 0.25, LFO2Rate: 5.5,
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.LFO2, "pitch", 0.02), mod(modmatrix.Wheel, "detune", 0.3)}, Level: 0.8613}
}

// ScreamingSaw is a resonant, biting lead: the velocity opens it and pushes the resonance
func ScreamingSaw() Config {
	return Config{Detune: 0.25, Mix: 0.5, Stereo: 0.5, HighPass: true, Cutoff: 1500, Resonance: 0.6, FilterEnvAmount: 2.5,
		KeyTrack: 0.8, FilterEnv: env(5, 500, 0.4, 250), AmpEnv: env(5, 200, 0.9, 250),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Velocity, "cutoff", 0.15), mod(modmatrix.Velocity, "resonance", 0.2)}, Level: 0.7369}
}

// HardstyleLead is a hard, screeching lead: seven fat saws with a sub, a short pitch kick at the start
func HardstyleLead() Config {
	return Config{Detune: 0.5, Mix: 1, Stereo: 0.9, Sub: 0.3, HPCutoff: 180, Cutoff: 5000, Resonance: 0.2, FilterEnvAmount: 1,
		KeyTrack: 0.5, FilterEnv: env(2, 120, 0.5, 100), AmpEnv: env(2, 150, 0.85, 80),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.FilterEnv, "pitch", 0.15)}, Level: 0.7308}
}

// ArpSaw is a short sequenced saw for arpeggios: a snappy filter, the velocity as the accent
func ArpSaw() Config {
	return Config{Saws: 5, Detune: 0.3, Mix: 0.6, Stereo: 0.7, HighPass: true, Cutoff: 700, Resonance: 0.25, FilterEnvAmount: 3.5,
		KeyTrack: 0.8, FilterEnv: env(1, 160, 0, 80), AmpEnv: env(1, 220, 0, 80),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Velocity, "cutoff", 0.2)}, Level: 0.9175}
}

// GatedPad is the trance gate: a pad chopped into eighth notes by the level, in time with the tempo
func GatedPad() Config {
	return Config{Detune: 0.35, Mix: 0.7, Stereo: 1, HighPass: true, Cutoff: 3500, Resonance: 0.1, KeyTrack: 0.6,
		FilterEnv: env(10, 200, 1, 200), AmpEnv: env(40, 200, 1, 300), LFOSync: 0.5,
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.LFO1, "level", 0.45)}, Level: 0.4031}
}

// PumpPad is a sidechained pad: it ducks on every beat and swells back, the filter breathing with it
func PumpPad() Config {
	return Config{Detune: 0.3, Mix: 0.65, Stereo: 1, HighPass: true, Cutoff: 2500, Resonance: 0.1, KeyTrack: 0.6,
		FilterEnv: env(10, 200, 1, 200), AmpEnv: env(60, 200, 1, 500), LFOSync: 1, Ensemble: 0.3,
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.LFO1, "level", -0.4), mod(modmatrix.LFO1, "cutoff", -0.15)}, Level: 0.6384}
}

// DubstepGrowl is a bass with its cutoff swept four times a beat, the wheel speeding the sweep
func DubstepGrowl() Config {
	return Config{Detune: 0.15, Mix: 0.3, Stereo: 0.4, Sub: 0.7, Cutoff: 200, Resonance: 0.5, KeyTrack: 0.3, LFOSync: 0.25,
		LFOCutoff: 3, FilterEnv: env(2, 200, 1, 100), AmpEnv: env(2, 300, 0.95, 100), Glide: 0.02,
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Wheel, "lfoRate", 0.35)}, Level: 0.3338}
}

// HouseStab is a chord stab with the lows cut at 150 Hz: short, bright and punchy, the velocity opens it
func HouseStab() Config {
	return Config{Detune: 0.3, Mix: 0.6, Stereo: 0.9, Sub: 0.3, HPCutoff: 150, Cutoff: 1500, Resonance: 0.2, FilterEnvAmount: 3,
		KeyTrack: 0.6, FilterEnv: env(2, 140, 0.15, 100), AmpEnv: env(2, 200, 0.3, 120),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Velocity, "cutoff", 0.15)}, Level: 1.14}
}

// Dive is an effect that falls: the pitch drops an octave over the first seconds as the filter closes
func Dive() Config {
	return Config{Detune: 0.8, Mix: 1, Stereo: 1, HighPass: true, Cutoff: 5000, Resonance: 0.2, FilterEnvAmount: -2,
		FilterEnv: env(1, 1800, 0, 800), AmpEnv: env(1, 2000, 0, 800),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.FilterEnv, "pitch", 1)}, Level: 0.832}
}

// AirPad is a light, open pad: few detuned saws, the lows cut and the ensemble around it
func AirPad() Config {
	return Config{Detune: 0.2, Mix: 0.3, Stereo: 1, HPCutoff: 600, Cutoff: 3800, Resonance: 0.05, KeyTrack: 0.8,
		FilterEnv: env(700, 800, 0.8, 1200), AmpEnv: env(1000, 800, 0.9, 2000), Ensemble: 0.7, EnsembleRate: 0.5, Level: 2.012}
}

// BrassSection is a saw brass: five close saws, a swell of the filter that follows the velocity
func BrassSection() Config {
	return Config{Saws: 5, Detune: 0.08, Mix: 0.5, Stereo: 0.5, Cutoff: 500, Resonance: 0.15, FilterEnvAmount: 3, KeyTrack: 0.6,
		Velocity: 0.8, FilterEnv: env(90, 400, 0.6, 250), AmpEnv: env(40, 200, 0.9, 250), Ensemble: 0.3, Level: 1.041}
}

// TrioLead is a lead of three saws: tight, a little glide and a vibrato
func TrioLead() Config {
	return Config{Saws: 3, Detune: 0.25, Mix: 0.5, Stereo: 0.5, HighPass: true, Cutoff: 2800, Resonance: 0.15, FilterEnvAmount: 1,
		KeyTrack: 0.8, FilterEnv: env(5, 350, 0.6, 250), AmpEnv: env(5, 150, 0.9, 220), Glide: 0.1, LFO2Rate: 5.2,
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.LFO2, "pitch", 0.015)}, Level: 0.7656}
}

// DropBass is a sub drop: a sine-like low end that falls a fifth in the first moments
func DropBass() Config {
	return Config{Detune: 0.1, Mix: 0.3, Stereo: 0.2, Sub: 1, Cutoff: 220, Resonance: 0.2, FilterEnvAmount: 2, KeyTrack: 0.2,
		FilterEnv: env(1, 60, 0, 50), AmpEnv: env(2, 800, 0, 150),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.FilterEnv, "pitch", 0.5)}, Level: 0.4903}
}

// ReeseBass is a rolling reese: three wide saws and a sub under a filter moving slowly
func ReeseBass() Config {
	return Config{Saws: 3, Detune: 0.3, Mix: 0.9, Stereo: 0.6, Sub: 0.5, Cutoff: 450, Resonance: 0.25, KeyTrack: 0.3,
		FilterEnv: env(5, 300, 1, 200), AmpEnv: env(5, 300, 0.95, 200), LFORate: 0.2, LFOCutoff: 0.7, Level: 0.4384}
}

// RavePluck is a bright plucked chord with a touch of ensemble: the filter snaps shut
func RavePluck() Config {
	return Config{Detune: 0.5, Mix: 0.7, Stereo: 1, HighPass: true, Cutoff: 1200, Resonance: 0.3, FilterEnvAmount: 3.5, KeyTrack: 0.6,
		FilterEnv: env(1, 260, 0, 200), AmpEnv: env(1, 500, 0, 250), Ensemble: 0.3, Level: 1.25}
}

// AnalogPad is a pad that is never the same twice: every note gets a random detune and a hair of random pitch
func AnalogPad() Config {
	return Config{Saws: 5, Detune: 0.12, Mix: 0.3, Stereo: 0.6, Sub: 0.2, Cutoff: 1100, Resonance: 0.2, FilterEnvAmount: 1.5,
		KeyTrack: 0.4, FilterEnv: env(500, 1200, 0.5, 1500), AmpEnv: env(350, 800, 0.9, 1800),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Random, "detune", 0.3), mod(modmatrix.Random, "pitch", 0.01)}, Level: 0.8031}
}

// ExpressiveLead answers the player: the velocity opens the filter and spreads the saws, the wheel brings in the ensemble
func ExpressiveLead() Config {
	return Config{Saws: 5, Detune: 0.18, Mix: 0.25, Stereo: 0.7, Cutoff: 900, Resonance: 0.1, FilterEnvAmount: 1.2, Ensemble: 0.15,
		KeyTrack: 0.5, FilterEnv: env(8, 400, 0.4, 300), AmpEnv: env(8, 200, 0.9, 300), Glide: 0.03,
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Velocity, "cutoff", 0.25), mod(modmatrix.Velocity, "detune", 0.3),
			mod(modmatrix.Wheel, "ensemble", 0.5), mod(modmatrix.Wheel, "resonance", 0.2)}, Level: 0.9325}
}

// MorphPad changes with the wheel: from a narrow, dry pad to a wide, thick ensemble
func MorphPad() Config {
	return Config{Detune: 0.1, Mix: 0.4, Stereo: 0.6, HighPass: true, Cutoff: 2400, Resonance: 0.1, KeyTrack: 0.6,
		FilterEnv: env(600, 1000, 0.8, 1200), AmpEnv: env(600, 800, 0.9, 1800),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.Wheel, "detune", 0.8), mod(modmatrix.Wheel, "mix", 0.4),
			mod(modmatrix.Wheel, "ensemble", 0.6)}, Level: 1.189}
}

// RiseUp is a riser: the pitch and the cutoff climb for three seconds
func RiseUp() Config {
	return Config{Detune: 0.6, Mix: 0.9, Stereo: 1, HighPass: true, Cutoff: 100, Resonance: 0.3, FilterEnvAmount: 6,
		FilterEnv: env(3200, 500, 1, 400), AmpEnv: env(2500, 500, 1, 400),
		Mods: [modmatrix.Slots]modmatrix.Slot{mod(modmatrix.FilterEnv, "pitch", 0.6)}, Level: 1.066}
}

// SoftPluck is a gentle ensemble pluck: three saws, a mellow filter closing, the chorus wide
func SoftPluck() Config {
	return Config{Saws: 3, Detune: 0.2, Mix: 0.5, Stereo: 0.8, HighPass: true, Cutoff: 600, Resonance: 0.1, FilterEnvAmount: 2.5,
		KeyTrack: 0.7, FilterEnv: env(1, 350, 0, 300), AmpEnv: env(1, 700, 0, 400), Ensemble: 0.5, Level: 1.484}
}

// GliderBass is a mono bass that slides between its notes, three saws and a sub
func GliderBass() Config {
	return Config{Saws: 3, Detune: 0.12, Mix: 0.5, Stereo: 0.3, Sub: 0.6, Cutoff: 400, Resonance: 0.25, FilterEnvAmount: 2.5,
		KeyTrack: 0.3, FilterEnv: env(2, 250, 0.2, 150), AmpEnv: env(2, 300, 0.85, 120), Glide: 0.08, Level: 0.4304}
}
