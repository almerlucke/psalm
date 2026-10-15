package fm

import (
	"math"
	"testing"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

// steady returns a config with sustained envelopes and only operator 1 and 2 sounding
func steady() Config {
	cfg := DefaultConfig()
	for i := range cfg.Ops {
		cfg.Ops[i].Env = *adsr.NewSetting(1.0, 1.0, 1.0, 10.0, 1.0, 50.0)
		cfg.Ops[i].Level = 0
	}
	cfg.Ops[0].Level = 1.0
	return cfg
}

// render holds a note and returns the output after the attack
func render(cfg Config, msg psalm.Message) []float64 {
	v := New(cfg)
	v.NoteOn(0.5, msg, psalm.CtrlTick{}, v.Config())
	var out []float64
	for b := range 20 {
		v.Synthesize()
		if b >= 4 {
			for _, s := range v.Output(0).Channels[0] {
				out = append(out, float64(s))
			}
		}
	}
	return out
}

// spectrum returns the energy around each frequency, relative to the total
func spectrum(x []float64, sr float64, freqs ...float64) []float64 {
	const n = 8192
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	for i := range n {
		re[i] = x[i] * w[i]
	}
	f.Forward(re, im)

	total := 0.0
	for k := 1; k < n/2; k++ {
		total += re[k]*re[k] + im[k]*im[k]
	}

	out := make([]float64, len(freqs))
	for i, freq := range freqs {
		c := int(math.Round(freq * n / sr))
		for k := c - 3; k <= c+3; k++ {
			out[i] += re[k]*re[k] + im[k]*im[k]
		}
		out[i] /= total
	}
	return out
}

const sr = 44100.0

func TestAlgorithmsAreValid(t *testing.T) {
	for i, alg := range Algorithms {
		if len(alg.Carriers) == 0 {
			t.Fatalf("algorithm %d has no carriers", i)
		}
		for _, m := range alg.Mods {
			if m[0] <= m[1] {
				t.Fatalf("algorithm %d: operator %d can't modulate operator %d", i, m[0]+1, m[1]+1)
			}
		}
	}
}

func TestPureCarrierAndRatio(t *testing.T) {
	out := render(steady(), psalm.Message{"frequency": 440.0})
	if e := spectrum(out, sr, 440)[0]; e < 0.99 {
		t.Fatalf("expected a pure 440 Hz sine, only %v of the energy at 440", e)
	}

	cfg := steady()
	cfg.Ops[0].Ratio = 2
	out = render(cfg, psalm.Message{"frequency": 440.0})
	if e := spectrum(out, sr, 880)[0]; e < 0.99 {
		t.Fatalf("expected ratio 2 to play 880 Hz, only %v of the energy there", e)
	}
}

// besselJ computes the Bessel function of the first kind J_n(x) from its power series
func besselJ(n int, x float64) float64 {
	sum := 0.0
	for k := range 30 {
		term := math.Pow(-1, float64(k)) * math.Pow(x/2, float64(2*k+n))
		term /= math.Gamma(float64(k+1)) * math.Gamma(float64(k+n+1))
		sum += term
	}
	return sum
}

func TestModulationMatchesFMTheory(t *testing.T) {
	// a 440 Hz carrier modulated at 220 Hz (ratio 0.5) with index beta = 2*pi*level has partials
	// at 440 +- k*220 with amplitudes J_k(beta): energy J_0^2 in the carrier, J_1^2 per first sideband
	const level = 0.1
	beta := 2 * math.Pi * level

	cfg := steady()
	cfg.Ops[1].Ratio = 0.5
	cfg.Ops[1].Level = level

	e := spectrum(render(cfg, psalm.Message{"frequency": 440.0}), sr, 220, 440, 660)

	j0, j1 := besselJ(0, beta), besselJ(1, beta)
	if math.Abs(e[1]-j0*j0) > 0.03 {
		t.Fatalf("carrier energy %v, theory %v", e[1], j0*j0)
	}
	for i, side := range []float64{e[0], e[2]} {
		if math.Abs(side-j1*j1) > 0.02 {
			t.Fatalf("sideband %d energy %v, theory %v", i, side, j1*j1)
		}
	}
}

// brightness is the fraction of energy above the fundamental's neighbourhood
func brightness(x []float64, fundamental float64) float64 {
	return 1.0 - spectrum(x, sr, fundamental)[0]
}

func TestFeedbackAddsHarmonics(t *testing.T) {
	// algorithm 7 (all carriers) with only operator 4 sounding, so feedback is heard directly
	cfg := steady()
	cfg.Algorithm = 7
	cfg.Ops[0].Level = 0
	cfg.Ops[3].Level = 1

	clean := brightness(render(cfg, psalm.Message{"frequency": 220.0}), 220)
	cfg.Feedback = 0.7
	fb := brightness(render(cfg, psalm.Message{"frequency": 220.0}), 220)

	if fb < clean+0.1 {
		t.Fatalf("expected feedback to add harmonics: %v vs %v", fb, clean)
	}
}

func TestVelocityAndModDepthControlBrightness(t *testing.T) {
	cfg := steady()
	cfg.Ops[1].Level = 0.5
	cfg.Ops[1].VelocitySens = 1.0

	soft := brightness(render(cfg, psalm.Message{"frequency": 220.0, "velocity": 0.2}), 220)
	hard := brightness(render(cfg, psalm.Message{"frequency": 220.0, "velocity": 1.0}), 220)
	if hard < soft*2 {
		t.Fatalf("expected harder notes to be brighter: %v vs %v", hard, soft)
	}

	v := New(cfg)
	v.Receive(0.1, 1) // mod depth through value input 1
	if v.cfg.ModDepth != 0.1 {
		t.Fatal("value input 1 should set the mod depth")
	}
	dim := cfg
	dim.ModDepth = 0.1
	if d := brightness(render(dim, psalm.Message{"frequency": 220.0}), 220); d > hard*0.5 {
		t.Fatalf("expected a low mod depth to be darker: %v vs %v", d, hard)
	}
}

func TestPresets(t *testing.T) {
	for name, preset := range Presets {
		t.Run(name, func(t *testing.T) {
			v := New(preset())
			v.Note(0.5, 0.5, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())

			peak := 0.0
			for blocks := 0; v.IsActive(); blocks++ {
				if blocks > 1000 {
					t.Fatal("voice never finished")
				}
				v.Synthesize()
				for _, s := range v.Output(0).Channels[0] {
					if math.IsNaN(float64(s)) {
						t.Fatal("NaN output")
					}
					peak = math.Max(peak, math.Abs(float64(s)))
				}
			}

			if peak < 0.05 || peak > 1.5 {
				t.Fatalf("unexpected peak %v", peak)
			}
			t.Logf("peak %.3f", peak)
		})
	}
}

// boundaryJumpRatio plays a sustained pad note while setting a value input to a new value every
// buffer (like an LFO does), and compares the largest sudden jump (third difference) at buffer
// boundaries with the largest one elsewhere. Stepping parameters would click at the boundaries
func boundaryJumpRatio(input int, value func(t float64) float64) float64 {
	v := New(PadPreset())
	v.NoteOn(0.5, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())

	bufSize := v.Config().BufferSize
	var out []float64
	for b := range 200 {
		v.Receive(value(float64(b*bufSize)/v.Config().SampleRate), input)
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}

	boundary, inner := 0.0, 0.0
	for i := 100 * bufSize; i+3 < len(out); i++ {
		d := math.Abs(out[i+3] - 3*out[i+2] + 3*out[i+1] - out[i])
		if (i+2)%bufSize == 0 || (i+1)%bufSize == 0 || i%bufSize == 0 {
			boundary = math.Max(boundary, d)
		} else {
			inner = math.Max(inner, d)
		}
	}
	return boundary / inner
}

func TestModulatedSettingsDontClick(t *testing.T) {
	modDepth := boundaryJumpRatio(1, func(t float64) float64 { return 1.05 + 0.95*math.Sin(2*math.Pi*0.3*t) })
	feedback := boundaryJumpRatio(2, func(t float64) float64 { return 0.5 + 0.5*math.Sin(2*math.Pi*0.3*t) })
	t.Logf("boundary/inner jump ratio: mod depth %.2f, feedback %.2f", modDepth, feedback)

	if modDepth > 1.1 || feedback > 1.1 {
		t.Fatalf("parameter changes click at buffer boundaries: mod depth %.2f, feedback %.2f", modDepth, feedback)
	}
}

// roughness is the energy of what changes fast: the difference of neighbouring samples
func roughness(cfg Config) float64 {
	v := New(cfg)
	v.NoteOn(0.8, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	var e, prev float64
	for range 30 {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			d := float64(s) - prev
			e += d * d
			prev = float64(s)
		}
	}
	return e
}

func TestFilterDarkens(t *testing.T) {
	open := DefaultConfig()
	open.Ops[1].Level = 1.5
	closed := open
	closed.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 600, KeyTrack: 0}
	o, c := roughness(open), roughness(closed)
	if o == 0 || c > o*0.5 {
		t.Errorf("roughness %g without a filter, %g with a low pass at 600 Hz", o, c)
	}
	v := New(open)
	v.ReceiveMessage(psalm.Message{"filterType": int(voicefilter.Ladder24), "cutoff": 500.0})
	if v.cfg.Filter.Type != voicefilter.Ladder24 || v.cfg.Filter.Cutoff != 500 {
		t.Errorf("filter settings %+v", v.cfg.Filter)
	}
}

