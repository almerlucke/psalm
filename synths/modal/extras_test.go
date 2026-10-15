package modal

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

func rms(x []float64) float64 {
	var e float64
	for _, v := range x {
		e += v * v
	}
	return math.Sqrt(e / float64(len(x)))
}

func rough(x []float64) float64 {
	var e float64
	for i := 1; i < len(x); i++ {
		d := x[i] - x[i-1]
		e += d * d
	}
	return e
}

func finite(t *testing.T, name string, x []float64) {
	t.Helper()
	for _, v := range x {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("%s: not finite", name)
		}
	}
}

func TestFilterDarkens(t *testing.T) {
	open := Glockenspiel()
	closed := open
	closed.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 500, KeyTrack: 0}
	_, a := strike(open, 72, 0.3)
	_, b := strike(closed, 72, 0.3)
	if o, c := rough(a), rough(b); o == 0 || c > o*0.3 {
		t.Errorf("roughness %g open, %g with a low pass at 500 Hz", o, c)
	}
	v := New(open)
	v.ReceiveMessage(psalm.Message{"filterType": int(voicefilter.Ladder24), "cutoff": 700.0})
	if v.cfg.Filter.Type != voicefilter.Ladder24 || v.cfg.Filter.Cutoff != 700 {
		t.Errorf("filter %+v", v.cfg.Filter)
	}
}

func TestMatrixMovesDestinations(t *testing.T) {
	base := Marimba()
	base.Noise = 0
	_, plain := strike(base, 60, 0.3)

	// level from ext 1, cabled in
	cfg := base
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -0.25}
	v := New(cfg)
	v.Receive(1.0, InExt1)
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.3)
	if r := rms(x) / rms(plain); math.Abs(r-(base.Level-0.25)/base.Level) > 0.08 {
		t.Errorf("level moved to %g of the plain", r)
	}
	if v.cfg.Ext[0] != 1 {
		t.Errorf("ext not kept: %v", v.cfg.Ext)
	}

	// decay at the strike: velocity... use ext 2 to lengthen
	tail := func(c Config, ext float64) float64 {
		v := New(c)
		v.Receive(ext, InExt1+1)
		v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
		x := run(v, 1.5)
		return rms(x[len(x)-int(0.2*sr()):])
	}
	cfg = base
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Ext2, Dest: "decay", Amount: 0.5}
	if short, long := tail(cfg, 0), tail(cfg, 1); long < short*1.5 {
		t.Errorf("the tail %g without, %g with decay moved up", short, long)
	}

	// the LFO and the amp env as sources, tremolo depth and cutoff as destinations
	cfg = base
	cfg.Decay = 100
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "tremoloDepth", Amount: 1}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.AmpEnv, Dest: "cutoff", Amount: -0.5}
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 3000}
	_, y := strike(cfg, 60, 0.5)
	finite(t, "lfo", y)
	if rms(y) == 0 {
		t.Errorf("silent")
	}

	// everything from every source stays finite
	for _, d := range Dests {
		for src := modmatrix.Source(1); src < modmatrix.NumSources; src++ {
			c := Gamelan()
			c.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 1000}
			c.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
			c.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
			c.Ext = [4]float64{1, 1, 1, 1}
			_, z := strike(c, 64, 0.2)
			finite(t, d.Key, z)
		}
	}

	v = New(base)
	v.ReceiveMessage(psalm.Message{"mod1Src": int(modmatrix.Wheel), "mod1Dest": 9, "mod1Amount": 0.5})
	if s := v.cfg.Mods[0]; s.Src != modmatrix.Wheel || s.Dest != "cutoff" || s.Amount != 0.5 {
		t.Errorf("slot %+v", s)
	}
}

