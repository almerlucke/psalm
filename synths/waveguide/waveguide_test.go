package waveguide

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

func hz(p float64) float64 { return 440 * math.Exp2((p-69)/12) }

func run(v *Voice, seconds float64) []float64 {
	var x []float64
	for range int(math.Ceil(seconds * sr() / float64(v.Config().BufferSize))) {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func play(cfg Config, pitch, seconds float64) (*Voice, []float64) {
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())
	return v, run(v, seconds)
}

// pitchOf estimates the frequency of x (YIN)
func pitchOf(x []float64) float64 {
	maxLag := int(sr() / 30)
	w := min(len(x)-maxLag-2, 4096)
	d := make([]float64, maxLag+2)
	for tau := 1; tau <= maxLag+1; tau++ {
		for i := range w {
			e := x[i] - x[i+tau]
			d[tau] += e * e
		}
	}
	cm := make([]float64, maxLag+2)
	sum := 0.0
	for tau := 1; tau <= maxLag+1; tau++ {
		sum += d[tau]
		cm[tau] = d[tau] * float64(tau) / sum
	}
	for tau := 2; tau <= maxLag; tau++ {
		if cm[tau] < 0.15 {
			for tau+1 <= maxLag && cm[tau+1] < cm[tau] {
				tau++
			}
			a, b, c := cm[tau-1], cm[tau], cm[tau+1]
			return sr() / (float64(tau) + 0.5*(a-c)/(a-2*b+c))
		}
	}
	return 0
}

// amplitude of a frequency in x (Goertzel, Hann window)
func amplitude(x []float64, freq float64) float64 {
	w := 2 * math.Pi * freq / sr()
	var s1, s2, norm float64
	for i, v := range x {
		h := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(x)-1))
		s := v*h + 2*math.Cos(w)*s1 - s2
		s2, s1 = s1, s
		norm += h
	}
	return 2 * math.Hypot(s1-s2*math.Cos(w), s2*math.Sin(w)) / norm
}

func cents(got, want float64) float64 { return 1200 * math.Log2(got/want) }

func dry(cfg Config) Config {
	cfg.BodyMix = 0
	return cfg
}

func TestPluckInTune(t *testing.T) {
	for _, p := range []float64{28, 40, 52, 64, 76, 88, 96} {
		_, x := play(dry(Steel()), p, 0.5)
		if c := cents(pitchOf(x[int(0.1*sr()):]), hz(p)); math.Abs(c) > 3 {
			t.Errorf("pitch %g: %.1f cents out", p, c)
		}
	}
}

func TestPluckDecay(t *testing.T) {
	// the fundamental falls 60 dB in Decay at 220 Hz: 20 dB in a third of it
	cfg := dry(Guitar())
	cfg.Decay = 1.5
	_, x := play(cfg, 57, 1.2)
	win := int(0.1 * sr())
	a := amplitude(x[int(0.1*sr()):int(0.1*sr())+win], 220)
	b := amplitude(x[int(0.6*sr()):int(0.6*sr())+win], 220)
	if d := 20 * math.Log10(a/b); math.Abs(d-20) > 3 {
		t.Errorf("in 0.5 s the fundamental falls %.1f dB, want 20", d)
	}
	// letting go damps it in Release, a harp (Release 0) rings on
	cfg.Release = 0.1
	v, _ := play(cfg, 57, 0.2)
	v.NoteOff()
	run(v, 0.3)
	if v.IsActive() {
		t.Errorf("a string let go still rings")
	}
	v, _ = play(dry(Harp()), 57, 0.2)
	v.NoteOff()
	if x := run(v, 0.3); amplitude(x[len(x)-4096:], 220) < 0.01 {
		t.Errorf("a harp string stops when let go")
	}
}

// band sums the amplitudes of partials of a pitch
func band(x []float64, f float64, partials ...int) float64 {
	sum := 0.0
	for _, k := range partials {
		sum += amplitude(x, float64(k)*f)
	}
	return sum
}

func TestPickPosition(t *testing.T) {
	// plucked in the middle, the even partials have a node there: they are missing
	ratio := func(pick float64) float64 {
		cfg := dry(Steel())
		cfg.PickPosition = pick
		_, x := play(cfg, 45, 0.5)
		y := x[int(0.05*sr()) : int(0.05*sr())+8192]
		return band(y, hz(45), 2, 4, 6, 8) / band(y, hz(45), 1, 3, 5, 7)
	}
	if mid, near := ratio(0.5), ratio(0.12); mid > near*0.1 {
		t.Errorf("the even partials plucked in the middle %.3f, near the bridge %.3f", mid, near)
	}
}

func TestBrightness(t *testing.T) {
	high := func(b float64) float64 {
		cfg := dry(Guitar())
		cfg.Brightness = b
		r := 0.0
		for range 4 { // plucks differ: an average
			_, x := play(cfg, 45, 0.3)
			y := x[int(0.02*sr()) : int(0.02*sr())+4096]
			r += band(y, hz(45), 10, 11, 12, 13, 14, 15, 16) / band(y, hz(45), 1, 2, 3, 4)
		}
		return r
	}
	if soft, hard := high(0.1), high(0.9); hard < 3*soft {
		t.Errorf("the high partials: soft pluck %.4f, hard %.4f", soft, hard)
	}
}

