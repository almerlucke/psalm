package vocoder

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a vocoder set as cfg (see params)
func Describe(cfg Config) params.Set {
	formant := cfg.FormantShift
	if formant <= 0 {
		formant = 1
	}
	return params.Set{Name: "vocoder", Params: []params.Param{
		params.K("Vocoder", "attack", 1, 500, max(1, cfg.Attack), params.Exp(), params.Unit("ms")),
		params.K("Vocoder", "release", 5, 2000, max(5, cfg.Release), params.Exp(), params.Unit("ms")),
		params.K("Vocoder", "flatten", 0, 1, cfg.Flatten),
		params.K("Vocoder", "formantShift", 0.5, 2, formant, params.Exp(), params.Label("formant shift")),
		params.K("Vocoder", "sibilance", 0, 1, cfg.Sibilance),
		params.K("Vocoder", "sibilanceFreq", 1000, 16000, max(1000, cfg.SibilanceFreq), params.Exp(), params.Unit("Hz"), params.Label("sibilance freq")),
		params.K("Vocoder", "gain", 0, 4, cfg.Gain),
	}}
}
