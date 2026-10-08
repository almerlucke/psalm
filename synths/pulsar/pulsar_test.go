package pulsar

import (
	"math"
	"testing"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

const sr = 44100.0

func steady() Config {
	cfg := DefaultConfig()
	cfg.StereoSpread = 0
	cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 1.0, 10.0, 1.0, 50.0)
	return cfg
}

// render holds a note and returns the left channel after the attack
func render(cfg Config, msg psalm.Message, buffers int) []float64 {
	v := New(cfg)
	v.NoteOn(0.5, msg, psalm.CtrlTick{}, v.Config())
	var out []float64
	for b := range buffers {
		v.Synthesize()
		if b >= 4 {
			for _, s := range v.Output(0).Channels[0] {
				out = append(out, float64(s))
			}
		}
	}
	return out
}

// spectrum returns the power spectrum averaged over overlapping windows
func spectrum(x []float64) []float64 {
	const n = 16384
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	power := make([]float64, n/2)
	for start := 0; start+n <= len(x); start += n / 2 {
		for i := range n {
			re[i] = x[start+i] * w[i]
			im[i] = 0
		}
		f.Forward(re, im)
		for k := range power {
			power[k] += re[k]*re[k] + im[k]*im[k]
		}
	}
	return power
}

func binFreq(k int) float64 { return float64(k) * sr / 16384 }

func peakBin(p []float64) int {
	best := 1
	for k := 1; k < len(p); k++ {
		if p[k] > p[best] {
			best = k
		}
	}
	return best
}

func centroid(p []float64) float64 {
	num, den := 0.0, 0.0
	for k := 1; k < len(p); k++ {
		num += binFreq(k) * p[k]
		den += p[k]
	}
	return num / den
}

func rms(x []float64) float64 {
	sum := 0.0
	for _, v := range x {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(x)))
}

func TestHarmonicsAndFormant(t *testing.T) {
	for _, f0 := range []float64{110, 220} {
		cfg := steady()
		cfg.Formant = 880
		p := spectrum(render(cfg, psalm.Message{"frequency": f0}, 60))
		peak := binFreq(peakBin(p))

		// the strongest partial is a harmonic of the pitch, at (or next to) the formant
		harmonic := math.Round(peak / f0)
		if math.Abs(peak-harmonic*f0) > 3 {
			t.Fatalf("f0 %v: strongest partial %v Hz is not a harmonic", f0, peak)
		}
		if math.Abs(peak-880) > f0 {
			t.Fatalf("f0 %v: strongest partial %v Hz is not near the 880 Hz formant", f0, peak)
		}
	}
}

func TestFormantMovesTheSpectrum(t *testing.T) {
	low, high := steady(), steady()
	low.Formant, high.Formant = 500, 2000
	cl := centroid(spectrum(render(low, psalm.Message{"frequency": 110.0}, 60)))
	ch := centroid(spectrum(render(high, psalm.Message{"frequency": 110.0}, 60)))
	if ch < cl*2 {
		t.Fatalf("expected a higher formant to brighten the sound: %v vs %v Hz", ch, cl)
	}
}

func TestLowNotesBecomeRhythms(t *testing.T) {
	cfg := steady()
	cfg.Duty = 0.05
	out := render(cfg, psalm.Message{"frequency": 4.0}, 4+int(3*sr)/1024)

	// count bursts: runs of sound separated by silence
	bursts, quiet := 0, 0
	sounding := false
	for _, v := range out {
		if math.Abs(v) > 0.01 {
			if !sounding && quiet > 1000 {
				bursts++
			}
			sounding, quiet = true, 0
		} else {
			quiet++
			if quiet > 1000 {
				sounding = false
			}
		}
	}
	if bursts < 9 || bursts > 13 {
		t.Fatalf("expected about 4 pulses per second over 3 seconds, counted %d", bursts)
	}
}

func TestMasking(t *testing.T) {
	full := steady()
	full.Masking = 1
	if r := rms(render(full, psalm.Message{"frequency": 220.0}, 30)); r > 1e-4 {
		t.Fatalf("full masking should be silent, rms %v", r)
	}

	none, half := steady(), steady()
	half.Masking = 0.5
	rn := rms(render(none, psalm.Message{"frequency": 220.0}, 60))
	rh := rms(render(half, psalm.Message{"frequency": 220.0}, 60))
	if ratio := rh * rh / (rn * rn); ratio < 0.35 || ratio > 0.65 {
		t.Fatalf("half masking should keep about half the energy, got %v", ratio)
	}
}

// The level stays steady across duty and window while a pulse holds about a formant cycle (800 Hz
// at 220 Hz from a duty of about 0.25). Shorter pulses hold too little of the formant, their boost
// is capped so they get quieter instead of spiking
func TestLevelIsSteadyAcrossDutyAndWindow(t *testing.T) {
	ref := rms(render(steady(), psalm.Message{"frequency": 220.0}, 40))
	for _, w := range []Window{Hann, Gaussian, Percussive, Rectangle, Tapered} {
		for _, duty := range []float64{0.25, 0.5, 1.0} {
			cfg := steady()
			cfg.Window = w
			cfg.Duty = duty
			if ratio := rms(render(cfg, psalm.Message{"frequency": 220.0}, 40)) / ref; ratio < 0.5 || ratio > 2 {
				t.Errorf("window %d duty %v: level ratio %v", w, duty, ratio)
			}
		}

		cfg := steady()
		cfg.Window = w
		cfg.Duty = 0.03
		out := render(cfg, psalm.Message{"frequency": 220.0}, 40)
		if ratio := rms(out) / ref; ratio > 1.2 {
			t.Errorf("window %d: very short pulses should not get louder, level ratio %v", w, ratio)
		}
	}
}

