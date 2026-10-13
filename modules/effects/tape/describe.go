package tape

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a tape echo set as cfg (see params). The echo time is
// described in beats ("time"), which follows the tempo; "timeMs" is left out: setting the time in
// beats turns it off
func Describe(cfg Config) params.Set {
	drive, highCut := cfg.Drive, cfg.HighCut
	if drive == 0 {
		drive = 3 // 0 is a gentle 3 dB
	}
	if highCut <= 0 {
		highCut = 20000 // off: as open as it goes
	}
	return params.Set{Name: "tape", Params: []params.Param{
		params.K("Tape", "time", 0.0625, 4, max(0.0625, cfg.Time), params.Exp(), params.Unit("beats")),
		params.K("Tape", "feedback", 0, maxFeedback, cfg.Feedback),
		params.K("Tape", "wow", 0, 1, cfg.Wow),
		params.K("Tape", "flutter", 0, 1, cfg.Flutter),
		params.K("Tape", "drive", 1, 24, drive, params.Unit("dB")),
		params.K("Tape", "highCut", 500, 20000, highCut, params.Exp(), params.Unit("Hz"), params.Label("high cut")),
		params.K("Tape", "lowCut", 20, 2000, max(20, cfg.LowCut), params.Exp(), params.Unit("Hz"), params.Label("low cut")),
		params.K("Tape", "mix", 0, 1, cfg.Mix),
	}}
}
