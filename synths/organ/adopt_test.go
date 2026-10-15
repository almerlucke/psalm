package organ

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func peakOf(x []float64) float64 {
	p := 0.0
	for _, v := range x {
		p = math.Max(p, math.Abs(v))
	}
	return p
}

// an organ that adopts the organ made at a preset is that organ: same sound, whatever it did before
func TestAdoptIsTheOrganMade(t *testing.T) {
	a := New(Presets["jazz"]())
	down(a, "x", 60)
	down(a, "y", 64)
	a.ReceiveMessage(psalm.Message{"sustain": 1.0, "leslie": 1.0})
	render(a, 0.3)
	up(a, "x")
	b, want := New(Presets["warble"]()), New(Presets["warble"]())
	if !a.Adopt(b) {
		t.Fatal("an organ doesn't adopt an organ")
	}
	if a.sustain || len(a.keys) != 0 || len(a.sustained) != 0 || len(a.ids) != 0 || len(a.events) != 0 {
		t.Error("what was held or sustained is still there")
	}
	if a.cfg.Registration() != want.cfg.Registration() || a.les != b.les || a.mods != b.mods || a.phase != want.phase {
		t.Error("the registration is not the preset's")
	}
	down(a, "p", 57)
	down(want, "p", 57)
	got, ref := render(a, 1.0), render(want, 1.0)
	n := a.Config().BufferSize
	for i := n; i < len(ref); i++ { // the first buffer has the old sound fading out in it
		if got[i] != ref[i] {
			t.Fatalf("sample %d: %v against %v", i, got[i], ref[i])
		}
	}
	// a note off for a note of the old organ does nothing
	up(a, "x")
	up(a, "y")
	render(a, 0.1)
}

// what sounds fades out over the buffer after the adopt (no click), and nothing is left stuck
func TestAdoptFadesWhatSounds(t *testing.T) {
	a := New(Presets["full"]())
	for i, p := range []float64{48, 55, 60, 64} {
		down(a, string(rune('a'+i)), p)
	}
	a.ReceiveMessage(psalm.Message{"sustain": 1.0})
	render(a, 0.5)
	last := float64(a.Output(0).Channels[0][a.Config().BufferSize-1])
	if !a.Adopt(New(Presets["flute"]())) {
		t.Fatal("no adopt")
	}
	x := render(a, 0.05)
	n := a.Config().BufferSize
	if p := peakOf(x[:n]); p == 0 {
		t.Error("what sounded is cut at once")
	}
	if step := math.Abs(x[0] - last); step > 0.2*peakOf(x[:n]) {
		t.Errorf("a jump of %v at the adopt (peak %v)", step, peakOf(x[:n]))
	}
	if end := math.Abs(x[n-1]); end > 0.02*peakOf(x[:n]) {
		t.Errorf("the fade ends at %v", end)
	}
	if p := peakOf(x[n:]); p > 1e-9 {
		t.Errorf("sound left after the fade: %v", p)
	}
	// the pedal is up again: a key played and let go dies away
	down(a, "q", 60)
	render(a, 0.2)
	up(a, "q")
	if tail := peakOf(render(a, 1.0)[44100*3/4:]); tail > 1e-4 {
		t.Errorf("a note let go sounds on: %v", tail)
	}
}

type notOrgan struct{ *psalm.Module }

func (*notOrgan) Synthesize() {}

func TestAdoptRefuses(t *testing.T) {
	a := New(Jazz())
	if a.Adopt(a) {
		t.Error("an organ adopts itself")
	}
	if a.Adopt(nil) {
		t.Error("an organ adopts nothing")
	}
	mono := Jazz()
	mono.NumChannels = psalm.Mono
	if a.Adopt(New(mono)) {
		t.Error("an organ adopts one with other channels")
	}
	if a.Adopt(&notOrgan{psalm.NewModule(nil, []int{psalm.Stereo})}) {
		t.Error("an organ adopts what is no organ")
	}
}

func TestAdoptDoesNotAllocate(t *testing.T) {
	a := New(Jazz())
	var others []*Organ
	for range 8 {
		others = append(others, New(Presets["gospel"]()))
	}
	down(a, "x", 60)
	render(a, 0.1)
	i := 0
	if n := testing.AllocsPerRun(5, func() {
		a.Adopt(others[i])
		i++
		a.Synthesize()
	}); n != 0 {
		t.Errorf("%v allocations", n)
	}
}
