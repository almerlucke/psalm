package piano

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
)

// renderStereo plays a note (with a duration) on a fresh seeded piano, returns both channels
func renderStereo(cfg Config, pitch, velocity, duration, seconds float64) ([]float64, []float64) {
	psalm.Seed(5)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	p := NewPoly(cfg, 2)
	p.Note(duration, 1, psalm.Message{"pitch": pitch, "velocity": velocity}, psalm.CtrlTick{}, p.Config())
	var l, r []float64
	for range int(seconds*sr/float64(p.Config().BufferSize)) + 1 {
		p.Synthesize()
		ch := p.Output(0).Channels
		l = append(l, toFloat(ch[0])...)
		r = append(r, toFloat(ch[1])...)
	}
	return l, r
}

func mono(l, r []float64) []float64 {
	x := make([]float64, len(l))
	for i := range x {
		x[i] = (l[i] + r[i]) / 2
	}
	return x
}

// the new presets render finite, audible notes on low, middle and high keys, at soft and hard
// velocities, and after the release
func TestMorePresetsRender(t *testing.T) {
	for name := range morePresets {
		cfg := Presets[name]()
		for _, key := range []float64{28, 48, 64, 84} {
			for _, vel := range []float64{0.2, 0.9} {
				l, r := renderStereo(cfg, key, vel, 0.3, 0.6)
				peak := 0.0
				for i := range l {
					if math.IsNaN(l[i]) || math.IsInf(l[i], 0) || math.IsNaN(r[i]) || math.IsInf(r[i], 0) {
						t.Fatalf("%s key %v velocity %v: not finite", name, key, vel)
					}
					peak = max(peak, max(math.Abs(l[i]), math.Abs(r[i])))
				}
				if peak < 1e-3 {
					t.Errorf("%s key %v velocity %v: silent, peak %g", name, key, vel, peak)
				}
				if peak > 4 {
					t.Errorf("%s key %v velocity %v: too loud, peak %g", name, key, vel, peak)
				}
			}
		}
	}
}

type fingerprint struct {
	bands  [24]float64 // dB per band, the mean removed
	attack float64     // log2 of the seconds to the peak
	centr  float64     // log2 of the centroid
	decay  float64     // dB from the second 0.3-0.5 to 1.5-1.9
	tail   float64     // dB of the tail of a short note, 1.0-1.4 s against 0.3-0.5 s
	side   float64     // dB of the side against the mid
}

func fingerprintOf(cfg Config) fingerprint {
	var f fingerprint
	l, r := renderStereo(cfg, 60, 0.8, 2, 2)
	x := mono(l, r)
	s := spectrum(x[:int(sr)], 65536)
	for b := range f.bands {
		lo := 80 * math.Pow(14000.0/80, float64(b)/24)
		hi := 80 * math.Pow(14000.0/80, float64(b+1)/24)
		sum := 0.0
		for i := int(lo * 65536 / sr); i <= int(hi*65536/sr) && i < len(s); i++ {
			sum += s[i] * s[i]
		}
		f.bands[b] = 10 * math.Log10(sum+1e-12)
	}
	mean := 0.0
	for _, v := range f.bands {
		mean += v / 24
	}
	for b := range f.bands {
		f.bands[b] -= mean
	}
	peak, at := 0.0, 0
	for i, v := range x[:int(0.5*sr)] {
		if math.Abs(v) > peak {
			peak, at = math.Abs(v), i
		}
	}
	f.attack = math.Log2(float64(at)/sr + 0.0005)
	f.centr = math.Log2(centroid(x, 1))
	f.decay = rmsDB(x, 1.5, 1.9) - rmsDB(x, 0.3, 0.5)
	l2, r2 := renderStereo(cfg, 60, 0.8, 0.3, 1.5)
	x2 := mono(l2, r2)
	f.tail = rmsDB(x2, 1.0, 1.4) - rmsDB(x2, 0.3, 0.5)
	side := make([]float64, len(l))
	for i := range side {
		side[i] = (l[i] - r[i]) / 2
	}
	f.side = rmsDB(side, 0.1, 1.5) - rmsDB(x, 0.1, 1.5)
	return f
}

