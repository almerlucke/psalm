package poly

import (
	"fmt"

	"github.com/almerlucke/psalm"
)

type Voice interface {
	psalm.Modular
	NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	NoteOff()
	Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	Clear()
	IsActive() bool
}

type voiceInfo[T Voice] struct {
	triggeredAt int64
	voice       T
}

type Poly[T Voice] struct {
	*psalm.Module
	freePool     []*voiceInfo[T]
	activePool   []*voiceInfo[T]
	fadeOut      psalm.Channels
	hasFadeOut   bool
	triggerCount int64

	// mono: one voice, the notes held in a stack (the last one sounds)
	mono      bool
	legato    bool
	monoVoice *voiceInfo[T]
	held      []heldNote
	timers    []monoTimer
	durations int64 // names the notes of a duration, as notes held
}

// heldNote is a note held down in mono: its name, its note message and amplitude
type heldNote struct {
	id  string
	msg psalm.Message
	amp float64
}

// monoTimer lets go of a note of a duration in mono, after its samples
type monoTimer struct {
	id      string
	samples int
}

func New[T Voice](numChannels int, voices []T) *Poly[T] {
	poly := &Poly[T]{
		Module: psalm.NewModule([]int{psalm.Mono}, []int{numChannels}),
	}

	pool := make([]voiceInfo[T], len(voices))
	poly.freePool = make([]*voiceInfo[T], len(voices))
	for i := range pool {
		poly.freePool[i] = &pool[i]
		poly.freePool[i].voice = voices[i]
	}
	poly.activePool = make([]*voiceInfo[T], 0, len(voices))

	poly.fadeOut = psalm.NewChannels(numChannels, poly.Config().BufferSize)

	poly.SetSelf(poly)

	return poly
}

// SetMono plays one note at a time on one voice: a new note takes over from the one held (and when
// it is let go the one still held comes back), legato changes the pitch of the sounding note
// without starting it again, so a synth with a glide slides between them; otherwise every note
// starts the voice again (from the pitch it was at). Off, the synth is polyphonic again. Also by
// the messages "mono" and "legato" (above 0.5 is on)
func (p *Poly[T]) SetMono(mono, legato bool) *Poly[T] {
	if p.mono && !mono {
		p.AllNotesOff()
	}
	p.mono, p.legato = mono, legato
	if p.monoVoice == nil && len(p.freePool)+len(p.activePool) > 0 {
		if len(p.activePool) > 0 {
			p.monoVoice = p.activePool[0]
		} else {
			p.monoVoice = p.freePool[0]
		}
	}
	return p
}

// AdoptVoices takes the voices of another Poly of the same kind of voice in place of its own: what a
// synth made at a preset is, handed to the synth already in a patch (its cables stay). The notes
// that sound are cut with the fade of a stolen voice (no click), the notes held in mono are let go;
// whether it plays mono, and legato, stays as it was. It returns false (and changes nothing) when
// other isn't a Poly of this kind of voice or has another number of channels. Call it on the audio
// thread (patch.Do): it makes nothing, so it doesn't allocate once the other is made. The other is
// left without voices
func (p *Poly[T]) AdoptVoices(other psalm.Modular) bool {
	o, ok := other.(*Poly[T])
	if !ok || o == p || len(o.Output(0).Channels) != len(p.Output(0).Channels) {
		return false
	}
	if len(p.activePool) > 0 {
		p.hasFadeOut = true
		inv := 1.0 / float32(len(p.fadeOut[0]))
		for _, vi := range p.activePool {
			vi.voice.Synthesize()
			p.fadeOut.CopyWithFunc(vi.voice.Output(0).Channels, func(buf psalm.Buffer, samp float32, index int) {
				buf[index] += samp * (1.0 - inv*float32(index))
			})
		}
	}
	p.freePool, p.activePool = o.freePool, o.activePool
	o.freePool, o.activePool = nil, nil
	p.held, p.timers = p.held[:0], p.timers[:0]
	p.monoVoice = nil
	if p.mono {
		p.SetMono(true, p.legato)
	}
	return true
}

// Adopt is AdoptVoices, for an Adopter
func (p *Poly[T]) Adopt(other psalm.Modular) bool { return p.AdoptVoices(other) }

// Mono returns whether it plays one note at a time, and legato
func (p *Poly[T]) Mono() (mono, legato bool) { return p.mono, p.legato }

