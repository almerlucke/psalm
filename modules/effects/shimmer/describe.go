package shimmer

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a shimmer reverb set as cfg (see params): the shimmer and the
// reverb tank
func Describe(cfg Config) params.Set {
	r := cfg.Reverb
	return params.Set{Name: "shimmer", Params: []params.Param{
		params.K("Shimmer", "shimmer", 0, 1, cfg.Shimmer),
		params.K("Shimmer", "semitones", -24, 24, cfg.Semitones, params.Unit("st")),
		params.K("Shimmer", "lowCut", 20, 2000, max(20, cfg.LowCut), params.Exp(), params.Unit("Hz"), params.Label("low cut")),
		params.K("Shimmer", "mix", 0, 1, cfg.Mix),

		params.K("Reverb", "size", 0, 1, r.Size),
		params.K("Reverb", "decay", 0.1, 30, max(0.1, r.Decay), params.Exp(), params.Unit("s")),
		params.K("Reverb", "damping", 0, 1, r.Damping),
		params.K("Reverb", "preDelay", 0, 490, r.PreDelay, params.Unit("ms"), params.Label("pre-delay")),
		params.K("Reverb", "diffusion", 0, 1, r.Diffusion),
		params.K("Reverb", "modDepth", 0, 4, r.ModDepth, params.Unit("ms"), params.Label("mod depth")),
		params.K("Reverb", "modRate", 0.05, 5, max(0.05, r.ModRate), params.Exp(), params.Unit("Hz"), params.Label("mod rate")),
		params.K("Reverb", "width", 0, 1, r.Width),
	}}
}
