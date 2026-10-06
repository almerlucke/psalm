package wavetable

import "github.com/almerlucke/genny/float/envelopes/adsr"

// Presets take the table to play, the character comes from the settings so every preset works
// with any table. The position is where the preset starts, tables differ a lot in what they hold
// along their frames, so try moving it

// Presets maps preset names to their constructors
var Presets = map[string]func(table *Table) Config{
	"pad":     PadPreset,
	"lead":    LeadPreset,
	"pluck":   PluckPreset,
	"bass":    BassPreset,
	"growl":   GrowlPreset,
	"sweep":   SweepPreset,
	"keys":    KeysPreset,
	"anthem":  AnthemPreset,
	"strings": StringsPreset,
	"vox":     VoxPreset,
	"brass":   BrassPreset,
	"organ":   OrganPreset,
	"reese":   ReesePreset,
	"acid":    AcidPreset,
	"stab":    StabPreset,
	"drone":   DronePreset,
	"glass":   GlassPreset,
	"riser":   RiserPreset,
}

func preset(table *Table) Config {
	cfg := DefaultConfig()
	cfg.Table = table
	return cfg
}

// PadPreset is a wide, slow pad: five detuned oscillators drifting slowly through the table
func PadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.4
	cfg.PositionLFORate = 0.07
	cfg.PositionLFODepth = 0.25
	cfg.Unison = 5
	cfg.Detune = 18
	cfg.Spread = 0.9
	cfg.Filter = LowPass
	cfg.Cutoff = 3000
	cfg.Resonance = 0.1
	cfg.KeyTrack = 0.3
	cfg.AmpEnv = *adsr.NewSetting(1.0, 900.0, 0.9, 500.0, 1.0, 1800.0)
	cfg.Level = 0.5124 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// LeadPreset is a singing lead: three oscillators, vibrato, glide and a filter that opens a little
// at every note
func LeadPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.3
	cfg.Unison = 3
	cfg.Detune = 12
	cfg.Spread = 0.5
	cfg.Filter = LowPass
	cfg.Cutoff = 2500
	cfg.Resonance = 0.25
	cfg.FilterEnvAmount = 1
	cfg.FilterEnv = *adsr.NewSetting(1.0, 10.0, 0.4, 400.0, 1.0, 300.0)
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.12
	cfg.Glide = 60
	cfg.AmpEnv = *adsr.NewSetting(1.0, 10.0, 0.85, 200.0, 1.0, 250.0)
	cfg.Level = 0.4749 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PluckPreset is a bright pluck: the position jumps up the table at every note and falls back,
// while the filter closes
func PluckPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.1
	cfg.PositionEnvAmount = 0.6
	cfg.PositionEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 300.0, 1.0, 200.0)
	cfg.Unison = 2
	cfg.Detune = 8
	cfg.Spread = 0.6
	cfg.Filter = LowPass
	cfg.Cutoff = 500
	cfg.Resonance = 0.2
	cfg.FilterEnvAmount = 4
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 250.0, 1.0, 200.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.0, 700.0, 1.0, 200.0)
	cfg.Level = 0.8286 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BassPreset is a solid bass: one oscillator with a sine sub and a lowpass that snaps shut
func BassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.2
	cfg.SubLevel = 0.5
	cfg.Filter = LowPass
	cfg.Cutoff = 300
	cfg.Resonance = 0.3
	cfg.KeyTrack = 0.5
	cfg.FilterEnvAmount = 3
	cfg.FilterEnv = *adsr.NewSetting(1.0, 2.0, 0.1, 220.0, 1.0, 150.0)
	cfg.Glide = 20
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3.0, 0.8, 200.0, 1.0, 100.0)
	cfg.Level = 0.5003 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GrowlPreset is a moving bass: a fast LFO rocks the position, so the timbre growls
func GrowlPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.PositionLFORate = 4
	cfg.PositionLFODepth = 0.3
	cfg.Unison = 2
	cfg.Detune = 10
	cfg.Spread = 0.4
	cfg.SubLevel = 0.4
	cfg.Filter = LowPass
	cfg.Cutoff = 1800
	cfg.Resonance = 0.35
	cfg.Glide = 30
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3.0, 0.9, 200.0, 1.0, 120.0)
	cfg.Level = 0.4101 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// SweepPreset is an evolving texture: the position travels through the whole table over four
