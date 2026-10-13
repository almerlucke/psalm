package pd

import "github.com/almerlucke/psalm/params"

// Waveforms are the names of the waveforms, by number
var Waveforms = []string{"saw", "square", "pulse", "saw pulse", "reso 1", "reso 2", "reso 3"}

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). "amp" and "dcw" set the envelopes of both lines, they are
// described from line 1
func Describe(cfg Config) params.Set {
	l1, l2 := cfg.Line1, cfg.Line2
	return params.Set{Name: "pd", Params: []params.Param{
		params.C("Line 1", "waveform1", Waveforms, int(l1.Waveform), params.Label("waveform")),
		params.K("Line 1", "dcw1", 0, 1, l1.DCW, params.Label("dcw")),

		params.C("Line 2", "waveform2", Waveforms, int(l2.Waveform), params.Label("waveform")),
		params.K("Line 2", "dcw2", 0, 1, l2.DCW, params.Label("dcw")),
		params.K("Line 2", "detune", -1200, 1200, l2.Detune, params.Unit("ct")),

		params.K("Voice", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato")),
		params.K("Voice", "level", 0, 4, orOne(cfg.Level)),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", l1.DCAEnv.AttackDuration, l1.DCAEnv.DecayDuration, l1.DCAEnv.DecayLevel, l1.DCAEnv.ReleaseDuration),
		params.E("Envelopes", "dcw", "dcw", l1.DCWEnv.AttackDuration, l1.DCWEnv.DecayDuration, l1.DCWEnv.DecayLevel, l1.DCWEnv.ReleaseDuration),
	}}
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
