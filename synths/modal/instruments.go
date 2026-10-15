package modal

import (
	"math"
	"sort"
	"sync"
)

// Instrument is a physical model of a struck body: its modes' pitch ratios, how each decays and how
// the strike position wakes it (the mode shapes), taken from the beam, plate and shell theory and
// from measured instruments. Custom (the default) rings in the Modes of the Config instead.
//
// The note's pitch is the ratio 1 of the instrument: the bar's fundamental, the strike tone of a
// tubular bell, the prime of a bell, the lowest mode of a plate. Position (0.01 to 1) is where the
// mallet hits: along a bar from its end, from the clamp of a tine, from the centre to the rim of a
// plate, a gong or a steel pan's note area, from the crown down to the lip of a bell or a bowl.
type Instrument int

const (
	Custom              Instrument = iota // the Modes of the Config
	InstBar                               // a uniform bar free at both ends: 1 : 2.756 : 5.404 : 8.933 : 13.34 : 18.64
	InstMarimbaBar                        // an undercut rosewood bar tuned 1 : 4 : 10, over a resonator tube
	InstVibraphoneBar                     // an undercut aluminum bar tuned 1 : 4 : 10, over a resonator tube
	InstXylophoneBar                      // an undercut bar tuned 1 : 3 : 6
	InstGlockenspielBar                   // a steel bar, the uniform bar's partials, ringing long
	InstTineBar                           // a clamped tine (kalimba, music box): 1 : 6.267 : 17.55 : 34.4
	InstTubularBell                       // a chime: the modes 4 to 8 at 2 : 3 : 4 : 5 : 6 over a strike tone with nothing sounding at it
	InstChurchBell                        // a cast bell: hum, prime, tierce, quint, nominal ..., each a pair
	InstHandbell                          // a thin tuned handbell, close to the harmonic series
	InstPlate                             // a circular plate with a free edge (Kirchhoff modes, Bessel shapes)
	InstGong                              // a tam-tam: the modes of a disc as the zeros of the Bessel derivatives
	InstCymbal                            // the same, dense: a wash of modes
	InstSingingBowl                       // a bowl: the bending modes of a shell, n(n^2-1)/sqrt(n^2+1), each a pair
	InstSteelPan                          // a pan's note: fundamental, octave, twelfth and double octave, shaped like a drum head
)

// Instruments are the names of the instruments, by number
var Instruments = []string{"custom", "bar", "marimba", "vibraphone", "xylophone", "glockenspiel", "tine",
	"tubular bell", "church bell", "handbell", "plate", "gong", "cymbal", "singing bowl", "steel pan"}

type shapeKind int

const (
	shFlat  shapeKind = iota // every mode equally
	shBar                    // free-free beam
	shTine                   // clamped-free beam
	shPlate                  // J(kr) + C I(kr) times cos(m theta)
	shDome                   // J0(j r)
	shRing                   // a shell: the higher the nodal diameters, the nearer the lip
)

// partial is a mode of an instrument
type partial struct {
	ratio, amp, decay float64
	kind              shapeKind
	n, m              int     // the number of the beam mode, the nodal diameters (a degenerate pair when above 0)
	k, c, norm        float64 // plate: the wavenumber, the edge constant, the scale
}

// model is what an instrument is made of
type model struct {
	parts []partial
	q     float64 // 0-1: the share of the damping that grows with the square of the frequency (thermoelastic)
	split float64 // how far apart the two modes of a degenerate pair are, as a fraction of their pitch
}

var (
	modelOnce sync.Once
	models    [InstSteelPan + 1]*model
)

func modelOf(ins Instrument) *model {
	modelOnce.Do(buildModels)
	if ins <= Custom || int(ins) >= len(models) {
		return nil
	}
	return models[ins]
}

// beam is the roots of cos x cosh x = 1 (free-free) or -1 (clamped-free) with the constants of the
// mode shapes
type beam struct{ x, sigma float64 }

func beams(free bool, n int) []beam {
	out := make([]beam, n)
	for i := range out {
		var x, s float64
		if free {
			x = (float64(i+1) + 0.5) * math.Pi
			for range 40 {
				f := math.Cos(x)*math.Cosh(x) - 1
				d := -math.Sin(x)*math.Cosh(x) + math.Cos(x)*math.Sinh(x)
				x -= f / d
			}
			s = (math.Cosh(x) - math.Cos(x)) / (math.Sinh(x) - math.Sin(x))
		} else {
			x = (float64(i+1) - 0.5) * math.Pi
			for range 40 {
				f := math.Cos(x)*math.Cosh(x) + 1
				d := -math.Sin(x)*math.Cosh(x) + math.Cos(x)*math.Sinh(x)
				x -= f / d
			}
			s = (math.Cosh(x) + math.Cos(x)) / (math.Sinh(x) + math.Sin(x))
		}
		out[i] = beam{x, s}
	}
	return out
}

