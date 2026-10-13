package fdnreverb

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a reverb set as cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "fdnreverb", Params: []params.Param{
		params.K("Reverb", "size", 0, 1, cfg.Size),
		params.K("Reverb", "decay", 0.1, 30, max(0.1, cfg.Decay), params.Exp(), params.Unit("s")),
		params.K("Reverb", "damping", 0, 1, cfg.Damping),
		params.K("Reverb", "preDelay", 0, 490, cfg.PreDelay, params.Unit("ms"), params.Label("pre delay")),
		params.K("Reverb", "diffusion", 0, 1, cfg.Diffusion),
		params.K("Reverb", "modDepth", 0, 4, cfg.ModDepth, params.Unit("ms"), params.Label("mod depth")),
		params.K("Reverb", "modRate", 0.05, 5, max(0.05, cfg.ModRate), params.Exp(), params.Unit("Hz"), params.Label("mod rate")),
		params.K("Reverb", "width", 0, 1, cfg.Width),
		params.K("Reverb", "mix", 0, 1, cfg.Mix),
	}}
}
