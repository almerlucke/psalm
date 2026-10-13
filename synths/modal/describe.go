package modal

import (
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The ring and the damper are the "amp" envelope: a decay and a
// release ("decay" and "release" set them too, in seconds)
func Describe(cfg Config) params.Set {
	pos := cfg.Position
	if pos <= 0 {
		pos = 0.25
	}
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	return params.Set{Name: "modal", Params: []params.Param{
		params.K("Mallet", "hardness", 0, 1, cfg.Hardness),
		params.K("Mallet", "position", 0.01, 1, pos),
		params.K("Mallet", "noise", 0, 1, cfg.Noise),

		params.K("Body", "highDamping", 0, 2, cfg.HighDamping, params.Label("high damping")),

		params.K("Tremolo", "tremolo", 0, 15, cfg.Tremolo, params.Unit("Hz"), params.Label("rate")),
		params.K("Tremolo", "tremoloDepth", 0, 1, cfg.TremoloDepth, params.Label("depth")),

		params.K("Output", "level", 0, 4, level),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "amp", Group: "Envelopes", Stages: "DR", Decay: cfg.Decay * 1000, Release: cfg.Release * 1000},
	}}
}
