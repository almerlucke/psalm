package player

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/tools/float"
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
	if numChannels == 0 {
		if sndFile != nil {
			numChannels = sndFile.NumChannels()
		} else {
			numChannels = 1
		}
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
	if sf := p.sf; sf != nil && sf.NumFrames() > 0 {
		p.inc = (p.speed * sf.SampleRate() / p.Config().SampleRate) / float64(sf.NumFrames())
		p.depth = sndfile.SpeedToMipMapDepth(p.speed)
		if p.depth >= sf.Depth() {
			p.depth = sf.Depth() - 1
		}
	}
}

func (p *Player) snapToZeroCrossings() {
	if p.useZeroCrossings {
		zc := p.sf.ZeroCrossings(0)
		if zc != nil {
			if cr, ok := p.sf.ZeroCrossings(0).NearestPos(p.startOffset, sndfile.DirectionAny); ok {
				p.startOffset = cr.Position
			}
			if cr, ok := p.sf.ZeroCrossings(0).NearestPos(p.endOffset, sndfile.DirectionAny); ok {
				p.endOffset = cr.Position
			}
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
	if p.endOffset == 0.0 && p.offsetRange == 0.0 {
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
}

func (p *Player) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	p.activate(amplitude, message, tick)
}

func (p *Player) Note(_ float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	p.activate(amplitude, message, tick)
}

func (p *Player) NoteOff() {}

func (p *Player) Clear() {}

func (p *Player) activate(amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	p.ReceiveMessage(message)

	p.waitSamps = tick.SampleOffset

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
	outSocket.Clear()

	if p.done || p.sf == nil {
		return
	}

	outChannels := outSocket.Channels
	bufferSize := p.Config().BufferSize
	numFramesMinus1 := float64(p.sf.NumFrames() - 1)
	isMonoSource := p.sf.NumChannels() == 1
	isMonoOut := len(outChannels) == 1
	amp := float32(p.amp)
	loopRange := p.endOffset - p.startOffset
	offsetEqual := float.Equal(p.startOffset, p.endOffset)

	for i := p.waitSamps; i < bufferSize; i++ {
		frame := p.sf.LookupAll(p.phase*numFramesMinus1, p.depth, p.loop)
		if isMonoSource {
			val := frame[0] * amp
			for _, channel := range outChannels {
				channel[i] = val
			}
		} else if isMonoOut {
			// Downmix multi-channel source to mono
			outChannels[0][i] = (frame[0] + frame[1]) * 0.5 * amp
		} else {
			for chanIndex, channel := range outChannels {
				channel[i] = frame[chanIndex] * amp
			}
		}

		p.phase += p.inc

		if !p.loop {
			if p.inc < 0.0 {
				if p.phase <= p.startOffset {
					p.done = true
					break
				}
			} else if p.phase >= p.endOffset {
				p.done = true
				break
			}
		} else if offsetEqual {
			p.phase = p.startOffset
		} else {
			np := p.phase - p.startOffset

			for np >= loopRange {
				np -= loopRange
			}
			for np < 0.0 {
				np += loopRange
			}

			p.phase = p.startOffset + np
		}
	}

	p.waitSamps = 0
}

/*
1.
NoteOff():
◦
Currently an empty no-op {}.
◦
For looping or sustained sample playback, NoteOff() should stop playback or start a release phase (p.done = true or p.loop = false).
2.
Clear():
◦
Currently an empty no-op {}.
◦
Should reset p.done = true, p.waitSamps = 0, and clear output buffers.
3.
Note(duration, ...):
◦
Ignores the duration parameter. Calculating a sample countdown (durationSamps = int(duration * sampleRate)) allows one-shot and looping notes to automatically stop after the specified duration.
*/
