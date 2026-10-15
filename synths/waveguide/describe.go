package waveguide

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The "amp" envelope is the bow speeding up (attack), the ring
// of a plucked string (decay) and a string let go (release: the bow's of a bowed string, the hand's
// of a plucked one; "ampRelease" sets both)
func Describe(cfg Config) params.Set {
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	release := cfg.Release
	if cfg.Model == Bow {
		release = cfg.BowRelease
	}
	lfo2 := cfg.LFO2Rate
	if lfo2 <= 0 {
		lfo2 = 2
	}
	set := params.Set{Name: "waveguide", Params: []params.Param{
		params.K("Pluck", "damping", 0, 1, cfg.Damping),
		params.K("Pluck", "brightness", 0, 1, cfg.Brightness),
		params.K("Pluck", "pickPosition", 0, 0.5, cfg.PickPosition, params.Label("position")),

		params.K("Bow", "bowPressure", 0, 1, cfg.Pressure, params.Label("pressure")),
		params.K("Bow", "bowPosition", 0.02, 0.5, max(0.02, min(cfg.BowPosition, 0.5)), params.Label("position")),
		params.K("Bow", "vibratoRate", 0, 12, cfg.VibratoRate, params.Unit("Hz"), params.Label("vibrato rate")),
		params.K("Bow", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato depth")),

		params.K("Output", "bodyMix", 0, 1, cfg.BodyMix, params.Label("body")),
		params.K("Output", "sympathetic", 0, 1, cfg.Sympathetic),
		params.K("Output", "level", 0, 4, level),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "amp", Group: "Envelopes", Stages: "ADR", Attack: cfg.BowAttack * 1000, Decay: cfg.Decay * 1000, Release: release * 1000},
	}}
	fp, fe := voicefilter.Describe(cfg.Filter, "Filter")
	set.Params = append(set.Params, fp...)
	set.Envelopes = append(set.Envelopes, fe)
	m := modmatrix.New(Dests)
	for i, sl := range cfg.Mods {
		m.Set(i, sl)
	}
	for i, e := range cfg.Ext {
		m.SetExt(i, e)
	}
	set.Params = append(set.Params, m.Describe("Modulation")...)
	set.Params = append(set.Params, params.K("Modulation", "lfo2Rate", 0.05, 20, lfo2, params.Exp(), params.Unit("Hz"), params.Label("lfo 2 rate")))
	return set
}
