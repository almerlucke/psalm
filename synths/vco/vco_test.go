package vco

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*VCO)(nil)

func TestVCONoteLifecycle(t *testing.T) {
	v := New(DefaultConfig())

	v.Note(0.1, 0.8, psalm.Message{"pitch": 48.0}, psalm.CtrlTick{SampleOffset: 10}, v.Config())

	peak := float32(0)
	blocks := 0
	for v.IsActive() && blocks < 100 {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) {
				t.Fatalf("invalid sample %v", s)
			}
			peak = max(peak, float32(math.Abs(float64(s))))
		}
		blocks++
	}

	if v.IsActive() {
		t.Fatalf("voice still active after %d blocks", blocks)
	}
	if peak == 0 || peak > 1.5 {
		t.Fatalf("unexpected peak %v", peak)
	}
}

func TestVCONoteOnOff(t *testing.T) {
	v := New(DefaultConfig())
	v.NoteOn(0.8, psalm.Message{"frequency": 220.0}, psalm.CtrlTick{}, v.Config())

	for range 20 {
		v.Synthesize()
	}
	if !v.IsActive() {
		t.Fatal("voice should sustain until note off")
	}

	v.NoteOff()
	for range 50 {
		v.Synthesize()
	}
	if v.IsActive() {
		t.Fatal("voice should be released")
	}
}

func TestPolyForwardsCutoff(t *testing.T) {
	p := NewPoly(DefaultConfig(), 2)
	p.Receive(1234.0, 1)

	p.CallVoices(func(v *VCO) {
		if v.cfg.Cutoff != 1234.0 {
			t.Fatalf("expected cutoff 1234, got %v", v.cfg.Cutoff)
		}
	})
}

func TestPresets(t *testing.T) {
	for name, preset := range Presets {
		t.Run(name, func(t *testing.T) {
			v := New(preset())
			v.Note(0.5, 0.8, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())

			peak := 0.0
			blocks := 0
			for v.IsActive() && blocks < 1000 {
				v.Synthesize()
				for _, s := range v.Output(0).Channels[0] {
					if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) {
						t.Fatalf("invalid sample %v", s)
					}
					peak = max(peak, math.Abs(float64(s)))
				}
				blocks++
			}

			if v.IsActive() {
				t.Fatalf("voice still active after %d blocks", blocks)
			}
			if peak < 0.01 || peak > 2.0 {
				t.Fatalf("unexpected peak %v", peak)
			}
			t.Logf("peak %.3f", peak)
		})
	}
}

func TestLFOWaveformsInRange(t *testing.T) {
	for wf := LFOSine; wf <= LFOSampleHold; wf++ {
		cfg := &LFOConfig{Waveform: wf, Rate: 5.0}
		l := &lfo{}
		l.trigger(cfg, 44100.0)
		for range 44100 {
			if v := l.tick(cfg, 44100.0); v < -1.0 || v > 1.0 || math.IsNaN(v) {
				t.Fatalf("waveform %d out of range: %v", wf, v)
			}
		}
	}
}

func TestLFODelayFadesIn(t *testing.T) {
	cfg := &LFOConfig{Waveform: LFOSquare, Rate: 1.0, Delay: 100.0}
	l := &lfo{}
	l.trigger(cfg, 1000.0)

	if v := l.tick(cfg, 1000.0); math.Abs(v) > 0.02 {
		t.Fatalf("expected faded out lfo at start, got %v", v)
	}
	for range 200 {
		l.tick(cfg, 1000.0)
	}
	if v := l.tick(cfg, 1000.0); math.Abs(v) != 1.0 {
		t.Fatalf("expected fully faded in lfo, got %v", v)
	}
}

func TestGlide(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Glide = 50.0
	v := New(cfg)

	v.NoteOn(0.8, psalm.Message{"frequency": 220.0}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	v.NoteOn(0.8, psalm.Message{"frequency": 440.0}, psalm.CtrlTick{}, v.Config())

	if v.glideFreq >= 440.0 {
		t.Fatal("expected glide to start below target")
	}
	for range 20 {
		v.Synthesize()
	}
	if math.Abs(v.glideFreq-440.0) > 1.0 {
		t.Fatalf("expected glide to reach 440, got %v", v.glideFreq)
	}
}

func TestAllFeatures(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Osc1Waveform = blosc.Rectangle
	cfg.SubLevel = 0.3
	cfg.NoiseLevel = 0.1
	cfg.NoiseType = PinkNoise
	cfg.RingModLevel = 0.3
	cfg.Sync = true
	cfg.FMAmount = 2.0
	cfg.Glide = 30.0
	cfg.FilterEnvToPitch = 12.0
	cfg.LFO1 = LFOConfig{Waveform: LFOTriangle, Rate: 5.0, Delay: 50.0, Pitch: 0.5, PulseWidth: 0.3}
	cfg.LFO2 = LFOConfig{Waveform: LFOSampleHold, Rate: 8.0, KeySync: true, Cutoff: 1.0, Amp: 0.5}

	v := New(cfg)
	v.Note(0.3, 0.8, psalm.Message{"pitch": 48.0}, psalm.CtrlTick{}, v.Config())

	peak := 0.0
	for blocks := 0; v.IsActive(); blocks++ {
		if blocks > 1000 {
			t.Fatal("voice never finished")
		}
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) {
				t.Fatalf("invalid sample %v", s)
			}
			peak = max(peak, math.Abs(float64(s)))
		}
	}

	if peak < 0.01 || peak > 2.0 {
		t.Fatalf("unexpected peak %v", peak)
	}
}
