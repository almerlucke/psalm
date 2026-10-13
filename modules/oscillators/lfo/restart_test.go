package lfo

import (
	"testing"

	"github.com/almerlucke/psalm"
)

type values struct {
	*psalm.Module
	got []float64
}

func (v *values) Receive(x any, _ int) {
	if f, ok := x.(float64); ok {
		v.got = append(v.got, f)
	}
}

func TestRestart(t *testing.T) {
	psalm.New(nil)
	l := New(true, 0.5, 0, 1, Sine)
	rec := &values{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.ValueIn(l)
	tick := int64(0)
	ctrl := func() { l.CtrlTick(psalm.CtrlTick{TickTimestamp: tick}); tick++ }
	for range 300 {
		ctrl()
	}
	start := rec.got[:3]
	l.Restart()
	rec.got, tick = rec.got[:0:0], 0
	for range 3 {
		ctrl()
	}
	for i := range 3 {
		if rec.got[i] != start[i] {
			t.Fatalf("after a restart the LFO should start from its starting phase: %v, not %v", rec.got, start)
		}
	}
}

// at control rate the LFO follows the clock: a gap, or a late start, leaves it where it would be
func TestFollowsTheClock(t *testing.T) {
	psalm.New(nil)
	run := func(ticks ...int64) []float64 {
		l := New(true, 3, 0, 1, Sine)
		rec := &values{Module: psalm.NewControl()}
		rec.SetSelf(rec)
		rec.ValueIn(l)
		for _, k := range ticks {
			l.CtrlTick(psalm.CtrlTick{TickTimestamp: k})
		}
		return rec.got
	}
	var all []int64
	for k := int64(0); k < 50; k++ {
		all = append(all, k)
	}
	every := run(all...)
	gap := run(append(all[:10:10], all[30:]...)...)
	if gap[10] != every[30] || gap[len(gap)-1] != every[49] {
		t.Fatal("after missing ticks the LFO should be where it would have been")
	}
	late := run(all[20:]...)
	if late[0] != every[20] {
		t.Fatal("an LFO starting later should be where it would be from the start")
	}
}
