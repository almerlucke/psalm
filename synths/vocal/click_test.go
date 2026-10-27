package vocal

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
)

// a voice that jumps to a new vowel every sixteenth note does not click when it does: nothing in what it sings stands out against the sound around it (the third
// difference of the samples, against the median of the stretch of 50 ms it is in)
func TestVowelStepsDoNotClick(t *testing.T) {
	for _, pitch := range []float64{50, 57, 62, 69, 74} {
		psalm.Seed(1)
		v := New(Talkbox())
		v.NoteOn(1, psalm.Message{"pitch": pitch, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
		l, _ := run(v, 4)

		if clicks := countClicks(l); clicks > 0 {
			t.Errorf("pitch %v: %d clicks in 4 seconds of a talking voice", pitch, clicks)
		}
	}
}

// countClicks counts the samples that stand out: the third difference over 0.04 and over 25 times the median of the 50 ms around
func countClicks(l []float64) int {

	third := make([]float64, 0, len(l))
	for i := 3; i < len(l); i++ {
		third = append(third, math.Abs(l[i]-3*l[i-1]+3*l[i-2]-l[i-3]))
	}

	window := int(0.05 * sr())
	clicks := 0

	for from := 0; from+window <= len(third); from += window {
		seg := third[from : from+window]
		sorted := append([]float64(nil), seg...)
		sort.Float64s(sorted)
		median := sorted[len(sorted)/2] + 1e-9

		for _, x := range seg {
			if x > 0.04 && x > 25*median {
				clicks++
			}
		}
	}

	return clicks
}
