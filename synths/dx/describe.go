package dx

import (
	"fmt"
	"strings"

	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
)

// Describe describes the settings of a patch (see params; the dxeditor plugin of the panel is an
// editor made for it, with the algorithm drawn)
func Describe(p Patch) params.Set {
	s := params.Set{Name: "dx", Params: []params.Param{
		params.K("Voice", "algorithm", 1, 32, float64(p.Algorithm+1), params.Step(1)),
	}}
	for _, st := range Settings {
		v, _ := p.Get(st.Name)
		group, label := "Voice", st.Name
		if strings.HasPrefix(st.Name, "lfo") || st.Name == "pitchModSens" {
			group = "LFO"
		}
		if strings.HasPrefix(st.Name, "pitchRate") || strings.HasPrefix(st.Name, "pitchLevel") {
			group = "Pitch envelope"
		}
		if st.Name == "filterType" {
			s.Params = append(s.Params, params.C("Filter", st.Name, voicefilter.TypeNames, int(v), params.Label("type")))
			continue
		}
		if strings.HasPrefix(st.Name, "filter") {
			group, label = "Filter", strings.ToLower(st.Name[6:7])+st.Name[7:]
		}
		if st.Op > 0 {
			group = fmt.Sprintf("Operator %d", st.Op)
			label = strings.ToLower(st.Name[3:4]) + st.Name[4:]
		}
		if st.Max == 1 && (strings.HasSuffix(st.Name, "Fixed") || strings.HasSuffix(st.Name, "Sync")) {
			s.Params = append(s.Params, params.S(group, st.Name, v > 0, params.Label(label)))
			continue
		}
		s.Params = append(s.Params, params.K(group, st.Name, 0, float64(st.Max), v, params.Step(1), params.Label(label)))
	}
	return s
}
