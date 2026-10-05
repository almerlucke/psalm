package pitchshift

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/pitchshift"
)

// PitchShift is a stereo pitch shifter. Value input 1 (or message "semitones") sets the shift,
// message "mix" the dry/wet balance
type PitchShift struct {
	*psalm.Module
	shifters [2]*pitchshift.PitchShift
	mix      float64
}

// New creates a pitch shifter, window is the grain size in milliseconds (around 30-100)
func New(semitones float64, windowMs float64, mix float64) *PitchShift {
	p := &PitchShift{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		mix:    mix,
	}

	sr := p.Config().SampleRate
	for c := range p.shifters {
		p.shifters[c] = pitchshift.New(semitones, windowMs, sr)
	}

	p.SetSelf(p)

	return p
}

func (p *PitchShift) SetSemitones(semitones float64) {
	for _, s := range p.shifters {
		s.SetSemitones(semitones)
	}
}

func (p *PitchShift) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}

	if index == 1 {
		p.SetSemitones(psalm.FltVal[float64](value))
	}
}

func (p *PitchShift) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("semitones", p.SetSemitones)
	msg.IfFltVal("mix", func(f float64) { p.mix = f })
}

func (p *PitchShift) Synthesize() {
	ins := p.Input(0).Channels
	outs := p.Output(0).Channels
	wet, dry := p.mix, 1.0-p.mix

	for i := range p.Config().BufferSize {
		for c, s := range p.shifters {
			x := float64(ins[c][i])
			outs[c][i] = float32(dry*x + wet*s.Process(x))
		}
	}
}
