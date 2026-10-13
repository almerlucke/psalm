package follower

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a follower set as cfg (see params)
func Describe(cfg Config) params.Set {
	hi := cfg.Max
	if cfg.Min == 0 && hi == 0 {
		hi = 1 // both 0: 0 to 1
	}
	return params.Set{Name: "follower", Params: []params.Param{
		params.K("Follower", "attack", 0.1, 500, max(0.1, cfg.Attack), params.Exp(), params.Unit("ms")),
		params.K("Follower", "release", 5, 5000, max(5, cfg.Release), params.Exp(), params.Unit("ms")),
		params.K("Follower", "gain", -12, 36, cfg.Gain, params.Unit("dB")),
		params.K("Follower", "min", 0, 1, cfg.Min),
		params.K("Follower", "max", 0, 1, hi),
	}}
}

// DescribeWah describes the settings of an auto-wah set as cfg (see params)
func DescribeWah(cfg WahConfig) params.Set {
	q := cfg.Q
	if q <= 0 {
		q = 4
	}
	return params.Set{Name: "wah", Params: []params.Param{
		params.K("Wah", "min", 50, 10000, max(50, cfg.Min), params.Exp(), params.Unit("Hz")),
		params.K("Wah", "max", 50, 10000, max(50, cfg.Max), params.Exp(), params.Unit("Hz")),
		params.K("Wah", "q", 0.5, 20, q, params.Exp(), params.Label("Q")),
		params.K("Wah", "gain", -12, 36, cfg.Gain, params.Unit("dB")),
		params.K("Wah", "attack", 0.1, 500, max(0.1, cfg.Attack), params.Exp(), params.Unit("ms")),
		params.K("Wah", "release", 5, 5000, max(5, cfg.Release), params.Exp(), params.Unit("ms")),
		params.K("Wah", "mix", 0, 1, cfg.Mix),
		params.C("Wah", "kind", []string{"band pass", "low pass"}, int(cfg.Kind), params.Label("filter")),
		params.S("Wah", "down", cfg.Down, params.Label("sweep down")),
	}}
}
