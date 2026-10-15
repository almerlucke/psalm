package waveguide

import (
	"math"
	"math/cmplx"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// sortedMore is the names of the presets of presets_more.go
func sortedMore() []string {
	var names []string
	for n := range morePresets {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func warm(cfg Config) {
	WarmBow(cfg)
	WaitBow()
}

// family pitches the presets are played at in the tests
func rangeOf(cfg Config, name string) []float64 {
	switch name {
	case "contrabass":
		return []float64{33, 40, 48, 57}
	case "bassClarinet", "chalumeau":
		return []float64{45, 52, 60, 67}
	case "slapBass", "fretlessBass", "pickBass", "uprightWalking":
		return []float64{33, 40, 48, 57}
	}
	switch cfg.Model {
	case Bow:
		return []float64{43, 55, 67, 79}
	case Pipe:
		return []float64{55, 62, 72, 84}
	case Membrane:
		return []float64{40, 52, 64, 76}
	}
	return []float64{40, 52, 64, 76, 88}
}

func TestMorePresetsRender(t *testing.T) {
	for _, name := range sortedMore() {
		cfg := morePresets[name]()
		warm(cfg)
		for _, p := range rangeOf(cfg, name) {
			for _, vel := range []float64{0.3, 1} {
				sameNoise()
				v := New(cfg)
				v.NoteOn(1, psalm.Message{"pitch": p, "velocity": vel}, psalm.CtrlTick{}, v.Config())
				x := run(v, 0.8)
				peak := 0.0
				for _, s := range x {
					if math.IsNaN(s) || math.IsInf(s, 0) {
						t.Fatalf("%s pitch %g velocity %g: %g", name, p, vel, s)
					}
					peak = math.Max(peak, math.Abs(s))
				}
				if peak < 0.005 || peak > 6 {
					t.Errorf("%s pitch %g velocity %g: peak %g", name, p, vel, peak)
				}
				// a note held, then let go
				v.NoteOff()
				for _, s := range run(v, 2+cfg.BowRelease) {
					if math.IsNaN(s) || math.IsInf(s, 0) || math.Abs(s) > 6 {
						t.Fatalf("%s pitch %g: released %g", name, p, s)
					}
				}
				if cfg.Model == Bow && v.IsActive() {
					t.Errorf("%s pitch %g: plays on after the bow stopped", name, p)
				}
			}
		}
	}
}

// harmonicRatio is the partial of the string a preset with a finger on a node sounds at
func harmonicRatio(cfg Config) float64 {
	if cfg.Model == Pluck && cfg.Harmonic > 0 {
		return math.Round(1 / cfg.Harmonic)
	}
	return 1
}

// pluckCents is how far the strongest of the first partials of a plucked string is from where a
// string of that stiffness has it (a finger on a node leaves the partials of its multiples)
func pluckCents(y []float64, f0, b, ratio float64) float64 {
	best, bc := 0.0, 0.0
	bs := stiffnessAt(b, f0)
	for k := 1.0; k <= 4; k++ {
		n := k * ratio
		want := f0 * n * stretch(bs, int(n))
		c, ca := want, 0.0
		for i := -50; i <= 50; i++ {
			g := want * (1 + 0.02*float64(i)/50)
			if a := amplitude(y, g); a > ca {
				c, ca = g, a
			}
		}
		if ca > best {
			best, bc = ca, cents(c, want)
		}
	}
	return bc
}

func TestMorePresetsInTune(t *testing.T) {
	for _, name := range sortedMore() {
		cfg := morePresets[name]()
		if cfg.Model == Membrane {
			continue
		}
		warm(cfg)
		for _, p := range []float64{45, 57, 69} {
			if cfg.Model == Pipe && p < 50 && name != "bassClarinet" && name != "chalumeau" {
				continue
			}
			if cfg.Model == Pluck && cfg.Harmonic > 0 && p > 57 {
				continue // the harmonic is out of the range of the sampling's pitch finder
			}
			c := dry(cfg)
			c.VibratoDepth, c.Mods, c.Slide, c.Sympathetic = 0, [modmatrix.Slots]modmatrix.Slot{}, 0, 0
			c.BowReverse, c.BowNoise = 0, 0
			sameNoise()
			_, x := play(c, p, 2)
			want := hz(p) * harmonicRatio(cfg)
			switch cfg.Model {
			case Bow:
				y := x[int(1.2*sr()):]
				if got := partialCents(y, hz(p)); math.Abs(got) > 8 {
					t.Errorf("%s %g: %.1f cents out", name, p, got)
				}
			case Pipe:
				y := at(x, 1, 1.8)
				if peakOf(y) < 0.05 {
					t.Errorf("%s %g: silent", name, p)
					continue
				}
				if got := cents(pitchOf(y), want); math.Abs(got) > 6 {
					t.Errorf("%s %g: %.1f cents out", name, p, got)
				}
			default:
				// the plucks: a window once the first twang is over, in the notes that die fast sooner
				from := 0.25
				if cfg.Decay < 0.5 {
					from = 0.08
				}
				y := x[int(from*sr()):]
				tol := 6.0
				if cfg.FretBuzz > 0 || cfg.TensionMod > 0.5 || cfg.Polarization > 0.6 {
					tol = 14 // the string that buzzes, bends or beats is a little off
				}
				if peakOf(y[:4096]) < 0.005 {
					continue
				}
				got := pluckCents(y[:6000], hz(p), cfg.Inharmonicity, harmonicRatio(cfg))
				if math.Abs(got) > tol {
					t.Errorf("%s %g: %.1f cents out", name, p, got)
				}
			}
		}
	}
}

func TestMoreMembranesSoundAndDie(t *testing.T) {
	for _, name := range sortedMore() {
		cfg := morePresets[name]()
		if cfg.Model != Membrane {
			continue
		}
		sameNoise()
		_, x := play(cfg, 52, 4)
		head := peakOf(at(x, 0, 0.3))
		if head < 0.05 {
			t.Errorf("%s is quiet: %.3f", name, head)
		}
		if tail := peakOf(at(x, 3, 4)); tail > 0.1*head {
			t.Errorf("%s doesn't die: %.3f then %.3f", name, head, tail)
		}
		// harder is louder and brighter
		sameNoise()
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 52.0, "velocity": 0.3}, psalm.CtrlTick{}, v.Config())
		soft := peakOf(run(v, 0.3))
		if soft >= head {
			t.Errorf("%s: a soft hit %.3f is not softer than a hard one %.3f", name, soft, head)
		}
	}
}

func TestMorePresetsMatrixResponds(t *testing.T) {
	for _, name := range sortedMore() {
		cfg := morePresets[name]()
		warm(cfg)
		for i, sl := range cfg.Mods {
			if sl.Dest == "" {
				continue
			}
			msg := psalm.Message{"pitch": 57.0, "velocity": 0.9, "modWheel": 1.0}
			render := func(c Config, seed uint64) []float64 {
				psalm.Seed(seed)
				psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
				v := New(c)
				v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
				return run(v, 1)
			}
			// a random source is small for some seeds: the best of a few
			rel := 0.0
			for seed := uint64(1); seed <= 6; seed++ {
				with := render(cfg, seed)
				c := cfg
				c.Mods[i] = modmatrix.Slot{}
				without := render(c, seed)
				d, ref := 0.0, 0.0
				for k := range with {
					d += (with[k] - without[k]) * (with[k] - without[k])
					ref += without[k] * without[k]
				}
				rel = math.Max(rel, math.Sqrt(d/(ref+1e-12)))
			}
			if rel < 0.02 {
				t.Errorf("%s: slot %d (%v to %s) changes the sound by %.4f", name, i, sl.Src, sl.Dest, rel)
			}
		}
	}
}

// ---- distinctness

// fingerprint is how a preset sounds: its spectrum in 24 bands (dB, the level taken out), the time
// it takes to speak, the centroid and the fall of the level
type fingerprint struct {
	bands    [24]float64
	attack   float64 // log2 of the milliseconds to 70% of the loudest
	centroid float64 // log2 of Hz
	decay    float64 // dB, the level at 1.2-1.5 s below the loudest, at most 40
}

func fft(a []complex128) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}
	for l := 2; l <= n; l <<= 1 {
		w := cmplx.Exp(complex(0, -2*math.Pi/float64(l)))
		for i := 0; i < n; i += l {
			wn := complex(1, 0)
			for k := 0; k < l/2; k++ {
				u, v := a[i+k], a[i+k+l/2]*wn
				a[i+k], a[i+k+l/2] = u+v, u-v
				wn *= w
			}
		}
	}
}

