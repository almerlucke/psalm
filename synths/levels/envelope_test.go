package levels

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// TestEnvelopeMessages: every synth with an amplitude envelope takes it by message the same way:
// a long "ampRelease" rings on after the note, a short one doesn't (with a quick attack)
func TestEnvelopeMessages(t *testing.T) {
	skip := map[string]bool{"drumsynth": true, "organ": true, "dx": true}
	for _, e := range expressives(t) {
		if skip[e.name] {
			continue
		}
		name := "ampRelease"
		tail := func(ms float64) float64 {
			fresh()
			v := e.make()
			v.ReceiveMessage(psalm.Message{name: ms, "ampAttack": 5.0, "response": 0.05})
			v.Note(0.3, 1, e.note, psalm.CtrlTick{}, v.Config())
			x := left(v, 1.2)
			sr := v.Config().SampleRate
			sum := 0.0
			for _, s := range x[int(0.5*sr):int(0.9*sr)] {
				sum += s * s
			}
			return math.Sqrt(sum / (0.4 * sr))
		}
		short, long := tail(20), tail(3000)
		if long < 3*short || long == 0 {
			t.Errorf("%s: %s 20 ms leaves %.4f after the note, 3000 ms %.4f", e.name, name, short, long)
		}
	}
}
