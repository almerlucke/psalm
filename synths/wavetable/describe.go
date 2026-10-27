package wavetable

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/params"
)

// FilterModes are the names of the filter modes, by number
var FilterModes = []string{"off", "low pass", "band pass", "high pass"}

// WarpModes are the names of the warp modes, by number
var WarpModes = []string{"off", "bend +", "bend -", "sync", "mirror", "fm b", "fm sine"}

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The table, not a number, is left out
func Describe(cfg Config) params.Set {
	s := params.Set{Name: "wavetable", Params: []params.Param{
		params.K("Position", "position", 0, 1, cfg.Position),
		params.K("Position", "positionEnvAmount", -1, 1, cfg.PositionEnvAmount, params.Label("env amount")),
		params.K("Position", "positionLFORate", 0.01, 20, max(0.01, cfg.PositionLFORate), params.Exp(), params.Unit("Hz"), params.Label("lfo rate")),
		params.K("Position", "positionLFODepth", 0, 1, cfg.PositionLFODepth, params.Label("lfo depth")),
		params.S("Position", "normalize", cfg.Normalize),

		params.K("Oscillator", "unison", 1, MaxUnison, float64(max(1, min(cfg.Unison, MaxUnison))), params.Step(1)),
		params.K("Oscillator", "detune", 0, 100, cfg.Detune, params.Unit("ct")),
		params.K("Oscillator", "spread", 0, 1, cfg.Spread),
		params.K("Oscillator", "subLevel", 0, 1, cfg.SubLevel, params.Label("sub")),

		params.C("Oscillator", "warpMode", WarpModes, int(cfg.WarpMode), params.Label("warp mode")),
		params.K("Oscillator", "warp", 0, 1, cfg.Warp),
		params.C("Oscillator", "oversample", []string{"once", "2x", "4x"}, max(0, min(cfg.Oversample, 2))),

		params.K("Oscillator B", "oscBLevel", 0, 1, cfg.OscBLevel, params.Label("level")),
		params.K("Oscillator B", "oscBPosition", 0, 1, cfg.OscBPosition, params.Label("position")),
		params.K("Oscillator B", "oscBOctave", -3, 3, float64(max(-3, min(cfg.OscBOctave, 3))), params.Step(1), params.Label("octave")),
		params.K("Oscillator B", "oscBSemi", -12, 12, cfg.OscBSemi, params.Unit("st"), params.Label("semitones")),
		params.K("Oscillator B", "oscBDetune", -50, 50, cfg.OscBDetune, params.Unit("ct"), params.Label("detune")),
		params.K("Oscillator B", "oscBUnison", 1, MaxUnison, float64(max(1, min(cfg.OscBUnison, MaxUnison))), params.Step(1), params.Label("unison")),
		params.K("Oscillator B", "oscBSpread", 0, 100, cfg.OscBSpread, params.Unit("ct"), params.Label("spread")),

		params.K("Modulation", "lfo1Rate", 0.01, 20, max(0.01, orOneF(cfg.LFO1Rate)), params.Exp(), params.Unit("Hz"), params.Label("lfo 1 rate")),
		params.K("Modulation", "lfo2Rate", 0.01, 20, max(0.01, orOneF(cfg.LFO2Rate)), params.Exp(), params.Unit("Hz"), params.Label("lfo 2 rate")),

		params.C("Filter", "filter", FilterModes, int(cfg.Filter), params.Label("mode")),
		params.K("Filter", "cutoff", 20, 18000, max(20, cfg.Cutoff), params.Exp(), params.Unit("Hz")),
		params.K("Filter", "resonance", 0, 1, cfg.Resonance),
		params.K("Filter", "filterEnvAmount", -6, 6, cfg.FilterEnvAmount, params.Unit("oct"), params.Label("env amount")),
		params.K("Filter", "keyTrack", 0, 1, cfg.KeyTrack, params.Label("key track")),

		params.K("Pitch", "vibratoRate", 0.1, 20, max(0.1, cfg.VibratoRate), params.Exp(), params.Unit("Hz"), params.Label("vibrato rate")),
		params.K("Pitch", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato depth")),
		params.K("Pitch", "glide", 0, 2000, cfg.Glide, params.Unit("ms")),
		params.K("Pitch", "level", 0, 4, orOne(cfg.Level)),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", cfg.AmpEnv.AttackDuration, cfg.AmpEnv.DecayDuration, cfg.AmpEnv.DecayLevel, cfg.AmpEnv.ReleaseDuration),
		params.E("Envelopes", "filter", "filter", cfg.FilterEnv.AttackDuration, cfg.FilterEnv.DecayDuration, cfg.FilterEnv.DecayLevel, cfg.FilterEnv.ReleaseDuration),
		params.E("Envelopes", "position", "position", cfg.PositionEnv.AttackDuration, cfg.PositionEnv.DecayDuration, cfg.PositionEnv.DecayLevel, cfg.PositionEnv.ReleaseDuration),
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

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