func TestExcitation(t *testing.T) {
	cfg := Vibraphone()
	cfg.Noise, cfg.Tremolo = 0, 0
	one := func(e Excitation) (*Voice, []float64) {
		c := cfg
		c.Excitation = e
		return strike(c, 69, 1)
	}
	_, strikeX := one(Strike)
	_, bowX := one(Bow)
	_, burstX := one(Burst)
	finite(t, "bow", bowX)
	finite(t, "burst", burstX)
	for name, x := range map[string][]float64{"bow": bowX, "burst": burstX} {
		if rms(x) == 0 {
			t.Errorf("%s is silent", name)
		}
		same := true
		for i := range strikeX {
			if strikeX[i] != x[i] {
				same = false
				break
			}
		}
		if same {
			t.Errorf("%s sounds like a strike", name)
		}
	}
	t.Logf("rms strike %g bow %g burst %g", rms(strikeX), rms(bowX), rms(burstX))
	// a bow keeps the sound up where a strike has fallen
	late := func(x []float64) float64 { return rms(x[len(x)-int(0.2*sr()):]) }
	c := Config{Modes: []Mode{{1, 1, 1}}, Decay: 0.3, Hardness: 0.5, Level: 1}
	_, s := strike(c, 69, 1.2)
	c.Excitation = Bow
	v, b := strike(c, 69, 1.2)
	if late(b) < 20*late(s) || late(b) == 0 {
		t.Errorf("after 1 s a bow rings %g, a strike %g", late(b), late(s))
	}
	// and lifts when let go
	v.NoteOff()
	r := run(v, 1.5)
	if late(r) > late(b)*0.1 {
		t.Errorf("the bow rings on: %g after %g", late(r), late(b))
	}
	// a bowed note of a duration ends
	v = New(c)
	v.Note(0.2, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, 3)
	if v.IsActive() {
		t.Errorf("a bowed note of 0.2 s still plays after 3 s")
	}
}

func TestInharmonicity(t *testing.T) {
	cfg := Config{Modes: []Mode{{1, 1, 1}, {4, 1, 1}}, Decay: 2, Hardness: 1, Level: 1}
	f := hz(57)
	peak := func(inh float64) float64 {
		c := cfg
		c.Inharmonicity = inh
		_, x := strike(c, 57, 0.5)
		y := window(x, 0.01, 0.3)
		best, bf := 0.0, 0.0
		for g := 3.5 * f; g < 5.5*f; g += f * 0.005 {
			if a := amplitude(y, g); a > best {
				best, bf = a, g
			}
		}
		return bf / f
	}
	if p := peak(0); math.Abs(p-4) > 0.05 {
		t.Errorf("upper mode at %g, want 4", p)
	}
	if p := peak(1); p < 4.3 || p > 4.8 {
		t.Errorf("stretched upper mode at %g, want about 4.56", p)
	}
	v := New(cfg)
	v.ReceiveMessage(psalm.Message{"inharmonicity": 0.5, "excitation": 2})
	if v.cfg.Inharmonicity != 0.5 || v.cfg.Excitation != Burst {
		t.Errorf("settings %+v", v.cfg)
	}
}

func TestMaterialPresets(t *testing.T) {
	for _, name := range []string{"wood", "metal", "glass"} {
		f, ok := Presets[name]
		if !ok {
			t.Fatalf("no preset %s", name)
		}
		_, x := strike(f(), 69, 0.5)
		finite(t, name, x)
		p := 0.0
		for _, s := range x {
			p = math.Max(p, math.Abs(s))
		}
		if p < 0.05 || p > 3 {
			t.Errorf("%s peaks at %g", name, p)
		}
	}
	if Presets["metal"]().Decay <= Presets["wood"]().Decay*3 {
		t.Errorf("metal should ring much longer than wood")
	}
}

func TestLFO2MovesLevel(t *testing.T) {
	cfg := Marimba()
	cfg.Noise = 0
	cfg.Decay = 5
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "level", Amount: 0.9}
	cfg.LFO2Rate = 4
	v, x := strike(cfg, 60, 1)
	finite(t, "lfo2", x)
	win := len(x) / 16
	lo, hi := math.Inf(1), 0.0
	for i := 0; i+win <= len(x); i += win {
		r := rms(x[i : i+win])
		lo, hi = math.Min(lo, r), math.Max(hi, r)
	}
	if hi == 0 || lo > hi*0.7 {
		t.Errorf("level swings %g to %g under a 4 Hz LFO 2", lo, hi)
	}
	v.ReceiveMessage(psalm.Message{"lfo2Rate": 2.0})
	if v.cfg.LFO2Rate != 2 {
		t.Errorf("lfo2Rate %g", v.cfg.LFO2Rate)
	}
	plain := New(Marimba())
	plain.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, plain.Config())
	run(plain, 0.1)
	if plain.lfo2Phase != 0 {
		t.Error("LFO 2 runs unused")
	}
}

func TestPitchModulationIsSmooth(t *testing.T) {
	cfg := Marimba()
	cfg.Noise = 0
	cfg.LFO2Rate = 20
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "pitch", Amount: 0.1}
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	for b := range 12 {
		v.Synthesize()
		if b < 2 {
			continue
		}
		got := 12 * math.Log2(v.bentTo)
		want := 1.2 * math.Sin(2*math.Pi*v.lfo2Phase)
		if math.Abs(got-want) > 0.3 {
			t.Fatalf("buffer %d: bent %g semitones, the LFO is at %g: the pitch lags a buffer", b, got, want)
		}
	}
}
