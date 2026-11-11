package piano

import (
	"math"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

// quietChord is a chord without the random noise of the action, so renders can be subtracted
func quietCfg(bridge float64) Config {
	cfg := grand()
	cfg.Mechanics = 0
	cfg.Bridge = bridge
	return cfg
}

var lowChord = []psalm.Message{{"pitch": 36.0}, {"pitch": 48.0}, {"pitch": 52.0}, {"pitch": 55.0}, {"pitch": 60.0}, {"pitch": 67.0}}

// With the bridge off every key drains on its own and a chord is the sum of its notes; with the
// strings coupled through the shared board it is not: the notes drive and load each other
func TestChordIsNotTheSum(t *testing.T) {
	chord := func(bridge float64) (a, sum []float64) {
		cfg := quietCfg(bridge)
		a = seeded(cfg, lowChord, 0.5, 5, nil)
		sum = make([]float64, len(a))
		for _, m := range lowChord {
			x := seeded(cfg, []psalm.Message{m}, 0.5, 5, nil)
			for i := range sum {
				sum[i] += x[i]
			}
		}
		return
	}
	a, sum := chord(0)
	if r := rmsDB(difference(a, sum), 0, 2) - rmsDB(a, 0, 2); r > -80 {
		t.Errorf("without the bridge a chord should be the sum of its notes, the difference is %.1f dB", r)
	}
	a, sum = chord(1)
	residual := rmsDB(difference(a, sum), 0, 2) - rmsDB(a, 0, 2)
	t.Logf("residual %.1f dB, mid chord %.2f sum %.2f, tail chord %.2f sum %.2f", residual, rmsDB(a, 0.5, 1.5), rmsDB(sum, 0.5, 1.5), rmsDB(a, 3, 5), rmsDB(sum, 3, 5))
	if residual < -30 {
		t.Errorf("with the bridge a chord should differ from the sum of its notes, the difference is only %.1f dB", residual)
	}

	// the spectrum differs: the partials the notes share exchange their energy (some lose, some gain)
	sa, ss := spectrum(a[int(0.5*sr):int(2.5*sr)], 131072), spectrum(sum[int(0.5*sr):int(2.5*sr)], 131072)
	var worst, moved float64
	count := 0
	size := 131072.0
	for k := int(100 * size / sr); k < int(3000*size/sr); k++ {
		if sa[k] > 1e-3*maxOf(sa) && ss[k] > 1e-3*maxOf(ss) {
			d := math.Abs(20 * math.Log10(sa[k]/ss[k]))
			worst = max(worst, d)
			if d > 3 {
				moved++
			}
			count++
		}
	}
	if worst < 6 || moved < 5 {
		t.Errorf("the spectrum of the chord should differ from the sum of its notes: %v of %d bins differ by more than 3 dB, the most %.1f dB", moved, count, worst)
	}
	// and the decay: the notes' tails are not the sum of the tails
	if d := math.Abs(rmsDB(a, 0.5, 1.5) - rmsDB(sum, 0.5, 1.5)); d < 0.5 {
		t.Errorf("the decay of the chord should differ from that of the sum of its notes: %.2f dB", d)
	}
}

func maxOf(x []float64) float64 {
	m := 0.0
	for _, v := range x {
		m = max(m, v)
	}
	return m
}

// The sustain pedal lets all strings ring and wash through the shared board: the tail is far
// longer than without the pedal, and the bridge lengthens it further (the energy that the modes of
// the board drain at the partials on them the strings in between keep)
func TestSustainWash(t *testing.T) {
	tail := func(bridge, pedal float64) float64 {
		a := seeded(quietCfg(bridge), lowChord, 0.3, 8, psalm.Message{"sustain": pedal})
		return rmsDB(a, 4, 5)
	}
	up, down, wash := tail(0.6, 0), tail(0.6, 1), tail(1, 1)
	plain := tail(0, 1)
	t.Logf("tail 4-5 s: no pedal %.1f dB, pedal %.1f dB, pedal and full bridge %.1f dB, pedal without bridge %.1f dB", up, down, wash, plain)
	if down < up+30 {
		t.Errorf("the pedal should lengthen the tail: %.1f dB against %.1f dB", down, up)
	}
	if wash < plain+1.5 {
		t.Errorf("the bridge should lengthen the washed tail: %.1f dB against %.1f dB", wash, plain)
	}
}

// The bridge strikes a balance: it shapes how a string drains, not how much. A single note decays
// as long as it does without it, within a dB or two
func TestBridgeKeepsLevel(t *testing.T) {
	for _, p := range []float64{40.0, 60.0, 80.0} {
		on, off := quietCfg(1), quietCfg(0)
		a := seeded(on, []psalm.Message{{"pitch": p}}, 3, 2, nil)
		b := seeded(off, []psalm.Message{{"pitch": p}}, 3, 2, nil)
		if d := math.Abs(rmsDB(a, 0.2, 1.2) - rmsDB(b, 0.2, 1.2)); d > 3 {
			t.Errorf("key %v: the bridge changes the level by %.1f dB", p, d)
		}
	}
}

// The strings on the bridge only lose to it: no key blows up, with the bridge full and the
// pedal down, whatever is played
func TestBridgeStable(t *testing.T) {
	cfg := quietCfg(1)
	p := NewPoly(cfg, 16)
	p.ReceiveMessage(psalm.Message{"sustain": 1.0})
	for k := range 12 {
		p.Note(0.5, 1, psalm.Message{"pitch": float64(24 + 7*k), "velocity": 1.0}, psalm.CtrlTick{}, p.Config())
	}
	peak := 0.0
	for range 400 {
		p.Synthesize()
		for _, ch := range p.Output(0).Channels {
			for _, x := range ch {
				if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
					t.Fatal("not a number")
				}
				peak = max(peak, math.Abs(float64(x)))
			}
		}
	}
	if peak > 50 {
		t.Errorf("unstable: %v", peak)
	}
}

