package waveguide

import (
	"math"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

// partialCents is how far the strongest of the first four partials of a note is from the harmonic
// of f0 it belongs to, in cents (spectral peaks: the fundamental of a bowed string can be weak)
func partialCents(y []float64, f0 float64) float64 {
	best, bc := 0.0, 0.0
	for n := 1.0; n <= 4; n++ {
		// a coarse scan and a fine one around its best
		c, ca := n*f0, 0.0
		for i := -20; i <= 20; i++ {
			g := n * f0 * (1 + 0.02*float64(i)/20)
			if a := amplitude(y, g); a > ca {
				c, ca = g, a
			}
		}
		span := n * f0 * 0.002
		g0 := c
		for i := -20; i <= 20; i++ {
			g := g0 + span*float64(i)/20
			if a := amplitude(y, g); a > ca {
				c, ca = g, a
			}
		}
		if ca > best {
			best, bc = ca, cents(c, n*f0)
		}
	}
	return bc
}

func TestStrongStiffnessPartials(t *testing.T) {
	// a wound bass or a piano string: B 2e-3 at 220 Hz, the first twelve partials within 5 cents
	for _, tc := range []struct{ b, pitch float64 }{{2e-3, 57}, {2e-3, 69}, {1e-3, 57}, {2e-3, 50}} {
		sameNoise()
		cfg := ringing(Steel())
		cfg.Inharmonicity = tc.b
		cfg.Decay = 5
		f0 := hz(tc.pitch)
		b := stiffnessAt(tc.b, f0)
		_, x := play(cfg, tc.pitch, 0.7)
		y := x[int(0.1*sr()):int(0.7*sr())]
		if c := cents(peakNear(y, f0, f0*0.02), f0); math.Abs(c) > 3 {
			t.Errorf("B %g pitch %g: the fundamental is %.1f cents out", tc.b, tc.pitch, c)
		}
		for n := 2; n <= 12; n++ {
			want := f0 * float64(n) * stretch(b, n)
			if want > sr()/3 {
				break
			}
			if c := cents(peakNear(y, want, want*0.01), want); math.Abs(c) > 5 {
				t.Errorf("B %g pitch %g: partial %d is %.1f cents from %.1f Hz", tc.b, tc.pitch, n, c, want)
			}
		}
	}
	// the design is the exact phase of the loop: the partials it is told are within a few cents
	for _, f := range []float64{110, 220, 440} {
		_, m, delay, worst := fitDispersion(2e-3, f, sr())
		if m == 0 || worst > 3 || sr()/f-delay < 4 {
			t.Errorf("%g Hz: %d sections, delay %.1f, worst %.1f cents", f, m, delay, worst)
		}
	}
}

func polyRun(p *poly.Poly[*Voice], seconds float64) []float64 {
	var x []float64
	n := p.Config().BufferSize
	for range int(math.Ceil(seconds * sr() / float64(n))) {
		p.Synthesize()
		for _, s := range p.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func monoOn(p *poly.Poly[*Voice], id string, pitch float64) {
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOn": id, "amplitude": 1.0,
		"message": psalm.Message{"pitch": pitch}})
}

func monoOff(p *poly.Poly[*Voice], id string) {
	p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOff": id})
}

// burst is the loudest 5 ms of x
func burst(x []float64) float64 {
	w := int(0.005 * sr())
	best := 0.0
	for i := 0; i+w <= len(x); i += w / 2 {
		best = max(best, rms(x[i:i+w]))
	}
	return best
}

func TestMonoLegatoSlide(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
	}{{"steel", Steel()}, {"guitar", Guitar()}, {"upright", UprightBass()}, {"violin", Violin()}, {"cello", Cello()}} {
		sameNoise()
		cfg := dry(tc.cfg)
		cfg.Slide, cfg.VibratoDepth = 0.25, 0
		if cfg.Model == Pluck {
			cfg.Decay = 4
		}
		p := NewPoly(cfg, 2)
		p.SetMono(true, true)
		from, to := 52.0, 59.0
		monoOn(p, "a", from)
		before := polyRun(p, 0.5)
		monoOn(p, "b", to) // held over a: the same string, its pitch moves
		after := polyRun(p, 0.9)
		// the string is the one that sounded: no second attack, no burst at the transition
		// (a string that glides into a resonance of the body may swell, but not jump)
		pre := burst(before[len(before)-int(0.1*sr()):])
		if first := burst(after[:int(0.01*sr())]); first > 1.25*pre {
			t.Errorf("%s: the legato has a burst: %.3f against %.3f before", tc.name, first, pre)
		}
		w := int(0.005 * sr())
		for i := w; i+w <= int(0.15*sr()); i += w {
			if a, b := rms(after[i-w:i]), rms(after[i:i+w]); b > 1.5*a+1e-4 {
				t.Errorf("%s: the level jumps from %.3f to %.3f at %d ms", tc.name, a, b, i*1000/int(sr()))
				break
			}
		}
		if r := rms(after[:int(0.01*sr())]) / rms(before[len(before)-int(0.05*sr()):]); r > 1.4 || r < 0.6 {
			t.Errorf("%s: the level at the transition is %.2f of what it was", tc.name, r)
		}
		// it starts at the old pitch and glides: the first 40 ms are nearer the old than the new
		start := after[:int(0.04*sr())]
		if amplitude(start, hz(from)*1.03) < amplitude(start, hz(to)) && amplitude(start, hz(from)) < amplitude(start, hz(to)) {
			t.Errorf("%s: the slide doesn't start from the old pitch", tc.name)
		}
		// the pitch has risen part of the way after half the slide, and ends on the new note
		end := after[int(0.5*sr()):]
		f1 := hz(to)
		if c := cents(peakNear(end, f1, f1*0.03), f1); math.Abs(c) > 6 {
			t.Errorf("%s: the glide ends %.1f cents out", tc.name, c)
		}
		for _, s := range after {
			if math.IsNaN(s) || math.Abs(s) > 20 {
				t.Fatalf("%s: %g", tc.name, s)
			}
		}
		// the held note let go: the first note comes back by gliding too, still not plucked again
		monoOff(p, "b")
		back := polyRun(p, 0.3)
		if r := burst(back[:int(0.01*sr())]) / burst(after[len(after)-int(0.01*sr()):]); r > 1.4 {
			t.Errorf("%s: coming back is a new attack: %.2f", tc.name, r)
		}
	}
	// without a Slide the next note of a mono voice is the pitch alone, the loop as it was (nothing glides)
	sameNoise()
	cfg := dry(Steel())
	p := NewPoly(cfg, 1)
	p.SetMono(true, true)
	monoOn(p, "a", 52)
	polyRun(p, 0.3)
	monoOn(p, "b", 59)
	x := polyRun(p, 0.3)
	if c := cents(pitchOf(x[int(0.1*sr()):]), hz(52)); math.Abs(c) > 20 && math.Abs(c-700) > 20 {
		t.Logf("without a slide: %.1f cents from the old note", c)
	}
}

