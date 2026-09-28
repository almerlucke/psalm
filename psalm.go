package psalm

import (
	"errors"
	"log"
	"math"
	"syscall"

	"github.com/almerlucke/psalm/audio"
	"github.com/almerlucke/psalm/tools/signal"
	"github.com/almerlucke/sndfile/writer"
	"github.com/dh1tw/gosamplerate"
)

type Psalm struct {
	*Patch

	// Control
	samplesPerTick  float64
	sampleCnt       float64
	sampleTimestamp int64
	tickTimestamp   int64

	// Audio
	engine audio.Engine
}

func New(engine audio.Engine) *Psalm {
	p := &Psalm{
		engine: engine,
	}

	if engine != nil {
		engine.SetCallback(p.audioCallback)
		err := engine.Init()
		if err != nil {
			panic(err)
		}
		PushConfig(Config{
			SampleRate: engine.GetSampleRate(),
			BufferSize: engine.GetBufferSize(),
			BPM:        BPM(),
			PPQ:        PPQ(),
		})
	}

	p.Patch = NewPatch([]int{2}, []int{2})
	p.SetSelf(p)
	p.SetID("psalm")
	p.SetSkipInputs(true)

	AddConfigListener(p)

	p.updateCtrlTick()

	return p
}

func (p *Psalm) OnConfigChanged() {
	p.updateCtrlTick()
}

func (p *Psalm) updateCtrlTick() {
	p.samplesPerTick = TickDuration() * p.config.SampleRate
}

func (p *Psalm) ToSoundFile(filePath string, fileFormat writer.FileFormat, numSeconds float64, sampleRate float64, normalize bool) error {
	config := p.config
	inputSampleRate := config.SampleRate
	numChannels := 2
	writeBuffers := make([][]float32, numChannels)
	pipe := p.Output(0).Channels

	for i := 0; i < numChannels; i++ {
		writeBuffers[i] = pipe[i]
	}

	wr, err := writer.NewWithOptions(filePath, fileFormat, numChannels, sampleRate, writer.Options{
		InputConverter:    writer.NewChannelConverter[float32](config.BufferSize, 2),
		Normalize:         normalize,
		ConvertSampleRate: inputSampleRate != sampleRate,
		InputSampleRate:   inputSampleRate,
		SrConvQuality:     gosamplerate.SRC_SINC_BEST_QUALITY,
	})
	if err != nil {
		return err
	}

	defer func() {
		_ = wr.Close()
	}()

	numCycles := int64(math.Ceil((config.SampleRate * numSeconds) / float64(config.BufferSize)))

	if numCycles == 0 {
		return nil
	}

	for numCycles > 1 {
		p.ResetRun()
		p.Run()
		err = wr.Write(writeBuffers, false)
		if err != nil {
			return err
		}
		numCycles--
	}

	p.ResetRun()
	p.Run()

	return wr.Write(writeBuffers, true)
}

func (p *Psalm) AudioStart() error {
	if p.engine == nil {
		return errors.New("engine not initialized")
	}
	return p.engine.Start()
}

func (p *Psalm) AudioStop() error {
	if p.engine == nil {
		return errors.New("engine not initialized")
	}

	return p.engine.Stop()
}

func (p *Psalm) Listen() error {
	err := p.AudioStart()
	if err != nil {
		return err
	}

	log.Printf("playing audio until terminated...")

	signal.WaitForSignal(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Printf("stopping audio")

	return p.AudioStop()
}

func (p *Psalm) Close() {
	if p.engine != nil {
		_ = p.engine.Close()
	}
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
	for n := range p.config.BufferSize {
		p.sampleTimestamp++
		p.sampleCnt += 1.0
		for p.sampleCnt >= p.samplesPerTick {
			p.sampleCnt -= p.samplesPerTick
			p.tickTimestamp++
			p.CtrlTick(CtrlTick{
				SampleTimestamp: p.sampleTimestamp,
				TickTimestamp:   p.tickTimestamp,
				SampleOffset:    n,
			})
		}
	}

	p.Patch.Run()
}