// The wound strings of the bass are less inharmonic than the plain strings of the same pitch would
// be, with a step where the plain strings begin: the partials the strings are designed with are
// where the model puts them
func TestWoundStep(t *testing.T) {
	sr := 44100.0
	cents := func(key int, wound float64) (got, want float64) {
		d := dispersion(sr, 1, wound)[key]
		f0 := 440 * math.Exp2((float64(key)-69)/12)
		p := designString(stringDesign{sr: sr, f0: f0, t60: 10, stages: d.stages, ad: d.ad})
		b := inharmonicityAt(float64(key), wound)
		k := 8.0
		w0 := 2 * math.Pi * f0 / sr
		w := partial(p, k, k*w0*math.Sqrt((1+b*k*k)/(1+b)))
		return 1200 * math.Log2(w/(k*w0)), 600 * math.Log2((1+b*k*k)/(1+b))
	}
	for _, key := range []int{30, 36, 40, 42, 47, 50, 55} {
		got, want := cents(key, 1)
		if math.Abs(got-want) > 2.5 {
			t.Errorf("key %d: partial 8 is %.1f cents sharp, the model has %.1f", key, got, want)
		}
	}
	// the step: the first plain strings are much stiffer than the last wound ones
	below, _ := cents(42, 1)
	above, _ := cents(47, 1)
	flat42, _ := cents(42, 0)
	flat47, _ := cents(47, 0)
	t.Logf("partial 8: wound %.1f, %.1f cents at keys 42, 47; plain %.1f, %.1f", below, above, flat42, flat47)
	if above < 1.8*below {
		t.Errorf("no step in the inharmonicity at the break: %.1f cents below, %.1f above", below, above)
	}
	if flat47 > 1.5*flat42 {
		t.Errorf("without wound strings the inharmonicity runs on smoothly: %.1f, %.1f", flat42, flat47)
	}
	// smooth: the transition takes more than one key, and the curve has no jump anywhere else
	w := func(k float64) float64 { return inharmonicityAt(k, 1) }
	if r := w(45) / w(44); r < 1.15 || r > 2 {
		t.Errorf("the break should take a few keys: %.2f", r)
	}
	for k := 21.0; k < 107; k++ {
		if k >= 42 && k <= 47 {
			continue
		}
		if r := w(k+1) / w(k); r > 1.6 || r < 1/1.6 {
			t.Errorf("a jump in the inharmonicity at key %v: %.2f", k, r)
		}
	}
	// and the decay: copper wrap rings shorter and darker
	cfg := GrandPreset()
	ringW, dampW, massW := woundTraits(36, &cfg)
	ringP, dampP, massP := woundTraits(60, &cfg)
	if ringW >= ringP || dampW <= dampP || massW <= massP {
		t.Errorf("wound strings should ring shorter, darker and heavier: %v %v %v against %v %v %v", ringW, dampW, massW, ringP, dampP, massP)
	}
	cfg.Wound = 0
	if r, d, m := woundTraits(36, &cfg); r != 1 || d != 1 || m != 1 {
		t.Errorf("no wound strings, no change: %v %v %v", r, d, m)
	}
}

// The wound strings are heard: the partials of a bass note are where the inharmonicity of the wound
// string puts them, and the plain model puts them elsewhere
func TestWoundPartials(t *testing.T) {
	p := 40.0
	for _, wound := range []float64{0, 1} {
		cfg := single()
		cfg.Wound = wound
		x, _ := play(cfg, psalm.Message{"pitch": p, "velocity": 0.6}, 3, 1.6)
		seg := x[int(0.3*sr):int(1.5*sr)]
		f0 := 440 * math.Exp2((stretched(p, &cfg)-69)/12)
		b := inharmonicityAt(p, wound)
		f1 := peakFrequency(seg, f0, 0.01)
		for _, n := range []float64{6, 10} {
			want := n * f1 * math.Sqrt((1+b*n*n)/(1+b))
			got := peakFrequency(seg, want, 0.006)
			if c := 1200 * math.Log2(got/want); math.Abs(c) > 2.5 {
				t.Errorf("wound %v: partial %v is %.1f cents from %.1f Hz", wound, n, c, want)
			}
		}
	}
}