func TestSlideOnPitchMessage(t *testing.T) {
	// a sounding string takes a new pitch as a message and glides there over the Slide
	sameNoise()
	cfg := dry(plain(Guitar()))
	cfg.Slide, cfg.Decay = 0.2, 4
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 52.0}, psalm.CtrlTick{}, v.Config())
	run(v, 0.3)
	v.ReceiveMessage(psalm.Message{"pitch": 57.0})
	x := run(v, 0.7)
	if c := cents(peakNear(x[int(0.4*sr()):], hz(57), hz(57)*0.03), hz(57)); math.Abs(c) > 3 {
		t.Errorf("the message ends the glide %.1f cents out", c)
	}
	// the slide in progress takes up a second note from where the pitch is now
	sameNoise()
	v = New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 52.0}, psalm.CtrlTick{}, v.Config())
	run(v, 0.3)
	v.ReceiveMessage(psalm.Message{"pitch": 64.0})
	run(v, 0.1)
	v.ReceiveMessage(psalm.Message{"pitch": 55.0})
	y := run(v, 0.8)
	for _, s := range y {
		if math.IsNaN(s) {
			t.Fatalf("NaN")
		}
	}
	if c := cents(peakNear(y[int(0.5*sr()):], hz(55), hz(55)*0.03), hz(55)); math.Abs(c) > 3 {
		t.Errorf("the second glide ends %.1f cents out", c)
	}
	// without Slide a pitch message does nothing to the string that sounds
	cfg.Slide = 0
	sameNoise()
	v = New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 52.0}, psalm.CtrlTick{}, v.Config())
	run(v, 0.2)
	v.ReceiveMessage(psalm.Message{"pitch": 57.0})
	z := run(v, 0.3)
	if c := cents(pitchOf(z[int(0.1*sr()):]), hz(52)); math.Abs(c) > 5 {
		t.Errorf("without a slide the sounding string moved %.1f cents", c)
	}
}

