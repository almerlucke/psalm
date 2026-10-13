package pan

import "github.com/almerlucke/psalm/params"

// Describe describes the setting of a pan as it is set now (see params)
func Describe(p *Pan) params.Set {
	return params.Set{Name: "pan", Params: []params.Param{
		params.K("Pan", "pan", -1, 1, p.Pan()),
	}}
}
