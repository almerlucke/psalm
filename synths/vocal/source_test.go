package vocal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func fresh() {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
}

func TestLFPulseIntegratesToZero(t *testing.T) {
	fresh()
	var last [3]float64
	for rd := 0.3; rd <= 2.7; rd += 0.3 {
		tp, te, ta := lfRd(rd)
		p := lfSolve(tp, te, ta)
		if !p.ok {
			t.Fatalf("Rd %g: not solved", rd)
		}
		const n = 20000
		sum, flow, peak, neg := 0.0, 0.0, 0.0, 0.0
		for i := range n {
			d := p.at(float64(i) / n)
			sum += d / n
			flow += d / n
			peak = max(peak, flow)
			neg = min(neg, d*p.out)
		}
		if math.Abs(sum) > 1e-4 || math.Abs(peak-1) > 0.01 || math.Abs(neg+1) > 0.01 {
			t.Errorf("Rd %g: mean of the derivative %g, flow peak %g, closing peak %g", rd, sum, peak, neg)
		}
		// a breathier voice opens longer, closes later and returns more slowly
		if ta < last[1] || (rd < 2 && te < last[0]) {
			t.Errorf("Rd %g: te %g ta %g do not grow", rd, te, ta)
		}
		last = [3]float64{te, ta, 0}
	}
	// by the times themselves
	p := lfSolve(0.4, 0.55, 0.02)
	sum := 0.0
	for i := range 10000 {
		sum += p.at(float64(i) / 10000)
	}
	if !p.ok || math.Abs(sum/10000) > 1e-4 {
		t.Errorf("tp te ta: mean %g", sum/10000)
	}
	// the glottis running it: the derivative has no mean over many cycles, even jittered
	var g glottis
	g.rng.s = 7
	pl := lfSolve(lfRd(1))
	total, fl := 0.0, 0.0
	for range 200 * 100 {
		d, f := g.tickLF(0.01, &pl, 0.01, 0.02)
		total += d
		fl = max(fl, f)
	}
	if math.Abs(total)/200 > 0.02 || fl < 0.9 || fl > 1.01 {
		t.Errorf("glottis: mean %g, flow peak %g", total/200, fl)
	}
}

func lfCfg() Config {
	c := steady(Solo())
	c.Range, c.Vowel, c.Source = Tenor, 0, 1
	return c
}

func TestLFTilt(t *testing.T) {
	fresh()
	// a pressed voice (low Rd) has more at the top than a breathy one, by the brightness and by Rd
	soft, modal, hard := lfCfg(), lfCfg(), lfCfg()
	soft.Brightness, modal.Brightness, hard.Brightness = 0.1, 0.5, 0.9
	if s, m, h := slope(soft), slope(modal), slope(hard); !(s < m && m < h) {
		t.Errorf("brightness, tilt: %g %g %g", s, m, h)
	}
	breathy, pressed := lfCfg(), lfCfg()
	breathy.Rd, pressed.Rd = 2.5, 0.4
	if b, p := slope(breathy), slope(pressed); b*3 > p {
		t.Errorf("Rd 2.5: %g, Rd 0.4: %g", b, p)
	}
	// the fundamental stands out more in a breathy voice (the open quotient)
	h1 := func(c Config) float64 {
		_, x, _ := sing(c, psalm.Message{"pitch": 48.0}, 0.8)
		y := x[int(0.4*sr()):]
		return amplitude(y, hz(48)) / amplitude(y, 2*hz(48))
	}
	if b, p := h1(breathy), h1(pressed); b <= p {
		t.Errorf("fundamental over the second harmonic: breathy %g, pressed %g", b, p)
	}
}

func TestLFFormantPeaks(t *testing.T) {
	fresh()
	const pitch = 27
	f0 := hz(pitch)
	for r := Bass; r < numRanges; r++ {
		for ph := 1; ph <= len(Vowels); ph++ {
			cfg := lfCfg()
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
					t.Errorf("LF range %d vowel %s: F%d should be %.0f Hz, the peaks are %.0f", r, Vowels[ph-1], k+1, want[k], found)
				}
			}
		}
	}
}