func TestHarmonicDamper(t *testing.T) {
	// a string ringing open, a finger laid on its middle: the partials without a node there die at once,
	// the others ring on (a harmonic that is pure, an octave up)
	for _, pol := range []float64{0, 0.6} {
		sameNoise()
		cfg := ringing(Steel())
		cfg.Polarization = pol
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 45.0}, psalm.CtrlTick{}, v.Config())
		x := run(v, 0.3)
		v.ReceiveMessage(psalm.Message{"harmonic": 0.5})
		x = append(x, run(v, 1)...)
		f := hz(45)
		at := func(n int, t0 float64) float64 {
			return amplitude(x[int(t0*sr()):int((t0+0.05)*sr())], f*float64(n))
		}
		for n := 1; n <= 7; n++ {
			drop := 20 * math.Log10(at(n, 0.25)/(at(n, 0.7)+1e-12))
			per := drop / 0.45
			if n%2 == 1 && drop < 30 {
				t.Errorf("pol %g partial %d without a node falls only %.1f dB", pol, n, drop)
			}
			if n%2 == 0 && per > 30 {
				t.Errorf("pol %g partial %d with a node falls %.1f dB/s", pol, n, per)
			}
		}
		// the ring that is left is on pitch: the octave
		if c := cents(peakNear(x[int(0.8*sr()):], 2*f, f*0.03), 2*f); math.Abs(c) > 5 {
			t.Errorf("pol %g: the harmonic is %.1f cents out", pol, c)
		}
	}
	// plucked with the finger down from the start: the same pure sound, and the decay of the
	// partials without a node is several times that of the ones with one
	sameNoise()
	cfg := ringing(Steel())
	cfg.Harmonic, cfg.PickPosition = 0.5, 0.2
	_, x := play(cfg, 45, 1)
	if r := amplitude(x[int(0.1*sr()):], hz(45)) / amplitude(x[int(0.1*sr()):], 2*hz(45)); r > 0.05 {
		t.Errorf("the open string rings with the harmonic: %.3f", r)
	}
	// 0 leaves the loop as it was: the damper isn't there
	sameNoise()
	cfg.Harmonic = 0
	_, y := play(cfg, 45, 0.3)
	sameNoise()
	cfg.FretBuzz = 0
	_, z := play(cfg, 45, 0.3)
	for i := range y {
		if y[i] != z[i] {
			t.Fatalf("differs at %d", i)
		}
	}
}

func TestFretContact(t *testing.T) {
	// the contact is a spring and a damper that act only where the string goes into the fret, and keep
	// the string stable at any position and pitch
	for _, fp := range []float64{0, 0.06, 0.3, 0.5, 0.8, 0.98} {
		for _, p := range []float64{28, 45, 69, 96} {
			sameNoise()
			cfg := dry(Steel())
			cfg.FretBuzz, cfg.FretPosition, cfg.Polarization = 1, fp, 1
			_, x := play(cfg, p, 0.4)
			for _, s := range x {
				if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 10 {
					t.Fatalf("position %g pitch %g: %g", fp, p, s)
				}
			}
		}
	}
	// it takes energy from the string where it hits (the low action chokes a loud pluck)
	sameNoise()
	cfg := dry(plain(Steel()))
	_, open := play(cfg, 45, 0.5)
	sameNoise()
	cfg.FretBuzz = 1
	_, low := play(cfg, 45, 0.5)
	if rms(low[int(0.2*sr()):]) > 0.8*rms(open[int(0.2*sr()):]) {
		t.Errorf("a low action doesn't cost the string anything: %.3f vs %.3f", rms(low[int(0.2*sr()):]), rms(open[int(0.2*sr()):]))
	}
}

