package oversample

import (
	"math"
	"testing"
)

// passing a sine through with nothing in between keeps it: the same level, delayed by the latency
func TestTransparent(t *testing.T) {
	const sr = 44100.0
	for _, f := range []float64{100, 1000, 10000, 16000} {
		o := New(4, TapsPerPhase)
		var in, out []float64
		for n := range 8000 {
			x := math.Sin(2 * math.Pi * f * float64(n) / sr)
			in = append(in, x)
			out = append(out, o.Process(x, func(v float64) float64 { return v }))
		}
		lat := o.Latency()
		peak, errMax := 0.0, 0.0
		for n := 2000; n < 8000; n++ {
			peak = math.Max(peak, math.Abs(out[n]))
			errMax = math.Max(errMax, math.Abs(out[n]-in[n-lat]))
		}
		if math.Abs(20*math.Log10(peak)) > 0.5 {
			t.Errorf("%v Hz: the level should stay, %.2f dB", f, 20*math.Log10(peak))
		}
		if f <= 10000 && errMax > 0.02 {
			t.Errorf("%v Hz: the same signal, %d samples late (error %.4f)", f, lat, errMax)
		}
	}
}
