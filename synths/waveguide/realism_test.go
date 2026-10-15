package waveguide

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

// peakNear is the frequency of the strongest peak near f (a scan of Goertzel)
func peakNear(x []float64, f, span float64) float64 {
	best, bf := 0.0, f
	for i := -200; i <= 200; i++ {
		g := f + span*float64(i)/200
		if a := amplitude(x, g); a > best {
			best, bf = a, g
		}
	}
	return bf
}

// ringing is a string that rings long with its high partials
func ringing(cfg Config) Config {
	cfg = dry(plain(cfg))
	cfg.Decay, cfg.Damping, cfg.Brightness, cfg.PickPosition = 4, 0.05, 1, 0.13
	return cfg
}

func TestInharmonicPartials(t *testing.T) {
	for _, tc := range []struct{ b, pitch float64 }{{3e-5, 57}, {1.2e-4, 57}, {3e-4, 57}, {3e-4, 45}, {3e-4, 69}, {1e-3, 69}} {
		sameNoise()
		cfg := ringing(Steel())
		cfg.Inharmonicity = tc.b
		f0 := hz(tc.pitch)
		b := stiffnessAt(tc.b, f0)
		_, x := play(cfg, tc.pitch, 1)
		y := x[int(0.1*sr()):int(0.9*sr())]
		// the fundamental is the pitch asked for
		if c := cents(peakNear(y, f0, f0*0.02), f0); math.Abs(c) > 2 {
			t.Errorf("B %g pitch %g: the fundamental is %.1f cents out", tc.b, tc.pitch, c)
		}
		for n := 2; n <= 8 && f0*float64(n) < sr()/6; n++ {
			// the partials of a stiff string: f0 n sqrt(1+B n^2), the first kept at f0
			want := f0 * float64(n) * math.Sqrt((1+b*float64(n*n))/(1+b))
			if c := cents(peakNear(y, want, want*0.015), want); math.Abs(c) > 4 {
				t.Errorf("B %g pitch %g: partial %d is %.1f cents from %.1f Hz", tc.b, tc.pitch, n, c, want)
			}
		}
	}
	// the stiffness does show: the 8th partial of B 3e-4 at 220 Hz is 17 cents above the 8th harmonic
	if c := cents(math.Sqrt(1+3e-4*64), 1); c < 15 {
		t.Errorf("the test string is not stiff enough: %.1f", c)
	}
	// B stays where the setting says at 220 Hz, and rises for the low strings
	if stiffnessAt(1e-4, 220) != 1e-4 || stiffnessAt(1e-4, 55) <= 3e-4 || stiffnessAt(1e-4, 880) >= 1e-4 {
		t.Errorf("stiffness by pitch: %g %g %g", stiffnessAt(1e-4, 55), stiffnessAt(1e-4, 220), stiffnessAt(1e-4, 880))
	}
}

func TestStiffStringsStayInTune(t *testing.T) {
	// the fundamental of every string, wound bass to a thin top one, is where the pitch says
	for _, b := range []float64{3e-5, 4e-4, 2e-3} {
		for _, p := range []float64{28, 40, 52, 64, 76, 88} {
			sameNoise()
			cfg := ringing(Steel())
			cfg.Inharmonicity = b
			_, x := play(cfg, p, 0.8)
			y := x[int(0.1*sr()):]
			f0 := hz(p)
			if c := cents(peakNear(y, f0, f0*0.02), f0); math.Abs(c) > 3 {
				t.Errorf("B %g pitch %g: %.1f cents out", b, p, c)
			}
		}
	}
	// a pitch bend or a vibrato keep working with the stiffness
	sameNoise()
	cfg := ringing(Steel())
	cfg.Inharmonicity = 3e-4
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 57.0, "pitchBend": 1.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.5)
	for _, s := range x {
		if math.IsNaN(s) {
			t.Fatalf("NaN")
		}
	}
}