var freeBeams, clampedBeams []beam

// BarRatio is the frequency of mode n (from 1) of a uniform free-free bar over its fundamental
func BarRatio(n int) float64 {
	modelOnce.Do(buildModels)
	return math.Pow(freeBeams[n-1].x/freeBeams[0].x, 2)
}

// BarShape is the amplitude of mode n (from 1) of a free-free bar at x (0 to 1 along it), the mode
// normalised to a mean square of 1 (2 at the ends)
func BarShape(n int, x float64) float64 {
	modelOnce.Do(buildModels)
	b := freeBeams[n-1]
	t := b.x * x
	return math.Cosh(t) + math.Cos(t) - b.sigma*(math.Sinh(t)+math.Sin(t))
}

func tineShape(n int, x float64) float64 {
	b := clampedBeams[n-1]
	t := b.x * x
	return math.Cosh(t) - math.Cos(t) - b.sigma*(math.Sinh(t)-math.Sin(t))
}

// besselI is the modified Bessel function of order m
func besselI(m int, x float64) float64 {
	h := x / 2
	term := 1.0
	for i := 1; i <= m; i++ {
		term *= h / float64(i)
	}
	sum := term
	for j := 1; j < 80; j++ {
		term *= h * h / (float64(j) * float64(j+m))
		sum += term
		if term < 1e-17*sum {
			break
		}
	}
	return sum
}

// plateRadial is the radial shape of a plate mode J_m(kr) + c I_m(kr)
func plateRadial(m int, k, c, r float64) float64 {
	return math.Jn(m, k*r) + c*besselI(m, k*r)
}

// freeEdge works out the constant c of a plate mode so the bending moment at the free edge is nil
// (Poisson's ratio 0.33), and scales it to a mean square of 1 over the disc
func freeEdge(p *partial, free bool) {
	const nu, h = 0.33, 1e-4
	if free {
		moment := func(f func(float64) float64) float64 {
			d1 := (f(1+h) - f(1-h)) / (2 * h)
			d2 := (f(1+h) - 2*f(1) + f(1-h)) / (h * h)
			return d2 + nu*(d1-float64(p.m*p.m)*f(1))
		}
		a := moment(func(r float64) float64 { return math.Jn(p.m, p.k*r) })
		b := moment(func(r float64) float64 { return besselI(p.m, p.k*r) })
		p.c = -a / b
	}
	var e float64
	const steps = 400
	for i := range steps {
		r := (float64(i) + 0.5) / steps
		v := plateRadial(p.m, p.k, p.c, r)
		e += v * v * r / steps
	}
	p.norm = math.Sqrt(2 * e)
}

// shape is how much the strike at pos wakes the mode (a mean square of 1 over the body)
func (p *partial) shape(pos float64) float64 {
	pos = math.Max(0.01, math.Min(pos, 1))
	switch p.kind {
	case shBar:
		return math.Abs(BarShape(p.n, pos))
	case shTine:
		return math.Abs(tineShape(p.n, pos))
	case shPlate:
		return math.Abs(plateRadial(p.m, p.k, p.c, pos)) / p.norm
	case shDome:
		return math.Abs(math.J0(p.k*pos) / math.J1(p.k))
	case shRing:
		return math.Max(0.03, math.Pow(pos, float64(p.m-1)/2))
	}
	return 1
}

// zerosOfDerivative are the first zeros of the derivative of the Bessel function of order m
func zerosOfDerivative(m int, upTo float64) []float64 {
	f := func(x float64) float64 {
		if m == 0 {
			return -math.J1(x)
		}
		return math.Jn(m-1, x) - math.Jn(m+1, x)
	}
	var z []float64
	prev := f(0.5)
	for x := 0.5 + 0.01; x < upTo; x += 0.01 {
		cur := f(x)
		if (prev < 0) != (cur < 0) {
			lo, hi := x-0.01, x
			for range 50 {
				mid := (lo + hi) / 2
				if (f(mid) < 0) == (f(lo) < 0) {
					lo = mid
				} else {
					hi = mid
				}
			}
			z = append(z, (lo+hi)/2)
		}
		prev = cur
	}
	return z
}

