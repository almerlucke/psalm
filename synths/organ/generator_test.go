package organ

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// crossings returns the times (in samples) of the upward zero crossings of x
func crossings(x []float64) []float64 {
	var c []float64
	for i := 1; i < len(x); i++ {
		if x[i-1] < 0 && x[i] >= 0 {
			c = append(c, float64(i-1)+x[i-1]/(x[i-1]-x[i]))
		}
	}
	return c
}

// measuredHz is the frequency of a tone from the first to the last of its zero crossings
func measuredHz(x []float64) float64 {
	c := crossings(x)
	return float64(len(c)-1) * sr() / (c[len(c)-1] - c[0])
}

func cents(a, b float64) float64 { return 1200 * math.Log2(a/b) }

// the table of the gears, in the test's own words: the frequency of a wheel is 40 Hz (20 Hz of the
// shaft, 2 teeth) times the teeth of the driver over those driven, doubled for every octave
func TestGearRatioFrequencies(t *testing.T) {
	table := [12][2]float64{{85, 104}, {84, 97}, {78, 85}, {35, 36}, {103, 100}, {24, 22},
		{37, 32}, {49, 40}, {61, 47}, {33, 24}, {67, 46}, {71, 46}}
	if got := wheelHz(0); math.Abs(got-32.692) > 0.001 {
		t.Errorf("the first wheel turns at %g Hz, the Hammond's is 32.692", got)
	}
	for _, pitch := range []int{36, 45, 48, 57, 60, 69, 84, 96} {
		w := pitch - lowWheel
		want := 40 * table[w%12][0] / table[w%12][1] * math.Exp2(float64(w/12))
		o := New(clean("008000000"))
		down(o, "a", float64(pitch))
		x := render(o, 3)[int(0.3*sr()):]
		if c := cents(measuredHz(x), want); math.Abs(c) > 0.1 {
			t.Errorf("pitch %d plays %.4f Hz, the gears say %.4f (%.3f cents)", pitch, measuredHz(x), want, c)
		}
	}
	// the slight mistuning: a few tenths of a cent to a cent off equal temperament, wheel by wheel,
	// so that the semitones are not all 100 cents, the octaves are
	worst, uneven := 0.0, 0
	for w := range numWheels {
		worst = max(worst, math.Abs(wheelCents(w)))
		if w > 0 && math.Abs(cents(wheelHz(w), wheelHz(w-1))-100) > 0.3 {
			uneven++
		}
		if w >= 12 && math.Abs(cents(wheelHz(w), wheelHz(w-12))-1200) > 1e-9 {
			t.Errorf("wheel %d is not an octave above %d", w, w-12)
		}
	}
	if worst < 0.3 || worst > 3 || uneven < 10 {
		t.Errorf("the mistuning: %.2f cents at most, %d uneven semitones", worst, uneven)
	}
	// with the generator ideal the tuning is equal temperament
	cfg := clean("008000000")
	cfg.Tonewheel = 0
	o := New(cfg)
	down(o, "a", 60)
	if c := cents(measuredHz(render(o, 2)[4000:]), pitchHz(60)); math.Abs(c) > 0.1 {
		t.Errorf("an ideal generator is %.3f cents from equal temperament", c)
	}
	o.ReceiveMessage(psalm.Message{"tonewheel": 1.0})
	if c := cents(o.inc[36]*sr(), wheelHz(36)); math.Abs(c) > 1e-6 {
		t.Errorf("the tonewheel message leaves the wheel %g cents off", c)
	}
}

func TestWheelsAreNotPureSines(t *testing.T) {
	o := New(clean("008000000"))
	down(o, "a", 48)
	x := render(o, 1)[int(0.3*sr()):]
	f := gearHz(48)
	h2, h3 := amplitude(x, 2*f)/amplitude(x, f), amplitude(x, 3*f)/amplitude(x, f)
	if h2 < 0.005 || h2 > 0.06 || h3 < 0.002 || h3 > 0.04 {
		t.Errorf("harmonics of a wheel: %.4f, %.4f", h2, h3)
	}
	cfg := clean("008000000")
	cfg.Tonewheel = 0
	o = New(cfg)
	down(o, "a", 48)
	x = render(o, 1)[int(0.3*sr()):]
	if h := amplitude(x, 2*gearHz(48)) / amplitude(x, gearHz(48)); h > 1e-3 {
		t.Errorf("an ideal wheel has a second harmonic of %g", h)
	}
}

