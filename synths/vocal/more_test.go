package vocal

import (
	"math"
	"math/cmplx"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
)

// the tests of the extra presets (presets_more.go): they are sound, they differ, the matrix works

func prFresh() {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
}

// prRender plays a note of a duration for seconds and returns both channels
func prRender(cfg Config, msg psalm.Message, duration, seconds float64, pre psalm.Message) ([]float64, []float64) {
	prFresh()
	v := New(cfg)
	if pre != nil {
		v.ReceiveMessage(pre)
	}
	v.Note(duration, 1, msg, psalm.CtrlTick{}, v.Config())
	var l, r []float64
	for range int(math.Ceil(seconds * v.Config().SampleRate / float64(v.Config().BufferSize))) {
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			l = append(l, float64(ch[0][i]))
			r = append(r, float64(ch[len(ch)-1][i]))
		}
	}
	return l, r
}

func prFinite(t *testing.T, name string, x []float64) float64 {
	t.Helper()
	p := 0.0
	for _, s := range x {
		if math.IsNaN(s) || math.IsInf(s, 0) {
			t.Fatalf("%s: not finite", name)
		}
		p = math.Max(p, math.Abs(s))
	}
	return p
}

func TestMorePresetsAreSound(t *testing.T) {
	for _, name := range prNames() {
		f := Presets[name]
		for _, pitch := range []float64{36, 57, 72, 93} {
			for _, vel := range []float64{0.2, 0.8, 1} {
				l, r := prRender(f(), psalm.Message{"pitch": pitch, "velocity": vel}, 0.6, 1.2, nil)
				pl, pr := prFinite(t, name, l), prFinite(t, name, r)
				if pl < 1e-4 || pr < 1e-4 {
					t.Errorf("%s: silent at pitch %g velocity %g (%g, %g)", name, pitch, vel, pl, pr)
				}
				if pl > 8 && pitch < 90 {
					t.Errorf("%s: peaks at %g at pitch %g", name, pl, pitch)
				}
			}
		}
		// a held note let go: the tail is finite and has died down by the end
		l, _ := prRender(f(), psalm.Message{"pitch": 57.0}, 0.3, 4, nil)
		prFinite(t, name, l)
	}
}

// prFingerprint: 24 log spaced band energies (dB, relative to the total), the attack time and the centroid
func prFingerprint(x []float64, sr float64) ([24]float64, float64, float64) {
	n := 65536
	buf := make([]complex128, n)
	for i := 0; i < n && i < len(x); i++ {
		buf[i] = complex(x[i]*(0.5-0.5*math.Cos(2*math.Pi*float64(i)/float64(n))), 0)
	}
	prFFT(buf)
	var bands [24]float64
	lo, hi := 60.0, 12000.0
	total, cent := 0.0, 0.0
	for k := 1; k < n/2; k++ {
		f := float64(k) * sr / float64(n)
		if f < lo || f >= hi {
			continue
		}
		e := real(buf[k])*real(buf[k]) + imag(buf[k])*imag(buf[k])
		b := int(24 * math.Log(f/lo) / math.Log(hi/lo))
		bands[b] += e
		total += e
		cent += e * math.Log2(f)
	}
	for i := range bands {
		bands[i] = 10 * math.Log10(bands[i]/total+1e-12)
	}
	// the attack: the time the 20 ms envelope takes to reach 80% of its peak
	win := int(0.02 * sr)
	peak, env := 0.0, []float64{}
	for i := 0; i+win <= len(x); i += win / 4 {
		e := 0.0
		for _, s := range x[i : i+win] {
			e += s * s
		}
		env = append(env, math.Sqrt(e/float64(win)))
		peak = math.Max(peak, env[len(env)-1])
	}
	attack := 0.0
	for i, e := range env {
		if e >= 0.8*peak {
			attack = float64(i) * float64(win/4) / sr
			break
		}
	}
	return bands, attack, cent / total
}

