package poly

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/internal/containers/list"
)

type Voice interface {
	psalm.Modular
	NoteOn(amplitude float64, message psalm.Message, config *psalm.Config)
	NoteOff()
	Note(duration float64, amplitude float64, message psalm.Message, config *psalm.Config)
	Clear()
	IsActive() bool
}

type voiceInfo struct {
	age            int64
	isStolen       bool
	nextMsg        psalm.Message
	nextIdentifier string
	voice          Voice
}

type Poly struct {
	*psalm.Module
	freePool   *list.List[*voiceInfo]
	activePool *list.List[*voiceInfo]
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

	poly.SetSelf(poly)

	return poly
}

func (p *Poly) noteOff(identifier string) {
	p.CallActiveVoiceInfo(func(info *voiceInfo) bool {
		if info.isStolen && info.nextIdentifier == identifier {
			info.isStolen = false
			info.nextMsg = nil
			info.nextIdentifier = ""
		} else if info.voice.ID() == identifier {
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
		info.isStolen = false
		info.nextMsg = nil
		info.nextIdentifier = ""
		return true
	})
}

func (p *Poly) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}
}

func (p *Poly) handleTriggerMessage(msg psalm.Message, identifier string, duration float64, isNoteOn bool) {
	v := p.getFreeVoice()
	if v != nil {
		if isNoteOn {
			v.SetID(identifier)
			v.NoteOn(msg.MustFltVal("amplitude"), msg.MustMsg("message"), p.Config())
		} else {
			v.Note(duration, msg.MustFltVal("amplitude"), msg.MustMsg("message"), p.Config())
		}
	} else {
		// Steal oldest active voice
		info := p.getOldestActiveVoiceInfo()
		if info != nil {
			info.isStolen = true
			info.nextMsg = msg
			info.nextIdentifier = identifier
		}
	}
}

func (p *Poly) activateStolenVoiceInfo(info *voiceInfo) {
	if noteOnIdentifier, ok := info.nextMsg.StrVal("noteOn"); ok {
		info.voice.Clear()
		info.voice.SetID(noteOnIdentifier)
		info.voice.NoteOn(info.nextMsg.MustFltVal("amplitude"), info.nextMsg.MustMsg("message"), p.Config())
	} else if duration, ok := info.nextMsg.FltVal("duration"); ok {
		info.voice.Clear()
		info.voice.Note(duration, info.nextMsg.MustFltVal("amplitude"), info.nextMsg.MustMsg("message"), p.Config())
	}
	info.isStolen = false
	info.nextMsg = nil
	info.age = 0
	info.nextIdentifier = ""
}

// ReceiveMessage is used to activate voices
func (p *Poly) ReceiveMessage(msg psalm.Message) {
	if command, ok := msg.StrVal("command"); ok {
		if command == "trigger" {
			// Trigger a voice
			if noteOffIdentifier, ok := msg.StrVal("noteOff"); ok {
				p.noteOff(noteOffIdentifier)
			} else if noteOnIdentifier, ok := msg.StrVal("noteOn"); ok {
				p.handleTriggerMessage(msg, noteOnIdentifier, 0, true)
			} else if duration, ok := msg.FltVal("duration"); ok {
				p.handleTriggerMessage(msg, "", duration, false)
			}
		} else if command == "voice" {
			// Pass message to all voices
			p.CallVoices(func(v Voice) {
				v.ReceiveMessage(msg)
			})
		}
	}
}

func (p *Poly) getOldestActiveVoiceInfo() *voiceInfo {
	var oldest *voiceInfo

	p.activePool.ForEach(func(v *voiceInfo, _ int) {
		if !v.isStolen && (oldest == nil || v.age > oldest.age) {
			oldest = v
		}
	})

	return oldest
}

func (p *Poly) getFreeVoice() Voice {
	e := p.freePool.PopElement()
	if e != nil {
		p.activePool.PushElement(e)
		e.Value.age = 0
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
	p.Output(0).Clear()

	bufferSize := p.Config().BufferSize

	// Run active voices
	p.activePool.ForEachElement(func(e *list.Element[*voiceInfo], _ int) {
		info := e.Value
		voice := info.voice

		if voice.IsActive() {
			// Add voice output to buffer
			voice.Synthesize()

			if info.isStolen {
				// Fade out voice over 1 buffer cycle
				invBufferSize := 1.0 / float32(bufferSize)
				p.Output(0).Channels.CopyWithFunc(voice.Output(0).Channels, func(buf psalm.Buffer, samp float32, index int) {
					buf[index] += samp * (1.0 - invBufferSize*float32(index))
				})
				p.activateStolenVoiceInfo(info)
			} else {
				info.age += int64(bufferSize)
				p.Output(0).Channels.Accum(voice.Output(0).Channels)
			}
		} else if info.isStolen {
			p.activateStolenVoiceInfo(info)
		} else {
			p.freePool.PushElement(e.Unlink())
		}
	})
}
