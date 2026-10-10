package timer

import (
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
	tm.Restart()
	rec.at = nil
	run(14)
	if len(rec.at) != 3 || rec.at[0] != 11 || rec.at[1] != 15 || rec.at[2] != 23 {
		t.Fatalf("after a restart the durations should start again from the first: %v", rec.at)
	}
}