// slope is the fall of a decay in dB a second, between two times
func slope(x []float64, f, t0, t1 float64) float64 {
	win := 0.1
	a := amplitude(x[int(t0*sr()):int((t0+win)*sr())], f)
	b := amplitude(x[int(t1*sr()):int((t1+win)*sr())], f)
	return 20 * math.Log10(a/b) / (t1 - t0)
}

func TestTwoPolarizations(t *testing.T) {
	pitch := 52.0
	f := hz(pitch)
	decay := func(pol float64) []float64 {
		sameNoise()
		cfg := dry(plain(Guitar()))
		cfg.Decay, cfg.Polarization = 4, pol
		_, x := play(cfg, pitch, 3)
		return x
	}
	// a single polarization falls at one rate; with two it falls fast, then slowly (the aftersound)
	single := decay(0)
	if early, late := slope(single, f, 0.05, 0.6), slope(single, f, 1.5, 2.5); late < 0.7*early || late > 1.3*early {
		t.Errorf("one polarization falls %.1f then %.1f dB/s", early, late)
	}
	two := decay(1)
	early, late := slope(two, f, 0.05, 0.6), slope(two, f, 1.5, 2.5)
	if early < 2.5*late {
		t.Errorf("two polarizations fall %.1f dB/s, then %.1f: no aftersound", early, late)
	}
	if late <= 0 {
		t.Errorf("the aftersound doesn't decay: %.1f dB/s", late)
	}
	// the two are a little apart in pitch: the sum beats, its envelope falls to a notch and comes
	// back (a single string only falls)
	rises := func(x []float64) float64 {
		var env []float64
		for t0 := 0.05; t0 < 2.9; t0 += 0.1 {
			i := int(t0 * sr())
			env = append(env, 20*math.Log10(amplitude(x[i:i+int(0.1*sr())], f)+1e-9))
		}
		r := 0.0
		for i := 1; i < len(env); i++ {
			r = math.Max(r, env[i]-env[i-1])
		}
		return r
	}
	if r := rises(decay(0.6)); r < 3 {
		t.Errorf("the polarizations don't beat: the envelope rises only %.1f dB out of a notch", r)
	}
	if r := rises(single); r > 0.5 {
		t.Errorf("a single string beats: %.1f dB", r)
	}
	// 0 is the string it was: one polarization, nothing of the sideways one in the loop
	sameNoise()
	a := dry(plain(Guitar()))
	_, xa := play(a, pitch, 0.5)
	sameNoise()
	a.Polarization = 0
	_, xb := play(a, pitch, 0.5)
	for i := range xa {
		if xa[i] != xb[i] {
			t.Fatalf("differs at %d", i)
		}
	}
}

func TestBodyCoupling(t *testing.T) {
	// a note on the air resonance of the guitar (98 Hz) gives its energy to the body and dies faster
	// than the same string not coupled; a note away from the modes hardly changes
	late := func(pitch, coupling float64) float64 {
		sameNoise()
		cfg := dry(plain(Guitar()))
		cfg.Decay, cfg.BodyCoupling = 4, coupling
		_, x := play(cfg, pitch, 2)
		return amplitude(x[int(1.5*sr()):int(1.7*sr())], hz(pitch))
	}
	on, off := late(43, 1), late(43, 0)
	if 20*math.Log10(off/on) < 6 {
		t.Errorf("on the body's mode the coupling takes %.1f dB, want 6", 20*math.Log10(off/on))
	}
	on, off = late(40, 1), late(40, 0)
	if d := 20 * math.Log10(off/on); d > 6 || d < -1 {
		t.Errorf("off the modes the coupling changes the string by %.1f dB", d)
	}
	// the pitch is kept
	sameNoise()
	cfg := dry(plain(Guitar()))
	cfg.BodyCoupling = 1
	_, x := play(cfg, 47, 0.8)
	if c := cents(pitchOf(x[int(0.2*sr()):]), hz(47)); math.Abs(c) > 4 {
		t.Errorf("the coupling pulls the pitch %.1f cents", c)
	}
	// the body rings on in the string: the modes that are fed back make the wave different from
	// the plain one
	sameNoise()
	cfg.BodyCoupling = 0
	_, y := play(cfg, 47, 0.8)
	diff := 0.0
	for i := range x {
		diff += math.Abs(x[i] - y[i])
	}
	if diff < 1 {
		t.Errorf("the coupling changes nothing")
	}
}

