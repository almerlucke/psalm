package grains

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// segments is a sample of tones one after the other
func segments(seconds float64, freqs ...float64) *testSample {
	var buf []float32
	for _, f := range freqs {
		s := sine(f, seconds)
		buf = append(buf, s.buf...)
	}
	return &testSample{buf: buf, out: make([]float32, 1)}
}

func TestFilterCutsGrains(t *testing.T) {
	cfg := base()
	cfg.Sample = sine(3000, 2)
	cfg.RootNote = 69
	cfg.Scan = 1
	cfg.GrainSize = 200
	cfg.Density = 20
	open := rms(renderLong(cfg, psalm.Message{"pitch": 69.0}, 60))
	low := rms(renderLong(cfg, psalm.Message{"pitch": 69.0, "filter": 1.0, "cutoff": 300.0}, 60))
	high := rms(renderLong(cfg, psalm.Message{"pitch": 69.0, "filter": 2.0, "cutoff": 300.0}, 60))
	if low > open*0.1 {
		t.Fatalf("a lowpass at 300 Hz let a 3 kHz tone through: %v of %v", low, open)
	}
	if high < open*0.6 || high > open*1.4 {
		t.Fatalf("a highpass at 300 Hz changed a 3 kHz tone: %v of %v", high, open)
	}
}

func TestFilterStream(t *testing.T) {
	// a list of cutoffs: the grains alternate between a closed and an open filter
	cfg := base()
	cfg.Sample = sine(3000, 2)
	cfg.Scan = 1
	cfg.Filter = FilterLowpass
	cfg.Streams.Cutoff = func() Stream { return Seq(100, 100, 100, 100) }
	closed := rms(renderLong(cfg, psalm.Message{"pitch": 69.0}, 60))
	cfg.Streams.Cutoff = func() Stream { return Seq(10000) }
	open := rms(renderLong(cfg, psalm.Message{"pitch": 69.0}, 60))
	if closed > open*0.1 {
		t.Fatalf("cutoffs of a stream do not close the grains: %v of %v", closed, open)
	}
}

func TestModulationMovesPitch(t *testing.T) {
	cfg := base()
	cfg.Scan = 1
	cfg.GrainSize = 300
	cfg.Density = 12
	binHz := sr / 16384
	// the velocity (1) of the note goes to the pitch, an octave
	left := renderLong(cfg, psalm.Message{"pitch": 69.0, "mod1Source": float64(ModVelocity), "mod1Dest": float64(ModPitch), "mod1Amount": 1.0}, 120)
	if f := peakFrequency(left); math.Abs(f-880) > 2*binHz {
		t.Fatalf("expected 880 Hz, got %v", f)
	}
	// nothing is on: nothing changes
	left = renderLong(cfg, psalm.Message{"pitch": 69.0, "mod1Dest": float64(ModPitch), "mod1Amount": 1.0}, 120)
	if f := peakFrequency(left); math.Abs(f-440) > 2*binHz {
		t.Fatalf("a slot without a source moved the pitch: %v", f)
	}
}

func TestModulationLFOMovesLevel(t *testing.T) {
	cfg := base()
	cfg.Density = 80
	cfg.GrainSize = 20
	cfg.Scan = 1
	left := renderLong(cfg, psalm.Message{"mod1Source": float64(ModLFO), "mod1Dest": float64(ModAmp), "mod1Amount": 1.0, "mod1Rate": 2.0}, 400)
	// two cycles a second: the level of an eighth of a second differs a lot between the peaks and the troughs of the lfo
	q := 5512
	lo, hi := math.Inf(1), 0.0
	for i := 0; i+q <= len(left); i += q {
		r := rms(left[i : i+q])
		lo, hi = min(lo, r), max(hi, r)
	}
	if hi < lo*2 {
		t.Fatalf("an lfo on the level did not move it: %v to %v", lo, hi)
	}
}

func TestStretchKeepsPitchAndMovesSlowly(t *testing.T) {
	binHz := sr / 16384
	// 4 s of 220 Hz and then 4 s of 880 Hz; at the root note the grains sound as they were
	cfg := base()
	cfg.Sample = segments(4, 220, 880)
	cfg.RootNote = 69
	cfg.Position = 0
	cfg.Spray = 0
	cfg.GrainSize = 300
	cfg.Density = 12
	// as fast as the recording: after 5.5 s (the buffers kept are the last ones) we are in the 880 part
	buffers := int(5.5 * sr / float64(psalm.BufferSize()))
	pick := func(stretch float64) float64 {
		left := renderLong(cfg, psalm.Message{"stretch": stretch, "pitch": 69.0}, buffers)
		return peakFrequency(left[len(left)-int(sr):])
	}
	if f := pick(1); math.Abs(f-880) > 3*binHz {
		t.Fatalf("stretch 1: expected 880 Hz after 5 s, got %v", f)
	}
	// four times as slow: 5.5 s is 1.4 s into the recording, still the 220 part, at its own pitch
	if f := pick(4); math.Abs(f-440) > 3*binHz && math.Abs(f-220) > 3*binHz {
		t.Fatalf("stretch 4: expected the 220 part (at the pitch of the note), got %v", f)
	}
}

