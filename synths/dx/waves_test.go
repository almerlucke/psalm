package dx

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func TestWaves(t *testing.T) {
	if len(WaveNames) != 8 {
		t.Fatalf("%d wave names", len(WaveNames))
	}
	for w := range 8 {
		var lo, hi float64
		for i := range 1000 {
			v := wave(w, float64(i)/1000)
			if math.IsNaN(v) || v < -1.0001 || v > 1.0001 {
				t.Fatalf("wave %d at %d: %g", w, i, v)
			}
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
		if hi < 0.9 {
			t.Errorf("wave %s tops at %g", WaveNames[w], hi)
		}
		if w != 0 && w != 4 && w != 6 && w != 7 && lo < -1e-9 {
			t.Errorf("wave %s goes below 0 (%g)", WaveNames[w], lo)
		}
		if w == 0 && math.Abs(wave(0, 0.25)-1) > 1e-9 {
			t.Errorf("the sine at a quarter: %g", wave(0, 0.25))
		}
		// periodic, also for negative phases (a modulated phase)
		if math.Abs(wave(w, 0.3)-wave(w, -0.7)) > 1e-9 {
			t.Errorf("wave %s isn't periodic", WaveNames[w])
		}
	}
}

// a wave of an operator is a setting by name and changes the sound
func TestOperatorWave(t *testing.T) {
	p := Presets["bellPad"]()
	if v, ok := p.Get("op1Wave"); !ok || v != 0 {
		t.Fatalf("op1Wave %v %v", v, ok)
	}
	q := p
	q.Set("op1Wave", 6)
	if v, _ := q.Get("op1Wave"); v != 6 || q.Ops[0].Wave != 6 {
		t.Errorf("op1Wave %v", v)
	}
	render := func(p Patch) (e float64) {
		v := New(p, 2)
		v.NoteOn(0.8, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
		for range 10 {
			v.Synthesize()
			for _, s := range v.Output(0).Channels[0] {
				e += math.Abs(float64(s))
			}
		}
		return
	}
	if a, b := render(p), render(q); a == b || math.IsNaN(b) {
		t.Errorf("a square operator sounds the same: %g %g", a, b)
	}
}
