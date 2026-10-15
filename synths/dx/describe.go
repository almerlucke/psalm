package dx

import (
	"fmt"
	"strings"

	"github.com/almerlucke/psalm/blocks/modmatrix"
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
		if st.Max == 1 && (strings.HasSuffix(st.Name, "Fixed") || strings.HasSuffix(st.Name, "Sync") || strings.HasSuffix(st.Name, "Exact")) {
			s.Params = append(s.Params, params.S(group, st.Name, v > 0, params.Label(label)))
			continue
		}
		s.Params = append(s.Params, params.K(group, st.Name, 0, float64(st.Max), v, params.Step(1), params.Label(label)))
	}
	// the exact values of the operators (by message, from the next note; they play when the operator is
	// exact), and what goes with them
	for i, op := range p.Ops {
		g, k, f := fmt.Sprintf("Operator %d", i+1), fmt.Sprintf("op%d", i+1), op.Free
		s.Params = append(s.Params,
			params.K(g, k+"Ratio", 0.125, 32, max(0.125, f.Ratio), params.Exp(), params.Label("exact ratio")),
			params.K(g, k+"Cents", -100, 100, f.Detune, params.Unit("ct"), params.Label("exact detune")),
			params.K(g, k+"Gain", 0, 2, f.Level, params.Label("exact gain")),
		)
	}
	{
		s.Params = append(s.Params,
			params.K("Exact", "modDepth", 0.05, 4, max(0.05, p.ModDepth, 1*boolf(p.ModDepth == 0)), params.Exp(), params.Label("mod depth")),
			params.K("Exact", "freeFeedback", 0, 1, p.FreeFeedback, params.Label("feedback")),
			params.K("Exact", "glide", 0, 2000, p.Glide, params.Unit("ms")),
			params.K("Exact", "vibratoRate", 0.1, 20, max(0.1, p.VibratoRate), params.Exp(), params.Unit("Hz"), params.Label("vibrato rate")),
			params.K("Exact", "vibratoDepth", 0, 2, p.VibratoDepth, params.Unit("st"), params.Label("vibrato depth")),
			params.K("Exact", "vibratoDelay", 0, 3000, p.VibratoDelay, params.Unit("ms"), params.Label("vibrato delay")),
		)
	}

	// the modulation matrix
	m := modmatrix.New(Dests)
	for i, sl := range p.Mods {
		m.Set(i, sl)
	}
	for i, e := range p.Ext {
		m.SetExt(i, e)
	}
	s.Params = append(s.Params, params.K("Modulation", "lfo2Rate", 0, 40, p.LFO2Rate, params.Unit("Hz"), params.Label("lfo 2 rate")))
	s.Params = append(s.Params, m.Describe("Modulation")...)
	return s
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
