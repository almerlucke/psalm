package gain

import "github.com/almerlucke/psalm/params"

// Describe describes the setting of a gain as it is set now (see params)
func Describe(g *Gain) params.Set {
	return params.Set{Name: "gain", Params: []params.Param{
		params.K("Gain", "volume", -60, 12, max(-60, min(g.Volume(), 12)), params.Unit("dB")),
	}}
}
