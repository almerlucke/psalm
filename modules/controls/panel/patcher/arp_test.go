package patcher_test

import (
	"slices"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// arpRig is an arpeggiator playing into a synth that records
type arpRig struct {
	t   *testing.T
	arp *patcher.Arp
	rec *played
	ts  int64
}

func newArpRig(t *testing.T) *arpRig {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	t.Cleanup(func() { psalm.PopConfig() })
	r := &arpRig{t: t, arp: patcher.NewArp(), rec: &played{Module: psalm.NewControl()}}
	r.rec.SetSelf(r.rec)
	r.arp.SetTargets([]psalm.Receiver{r.rec})
	return r
}

func (r *arpRig) on(id string, pitch float64) {
	r.arp.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": id, "tick": psalm.CtrlTick{SampleTimestamp: r.ts},
		"amplitude": 0.8, "message": psalm.Message{"pitch": pitch, "velocity": 0.8}})
}

func (r *arpRig) off(id string) {
	r.arp.ReceiveMessage(psalm.Message{"command": "trigger", "noteOff": id, "tick": psalm.CtrlTick{SampleTimestamp: r.ts}})
}

// step is a trigger of the clock: a sixteenth, an eighth of a second
func (r *arpRig) steps(n int) {
	for range n {
		r.ts += 5512
		tick := psalm.CtrlTick{SampleTimestamp: r.ts}
		r.arp.Receive(patcher.Event{Tick: tick, Seconds: 0.125}, 1)
		patcher.ArpLate(r.arp, tick)
	}
}

func (r *arpRig) set(k string, v any) { r.arp.ReceiveMessage(psalm.Message{k: v}) }

func TestArpOrders(t *testing.T) {
	for _, c := range []struct {
		mode    int
		octaves int
		want    []float64
	}{
		{0, 1, []float64{60, 64, 67, 60, 64, 67}},
		{1, 1, []float64{67, 64, 60, 67, 64, 60}},
		{2, 1, []float64{60, 64, 67, 64, 60, 64}},
		{3, 1, []float64{67, 64, 60, 64, 67, 64}},
		{0, 2, []float64{60, 64, 67, 72, 76, 79, 60}},
	} {
		r := newArpRig(t)
		r.set("mode", c.mode)
		r.set("octaves", c.octaves)
		r.on("a", 67)
		r.on("b", 60)
		r.on("c", 64)
		r.steps(len(c.want))
		if !slices.Equal(r.rec.pitches, c.want) {
			t.Errorf("mode %s, %d octaves: %v, want %v", patcher.ArpModes[c.mode], c.octaves, r.rec.pitches, c.want)
		}
	}
}

func TestArpPlayedRandomChordAndGate(t *testing.T) {
	r := newArpRig(t)
	r.set("mode", 4)
	r.on("a", 67)
	r.on("b", 60)
	r.on("c", 64)
	r.steps(4)
	if want := []float64{67, 60, 64, 67}; !slices.Equal(r.rec.pitches, want) {
		t.Errorf("as played: %v, want %v", r.rec.pitches, want)
	}
	if got, want := r.rec.seconds[0], 0.8*0.125; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("a note lasts %g s, want %g (0.8 of a step)", got, want)
	}

	r = newArpRig(t)
	r.set("mode", 6)
	r.on("a", 60)
	r.on("b", 64)
	r.steps(1)
	if len(r.rec.pitches) != 2 {
		t.Errorf("chord: %v, want both notes in one step", r.rec.pitches)
	}

	r = newArpRig(t)
	r.set("mode", 5)
	r.on("a", 60)
	r.on("b", 72)
	r.steps(40)
	seen := map[float64]bool{}
	for _, p := range r.rec.pitches {
		seen[p] = true
	}
	if len(r.rec.pitches) != 40 || len(seen) != 2 {
		t.Errorf("random: %d notes of %v", len(r.rec.pitches), seen)
	}
}

func TestArpReleaseAndHold(t *testing.T) {
	r := newArpRig(t)
	r.on("a", 60)
	r.on("b", 64)
	r.off("b")
	r.steps(3)
	if want := []float64{60, 60, 60}; !slices.Equal(r.rec.pitches, want) {
		t.Errorf("after letting one go: %v, want %v", r.rec.pitches, want)
	}
	r.off("a")
	n := len(r.rec.pitches)
	r.steps(2)
	if len(r.rec.pitches) != n {
		t.Errorf("it plays with no key down")
	}

	// hold: the notes stay after the keys, until new ones
	r = newArpRig(t)
	r.set("hold", 1.0)
	r.on("a", 60)
	r.on("b", 64)
	r.off("a")
	r.off("b")
	r.steps(2)
	if want := []float64{60, 64}; !slices.Equal(r.rec.pitches, want) {
		t.Errorf("held: %v, want %v", r.rec.pitches, want)
	}
	r.on("c", 72)
	r.steps(2)
	if want := []float64{60, 64, 72, 72}; !slices.Equal(r.rec.pitches, want) {
		t.Errorf("a new key replaces the held notes: %v, want %v", r.rec.pitches, want)
	}
}

func TestArpKind(t *testing.T) {
	k := patcher.ArpKind()
	if k.Name != "arp" || len(k.Ins) != 2 || k.Ins[0].Type != patcher.Notes || k.Ins[1].Type != patcher.Trigger || len(k.Outs) != 1 || k.Outs[0].Type != patcher.Notes {
		t.Errorf("ports %+v %+v", k.Ins, k.Outs)
	}
	if !slices.ContainsFunc(patcher.Generators(), func(g patcher.Kind) bool { return g.Name == "arp" }) {
		t.Error("not among the generators")
	}
}

// in a patch: a clock of sixteenths triggers the arp, which plays the notes held into a synth
func TestArpInPatch(t *testing.T) {
	r := newSongRig(t, "A 4")
	clock, arp, syn := r.add("clock"), r.add("arp"), r.add("rec")
	r.pt.SettingsOf(clock).Apply(map[string]float64{"division": 0}) // quarters: 2 a second
	if err := r.pt.SetText(clock, "pattern", "x"); err != nil {
		t.Fatal(err)
	}
	r.connect(patcher.Cable{From: clock, Out: 0, To: arp, In: 1})
	r.connect(patcher.Cable{From: arp, Out: 0, To: syn, In: 0})
	r.pt.Module(arp).ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "k1", "tick": psalm.CtrlTick{},
		"amplitude": 0.8, "message": psalm.Message{"pitch": 60.0, "velocity": 0.8}})
	r.pt.Module(arp).ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "k2", "tick": psalm.CtrlTick{},
		"amplitude": 0.8, "message": psalm.Message{"pitch": 67.0, "velocity": 0.8}})
	r.run(2.4)
	if len(r.rec.pitches) < 4 || r.rec.pitches[0] != 60 || r.rec.pitches[1] != 67 || r.rec.pitches[2] != 60 {
		t.Errorf("played %v, want 60 67 60 67 on the clock", r.rec.pitches)
	}
}
