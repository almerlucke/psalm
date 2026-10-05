package dx

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/almerlucke/psalm"
)

// TestSysExBanks loads every bank in assets/dx7 (if there are any) and plays every voice: no NaN,
// no clipping, no DC offset, and every voice finishes after its note is released
func TestSysExBanks(t *testing.T) {
	files, _ := filepath.Glob("../../assets/dx7/*.syx")
	if len(files) == 0 {
		t.Skip("no banks in assets/dx7")
	}

	voices := 0
	for _, file := range files {
		patches, err := LoadSysEx(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}

		for _, p := range patches {
			voices++
			v := New(p, psalm.Mono)
			v.NoteOn(0.5, psalm.Message{"pitch": 60.0, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())

			sr := v.Config().SampleRate
			bs := v.Config().BufferSize
			release := int(0.5*sr) / bs
			var dc float64
			n := 0

			for b := 0; ; b++ {
				if b == release {
					v.NoteOff()
				}
				if b > release && !v.IsActive() {
					break
				}
				if b > int(40*sr)/bs {
					t.Fatalf("%s %q: never finished after release", filepath.Base(file), p.Name)
				}
				v.Synthesize()
				for _, s := range v.Output(0).Channels[0] {
					x := float64(s)
					if math.IsNaN(x) || math.Abs(x) > 1.0 {
						t.Fatalf("%s %q: invalid sample %v", filepath.Base(file), p.Name, x)
					}
					if b < release {
						dc += x
						n++
					}
				}
			}

			if dc /= float64(n); math.Abs(dc) > 0.02 {
				t.Fatalf("%s %q: DC offset %.3f", filepath.Base(file), p.Name, dc)
			}
		}
	}
	t.Logf("played %d voices from %d banks", voices, len(files))
}
