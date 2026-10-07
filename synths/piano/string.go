package piano

import (
	"math"
	"math/cmplx"
	"sync"
)

// dispersionStages is the number of allpass filters that make a string stiff
const dispersionStages = 4

// highDamping sets how much faster the highs of a string die: 1/T60(f) = 1/T60 + highDamping f²
const highDamping = 3e-8

// str is a string: a delay line closed by a loss filter, a dispersion allpass chain and a tuning
// allpass. The strike is played into it, what comes back around the loop goes to the bridge
type str struct {
	params stringParams
	buf    []float64
	pos    int
	loss   float64                   // loss filter state
	dx, dy [dispersionStages]float64 // dispersion allpass states
	tx, ty float64                   // tuning allpass state
	in     float64                   // how much of the strike reaches this string
	y      float64                   // the last sample around the loop
	delay  int                       // samples the strike reaches this string late
	shift  float64                   // samples the loop is shorter for now (a string played hard settles)
	hist   []float64                 // the loop signal, for the irregular losses
	hpos   int
	damper float64 // gain per loop the damper takes away, 1 is lifted
	damped float64 // that gain with the damper down
}

// stringParams are the filters of a string at its pitch
type stringParams struct {
	n      int     // delay line length in samples
	g, a   float64 // loss filter: dc gain per loop and pole
	stages int     // dispersion allpasses
	ad     float64 // their coefficient
	at     float64 // tuning allpass coefficient
	damped float64 // loop gain the damper adds when down
	m      int     // the irregular losses: a comb of m samples
	eps    float64 // and its depth
}

func newString(p stringParams) *str {
	s := &str{}
	s.reset(p)
	return s
}

// reset retunes the string and silences it (a new strike of the key: the damper had stopped it,
// or the voice was free)
func (s *str) reset(p stringParams) {
	s.params = p
	if cap(s.buf) < p.n {
		s.buf = make([]float64, p.n)
	}
	s.buf = s.buf[:p.n]
	clear(s.buf)
	s.pos = 0
	s.loss = 0
	s.dx, s.dy = [dispersionStages]float64{}, [dispersionStages]float64{}
	s.tx, s.ty = 0, 0
	s.in = 1
	s.shift = 0
	if p.eps > 0 {
		if cap(s.hist) < p.m+1 {
			s.hist = make([]float64, p.m+1)
		}
		s.hist = s.hist[:p.m+1]
		clear(s.hist)
		s.hpos = 0
	}
	s.damper = 1
	s.damped = p.damped
}

// tick plays a sample of the strike into the string and returns what reaches the bridge
func (s *str) tick(x float64) float64 {
	y := s.next()
	s.push(x*s.in + y)
	return y
}

// next returns what comes around the loop this sample (it reaches the bridge); push must follow
func (s *str) next() float64 {
	p := &s.params
	u := s.buf[s.pos]
	if s.shift > 1e-4 {
		// shorter for now: read the loop a fraction of a sample later (linear, which never adds
		// gain, so the loop stays stable)
		n := p.n
		at := float64(s.pos) + math.Min(s.shift, float64(n-2))
		i := int(at)
		f := at - float64(i)
		u = s.buf[i%n] + (s.buf[(i+1)%n]-s.buf[i%n])*f
	}

	// the irregular losses: some partials lose more than others (the bridge takes more at some
	// frequencies), a shallow comb
	if p.eps > 0 {
		m := len(s.hist)
		old := s.hist[(s.hpos+1)%m] // params.m samples ago
		s.hist[s.hpos] = u
		s.hpos = (s.hpos + 1) % m
		u = (1-p.eps)*u + p.eps*old
	}

	// losses: a one pole lowpass, the highs lose more every loop
	s.loss = p.g*(1+p.a)*u - p.a*s.loss
	u = s.loss

	// stiffness: allpasses delay the lows more than the highs, the partials go sharp
	for k := range p.stages {
		y := p.ad*u + s.dx[k] - p.ad*s.dy[k]
		s.dx[k], s.dy[k] = u, y
		u = y
	}

	// the fraction of a sample that tunes the loop
	y := p.at*u + s.tx - p.at*s.ty
	s.tx, s.ty = u, y

	return y * s.damper
}

// push writes the sample that goes around the loop again
func (s *str) push(v float64) {
	s.buf[s.pos] = v
	s.pos++
	if s.pos == s.params.n {
		s.pos = 0
	}
}

