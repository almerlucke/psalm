package grains

import (
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	level := cfg.Level
	if level == 0 {
		level = 1
	}
	return params.Set{Name: "grains", Params: []params.Param{
		params.K("Sample", "rootNote", 0, 127, cfg.RootNote, params.Label("root note")),
		params.K("Sample", "position", 0, 1, cfg.Position),
		params.K("Sample", "scan", -2, 2, cfg.Scan),
		params.K("Sample", "spray", 0, 1, cfg.Spray, params.Unit("s")),

		params.K("Grains", "density", 1, 200, cfg.Density, params.Exp(), params.Unit("Hz")),
		params.K("Grains", "grainSize", 5, 1000, cfg.GrainSize, params.Exp(), params.Unit("ms"), params.Label("size")),
		params.K("Grains", "shape", 0, 1, cfg.Shape),
		params.K("Grains", "jitter", 0, 1, cfg.Jitter),
		params.K("Grains", "reverse", 0, 1, cfg.Reverse),

		params.K("Spread", "pitchSpread", 0, 12, cfg.PitchSpread, params.Unit("st"), params.Label("pitch")),
		params.K("Spread", "octave", 0, 1, cfg.Octave),
		params.K("Spread", "stereoSpread", 0, 1, cfg.StereoSpread, params.Label("stereo")),

		params.K("Output", "level", 0, 16, level),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", cfg.AmpEnv.AttackDuration, cfg.AmpEnv.DecayDuration, cfg.AmpEnv.DecayLevel, cfg.AmpEnv.ReleaseDuration),
	}}
}
