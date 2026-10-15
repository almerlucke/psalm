package piano

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func keyOn(p *Piano, id string, pitch float64) {
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOn": id, "amplitude": 0.8,
		"message": psalm.Message{"pitch": pitch, "velocity": 0.8}})
}

func keyOff(p *Piano, id string) {
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOff": id})
}

func run(p *Piano, seconds float64) []float64 {
	var x []float64
	for range int(seconds * sr / float64(p.Config().BufferSize)) {
		p.Synthesize()
		for i := range p.Output(0).Channels[0] {
			x = append(x, float64(p.Output(0).Channels[0][i])+float64(p.Output(0).Channels[1][i]))
		}
	}
	return x
}

func peakOf(x []float64) float64 {
	m := 0.0
	for _, v := range x {
		m = math.Max(m, math.Abs(v))
	}
	return m
}

// a piano that adopts the piano made at a preset is that piano, whatever it did before: pedals,
// keys held, strings ringing along
func TestAdoptIsThePianoMade(t *testing.T) {
	a := NewPoly(Presets["grand"](), 8)
	keyOn(a, "x", 60)
	keyOn(a, "y", 67)
	a.ReceiveMessage(psalm.Message{"sustain": 1.0, "sostenuto": 1.0})
	run(a, 0.4)
	keyOff(a, "x")
	// each in an environment of its own, seeded alike: the random streams the voices take repeat
	var b, want *Piano
	for _, at := range []**Piano{&b, &want} {
		e := psalm.New(nil)
		e.Env().Seed(7)
		e.Within(func() { *at = NewPoly(Presets["muffled"](), 8) })
	}
	if !a.Adopt(b) {
		t.Fatal("a piano doesn't adopt a piano")
	}
	if a.sustain != 0 || a.sostenuto || a.caught != [128]bool{} || a.fading || a.board != b.board || a.bank[0] != b.bank[0] {
		t.Error("the pedals, or what was taken, are not the made piano's")
	}
	for _, v := range a.voices {
		if v.IsActive() || v.Held() || v.sostenuto != &a.caught || v.bridge != a.br {
			t.Fatal("a voice is not the made one's, or is not looking at this piano")
		}
	}
	keyOn(a, "p", 57)
	keyOn(want, "p", 57)
	got, ref := run(a, 1.0), run(want, 1.0)
	n := a.Config().BufferSize
	for i := n; i < len(ref); i++ { // the first buffer has the old sound fading out in it
		if math.Abs(got[i]-ref[i]) > 1e-6 {
			t.Fatalf("sample %d: %v against %v", i, got[i], ref[i])
		}
	}
	keyOff(a, "x") // notes of the old piano
	keyOff(a, "y")
	run(a, 0.1)
}

// what sounds fades out over the buffer after the adopt (no click), no pedal or key is left stuck
func TestAdoptFadesWhatSounds(t *testing.T) {
	a := NewPoly(Presets["grand"](), 8)
	for i, p := range []float64{48, 55, 60, 64, 72} {
		keyOn(a, string(rune('a'+i)), p)
	}
	a.ReceiveMessage(psalm.Message{"sustain": 1.0})
	x := run(a, 0.5)
	last := x[len(x)-1]
	if !a.Adopt(NewPoly(Presets["upright"](), 8)) {
		t.Fatal("no adopt")
	}
	y := run(a, 0.1)
	n := a.Config().BufferSize
	if peakOf(y[:2*n]) == 0 {
		t.Error("what sounded is cut at once")
	}
	if step := math.Abs(y[0] - last); step > 0.2*peakOf(y[:2*n]) {
		t.Errorf("a jump of %v at the adopt", step)
	}
	if end := math.Abs(y[2*n-1]); end > 0.02*peakOf(y[:2*n]) {
		t.Errorf("the fade ends at %v", end)
	}
	if p := peakOf(y[2*n:]); p > 1e-6 {
		t.Errorf("sound left after the fade: %v", p)
	}
	// the pedal is up: a key let go is damped
	keyOn(a, "q", 60)
	run(a, 0.3)
	keyOff(a, "q")
	if tail := peakOf(run(a, 3.0)[44100*2:]); tail > 1e-3 {
		t.Errorf("a key let go still sounds after 2 s: %v", tail)
	}
}

func TestAdoptRefuses(t *testing.T) {
	a := NewPoly(Presets["grand"](), 4)
	if a.Adopt(a) {
		t.Error("a piano adopts itself")
	}
	if a.Adopt(nil) {
		t.Error("a piano adopts nothing")
	}
	mono := Presets["grand"]()
	mono.NumChannels = psalm.Mono
	if a.Adopt(NewPoly(mono, 4)) {
		t.Error("a piano adopts one with other channels")
	}
	if a.Adopt(New(Presets["grand"]())) {
		t.Error("a piano adopts a voice")
	}
}

func TestAdoptDoesNotAllocate(t *testing.T) {
	a := NewPoly(Presets["grand"](), 4)
	var others []*Piano
	for range 7 {
		others = append(others, NewPoly(Presets["bright"](), 4))
	}
	keyOn(a, "x", 60)
	run(a, 0.1)
	i := 0
	if n := testing.AllocsPerRun(5, func() {
		a.Adopt(others[i])
		i++
		a.Synthesize()
	}); n != 0 {
		t.Errorf("%v allocations", n)
	}
}
