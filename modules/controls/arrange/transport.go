package arrange

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
)

// JumpAt is when a queued section starts
type JumpAt int32

const (
	// AtSectionEnd starts a queued section when the section playing ends (the default)
	AtSectionEnd JumpAt = iota
	// AtBar starts a queued section at the next bar line
	AtBar
)

// Transport is the song clock of a form: it says where the song is, and decides which section
// comes next. Without changes it plays the sections in order, so the song plays as written. While
// it plays (from the panel, or from code on any goroutine) sections can be switched off (the song
// skips them), a section can be queued (it starts when the section playing ends, or at the next
// bar), and the section playing can be repeated. Every part of the form follows it: the parts and
// streams, the automation lanes, the tempo and the events of the arrangement. Get it with
// Form.Transport
type Transport struct {
	form *Form

	// set from anywhere
	off    []atomic.Bool // sections switched off
	queued atomic.Int32  // the section asked for next, -1 none
	repeat atomic.Bool   // the section playing plays again
	jumpAt atomic.Int32

	// what is playing, for the page and other goroutines
	playing  atomic.Int32  // the section playing, -1 before the start and after the end
	progress atomic.Uint64 // through it, 0-1 (float64 bits)
	over     atomic.Bool   // the song ended (a form that doesn't loop)

	// the clock, on the audio thread
	mu      sync.Mutex
	started bool
	last    int64 // the patch tick of the position
	song    int64 // ticks into the song
	loop    int
	ended   bool
	pos     Position
	ppq     int
}

// Transport returns the song clock of the form, made the first time it is asked for
func (f *Form) Transport() *Transport {
	if f.transport == nil {
		t := &Transport{form: f, off: make([]atomic.Bool, len(f.Sections))}
		t.queued.Store(-1)
		t.playing.Store(-1)
		f.transport = t
	}
	return f.transport
}

// index returns the index of a section, panicking on a name the form doesn't have
func (t *Transport) index(name string) int {
	for i, s := range t.form.Sections {
		if s.Name == name {
			return i
		}
	}
	panic(fmt.Sprintf("arrange: the form has no section %q", name))
}

// Sections returns the names of the sections, in order
func (t *Transport) Sections() []string {
	names := make([]string, len(t.form.Sections))
	for i, s := range t.form.Sections {
		names[i] = s.Name
	}
	return names
}

// SetEnabled switches a section on or off: a section that is off is skipped (it still plays
// when it is queued). It panics on a name the form doesn't have
func (t *Transport) SetEnabled(section string, on bool) *Transport {
	t.off[t.index(section)].Store(!on)
	return t
}

// Enabled tells whether a section is on
func (t *Transport) Enabled(section string) bool { return !t.off[t.index(section)].Load() }

// Queue asks for a section to play next (see SetJump for when it starts), "" takes the request
// back. It panics on a name the form doesn't have
func (t *Transport) Queue(section string) *Transport {
	if section == "" {
		t.queued.Store(-1)
	} else {
		t.queued.Store(int32(t.index(section)))
	}
	return t
}

// Queued returns the section asked for next, "" for none
func (t *Transport) Queued() string {
	if i := t.queued.Load(); i >= 0 {
		return t.form.Sections[i].Name
	}
	return ""
}

// SetRepeat makes the section playing play again and again (until it is let go or another section
// is queued)
func (t *Transport) SetRepeat(on bool) *Transport {
	t.repeat.Store(on)
	return t
}

// Repeat tells whether the section playing repeats
func (t *Transport) Repeat() bool { return t.repeat.Load() }

// SetJump sets when a queued section starts: at the end of the section playing (the default) or
// at the next bar
func (t *Transport) SetJump(at JumpAt) *Transport {
	t.jumpAt.Store(int32(at))
	return t
}

// Jump returns when a queued section starts
func (t *Transport) Jump() JumpAt { return JumpAt(t.jumpAt.Load()) }

// Playing returns the section playing and how far it is (0-1), "" before the song starts and after
// it ended
func (t *Transport) Playing() (section string, progress float64) {
	i := t.playing.Load()
	if i < 0 {
		return "", 0
	}
	return t.form.Sections[i].Name, math.Float64frombits(t.progress.Load())
}

// Ended tells whether the song ended: a form that doesn't loop played its last section (and
// nothing was asked for after it). A playlist ends a song when its transport ended
func (t *Transport) Ended() bool { return t.over.Load() }