func fmEnergy(v *Voice, blocks int) float64 {
	v.NoteOn(0.8, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	var e float64
	for range blocks {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			e += float64(s) * float64(s)
		}
	}
	return e
}

// a value cabled in moves any setting a slot joins it to
func TestMatrixExtMovesLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Ext1, Dest: "level", Amount: -1}
	loud, quiet := New(cfg), New(cfg)
	quiet.Receive(1.0, InExt1)
	if el, eq := fmEnergy(loud, 40), fmEnergy(quiet, 40); el < 1 || eq > el*0.02 || quiet.cfg.Ext[0] != 1 {
		t.Errorf("energy %g with the value at 0, %g at 1", el, eq)
	}
}

// an LFO through the matrix moves the modulation index, so the sound changes
func TestMatrixLFOToModDepth(t *testing.T) {
	plain := fmEnergy(New(DefaultConfig()), 40)
	cfg := DefaultConfig()
	cfg.LFO1Rate = 6
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO1, Dest: "op2Level", Amount: 1}
	if e := fmEnergy(New(cfg), 40); e == plain || math.IsNaN(e) {
		t.Errorf("energy %g, as without (%g)", e, plain)
	}
}

func TestMatrixEverything(t *testing.T) {
	for src := modmatrix.FilterEnv; src < modmatrix.NumSources; src++ {
		for _, d := range Dests {
			cfg := DefaultConfig()
			cfg.LFO1Rate, cfg.LFO2Rate = 3, 4
			cfg.Filter = voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 2000}
			cfg.Mods[0] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: 1}
			cfg.Mods[1] = modmatrix.Slot{Src: src, Dest: d.Key, Amount: -1}
			v := New(cfg)
			v.NoteOn(1, psalm.Message{"pitch": 36.0}, psalm.CtrlTick{}, v.Config())
			for range 20 {
				v.Synthesize()
				for _, s := range v.Output(0).Channels[0] {
					if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) || math.Abs(float64(s)) > 20 {
						t.Fatalf("%s to %s: sample %v", modmatrix.SourceNames[src], d.Key, s)
					}
				}
			}
		}
	}
}
