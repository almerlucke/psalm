package blosc

import (
	"github.com/almerlucke/psalm"

	bloscc "github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

type Blosc struct {
	*psalm.Module
	osc *bloscc.Blosc
}

func New(fc float64, phase float64, pw float64, amp float64, waveform bloscc.Waveform) *Blosc {
	b := &Blosc{
		Module: psalm.NewModule(
			[]int{psalm.Mono, psalm.Mono, psalm.Mono, psalm.Mono, psalm.Mono}, []int{psalm.Mono},
		),
		osc: bloscc.New(fc, psalm.SampleRate(), phase, pw, amp, waveform),
	}

	b.SetSelf(b)

	return b
}

func (b *Blosc) SetFrequency(fc float64) {
	b.osc.SetFrequency(fc, b.Config().SampleRate)
}

func (b *Blosc) SetPulseWidth(pw float64) {
	b.osc.SetPulseWidth(pw)
}

func (b *Blosc) SetWaveform(wf int) {
	b.osc.SetWaveform(bloscc.Waveform(wf))
}

//func (b *Blosc) Receive(value any, index int) {
//	if msg, ok := david.MsgVal(value); ok {
//		b.ReceiveMessage(msg)
//		return
//	}
//
//	switch index {
//	case 0:
//		b.SetFrequency(david.FltVal[float64](value))
//	case 1:
//		b.SetPulseWidth(david.FltVal[float64](value))
//	case 2:
//		if wf, ok := value.(bloscc.Waveform); ok {
//			b.SetWaveform(int(wf))
//		} else {
//			b.SetWaveform(david.IntVal[int](value))
//		}
//	}
//}
//
//func (b *Blosc) ReceiveMessage(msg david.Message) {
//	msg.IfFltVal("frequency", func(v float64) {
//		b.SetFrequency(v)
//	})
//	msg.IfFltVal("pulseWidth", func(v float64) {
//		b.SetPulseWidth(v)
//	})
//	msg.IfIntVal("waveform", func(v int) {
//		b.SetWaveform(v)
//	})
//}

func (b *Blosc) Synthesize() {
	config := b.Config()
	output := b.Output(0).Channels[0]
	phaseMod := 0.0
	syncPhase := 0.0
	sync := false

	for i := 0; i < config.BufferSize; i++ {
		//if b.Input(0).IsConnected() {
		//	b.osc.SetFrequency(b.Input(0).Channels[0][i], config.SampleRate)
		//}
		//
		//if b.Input(1).IsConnected() {
		//	phaseMod = b.Input(1).Channels[0][i]
		//}
		//
		//if b.Input(2).IsConnected() {
		//	b.osc.SetPulseWidth(b.Input(2).Channels[0][i])
		//}
		//
		//if b.Input(3).IsConnected() {
		//	if b.Input(3).Channels[0][i] == 1.0 {
		//		sync = true
		//	}
		//}
		//
		//if b.Input(4).IsConnected() {
		//	if sync {
		//		syncPhase = b.Input(4).Channels[0][i]
		//	}
		//}

		output[i] = float32(b.osc.Tick(phaseMod, sync, syncPhase))
	}
}
