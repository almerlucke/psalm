package pd

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

// More presets, in the territory of the Casio CZ: the resonant waveforms for sweeps, voices and
// bells, the DCO envelope for kicks, zaps and the scoop into a bass note, the depth envelope and the
// filter for darker patches, the matrix for a velocity that brightens and a wheel that sweeps.
//
// The levels of the lines are 1 unless they play a supporting part

// filt is a filter of the type, cutoff (Hz), resonance, envelope amount (octaves) and key tracking,
// the envelope opens from the attack to the sustain level
func filt(t voicefilter.Type, cutoff, res, amount, track float64, e ...float64) voicefilter.Config {
	f := voicefilter.Config{Type: t, Cutoff: cutoff, Resonance: res, EnvAmount: amount, KeyTrack: track,
		Env: env(1, 300, 0.3, 300)}
	if len(e) == 4 {
		f.Env = env(e[0], e[1], e[2], e[3])
	}
	return f
}

// route fills slot i of the matrix
func route(cfg *Config, i int, src modmatrix.Source, dest string, amount float64) {
	cfg.Mods[i] = modmatrix.Slot{Src: src, Dest: dest, Amount: amount}
}

// Basses

// DCOBass is a synth bass with the DCO envelope: the pitch scoops up five semitones into every note
// while the distortion snaps shut, a square an octave under
func DCOBass() Config {
	cfg := Config{
		Line1: Line{Waveform: Saw, Level: 1, DCW: 0.8, DCWEnv: env(2, 200, 0.3, 80), Velocity: 0.4,
			DCAEnv: env(2, 500, 0.8, 80)},
		Line2: Line{Waveform: Square, Octave: -1, Level: 0.5, DCW: 0.3, DCWEnv: env(2, 150, 0.3, 80),
			DCAEnv: env(2, 500, 0.8, 80)},
		DCOAmount: -5, DCOEnv: env(1, 45, 0, 40),
	}
	cfg.Level = 0.6372 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SubBass is a sine sub: an almost undistorted square and the octave under it
func SubBass() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.06, DCWEnv: env(2, 100, 1, 80), DCAEnv: env(4, 200, 0.9, 120)},
		Line2: Line{Waveform: Square, Octave: -1, Level: 0.6, DCW: 0.03, DCWEnv: env(2, 100, 1, 80),
			DCAEnv: env(4, 200, 0.9, 120)},
	}
	cfg.Level = 0.5565 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SquareBass is a hollow square bass with a lowpass that snaps shut, a detuned square under
func SquareBass() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.8, DCWEnv: env(2, 150, 0.4, 80), DCAEnv: env(2, 400, 0.85, 90)},
		Line2: Line{Waveform: Square, Octave: -1, Detune: 5, Level: 0.6, DCW: 0.6, DCWEnv: env(2, 150, 0.4, 80),
			DCAEnv: env(2, 400, 0.85, 90)},
		Filter: filt(voicefilter.LowPass, 600, 0.2, 2, 0.4, 1, 250, 0.1, 100),
	}
	cfg.Level = 0.5315 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AcidBass is a squelching line: the resonance waveform under a resonant ladder filter that
