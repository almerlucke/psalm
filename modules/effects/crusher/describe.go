package crusher

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a bit crusher set as cfg (see params)
func Describe(cfg Config) params.Set {
	bits := cfg.Bits
	if bits <= 0 || bits > 24 {
		bits = 24 // off: as fine as it goes
	}
	rate := cfg.Rate
	if rate <= 0 || rate > 44100 {
		rate = 44100 // off: as high as it goes
	}
	return params.Set{Name: "crusher", Params: []params.Param{
		params.K("Crusher", "bits", 1, 24, max(1, bits)),
		params.K("Crusher", "rate", 100, 44100, max(100, rate), params.Exp(), params.Unit("Hz")),
		params.K("Crusher", "output", -24, 24, cfg.Output, params.Unit("dB")),
		params.K("Crusher", "mix", 0, 1, cfg.Mix),
	}}
}