func fingerprintOf(x []float64) fingerprint {
	var f fingerprint
	// spectrum
	n := 1 << 15
	a := make([]complex128, n)
	start := int(0.02 * sr())
	for i := range n {
		w := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(n-1))
		if start+i < len(x) {
			a[i] = complex(x[start+i]*w, 0)
		}
	}
	fft(a)
	var e [24]float64
	lo, hi := 100.0, 12000.0
	num, den := 0.0, 0.0
	for k := 1; k < n/2; k++ {
		fr := float64(k) * sr() / float64(n)
		p := real(a[k])*real(a[k]) + imag(a[k])*imag(a[k])
		if fr >= 80 {
			num += p * fr
			den += p
		}
		if fr < lo || fr >= hi {
			continue
		}
		b := int(24 * math.Log(fr/lo) / math.Log(hi/lo))
		e[b] += p
	}
	mean := 0.0
	for i := range e {
		f.bands[i] = 10 * math.Log10(e[i]+1e-12)
		mean += f.bands[i] / 24
	}
	for i := range f.bands {
		f.bands[i] = math.Max(f.bands[i]-mean, -40)
	}
	f.centroid = math.Log2(num / (den + 1e-12))
	// envelope
	win := int(0.005 * sr())
	var env []float64
	top := 0.0
	for i := 0; i+win <= len(x); i += win {
		pk := 0.0
		for _, s := range x[i : i+win] {
			pk = math.Max(pk, math.Abs(s))
		}
		env = append(env, pk)
		top = math.Max(top, pk)
	}
	for i, pk := range env {
		if pk >= 0.7*top {
			f.attack = math.Log2(float64(i)*5 + 2.5)
			break
		}
	}
	late, cnt := 0.0, 0.0
	for i := int(1.2 / 0.005); i < len(env); i++ {
		late = math.Max(late, env[i])
		cnt++
	}
	f.decay = math.Min(40, 20*math.Log10(top/(late+1e-9)))
	return f
}

