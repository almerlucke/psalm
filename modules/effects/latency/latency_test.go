package latency

import (
	"testing"

	"github.com/almerlucke/psalm"
)

func TestDelaysBySamples(t *testing.T) {
	for _, delay := range []int{0, 1, 100, 1024, 3000} {
		l := New(psalm.Stereo, delay)
		bufSize := l.Config().BufferSize

		var out []float32
		for b := range 5 {
			for i := range bufSize {
				v := float32(b*bufSize + i + 1)
				l.Input(0).Channels[0][i] = v
				l.Input(0).Channels[1][i] = -v
			}
			l.Synthesize()
			out = append(out, l.Output(0).Channels[0]...)
			if r := l.Output(0).Channels[1]; r[bufSize-1] != -out[len(out)-1] {
				t.Fatalf("delay %d: right channel mismatch", delay)
			}
		}

		for i, v := range out {
			want := float32(0)
			if i >= delay {
				want = float32(i - delay + 1)
			}
			if v != want {
				t.Fatalf("delay %d: sample %d got %v want %v", delay, i, v, want)
			}
		}
	}
}
