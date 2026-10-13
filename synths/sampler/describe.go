package sampler

import (
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a sampler set as cfg: every message that changes one, its
// range, and where it is in cfg (see params). The sound is the instrument's (its regions): only the
// level of the whole is a setting. The "amp" envelope messages set an envelope over the regions'
// own, kept by the voice and not in cfg, so the description has no envelopes
func Describe(cfg Config) params.Set {
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	return params.Set{Name: "sampler", Params: []params.Param{
		params.K("Output", "level", 0, 4, level),
	}}
}