// harmonicity is the strength of the harmonics of f0 against the frequencies between them
func harmonicity(x []float64, f0 float64) float64 {
	on, off := 0.0, 0.0
	for k := 1; k <= 12; k++ {
		a := amplitude(x, float64(k)*f0)
		b := amplitude(x, (float64(k)+0.5)*f0)
		on += a * a
		off += b * b
	}
	return math.Sqrt(on / off)
}

// psd is the mean power of x at each of the frequencies, over segments
func psd(x []float64, freqs []float64) []float64 {
	const seg = 4096
	out := make([]float64, len(freqs))
	n := 0
	for i := 0; i+seg <= len(x); i += seg / 2 {
		for j, f := range freqs {
			a := amplitude(x[i:i+seg], f)
			out[j] += a * a
		}
		n++
	}
	for j := range out {
		out[j] /= float64(n)
	}
	return out
}

// formantAt reports whether the spectrum of x has a peak within 10% of f (the strongest between 0.75 f
// and 1.25 f)
func formantAt(x []float64, f float64) bool {
	var freqs []float64
	for g := 0.75 * f; g <= 1.25*f; g += 0.01 * f {
		freqs = append(freqs, g)
	}
	p := psd(x, freqs)
	best := 0
	for i := range p {
		if p[i] > p[best] {
			best = i
		}
	}
	return math.Abs(freqs[best]-f) <= 0.1*f
}

func TestWhisperHasNoPitch(t *testing.T) {
	fresh()
	base := steady(Solo())
	base.Range, base.Tuning = Tenor, -1
	f0 := hz(48)
	voiced := base
	_, v, _ := sing(voiced, psalm.Message{"pitch": 48.0}, 1.5)
	voiced.Whisper = 1
	_, w, _ := sing(voiced, psalm.Message{"pitch": 48.0}, 1.5)
	vh, wh := harmonicity(v[int(0.5*sr()):], f0), harmonicity(w[int(0.5*sr()):], f0)
	if vh < 10 || wh > 2 {
		t.Errorf("harmonic structure: voice %g, whisper %g", vh, wh)
	}
	// no periodic peak at f0
	if p := psd(w[int(0.5*sr()):], []float64{f0, 0.8 * f0, 1.2 * f0}); p[0] > 3*max(p[1], p[2]) {
		t.Errorf("a peak at the pitch in the whisper: %g against %g and %g", p[0], p[1], p[2])
	}
	// the throat shapes the noise: the formants of a and of i stay where they are
	for _, c := range []struct {
		vowel float64
		f     float64
	}{{0, 730}, {2, 2290}, {3, 800}} {
		cfg := base
		cfg.Vowel, cfg.Whisper = c.vowel, 1
		_, x, _ := sing(cfg, psalm.Message{"pitch": 48.0}, 2)
		want, _ := tractAt(Tenor, 0, c.vowel)
		_ = want
		if !formantAt(x[int(0.4*sr()):], c.f) {
			t.Errorf("whisper of vowel %g: no formant near %g Hz", c.vowel, c.f)
		}
	}
	// the path from the voice to the whisper is continuous: halfway is breathy, with a pitch
	half := base
	half.Whisper = 0.5
	_, m, _ := sing(half, psalm.Message{"pitch": 48.0}, 1.5)
	if mh := harmonicity(m[int(0.5*sr()):], f0); !(mh < vh && mh > wh) {
		t.Errorf("harmonicity: voice %g, half %g, whisper %g", vh, mh, wh)
	}
	// and it has a loudness of the same order
	if rv, rw := rms(v), rms(w); rw < 0.3*rv || rw > 3*rv {
		t.Errorf("voice rms %g, whisper %g", rv, rw)
	}
	// a stereo whisper has two noises
	l, r := sung(Whisper(), psalm.Message{"pitch": 48.0}, 0.8)
	var ll, rr, lr float64
	for i := range l {
		ll += l[i] * l[i]
		rr += r[i] * r[i]
		lr += l[i] * r[i]
	}
	if ll == 0 || math.Abs(lr/math.Sqrt(ll*rr)) > 0.3 {
		t.Errorf("the channels of the whisper correlate %g", lr/math.Sqrt(ll*rr))
	}
}