// seconds and back on release
func SweepPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0
	cfg.PositionEnvAmount = 1
	cfg.PositionEnv = *adsr.NewSetting(1.0, 4000.0, 1.0, 10.0, 1.0, 3000.0)
	cfg.Unison = 3
	cfg.Detune = 10
	cfg.Spread = 0.8
	cfg.AmpEnv = *adsr.NewSetting(1.0, 600.0, 0.9, 500.0, 1.0, 2500.0)
	cfg.Level = 0.4564 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// KeysPreset is a bell like key: a slightly detuned pair, no filter, a struck attack and a long
// decay that also moves a little down the table
func KeysPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.3
	cfg.PositionEnvAmount = 0.15
	cfg.PositionEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 1200.0, 1.0, 600.0)
	cfg.Unison = 2
	cfg.Detune = 6
	cfg.Spread = 0.5
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.0, 1800.0, 1.0, 500.0)
	cfg.Level = 0.6712 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AnthemPreset is a huge stack: seven oscillators wide apart, for chords that fill the room
func AnthemPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.6
	cfg.Unison = 7
	cfg.Detune = 35
	cfg.Spread = 1
	cfg.Filter = LowPass
	cfg.Cutoff = 4000
	cfg.Resonance = 0.15
	cfg.FilterEnvAmount = 0.5
	cfg.FilterEnv = *adsr.NewSetting(1.0, 300.0, 0.5, 800.0, 1.0, 600.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 30.0, 0.9, 300.0, 1.0, 700.0)
	cfg.Level = 0.4572 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// StringsPreset is a string section: five gently detuned oscillators, a slow bow, a soft lowpass
// and a little vibrato
func StringsPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.35
	cfg.PositionLFORate = 0.15
	cfg.PositionLFODepth = 0.05
	cfg.Unison = 5
	cfg.Detune = 12
	cfg.Spread = 0.8
	cfg.Filter = LowPass
	cfg.Cutoff = 2200
	cfg.Resonance = 0.1
	cfg.KeyTrack = 0.6
	cfg.VibratoRate = 5
	cfg.VibratoDepth = 0.06
	cfg.AmpEnv = *adsr.NewSetting(1.0, 350.0, 0.9, 300.0, 1.0, 700.0)
	cfg.Level = 0.4759 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// VoxPreset is a vocal like lead: a resonant bandpass picks out a formant while the position
// drifts, with vibrato
func VoxPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.PositionLFORate = 0.4
	cfg.PositionLFODepth = 0.15
	cfg.Unison = 3
	cfg.Detune = 8
	cfg.Spread = 0.5
	cfg.Filter = BandPass
	cfg.Cutoff = 1100
	cfg.Resonance = 0.55
	cfg.KeyTrack = 0.2
	cfg.VibratoRate = 5.2
	cfg.VibratoDepth = 0.15
	cfg.Glide = 50
	cfg.AmpEnv = *adsr.NewSetting(1.0, 60.0, 0.85, 300.0, 1.0, 350.0)
	cfg.Level = 0.7487 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BrassPreset is a brass section: the filter swells open during the attack and settles back, like
// players digging into the note
func BrassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.6
	cfg.Unison = 3
	cfg.Detune = 10
	cfg.Spread = 0.6
	cfg.Filter = LowPass
	cfg.Cutoff = 600
	cfg.Resonance = 0.15
	cfg.KeyTrack = 0.7
	cfg.FilterEnvAmount = 2.5
	cfg.FilterEnv = *adsr.NewSetting(1.0, 90.0, 0.5, 400.0, 1.0, 250.0)
	cfg.VibratoRate = 4.5
	cfg.VibratoDepth = 0.05
	cfg.AmpEnv = *adsr.NewSetting(1.0, 50.0, 0.85, 300.0, 1.0, 250.0)
	cfg.Level = 0.5112 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// OrganPreset is a steady organ: one oscillator with a sub, no filter, full sustain and a fast