func buildModels() {
	freeBeams = beams(true, 10)
	clampedBeams = beams(false, 6)
	bar := func(idx []int, ratios, amps, decays []float64, q float64) *model {
		md := &model{q: q}
		for i, n := range idx {
			r := math.Pow(freeBeams[n-1].x/freeBeams[0].x, 2)
			if ratios != nil {
				r = ratios[i]
			}
			md.parts = append(md.parts, partial{ratio: r, amp: amps[i], decay: decays[i], kind: shBar, n: n})
		}
		return md
	}
	ones := []float64{1, 1, 1, 1, 1, 1, 1, 1}

	models[InstBar] = bar([]int{1, 2, 3, 4, 5, 6}, nil, []float64{1, 0.55, 0.3, 0.15, 0.08, 0.04}, ones, 0.1)
	models[InstMarimbaBar] = bar([]int{1, 2, 3, 4}, []float64{1, 4, 10, 19.5}, []float64{1, 0.5, 0.25, 0.1}, ones, 0.1)
	models[InstVibraphoneBar] = bar([]int{1, 2, 3, 4}, []float64{1, 4, 10, 19.9}, []float64{1, 0.35, 0.12, 0.04}, ones, 0.03)
	models[InstXylophoneBar] = bar([]int{1, 2, 3, 4}, []float64{1, 3, 6, 10.2}, []float64{1, 0.6, 0.3, 0.12}, ones, 0.12)
	models[InstGlockenspielBar] = bar([]int{1, 2, 3, 4, 5}, nil, []float64{1, 0.55, 0.3, 0.15, 0.07}, ones, 0.06)

	tine := &model{q: 0.05}
	for i, a := range []float64{1, 0.35, 0.15, 0.06} {
		tine.parts = append(tine.parts, partial{ratio: math.Pow(clampedBeams[i].x/clampedBeams[0].x, 2), amp: a, decay: 1, kind: shTine, n: i + 1})
	}
	models[InstTineBar] = tine

	// a chime: the beam's first three modes under the strike tone, the next five tuned 2 : 3 : 4 : 5 : 6
	chime := &model{q: 0.04}
	c := 2 / math.Pow(freeBeams[3].x/freeBeams[0].x, 2)
	chimeAmps := []float64{0.35, 0.5, 0.6, 1, 0.9, 0.7, 0.45, 0.3}
	chimeDecay := []float64{1.2, 1, 0.9, 0.9, 0.8, 0.7, 0.6, 0.5}
	for i := range 8 {
		r := c * math.Pow(freeBeams[i].x/freeBeams[0].x, 2)
		if i >= 3 {
			r = float64(i - 1)
		}
		chime.parts = append(chime.parts, partial{ratio: r, amp: chimeAmps[i], decay: chimeDecay[i], kind: shBar, n: i + 1})
	}
	models[InstTubularBell] = chime

	ring := func(q, split float64, ratios, amps, decays []float64, ms []int) *model {
		md := &model{q: q, split: split}
		for i := range ratios {
			md.parts = append(md.parts, partial{ratio: ratios[i], amp: amps[i], decay: decays[i], kind: shRing, m: ms[i]})
		}
		return md
	}
	// a church bell, the partials of a cast bell over the prime: hum, prime, tierce, quint, nominal ...
	models[InstChurchBell] = ring(0.08, 0.003,
		[]float64{0.5, 1, 1.183, 1.506, 2, 2.514, 2.662, 3.011, 4.166},
		[]float64{0.6, 0.8, 0.6, 0.4, 1, 0.4, 0.35, 0.3, 0.2},
		[]float64{1.6, 1, 0.8, 0.7, 0.6, 0.45, 0.4, 0.35, 0.3},
		[]int{2, 2, 3, 3, 4, 4, 5, 5, 6})
	models[InstHandbell] = ring(0.1, 0.002,
		[]float64{1, 2.01, 2.99, 4.13, 5.28, 6.92},
		[]float64{1, 0.8, 0.5, 0.35, 0.2, 0.1},
		[]float64{1, 0.8, 0.6, 0.45, 0.35, 0.25},
		[]int{2, 3, 4, 5, 6, 7})
	bowl := &model{q: 0.1, split: 0.004}
	for i, n := range []int{2, 3, 4, 5, 6} {
		nn := float64(n)
		f := nn * (nn*nn - 1) / math.Sqrt(nn*nn+1)
		bowl.parts = append(bowl.parts, partial{ratio: f / (2 * 3 / math.Sqrt(5)), amp: []float64{1, 0.6, 0.3, 0.15, 0.07}[i],
			decay: []float64{1, 0.8, 0.6, 0.45, 0.35}[i], kind: shRing, m: n})
	}
	models[InstSingingBowl] = bowl

	// a circular plate with a free edge (Leissa, Poisson's ratio 0.33): (diameters, circles) and lambda squared
	plate := &model{q: 0.15, split: 0.003}
	for i, e := range []struct {
		m int
		l float64
	}{{2, 5.253}, {0, 9.084}, {3, 12.23}, {1, 20.52}, {4, 21.60}, {5, 33.0}, {2, 35.25}, {0, 38.5}, {6, 47.8}, {3, 52.9}, {1, 59.8}, {4, 70.0}} {
		p := partial{ratio: e.l / 5.253, amp: math.Max(0.3, 1-0.06*float64(i)), decay: math.Max(0.6, 1-0.04*float64(i)), kind: shPlate, m: e.m, k: math.Sqrt(e.l)}
		freeEdge(&p, true)
		plate.parts = append(plate.parts, p)
	}
	models[InstPlate] = plate

	// a gong and a cymbal: the zeros of the derivatives of the Bessel functions, frequency with their square
	type zero struct {
		m int
		k float64
	}
	var zs []zero
	for m := 0; m <= 10; m++ {
		for i, k := range zerosOfDerivative(m, 45) {
			if m == 1 && i == 0 {
				continue
			}
			zs = append(zs, zero{m, k})
		}
	}
	sort.Slice(zs, func(i, j int) bool { return zs[i].k < zs[j].k })
	disc := func(n int, q, split float64, amp, decay func(i int) float64) *model {
		md := &model{q: q, split: split}
		for i, z := range zs[:n] {
			p := partial{ratio: math.Pow(z.k/zs[0].k, 2), amp: amp(i), decay: decay(i), kind: shPlate, m: z.m, k: z.k}
			freeEdge(&p, false)
			md.parts = append(md.parts, p)
		}
		return md
	}
	models[InstGong] = disc(16, 0.12, 0.002,
		func(i int) float64 { return math.Max(0.25, 1-0.05*float64(i)) },
		func(i int) float64 { return math.Max(0.4, 1-0.04*float64(i)) })
	models[InstCymbal] = disc(30, 0.3, 0.004,
		func(i int) float64 { return 0.5 + 0.5*math.Sin(float64(i)*2.1+1) },
		func(i int) float64 { return 1 })

	pan := &model{q: 0.1}
	zero0 := []float64{2.404826, 5.520078, 8.653728, 11.791534}
	for i, r := range []float64{1, 2.005, 2.99, 4.03} {
		pan.parts = append(pan.parts, partial{ratio: r, amp: []float64{1, 0.65, 0.35, 0.15}[i],
			decay: []float64{1, 0.8, 0.6, 0.4}[i], kind: shDome, n: i + 1, k: zero0[i]})
	}
	models[InstSteelPan] = pan
}

