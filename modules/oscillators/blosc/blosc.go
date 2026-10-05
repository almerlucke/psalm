package blosc

import (
	"github.com/almerlucke/psalm"

	bloscc "github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

// Input indices, audio and value inputs with the same index control the same parameter
const (
	InFrequency  = 0 // audio: frequency in Hz (replaces the set frequency), value: set frequency
	InFM         = 1 // audio: linear FM in Hz added to the frequency (through zero)
	InPulseWidth = 2 // audio and value: pulse width 0-1
	InSync       = 3 // audio: sync on a rising edge above 0, value: any value syncs right away
	InSyncPhase  = 4 // audio and value: phase (0-1) the oscillator restarts at on sync
	InWaveform   = 5 // value: waveform
	InAmplitude  = 6 // value: amplitude
)

// Blosc is a band limited oscillator module with one mono audio output. Its five mono audio
// inputs modulate it per sample (frequency, FM, pulse width, sync trigger, sync phase), value
// inputs and messages set its parameters
type Blosc struct {
	*psalm.Module
	osc       *bloscc.Blosc
	syncPhase float64
	prevSync  float32
	syncNow   bool
}

func New(fc float64, phase float64, pw float64, amp float64, waveform bloscc.Waveform) *Blosc {
	b := &Blosc{
		Module: psalm.NewModule(
			[]int{psalm.Mono, psalm.Mono, psalm.Mono, psalm.Mono, psalm.Mono}, []int{psalm.Mono},
		),
	}

	b.osc = bloscc.New(fc, b.Config().SampleRate, phase, pw, amp, waveform)

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

func (b *Blosc) SetAmplitude(amp float64) {
	b.osc.SetAmplitude(amp)
}

// SetSyncPhase sets the phase (0-1) the oscillator restarts at when it is synced
func (b *Blosc) SetSyncPhase(phase float64) {
	b.syncPhase = phase
}

// Sync restarts the oscillator at the sync phase on the next sample
func (b *Blosc) Sync() {
	b.syncNow = true
}

func (b *Blosc) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		b.ReceiveMessage(msg)
		return
	}

	switch index {
	case InFrequency:
		b.SetFrequency(psalm.FltVal[float64](value))
	case InPulseWidth:
		b.SetPulseWidth(psalm.FltVal[float64](value))
	case InSync:
		b.Sync()
	case InSyncPhase:
		b.SetSyncPhase(psalm.FltVal[float64](value))
	case InWaveform:
		if wf, ok := value.(bloscc.Waveform); ok {
			b.SetWaveform(int(wf))
		} else {
			b.SetWaveform(psalm.IntVal[int](value))
		}
	case InAmplitude:
		b.SetAmplitude(psalm.FltVal[float64](value))
	}
}

func (b *Blosc) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("frequency", b.SetFrequency)
	msg.IfFltVal("pulseWidth", b.SetPulseWidth)
	msg.IfIntVal("waveform", b.SetWaveform)
	msg.IfFltVal("amplitude", b.SetAmplitude)
	msg.IfFltVal("syncPhase", b.SetSyncPhase)
	if _, ok := msg["sync"]; ok {
		b.Sync()
	}
}

func (b *Blosc) Synthesize() {
	var (
		config    = b.Config()
		sr        = config.SampleRate
		output    = b.Output(0).Channels[0]
		freqIn    = b.Input(InFrequency)
		fmIn      = b.Input(InFM)
		pwIn      = b.Input(InPulseWidth)
		syncIn    = b.Input(InSync)
		phaseIn   = b.Input(InSyncPhase)
		freqConn  = freqIn.IsConnected()
		fmConn    = fmIn.IsConnected()
		pwConn    = pwIn.IsConnected()
		syncConn  = syncIn.IsConnected()
		phaseConn = phaseIn.IsConnected()
	)

	for i := range config.BufferSize {
		if freqConn {
			b.osc.SetFrequency(float64(freqIn.Channels[0][i]), sr)
		}

		// FM in Hz becomes an extra phase increment per sample
		fm := 0.0
		if fmConn {
			fm = float64(fmIn.Channels[0][i]) / sr
		}

		if pwConn {
			b.osc.SetPulseWidth(float64(pwIn.Channels[0][i]))
		}

		if phaseConn {
			b.syncPhase = float64(phaseIn.Channels[0][i])
		}

		// sync on a rising edge of the sync input, or when requested through a value or message
		sync := b.syncNow
		b.syncNow = false
		if syncConn {
			v := syncIn.Channels[0][i]
			if v > 0 && b.prevSync <= 0 {
				sync = true
			}
			b.prevSync = v
		}

		output[i] = float32(b.osc.Tick(fm, sync, b.syncPhase))
	}
}
