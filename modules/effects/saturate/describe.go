package saturate

import "github.com/almerlucke/psalm/params"

// Shapes are the names of the shapes, by number
var Shapes = []string{"soft", "tube", "hard", "fold"}

// Describe describes the settings of a saturation set as cfg (see params)
func Describe(cfg Config) params.Set {
	tone := cfg.Tone
	if tone <= 0 {
		tone = 20000 // off: as open as it goes
	}
	return params.Set{Name: "saturate", Params: []params.Param{
		params.C("Saturate", "shape", Shapes, int(cfg.Shape)),
		params.K("Saturate", "drive", 0, 36, cfg.Drive, params.Unit("dB")),
		params.K("Saturate", "bias", -1, 1, cfg.Bias),
		params.K("Saturate", "tone", 500, 20000, tone, params.Exp(), params.Unit("Hz")),
		params.K("Saturate", "output", -24, 12, cfg.Output, params.Unit("dB")),
		params.K("Saturate", "mix", 0, 1, cfg.Mix),
	}}
}
