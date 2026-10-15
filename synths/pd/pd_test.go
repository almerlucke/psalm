package pd

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
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

// steady is one line holding a waveform at a fixed distortion
func steady(w Waveform, dcw float64) Config {
	hold := env(1, 10, 1, 10)
	return Config{Line1: Line{Waveform: w, Level: 1, DCW: dcw, DCWEnv: hold, DCAEnv: hold}, Level: 1}
}

func harmonics(cfg Config, f float64, n int) []float64 {
	_, x := play(cfg, psalm.Message{"frequency": f, "velocity": 0.5}, 0.4)
	y := x[int(0.1*sr()):]
	h := make([]float64, n+1)
	for k := 1; k <= n; k++ {
		h[k] = amplitude(y, float64(k)*f) / amplitude(y, f)
	}
	return h
}

func TestUndistortedIsASine(t *testing.T) {
	for _, w := range []Waveform{Saw, Square, Pulse, SawPulse} {
		h := harmonics(steady(w, 0), 220, 5)
		for k := 2; k <= 5; k++ {
			if h[k] > 0.01 {
				t.Errorf("waveform %d undistorted has harmonic %d at %.3f", w, k, h[k])
			}
		}
	}
}

func TestShapes(t *testing.T) {
	// a distorted saw has all harmonics, a square mostly the odd ones
	saw := harmonics(steady(Saw, 0.95), 110, 6)
	if saw[2] < 0.2 || saw[3] < 0.1 {
		t.Errorf("the saw's harmonics: %.3f", saw[1:])
	}
	sq := harmonics(steady(Square, 0.95), 110, 6)
	if sq[3] < 0.15 || sq[2] > sq[3]*0.2 || sq[4] > sq[3]*0.2 {
		t.Errorf("the square's harmonics: %.3f", sq[1:])
	}
	// a resonance waveform peaks at the distortion's pitch: 1 + 15 d times the note
	res := harmonics(steady(Reso1, 0.6), 110, 12)
	best := 2
	for k := 2; k <= 12; k++ {
		if res[k] > res[best] {
			best = k
		}
	}
	if best < 9 || best > 11 {
		t.Errorf("the resonance peaks at harmonic %d, want 10: %.2f", best, res[1:])
	}
}

func TestNoAliasing(t *testing.T) {
	// at 3 kHz, fully distorted: what lies between the harmonics is far below them
	f := 3000.0
	_, x := play(steady(Saw, 1), psalm.Message{"frequency": f}, 0.4)
	y := x[int(0.1*sr()):]
	fund := amplitude(y, f)
	for _, between := range []float64{1.5, 2.5, 3.5, 4.5, 5.5} {
		if a := amplitude(y, between*f); a > fund*0.01 {
			t.Errorf("at %.0f Hz (between harmonics): %.1f dB", between*f, 20*math.Log10(a/fund))
		}
	}
}

func TestDCWEnvelope(t *testing.T) {
	// the bass's distortion falls after the attack: its second harmonic too
	_, x := play(Bass(), psalm.Message{"pitch": 45.0}, 1)
	f := 110.0
	early := x[int(0.01*sr()):int(0.06*sr())]
	late := x[int(0.6*sr()):int(0.65*sr())]
	ratio := func(y []float64) float64 { return amplitude(y, 4*f) / amplitude(y, f) }
	if ratio(early) < 2*ratio(late) {
		t.Errorf("the 4th harmonic: %.3f at the start, %.3f later", ratio(early), ratio(late))
	}
	// key follow: the same distortion is gentler high up
	cfg := steady(Saw, 0.8)
	cfg.Line1.KeyFollow = 1
	low, high := harmonics(cfg, 131, 2), harmonics(cfg, 1046, 2)
	if high[2] > low[2]*0.8 {
		t.Errorf("key follow: the 2nd harmonic %.3f low, %.3f high", low[2], high[2])
	}
}

func TestRing(t *testing.T) {
	hold := env(1, 10, 1, 10)
	cfg := Config{
		Line1: Line{Waveform: Saw, Level: 1, DCWEnv: hold, DCAEnv: hold},
		Line2: Line{Waveform: Saw, Level: 1, Detune: 702 + 1200, DCWEnv: hold, DCAEnv: hold},
		Ring:  true, Level: 1,
	}
	_, x := play(cfg, psalm.Message{"frequency": 200.0}, 0.4)
	y := x[int(0.1*sr()):]
	// two sines, 200 and 600 Hz (an octave and a fifth): ring modulated, 400 and 800, and 200 from line 1
	if a, b := amplitude(y, 400), amplitude(y, 600); a < 5*b {
		t.Errorf("ring modulation: the difference %.3f, line 2 itself %.3f", a, b)
	}
}

