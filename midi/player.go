package midi

import (
	"math"
	"slices"
	"sync"
)

// Player plays a MIDI file: an input that hands the messages of the file over buffer by buffer,
// each at the frame it falls on, following the tempo changes of the file. A keyboard
// (modules/controls/keyboard) reading it plays synths from the file as from a keyboard; a file of
// many channels plays a synth for each through a keyboard for each channel, all reading one
// player through Tee.
//
// It plays, stops (where it is, the notes let go), seeks (the program, the controllers, the bend
// and the pressure of every channel set to what they are at that point), loops and plays faster
// or slower; channels and tracks can be muted. Every method can be called from any goroutine.
// The tracks of a file of format 2 (sequences of their own) play together, as in format 1
type Player struct {
	mu       sync.Mutex
	file     *File
	events   []playEvent // the channel messages of every track, in order of time
	length   float64     // seconds
	pos      float64     // seconds into the file
	next     int         // the first event at or after pos
	playing  bool
	loop     bool
	speed    float64
	tempoMap []tempoChange
	division int
	tempo    func() float64 // the tempo followed (quarter notes a minute), nil: the file's
	used     [16]bool
	mutedCh  [16]bool
	mutedTr  []bool
	sounding [16][128]int // the track a note sounds from (+1), 0 it doesn't
	pending  []Message    // sent at the start of the next buffer
}

type playEvent struct {
	at    float64
	track int
	msg   Message
}

// NewPlayer returns a player of a file, stopped at its start
func NewPlayer(f *File) *Player {
	p := &Player{file: f, speed: 1, mutedTr: make([]bool, len(f.Tracks)), division: f.Division}
	tempo := f.tempoMap()
	p.tempoMap = tempo
	for i, t := range f.Tracks {
		for _, e := range t.Events {
			if !e.Message.IsChannel() {
				continue
			}
			p.events = append(p.events, playEvent{secondsAt(tempo, e.Tick), i, e.Message})
			p.used[e.Message.Channel()-1] = true
		}
	}
	// at the same time a note let go before one struck (the same note let go and struck again)
	slices.SortStableFunc(p.events, func(a, b playEvent) int {
		switch {
		case a.at < b.at:
			return -1
		case a.at > b.at:
			return 1
		}
		return isOff(b.msg) - isOff(a.msg)
	})
	p.length = secondsAt(tempo, f.Ticks())
	return p
}

func isOff(m Message) int {
	if m.Kind() == NoteOff {
		return 1
	}
	return 0
}

// File returns the file played
func (p *Player) File() *File { return p.file }

// Play plays from where the player is (at the end, from the start)
func (p *Player) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pos >= p.length && p.next >= len(p.events) {
		p.pending = append(p.pending, p.seek(0)...)
	}
	p.playing = true
}

// Stop stops where the player is, the notes sounding let go
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playing = false
	p.pending = append(p.pending, p.release(func(int, int) bool { return true })...)
}

// Playing says whether the player plays (it stops at the end, unless it loops)
func (p *Player) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// Seek moves to a time (seconds into the file), playing or not
func (p *Player) Seek(seconds float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, p.seek(seconds)...)
}

// Position returns the time played to, in seconds into the file
func (p *Player) Position() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pos
}

// Duration returns the length of the file in seconds
func (p *Player) Duration() float64 { return p.length }

// SetLoop makes the player start over at the end
func (p *Player) SetLoop(loop bool) {
	p.mu.Lock()
	p.loop = loop
	p.mu.Unlock()
}

// Loop says whether the player loops
func (p *Player) Loop() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loop
}

// SetSpeed sets how fast the file plays: 1 at its tempo, 0.5 at half of it (0.05 to 8)
func (p *Player) SetSpeed(speed float64) {
	p.mu.Lock()
	p.speed = max(0.05, min(speed, 8))
	p.mu.Unlock()
}

