package paengine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"

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
	input       string // the input device chosen ("" or "default": the system's)
	mono        [][]float32
	sources     map[string]string // the sound sources of PulseAudio (and PipeWire): what they are called here, and their names
}

// DefaultInput is the name of the input the system chooses
const DefaultInput = "default"

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

	paStream, err := e.open()

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

// deviceName is how an input device is told apart: its name and the host api it is of
func deviceName(d *portaudio.DeviceInfo) string { return d.Name + " (" + d.HostApi.Name + ")" }

// pulseSources asks PulseAudio (PipeWire speaks it too) for its sound sources, the microphones and lines in (not what the outputs play: the monitors): a name for people, and
// the name the server knows it by. It uses pactl when the computer has it; there are none without
func pulseSources() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pactl", "-f", "json", "list", "sources")
	cmd.Env = append(os.Environ(), "LC_ALL=C") // numbers with a point: in some languages pactl writes 0,00 and that is no json
	b, err := cmd.Output()
	if err != nil {
		return nil
	}
	var list []struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Monitor     *string `json:"monitor_of_sink"`
	}
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	out := map[string]string{}
	for _, s := range list {
		if s.Monitor != nil || strings.HasSuffix(s.Name, ".monitor") {
			continue
		}
		label := s.Description + " (pulse)"
		if _, dup := out[label]; dup {
			label = s.Name + " (pulse)"
		}
		out[label] = s.Name
	}
	return out
}

// InputDevices are the input devices to listen to (after Init), the system's default first: the sound sources of PulseAudio or PipeWire by name (a headset, the microphone of the
// computer), then the devices of the host apis that run at the sample rate of the engine
func (e *Engine) InputDevices() []string {
	e.initLock.Lock()
	defer e.initLock.Unlock()
	if !e.initialized {
		return []string{DefaultInput}
	}
	out := []string{DefaultInput}
	e.sources = pulseSources()
	var names []string
	for label := range e.sources {
		names = append(names, label)
	}
	slices.Sort(names)
	out = append(out, names...)
	devs, err := portaudio.Devices()
	if err != nil {
		return out
	}
	for _, d := range devs {
		if d.MaxInputChannels > 0 && d.DefaultSampleRate == e.sampleRate && d.Name != "default" && d.Name != "pulse" && d.Name != "pipewire" && d.Name != "jack" {
			out = append(out, deviceName(d))
		}
	}
	return out
}

// InputDevice is the input device listened to
func (e *Engine) InputDevice() string {
	if e.input == "" {
		return DefaultInput
	}
	return e.input
}

// SetInputDevice listens to another input device: the stream is opened again at once when the engine is open (and started again when it played). A device with one channel
// (a headset's microphone) is heard on both sides. Call it from a goroutine of its own, never from the audio callback
func (e *Engine) SetInputDevice(name string) error {
	if name == "" {
		name = DefaultInput
	}
	if name != DefaultInput {
		found := false
		for _, n := range e.InputDevices() {
			found = found || n == name
		}
		if !found && e.initialized {
			return errors.New("no input device " + name)
		}
	}
	e.startLock.Lock()
	defer e.startLock.Unlock()
	old := e.input
	e.input = name
	if !e.initialized {
		return nil
	}
	was := e.started
	if e.started {
		_ = e.paStream.Stop()
		e.started = false
	}
	_ = e.paStream.Close()
	stream, err := e.open()
	if err != nil { // the one before stays
		e.input = old
		stream, err2 := e.open()
		if err2 == nil {
			e.paStream = stream
			if was {
				_ = stream.Start()
				e.started = true
			}
		}
		return err
	}
	e.paStream = stream
	if was {
		if err := stream.Start(); err != nil {
			return err
		}
		e.started = true
	}
	return nil
}

// open opens the stream: the default device, or the input chosen with the output of its host api (a stream has both of one api), two channels out and one or two in
func (e *Engine) open() (*portaudio.Stream, error) {
	if e.input == "" || e.input == DefaultInput {
		_ = os.Unsetenv("PULSE_SOURCE")
		return portaudio.OpenDefaultStream(2, 2, e.sampleRate, e.bufferSize, e.callback)
	}
	devs, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}
	if e.sources == nil && e.input != "" && e.input != DefaultInput {
		e.sources = pulseSources()
	}
	if source, ok := e.sources[e.input]; ok { // a source of PulseAudio: its "pulse" device listens to the one the environment names
		for _, d := range devs {
			if d.Name == "pulse" && d.MaxInputChannels > 0 && d.HostApi.DefaultOutputDevice != nil {
				_ = os.Setenv("PULSE_SOURCE", source)
				p := portaudio.LowLatencyParameters(d, d.HostApi.DefaultOutputDevice)
				p.Input.Channels, p.Output.Channels = 2, 2
				p.SampleRate, p.FramesPerBuffer = e.sampleRate, e.bufferSize
				return portaudio.OpenStream(p, e.callback)
			}
		}
		return nil, errors.New("PulseAudio is not there to listen to " + e.input)
	}
	_ = os.Unsetenv("PULSE_SOURCE")
	for _, d := range devs {
		if d.MaxInputChannels == 0 || deviceName(d) != e.input {
			continue
		}
		out := d.HostApi.DefaultOutputDevice
		if out == nil {
			return nil, errors.New(e.input + ": its host api has no output device")
		}
		chans := min(2, d.MaxInputChannels)
		p := portaudio.LowLatencyParameters(d, out)
		p.Input.Channels, p.Output.Channels = chans, 2
		p.SampleRate, p.FramesPerBuffer = e.sampleRate, e.bufferSize
		cb := e.callback
		if chans == 1 { // heard on both sides
			e.mono = make([][]float32, 2)
			return portaudio.OpenStream(p, func(in, out [][]float32) {
				e.mono[0], e.mono[1] = in[0], in[0]
				cb(e.mono, out)
			})
		}
		return portaudio.OpenStream(p, cb)
	}
	return nil, errors.New("no input device " + e.input)
}