func TestDrawbarBusesAndFoldback(t *testing.T) {
	// every drawbar of a key sounds at the wheel its footage says
	for b, foot := range footages {
		bars := []byte("000000000")
		bars[b] = '8'
		o := New(clean(string(bars)))
		down(o, "a", 48)
		x := render(o, 1)[int(0.3*sr()):]
		f := gearHz(48 + foot)
		if a := amplitude(x, f); a < 0.01 {
			t.Errorf("drawbar %d: nothing at %.1f Hz (%g)", b+1, f, a)
		}
	}
	// the top key's high drawbars fold back an octave until a wheel has them: C7's 2' is C8, there is
	// no wheel for C#8's... the 1' of C7 (C10) comes out two octaves down
	if wheelOf(96+24) != wheelOf(108) || wheelOf(96+36) != wheelOf(108) || wheelOf(96+31) != wheelOf(103) {
		t.Errorf("foldback: %d %d %d", wheelOf(96+24), wheelOf(96+36), wheelOf(96+31))
	}
	o := New(clean("000008000"))
	down(o, "a", 96) // the 2' of C7 is at 120, no wheel: it plays 108
	x := render(o, 1)[int(0.3*sr()):]
	want := wheelHz(108 - lowWheel)
	if want > sr()/2 {
		t.Fatal("test sample rate too low")
	}
	if a := amplitude(x, want); a < 0.01 {
		t.Errorf("foldback: nothing at the wheel for 108 (%.1f Hz): %g", want, a)
	}
	// the bottom folds up
	o = New(clean("800000000"))
	down(o, "a", 30) // 16' of F#1 is 18, no wheel
	x = render(o, 1)[int(0.3*sr()):]
	if a := amplitude(x, gearHz(30)); a < 0.01 {
		t.Errorf("bottom foldback: %g", a)
	}
}

// attack returns the milliseconds the level of a key takes from 10 to 90% of its steady level
func attackMs(cfg Config) float64 {
	o := New(cfg)
	down(o, "a", 48)
	x := render(o, 0.06)
	win := int(0.0006 * sr())
	var env []float64
	for i := 0; i+win < len(x); i += win / 4 {
		env = append(env, rms(x[i:i+win]))
	}
	final := 0.0
	for _, e := range env[len(env)-20:] {
		final = max(final, e)
	}
	t10, t90 := -1, -1
	for i, e := range env {
		if t10 < 0 && e > 0.1*final {
			t10 = i
		}
		if t90 < 0 && e > 0.9*final {
			t90 = i
		}
	}
	return float64(t90-t10) * float64(win/4) / sr() * 1000
}

func TestContactSpreadMakesTheAttack(t *testing.T) {
	cfg := clean("888888888")
	cfg.Click = 0.5
	cfg.ContactSpread = 0
	instant := attackMs(cfg)
	cfg.ContactSpread = 4
	spread := attackMs(cfg)
	if spread < 1 || spread > 8 || spread < instant+0.7 {
		t.Errorf("attack of %.2f ms with the contacts 4 ms apart, %.2f ms with all together", spread, instant)
	}
	// contacts of a key close in the order of the bus bars and not on the same sample
	o := New(cfg)
	seen := map[int]bool{}
	for b := range 9 {
		seen[o.contactDelay(60, b)] = true
	}
	if len(seen) < 6 {
		t.Errorf("the contacts close at %d different times", len(seen))
	}
	// the sharper they close the louder the click
	if hard, soft := attackKick(1), attackKick(0); hard < 3*soft {
		t.Errorf("click kick %g, %g", hard, soft)
	}
}

// attackKick is the energy of the second difference (what a step in a smooth tone adds) at the
// start of a note
func attackKick(click float64) float64 {
	cfg := clean("008000000")
	cfg.Click = click
	o := New(cfg)
	down(o, "a", 36)
	x := render(o, 0.02)
	e := 0.0
	for i := 2; i < len(x); i++ {
		d := x[i] - 2*x[i-1] + x[i-2]
		e += d * d
	}
	return e
}

func TestScannerFollowsTheRealOne(t *testing.T) {
	// V1 to V3 use 3, 6 and 9 sections of the line: the depth 1:2:3, the chorus C the same depth
	dev := func(v Vibrato) float64 {
		cfg := clean("008000000")
		cfg.Vibrato = v
		o := New(cfg)
		down(o, "a", 69)
		return spread(render(o, 1)[int(0.2*sr()):])
	}
	v1, v2, v3 := dev(V1), dev(V2), dev(V3)
	if math.Abs(v2/v1-2) > 0.3 || math.Abs(v3/v1-3) > 0.45 {
		t.Errorf("depth V1 %.4f V2 %.4f V3 %.4f, want 1:2:3", v1, v2, v3)
	}
	// V3 swings the pitch about +-12 cents
	if c := 1200 * math.Log2(1+v3) / 2; c < 5 || c > 20 {
		t.Errorf("V3 swings %.1f cents", c)
	}
	// the pitch moves at the scanner's rate, 6.87 Hz
	cfg := clean("008000000")
	cfg.Vibrato = V3
	o := New(cfg)
	down(o, "a", 69)
	x := render(o, 3)[int(0.3*sr()):]
	if r := modRate(x); math.Abs(r-6.87) > 0.4 {
		t.Errorf("the vibrato runs at %.2f Hz", r)
	}
	// the chorus keeps the dry sound: the 440 Hz stays at full strength
	cfg.Vibrato = C3
	o = New(cfg)
	down(o, "a", 69)
	c3 := render(o, 1)[int(0.2*sr()):]
	if a := amplitude(c3, gearHz(69)); a < 0.5*amplitude(x[:len(c3)], gearHz(69)) {
		t.Errorf("chorus lost the dry sound: %g", a)
	}
}

