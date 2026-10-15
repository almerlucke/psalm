package modal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

func seeded() {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
}

// peakNear is the frequency of the strongest partial within cents of f, found by a fine scan
func peakNear(x []float64, f, cents float64) (float64, float64) {
	best, bf := 0.0, f
	for c := -cents; c <= cents; c += 0.5 {
		g := f * math.Exp2(c/1200)
		if a := amplitude(x, g); a > best {
			best, bf = a, g
		}
	}
	return bf, best
}

func TestBarModeData(t *testing.T) {
	for n, want := range map[int]float64{1: 1, 2: 2.756, 3: 5.404, 4: 8.933, 5: 13.34} {
		if got := BarRatio(n); math.Abs(got/want-1) > 0.001 {
			t.Errorf("mode %d of a free-free bar at %.4f, want %g", n, got, want)
		}
	}
	// the nodes of the first mode lie at 0.224 and 0.776, of the second at 0.132, 0.5 and 0.868
	if a := BarShape(1, 0.2242); math.Abs(a) > 0.01 {
		t.Errorf("mode 1 at its node: %g", a)
	}
	if a := BarShape(2, 0.5); math.Abs(a) > 0.01 {
		t.Errorf("mode 2 in the middle: %g", a)
	}
	if a := BarShape(1, 0); math.Abs(a-2) > 1e-6 {
		t.Errorf("mode 1 at the end: %g", a)
	}
	for n := 1; n <= 5; n++ { // a mean square of 1
		var e float64
		for i := range 2000 {
			s := BarShape(n, (float64(i)+0.5)/2000)
			e += s * s / 2000
		}
		if math.Abs(e-1) > 0.01 {
			t.Errorf("mode %d has a mean square of %g", n, e)
		}
	}
	// tuned bars and the chime
	for ins, want := range map[Instrument][]float64{InstMarimbaBar: {1, 4, 10}, InstVibraphoneBar: {1, 4, 10}, InstXylophoneBar: {1, 3, 6},
		InstTubularBell: {0, 0, 0, 2, 3, 4, 5, 6}} {
		md := modelOf(ins)
		for i, r := range want {
			if r != 0 && md.parts[i].ratio != r {
				t.Errorf("%s mode %d at %g, want %g", Instruments[ins], i+1, md.parts[i].ratio, r)
			}
		}
	}
}

func TestInstrumentsRingAtTheirRatios(t *testing.T) {
	for ins := InstBar; ins <= InstSteelPan; ins++ {
		md := modelOf(ins)
		cfg := Config{Instrument: ins, Decay: 6, Hardness: 1, Position: 0.45, Level: 1}
		_, x := strike(cfg, 69, 0.7)
		y := window(x, 0.02, 0.6)
		f := hz(69)
		strongest := 0.0
		for i := range md.parts {
			p := &md.parts[i]
			strongest = math.Max(strongest, p.amp*p.shape(0.45))
		}
		checked := 0
		for i := range md.parts {
			p := &md.parts[i]
			if p.amp*p.shape(0.45) < 0.2*strongest || p.ratio*f > 0.4*sr() {
				continue
			}
			alone := true
			for j := range md.parts {
				if j != i && math.Abs(math.Log2(md.parts[j].ratio/p.ratio)) < 0.04 {
					alone = false
				}
			}
			if !alone {
				continue
			}
			got, a := peakNear(y, p.ratio*f, 40)
			checked++
			if c := 1200 * math.Log2(got/(p.ratio*f)); math.Abs(c) > 5 || a == 0 {
				t.Errorf("%s: mode %d should sound at %.1f Hz, the peak is %.1f cents off", Instruments[ins], i+1, p.ratio*f, c)
			}
		}
		if checked == 0 {
			t.Errorf("%s: nothing measured", Instruments[ins])
		}
	}
}