func TestTensionModulation(t *testing.T) {
	// a hard pluck stretches the string: it starts sharp and glides down in the first 100 ms; a soft one doesn't
	glide := func(vel, tm float64) float64 {
		sameNoise()
		cfg := dry(plain(Steel()))
		cfg.TensionMod = tm
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 45.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
		x := run(v, 0.6)
		win := int(0.07 * sr())
		a := pitchOf(x[int(0.005*sr()) : int(0.005*sr())+win])
		b := pitchOf(x[int(0.4*sr()) : int(0.4*sr())+win])
		return cents(a, b)
	}
	if g := glide(1, 1); g < 8 {
		t.Errorf("a hard pluck glides down only %.1f cents", g)
	}
	if g := glide(0.2, 1); math.Abs(g) > 3 {
		t.Errorf("a soft pluck glides %.1f cents", g)
	}
	if g := glide(1, 0); math.Abs(g) > 2 {
		t.Errorf("with the tension off a hard pluck glides %.1f cents", g)
	}
	// the glide ends on the pitch
	sameNoise()
	cfg := dry(plain(Steel()))
	cfg.TensionMod = 1
	_, x := play(cfg, 45, 1.2)
	if c := cents(pitchOf(x[int(0.8*sr()):]), hz(45)); math.Abs(c) > 3 {
		t.Errorf("after the glide the string is %.1f cents out", c)
	}
}

func TestPlectrumBrighterThanFinger(t *testing.T) {
	hi := func(plectrum, width float64) float64 {
		sameNoise()
		cfg := dry(plain(Steel()))
		cfg.PluckWidth, cfg.Plectrum = width, plectrum
		_, x := play(cfg, 45, 0.4)
		return hiRatio(x, hz(45))
	}
	for _, w := range []float64{0.03, 0.08} {
		if finger, pick := hi(0, w), hi(1, w); pick < 1.3*finger {
			t.Errorf("width %g: the high partials of the finger %.4f, the plectrum %.4f", w, finger, pick)
		}
	}
	// a wide finger is darker than a thin one
	if wide, thin := hi(0, 0.12), hi(0, 0.02); wide > 0.5*thin {
		t.Errorf("a wide finger %.4f, a thin one %.4f", wide, thin)
	}
	// plucked in the middle, the shape has no even partials
	sameNoise()
	cfg := dry(plain(Steel()))
	cfg.PluckWidth, cfg.PickPosition = 0.03, 0.5
	_, x := play(cfg, 45, 0.5)
	y := x[int(0.05*sr()) : int(0.05*sr())+8192]
	if r := band(y, hz(45), 2, 4, 6) / band(y, hz(45), 1, 3, 5); r > 0.05 {
		t.Errorf("a string pulled up in the middle has even partials: %.3f", r)
	}
}

func hiRatio(x []float64, f float64) float64 {
	y := x[int(0.02*sr()) : int(0.02*sr())+8192]
	return band(y, f, 8, 9, 10, 11, 12, 13, 14) / band(y, f, 1, 2, 3, 4)
}

