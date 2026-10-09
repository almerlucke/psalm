package dispatch

import (
	"github.com/almerlucke/psalm"
	"golang.org/x/exp/slices"
)

type EventType int

const (
	Mute EventType = iota
	Unmute
	Message
)

type Event struct {
	Who   psalm.Modular
	When  int
	Which EventType
	What  any
}

// NewEvent to WHO is it addressed? WHEN will it be dispatched? WHICH event type is it?, WHAT is the payload?
func NewEvent(who psalm.Modular, when int, which EventType, what ...any) *Event {
	var w any

	if len(what) > 0 {
		w = what[0]
	}

	return &Event{
		Who:   who,
		When:  when,
		Which: which,
		What:  w,
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

// Restart dispatches the events done again, from the start of the patch (psalm.Restarter)
func (d *Dispatch) Restart() {
	d.events = append(d.done, d.events...)
	d.done = d.done[:0:0]
}

func (d *Dispatch) CtrlTick(tick psalm.CtrlTick) {
	for len(d.events) > 0 {
		event := d.events[0]
		if tick.TickTimestamp >= int64(event.When) {
			switch event.Which {
			case Mute:
				event.Who.SetMuted(true)
			case Unmute:
				event.Who.SetMuted(false)
			case Message:
				event.Who.ReceiveMessage(event.What.(psalm.Message))
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
