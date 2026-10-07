package jackengine

import (
	"fmt"

	"github.com/almerlucke/go-jack"
	"github.com/almerlucke/psalm/midi"
)

type Engine struct {
	client     *jack.Client
	portsIn    []*jack.Port
	portsOut   []*jack.Port
	inBuffers  [][]float32
	outBuffers [][]float32
	callback   func(in, out [][]float32)
	midiIns    []*MidiInput
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

// MidiIn registers a MIDI input port (call it before Start). Its events come with the frame they
// fall on in the buffer: sample accurate
func (e *Engine) MidiIn(name string) (*MidiInput, error) {
	port := e.client.PortRegister(name, jack.DEFAULT_MIDI_TYPE, jack.PortIsInput, 0)
	if port == nil {
		return nil, fmt.Errorf("jack: can't register midi port %q", name)
	}
	in := &MidiInput{port: port}
	e.midiIns = append(e.midiIns, in)
	return in, nil
}

// MidiInput is a JACK MIDI input port, a midi.Input
type MidiInput struct {
	port    *jack.Port
	pending []midi.Event
	stale   int // how many of the pending events are from buffers before the last
}

// capture takes the events of the port for a buffer (in the process callback, before the patch
// runs). Events not read in an earlier buffer stay, at the start of this one
func (m *MidiInput) capture(nframes uint32) {
	m.stale = len(m.pending)
	for _, ev := range m.port.GetMidiEvents(nframes) {
		m.pending = append(m.pending, midi.Event{Message: midi.Message(ev.Buffer), Offset: int(ev.Time)})
	}
}

// Read implements midi.Input
func (m *MidiInput) Read(events []midi.Event, frames int, _ float64) []midi.Event {
	for i, ev := range m.pending {
		if i < m.stale {
			ev.Offset = 0
		}
		ev.Offset = max(0, min(ev.Offset, frames-1))
		events = append(events, ev)
	}
	clear(m.pending)
	m.pending = m.pending[:0]
	m.stale = 0
	return events
}

func (e *Engine) process(nframes uint32) int {
	for _, in := range e.midiIns {
		in.capture(nframes)
	}
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
