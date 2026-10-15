package organ

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

func play(cfg Config, seconds float64) []float64 {
	o := New(cfg)
	down(o, "a", 60)
	x := render(o, seconds)
	for _, v := range x {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			panic("NaN")
		}
	}
	return x
}

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

func TestFilterDarkens(t *testing.T) {
	open := DefaultConfig()
	open.Drawbars = Bars("888888888")
	closed := open
	closed.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 400, KeyTrack: 0}
	o, c := rough(play(open, 0.5)), rough(play(closed, 0.5))
	if o == 0 || c > o*0.5 {
		t.Errorf("roughness %g open, %g with a low pass at 400 Hz", o, c)
	}
	o2 := New(open)
	o2.ReceiveMessage(psalm.Message{"filterType": int(voicefilter.Ladder24), "cutoff": 500.0})
	if o2.cfg.Filter.Type != voicefilter.Ladder24 || o2.cfg.Filter.Cutoff != 500 {
		t.Errorf("filter %+v", o2.cfg.Filter)
	}
}

func TestLeakageAddsFaintWheels(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Drawbars = Bars("008000000")
	cfg.Click = 0
	clean := play(cfg, 0.5)
	cfg.Leakage = 1
	leaky := play(cfg, 0.5)
	diff := make([]float64, len(clean))
	for i := range diff {
		diff[i] = leaky[i] - clean[i]
	}
	if r := rms(diff); r == 0 || r > 0.1*rms(clean) {
		t.Errorf("leakage %g against a signal of %g", r, rms(clean))
	}
	// it is the neighbouring wheels, 40 dB down at most, the same on every key press
	cfg.Leakage = 0
	x := play(cfg, 1)[int(0.2*sr()):]
	cfg.Leakage = 1
	y := play(cfg, 1)[int(0.2*sr()):]
	w := wheelOf(60)
	main := amplitude(y, wheelHz(w))
	diff = make([]float64, len(x))
	for i := range diff {
		diff[i] = y[i] - x[i]
	}
	for _, n := range []int{w - 1, w + 1, w + 12, w + 19} {
		if d := db(amplitude(diff, wheelHz(n)) / main); d < -52 || d > -38 {
			t.Errorf("wheel %d leaks at %.1f dB, want -52 to -38", n, d)
		}
	}
}

func TestClickRandomVaries(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Drawbars = Bars("000000000")
	cfg.Click = 1
	cfg.ClickRand = 1
	o := New(cfg)
	seen := map[float64]bool{}
	for range 6 {
		o.click.env = 0
		o.apply(event{pitch: 60, amp: 1, down: true})
		seen[o.click.env] = true
		o.AllNotesOff()
		o.keys = nil
	}
	if len(seen) < 4 {
		t.Errorf("the click is the same on every key: %v", seen)
	}
	cfg.ClickRand = 0
	o = New(cfg)
	o.apply(event{pitch: 60, amp: 1, down: true})
	if o.click.env != 1 {
		t.Errorf("click %g without randomness", o.click.env)
	}
}

func TestMatrixMovesDestinations(t *testing.T) {
	base := DefaultConfig()
	base.Drawbars = Bars("888888888")
	level := func(mod modmatrix.Slot) float64 {
		cfg := base
		cfg.Mods[0] = mod
		return rms(play(cfg, 0.3))
	}
	plain := level(modmatrix.Slot{})
	quiet := level(modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -0.5})
	if math.Abs(quiet-plain) > 1e-9 {
		t.Errorf("level %g, %g with ext 1 at 0 (no change expected)", plain, quiet)
	}
	// ext 1 cabled in at 1 lowers the level by half
	cfg := base
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -0.5}
	o := New(cfg)
	o.Receive(1.0, InExt1)
	down(o, "a", 60)
	x := render(o, 0.3)
	if got := rms(x); math.Abs(got/plain-0.5) > 0.1 {
		t.Errorf("level %g against %g: expected half", got, plain)
	}
	if o.cfg.Ext[0] != 1 {
		t.Errorf("ext not kept: %v", o.cfg.Ext)
	}
	// every destination, from every source, stays finite
	for _, d := range Dests {
		for src := modmatrix.Source(1); src < modmatrix.NumSources; src++ {
			cfg := base
			cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 1000}
			cfg.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
			cfg.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
			cfg.Ext = [4]float64{1, 1, 1, 1}
			play(cfg, 0.1)
		}
	}
}

func TestMatrixFilterEnvAndMessages(t *testing.T) {
	o := New(DefaultConfig())
	o.ReceiveMessage(psalm.Message{"mod1Src": int(modmatrix.Wheel), "mod1Dest": 5, "mod1Amount": 0.5})
	if s := o.cfg.Mods[0]; s.Src != modmatrix.Wheel || s.Dest != "cutoff" || s.Amount != 0.5 {
		t.Errorf("slot %+v", s)
	}
	cfg := DefaultConfig()
	cfg.Drawbars = Bars("888888888")
	cfg.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 300, KeyTrack: 0}
	shut := rough(play(cfg, 0.4))
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "cutoff", Amount: 1}
	op := New(cfg)
	op.Receive(1.0, InExt1)
	down(op, "a", 60)
	if open := rough(render(op, 0.4)); open < shut*2 {
		t.Errorf("roughness %g shut, %g with the cutoff moved up", shut, open)
	}
}

func TestFilterReleaseAndNoNaN(t *testing.T) {
	cfg := Jazz()
	cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder12, Cutoff: 800, Resonance: 0.9, EnvAmount: 3}
	cfg.Leakage, cfg.ClickRand = 1, 1
	o := New(cfg)
	down(o, "a", 48)
	render(o, 0.2)
	up(o, "a")
	for _, v := range render(o, 0.5) {
		if math.IsNaN(v) {
			t.Fatal("NaN")
		}
	}
}

func TestLFO2MovesLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "level", Amount: 1}
	cfg.LFO2Rate = 2
	x := play(cfg, 1)
	win := len(x) / 8
	lo, hi := math.Inf(1), 0.0
	for i := 0; i+win <= len(x); i += win {
		r := rms(x[i : i+win])
		lo, hi = math.Min(lo, r), math.Max(hi, r)
	}
	if hi == 0 || lo > hi*0.6 {
		t.Errorf("level swings %g to %g under a 2 Hz LFO 2", lo, hi)
	}
	o := New(DefaultConfig())
	o.ReceiveMessage(psalm.Message{"lfo2Rate": 3.0})
	if o.cfg.LFO2Rate != 3 {
		t.Errorf("lfo2Rate %g", o.cfg.LFO2Rate)
	}
	// without a slot using it the LFO does not run
	plain := New(DefaultConfig())
	down(plain, "a", 60)
	render(plain, 0.1)
	if plain.lfo2Phase != 0 {
		t.Error("LFO 2 runs unused")
	}
}
