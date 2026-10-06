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
	BPM        float64
	PPQ        int
}

func (c Config) ControlRate() float64 {
	return 1.0 / c.TickDuration()
}

func (c Config) TickDuration() float64 {
	return 60.0 / (c.BPM * float64(c.PPQ))
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

// PushConfig sets the config of the current environment (the patch being built, or the next
// psalm.New), keeping the one before on a stack for PopConfig
func PushConfig(config Config) {
	_configList.PushFront(config)
	CurrentEnv().setConfig(config)
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
	CurrentEnv().setConfig(CurrentStackConfig())
	for listener := range _configListeners {
		listener.OnConfigChanged()
	}
	return cfg
}

// CurrentConfig returns the config of the current environment, with its current tempo
func CurrentConfig() Config {
	return CurrentEnv().Config()
}

// CurrentStackConfig returns the config on top of the PushConfig stack
func CurrentStackConfig() Config {
	var cfg Config
	front := _configList.Front()
	if front == nil {
		return cfg
	}
	return front.Value.(Config)
}

func SampleRate() float64 {
	return CurrentConfig().SampleRate
}

func ControlRate() float64 {
	return CurrentConfig().ControlRate()
}

func BufferSize() int {
	return CurrentConfig().BufferSize
}

func BPM() float64 {
	return CurrentConfig().BPM
}

func PPQ() int {
	return CurrentConfig().PPQ
}

// TickDuration returns the duration of a single control tick in seconds.
func TickDuration() float64 {
	return CurrentConfig().TickDuration()
}
