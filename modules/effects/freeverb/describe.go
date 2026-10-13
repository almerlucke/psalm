package freeverb

import "github.com/almerlucke/psalm/params"

// Describe describes the settings of a reverb as it is set now (see params)
func Describe(fv *FreeVerb) params.Set {
	return params.Set{Name: "freeverb", Params: []params.Param{
		params.K("Reverb", "roomSize", 0, 1, (fv.roomsize-offsetroom)/scaleroom, params.Label("room size")),
		params.K("Reverb", "damp", 0, 1, fv.damp/scaledamp),
		params.K("Reverb", "width", 0, 1, fv.width),
		params.K("Reverb", "wet", 0, 1, fv.wet/scalewet),
		params.K("Reverb", "dry", 0, 1, fv.dry/scaledry),
		params.S("Reverb", "mode", fv.mode >= freezemode, params.Label("freeze")),
	}}
}
