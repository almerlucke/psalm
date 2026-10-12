package pd

import "github.com/almerlucke/genny/float/envelopes/adsr"

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"brass":    Brass,
	"strings":  Strings,
	"ePiano":   EPiano,
	"bass":     Bass,
	"resoBass": ResoBass,
	"organ":    Organ,
	"bell":     Bell,
	"lead":     Lead,
	"sweep":    Sweep,
	"clav":     Clav,

	"wahBass":    WahBass,
	"digiChoir":  DigiChoir,
	"zap":        Zap,
	"glassPad":   GlassPad,
	"resoArp":    ResoArp,
	"metalDrone": MetalDrone,
}

// env is an envelope: attack to 1, decay to a level, release (milliseconds)
func env(attack, decay, sustain, release float64) adsr.Setting {
	return *adsr.NewSetting(1, attack, sustain, decay, 1, release)
}

// Brass is a saw whose distortion swells in with the note, two lines detuned
func Brass() Config {
	l := Line{Waveform: Saw, Level: 1, DCW: 0.8, DCWEnv: env(80, 400, 0.6, 200), KeyFollow: 0.4, Velocity: 0.3,
		DCAEnv: env(30, 300, 0.85, 200)}
	l2 := l
	l2.Detune = 9
	return Config{Line1: l, Line2: l2, VibratoRate: 5, VibratoDepth: 0.08, VibratoDelay: 0.4, Width: 0.3, Level: 0.47}
}

// Strings are two slow saws, wide
func Strings() Config {
	l := Line{Waveform: Saw, Level: 1, DCW: 0.55, DCWEnv: env(500, 800, 0.8, 800), KeyFollow: 0.5,
		DCAEnv: env(350, 500, 0.9, 900)}
	l2 := l
	l2.Detune = 12
	return Config{Line1: l, Line2: l2, VibratoRate: 5.5, VibratoDepth: 0.06, VibratoDelay: 0.5, Width: 0.7, Level: 0.49}
}

// EPiano is a soft pulse with a quick bright attack, and a ringing octave
func EPiano() Config {
	return Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.5, DCWEnv: env(2, 600, 0.1, 300), KeyFollow: 0.6, Velocity: 0.5,
			DCAEnv: env(2, 2500, 0.2, 400)},
		Line2: Line{Waveform: Saw, Octave: 1, Level: 0.35, DCW: 0.2, DCWEnv: env(2, 300, 0, 200),
			DCAEnv: env(2, 900, 0, 300)},
		Width: 0.2, Level: 0.8,
	}
}

// Bass is a punchy saw, the distortion snapping shut
func Bass() Config {
	return Config{
		Line1: Line{Waveform: Saw, Level: 1, DCW: 0.9, DCWEnv: env(2, 250, 0.25, 100), Velocity: 0.4,
			DCAEnv: env(2, 600, 0.7, 80)},
		Line2: Line{Waveform: Square, Octave: -1, Level: 0.5, DCW: 0.3, DCWEnv: env(2, 200, 0.2, 100),
			DCAEnv: env(2, 600, 0.7, 80)},
		Level: 0.7,
	}
}

// ResoBass is the CZ's resonant bass: the resonance sweeping down
func ResoBass() Config {
	return Config{
		Line1: Line{Waveform: Reso1, Level: 1, DCW: 0.85, DCWEnv: env(2, 350, 0.15, 120), Velocity: 0.4,
			DCAEnv: env(2, 800, 0.7, 100)},
		Level: 1,
	}
}

// Organ is a square barely distorted, an octave over a soft saw
func Organ() Config {
	return Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.35, DCWEnv: env(5, 100, 1, 50), DCAEnv: env(5, 100, 1, 60)},
		Line2: Line{Waveform: Saw, Octave: 1, Level: 0.5, DCW: 0.2, DCWEnv: env(5, 100, 1, 50), DCAEnv: env(5, 100, 1, 60)},
		Width: 0.3, Level: 0.45,
	}
}

// Bell rings a pulse modulated by a detuned sine an octave and a fifth up
func Bell() Config {
	return Config{
		Line1: Line{Waveform: Pulse, Level: 1, DCW: 0.4, DCWEnv: env(2, 1500, 0, 1500), DCAEnv: env(2, 3500, 0, 2500)},
		Line2: Line{Waveform: Saw, Octave: 1, Detune: 702, Level: 1, DCW: 0.1, DCWEnv: env(2, 1000, 0, 1000),
			DCAEnv: env(2, 3000, 0, 2500)},
		Ring: true, Width: 0.3, Level: 0.78,
	}
}

// Lead alternates a saw and a pulse cycle by cycle, with a vibrato
func Lead() Config {
	l := Line{Waveform: Saw, Second: Pulse, Alternate: true, Level: 1, DCW: 0.75, DCWEnv: env(20, 300, 0.7, 150),
		Velocity: 0.3, DCAEnv: env(10, 200, 0.9, 150)}
	l2 := l
	l2.Detune, l2.Level = -6, 0.7
	return Config{Line1: l, Line2: l2, PitchSweep: -0.5, PitchTime: 0.08, VibratoRate: 5.5, VibratoDepth: 0.15,
		VibratoDelay: 0.3, Width: 0.25, Level: 0.53}
}

