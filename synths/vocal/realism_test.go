package vocal

import (
	"fmt"
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

// peaks returns the frequencies of the peaks of the envelope of a steady vowel: the local maxima of
// its harmonics, placed by a parabola through the three around them
func peaks(x []float64, f0, top float64) []float64 {
	var a []float64
	for k := 1; float64(k)*f0 < top; k++ {
		a = append(a, math.Log(amplitude(x, float64(k)*f0)+1e-12))
	}
	var out []float64
	for i := 1; i < len(a)-1; i++ {
		if a[i] > a[i-1] && a[i] >= a[i+1] {
			d := 0.5 * (a[i-1] - a[i+1]) / (a[i-1] - 2*a[i] + a[i+1])
			out = append(out, (float64(i+1)+d)*f0)
		}
	}
	return out
}

// finely makes the voices of a test render in small buffers, so the state can be read at an instant
func finely(t *testing.T) {
	old := psalm.CurrentEnv()
	c := psalm.CurrentStackConfig()
	c.BufferSize = 32
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(c))
	t.Cleanup(func() { psalm.SetCurrentEnv(old) })
}

func TestFormantPeaks(t *testing.T) {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	const pitch = 27 // 39 Hz: the harmonics are close enough to see the formants between
	f0 := hz(pitch)
	for r := Bass; r < numRanges; r++ {
		for ph := 1; ph <= len(Vowels); ph++ {
			cfg := steady(Solo())
			cfg.Range, cfg.Phoneme, cfg.Tuning = r, ph, -1
			_, x, _ := sing(cfg, psalm.Message{"pitch": float64(pitch)}, 1.2)
			found := peaks(x[int(0.4*sr()):], f0, 4000)
			want, _ := Formants(r, ph)
			for k := range 2 {
				ok := false
				for _, p := range found {
					if math.Abs(p-want[k]) < 0.1*want[k] {
						ok = true
					}
				}
				if !ok {
					t.Errorf("range %d vowel %s: F%d should be %.0f Hz, the peaks are %.0f", r, Vowels[ph-1], k+1, want[k], found)
				}
			}
		}
	}
}

func TestVoiceTypesDiffer(t *testing.T) {
	// a longer tract has lower formants: bass < tenor, alto < soprano < child for every vowel
	for ph := 1; ph <= len(Vowels); ph++ {
		var f [numRanges][5]float64
		for r := Bass; r < numRanges; r++ {
			f[r], _ = Formants(r, ph)
			for k := 1; k < 5; k++ {
				if f[r][k] <= f[r][k-1] {
					t.Errorf("range %d vowel %d: F%d %v", r, ph, k+1, f[r])
				}
			}
		}
		if !(f[Bass][1] < f[Tenor][1] && f[Alto][1] < f[Soprano][1] && f[Tenor][1] < f[ChildVoice][1]) {
			t.Errorf("vowel %s: F2 of bass, tenor, alto, soprano, child: %.0f %.0f %.0f %.0f %.0f", Vowels[ph-1],
				f[Bass][1], f[Tenor][1], f[Alto][1], f[Soprano][1], f[ChildVoice][1])
		}
	}
}

func TestScaleStillWorks(t *testing.T) {
	// the scale a e i o u lands on the vowels of the list
	for i, ph := range scaleRows {
		a, _ := tractAt(Tenor, 0, float64(i))
		b, _ := Formants(Tenor, ph+1)
		if a != b {
			t.Errorf("scale %d: %v, want %v", i, a, b)
		}
	}
	// halfway between a and e is between them
	a, _ := tractAt(Tenor, 0, 0)
	e, _ := tractAt(Tenor, 0, 1)
	m, _ := tractAt(Tenor, 0, 0.5)
	for k := range m {
		if m[k] < math.Min(a[k], e[k])-1e-9 || m[k] > math.Max(a[k], e[k])+1e-9 {
			t.Errorf("F%d of the mix %g not between %g and %g", k+1, m[k], a[k], e[k])
		}
	}
}

// slope is the energy of the top of the spectrum (1.5-4 kHz) against the bottom (100-500 Hz)
func slope(cfg Config) float64 {
	_, x, _ := sing(cfg, psalm.Message{"pitch": 48.0}, 0.8)
	y := x[int(0.4*sr()):]
	return harmonicsIn(y, hz(48), 1500, 4000) / harmonicsIn(y, hz(48), 100, 500)
}

