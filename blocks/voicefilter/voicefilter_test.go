package voicefilter

import (
	"math"
	"testing"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
)

// run plays a sine of a frequency through the filter and returns its level at the end
func run(f *Filter, hz float64) float64 {
	f.Trigger(0, adsr.NoteOff)
	var peak float64
	for i := range 20000 {
		f.Tick(261.6, 0, 0)
		y := f.Process(0, math.Sin(2*math.Pi*hz*float64(i)/44100))
		if i > 15000 {
			peak = max(peak, math.Abs(y))
		}
	}
	return peak
}

func TestFilterTypes(t *testing.T) {
	cfg := Default()
	cfg.Cutoff, cfg.KeyTrack, cfg.EnvAmount, cfg.Resonance = 1000, 0, 0, 0
	f := New(cfg, 44100, 1)
	if lo, hi := run(f, 100), run(f, 8000); lo < 0.9 || hi > 0.05 {
		t.Errorf("low pass: 100 Hz at %g, 8 kHz at %g", lo, hi)
	}
	f.Receive(psalm.Message{"filterType": int(HighPass)})
	if lo, hi := run(f, 100), run(f, 8000); lo > 0.1 || hi < 0.9 {
		t.Errorf("high pass: 100 Hz at %g, 8 kHz at %g", lo, hi)
	}
	f.Receive(psalm.Message{"filterType": int(Ladder24)})
	if hi24 := run(f, 8000); hi24 > 0.01 {
		t.Errorf("ladder: 8 kHz at %g", hi24)
	}
	f.Receive(psalm.Message{"filterType": int(Off)})
	if f.On() || run(f, 8000) < 0.99 {
		t.Error("off filters")
	}
}

// the envelope opens the cutoff
func TestFilterEnvelopeOpens(t *testing.T) {
	cfg := Default()
	cfg.Cutoff, cfg.KeyTrack, cfg.Resonance = 500, 0, 0
	cfg.Env = *adsr.NewSetting(1.0, 5.0, 1.0, 50.0, 1.0, 50.0)
	closed, open := cfg, cfg
	closed.EnvAmount, open.EnvAmount = 0, 4
	if c, o := run(New(closed, 44100, 1), 4000), run(New(open, 44100, 1), 4000); o < c*4 {
		t.Errorf("4 kHz through a closed filter %g, through an opened one %g", c, o)
	}
}

func TestFilterDescribe(t *testing.T) {
	f := New(Default(), 44100, 2)
	ps, env := f.Describe("Filter")
	for _, p := range ps {
		v := p.Min + (p.Max-p.Min)*0.3
		if p.Exp {
			v = p.Min * math.Pow(p.Max/p.Min, 0.3)
		}
		if p.Step == 1 || p.Kind != 0 {
			v = p.Max
		}
		f.Receive(psalm.Message{p.Key: v})
	}
	if env.Prefix != "filter" || f.Config().Type != Ladder12 || f.Config().Cutoff < 20 {
		t.Errorf("after the messages: %+v", f.Config())
	}
}
