package fm

import (
	"fmt"

	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The envelopes are the operators' ("op1".."op4"); "amp", the
// envelope of the carriers of the algorithm, is a shortcut over those and left out
func Describe(cfg Config) params.Set {
	algorithms := make([]string, len(Algorithms))
	for i, a := range Algorithms {
		algorithms[i] = a.Name
	}
	s := params.Set{Name: "fm", Params: []params.Param{
		params.C("Voice", "algorithm", algorithms, cfg.Algorithm),
		params.K("Voice", "feedback", 0, 1, cfg.Feedback),
		params.K("Voice", "modDepth", 0, 4, orOne(cfg.ModDepth), params.Label("mod depth")),
		params.K("Voice", "glide", 0, 2000, cfg.Glide, params.Unit("ms")),
		params.K("Voice", "level", 0, 4, orOne(cfg.Level)),

		params.K("Vibrato", "vibratoRate", 0.1, 20, max(0.1, cfg.VibratoRate), params.Exp(), params.Unit("Hz"), params.Label("rate")),
		params.K("Vibrato", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("depth")),
		params.K("Vibrato", "vibratoDelay", 0, 3000, cfg.VibratoDelay, params.Unit("ms"), params.Label("delay")),
	}}
	for i, op := range cfg.Ops {
		g, k := fmt.Sprintf("Op %d", i+1), fmt.Sprintf("op%d", i+1)
		s.Params = append(s.Params,
			params.K(g, k+"Ratio", 0.125, 32, orOne(op.Ratio), params.Exp(), params.Label("ratio")),
			params.K(g, k+"Fixed", 0, 10000, op.Fixed, params.Unit("Hz"), params.Label("fixed")),
			params.K(g, k+"Detune", -100, 100, op.Detune, params.Unit("ct"), params.Label("detune")),
			params.K(g, k+"Level", 0, 2, op.Level, params.Label("level")),
			params.K(g, k+"VelocitySens", 0, 1, op.VelocitySens, params.Label("velocity")),
		)
		s.Envelopes = append(s.Envelopes, params.E(g, fmt.Sprintf("op %d", i+1), k,
			op.Env.AttackDuration, op.Env.DecayDuration, op.Env.DecayLevel, op.Env.ReleaseDuration))
	}
	fp, fe := voicefilter.Describe(cfg.Filter, "Filter")
	s.Params = append(s.Params, fp...)
	s.Envelopes = append(s.Envelopes, fe)
	return s
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
