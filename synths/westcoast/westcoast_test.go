package westcoast

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

func run(v *Voice, seconds float64) []float64 {
	var x []float64
	for range int(math.Ceil(seconds * sr() / float64(v.Config().BufferSize))) {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func play(cfg Config, msg psalm.Message, seconds float64) (*Voice, []float64) {
	v := New(cfg)
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	return v, run(v, seconds)
}

// amplitude of a frequency in x (Goertzel, Hann window)
func amplitude(x []float64, freq float64) float64 {
	w := 2 * math.Pi * freq / sr()
	var s1, s2, norm float64
	for i, v := range x {
		h := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(len(x)-1))
		s := v*h + 2*math.Cos(w)*s1 - s2
		s2, s1 = s1, s
		norm += h
	}
	return 2 * math.Hypot(s1-s2*math.Cos(w), s2*math.Sin(w)) / norm
}

// open is a held note through a gate wide open: the oscillator and the folder as they are
func open() Config {
	return Config{Sustain: true, Decay: 1, Gate: VCA, Level: 1}
}

func harmonics(cfg Config, f float64, n int) []float64 {
	_, x := play(cfg, psalm.Message{"frequency": f, "velocity": 1.0}, 0.5)
	y := x[int(0.2*sr()):]
	h := make([]float64, n+1)
	for k := 1; k <= n; k++ {
		h[k] = amplitude(y, float64(k)*f) / amplitude(y, f)
	}
	return h
}

func TestFolder(t *testing.T) {
	plain := harmonics(open(), 220, 5)
	for k := 2; k <= 5; k++ {
		if plain[k] > 0.005 {
			t.Errorf("unfolded, harmonic %d is %.4f", k, plain[k])
		}
	}
	cfg := open()
	cfg.Fold = 0.5
	folded := harmonics(cfg, 220, 5)
	if folded[3] < 0.1 || folded[2] > 0.01 {
		t.Errorf("folded: the odd harmonics %.3f, the even %.3f", folded[3], folded[2])
	}
	cfg.Symmetry = 0.5
	if h := harmonics(cfg, 220, 2); h[2] < 0.05 {
		t.Errorf("folded off center: the second harmonic %.3f", h[2])
	}
}

func TestFM(t *testing.T) {
	cfg := open()
	cfg.ModRatio, cfg.FM = 1.5, 1
	_, x := play(cfg, psalm.Message{"frequency": 200.0, "velocity": 1.0}, 0.5)
	y := x[int(0.2*sr()):]
	// sidebands at 200 ± 300 Hz: 500 (and 100)
	if a, c := amplitude(y, 500), amplitude(y, 200); a < 0.2*c {
		t.Errorf("FM: the sideband %.3f against the carrier %.3f", a, c)
	}
	if a := amplitude(y, 350); a > 0.01*amplitude(y, 200) {
		t.Errorf("FM: energy between the sidebands %.4f", a)
	}
}

func TestLowPassGate(t *testing.T) {
	// struck, the vactrol lights at once and dies slowly; it gets darker as it dies
	cfg := Pluck()
	cfg.EnvToFold, cfg.Fold = 0, 0.5
	_, x := play(cfg, psalm.Message{"frequency": 220.0, "velocity": 1.0}, 0.6)
	peakAt, peak := 0, 0.0
	for i, s := range x {
		if math.Abs(s) > peak {
			peakAt, peak = i, math.Abs(s)
		}
	}
	if float64(peakAt)/sr() > 0.03 {
		t.Errorf("the gate peaks after %.0f ms", float64(peakAt)/sr()*1000)
	}
	early, late := x[int(0.02*sr()):int(0.07*sr())], x[int(0.25*sr()):int(0.3*sr())]
	bright := func(y []float64) float64 { return amplitude(y, 5*220) / amplitude(y, 220) }
	if bright(late) > bright(early)*0.5 {
		t.Errorf("the gate closing: the 5th harmonic %.3f early, %.3f late", bright(early), bright(late))
	}
	if amplitude(late, 220) > amplitude(early, 220)*0.7 {
		t.Errorf("the gate doesn't close")
	}
}

func TestBongo(t *testing.T) {
	v, x := play(Bongo(), psalm.Message{"pitch": 57.0}, 1.5)
	if v.IsActive() {
		t.Errorf("the bongo still sounds after 1.5 s")
	}
	p := 0.0
	for _, s := range x[:int(0.05*sr())] {
		p = math.Max(p, math.Abs(s))
	}
	if p < 0.05 {
		t.Errorf("the bongo peaks at %g", p)
	}
	if a, b := amplitude(x[:int(0.1*sr())], 220), amplitude(x[:int(0.1*sr())], 330); a < 5*b {
		t.Errorf("the bongo rings at %g at its pitch, %g beside it", a, b)
	}
}

func TestNotes(t *testing.T) {
	v := New(Bass())
	v.NoteOn(1, psalm.Message{"pitch": 40.0}, psalm.CtrlTick{SampleOffset: 15}, v.Config())
	v.Synthesize()
	if out := v.Output(0).Channels[0]; out[14] != 0 {
		t.Errorf("it plays before its sample")
	}
	v.Clear()
	v.Note(0.2, 1, psalm.Message{"pitch": 40.0}, psalm.CtrlTick{}, v.Config())
	run(v, 3)
	if v.IsActive() {
		t.Errorf("a held note of 0.2 s plays after 3 s")
	}
	for name, f := range Presets {
		_, x := play(f(), psalm.Message{"pitch": 57.0}, 1)
		p := 0.0
		for _, s := range x {
			p = math.Max(p, math.Abs(s))
		}
		if p < 0.01 || p > 4 || math.IsNaN(p) {
			t.Errorf("%s peaks at %g", name, p)
		}
	}
}

func energy(x []float64) float64 {
	var e float64
	for _, s := range x {
		e += s * s
	}
	return e
}

// a value cabled in on an Ext input moves a setting a slot joins it to
func TestMatrixExtMovesLevel(t *testing.T) {
	cfg := open()
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -1}
	loud, quiet := New(cfg), New(cfg)
	quiet.Receive(1.0, InExt1)
	msg := psalm.Message{"frequency": 220.0, "velocity": 1.0}
	loud.NoteOn(1, msg, psalm.CtrlTick{}, loud.Config())
	quiet.NoteOn(1, msg, psalm.CtrlTick{}, quiet.Config())
	el, eq := energy(run(loud, 0.3)), energy(run(quiet, 0.3))
	if el < 1 || eq > el*0.01 {
		t.Errorf("energy %g with the value at 0, %g at 1: want it silent at 1", el, eq)
	}
	if quiet.cfg.Ext[0] != 1 {
		t.Errorf("the value isn't kept in the config: %v", quiet.cfg.Ext)
	}
}

