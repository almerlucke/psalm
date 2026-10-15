package vocal

import (
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// Ranges are the names of the voice ranges, by number
var Ranges = []string{"auto", "bass", "tenor", "alto", "soprano"}

// Describe describes the settings of a voice set as cfg: every message that changes one, its range,
// and where it is in cfg (see params)
func Describe(cfg Config) params.Set {
	clevel := cfg.ConsonantLevel
	if clevel <= 0 {
		clevel = 0.5
	}
	level := cfg.Level
	if level == 0 {
		level = 1
	}
	set := params.Set{Name: "vocal", Params: []params.Param{
		params.C("Voice", "range", Ranges, int(cfg.Range)),
		params.K("Voice", "vowel", 0, 4, cfg.Vowel),
		params.K("Voice", "vowelGlide", 0, 2, cfg.VowelGlide, params.Unit("s"), params.Label("vowel glide")),
		params.K("Voice", "brightness", 0, 1, cfg.Brightness),
		params.K("Voice", "breath", 0, 1, cfg.Breath),

		params.K("Vibrato", "vibratoRate", 0, 12, cfg.VibratoRate, params.Unit("Hz"), params.Label("rate")),
		params.K("Vibrato", "vibratoDepth", 0, 2, cfg.VibratoDepth, params.Unit("st"), params.Label("depth")),

		params.C("Consonant", "consonant", ConsonantNames, int(cfg.Consonant)),
		params.K("Consonant", "consonantLevel", 0, 1, clevel, params.Label("level")),

		params.K("Output", "level", 0, 4, level),
	}, Envelopes: []params.Envelope{
		{Prefix: "amp", Label: "amp", Group: "Envelopes", Stages: "AR", Attack: cfg.Attack * 1000, Release: cfg.Release * 1000},
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
	return set
}
