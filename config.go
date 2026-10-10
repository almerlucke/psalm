package psalm

import (
	"container/list"
	"sync"
)

type Listener interface {
	OnConfigChanged()
}

const (
	DefaultSampleRate = 44100.0
	DefaultBufferSize = 1024
	DefaultBPM        = 120
	DefaultPPQ        = 240
)

var (
	configMu         sync.Mutex // guards the stack and the listeners: configs are pushed from any goroutine
	_configList      *list.List
	_configListeners = make(map[Listener]bool)
)

// configChanged tells the listeners, outside the lock
func configChanged() {
	configMu.Lock()
	listeners := make([]Listener, 0, len(_configListeners))
	for l := range _configListeners {
		listeners = append(listeners, l)
	}
	configMu.Unlock()
	for _, l := range listeners {
		l.OnConfigChanged()
	}
}

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
	configMu.Lock()
	_configListeners[listener] = true
	configMu.Unlock()
}

func RemoveConfigListener(listener Listener) {
	configMu.Lock()
	delete(_configListeners, listener)
	configMu.Unlock()
}

// PushConfig sets the config of the current environment (the patch being built, or the next
// psalm.New), keeping the one before on a stack for PopConfig. It can be called from any goroutine;
// the stack is one for the program, so push and pop in pairs
func PushConfig(config Config) {
	configMu.Lock()
	_configList.PushFront(config)
	configMu.Unlock()
	CurrentEnv().setConfig(config)
	configChanged()
}

func PopConfig() Config {
	configMu.Lock()
	var cfg Config
	front := _configList.Front()
	if front == nil {
		configMu.Unlock()
		return cfg
	}
	cfg = front.Value.(Config)
	_configList.Remove(front)
	configMu.Unlock()
	CurrentEnv().setConfig(CurrentStackConfig())
	configChanged()
	return cfg
}

// CurrentConfig returns the config of the current environment, with its current tempo
func CurrentConfig() Config {
	return CurrentEnv().Config()
}

// CurrentStackConfig returns the config on top of the PushConfig stack
func CurrentStackConfig() Config {
	configMu.Lock()
	defer configMu.Unlock()
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
