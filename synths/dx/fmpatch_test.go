package dx

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// the presets of the FM synth that dx took over: four free operators, the rest off, and they sound
func TestFMPresets(t *testing.T) {
	if len(fmPresets) < 49 {
		t.Fatalf("%d presets of the FM synth", len(fmPresets))
	}
	for name, f := range fmPresets {
		p := f()
		if p.Name != name || p.Routing == nil || p.Routing.Check() != nil {
			t.Errorf("%s: name %q, routing %+v", name, p.Name, p.Routing)
		}
		for i, op := range p.Ops {
			if op.FreeOn != (i < 4) {
				t.Errorf("%s: operator %d exact %v", name, i+1, op.FreeOn)
			}
		}
		v := New(p, 2)
		v.NoteOn(0.8, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
		var peak float32
		for range 40 {
			v.Synthesize()
			for _, s := range v.Output(0).Channels[0] {
				if math.IsNaN(float64(s)) {
					t.Fatalf("%s: NaN", name)
				}
				peak = max(peak, s, -s)
			}
		}
		if peak < 0.01 {
			t.Errorf("%s: silent (peak %g)", name, peak)
		}
	}
	// glide and vibrato came along
	if lead := LeadPreset(); lead.Glide != 30 || lead.VibratoDepth != 0.2 || lead.FreeFeedback != 0.6 {
		t.Errorf("lead: glide %g, vibrato %g, feedback %g", lead.Glide, lead.VibratoDepth, lead.FreeFeedback)
	}
}

// the free operators take their values by message, for the next note
func TestFreeOperatorMessages(t *testing.T) {
	v := New(BellPreset(), 2)
	v.ReceiveMessage(psalm.Message{"op2Ratio": 5.0, "op3Gain": 0.9, "op2Cents": 7.0, "modDepth": 1.5, "freeFeedback": 0.3,
		"op1Attack": 40.0, "ampRelease": 900.0, "fmAlgorithm": 5})
	p := v.Patch()
	if p.Ops[1].Free.Ratio != 5 || p.Ops[2].Free.Level != 0.9 || p.Ops[1].Free.Detune != 7 || p.ModDepth != 1.5 || p.FreeFeedback != 0.3 {
		t.Errorf("free values %+v", p.Ops[1].Free)
	}
	if p.Ops[0].Free.Env.AttackDuration != 40 || p.Ops[0].Free.Env.ReleaseDuration != 900 {
		t.Errorf("the envelope of operator 1: %+v", p.Ops[0].Free.Env)
	}
	if len(p.Routing.Carriers) != 3 { // 4>(3, 2, 1)
		t.Errorf("routing %+v", p.Routing)
	}
}
