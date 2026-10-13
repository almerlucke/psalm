package ringmod

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a ring modulator set as cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "ringmod", Params: []params.Param{
		params.C("Ring mod", "mode", []string{"ring", "shift"}, int(cfg.Mode)),
		params.K("Ring mod", "freq", -4000, 4000, cfg.Freq, params.Unit("Hz")),
		params.K("Ring mod", "stereo", -50, 50, cfg.Stereo, params.Unit("Hz")),
		params.K("Ring mod", "mix", 0, 1, cfg.Mix),
	}}
}