// A hard strike in the bass is preceded by a burst of longitudinal waves: its own energy at high
// frequencies in the first milliseconds, which a soft strike (and the treble) doesn't have
func TestPrecursor(t *testing.T) {
	burst := func(pitch, vel float64) float64 {
		cfg := quietCfg(0.6)
		cfg.Bark = 0
		off := cfg
		off.Precursor = 0
		cfg.Precursor = 1
		m := []psalm.Message{{"pitch": pitch, "velocity": vel}}
		a := seeded(cfg, m, 1, 0.3, nil)
		b := seeded(off, m, 1, 0.3, nil)
		d := difference(a, b)
		return rmsDB(d, 0, 0.015) - rmsDB(b, 0, 0.015)
	}
	loud, soft, treble := burst(33, 1), burst(33, 0.3), burst(84, 1)
	t.Logf("precursor against the note in the first 15 ms: loud bass %.1f dB, soft bass %.1f dB, loud treble %.1f dB", loud, soft, treble)
	if loud < -32 || loud > -10 {
		t.Errorf("the precursor should be heard in a loud bass note: %.1f dB", loud)
	}
	if soft > loud-15 || treble > loud-30 {
		t.Errorf("the precursor should not be in soft or treble notes: %.1f dB soft, %.1f dB treble, %.1f dB loud", soft, treble, loud)
	}
	// it is a burst: gone before the note has run its course (the voice alone, the soundboard
	// would ring it on)
	cfg := quietCfg(0.6)
	off := cfg
	off.Precursor = 0
	m := psalm.Message{"pitch": 33.0, "velocity": 1.0}
	x, _ := play(cfg, m, 1, 0.5)
	y, _ := play(off, m, 1, 0.5)
	d := difference(x, y)
	if early, late := rmsDB(d, 0, 0.03), rmsDB(d, 0.2, 0.5); late > early-40 {
		t.Errorf("the precursor should be a burst: %.1f dB early, %.1f dB later", early, late)
	}
}

// The strike point moves with the register: 1/8 in the bass towards 1/12 in the treble
func TestStrikeRegister(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Variation = 0
	bass, treble := keyStrikeAt(&cfg, 30), keyStrikeAt(&cfg, 105)
	if math.Abs(bass-0.125) > 0.01 || math.Abs(treble-1.0/12) > 0.01 {
		t.Errorf("the strike point should go from 1/8 to 1/12: %.3f, %.3f", bass, treble)
	}
	cfg.StrikeRegister = 0
	if keyStrikeAt(&cfg, 105) != cfg.StrikePosition {
		t.Errorf("without the register the strike point stays")
	}
}

// Ten notes for eight seconds render well within real time
func TestCPU(t *testing.T) {
	cfg := GrandPreset()
	p := NewPoly(cfg, 16)
	p.ReceiveMessage(psalm.Message{"sustain": 1.0})
	for k := range 10 {
		p.Note(6, 0.9, psalm.Message{"pitch": float64(36 + 4*k), "velocity": 0.8}, psalm.CtrlTick{}, p.Config())
	}
	seconds := 8.0
	start := time.Now()
	for range int(seconds*sr/float64(p.Config().BufferSize)) + 1 {
		p.Synthesize()
	}
	took := time.Since(start).Seconds()
	t.Logf("%.1f s rendered in %.2f s (%.0f%% of real time)", seconds, took, 100*took/seconds)
	if took > 0.5*seconds {
		t.Errorf("too slow: %.2f s for %.0f s of audio", took, seconds)
	}
}

// The new settings are described and reach the instrument
func TestDescribeBridge(t *testing.T) {
	p := NewPoly(GrandPreset(), 2)
	paramstest.Check(t, func() params.Set { return Describe(p.cfg) }, func(m psalm.Message) { p.ReceiveMessage(m) })
	found := map[string]bool{}
	for _, q := range Describe(p.cfg).Params {
		found[q.Key] = true
	}
	for _, k := range []string{"bridge", "wound", "precursor", "strikeRegister"} {
		if !found[k] {
			t.Errorf("%s is not described", k)
		}
	}
}

// No NaN on any key at any velocity with every new feature up, in an instrument
func TestNoNaNBridge(t *testing.T) {
	cfg := GrandPreset()
	cfg.Bridge, cfg.Wound, cfg.Precursor = 1, 1, 1
	for key := 21; key <= 108; key++ {
		p := NewPoly(cfg, 3)
		for _, vel := range []float64{0, 0.3, 0.8, 1} {
			p.Note(0.1, 1, psalm.Message{"pitch": float64(key), "velocity": vel}, psalm.CtrlTick{}, p.Config())
		}
		for range 6 {
			p.Synthesize()
			for _, ch := range p.Output(0).Channels {
				for _, x := range ch {
					if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) || math.Abs(float64(x)) > 100 {
						t.Fatalf("key %d: %v", key, x)
					}
				}
			}
		}
	}
}
