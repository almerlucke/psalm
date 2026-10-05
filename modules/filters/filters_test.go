package filters

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm/blocks/filters/biquad"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
)

// level runs sines through the filter for a number of buffers and returns the output RMS of the
// last buffer per channel
func level(f *Filter, numBuffers int, freqL, freqR float64) (float64, float64) {
	sr := f.Config().SampleRate
	bufSize := f.Config().BufferSize
	var sumL, sumR float64

	for b := range numBuffers {
		for i := range bufSize {
			n := float64(b*bufSize + i)
			f.Input(0).Channels[0][i] = float32(0.1 * math.Sin(2*math.Pi*freqL*n/sr))
			f.Input(0).Channels[1][i] = float32(0.1 * math.Sin(2*math.Pi*freqR*n/sr))
		}
		f.Synthesize()
	}

	for i := range bufSize {
		l := float64(f.Output(0).Channels[0][i])
		r := float64(f.Output(0).Channels[1][i])
		sumL += l * l
		sumR += r * r
	}

	return math.Sqrt(sumL / float64(bufSize)), math.Sqrt(sumR / float64(bufSize))
}

const inRMS = 0.1 / math.Sqrt2

func TestFiltersPassAndStop(t *testing.T) {
	cases := []struct {
		name   string
		filter *Filter
	}{
		{"svf", NewSVF(svf.LowPass, 500, 0.0)},
		{"ladder", NewLadder(ladder.LowPass24, 500, 0.0, 1.0)},
		{"biquad", NewBiquad(biquad.LowPass, 500, 0.707, 0)},
	}

	for _, c := range cases {
		pass, stop := level(c.filter, 20, 60, 8000)
		if math.Abs(pass/inRMS-1.0) > 0.1 {
			t.Fatalf("%s: passband level %v, want %v", c.name, pass, inRMS)
		}
		if stop > inRMS*0.05 {
			t.Fatalf("%s: stopband level %v too high", c.name, stop)
		}
	}
}

func TestCutoffInputGlides(t *testing.T) {
	f := NewSVF(svf.LowPass, 200, 0.0)
	_, before := level(f, 20, 2000, 2000)

	// open the filter through value input 1
	f.Receive(10000.0, 1)

	_, after := level(f, 10, 2000, 2000)
	if after < before*5 {
		t.Fatalf("expected opening the cutoff to raise the 2 kHz level: before %v after %v", before, after)
	}
	if math.Abs(after/inRMS-1.0) > 0.1 {
		t.Fatalf("expected near unity after opening, got %v", after/inRMS)
	}
}

func TestStereoChannelsAreIndependent(t *testing.T) {
	f := NewLadder(ladder.LowPass24, 500, 0.3, 1.0)
	l, r := level(f, 20, 60, 8000)
	if l < inRMS*0.5 || r > inRMS*0.05 {
		t.Fatalf("expected left to pass and right to be filtered, got %v %v", l, r)
	}
}

func TestFormantVowelInput(t *testing.T) {
	// 1090 Hz is a formant of "a" but not of "i"
	f := NewFormant(0, 10, 1)
	a, _ := level(f, 20, 1090, 1090)

	f.Receive(2.0, 3) // vowel i
	i, _ := level(f, 20, 1090, 1090)

	if a < i*3 {
		t.Fatalf("expected vowel a to pass 1090 Hz much more than vowel i: %v vs %v", a, i)
	}
}

func TestMessages(t *testing.T) {
	f := NewBiquad(biquad.Peak, 1000, 1, 0)
	f.ReceiveMessage(map[string]any{"gainDB": 12.0, "cutoff": 1000.0})
	boosted, _ := level(f, 20, 1000, 1000)

	gain := 20 * math.Log10(boosted/inRMS)
	if math.Abs(gain-12) > 0.5 {
		t.Fatalf("expected 12 dB boost, got %v dB", gain)
	}
}
