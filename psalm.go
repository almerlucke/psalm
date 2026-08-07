package psalm

import (
	"fmt"
	"log"
	"sync/atomic"
	"syscall"

	"github.com/almerlucke/psalm/internal/signal"

	"github.com/gordonklaus/portaudio"
)

type Psalm struct {
	*Patch

	// Control
	ppq             int // 120.0 pulses per quarter note
	bpm             int // 120.0 beats per minute
	samplesPerTick  float64
	sampleCnt       float64
	sampleTimestamp int64
	tickTimestamp   int64

	// Audio
	started     atomic.Bool
	initialized atomic.Bool
	paStream    *portaudio.Stream
}

func New() *Psalm {
	p := &Psalm{
		Patch: NewPatch([]int{2}, []int{2}),
		bpm:   DefaultBPM,
		ppq:   DefaultPPQ,
	}

	p.SetSelf(p)
	p.SetID("psalm")
	p.SetSkipInputs(true)
	p.updateCtrlTick()

	return p
}

func (p *Psalm) BPM() int {
	return p.bpm
}

func (p *Psalm) SetBPM(bpm int) {
	p.bpm = bpm
	p.updateCtrlTick()
}

func (p *Psalm) PPQ() int {
	return p.ppq
}

func (p *Psalm) SetPPQ(ppq int) {
	p.ppq = ppq
	p.updateCtrlTick()
}

func (p *Psalm) updateCtrlTick() {
	tickDuration := 60.0 / (float64(p.bpm) * float64(p.ppq))
	p.samplesPerTick = tickDuration * p.config.SampleRate
}

func (p *Psalm) InitializeAudio() error {
	if p.initialized.Load() {
		return nil
	}

	err := portaudio.Initialize()
	if err != nil {
		return err
	}

	paStream, err := portaudio.OpenDefaultStream(
		2,
		2,
		p.config.SampleRate,
		p.config.BufferSize,
		p.audioCallback,
	)

	if err != nil {
		_ = portaudio.Terminate()
		return err
	}

	p.paStream = paStream
	p.initialized.Store(true)

	return nil
}

func (p *Psalm) TerminateAudio() {
	if !p.initialized.Load() {
		return
	}

	_ = p.paStream.Close()
	p.paStream = nil

	_ = portaudio.Terminate()

	p.initialized.Store(false)
}

func (p *Psalm) Start() error {
	if p.started.Load() {
		return nil
	}

	if !p.initialized.Load() {
		return fmt.Errorf("portaudio not initialized")
	}

	err := p.paStream.Start()
	if err != nil {
		return err
	}

	p.started.Store(true)

	return nil
}

func (p *Psalm) Stop() error {
	if !p.started.Load() {
		return nil
	}

	err := p.paStream.Stop()
	if err != nil {
		return err
	}

	p.started.Store(false)

	return nil
}

func (p *Psalm) Listen() error {
	err := p.InitializeAudio()
	if err != nil {
		return err
	}

	defer p.TerminateAudio()

	err = p.Start()
	if err != nil {
		return err
	}

	log.Printf("playing audio until terminated...")

	signal.WaitForSignal(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Printf("stopping audio")

	return p.Stop()
}

func (p *Psalm) audioCallback(in, out [][]float32) {
	inputPipe := p.inputModules[0].Output(0).Channels

	for i := 0; i < p.config.BufferSize; i++ {
		for j := 0; j < 2; j++ {
			inputPipe[j][i] = in[j][i]
		}
	}

	p.ResetRun()
	p.Run()

	channels := p.outputs[0].Channels

	for i := 0; i < p.config.BufferSize; i++ {
		for j := 0; j < 2; j++ {
			out[j][i] = channels[j][i]
		}
	}
}

func (p *Psalm) Run() {
	for range p.config.BufferSize {
		p.sampleTimestamp++
		p.sampleCnt += 1.0
		for p.sampleCnt >= p.samplesPerTick {
			p.sampleCnt -= p.samplesPerTick
			p.tickTimestamp++
			p.CtrlTick(CtrlTick{
				SampleTimestamp: p.sampleTimestamp,
				TickTimestamp:   p.tickTimestamp,
			})
		}
	}

	p.Patch.Run()
}

func (p *Psalm) Render(file string) error {

	return nil
}
