package piano

import (
	"github.com/almerlucke/psalm/params"
)

// Positions are the names of the places the piano is heard from, by number
var Positions = []string{"player", "studio", "hall"}

// Describe describes the settings of a piano set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The pedals ("sustain", "sostenuto", "soft") are playing, not
// settings, and are not in it
func Describe(cfg Config) params.Set {
	strings := cfg.Strings
	if strings <= 0 {
		strings = 3
	}
	level := cfg.Level
	if level == 0 {
		level = 1
	}
	return params.Set{Name: "piano", Params: []params.Param{
		params.K("Hammer", "hardness", 0, 1, cfg.Hardness),
		params.K("Hammer", "strikePosition", 0.02, 0.5, cfg.StrikePosition, params.Label("strike position")),
		params.K("Hammer", "velocitySens", 0, 1, cfg.VelocitySens, params.Label("velocity")),
		params.K("Hammer", "variation", 0, 1, cfg.Variation),
		params.K("Hammer", "strikeRegister", 0, 1, cfg.StrikeRegister, params.Label("strike by register")),

		params.K("Strings", "stiffness", 0, 3, cfg.Stiffness),
		params.K("Strings", "decay", 0.05, 4, cfg.Decay, params.Exp()),
		params.K("Strings", "damping", 0, 4, cfg.Damping),
		params.K("Strings", "unison", 0, 20, cfg.Unison, params.Unit("ct")),
		params.K("Strings", "strings", 1, 3, float64(strings), params.Step(1)),
		params.K("Strings", "stretch", 0, 2, cfg.Stretch),
		params.K("Strings", "bark", 0, 1, cfg.Bark),
		params.K("Strings", "wound", 0, 1, cfg.Wound, params.Label("wound bass")),
		params.K("Strings", "precursor", 0, 1, cfg.Precursor),
		params.K("Strings", "resonance", 0, 1, cfg.Resonance),
		params.K("Strings", "damperTime", 0.02, 2, max(0.02, cfg.DamperTime), params.Exp(), params.Unit("s"), params.Label("damper time")),

		params.K("Body", "soundboard", 0, 1, cfg.Soundboard),
		params.K("Body", "body", 0, 1, cfg.Body),
		params.K("Body", "bridge", 0, 1, cfg.Bridge),
		params.K("Body", "mechanics", 0, 1, cfg.Mechanics),
		params.K("Body", "keyBed", 0, 1, cfg.KeyBed, params.Label("key bed")),

		params.C("Output", "position", Positions, int(cfg.Position), params.Label("heard from")),
		params.K("Output", "width", 0, 1, cfg.Width),
		params.K("Output", "treble", -24, 12, cfg.Treble, params.Unit("dB")),
		params.K("Output", "level", 0, 32, level),
	}}
}
