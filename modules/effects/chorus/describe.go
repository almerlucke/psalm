package chorus

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a chorus set as cfg (see params)
func Describe(cfg Config) params.Set {
	tone := cfg.Tone
	if tone <= 0 {
		tone = 20000 // off: as open as it goes
	}
	return params.Set{Name: "chorus", Params: []params.Param{
		params.K("Chorus", "rate", 0.05, 10, max(0.05, cfg.Rate), params.Exp(), params.Unit("Hz")),
		params.K("Chorus", "depth", 0, 10, cfg.Depth, params.Unit("ms")),
		params.K("Chorus", "delay", 1, 35, max(1, cfg.Delay), params.Unit("ms")),
		params.K("Chorus", "spread", 0, 1, cfg.Spread),
		params.K("Chorus", "feedback", 0, 0.9, cfg.Feedback),
		params.K("Chorus", "tone", 500, 20000, tone, params.Exp(), params.Unit("Hz")),
		params.K("Chorus", "mix", 0, 1, cfg.Mix),
	}}
}
