package organ

import (
	"strconv"

	"github.com/almerlucke/psalm/params"
)

// Footages are the names of the drawbars, by number (drawbar1 to drawbar9)
var Footages = []string{"16'", "5 1/3'", "8'", "4'", "2 2/3'", "2'", "1 3/5'", "1 1/3'", "1'"}

// Percussions are the names of the percussion settings, by number
var Percussions = []string{"off", "second", "third"}

// Vibratos are the names of the vibrato scanner settings, by number
var Vibratos = []string{"off", "V1", "V2", "V3", "C1", "C2", "C3"}

// Describe describes the settings of an organ set as cfg: every message that changes one, its
// range, and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	s := params.Set{Name: "organ"}
	for i, f := range Footages {
		s.Params = append(s.Params, params.K("Drawbars", "drawbar"+strconv.Itoa(i+1), 0, 8, cfg.Drawbars[i], params.Step(1), params.Label(f)))
	}
	vibratoHz := cfg.VibratoHz
	if vibratoHz <= 0 {
		vibratoHz = 6.9
	}
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	s.Params = append(s.Params,
		params.C("Percussion", "percussion", Percussions, int(cfg.Percussion), params.Label("harmonic")),
		params.S("Percussion", "percFast", cfg.PercFast, params.Label("fast")),
		params.S("Percussion", "percSoft", cfg.PercSoft, params.Label("soft")),

		params.C("Vibrato", "vibrato", Vibratos, int(cfg.Vibrato), params.Label("scanner")),
		params.K("Vibrato", "vibratoHz", 0.5, 20, vibratoHz, params.Exp(), params.Unit("Hz"), params.Label("speed")),

		params.K("Output", "click", 0, 1, cfg.Click),
		params.K("Output", "drive", 0, 1, cfg.Drive),
		params.K("Output", "level", 0, 4, level),
	)
	return s
}