// release
func OrganPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.15
	cfg.SubLevel = 0.3
	cfg.AmpEnv = *adsr.NewSetting(1.0, 8.0, 1.0, 10.0, 1.0, 60.0)
	cfg.Level = 0.3758 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// ReesePreset is a reese bass: two oscillators detuned far apart so they beat and phase, kept
// narrow, with a sine sub under a dark lowpass
func ReesePreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.6
	cfg.PositionLFORate = 0.2
	cfg.PositionLFODepth = 0.1
	cfg.Unison = 2
	cfg.Detune = 28
	cfg.Spread = 0.2
	cfg.SubLevel = 0.5
	cfg.Filter = LowPass
	cfg.Cutoff = 700
	cfg.Resonance = 0.2
	cfg.KeyTrack = 0.3
	cfg.Glide = 40
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.9, 200.0, 1.0, 150.0)
	cfg.Level = 0.4332 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// AcidPreset is a squelchy bass line: a resonant lowpass that snaps open at every note and glide
// between notes
func AcidPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.7
	cfg.Filter = LowPass
	cfg.Cutoff = 250
	cfg.Resonance = 0.8
	cfg.KeyTrack = 0.4
	cfg.FilterEnvAmount = 4
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 180.0, 1.0, 100.0)
	cfg.Glide = 60
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.7, 200.0, 1.0, 60.0)
	cfg.Level = 0.299 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// StabPreset is a short chord stab: four wide oscillators and a filter that closes fast
func StabPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.Unison = 4
	cfg.Detune = 22
	cfg.Spread = 0.9
	cfg.Filter = LowPass
	cfg.Cutoff = 800
	cfg.Resonance = 0.25
	cfg.FilterEnvAmount = 3
	cfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 200.0, 1.0, 100.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.0, 350.0, 1.0, 150.0)
	cfg.Level = 0.8376 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// DronePreset is a slowly breathing drone: seven close oscillators, a very slow sweep through half
// the table and a long fade in and out
func DronePreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.5
	cfg.PositionLFORate = 0.03
	cfg.PositionLFODepth = 0.45
	cfg.Unison = 7
	cfg.Detune = 8
	cfg.Spread = 1
	cfg.SubLevel = 0.3
	cfg.Filter = LowPass
	cfg.Cutoff = 1500
	cfg.Resonance = 0.2
	cfg.KeyTrack = 0.2
	cfg.AmpEnv = *adsr.NewSetting(1.0, 3000.0, 1.0, 10.0, 1.0, 4000.0)
	cfg.Level = 0.7687 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// GlassPreset is a thin, airy bell: a highpass removes the body, the position falls a little after
// the strike
func GlassPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0.4
	cfg.PositionEnvAmount = 0.25
	cfg.PositionEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 600.0, 1.0, 400.0)
	cfg.Unison = 3
	cfg.Detune = 5
	cfg.Spread = 0.8
	cfg.Filter = HighPass
	cfg.Cutoff = 700
	cfg.Resonance = 0.2
	cfg.KeyTrack = 1
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 2200.0, 1.0, 800.0)
	cfg.Level = 0.4878 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// RiserPreset builds tension: over six seconds the position climbs through the table while the
// filter opens, hold a note to ride it up
func RiserPreset(table *Table) Config {
	cfg := preset(table)
	cfg.Position = 0
	cfg.PositionEnvAmount = 1
	cfg.PositionEnv = *adsr.NewSetting(1.0, 6000.0, 1.0, 10.0, 1.0, 500.0)
	cfg.Unison = 5
	cfg.Detune = 25
	cfg.Spread = 1
	cfg.Filter = LowPass
	cfg.Cutoff = 300
	cfg.Resonance = 0.4
	cfg.KeyTrack = 0
	cfg.FilterEnvAmount = 5
	cfg.FilterEnv = *adsr.NewSetting(1.0, 6000.0, 1.0, 10.0, 1.0, 500.0)
	cfg.AmpEnv = *adsr.NewSetting(1.0, 2000.0, 1.0, 10.0, 1.0, 600.0)
	cfg.Level = 0.6774 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