func TestHarmonic(t *testing.T) {
	// a finger on the node: at the middle the octave, at a third the twelfth, at a quarter two octaves
	for _, tc := range []struct{ node, ratio float64 }{{0.5, 2}, {1.0 / 3, 3}, {0.25, 4}} {
		sameNoise()
		cfg := dry(plain(Steel()))
		cfg.Harmonic, cfg.PickPosition = tc.node, 0.15
		_, x := play(cfg, 45, 1)
		y := x[int(0.1*sr()):]
		want := hz(45) * tc.ratio
		if c := cents(pitchOf(y), want); math.Abs(c) > 6 {
			t.Errorf("node %.2f: %.1f cents from the harmonic %g Hz", tc.node, c, want)
		}
		if r := amplitude(y, hz(45)) / amplitude(y, want); r > 0.1 {
			t.Errorf("node %.2f: the open string is still there: %.3f of the harmonic", tc.node, r)
		}
	}
	// the 12th fret: Harmonic 0.5 is an octave above the stopped note
	sameNoise()
	cfg := dry(Guitar())
	cfg.Harmonic = 0.5
	_, x := play(cfg, 52, 1)
	if c := cents(pitchOf(x[int(0.1*sr()):]), hz(64)); math.Abs(c) > 8 {
		t.Errorf("the guitar's harmonic is %.1f cents from the octave", c)
	}
}

func TestSlide(t *testing.T) {
	sameNoise()
	cfg := dry(plain(Guitar()))
	cfg.Slide, cfg.Decay = 0.2, 4
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 52.0}, psalm.CtrlTick{}, v.Config())
	before := run(v, 0.3)
	v.NoteOn(1, psalm.Message{"pitch": 59.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.8)
	// it is not plucked again: as loud as the ring was
	r := rms(x[:int(0.02*sr())]) / rms(before[len(before)-int(0.02*sr()):])
	if r > 1.3 || r < 0.7 {
		t.Errorf("the legato plucks again: %.2f of the level before", r)
	}
	// the glide starts at the old pitch and ends on the new one
	start := x[:int(0.05*sr())]
	if amplitude(start, hz(52)*1.1) < amplitude(start, hz(59)) {
		t.Errorf("the slide doesn't start from the old pitch")
	}
	if c := cents(pitchOf(x[int(0.4*sr()):]), hz(59)); math.Abs(c) > 3 {
		t.Errorf("the slide ends %.1f cents out", c)
	}
	// without a slide it plucks anew
	sameNoise()
	cfg.Slide = 0
	v = New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 52.0}, psalm.CtrlTick{}, v.Config())
	run(v, 0.3)
	v.NoteOn(1, psalm.Message{"pitch": 59.0}, psalm.CtrlTick{}, v.Config())
	y := run(v, 0.2)
	if c := cents(pitchOf(y[int(0.1*sr()):]), hz(59)); math.Abs(c) > 20 {
		t.Logf("pluck anew %.1f cents", c)
	}
}

