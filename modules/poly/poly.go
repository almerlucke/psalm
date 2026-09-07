package poly

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/tools/containers/list"
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
	freePool     *list.List[*voiceInfo[T]]
	activePool   *list.List[*voiceInfo[T]]
	fadeOut      psalm.Channels
	hasFadeOut   bool
	triggerCount int64
}

func New[T Voice](numChannels int, voices []T) *Poly[T] {
	poly := &Poly[T]{
		Module: psalm.NewModule([]int{psalm.Mono}, []int{numChannels}),
	}

	poly.freePool = list.New[*voiceInfo[T]]()
	poly.activePool = list.New[*voiceInfo[T]]()

	for _, voice := range voices {
		poly.freePool.Push(&voiceInfo[T]{
			voice: voice,
		})
	}

	poly.fadeOut = psalm.NewChannels(numChannels, poly.Config().BufferSize)

	poly.SetSelf(poly)

	return poly
}

func (p *Poly[T]) noteOff(identifier string) {
	p.CallActiveVoiceInfo(func(info *voiceInfo[T]) bool {
		if info.voice.ID() == identifier {
			info.voice.NoteOff()
			info.voice.SetID("")
			return false
		}

		return true
	})
}

func (p *Poly[T]) AllNotesOff() {
	p.CallActiveVoiceInfo(func(info *voiceInfo[T]) bool {
		info.voice.NoteOff()
		info.voice.SetID("")
		return true
	})
}

func (p *Poly[T]) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}
}

func (p *Poly[T]) handleTriggerMessage(msg psalm.Message, tick psalm.CtrlTick, identifier string, duration float64, isNoteOn bool) {
	// Get free voice or steal one
	v := p.getFreeVoice()
	if v == nil {
		// Steal oldest active voice
		v = p.getOldestActiveVoice()
		if v != nil {
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
	if v != nil {
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

func (p *Poly[T]) getOldestActiveVoice() Voice {
	var oldest *voiceInfo[T]

	p.activePool.ForEachElement(func(e *list.Element[*voiceInfo[T]], _ int) {
		if oldest == nil || e.Value.triggeredAt < oldest.triggeredAt {
			oldest = e.Value
		}
	})

	if oldest != nil {
		oldest.triggeredAt = p.triggerCount
	}

	return oldest.voice
}

func (p *Poly[T]) getFreeVoice() Voice {
	e := p.freePool.PopElement()
	if e != nil {
		p.activePool.PushElement(e)
		e.Value.triggeredAt = p.triggerCount
		return e.Value.voice
	}

	return nil
}

func (p *Poly[T]) CallVoices(f func(T)) {
	p.CallActiveVoices(func(v T) bool {
		f(v)
		return true
	})
	p.CallInactiveVoices(func(v T) bool {
		f(v)
		return true
	})
}

func (p *Poly[T]) CallActiveVoiceInfo(f func(*voiceInfo[T]) bool) {
	p.activePool.Until(func(v *voiceInfo[T], _ int) bool {
		return f(v)
	})
}

func (p *Poly[T]) CallActiveVoices(f func(T) bool) {
	p.CallActiveVoiceInfo(func(info *voiceInfo[T]) bool {
		return f(info.voice)
	})
}

func (p *Poly[T]) CallInactiveVoices(f func(T) bool) {
	p.freePool.Until(func(v *voiceInfo[T], _ int) bool {
		return f(v.voice)
	})
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
	p.activePool.ForEachElement(func(e *list.Element[*voiceInfo[T]], _ int) {
		voice := e.Value.voice
		if voice.IsActive() {
			// Add voice output to buffer
			voice.Synthesize()
			out.Accum(voice.Output(0).Channels)
		} else {
			p.freePool.PushElement(e.Unlink())
		}
	})
}