// Sweep is a pad whose resonance rises and falls slowly
func Sweep() Config {
	l := Line{Waveform: Reso2, Level: 1, DCW: 0.7, DCWEnv: env(1800, 2500, 0.3, 1500), DCAEnv: env(600, 1000, 0.85, 1500)}
	l2 := l
	l2.Waveform, l2.Detune = Reso3, 7
	return Config{Line1: l, Line2: l2, Width: 0.8, Level: 0.47}
}

// Clav is a short, plucked pulse
func Clav() Config {
	return Config{
		Line1: Line{Waveform: Pulse, Level: 1, DCW: 0.7, DCWEnv: env(1, 180, 0.3, 60), Velocity: 0.4,
			DCAEnv: env(1, 700, 0.3, 60)},
		Line2: Line{Waveform: SawPulse, Octave: 1, Level: 0.4, DCW: 0.6, DCWEnv: env(1, 150, 0.2, 60),
			DCAEnv: env(1, 400, 0.2, 60)},
		Width: 0.15, Level: 0.84,
	}
}

// WahBass is the resonance swept up and down in quarter notes, with the tempo: a talking CZ bass
func WahBass() Config {
	return Config{
		Line1: Line{Waveform: Reso2, Level: 0.696, DCW: 0.45, DCWEnv: env(2, 300, 0.8, 100), Velocity: 0.3,
			DCAEnv: env(2, 400, 0.85, 90)},
		Line2: Line{Waveform: Square, Octave: -1, Level: 0.45, DCW: 0.2, DCWEnv: env(2, 200, 0.5, 100),
			DCAEnv: env(2, 400, 0.85, 90)},
		SweepSync: 1, SweepDepth: 0.4, Level: 1,
	}
}

// DigiChoir is the CZ's glassy digital choir: the resonance in a window that holds, barely moving,
// two lines apart with a slow vibrato
func DigiChoir() Config {
	l := Line{Waveform: Reso3, Level: 0.527, DCW: 0.25, DCWEnv: env(700, 1500, 0.8, 1200), KeyFollow: 0.6,
		DCAEnv: env(500, 800, 0.9, 1400)}
	l2 := l
	l2.Detune, l2.Waveform = 14, Reso2
	return Config{Line1: l, Line2: l2, VibratoRate: 4.8, VibratoDepth: 0.12, VibratoDelay: 0.3, Width: 0.9,
		SweepRate: 0.25, SweepDepth: 0.08, Level: 1}
}

// Zap is a laser: the pitch falls three octaves in an instant, the resonance closing with it
func Zap() Config {
	return Config{
		Line1:      Line{Waveform: Reso1, Level: 1.39, DCW: 1, DCWEnv: env(1, 150, 0, 60), DCAEnv: env(1, 300, 0, 60)},
		PitchSweep: 36, PitchTime: 0.12, Level: 1,
	}
}

// GlassPad rings a pulse against a square a fifth up, the distortion breathing slowly
func GlassPad() Config {
	l := Line{Waveform: Pulse, Second: Square, Alternate: true, Level: 0.545, DCW: 0.35, DCWEnv: env(1200, 2000, 0.6, 1800),
		DCAEnv: env(800, 1000, 0.9, 2000)}
	l2 := l
	l2.Detune, l2.Level = 702, 0.6
	return Config{Line1: l, Line2: l2, Ring: true, Width: 0.8, SweepRate: 0.18, SweepDepth: 0.25, Level: 1}
}

// ResoArp is short: the resonance snaps from high to low on every note, for arpeggios
func ResoArp() Config {
	return Config{
		Line1: Line{Waveform: Reso1, Level: 1.01, DCW: 0.95, DCWEnv: env(1, 180, 0.1, 80), Velocity: 0.4,
			DCAEnv: env(1, 350, 0.2, 100)},
		Line2: Line{Waveform: Reso3, Octave: 1, Detune: 5, Level: 0.4, DCW: 0.8, DCWEnv: env(1, 120, 0.05, 80),
			DCAEnv: env(1, 250, 0.1, 100)},
		Width: 0.5, Level: 1,
	}
}

// MetalDrone ring modulates two saws a tritone apart, swept slowly: a held, clanging drone
func MetalDrone() Config {
	l := Line{Waveform: Saw, Level: 0.607, DCW: 0.6, DCWEnv: env(2000, 3000, 0.7, 2500), DCAEnv: env(1500, 1000, 0.9, 2500)}
	l2 := l
	l2.Waveform, l2.Detune = SawPulse, 600
	return Config{Line1: l, Line2: l2, Ring: true, Width: 0.6, SweepRate: 0.07, SweepDepth: 0.35, Level: 1}
}
