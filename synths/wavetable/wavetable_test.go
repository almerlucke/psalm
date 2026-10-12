package wavetable

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

const sr = 44100.0

const tableDir = "../../assets/wavetables2048"

func steady() Config {
	cfg := DefaultConfig()
	cfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 1.0, 10.0, 1.0, 50.0)
	return cfg
}

// render holds a note and returns both channels after the first buffers
func render(cfg Config, msg psalm.Message, buffers int) ([]float64, []float64) {
	v := New(cfg)
	v.NoteOn(1.0, msg, psalm.CtrlTick{}, v.Config())
	var l, r []float64
	for b := range buffers {
		v.Synthesize()
		if b >= 2 {
			for i := range v.Output(0).Channels[0] {
				l = append(l, float64(v.Output(0).Channels[0][i]))
				r = append(r, float64(v.Output(0).Channels[1][i]))
			}
		}
	}
	return l, r
}

// spectrum returns the power spectrum averaged over overlapping windows
func spectrum(x []float64) []float64 {
	const n = 16384
	f := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	w := fft.HannWindow(n)
	power := make([]float64, n/2)
	for start := 0; start+n <= len(x); start += n / 2 {
		for i := range n {
			re[i] = x[start+i] * w[i]
			im[i] = 0
		}
		f.Forward(re, im)
		for k := range power {
			power[k] += re[k]*re[k] + im[k]*im[k]
		}
	}
	return power
}

func binFreq(k int) float64 { return float64(k) * sr / 16384 }

// harmonicEnergy returns the energy at the harmonics of f0 and between them
func harmonicEnergy(p []float64, f0 float64) (float64, float64) {
	harm, between := 0.0, 0.0
	for k := 1; k < len(p); k++ {
		f := binFreq(k)
		if d := math.Abs(f - f0*math.Round(f/f0)); d < 10 {
			harm += p[k]
		} else if d > 30 {
			between += p[k]
		}
	}
	return harm, between
}

func rms(x []float64) float64 {
	sum := 0.0
	for _, v := range x {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(x)))
}

func TestTablesLoad(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join(tableDir, "*.wav"))
	if len(files) == 0 {
		t.Skip("no wavetables in assets")
	}
	for _, file := range files {
		table, err := Load(file, 2048)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		info, _ := os.Stat(file)
		if table.NumFrames() < 1 || table.NumFrames() > int(info.Size())/2048/4 {
			t.Fatalf("%s: %d frames", file, table.NumFrames())
		}
		peak := 0.0
		for _, levels := range table.frames {
			for _, l := range levels {
				for _, x := range l.data {
					if math.IsNaN(float64(x)) {
						t.Fatalf("%s: NaN in table", file)
					}
					peak = math.Max(peak, math.Abs(float64(x)))
				}
			}
		}
		if math.Abs(peak-1) > 1e-3 {
			t.Fatalf("%s: peak %v, expected the table normalized to 1", file, peak)
		}
	}
}

func TestPitch(t *testing.T) {
	l, _ := render(steady(), psalm.Message{"frequency": 440.0}, 40)
	p := spectrum(l)
	best := 1
	for k := range p {
		if p[k] > p[best] {
			best = k
		}
	}
	if f := binFreq(best); math.Abs(f-440) > 3 {
		t.Fatalf("expected a 440 Hz sine, strongest partial at %v Hz", f)
	}
}

// A sawtooth played high must not alias: all energy stays at the harmonics
func TestBandLimited(t *testing.T) {
	for _, f0 := range []float64{220, 1000, 3100, 7300} {
		cfg := steady()
		cfg.Position = 2.0 / 3.0 // sawtooth
		l, _ := render(cfg, psalm.Message{"frequency": f0}, 60)
		harm, between := harmonicEnergy(spectrum(l), f0)
		if db := 10 * math.Log10(between/harm); db > -60 {
			t.Errorf("%v Hz: aliasing at %.1f dB", f0, db)
		}
	}
}