func TestEffortTilt(t *testing.T) {
	base := steady(Solo())
	base.Range, base.Vowel = Tenor, 0
	soft, modal, hard := base, base, base
	soft.Brightness, modal.Brightness, hard.Brightness = 0.1, 0.5, 0.9
	if s, m, h := slope(soft), slope(modal), slope(hard); !(s < m && m < h) {
		t.Errorf("breathy, modal, pressed: %g %g %g", s, m, h)
	}
	// the open quotient alone: open wide is darker than pressed shut
	open, shut := modal, modal
	open.OpenQuotient, shut.OpenQuotient = 0.9, 0.3
	if o, s := slope(open), slope(shut); o >= s {
		t.Errorf("open quotient 0.9: %g, 0.3: %g", o, s)
	}
	// and so is the return phase
	long, short := modal, modal
	long.ReturnPhase, short.ReturnPhase = 1, 0.1
	if l, s := slope(long), slope(short); l >= s {
		t.Errorf("return phase 1: %g, 0.1: %g", l, s)
	}
	// the speed quotient is a setting too
	skew := modal
	skew.SpeedQuotient = 3
	if s := slope(skew); s == slope(modal) {
		t.Errorf("speed quotient does nothing")
	}
}

func TestGlottalPulse(t *testing.T) {
	// the flow derivative has no mean (the flow returns to zero) and its flow peaks at 1
	var g glottis
	sum, peak := 0.0, 0.0
	for range 100 * 100 {
		d, f := g.tick(0.01, 0.6, 1.4, 0, 0)
		sum += d
		peak = math.Max(peak, f)
	}
	if math.Abs(sum)/100 > 0.01 || math.Abs(peak-1) > 0.02 {
		t.Errorf("mean of the derivative %g, flow peak %g", sum/100, peak)
	}
}

func cycles(cfg Config, seconds float64) []float64 {
	psalm.Seed(1)
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	var out []float64
	last := v.singers[0].g.cycles
	for range int(seconds * sr() / float64(v.Config().BufferSize)) {
		v.Synthesize()
		if g := &v.singers[0].g; g.cycles != last {
			last = g.cycles
			out = append(out, g.cycle)
		}
	}
	return out[1:]
}

func spread(x []float64) float64 { // standard deviation over the mean
	m, s := 0.0, 0.0
	for _, v := range x {
		m += v
	}
	m /= float64(len(x))
	for _, v := range x {
		s += (v - m) * (v - m)
	}
	return math.Sqrt(s/float64(len(x))) / m
}

func TestJitter(t *testing.T) {
	cfg := steady(Solo())
	cfg.Jitter = 0.01
	if s := spread(cycles(cfg, 3)); s < 0.006 || s > 0.014 {
		t.Errorf("a jitter of 1%% varies the periods %.2f%%", s*100)
	}
	cfg.Jitter = 0
	if s := spread(cycles(cfg, 1)); s > 1e-9 {
		t.Errorf("no jitter varies the periods %g", s)
	}
	// the sound with no jitter repeats exactly, with it does not (a period of 100 samples)
	period := 100
	repeat := func(jit float64) float64 {
		c := steady(Solo())
		c.Range, c.Jitter = Tenor, jit
		_, x, _ := sing(c, psalm.Message{"frequency": sr() / float64(period)}, 1)
		y := x[int(0.6*sr()):]
		var e, d float64
		for i := 0; i+period < len(y); i++ {
			e += y[i] * y[i]
			d += (y[i] - y[i+period]) * (y[i] - y[i+period])
		}
		return d / e
	}
	if z, j := repeat(0), repeat(0.02); z > 1e-6 || j < 100*z+1e-3 {
		t.Errorf("repeat error with no jitter %g, with %g", z, j)
	}
}

func TestShimmer(t *testing.T) {
	var g glottis
	g.rng.s = 12345
	var lows []float64
	low, cycles := 0.0, 0
	for range 400 * 100 {
		d, _ := g.tick(0.01, 0.6, 1.4, 0, 0.05)
		if g.cycles != cycles { // a cycle ended with this sample: its lowest point
			if cycles > 0 && low < 0 {
				lows = append(lows, -low)
			}
			cycles, low = g.cycles, 0
		}
		low = math.Min(low, d)
	}
	if len(lows) < 100 {
		t.Fatalf("%d cycles", len(lows))
	}
	if s := spread(lows); s < 0.03 || s > 0.07 {
		t.Errorf("a shimmer of 5%% varies the peaks %.1f%%", s*100)
	}
}

