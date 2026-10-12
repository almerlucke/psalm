package keyboard

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// brightness plays a note through a keyboard with the wheel routed to the cutoff, at a position:
// the energy of the differences of the samples over the energy (more high partials, higher)
func brightness(t *testing.T, wheel int) float64 {
	p := psalm.New(nil)
	q := midi.NewQueue()
	synth := subtractive.NewPoly(subtractive.LeadPreset(), 4)
	synth.AddTo(p)
	New(q, Config{Routes: map[int][]Route{
		midi.CCModWheel: {{Name: "cutoff", Min: 200, Max: 8000, Exp: true}},
	}}, synth).AddTo(p)
	p.Out(synth)
	q.PushOffset(midi.ControlChangeMsg(1, midi.CCModWheel, wheel), 0)
	q.PushOffset(midi.NoteOnMsg(1, 57, 100), 10)
	var d, e, prev float64
	for range 40 {
		run(p)
		for _, s := range synth.Output(0).Channels[0] {
			x := float64(s)
			d += (x - prev) * (x - prev)
			e += x * x
			prev = x
		}
	}
	if e == 0 {
		t.Fatal("silent")
	}
	return d / e
}

func TestRouteToSynth(t *testing.T) {
	if low, high := brightness(t, 0), brightness(t, 127); high < 4*low {
		t.Errorf("the wheel routed to the cutoff: brightness %.4f down, %.4f up", low, high)
	}
}
