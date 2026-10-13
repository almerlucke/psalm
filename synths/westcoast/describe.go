package westcoast

import (
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The function generator is the "amp" envelope: an attack and a
// decay ("ampRelease" sets the decay too, "decay" the same in seconds)
func Describe(cfg Config) params.Set {
	ratio := cfg.ModRatio
	if ratio <= 0 {
		ratio = 1
	}
	response := cfg.Response
	if response <= 0 {
		response = 0.15
	}
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	return params.Set{Name: "westcoast", Params: []params.Param{
		params.K("Oscillator", "modRatio", 0.125, 8, ratio, params.Exp(), params.Label("mod ratio")),
		params.K("Oscillator", "fm", 0, 4, cfg.FM),
		params.K("Oscillator", "am", 0, 1, cfg.AM),

		params.K("Timbre", "fold", 0, 1, cfg.Fold),
		params.K("Timbre", "symmetry", -1, 1, cfg.Symmetry),

		params.K("Gate", "resonance", 0, 1, cfg.Resonance),
		params.K("Gate", "response", 0.01, 4, response, params.Exp(), params.Unit("s")),

		params.K("Output", "level", 0, 10, level),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "function", Group: "Envelopes", Stages: "AD", Attack: cfg.Attack * 1000, Decay: cfg.Decay * 1000},
	}}
}
