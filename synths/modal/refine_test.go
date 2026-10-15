package modal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

func cents(a, b float64) float64 { return 1200 * math.Log2(a/b) }

func TestUndercutSolverReproducesTheUniformBar(t *testing.T) {
	r := UndercutRatios(0, 0.3, 1)
	for i, want := range []float64{1, 2.756, 5.404, 8.933, 13.344, 18.638} {
		if math.Abs(r[i]/want-1) > 0.005 {
			t.Errorf("uniform bar mode %d at %.4f, want %.3f", i+1, r[i], want)
		}
	}
	// the shapes agree with the closed form of the free-free beam
	s := solveBeam(archProfile(0, 0.3, 1), newSubspace())
	for k := range 3 {
		for _, x := range []float64{0.05, 0.3, 0.45, 0.8, 0.97} {
			if got, want := s.at(k, x), BarShape(k+1, x); math.Abs(got-want) > 0.03*2 {
				t.Errorf("mode %d at %.2f: %.3f, want %.3f", k+1, x, got, want)
			}
		}
	}
}

func TestUndercutBarsAreTuned(t *testing.T) {
	for _, c := range []struct {
		ins  Instrument
		want []float64
	}{{InstMarimbaBar, []float64{1, 4, 10}}, {InstVibraphoneBar, []float64{1, 4, 10}}, {InstXylophoneBar, []float64{1, 3, 6}}} {
		md := modelOf(c.ins)
		for i, w := range c.want {
			if got := md.arch.ratio(i, 1); math.Abs(cents(got, w)) > 2 {
				t.Errorf("%s mode %d at %.4f, want %g", Instruments[c.ins], i+1, got, w)
			}
		}
		// the arch moves the pitch of the upper modes: undercut 0 is the uniform bar
		if got := md.arch.ratio(1, 0); math.Abs(got/2.756-1) > 0.005 {
			t.Errorf("%s undercut 0: mode 2 at %.3f", Instruments[c.ins], got)
		}
		mid := md.arch.ratio(1, 0.5)
		if !(mid > 2.8 && mid < c.want[1]) {
			t.Errorf("%s undercut 0.5: mode 2 at %.3f", Instruments[c.ins], mid)
		}
	}
	// the nodes of mode 1 move with the arch: at the uniform bar's node a marimba bar still rings
	md := modelOf(InstMarimbaBar)
	if md.arch.shape(0, 0.2242, 0) > 0.02 || md.arch.shape(0, 0.2242, 1) < 0.1 {
		t.Errorf("node of mode 1 at 0.2242: uniform %.3f, arched %.3f", md.arch.shape(0, 0.2242, 0), md.arch.shape(0, 0.2242, 1))
	}
}

func TestUndercutMovesTheRingingPartials(t *testing.T) {
	ring := func(u float64) (float64, float64) {
		cfg := Marimba()
		cfg.Noise, cfg.Hardness, cfg.Resonator, cfg.Decay = 0, 1, 0, 3
		cfg.Undercut = u
		_, x := strike(cfg, 57, 0.6)
		y := window(x, 0.01, 0.3)
		f := hz(57)
		return amplitude(y, 4*f), amplitude(y, 2.756*f)
	}
	a4, a276 := ring(1)
	b4, b276 := ring(0.01)
	if a4 < a276*10 || b276 < b4*10 {
		t.Errorf("undercut 1: %g at 4, %g at 2.756; uniform: %g at 4, %g at 2.756", a4, a276, b4, b276)
	}
	v := New(Marimba())
	v.ReceiveMessage(psalm.Message{"undercut": 0.4, "nonlinearity": 0.7})
	if v.cfg.Undercut != 0.4 || v.cfg.Nonlinearity != 0.7 {
		t.Errorf("settings %+v", v.cfg)
	}
}

