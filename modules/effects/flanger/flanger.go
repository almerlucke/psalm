package flanger

import (
	"math"

	"github.com/almerlucke/genny/float/line"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/internal/timing"
)

const (
	FlangeMinPos = 1.0
	FlangeMaxPos = 20.0
)

type Flanger struct {
	*psalm.Module
	delayLeft  *delay.Delay
	delayRight *delay.Delay
	fb         float64
	depth      float64
	newDepth   float64
	mix        float64
}

func New(depth float64, feedback float64, mix float64) *Flanger {
	delaySize := int(math.Ceil(timing.MsToSampsf(FlangeMaxPos, psalm.SampleRate())))

	f := &Flanger{
		Module:     psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		delayLeft:  delay.New(delaySize),
		delayRight: delay.New(delaySize),
		depth:      depth,
		fb:         feedback,
		mix:        mix,
	}

	f.SetSelf(f)

	return f
}

func (f *Flanger) SetDepth(depth float64) {
	f.newDepth = depth
}

func (f *Flanger) SetFeedback(fb float64) {
	f.fb = fb
}

func (f *Flanger) SetMix(mix float64) {
	f.mix = mix
}

func (f *Flanger) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		f.ReceiveMessage(msg)
		return
	}

	switch index {
	case 0:
		f.SetDepth(psalm.FltVal[float64](value))
	case 1:
		f.SetFeedback(psalm.FltVal[float64](value))
	case 2:
		f.SetMix(psalm.FltVal[float64](value))
	}
}

func (f *Flanger) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("depth", func(v float64) {
		f.SetDepth(v)
	})
	msg.IfFltVal("feedback", func(v float64) {
		f.SetFeedback(v)
	})
	msg.IfFltVal("mix", func(v float64) {
		f.SetMix(v)
	})
}

func (f *Flanger) Synthesize() {
	var (
		inBufLeft   = f.Input(0).Channels[0]
		inBufRight  = f.Input(0).Channels[1]
		outBufLeft  = f.Output(0).Channels[0]
		outBufRight = f.Output(0).Channels[1]
		readLine    line.Line
		config      = f.Config()
	)

	readLine.From(timing.MsToSampsf(f.depth*(FlangeMaxPos-FlangeMinPos)+FlangeMinPos, config.SampleRate))
	if f.depth != f.newDepth {
		readLine.To(timing.MsToSampsf(f.newDepth*(FlangeMaxPos-FlangeMinPos)+FlangeMinPos, config.SampleRate), config.BufferSize)
		f.depth = f.newDepth
	}

	dry := 1.0 - f.mix
	wet := f.mix

	for i := 0; i < config.BufferSize; i++ {
		inLeft := float64(inBufLeft[i])
		inRight := float64(inBufRight[i])
		readPos := readLine.Generate()
		delOutLeft := f.delayLeft.ReadLinear(readPos)
		delOutRight := f.delayRight.ReadLinear(readPos)
		f.delayLeft.Write(inLeft + delOutLeft*f.fb)
		f.delayRight.Write(inRight + delOutRight*f.fb)
		flangOutLeft := inLeft + delOutLeft
		flangOutRight := inRight + delOutRight
		outBufLeft[i] = float32(inLeft*dry + flangOutLeft*wet)
		outBufRight[i] = float32(inRight*dry + flangOutRight*wet)
	}
}
