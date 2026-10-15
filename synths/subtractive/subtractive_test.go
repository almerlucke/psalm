package subtractive

import (
	"math"
	"testing"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

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

	p.CallVoices(func(v *Voice) {
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
			if v := l.tick(cfg, 44100.0, cfg.Rate); v < -1.0 || v > 1.0 || math.IsNaN(v) {
				t.Fatalf("waveform %d out of range: %v", wf, v)
			}
		}
	}
}

func TestLFODelayFadesIn(t *testing.T) {
	cfg := &LFOConfig{Waveform: LFOSquare, Rate: 1.0, Delay: 100.0}
	l := &lfo{}
	l.trigger(cfg, 1000.0)

	if v := l.tick(cfg, 1000.0, cfg.Rate); math.Abs(v) > 0.02 {
		t.Fatalf("expected faded out lfo at start, got %v", v)
	}
	for range 200 {
		l.tick(cfg, 1000.0, cfg.Rate)
	}
	if v := l.tick(cfg, 1000.0, cfg.Rate); math.Abs(v) != 1.0 {
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

// renderRMS plays a sustained note and returns the RMS of the output after the attack
func renderRMS(cfg Config, pitch float64) float64 {
	v := New(cfg)
	v.NoteOn(0.8, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())

	sum, n := 0.0, 0
	for b := range 20 {
		v.Synthesize()
		if b < 5 {
			continue
		}
		for _, s := range v.Output(0).Channels[0] {
			sum += float64(s) * float64(s)
			n++
		}
	}
	return math.Sqrt(sum / float64(n))
}

func steadyConfig() Config {
	cfg := DefaultConfig()
	cfg.FilterEnvAmount = 0.0
	cfg.KeyTrack = 0.0
	cfg.Resonance = 0.0
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 1.0, 10.0, 1.0, 50.0)
	return cfg
}

func TestLadderFilterType(t *testing.T) {
	dark := steadyConfig()
	dark.FilterType = FilterLadder
	dark.Cutoff = 150.0

	bright := dark
	bright.Cutoff = 8000.0

	// a 440 Hz saw through a 150 Hz 24 dB ladder is much quieter than through an open one
	if d, b := renderRMS(dark, 69), renderRMS(bright, 69); d > b*0.25 {
		t.Fatalf("expected the closed ladder to be much darker: %v vs %v", d, b)
	}
}

func TestFormantVowelChangesSound(t *testing.T) {
	a := steadyConfig()
	a.FilterType = FilterFormant
	a.Vowel = 0

	i := a
	i.Vowel = 2

	ra, ri := renderRMS(a, 45), renderRMS(i, 45)
	if ra == 0 || ri == 0 || math.Abs(ra-ri)/math.Max(ra, ri) < 0.05 {
		t.Fatalf("expected vowels a and i to sound different: %v vs %v", ra, ri)
	}
}

func TestSVFNotchMode(t *testing.T) {
	// a sine right on the notch frequency is removed
	cfg := steadyConfig()
	cfg.Osc1Waveform = blosc.Sine
	cfg.Osc2Level = 0.0
	cfg.FilterMode = Notch
	cfg.Cutoff = 440.0
	cfg.Resonance = 0.5

	open := cfg
	open.FilterMode = LowPass
	open.Cutoff = 15000.0

	if n, o := renderRMS(cfg, 69), renderRMS(open, 69); n > o*0.1 {
		t.Fatalf("expected the notch to remove the sine: %v vs %v", n, o)
	}
}

func TestFilterTypesWithModulationAreStable(t *testing.T) {
	for _, ft := range []FilterType{FilterSVF, FilterLadder, FilterFormant} {
		cfg := DefaultConfig()
		cfg.FilterType = ft
		cfg.Resonance = 0.95
		cfg.FilterDrive = 3.0
		cfg.FilterEnvAmount = 4.0
		cfg.LFO1 = LFOConfig{Waveform: LFOSampleHold, Rate: 10, Cutoff: 3.0}

		v := New(cfg)
		v.Note(0.4, 0.8, psalm.Message{"pitch": 40.0}, psalm.CtrlTick{}, v.Config())
		for blocks := 0; v.IsActive(); blocks++ {
			if blocks > 1000 {
				t.Fatalf("filter type %d: voice never finished", ft)
			}
			v.Synthesize()
			for _, s := range v.Output(0).Channels[0] {
				if math.IsNaN(float64(s)) || math.Abs(float64(s)) > 4 {
					t.Fatalf("filter type %d: invalid sample %v", ft, s)
				}
			}
		}
	}
}

// jumpRatio plays a sustained note while setting a value input to a new value every buffer (like an
// LFO does), and compares the largest sudden jump (third difference) at buffer boundaries with the
// largest one elsewhere. Parameters that step instead of glide would click at the boundaries
func jumpRatio(cfg Config, input int, value func(t float64) float64) float64 {
	v := New(cfg)
	v.NoteOn(0.5, psalm.Message{"pitch": 45.0}, psalm.CtrlTick{}, v.Config())
	bufSize := v.Config().BufferSize
	var out []float64
	for b := range 200 {
		if input > 0 {
			v.Receive(value(float64(b*bufSize)/v.Config().SampleRate), input)
		}
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}
	boundary, inner := 0.0, 0.0
	for i := 100 * bufSize; i+3 < len(out); i++ {
		d := math.Abs(out[i+3] - 3*out[i+2] + 3*out[i+1] - out[i])
		if (i+2)%bufSize == 0 || (i+1)%bufSize == 0 || i%bufSize == 0 {
			boundary = math.Max(boundary, d)
		} else {
			inner = math.Max(inner, d)
		}
	}
	return boundary / inner
}

func TestModulatedInputsDontClick(t *testing.T) {
	// a sine source, so the only sudden changes come from the parameters
	base := DefaultConfig()
	base.Osc1Waveform = blosc.Sine
	base.Osc2Level = 0
	base.Cutoff = 400
	base.FilterEnvAmount = 0
	base.AmpEnv = *adsr.NewSetting(1.0, 5.0, 1.0, 10.0, 1.0, 100.0)
	sweep := func(lo, hi float64) func(float64) float64 {
		return func(t float64) float64 { return lo + (hi-lo)*(0.5+0.5*math.Sin(2*math.Pi*0.3*t)) }
	}

	pw := base
	pw.Osc1Waveform = blosc.Rectangle
	pw.Cutoff = 4000

	formantCfg := base
	formantCfg.FilterType = FilterFormant
	formantCfg.Resonance = 0.5

	cases := []struct {
		name  string
		cfg   Config
		input int
		value func(float64) float64
	}{
		{"cutoff", base, 1, sweep(200, 3000)},
		{"resonance", base, 2, sweep(0, 0.9)},
		{"pulse width", pw, 3, sweep(0.1, 0.9)},
		{"vowel", formantCfg, 4, sweep(0, 4)},
	}

	for _, c := range cases {
		if r := jumpRatio(c.cfg, c.input, c.value); r > 1.1 {
			t.Fatalf("%s changes click at buffer boundaries (jump ratio %.2f)", c.name, r)
		}
	}
}

// energy plays a note and returns the sum of the squares of what it sounds
func energy(v *Voice, blocks int) float64 {
	v.NoteOn(0.8, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	var e float64
	for range blocks {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			e += float64(s) * float64(s)
		}
	}
	return e
}

// a value cabled in on an Ext input moves any setting a slot joins it to
func TestMatrixExtMovesLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -1}
	loud, quiet := New(cfg), New(cfg)
	quiet.Receive(1.0, InExt1)
	el, eq := energy(loud, 40), energy(quiet, 40)
	if el < 1 || eq > el*0.01 {
		t.Errorf("energy %g with the value at 0, %g at 1: want it silent at 1", el, eq)
	}
	if quiet.cfg.Ext[0] != 1 {
		t.Errorf("the value isn't kept in the config: %v", quiet.cfg.Ext)
	}
}

