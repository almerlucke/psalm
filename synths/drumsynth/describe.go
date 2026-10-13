package drumsynth

import (
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a drum synth set as cfg: every message that changes one, its
// range, and where it is in cfg (see params). The sounds are set per sound in the kit, not by
// message: only the level of the whole is
func Describe(cfg Config) params.Set {
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	return params.Set{Name: "drumsynth", Params: []params.Param{
		params.K("Output", "level", 0, 4, level),
	}}
}
