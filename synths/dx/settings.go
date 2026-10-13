package dx

import (
	"fmt"
	"math"
	"strings"
)

// Setting is a setting of a patch by name: its range (the DX7's own) and where it is
type Setting struct {
	Name string
	Max  int // from 0
	Op   int // the operator (1-6), 0 for the voice
	get  func(p *Patch) *int
	flag func(p *Patch) *bool
}

// Settings are the settings of a patch by name, in the order of the DX7's own edit buttons: the
// voice's ("algorithm" 1-32, "feedback", the LFO, the pitch envelope, "transpose" 0-48 with 24 in
// tune), then each operator's, named "op<N><Setting>" ("op1Coarse", "op3Rate2", "op6Level4")
var Settings = func() []Setting {
	ints := func(name string, max int, get func(p *Patch) *int) Setting {
		return Setting{Name: name, Max: max, get: get}
	}
	bools := func(name string, get func(p *Patch) *bool) Setting {
		return Setting{Name: name, Max: 1, flag: get}
	}
	s := []Setting{
		ints("feedback", 7, func(p *Patch) *int { return &p.Feedback }),
		ints("transpose", 48, func(p *Patch) *int { return &p.Transpose }),
		bools("oscSync", func(p *Patch) *bool { return &p.OscKeySync }),
		ints("lfoWaveform", 5, func(p *Patch) *int { return &p.LFOWaveform }),
		ints("lfoSpeed", 99, func(p *Patch) *int { return &p.LFOSpeed }),
		ints("lfoDelay", 99, func(p *Patch) *int { return &p.LFODelay }),
		ints("lfoPitchDepth", 99, func(p *Patch) *int { return &p.LFOPitchDepth }),
		ints("lfoAmpDepth", 99, func(p *Patch) *int { return &p.LFOAmpDepth }),
		bools("lfoSync", func(p *Patch) *bool { return &p.LFOKeySync }),
		ints("pitchModSens", 7, func(p *Patch) *int { return &p.PitchModSens }),
	}
	for i := range 4 {
		s = append(s, ints(fmt.Sprintf("pitchRate%d", i+1), 99, func(p *Patch) *int { return &p.PitchEGRates[i] }))
	}
	for i := range 4 {
		s = append(s, ints(fmt.Sprintf("pitchLevel%d", i+1), 99, func(p *Patch) *int { return &p.PitchEGLevels[i] }))
	}
	for o := range NumOperators {
		op := func(p *Patch) *Operator { return &p.Ops[o] }
		add := func(name string, max int, get func(op *Operator) *int) {
			s = append(s, Setting{Name: fmt.Sprintf("op%d%s", o+1, name), Max: max, Op: o + 1, get: func(p *Patch) *int { return get(op(p)) }})
		}
		add("Coarse", 31, func(op *Operator) *int { return &op.Coarse })
		add("Fine", 99, func(op *Operator) *int { return &op.Fine })
		add("Detune", 14, func(op *Operator) *int { return &op.Detune })
		add("Level", 99, func(op *Operator) *int { return &op.OutputLevel })
		s = append(s, Setting{Name: fmt.Sprintf("op%dFixed", o+1), Max: 1, Op: o + 1, flag: func(p *Patch) *bool { return &op(p).FixedFreq }})
		add("VelocitySens", 7, func(op *Operator) *int { return &op.VelocitySens })
		add("RateScaling", 7, func(op *Operator) *int { return &op.RateScaling })
		add("AmpModSens", 3, func(op *Operator) *int { return &op.AmpModSens })
		for i := range 4 {
			add(fmt.Sprintf("Rate%d", i+1), 99, func(op *Operator) *int { return &op.EGRates[i] })
		}
		for i := range 4 {
			add(fmt.Sprintf("Level%d", i+1), 99, func(op *Operator) *int { return &op.EGLevels[i] })
		}
		add("BreakPoint", 99, func(op *Operator) *int { return &op.BreakPoint })
		add("LeftDepth", 99, func(op *Operator) *int { return &op.LeftDepth })
		add("RightDepth", 99, func(op *Operator) *int { return &op.RightDepth })
		add("LeftCurve", 3, func(op *Operator) *int { return &op.LeftCurve })
		add("RightCurve", 3, func(op *Operator) *int { return &op.RightCurve })
	}
	return s
}()

var settingsByName = func() map[string]Setting {
	m := map[string]Setting{}
	for _, s := range Settings {
		m[strings.ToLower(s.Name)] = s
	}
	return m
}()

// Set changes a setting by its name (see Settings; "algorithm" is 1-32, the others from 0), the
// value rounded and kept in its range. It says whether the name is a setting
func (p *Patch) Set(name string, value float64) bool {
	if name == "algorithm" {
		p.Algorithm = max(0, min(int(math.Round(value))-1, 31))
		return true
	}
	s, ok := settingsByName[strings.ToLower(name)]
	if !ok {
		return false
	}
	v := max(0, min(int(math.Round(value)), s.Max))
	if s.flag != nil {
		*s.flag(p) = v > 0
	} else {
		*s.get(p) = v
	}
	return true
}

// Get returns a setting by its name (see Set)
func (p *Patch) Get(name string) (float64, bool) {
	if name == "algorithm" {
		return float64(p.Algorithm + 1), true
	}
	s, ok := settingsByName[strings.ToLower(name)]
	if !ok {
		return 0, false
	}
	if s.flag != nil {
		if *s.flag(p) {
			return 1, true
		}
		return 0, true
	}
	return float64(*s.get(p)), true
}