// centroid is the mean of the ratios weighted by the loudness of the partials
func centroid(x []float64, f float64, ratios []float64) float64 {
	var s, w float64
	for _, r := range ratios {
		a := amplitude(x, r*f)
		s += a * r
		w += a
	}
	return s / w
}

func TestHarderAndFasterIsBrighter(t *testing.T) {
	ratios := []float64{1, 2.756, 5.404, 8.933}
	f := hz(72)
	bright := func(cfg Config, vel float64) float64 {
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 72.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
		return centroid(window(run(v, 0.2), 0.005, 0.12), f, ratios)
	}
	cfg := Glockenspiel()
	cfg.Noise, cfg.Hardness, cfg.Contact = 0, 0.4, 0.5
	if soft, loud := bright(cfg, 0.2), bright(cfg, 1); loud < soft*1.15 {
		t.Errorf("centroid %.2f at velocity 0.2, %.2f at 1: a faster blow should be brighter", soft, loud)
	}
	hard, soft := cfg, cfg
	hard.Hardness, soft.Hardness = 1, 0
	if s, h := bright(soft, 0.8), bright(hard, 0.8); h < s*1.2 {
		t.Errorf("centroid %.2f soft, %.2f hard", s, h)
	}
	light, heavy := cfg, cfg
	light.Mass, heavy.Mass = 0, 1
	if l, h := bright(light, 0.8), bright(heavy, 0.8); h > l {
		t.Errorf("a heavier mallet is brighter: %.2f against %.2f", h, l)
	}
	// the contact lasts as velocity^-0.2: slower is longer
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 72.0, "velocity": 0.3}, psalm.CtrlTick{}, v.Config())
	slow := len(v.pulse)
	v.NoteOn(1, psalm.Message{"pitch": 72.0, "velocity": 1.0}, psalm.CtrlTick{}, v.Config())
	if fast := len(v.pulse); fast >= slow {
		t.Errorf("the contact lasts %d samples slow, %d fast", slow, fast)
	}
}

func TestStrikePositionFollowsTheModeShapes(t *testing.T) {
	// a uniform bar: the middle is the node of mode 2, 0.2242 the node of mode 1
	bar := func(pos float64) (float64, float64) {
		cfg := Config{Instrument: InstBar, Decay: 3, Hardness: 1, Position: pos, Level: 1}
		_, x := strike(cfg, 60, 0.4)
		y := window(x, 0.01, 0.2)
		return amplitude(y, hz(60)), amplitude(y, BarRatio(2)*hz(60))
	}
	m1mid, m2mid := bar(0.5)
	m1node, m2side := bar(0.2242)
	if m2mid > m1mid*0.02 {
		t.Errorf("mode 2 struck at its node: %.4f of mode 1 %.4f", m2mid, m1mid)
	}
	if m1node > m2side*0.02 {
		t.Errorf("mode 1 struck at its node: %.4f of mode 2 %.4f", m1node, m2side)
	}
	// a plate struck at its centre rings only in the modes without nodal diameters: (0,1) at 1.73, not (2,0)
	plate := func(pos float64) (float64, float64) {
		cfg := Config{Instrument: InstPlate, Decay: 3, Hardness: 1, Position: pos, Level: 1}
		_, x := strike(cfg, 60, 0.4)
		y := window(x, 0.01, 0.2)
		return amplitude(y, hz(60)), amplitude(y, 9.084/5.253*hz(60))
	}
	c20, c01 := plate(0.01)
	e20, _ := plate(0.7)
	if c20 > c01*0.05 || e20 < c20*10 {
		t.Errorf("a plate struck at the centre: (2,0) %.4f, (0,1) %.4f; at 0.7 (2,0) %.4f", c20, c01, e20)
	}
}