func TestOverlapSetsDensity(t *testing.T) {
	cfg := base()
	cfg.Overlap = 4
	cfg.GrainSize = 100
	cfg.Jitter = 0
	v := New(cfg)
	v.NoteOn(0.5, psalm.Message{}, psalm.CtrlTick{}, v.Config())
	most := 0
	for range 80 {
		v.Synthesize()
		most = max(most, len(v.grains))
	}
	if most < 3 || most > 6 {
		t.Fatalf("an overlap of 4 had %d grains at once", most)
	}
}

func TestFeedbackIsHeardAndBounded(t *testing.T) {
	cfg := base()
	cfg.Scan = 0
	cfg.Density = 20
	cfg.GrainSize = 150
	plain := renderLong(cfg, psalm.Message{"pitch": 69.0}, 300)
	fb := renderLong(cfg, psalm.Message{"pitch": 69.0, "feedback": 0.9, "feedbackTime": 0.2, "feedbackPitch": 12.0}, 300)
	peak := 0.0
	for _, x := range fb {
		if math.IsNaN(x) {
			t.Fatal("NaN")
		}
		peak = max(peak, math.Abs(x))
	}
	if peak > 4 {
		t.Fatalf("the feedback runs away: peak %v", peak)
	}
	// the grains of the feedback sound an octave above what they heard: there is energy at 880 Hz that the plain voice does not have
	if energyAt(fb, 880) < 3*energyAt(plain, 880) {
		t.Fatalf("no octave in the feedback: %v against %v", energyAt(fb, 880), energyAt(plain, 880))
	}
}

// energyAt is the power of the signal near a frequency (a Goertzel filter)
func energyAt(x []float64, f float64) float64 {
	w := 2 * math.Pi * f / sr
	c := 2 * math.Cos(w)
	var s1, s2 float64
	for _, v := range x {
		s := v + c*s1 - s2
		s2, s1 = s1, s
	}
	return (s1*s1 + s2*s2 - c*s1*s2) / float64(len(x))
}

func TestSlicesPlayOnePart(t *testing.T) {
	binHz := sr / 16384
	cfg := base()
	cfg.Sample = segments(1, 200, 400, 800, 1600)
	cfg.RootNote = 69
	cfg.Scan = 1
	cfg.Spray = 0.4 // far wider than a slice: the grains still stay in it
	cfg.GrainSize = 200
	cfg.Density = 20
	for slice, want := range []float64{200, 400, 800, 1600} {
		left := renderLong(cfg, psalm.Message{"pitch": 69.0, "slices": 2.0, "slice": float64(slice), "position": 0.5}, 150)
		if f := peakFrequency(left); math.Abs(f-want) > 3*binHz {
			t.Fatalf("slice %d: expected %v Hz, got %v", slice, want, f)
		}
	}
	// the stream picks a slice for each grain, round and round
	cfg.Slices = 2
	cfg.Streams.Slice = func() Stream { return Seq(1) }
	left := renderLong(cfg, psalm.Message{"pitch": 69.0}, 150)
	if f := peakFrequency(left); math.Abs(f-400) > 3*binHz {
		t.Fatalf("a stream of slices: expected 400 Hz, got %v", f)
	}
}

func TestOnsets(t *testing.T) {
	// three bursts with silence between them
	buf := make([]float32, int(4*sr))
	for _, at := range []float64{0.5, 1.5, 2.5} {
		for i := int(at * sr); i < int((at+0.3)*sr); i++ {
			buf[i] = float32(0.5 * math.Sin(2*math.Pi*440*float64(i)/sr))
		}
	}
	s := &testSample{buf: buf, out: make([]float32, 1)}
	b := NewSampleSource(s).(sliceable).sliceBounds(1)
	if len(b) != 5 {
		t.Fatalf("expected the start, 3 onsets and the end, got %v", b)
	}
	for i, at := range []float64{0.5, 1.5, 2.5} {
		if math.Abs(b[i+1]*4-at) > 0.03 {
			t.Fatalf("onset %d at %v s, expected %v", i, b[i+1]*4, at)
		}
	}
}
