package wavetable

import (
	"math"
	"sync"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// More presets, grouped by family. The matrix is used the way a sound designer would: the filter
// envelope (also free to move other settings, it only needs its shape) bends the warp or opens the
// second oscillator at every note, the velocity brightens, the wheel moves the warp or the filter,
// slow LFOs make the pads evolve and a random value per note keeps arpeggios alive.
//
// A few presets bring a table of their own (an additive spectrum: drawbars, pipes, a reed, a choir)
// and ignore the table they are given; they say so

// env is an envelope: attack, decay, sustain level, release (milliseconds)
func env(attack, decay, sustain, release float64) adsr.Setting {
	return *adsr.NewSetting(1.0, attack, sustain, decay, 1.0, release)
}

// route fills slot i of the matrix
func route(cfg *Config, i int, src modmatrix.Source, dest string, amount float64) {
	cfg.Mods[i] = modmatrix.Slot{Src: src, Dest: dest, Amount: amount}
}

// harmonicMorph builds a table that morphs from the harmonics of one frame to those of another
func harmonicMorph(from, to []float64, frames int) *Table {
	return FromHarmonics(frames, 2048, func(position float64, amps []float64) {
		for k := range max(len(from), len(to)) {
			var a, b float64
			if k < len(from) {
				a = from[k]
			}
			if k < len(to) {
				b = to[k]
			}
			amps[k] = a + (b-a)*position
		}
	})
}

// the tables of the presets that bring their own, built once
var (
	tablesOnce                                                  sync.Once
	drawbarTable, pipesTable, reedTable, fluteTable, choirTable *Table
)

func ownTables() {
	tablesOnce.Do(func() {
		// drawbars 16' 5 1/3' 8' 4' 2 2/3' 2' 1 3/5' 1 1/3' 1': the harmonics 1 2 3 4 5 6 8 10 12 16 are
		// added, from a soft registration to the full organ
		bars := func(levels ...float64) []float64 {
			h := make([]float64, 17)
			for i, k := range []int{0, 1, 2, 3, 4, 5, 7, 9, 11, 15} {
				if i < len(levels) {
					h[k] = levels[i]
				}
			}
			return h
		}
		drawbarTable = harmonicMorph(bars(0.3, 0, 1, 0.3, 0, 0, 0, 0, 0), bars(1, 0.6, 1, 0.9, 0.6, 0.7, 0.5, 0.4, 0.5, 0.3), 16)
		principal := make([]float64, 24)
		flute := []float64{1, 0.25, 0.08, 0.02}
		for k := range principal {
			principal[k] = math.Pow(0.82, float64(k)) * (1 + 0.5*float64((k+1)%2))
		}
		pipesTable = harmonicMorph(flute, principal, 16)
		reed := make([]float64, 30)
		for k := 0; k < len(reed); k += 2 { // odd harmonics only: a clarinet
			reed[k] = 1 / float64(k+1) * math.Pow(0.93, float64(k))
		}
		reedTable = harmonicMorph([]float64{1, 0, 0.12, 0, 0.05}, reed, 12)
		fluteTable = harmonicMorph([]float64{1, 0.04, 0.02}, []float64{1, 0.35, 0.12, 0.06, 0.03, 0.02}, 8)
		// a choir: the harmonics under the formants of "ah" (800, 1150 Hz), "oh" (500, 800) and "oo" (300, 800),
		// for a voice at about 200 Hz
		formant := func(f1, f2 float64) []float64 {
			h := make([]float64, 40)
			for k := range h {
				f := 200 * float64(k+1)
				h[k] = (math.Exp(-math.Pow((f-f1)/180, 2)) + 0.6*math.Exp(-math.Pow((f-f2)/260, 2)) + 0.03) / float64(k+1)
			}
			return h
		}
		ah, oh, oo := formant(800, 1150), formant(500, 800), formant(300, 800)
		choirTable = FromHarmonics(12, 2048, func(position float64, amps []float64) {
			a, b, t := ah, oh, position*2
			if position > 0.5 {
				a, b, t = oh, oo, position*2-1
			}
			for k := range a {
				amps[k] = a[k] + (b[k]-a[k])*t
			}
		})
	})
}

// Basses

// SubBassPreset is a sine sub for the bottom of a mix: a barely brighter wave over a pure sine an
// octave down, a soft lowpass and a short glide
func SubBassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.05
	cfg.SubLevel = 0.7
	cfg.Filter = LowPass
	cfg.Cutoff = 220
	cfg.Resonance = 0.1
	cfg.KeyTrack = 0.6
	cfg.Glide = 15
	cfg.AmpEnv = env(3, 150, 0.9, 120)
	cfg.Level = 0.5258 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WobbleBassPreset is the dubstep wobble: a slow LFO sweeps the cutoff of a resonant lowpass over
// a pair of detuned oscillators, the wheel speeds the wobble up
func WobbleBassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.55
	cfg.Unison = 2
	cfg.Detune = 8
	cfg.Spread = 0.3
	cfg.SubLevel = 0.4
	cfg.Filter = LowPass
	cfg.Cutoff = 450
	cfg.Resonance = 0.45
	cfg.KeyTrack = 0.3
	cfg.LFO1Rate = 2.5
	route(&cfg, 0, modmatrix.LFO1, "cutoff", 0.35)
	route(&cfg, 1, modmatrix.Wheel, "lfo1Rate", 0.5)
	cfg.Glide = 20
	cfg.AmpEnv = env(3, 100, 1, 120)
	cfg.Level = 0.4435 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FMBassPreset is a clanking FM bass: a sine whose phase is modulated by a sine at its own pitch,
// the filter envelope opening the modulation at every note, harder notes opening it more
func FMBassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.25
	cfg.WarpMode = WarpFMSine
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.Warp = 0.1
	cfg.SubLevel = 0.3
	cfg.Filter = LowPass
	cfg.Cutoff = 1500
	cfg.Resonance = 0.1
	cfg.FilterEnv = env(1, 300, 0.15, 150)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.5)
	route(&cfg, 1, modmatrix.Velocity, "warp", 0.2)
	cfg.AmpEnv = env(2, 400, 0.8, 100)
	cfg.Level = 0.7655 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SyncBassPreset is a hard sync bass: the frame is played faster and faster at every note, the