// friction presets
var frictionPresets = []string{"violin", "cello", "slowStrings", "ponticello", "flautando", "ghostBow", "rosin"}

func TestBowedPresetsHelmholtz(t *testing.T) {
	for _, name := range frictionPresets {
		cfg := Presets[name]()
		if cfg.BowModel != 1 {
			t.Errorf("%s: not a friction bow", name)
		}
		WarmBow(cfg)
		WaitBow()
		for p := 43.0; p <= 96; p += 4 {
			sameNoise()
			c := dry(cfg)
			c.VibratoDepth, c.BowReverse, c.BowNoise = 0, 0, 0
			_, x := play(c, p, 1.3)
			y := x[int(0.7*sr()):]
			f0 := hz(p)
			if c := partialCents(y, f0); math.Abs(c) > 6 {
				t.Errorf("%s pitch %g: %.1f cents out", name, p, c)
			}
			if r := rms(y); r < 0.03 {
				t.Errorf("%s pitch %g: quiet, %.3f", name, p, r)
			}
		}
	}
	// the table bow stays reachable with the switch, and ignores the friction settings
	sameNoise()
	cfg := dry(Violin())
	cfg.VibratoDepth, cfg.BowModel = 0, 0
	_, table := play(cfg, 55, 0.6)
	sameNoise()
	cfg.BowForce, cfg.BowSpeed = 0.7, 0.7
	_, still := play(cfg, 55, 0.6)
	for i := range table {
		if table[i] != still[i] {
			t.Fatalf("the table bow follows the friction settings at %d", i)
		}
	}
	sameNoise()
	cfg.BowModel = 1
	_, fric := play(cfg, 55, 0.6)
	same := true
	for i := range table {
		if table[i] != fric[i] {
			same = false
			break
		}
	}
	if same || rms(table) < 0.03 {
		t.Errorf("the switch changes nothing, or the table bow is silent: %.3f", rms(table))
	}
	// the message
	v := New(Violin())
	v.ReceiveMessage(psalm.Message{"bowModel": 0.0})
	if v.cfg.BowModel != 0 {
		t.Errorf("bowModel message")
	}
	v.ReceiveMessage(psalm.Message{"bowModel": 1.0, "fretPosition": 0.4})
	if v.cfg.BowModel != 1 || v.cfg.FretPosition != 0.4 {
		t.Errorf("bowModel, fretPosition messages")
	}
	// the other bowed presets still sound
	for _, name := range []string{"scrape", "eBow"} {
		sameNoise()
		_, x := play(dry(Presets[name]()), 55, 1)
		if rms(x[int(0.5*sr()):]) < 0.02 {
			t.Errorf("%s is silent", name)
		}
	}
}

func TestBowedVelocityStaysOnTheString(t *testing.T) {
	// the force follows the speed: the soft and the loud bow hold the string, both
	for _, vel := range []float64{0.2, 0.5, 1} {
		sameNoise()
		cfg := dry(Violin())
		cfg.VibratoDepth = 0
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 57.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
		x := run(v, 1.5)
		y := x[int(0.8*sr()):]
		if r := periodicity(y, hz(57)); r < 0.9 {
			t.Errorf("velocity %g: the motion repeats itself only %.2f", vel, r)
		}
	}
}

