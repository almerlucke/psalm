package pitchshift

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a pitch shifter as they are now (see params)
func Describe(p *PitchShift) params.Set {
	return params.Set{Name: "pitchshift", Params: []params.Param{
		params.K("Pitch shift", "semitones", -24, 24, p.Semitones(), params.Unit("st")),
		params.K("Pitch shift", "mix", 0, 1, p.Mix()),
	}}
}
