package rotary

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a rotary speaker set as cfg (see params)
func Describe(cfg Config) params.Set {
	horn, spread := cfg.Horn, cfg.Spread
	if horn <= 0 {
		horn = 0.5 // 0 is as loud as the drum
	}
	if spread <= 0 {
		spread = 1
	}
	return params.Set{Name: "rotary", Params: []params.Param{
		params.S("Rotary", "fast", cfg.Fast),
		params.K("Rotary", "drive", 0, 24, cfg.Drive, params.Unit("dB")),
		params.K("Rotary", "horn", 0.01, 1, horn),
		params.K("Rotary", "spread", 0.01, 1, spread),
		params.K("Rotary", "mix", 0, 1, cfg.Mix),
	}}
}
