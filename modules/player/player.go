package player

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/internal/float"
	"github.com/almerlucke/sndfile"
)

type Config struct {
	SndFile          sndfile.SoundFiler[float32]
	SndBank          sndfile.SoundBank[float32]
	SndName          string
	NumChannels      int
	StartOffset      float64 // 0-1 range
	EndOffset        float64 // 0-1 range
	OffsetRange      float64 // 0-1 range
	Speed            float64
	Amp              float64
	Loop             bool
	UseZeroCrossings bool
}

type Player struct {
	*psalm.Module
	sf               sndfile.SoundFiler[float32]
	sndBank          sndfile.SoundBank[float32]
	startOffset      float64
	endOffset        float64
	offsetRange      float64
	phase            float64
	inc              float64
	speed            float64
	amp              float64
	depth            int
	loop             bool
	useZeroCrossings bool
	done             bool
	waitSamps        int
}

func New(cfg Config) *Player {
	sndFile := cfg.SndFile
	if sndFile == nil {
		if cfg.SndBank != nil && cfg.SndName != "" {
			if sf, ok := cfg.SndBank[cfg.SndName]; ok {
				sndFile = sf
			}
		}
	}

	numChannels := cfg.NumChannels
	if numChannels == 0 && sndFile != nil {
		numChannels = sndFile.NumChannels()
	} else {
		numChannels = 1
	}

	p := &Player{
		Module:           psalm.NewModule(nil, []int{numChannels}),
		speed:            cfg.Speed,
		loop:             cfg.Loop,
		done:             !cfg.Loop,
		sf:               sndFile,
		sndBank:          cfg.SndBank,
		amp:              cfg.Amp,
		startOffset:      cfg.StartOffset,
		endOffset:        cfg.EndOffset,
		useZeroCrossings: cfg.UseZeroCrossings,
	}

	p.SetSelf(p)
	p.sanitizeOffsets()
	p.recalculate()

	return p
}

func (p *Player) recalculate() {
	if sf := p.sf; sf != nil {
		p.inc = (p.speed * sf.SampleRate() / p.Config().SampleRate) / float64(sf.NumFrames())
		p.depth = sndfile.SpeedToMipMapDepth(p.speed)
		if p.depth >= sf.Depth() {
			p.depth = sf.Depth() - 1
		}
	}
}

func (p *Player) snapToZeroCrossings() {
	if p.useZeroCrossings {
		if cr, ok := p.sf.ZeroCrossings(0).NearestPos(p.startOffset, sndfile.DirectionAny); ok {
			p.startOffset = cr.Position
		}
		if cr, ok := p.sf.ZeroCrossings(0).NearestPos(p.endOffset, sndfile.DirectionAny); ok {
			p.endOffset = cr.Position
		}
	}
}

func (p *Player) sanitizeOffsets() {
	if p.startOffset < 0.0 {
		p.startOffset = 0.0
	}
	if p.startOffset > 1.0 {
		p.startOffset = 1.0
	}

	if p.offsetRange > 0.0 {
		p.endOffset = p.startOffset + p.offsetRange
	}

	if p.endOffset < 0.0 {
		p.endOffset = 0.0
	}
	if p.endOffset > 1.0 {
		p.endOffset = 1.0
	}
	if p.startOffset > p.endOffset {
		tmp := p.startOffset
		p.startOffset = p.endOffset
		p.endOffset = tmp
	}

	p.snapToZeroCrossings()
}

// offset in seconds
func (p *Player) normalizeOffset(offset float64) float64 {
	if p.sf == nil {
		return 0.0
	}

	numFrames := float64(p.sf.NumFrames())
	frameOffset := offset * p.sf.SampleRate()
	if frameOffset >= numFrames {
		frameOffset = numFrames - 1.0
	}

	return frameOffset / numFrames
}

func (p *Player) SetSoundBank(sndBank sndfile.SoundBank[float32]) *Player {
	p.sndBank = sndBank
	return p
}

func (p *Player) SetSound(sound string) *Player {
	if p.sndBank != nil {
		if sf, ok := p.sndBank[sound]; ok {
			p.SetSoundFile(sf)
		}
	}
	return p
}

func (p *Player) SetSoundFile(sf sndfile.SoundFiler[float32]) {
	p.sf = sf
	p.recalculate()
}

func (p *Player) SetAmp(amp float64) *Player {
	p.amp = amp
	return p
}

func (p *Player) SetSpeed(speed float64) *Player {
	p.speed = speed
	p.recalculate()
	return p
}

func (p *Player) SetOffsets(start float64, end float64) *Player {
	p.startOffset = start
	p.endOffset = end
	p.sanitizeOffsets()
	return p
}

func (p *Player) SetOffsetsSec(start float64, end float64) *Player {
	if p.sf == nil {
		return p
	}
	p.startOffset = p.normalizeOffset(start)
	p.endOffset = p.normalizeOffset(end)
	p.sanitizeOffsets()
	return p
}

