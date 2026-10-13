package supersaw

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	saws := cfg.Saws
	if saws <= 0 {
		saws = 7
	}
	return params.Set{Name: "supersaw", Params: []params.Param{
		params.K("Saws", "saws", 1, maxSaws, float64(min(saws, maxSaws)), params.Step(1)),
		params.K("Saws", "detune", 0, 1, cfg.Detune),
		params.K("Saws", "mix", 0, 1, cfg.Mix),
		params.K("Saws", "stereo", 0, 1, cfg.Stereo),
		params.K("Saws", "sub", 0, 1, cfg.Sub),

		params.K("Filter", "cutoff", 20, 18000, max(20, cfg.Cutoff), params.Exp(), params.Unit("Hz")),
		params.K("Filter", "resonance", 0, 1, cfg.Resonance),
		params.K("Filter", "filterEnvAmount", -6, 6, cfg.FilterEnvAmount, params.Unit("oct"), params.Label("env amount")),

		params.K("Voice", "level", 0, 4, orOne(cfg.Level)),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", cfg.AmpEnv.AttackDuration, cfg.AmpEnv.DecayDuration, cfg.AmpEnv.DecayLevel, cfg.AmpEnv.ReleaseDuration),
		params.E("Envelopes", "filter", "filter", cfg.FilterEnv.AttackDuration, cfg.FilterEnv.DecayDuration, cfg.FilterEnv.DecayLevel, cfg.FilterEnv.ReleaseDuration),
	}}
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