// tearing sound of a sync sweep, closed down by a lowpass
func SyncBassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.4
	cfg.WarpMode = WarpSync
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.SubLevel = 0.4
	cfg.Filter = LowPass
	cfg.Cutoff = 900
	cfg.Resonance = 0.3
	cfg.FilterEnv = env(1, 250, 0.3, 120)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.55)
	route(&cfg, 1, modmatrix.Velocity, "cutoff", 0.2)
	cfg.AmpEnv = env(2, 200, 0.85, 100)
	cfg.Level = 0.4229 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HooverBassPreset is the rave hoover: seven oscillators far apart, the phase bent, and a pitch
// that scoops down into the note
func HooverBassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.6
	cfg.Unison = 7
	cfg.Detune = 70
	cfg.Spread = 1
	cfg.WarpMode = WarpBendUp
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Warp = 0.5
	cfg.Filter = LowPass
	cfg.Cutoff = 2200
	cfg.Resonance = 0.3
	cfg.SubLevel = 0.3
	cfg.LFO2Rate = 0.3
	cfg.FilterEnv = env(1, 120, 0, 100)
	route(&cfg, 0, modmatrix.FilterEnv, "pitch", 0.25)
	route(&cfg, 1, modmatrix.LFO2, "warp", 0.1)
	cfg.Glide = 90
	cfg.AmpEnv = env(8, 100, 0.9, 150)
	cfg.Level = 0.4706 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Leads

// SyncLeadPreset is a screaming sync lead: the wheel pushes the sync ratio up, harder notes sync
// further
func SyncLeadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.55
	cfg.Unison = 2
	cfg.Detune = 10
	cfg.WarpMode = WarpSync
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Warp = 0.15
	cfg.Filter = LowPass
	cfg.Cutoff = 4000
	cfg.Resonance = 0.2
	route(&cfg, 0, modmatrix.Wheel, "warp", 0.5)
	route(&cfg, 1, modmatrix.Velocity, "warp", 0.2)
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.1
	cfg.Glide = 40
	cfg.AmpEnv = env(8, 100, 0.85, 200)
	cfg.Level = 0.4709 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MirrorLeadPreset is a hollow, folding lead: the phase is played forward then backward, and a slow
// LFO moves how far
func MirrorLeadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.35
	cfg.Unison = 3
	cfg.Detune = 14
	cfg.Spread = 0.7
	cfg.WarpMode = WarpMirror
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Warp = 0.5
	cfg.Filter = LowPass
	cfg.Cutoff = 5000
	cfg.LFO1Rate = 0.8
	route(&cfg, 0, modmatrix.LFO1, "warp", 0.25)
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.08
	cfg.AmpEnv = env(10, 100, 0.85, 220)
	cfg.Level = 0.6377 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FMLeadPreset is a glassy FM lead: a sine an octave up modulates the phase, a burst of
