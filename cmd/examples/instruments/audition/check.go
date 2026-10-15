package audition

import (
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
)

// Checks says what Check tries beyond the phrases
type Checks struct {
	Start     []string // presets the page starts on, each plays the first phrase
	Chosen    []string // presets chosen on the page, in this order, each the synth made at it
	Sustained []string // of those chosen, the ones that go on sounding half a second into a held chord
}

// peakOf listens to the master for a while (sampled on the audio thread) and returns the loudest sample
func peakOf(p *psalm.Psalm, in *Instrument, d time.Duration) float32 {
	var peak atomic.Uint32
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		p.Do(func() {
			for _, ch := range in.Mix.Master().Output(0).Channels {
				for _, s := range ch {
					if s < 0 {
						s = -s
					}
					if s > math.Float32frombits(peak.Load()) {
						peak.Store(math.Float32bits(s))
					}
				}
			}
		})
		time.Sleep(2 * time.Millisecond)
	}
	return math.Float32frombits(peak.Load())
}

func start(t *testing.T, spec Spec, preset string, voices int) (*psalm.Psalm, *Instrument) {
	t.Helper()
	p := psalm.New(paneltest.NewEngine(t))
	in, err := Build(p, spec, Options{Start: preset, Voices: voices, Hall: 0, PresetDir: t.TempDir(), Theme: panel.Themes["default"]})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.AudioStop() })
	return p, in
}

// Check tests the page of a synth the way a player meets it: quiet until something is played, every
// test phrase makes sound, presets the page starts on play, presets chosen on the page are the
// presets (and go on sounding when they should), an unknown preset is an error
func Check(t *testing.T, spec Spec, c Checks) {
	t.Helper()
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()

	// the phrases
	p, in := start(t, spec, spec.Default, 6)
	if quiet := peakOf(p, in, 300*time.Millisecond); quiet != 0 {
		t.Errorf("sound before anything is played: %v", quiet)
	}
	for _, name := range spec.Order {
		in.Run(name)
		if got := peakOf(p, in, 2*time.Second); got < 0.001 {
			t.Errorf("%s: peak %v, want sound", name, got)
		}
		in.Stop()
		time.Sleep(200 * time.Millisecond)
	}

	// presets the page starts on
	for _, name := range c.Start {
		p, in := start(t, spec, name, 4)
		in.Run(spec.Order[0])
		if got := peakOf(p, in, 2*time.Second); got < 0.001 {
			t.Errorf("%s: peak %v", name, got)
		}
		in.Stop()
		_ = p.AudioStop()
	}
	if _, err := Build(psalm.New(nil), spec, Options{Start: "nope", Voices: 1, PresetDir: t.TempDir(), Theme: panel.Themes["default"]}); err == nil {
		t.Error("an unknown preset should be an error")
	}

	// presets chosen on the page, which started on the default (a model, a bow or a pipe isn't a
	// setting of the one before: a burst and then nothing would be the one before's settings moved)
	p, in = start(t, spec, spec.Default, 4)
	sustained := map[string]bool{}
	for _, n := range c.Sustained {
		sustained[n] = true
	}
	for _, name := range c.Chosen {
		if err := in.Editor.Presets().Load(name, false); err != nil {
			t.Fatal(err)
		}
		in.Swap.Wait()
		if in.Editor.Base() != name {
			t.Errorf("%s: the editor's base is %q", name, in.Editor.Base())
		}
		in.Run("Chord")
		time.Sleep(500 * time.Millisecond)
		if got := peakOf(p, in, 600*time.Millisecond); sustained[name] && got < 0.01 {
			t.Errorf("%s: peak %v half a second into a held chord, want a sustained sound", name, got)
		}
		in.Stop()
		time.Sleep(300 * time.Millisecond)
	}
}
