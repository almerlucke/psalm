package pan

import (
	"math"

	"github.com/almerlucke/psalm"
)

// Pan places a mono signal (stereo input is downmixed) in the stereo field with an equal power
// pan law, scaled so a centered signal keeps its level. Pan ranges from -1 (left) to 1 (right),
// changes are ramped over a buffer so it can be modulated (value input 0, or message "pan")
type Pan struct {
	*psalm.Module
	pan    float64
	target float64
}

func New(pan float64) *Pan {
	p := &Pan{
		Module: psalm.NewModule([]int{psalm.Mono}, []int{psalm.Stereo}),
		pan:    clamp(pan),
		target: clamp(pan),
	}

	p.SetSelf(p)

	return p
}

func clamp(pan float64) float64 {
	return max(-1.0, min(1.0, pan))
}

// gains returns the left and right gain for a pan position
func gains(pan float64) (float64, float64) {
	theta := (pan + 1.0) * math.Pi / 4.0
	return math.Cos(theta) * math.Sqrt2, math.Sin(theta) * math.Sqrt2
}

func (p *Pan) SetPan(pan float64) {
	p.target = clamp(pan)
}

// Pan returns the pan position it is set to, -1 (left) to 1 (right)
func (p *Pan) Pan() float64 { return p.target }

func (p *Pan) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}

	if index == 0 {
		p.SetPan(psalm.FltVal[float64](value))
	}
}

func (p *Pan) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("pan", p.SetPan)
}

func (p *Pan) Synthesize() {
	in := p.Input(0).Channels[0]
	out := p.Output(0).Channels
	bufSize := p.Config().BufferSize
	inc := (p.target - p.pan) / float64(bufSize)

	for i := range bufSize {
		p.pan += inc
		gl, gr := gains(p.pan)
		out[0][i] = in[i] * float32(gl)
		out[1][i] = in[i] * float32(gr)
	}

	p.pan = p.target
}

// Tail lets the pan rest while silent
func (p *Pan) Tail() float64 {
	return 0
}
