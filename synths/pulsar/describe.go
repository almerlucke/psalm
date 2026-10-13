package pulsar

import "github.com/almerlucke/psalm/params"

// Windows are the names of the pulsaret windows, by number
var Windows = []string{"hann", "gaussian", "percussive", "rectangle", "tapered"}

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "pulsar", Params: []params.Param{
		params.K("Pulse", "duty", 0.01, 1, max(0.01, min(cfg.Duty, 1))),
		params.C("Pulse", "window", Windows, int(cfg.Window)),
		params.K("Pulse", "masking", 0, 1, cfg.Masking),
		params.K("Pulse", "jitter", 0, 1, cfg.Jitter),
		params.K("Pulse", "stereoSpread", 0, 1, cfg.StereoSpread, params.Label("pulse spread")),
		params.K("Pulse", "noteSpread", 0, 1, cfg.NoteSpread, params.Label("note spread")),

		params.K("Formant", "formant", 50, 12000, cfg.Formant, params.Exp(), params.Unit("Hz")),
		params.K("Formant", "formant2", 0, 8000, cfg.Formant2, params.Unit("Hz"), params.Label("formant 2")),
		params.K("Formant", "formant2Level", 0, 1, cfg.Formant2Level, params.Label("level 2")),
		params.K("Formant", "formantTracking", 0, 1, cfg.FormantTracking, params.Label("tracking")),
		params.K("Formant", "formantEnvAmount", -6, 6, cfg.FormantEnvAmount, params.Unit("oct"), params.Label("env amount")),
		params.K("Formant", "formantLFORate", 0.01, 20, max(0.01, cfg.FormantLFORate), params.Exp(), params.Unit("Hz"), params.Label("lfo rate")),
		params.K("Formant", "formantLFODepth", 0, 2, cfg.FormantLFODepth, params.Unit("oct"), params.Label("lfo depth")),

		params.K("Pitch", "vibratoRate", 0.1, 20, max(0.1, cfg.VibratoRate), params.Exp(), params.Unit("Hz"), params.Label("vibrato rate")),
		params.K("Pitch", "vibratoDepth", 0, 12, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato depth")),
		params.K("Pitch", "glide", 0, 2000, cfg.Glide, params.Unit("ms")),
		params.K("Pitch", "level", 0, 4, orOne(cfg.Level)),
	}, Envelopes: []params.Envelope{
		params.E("Envelopes", "amp", "amp", cfg.AmpEnv.AttackDuration, cfg.AmpEnv.DecayDuration, cfg.AmpEnv.DecayLevel, cfg.AmpEnv.ReleaseDuration),
		params.E("Envelopes", "formant", "formant", cfg.FormantEnv.AttackDuration, cfg.FormantEnv.DecayDuration, cfg.FormantEnv.DecayLevel, cfg.FormantEnv.ReleaseDuration),
	}}
}

func orOne(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
