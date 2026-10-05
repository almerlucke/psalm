package pitchshift

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm/blocks/fft"
)

// bandFraction returns the fraction of the spectral energy of the last 8192 samples within
// +-width Hz of freq
func bandFraction(x []float64, sr float64, freq float64, width float64) float64 {
	const n = 8192
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	for i := range n {
		re[i] = x[len(x)-n+i] * w[i]
	}
	f.Forward(re, im)

	band, total := 0.0, 0.0
	for k := 1; k < n/2; k++ {
		e := re[k]*re[k] + im[k]*im[k]
		total += e
		if math.Abs(float64(k)*sr/n-freq) <= width {
			band += e
		}
	}
	return band / total
}

func TestShiftsBothChannels(t *testing.T) {
	p := New(12, 50, 1.0)
	sr := p.Config().SampleRate
	bufSize := p.Config().BufferSize

	var left, right []float64
	for b := range 40 {
		for i := range bufSize {
			n := float64(b*bufSize + i)
			p.Input(0).Channels[0][i] = float32(math.Sin(2 * math.Pi * 300 * n / sr))
			p.Input(0).Channels[1][i] = float32(math.Sin(2 * math.Pi * 500 * n / sr))
		}
		p.Synthesize()
		for i := range bufSize {
			left = append(left, float64(p.Output(0).Channels[0][i]))
			right = append(right, float64(p.Output(0).Channels[1][i]))
		}
	}

	// an octave up, within the head rate sidebands of the shifter
	if f := bandFraction(left, sr, 600, 30); f < 0.8 {
		t.Fatalf("left: only %.2f of the energy near 600 Hz", f)
	}
	if f := bandFraction(right, sr, 1000, 30); f < 0.8 {
		t.Fatalf("right: only %.2f of the energy near 1000 Hz", f)
	}
}
