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

type voiceInfo struct {
	triggeredAt int64
	voice       Voice
}

type Poly struct {
	*psalm.Module
	freePool     *list.List[*voiceInfo]
	activePool   *list.List[*voiceInfo]
	fadeOut      psalm.Channels
	hasFadeOut   bool
	triggerCount int64
}

func New(numChannels int, voices []Voice) *Poly {
	poly := &Poly{
		Module: psalm.NewModule([]int{psalm.Mono}, []int{numChannels}),
	}

	poly.freePool = list.New[*voiceInfo]()
	poly.activePool = list.New[*voiceInfo]()

	for _, voice := range voices {
		poly.freePool.Push(&voiceInfo{
			voice: voice,
		})
	}

	poly.fadeOut = psalm.NewChannels(numChannels, poly.Config().BufferSize)

	poly.SetSelf(poly)

	return poly
}

func (p *Poly) noteOff(identifier string) {
	p.CallActiveVoiceInfo(func(info *voiceInfo) bool {
		if info.voice.ID() == identifier {
			info.voice.NoteOff()
			info.voice.SetID("")
			return false
		}

		return true
	})
}

func (p *Poly) AllNotesOff() {
	p.CallActiveVoiceInfo(func(info *voiceInfo) bool {
		info.voice.NoteOff()
		info.voice.SetID("")
		return true
	})
}

func (p *Poly) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}
}

func (p *Poly) handleTriggerMessage(msg psalm.Message, tick psalm.CtrlTick, identifier string, duration float64, isNoteOn bool) {
	v := p.getFreeVoice()
	if v == nil {
		// Steal oldest active voice
		v = p.getOldestActiveVoice()
		if v != nil {
			p.hasFadeOut = true
			v.Synthesize()
			invBufferSize := 1.0 / float32(len(p.fadeOut[0]))
			p.fadeOut.CopyWithFunc(v.Output(0).Channels, func(buf psalm.Buffer, samp float32, index int) {
				buf[index] += samp * (1.0 - invBufferSize*float32(index))
			})
		}
	}

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
func (p *Poly) ReceiveMessage(msg psalm.Message) {
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
			p.CallVoices(func(v Voice) {
				v.ReceiveMessage(msg)
			})
		}
	}
}

func (p *Poly) getOldestActiveVoice() Voice {
	var oldest *voiceInfo

	p.activePool.ForEachElement(func(e *list.Element[*voiceInfo], _ int) {
		if oldest == nil || e.Value.triggeredAt < oldest.triggeredAt {
			oldest = e.Value
		}
	})

	if oldest != nil {
		oldest.triggeredAt = p.triggerCount
	}

	return oldest.voice
}

func (p *Poly) getFreeVoice() Voice {
	e := p.freePool.PopElement()
	if e != nil {
		p.activePool.PushElement(e)
		e.Value.triggeredAt = p.triggerCount
		return e.Value.voice
	}

	return nil
}

func (p *Poly) CallVoices(f func(Voice)) {
	p.CallActiveVoices(func(v Voice) bool {
		f(v)
		return true
	})
	p.CallInactiveVoices(func(v Voice) bool {
		f(v)
		return true
	})
}

func (p *Poly) CallActiveVoiceInfo(f func(*voiceInfo) bool) {
	p.activePool.Until(func(v *voiceInfo, _ int) bool {
		return f(v)
	})
}

func (p *Poly) CallActiveVoices(f func(Voice) bool) {
	p.CallActiveVoiceInfo(func(info *voiceInfo) bool {
		return f(info.voice)
	})
}

func (p *Poly) CallInactiveVoices(f func(Voice) bool) {
	p.freePool.Until(func(v *voiceInfo, _ int) bool {
		return f(v.voice)
	})
}

func (p *Poly) Synthesize() {
	if p.hasFadeOut {
		// Copy fade out buffer to output
		p.Output(0).Channels.Copy(p.fadeOut)
		p.fadeOut.Clear()
		p.hasFadeOut = false
	} else {
		p.Output(0).Clear()
	}

	// Run active voices
	p.activePool.ForEachElement(func(e *list.Element[*voiceInfo], _ int) {
		voice := e.Value.voice

		if voice.IsActive() {
			// Add voice output to buffer
			voice.Synthesize()
			p.Output(0).Channels.Accum(voice.Output(0).Channels)
		} else {
			p.freePool.PushElement(e.Unlink())
		}
	})
}
