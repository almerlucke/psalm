package audio

type Engine interface {
	GetSampleRate() float64
	GetBufferSize() int
	SetCallback(func(in, out [][]float32))
	Init() error
	Start() error
	Stop() error
	Close() error
}

// InputSelector is an engine that can choose which input device it listens to (the microphone, the headset, the line in) while it is open
type InputSelector interface {
	// InputDevices are the names of the devices there are to listen to, "default" (the system's) first
	InputDevices() []string
	// InputDevice is the name of the one listened to
	InputDevice() string
	// SetInputDevice listens to another: the stream is opened again (a moment of silence), so call it from a goroutine of its own and not from the audio callback
	SetInputDevice(name string) error
}
