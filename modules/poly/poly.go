package poly

import (
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

// ReceiveMessage is used to activate voices
func (p *Poly[T]) ReceiveMessage(msg psalm.Message) {
	if command, ok := msg.StrVal("command"); ok {
		if command == "trigger" {
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
		} else if command == "voice" {
			// Pass message to all voices
			p.CallVoices(func(v T) {
				v.ReceiveMessage(msg)
			})
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
