package shaper

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// OversampleNames are the names of the Oversamples, by number
var OversampleNames = []string{"auto", "off", "2x", "4x", "8x"}

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The chains are text (ChainKeys), not described
func Describe(cfg Config) params.Set {
	l1, l2 := cfg.Line1, cfg.Line2
	over := 0
	for i, f := range Oversamples {
		if f == oversampleFactor(cfg.Oversample) {
			over = i
		}
	}
	amp, morph, shape := cfg.AmpEnv, cfg.MorphEnv, cfg.ShapeEnv
	if amp == (adsr.Setting{}) {
		amp = *adsr.NewSetting(1, 5, 0.8, 200, 1, 300)
	}
	if morph == (adsr.Setting{}) {
		morph = *adsr.NewSetting(1, 5, 0.5, 400, 1, 300)
	}
	if shape == (adsr.Setting{}) {
		shape = *adsr.NewSetting(1, 5, 0.5, 400, 1, 300)
	}
	s := params.Set{Name: "shaper", Params: []params.Param{
		params.K("Morph", "morph", 0, 1, cfg.Morph),
		params.K("Morph", "macro1", 0, 1, cfg.Macros[0], params.Label("macro 1 (m1)")),
		params.K("Morph", "macro2", 0, 1, cfg.Macros[1], params.Label("macro 2 (m2)")),
		params.K("Morph", "macro3", 0, 1, cfg.Macros[2], params.Label("macro 3 (m3)")),
		params.K("Morph", "macro4", 0, 1, cfg.Macros[3], params.Label("macro 4 (m4)")),
		params.K("Morph", "morphAmount", -1, 1, cfg.MorphAmount, params.Label("morph env amount")),
		params.K("Morph", "shapeAmount", -1, 1, cfg.ShapeAmount, params.Label("m1 env amount")),

		params.K("Line 1", "octave1", -4, 4, float64(l1.Octave), params.Step(1), params.Label("octave")),
		params.K("Line 1", "detune1", -1200, 1200, l1.Detune, params.Unit("ct"), params.Label("detune")),
		params.K("Line 1", "level1", 0, 2, orOne(l1.Level), params.Label("level")),
		params.K("Line 1", "morph1", -1, 1, l1.Morph, params.Label("morph offset")),

		params.K("Line 2", "octave2", -4, 4, float64(l2.Octave), params.Step(1), params.Label("octave")),
		params.K("Line 2", "detune2", -1200, 1200, l2.Detune, params.Unit("ct"), params.Label("detune")),
		params.K("Line 2", "level2", 0, 2, l2.Level, params.Label("level (0: off)")),
		params.K("Line 2", "morph2", -1, 1, l2.Morph, params.Label("morph offset")),

		params.C("Voice", "oversample", OversampleNames, over),
		params.K("Voice", "width", 0, 1, cfg.Width),
		params.K("Voice", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato")),
		params.K("Voice", "vibratoRate", 0.1, 12, orDefault(cfg.VibratoRate, 5), params.Exp(), params.Unit("Hz"), params.Label("vibrato rate")),
		params.K("Voice", "lfo1Rate", 0, 20, cfg.LFO1Rate, params.Unit("Hz"), params.Label("lfo 1 rate")),
		params.K("Voice", "lfo2Rate", 0, 20, cfg.LFO2Rate, params.Unit("Hz"), params.Label("lfo 2 rate")),
		params.C("Voice", "lfo2Shape", []string{"sine", "triangle"}, max(0, min(cfg.LFO2Shape, 1)), params.Label("lfo 2 shape")),
		params.K("Voice", "level", 0, 4, orOne(cfg.Level)),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", amp.AttackDuration, amp.DecayDuration, amp.DecayLevel, amp.ReleaseDuration),
		params.E("Envelopes", "morph", "morphEnv", morph.AttackDuration, morph.DecayDuration, morph.DecayLevel, morph.ReleaseDuration),
		params.E("Envelopes", "m1", "shapeEnv", shape.AttackDuration, shape.DecayDuration, shape.DecayLevel, shape.ReleaseDuration),
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

func orDefault(v, def float64) float64 {
	if v <= 0 {
		return def
	}
	return v
}