// SetTempo plays the file at a tempo (quarter notes a minute) instead of its own: every beat at
// that tempo, the tempo changes of the file left out, so it plays in step with whatever keeps that
// tempo; 0 plays it at its own tempo again. The speed applies on top. A file timed in SMPTE frames
// has no beats: it keeps its time
func (p *Player) SetTempo(bpm float64) {
	if bpm <= 0 {
		p.FollowTempo(nil)
		return
	}
	p.FollowTempo(func() float64 { return bpm })
}

// FollowTempo plays the file at a tempo that may change, asked for every buffer (on the goroutine
// reading the player: the audio thread, so a patch's tempo can be followed, or a MIDI clock):
// as SetTempo. Nil (or a tempo of 0) plays it at its own tempo
func (p *Player) FollowTempo(bpm func() float64) {
	p.mu.Lock()
	p.tempo = bpm
	p.mu.Unlock()
}

// Tempo returns the tempo of the file where it is (quarter notes a minute; 0 for SMPTE time)
func (p *Player) Tempo() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fileTempo(p.pos)
}

// fileTempo returns the tempo of the file at a time
func (p *Player) fileTempo(seconds float64) float64 {
	if p.division <= 0 {
		return 0
	}
	i, _ := slices.BinarySearchFunc(p.tempoMap, seconds, func(c tempoChange, s float64) int {
		if c.seconds <= s {
			return -1
		}
		return 1
	})
	c := p.tempoMap[max(0, i-1)]
	return 60 / (c.perTick * float64(p.division))
}

// PlayTime returns the seconds it takes to play from a time in the file to its end, at the speed
// and the tempo followed now
func (p *Player) PlayTime(from float64) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	from = max(0, min(from, p.length))
	bpm := 0.0
	if p.tempo != nil {
		bpm = p.tempo()
	}
	if bpm <= 0 || p.division <= 0 {
		return (p.length - from) / p.speed
	}
	total := 0.0
	for i, c := range p.tempoMap {
		end := p.length
		if i+1 < len(p.tempoMap) {
			end = min(end, p.tempoMap[i+1].seconds)
		}
		if seg := end - max(from, c.seconds); seg > 0 {
			total += seg * (60 / (c.perTick * float64(p.division))) / bpm
		}
	}
	return total / p.speed
}

// Rate returns how fast the file plays now against its own time: the speed, and the tempo followed
// against the file's (0 when it is stopped)
func (p *Player) Rate() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.playing {
		return 0
	}
	return p.rate()
}

// nextTempoChange returns the time of the file's next tempo change after where it is
func (p *Player) nextTempoChange() (float64, bool) {
	i, _ := slices.BinarySearchFunc(p.tempoMap, p.pos+1e-9, func(c tempoChange, s float64) int {
		if c.seconds <= s {
			return -1
		}
		return 1
	})
	if i >= len(p.tempoMap) {
		return 0, false
	}
	return p.tempoMap[i].seconds, true
}

// rate returns how fast the file plays now: the speed, and the tempo followed against the file's
func (p *Player) rate() float64 {
	r := p.speed
	if p.tempo == nil {
		return r
	}
	if bpm, own := p.tempo(), p.fileTempo(p.pos); bpm > 0 && own > 0 {
		r *= bpm / own
	}
	return max(0.01, r)
}

// Speed returns how fast the file plays
func (p *Player) Speed() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.speed
}

// MuteChannel mutes the notes of a channel (1-16) or plays them again; the notes sounding let go.
// The controllers of a muted channel still go through, so it plays on as it should when unmuted
func (p *Player) MuteChannel(channel int, mute bool) {
	if channel < 1 || channel > 16 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mutedCh[channel-1] = mute
	if mute {
		p.pending = append(p.pending, p.notesOff(func(ch, _ int) bool { return ch == channel-1 })...)
	}
}

// ChannelMuted says whether a channel (1-16) is muted
func (p *Player) ChannelMuted(channel int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return channel >= 1 && channel <= 16 && p.mutedCh[channel-1]
}

