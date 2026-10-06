package pitchshift

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
	"github.com/almerlucke/psalm/blocks/pitchshift"
)

// PitchShift is a stereo pitch shifter. Value input 1 (or message "semitones") sets the shift,
// message "mix" the dry/wet balance (an equal power crossfade that keeps the loudness, it glides
// when changed)
type PitchShift struct {
	*psalm.Module
	shifters [2]*pitchshift.PitchShift
	mix      float64
	smooth   mix.Smoother
}

// New creates a pitch shifter, window is the grain size in milliseconds (around 30-100)
func New(semitones float64, windowMs float64, m float64) *PitchShift {
	p := &PitchShift{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		mix:    m,
	}

	sr := p.Config().SampleRate
	p.smooth = mix.NewSmoother(clamp(m), mix.SmoothMs, sr)
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
	target := clamp(p.mix)

	for i := range p.Config().BufferSize {
		dry, wet := mix.EqualPower(p.smooth.Next(target))
		for c, s := range p.shifters {
			x := float64(ins[c][i])
			outs[c][i] = float32(dry*x + wet*s.Process(x))
		}
	}
}

func clamp(m float64) float64 { return max(0, min(m, 1)) }

// Tail lets the pitch shifter rest while silent: its window is shorter than this
func (p *PitchShift) Tail() float64 {
	return 0.2
}