func TestRandomStreamUntouched(t *testing.T) {
	next := func(cfg Config) uint64 {
		psalm.Seed(1)
		psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
		run(v, 0.2)
		return v.Rand().Uint64()
	}
	base := next(Solo())
	cfg := Solo()
	cfg.Phoneme, cfg.PhonemeTo, cfg.Nasal, cfg.Training, cfg.OpenQuotient = 3, 9, 0.5, 1, 0.5
	cfg.Jitter, cfg.Shimmer, cfg.Consonant = 0.02, 0.1, N
	if next(cfg) != base {
		t.Errorf("the voice's random stream moved")
	}
}

func TestDiphthongGlide(t *testing.T) {
	finely(t)
	cfg := steady(Solo())
	cfg.Range, cfg.Tuning = Tenor, -1
	cfg.DiphthongTime = 0.4
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 36.0, "diphthong": 1}, psalm.CtrlTick{}, v.Config()) // ai
	if v.cfg.Phoneme != 6 || v.cfg.PhonemeTo != 2 {
		t.Fatalf("ai is %d to %d", v.cfg.Phoneme, v.cfg.PhonemeTo)
	}
	f0 := hz(36)
	x, _ := run(v, 1)
	early, late := x[int(0.05*sr()):int(0.3*sr())], x[int(0.7*sr()):]
	r := func(y []float64) float64 { return harmonicsIn(y, f0, 1700, 2300) / harmonicsIn(y, f0, 900, 1300) }
	if r(late) < 10*r(early) {
		t.Errorf("the second formant doesn't glide up: %g then %g", r(early), r(late))
	}
	a, _ := Formants(Tenor, 6)
	i, _ := Formants(Tenor, 2)
	if math.Abs(v.fnow[1]-i[1]) > 0.03*i[1] || a[1] > 0.6*i[1] {
		t.Errorf("F2 ends at %g, should be %g", v.fnow[1], i[1])
	}
	// held before it glides
	c := cfg
	c.Phoneme, c.PhonemeTo, c.DiphthongHold = 6, 2, 0.5
	w := New(c)
	w.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, w.Config())
	run(w, 0.4)
	if math.Abs(w.fnow[1]-a[1]) > 0.02*a[1] {
		t.Errorf("held vowel F2 %g, want %g", w.fnow[1], a[1])
	}
	// a word with a diphthong
	cfg = steady(Solo())
	cfg.Word, _ = ParseWord("mai")
	v = New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 48.0}, psalm.CtrlTick{}, v.Config())
	l, r2 := run(v, 0.8)
	finite(t, l, r2)
	if v.sylPh != 6 || v.sylTo != 2 {
		t.Errorf("mai sings %d to %d", v.sylPh, v.sylTo)
	}
}

func TestNasals(t *testing.T) {
	finely(t)
	// in the murmur of an m the first formant is low and the energy is down under 400 Hz
	f0 := hz(36)
	low := func(y []float64) float64 { return harmonicsIn(y, f0, 100, 450) / harmonicsIn(y, f0, 600, 4000) }
	cfg := steady(Solo())
	cfg.Range, cfg.Tuning, cfg.Attack, cfg.Vowel = Tenor, -1, 0.005, 0 // an a
	cfg.Consonant = M
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, v.Config())
	x, _ := run(v, 0.5)
	if v.fnow[0] < 600 {
		t.Errorf("after the m the vowel has F1 %g", v.fnow[0])
	}
	cfg.Consonant = NoConsonant
	_, y, _ := sing(cfg, psalm.Message{"pitch": 36.0}, 0.5)
	// the first 40 ms, held at the locus
	if m, a := low(x[int(0.01*sr()):int(0.05*sr())]), low(y[int(0.01*sr()):int(0.05*sr())]); m < 3*a {
		t.Errorf("m murmurs low %g, a does %g", m, a)
	}
	// and the F1 at the start is the locus
	w := New(steadyWith(M))
	w.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, w.Config())
	run(w, 0.02)
	if w.fnow[0] > 330 || w.tract[0].nasal < 0.9 {
		t.Errorf("m starts with F1 %g and the nose at %g", w.fnow[0], w.tract[0].nasal)
	}
	// nasalised vowel: the nose pulls the energy down into the first resonance
	n0 := steady(Solo())
	n0.Range, n0.Vowel, n0.Tuning = Tenor, 1, -1
	n1 := n0
	n1.Nasal = 1
	ratio := func(c Config) float64 {
		_, y, _ := sing(c, psalm.Message{"pitch": 36.0}, 0.8)
		y = y[int(0.4*sr()):]
		return harmonicsIn(y, f0, 100, 450) / harmonicsIn(y, f0, 700, 2500)
	}
	if ratio(n1) < 3*ratio(n0) {
		t.Errorf("a nasal vowel: low against high %g, oral %g", ratio(n1), ratio(n0))
	}
}

