package modal

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The ring and the damper are the "amp" envelope: a decay and a
// release ("decay" and "release" set them too, in seconds)
func Describe(cfg Config) params.Set {
	pos := cfg.Position
	if pos <= 0 {
		pos = 0.25
	}
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	lfoRate := cfg.LFORate
	if lfoRate <= 0 {
		lfoRate = 5
	}
	lfo2Rate := cfg.LFO2Rate
	if lfo2Rate <= 0 {
		lfo2Rate = 1
	}
	s := params.Set{Name: "modal", Params: []params.Param{
		params.C("Mallet", "excitation", Excitations, int(cfg.Excitation)),
		params.K("Mallet", "hardness", 0, 1, cfg.Hardness),
		params.K("Mallet", "position", 0.01, 1, pos),
		params.K("Mallet", "noise", 0, 1, cfg.Noise),

		params.K("Body", "highDamping", 0, 2, cfg.HighDamping, params.Label("high damping")),
		params.K("Body", "inharmonicity", 0, 1, cfg.Inharmonicity),

		params.K("Tremolo", "tremolo", 0, 15, cfg.Tremolo, params.Unit("Hz"), params.Label("rate")),
		params.K("Tremolo", "tremoloDepth", 0, 1, cfg.TremoloDepth, params.Label("depth")),

		params.K("Modulation", "lfoRate", 0.1, 20, lfoRate, params.Exp(), params.Unit("Hz"), params.Label("lfo 1 rate")),
		params.K("Modulation", "lfo2Rate", 0.05, 20, lfo2Rate, params.Exp(), params.Unit("Hz"), params.Label("lfo 2 rate")),

		params.K("Output", "level", 0, 4, level),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "amp", Group: "Envelopes", Stages: "DR", Decay: cfg.Decay * 1000, Release: cfg.Release * 1000},
	}}
	fp, fe := voicefilter.Describe(cfg.Filter, "Filter")
	s.Params = append(s.Params, fp...)
	s.Envelopes = append(s.Envelopes, fe)
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