// instrument adds the modes of an instrument for the strike: their pitch (stretched by the
// inharmonicity), how long each rings (the damping of the instrument, thermoelastic growing with the
// square of the frequency, and the high damping on top), how much each is woken by the strike
// position, and the splitting of the degenerate pairs
func (v *Voice) instrument(md *model, pos, beat, highDamping, inharm, decay, tube float64, add func(f, amp, t60 float64, tube bool)) {
	cfg := &v.cfg
	r0 := md.parts[0].ratio
	// the pairs lie at an angle to the blow; with the drift every note turns them, splits them by a
	// little more or less and detunes everything a few cents (the temperature)
	theta, splitScale, detune := 0.35, 1.0, 1.0
	if drift := math.Max(0, math.Min(cfg.Drift, 1)); drift > 0 {
		rn := v.Rand()
		theta = rn.Float64() * math.Pi
		splitScale = 1 + (rn.Float64()*2-1)*0.5*drift
		detune = 1 + (rn.Float64()*2-1)*0.002*drift
	}
	spread := math.Max(0, cfg.Spread)
	for k := range md.parts {
		p := &md.parts[k]
		f := v.freq * p.ratio
		if inharm > 0 {
			x := p.ratio / r0
			f = v.freq * r0 * x * math.Sqrt(math.Max(0.1, 1+0.02*inharm*(x*x-1)))
		}
		f *= detune
		t60 := decay * p.decay / ((1 - md.q) + md.q*p.ratio*p.ratio) * math.Pow(r0/p.ratio, highDamping)
		if k == 0 && tube > 0 {
			t60 *= 1 + 0.3*tube
		}
		amp := p.amp * p.shape(pos)
		switch {
		case p.m > 0 && spread > 0 && md.split > 0:
			s := md.split * spread * splitScale
			phi := float64(p.m) * theta
			add(f*(1-s/2), amp*math.Abs(math.Cos(phi)), t60, false)
			add(f*(1+s/2), amp*math.Abs(math.Sin(phi)), t60, false)
		case beat > 0:
			add(f*(1-beat/2), amp/2, t60, false)
			add(f*(1+beat/2), amp/2, t60, false)
		default:
			add(f, amp, t60, false)
		}
	}
}
