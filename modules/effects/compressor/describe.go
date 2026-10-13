package compressor

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a compressor set as cfg (see params)
func Describe(cfg Config) params.Set {
	sideCut := cfg.SideCut
	if sideCut <= 0 {
		sideCut = 20 // off: as low as it goes
	}
	return params.Set{Name: "compressor", Params: []params.Param{
		params.K("Compressor", "threshold", -60, 0, max(-60, cfg.Threshold), params.Unit("dB")),
		params.K("Compressor", "ratio", 1, 20, max(1, cfg.Ratio), params.Exp()),
		params.K("Compressor", "knee", 0, 24, cfg.Knee, params.Unit("dB")),
		params.K("Time", "attack", 0.1, 200, max(0.1, cfg.Attack), params.Exp(), params.Unit("ms")),
		params.K("Time", "release", 5, 2000, max(5, cfg.Release), params.Exp(), params.Unit("ms")),
		params.K("Output", "makeup", -12, 24, cfg.Makeup, params.Unit("dB")),
		params.K("Output", "mix", 0, 1, cfg.Mix),
		params.K("Sidechain", "sideCut", 20, 500, sideCut, params.Exp(), params.Unit("Hz"), params.Label("side cut")),
	}}
}
