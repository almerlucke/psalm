package latency

import "github.com/almerlucke/psalm"

// Latency delays its input by a fixed number of samples, used to line up a signal with one that
// went through a module with latency (like the vocoder)
type Latency struct {
	*psalm.Module
	buffers [][]float32
	pos     int
}

func New(numChannels int, samples int) *Latency {
	l := &Latency{
		Module:  psalm.NewModule([]int{numChannels}, []int{numChannels}),
		buffers: make([][]float32, numChannels),
	}

	for i := range l.buffers {
		l.buffers[i] = make([]float32, max(samples, 0))
	}

	l.SetSelf(l)

	return l
}

func (l *Latency) Synthesize() {
	ins := l.Input(0).Channels
	outs := l.Output(0).Channels
	n := len(l.buffers[0])

	if n == 0 {
		outs.Copy(ins)
		return
	}

	for i := range l.Config().BufferSize {
		for c, buf := range l.buffers {
			outs[c][i] = buf[l.pos]
			buf[l.pos] = ins[c][i]
		}
		l.pos = (l.pos + 1) % n
	}
}