func TestNotes(t *testing.T) {
	v := New(Brass())
	v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{SampleOffset: 25}, v.Config())
	v.Synthesize()
	if out := v.Output(0).Channels[0]; out[24] != 0 {
		t.Errorf("it plays before its sample")
	}
	v.Clear()
	v.Note(0.2, 1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	run(v, 1.5)
	if v.IsActive() {
		t.Errorf("a note of 0.2 s plays after 1.5 s")
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

func energyOf(x []float64) float64 {
	var e float64
	for _, s := range x {
		e += s * s
	}
	return e
}

// the filter after the lines darkens the sound
func TestFilterDarkens(t *testing.T) {
	plain := steady(Saw, 0.95)
	filt := plain
	filt.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 400, EnvAmount: 0, Env: env(1, 10, 1, 10)}
	msg := psalm.Message{"frequency": 110.0}
	_, a := play(plain, msg, 0.4)
	_, b := play(filt, msg, 0.4)
	a, b = a[int(0.1*sr()):], b[int(0.1*sr()):]
	hi := func(x []float64) float64 { return amplitude(x, 1100) / amplitude(x, 110) }
	if hi(b) > 0.3*hi(a) {
		t.Errorf("the 10th harmonic: %.3f plain, %.3f filtered", hi(a), hi(b))
	}
	// by message
	v := New(plain)
	v.ReceiveMessage(psalm.Message{"filterType": int(voicefilter.LowPass), "cutoff": 500.0})
	if v.cfg.Filter.Type != voicefilter.LowPass || v.cfg.Filter.Cutoff != 500 {
		t.Errorf("filter %+v", v.cfg.Filter)
	}
}

// a pitch envelope bends the note, a depth envelope distorts over the note
func TestDCOAndDepthEnvelopes(t *testing.T) {
	cfg := steady(Saw, 0)
	cfg.DCOAmount = 12
	cfg.DCOEnv = env(1, 100, 0, 100)
	_, x := play(cfg, psalm.Message{"frequency": 220.0}, 0.6)
	early := x[int(0.002*sr()):int(0.03*sr())]
	late := x[int(0.4*sr()):int(0.5*sr())]
	// early on it sits an octave up
	if amplitude(early, 440) < 2*amplitude(early, 220) {
		t.Errorf("early: 440 Hz %.3f, 220 Hz %.3f", amplitude(early, 440), amplitude(early, 220))
	}
	if amplitude(late, 220) < 2*amplitude(late, 440) {
		t.Errorf("late: 440 Hz %.3f, 220 Hz %.3f", amplitude(late, 440), amplitude(late, 220))
	}
	// the depth envelope adds harmonics on a sine
	cfg = steady(Saw, 0)
	cfg.DepthAmount = 0.9
	cfg.DepthEnv = env(1, 100, 0, 100)
	_, x = play(cfg, psalm.Message{"frequency": 110.0}, 0.6)
	early = x[int(0.005*sr()):int(0.04*sr())]
	late = x[int(0.4*sr()):int(0.5*sr())]
	r := func(y []float64) float64 { return amplitude(y, 330) / amplitude(y, 110) }
	if r(early) < 5*r(late) {
		t.Errorf("3rd harmonic: %.3f early, %.3f late", r(early), r(late))
	}
	v := New(Brass())
	v.ReceiveMessage(psalm.Message{"dcoAttack": 33.0, "depthRelease": 777.0, "dcoAmount": 5.0})
	if v.cfg.DCOEnv.AttackDuration != 33 || v.cfg.DepthEnv.ReleaseDuration != 777 || v.cfg.DCOAmount != 5 {
		t.Errorf("config %+v", v.cfg)
	}
}

// line 1 detunes against line 2
func TestDetune1(t *testing.T) {
	v := New(Brass())
	v.ReceiveMessage(psalm.Message{"detune1": -30.0, "detune2": 25.0})
	if v.cfg.Line1.Detune != -30 || v.cfg.Line2.Detune != 25 {
		t.Errorf("detunes %g %g", v.cfg.Line1.Detune, v.cfg.Line2.Detune)
	}
}

// a value cabled in moves a setting through the matrix
func TestMatrixMoves(t *testing.T) {
	cfg := Brass()
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -1}
	loud, quiet := New(cfg), New(cfg)
	quiet.Receive(1.0, InExt1)
	loud.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, loud.Config())
	quiet.NoteOn(1, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, quiet.Config())
	if el, eq := energyOf(run(loud, 0.3)), energyOf(run(quiet, 0.3)); el < 0.1 || eq > el*0.01 {
		t.Errorf("energy %g at 0, %g at 1", el, eq)
	}
	// the wheel to the distortion brightens a sine
	cfg = steady(Saw, 0)
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "dcw1", Amount: 1}
	v := New(cfg)
	v.Receive(1.0, InExt1)
	v.NoteOn(1, psalm.Message{"frequency": 110.0}, psalm.CtrlTick{}, v.Config())
	x := run(v, 0.4)[int(0.1*sr()):]
	if h := amplitude(x, 330) / amplitude(x, 110); h < 0.05 {
		t.Errorf("3rd harmonic %.3f: the matrix doesn't distort", h)
	}
	v.ReceiveMessage(psalm.Message{"mod3Src": int(modmatrix.Wheel), "mod3Dest": 3, "mod3Amount": 0.5})
	if s := v.cfg.Mods[2]; s.Src != modmatrix.Wheel || s.Dest != "pitch" || s.Amount != 0.5 {
		t.Errorf("slot %+v", s)
	}
}

// every source into every destination plays no NaN, with the filter and the envelopes on
func TestMatrixEverything(t *testing.T) {
	for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
		for _, d := range Dests {
			cfg := Brass()
			cfg.SweepRate, cfg.LFO2Rate = 3, 4
			cfg.DCOAmount, cfg.DepthAmount = 3, 0.3
			cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 2000, EnvAmount: 1}
			cfg.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
			cfg.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
			v := New(cfg)
			v.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, v.Config())
			for _, s := range run(v, 0.3) {
				if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 20 {
					t.Fatalf("%s to %s: sample %v", modmatrix.SourceNames[src], d.Key, s)
				}
			}
		}
	}
}
