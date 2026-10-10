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
	for range 300 {
		l.CtrlTick(psalm.CtrlTick{})
	}
	start := rec.got[:3]
	l.Restart()
	rec.got = rec.got[:0:0]
	for range 3 {
		l.CtrlTick(psalm.CtrlTick{})
	}
	for i := range 3 {
		if rec.got[i] != start[i] {
			t.Fatalf("after a restart the LFO should start from its starting phase: %v, not %v", rec.got, start)
		}
	}
}
