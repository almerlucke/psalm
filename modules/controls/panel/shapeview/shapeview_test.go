package shapeview

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/almerlucke/psalm/blocks/shapechain"
)

func TestPlot(t *testing.T) {
	texts := [8]string{"sin | fold m1[1,4]", "bipolar | bogus 3", "", "", "", "", "", ""}
	st := plot(texts, [8]string{}, 0, shapechain.Macros{0.5})
	if st.Used != 1 {
		t.Errorf("%d chains in use, want 1 (B does not read)", st.Used)
	}
	a := st.Chains[0]
	if len(a.Wave) != wavePoints || len(a.Spectrum) != harmonics || len(a.Stages) != 2 || !a.Macros[0] || a.Macros[1] {
		t.Errorf("chain A: %d points, %d harmonics, %d stages, macros %v", len(a.Wave), len(a.Spectrum), len(a.Stages), a.Macros)
	}
	if a.Stages[1].Text != "fold m1[1,4]" || len(a.Stages[1].Curve) != curvePoints || len(a.Stages[1].Wave) == 0 {
		t.Errorf("the fold card: %+v", a.Stages[1])
	}
	// a fold of the macro adds harmonics: the spectrum at m1 = 1 reaches higher than at 0
	hi := func(m float64) int {
		s := plot([8]string{texts[0]}, [8]string{}, 0, shapechain.Macros{m}).Chains[0].Spectrum
		top := 0
		for i, v := range s {
			if v > 0.02 {
				top = i + 1
			}
		}
		return top
	}
	if hi(1) <= hi(0) {
		t.Errorf("the fold adds no harmonics: %d at 1, %d at 0", hi(1), hi(0))
	}
	b := st.Chains[1]
	if b.Err == "" || !strings.Contains(b.Err, "bogus") || len(b.Stages) != 2 || b.Stages[1].Err == "" {
		t.Errorf("chain B should say what is wrong and still show its first stage: %+v", b)
	}
	// the blend: half way between A and B (both read) is the mean of the two
	two := plot([8]string{"sin", "bipolar"}, [8]string{}, 0.5, shapechain.Macros{})
	if two.Used != 2 || len(two.Blend) != wavePoints {
		t.Fatalf("two chains: used %d", two.Used)
	}
	i := 64
	want := (two.Chains[0].Wave[i] + two.Chains[1].Wave[i]) / 2
	if d := two.Blend[i] - want; d > 1e-3 || d < -1e-3 {
		t.Errorf("the blend at %d is %v, want %v", i, two.Blend[i], want)
	}
	// line 2 without chains of its own has the wave of line 1; with its own, its own
	if st.Used2 != 0 || len(st.Blend2) != wavePoints {
		t.Errorf("line 2 without chains: used %d, %d points", st.Used2, len(st.Blend2))
	}
	own := plot([8]string{"sin", "", "", "", "bipolar"}, [8]string{}, 0, shapechain.Macros{})
	if own.Used2 != 1 || own.Blend2[64] == own.Blend[64] {
		t.Errorf("line 2 with a chain of its own: used %d", own.Used2)
	}
	if _, err := json.Marshal(st); err != nil {
		t.Error(err)
	}
}