// stringDesign describes the string to design
type stringDesign struct {
	sr, f0     float64
	t60        float64 // seconds the fundamental takes to fall 60 dB
	damping    float64 // highDamping times the setting
	damperTime float64 // seconds to fall 60 dB with the damper down
	stages     int     // dispersion allpasses (see dispersion)
	m          int     // the irregular losses (see irregular)
	eps        float64
	ad         float64 // and their coefficient
}

// designString finds the filters of a string: the loss filter for the ring time and the delay
// length and tuning allpass that put the fundamental exactly on pitch with the dispersion
func designString(d stringDesign) stringParams {
	var (
		p      stringParams
		period = d.sr / d.f0 // loop length in samples
		w0     = 2 * math.Pi * d.f0 / d.sr
	)

	// loss filter: the loop gain at the fundamental for the ring time, and a pole that makes a
	// higher frequency lose as much as the damping asks
	// the fundamental rings for t60, the partials above it shorter: 1/T60(f) = 1/t60 + damping
	// (f² - f0²), so in the treble, where the fundamental itself is high, only the partials lose
	// more
	g0 := math.Pow(10, -3/(d.f0*d.t60))
	p.g = g0
	fh := math.Min(math.Max(6*d.f0, 5000), 0.4*d.sr) // fitted high: a one pole follows f² below it
	if fh > d.f0*1.5 && d.damping > 0 {
		gh := math.Pow(10, -3/(d.f0/(1/d.t60+d.damping*(fh*fh-d.f0*d.f0))))
		p.a = lossPole(gh/g0, 2*math.Pi*fh/d.sr)
		// the pole takes a little of the fundamental too: give it back, as long as the loop stays
		// below unity at dc (otherwise the highs get a little less loss)
		w0 := 2 * math.Pi * d.f0 / d.sr
		for range 20 {
			at := cmplx.Abs(complex(1+p.a, 0) / (1 + complex(p.a, 0)*cmplx.Exp(complex(0, -w0))))
			if g := g0 / at; g <= 0.99999 {
				p.g = g
				break
			}
			p.a *= 0.8
		}
	}

	// the damper: the extra loss per loop that rings the string out in damperTime
	p.damped = 1
	if d.damperTime > 0 {
		p.damped = math.Pow(10, -3/(d.f0*d.damperTime))
	}

	p.stages, p.ad = d.stages, d.ad
	p.m, p.eps = d.m, d.eps
	if float64(p.stages)*allpassDelay(p.ad, w0) > 0.5*period { // a stage count from another key
		p.stages, p.ad = 0, 0
	}
	p.n, p.at = tune(period, w0, p, p.ad)
	return p
}

// dispersionKey is a table of dispersion filters for every key
type dispersionKey struct {
	sr, stiffness float64
}

type dispersionFilter struct {
	stages int
	ad     float64
}

var dispersionTables sync.Map // dispersionKey -> *[128]dispersionFilter

// dispersion returns the dispersion filters of every key for a stiffness (scaling the
// inharmonicity of a grand) at a sample rate. Fitting them takes a while, so they are fitted once
// per stiffness and sample rate (when the voices are created) and kept
func dispersion(sr, stiffness float64) *[128]dispersionFilter {
	key := dispersionKey{sr, stiffness}
	if t, ok := dispersionTables.Load(key); ok {
		return t.(*[128]dispersionFilter)
	}
	var t [128]dispersionFilter
	for k := range t {
		pitch := math.Max(21, math.Min(float64(k), 108))
		t[k].stages, t[k].ad = fitDispersion(sr, 440*math.Exp2((pitch-69)/12), inharmonicity(pitch)*stiffness, dispersionStages)
	}
	stored, _ := dispersionTables.LoadOrStore(key, &t)
	return stored.(*[128]dispersionFilter)
}