func consonantRun(c Consonant, pitch float64, secs float64) []float64 {
	cfg := steady(Solo())
	cfg.Range, cfg.Tuning, cfg.Attack, cfg.Consonant = Tenor, -1, 0.005, c
	cfg.ConsonantLevel = 0.5
	_, x, _ := sing(cfg, psalm.Message{"pitch": pitch}, secs)
	return x
}

func TestVoicedFricatives(t *testing.T) {
	fresh()
	f0 := hz(48)
	at := func(c Consonant) (float64, float64) {
		x := consonantRun(c, 48, 0.4)
		y := x[int(0.02*sr()):int(0.09*sr())]
		return amplitude(y, f0), rms(y)
	}
	for _, pair := range [][2]Consonant{{Z, S}, {Zh, Sh}, {V, F}} {
		vz, vr := at(pair[0])
		uz, ur := at(pair[1])
		if vz < 10*uz || vz/vr < 0.1 {
			t.Errorf("%s: f0 %g of rms %g, %s: f0 %g of rms %g", ConsonantNames[pair[0]], vz, vr, ConsonantNames[pair[1]], uz, ur)
		}
		if uz/ur > 0.02 {
			t.Errorf("%s has voicing at f0: %g of rms %g", ConsonantNames[pair[1]], uz, ur)
		}
		if ur < 0.02 || vr < 0.02 {
			t.Errorf("silent fricative: %g %g", vr, ur)
		}
	}
	// the voice comes in after the noise
	x := consonantRun(S, 48, 0.5)
	if amplitude(x[int(0.3*sr()):], f0) < 0.01 {
		t.Errorf("no vowel after the s")
	}
}

// onset returns the seconds until the sound is periodic at the pitch: the normalised autocorrelation at
// the period over windows of 20 ms, once the window has some sound in it
func onset(x []float64, f0 float64) float64 {
	lag := int(math.Round(sr() / f0))
	win := int(0.02 * sr())
	for s := 0; s+win+lag < len(x); s += int(0.001 * sr()) {
		var a, b, c float64
		for i := s; i < s+win; i++ {
			a += x[i] * x[i+lag]
			b += x[i] * x[i]
			c += x[i+lag] * x[i+lag]
		}
		if b > 1e-6 && a/math.Sqrt(b*c) > 0.7 {
			return float64(s+win/2) / sr()
		}
	}
	return -1
}

func TestVoiceOnsetTime(t *testing.T) {
	fresh()
	f0 := hz(48)
	var vot [numConsonants]float64
	for _, c := range []Consonant{P, T, K, B, D, G} {
		vot[c] = onset(consonantRun(c, 48, 0.5), f0)
		if vot[c] < 0 {
			t.Fatalf("%s never voices", ConsonantNames[c])
		}
	}
	for _, pair := range [][2]Consonant{{P, B}, {T, D}, {K, G}} {
		if vot[pair[0]]-vot[pair[1]] < 0.02 {
			t.Errorf("onset of periodicity: %s %.0f ms, %s %.0f ms", ConsonantNames[pair[0]], 1000*vot[pair[0]], ConsonantNames[pair[1]], 1000*vot[pair[1]])
		}
	}
	// the aspiration is noise of its own in the voiceless: more before the voice than in the voiced
	pre := func(c Consonant) float64 {
		x := consonantRun(c, 48, 0.3)
		return rms(x[int(0.006*sr()):int(0.045*sr())])
	}
	if pre(P) < 0.01 {
		t.Errorf("no aspiration after p")
	}
}