// modRate returns how often the frequency of a tone swings per second, from the zero crossings
func modRate(x []float64) float64 {
	c := crossings(x)
	f := make([]float64, 0, len(c))
	ts := make([]float64, 0, len(c))
	for i := 1; i < len(c); i++ {
		f = append(f, sr()/(c[i]-c[i-1]))
		ts = append(ts, (c[i]+c[i-1])/2/sr())
	}
	return swingRate(f, ts, 9)
}

// swingRate is the rate at which values (at times ts) cross their mean, smoothed over n values
func swingRate(v, ts []float64, n int) float64 {
	sm := make([]float64, len(v))
	mean := 0.0
	for i := range v {
		lo, hi := max(0, i-n/2), min(len(v), i+n/2+1)
		for _, y := range v[lo:hi] {
			sm[i] += y
		}
		sm[i] /= float64(hi - lo)
		mean += sm[i]
	}
	mean /= float64(len(sm))
	up := 0
	var first, last float64
	for i := 1; i < len(sm); i++ {
		if sm[i-1] < mean && sm[i] >= mean {
			if up == 0 {
				first = ts[i]
			}
			last = ts[i]
			up++
		}
	}
	if up < 2 {
		return 0
	}
	return float64(up-1) / (last - first)
}

func TestPreampAsymmetryAndTone(t *testing.T) {
	harm := func(drive float64) float64 {
		cfg := clean("008000000")
		cfg.Tonewheel = 0
		cfg.Drive = drive
		o := New(cfg)
		down(o, "a", 57)
		x := render(o, 1)[int(0.3*sr()):]
		return amplitude(x, 2*gearHz(57)) / amplitude(x, gearHz(57))
	}
	if clean, driven := harm(0), harm(1); clean > 1e-3 || driven < 0.02 {
		t.Errorf("even harmonics: %g clean, %g driven (an asymmetric tube makes them)", clean, driven)
	}
	bright := func(tone float64) float64 {
		cfg := clean("808000008")
		cfg.Tonewheel = 0
		cfg.Tone = tone
		o := New(cfg)
		down(o, "a", 60)
		x := render(o, 1)[int(0.3*sr()):]
		return amplitude(x, pitchHz(96)) / amplitude(x, pitchHz(48)) // the 1' against the 16'
	}
	if dark, mid, light := bright(0), bright(0.5), bright(1); !(dark < mid && mid < light) {
		t.Errorf("tone: %g %g %g", dark, mid, light)
	}
	// the tone control at the middle is transparent
	a := play(clean("888000000"), 0.2)
	cfg := clean("888000000")
	cfg.Tone = 0.5
	b := play(cfg, 0.2)
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("tone at 0.5 changes the sound")
		}
	}
}

func TestNewSettingsAndMatrixAreFinite(t *testing.T) {
	cfg := Jazz()
	cfg.Leslie, cfg.LeslieSpeed, cfg.Leakage, cfg.Drive, cfg.Tone = 1, 1, 1, 1, 0.9
	cfg.ClickRand, cfg.Click = 1, 1
	for _, d := range []string{"leslie", "leslieSpeed", "tone", "drive", "pitch"} {
		for src := modmatrix.Source(1); src < modmatrix.NumSources; src++ {
			c := cfg
			c.Mods[0] = modmatrix.Slot{Src: src, Dest: d, Amount: 1}
			c.Mods[1] = modmatrix.Slot{Src: src, Dest: d, Amount: -1}
			c.Ext = [4]float64{1, 1, 1, 1}
			for _, v := range play(c, 0.15) {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Fatalf("%s from source %d: %g", d, src, v)
				}
			}
		}
	}
	o := New(DefaultConfig())
	o.ReceiveMessage(psalm.Message{"leslie": 0.7, "leslieSpeed": 1.0, "lesliePosition": 0.2, "tone": 0.8,
		"contactSpread": 2.0, "vibratoScale": 2.0})
	c := o.cfg
	if c.Leslie != 0.7 || c.LeslieSpeed != 1 || c.LesliePosition != 0.2 || c.Tone != 0.8 || c.ContactSpread != 2 || c.VibratoScale != 2 {
		t.Errorf("settings %+v", c)
	}
}
