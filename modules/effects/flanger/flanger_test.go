package flanger

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

const sr = 44100.0

func run(f *Flanger, buffers int, in func(n int) float64) ([]float64, []float64) {
	var l, r []float64
	bs := f.Config().BufferSize
	for b := range buffers {
		for i := range bs {
			v := float32(in(b*bs + i))
			f.Input(0).Channels[0][i] = v
			f.Input(0).Channels[1][i] = v
		}
		f.Synthesize()
		for i := range bs {
			l = append(l, float64(f.Output(0).Channels[0][i]))
			r = append(r, float64(f.Output(0).Channels[1][i]))
		}
	}
	return l, r
}

func rms(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

// A fixed delay (no sweep) notches the frequency whose half period it is: 1 ms notches 500 Hz
func TestNotch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Rate, cfg.Depth, cfg.Feedback, cfg.Manual, cfg.Mix = 0, 0, 0, 1, 0.5
	level := func(freq float64) float64 {
		l, _ := run(New(cfg), 20, func(n int) float64 { return math.Sin(2 * math.Pi * freq * float64(n) / sr) })
		return rms(l[5*1024:])
	}
	if notch, pass := level(500), level(1000); notch > 0.01 || pass < 0.6 {
		t.Fatalf("expected a notch at 500 Hz and a peak at 1 kHz: %v, %v", notch, pass)
	}
}

// Fully wet keeps the level of the input (the old flanger added the dry signal on top)
func TestLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mix, cfg.Feedback = 1, 0
	l, _ := run(New(cfg), 40, func(n int) float64 { return 0.5 * math.Sin(2*math.Pi*300*float64(n)/sr) })
	if g := 20 * math.Log10(rms(l[10*1024:])/(0.5/math.Sqrt2)); math.Abs(g) > 0.5 {
		t.Fatalf("expected unity gain, got %.1f dB", g)
	}
}

// The sweep follows the tempo when it is set in beats, and stereo offsets the right side
func TestBeatsAndStereo(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Beats = 2
	f := New(cfg)
	cfg2 := psalm.CurrentConfig()
	run(f, 1, func(int) float64 { return 0 })
	// at 120 BPM two beats last a second: after a second the phase is back where it started
	run(f, 42, func(int) float64 { return 0 }) // 43 buffers: about a second
	if bpm := cfg2.BPM; bpm == 120 && math.Abs(f.phase-math.Round(f.phase)) > 0.03 {
		t.Fatalf("expected a sweep per 2 beats, phase after a second %v", f.phase)
	}
	cfg.Stereo = 1
	cfg.Mix = 1
	l, r := run(New(cfg), 40, func(n int) float64 { return math.Sin(2 * math.Pi * 2000 * float64(n) / sr) })
	same := true
	for i := 10 * 1024; i < len(l); i++ {
		if math.Abs(l[i]-r[i]) > 1e-3 {
			same = false
			break
		}
	}
	if same {
		t.Fatal("with stereo the sides should sweep differently")
	}
}

func TestFeedbackIsBounded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Feedback = -5
	l, _ := run(New(cfg), 200, func(n int) float64 { return math.Sin(float64(n) * 0.3) })
	for _, v := range l {
		if math.IsNaN(v) || math.Abs(v) > 40 {
			t.Fatalf("unbounded %v", v)
		}
	}
}
