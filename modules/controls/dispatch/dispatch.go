package dispatch

import (
	"github.com/almerlucke/psalm"
	"golang.org/x/exp/slices"
)

type EventType int

const (
	Mute EventType = iota
	Unmute
)

type Event struct {
	Module psalm.Modular
	When   int
	Action EventType
}

func NewEvent(m psalm.Modular, when int, action EventType) *Event {
	return &Event{
		Module: m,
		When:   when,
		Action: action,
	}
}

type Dispatch struct {
	*psalm.Module
	events []*Event
	done   []*Event
}

func New(events ...*Event) *Dispatch {
	size := len(events)
	d := &Dispatch{
		Module: psalm.NewControl(),
		events: make([]*Event, 0, size),
		done:   make([]*Event, 0, size),
	}
	d.SetSelf(d)
	d.AddEvents(events...)
	return d
}

func (d *Dispatch) AddEvents(events ...*Event) *Dispatch {
	d.events = append(d.events, events...)
	slices.SortFunc(d.events, func(a, b *Event) int {
		return a.When - b.When
	})
	return d
}

func (d *Dispatch) CtrlTick(tick psalm.CtrlTick) {
	for len(d.events) > 0 {
		event := d.events[0]
		if tick.TickTimestamp >= int64(event.When) {
			switch event.Action {
			case Mute:
				event.Module.SetMuted(true)
			case Unmute:
				event.Module.SetMuted(false)
			default:
				break
			}
			d.events = d.events[1:]
			d.done = append(d.done, event)
			continue
		}
		break
	}
}
