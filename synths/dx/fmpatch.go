package dx

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
)

// The presets of the FM synth (four sine operators, ADSR envelopes) that dx took over are patches of
// free operators (Operator.Free): the same sound, with the 0-99 values of the DX7 set to the nearest
// they can be, for the editor and the SysEx. fmOp and fmPatch make them; presets_fm.go lists them.

// fmOp is an operator of a patch of the FM synth
type fmOp struct {
	Ratio, Level, Velocity, Detune, Fixed float64
	Env                                   *adsr.Setting
}

// fmExtras are the settings of such a patch beyond its operators
type fmExtras struct {
	Glide, VibratoRate, VibratoDepth, VibratoDelay, ModDepth float64
}

// fmAlgorithms are the routings of the four operators: which operator modulates which (the modulator
// first, from 0), and the carriers. The feedback is always the fourth operator into itself
var fmAlgorithms = []Algorithm{
	{Mods: [][2]int{{3, 2}, {2, 1}, {1, 0}}, Carriers: []int{0}},       // 4>3>2>1
	{Mods: [][2]int{{3, 1}, {2, 1}, {1, 0}}, Carriers: []int{0}},       // (4+3)>2>1
	{Mods: [][2]int{{2, 1}, {1, 0}, {3, 0}}, Carriers: []int{0}},       // (4 + 3>2)>1
	{Mods: [][2]int{{3, 2}, {2, 0}, {1, 0}}, Carriers: []int{0}},       // (4>3 + 2)>1
	{Mods: [][2]int{{3, 2}, {1, 0}}, Carriers: []int{0, 2}},            // 4>3, 2>1
	{Mods: [][2]int{{3, 2}, {3, 1}, {3, 0}}, Carriers: []int{0, 1, 2}}, // 4>(3, 2, 1)
	{Mods: [][2]int{{3, 2}}, Carriers: []int{0, 1, 2}},                 // 4>3, 2, 1
	{Mods: nil, Carriers: []int{0, 1, 2, 3}},                           // 4, 3, 2, 1
}

// fmPatch makes a patch of the FM synth: its name, the routing (a number of fmAlgorithms), the
// feedback of the fourth operator (0-1), the level of the whole, the four operators and the rest
func fmPatch(name string, algorithm int, feedback, level float64, extras fmExtras, ops [4]fmOp) Patch {
	p := InitVoice()
	p.Name = name
	route := fmAlgorithms[algorithm]
	route.FeedbackFrom, route.FeedbackTo = 3, 3
	p.Routing = &route
	p.Algorithm = 0
	p.Level = level
	p.FreeFeedback = feedback
	p.Feedback = int(math.Round(feedback * 7))
	p.Glide, p.VibratoRate, p.VibratoDepth, p.VibratoDelay, p.ModDepth =
		extras.Glide, extras.VibratoRate, extras.VibratoDepth, extras.VibratoDelay, extras.ModDepth
	carrier := map[int]bool{}
	for _, c := range route.Carriers {
		carrier[c] = true
	}
	for i := range p.Ops {
		p.Ops[i].OutputLevel = 0
	}
	for i, o := range ops {
		op := &p.Ops[i]
		ratio := o.Ratio
		if ratio <= 0 {
			ratio = 1
		}
		op.FreeOn = true
		op.Free = FreeOp{Ratio: ratio, Fixed: o.Fixed, Detune: o.Detune, Level: o.Level, VelocitySens: o.Velocity, Env: *o.Env}
		// the 0-99 values nearest to it
		op.OutputLevel = nearestLevel(o.Level, carrier[i])
		op.VelocitySens = int(math.Round(o.Velocity * 7))
		op.Detune = max(0, min(14, 7+int(math.Round(o.Detune/1.5))))
		if o.Fixed > 0 {
			op.FixedFreq = true
			decade := max(0, min(3, int(math.Floor(math.Log10(o.Fixed)))))
			op.Coarse = decade
			op.Fine = max(0, min(99, int(math.Round(100*(math.Log10(o.Fixed)-float64(decade))))))
		} else {
			op.Coarse, op.Fine = nearestRatio(ratio)
		}
		op.EGRates, op.EGLevels = nearestEnvelope(o.Env)
	}
	return p
}

// nearestLevel is the output level (0-99) with the amplitude of a level: a carrier's level, or the
// index of a modulator over the index of a full modulator
func nearestLevel(level float64, carrier bool) int {
	amp := level
	if !carrier {
		amp = level / modIndex
	}
	if amp <= 0 {
		return 0
	}
	return max(0, min(99, int(math.Round(99+math.Log10(math.Min(amp, 1))/0.0375))))
}

// nearestRatio is the coarse and fine values of a ratio (0.5 to 31.99)
func nearestRatio(r float64) (coarse, fine int) {
	if r < 1 {
		return 0, max(0, min(99, int(math.Round((r/0.5-1)*100))))
	}
	coarse = max(1, min(31, int(r)))
	return coarse, max(0, min(99, int(math.Round((r/float64(coarse)-1)*100))))
}

// nearestEnvelope is the rates and levels of an ADSR: the rate of a stage is the one that takes its
// time to cross the part of the 99 steps it crosses
func nearestEnvelope(s *adsr.Setting) (rates, levels [4]int) {
	rate := func(ms, steps float64) int {
		if ms <= 0 {
			return 99
		}
		seconds := ms / 1000
		full := seconds * 99 / math.Max(steps, 1) // the time the whole 0-99 would take at this speed
		return max(0, min(99, int(math.Round(6.5*math.Log2(38/full)))))
	}
	level := func(l float64) int {
		if l <= 0 {
			return 0
		}
		return max(0, min(99, int(math.Round(99+math.Log10(math.Min(l, 1))/0.0375))))
	}
	sustain := level(s.DecayLevel)
	levels = [4]int{99, sustain, sustain, 0}
	rates = [4]int{rate(s.AttackDuration, 99/4), rate(s.DecayDuration, float64(99-sustain)), 50, rate(s.ReleaseDuration, float64(sustain))}
	return
}
