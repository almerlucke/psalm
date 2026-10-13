package gate

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a gate set as cfg (see params)
func Describe(cfg Config) params.Set {
	ratio := cfg.Ratio
	if ratio <= 0 || ratio > 100 {
		ratio = 100 // a gate
	}
	rng := cfg.Range
	if rng <= 0 {
		rng = 80
	}
	sideCut := cfg.SideCut
	if sideCut <= 0 {
		sideCut = 20 // off: as low as it goes
	}
	return params.Set{Name: "gate", Params: []params.Param{
		params.K("Gate", "threshold", -80, 0, max(-80, cfg.Threshold), params.Unit("dB")),
		params.K("Gate", "ratio", 1, 100, max(1, ratio), params.Exp()),
		params.K("Gate", "range", 1, 100, max(1, min(rng, 100)), params.Unit("dB")),
		params.K("Gate", "attack", 0.1, 100, max(0.1, cfg.Attack), params.Exp(), params.Unit("ms")),
		params.K("Gate", "hold", 0, 500, cfg.Hold, params.Unit("ms")),
		params.K("Gate", "release", 5, 2000, max(5, cfg.Release), params.Exp(), params.Unit("ms")),
		params.K("Gate", "sideCut", 20, 500, sideCut, params.Exp(), params.Unit("Hz"), params.Label("side cut")),
	}}
}
