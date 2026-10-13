package dx

import (
	"errors"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

const sr = 44100.0

func TestAlgorithmsAreValid(t *testing.T) {
	for i, alg := range Algorithms {
		used := [NumOperators]bool{}
		if len(alg.Carriers) == 0 {
			t.Fatalf("algorithm %d has no carriers", i+1)
		}
		for _, c := range alg.Carriers {
			used[c] = true
		}
		for _, m := range alg.Mods {
			if m[0] <= m[1] {
				t.Fatalf("algorithm %d: operator %d can't modulate operator %d", i+1, m[0]+1, m[1]+1)
			}
			used[m[0]] = true
		}
		for op, u := range used {
			if !u {
				t.Fatalf("algorithm %d: operator %d is not connected", i+1, op+1)
			}
		}
		if alg.FeedbackFrom < 0 || alg.FeedbackFrom >= NumOperators || alg.FeedbackTo < 0 || alg.FeedbackTo >= NumOperators {
			t.Fatalf("algorithm %d: invalid feedback", i+1)
		}
	}
}

// randomPatch creates a patch with every parameter at a random valid value
func randomPatch(r *rand.Rand, name string) Patch {
	n := func(hi int) int { return r.IntN(hi + 1) }
	p := Patch{
		Name:          name,
		PitchEGRates:  [4]int{n(99), n(99), n(99), n(99)},
		PitchEGLevels: [4]int{n(99), n(99), n(99), n(99)},
		Algorithm:     n(31),
		Feedback:      n(7),
		OscKeySync:    r.IntN(2) == 1,
		LFOSpeed:      n(99),
		LFODelay:      n(99),
		LFOPitchDepth: n(99),
		LFOAmpDepth:   n(99),
		LFOKeySync:    r.IntN(2) == 1,
		LFOWaveform:   n(5),
		PitchModSens:  n(7),
		Transpose:     n(48),
	}
	for i := range p.Ops {
		p.Ops[i] = Operator{
			EGRates:      [4]int{n(99), n(99), n(99), n(99)},
			EGLevels:     [4]int{n(99), n(99), n(99), n(99)},
			BreakPoint:   n(99),
			LeftDepth:    n(99),
			RightDepth:   n(99),
			LeftCurve:    n(3),
			RightCurve:   n(3),
			RateScaling:  n(7),
			AmpModSens:   n(3),
			VelocitySens: n(7),
			OutputLevel:  n(99),
			FixedFreq:    r.IntN(2) == 1,
			Coarse:       n(31),
			Fine:         n(99),
			Detune:       n(14),
		}
	}
	return p
}

func TestBankRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	var bank []Patch
	for i := range 32 {
		bank = append(bank, randomPatch(r, "VOICE "+string(rune('A'+i%26))))
	}

	data := EncodeBank(bank)
	if len(data) != 4104 {
		t.Fatalf("bulk dump is %d bytes, expected 4104", len(data))
	}

	parsed, err := ParseSysEx(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 32 {
		t.Fatalf("expected 32 voices, got %d", len(parsed))
	}
	for i := range bank {
		if parsed[i] != bank[i] {
			t.Fatalf("voice %d differs after a round trip:\n%+v\n%+v", i, bank[i], parsed[i])
		}
	}
}

func TestSingleVoiceRoundTrip(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	a, b := randomPatch(r, "FIRST"), randomPatch(r, "SECOND")

	data := append(EncodeSingle(a), EncodeSingle(b)...)
	if len(data) != 2*163 {
		t.Fatalf("single dumps are %d bytes, expected %d", len(data), 2*163)
	}

	parsed, err := ParseSysEx(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 || parsed[0] != a || parsed[1] != b {
		t.Fatalf("voices differ after a round trip")
	}
}

func TestSysExErrors(t *testing.T) {
	data := EncodeBank([]Patch{InitVoice()})
	data[100] ^= 0x01
	if _, err := ParseSysEx(data); !errors.Is(err, ErrChecksum) {
		t.Fatalf("expected a checksum error, got %v", err)
	}

	if _, err := ParseSysEx([]byte{0xF0, 0x41, 0x10, 0x42, 0x12, 0x00, 0xF7}); !errors.Is(err, ErrNotSysEx) {
		t.Fatalf("expected a not sysex error, got %v", err)
	}

	if _, err := ParseSysEx(EncodeBank(nil)[:2000]); !errors.Is(err, ErrTruncated) {
		t.Fatalf("expected a truncated error, got %v", err)
	}
}

// render holds a note and returns the output after the attack
func render(p Patch, msg psalm.Message) []float64 {
	v := New(p, psalm.Mono)
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

// peakFrequency returns the strongest frequency
func peakFrequency(x []float64) float64 {
	const n = 8192
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	for i := range n {
		re[i] = x[i] * w[i]
	}
	f.Forward(re, im)
	best, bestMag := 0, 0.0
	for k := 1; k < n/2; k++ {
		if mag := re[k]*re[k] + im[k]*im[k]; mag > bestMag {
			best, bestMag = k, mag
		}
	}
	return float64(best) * sr / n
}

func rms(x []float64) float64 {
	sum := 0.0
	for _, v := range x {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(x)))
}

const binHz = sr / 8192

func TestInitVoiceIsASine(t *testing.T) {
	out := render(InitVoice(), psalm.Message{"pitch": 69.0})
	if f := peakFrequency(out); math.Abs(f-440) > 2*binHz {
		t.Fatalf("expected 440 Hz, got %v", f)
	}
	// a pure sine at amplitude 0.5 has an rms of 0.5 / sqrt(2)
	if r := rms(out); math.Abs(r-0.5/math.Sqrt2) > 0.01 {
		t.Fatalf("expected a full level sine, rms %v", r)
	}
}

func TestFrequencies(t *testing.T) {
	ratio := InitVoice()
	ratio.Ops[0].Coarse = 2
	if f := peakFrequency(render(ratio, psalm.Message{"pitch": 57.0})); math.Abs(f-440) > 2*binHz {
		t.Fatalf("coarse 2 on 220 Hz: expected 440 Hz, got %v", f)
	}

	fixed := InitVoice()
	fixed.Ops[0].FixedFreq = true
	fixed.Ops[0].Coarse = 2 // 100 Hz
	fixed.Ops[0].Fine = 30  // * 10^0.3, about 199.5 Hz
	if f := peakFrequency(render(fixed, psalm.Message{"pitch": 80.0})); math.Abs(f-199.5) > 2*binHz {
		t.Fatalf("fixed frequency: expected about 199.5 Hz, got %v", f)
	}

	transposed := InitVoice()
	transposed.Transpose = 36 // an octave up
	if f := peakFrequency(render(transposed, psalm.Message{"pitch": 57.0})); math.Abs(f-440) > 2*binHz {
		t.Fatalf("transpose: expected 440 Hz, got %v", f)
	}
}

func TestVelocitySensitivity(t *testing.T) {
	p := InitVoice()
	p.Ops[0].VelocitySens = 7
	loud := rms(render(p, psalm.Message{"pitch": 60.0, "velocity": 1.0}))
	soft := rms(render(p, psalm.Message{"pitch": 60.0, "velocity": 0.3}))
	if soft > loud*0.3 {
		t.Fatalf("expected soft notes to be much quieter: %v vs %v", soft, loud)
	}
}

func TestEnvelopeSustainAndRelease(t *testing.T) {
	p := InitVoice()
	p.Ops[0].EGRates = [4]int{80, 60, 60, 50}
	p.Ops[0].EGLevels = [4]int{99, 90, 80, 0}

	v := New(p, psalm.Mono)
	v.NoteOn(1.0, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	for range 200 {
		v.Synthesize()
	}
	if !v.IsActive() {
		t.Fatal("a held note should sustain")
	}
	if l := v.envs[0].level; math.Abs(l-80) > 0.01 {
		t.Fatalf("expected the envelope to sustain at level 3 (80), got %v", l)
	}

	v.NoteOff()
	for blocks := 0; v.IsActive(); blocks++ {
		if blocks > 2000 {
			t.Fatal("voice never released")
		}
		v.Synthesize()
	}
}

func TestTimedNoteEnds(t *testing.T) {
	v := New(InitVoice(), psalm.Mono)
	v.Note(0.2, 0.5, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
	for blocks := 0; v.IsActive(); blocks++ {
		if blocks > 100 {
			t.Fatal("timed note never ended")
		}
		v.Synthesize()
	}
}

func TestAllAlgorithmsStable(t *testing.T) {
	for alg := range Algorithms {
		p := InitVoice()
		p.Algorithm = alg
		p.Feedback = 7
		for i := range p.Ops {
			p.Ops[i].OutputLevel = 99
			p.Ops[i].Coarse = i + 1
		}

		out := render(p, psalm.Message{"pitch": 48.0})
		for _, s := range out {
			if math.IsNaN(s) || math.Abs(s) > 1.0 {
				t.Fatalf("algorithm %d: invalid sample %v", alg+1, s)
			}
		}
		if rms(out) < 0.01 {
			t.Fatalf("algorithm %d is silent", alg+1)
		}
	}
}

func TestPresets(t *testing.T) {
	for name, preset := range Presets {
		t.Run(name, func(t *testing.T) {
			v := New(preset(), psalm.Mono)
			v.Note(0.5, 0.5, psalm.Message{"pitch": 57.0, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())

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
			if peak < 0.05 || peak > 1.0 {
				t.Fatalf("unexpected peak %v", peak)
			}
			t.Logf("peak %.3f", peak)
		})
	}
}

func TestNoStandardCarrierModulates(t *testing.T) {
	// the carriers of the 32 algorithms only sound: their modulation output is never read, so
	// algorithms of your own (whose carriers may modulate) leave them as they were
	for i, a := range Algorithms {
		if err := a.Check(); err != nil {
			t.Errorf("algorithm %d: %v", i+1, err)
		}
		for _, m := range a.Mods {
			for _, c := range a.Carriers {
				if m[0] == c {
					t.Errorf("algorithm %d: carrier %d modulates", i+1, c+1)
				}
			}
		}
	}
}

func TestSettingsByName(t *testing.T) {
	p := InitVoice()
	for _, c := range []struct {
		name string
		v    float64
		get  func() int
	}{
		{"algorithm", 5, func() int { return p.Algorithm + 1 }},
		{"feedback", 9, func() int { return p.Feedback }}, // kept in its range
		{"op3Coarse", 2, func() int { return p.Ops[2].Coarse }},
		{"op6Rate4", 12.4, func() int { return p.Ops[5].EGRates[3] }},
		{"op1Level2", 80, func() int { return p.Ops[0].EGLevels[1] }},
		{"op2Level", 70, func() int { return p.Ops[1].OutputLevel }},
		{"pitchLevel3", 60, func() int { return p.PitchEGLevels[2] }},
	} {
		if !p.Set(c.name, c.v) {
			t.Errorf("%s isn't a setting", c.name)
		}
		if got, want := c.get(), max(0, min(int(c.v+0.5), func() int {
			if c.name == "feedback" {
				return 7
			}
			return 99
		}())); got != want {
			t.Errorf("%s = %d, want %d", c.name, got, want)
		}
		if v, ok := p.Get(c.name); !ok || int(v) != c.get() {
			t.Errorf("get %s: %v", c.name, v)
		}
	}
	p.Set("op4Fixed", 1)
	if !p.Ops[3].FixedFreq {
		t.Errorf("op4Fixed")
	}
	if p.Set("pitch", 60) || p.Set("velocity", 1) {
		t.Errorf("a note's pitch or velocity taken as a setting")
	}
}

func TestRoutingOfYourOwn(t *testing.T) {
	bad := Algorithm{Carriers: []int{0}, Mods: [][2]int{{0, 1}}, FeedbackFrom: 0, FeedbackTo: 0}
	if bad.Check() == nil {
		t.Errorf("operator 1 modulating 2 should be refused")
	}
	// operator 2 modulates operator 1 and is heard itself: louder and brighter than 1 alone
	p := InitVoice()
	p.Ops[1].OutputLevel = 90
	alone := New(p, 1)
	own := New(p, 1)
	own.ReceiveMessage(psalm.Message{"routing": Algorithm{Carriers: []int{0, 1}, Mods: [][2]int{{1, 0}}, FeedbackFrom: 5, FeedbackTo: 5}})
	if own.alg.Carriers[1] != 1 || len(own.modTargets[0]) != 1 {
		t.Fatalf("the routing isn't played: %+v", own.alg)
	}
	for _, v := range []*Voice{alone, own} {
		v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
		v.Synthesize()
	}
	diff := 0.0
	for i, s := range own.Output(0).Channels[0] {
		diff += math.Abs(float64(s - alone.Output(0).Channels[0][i]))
	}
	if diff == 0 {
		t.Errorf("a routing of your own sounds like algorithm 1")
	}
	// a routing that can't play falls back to the algorithm
	own.ReceiveMessage(psalm.Message{"routing": bad})
	if own.patch.Routing != nil {
		t.Errorf("a broken routing was kept")
	}
	own.ReceiveMessage(psalm.Message{"op2Coarse": 3.0, "algorithm": 32.0})
	if own.patch.Ops[1].Coarse != 3 || own.alg.Carriers[5] != 5 {
		t.Errorf("settings by message: coarse %d, carriers %v", own.patch.Ops[1].Coarse, own.alg.Carriers)
	}
}
