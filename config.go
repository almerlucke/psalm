package psalm

import "container/list"

type Listener interface {
	OnConfigChanged()
}

const (
	DefaultSampleRate = 44100.0
	DefaultBufferSize = 1024
	DefaultBPM        = 120
	DefaultPPQ        = 240
)

var _configList *list.List
var _configListeners = make(map[Listener]bool)

type Config struct {
	SampleRate float64
	BufferSize int
	BPM        int
	PPQ        int
}

func init() {
	_configList = list.New()
	_configList.PushFront(Config{
		SampleRate: DefaultSampleRate,
		BufferSize: DefaultBufferSize,
		BPM:        DefaultBPM,
		PPQ:        DefaultPPQ,
	})
}

func AddConfigListener(listener Listener) {
	_configListeners[listener] = true
}

func RemoveConfigListener(listener Listener) {
	delete(_configListeners, listener)
}

func PushConfig(config Config) {
	_configList.PushFront(config)
	for listener := range _configListeners {
		listener.OnConfigChanged()
	}
}

func PopConfig() Config {
	var cfg Config
	front := _configList.Front()
	if front == nil {
		return cfg
	}
	cfg = front.Value.(Config)
	_configList.Remove(front)
	for listener := range _configListeners {
		listener.OnConfigChanged()
	}
	return cfg
}

func CurrentConfig() Config {
	var cfg Config
	front := _configList.Front()
	if front == nil {
		return cfg
	}
	return front.Value.(Config)
}

func SampleRate() float64 {
	config := CurrentConfig()
	return config.SampleRate
}

func BufferSize() int {
	config := CurrentConfig()
	return config.BufferSize
}

func BPM() int {
	config := CurrentConfig()
	return config.BPM
}

func PPQ() int {
	config := CurrentConfig()
	return config.PPQ
}

// TickDuration returns the duration of a single control tick in seconds.
func TickDuration() float64 {
	return 60.0 / (float64(BPM()) * float64(PPQ()))
}