// snaps open on every note, the velocity opening it further
func AcidBass() Config {
	cfg := Config{
		Line1:  Line{Waveform: Reso1, Level: 1, DCW: 0.7, DCWEnv: env(1, 180, 0.1, 60), DCAEnv: env(1, 400, 0.8, 70)},
		Filter: filt(voicefilter.Ladder24, 300, 0.75, 3.5, 0.4, 1, 200, 0, 80),
	}
	route(&cfg, 0, modmatrix.Velocity, "cutoff", 0.3)
	route(&cfg, 1, modmatrix.Velocity, "resonance", 0.1)
	cfg.Level = 1.4001 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SlapBass is a slap: a pulse with a hard snap of distortion and a pitch that drops in a flash, a saw
// an octave down for the body; the harder the note the brighter
func SlapBass() Config {
	cfg := Config{
		Line1: Line{Waveform: Pulse, Level: 1, DCW: 0.9, DCWEnv: env(1, 90, 0.15, 60), Velocity: 0.4,
			DCAEnv: env(1, 450, 0.5, 70)},
		Line2: Line{Waveform: Saw, Octave: -1, Level: 0.7, DCW: 0.4, DCWEnv: env(1, 150, 0.2, 60),
			DCAEnv: env(1, 450, 0.5, 70)},
		DCOAmount: 3, DCOEnv: env(1, 25, 0, 20),
	}
	route(&cfg, 0, modmatrix.Velocity, "dcw1", 0.5)
	cfg.Level = 0.8201 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PluckBass is a plucked bass: a pulse over a saw an octave down, dying away without sustain under a closing lowpass
func PluckBass() Config {
	cfg := Config{
		Line1: Line{Waveform: Pulse, Level: 1, DCW: 0.6, DCWEnv: env(1, 260, 0, 100), DCAEnv: env(1, 700, 0, 100)},
		Line2: Line{Waveform: Saw, Octave: -1, Level: 0.6, DCW: 0.5, DCWEnv: env(1, 200, 0, 100),
			DCAEnv: env(1, 700, 0, 100)},
		Filter: filt(voicefilter.LowPass, 700, 0.35, 2.5, 0.4, 1, 220, 0, 100),
	}
	cfg.Level = 0.9583 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GrowlBass is a growling bass: the resonance wobbling fast, the wheel quickening it
func GrowlBass() Config {
	cfg := Config{
		Line1:     Line{Waveform: Reso3, Level: 1, DCW: 0.5, DCWEnv: env(2, 100, 1, 80), DCAEnv: env(3, 100, 1, 100)},
		Line2:     Line{Waveform: Saw, Octave: -1, Level: 0.5, DCW: 0.5, DCWEnv: env(2, 100, 1, 80), DCAEnv: env(3, 100, 1, 100)},
		SweepRate: 6, SweepDepth: 0.25,
	}
	route(&cfg, 0, modmatrix.Wheel, "sweepRate", 0.5)
	cfg.Level = 0.4350 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Leads

// ResoLead is a sweeping lead: two resonances a few cents apart, a slow wah and a vibrato that
// comes in late, sliding into every note from below
func ResoLead() Config {
	l := Line{Waveform: Reso2, Level: 1, DCW: 0.6, DCWEnv: env(30, 300, 0.7, 150), DCAEnv: env(10, 200, 0.9, 150)}
	l2 := l
	l2.Waveform, l2.Detune, l2.Level = Reso3, 8, 0.7
	cfg := Config{Line1: l, Line2: l2, SweepRate: 0.3, SweepDepth: 0.1, PitchSweep: -0.4, PitchTime: 0.08,
		VibratoRate: 5.5, VibratoDepth: 0.15, VibratoDelay: 0.4, Width: 0.3}
	cfg.Level = 0.5093 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SquareLead is a hollow square lead; the wheel deepens the vibrato
func SquareLead() Config {
	l := Line{Waveform: Square, Level: 1, DCW: 0.85, DCWEnv: env(10, 200, 0.8, 150), DCAEnv: env(8, 150, 0.9, 150)}
	l2 := l
	l2.Detune, l2.Level = -9, 0.8
	cfg := Config{Line1: l, Line2: l2, VibratoRate: 5.5, VibratoDepth: 0.2, VibratoDelay: 0.4, Width: 0.3}
	route(&cfg, 0, modmatrix.Wheel, "vibratoDepth", 0.5)
	cfg.Level = 0.3787 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PulseLead is a pulse whose width is swept by a slow LFO: the thin and the hollow in turn. The
// wheel deepens the sweep and a harder touch distorts more
func PulseLead() Config {
	l := Line{Waveform: Pulse, Level: 1, DCW: 0.5, DCWEnv: env(5, 100, 1, 100), DCAEnv: env(5, 100, 0.9, 150)}
	l2 := l
	l2.Detune = 10
	cfg := Config{Line1: l, Line2: l2, SweepRate: 0.6, SweepDepth: 0.35, Width: 0.6}
	route(&cfg, 0, modmatrix.Wheel, "sweepDepth", 0.4)
	route(&cfg, 1, modmatrix.Velocity, "dcw1", 0.3)
	cfg.Level = 0.5122 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FluteLead is a flute: nearly a sine, a chiff of pitch at the start and a vibrato that blooms
func FluteLead() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.12, DCWEnv: env(60, 100, 1, 100), DCAEnv: env(70, 100, 0.95, 150)},
		Line2: Line{Waveform: Square, Octave: 1, Level: 0.1, DCW: 0.1, DCWEnv: env(60, 100, 1, 100),
			DCAEnv: env(90, 100, 0.95, 150)},
		DCOAmount: -1, DCOEnv: env(1, 80, 0, 50),
		VibratoRate: 5, VibratoDepth: 0.08, VibratoDelay: 0.5, Width: 0.2,
	}
	cfg.Level = 0.5815 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WhistleLead is a whistle: a pure tone, a slide up into the note and a late wide vibrato
func WhistleLead() Config {
	cfg := Config{
		Line1:      Line{Waveform: Saw, Level: 1, DCW: 0.02, DCWEnv: env(10, 100, 1, 100), DCAEnv: env(80, 100, 1, 200)},
		PitchSweep: -1.5, PitchTime: 0.1, VibratoRate: 5.5, VibratoDepth: 0.3, VibratoDelay: 0.6,
	}
	cfg.Level = 0.5857 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Brass and strings

// Horns are slow French horns: a saw swelling in, a square close by, a soft lowpass
func Horns() Config {
	l := Line{Waveform: Saw, Level: 1, DCW: 0.6, DCWEnv: env(250, 400, 0.7, 250), KeyFollow: 0.4,
		DCAEnv: env(180, 300, 0.9, 250)}
	l2 := l
	l2.Waveform, l2.Detune, l2.Level = Square, 6, 0.7
	cfg := Config{Line1: l, Line2: l2, VibratoRate: 4.8, VibratoDepth: 0.05, VibratoDelay: 0.5, Width: 0.4,
		Filter: filt(voicefilter.LowPass, 2500, 0.1, 0, 0.5)}
	cfg.Level = 0.5148 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BrassStab is a brass stab: the distortion at its fullest at the strike, a burst of depth on top,
// a lowpass that the velocity opens
func BrassStab() Config {
	l := Line{Waveform: Saw, Level: 1, DCW: 0.9, DCWEnv: env(4, 150, 0.2, 100), KeyFollow: 0.3,
		DCAEnv: env(4, 250, 0.2, 100)}
	l2 := l
	l2.Detune = 10
	cfg := Config{Line1: l, Line2: l2, DepthAmount: 0.2, DepthEnv: env(2, 100, 0, 50), Width: 0.5,
		Filter: filt(voicefilter.LowPass, 3000, 0.15, 1, 0.5, 1, 200, 0.3, 100)}
	route(&cfg, 0, modmatrix.Velocity, "cutoff", 0.3)
	cfg.Level = 0.8791 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Ensemble is a string machine: two soft saws wide apart, slow, with a lowpass and a shimmer
func Ensemble() Config {
	l := Line{Waveform: Saw, Level: 1, DCW: 0.45, DCWEnv: env(400, 600, 0.8, 600), KeyFollow: 0.5,
		DCAEnv: env(450, 500, 0.9, 800)}
	l2 := l
	l2.Detune = -14
	cfg := Config{Line1: l, Line2: l2, VibratoRate: 5, VibratoDepth: 0.05, VibratoDelay: 0.6, Width: 1,
		Filter: filt(voicefilter.LowPass, 3500, 0.1, 0, 0.6)}
	cfg.Level = 0.4863 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Cello is a solo cello: a saw with a square an octave under, a bow that bites and a lowpass that
// follows the pitch up
func Cello() Config {
	cfg := Config{
		Line1: Line{Waveform: Saw, Level: 1, DCW: 0.5, DCWEnv: env(200, 500, 0.6, 300), KeyFollow: 0.5,
			DCAEnv: env(220, 300, 0.95, 300)},
		Line2: Line{Waveform: Square, Octave: -1, Level: 0.5, DCW: 0.3, DCWEnv: env(200, 500, 0.6, 300),
			DCAEnv: env(220, 300, 0.95, 300)},
		VibratoRate: 5, VibratoDepth: 0.15, VibratoDelay: 0.3,
		Filter: filt(voicefilter.LowPass, 1500, 0.1, 1, 0.8, 200, 400, 0.5, 300),
	}
	route(&cfg, 0, modmatrix.Key, "cutoff", 0.3)
	cfg.Level = 0.5544 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Keys

// Vibes are a vibraphone: a soft ring with a thin overtone two octaves up, whose brightness wobbles
// with the motor, harder notes ring brighter
func Vibes() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.08, DCWEnv: env(1, 100, 1, 100), DCAEnv: env(1, 2500, 0, 600)},
		Line2: Line{Waveform: Square, Octave: 2, Level: 0.2, DCW: 0.1, DCWEnv: env(1, 100, 1, 100),
			DCAEnv: env(1, 300, 0, 200)},
		SweepRate: 5.5, SweepDepth: 0.08, Width: 0.3,
	}
	route(&cfg, 0, modmatrix.Velocity, "dcw2", 0.5)
	cfg.Level = 0.9085 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Harpsichord is a harpsichord: a bright saw pulse, an octave of pulse over it, plucked and
// plain, every note a little different
func Harpsichord() Config {
	cfg := Config{
		Line1: Line{Waveform: SawPulse, Level: 1, DCW: 0.8, DCWEnv: env(1, 150, 0.5, 40), DCAEnv: env(1, 600, 0.1, 40)},
		Line2: Line{Waveform: Pulse, Octave: 1, Detune: 3, Level: 0.5, DCW: 0.7, DCWEnv: env(1, 120, 0.4, 40),
			DCAEnv: env(1, 400, 0.05, 40)},
		Width: 0.3,
	}
	route(&cfg, 0, modmatrix.Random, "dcw1", 0.2)
	cfg.Level = 1.1220 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Xylo is a xylophone: a hard, dry sine with a short overtone two octaves up
func Xylo() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.1, DCWEnv: env(1, 100, 1, 50), DCAEnv: env(1, 300, 0, 80)},
		Line2: Line{Waveform: Square, Octave: 2, Level: 0.4, DCW: 0.05, DCWEnv: env(1, 100, 1, 50),
			DCAEnv: env(1, 60, 0, 40)},
	}
	cfg.Level = 1.2867 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ToyPiano is a toy piano: a pulse ring modulated by a square a little over an octave up: tinny,
// out of tune and sweet
func ToyPiano() Config {
	cfg := Config{
		Line1: Line{Waveform: Pulse, Level: 1, DCW: 0.3, DCWEnv: env(1, 400, 0.1, 100), DCAEnv: env(1, 900, 0, 150)},
		Line2: Line{Waveform: Square, Octave: 1, Detune: 15, Level: 1, DCW: 0.1, DCWEnv: env(1, 300, 0, 100),
			DCAEnv: env(1, 900, 0, 150)},
		Ring: true, Width: 0.3,
	}
	cfg.Level = 0.9173 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PipeOrgan is a pipe organ: a square with an octave of pulse over it, a breath of attack and a
// soft lowpass
func PipeOrgan() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.4, DCWEnv: env(30, 100, 1, 100), DCAEnv: env(40, 100, 1, 300)},
		Line2: Line{Waveform: Pulse, Octave: 1, Level: 0.6, DCW: 0.3, DCWEnv: env(30, 100, 1, 100),
			DCAEnv: env(40, 100, 1, 300)},
		Width: 0.4, Filter: filt(voicefilter.LowPass, 5000, 0.05, 0, 0.5),
	}
	cfg.Level = 0.4319 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Bells

// Glockenspiel is a glockenspiel: a high ring modulated bell that rings and rings
func Glockenspiel() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Octave: 1, Level: 1, DCW: 0.05, DCWEnv: env(1, 100, 1, 100), DCAEnv: env(1, 1800, 0, 1200)},
		Line2: Line{Waveform: Square, Octave: 2, Detune: 700, Level: 1, DCW: 0.05, DCWEnv: env(1, 100, 1, 100),
			DCAEnv: env(1, 1500, 0, 1200)},
		Ring: true, Width: 0.3,
	}
	cfg.Level = 0.7622 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Chime is a long chime: a resonance modulated by a sine at an odd interval, ringing for seconds
func Chime() Config {
	cfg := Config{
		Line1: Line{Waveform: Reso1, Level: 1, DCW: 0.3, DCWEnv: env(1, 2500, 0, 1500), DCAEnv: env(1, 5000, 0, 3000)},
		Line2: Line{Waveform: Square, Octave: 1, Detune: 1100, Level: 1, DCW: 0.05, DCWEnv: env(1, 100, 1, 100),
			DCAEnv: env(1, 4500, 0, 3000)},
		Ring: true, Width: 0.5,
	}
	cfg.Level = 0.8081 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MusicBox is a music box: a pulse an octave up with a thin square two octaves up, plinking
func MusicBox() Config {
	cfg := Config{
		Line1: Line{Waveform: Pulse, Octave: 1, Level: 1, DCW: 0.2, DCWEnv: env(1, 400, 0.1, 100), DCAEnv: env(1, 900, 0, 300)},
		Line2: Line{Waveform: Square, Octave: 2, Level: 0.3, DCW: 0.1, DCWEnv: env(1, 100, 1, 100),
			DCAEnv: env(1, 500, 0, 200)},
		Width: 0.4,
	}
	cfg.Level = 1.0148 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Pads and voices

// DarkPad is a dark pad: a saw pair under a nearly closed lowpass that a slow LFO opens and closes
func DarkPad() Config {
	l := Line{Waveform: Saw, Level: 1, DCW: 0.3, DCWEnv: env(1500, 1000, 0.8, 1500), DCAEnv: env(1500, 800, 0.9, 2000)}
	l2 := l
	l2.Detune = 11
	cfg := Config{Line1: l, Line2: l2, Width: 1, SweepRate: 0.1, SweepDepth: 0.05, LFO2Rate: 0.08,
		Filter: filt(voicefilter.LowPass, 600, 0.2, 0.5, 0.3, 1500, 800, 0.9, 1500)}
	route(&cfg, 0, modmatrix.LFO2, "cutoff", 0.2)
	cfg.Level = 0.5199 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WarmPad is a warm pad: a saw pulse against a square a little flat, a soft lowpass
func WarmPad() Config {
	l := Line{Waveform: SawPulse, Level: 1, DCW: 0.4, DCWEnv: env(900, 1000, 0.8, 1500), DCAEnv: env(900, 800, 0.9, 1500)}
	l2 := l
	l2.Waveform, l2.Detune = Square, -9
	cfg := Config{Line1: l, Line2: l2, Width: 0.9, VibratoRate: 4.5, VibratoDepth: 0.04, VibratoDelay: 0.6,
		Filter: filt(voicefilter.LowPass, 1800, 0.1, 0, 0.5)}
	cfg.Level = 0.4452 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AirPad is an airy pad: resonances an octave apart under a highpass, the detune breathing with a
// slow LFO
func AirPad() Config {
	l := Line{Waveform: Reso3, Level: 1, DCW: 0.2, DCWEnv: env(1500, 1000, 0.9, 2000), DCAEnv: env(1800, 800, 0.9, 2200)}
	l2 := l
	l2.Waveform, l2.Octave, l2.Level = Reso2, 1, 0.4
	cfg := Config{Line1: l, Line2: l2, Width: 1, SweepRate: 0.2, SweepDepth: 0.1, LFO2Rate: 0.3,
		Filter: filt(voicefilter.HighPass, 400, 0.1, 0, 0.5)}
	route(&cfg, 0, modmatrix.LFO2, "detune", 0.4)
	cfg.Level = 0.8181 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AhChoir is a choir on "ah": the resonance sits at a formant, the two lines apart, the vowel
// swaying a little
func AhChoir() Config {
	l := Line{Waveform: Reso2, Level: 1, DCW: 0.17, DCWEnv: env(400, 800, 0.9, 800), DCAEnv: env(400, 600, 0.9, 1000)}
	l2 := l
	l2.Waveform, l2.DCW, l2.Detune = Reso3, 0.24, 10
	cfg := Config{Line1: l, Line2: l2, Width: 0.8, VibratoRate: 5, VibratoDepth: 0.1, VibratoDelay: 0.4, LFO2Rate: 0.25}
	route(&cfg, 0, modmatrix.LFO2, "dcw1", 0.2)
	route(&cfg, 1, modmatrix.LFO2, "dcw2", -0.2)
	cfg.Level = 0.5505 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// OhChoir is a rounder choir on "oh": the resonance low, under a lowpass
func OhChoir() Config {
	l := Line{Waveform: Reso1, Level: 1, DCW: 0.1, DCWEnv: env(500, 800, 0.9, 800), DCAEnv: env(500, 600, 0.9, 1000)}
	l2 := l
	l2.Waveform, l2.DCW, l2.Detune = Reso2, 0.15, 7
	cfg := Config{Line1: l, Line2: l2, Width: 0.8, VibratoRate: 4.8, VibratoDepth: 0.1, VibratoDelay: 0.5,
		Filter: filt(voicefilter.LowPass, 2000, 0.1, 0, 0.5)}
	cfg.Level = 0.7056 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Effects

// LaserDown is a laser: the pitch falls two octaves in a flash through the DCO envelope, a square
// that stays bright
func LaserDown() Config {
	cfg := Config{
		Line1:     Line{Waveform: Square, Level: 1, DCW: 0.7, DCWEnv: env(1, 250, 0, 50), DCAEnv: env(1, 350, 0, 50)},
		DCOAmount: 24, DCOEnv: env(1, 200, 0, 50),
	}
	cfg.Level = 1.2055 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Siren is an alarm: an LFO swings the pitch up and down
func Siren() Config {
	cfg := Config{
		Line1:     Line{Waveform: Saw, Level: 1, DCW: 0.5, DCWEnv: env(5, 100, 1, 100), DCAEnv: env(30, 100, 1, 200)},
		SweepRate: 0.8, Filter: filt(voicefilter.LowPass, 3000, 0.2, 0, 0.3),
	}
	route(&cfg, 0, modmatrix.LFO1, "pitch", 0.7)
	cfg.Level = 0.5854 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SonarPing is a sonar: a pure ping, a little flat at the start, two lines beating
func SonarPing() Config {
	cfg := Config{
		Line1: Line{Waveform: Square, Level: 1, DCW: 0.05, DCWEnv: env(1, 100, 1, 100), DCAEnv: env(1, 2500, 0, 1000)},
		Line2: Line{Waveform: Square, Detune: 3, Level: 0.6, DCW: 0.05, DCWEnv: env(1, 100, 1, 100),
			DCAEnv: env(1, 2500, 0, 1000)},
		DCOAmount: -0.5, DCOEnv: env(1, 100, 0, 50), Width: 0.6,
	}
	cfg.Level = 0.8036 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RiseSweep is a riser: the pitch climbs two octaves while a resonance opens, over four seconds
func RiseSweep() Config {
	cfg := Config{
		Line1:     Line{Waveform: Reso2, Level: 1, DCW: 0.5, DCWEnv: env(4000, 100, 1, 300), DCAEnv: env(2000, 100, 1, 500)},
		DCOAmount: 24, DCOEnv: env(4000, 100, 1, 300), DepthAmount: 0.3, DepthEnv: env(4000, 100, 1, 300),
	}
	cfg.Level = 0.9403 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Percussion

// DCOKick is a kick: a sine whose pitch falls two octaves in a few milliseconds, with a click on top
func DCOKick() Config {
	cfg := Config{
		Line1:     Line{Waveform: Saw, Level: 1, DCW: 0.02, DCWEnv: env(1, 100, 1, 100), DCAEnv: env(1, 350, 0, 60)},
		Line2:     Line{Waveform: Pulse, Octave: 2, Level: 0.2, DCW: 1, DCWEnv: env(1, 15, 0, 10), DCAEnv: env(1, 15, 0, 10)},
		DCOAmount: 24, DCOEnv: env(1, 45, 0, 40),
	}
	cfg.Level = 1.4687 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DCOTom is a tom: the pitch falls an octave, slower, over a skin a fifth up
func DCOTom() Config {
	cfg := Config{
		Line1: Line{Waveform: Saw, Level: 1, DCW: 0.03, DCWEnv: env(1, 100, 1, 100), DCAEnv: env(1, 500, 0, 80)},
		Line2: Line{Waveform: Saw, Detune: 700, Level: 0.3, DCW: 0.1, DCWEnv: env(1, 100, 1, 100),
			DCAEnv: env(1, 250, 0, 60)},
		DCOAmount: 12, DCOEnv: env(1, 120, 0, 50), Width: 0.3,
	}
	cfg.Level = 1.2503 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RingHat is a hi-hat: two pulses far up and out of tune, ring modulated into a metallic hiss
// that a highpass thins, every hit a little different
func RingHat() Config {
	cfg := Config{
		Line1: Line{Waveform: Pulse, Octave: 3, Level: 1, DCW: 1, DCWEnv: env(1, 100, 1, 100), DCAEnv: env(1, 50, 0, 30)},
		Line2: Line{Waveform: Square, Octave: 3, Detune: 560, Level: 1, DCW: 1, DCWEnv: env(1, 100, 1, 100),
			DCAEnv: env(1, 50, 0, 30)},
		Ring: true, Width: 0.5, Filter: filt(voicefilter.HighPass, 6000, 0.1, 0, 0),
	}
	route(&cfg, 0, modmatrix.Random, "detune", 0.3)
	cfg.Level = 1.2450 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Clack is a woodblock: a short pulse, the pitch dropping a flash, a resonance two octaves up
func Clack() Config {
	cfg := Config{
		Line1:     Line{Waveform: Pulse, Level: 1, DCW: 1, DCWEnv: env(1, 15, 0, 10), DCAEnv: env(1, 40, 0, 20)},
		Line2:     Line{Waveform: Reso1, Octave: 2, Level: 0.5, DCW: 0.4, DCWEnv: env(1, 20, 0, 10), DCAEnv: env(1, 30, 0, 20)},
		DCOAmount: 7, DCOEnv: env(1, 15, 0, 10),
	}
	cfg.Level = 1.2190 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
