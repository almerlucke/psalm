package main

import (
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
)

// peakOf listens to the master for a while (sampled on the audio thread) and returns the loudest sample
func peakOf(p *psalm.Psalm, in *instrument, d time.Duration) float32 {
	var peak atomic.Uint32
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		p.Do(func() {
			for _, ch := range in.mix.Master().Output(0).Channels {
				for _, s := range ch {
					if s < 0 {
						s = -s
					}
					if cur := float32FromBits(peak.Load()); s > cur {
						peak.Store(float32ToBits(s))
					}
				}
			}
		})
		time.Sleep(2 * time.Millisecond)
	}
	return float32FromBits(peak.Load())
}

// the test phrases make sound, and Stop makes it end
func TestPhrasesSound(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(paneltest.NewEngine(t))
	in, err := build(p, options{start: "guitar", voices: 6, hall: 0, presetDir: t.TempDir(), theme: panel.Themes["default"]})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })

	if quiet := peakOf(p, in, 300*time.Millisecond); quiet != 0 {
		t.Errorf("sound before anything is played: %v", quiet)
	}
	for _, name := range phraseOrder {
		in.run(name)
		if got := peakOf(p, in, 1500*time.Millisecond); got < 0.01 {
			t.Errorf("%s: peak %v, want sound", name, got)
		}
		in.stop()
		time.Sleep(200 * time.Millisecond)
	}
}

// every preset can be chosen on the page and plays a note
func TestPresetsPlay(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	for _, start := range []string{"violin", "tom", "flute", "harp"} {
		p := psalm.New(paneltest.NewEngine(t))
		in, err := build(p, options{start: start, voices: 4, hall: 0, presetDir: t.TempDir(), theme: panel.Themes["default"]})
		if err != nil {
			t.Fatal(err)
		}
		if err := p.AudioStart(); err != nil {
			t.Fatal(err)
		}
		in.run("Chord")
		if got := peakOf(p, in, 2*time.Second); got < 0.01 {
			t.Errorf("%s: peak %v", start, got)
		}
		in.stop()
		_ = p.AudioStop()
	}
	if _, err := build(psalm.New(nil), options{start: "nope", voices: 1, presetDir: t.TempDir(), theme: panel.Themes["default"]}); err == nil {
		t.Error("an unknown preset should be an error")
	}
}

func float32ToBits(f float32) uint32   { return math.Float32bits(f) }
func float32FromBits(u uint32) float32 { return math.Float32frombits(u) }