func TestNoteLifecycle(t *testing.T) {
	v := New(steady())
	v.Note(0.2, 0.5, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	for blocks := 0; v.IsActive(); blocks++ {
		if blocks > 100 {
			t.Fatal("timed note never ended")
		}
		v.Synthesize()
	}
}

func TestExtremesAreBounded(t *testing.T) {
	cfg := steady()
	cfg.Formant = 15000
	cfg.Formant2 = 9000
	cfg.Formant2Level = 1
	cfg.Duty = 0.01
	cfg.Window = Rectangle
	cfg.Jitter = 1
	cfg.StereoSpread = 1
	cfg.FormantEnvAmount = 4
	cfg.FormantLFODepth = 3
	cfg.FormantLFORate = 20
	cfg.VibratoDepth = 12
	cfg.VibratoRate = 30
	out := render(cfg, psalm.Message{"pitch": 100.0}, 30)
	for _, v := range out {
		if math.IsNaN(v) || math.Abs(v) > 4 {
			t.Fatalf("invalid sample %v", v)
		}
	}
}

// lowNotes are the frequencies the texture and rhythm presets are meant for
var lowNotes = map[string]float64{"rhythm": 6, "crackle": 50, "insects": 45, "raindrops": 12, "motor": 25}

func TestPresets(t *testing.T) {
	for name, preset := range Presets {
		t.Run(name, func(t *testing.T) {
			v := New(preset())
			msg := psalm.Message{"pitch": 57.0}
			if f, ok := lowNotes[name]; ok {
				msg = psalm.Message{"frequency": f}
			}
			v.Note(1.0, 0.5, msg, psalm.CtrlTick{}, v.Config())

			sum, n, peak := 0.0, 0, 0.0
			for blocks := 0; v.IsActive(); blocks++ {
				if blocks > 1000 {
					t.Fatal("voice never finished")
				}
				v.Synthesize()
				for _, ch := range v.Output(0).Channels {
					for _, s := range ch {
						if math.IsNaN(float64(s)) {
							t.Fatal("NaN output")
						}
						peak = math.Max(peak, math.Abs(float64(s)))
						sum += float64(s) * float64(s)
						n++
					}
				}
			}
			if peak < 0.05 || peak > 1.5 {
				t.Fatalf("unexpected peak %v", peak)
			}
			t.Logf("peak %.3f rms %.3f", peak, math.Sqrt(sum/float64(n)))
		})
	}
}

// noiseBetweenHarmonics returns the energy between the harmonics of a 220 Hz note relative to the
// energy at the harmonics, in dB. A clean pulse train only has energy at the harmonics
func noiseBetweenHarmonics(cfg Config) float64 {
	cfg.VibratoDepth = 0 // keeps the harmonics in place
	cfg.AmpEnv.AttackDuration = 5
	cfg.AmpEnv.DecayLevel = 1
	cfg.FormantEnvAmount = 0
	p := spectrum(render(cfg, psalm.Message{"frequency": 220.0}, 100))
	harm, between := 0.0, 0.0
	for k := 1; k < len(p); k++ {
		f := binFreq(k)
		if d := math.Abs(f - 220*math.Round(f/220)); d < 8 {
			harm += p[k]
		} else if d > 30 {
			between += p[k]
		}
	}
	return 10 * math.Log10(between/harm)
}

// Tonal presets must not crackle: random pans or periods per pulse turn into noise at pitch rates,
// the formant LFO and per note spread don't
func TestTonalPresetsAreClean(t *testing.T) {
	for _, name := range []string{"vowelLead", "buzzBass", "glass", "formantPad", "laser", "brass", "subBass"} {
		if n := noiseBetweenHarmonics(Presets[name]()); n > -60 {
			t.Errorf("%s: noise between the harmonics %.1f dB", name, n)
		}
	}

	cfg := steady()
	cfg.StereoSpread = 0.6
	if n := noiseBetweenHarmonics(cfg); n < -30 {
		t.Errorf("expected per pulse spread to add noise, got %.1f dB", n)
	}
}

// The tapered window keeps the buzz of the rectangle without the aliasing of its hard edges
func TestTaperedWindowAvoidsAliasing(t *testing.T) {
	cfg := WobbleBassPreset()
	cfg.FormantLFODepth = 0 // the wobble itself puts sidebands between the harmonics
	tapered := noiseBetweenHarmonics(cfg)
	cfg.Window = Rectangle
	rectangle := noiseBetweenHarmonics(cfg)
	if tapered > -60 || tapered > rectangle-20 {
		t.Fatalf("tapered %.1f dB, rectangle %.1f dB", tapered, rectangle)
	}
}

// Formants stay below half the sample rate: the first one stops there, a second one past it is
// left out (they would fold back as noise)
func TestFormantCeiling(t *testing.T) {
	cfg := GlassPreset()
	cfg.Formant = 3200
	cfg.FormantTracking = 1 // the formants follow the pitch all the way up
	v := New(cfg)
	v.Note(0.5, 1, psalm.Message{"pitch": 100.0}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	ceiling := maxFormant * v.Config().SampleRate
	if v.formant1 > ceiling || v.formant1 <= 0 {
		t.Fatalf("the first formant should stay at or below %.0f Hz, it is %.0f", ceiling, v.formant1)
	}
	if v.formant2 != 0 {
		t.Fatalf("a second formant above the ceiling should be left out, it is %.0f", v.formant2)
	}
}
