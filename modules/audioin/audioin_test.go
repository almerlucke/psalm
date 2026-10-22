package audioin

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

type source struct{ *psalm.Module }

func TestInputReadsTheSource(t *testing.T) {
	src := &source{psalm.NewModule(nil, []int{psalm.Stereo})}
	src.SetSelf(src)
	in := New(src, Config{Gain: 0, Mono: true})
	for i := range src.Output(0).Channels[0] {
		src.Output(0).Channels[0][i] = 0.5
		src.Output(0).Channels[1][i] = 0.1
	}
	for range 40 { // the gain settles
		in.Synthesize()
	}
	out := in.Output(0).Channels
	last := len(out[0]) - 1
	if math.Abs(float64(out[0][last])-0.5) > 1e-3 || math.Abs(float64(out[1][last])-0.5) > 1e-3 {
		t.Errorf("mono: %v %v, want 0.5 on both sides", out[0][last], out[1][last])
	}
	in.SetMono(false)
	in.SetGain(-6.0206)
	for range 40 {
		in.Synthesize()
	}
	if math.Abs(float64(out[0][last])-0.25) > 1e-3 || math.Abs(float64(out[1][last])-0.05) > 1e-3 {
		t.Errorf("stereo, -6 dB: %v %v, want 0.25 and 0.05", out[0][last], out[1][last])
	}
}

func TestSilentWithoutSource(t *testing.T) {
	in := New(nil, DefaultConfig())
	in.Synthesize()
	for _, v := range in.Output(0).Channels[0] {
		if v != 0 {
			t.Fatal("not silent")
		}
	}
}

func TestDescribe(t *testing.T) {
	in := New(nil, DefaultConfig())
	paramstest.Check(t, func() params.Set { return Describe(in.Settings()) }, in.ReceiveMessage)
}
