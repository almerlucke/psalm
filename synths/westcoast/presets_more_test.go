package westcoast

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

func fresh() {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
}

func newVoice(cfg Config, pitch, velocity, hold float64) *Voice {
	v := New(cfg)
	msg := psalm.Message{"pitch": pitch, "velocity": velocity}
	if hold > 0 {
		v.Note(hold, 1, msg, psalm.CtrlTick{}, v.Config())
	} else {
		v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	}
	return v
}

func finitePeak(t *testing.T, name string, x []float64) float64 {
	t.Helper()
	peak := 0.0
	for _, s := range x {
		if math.IsNaN(s) || math.IsInf(s, 0) {
			t.Fatalf("%s: sample %v", name, s)
		}
		peak = math.Max(peak, math.Abs(s))
	}
	return peak
}

func TestMorePresetsSound(t *testing.T) {
	for name, f := range morePresets {
		for _, pitch := range []float64{36, 57, 84} {
			for _, vel := range []float64{0.2, 0.8, 1} {
				fresh()
				v := newVoice(f(), pitch, vel, 0)
				x := run(v, 2.5)
				if peak := finitePeak(t, name, x); peak < 1e-3 {
					t.Errorf("%s at pitch %g, velocity %g: silent (peak %g)", name, pitch, vel, peak)
				} else if peak > 8 {
					t.Errorf("%s at pitch %g, velocity %g: peak %g", name, pitch, vel, peak)
				}
			}
		}
	}
}

func TestMorePresetsRelease(t *testing.T) {
	for name, f := range morePresets {
		fresh()
		cfg := f()
		v := newVoice(cfg, 57, 0.8, 0)
		finitePeak(t, name, run(v, 3.5))
		if (cfg.Sustain || cfg.Cycle) && !v.IsActive() {
			t.Errorf("%s: gone while the key is held", name)
		}
		v.NoteOff()
		finitePeak(t, name, run(v, 30))
		if v.IsActive() {
			t.Errorf("%s: still sounding 30 seconds after the release", name)
		}
	}
}

// a fingerprint: 24 log-spaced band levels (dB, relative to the whole), the attack time and the centroid
type fingerprint struct {
	bands  [24]float64
	attack float64 // seconds to 90% of the loudest 10 ms
	cent   float64 // Hz
}

func fingerprintOf(x []float64) fingerprint {
	const n = 16384
	var fp fingerprint
	f := fft.New(n)
	re, im := make([]float64, n), make([]float64, n)
	w := fft.HannWindow(n)
	p := make([]float64, n/2)
	for start := 0; start+n <= len(x); start += n / 2 {
		for i := range n {
			re[i], im[i] = x[start+i]*w[i], 0
		}
		f.Forward(re, im)
		for k := range p {
			p[k] += re[k]*re[k] + im[k]*im[k]
		}
	}
	total, num := 0.0, 0.0
	var e [24]float64
	const lo, hi = 50.0, 18000.0
	for k := 1; k < len(p); k++ {
		fq := float64(k) * sr() / n
		total += p[k]
		num += fq * p[k]
		if fq >= lo && fq < hi {
			e[int(24*math.Log(fq/lo)/math.Log(hi/lo))] += p[k]
		}
	}
	for b := range e {
		fp.bands[b] = 10 * math.Log10(e[b]/total+1e-9)
	}
	fp.cent = num / total

	win := int(0.01 * sr())
	var env []float64
	peak := 0.0
	for s := 0; s+win <= len(x); s += win {
		sum := 0.0
		for _, v := range x[s : s+win] {
			sum += v * v
		}
		r := math.Sqrt(sum / float64(win))
		env = append(env, r)
		peak = math.Max(peak, r)
	}
	for i, r := range env {
		if r >= 0.9*peak {
			fp.attack = float64(i) * 0.01
			break
		}
	}
	return fp
}

// distance is the spectral difference in dB (root mean square over the bands), plus octaves of
// centroid and of attack time (in milliseconds, from 10), each counted for a few dB
func distance(a, b fingerprint) float64 {
	sum := 0.0
	for i := range a.bands {
		d := a.bands[i] - b.bands[i]
		sum += d * d
	}
	d := math.Sqrt(sum / 24)
	d += 4 * math.Abs(math.Log2(a.cent/b.cent))
	d += 2 * math.Abs(math.Log2((a.attack*1000+10)/(b.attack*1000+10)))
	return d
}

func fingerprints(names []string) map[string]fingerprint {
	out := map[string]fingerprint{}
	for _, n := range names {
		fresh()
		out[n] = fingerprintOf(run(newVoice(Presets[n](), 57, 0.8, 0), 1.5))
	}
	return out
}

// TestMorePresetsDistinct makes sure no new preset is a copy of another one (A3, velocity 0.8, 1.5 s)
func TestMorePresetsDistinct(t *testing.T) {
	var all, added []string
	for n := range Presets {
		all = append(all, n)
		if _, ok := morePresets[n]; ok {
			added = append(added, n)
		}
	}
	sort.Strings(all)
	fp := fingerprints(all)
	const threshold = 5
	for _, a := range added {
		for _, b := range all {
			if a == b || (morePresets[b] != nil && b < a) {
				continue
			}
			if d := distance(fp[a], fp[b]); d < threshold {
				t.Errorf("%s and %s sound alike: distance %.2f", a, b, d)
			}
		}
	}
}

func TestMorePresetsMatrix(t *testing.T) {
	for name, f := range morePresets {
		cfg := f()
		used := false
		for _, sl := range cfg.Mods {
			used = used || sl.Src != modmatrix.None
		}
		if !used {
			continue
		}
		bare := cfg
		bare.Mods = [modmatrix.Slots]modmatrix.Slot{}
		msg := psalm.Message{"pitch": 57.0, "velocity": 1.0}
		for _, sl := range cfg.Mods {
			if sl.Src == modmatrix.Wheel {
				msg["modWheel"] = 1.0 // the wheel alone folds the wave further, which hides the other slots
			}
		}
		play := func(c Config) []float64 {
			fresh()
			v := New(c)
			v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
			return run(v, 2)
		}
		with, without := play(cfg), play(bare)
		diff, norm := 0.0, 0.0
		for i := range with {
			d := with[i] - without[i]
			diff += d * d
			norm += without[i] * without[i]
		}
		if rel := math.Sqrt(diff / math.Max(norm, 1e-12)); rel < 0.01 {
			t.Errorf("%s: the matrix changes the sound by only %.4f", name, rel)
		}
	}
}
