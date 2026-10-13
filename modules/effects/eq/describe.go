package eq

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of an EQ set as cfg (see params)
func Describe(cfg Config) params.Set {
	lowCut, highCut := cfg.LowCut, cfg.HighCut
	if lowCut <= 0 {
		lowCut = 20 // off: as low as it goes
	}
	if highCut <= 0 || highCut > 20000 {
		highCut = 20000 // off: as open as it goes
	}
	q := func(q float64) float64 {
		if q <= 0 {
			return 1
		}
		return q
	}
	return params.Set{Name: "eq", Params: []params.Param{
		params.K("EQ", "lowCut", 20, 1000, max(20, lowCut), params.Exp(), params.Unit("Hz"), params.Label("low cut")),
		params.K("EQ", "lowFreq", 20, 1000, max(20, cfg.Low.Freq), params.Exp(), params.Unit("Hz"), params.Label("low")),
		params.K("EQ", "lowGain", -18, 18, cfg.Low.Gain, params.Unit("dB"), params.Label("low gain")),
		params.K("EQ", "mid1Freq", 40, 18000, max(40, cfg.Mid1.Freq), params.Exp(), params.Unit("Hz"), params.Label("mid 1")),
		params.K("EQ", "mid1Gain", -18, 18, cfg.Mid1.Gain, params.Unit("dB"), params.Label("mid 1 gain")),
		params.K("EQ", "mid1Q", 0.1, 10, q(cfg.Mid1.Q), params.Exp(), params.Label("mid 1 Q")),
		params.K("EQ", "mid2Freq", 40, 18000, max(40, cfg.Mid2.Freq), params.Exp(), params.Unit("Hz"), params.Label("mid 2")),
		params.K("EQ", "mid2Gain", -18, 18, cfg.Mid2.Gain, params.Unit("dB"), params.Label("mid 2 gain")),
		params.K("EQ", "mid2Q", 0.1, 10, q(cfg.Mid2.Q), params.Exp(), params.Label("mid 2 Q")),
		params.K("EQ", "highFreq", 1000, 18000, max(1000, cfg.High.Freq), params.Exp(), params.Unit("Hz"), params.Label("high")),
		params.K("EQ", "highGain", -18, 18, cfg.High.Gain, params.Unit("dB"), params.Label("high gain")),
		params.K("EQ", "highCut", 1000, 20000, max(1000, highCut), params.Exp(), params.Unit("Hz"), params.Label("high cut")),
		params.K("EQ", "output", -24, 24, cfg.Output, params.Unit("dB")),
	}}
}
