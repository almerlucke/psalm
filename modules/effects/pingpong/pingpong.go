package pingpong

import (
	"math"

	"github.com/almerlucke/genny/float/line"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/tools/timing"
)

type PingPong struct {
	*psalm.Module
	left    *delay.Delay
	right   *delay.Delay
	read    float64
	newRead float64
	mix     float64
	fb      float64
}

func New(delayLengthMs float64, readLocMs float64, feedback float64, mix float64) *PingPong {
	delayLengthSamps := int(math.Ceil(timing.MsToSampsf(delayLengthMs, psalm.SampleRate())))

	pp := &PingPong{
		Module:  psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		left:    delay.New(delayLengthSamps),
		right:   delay.New(delayLengthSamps),
		read:    readLocMs,
		newRead: readLocMs,
		mix:     mix,
		fb:      feedback,
	}

	pp.SetSelf(pp)

	return pp
}

func (pp *PingPong) SetRead(read float64) {
	pp.newRead = read
}

func (pp *PingPong) SetFeedback(fb float64) {
	pp.fb = fb
}

func (pp *PingPong) SetMix(mix float64) {
	pp.mix = mix
}

func (pp *PingPong) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		pp.ReceiveMessage(msg)
		return
	}

	switch index {
	case 0: // Read Location in ms
		pp.SetRead(psalm.FltVal[float64](value))
	case 1:
		pp.SetFeedback(psalm.FltVal[float64](value))
	case 2:
		pp.SetMix(psalm.FltVal[float64](value))
	}
}

func (pp *PingPong) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("location", func(v float64) {
		pp.SetRead(v)
	})
	msg.IfFltVal("feedback", func(v float64) {
		pp.SetFeedback(v)
	})
	msg.IfFltVal("mix", func(v float64) {
		pp.SetMix(v)
	})
}

func (pp *PingPong) Synthesize() {
	var (
		inLeft        = pp.Input(0).Channels[0]
		inRight       = pp.Input(0).Channels[1]
		outLeft       = pp.Output(0).Channels[0]
		outRight      = pp.Output(0).Channels[1]
		delayTimeLine line.Line
		config        = pp.Config()
	)

	delayTimeLine.From(timing.MsToSampsf(pp.read, config.SampleRate))

	if pp.newRead != pp.read {
		delayTimeLine.To(timing.MsToSampsf(pp.newRead, config.SampleRate), config.BufferSize)
		pp.read = pp.newRead
	}

	dry := 1.0 - pp.mix
	wet := pp.mix

	for i := 0; i < config.BufferSize; i++ {
		lookup := delayTimeLine.Generate()
		left := pp.left.ReadLinear(lookup)
		right := pp.right.ReadLinear(lookup)

		il := float64(inLeft[i])
		ir := float64(inRight[i])

		pp.left.Write(il + right*pp.fb)
		pp.right.Write(ir + left)

		outLeft[i] = float32(il*dry + left*wet)
		outRight[i] = float32(ir*dry + right*wet)
	}
}