// gongState plays a gong and reports, every tenth of a second, the energy of the upper modes in dB and
// how sharp the lowest mode is
func gongState(vel, nl float64, seconds float64) (high, shift []float64) {
	seeded()
	cfg := Gong()
	cfg.Noise, cfg.Nonlinearity = 0, nl
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 57.0, "velocity": vel}, psalm.CtrlTick{}, v.Config())
	w0 := v.modes[0].w
	for range int(seconds * 10) {
		run(v, 0.1)
		e := 0.0
		for k := range v.n {
			if m := &v.modes[k]; m.w > 6*w0 {
				e += m.re*m.re + m.im*m.im
			}
		}
		high = append(high, 10*math.Log10(e+1e-30))
		shift = append(shift, math.Atan2(v.modes[0].s, v.modes[0].c)/w0-1)
	}
	return
}

func TestGongBloomAndPitchGlide(t *testing.T) {
	hard, hardShift := gongState(1, 0.6, 4)
	soft, softShift := gongState(0.2, 0.6, 4)
	linear, linearShift := gongState(1, 0, 4)
	peak := func(x []float64) (int, float64) {
		best, at := math.Inf(-1), 0
		for i := 1; i < len(x); i++ {
			if x[i] > best {
				best, at = x[i], i
			}
		}
		return at, best
	}
	// a hard blow blooms: the upper modes rise after the strike, peak later than the first tenth and settle
	at, top := peak(hard)
	if at < 2 || top < hard[0]+2 || hard[len(hard)-1] > top-30 {
		t.Errorf("hard gong: the upper modes %v dB, peak at %d: no bloom", hard[:8], at)
	}
	if at, top := peak(soft); at > 1 && top > soft[0]+1 {
		t.Errorf("soft gong blooms: %v", soft[:8])
	}
	if at, top := peak(linear); at > 1 && top > linear[0]+0.5 {
		t.Errorf("a linear gong blooms: %v", linear[:8])
	}
	// the pitch starts sharp and falls: more than 20 cents for a hard blow, nothing for a soft one
	if c := cents(1+hardShift[0], 1); c < 20 {
		t.Errorf("hard blow only %.1f cents sharp", c)
	}
	if c := cents(1+hardShift[0], 1+hardShift[len(hardShift)-1]); c < 20 {
		t.Errorf("the pitch fell %.1f cents", c)
	}
	if c := cents(1+softShift[0], 1); c > 3 {
		t.Errorf("a soft blow %.1f cents sharp", c)
	}
	for _, s := range linearShift {
		if s != 0 {
			t.Fatalf("a linear gong glides: %g", s)
		}
	}
	// and it can be heard: the early partial sits above the late one
	f := hz(57)
	cfg := Gong()
	cfg.Noise = 0
	seeded()
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 57.0, "velocity": 1.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 4)
	early, _ := peakNear(window(x, 0.03, 0.3), f*1.0, 200)
	late, _ := peakNear(window(x, 3, 0.5), f*1.0, 200)
	if cents(early, late) < 10 {
		t.Errorf("the measured pitch falls %.1f cents", cents(early, late))
	}
}

func TestNonlinearityIsOffByDefaultAndDeterministic(t *testing.T) {
	for name, f := range Presets {
		if prIsMore(name) {
			continue // the extra gongs and cymbals bloom too (presets_more.go)
		}
		on := name == "gong" || name == "cymbal"
		if (f().Nonlinearity > 0) != on {
			t.Errorf("%s: nonlinearity %g", name, f().Nonlinearity)
		}
	}
	seeded()
	_, a := strike(Cymbal(), 64, 0.5)
	seeded()
	_, b := strike(Cymbal(), 64, 0.5)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("the cymbal differs at %d", i)
		}
	}
	// a linear instrument draws nothing from the voice's noise stream
	v := New(Marimba())
	before := v.rng
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, 0.1)
	if v.nlStrength != 0 || v.rng == 0 || before == 0 {
		t.Errorf("marimba has a nonlinear strength %g", v.nlStrength)
	}
}

