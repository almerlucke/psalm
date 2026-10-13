package flanger

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a flanger set as cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "flanger", Params: []params.Param{
		params.K("Flanger", "rate", 0.02, 10, max(0.02, cfg.Rate), params.Exp(), params.Unit("Hz")),
		params.K("Flanger", "beats", 0, 32, cfg.Beats),
		params.K("Flanger", "manual", 0.1, 10, max(0.1, cfg.Manual), params.Exp(), params.Unit("ms")),
		params.K("Flanger", "depth", 0, 20, cfg.Depth, params.Unit("ms")),
		params.K("Flanger", "feedback", -0.95, 0.95, cfg.Feedback),
		params.K("Flanger", "stereo", 0, 1, cfg.Stereo),
		params.K("Flanger", "mix", 0, 1, cfg.Mix),
	}}
}