func TestBowSustainsInTune(t *testing.T) {
	for _, name := range []string{"violin", "cello", "slowStrings", "flautando", "ponticello", "eBow", "ghostBow", "scrape"} {
		for _, p := range []float64{43, 55, 67, 74} {
			cfg := dry(Presets[name]())
			cfg.VibratoDepth = 0
			v, x := play(cfg, p, 2)
			if c := cents(pitchOf(x[int(1.2*sr()):]), hz(p)); math.Abs(c) > 15 {
				t.Errorf("%s %g: %.1f cents out", name, p, c)
			}
			// it holds: as loud at the end as at 1 s
			a, b := amplitude(x[int(0.9*sr()):int(1.1*sr())], hz(p)), amplitude(x[len(x)-int(0.2*sr()):], hz(p))
			if b < a*0.5 {
				t.Errorf("%s %g: the bowed string dies: %.3f then %.3f", name, p, a, b)
			}
			v.NoteOff()
			run(v, cfg.BowRelease+0.5)
			if v.IsActive() {
				t.Errorf("%s %g: plays on after the bow stopped", name, p)
			}
		}
	}
}

func TestVibrato(t *testing.T) {
	cfg := dry(Violin())
	cfg.VibratoDelay, cfg.VibratoDepth, cfg.VibratoRate = 0, 0.3, 5
	_, x := play(cfg, 64, 1.5)
	lo, hi := math.Inf(1), 0.0
	for s := 0.5; s < 1.4; s += 0.025 {
		f := pitchOf(x[int(s*sr()) : int(s*sr())+int(0.03*sr())+int(sr()/30)+4])
		lo, hi = math.Min(lo, f), math.Max(hi, f)
	}
	if spread := cents(hi, lo); spread < 30 || spread > 90 {
		t.Errorf("a vibrato of 0.3 semitones swings %.0f cents", spread)
	}
}

func TestPressureColors(t *testing.T) {
	bright := func(pressure float64) float64 {
		cfg := dry(Violin())
		cfg.VibratoDepth, cfg.Pressure = 0, pressure
		_, x := play(cfg, 55, 1)
		y := x[int(0.6*sr()):]
		return amplitude(y, 6*hz(55)) / amplitude(y, hz(55))
	}
	if light, heavy := bright(0.25), bright(0.8); math.Abs(20*math.Log10(heavy/light)) < 1 {
		t.Errorf("the pressure doesn't change the sound: 6th partial %.3f light, %.3f heavy", light, heavy)
	}
}

func TestNotes(t *testing.T) {
	for _, cfg := range []Config{Guitar(), Violin()} {
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{SampleOffset: 40}, v.Config())
		v.Synthesize()
		out := v.Output(0).Channels[0]
		if out[39] != 0 {
			t.Errorf("model %d sounds before its sample", cfg.Model)
		}
		v.Clear()
		v.Note(0.2, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
		run(v, 2)
		if v.IsActive() {
			t.Errorf("model %d: a note of 0.2 s still plays after 2 s", cfg.Model)
		}
	}
	// every preset plays without blowing up
	for name, f := range Presets {
		_, x := play(f(), 57, 1)
		p := 0.0
		for _, s := range x {
			p = math.Max(p, math.Abs(s))
		}
		if p < 0.01 || p > 4 || math.IsNaN(p) {
			t.Errorf("%s peaks at %g", name, p)
		}
	}
}

func TestBowReleaseHasNoRing(t *testing.T) {
	// let go, a bowed string fades: it doesn't swell again, nor ring on after the bow as if plucked
	for _, name := range []string{"violin", "cello", "slowStrings", "ponticello", "flautando", "eBow", "ghostBow", "scrape"} {
		for _, p := range []float64{43, 55, 67, 79} {
			v := New(Presets[name]())
			v.NoteOn(1, psalm.Message{"pitch": p}, psalm.CtrlTick{}, v.Config())
			run(v, 1)
			v.NoteOff()
			x := run(v, 1.5)
			top := math.Inf(-1)
			win := int(0.02 * sr())
			lowest := math.Inf(1)
			for i := 0; i+win <= len(x); i += win {
				pk := 0.0
				for _, s := range x[i : i+win] {
					pk = math.Max(pk, math.Abs(s))
				}
				level := 20 * math.Log10(pk+1e-12)
				if math.IsInf(top, -1) {
					top = level
				}
				if level < top-60 {
					break // inaudible below the note
				}
				if level > lowest+1 {
					t.Errorf("%s %g: %.0f ms after the release it swells from %.1f to %.1f dB", name, p, float64(i)/sr()*1000, lowest, level)
					break
				}
				lowest = math.Min(lowest, level)
			}
		}
	}
}
