package subtractive

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

// adsrEnv is an envelope: attack to full, decay to the sustain level, release (milliseconds)
func adsrEnv(attack, decay, sustain, release float64) adsr.Setting {
	return *adsr.NewSetting(1.0, attack, sustain, decay, 1.0, release)
}

// MoogBassPreset is the Minimoog bass: two saws an octave down and a square, a 24 dB ladder snapping shut
func MoogBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Octave = -1
	cfg.Osc2Detune = 0.06
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.45
	cfg.SubLevel = 0.2
	cfg.Drift = 4
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 160
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 3.2
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = adsrEnv(1, 260, 0.1, 140)
	cfg.AmpEnv = adsrEnv(1, 300, 0.7, 120)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.15}
	cfg.Level = 1.2120 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PulseBassPreset is a thin pulse bass whose pulse width follows the velocity: a hard note is hollower and nasal
func PulseBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.Osc2Level = 0
	cfg.PulseWidth = 0.5
	cfg.Osc1Level = 0.6
	cfg.SubLevel = 0.25
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass12
	cfg.Cutoff = 500
	cfg.Resonance = 0.25
	cfg.FilterEnvAmount = 2
	cfg.KeyTrack = 0.4
	cfg.FilterEnv = adsrEnv(1, 180, 0.2, 100)
	cfg.AmpEnv = adsrEnv(1, 250, 0.75, 90)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "pulseWidth", Amount: -0.9}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.2}
	cfg.Level = 0.842 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GrowlBassPreset is a snarling bass: a saw phase modulated by its own octave, driven, the wheel pushes the FM
func GrowlBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.02
	cfg.Osc1Level = 0.2
	cfg.Osc2Level = 0.5
	cfg.FMAmount = 1.5
	cfg.Drive = 3
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.FilterDrive = 2
	cfg.Cutoff = 300
	cfg.Resonance = 0.45
	cfg.FilterEnvAmount = 1.5
	cfg.KeyTrack = 0.3
	cfg.Filter2Mode = Filter2Band
	cfg.Filter2Cutoff = 900
	cfg.Filter2Resonance = 0.6
	cfg.Filter2Parallel = true
	cfg.FilterEnv = adsrEnv(1, 400, 0.4, 120)
	cfg.AmpEnv = adsrEnv(1, 300, 0.9, 100)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 3, KeySync: true}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "fmAmount", Amount: 0.5}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "filter2Cutoff", Amount: 0.3}
	cfg.Level = 1.051 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Bass808Preset is the long sub of a 808: a sine with a short pitch dive at the start and a slow decay
func Bass808Preset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Level = 0
	cfg.Osc1Level = 0.8
	cfg.FilterEnvToPitch = 12
	cfg.Drive = 1.8
	cfg.Cutoff = 1200
	cfg.Resonance = 0
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0
	cfg.FilterEnv = adsrEnv(1, 45, 0, 30)
	cfg.AmpEnv = adsrEnv(1, 1400, 0, 250)
	cfg.Level = 0.4922 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AcidSquarePreset is the square-ish acid line: a pulse through a screaming ladder, glided, accents from the velocity
func AcidSquarePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.5
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0
	cfg.Drive = 2.5
	cfg.Glide = 60
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.FilterDrive = 1.8
	cfg.Cutoff = 300
	cfg.Resonance = 0.85
	cfg.FilterEnvAmount = 3
	cfg.KeyTrack = 0.6
	cfg.FilterEnv = adsrEnv(1, 150, 0, 60)
	cfg.AmpEnv = adsrEnv(1, 60, 0.9, 50)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.35}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "resonance", Amount: 0.1}
	cfg.Level = 0.6367 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DubWobblePreset is a dubstep wobble: a low saw pair, a resonant ladder swept by a triangle LFO, the wheel speeds it up
func DubWobblePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Octave = -1
	cfg.Osc2Detune = 0.03
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.4
	cfg.Drive = 2
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 180
	cfg.Resonance = 0.55
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.3
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 60
	cfg.FilterEnv = adsrEnv(1, 100, 1, 100)
	cfg.AmpEnv = adsrEnv(2, 100, 0.95, 90)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 2.2, KeySync: true, Cutoff: 3.2}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "lfo1Rate", Amount: 0.35}
	cfg.Level = 1.0333 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PluckBassPreset is a short, woody bass pluck, the velocity opens the filter
func PluckBassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Detune = 0
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 140
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 3.2
	cfg.KeyTrack = 0.7
	cfg.FilterEnv = adsrEnv(1, 110, 0, 60)
	cfg.AmpEnv = adsrEnv(1, 220, 0, 80)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.3}
	cfg.Level = 1.559 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WideReesePreset is a reese of five detuned unison saws, drifting, the wheel spreads them further
func WideReesePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.2
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.Unison = 5
	cfg.UnisonDetune = 35
	cfg.Drift = 8
	cfg.SubLevel = 0.35
	cfg.Drive = 1.6
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 260
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 0.8
	cfg.KeyTrack = 0.3
	cfg.FilterEnv = adsrEnv(30, 600, 0.6, 250)
	cfg.AmpEnv = adsrEnv(5, 200, 0.95, 200)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 0.25, Cutoff: 0.8}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "unisonDetune", Amount: 0.7}
	cfg.Level = 0.6709 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MonoLeadPreset is a classic mono lead: two drifting saws through a ladder, glided, a late vibrato
func MonoLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Detune = 0.08
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.3
	cfg.Drift = 5
	cfg.Glide = 70
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 1400
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 1.4
	cfg.KeyTrack = 0.9
	cfg.FilterEnv = adsrEnv(6, 400, 0.5, 250)
	cfg.AmpEnv = adsrEnv(6, 200, 0.85, 220)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.2, Delay: 450, KeySync: true, Pitch: 0.22}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.2}
	cfg.Level = 0.6784 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SquareLeadPreset is a hollow square lead with a narrow pulse an octave up and a soft vibrato
func SquareLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.3
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.04
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.15
	cfg.Cutoff = 3200
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 0.8
	cfg.KeyTrack = 0.9
	cfg.FilterEnv = adsrEnv(3, 250, 0.6, 200)
	cfg.AmpEnv = adsrEnv(3, 150, 0.9, 180)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.8, Delay: 300, KeySync: true, Pitch: 0.18}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "pulseWidth", Amount: 0.5}
	cfg.Level = 0.3681 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PWMLeadPreset is a pulse lead with the width swept by a slow LFO and two unison voices, a Pro-One style PWM
func PWMLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.Osc2Detune = 0.1
	cfg.PulseWidth = 0.5
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.Unison = 2
	cfg.UnisonDetune = 12
	cfg.Cutoff = 2400
	cfg.Resonance = 0.25
	cfg.FilterEnvAmount = 1
	cfg.KeyTrack = 0.8
	cfg.FilterEnv = adsrEnv(4, 300, 0.5, 200)
	cfg.AmpEnv = adsrEnv(4, 150, 0.9, 200)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 0.9, PulseWidth: 0.38}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "unisonDetune", Amount: 0.5}
	cfg.Level = 0.4084 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// TranceLeadPreset is a big seven voice unison saw lead, bright and wide, a fifth and an octave in the second oscillator
func TranceLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0
	cfg.Osc2Octave = 1
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.25
	cfg.Unison = 7
	cfg.UnisonDetune = 30
	cfg.Drive = 1.2
	cfg.Cutoff = 3800
	cfg.Resonance = 0.25
	cfg.FilterEnvAmount = 1.2
	cfg.KeyTrack = 0.8
	cfg.FilterEnv = adsrEnv(4, 350, 0.55, 250)
	cfg.AmpEnv = adsrEnv(4, 150, 0.9, 250)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.15}
	cfg.Level = 0.509 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PortaLeadPreset is a legato lead that slides a long way between notes, the wheel adds a vibrato rate and bite
func PortaLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Detune = 0.05
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.Glide = 220
	cfg.Drift = 3
	cfg.Cutoff = 1800
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 0.6
	cfg.KeyTrack = 0.9
	cfg.FilterEnv = adsrEnv(30, 400, 0.7, 300)
	cfg.AmpEnv = adsrEnv(30, 200, 0.9, 300)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 4.8, Delay: 600, KeySync: true, Pitch: 0.3}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "lfo1Rate", Amount: 0.2}
	cfg.Level = 0.577 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ScreamLeadPreset is a hard synced lead: the velocity sweeps the synced oscillator, the harder the higher the scream
func ScreamLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Sync = true
	cfg.Osc2Detune = 3
	cfg.Osc1Level = 0.1
	cfg.Osc2Level = 0.55
	cfg.Drive = 2.2
	cfg.Cutoff = 3000
	cfg.Resonance = 0.35
	cfg.FilterEnvAmount = 1.2
	cfg.KeyTrack = 0.8
	cfg.FilterEnv = adsrEnv(2, 500, 0.5, 250)
	cfg.AmpEnv = adsrEnv(2, 200, 0.9, 200)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.5, Delay: 250, KeySync: true, Pitch: 0.2}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "osc2Detune", Amount: 0.6}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.FilterEnv, Dest: "osc2Detune", Amount: 0.5}
	cfg.Level = 0.3953 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SineLeadPreset is a soft, pure lead: a sine with a touch of triangle, a slow vibrato and a slight glide
func SineLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.02
	cfg.Osc1Level = 0.6
	cfg.Osc2Level = 0.12
	cfg.Glide = 45
	cfg.Cutoff = 5000
	cfg.Resonance = 0
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.6
	cfg.AmpEnv = adsrEnv(25, 100, 0.95, 260)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.0, Delay: 500, KeySync: true, Pitch: 0.12}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "osc2Level", Amount: 0.2}
	cfg.Level = 0.4008 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BrassLeadPreset is a brassy solo line: saws through a 12 dB ladder that blares open with the velocity
func BrassLeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.05
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.45
	cfg.Drive = 1.5
	cfg.Glide = 25
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass12
	cfg.Cutoff = 450
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 2.2
	cfg.KeyTrack = 0.7
	cfg.FilterEnv = adsrEnv(70, 250, 0.7, 200)
	cfg.AmpEnv = adsrEnv(40, 100, 0.9, 180)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.5, Delay: 600, KeySync: true, Pitch: 0.12}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.4}
	cfg.Level = 0.5565 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WarmPadPreset is a soft analog pad: triangle and saw, drifting, a 12 dB ladder opening slowly
func WarmPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Triangle
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.3
	cfg.Drift = 7
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass12
	cfg.Cutoff = 700
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 1.2
	cfg.KeyTrack = 0.6
	cfg.FilterEnv = adsrEnv(1400, 1500, 0.6, 1800)
	cfg.AmpEnv = adsrEnv(900, 800, 0.9, 1800)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Random, Dest: "osc2Detune", Amount: 0.01}
	cfg.Level = 0.8349 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GlassPadPreset is a cold, airy pad of sines and an octave of triangle with the lows cut away
func GlassPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Triangle
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.08
	cfg.Osc1Level = 0.35
	cfg.Osc2Level = 0.3
	cfg.RingModLevel = 0.1
	cfg.NoiseLevel = 0.01
	cfg.Cutoff = 6500
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.8
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 500
	cfg.Filter2KeyTrack = 0.5
	cfg.AmpEnv = adsrEnv(1200, 800, 0.9, 2500)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 0.3, Cutoff: 0.6}
	cfg.Level = 2.013 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// StringMachinePreset is the string machine: three unison saws, an airy low pass and a slow swell
func StringMachinePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.12
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.Unison = 3
	cfg.UnisonDetune = 18
	cfg.Drift = 3
	cfg.Cutoff = 2800
	cfg.Resonance = 0.05
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.9
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 200
	cfg.AmpEnv = adsrEnv(450, 600, 0.95, 900)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.6, Delay: 800, Pitch: 0.06, Amp: 0.1}
	cfg.Level = 0.8529 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// EvolvingPadPreset is a pad that never stops moving: a slow LFO on the cutoff and the width, another on the resonance
func EvolvingPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.14
	cfg.PulseWidth = 0.5
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.35
	cfg.Unison = 2
	cfg.UnisonDetune = 14
	cfg.Drift = 5
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 600
	cfg.Resonance = 0.35
	cfg.FilterEnvAmount = 1.5
	cfg.KeyTrack = 0.6
	cfg.FilterEnv = adsrEnv(2500, 3000, 0.5, 2500)
	cfg.AmpEnv = adsrEnv(1800, 800, 0.9, 3000)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 0.11, Cutoff: 1.4, PulseWidth: 0.3}
	cfg.LFO2 = LFOConfig{Waveform: LFOSine, Rate: 0.07}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "resonance", Amount: 0.3}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "unisonDetune", Amount: 0.4}
	cfg.Level = 0.7324 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AhChoirPreset is a human "aah": unison saws through the formant filter, the vowel moves a little with a slow LFO
func AhChoirPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.45
	cfg.Unison = 3
	cfg.UnisonDetune = 22
	cfg.NoiseLevel = 0.02
	cfg.NoiseType = PinkNoise
	cfg.Drift = 4
	cfg.FilterType = FilterFormant
	cfg.Vowel = 0
	cfg.Resonance = 0.5
	cfg.FormantShift = 1.05
	cfg.AmpEnv = adsrEnv(700, 800, 0.9, 1500)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 0.2, Cutoff: 0.3}
	cfg.LFO2 = LFOConfig{Waveform: LFOSine, Rate: 5.0, Delay: 700, Pitch: 0.07}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "vowel", Amount: 0.1}
	cfg.Level = 3.419 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// VowelPadPreset gets its vowel colour from a second filter: a low pass and a resonant band beside it, an "oh" pad
func VowelPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.4
	cfg.Osc2Detune = 0.09
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.35
	cfg.Unison = 3
	cfg.UnisonDetune = 15
	cfg.Cutoff = 700
	cfg.Resonance = 0.25
	cfg.KeyTrack = 0.3
	cfg.Filter2Mode = Filter2Band
	cfg.Filter2Cutoff = 1150
	cfg.Filter2Resonance = 0.75
	cfg.Filter2Parallel = true
	cfg.Filter2KeyTrack = 0.1
	cfg.AmpEnv = adsrEnv(800, 800, 0.9, 1800)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 0.18}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "filter2Cutoff", Amount: 0.12}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "filter2Cutoff", Amount: 0.3}
	cfg.Level = 1.1 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DarkPadPreset is a brooding pad: saws an octave down under a closed 24 dB ladder
func DarkPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Octave = -1
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.35
	cfg.Osc2Level = 0.45
	cfg.Drift = 6
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 220
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 2.2
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = adsrEnv(2500, 2500, 0.5, 3000)
	cfg.AmpEnv = adsrEnv(1500, 800, 0.9, 3000)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 0.15, Cutoff: 0.5}
	cfg.Level = 0.9441 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BrightPadPreset is an open pad of saw and pulse in unison with a high pass keeping it light
func BrightPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.35
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.3
	cfg.Unison = 3
	cfg.UnisonDetune = 20
	cfg.Cutoff = 6000
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 0.5
	cfg.KeyTrack = 0.9
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 450
	cfg.Filter2KeyTrack = 0.5
	cfg.FilterEnv = adsrEnv(500, 900, 0.8, 1000)
	cfg.AmpEnv = adsrEnv(350, 500, 0.9, 1200)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "unisonDetune", Amount: 0.4}
	cfg.Level = 0.9721 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HugePadPreset is a wall of seven unison voices spread wide, slow and enormous
func HugePadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Octave = -1
	cfg.Osc2Detune = 0.05
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.3
	cfg.Unison = 7
	cfg.UnisonDetune = 55
	cfg.Drift = 5
	cfg.Cutoff = 1500
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 1.5
	cfg.KeyTrack = 0.6
	cfg.FilterEnv = adsrEnv(1800, 2000, 0.5, 2500)
	cfg.AmpEnv = adsrEnv(1200, 800, 0.9, 2500)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Random, Dest: "unisonDetune", Amount: 0.3}
	cfg.Level = 0.731 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ClavinetPreset is a funky clavinet: a narrow pulse, a high pass taking away the body, a snap that follows the velocity
func ClavinetPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.22
	cfg.Osc2Detune = 0.02
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.25
	cfg.Cutoff = 700
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 3
	cfg.KeyTrack = 0.9
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 350
	cfg.Filter2KeyTrack = 0.7
	cfg.FilterEnv = adsrEnv(1, 90, 0.05, 80)
	cfg.AmpEnv = adsrEnv(1, 350, 0.05, 80)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.45}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "pulseWidth", Amount: -0.3}
	cfg.Level = 0.7862 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HarpsichordPreset is a plucked harpsichord: two bright saws an octave apart, a click and no sustain
func HarpsichordPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.3
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.03
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.3
	cfg.NoiseLevel = 0.04
	cfg.Cutoff = 3500
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 1.6
	cfg.KeyTrack = 1
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 300
	cfg.Filter2KeyTrack = 0.8
	cfg.FilterEnv = adsrEnv(1, 150, 0, 100)
	cfg.AmpEnv = adsrEnv(1, 900, 0, 70)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.15}
	cfg.Level = 1.075 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WurliPreset is a Wurlitzer style reed piano: a triangle with a sine an octave up and a bark that follows the velocity
func WurliPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Triangle
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0.01
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.3
	cfg.FMAmount = 0.3
	cfg.Drive = 1.8
	cfg.Cutoff = 1800
	cfg.Resonance = 0.05
	cfg.FilterEnvAmount = 1.2
	cfg.KeyTrack = 0.8
	cfg.FilterEnv = adsrEnv(1, 400, 0.3, 250)
	cfg.AmpEnv = adsrEnv(2, 1200, 0.25, 350)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.2, Amp: 0.18}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "fmAmount", Amount: 0.5}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "drive", Amount: 0.2}
	cfg.Level = 0.7385 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MarimbaPreset is a wooden mallet: a sine with a fast dying high partial two octaves and a bit up
func MarimbaPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Octave = 2
	cfg.Osc2Detune = 0.25
	cfg.Osc1Level = 0.6
	cfg.Osc2Level = 0.25
	cfg.NoiseLevel = 0.03
	cfg.Cutoff = 2200
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 1.5
	cfg.KeyTrack = 1
	cfg.FilterEnv = adsrEnv(1, 70, 0, 50)
	cfg.AmpEnv = adsrEnv(1, 450, 0, 90)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "osc2Level", Amount: 0.25}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "noiseLevel", Amount: 0.05}
	cfg.Level = 0.8389 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PizzicatoPreset is a plucked string section note: a short saw with the filter closing quickly
func PizzicatoPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.Unison = 3
	cfg.UnisonDetune = 10
	cfg.Cutoff = 700
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 2.2
	cfg.KeyTrack = 0.9
	cfg.FilterEnv = adsrEnv(1, 120, 0, 60)
	cfg.AmpEnv = adsrEnv(1, 260, 0, 70)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.25}
	cfg.Level = 1.254 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// TubularBellPreset is a church bell: a ring modulated pair at an inharmonic ratio, FM on top, ringing out
func TubularBellPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Detune = 17.5
	cfg.Osc1Level = 0.3
	cfg.Osc2Level = 0.1
	cfg.RingModLevel = 0.4
	cfg.FMAmount = 1
	cfg.Cutoff = 7000
	cfg.Resonance = 0
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = adsrEnv(1, 3000, 0, 1500)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.AmpEnv, Dest: "fmAmount", Amount: 0.3}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "osc2Level", Amount: 0.15}
	cfg.Level = 0.724 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AnalogSeqPreset is a step sequencer bass note: a short resonant saw with a random cutoff nudge on every step
func AnalogSeqPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.05
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.3
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 500
	cfg.Resonance = 0.55
	cfg.FilterEnvAmount = 2
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = adsrEnv(1, 130, 0, 60)
	cfg.AmpEnv = adsrEnv(1, 160, 0.1, 60)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Random, Dest: "cutoff", Amount: 0.12}
	cfg.Level = 1.7222 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// JunoBrassPreset is the Juno brass: a saw and a pulse in unison with a sub, a quick swell and a high pass
func JunoBrassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Rectangle
	cfg.PulseWidth = 0.5
	cfg.Osc2Detune = 0.07
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.35
	cfg.SubLevel = 0.2
	cfg.Unison = 2
	cfg.UnisonDetune = 10
	cfg.Cutoff = 700
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 2.4
	cfg.KeyTrack = 0.6
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 150
	cfg.FilterEnv = adsrEnv(60, 400, 0.6, 250)
	cfg.AmpEnv = adsrEnv(30, 200, 0.9, 250)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.25}
	cfg.Level = 0.6109 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// JupiterBrassPreset is the Jupiter-8 brass: two driven saws in a 12 dB ladder, a bite of resonance at the swell
func JupiterBrassPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.12
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.5
	cfg.Drive = 1.8
	cfg.Drift = 3
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass12
	cfg.Cutoff = 380
	cfg.Resonance = 0.35
	cfg.FilterEnvAmount = 3
	cfg.KeyTrack = 0.5
	cfg.FilterEnv = adsrEnv(120, 450, 0.5, 300)
	cfg.AmpEnv = adsrEnv(60, 200, 0.85, 300)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.FilterEnv, Dest: "resonance", Amount: -0.2}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.2}
	cfg.Level = 0.5465 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// StringSectionPreset is a full string section: five unison saws drifting with a breath of vibrato, a slow bow
func StringSectionPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.07
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.45
	cfg.Unison = 5
	cfg.UnisonDetune = 24
	cfg.Drift = 6
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass12
	cfg.Cutoff = 1800
	cfg.Resonance = 0.05
	cfg.FilterEnvAmount = 0.8
	cfg.KeyTrack = 0.9
	cfg.FilterEnv = adsrEnv(350, 500, 0.8, 600)
	cfg.AmpEnv = adsrEnv(280, 400, 0.95, 700)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 5.3, Delay: 900, KeySync: true, Pitch: 0.1}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.25}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "unisonDetune", Amount: 0.2}
	cfg.Level = 0.5974 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PercOrganPreset is a percussive organ: drawbar sounds at 8', 4' and 16' with a fast key click that fades
func PercOrganPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Octave = 1
	cfg.Osc2Detune = 0
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.0
	cfg.SubLevel = 0.15
	cfg.Cutoff = 6000
	cfg.Resonance = 0
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = adsrEnv(2, 80, 0.9, 45)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 6.7, Amp: 0.12}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.AmpEnv, Dest: "osc2Level", Amount: 0.7}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "osc2Level", Amount: 0.3}
	cfg.Level = 0.3965 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RockOrganPreset is an overdriven rock organ, the wheel pushes it further into the drive
func RockOrganPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Detune = 7.02
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.2
	cfg.SubLevel = 0.2
	cfg.Drive = 3
	cfg.Cutoff = 3500
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.6
	cfg.AmpEnv = adsrEnv(2, 80, 1, 50)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 6.1, Pitch: 0.04}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "drive", Amount: 0.4}
	cfg.Level = 0.2755 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DroneFifthPreset is a held drone of an open fifth in unison saws, slowly breathing
func DroneFifthPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 7.02
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.3
	cfg.Unison = 3
	cfg.UnisonDetune = 14
	cfg.Drift = 9
	cfg.SubLevel = 0.2
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 500
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 0
	cfg.KeyTrack = 0.6
	cfg.AmpEnv = adsrEnv(2500, 500, 1, 3500)
	cfg.LFO1 = LFOConfig{Waveform: LFOSine, Rate: 0.08, Cutoff: 1}
	cfg.LFO2 = LFOConfig{Waveform: LFOTriangle, Rate: 0.13}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "unisonDetune", Amount: 0.3}
	cfg.Level = 1.092 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DarkDronePreset is a low rumbling drone: a sub, a dark saw and a band of rustling noise
func DarkDronePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sine
	cfg.Osc2Octave = -1
	cfg.Osc2Detune = 0.05
	cfg.Osc1Level = 0.2
	cfg.Osc2Level = 0.5
	cfg.NoiseLevel = 0.25
	cfg.NoiseType = PinkNoise
	cfg.Drift = 12
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 200
	cfg.Resonance = 0.5
	cfg.KeyTrack = 0.3
	cfg.AmpEnv = adsrEnv(3000, 500, 1, 4000)
	cfg.LFO1 = LFOConfig{Waveform: LFOSampleHold, Rate: 0.9, Cutoff: 0.6}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "noiseLevel", Amount: 0.2}
	cfg.Level = 1.3845 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GalePreset is a howling wind: noise through a resonant band that a random sample and hold keeps moving
func GalePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Level = 0
	cfg.Osc2Level = 0
	cfg.NoiseLevel = 0.7
	cfg.FilterMode = BandPass
	cfg.Cutoff = 700
	cfg.Resonance = 0.85
	cfg.KeyTrack = 0.5
	cfg.FilterEnvAmount = 0
	cfg.Filter2Mode = Filter2Low
	cfg.Filter2Cutoff = 2500
	cfg.AmpEnv = adsrEnv(1500, 500, 1, 2500)
	cfg.LFO1 = LFOConfig{Waveform: LFOSampleHold, Rate: 0.7, Cutoff: 1.2}
	cfg.LFO2 = LFOConfig{Waveform: LFOSine, Rate: 0.2, Cutoff: 0.5}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "level", Amount: 0.3}
	cfg.Level = 0.8275 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// LaserPreset is a sci-fi laser: a resonant saw falling several octaves in a flash
func LaserPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Detune = 0.2
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.25
	cfg.FilterEnvToPitch = 48
	cfg.Cutoff = 800
	cfg.Resonance = 0.6
	cfg.FilterEnvAmount = 3
	cfg.KeyTrack = 0.3
	cfg.FilterEnv = adsrEnv(1, 180, 0, 60)
	cfg.AmpEnv = adsrEnv(1, 330, 0, 90)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.FilterEnv, Dest: "fmAmount", Amount: 0.4}
	cfg.Level = 0.9495 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SirenFxPreset is a rescue siren: a wide triangle LFO rising and falling in pitch, the wheel speeds it
func SirenFxPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc2Level = 0
	cfg.Osc1Level = 0.4
	cfg.Cutoff = 2500
	cfg.Resonance = 0.3
	cfg.KeyTrack = 0.5
	cfg.AmpEnv = adsrEnv(30, 100, 1, 150)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 1.2, KeySync: true, Pitch: 7}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "lfo1Rate", Amount: 0.3}
	cfg.Level = 0.4073 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// NoiseTexturePreset is a tonal noise bed: noise through a swept notch and a high pass, shifting from note to note
func NoiseTexturePreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Level = 0
	cfg.Osc2Level = 0
	cfg.NoiseLevel = 0.5
	cfg.NoiseType = PinkNoise
	cfg.FilterMode = Notch
	cfg.Cutoff = 1500
	cfg.Resonance = 0.7
	cfg.KeyTrack = 0.7
	cfg.FilterEnvAmount = 0
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 800
	cfg.AmpEnv = adsrEnv(600, 400, 1, 1500)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 0.35, Cutoff: 2}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Random, Dest: "cutoff", Amount: 0.25}
	cfg.Level = 1.062 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Kick808Preset is a deep, round kick: a sine with a fast pitch fall and a long boomy tail, play around MIDI 33
func Kick808Preset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Level = 0
	cfg.Osc1Level = 0.9
	cfg.FilterEnvToPitch = 24
	cfg.Drive = 2.2
	cfg.Cutoff = 800
	cfg.Resonance = 0
	cfg.KeyTrack = 0
	cfg.FilterEnvAmount = 0
	cfg.FilterEnv = adsrEnv(1, 18, 0, 20)
	cfg.AmpEnv = adsrEnv(1, 500, 0, 150)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "drive", Amount: 0.15}
	cfg.Level = 0.5788 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ClapPreset is a hand clap: noise through a band pass around 1.2 kHz, with a quick double burst of the envelope
func ClapPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Level = 0
	cfg.Osc2Level = 0
	cfg.NoiseLevel = 0.8
	cfg.FilterMode = BandPass
	cfg.Cutoff = 1200
	cfg.Resonance = 0.45
	cfg.KeyTrack = 0
	cfg.FilterEnvAmount = 0.8
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 600
	cfg.FilterEnv = adsrEnv(1, 100, 0, 40)
	cfg.AmpEnv = adsrEnv(1, 160, 0, 60)
	cfg.LFO1 = LFOConfig{Waveform: LFOSquare, Rate: 38, Amp: 0.5}
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.1}
	cfg.Level = 1.522 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RimShotPreset is a rim shot: two short, high pulses with a click of noise through a band pass
func RimShotPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Detune = 8.3
	cfg.Osc1Level = 0.3
	cfg.Osc2Level = 0.3
	cfg.NoiseLevel = 0.15
	cfg.FilterMode = BandPass
	cfg.Cutoff = 1700
	cfg.Resonance = 0.55
	cfg.KeyTrack = 0.2
	cfg.FilterEnvAmount = 1
	cfg.FilterEnv = adsrEnv(1, 20, 0, 15)
	cfg.AmpEnv = adsrEnv(1, 45, 0, 25)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "noiseLevel", Amount: 0.2}
	cfg.Level = 1.502 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// CowbellPreset is a cowbell: two squares a tritone apart through a band pass, a short knock
func CowbellPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Square
	cfg.Osc2Waveform = blosc.Square
	cfg.Osc2Detune = 6.8
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.4
	cfg.FilterMode = BandPass
	cfg.Cutoff = 1000
	cfg.Resonance = 0.35
	cfg.KeyTrack = 0.6
	cfg.FilterEnvAmount = 0.5
	cfg.FilterEnv = adsrEnv(1, 60, 0.2, 40)
	cfg.AmpEnv = adsrEnv(1, 280, 0, 60)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.AmpEnv, Dest: "osc2Level", Amount: 0.3}
	cfg.Level = 0.9898 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// JunoPadPreset is the Juno pad: a pulse swept by the LFO with a sub and saw, a high pass and the chorus-like unison
func JunoPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.PulseWidth = 0.5
	cfg.Osc2Detune = 0.1
	cfg.Osc1Level = 0.4
	cfg.Osc2Level = 0.25
	cfg.SubLevel = 0.2
	cfg.Unison = 2
	cfg.UnisonDetune = 14
	cfg.Drift = 3
	cfg.Cutoff = 1500
	cfg.Resonance = 0.1
	cfg.FilterEnvAmount = 0.5
	cfg.KeyTrack = 0.7
	cfg.Filter2Mode = Filter2High
	cfg.Filter2Cutoff = 180
	cfg.FilterEnv = adsrEnv(400, 800, 0.7, 900)
	cfg.AmpEnv = adsrEnv(250, 500, 0.9, 900)
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 0.6, PulseWidth: 0.3}
	cfg.Level = 0.7418 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// OBStabPreset is an Oberheim style stab: two saws, a short open ladder swell and a little drift
func OBStabPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.14
	cfg.Osc1Level = 0.5
	cfg.Osc2Level = 0.5
	cfg.Unison = 3
	cfg.UnisonDetune = 16
	cfg.Drift = 4
	cfg.Drive = 1.4
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass12
	cfg.Cutoff = 600
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 2.8
	cfg.KeyTrack = 0.7
	cfg.FilterEnv = adsrEnv(1, 220, 0.1, 150)
	cfg.AmpEnv = adsrEnv(1, 300, 0.25, 140)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Velocity, Dest: "cutoff", Amount: 0.3}
	cfg.Level = 1.187 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// JupiterPadPreset is the Jupiter-8 pad: bright detuned saws, a resonant filter that opens and a second band for air
func JupiterPadPreset() Config {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Sawtooth
	cfg.Osc2Waveform = blosc.Sawtooth
	cfg.Osc2Detune = 0.11
	cfg.Osc1Level = 0.45
	cfg.Osc2Level = 0.45
	cfg.Unison = 2
	cfg.UnisonDetune = 10
	cfg.Drift = 4
	cfg.FilterType = FilterLadder
	cfg.LadderMode = ladder.LowPass24
	cfg.Cutoff = 1100
	cfg.Resonance = 0.3
	cfg.FilterEnvAmount = 1.8
	cfg.KeyTrack = 0.7
	cfg.Filter2Mode = Filter2Band
	cfg.Filter2Cutoff = 3500
	cfg.Filter2Resonance = 0.3
	cfg.Filter2Parallel = true
	cfg.FilterEnv = adsrEnv(900, 1400, 0.55, 1400)
	cfg.AmpEnv = adsrEnv(500, 800, 0.9, 1500)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Random, Dest: "unisonDetune", Amount: 0.2}
	cfg.Level = 1.665 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
