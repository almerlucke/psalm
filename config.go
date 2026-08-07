package psalm

import "container/list"

const (
	DefaultSampleRate = 44100.0
	DefaultBufferSize = 1024
	DefaultBPM        = 120
	DefaultPPQ        = 240
)

var _configList *list.List

type Config struct {
	SampleRate float64
	BufferSize int
}

func init() {
	_configList = list.New()
	_configList.PushFront(&Config{
		SampleRate: DefaultSampleRate,
		BufferSize: DefaultBufferSize,
	})
}

func PushConfig(config *Config) {
	_configList.PushFront(config)
}

func PopConfig() *Config {
	front := _configList.Front()
	if front == nil {
		return nil
	}
	config := front.Value.(*Config)
	_configList.Remove(front)
	return config
}

func CurrentConfig() *Config {
	front := _configList.Front()
	if front == nil {
		return nil
	}
	return front.Value.(*Config)
}

func SampleRate() float64 {
	config := CurrentConfig()
	if config != nil {
		return config.SampleRate
	}
	return DefaultSampleRate
}

func BufferSize() int {
	config := CurrentConfig()
	if config != nil {
		return config.BufferSize
	}
	return DefaultBufferSize
}