func TestFretNoiseAndBuzz(t *testing.T) {
	sameNoise()
	cfg := dry(plain(Steel()))
	_, quiet := play(cfg, 52, 0.6)
	sameNoise()
	cfg.FretNoise = 1
	_, noisy := play(cfg, 52, 0.6)
	diff := make([]float64, len(quiet))
	for i := range diff {
		diff[i] = noisy[i] - quiet[i]
	}
	if a, b := rms(quiet[:int(0.04*sr())]), rms(diff[:int(0.04*sr())]); b < 0.1*a {
		t.Errorf("the finger on the string adds nothing at the start: %g against the string's %g", b, a)
	}
	// it is only at the start
	if b := rms(diff[int(0.1*sr()):]); b > 1e-6 {
		t.Errorf("the fret noise goes on: %g", b)
	}
	// the fret buzz hits a loud string and not a quiet one: more high partials from the same pluck
	buzz := func(b, vel float64) float64 {
		sameNoise()
		c := dry(plain(Steel()))
		c.FretBuzz, c.Brightness = b, 0
		v := New(c)
		v.NoteOn(1, psalm.Message{"pitch": 45.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
		x := run(v, 0.5)
		y := x[int(0.05*sr()) : int(0.05*sr())+8192]
		return band(y, hz(45), 16, 17, 18, 19, 20, 21, 22) / band(y, hz(45), 1, 2, 3)
	}
	if clean, buzzing := buzz(0, 1), buzz(1, 1); buzzing < 2*clean {
		t.Errorf("a buzzing string has %.4f of high partials, a clean one %.4f", buzzing, clean)
	}
}

// periodicity is how much a bowed note repeats itself: 1 for a steady Helmholtz motion
func periodicity(x []float64, f float64) float64 {
	T := sr() / f
	best := 0.0
	for _, lag := range []int{int(T), int(T) + 1} {
		var a, b, c float64
		for i := 0; i+lag < len(x); i++ {
			a += x[i] * x[i+lag]
			b += x[i] * x[i]
			c += x[i+lag] * x[i+lag]
		}
		best = math.Max(best, a/math.Sqrt(b*c+1e-30))
	}
	return best
}

func bowedWith(force, speed float64, pitch float64) (*Voice, []float64) {
	sameNoise()
	cfg := dry(Violin())
	cfg.VibratoDepth = 0
	cfg.BowForce, cfg.BowSpeed = force, speed
	return play(cfg, pitch, 1.5)
}

func TestBowStickSlip(t *testing.T) {
	// a force that fits the speed: the bow sticks and slips in step with the string, the Helmholtz
	// motion, a steady note on pitch
	for _, p := range []float64{55, 69} {
		_, x := bowedWith(0.3, 0.5, p)
		y := x[int(0.8*sr()):]
		if c := cents(pitchOf(y), hz(p)); math.Abs(c) > 12 {
			t.Errorf("pitch %g: the bow note is %.1f cents out", p, c)
		}
		if r := periodicity(y, hz(p)); r < 0.95 {
			t.Errorf("pitch %g: the bow note repeats itself only %.2f", p, r)
		}
		if r := rms(y); r < 0.05 {
			t.Errorf("pitch %g: a quiet bow note %.3f", p, r)
		}
	}
	// too heavy for the speed: the string is choked, a scratchy sound without a period
	for _, p := range []float64{55, 69} {
		_, x := bowedWith(1, 0.1, p)
		y := x[int(0.8*sr()):]
		if r := periodicity(y, hz(p)); r > 0.6 {
			t.Errorf("pitch %g: a scraping bow still repeats itself: %.2f", p, r)
		}
		if rms(y) < 0.02 {
			t.Errorf("pitch %g: the scratch is silent", p)
		}
	}
	// faster, the same weight holds again
	_, x := bowedWith(0.7, 0.8, 69)
	if r := periodicity(x[int(0.8*sr()):], hz(69)); r < 0.9 {
		t.Errorf("a heavy bow at speed doesn't hold: %.2f", r)
	}
	// the speed is the loudness
	_, slow := bowedWith(0.5, 0.3, 55)
	_, fast := bowedWith(0.5, 0.8, 55)
	if rms(fast[int(0.8*sr()):]) < 1.3*rms(slow[int(0.8*sr()):]) {
		t.Errorf("a faster bow isn't louder: %g vs %g", rms(slow[int(0.8*sr()):]), rms(fast[int(0.8*sr()):]))
	}
	// without BowForce the pressure plays as it did
	sameNoise()
	cfg := dry(Violin())
	cfg.VibratoDepth = 0
	_, a := play(cfg, 55, 0.5)
	sameNoise()
	cfg.BowSpeed, cfg.BowNoise, cfg.BowReverse = 0.9, 0.5, 0.1
	_, b := play(cfg, 55, 0.5)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("without a force the friction settings change the bow at %d", i)
		}
	}
}