func TestRefinedDescribeAndMatrix(t *testing.T) {
	for _, cfg := range []Config{Violin(), Harmonics(), UprightBass(), Scrape()} {
		w := New(cfg)
		paramstest.Check(t, func() params.Set { return Describe(w.cfg) }, func(m psalm.Message) { w.ReceiveMessage(m) })
	}
	// everything on at once, across the matrix
	all := Steel()
	all.Inharmonicity, all.Polarization, all.BodyCoupling, all.TensionMod = 0.005, 1, 1, 1
	all.FretNoise, all.FretBuzz, all.FretPosition, all.Harmonic, all.Slide = 1, 1, 0.3, 0.5, 0.1
	bowed := Violin()
	bowed.Slide = 0.1
	for _, cfg := range []Config{all, bowed, Cello(), Scrape()} {
		for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
			for d := range Dests {
				c := cfg
				c.Mods[0] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: 1}
				c.Mods[1] = modmatrix.Slot{Src: src, Dest: Dests[d].Key, Amount: -1}
				c.Ext = [4]float64{1, 1, 1, 1}
				x := playMsg(c, psalm.Message{"pitch": 40.0, "modWheel": 1.0}, 0.2)
				for _, s := range x {
					if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 20 {
						t.Fatalf("model %v src %v dest %s: %g", c.Model, src, Dests[d].Key, s)
					}
				}
			}
		}
	}
}

func TestFirstNoteDoesNotCalibrate(t *testing.T) {
	// a bow nobody has calibrated: its first note at a pitch that is not cached is tuned from an
	// estimate at once (the calibration is made in the background), and is nearly in tune
	WaitBow()
	cfg := dry(Violin())
	cfg.BowPosition, cfg.BowForce, cfg.VibratoDepth = 0.131, 0.137, 0
	p := 76.0
	sameNoise()
	v := New(cfg)
	t0 := time.Now()
	v.NoteOn(1, psalm.Message{"pitch": p}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	first := time.Since(t0)
	limit := 2 * time.Millisecond
	if raceOn {
		limit *= 10
	}
	if first > limit {
		t.Errorf("the first buffer of a note at an uncached pitch takes %v", first)
	}
	t.Logf("first buffer at an uncached pitch: %v", first)
	x := append([]float64(nil), float64s(v.Output(0).Channels[0])...)
	x = append(x, run(v, 1.2)...)
	if c := partialCents(x[int(0.7*sr()):], hz(p)); math.Abs(c) > 20 {
		t.Errorf("the estimated note is %.1f cents out", c)
	}
	// once the background has made them it is in tune
	WaitBow()
	sameNoise()
	_, y := play(cfg, p, 1.3)
	if c := partialCents(y[int(0.7*sr()):], hz(p)); math.Abs(c) > 6 {
		t.Errorf("the calibrated note is %.1f cents out", c)
	}
	// and the first buffer of one that is cached is as quick
	t0 = time.Now()
	w := New(cfg)
	w.NoteOn(1, psalm.Message{"pitch": p}, psalm.CtrlTick{}, w.Config())
	w.Synthesize()
	if d := time.Since(t0); d > limit {
		t.Errorf("a cached note takes %v", d)
	}
}

func float64s(b []float32) []float64 {
	x := make([]float64, len(b))
	for i, s := range b {
		x[i] = float64(s)
	}
	return x
}

func TestBowCacheEstimate(t *testing.T) {
	// the estimate between two cached pitches is between their offsets; with none cached it is the
	// offset of a bow at its position
	WaitBow()
	k := fricKey{freq: 300, pos: 0.123456, force: 0.31, speed: 0.12, attack: 0.08, sr: sr(), follow: true}
	if f := fricEstimate(k); !f.holds || f.d < 0.3 || f.d > 0.6 {
		t.Errorf("the offset of a bow near the bridge is %.2f", f.d)
	}
	lo, hi := k, k
	lo.freq, hi.freq = 200, 400
	fricPublish(lo, fricFit{0.2, 1, true})
	fricPublish(hi, fricFit{0.6, 1.15, true})
	k.freq = 283 // between them, in the log
	if f := fricEstimate(k); f.d <= 0.2 || f.d >= 0.6 {
		t.Errorf("the estimate %.2f is not between the neighbours", f.d)
	}
	if f, ok := fricLookup(lo); !ok || f.d != 0.2 {
		t.Errorf("lookup")
	}
}
