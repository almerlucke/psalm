package granulator

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a granulator set as cfg (see params)
func Describe(cfg Config) params.Set {
	buffer := cfg.BufferSeconds
	if buffer <= 0 {
		buffer = DefaultConfig().BufferSeconds
	}
	return params.Set{Name: "granulator", Params: []params.Param{
		params.K("Grains", "density", 0.1, 200, max(0.1, cfg.Density), params.Exp(), params.Unit("/s")),
		params.K("Grains", "grainSize", 5, 1000, max(5, cfg.GrainSize), params.Exp(), params.Unit("ms"), params.Label("size")),
		params.K("Grains", "shape", 0, 1, cfg.Shape),
		params.K("Grains", "jitter", 0, 1, cfg.Jitter),
		params.K("Buffer", "delay", 0, buffer*1000, min(cfg.Delay, buffer*1000), params.Unit("ms")),
		params.K("Buffer", "spray", 0, 1000, cfg.Spray, params.Unit("ms")),
		params.K("Buffer", "feedback", 0, 0.98, cfg.Feedback),
		params.S("Buffer", "freeze", cfg.Freeze),
		params.K("Pitch", "pitch", -24, 24, cfg.Pitch, params.Unit("st")),
		params.K("Pitch", "pitchSpread", 0, 12, cfg.PitchSpread, params.Unit("st"), params.Label("spread")),
		params.K("Pitch", "octave", 0, 1, cfg.Octave),
		params.K("Pitch", "reverse", 0, 1, cfg.Reverse),
		params.K("Output", "stereoSpread", 0, 1, cfg.StereoSpread, params.Label("stereo")),
		params.K("Output", "mix", 0, 1, cfg.Mix),
	}}
}
