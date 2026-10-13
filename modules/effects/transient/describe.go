package transient

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a transient shaper set as cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "transient", Params: []params.Param{
		params.K("Transient", "attack", -24, 24, cfg.Attack, params.Unit("dB")),
		params.K("Transient", "sustain", -24, 24, cfg.Sustain, params.Unit("dB")),
		params.K("Transient", "output", -24, 12, cfg.Output, params.Unit("dB")),
		params.K("Transient", "mix", 0, 1, cfg.Mix),
	}}
}