func TestTubularBellStrikeTone(t *testing.T) {
	cfg := TubularBells()
	cfg.Noise = 0
	_, x := strike(cfg, 69, 1)
	f := hz(69)
	y := window(x, 0.01, 0.4)
	ref := amplitude(y, 3*f)
	for _, r := range []float64{2, 3, 4, 5} {
		if a := amplitude(y, r*f); db(a/ref) < -26 {
			t.Errorf("the mode at %g x the strike tone is %.1f dB", r, db(a/ref))
		}
	}
	if a := amplitude(y, f); db(a/ref) > -30 {
		t.Errorf("the strike tone itself sounds: %.1f dB", db(a/ref))
	}
	md := modelOf(InstTubularBell)
	if md.parts[0].ratio >= 1 || md.parts[2].ratio >= 1.3 {
		t.Errorf("the lowest modes %v should lie below the 2:3:4 set", md.parts[:3])
	}
	// the ear hears the period of the strike tone: the signal repeats every 1/f
	z := window(x, 0.01, 0.3)
	corr := func(lag int) float64 {
		var a, b, c float64
		for i := 0; i+lag < len(z); i++ {
			a += z[i] * z[i+lag]
			b += z[i] * z[i]
			c += z[i+lag] * z[i+lag]
		}
		return a / math.Sqrt(b*c)
	}
	T := sr() / f
	best, at := -1.0, 0.0
	for lag := T - 3; lag <= T+3; lag++ {
		i := int(lag)
		if c := corr(i); c > best {
			best, at = c, lag
		}
	}
	if best < 0.3 || math.Abs(at-T) > 3 { // the lower modes, not harmonic of it, spoil some of the match
		t.Errorf("the signal repeats at %.1f samples with %.2f, want %.1f", at, best, T)
	}
	if h := corr(int(T / 2)); h > best-0.2 {
		t.Errorf("half the period correlates %.2f", h)
	}
}

func TestHighModesDieSooner(t *testing.T) {
	for _, ins := range []Instrument{InstBar, InstGlockenspielBar, InstMarimbaBar, InstChurchBell, InstPlate} {
		md := modelOf(ins)
		cfg := Config{Instrument: ins, Decay: 2, Hardness: 1, Position: 0.45, Level: 1}
		if ins == InstChurchBell {
			cfg.Position = 0.9
		}
		_, x := strike(cfg, 69, 1.2)
		fall := func(r float64) float64 {
			return db(amplitude(window(x, 0.03, 0.15), r*hz(69)) / amplitude(window(x, 0.9, 0.15), r*hz(69)))
		}
		var low, high int
		for i, p := range md.parts {
			if p.ratio == 1 {
				low = i
			}
		}
		high = len(md.parts) - 1
		for i, p := range md.parts {
			if p.ratio > 4 && p.ratio < 12 && p.amp*p.shape(cfg.Position) > 0.05 {
				high = i
				break
			}
		}
		if lo, hi := fall(md.parts[low].ratio), fall(md.parts[high].ratio); hi < lo*1.5 {
			t.Errorf("%s: the mode at %g falls %.1f dB, the one at %g %.1f dB", Instruments[ins], md.parts[high].ratio, hi, md.parts[low].ratio, lo)
		}
	}
}

func TestResonatorTube(t *testing.T) {
	cfg := Marimba()
	cfg.Noise, cfg.Decay = 0, 1
	bare := cfg
	bare.Resonator = 0
	_, a := strike(bare, 57, 1.5)
	_, b := strike(cfg, 57, 1.5)
	f := hz(57)
	early := func(x []float64) float64 { return amplitude(window(x, 0.02, 0.2), f) }
	late := func(x []float64) float64 { return amplitude(window(x, 1.0, 0.2), f) }
	if early(b) < early(a)*1.3 {
		t.Errorf("the tube adds no energy at the fundamental: %.4f against %.4f", early(b), early(a))
	}
	if late(b)/early(b) < late(a)/early(a)*1.3 {
		t.Errorf("the tube doesn't lengthen the ring: %.4f against %.4f", late(b)/early(b), late(a)/early(a))
	}
}