// distance is about how many dB (and octaves, scaled to dB) two fingerprints are apart
func distance(a, b fingerprint) float64 {
	s := 0.0
	for i := range a.bands {
		d := a.bands[i] - b.bands[i]
		s += d * d / 24
	}
	d := math.Sqrt(s)
	d += math.Abs(a.attack-b.attack) * 3
	d += math.Abs(a.centr-b.centr) * 12
	d += math.Abs(a.decay-b.decay) * 0.4
	d += math.Abs(a.tail-b.tail) * 0.4
	d += math.Abs(a.side-b.side) * 0.4
	return d
}

// the new presets are different instruments: no two are alike in tone, attack, decay, tail and
// stereo image
func TestMorePresetsDistinct(t *testing.T) {
	var names []string
	for n := range morePresets {
		names = append(names, n)
	}
	for n := range Presets {
		if _, ok := morePresets[n]; !ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	fp := map[string]fingerprint{}
	for _, n := range names {
		fp[n] = fingerprintOf(Presets[n]())
	}
	type pair struct {
		a, b string
		d    float64
	}
	var pairs []pair
	for i, a := range names {
		for _, b := range names[i+1:] {
			pairs = append(pairs, pair{a, b, distance(fp[a], fp[b])})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].d < pairs[j].d })
	for _, p := range pairs[:12] {
		t.Logf("closest: %s %s %.1f", p.a, p.b, p.d)
	}
	for _, p := range pairs {
		if p.d < distinctMin && !(exempt[p.a+" "+p.b] || exempt[p.b+" "+p.a]) {
			t.Errorf("%s and %s are too much alike: %.1f", p.a, p.b, p.d)
		}
	}
}

// distinctMin is the least fingerprint distance of two presets
const distinctMin = 2.5

// presets that differ in the noise of the action or in the bridge only, which a single note in
// the middle doesn't show (TestMorePresetsFeatures measures those)
var exempt = map[string]bool{"grand stretchedWide": true}

// the presets that name a feature show it in measurement
func TestMorePresetsFeatures(t *testing.T) {
	cents := func(name string, key float64) float64 {
		l, r := renderStereo(Presets[name](), key, 0.8, 1, 1)
		f0 := 440 * math.Exp2((key-69)/12)
		return 1200 * math.Log2(peakFrequency(mono(l, r), f0, 0.03)/f0)
	}
	if s, g, e := cents("stretchedWide", 100), cents("grand", 100), cents("equalTempered", 100); !(s > g+10 && g > e+10) {
		t.Errorf("the top should be sharper the more stretch: %.1f, %.1f, %.1f cents", s, g, e)
	}

	side := func(name string) float64 {
		l, r := renderStereo(Presets[name](), 96, 0.8, 1, 1)
		sd := make([]float64, len(l))
		for i := range sd {
			sd[i] = (l[i] - r[i]) / 2
		}
		return rmsDB(sd, 0.05, 0.9) - rmsDB(mono(l, r), 0.05, 0.9)
	}
	if w, g, m := side("widePiano"), side("grand"), side("monoPiano"); !(w > g+1 && g > m+3) {
		t.Errorf("the treble should be wider with the width: %.1f, %.1f, %.1f dB", w, g, m)
	}

	bright := func(name string) float64 {
		l, r := renderStereo(Presets[name](), 60, 0.8, 1, 1)
		return centroid(mono(l, r), 0.3)
	}
	if tk, g, mf := bright("tack"), bright("grand"), bright("muffled"); !(tk > g*1.1 && g > mf*1.1) {
		t.Errorf("hammers: tack, grand, muffled should get duller: %.0f, %.0f, %.0f Hz", tk, g, mf)
	}
	if u, g := bright("unaCorda"), bright("grand"); u > g*0.98 {
		t.Errorf("una corda should be duller than the grand: %.0f against %.0f Hz", u, g)
	}

	tail := func(name string) float64 {
		l, r := renderStereo(Presets[name](), 60, 0.8, 0.3, 1.5)
		x := mono(l, r)
		return rmsDB(x, 1.0, 1.4) - rmsDB(x, 0.3, 0.5)
	}
	if w, g := tail("pedalWash"), tail("grand"); w < g+15 {
		t.Errorf("the pedal wash should ring on after the release: %.1f against %.1f dB", w, g)
	}

	early := func(name string) float64 {
		l, r := renderStereo(Presets[name](), 28, 1, 0.5, 0.5)
		return rmsDB(mono(l, r), 0, 0.02)
	}
	if n, s := early("noisyAction"), early("silentAction"); n < s+3 {
		t.Errorf("the noisy action should be heard in the attack: %.1f against %.1f dB", n, s)
	}
}