// activate puts a voice in the active pool (from the free one)
func (p *Poly[T]) activate(vi *voiceInfo[T]) {
	for i, f := range p.freePool {
		if f == vi {
			p.freePool = append(p.freePool[:i], p.freePool[i+1:]...)
			p.activePool = append(p.activePool, vi)
			return
		}
	}
}

// monoOn plays a note in mono: the voice starts, or (legato, a note still held) slides to it
func (p *Poly[T]) monoOn(id string, msg psalm.Message, amp float64, tick psalm.CtrlTick) {
	vi := p.monoVoice
	v := vi.voice
	slide := p.legato && len(p.held) > 0 && v.IsActive()
	p.held = append(p.held, heldNote{id: id, msg: msg, amp: amp})
	if slide {
		v.ReceiveMessage(msg)
		return
	}
	if !v.IsActive() {
		v.Clear()
	}
	p.activate(vi)
	v.NoteOn(amp, msg, tick, p.Config())
}

// monoOff lets go of a note in mono: the note held before it comes back, or the voice lets go
func (p *Poly[T]) monoOff(id string, tick psalm.CtrlTick) {
	i := -1
	for k, h := range p.held {
		if h.id == id {
			i = k
		}
	}
	if i < 0 {
		return
	}
	sounding := i == len(p.held)-1
	p.held = append(p.held[:i], p.held[i+1:]...)
	if !sounding {
		return
	}
	v := p.monoVoice.voice
	if len(p.held) == 0 {
		v.NoteOff()
		return
	}
	back := p.held[len(p.held)-1]
	if p.legato && v.IsActive() {
		v.ReceiveMessage(back.msg)
		return
	}
	v.NoteOn(back.amp, back.msg, tick, p.Config())
}

func (p *Poly[T]) noteOff(identifier string) {
	for _, v := range p.activePool {
		if v.voice.ID() == identifier {
			v.voice.NoteOff()
			v.voice.SetID("")
			return
		}
	}
}

func (p *Poly[T]) AllNotesOff() {
	p.held, p.timers = p.held[:0], p.timers[:0]
	for _, v := range p.activePool {
		v.voice.NoteOff()
		v.voice.SetID("")
	}
}

func (p *Poly[T]) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}

	// Forward plain values (e.g. from an LFO) on inputs > 0 to all voices
	if index > 0 {
		p.CallVoices(func(v T) {
			v.Receive(value, index)
		})
	}
}

func (p *Poly[T]) handleTriggerMessage(msg psalm.Message, tick psalm.CtrlTick, identifier string, duration float64, isNoteOn bool) {
	// Get free voice or steal one
	v, ok := p.getFreeVoice()
	if !ok {
		// Steal oldest active voice
		v, ok = p.getOldestActiveVoice()
		if ok {
			// Fade out stolen voice
			p.hasFadeOut = true
			v.Synthesize()
			invBufferSize := 1.0 / float32(len(p.fadeOut[0]))
			p.fadeOut.CopyWithFunc(v.Output(0).Channels, func(buf psalm.Buffer, samp float32, index int) {
				buf[index] += samp * (1.0 - invBufferSize*float32(index))
			})
		}
	}

	// Activate voice
	if ok {
		v.Clear()
		if isNoteOn {
			v.SetID(identifier)
			v.NoteOn(msg.MustFltVal("amplitude"), msg.MustMsg("message"), tick, p.Config())
		} else {
			v.Note(duration, msg.MustFltVal("amplitude"), msg.MustMsg("message"), tick, p.Config())
		}
	}

	p.triggerCount++
}

