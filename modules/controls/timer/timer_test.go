package timer

import (
	"slices"
	"testing"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/psalm"
)

// bangs keeps the ticks the timer banged at
type bangs struct {
	*psalm.Module
	at   []int64
	tick int64
}

func (b *bangs) Receive(v any, _ int) {
	if psalm.IsBang(v) {
		b.at = append(b.at, b.tick)
	}
}

func TestRestart(t *testing.T) {
	psalm.New(nil)
	tm := New(sequence.NewLoop(4.0, 8.0, 2.0))
	rec := &bangs{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.ValueIn(tm)
	run := func(n int) {
		for range n {
			rec.tick++
			tm.CtrlTick(psalm.CtrlTick{TickTimestamp: rec.tick})
		}
	}
	run(10) // bangs at 1, 5 (4 later), 13 not yet
	if len(rec.at) != 2 || rec.at[1] != 5 {
		t.Fatalf("bangs at the durations: %v", rec.at)
	}
	tm.Restart() // as Psalm.Restart does: the clock goes back to the start too
	rec.at, rec.tick = nil, 0
	run(14)
	if len(rec.at) != 3 || rec.at[0] != 1 || rec.at[1] != 5 || rec.at[2] != 13 {
		t.Fatalf("after a restart the durations should start again from the first: %v", rec.at)
	}
}

// a timer runs from a pattern of 4, 8 and 2 ticks: bangs at 1, 5, 13, 15, 19, 27, 29, ...
func bangsOf(t *testing.T, fromNow bool, ticks ...int64) []int64 {
	t.Helper()
	tm := New(sequence.NewLoop(4.0, 8.0, 2.0))
	if fromNow {
		tm.FromNow()
	}
	rec := &bangs{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.ValueIn(tm)
	for _, k := range ticks {
		rec.tick = k
		tm.CtrlTick(psalm.CtrlTick{TickTimestamp: k})
	}
	return rec.at
}

func span(from, to int64) []int64 {
	var ts []int64
	for k := from; k <= to; k++ {
		ts = append(ts, k)
	}
	return ts
}

func TestFollowsTheClock(t *testing.T) {
	psalm.New(nil)
	all := bangsOf(t, false, span(1, 30)...)
	if !slices.Equal(all, []int64{1, 5, 13, 15, 19, 27, 29}) {
		t.Fatalf("every tick: %v", all)
	}
	// it misses the ticks 6 to 16: it goes on where it would have been, nothing banged in between
	gap := bangsOf(t, false, append(span(1, 5), span(17, 30)...)...)
	if !slices.Equal(gap, []int64{1, 5, 19, 27, 29}) {
		t.Fatalf("after missing ticks it should be in its place: %v", gap)
	}
	// a gap that ends right before a bang: the bang comes on the tick after the gap
	gap = bangsOf(t, false, append(span(1, 5), span(13, 16)...)...)
	if !slices.Equal(gap, []int64{1, 5, 13, 15}) {
		t.Fatalf("a bang due on the first tick after a gap: %v", gap)
	}
	// added while playing: its first tick is 10, it is where it would be from the start
	late := bangsOf(t, false, span(10, 30)...)
	if !slices.Equal(late, []int64{13, 15, 19, 27, 29}) {
		t.Fatalf("a timer added later should line up with the clock: %v", late)
	}
	// FromNow: its first duration starts at its first tick
	now := bangsOf(t, true, span(10, 30)...)
	if !slices.Equal(now, []int64{10, 14, 22, 24, 28}) {
		t.Fatalf("FromNow should start at its first tick: %v", now)
	}
	// the same tick twice does nothing; the clock going back starts it again
	back := bangsOf(t, false, 1, 2, 2, 3, 4, 5, 6, 1, 2, 3, 4, 5)
	if !slices.Equal(back, []int64{1, 5, 1, 5}) {
		t.Fatalf("a clock going back should start it again: %v", back)
	}
}