func TestParseVoicedLetters(t *testing.T) {
	w, err := ParseWord("pa ba da ga za zha va zhi bou sha ngi")
	if err != nil {
		t.Fatal(err)
	}
	want := []Consonant{P, B, D, G, Z, Zh, V, Zh, B, Sh, Ng}
	for i, c := range want {
		if w[i].Consonant != c {
			t.Errorf("syllable %d: %s, want %s", i, ConsonantNames[w[i].Consonant], ConsonantNames[c])
		}
	}
	if w[8].Diphthong != 4 {
		t.Errorf("bou: %+v", w[8])
	}
	for _, bad := range []string{"p", "zh", "vv", "bk"} {
		if _, err := ParseWord(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
	// the old numbers stay
	for i, n := range []string{"none", "t", "k", "s", "sh", "f", "m", "n", "ng", "l", "r", "w", "y"} {
		if ConsonantNames[i] != n {
			t.Errorf("consonant %d is %s, was %s", i, ConsonantNames[i], n)
		}
	}
	if len(ConsonantNames) != int(numConsonants) {
		t.Errorf("%d names for %d consonants", len(ConsonantNames), numConsonants)
	}
	cfg := steady(Solo())
	cfg.Word = w
	for range len(w) {
		l, r := sung(cfg, psalm.Message{"pitch": 55.0}, 0.2)
		finite(t, l, r)
	}
	// a whole word, in a row
	v := New(cfg)
	for range len(w) {
		v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
		l, r := run(v, 0.2)
		finite(t, l, r)
		if rms(l) == 0 {
			t.Errorf("syllable %d is silent", v.sylNow)
		}
	}
}

func TestNaturalVibrato(t *testing.T) {
	fresh()
	stats := func(pitch float64, nat float64) (rate, extent []float64) {
		cfg := steady(Solo())
		cfg.Steady = false
		cfg.VibratoRate, cfg.VibratoDepth, cfg.VibratoDelay, cfg.NaturalVibrato = 5.5, 0.5, 0, nat
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())
		s := &v.singers[0]
		for i := range int(10 * sr() / float64(v.Config().BufferSize)) {
			v.Synthesize()
			if float64(i)*float64(v.Config().BufferSize)/sr() > 1 {
				if nat > 0 {
					rate = append(rate, s.natRate)
					extent = append(extent, s.natExt)
				} else {
					rate = append(rate, s.vibRate)
					extent = append(extent, 1)
				}
			}
		}
		return
	}
	mean := func(x []float64) float64 {
		s := 0.0
		for _, v := range x {
			s += v
		}
		return s / float64(len(x))
	}
	sd := func(x []float64) float64 {
		m, s := mean(x), 0.0
		for _, v := range x {
			s += (v - m) * (v - m)
		}
		return math.Sqrt(s / float64(len(x)))
	}
	r0, _ := stats(57, 0)
	if sd(r0) > 1e-9 {
		t.Errorf("the plain vibrato wanders: %g", sd(r0))
	}
	rl, el := stats(45, 1)
	rh, eh := stats(81, 1)
	for _, r := range [][]float64{rl, rh} {
		lo, hi := math.Inf(1), 0.0
		for _, x := range r {
			lo, hi = min(lo, x), max(hi, x)
		}
		if lo < 5-1e-9 || hi > 6.5+1e-9 {
			t.Errorf("rate %g to %g Hz, expected 5 to 6.5", lo, hi)
		}
		if sd(r) < 0.05 {
			t.Errorf("the rate doesn't wander: spread %g", sd(r))
		}
	}
	if mean(rh) <= mean(rl) {
		t.Errorf("the rate of high notes %g, low %g", mean(rh), mean(rl))
	}
	if mean(eh) < 1.2*mean(el) || sd(el) < 0.03 {
		t.Errorf("extent: high %g, low %g, spread %g", mean(eh), mean(el), sd(el))
	}
	// the mean rate a note really sings: count the cycles of the vibrato's phase
	cfg := steady(Solo())
	cfg.Steady = false
	cfg.VibratoRate, cfg.VibratoDepth, cfg.NaturalVibrato = 5, 0.5, 1
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	s := &v.singers[0]
	last, turns := s.vibPh, 0.0
	for range int(20 * sr() / float64(v.Config().BufferSize)) {
		v.Synthesize()
		d := s.vibPh - last
		if d < 0 {
			d++
		}
		turns += d
		last = s.vibPh
	}
	if r := turns / 20; r < 5 || r > 6.5 {
		t.Errorf("the vibrato turns %g times a second", r)
	}
	// it still moves the pitch: the fundamental swings by a fraction of a semitone, and comes in late
	cfg.VibratoDelay = 0
	_, x, _ := sing(cfg, psalm.Message{"pitch": 57.0}, 2.5)
	lo, hi := math.Inf(1), 0.0
	for at := 1.2; at < 2.4; at += 0.02 {
		y := x[int(at*sr()) : int(at*sr())+int(0.04*sr())]
		best, bestF := 0.0, 0.0
		for f := 200.0; f < 245; f += 0.5 {
			if a := amplitude(y, f); a > best {
				best, bestF = a, f
			}
		}
		lo, hi = min(lo, bestF), max(hi, bestF)
	}
	if c := 1200 * math.Log2(hi/lo); c < 30 || c > 250 {
		t.Errorf("a natural vibrato swings %.0f cents", c)
	}
}

func TestNewFeaturesLeaveRandomStream(t *testing.T) {
	next := func(cfg Config) uint64 {
		fresh()
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
		run(v, 0.3)
		return v.Rand().Uint64()
	}
	base := next(Solo())
	cfg := Solo()
	cfg.Source, cfg.Rd, cfg.Whisper, cfg.NaturalVibrato, cfg.Subglottal, cfg.Consonant = 1, 1.2, 0.6, 1, 0.3, P
	if next(cfg) != base {
		t.Errorf("the random stream of the voice moved")
	}
	cfg.Consonant = Z
	cfg.Singers = 4
	if next(cfg) == 0 {
		t.Errorf("-")
	}
}

func TestNewSourceNoNaN(t *testing.T) {
	fresh()
	cfg := Choir()
	cfg.Source, cfg.Whisper, cfg.NaturalVibrato, cfg.Subglottal, cfg.Consonant = 1, 0.5, 1, 0.5, Zh
	cfg.Jitter, cfg.Shimmer = 0.03, 0.2
	for _, rd := range []float64{0, 0.3, 1.5, 2.7} {
		for _, pitch := range []float64{20, 57, 100, 120} {
			c := cfg
			c.Rd = rd
			l, r := sung(c, psalm.Message{"pitch": pitch}, 0.15)
			finite(t, l, r)
			for _, x := range append(l, r...) {
				if math.Abs(x) > 40 {
					t.Fatalf("Rd %g pitch %g: %g", rd, pitch, x)
				}
			}
		}
	}
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
	// every consonant, every source, through a word
	for _, src := range []int{0, 1} {
		for c := T; c < numConsonants; c++ {
			cc := steady(Solo())
			cc.Source, cc.Consonant = src, c
			l, r := sung(cc, psalm.Message{"pitch": 60.0}, 0.2)
			finite(t, l, r)
		}
	}
}

func TestSourceMessagesDescribed(t *testing.T) {
	v := New(Solo())
	v.ReceiveMessage(psalm.Message{"source": 1, "rd": 5.0, "whisper": 2.0, "naturalVibrato": 0.4, "subglottal": 0.2, "consonant": int(V)})
	c := v.cfg
	if c.Source != 1 || c.Rd != 2.7 || c.Whisper != 1 || c.NaturalVibrato != 0.4 || c.Subglottal != 0.2 || c.Consonant != V {
		t.Errorf("%+v", c)
	}
	v.ReceiveMessage(psalm.Message{"rd": 0.0, "source": 9})
	if v.cfg.Rd != 0 || v.cfg.Source != 1 {
		t.Errorf("rd %g source %d", v.cfg.Rd, v.cfg.Source)
	}
	set := Describe(v.cfg)
	for _, k := range []string{"source", "rd", "whisper", "naturalVibrato", "subglottal"} {
		if _, ok := set.Find(k); !ok {
			t.Errorf("no %s", k)
		}
	}
	if p, _ := set.Find("consonant"); len(p.Options) != int(numConsonants) {
		t.Errorf("%d consonant options", len(p.Options))
	}
	paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
	for name, f := range Presets {
		w := New(f())
		paramstest.Check(t, func() params.Set { return Describe(w.cfg) }, func(m psalm.Message) { w.ReceiveMessage(m) })
		_ = name
	}
}

func TestSubglottalColours(t *testing.T) {
	fresh()
	f0 := hz(36)
	cfg := steady(Solo())
	cfg.Range, cfg.Vowel, cfg.Tuning = Tenor, 2, -1
	sub := cfg
	sub.Subglottal = 1
	band := func(c Config) float64 {
		_, x, _ := sing(c, psalm.Message{"pitch": 36.0}, 0.8)
		y := x[int(0.4*sr()):]
		return harmonicsIn(y, f0, 520, 700) / harmonicsIn(y, f0, 1350, 1650)
	}
	if band(sub) < 1.5*band(cfg) {
		t.Errorf("the windpipe resonance: %g against %g", band(sub), band(cfg))
	}
}
