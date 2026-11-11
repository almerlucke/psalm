package pulsar

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// native is the note of the texture presets made for very low notes (a few Hz to some tens of Hz)
var native = map[string]float64{
	"rhythm": 6, "crackle": 50, "insects": 45, "raindrops": 12, "motor": 25,
	"grainCloud": 40, "shimmerDust": 60, "lowCloud": 25, "scatterBells": 20, "swarmCloud": 80,
	"stutterTick": 12, "gallop": 10, "clavePulse": 8, "machineGun": 18,
	"fireCrackle": 45, "crickets": 40, "drizzle": 15, "geigerTicks": 30,
}

func fresh() {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
}

// message is a note at a pitch, or the native frequency of a texture
func message(name string, pitch, velocity float64) psalm.Message {
	if f, ok := native[name]; ok {
		return psalm.Message{"frequency": f}
	}
	return psalm.Message{"pitch": pitch, "velocity": velocity}
}

// playMono plays a note of the given length (0 holds it) and returns both channels summed over a time
func playMono(cfg Config, msg psalm.Message, amp, hold, seconds float64) []float64 {
	v := New(cfg)
	if hold > 0 {
		v.Note(hold, amp, msg, psalm.CtrlTick{}, v.Config())
	} else {
		v.NoteOn(amp, msg, psalm.CtrlTick{}, v.Config())
	}
	var x []float64
	for range int(seconds * sr / float64(v.Config().BufferSize)) {
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			x = append(x, float64(ch[0][i]+ch[len(ch)-1][i]))
		}
	}
	return x
}

func finitePeak(t *testing.T, name string, x []float64) float64 {
	t.Helper()
	peak := 0.0
	for _, s := range x {
		if math.IsNaN(s) || math.IsInf(s, 0) {
			t.Fatalf("%s: sample %v", name, s)
		}
		peak = max(peak, math.Abs(s))
	}
	return peak
}

func TestMorePresetsSound(t *testing.T) {
	for name, f := range morePresets {
		for _, pitch := range []float64{36, 57, 84} {
			for _, vel := range []float64{0.2, 0.8, 1} {
				fresh()
				x := playMono(f(), message(name, pitch, vel), vel, 0, 2.5)
				if peak := finitePeak(t, name, x); peak < 1e-3 {
					t.Errorf("%s at pitch %g, velocity %g: silent (peak %g)", name, pitch, vel, peak)
				}
				if peak := finitePeak(t, name, x); peak > 8 {
					t.Errorf("%s at pitch %g, velocity %g: peak %g", name, pitch, vel, peak)
				}
			}
		}
	}
}

func TestMorePresetsRelease(t *testing.T) {
	for name, f := range morePresets {
		fresh()
		v := New(f())
		v.NoteOn(0.8, message(name, 57, 0.8), psalm.CtrlTick{}, v.Config())
		run := func(seconds float64) {
			for range int(seconds * sr / float64(v.Config().BufferSize)) {
				v.Synthesize()
				for _, ch := range v.Output(0).Channels {
					for _, s := range ch {
						if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) {
							t.Fatalf("%s: sample %v", name, s)
						}
					}
				}
			}
		}
		run(3.5) // long enough for the slowest attack
		if !v.IsActive() {
			t.Errorf("%s: gone while the key is held", name)
		}
		v.NoteOff()
		run(12)
		if v.IsActive() {
			t.Errorf("%s: still sounding 12 seconds after the release", name)
		}
	}
}

// fingerprint: 24 log-spaced band levels (dB, relative to the whole), the attack time and the centroid
type fingerprint struct {
	bands  [24]float64
	attack float64 // seconds to 90% of the loudest 10 ms
	cent   float64 // Hz
}

func fingerprintOf(x []float64) fingerprint {
	var fp fingerprint
	p := spectrum(x)
	total := 0.0
	for k := 1; k < len(p); k++ {
		total += p[k]
	}
	const lo, hi = 50.0, 18000.0
	var e [24]float64
	for k := 1; k < len(p); k++ {
		f := binFreq(k)
		if f < lo || f >= hi {
			continue
		}
		b := int(24 * math.Log(f/lo) / math.Log(hi/lo))
		e[b] += p[k]
	}
	for b := range e {
		fp.bands[b] = 10 * math.Log10(e[b]/total+1e-9)
	}
	fp.cent = centroid(p)

	win := int(0.01 * sr)
	var env []float64
	peak := 0.0
	for s := 0; s+win <= len(x); s += win {
		sum := 0.0
		for _, v := range x[s : s+win] {
			sum += v * v
		}
		r := math.Sqrt(sum / float64(win))
		env = append(env, r)
		peak = max(peak, r)
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
		x := playMono(Presets[n](), message(n, 57, 0.8), 0.8, 0, 1.5)
		out[n] = fingerprintOf(x)
	}
	return out
}

// TestMorePresetsDistinct makes sure no new preset is a copy of another one
func TestMorePresetsDistinct(t *testing.T) {
	var all, fresh []string
	for n := range Presets {
		all = append(all, n)
		if _, ok := morePresets[n]; ok {
			fresh = append(fresh, n)
		}
	}
	sort.Strings(all)
	fp := fingerprints(all)
	const threshold = 5
	for _, a := range fresh {
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
		msg := message(name, 57, 1)
		msg["modWheel"] = 1.0
		fresh()
		with := playMono(cfg, msg, 1, 0, 2)
		fresh()
		without := playMono(bare, msg, 1, 0, 2)
		diff, norm := 0.0, 0.0
		for i := range with {
			d := with[i] - without[i]
			diff += d * d
			norm += without[i] * without[i]
		}
		if rel := math.Sqrt(diff / max(norm, 1e-12)); rel < 0.01 {
			t.Errorf("%s: the matrix changes the sound by only %.4f", name, rel)
		}
	}
}
