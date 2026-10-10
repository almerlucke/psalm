package dispatch

import (
	"maps"
	"slices"

	"github.com/almerlucke/psalm"
)

type EventType int

const (
	Mute EventType = iota
	Unmute
	Message
)

// Event is something dispatched to a module at a tick of the patch (When, compared with the
// TickTimestamp of the control ticks): a mute, an unmute or a message. Mutes and unmutes are states;
// a message is a moment (a note, a bang), unless State is set: then it is a setting (a cutoff), and
// after a gap only the latest counts (see Dispatch)
type Event struct {
	Who   psalm.Modular
	When  int
	Which EventType
	What  any
	State bool // a message that is a setting, not a moment
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

// NewSetting is an event with a message that is a setting (Event.State): after a gap only the latest
// setting counts, merged with the ones before it
func NewSetting(who psalm.Modular, when int, msg psalm.Message) *Event {
	return &Event{Who: who, When: when, Which: Message, What: msg, State: true}
}

// state tells whether an event is a state, of which only the latest counts after a gap
func (e *Event) state() bool { return e.Which != Message || e.State }

// Dispatch dispatches events at their ticks. It follows the clock of its patch: a dispatch that
// misses control ticks (out of the patch for a while) doesn't fire the events it missed all at once
// when the ticks reach it again. The states among them (mutes, unmutes, settings) are set as they
// should be by now, only the latest of every module (no flicker); the moments (messages) are left
// out. A dispatch added while the patch plays does the same with the events before its first tick.
// A restart (or the clock going back) dispatches the events from the start again
type Dispatch struct {
	*psalm.Module
	events []*Event
	done   []*Event
	last   int64 // the tick of the last control tick, 0 for none yet
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
	slices.SortStableFunc(d.events, func(a, b *Event) int { // events at the same tick in the order given
		return a.When - b.When
	})
	return d
}

// Restart dispatches the events done again, from the start of the patch (psalm.Restarter)
func (d *Dispatch) Restart() {
	d.events = append(d.done, d.events...)
	d.done = d.done[:0:0]
	d.last = 0
}

// catchUp passes the events before a tick that were missed: the latest state of every module is
// set (its settings merged, later keys over earlier ones), the moments are left out
func (d *Dispatch) catchUp(before int64) {
	type state struct {
		who  psalm.Modular
		mute bool // a mute state (else settings)
	}
	mutes := map[psalm.Modular]bool{}
	settings := map[psalm.Modular]psalm.Message{}
	var order []state // in the order of their first event
	n := 0
	for _, e := range d.events {
		if int64(e.When) >= before {
			break
		}
		n++
		if !e.state() {
			continue // a moment that is gone
		}
		switch e.Which {
		case Mute, Unmute:
			if _, ok := mutes[e.Who]; !ok {
				order = append(order, state{e.Who, true})
			}
			mutes[e.Who] = e.Which == Mute
		case Message:
			msg, ok := e.What.(psalm.Message)
			if !ok {
				continue
			}
			if _, ok := settings[e.Who]; !ok {
				order = append(order, state{e.Who, false})
				settings[e.Who] = psalm.Message{}
			}
			maps.Copy(settings[e.Who], msg)
		}
	}
	for _, s := range order {
		if s.mute {
			s.who.SetMuted(mutes[s.who])
		} else {
			s.who.ReceiveMessage(settings[s.who])
		}
	}
	d.done = append(d.done, d.events[:n]...)
	d.events = d.events[n:]
}

func (d *Dispatch) CtrlTick(tick psalm.CtrlTick) {
	now := tick.TickTimestamp
	switch {
	case d.last > 0 && now == d.last:
		return // a tick it had
	case d.last > 0 && now < d.last: // the clock went back: from the start
		d.Restart()
		d.catchUp(now)
	case d.last == 0 && now > 1, d.last > 0 && now > d.last+1: // started late, or ticks missed
		d.catchUp(now)
	}
	d.last = now

	for len(d.events) > 0 {
		event := d.events[0]
		if now >= int64(event.When) {
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