func TestBowRosinAndReversal(t *testing.T) {
	// the grain of the rosin roughens the note
	clean := func(noise float64) float64 {
		sameNoise()
		cfg := dry(Violin())
		cfg.VibratoDepth, cfg.BowForce, cfg.BowSpeed, cfg.BowNoise = 0, 0.3, 0.5, noise
		_, x := play(cfg, 55, 1.5)
		return periodicity(x[int(0.8*sr()):], hz(55))
	}
	if a, b := clean(0), clean(1); b > a-0.01 {
		t.Errorf("the rosin doesn't roughen the note: %.3f then %.3f", a, b)
	}
	// the bow turns round: the sound dips as it stops and comes back the other way, on pitch
	sameNoise()
	cfg := dry(Violin())
	cfg.VibratoDepth, cfg.BowForce, cfg.BowSpeed, cfg.BowReverse = 0, 0.3, 0.5, 0.5
	_, x := play(cfg, 55, 1.4)
	at := func(s float64) float64 { return rms(x[int(s*sr()):int((s+0.02)*sr())]) }
	dip := math.Inf(1)
	for s := 0.44; s < 0.54; s += 0.01 {
		dip = math.Min(dip, at(s))
	}
	if before := at(0.3); dip > 0.85*before {
		t.Errorf("the reversal doesn't dip: %.3f against %.3f", dip, before)
	}
	if after, before := at(0.8), at(0.3); after < 0.6*before {
		t.Errorf("the bow doesn't come back after the reversal: %.3f against %.3f", after, before)
	}
	if c := cents(pitchOf(x[int(0.75*sr()):int(0.95*sr())]), hz(55)); math.Abs(c) > 12 {
		t.Errorf("after the reversal the note is %.1f cents out", c)
	}
	// the matrix moves the force and the speed
	sameNoise()
	c2 := dry(Violin())
	c2.VibratoDepth, c2.BowForce, c2.BowSpeed = 0, 0.3, 0.3
	c2.Ext[0] = 1
	base := rms(run(func() *Voice {
		v := New(c2)
		v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
		return v
	}(), 0.8))
	c2.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "bowSpeed", Amount: 1}
	sameNoise()
	v := New(c2)
	v.NoteOn(1, psalm.Message{"pitch": 55.0}, psalm.CtrlTick{}, v.Config())
	if fast := rms(run(v, 0.8)); fast < 1.2*base {
		t.Errorf("the matrix doesn't speed the bow: %.3f vs %.3f", base, fast)
	}
}

func TestRealismMessages(t *testing.T) {
	v := New(Guitar())
	v.ReceiveMessage(psalm.Message{"stiffness": 2e-4, "polarization": 0.5, "bodyCoupling": 0.4, "tensionMod": 0.3,
		"pluckWidth": 0.1, "plectrum": 0.8, "fretNoise": 0.2, "fretBuzz": 0.1, "harmonic": 0.25, "slide": 0.3,
		"bowForce": 0.4, "bowSpeed": 0.6, "bowNoise": 0.2, "bowReverse": 1.5})
	c := v.cfg
	if c.Inharmonicity != 2e-4 || c.Polarization != 0.5 || c.BodyCoupling != 0.4 || c.TensionMod != 0.3 || c.PluckWidth != 0.1 ||
		c.Plectrum != 0.8 || c.FretNoise != 0.2 || c.FretBuzz != 0.1 || c.Harmonic != 0.25 || c.Slide != 0.3 ||
		c.BowForce != 0.4 || c.BowSpeed != 0.6 || c.BowNoise != 0.2 || c.BowReverse != 1.5 {
		t.Errorf("messages not taken: %+v", c)
	}
	v.ReceiveMessage(psalm.Message{"inharmonicity": 5e-4})
	if v.cfg.Inharmonicity != 5e-4 {
		t.Errorf("inharmonicity")
	}
	// every preset with the refinements is described
	for _, cfg := range []Config{Steel(), UprightBass(), Rosin(), Harmonics()} {
		w := New(cfg)
		paramstest.Check(t, func() params.Set { return Describe(w.cfg) }, func(m psalm.Message) { w.ReceiveMessage(m) })
	}
}