// fitDispersion fits the coefficient of a chain of allpasses so the partials 2-10 of a string
// lie where the inharmonicity puts them, f0 k sqrt(1 + B k²) / sqrt(1 + B) (least squares in
// cents, the low partials weigh more)
func fitDispersion(sr, f0, b float64, maxStages int) (int, float64) {
	period := sr / f0
	w0 := 2 * math.Pi * f0 / sr
	stages := maxStages
	for stages > 0 && float64(stages) > 0.4*(period-3) {
		stages--
	}
	top := math.Min(10, math.Floor(0.35*period))
	if b <= 0 || stages == 0 || top < 2 {
		return 0, 0
	}

	// the most negative coefficient keeps the allpasses within part of the loop
	q := math.Max(1, 0.4*(period-3)/float64(stages))
	aMin := math.Max(-0.95, (1-q)/(1+q))

	p := stringParams{g: 1, stages: stages}
	cost := func(a float64) float64 {
		p.ad = a
		p.n, p.at = tune(period, w0, p, a)
		e := 0.0
		for k := 2.0; k <= top; k++ {
			want := 600 * math.Log2((1+b*k*k)/(1+b))
			got := 1200 * math.Log2(partial(p, k, k*w0*math.Sqrt((1+b*k*k)/(1+b)))/(k*w0))
			e += (got - want) * (got - want) / k
		}
		return e
	}

	// golden section search
	const phi = 0.6180339887498949
	lo, hi := aMin, 0.0
	x1, x2 := hi-phi*(hi-lo), lo+phi*(hi-lo)
	f1, f2 := cost(x1), cost(x2)
	for range 30 {
		if f1 < f2 {
			hi, x2, f2 = x2, x1, f1
			x1 = hi - phi*(hi-lo)
			f1 = cost(x1)
		} else {
			lo, x1, f1 = x1, x2, f2
			x2 = lo + phi*(hi-lo)
			f2 = cost(x2)
		}
	}
	return stages, (lo + hi) / 2
}

// lossPole returns the pole of a one pole lowpass whose gain at w is ratio times its gain at dc
func lossPole(ratio, w float64) float64 {
	if ratio >= 1 {
		return 0
	}
	r2 := ratio * ratio
	c := math.Cos(w)
	// (r² - 1) a² + 2 (r² c - 1) a + (r² - 1) = 0, the root inside the unit circle
	A, B := r2-1, 2*(r2*c-1)
	disc := math.Sqrt(math.Max(0, B*B-4*A*A))
	for _, a := range []float64{(-B + disc) / (2 * A), (-B - disc) / (2 * A)} {
		if a <= 0 && a > -1 {
			return a
		}
	}
	return 0
}

// phaseDelay returns the phase delay in samples at w of (b0 + b1 z^-1) / (1 + a1 z^-1)
func phaseDelay(b0, b1, a1, w float64) float64 {
	z := cmplx.Exp(complex(0, -w))
	h := (complex(b0, 0) + complex(b1, 0)*z) / (1 + complex(a1, 0)*z)
	ph := cmplx.Phase(h)
	if ph > 1e-12 && b1 != 0 && a1 != 0 && math.Abs(b0) < 1 {
		ph -= 2 * math.Pi // an allpass lags between 0 and pi
	}
	return -ph / w
}

func allpassDelay(a, w float64) float64 { return phaseDelay(a, 1, a, w) }

// filterDelay returns the phase delay at w of the loss filter and the dispersion allpasses
func filterDelay(p stringParams, ad, w float64) float64 {
	d := phaseDelay(p.g*(1+p.a), 0, p.a, w)
	if p.eps > 0 {
		h := complex(1-p.eps, 0) + complex(p.eps, 0)*cmplx.Exp(complex(0, -w*float64(p.m)))
		d += -cmplx.Phase(h) / w
	}
	if p.stages > 0 {
		d += float64(p.stages) * allpassDelay(ad, w)
	}
	return d
}

// tune returns the delay line length and the tuning allpass coefficient that make the loop
// period samples long at w0
func tune(period, w0 float64, p stringParams, ad float64) (int, float64) {
	rest := period - filterDelay(p, ad, w0)
	n := int(math.Floor(rest - 0.5))
	n = max(n, 1)
	d := rest - float64(n) // 0.5-1.5, where a first order allpass is smooth and accurate
	at := (1 - d) / (1 + d)
	for range 3 {
		d += (rest - float64(n)) - allpassDelay(at, w0)
		at = (1 - d) / (1 + d)
	}
	return n, at
}