func steadyWith(c Consonant) Config {
	cfg := steady(Solo())
	cfg.Range, cfg.Tuning, cfg.Attack, cfg.Consonant = Tenor, -1, 0.005, c
	return cfg
}

func TestCoarticulation(t *testing.T) {
	finely(t)
	// y glides from the high second formant of an i into the a
	f2 := func(c Consonant, coart float64, at float64) (float64, float64) {
		cfg := steadyWith(c)
		cfg.Coarticulation = coart
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 48.0}, psalm.CtrlTick{}, v.Config())
		run(v, at)
		return v.fnow[1], v.fnow[2]
	}
	a, _ := Formants(Tenor, 6)
	if s, _ := f2(Y, 0, 0.01); s < 1800 {
		t.Errorf("y starts with F2 %g", s)
	}
	if mid, _ := f2(Y, 0, 0.05); !(mid > a[1]*1.1 && mid < 2200) {
		t.Errorf("y is on its way at 50 ms: F2 %g (a: %g)", mid, a[1])
	}
	if end, _ := f2(Y, 0, 0.2); math.Abs(end-a[1]) > 0.01*a[1] {
		t.Errorf("y ends with F2 %g, a has %g", end, a[1])
	}
	// a quick coarticulation is there sooner
	slow, _ := f2(Y, 0.15, 0.08)
	fast, _ := f2(Y, 0.01, 0.08)
	if !(slow > fast+300) {
		t.Errorf("F2 at 80 ms, slow %g fast %g", slow, fast)
	}
	// r has the low third formant
	if _, s := f2(R, 0, 0.01); s > 1900 {
		t.Errorf("r starts with F3 %g", s)
	}
	// w is low in F2, l is dark in F3 high
	if s, _ := f2(W, 0, 0.01); s > 900 {
		t.Errorf("w starts with F2 %g", s)
	}
	if _, s := f2(L, 0, 0.01); s < 2500 {
		t.Errorf("l starts with F3 %g", s)
	}
	// the noise consonants move too
	if s, _ := f2(T, 0, 0.01); s < 1500 || s > 2000 {
		t.Errorf("t starts with F2 %g", s)
	}
}

func TestFormantTuning(t *testing.T) {
	finely(t)
	cfg := steady(Solo())
	cfg.Range, cfg.Phoneme = Soprano, 6 // an a, F1 about 900 Hz
	pitch := 84.0                       // 1047 Hz
	nat, _ := Formants(Soprano, 6)
	off := cfg
	off.Tuning = -1
	vo := New(off)
	vo.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, vo.Config())
	run(vo, 0.2)
	vt := New(cfg)
	vt.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, vt.Config())
	run(vt, 0.2)
	if math.Abs(vo.fnow[0]-nat[0]) > 0.01*nat[0] {
		t.Errorf("an untuned F1 moves: %g against %g", vo.fnow[0], nat[0])
	}
	f := hz(pitch)
	if vt.fnow[0] < 1.05*f || vt.fnow[0] > 1.4*f {
		t.Errorf("tuned F1 %g for a pitch of %g", vt.fnow[0], f)
	}
	// a low note leaves it where it is
	vl := New(cfg)
	vl.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, vl.Config())
	run(vl, 0.2)
	if math.Abs(vl.fnow[0]-nat[0]) > 0.01*nat[0] {
		t.Errorf("a low note moves F1 to %g (from %g)", vl.fnow[0], nat[0])
	}
	// and the fundamental is stronger for it through the tract (the level stage after it keeps the overall
	// level the same at every pitch, so its gain is taken out)
	level := func(c Config) float64 {
		v, x, _ := sing(c, psalm.Message{"pitch": pitch}, 0.8)
		return amplitude(x[int(0.4*sr()):], f) / v.tuneNow
	}
	if level(cfg) < 1.5*level(off) {
		t.Errorf("tuning doesn't bring out the fundamental")
	}
}