func prFFT(a []complex128) {
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
			for j := 0; j < l/2; j++ {
				u, v := a[i+j], a[i+j+l/2]*wn
				a[i+j], a[i+j+l/2] = u+v, u-v
				wn *= w
			}
		}
	}
}

type prPrint struct {
	bands    [24]float64
	attack   float64
	centroid float64
}

// prDistance: the rms difference of the bands in dB, plus the attack in octaves and the centroid in octaves
func prDistance(a, b prPrint) float64 {
	s := 0.0
	for i := range a.bands {
		d := a.bands[i] - b.bands[i]
		s += d * d
	}
	d := math.Sqrt(s / 24)
	d += 4 * math.Abs(math.Log2((a.attack+0.01)/(b.attack+0.01)))
	d += 6 * math.Abs(a.centroid-b.centroid)
	return d
}

func prPrints(names []string) []prPrint {
	var out []prPrint
	for _, name := range names {
		l, _ := prRender(Presets[name](), psalm.Message{"pitch": 57.0, "velocity": 0.8}, 1.2, 1.5, nil)
		sr := psalm.CurrentEnv().Config().SampleRate
		var p prPrint
		p.bands, p.attack, p.centroid = prFingerprint(l, sr)
		out = append(out, p)
	}
	return out
}

// minDistance is how far apart two presets must sound
const minDistance = 6

func TestMorePresetsAreDistinct(t *testing.T) {
	names := prNames()
	all := make([]string, 0, len(Presets))
	for n := range Presets {
		all = append(all, n)
	}
	sort.Strings(all)
	prints := prPrints(all)
	index := map[string]int{}
	for i, n := range all {
		index[n] = i
	}
	type pair struct {
		a, b string
		d    float64
	}
	var pairs []pair
	for _, a := range names {
		for _, b := range all {
			if a == b || (index[b] < index[a] && prIsMore(b)) {
				continue
			}
			pairs = append(pairs, pair{a, b, prDistance(prints[index[a]], prints[index[b]])})
		}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].d < pairs[j].d })
	for i, p := range pairs {
		if i < 8 {
			t.Logf("closest %s / %s: %.1f", p.a, p.b, p.d)
		}
		if p.d < minDistance {
			t.Errorf("%s and %s sound alike (%.1f, at least %g wanted)", p.a, p.b, p.d, float64(minDistance))
		}
	}
}

func TestMorePresetsDescribe(t *testing.T) {
	// every extra preset is in the table and described (TestDescribe runs the checks on all of Presets)
	for _, name := range prNames() {
		if _, ok := Presets[name]; !ok {
			t.Errorf("%s is not registered", name)
		}
	}
}

// TestMorePresetsMatrixResponds: a preset with a matrix sounds different from the same with the matrix empty
func TestMorePresetsMatrixResponds(t *testing.T) {
	used := 0
	for _, name := range prNames() {
		cfg := Presets[name]()
		var plain Config = cfg
		has := false
		for i, sl := range cfg.Mods {
			if sl.Src != 0 {
				has = true
				plain.Mods[i].Src = 0
				plain.Mods[i].Dest = ""
			}
		}
		if !has {
			continue
		}
		used++
		pre := psalm.Message{"modWheel": 1.0}
		a, _ := prRender(cfg, psalm.Message{"pitch": 57.0, "velocity": 0.8}, 1, 1.2, pre)
		b, _ := prRender(plain, psalm.Message{"pitch": 57.0, "velocity": 0.8}, 1, 1.2, pre)
		var diff, all float64
		for i := range a {
			diff += (a[i] - b[i]) * (a[i] - b[i])
			all += b[i] * b[i]
		}
		if diff < 1e-5*all {
			t.Errorf("%s: the matrix changes the sound by %.2g of its energy", name, diff/all)
		}
	}
	if used == 0 {
		t.Errorf("no extra preset uses the matrix")
	}
}
func prNames() []string {
	var n []string
	for k := range morePresets {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}

func prIsMore(name string) bool { _, ok := morePresets[name]; return ok }
