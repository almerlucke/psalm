package dispatch

import (
	"fmt"
	"slices"
	"testing"

	"github.com/almerlucke/psalm"
)

// target keeps the messages it gets and its mutes, with the tick they came on
type target struct {
	*psalm.Module
	tick *int64
	log  []string
}

func (t *target) ReceiveMessage(m psalm.Message) {
	t.log = append(t.log, fmt.Sprintf("%d %v", *t.tick, m))
}

func (t *target) SetMuted(m bool) psalm.Modular {
	t.log = append(t.log, fmt.Sprintf("%d muted %v", *t.tick, m))
	return t.Module.SetMuted(m)
}

func setup() (*Dispatch, *target, *int64) {
	psalm.New(nil)
	tick := new(int64)
	tg := &target{Module: psalm.NewControl(), tick: tick}
	tg.SetSelf(tg)
	d := New(
		NewEvent(tg, 5, Mute),
		NewEvent(tg, 10, Message, psalm.Message{"note": 60}),
		NewSetting(tg, 12, psalm.Message{"cutoff": 500.0, "resonance": 0.2}),
		NewEvent(tg, 15, Unmute),
		NewSetting(tg, 18, psalm.Message{"cutoff": 900.0}),
		NewEvent(tg, 20, Message, psalm.Message{"note": 64}),
		NewEvent(tg, 25, Mute),
	)
	return d, tg, tick
}

func run(d *Dispatch, tick *int64, ticks ...int64) {
	for _, k := range ticks {
		*tick = k
		d.CtrlTick(psalm.CtrlTick{TickTimestamp: k})
	}
}

func span(from, to int64) []int64 {
	var ts []int64
	for k := from; k <= to; k++ {
		ts = append(ts, k)
	}
	return ts
}

func TestEveryTick(t *testing.T) {
	d, tg, tick := setup()
	run(d, tick, span(0, 30)...)
	want := []string{"5 muted true", "10 map[note:60]", "12 map[cutoff:500 resonance:0.2]", "15 muted false",
		"18 map[cutoff:900]", "20 map[note:64]", "25 muted true"}
	if !slices.Equal(tg.log, want) {
		t.Fatalf("every event at its tick:\n%v", tg.log)
	}
}

func TestMissedTicks(t *testing.T) {
	d, tg, tick := setup()
	run(d, tick, span(0, 6)...)
	tg.log = nil
	run(d, tick, span(22, 30)...) // ticks 7 to 21 missed
	// the latest mute state and the settings merged come at once, the notes in between don't
	want := []string{"22 map[cutoff:900 resonance:0.2]", "22 muted false", "25 muted true"} // in the order they first came
	if !slices.Equal(tg.log, want) {
		t.Fatalf("after missing ticks only the states as they should be by now:\n%v", tg.log)
	}
}

func TestStartedLate(t *testing.T) {
	d, tg, tick := setup()
	run(d, tick, span(16, 30)...) // added while the patch plays: its first tick is 16
	want := []string{"16 muted false", "16 map[cutoff:500 resonance:0.2]", "18 map[cutoff:900]", "20 map[note:64]", "25 muted true"}
	if !slices.Equal(tg.log, want) {
		t.Fatalf("started late it should set the states before it, and go on from there:\n%v", tg.log)
	}
}

func TestClockGoesBack(t *testing.T) {
	d, tg, tick := setup()
	run(d, tick, span(0, 12)...)
	tg.log = nil
	run(d, tick, 12) // the same tick: nothing
	run(d, tick, span(0, 10)...)
	want := []string{"5 muted true", "10 map[note:60]"}
	if !slices.Equal(tg.log, want) {
		t.Fatalf("the clock going back should dispatch from the start again:\n%v", tg.log)
	}
}