// an LFO on the pitch through the matrix sounds different from none
func TestMatrixLFOToPitch(t *testing.T) {
	plain := energy(New(DefaultConfig()), 40)
	cfg := DefaultConfig()
	cfg.LFO1.Rate = 5
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "pitch", Amount: 0.5}
	moved := New(cfg)
	if em := energy(moved, 40); em == plain || math.IsNaN(em) {
		t.Errorf("energy %g, the same as without (%g)", em, plain)
	}
}

// every source into every destination plays no NaN, and the slots arrive by message
func TestMatrixEverything(t *testing.T) {
	for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
		for _, d := range Dests {
			cfg := DefaultConfig()
			cfg.LFO1.Rate, cfg.LFO2.Rate = 3, 4
			cfg.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
			cfg.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
			v := New(cfg)
			v.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, v.Config())
			for range 20 {
				v.Synthesize()
				for _, s := range v.Output(0).Channels[0] {
					if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) || math.Abs(float64(s)) > 20 {
						t.Fatalf("%s to %s: sample %v", modmatrix.SourceNames[src], d.Key, s)
					}
				}
			}
		}
	}
	v := New(DefaultConfig())
	v.ReceiveMessage(psalm.Message{"mod3Src": int(modmatrix.Wheel), "mod3Dest": 2, "mod3Amount": 0.5})
	if s := v.cfg.Mods[2]; s.Src != modmatrix.Wheel || s.Dest != "cutoff" || s.Amount != 0.5 {
		t.Errorf("slot %+v", s)
	}
}

// render plays a note and returns what it sounded
func render(cfg Config, blocks int) []float32 {
	v := New(cfg)
	v.NoteOn(0.8, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	var out []float32
	for range blocks {
		v.Synthesize()
		out = append(out, v.Output(0).Channels[0]...)
	}
	return out
}

func TestUnison(t *testing.T) {
	one := DefaultConfig()
	many := one
	many.Unison, many.UnisonDetune = 5, 30
	a, b := render(one, 40), render(many, 40)
	var ea, eb, diff float64
	for i := range a {
		ea += float64(a[i]) * float64(a[i])
		eb += float64(b[i]) * float64(b[i])
		d := float64(a[i]) - float64(b[i])
		diff += d * d
	}
	if diff < ea*0.1 {
		t.Errorf("unison sounds the same as one pair: difference %g of %g", diff, ea)
	}
	if eb < ea*0.3 || eb > ea*3 || math.IsNaN(eb) {
		t.Errorf("unison energy %g against %g for one pair: gain not kept in check", eb, ea)
	}
	v := New(one)
	v.ReceiveMessage(psalm.Message{"unison": 4, "unisonDetune": 20.0, "drift": 10.0})
	if v.cfg.Unison != 4 || v.cfg.UnisonDetune != 20 || v.cfg.Drift != 10 {
		t.Errorf("settings %d %g %g", v.cfg.Unison, v.cfg.UnisonDetune, v.cfg.Drift)
	}
}

func TestDriftWanders(t *testing.T) {
	still := DefaultConfig()
	drifting := still
	drifting.Drift = 50
	a, b := render(still, 400), render(drifting, 400)
	var diff float64
	for i := range a {
		diff += math.Abs(float64(a[i]) - float64(b[i]))
	}
	if diff == 0 {
		t.Error("drift doesn't change the sound")
	}
}
