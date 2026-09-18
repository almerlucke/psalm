package x

import (
	"math"

	"github.com/almerlucke/genny"
)

type TimedGenerator[T any] interface {
	genny.Generator[T]
	SetTimestamp(int64)
}

type timedSelectorSlot[T any] struct {
	t int64
	g genny.Generator[T]
}

type TimedSelector[T any] struct {
	slots    []*timedSelectorSlot[T]
	selected genny.Generator[T]
	loop     bool
	until    int64
}

func NewTimedSelector[T any](loop bool, until int64, timeAndGens ...any) *TimedSelector[T] {
	var slots []*timedSelectorSlot[T]

	n := len(timeAndGens)

	for i := 0; i < n; i += 2 {
		var t int64
		switch tv := timeAndGens[i].(type) {
		case int64:
			t = tv
		case int:
			t = int64(tv)
		case float64:
			t = int64(tv)
		}
		g := timeAndGens[i+1].(genny.Generator[T])
		slots = append(slots, &timedSelectorSlot[T]{
			t: t,
			g: g,
		})
	}

	sel := &TimedSelector[T]{
		slots: slots,
		loop:  loop,
		until: until,
	}

	sel.SetTimestamp(0)

	return sel
}

func (s *TimedSelector[T]) SetTimestamp(t int64) {
	if s.loop {
		t = t % s.until
	}

	n := len(s.slots)
	nm1 := n - 1
	for i := range n {
		var start int64
		var end = int64(math.MaxInt64)

		start = s.slots[i].t
		if i < nm1 {
			end = s.slots[i+1].t
		}

		if start <= t && t < end {
			s.selected = s.slots[i].g
			break
		}
	}
}

func (s *TimedSelector[T]) Generate() T {
	var v T

	if s.selected == nil {
		return v
	}

	return s.selected.Generate()
}

func (s *TimedSelector[T]) Continuous() bool {
	if s.selected == nil {
		return false
	}

	return s.selected.Continuous()
}

func (s *TimedSelector[T]) Done() bool {
	if s.selected == nil {
		return false
	}

	return s.selected.Done()
}

func (s *TimedSelector[T]) Reset() {
	if s.selected != nil {
		s.selected.Reset()
	}
}