func TestChurchBellStructure(t *testing.T) {
	md := modelOf(InstChurchBell)
	want := []float64{0.5, 1, 1.2, 1.5, 2}
	for i, w := range want {
		if math.Abs(md.parts[i].ratio/w-1) > 0.002 {
			t.Errorf("partial %d at %g, want %g", i, md.parts[i].ratio, w)
		}
	}
	cfg := ChurchBell()
	cfg.Noise, cfg.Decay = 0, 12
	_, x := strike(cfg, 60, 5)
	f := hz(60)
	early := window(x, 0.05, 0.5)
	late := window(x, 4, 0.5)
	for i, w := range want {
		got, a := peakNear(early, w*f, 40)
		if a == 0 || math.Abs(cents(got, w*f)) > 25 {
			t.Errorf("partial %g sounds %.0f cents off", w, cents(got, w*f))
		}
		_ = i
	}
	// the hum outlasts the nominal; the pair of the tierce beats
	humDrop := db(amplitude(late, 0.5*f) / amplitude(early, 0.5*f))
	nomDrop := db(amplitude(late, 2*f) / amplitude(early, 2*f))
	if humDrop < nomDrop+3 {
		t.Errorf("the hum fell %.1f dB, the nominal %.1f dB", humDrop, nomDrop)
	}
	// the tierce is a pair: two lines a fraction of a hertz or so apart, so its envelope beats
	s := md.split * md.parts[2].beatScale
	if s <= 0 || s*1.2*f < 0.3 {
		t.Errorf("tierce split %g", s)
	}
}

func TestWoodBlockTempleBlockAndBottle(t *testing.T) {
	for _, name := range []string{"woodBlock", "templeBlock", "bottle"} {
		cfg := Presets[name]()
		cfg.Noise = 0
		_, x := strike(cfg, 72, 1)
		f := hz(72)
		y := window(x, 0.003, 0.05)
		cav := amplitude(y, f)
		for _, r := range []float64{0.5, 1.3} {
			if amplitude(y, r*f) > cav {
				t.Errorf("%s: the cavity at %.1f Hz isn't the strongest", name, f)
			}
		}
		if got, _ := peakNear(y, f, 60); math.Abs(cents(got, f)) > 20 {
			t.Errorf("%s: pitch %.0f cents off", name, cents(got, f))
		}
		if r := rms(window(x, 0.7, 0.2)); r > rms(window(x, 0.003, 0.05))*0.1 {
			t.Errorf("%s still rings at 0.7 s: %g", name, r)
		}
	}
	// a temple block's cavity rings longer than a wood block's
	ringTime := func(name string) float64 {
		cfg := Presets[name]()
		_, x := strike(cfg, 72, 1.5)
		return rms(window(x, 0.2, 0.1)) / rms(window(x, 0.005, 0.05))
	}
	if ringTime("templeBlock") < ringTime("woodBlock") {
		t.Errorf("temple block %g, wood block %g", ringTime("templeBlock"), ringTime("woodBlock"))
	}
}

func TestNoNaNAcrossTheMatrixForNonlinearInstruments(t *testing.T) {
	for _, ins := range []Instrument{InstGong, InstCymbal, InstMarimbaBar, InstVibraphoneBar} {
		for src := modmatrix.Source(1); src < modmatrix.NumSources; src++ {
			for i, d := range Dests {
				c := Config{Instrument: ins, Decay: 2, Hardness: 1, Position: 0.4, Spread: 1, Nonlinearity: 1, Undercut: 0.3, Level: 1}
				c.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
				c.Mods[1] = modmatrix.Slot{Src: modmatrix.Source(1 + (int(src)+i)%int(modmatrix.NumSources-1)), Dest: d.Key, Amount: -1}
				c.Ext = [4]float64{1, 0, 1, 0}
				_, z := strike(c, 60, 0.05)
				finite(t, Instruments[ins]+" "+d.Key, z)
			}
		}
	}
}