func (p *Player) SetStartOffset(offset float64) *Player {
	p.startOffset = offset
	p.sanitizeOffsets()
	return p
}

func (p *Player) SetStartOffsetSec(offset float64) *Player {
	if p.sf == nil {
		return p
	}
	p.startOffset = p.normalizeOffset(offset)
	p.sanitizeOffsets()
	return p
}

func (p *Player) SetEndOffset(offset float64) *Player {
	p.endOffset = offset
	p.sanitizeOffsets()
	return p
}

func (p *Player) SetEndOffsetSec(offset float64) *Player {
	if p.sf == nil {
		return p
	}
	p.endOffset = p.normalizeOffset(offset)
	p.sanitizeOffsets()
	return p
}

func (p *Player) SetOffsetRange(offset float64) *Player {
	p.offsetRange = offset
	p.sanitizeOffsets()
	return p
}

func (p *Player) Bang() {
	if !p.loop {
		p.done = false
		if p.inc < 0.0 {
			p.phase = p.endOffset
		} else {
			p.phase = p.startOffset
		}
	}
}

func (p *Player) Receive(value any, index int) {
	switch index {
	case 0: // Bang or msg
		if msg, ok := psalm.MsgVal(value); ok {
			p.ReceiveMessage(msg)
		}
		if psalm.IsBang(value) {
			p.Bang()
		}
	case 1: // Speed
		p.SetSpeed(psalm.FltVal[float64](value))
	case 2: // Amp
		p.SetAmp(psalm.FltVal[float64](value))
	case 3: // Start offset
		p.SetStartOffset(psalm.FltVal[float64](value))
	case 4: // End offset
		p.SetEndOffset(psalm.FltVal[float64](value))
	case 5: // Offset range
		p.SetOffsetRange(psalm.FltVal[float64](value))
	}
}

func (p *Player) ReceiveMessage(msg psalm.Message) {
	if sound, ok := msg.StrVal("sound"); ok {
		p.SetSound(sound)
	}

	if speed, ok := msg.FltVal("speed"); ok {
		p.SetSpeed(speed)
	}

	if amp, ok := msg.FltVal("amp"); ok {
		p.SetAmp(amp)
	}

	if startOffset, ok := msg.FltVal("startOffset"); ok {
		p.SetStartOffset(startOffset)
	}

	if endOffset, ok := msg.FltVal("endOffset"); ok {
		p.SetEndOffset(endOffset)
	}

	if startOffsetSec, ok := msg.FltVal("startOffsetSec"); ok {
		p.SetStartOffsetSec(startOffsetSec)
	}

	if endOffsetSec, ok := msg.FltVal("endOffsetSec"); ok {
		p.SetEndOffsetSec(endOffsetSec)
	}

	if offsetRange, ok := msg.FltVal("offsetRange"); ok {
		p.SetOffsetRange(offsetRange)
	}

	if tick, ok := msg["tick"].(psalm.CtrlTick); ok {
		p.waitSamps = tick.SampleOffset
	}
}

func (p *Player) NoteOn(amplitude float64, message psalm.Message, _ psalm.Config) {
	p.activate(amplitude, message)
}

func (p *Player) Note(_ float64, amplitude float64, message psalm.Message, _ psalm.Config) {
	p.activate(amplitude, message)
}

func (p *Player) NoteOff() {}

func (p *Player) Clear() {}

func (p *Player) activate(amplitude float64, message psalm.Message) {
	p.ReceiveMessage(message)
	if p.inc < 0.0 {
		p.phase = p.endOffset
	} else {
		p.phase = p.startOffset
	}
	p.amp = amplitude
	p.done = false
}

func (p *Player) IsActive() bool {
	return p.loop || !p.done
}

func (p *Player) Synthesize() {
	outSocket := p.Output(0)

	for i := p.waitSamps; i < p.Config().BufferSize; i++ {
		if p.done || p.sf == nil {
			for _, out := range p.Outputs() {
				out.Clear()
			}
			continue
		}

		frame := p.sf.LookupAll(p.phase*float64(p.sf.NumFrames()-1), p.depth, p.loop)
		if len(frame) == 1 {
			for _, channel := range outSocket.Channels {
				channel[i] = float32(float64(frame[0]) * p.amp)
			}
		} else {
			for chanIndex, channel := range outSocket.Channels {
				channel[i] = float32(float64(frame[chanIndex]) * p.amp)
			}
		}

		p.phase += p.inc

		if !p.loop {
			if p.inc < 0.0 {
				if p.phase <= p.startOffset {
					p.done = true
				}
			} else if p.phase >= p.endOffset {
				p.done = true
			}
		} else {
			if float.Equal(p.startOffset, p.endOffset) {
				p.phase = p.startOffset
			} else {
				r := p.endOffset - p.startOffset
				np := p.phase - p.startOffset

				for np >= r {
					np -= r
				}
				for np < 0.0 {
					np += r
				}

				p.phase = p.startOffset + np
			}
		}
	}

	p.waitSamps = 0
}