func TestSingersFormant(t *testing.T) {
	f0 := hz(48)
	band := func(training float64) float64 {
		cfg := steady(Solo())
		cfg.Range, cfg.Vowel, cfg.Training = Tenor, 0, training
		_, x, _ := sing(cfg, psalm.Message{"pitch": 48.0}, 0.8)
		y := x[int(0.4*sr()):]
		return harmonicsIn(y, f0, 2400, 3600) / harmonicsIn(y, f0, 200, 1200)
	}
	if t0, t1 := band(0), band(1); t1 < 4*t0 {
		t.Errorf("the singer's formant: %g against %g", t1, t0)
	}
}

func TestParseNewConsonants(t *testing.T) {
	w, err := ParseWord("ma ni nga la ra wa ya mai ou")
	if err != nil {
		t.Fatal(err)
	}
	want := []Consonant{M, N, Ng, L, R, W, Y, M, NoConsonant}
	for i, c := range want {
		if w[i].Consonant != c {
			t.Errorf("syllable %d: %s, want %s", i, ConsonantNames[w[i].Consonant], ConsonantNames[c])
		}
	}
	if w[7].Diphthong != 1 || w[8].Diphthong != 4 || w[8].Vowel != 3 {
		t.Errorf("diphthongs %+v %+v", w[7], w[8])
	}
	for _, bad := range []string{"m", "ng", "wy", "mia"} {
		if _, err := ParseWord(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
	// each consonant sings
	for c := T; c < numConsonants; c++ {
		cfg := steady(Solo())
		cfg.Consonant, cfg.ConsonantLevel = c, 1
		x, y := sung(cfg, psalm.Message{"pitch": 55.0}, 0.3)
		finite(t, x, y)
		if rms(x) < 0.01 {
			t.Errorf("%s is silent", ConsonantNames[c])
		}
	}
}

func TestEveryVowelRendersEverywhere(t *testing.T) {
	for r := Auto; r < numRanges; r++ {
		for ph := 0; ph <= len(Vowels); ph++ {
			cfg := Choir()
			cfg.Range, cfg.Phoneme = r, ph
			cfg.Jitter, cfg.Shimmer, cfg.Training, cfg.Nasal, cfg.FormantShift = 0.03, 0.2, 1, 0.5, 0.5
			l, rr := sung(cfg, psalm.Message{"pitch": 84.0}, 0.15)
			finite(t, l, rr)
			if rms(l) == 0 {
				t.Errorf("range %d vowel %d is silent", r, ph)
			}
		}
	}
}

func TestNoNaNNewFeatures(t *testing.T) {
	cfg := Choir()
	cfg.Phoneme, cfg.PhonemeTo, cfg.Nasal, cfg.Training = 4, 10, 0.7, 1
	cfg.Jitter, cfg.Shimmer, cfg.Consonant = 0.03, 0.2, Ng
	cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 1500, Resonance: 0.5}
	for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
		for d := range Dests {
			c := cfg
			c.Mods[0] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: 1}
			c.Mods[1] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: -1}
			c.Ext = [4]float64{1, 1, 1, 1}
			l, r := sung(c, psalm.Message{"pitch": 72.0, "modWheel": 1.0}, 0.2)
			for _, x := range append(l, r...) {
				if math.IsNaN(x) || math.IsInf(x, 0) || math.Abs(x) > 40 {
					t.Fatalf("src %v dest %s: %g", src, Dests[d].Key, x)
				}
			}
		}
	}
}

func TestNewMessages(t *testing.T) {
	v := New(Solo())
	v.ReceiveMessage(psalm.Message{"phoneme": 5, "phonemeTo": 3, "jitter": 0.01, "formantTuning": 0.0, "range": 5, "openQuotient": 0.1})
	c := v.cfg
	if c.Phoneme != 5 || c.PhonemeTo != 3 || c.Jitter != 0.01 || c.Tuning >= 0 || c.Range != ChildVoice || c.OpenQuotient != 0.2 {
		t.Errorf("%+v", c)
	}
	set := Describe(v.cfg)
	if p, ok := set.Find("formantTuning"); !ok || p.Value != 0 {
		t.Errorf("formantTuning %+v", p)
	}
	if p, ok := set.Find("phoneme"); !ok || len(p.Options) != len(Vowels)+1 {
		t.Errorf("phoneme %+v", p)
	}
	for _, k := range []string{"training", "nasal", "jitter", "shimmer", "openQuotient", "speedQuotient", "returnPhase", "phonemeTo", "diphthongTime", "coarticulation"} {
		if _, ok := set.Find(k); !ok {
			t.Errorf("no %s", k)
		}
	}
	paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
	_ = fmt.Sprint
}