// ReceiveMessage triggers notes ("command": "trigger"). A message without a command (or with
// "command": "voice") goes to all voices: a setting for every voice, like the sustain pedal of a
// piano. "command": "note" sends its "message" to the voice of one note, by its "id" (the id of
// its noteOn): the polyphonic aftertouch of a key
func (p *Poly[T]) ReceiveMessage(msg psalm.Message) {
	command, ok := msg.StrVal("command")
	if !ok {
		mono, legato := p.mono, p.legato
		_, m := msg["mono"]
		_, l := msg["legato"]
		msg.IfFltVal("mono", func(f float64) { mono = f > 0.5 })
		msg.IfFltVal("legato", func(f float64) { legato = f > 0.5 })
		if m || l {
			p.SetMono(mono, legato)
		}
	}
	switch {
	case p.mono && command == "trigger":
		tick := msg.MustTickVal("tick")
		if id, ok := msg.StrVal("noteOff"); ok {
			p.monoOff(id, tick)
		} else if id, ok := msg.StrVal("noteOn"); ok {
			p.monoOn(id, msg.MustMsg("message"), msg.MustFltVal("amplitude"), tick)
		} else {
			// a note of a duration: held, and let go after it
			p.durations++
			id := fmt.Sprintf("duration#%d", p.durations)
			samples := int(msg.MustFltVal("duration")*p.Config().SampleRate) + tick.SampleOffset
			p.timers = append(p.timers, monoTimer{id: id, samples: samples})
			p.monoOn(id, msg.MustMsg("message"), msg.MustFltVal("amplitude"), tick)
		}
		p.triggerCount++
	case command == "note":
		id, _ := msg.StrVal("id")
		m, ok := msg.Msg("message")
		if !ok || id == "" {
			return
		}
		if p.mono {
			if n := len(p.held); n > 0 && p.held[n-1].id == id && p.monoVoice != nil {
				p.monoVoice.voice.ReceiveMessage(m)
			}
			return
		}
		for _, v := range p.activePool {
			if v.voice.ID() == id {
				v.voice.ReceiveMessage(m)
			}
		}
	case !ok || command == "voice":
		p.CallVoices(func(v T) {
			v.ReceiveMessage(msg)
		})
	case command == "trigger":
		// get ctrl tick mainly for sample offset (sample accurate timing)
		tick := msg.MustTickVal("tick")
		// trigger note on/off/duration
		if noteOffIdentifier, ok := msg.StrVal("noteOff"); ok {
			p.noteOff(noteOffIdentifier)
		} else if noteOnIdentifier, ok := msg.StrVal("noteOn"); ok {
			p.handleTriggerMessage(msg, tick, noteOnIdentifier, 0, true)
		} else {
			duration := msg.MustFltVal("duration")
			p.handleTriggerMessage(msg, tick, "", duration, false)
		}
	}
}

func (p *Poly[T]) getOldestActiveVoice() (T, bool) {
	var oldest *voiceInfo[T]
	var nilVoice T

	for _, info := range p.activePool {
		if oldest == nil || info.triggeredAt < oldest.triggeredAt {
			oldest = info
		}
	}

	if oldest != nil {
		oldest.triggeredAt = p.triggerCount
		return oldest.voice, true
	}

	return nilVoice, false
}

func (p *Poly[T]) getFreeVoice() (T, bool) {
	var nilVoice T

	if len(p.freePool) == 0 {
		return nilVoice, false
	}

	v := p.freePool[len(p.freePool)-1]
	p.freePool = p.freePool[:len(p.freePool)-1]
	p.activePool = append(p.activePool, v)
	v.triggeredAt = p.triggerCount

	return v.voice, true
}

func (p *Poly[T]) CallVoices(f func(T)) {
	for _, v := range p.activePool {
		f(v.voice)
	}
	for _, v := range p.freePool {
		f(v.voice)
	}
}

func (p *Poly[T]) Synthesize() {
	out := p.Output(0).Channels
	// mono: the notes of a duration let go when their time is up
	if len(p.timers) > 0 {
		n := p.Config().BufferSize
		for i := 0; i < len(p.timers); {
			if p.timers[i].samples -= n; p.timers[i].samples <= 0 {
				id := p.timers[i].id
				p.timers = append(p.timers[:i], p.timers[i+1:]...)
				p.monoOff(id, psalm.CtrlTick{})
				continue
			}
			i++
		}
	}

	if p.hasFadeOut {
		// Copy fade out buffer to output
		out.Copy(p.fadeOut)
		p.fadeOut.Clear()
		p.hasFadeOut = false
	} else {
		out.Clear()
	}

	// Run active voices
	for i := 0; i < len(p.activePool); {
		v := p.activePool[i]
		voice := v.voice
		if voice.IsActive() {
			// Add voice output to buffer
			voice.Synthesize()
			out.Accum(voice.Output(0).Channels)
			i++
		} else {
			// Swap-and-pop deactivated voice
			last := len(p.activePool) - 1
			p.activePool[i] = p.activePool[last]
			p.activePool = p.activePool[:last]
			p.freePool = append(p.freePool, v)
		}
	}
}