// modulation at the start of the note settles to a singing tone
func FMLeadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.2
	cfg.WarpMode = WarpFMB
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.Warp = 0.2
	cfg.OscBPosition = 0
	cfg.OscBOctave = 1
	cfg.Unison = 2
	cfg.Detune = 6
	cfg.FilterEnv = env(1, 400, 0.3, 200)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.4)
	route(&cfg, 1, modmatrix.Velocity, "warp", 0.2)
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.12
	cfg.Glide = 30
	cfg.AmpEnv = env(5, 100, 0.9, 200)
	cfg.Level = 0.4080 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SuperLeadPreset is a supersaw lead: seven oscillators spread wide and bright, for trance
func SuperLeadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 1
	cfg.OscBLevel = 0.4
	cfg.OscBOctave = -1
	cfg.OscBPosition = 0.8
	cfg.Unison = 7
	cfg.Detune = 45
	cfg.Spread = 1
	cfg.Filter = LowPass
	cfg.Cutoff = 12000
	cfg.Resonance = 0.15
	cfg.Glide = 30
	cfg.AmpEnv = env(6, 100, 0.9, 250)
	cfg.Level = 0.4228 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ThereminLeadPreset is a theremin: a nearly pure tone, a late deep vibrato and a long glide
func ThereminLeadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.08
	cfg.OscBLevel = 0.15
	cfg.OscBOctave = 1
	cfg.OscBPosition = 0.05
	cfg.OscBDetune = 5
	cfg.Filter = LowPass
	cfg.Cutoff = 3000
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.5
	cfg.Glide = 180
	cfg.AmpEnv = env(120, 100, 1, 250)
	cfg.Level = 0.4169 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Pads

// GlassPadPreset is a pad of glass: a thin second oscillator two octaves up in a shimmering
// stack, whose timbre a slow LFO moves
func GlassPadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.25
	cfg.Unison = 3
	cfg.Detune = 8
	cfg.Spread = 0.9
	cfg.OscBLevel = 0.45
	cfg.OscBOctave = 2
	cfg.OscBPosition = 0.1
	cfg.OscBUnison = 3
	cfg.OscBSpread = 12
	cfg.Filter = LowPass
	cfg.Cutoff = 7000
	cfg.LFO1Rate = 0.2
	route(&cfg, 0, modmatrix.LFO1, "oscBPosition", 0.3)
	cfg.AmpEnv = env(1200, 100, 1, 1800)
	cfg.Level = 0.4330 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// EvolvingPadPreset never stands still: one slow LFO travels the table, another slow one bends
// the phase, the wheel opens the filter further
func EvolvingPadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.2
	cfg.Unison = 5
	cfg.Detune = 16
	cfg.Spread = 0.9
	cfg.WarpMode = WarpMirror
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Warp = 0.2
	cfg.Filter = LowPass
	cfg.Cutoff = 6000
	cfg.Resonance = 0.2
	cfg.LFO1Rate = 0.2
	cfg.LFO2Rate = 0.13
	route(&cfg, 0, modmatrix.LFO1, "position", 0.4)
	route(&cfg, 1, modmatrix.LFO2, "warp", 0.3)
	route(&cfg, 2, modmatrix.Wheel, "cutoff", 0.3)
	cfg.AmpEnv = env(1500, 100, 1, 2200)
	cfg.Level = 0.7146 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// VoxPadPreset is a vocal-ish pad: a resonant bandpass that a slow LFO sweeps over the vowels of
// a wide stack
func VoxPadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.6
	cfg.Unison = 4
	cfg.Detune = 12
	cfg.Spread = 0.8
	cfg.Filter = BandPass
	cfg.Cutoff = 900
	cfg.Resonance = 0.5
	cfg.KeyTrack = 0.5
	cfg.LFO1Rate = 0.25
	route(&cfg, 0, modmatrix.LFO1, "cutoff", 0.12)
	cfg.AmpEnv = env(900, 100, 1, 1500)
	cfg.Level = 0.8493 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// EnsemblePadPreset is a string machine: two stacks an octave apart, one detuned wide, a soft