func TestSplitAndDrift(t *testing.T) {
	cfg := ChurchBell()
	count := func(spread float64) int {
		c := cfg
		c.Spread = spread
		v := New(c)
		v.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
		return v.n
	}
	if off, on := count(0), count(1); on <= off {
		t.Errorf("%d modes without a spread, %d with", off, on)
	}
	// the pair of the hum beats: the fraction apart of the split
	split := func(spread float64) float64 {
		c := Config{Instrument: InstSingingBowl, Decay: 100, Hardness: 0.5, Spread: spread, Position: 0.9, Level: 1}
		_, x := strike(c, 69, 2)
		lo, hi := math.Inf(1), 0.0
		for at := 0.1; at < 1.9; at += 0.01 {
			a := amplitude(window(x, at, 0.03), 440)
			lo, hi = math.Min(lo, a), math.Max(hi, a)
		}
		return hi / lo
	}
	if plain, beating := split(0), split(1); beating < plain*3 {
		t.Errorf("the bowl beats %.1f times from the loudest to the softest with a spread, %.1f without", beating, plain)
	}

	// the random stream is left alone unless the drift is on
	next := func(drift float64, strikes int) uint64 {
		seeded()
		c := ChurchBell()
		c.Drift = drift
		v := New(c)
		for range strikes {
			v.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
		}
		return v.Rand().Uint64()
	}
	if next(0, 0) != next(0, 3) {
		t.Errorf("a strike without drift consumed the random stream")
	}
	if next(1, 0) == next(1, 3) {
		t.Errorf("the drift draws nothing")
	}
	seeded()
	c := ChurchBell()
	c.Drift = 1
	v := New(c)
	v.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	w0 := v.modes[1].w
	v.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	if w1 := v.modes[1].w; w1 == w0 || math.Abs(w1/w0-1) > 0.01 {
		t.Errorf("two notes with drift at %g and %g: want a different pitch within a few cents", w0, w1)
	}
	// deterministic with the seed
	seeded()
	v2 := New(c)
	v2.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v2.Config())
	if v2.modes[1].w != w0 {
		t.Errorf("the drift isn't deterministic")
	}
}

func TestInstrumentMessagesAndPresets(t *testing.T) {
	v := New(Marimba())
	v.ReceiveMessage(psalm.Message{"instrument": int(InstGong), "resonator": 0.3, "spread": 2.0, "drift": 0.5, "contact": 0.6, "mass": 0.7})
	if c := v.cfg; c.Instrument != InstGong || c.Resonator != 0.3 || c.Spread != 2 || c.Drift != 0.5 || c.Contact != 0.6 || c.Mass != 0.7 {
		t.Errorf("settings %+v", c)
	}
	v.ReceiveMessage(psalm.Message{"instrument": 99})
	if int(v.cfg.Instrument) != len(Instruments)-1 {
		t.Errorf("instrument %d", v.cfg.Instrument)
	}
	for _, name := range []string{"handbell", "gong", "cymbal", "celesta"} {
		if _, ok := Presets[name]; !ok {
			t.Errorf("no preset %s", name)
		}
	}
	for name, f := range Presets {
		if f().Drift != 0 {
			t.Errorf("%s drifts by default", name)
		}
	}
}

func TestInstrumentsAreFinite(t *testing.T) {
	for ins := Custom; int(ins) < len(Instruments); ins++ {
		for i, d := range Dests {
			src := modmatrix.Source(1 + (i+int(ins))%int(modmatrix.NumSources-1))
			c := Config{Instrument: ins, Modes: Marimba().Modes, Decay: 1, Hardness: 0.5, Position: 0.4, Spread: 1, Drift: 1,
				Resonator: 0.6, Contact: 1, Mass: 1, Noise: 0.2, Level: 1}
			c.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
			c.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
			c.Ext = [4]float64{1, 1, 1, 1}
			_, z := strike(c, 64, 0.1)
			finite(t, Instruments[ins]+" "+d.Key, z)
		}
	}
}
