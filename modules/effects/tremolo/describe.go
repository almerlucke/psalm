package tremolo

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a tremolo or auto-pan set as cfg (see params). "sync" at 0 is
// off: the cycle follows "rate"
func Describe(cfg Config) params.Set {
	return params.Set{Name: "tremolo", Params: []params.Param{
		params.C("Tremolo", "mode", []string{"tremolo", "pan"}, int(cfg.Mode)),
		params.C("Tremolo", "shape", []string{"sine", "triangle", "square"}, int(cfg.Shape)),
		params.K("Tremolo", "rate", 0.05, 20, max(0.05, cfg.Rate), params.Exp(), params.Unit("Hz")),
		params.K("Tremolo", "sync", 0, 4, max(0, cfg.Sync), params.Unit("beats")),
		params.K("Tremolo", "depth", 0, 1, cfg.Depth),
		params.K("Tremolo", "stereo", 0, 1, cfg.Stereo),
	}}
}