// A sawtooth gliding up across levels changes smoothly and stays alias free
func TestGlideAcrossLevels(t *testing.T) {
	cfg := steady()
	cfg.Position = 2.0 / 3.0
	cfg.Glide = 1500
	v := New(cfg)
	v.NoteOn(1.0, psalm.Message{"frequency": 300.0}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	v.ReceiveMessage(psalm.Message{"frequency": 6000.0})
	prev, maxJump := 0.0, 0.0
	for range 100 {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			maxJump = math.Max(maxJump, math.Abs(float64(s)-prev))
			prev = float64(s)
		}
	}
	// a band limited sawtooth of peak 0.6 jumps at most about 1.2 per cycle, but never more
	if maxJump > 1.5 {
		t.Fatalf("jump of %v while gliding", maxJump)
	}
}

func TestPositionMorphs(t *testing.T) {
	third := func(position float64) float64 {
		cfg := steady()
		cfg.Position = position
		l, _ := render(cfg, psalm.Message{"frequency": 220.0}, 40)
		p := spectrum(l)
		k1, k3 := int(math.Round(220*16384/sr)), int(math.Round(660*16384/sr))
		return p[k3] / p[k1]
	}
	if sine, square := third(0), third(1); sine > 1e-6 || square < 0.05 {
		t.Fatalf("expected the sine without and the square with a third harmonic: %v, %v", sine, square)
	}
}

// A sudden position change (value input 1) glides, so it doesn't step from one wave to the other
func TestPositionChangeIsSmooth(t *testing.T) {
	cfg := steady()
	cfg.Position = 0 // sine
	v := New(cfg)
	v.NoteOn(1.0, psalm.Message{"frequency": 100.0}, psalm.CtrlTick{}, v.Config())
	maxStep := func() float64 {
		prev, m := float64(v.Output(0).Channels[0][0]), 0.0
		for _, s := range v.Output(0).Channels[0] {
			m = math.Max(m, math.Abs(float64(s)-prev))
			prev = float64(s)
		}
		return m
	}
	for range 4 {
		v.Synthesize()
	}
	sineStep := maxStep()
	v.Receive(1.0/3.0, 1) // triangle, both have small steps
	v.Synthesize()
	if step := maxStep(); step > 2*sineStep {
		t.Fatalf("position change stepped: %v vs %v", step, sineStep)
	}
}

func TestUnison(t *testing.T) {
	one := steady()
	one.Position = 2.0 / 3.0
	seven := one
	seven.Unison = 7
	seven.Detune = 30
	seven.Spread = 1

	l1, _ := render(one, psalm.Message{"frequency": 110.0}, 80)
	l7, r7 := render(seven, psalm.Message{"frequency": 110.0}, 80)
	if ratio := (rms(l7) + rms(r7)) / 2 / rms(l1); ratio < 0.6 || ratio > 1.6 {
		t.Fatalf("unison level ratio %v", ratio)
	}

	corr, el, er := 0.0, 0.0, 0.0
	for i := range l7 {
		corr += l7[i] * r7[i]
		el += l7[i] * l7[i]
		er += r7[i] * r7[i]
	}
	if c := corr / math.Sqrt(el*er); c > 0.9 {
		t.Fatalf("expected a wide stereo unison, correlation %v", c)
	}
}

func TestFilter(t *testing.T) {
	high := func(filter FilterMode) float64 {
		cfg := steady()
		cfg.Position = 2.0 / 3.0
		cfg.Filter = filter
		cfg.Cutoff = 400
		cfg.KeyTrack = 0
		l, _ := render(cfg, psalm.Message{"frequency": 110.0}, 40)
		p := spectrum(l)
		e := 0.0
		for k := int(math.Round(3000 * 16384 / sr)); k < len(p); k++ {
			e += p[k]
		}
		return e
	}
	if open, closed := high(FilterOff), high(LowPass); closed > open*1e-3 {
		t.Fatalf("expected the lowpass to remove the highs: %v vs %v", closed, open)
	}
}

