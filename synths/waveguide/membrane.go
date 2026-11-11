package waveguide

import "math"

// The membrane is a drum skin struck: a stretched circular membrane rings in modes that are not
// harmonic, the zeros of the Bessel functions J_m. The tone of the lowest mode is 1, the next are
// 1.59, 2.14, 2.30, 2.65, 2.92 ... times it. A few of them, each a damped oscillator (cheaper than a
// waveguide mesh, and exact for the ideal skin), are struck at once: how loud each starts is the shape
// of the mode where the stick lands (J_m of the place), times how hard the stick is (a soft
// mallet leaves the high modes out). The higher modes die faster the more damping there is.

// besselZeros are the first zeros j_mn of J_m, per m (the circular order), in the order of m then n
var besselZeros = [...]struct {
	m int
	j float64
}{
	{0, 2.4048}, {1, 3.8317}, {2, 5.1356}, {0, 5.5201}, {3, 6.3802}, {1, 7.0156}, {4, 7.5883},
	{2, 8.4172}, {0, 8.6537}, {5, 8.7715}, {3, 9.7610}, {6, 9.9361}, {1, 10.1735}, {4, 11.0647},
}

const numModes = len(besselZeros)

// membraneGain is how much louder the skin is than the sum of its modes: the stick presses for a time
// and does not hit in a step, which leaves the same hit with less peak (and less in all), so the skin
// is made up for it, and the presets keep their levels
const membraneGain = 2.4

// modeRatio is the frequency of mode k over the lowest
func modeRatio(k int) float64 { return besselZeros[k].j / besselZeros[0].j }

// membrane is the state of the skin of a voice
type membrane struct {
	re, im [numModes]float64 // the modes' oscillators
	cs, sn [numModes]float64 // their rotation per sample
	decay  [numModes]float64 // their decay per sample
	ratio  float64           // the pitch ratio the rotations are set for now
	glide  float64           // how far the pitch is above its note, from the tension of the strike
	step   float64           // the glide's decay per sample
	count  int
	snare  float64 // the follower of the skin's level that shakes the wires
	prev   float64 // the wires' noise, before
	rel    float64 // the decay per sample of a hand laid on the skin
	live   bool
	// the stick's contact: a push on the skin that lasts a time (a raised cosine, as the stick presses
	// and lets go), not an instant. exc is what each mode gets in all, pn of pnN samples of it are given
	exc     [numModes]float64
	pn, pnN int
}

func (m *membrane) clear() { *m = membrane{} }

// setLoss sets how fast each mode dies, for a damping and the seconds of the lowest mode
func (m *membrane) setLoss(v *Voice, damping, decay float64) {
	sr := v.Config().SampleRate
	damping = max(0, min(damping, 1))
	decay = max(0.01, decay)
	for k := range numModes {
		t60 := decay / (1 + damping*4*(modeRatio(k)-1))
		m.decay[k] = math.Pow(10, -3/(sr*t60))
	}
}

// retune sets the rotation of every mode for a pitch ratio
func (m *membrane) retune(v *Voice, ratio float64) {
	sr := v.Config().SampleRate
	for k := range numModes {
		w := 2 * math.Pi * v.freq * ratio * modeRatio(k) / sr
		if w > math.Pi*0.9 { // above what the sampling can hold: the mode is gone
			m.cs[k], m.sn[k], m.re[k], m.im[k] = 0, 0, 0, 0
			continue
		}
		m.cs[k], m.sn[k] = math.Cos(w), math.Sin(w)
	}
	m.ratio = ratio
}

// strike hits the skin: the modes start with the loudness the place and the stick give them
func (v *Voice) strike() {
	cfg := &v.cfg
	m := &v.mem
	sr := v.Config().SampleRate
	if !v.active {
		m.clear()
	}
	hard := max(0, min(cfg.Hardness+0.4*(v.velocity-0.5), 1))
	place := max(0, min(cfg.Strike, 1))
	m.setLoss(v, cfg.Damping+v.mods.Mod(dDamping), cfg.Decay*math.Exp2(v.mods.Mod(dDecay)))
	m.ratio = 0 // retuned at the first sample
	sum := 0.0
	var w [numModes]float64
	for k := range numModes {
		r := modeRatio(k)
		if v.freq*r > sr*0.45 {
			continue
		}
		w[k] = math.Jn(besselZeros[k].m, besselZeros[k].j*place) * math.Exp(-r*(1-hard)*3)
		sum += math.Abs(w[k])
	}
	// a roll: what is left of the push before goes into the skin now
	m.flush()
	if sum > 0 {
		// whatever the stick, the skin starts with the same size of hit; the velocity is the size
		g := (0.3 + 0.7*v.velocity) / sum
		for k := range numModes {
			m.exc[k] = w[k] * g
		}
		// the stick presses for a time: a hard, fast stick for a fraction of a millisecond, a soft
		// one for several. A mode that turns over in less than that hardly feels it (the soft stick
		// leaves the highs out), and the sound doesn't start with a step
		ms := (0.4 + 7*math.Pow(1-hard, 1.5)) * (1.15 - 0.3*max(0, min(v.velocity, 1)))
		m.pnN = max(2, int(ms*0.001*sr))
		m.pn = 0
	}
	m.glide = max(0, min(cfg.Tension, 1)) * 0.35 * (0.4 + 0.6*v.velocity)
	m.step = math.Exp(-1 / (0.07 * sr))
	m.rel = math.Pow(10, -3/(sr*max(0.01, cfg.Release)))
	m.count = 0
}

// push is how much of the stick's push falls on the sample i of n: a raised cosine, which adds up to 1
func push(i, n int) float64 {
	return (1 - math.Cos(2*math.Pi*(float64(i)+0.5)/float64(n))) / float64(n)
}

// flush gives the skin what is left of the push at once
func (m *membrane) flush() {
	if m.pn >= m.pnN {
		return
	}
	rest := 0.0
	for i := m.pn; i < m.pnN; i++ {
		rest += push(i, m.pnN)
	}
	for k := range numModes {
		m.re[k] += m.exc[k] * rest
	}
	m.pn = m.pnN
}

// tick returns a sample of the skin; bend is the pitch bend
func (m *membrane) tick(v *Voice, bend float64) float64 {
	if m.pn < m.pnN {
		f := push(m.pn, m.pnN)
		for k := range numModes {
			m.re[k] += m.exc[k] * f
		}
		m.pn++
	}
	if m.count&7 == 0 {
		if ratio := bend * (1 + m.glide); math.Abs(ratio-m.ratio) > 1e-7 {
			m.retune(v, ratio)
		}
	}
	m.count++
	m.glide *= m.step
	damp := 1.0
	if v.released && v.cfg.Release > 0 {
		damp = m.rel
	}
	sum := 0.0
	for k := range numModes {
		re, im := m.re[k], m.im[k]
		d := m.decay[k] * damp
		nr := (re*m.cs[k] - im*m.sn[k]) * d
		ni := (re*m.sn[k] + im*m.cs[k]) * d
		if nr*nr+ni*ni < 1e-30 {
			nr, ni = 0, 0
		}
		m.re[k], m.im[k] = nr, ni
		sum += nr
	}
	s := sum * membraneGain
	if sn := v.cfg.Snare; sn > 0 {
		// the wires buzz as loud as the skin is moving
		m.snare = max(math.Abs(sum), m.snare*0.9993)
		v.rng = v.rng*1664525 + 1013904223
		n := float64(int32(v.rng)) / 2147483648
		hp := n - m.prev
		m.prev = n
		s += min(sn, 1) * 0.9 * math.Sqrt(m.snare) * 0.5 * hp
	}
	return s
}
