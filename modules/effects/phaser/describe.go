package phaser

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a phaser set as cfg (see params). "sync" at 0 is off: the
// sweep follows "rate"
func Describe(cfg Config) params.Set {
	return params.Set{Name: "phaser", Params: []params.Param{
		params.K("Phaser", "rate", 0.01, 10, max(0.01, cfg.Rate), params.Exp(), params.Unit("Hz")),
		params.K("Phaser", "sync", 0, 32, max(0, cfg.Sync), params.Unit("beats")),
		params.K("Phaser", "min", 20, 10000, max(20, cfg.Min), params.Exp(), params.Unit("Hz")),
		params.K("Phaser", "max", 100, 20000, max(100, cfg.Max), params.Exp(), params.Unit("Hz")),
		params.K("Phaser", "feedback", -0.95, 0.95, cfg.Feedback),
		params.K("Phaser", "stereo", 0, 1, cfg.Stereo),
		params.K("Phaser", "mix", 0, 1, cfg.Mix),
	}}
}
