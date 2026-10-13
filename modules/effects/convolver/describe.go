package convolver

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a convolver set as cfg that it takes by message (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "convolver", Params: []params.Param{
		params.K("Convolver", "mix", 0, 1, cfg.Mix),
		params.K("Convolver", "gain", 0, 4, cfg.Gain),
	}}
}
