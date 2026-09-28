package paengine

import (
	"errors"
	"sync"

	"github.com/gordonklaus/portaudio"
)

type Engine struct {
	started     bool
	initialized bool
	initLock    sync.Mutex
	startLock   sync.Mutex
	paStream    *portaudio.Stream
	sampleRate  float64
	bufferSize  int
	callback    func(in, out [][]float32)
}

func New(sampleRate float64, bufferSize int) *Engine {
	return &Engine{
		sampleRate: sampleRate,
		bufferSize: bufferSize,
	}
}

func (e *Engine) GetSampleRate() float64 {
	return e.sampleRate
}

func (e *Engine) GetBufferSize() int {
	return e.bufferSize
}

func (e *Engine) SetCallback(callback func(in, out [][]float32)) {
	e.callback = callback
}

func (e *Engine) Init() error {
	if e.initialized {
		return nil
	}

	if e.callback == nil {
		return errors.New("callback function is not set")
	}

	e.initLock.Lock()
	defer e.initLock.Unlock()

	err := portaudio.Initialize()
	if err != nil {
		return err
	}

	paStream, err := portaudio.OpenDefaultStream(
		2,
		2,
		e.sampleRate,
		e.bufferSize,
		e.callback,
	)

	if err != nil {
		_ = portaudio.Terminate()
		return err
	}

	e.initialized = true
	e.paStream = paStream

	return nil
}

func (e *Engine) Start() error {
	if e.started {
		return nil
	}

	if !e.initialized {
		return errors.New("not initialized")
	}

	e.startLock.Lock()
	defer e.startLock.Unlock()

	err := e.paStream.Start()
	if err != nil {
		return err
	}

	e.started = true

	return nil
}

func (e *Engine) Stop() error {
	if !e.started {
		return nil
	}

	e.startLock.Lock()
	defer e.startLock.Unlock()

	err := e.paStream.Stop()
	if err != nil {
		return err
	}

	e.started = false

	return nil
}

func (e *Engine) Close() error {
	if !e.initialized {
		return nil
	}

	var errs []error

	err := e.Stop()
	if err != nil {
		errs = append(errs, err)
	}
	err = e.paStream.Close()
	if err != nil {
		errs = append(errs, err)
	}
	err = portaudio.Terminate()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
