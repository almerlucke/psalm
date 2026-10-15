package westcoast

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/params"
)

// ModWaves are the names of the modulator waves, by number
var ModWaves = []string{"sine", "triangle", "saw", "square"}

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
	ratio2 := cfg.Osc2ModRatio
	if ratio2 <= 0 {
		ratio2 = 1
	}
	color := cfg.Color
	if color <= 0 {
		color = 0.8
	}
	cycleRate := cfg.CycleRate
	if cycleRate <= 0 {
		cycleRate = 2
	}
	lfo2 := cfg.LFO2Rate
	if lfo2 <= 0 {
		lfo2 = 1
	}
	set := params.Set{Name: "westcoast", Params: []params.Param{
		params.C("Oscillator", "modWave", ModWaves, int(cfg.ModWave), params.Label("mod wave")),
		params.K("Oscillator", "modRatio", 0.125, 8, ratio, params.Exp(), params.Label("mod ratio")),
		params.K("Oscillator", "fm", 0, 4, cfg.FM),
		params.K("Oscillator", "am", 0, 1, cfg.AM),

		params.K("Osc 2", "osc2Mix", 0, 1, cfg.Osc2Mix, params.Label("mix")),
		params.K("Osc 2", "osc2Detune", -24, 24, cfg.Osc2Detune, params.Unit("st"), params.Label("detune")),
		params.C("Osc 2", "osc2ModWave", ModWaves, int(cfg.Osc2ModWave), params.Label("mod wave")),
		params.K("Osc 2", "osc2ModRatio", 0.125, 8, ratio2, params.Exp(), params.Label("mod ratio")),

		params.K("Timbre", "fold", 0, 1, cfg.Fold),
		params.K("Timbre", "symmetry", -1, 1, cfg.Symmetry),

		params.K("Gate", "resonance", 0, 1, cfg.Resonance),
		params.K("Gate", "response", 0.01, 4, response, params.Exp(), params.Unit("s")),
		params.K("Gate", "color", 0, 1, color),
		params.K("Gate", "strike", 0, 1, cfg.Strike),
		params.K("Gate", "strikeDecay", 0, 2, cfg.StrikeDecay, params.Unit("s"), params.Label("strike decay")),

		params.S("Function", "cycle", cfg.Cycle),
		params.K("Function", "cycleRate", 0.05, 40, cycleRate, params.Exp(), params.Unit("Hz"), params.Label("cycle rate")),
		params.K("Function", "lfo2Rate", 0.05, 40, lfo2, params.Exp(), params.Unit("Hz"), params.Label("lfo 2 rate")),
		params.K("Function", "lfo2Sync", 0, 32, cfg.LFO2Sync, params.Unit("beats"), params.Label("lfo 2 beats")),

		params.K("Output", "level", 0, 10, level),
	}}
	m := modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		m.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		m.SetExt(i, e)
	}
	set.Params = append(set.Params, m.Describe("Modulation")...)
	set.Envelopes = []params.Envelope{
		{Prefix: "amp", Label: "function", Group: "Envelopes", Stages: "AD", Attack: cfg.Attack * 1000, Decay: cfg.Decay * 1000},
	}
	return set
}