// MuteTrack mutes the notes of a track (from 0) or plays them again, as MuteChannel
func (p *Player) MuteTrack(track int, mute bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if track < 0 || track >= len(p.mutedTr) {
		return
	}
	p.mutedTr[track] = mute
	if mute {
		p.pending = append(p.pending, p.notesOff(func(_, tr int) bool { return tr == track })...)
	}
}

// TrackMuted says whether a track is muted
func (p *Player) TrackMuted(track int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return track >= 0 && track < len(p.mutedTr) && p.mutedTr[track]
}

// Read implements Input
func (p *Player) Read(events []Event, frames int, sampleRate float64) []Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, m := range p.pending {
		events = append(events, Event{m, 0})
	}
	p.pending = p.pending[:0]
	if !p.playing || frames <= 0 {
		return events
	}
	done := 0
	for done < frames {
		speed := p.rate()
		// following a tempo, the rate changes where the file's tempo does: the buffer is played in
		// parts, up to the next change
		part := frames - done
		if p.tempo != nil {
			if next, ok := p.nextTempoChange(); ok && next < p.length {
				part = max(1, min(part, int(math.Ceil((next-p.pos)/speed*sampleRate-1e-6))))
			}
		}
		stop := done + part
		end := p.pos + float64(part)/sampleRate*speed
		for p.next < len(p.events) {
			e := p.events[p.next]
			// the frame it falls on (a hair added: the position is a sum of buffers, a time on a
			// frame may come out just before it)
			offset := done + int(math.Floor((e.at-p.pos)/speed*sampleRate+1e-6))
			if offset >= stop {
				break
			}
			p.next++
			if p.play(e) {
				events = append(events, Event{e.msg, max(done, offset)})
			}
		}
		if end < p.length {
			p.pos, done = end, stop
			continue
		}
		// the end: let go of everything, start over or stop
		done += max(1, int(math.Ceil((p.length-p.pos)/speed*sampleRate)))
		at := min(done, frames-1)
		var msgs []Message
		if p.loop && p.length > 0 {
			msgs = p.seek(0)
		} else {
			msgs = p.release(func(int, int) bool { return true })
			p.playing, p.pos = false, p.length
		}
		for _, m := range msgs {
			events = append(events, Event{m, at})
		}
		if !p.playing {
			break
		}
	}
	return events
}

// play says whether an event plays (it isn't muted), keeping the notes sounding
func (p *Player) play(e playEvent) bool {
	m := e.msg
	ch := m.Channel() - 1
	switch m.Kind() {
	case NoteOn:
		if p.mutedCh[ch] || p.mutedTr[e.track] {
			return false
		}
		p.sounding[ch][m.Note()] = e.track + 1
	case NoteOff:
		if p.sounding[ch][m.Note()] == 0 {
			return false
		}
		p.sounding[ch][m.Note()] = 0
	}
	return true
}

// notesOff lets go of the notes sounding a test picks (by channel and track)
func (p *Player) notesOff(pick func(ch, track int) bool) []Message {
	var msgs []Message
	for ch := range p.sounding {
		for note, tr := range p.sounding[ch] {
			if tr > 0 && pick(ch, tr-1) {
				msgs = append(msgs, NoteOffMsg(ch+1, note, 0))
				p.sounding[ch][note] = 0
			}
		}
	}
	return msgs
}

// release lets go of the notes and lifts the sustain pedal of every channel played
func (p *Player) release(pick func(ch, track int) bool) []Message {
	msgs := p.notesOff(pick)
	for ch, used := range p.used {
		if used {
			msgs = append(msgs, ControlChangeMsg(ch+1, CCSustain, 0), ControlChangeMsg(ch+1, CCAllNotesOff, 0))
		}
	}
	return msgs
}

