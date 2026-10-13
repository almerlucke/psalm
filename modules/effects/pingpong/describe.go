package pingpong

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a ping pong delay set as cfg (see params). The echo time is
// described in beats ("time"), which follows the tempo; "timeMs" is left out: setting the time in
// beats turns it off
func Describe(cfg Config) params.Set {
	highCut := cfg.HighCut
	if highCut <= 0 {
		highCut = 20000 // off: as open as it goes
	}
	return params.Set{Name: "pingpong", Params: []params.Param{
		params.K("Ping pong", "time", 0.0625, 4, max(0.0625, cfg.Time), params.Exp(), params.Unit("beats")),
		params.K("Ping pong", "feedback", 0, maxFeedback, cfg.Feedback),
		params.K("Ping pong", "stereo", 0, 1, cfg.Stereo),
		params.K("Ping pong", "highCut", 500, 20000, highCut, params.Exp(), params.Unit("Hz"), params.Label("high cut")),
		params.K("Ping pong", "lowCut", 20, 2000, max(20, cfg.LowCut), params.Exp(), params.Unit("Hz"), params.Label("low cut")),
		params.K("Ping pong", "mix", 0, 1, cfg.Mix),
	}}
}