// partial returns the frequency (radians per sample) of partial k of a string, where the phase
// around the loop is k turns, searched from a guess
func partial(p stringParams, k, guess float64) float64 {
	phase := func(w float64) float64 {
		return w*(float64(p.n)+filterDelay(p, p.ad, w)+allpassDelay(p.at, w)) - 2*math.Pi*k
	}
	w := math.Max(1e-4, math.Min(guess, math.Pi-1e-3))
	for range 12 {
		const h = 1e-7
		f := phase(w)
		step := f / ((phase(w+h) - f) / h)
		w = math.Max(1e-6, math.Min(w-step, math.Pi-1e-6))
		if math.Abs(step) < 1e-12 {
			break
		}
	}
	return w
}

// numStrings returns the strings of a key: one in the low bass, two up to the tenor, then three
func numStrings(pitch float64, most int) int {
	if most <= 0 {
		most = 3
	}
	n := 3
	switch {
	case pitch < 29:
		n = 1
	case pitch < 45:
		n = 2
	}
	return min(n, most)
}

// unisonOffsets returns how far (cents, for a unison of 1) the strings of a key are from its
// pitch: spread around it, a little different for every key like a real tuning
func unisonOffsets(pitch float64, n int) []float64 {
	jitter := func(i int) float64 { // the same for a key every time
		x := math.Sin(pitch*12.9898+float64(i)*78.233) * 43758.5453
		return (x - math.Floor(x) - 0.5) * 0.4
	}
	switch n {
	case 1:
		return []float64{0}
	case 2:
		return []float64{-0.5 + jitter(0), 0.5 + jitter(1)}
	default:
		return []float64{-0.7 + jitter(0), jitter(1), 0.7 + jitter(2)}
	}
}

// interpolate reads a table of (pitch, value) pairs, linearly in the log of the value
func interpolate(table [][2]float64, pitch float64) float64 {
	if pitch <= table[0][0] {
		return table[0][1]
	}
	for i := 1; i < len(table); i++ {
		if pitch <= table[i][0] {
			t := (pitch - table[i-1][0]) / (table[i][0] - table[i-1][0])
			return math.Exp(math.Log(table[i-1][1]) + t*(math.Log(table[i][1])-math.Log(table[i-1][1])))
		}
	}
	return table[len(table)-1][1]
}

// interpolateLinear looks up a table by key like interpolate, linearly (for values that can be
// negative)
func interpolateLinear(table [][2]float64, pitch float64) float64 {
	if pitch <= table[0][0] {
		return table[0][1]
	}
	for i := 1; i < len(table); i++ {
		if pitch <= table[i][0] {
			t := (pitch - table[i-1][0]) / (table[i][0] - table[i-1][0])
			return table[i-1][1] + t*(table[i][1]-table[i-1][1])
		}
	}
	return table[len(table)-1][1]
}

// inharmonicity of a grand: wound bass strings are fairly harmonic, the short thick strings at
// the top very stiff
var inharmonicityTable = [][2]float64{{21, 3e-4}, {33, 1.5e-4}, {45, 1.5e-4}, {57, 3e-4}, {69, 7e-4}, {81, 2e-3}, {93, 6e-3}, {108, 2e-2}}

func inharmonicity(pitch float64) float64 { return interpolate(inharmonicityTable, pitch) }

// ringTable is how long the fundamental of a string of a key takes to fall 60 dB (seconds) by its
// own losses: the long second stage of a note (the first, fast one is the bridge, see
// couplingTable), fitted to a recorded concert grand
var ringTable = [][2]float64{{21, 40}, {36, 45}, {48, 40}, {60, 30}, {72, 22}, {84, 8}, {96, 4}, {108, 1.5}}

// ringTime is how long the fundamental of a key takes to fall 60 dB
func ringTime(pitch float64) float64 { return interpolate(ringTable, pitch) }

// irregular returns the comb of the irregular losses of a string: a length that is a part of the
// loop (another for every key and string, so the pattern over the partials differs), deep enough
// that the partials it takes most from lose up to irregularDB dB per second more, times amount
func irregular(period, f0 float64, key, string int, amount float64) (int, float64) {
	if amount <= 0 {
		return 0, 0
	}
	r := 0.31 + 0.19*(keyRandom(key, 7+float64(string))+1)
	m := max(2, int(math.Round(period*r)))
	// a loss of 2 eps per loop at most, f0 loops a second
	return m, amount * irregularDB / (2 * 8.686 * f0)
}

// irregularDB is how much faster (dB per second) the partials decay that lose most
const irregularDB = 3.0
