package waveguide

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params). The "amp" envelope is the bow speeding up (attack), the ring
// of a plucked string or a drum (decay) and a string let go (release: the bow's of a bowed string, the
// hand's of a plucked one; "ampRelease" sets both). A pipe has the breath's attack and release. What a
// voice describes first depends on its model: the pluck and bow, the membrane or the wind
func Describe(cfg Config) params.Set {
	level := cfg.Level
	if level <= 0 {
		level = 1
	}
	release := cfg.Release
	if cfg.Model == Bow || cfg.Model == Pipe {
		release = cfg.BowRelease
	}
	lfo2 := cfg.LFO2Rate
	if lfo2 <= 0 {
		lfo2 = 2
	}
	var shape []params.Param
	env := params.Envelope{Prefix: "amp", Label: "amp", Group: "Envelopes", Stages: "ADR", Attack: cfg.BowAttack * 1000, Decay: cfg.Decay * 1000, Release: release * 1000}
	switch cfg.Model {
	case Membrane:
		env.Stages = "DR"
		shape = []params.Param{
			params.K("Membrane", "tension", 0, 1, cfg.Tension),
			params.K("Membrane", "damping", 0, 1, cfg.Damping),
			params.K("Membrane", "strike", 0, 1, cfg.Strike),
			params.K("Membrane", "hardness", 0, 1, cfg.Hardness),
			params.K("Membrane", "snare", 0, 1, cfg.Snare),
		}
	case Pipe:
		env.Stages = "AR"
		shape = []params.Param{
			params.K("Wind", "breath", 0, 1, cfg.Breath),
			params.K("Wind", "noise", 0, 1, cfg.Noise),
			params.K("Wind", "embouchure", 0, 1, cfg.Embouchure),
			params.S("Wind", "reed", cfg.Reed),
			params.K("Wind", "damping", 0, 1, cfg.Damping),
			params.K("Wind", "vibratoRate", 0, 12, cfg.VibratoRate, params.Unit("Hz"), params.Label("vibrato rate")),
			params.K("Wind", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato depth")),
		}
	default:
		shape = []params.Param{
			params.K("Pluck", "damping", 0, 1, cfg.Damping),
			params.K("Pluck", "brightness", 0, 1, cfg.Brightness),
			params.K("Pluck", "pickPosition", 0, 0.5, cfg.PickPosition, params.Label("position")),
			params.K("Pluck", "pluckWidth", 0, 0.2, cfg.PluckWidth, params.Label("width")),
			params.K("Pluck", "edge", 0, 1, cfg.Edge),
			params.K("Pluck", "plectrum", 0, 1, cfg.Plectrum),
			params.K("Pluck", "harmonic", 0, 0.5, cfg.Harmonic),
			params.K("Pluck", "fretNoise", 0, 1, cfg.FretNoise, params.Label("fret noise")),
			params.K("Pluck", "fretBuzz", 0, 1, cfg.FretBuzz, params.Label("fret buzz")),
			params.K("Pluck", "fretPosition", 0.05, 0.98, fretPosition(cfg), params.Label("fret position")),
			params.K("Pluck", "slide", 0, 2, cfg.Slide, params.Unit("s")),

			params.K("String", "inharmonicity", 0, 0.005, cfg.Inharmonicity),
			params.K("String", "polarization", 0, 1, cfg.Polarization),
			params.K("String", "tensionMod", 0, 1, cfg.TensionMod, params.Label("tension")),
			params.K("String", "bodyCoupling", 0, 1, cfg.BodyCoupling, params.Label("body coupling")),

			params.C("Bow", "bowModel", []string{"table", "friction"}, min(1, max(0, cfg.BowModel)), params.Label("model")),
			params.K("Bow", "bowPressure", 0, 1, cfg.Pressure, params.Label("pressure")),
			params.K("Bow", "bowPosition", 0.02, 0.5, max(0.02, min(cfg.BowPosition, 0.5)), params.Label("position")),
			params.K("Bow", "bowForce", 0, 1, cfg.BowForce, params.Label("force")),
			params.K("Bow", "bowSpeed", 0, 1, cfg.BowSpeed, params.Label("speed")),
			params.K("Bow", "bowNoise", 0, 1, cfg.BowNoise, params.Label("rosin")),
			params.K("Bow", "bowReverse", 0, 4, cfg.BowReverse, params.Unit("s"), params.Label("stroke")),
			params.K("Bow", "vibratoRate", 0, 12, cfg.VibratoRate, params.Unit("Hz"), params.Label("vibrato rate")),
			params.K("Bow", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("vibrato depth")),
		}
	}
	set := params.Set{Name: "waveguide", Params: append(shape,
		params.K("Output", "bodyMix", 0, 1, cfg.BodyMix, params.Label("body")),
		params.K("Output", "sympathetic", 0, 1, cfg.Sympathetic),
		params.K("Output", "level", 0, 4, level),
	), Envelopes: []params.Envelope{env}}
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
