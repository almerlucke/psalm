package pitchshift

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm/blocks/fft"
)

const sr = 44100.0

// peakFrequency returns the strongest frequency of the signal
func peakFrequency(x []float64) float64 {
	const n = 8192
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	for i := range n {
		re[i] = x[len(x)-n+i] * w[i]
	}
	f.Forward(re, im)

	best, bestMag := 0, 0.0
	for k := 1; k < n/2; k++ {
		if mag := re[k]*re[k] + im[k]*im[k]; mag > bestMag {
			best, bestMag = k, mag
		}
	}
	return float64(best) * sr / n
}

func shiftSine(semitones float64, freq float64) []float64 {
	p := New(semitones, 50, sr)
	out := make([]float64, int(sr))
	for i := range out {
		out[i] = p.Process(math.Sin(2 * math.Pi * freq * float64(i) / sr))
	}
	return out
}

// bandFraction returns the fraction of the spectral energy within +-width Hz of freq
func bandFraction(x []float64, freq float64, width float64) float64 {
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

func TestShiftsPitch(t *testing.T) {
	// the energy ends up around the shifted pitch (within the head rate sidebands, about 20 Hz for
	// an octave with a 50 ms window) and not at the original pitch, over a range of frequencies
	for _, semitones := range []float64{12, -12, 7} {
		for freq := 200.0; freq <= 1000; freq += 50 {
			out := shiftSine(semitones, freq)
			target := freq * math.Pow(2, semitones/12)

			if f := bandFraction(out, target, 30); f < 0.8 {
				t.Fatalf("%+v semitones, %v Hz: only %.2f of the energy near %.0f Hz", semitones, freq, f, target)
			}
			if f := bandFraction(out, freq, 10); f > 0.05 {
				t.Fatalf("%+v semitones, %v Hz: %.2f of the energy still at the original pitch", semitones, freq, f)
			}
		}
	}

	// without a shift the pitch is exact
	if got := peakFrequency(shiftSine(0, 440)); math.Abs(got-440) > 2*sr/8192 {
		t.Fatalf("0 semitones: peak at %.1f Hz", got)
	}
}

func TestLevelIsKept(t *testing.T) {
	out := shiftSine(12, 440)
	sum := 0.0
	for _, v := range out[len(out)/2:] {
		sum += v * v
	}
	rms := math.Sqrt(sum / float64(len(out)/2))
	if math.Abs(rms-1/math.Sqrt2) > 0.15 {
		t.Fatalf("expected roughly the input level, got rms %v", rms)
	}
}
