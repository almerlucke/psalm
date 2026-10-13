package widener

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a widener set as cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "widener", Params: []params.Param{
		params.K("Widener", "width", 0, 3, cfg.Width),
		params.K("Widener", "monoBelow", 20, 1000, max(20, cfg.MonoBelow), params.Exp(), params.Unit("Hz"), params.Label("mono below")),
	}}
}