// the function generator through the matrix folds the wave: the 3rd harmonic comes
func TestMatrixEnvToFold(t *testing.T) {
	cfg := open()
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.FilterEnv, Dest: "fold", Amount: 0.6}
	if h := harmonics(cfg, 220, 3); h[3] < 0.1 {
		t.Errorf("the third harmonic %.3f", h[3])
	}
}

// an LFO through the matrix moves the pitch
func TestMatrixLFOToPitch(t *testing.T) {
	_, plain := play(open(), psalm.Message{"frequency": 220.0}, 0.4)
	cfg := open()
	cfg.LFORate = 5
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "pitch", Amount: 0.5}
	_, moved := play(cfg, psalm.Message{"frequency": 220.0}, 0.4)
	if a, b := amplitude(plain[4000:], 220), amplitude(moved[4000:], 220); b > a*0.8 {
		t.Errorf("amplitude at 220 Hz %g plain, %g with vibrato", a, b)
	}
}

// every source into every destination plays no NaN, and the slots arrive by message
func TestMatrixEverything(t *testing.T) {
	for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
		for _, d := range Dests {
			cfg := Bell()
			cfg.LFORate, cfg.LFO2Rate = 3, 4
			cfg.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
			cfg.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
			cfg.Osc2Mix = 0.5
			v := New(cfg)
			v.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, v.Config())
			for _, s := range run(v, 0.3) {
				if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 20 {
					t.Fatalf("%s to %s: sample %v", modmatrix.SourceNames[src], d.Key, s)
				}
			}
		}
	}
	v := New(DefaultConfig())
	v.ReceiveMessage(psalm.Message{"mod3Src": int(modmatrix.Wheel), "mod3Dest": 1, "mod3Amount": 0.5})
	if s := v.cfg.Mods[2]; s.Src != modmatrix.Wheel || s.Dest != "fold" || s.Amount != 0.5 {
		t.Errorf("slot %+v", s)
	}
}

// the second oscillator, detuned, sounds at its own pitch; at mix 0 nothing changes
func TestOsc2(t *testing.T) {
	msg := psalm.Message{"frequency": 200.0, "velocity": 1.0}
	_, one := play(open(), msg, 0.5)
	cfg := open()
	cfg.Osc2Detune = 7
	_, still := play(cfg, msg, 0.5)
	for i := range one {
		if one[i] != still[i] {
			t.Fatalf("mix 0 changes the sound at %d", i)
		}
	}
	cfg.Osc2Mix = 1
	_, two := play(cfg, msg, 0.5)
	y := two[int(0.2*sr()):]
	f2 := 200 * math.Exp2(7.0/12)
	if a, b := amplitude(y, f2), amplitude(y, 200); a < 0.5*b || b == 0 {
		t.Errorf("osc 2 at %.1f Hz: %g against %g at 200", f2, a, b)
	}
}

// the cycling function generator keeps a held note going and moves the fold again and again
func TestCycle(t *testing.T) {
	cfg := Pluck()
	cfg.Decay = 0.05
	_, once := play(cfg, psalm.Message{"pitch": 57.0}, 1)
	cfg.Cycle, cfg.CycleRate = true, 4
	v, cyc := play(cfg, psalm.Message{"pitch": 57.0}, 1)
	if !v.IsActive() {
		t.Errorf("a cycling note stopped")
	}
	late := func(x []float64) float64 { return energy(x[int(0.7*sr()):]) }
	if late(once) > 0.01*late(cyc) || late(cyc) < 0.1 {
		t.Errorf("energy late in the note: %g once, %g cycling", late(once), late(cyc))
	}
	v.NoteOff()
	run(v, 3)
	if v.IsActive() {
		t.Errorf("a cycling note doesn't stop after its release")
	}
}

// strike lights the lamp at once: the gate is open at the first samples
func TestStrike(t *testing.T) {
	cfg := Bell()
	cfg.Attack = 0.2
	_, soft := play(cfg, psalm.Message{"pitch": 57.0}, 0.05)
	cfg.Strike = 1
	_, hit := play(cfg, psalm.Message{"pitch": 57.0}, 0.05)
	if energy(hit) < 5*energy(soft) {
		t.Errorf("energy %g struck, %g without", energy(hit), energy(soft))
	}
}
