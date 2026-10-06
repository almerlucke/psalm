package fm

import "github.com/almerlucke/genny/float/envelopes/adsr"

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"ePiano":   EPianoPreset,
	"bell":     BellPreset,
	"bass":     BassPreset,
	"brass":    BrassPreset,
	"organ":    OrganPreset,
	"mallet":   MalletPreset,
	"pad":      PadPreset,
	"pluck":    PluckPreset,
	"metallic": MetallicPreset,
	"lead":     LeadPreset,
}

func op(ratio float64, level float64, velSens float64, env *adsr.Setting) OperatorConfig {
	return OperatorConfig{Ratio: ratio, Level: level, VelocitySens: velSens, Env: *env}
}

// EPianoPreset is a tine electric piano: a 1:1 pair for the body and a high ratio pair for the
// bright, quickly decaying attack. Velocity makes it brighter
func EPianoPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4 // 4>3, 2>1
	cfg.Ops[0] = op(1, 1.0, 0.2, adsr.NewSetting(1.0, 2.0, 0.0, 2500.0, 1.0, 400.0))
	cfg.Ops[1] = op(1, 0.35, 0.8, adsr.NewSetting(1.0, 2.0, 0.1, 1200.0, 1.0, 300.0))
	cfg.Ops[2] = op(1, 0.25, 0.3, adsr.NewSetting(1.0, 1.0, 0.0, 600.0, 1.0, 200.0))
	cfg.Ops[3] = op(14, 0.12, 0.9, adsr.NewSetting(1.0, 1.0, 0.0, 80.0, 1.0, 50.0))
	cfg.Ops[2].Detune = 3.0
	cfg.Level = 1.133 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BellPreset is a clear bell with inharmonic partials from a 1:3.5 ratio
func BellPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, adsr.NewSetting(1.0, 1.0, 0.0, 4000.0, 1.0, 1500.0))
	cfg.Ops[1] = op(3.5, 0.6, 0.6, adsr.NewSetting(1.0, 1.0, 0.0, 2500.0, 1.0, 1000.0))
	cfg.Ops[2] = op(2, 0.5, 0.2, adsr.NewSetting(1.0, 1.0, 0.0, 3000.0, 1.0, 1200.0))
	cfg.Ops[3] = op(5.19, 0.3, 0.5, adsr.NewSetting(1.0, 1.0, 0.0, 1500.0, 1.0, 800.0))
	cfg.Level = 0.6964 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BassPreset is a punchy FM bass: a stack whose top operator snaps shut quickly
func BassPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0 // 4>3>2>1
	cfg.Ops[0] = op(1, 1.0, 0.0, adsr.NewSetting(1.0, 1.0, 0.8, 200.0, 1.0, 80.0))
	cfg.Ops[1] = op(1, 0.5, 0.6, adsr.NewSetting(1.0, 1.0, 0.25, 180.0, 1.0, 80.0))
	cfg.Ops[2] = op(2, 0.25, 0.5, adsr.NewSetting(1.0, 1.0, 0.0, 90.0, 1.0, 50.0))
	cfg.Ops[3] = op(1, 0.15, 0.3, adsr.NewSetting(1.0, 1.0, 0.0, 60.0, 1.0, 40.0))
	cfg.Feedback = 0.3
	cfg.Level = 0.4003 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// BrassPreset is a brass section like tone: the modulators swell in slower than the carrier
func BrassPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 1 // (4+3)>2>1
	cfg.Ops[0] = op(1, 1.0, 0.2, adsr.NewSetting(1.0, 40.0, 0.85, 200.0, 1.0, 200.0))
	cfg.Ops[1] = op(1, 0.45, 0.7, adsr.NewSetting(1.0, 90.0, 0.7, 300.0, 1.0, 200.0))
	cfg.Ops[2] = op(1, 0.2, 0.5, adsr.NewSetting(1.0, 120.0, 0.6, 300.0, 1.0, 200.0))
	cfg.Ops[3] = op(1, 0.2, 0.5, adsr.NewSetting(1.0, 80.0, 0.5, 300.0, 1.0, 200.0))
	cfg.Ops[3].Detune = 7.0
	cfg.Feedback = 0.25
	cfg.VibratoRate = 5.0
	cfg.VibratoDepth = 0.1
	cfg.VibratoDelay = 400.0
	cfg.Level = 0.4089 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// OrganPreset is an additive organ: four carriers at sub, unison, octave and octave plus fifth
func OrganPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 7 // all carriers
	sustain := adsr.NewSetting(1.0, 5.0, 1.0, 10.0, 1.0, 60.0)
	cfg.Ops[0] = op(1, 1.0, 0, sustain)
	cfg.Ops[1] = op(0.5, 0.7, 0, sustain)
	cfg.Ops[2] = op(2, 0.6, 0, sustain)
	cfg.Ops[3] = op(3, 0.4, 0, sustain)
	cfg.Feedback = 0.1
	cfg.Level = 0.8223 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MalletPreset is a marimba like mallet: a short, woody attack from a 1:4 modulator
func MalletPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 4
	cfg.Ops[0] = op(1, 1.0, 0.2, adsr.NewSetting(1.0, 1.0, 0.0, 600.0, 1.0, 200.0))
	cfg.Ops[1] = op(4, 0.4, 0.7, adsr.NewSetting(1.0, 1.0, 0.0, 60.0, 1.0, 40.0))
	cfg.Ops[2] = op(4, 0.25, 0.2, adsr.NewSetting(1.0, 1.0, 0.0, 300.0, 1.0, 100.0))
	cfg.Ops[3] = op(10, 0.2, 0.6, adsr.NewSetting(1.0, 1.0, 0.0, 30.0, 1.0, 20.0))
	cfg.Level = 1.266 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PadPreset is a warm, slowly evolving pad from detuned parallel carriers
func PadPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 5 // 4>(3, 2, 1)
	cfg.Ops[0] = op(1, 1.0, 0, adsr.NewSetting(1.0, 800.0, 0.9, 1000.0, 1.0, 1800.0))
	cfg.Ops[1] = op(1, 0.8, 0, adsr.NewSetting(1.0, 900.0, 0.9, 1000.0, 1.0, 1800.0))
	cfg.Ops[2] = op(2, 0.4, 0, adsr.NewSetting(1.0, 1100.0, 0.8, 1000.0, 1.0, 1800.0))
	cfg.Ops[3] = op(1, 0.15, 0, adsr.NewSetting(1.0, 2000.0, 0.6, 2000.0, 1.0, 2000.0))
	cfg.Ops[1].Detune = 6.0
	cfg.Ops[2].Detune = -5.0
	cfg.VibratoRate = 4.5
	cfg.VibratoDepth = 0.06
	cfg.VibratoDelay = 600.0
	cfg.Level = 0.8441 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// PluckPreset is a bright plucked string: a modulator that closes quickly, like a harp or koto
func PluckPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 2 // (4 + 3>2)>1
	cfg.Ops[0] = op(1, 1.0, 0.2, adsr.NewSetting(1.0, 1.0, 0.0, 900.0, 1.0, 300.0))
	cfg.Ops[1] = op(3, 0.3, 0.7, adsr.NewSetting(1.0, 1.0, 0.0, 150.0, 1.0, 100.0))
	cfg.Ops[2] = op(1, 0.2, 0.5, adsr.NewSetting(1.0, 1.0, 0.0, 200.0, 1.0, 100.0))
	cfg.Ops[3] = op(1, 0.3, 0.6, adsr.NewSetting(1.0, 1.0, 0.0, 120.0, 1.0, 80.0))
	cfg.Feedback = 0.2
	cfg.Level = 0.5593 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// MetallicPreset is a gong / metal plate sound from inharmonic ratios and deep modulation
func MetallicPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 1
	cfg.Ops[0] = op(1, 1.0, 0.2, adsr.NewSetting(1.0, 2.0, 0.0, 5000.0, 1.0, 2000.0))
	cfg.Ops[1] = op(1.41, 0.9, 0.5, adsr.NewSetting(1.0, 2.0, 0.0, 3500.0, 1.0, 1500.0))
	cfg.Ops[2] = op(2.77, 0.6, 0.5, adsr.NewSetting(1.0, 2.0, 0.0, 2500.0, 1.0, 1200.0))
	cfg.Ops[3] = op(3.93, 0.5, 0.5, adsr.NewSetting(1.0, 2.0, 0.0, 2000.0, 1.0, 1000.0))
	cfg.Feedback = 0.15
	cfg.Level = 0.3087 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}

// LeadPreset is a bright lead: feedback on the top of a stack gives a saw like edge
func LeadPreset() Config {
	cfg := DefaultConfig()
	cfg.Algorithm = 0
	cfg.Ops[0] = op(1, 1.0, 0.1, adsr.NewSetting(1.0, 5.0, 0.85, 200.0, 1.0, 250.0))
	cfg.Ops[1] = op(1, 0.35, 0.5, adsr.NewSetting(1.0, 10.0, 0.6, 400.0, 1.0, 250.0))
	cfg.Ops[2] = op(1, 0.2, 0.3, adsr.NewSetting(1.0, 10.0, 0.7, 400.0, 1.0, 250.0))
	cfg.Ops[3] = op(1, 0.15, 0.2, adsr.NewSetting(1.0, 10.0, 0.8, 400.0, 1.0, 250.0))
	cfg.Feedback = 0.6
	cfg.Glide = 30.0
	cfg.VibratoRate = 5.5
	cfg.VibratoDepth = 0.2
	cfg.VibratoDelay = 300.0
	cfg.Level = 0.3409 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return cfg
}