// BeatsLeft returns the beats before the song ends, when that is known: nothing is asked for next,
// the section playing doesn't repeat and the form doesn't loop, so the rest of the section playing
// and the sections after it that are on are what plays (true and 0 once it ended). Otherwise false.
// A playlist starts the next song this long before the end, to fade them over each other
func (t *Transport) BeatsLeft() (float64, bool) {
	if t.over.Load() {
		return 0, true
	}
	if t.repeat.Load() || t.queued.Load() >= 0 || t.form.Loop {
		return 0, false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.started || t.ppq <= 0 || t.form.Bars() <= 0 {
		return 0, false
	}
	cur := t.section()
	_, end := t.sectionTicks(cur)
	beats := float64(end-t.song) / float64(t.ppq)
	for i := cur + 1; i < len(t.form.Sections); i++ {
		if !t.off[i].Load() {
			beats += float64(t.form.Sections[i].Bars) * t.form.BeatsPerBar
		}
	}
	return beats, true
}

// Form returns the form the transport plays
func (t *Transport) Form() *Form { return t.form }

// Order returns the sections one pass of the song plays now: the sections that are on, in order
func (t *Transport) Order() []Section {
	var order []Section
	for i, s := range t.form.Sections {
		if !t.off[i].Load() {
			order = append(order, s)
		}
	}
	return order
}

// At returns the position of the song at a tick of the patch (the TickTimestamp of a control tick),
// moving the clock on to it. Every module following the form asks every tick, the first
// one to ask moves the clock, the others get the same position
func (t *Transport) At(tick int64, ppq int) Position {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.started && tick == t.last {
		return t.pos
	}
	if t.form.Bars() <= 0 || t.form.BeatsPerBar <= 0 {
		return t.form.PositionAt(tick, ppq)
	}
	if !t.started || tick < t.last || tick-t.last > int64(ppq)*4*64 {
		t.start(tick, ppq) // the patch starts (again), or a gap too long to catch up with
	} else {
		for t.last < tick {
			t.step()
		}
	}
	t.pos = t.position()
	t.over.Store(t.ended)
	if t.ended {
		t.playing.Store(-1)
	} else {
		t.playing.Store(int32(t.pos.Section.Index))
		t.progress.Store(math.Float64bits(t.pos.Progress))
	}
	return t.pos
}

// restart makes the clock start again with the song (at the next At), a section asked for next
// forgotten: the patch went back to its start
func (t *Transport) restart() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.started = false
	t.queued.Store(-1)
	t.over.Store(false)
}

// ticks returns the length of a bar and of the song in ticks
func (t *Transport) ticks() (bar, song int64) {
	bar = int64(math.Round(t.form.BeatsPerBar * float64(t.ppq)))
	return bar, int64(t.form.Beats() * float64(t.ppq))
}

// sectionTicks returns the first tick of a section and the tick after it
func (t *Transport) sectionTicks(i int) (start, end int64) {
	bar, _ := t.ticks()
	s := t.form.Sections[i]
	return int64(s.StartBar) * bar, int64(s.StartBar+s.Bars) * bar
}

// start puts the clock at the start of the song: the first section that is on (or the one queued)
func (t *Transport) start(tick int64, ppq int) {
	t.started, t.last, t.ppq, t.loop, t.ended = true, tick, ppq, 0, false
	if q := t.queued.Swap(-1); q >= 0 {
		t.song, _ = t.sectionTicks(int(q))
		return
	}
	for i := range t.form.Sections {
		if !t.off[i].Load() {
			t.song, _ = t.sectionTicks(i)
			return
		}
	}
	t.end()
}

// end stops the song: it stays at its last tick
func (t *Transport) end() {
	_, song := t.ticks()
	t.song = max(0, song-1)
	t.ended = true
}

// step moves the clock one tick on, deciding the next section at the end of a section (or at a bar
// line, when a queued section starts at the next bar)
func (t *Transport) step() {
	t.last++
	current := t.section()
	if t.ended {
		if q := t.queued.Swap(-1); q >= 0 { // a section asked for after the end starts at once
			t.ended = false
			t.goTo(int(q), current)
		}
		return
	}
	next := t.song + 1
	bar, _ := t.ticks()
	_, end := t.sectionTicks(current)
	if next == end {
		t.decide(current)
		return
	}
	if next%bar == 0 && t.Jump() == AtBar {
		if q := t.queued.Swap(-1); q >= 0 {
			t.goTo(int(q), current)
			return
		}
	}
	t.song = next
}

// decide picks the section after one that ends: the one queued, the same one when it repeats, else
// the next that is on (from the start again when the form loops), else the song ends
func (t *Transport) decide(current int) {
	if q := t.queued.Swap(-1); q >= 0 {
		t.goTo(int(q), current)
		return
	}
	if t.repeat.Load() {
		t.goTo(current, current)
		return
	}
	n := len(t.form.Sections)
	for k := 1; k <= n; k++ {
		i := current + k
		if i >= n {
			if !t.form.Loop {
				break
			}
			i -= n
		}
		if !t.off[i].Load() {
			t.goTo(i, current)
			return
		}
	}
	t.end()
}

// goTo starts a section; going back (or to the same section) counts as a new loop of the song
func (t *Transport) goTo(i, from int) {
	if i <= from {
		t.loop++
	}
	t.song, _ = t.sectionTicks(i)
}

// section returns the index of the section the clock is in
func (t *Transport) section() int {
	bar, _ := t.ticks()
	b := int(t.song / bar)
	for _, s := range t.form.Sections {
		if b < s.StartBar+s.Bars {
			return s.Index
		}
	}
	return len(t.form.Sections) - 1
}

// position returns the position of the clock
func (t *Transport) position() Position {
	pos := t.form.PositionAt(t.song, t.ppq)
	pos.Loop = t.loop
	pos.Ended = t.ended
	return pos
}