// distance is how far apart two presets sound: about 1 is a clearly other sound
func distance(a, b fingerprint) float64 {
	s := 0.0
	for i := range a.bands {
		d := (a.bands[i] - b.bands[i]) / 6
		s += d * d / 24
	}
	da := (a.attack - b.attack) / 1.5
	dc := (a.centroid - b.centroid) / 0.5
	dd := (a.decay - b.decay) / 8
	return math.Sqrt(s + da*da + dc*dc + dd*dd)
}

func TestMorePresetsAreDistinct(t *testing.T) {
	prints := map[string]fingerprint{}
	for n, f := range Presets {
		cfg := f()
		warm(cfg)
		sameNoise()
		v := New(cfg)
		v.NoteOn(1, psalm.Message{"pitch": 57.0, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
		prints[n] = fingerprintOf(run(v, 1.5))
	}
	var all []string
	for n := range prints {
		all = append(all, n)
	}
	sort.Strings(all)
	type pair struct {
		a, b string
		d    float64
	}
	var near []pair
	for i, a := range all {
		for _, b := range all[i+1:] {
			if _, ok := morePresets[a]; !ok {
				if _, ok := morePresets[b]; !ok {
					continue // two of the old ones
				}
			}
			if d := distance(prints[a], prints[b]); d < distinctThreshold {
				near = append(near, pair{a, b, d})
			}
		}
	}
	sort.Slice(near, func(i, j int) bool { return near[i].d < near[j].d })
	for _, p := range near {
		t.Errorf("%s and %s sound alike: distance %.2f", p.a, p.b, p.d)
	}
}

const distinctThreshold = 0.6