// lowpass and a slow bow
func EnsemblePadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.Unison = 6
	cfg.Detune = 22
	cfg.Spread = 0.9
	cfg.OscBLevel = 0.5
	cfg.OscBOctave = -1
	cfg.OscBPosition = 0.4
	cfg.OscBUnison = 3
	cfg.OscBSpread = 10
	cfg.Filter = LowPass
	cfg.Cutoff = 2400
	cfg.KeyTrack = 0.6
	cfg.VibratoRate = 5.2
	cfg.VibratoDepth = 0.05
	cfg.AmpEnv = env(500, 100, 1, 900)
	cfg.Level = 0.4186 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DarkPadPreset is a brooding pad: low, with a sub, the filter nearly shut and slowly breathing
func DarkPadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.15
	cfg.Unison = 5
	cfg.Detune = 10
	cfg.Spread = 0.8
	cfg.SubLevel = 0.4
	cfg.Filter = LowPass
	cfg.Cutoff = 700
	cfg.Resonance = 0.2
	cfg.KeyTrack = 0.3
	cfg.LFO1Rate = 0.06
	route(&cfg, 0, modmatrix.LFO1, "cutoff", 0.15)
	route(&cfg, 1, modmatrix.Velocity, "cutoff", 0.15)
	cfg.AmpEnv = env(2000, 100, 1, 2500)
	cfg.Level = 0.5076 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ShimmerPadPreset floats above the note: a second stack a twelfth up, fading in and out with a
// slow LFO, under a highpass that takes the weight away
func ShimmerPadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.1
	cfg.Unison = 3
	cfg.Detune = 10
	cfg.Spread = 0.8
	cfg.OscBLevel = 0.35
	cfg.OscBOctave = 1
	cfg.OscBSemi = 7
	cfg.OscBPosition = 0.15
	cfg.OscBUnison = 4
	cfg.OscBSpread = 25
	cfg.Filter = HighPass
	cfg.Cutoff = 300
	cfg.KeyTrack = 0.5
	cfg.LFO1Rate = 0.15
	route(&cfg, 0, modmatrix.LFO1, "oscBLevel", 0.25)
	cfg.AmpEnv = env(1600, 100, 1, 2400)
	cfg.Level = 0.4905 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ChoirPadPreset is a choir: its own table of vowels ("ah" to "oh" to "oo") sung by four voices,
// vowels moving slowly. It ignores the table it is given
func ChoirPadPreset(*Table) Config {
	ownTables()
	cfg := preset(choirTable)
	cfg.Position = 0.1
	cfg.Unison = 4
	cfg.Detune = 10
	cfg.Spread = 0.8
	cfg.Filter = LowPass
	cfg.Cutoff = 3500
	cfg.KeyTrack = 0.5
	cfg.LFO1Rate = 0.12
	route(&cfg, 0, modmatrix.LFO1, "position", 0.25)
	route(&cfg, 1, modmatrix.Wheel, "position", 0.2)
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.1
	cfg.AmpEnv = env(700, 100, 1, 1200)
	cfg.Level = 0.5930 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Plucks and keys

// SyncPluckPreset is a pluck whose sync sweeps down at every note, from tearing to round
func SyncPluckPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.WarpMode = WarpSync
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Unison = 2
	cfg.Detune = 8
	cfg.Spread = 0.5
	cfg.FilterEnv = env(1, 200, 0, 150)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.6)
	route(&cfg, 1, modmatrix.Velocity, "warp", 0.15)
	cfg.AmpEnv = env(1, 500, 0, 200)
	cfg.Level = 0.7859 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FMPluckPreset is a plucked FM string: the phase modulation bursts and dies with the note
func FMPluckPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.1
	cfg.WarpMode = WarpFMSine
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.Unison = 2
	cfg.Detune = 6
	cfg.Spread = 0.5
	cfg.FilterEnv = env(1, 150, 0, 120)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.45)
	cfg.AmpEnv = env(1, 700, 0, 250)
	cfg.Level = 0.7583 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HarpPluckPreset is a harp: a bright start that falls up and down the table, a lowpass closing
// on the pluck and a soft octave overtone
func HarpPluckPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.3
	cfg.PositionEnvAmount = 0.35
	cfg.PositionEnv = env(1, 250, 0, 200)
	cfg.OscBLevel = 0.25
	cfg.OscBOctave = 1
	cfg.OscBPosition = 0.2
	cfg.Filter = LowPass
	cfg.Cutoff = 1800
	cfg.KeyTrack = 0.8
	cfg.FilterEnvAmount = 1.5
	cfg.FilterEnv = env(1, 300, 0, 200)
	cfg.AmpEnv = env(1, 1400, 0, 300)
	cfg.Level = 0.6496 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MarimbaPreset is a marimba: a sine with a short ring two octaves up that dies in a blink
func MarimbaPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.1
	cfg.OscBLevel = 0.1
	cfg.OscBOctave = 2
	cfg.OscBPosition = 0
	cfg.FilterEnv = env(1, 70, 0, 50)
	route(&cfg, 0, modmatrix.FilterEnv, "oscBLevel", 0.6)
	cfg.Filter = LowPass
	cfg.Cutoff = 3000
	cfg.AmpEnv = env(1, 450, 0, 120)
	cfg.Level = 0.7528 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// TineKeysPreset is an electric piano: a near sine whose phase an overtone three octaves up bends
// at the strike, the bark harder with the velocity, ringing out long
func TineKeysPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.05
	cfg.WarpMode = WarpFMB
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.OscBPosition = 0
	cfg.OscBOctave = 3
	cfg.Unison = 2
	cfg.Detune = 5
	cfg.Spread = 0.5
	cfg.FilterEnv = env(1, 400, 0, 200)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.3)
	route(&cfg, 1, modmatrix.Velocity, "warp", 0.3)
	cfg.AmpEnv = env(1, 2200, 0, 300)
	cfg.Level = 0.5602 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ClavKeysPreset is a clavinet: a bright, thin wave with a snap, no sustain to speak of
func ClavKeysPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.8
	cfg.OscBLevel = 0.3
	cfg.OscBOctave = 1
	cfg.OscBPosition = 0.7
	cfg.Filter = BandPass
	cfg.Cutoff = 1800
	cfg.Resonance = 0.35
	cfg.FilterEnvAmount = 1.5
	cfg.FilterEnv = env(1, 120, 0, 80)
	cfg.AmpEnv = env(1, 250, 0.1, 80)
	cfg.Level = 0.9688 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Bells

// MusicBoxPreset is a music box: a sine with a thin tine three octaves and a fifth up, ringing
// short and high
func MusicBoxPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0
	cfg.OscBLevel = 0.1
	cfg.OscBOctave = 3
	cfg.OscBSemi = 7
	cfg.FilterEnv = env(1, 300, 0, 150)
	route(&cfg, 0, modmatrix.FilterEnv, "oscBLevel", 0.4)
	cfg.AmpEnv = env(1, 1500, 0, 400)
	cfg.Level = 0.6670 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FMBellPreset is a DX style bell: a sine modulated by a sine at 3.5 times its pitch, the
// modulation dying away so the bell turns pure as it rings
func FMBellPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.02
	cfg.WarpMode = WarpFMB
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.OscBPosition = 0
	cfg.OscBOctave = 1
	cfg.OscBSemi = 9.68
	cfg.FilterEnv = env(1, 1500, 0, 500)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.4)
	cfg.AmpEnv = env(1, 3500, 0, 1500)
	cfg.Level = 0.6328 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// TubularBellPreset is a tubular bell: a stack of sines with a strike partial at about 2.8 times
// the pitch, a long ring
func TubularBellPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.02
	cfg.Unison = 2
	cfg.Detune = 4
	cfg.OscBLevel = 0.3
	cfg.OscBOctave = 1
	cfg.OscBSemi = 6.2
	cfg.OscBPosition = 0.05
	cfg.OscBUnison = 3
	cfg.OscBSpread = 6
	cfg.FilterEnv = env(1, 1200, 0, 600)
	route(&cfg, 0, modmatrix.FilterEnv, "oscBLevel", 0.5)
	cfg.AmpEnv = env(1, 5000, 0, 2500)
	cfg.Level = 0.5022 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Brass and strings

// BrassSectionPreset is a big brass section: two stacks a few cents apart, the filter swelling
// into each note and the velocity digging in
func BrassSectionPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.55
	cfg.Unison = 5
	cfg.Detune = 16
	cfg.Spread = 0.7
	cfg.OscBLevel = 0.5
	cfg.OscBPosition = 0.45
	cfg.OscBDetune = 7
	cfg.Filter = LowPass
	cfg.Cutoff = 500
	cfg.Resonance = 0.15
	cfg.KeyTrack = 0.8
	cfg.FilterEnvAmount = 3.5
	cfg.FilterEnv = env(120, 500, 0.5, 250)
	route(&cfg, 0, modmatrix.Velocity, "cutoff", 0.2)
	cfg.AmpEnv = env(60, 100, 0.9, 250)
	cfg.Level = 0.3706 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SynthBrassPreset is the Jupiter brass: a bent saw with a squealing filter swell, solo
func SynthBrassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.85
	cfg.Unison = 2
	cfg.Detune = 9
	cfg.WarpMode = WarpBendUp
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Warp = 0.2
	cfg.Filter = LowPass
	cfg.Cutoff = 450
	cfg.Resonance = 0.3
	cfg.KeyTrack = 0.7
	cfg.FilterEnvAmount = 3.5
	cfg.FilterEnv = env(80, 350, 0.45, 200)
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.06
	cfg.AmpEnv = env(40, 100, 0.9, 200)
	cfg.Level = 0.4683 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// CelloPreset is a solo cello: a warm saw with its octave under, a slow bow and a heavy vibrato
func CelloPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.6
	cfg.Unison = 3
	cfg.Detune = 7
	cfg.Spread = 0.4
	cfg.OscBLevel = 0.5
	cfg.OscBOctave = -1
	cfg.OscBPosition = 0.5
	cfg.Filter = LowPass
	cfg.Cutoff = 1200
	cfg.KeyTrack = 0.7
	cfg.FilterEnvAmount = 0.8
	cfg.FilterEnv = env(250, 400, 0.6, 300)
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.12
	cfg.AmpEnv = env(250, 100, 0.95, 300)
	cfg.Level = 0.3823 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ViolinPreset is a solo violin: bright, a quick vibrato following the wheel in the filter and a
// bow that bites
func ViolinPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.55
	cfg.Unison = 2
	cfg.Detune = 6
	cfg.Spread = 0.4
	cfg.Filter = LowPass
	cfg.Cutoff = 3500
	cfg.KeyTrack = 1
	cfg.VibratoRate = 5.8
	cfg.VibratoDepth = 0.2
	cfg.FilterEnvAmount = 1
	cfg.FilterEnv = env(1, 150, 0.5, 200)
	cfg.AmpEnv = env(90, 100, 0.95, 250)
	cfg.Level = 0.4414 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Organs and winds