func TestNoteLifecycle(t *testing.T) {
	v := New(steady())
	v.Note(0.2, 0.5, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
	for blocks := 0; v.IsActive(); blocks++ {
		if blocks > 100 {
			t.Fatal("timed note never ended")
		}
		v.Synthesize()
	}
}

func BenchmarkUnison7(b *testing.B) {
	cfg := steady()
	cfg.Unison = 7
	cfg.Filter = LowPass
	cfg.PositionLFORate = 0.5
	cfg.PositionLFODepth = 0.3
	cfg.Position = 0.5
	v := New(cfg)
	v.NoteOn(1.0, psalm.Message{"frequency": 220.0}, psalm.CtrlTick{}, v.Config())
	for b.Loop() {
		v.Synthesize()
	}
}

// With Normalize the level stays steady while the position sweeps through a table whose frames
// differ a lot in loudness. Measured on a low note: on higher notes the brightest frames lose the
// harmonics above the Nyquist frequency and get quieter, as they should
func TestNormalizeKeepsTheLevel(t *testing.T) {
	path := filepath.Join(tableDir, "Add Synth3 Icarus.wav")
	if _, err := os.Stat(path); err != nil {
		t.Skip("table not in assets")
	}
	table := MustLoad(path, 2048)
	spreadOf := func(normalize bool) float64 {
		lo, hi := math.Inf(1), 0.0
		for _, pos := range []float64{0, 0.25, 0.5, 0.75, 0.9} {
			cfg := steady()
			cfg.Table = table
			cfg.Position = pos
			cfg.Normalize = normalize
			l, _ := render(cfg, psalm.Message{"frequency": 40.0}, 20)
			r := rms(l)
			lo, hi = math.Min(lo, r), math.Max(hi, r)
		}
		return hi / lo
	}
	raw, normalized := spreadOf(false), spreadOf(true)
	if normalized > 1.3 || raw < 1.5 {
		t.Fatalf("level spread over the positions: raw %.2f, normalized %.2f", raw, normalized)
	}
}

// The brightest frames of the asset tables played high stay alias free too. Noise like frames have
// their top harmonics as loud as the low ones, so the images of the interpolation (about -60 dB
// relative to a harmonic) come closer to the threshold than on a sawtooth
func TestBandLimitedTables(t *testing.T) {
	for _, tc := range []struct {
		file     string
		position float64
	}{
		{"KaossDrum2.wav", 0.75},
		{"MSelfFB02.wav", 0.75},
		{"OPFoldFMFB.wav", 1},
	} {
		path := filepath.Join(tableDir, tc.file)
		if _, err := os.Stat(path); err != nil {
			t.Skip("table not in assets")
		}
		for _, f0 := range []float64{440, 1900, 5100} {
			cfg := steady()
			cfg.Table = MustLoad(path, 2048)
			cfg.Position = tc.position
			l, _ := render(cfg, psalm.Message{"frequency": f0}, 60)
			harm, between := harmonicEnergy(spectrum(l), f0)
			if db := 10 * math.Log10(between/harm); db > -55 {
				t.Errorf("%s at %v Hz: aliasing at %.1f dB", tc.file, f0, db)
			}
		}
	}
}

// noteLength gives the slow presets the time to unfold
func noteLength(name string) float64 {
	switch name {
	case "riser":
		return 7
	case "drone", "sweep":
		return 5
	}
	return 1
}

func TestPresets(t *testing.T) {
	tables := []*Table{Basic()}
	if files, _ := filepath.Glob(filepath.Join(tableDir, "*.wav")); len(files) > 0 {
		for _, f := range files {
			tables = append(tables, MustLoad(f, 2048))
		}
	}
	for name, preset := range Presets {
		t.Run(name, func(t *testing.T) {
			lo, hi := math.Inf(1), 0.0
			for _, table := range tables {
				// the unison voices start at random phases: seeded, the peaks measure the same every run
				psalm.Seed(1)
				psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
				v := New(preset(table))
				v.Note(noteLength(name), 0.5, psalm.Message{"pitch": 57.0}, psalm.CtrlTick{}, v.Config())
				peak := 0.0
				for blocks := 0; v.IsActive(); blocks++ {
					if blocks > 2000 {
						t.Fatal("voice never finished")
					}
					v.Synthesize()
					for _, ch := range v.Output(0).Channels {
						for _, s := range ch {
							if math.IsNaN(float64(s)) {
								t.Fatal("NaN output")
							}
							peak = math.Max(peak, math.Abs(float64(s)))
						}
					}
				}
				lo, hi = math.Min(lo, peak), math.Max(hi, peak)
			}
			// a lowpassed preset on a noise like table (KaossDrum2 holds little below a few kHz)
			// is quiet (around 0.005, varying with the random unison phases), but never silent
			if lo < 0.001 || hi > 1.2 {
				t.Fatalf("peaks over the tables %.3f - %.3f", lo, hi)
			}
			t.Logf("%-8s peaks over the tables %.3f - %.3f", name, lo, hi)
		})
	}
}
