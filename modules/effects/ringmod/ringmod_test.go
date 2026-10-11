package ringmod

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func render(r *RingMod, freq float64) []float64 {
	bs, sr := r.Config().BufferSize, r.Config().SampleRate
	var out []float64
	n := 0
	for b := range int(2 * sr / float64(bs)) {
		for i := range bs {
			x := float32(0.5 * math.Sin(2*math.Pi*freq*float64(n)/sr))
			r.Input(0).Channels[0][i], r.Input(0).Channels[1][i] = x, x
			n++
		}
		r.Synthesize()
		if b > int(0.5*sr/float64(bs)) {
			for _, x := range r.Output(0).Channels[0] {
				out = append(out, float64(x))
			}
		}
	}
	return out
}

// level returns the level of a frequency in dB (Goertzel, over a whole number of its cycles)
func level(x []float64, freq, sr float64) float64 {
	n := int(math.Floor(float64(len(x))*freq/sr) * sr / freq)
	w := 2 * math.Pi * freq / sr
	c := 2 * math.Cos(w)
	var s1, s2 float64
	for _, v := range x[:n] {
		s1, s2 = v+c*s1-s2, s1
	}
	return 20 * math.Log10(2*math.Sqrt(s1*s1+s2*s2-c*s1*s2)/float64(n)+1e-12)
}

func TestRing(t *testing.T) {
	psalm.New(nil)
	sr := psalm.SampleRate()
	out := render(New(Config{Mode: Ring, Freq: 100, Mix: 1}), 1000)
	sum, diff, orig := level(out, 1100, sr), level(out, 900, sr), level(out, 1000, sr)
	if math.Abs(sum-diff) > 0.5 || orig > sum-40 {
		t.Fatalf("ring modulation: 900 and 1100 Hz as loud, 1000 gone: %.1f %.1f %.1f dB", diff, sum, orig)
	}
}

func TestShift(t *testing.T) {
	psalm.New(nil)
	sr := psalm.SampleRate()
	for _, f := range []float64{200, 1000, 5000} {
		out := render(New(Config{Mode: Shift, Freq: 100, Mix: 1}), f)
		up, image := level(out, f+100, sr), level(out, f-100, sr)
		if up < -7 || image > up-30 {
			t.Fatalf("%v Hz shifted up 100 Hz: %.1f dB there, the other side %.1f dB", f, up, image)
		}
	}
	out := render(New(Config{Mode: Shift, Freq: -150, Mix: 1}), 1000)
	if down, image := level(out, 850, sr), level(out, 1150, sr); image > down-30 {
		t.Fatalf("a negative shift goes down: %.1f vs %.1f dB", down, image)
	}
}
