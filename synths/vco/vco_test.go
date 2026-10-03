package vco

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
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
