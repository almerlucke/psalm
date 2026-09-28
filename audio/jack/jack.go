package jackengine

import (
	"fmt"

	"github.com/almerlucke/go-jack"
)

type Engine struct {
	client     *jack.Client
	portsIn    []*jack.Port
	portsOut   []*jack.Port
	inBuffers  [][]float32
	outBuffers [][]float32
	callback   func(in, out [][]float32)
}

func New(name string) *Engine {
	client, status := jack.ClientOpen(name, jack.NoStartServer)
	if status != 0 {
		panic(jack.StrError(status))
	}

	portsIn := make([]*jack.Port, 2)
	portsOut := make([]*jack.Port, 2)

	for i := range 2 {
		portIn := client.PortRegister(fmt.Sprintf("in_%d", i), jack.DEFAULT_AUDIO_TYPE, jack.PortIsInput, 0)
		portOut := client.PortRegister(fmt.Sprintf("out_%d", i), jack.DEFAULT_AUDIO_TYPE, jack.PortIsOutput, 0)
		portsIn[i] = portIn
		portsOut[i] = portOut
	}

	e := &Engine{
		client:     client,
		portsIn:    portsIn,
		portsOut:   portsOut,
		inBuffers:  make([][]float32, 2),
		outBuffers: make([][]float32, 2),
	}

	client.SetProcessCallback(e.process)

	return e
}

func (e *Engine) GetSampleRate() float64 {
	return float64(e.client.GetSampleRate())
}

func (e *Engine) GetBufferSize() int {
	return int(e.client.GetBufferSize())
}

func (e *Engine) process(nframes uint32) int {
	if e.callback != nil {
		for i, in := range e.portsIn {
			e.inBuffers[i] = in.GetBuffer(nframes)
		}
		for i, out := range e.portsOut {
			e.outBuffers[i] = out.GetBuffer(nframes)
		}
		e.callback(e.inBuffers, e.outBuffers)
	}
	return 0
}

func (e *Engine) SetCallback(callback func(in, out [][]float32)) {
	e.callback = callback
}

func (e *Engine) Init() error {
	// STUB
	return nil
}

func (e *Engine) Start() error {
	status := e.client.Activate()
	if status != 0 {
		return jack.StrError(status)
	}
	return nil
}

func (e *Engine) Stop() error {
	status := e.client.Deactivate()
	if status != 0 {
		return jack.StrError(status)
	}
	return nil
}

func (e *Engine) Close() error {
	status := e.client.Close()
	if status != 0 {
		return jack.StrError(status)
	}
	return nil
}
