package blosc

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	bloscc "github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

// source is a test module that outputs f(sample index)
type source struct {
	*psalm.Module
	f func(n int) float64
	n int
}

func newSource(f func(n int) float64) *source {
	s := &source{Module: psalm.NewModule(nil, []int{psalm.Mono}), f: f}
	s.SetSelf(s)
	return s
}

func (s *source) Synthesize() {
	out := s.Output(0).Channels[0]
	for i := range out {
		out[i] = float32(s.f(s.n))
		s.n++
	}
}

// render runs the oscillator (and the modules connected to it) for a number of buffers
func render(b *Blosc, numBuffers int, sources ...*source) []float64 {
	var out []float64
	for range numBuffers {
		b.ResetRun()
		for _, s := range sources {
			s.ResetRun()
		}
		b.Run()
		for _, v := range b.Output(0).Channels[0] {
			out = append(out, float64(v))
		}
	}
	return out
}

// frequency estimates the frequency from rising zero crossings
func frequency(x []float64, sr float64) float64 {
	crossings := 0
	for i := 1; i < len(x); i++ {
		if x[i-1] < 0 && x[i] >= 0 {
			crossings++
		}
	}
	return float64(crossings) / (float64(len(x)) / sr)
}

const buffers = 43 // about one second at 44100 Hz with 1024 sample buffers

func TestFrequency(t *testing.T) {
	b := New(441, 0, 0.5, 1, bloscc.Sine)
	sr := b.Config().SampleRate

	if f := frequency(render(b, buffers), sr); math.Abs(f-441) > 3 {
		t.Fatalf("initial frequency %v", f)
	}

	b.Receive(882.0, InFrequency)
	if f := frequency(render(b, buffers), sr); math.Abs(f-882) > 3 {
		t.Fatalf("value input frequency %v", f)
	}

	b.ReceiveMessage(psalm.Message{"frequency": 220.5})
	if f := frequency(render(b, buffers), sr); math.Abs(f-220.5) > 3 {
		t.Fatalf("message frequency %v", f)
	}
}

func TestFrequencyAudioInput(t *testing.T) {
	b := New(100, 0, 0.5, 1, bloscc.Sine)
	src := newSource(func(int) float64 { return 661.5 })
	b.In(src, 0, InFrequency)

	if f := frequency(render(b, buffers, src), b.Config().SampleRate); math.Abs(f-661.5) > 3 {
		t.Fatalf("audio input frequency %v", f)
	}
}

func TestFMInputAddsHz(t *testing.T) {
	b := New(441, 0, 0.5, 1, bloscc.Sine)
	src := newSource(func(int) float64 { return 441 })
	b.In(src, 0, InFM)

	if f := frequency(render(b, buffers, src), b.Config().SampleRate); math.Abs(f-882) > 3 {
		t.Fatalf("expected 441 + 441 Hz, got %v", f)
	}
}

func TestPulseWidthAudioInput(t *testing.T) {
	b := New(441, 0, 0.5, 1, bloscc.Rectangle)
	src := newSource(func(int) float64 { return 0.25 })
	b.In(src, 0, InPulseWidth)

	out := render(b, buffers, src)
	positive := 0
	for _, v := range out {
		if v > 0 {
			positive++
		}
	}
	if duty := float64(positive) / float64(len(out)); math.Abs(duty-0.25) > 0.03 {
		t.Fatalf("expected a 25%% duty cycle, got %v", duty)
	}
}

func TestSyncOnRisingEdges(t *testing.T) {
	// a 441 Hz sine synced by a 100 Hz pulse train restarts at phase 0.25 (sin = 1) after each edge
	b := New(441, 0, 0.5, 1, bloscc.Sine)
	sr := b.Config().SampleRate
	period := int(sr / 100)

	trig := newSource(func(n int) float64 {
		if n%period < period/2 {
			return 1
		}
		return 0
	})
	phase := newSource(func(int) float64 { return 0.25 })
	b.In(trig, 0, InSync, phase, 0, InSyncPhase)

	out := render(b, 10, trig, phase)

	for edge := period; edge+1 < len(out); edge += period {
		if math.Abs(out[edge+1]-1.0) > 1e-6 {
			t.Fatalf("expected the sample after the edge at %d to restart at phase 0.25, got %v", edge, out[edge+1])
		}
	}

	// between edges the oscillator keeps running (sync doesn't get stuck while the trigger is high)
	if math.Abs(out[period+20]-out[period+21]) < 1e-3 {
		t.Fatal("oscillator stopped after sync")
	}
}

func TestValueSyncHappensOnce(t *testing.T) {
	b := New(441, 0, 0.5, 1, bloscc.Sine)
	render(b, 1)

	b.Receive(0.25, InSyncPhase)
	b.Receive(1.0, InSync)
	out := render(b, 1)

	if math.Abs(out[1]-1.0) > 1e-6 {
		t.Fatalf("expected a restart at phase 0.25, got %v", out[1])
	}
	if math.Abs(out[2]-out[1]) < 1e-6 && math.Abs(out[3]-out[1]) < 1e-6 {
		t.Fatal("sync stuck after a single value sync")
	}
}

func TestWaveformAndAmplitudeInputs(t *testing.T) {
	b := New(441, 0, 0.5, 1, bloscc.Sine)
	b.Receive(0.5, InAmplitude)

	peak := 0.0
	for _, v := range render(b, 4) {
		peak = math.Max(peak, math.Abs(v))
	}
	if math.Abs(peak-0.5) > 0.01 {
		t.Fatalf("expected amplitude 0.5, got peak %v", peak)
	}

	// a square wave mostly sits at +-amplitude, a sine doesn't
	b.Receive(bloscc.Square, InWaveform)
	out := render(b, 4)
	atAmp := 0
	for _, v := range out {
		if math.Abs(math.Abs(v)-0.5) < 0.01 {
			atAmp++
		}
	}
	if float64(atAmp)/float64(len(out)) < 0.8 {
		t.Fatal("expected a square wave after changing the waveform")
	}
}
