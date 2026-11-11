package supersaw

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	saws := cfg.Saws
	if saws <= 0 {
		saws = 7
	}
	s := params.Set{Name: "supersaw", Params: []params.Param{
		params.K("Saws", "saws", 1, maxSaws, float64(min(saws, maxSaws)), params.Step(1)),
		params.K("Saws", "detune", 0, 1, cfg.Detune),
		params.K("Saws", "mix", 0, 1, cfg.Mix),
		params.K("Saws", "stereo", 0, 1, cfg.Stereo),
		params.K("Saws", "sub", 0, 1, cfg.Sub),

		params.K("Filter", "cutoff", 20, 18000, max(20, cfg.Cutoff), params.Exp(), params.Unit("Hz")),
		params.K("Filter", "resonance", 0, 1, cfg.Resonance),
		params.K("Filter", "hpCutoff", 20, 2000, max(20, min(cfg.HPCutoff, 2000)), params.Exp(), params.Unit("Hz"), params.Label("high pass (20 = off)")),
		params.K("Filter", "filterEnvAmount", -6, 6, cfg.FilterEnvAmount, params.Unit("oct"), params.Label("env amount")),

		params.K("LFO", "lfoRate", 0, 20, cfg.LFORate, params.Unit("Hz"), params.Label("rate")),
		params.K("LFO", "lfoCutoff", 0, 4, cfg.LFOCutoff, params.Unit("oct"), params.Label("to cutoff")),
		params.K("LFO", "lfoDetune", 0, 1, cfg.LFODetune, params.Label("to detune")),
		params.K("LFO", "lfo2Rate", 0, 20, cfg.LFO2Rate, params.Unit("Hz"), params.Label("lfo 2 rate")),
		params.C("LFO", "lfo2Shape", []string{"sine", "triangle"}, max(0, min(cfg.LFO2Shape, 1)), params.Label("lfo 2 shape")),

		params.K("Ensemble", "ensemble", 0, 1, cfg.Ensemble),
		params.K("Ensemble", "ensembleRate", 0.05, 8, orDefault(cfg.EnsembleRate, 0.6), params.Exp(), params.Unit("Hz"), params.Label("rate")),

		params.K("Voice", "level", 0, 4, orOne(cfg.Level)),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", cfg.AmpEnv.AttackDuration, cfg.AmpEnv.DecayDuration, cfg.AmpEnv.DecayLevel, cfg.AmpEnv.ReleaseDuration),
		params.E("Envelopes", "filter", "filter", cfg.FilterEnv.AttackDuration, cfg.FilterEnv.DecayDuration, cfg.FilterEnv.DecayLevel, cfg.FilterEnv.ReleaseDuration),
	}}
	m := modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		m.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		m.SetExt(i, e)
	}
	s.Params = append(s.Params, m.Describe("Modulation")...)
	return s
}

func orDefault(v, def float64) float64 {
	if v <= 0 {
		return def
	}
	return v
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