// seek moves to a time: the notes let go, the controllers back to rest, then every channel set as
// the file has it there
func (p *Player) seek(seconds float64) []Message {
	seconds = max(0, min(seconds, p.length))
	msgs := p.release(func(int, int) bool { return true })
	p.pos = seconds
	p.next, _ = slices.BinarySearchFunc(p.events, seconds, func(e playEvent, s float64) int {
		if e.at < s {
			return -1
		}
		return 1
	})
	type state struct {
		program, bend, pressure int
		controls                [120]int
	}
	var chase [16]state
	for ch := range chase {
		chase[ch] = state{program: -1, bend: -1, pressure: -1}
		for c := range chase[ch].controls {
			chase[ch].controls[c] = -1
		}
	}
	for _, e := range p.events[:p.next] {
		m := e.msg
		s := &chase[m.Channel()-1]
		switch m.Kind() {
		case ProgramChange:
			s.program = m.Program()
		case PitchBend:
			s.bend = m.Bend()
		case ChannelPressure:
			s.pressure = m.Pressure()
		case ControlChange:
			if c := m.Controller(); c < len(s.controls) {
				s.controls[c] = m.Value()
			} else if c == CCResetAll {
				*s = state{program: s.program, bend: -1, pressure: -1}
				for c := range s.controls {
					s.controls[c] = -1
				}
			}
		}
	}
	for ch, used := range p.used {
		if !used {
			continue
		}
		s := chase[ch]
		msgs = append(msgs, ControlChangeMsg(ch+1, CCResetAll, 0))
		// the bank before the program it selects
		for _, c := range []int{CCBankSelect, CCBankSelect + 32} {
			if v := s.controls[c]; v >= 0 {
				msgs = append(msgs, ControlChangeMsg(ch+1, c, v))
			}
		}
		if s.program >= 0 {
			msgs = append(msgs, ProgramChangeMsg(ch+1, s.program))
		}
		for c, v := range s.controls {
			if v >= 0 && c != CCBankSelect && c != CCBankSelect+32 {
				msgs = append(msgs, ControlChangeMsg(ch+1, c, v))
			}
		}
		if s.bend >= 0 {
			msgs = append(msgs, PitchBendMsg(ch+1, s.bend))
		}
		if s.pressure >= 0 {
			msgs = append(msgs, ChannelPressureMsg(ch+1, s.pressure))
		}
	}
	return msgs
}

// Tee shares an input between readers (a keyboard for every channel of a file): each reader gets
// every event of every buffer. The input is read once a buffer, by the first reader to read it,
// so every reader has to be read every buffer
func Tee(input Input, readers int) []Input {
	t := &tee{input: input}
	ins := make([]Input, readers)
	for i := range ins {
		ins[i] = &teeReader{tee: t}
	}
	return ins
}

type tee struct {
	mu     sync.Mutex
	input  Input
	events []Event
	reads  int64
}

type teeReader struct {
	*tee
	reads int64
}

func (r *teeReader) Read(events []Event, frames int, sampleRate float64) []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.reads >= r.tee.reads {
		clear(r.tee.events)
		r.tee.events = r.input.Read(r.tee.events[:0], frames, sampleRate)
		r.tee.reads++
	}
	r.reads = r.tee.reads
	return append(events, r.tee.events...)
}

// SeekBeat moves to a beat (quarter notes from the start of the file), playing or not; 0 for a file timed in SMPTE frames, which has no beats
func (p *Player) SeekBeat(beat float64) {
	if p.division <= 0 {
		return
	}
	p.Seek(secondsAt(p.tempoMap, int64(math.Round(max(0, beat)*float64(p.division)))))
}

// Beat returns where the player is in quarter notes from the start of the file (0 for a file timed in SMPTE frames)
func (p *Player) Beat() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.division <= 0 {
		return 0
	}
	i, _ := slices.BinarySearchFunc(p.tempoMap, p.pos+1e-9, func(c tempoChange, s float64) int {
		if c.seconds <= s {
			return -1
		}
		return 1
	})
	c := p.tempoMap[max(0, i-1)]
	return (float64(c.tick) + (p.pos-c.seconds)/c.perTick) / float64(p.division)
}
