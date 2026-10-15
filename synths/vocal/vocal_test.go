package vocal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

func hz(p float64) float64 { return 440 * math.Exp2((p-69)/12) }

// run renders a voice, both channels
func run(v *Voice, seconds float64) ([]float64, []float64) {
	var l, r []float64
	for range int(math.Ceil(seconds * sr() / float64(v.Config().BufferSize))) {
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			l = append(l, float64(ch[0][i]))
			r = append(r, float64(ch[len(ch)-1][i]))
		}
	}
	return l, r
}

func sing(cfg Config, msg psalm.Message, seconds float64) (*Voice, []float64, []float64) {
	v := New(cfg)
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	l, r := run(v, seconds)
	return v, l, r
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

// strongest returns the harmonic of f0 (1..n) with the most energy between lo and hi Hz
func harmonicsIn(x []float64, f0, lo, hi float64) float64 {
	sum := 0.0
	for k := 1; float64(k)*f0 < hi; k++ {
		if f := float64(k) * f0; f >= lo {
			a := amplitude(x, f)
			sum += a * a
		}
	}
	return sum
}

func steady(cfg Config) Config {
	cfg.VibratoDepth, cfg.Steady, cfg.Breath = 0, true, 0
	cfg.Jitter, cfg.Shimmer = 0, 0
	return cfg
}

func TestPitch(t *testing.T) {
	for _, p := range []float64{43, 55, 67, 79} {
		_, x, _ := sing(steady(Solo()), psalm.Message{"pitch": p}, 0.6)
		y := x[int(0.3*sr()):]
		f := hz(p)
		// the fundamental is there, and nothing between it and its octave
		if a, b := amplitude(y, f), amplitude(y, f*1.5); a < 10*b {
			t.Errorf("pitch %g: the fundamental %g, between the harmonics %g", p, a, b)
		}
	}
}

func TestVowels(t *testing.T) {
	// "a" has its first formant high (~650 Hz), "i" low (~290) and its second high (~1900)
	spectrum := func(vowel float64) (low, mid, high float64) {
		cfg := steady(Solo())
		cfg.Range, cfg.Vowel = Tenor, vowel
		_, x, _ := sing(cfg, psalm.Message{"pitch": 48.0}, 0.6)
		y := x[int(0.3*sr()):]
		f0 := hz(48)
		return harmonicsIn(y, f0, 200, 400), harmonicsIn(y, f0, 550, 800), harmonicsIn(y, f0, 1700, 2100)
	}
	aLow, aMid, aHigh := spectrum(0)
	iLow, iMid, iHigh := spectrum(2)
	if !(aMid > aLow && iLow > iMid && iHigh/iMid > aHigh/aMid) {
		t.Errorf("a: %.2g %.2g %.2g, i: %.2g %.2g %.2g (around 300, 650, 1900 Hz)", aLow, aMid, aHigh, iLow, iMid, iHigh)
	}
	// the vowel glides to a new one while the note sings
	cfg := steady(Solo())
	cfg.Range, cfg.Vowel, cfg.VowelGlide = Tenor, 0, 0.05
	v, _, _ := sing(cfg, psalm.Message{"pitch": 48.0}, 0.2)
	v.ReceiveMessage(psalm.Message{"vowel": 2.0})
	x, _ := run(v, 0.6)
	y := x[int(0.3*sr()):]
	f0 := hz(48)
	if low, mid := harmonicsIn(y, f0, 200, 400), harmonicsIn(y, f0, 550, 800); low < mid {
		t.Errorf("after the change to i: %.2g around 300 Hz, %.2g around 650", low, mid)
	}
}

func TestRanges(t *testing.T) {
	if rangeOf(40) != Bass || rangeOf(52) != Tenor || rangeOf(62) != Alto || rangeOf(76) != Soprano {
		t.Errorf("the ranges of the pitches")
	}
	v := New(Solo())
	v.NoteOn(1, psalm.Message{"pitch": 76.0}, psalm.CtrlTick{}, v.Config())
	if v.rangeNow != Soprano {
		t.Errorf("a high note sings as %d", v.rangeNow)
	}
}

func TestChoirIsWide(t *testing.T) {
	cfg := Choir()
	cfg.Spread = 1
	_, l, r := sing(cfg, psalm.Message{"pitch": 57.0}, 1)
	l, r = l[int(0.5*sr()):], r[int(0.5*sr()):]
	var ll, rr, lr float64
	for i := range l {
		ll += l[i] * l[i]
		rr += r[i] * r[i]
		lr += l[i] * r[i]
	}
	if c := lr / math.Sqrt(ll*rr); c > 0.9 { // the singers beat slowly: it varies, 0.3 to 0.8

		t.Errorf("the choir's channels correlate %.2f", c)
	}
	_, l, r = sing(Solo(), psalm.Message{"pitch": 57.0}, 0.5)
	for i := range l {
		if math.Abs(l[i]-r[i]) > 1e-6*(1+math.Abs(l[i])) { // the two pan gains differ in the last bit
			t.Fatalf("a solo voice is not in the middle")
		}
	}
}

func TestVibrato(t *testing.T) {
	cfg := steady(Solo())
	cfg.VibratoDepth, cfg.VibratoDelay, cfg.VibratoRate = 0.5, 0, 5
	_, x, _ := sing(cfg, psalm.Message{"pitch": 57.0}, 1.5)
	// the fundamental moves: measured over short windows, its peak frequency changes
	lo, hi := math.Inf(1), 0.0
	for at := 0.6; at < 1.4; at += 0.02 {
		y := x[int(at*sr()) : int(at*sr())+int(0.04*sr())]
		best, bestF := 0.0, 0.0
		for f := 200.0; f < 245; f += 0.5 {
			if a := amplitude(y, f); a > best {
				best, bestF = a, f
			}
		}
		lo, hi = math.Min(lo, bestF), math.Max(hi, bestF)
	}
	if c := 1200 * math.Log2(hi/lo); c < 40 || c > 160 {
		t.Errorf("a vibrato of half a semitone swings %.0f cents", c)
	}
}

func TestNotes(t *testing.T) {
	v := New(Solo())
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{SampleOffset: 20}, v.Config())
	v.Synthesize()
	if out := v.Output(0).Channels[0]; out[19] != 0 {
		t.Errorf("it sings before its sample")
	}
	v.Clear()
	v.Note(0.2, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, 2)
	if v.IsActive() {
		t.Errorf("a note of 0.2 s still sings after 2 s")
	}
	for name, f := range Presets {
		_, x, _ := sing(f(), psalm.Message{"pitch": 57.0}, 1)
		p := 0.0
		for _, s := range x {
			p = math.Max(p, math.Abs(s))
		}
		if p < 0.01 || p > 4 || math.IsNaN(p) {
			t.Errorf("%s peaks at %g", name, p)
		}
	}
}