// DrawbarOrganPreset is a drawbar organ with a key click: its own table, from a soft registration
// to the full organ, the click a quick flick towards the full registration, and the slight
// wobble of a rotating speaker. It ignores the table it is given
func DrawbarOrganPreset(*Table) Config {
	ownTables()
	cfg := preset(drawbarTable)
	cfg.Position = 0.5
	cfg.PositionEnvAmount = 0.4
	cfg.PositionEnv = env(1, 60, 0, 50)
	cfg.SubLevel = 0.2
	cfg.VibratoRate = 6.2
	cfg.VibratoDepth = 0.05
	cfg.AmpEnv = env(4, 100, 1, 50)
	cfg.Level = 0.6183 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ChurchOrganPreset is a pipe organ: its own table from a flute to a full principal chorus, a
// 16' pedal under, a breath of attack and a long release. It ignores the table it is given
func ChurchOrganPreset(*Table) Config {
	ownTables()
	cfg := preset(pipesTable)
	cfg.Position = 0.7
	cfg.Unison = 2
	cfg.Detune = 5
	cfg.Spread = 0.6
	cfg.OscBLevel = 0.5
	cfg.OscBOctave = -1
	cfg.OscBPosition = 0.2
	cfg.Filter = LowPass
	cfg.Cutoff = 6000
	cfg.AmpEnv = env(60, 100, 1, 400)
	cfg.Level = 0.6491 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PercOrganPreset is a rock organ: the drawbars out and a decaying third harmonic percussion that
// bites at every note. It ignores the table it is given
func PercOrganPreset(*Table) Config {
	ownTables()
	cfg := preset(drawbarTable)
	cfg.Position = 0.85
	cfg.OscBLevel = 0.05
	cfg.OscBOctave = 1
	cfg.OscBSemi = 7
	cfg.OscBPosition = 0.2
	cfg.FilterEnv = env(1, 250, 0, 100)
	route(&cfg, 0, modmatrix.FilterEnv, "oscBLevel", 0.5)
	cfg.AmpEnv = env(2, 100, 1, 40)
	cfg.Level = 0.6465 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ClarinetPreset is a clarinet: its own table of odd harmonics, darkest in the low register and
// brighter as it is blown harder. It ignores the table it is given
func ClarinetPreset(*Table) Config {
	ownTables()
	cfg := preset(reedTable)
	cfg.Position = 0.3
	route(&cfg, 0, modmatrix.Velocity, "position", 0.4)
	cfg.Filter = LowPass
	cfg.Cutoff = 2500
	cfg.KeyTrack = 0.8
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.05
	cfg.AmpEnv = env(50, 100, 0.9, 120)
	cfg.Level = 0.3176 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// FlutePreset is a breathy flute: its own, nearly pure table, a puff of the second harmonic and
// the phase wobbled by a sine for air. It ignores the table it is given
func FlutePreset(*Table) Config {
	ownTables()
	cfg := preset(fluteTable)
	cfg.Position = 0.2
	cfg.PositionEnvAmount = 0.3
	cfg.PositionEnv = env(1, 150, 0, 100)
	cfg.WarpMode = WarpFMSine
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.Warp = 0.04
	cfg.VibratoRate = 5.3
	cfg.VibratoDepth = 0.08
	cfg.AmpEnv = env(80, 100, 0.95, 150)
	cfg.Level = 0.3590 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Arps

// ArpPluckPreset is for arpeggios: short, the filter and the fold of the phase snapping closed, and
// a new random brightness at every note
func ArpPluckPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.Unison = 2
	cfg.Detune = 10
	cfg.WarpMode = WarpMirror
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Filter = LowPass
	cfg.Cutoff = 800
	cfg.Resonance = 0.25
	cfg.FilterEnvAmount = 3
	cfg.FilterEnv = env(1, 140, 0, 80)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.3)
	route(&cfg, 1, modmatrix.Random, "position", 0.25)
	cfg.AmpEnv = env(1, 220, 0, 80)
	cfg.Level = 1.1347 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ArpGlassPreset is a glassy arpeggio voice: a bright FM ping with a different amount of bite at
// every note
func ArpGlassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.15
	cfg.WarpMode = WarpFMSine
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.Warp = 0.05
	cfg.FilterEnv = env(1, 200, 0, 100)
	route(&cfg, 0, modmatrix.FilterEnv, "warp", 0.25)
	route(&cfg, 1, modmatrix.Random, "warp", 0.15)
	cfg.AmpEnv = env(1, 350, 0, 150)
	cfg.Level = 1.1072 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Drones

// DarkDronePreset is a cavern: seven slow oscillators, a sub, a second stack an octave down, the
// filter breathing almost shut
func DarkDronePreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.2
	cfg.Unison = 7
	cfg.Detune = 14
	cfg.Spread = 1
	cfg.SubLevel = 0.6
	cfg.OscBLevel = 0.4
	cfg.OscBOctave = -1
	cfg.OscBPosition = 0.1
	cfg.Filter = LowPass
	cfg.Cutoff = 400
	cfg.Resonance = 0.2
	cfg.KeyTrack = 0.2
	cfg.LFO1Rate = 0.04
	route(&cfg, 0, modmatrix.LFO1, "cutoff", 0.1)
	cfg.AmpEnv = env(4000, 100, 1, 5000)
	cfg.Level = 0.7604 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// TensionDronePreset is uneasy: a second oscillator a tritone away, the phase folding slowly
func TensionDronePreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.4
	cfg.Unison = 3
	cfg.Detune = 20
	cfg.Spread = 0.9
	cfg.OscBLevel = 0.7
	cfg.OscBSemi = 6
	cfg.OscBPosition = 0.5
	cfg.WarpMode = WarpMirror
	cfg.Oversample = 1 // the corners and steps of the warp fold back at a high note
	cfg.Warp = 0.2
	cfg.Filter = LowPass
	cfg.Cutoff = 1800
	cfg.LFO2Rate = 0.07
	route(&cfg, 0, modmatrix.LFO2, "warp", 0.3)
	cfg.AmpEnv = env(3000, 100, 1, 4000)
	cfg.Level = 0.9425 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Effects

// SirenFXPreset is an alarm: an LFO swings the pitch over an octave and a half, the wheel quickens it
func SirenFXPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.Filter = LowPass
	cfg.Cutoff = 3000
	cfg.Resonance = 0.2
	cfg.LFO1Rate = 0.8
	route(&cfg, 0, modmatrix.LFO1, "pitch", 0.6)
	route(&cfg, 1, modmatrix.Wheel, "lfo1Rate", 0.4)
	cfg.AmpEnv = env(30, 100, 1, 150)
	cfg.Level = 0.4260 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// LaserFXPreset is a laser: the pitch falls two octaves in an instant, the phase modulation with it
func LaserFXPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.6
	cfg.WarpMode = WarpFMSine
	cfg.Oversample = 2 // the FM of the warp folds back at a high note
	cfg.Warp = 0.1
	cfg.FilterEnv = env(1, 150, 0, 100)
	route(&cfg, 0, modmatrix.FilterEnv, "pitch", 1)
	route(&cfg, 1, modmatrix.FilterEnv, "warp", 0.4)
	cfg.AmpEnv = env(1, 400, 0, 100)
	cfg.Level = 1.0802 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// WindFXPreset is wind: heavy phase modulation turns seven oscillators to noise, a bandpass wanders
// over it
func WindFXPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.8
	cfg.WarpMode = WarpFMSine
	cfg.Warp = 0.95
	cfg.Unison = 7
	cfg.Detune = 100
	cfg.Spread = 1
	cfg.Filter = BandPass
	cfg.Cutoff = 800
	cfg.Resonance = 0.3
	cfg.KeyTrack = 0.2
	cfg.LFO1Rate = 0.2
	route(&cfg, 0, modmatrix.LFO1, "cutoff", 0.3)
	cfg.AmpEnv = env(2500, 100, 1, 3000)
	cfg.Level = 1.1762 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// Percussion

// KickDrumPreset is a kick: a sine whose pitch falls fast from two octaves up, played low
func KickDrumPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0
	cfg.OscBLevel = 0.08
	cfg.OscBOctave = 2
	cfg.OscBPosition = 0.5
	cfg.FilterEnv = env(1, 60, 0, 50)
	route(&cfg, 0, modmatrix.FilterEnv, "pitch", 1)
	route(&cfg, 1, modmatrix.FilterEnv, "oscBLevel", 0.3)
	cfg.AmpEnv = env(1, 300, 0, 80)
	cfg.Level = 0.9498 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// TomDrumPreset is a tom: a slower, shallower fall of the pitch and a longer ring, with a detuned
// partner for the skin
func TomDrumPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.05
	cfg.OscBLevel = 0.3
	cfg.OscBSemi = 7.2
	cfg.OscBPosition = 0
	cfg.FilterEnv = env(1, 180, 0, 80)
	route(&cfg, 0, modmatrix.FilterEnv, "pitch", 0.45)
	cfg.AmpEnv = env(1, 550, 0, 100)
	cfg.Level = 0.8189 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SnareHitPreset is a snare: a body and a burst of noise, the noise made from heavy phase
// modulation of a stack under a highpass
func SnareHitPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.7
	cfg.WarpMode = WarpFMSine
	cfg.Warp = 0.9
	cfg.Unison = 7
	cfg.Detune = 100
	cfg.OscBLevel = 0.5
	cfg.OscBPosition = 0
	cfg.Filter = HighPass
	cfg.Cutoff = 400
	cfg.KeyTrack = 0
	cfg.FilterEnv = env(1, 80, 0, 40)
	route(&cfg, 0, modmatrix.FilterEnv, "pitch", 0.3)
	cfg.AmpEnv = env(1, 170, 0, 60)
	cfg.Level = 0.6050 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// HatMetalPreset is a hi-hat: inharmonic phase modulation of five oscillators, only the top left
func HatMetalPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.9
	cfg.WarpMode = WarpFMB
	cfg.Warp = 0.95
	cfg.OscBPosition = 0.9
	cfg.OscBOctave = 2
	cfg.OscBSemi = 5.5
	cfg.Unison = 5
	cfg.Detune = 100
	cfg.Spread = 0.8
	cfg.Filter = HighPass
	cfg.Cutoff = 7000
	cfg.Resonance = 0.3
	cfg.KeyTrack = 0
	cfg.AmpEnv = env(1, 60, 0, 40)
	cfg.Level = 0.9881 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RimClickPreset is a rim shot: a very short, resonant click
func RimClickPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.9
	cfg.Filter = BandPass
	cfg.Cutoff = 1800
	cfg.Resonance = 0.6
	cfg.KeyTrack = 0.3
	cfg.FilterEnv = env(1, 20, 0, 20)
	route(&cfg, 0, modmatrix.FilterEnv, "pitch", 0.2)
	cfg.AmpEnv = env(1, 25, 0, 20)
	cfg.Level = 1.0335 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
