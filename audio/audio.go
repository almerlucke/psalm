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
