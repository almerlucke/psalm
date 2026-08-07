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

type fadeOut struct {
	channels psalm.Channels
}

func newFadeOutVoice(numChannels int, bufferSize int) *fadeOut {
	return &fadeOut{
		channels: psalm.NewChannels(numChannels, bufferSize),
	}
}

func (f *fadeOut) Fill(v Voice) {
	v.Synthesize()
	invBufferSize := 1.0 / float32(len(f.channels[0]))
	f.channels.CopyWithFunc(v.Output(0).Channels, func(buf psalm.Buffer, samp float32, index int) {
		buf[index] += samp * (1.0 - invBufferSize*float32(index))
	})
}

func (f *fadeOut) Copy(c psalm.Channels) {
	c.Accum(f.channels)
}

type voiceInfo struct {
	age   int64
	voice Voice
}

type Poly struct {
	*psalm.Module
	freePool          *list.List[*voiceInfo]
	activePool        *list.List[*voiceInfo]
	fadeOutFreePool   *list.List[*fadeOut]
	fadeOutActivePool *list.List[*fadeOut]
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

	poly.fadeOutFreePool = list.New[*fadeOut]()
	poly.fadeOutActivePool = list.New[*fadeOut]()
	for range len(voices) {
		fadeOut := newFadeOutVoice(numChannels, poly.Config().BufferSize)
		poly.fadeOutFreePool.Push(fadeOut)
	}

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
		e := p.getOldestActiveVoiceElement()
		if e != nil {
			fadeOut := p.getFadeOut()
			fadeOut.Fill(e.Value.voice)
			p.freePool.PushElement(e.Unlink())
			p.handleTriggerMessage(msg, identifier, duration, isNoteOn)
		}
	}
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

func (p *Poly) getOldestActiveVoiceElement() *list.Element[*voiceInfo] {
	var oldest *list.Element[*voiceInfo]

	p.activePool.ForEachElement(func(e *list.Element[*voiceInfo], _ int) {
		if oldest == nil || e.Value.age > oldest.Value.age {
			oldest = e
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

func (p *Poly) getFadeOut() *fadeOut {
	e := p.fadeOutFreePool.PopElement()
	if e != nil {
		p.fadeOutActivePool.PushElement(e)
		return e.Value
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
			p.Output(0).Channels.Accum(voice.Output(0).Channels)
			info.age += int64(bufferSize)
		} else {
			p.freePool.PushElement(e.Unlink())
		}
	})

	// Copy fade out voices to output
	p.fadeOutActivePool.ForEachElement(func(e *list.Element[*fadeOut], _ int) {
		e.Value.Copy(p.Output(0).Channels)
		p.fadeOutFreePool.PushElement(e.Unlink())
	})
}
