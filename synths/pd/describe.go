package pd

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// Waveforms are the names of the waveforms, by number
var Waveforms = []string{"saw", "square", "pulse", "saw pulse", "reso 1", "reso 2", "reso 3"}

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). "amp" and "dcw" set the envelopes of both lines, they are
// described from line 1
func Describe(cfg Config) params.Set {
	l1, l2 := cfg.Line1, cfg.Line2
	s := params.Set{Name: "pd", Params: []params.Param{
		params.C("Line 1", "waveform1", Waveforms, int(l1.Waveform), params.Label("waveform")),
		params.K("Line 1", "dcw1", 0, 1, l1.DCW, params.Label("dcw")),
		params.K("Line 1", "detune1", -1200, 1200, l1.Detune, params.Unit("ct"), params.Label("detune")),

		params.C("Line 2", "waveform2", Waveforms, int(l2.Waveform), params.Label("waveform")),
		params.K("Line 2", "dcw2", 0, 1, l2.DCW, params.Label("dcw")),
		params.K("Line 2", "detune", -1200, 1200, l2.Detune, params.Unit("ct")),

		params.K("Voice", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato")),
		params.K("Voice", "vibratoRate", 0.1, 12, orDefault(cfg.VibratoRate, 5), params.Exp(), params.Unit("Hz"), params.Label("vibrato rate")),
		params.K("Voice", "width", 0, 1, cfg.Width),
		params.K("Voice", "sweepRate", 0, 20, cfg.SweepRate, params.Unit("Hz"), params.Label("sweep rate")),
		params.K("Voice", "sweepDepth", 0, 1, cfg.SweepDepth, params.Label("sweep depth")),
		params.K("Voice", "lfo2Rate", 0, 20, cfg.LFO2Rate, params.Unit("Hz"), params.Label("lfo 2 rate")),
		params.C("Voice", "lfo2Shape", []string{"sine", "triangle"}, max(0, min(cfg.LFO2Shape, 1)), params.Label("lfo 2 shape")),
		params.K("Envelopes", "dcoAmount", -24, 24, cfg.DCOAmount, params.Unit("st"), params.Label("dco env amount")),
		params.K("Envelopes", "depthAmount", 0, 1, cfg.DepthAmount, params.Label("depth env amount")),
		params.K("Voice", "level", 0, 4, orOne(cfg.Level)),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", l1.DCAEnv.AttackDuration, l1.DCAEnv.DecayDuration, l1.DCAEnv.DecayLevel, l1.DCAEnv.ReleaseDuration),
		params.E("Envelopes", "dcw", "dcw", l1.DCWEnv.AttackDuration, l1.DCWEnv.DecayDuration, l1.DCWEnv.DecayLevel, l1.DCWEnv.ReleaseDuration),
	}}
	dco, depth := cfg.DCOEnv, cfg.DepthEnv
	if dco == (adsr.Setting{}) {
		dco = defaultEnv()
	}
	if depth == (adsr.Setting{}) {
		depth = defaultEnv()
	}
	s.Envelopes = append(s.Envelopes,
		params.E("Envelopes", "dco", "dco", dco.AttackDuration, dco.DecayDuration, dco.DecayLevel, dco.ReleaseDuration),
		params.E("Envelopes", "depth", "depth", depth.AttackDuration, depth.DecayDuration, depth.DecayLevel, depth.ReleaseDuration))
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

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
