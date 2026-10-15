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
	InstWoodBlock                         // a slit block: a short Helmholtz cavity under the pitch, the walls' bar modes above
	InstTempleBlock                       // a bigger, rounder hollow block: a longer cavity, softer wall modes
	InstBottle                            // a tapped bottle: the Helmholtz resonance of the body, the neck's air column, the glass
)

// Instruments are the names of the instruments, by number
var Instruments = []string{"custom", "bar", "marimba", "vibraphone", "xylophone", "glockenspiel", "tine",
	"tubular bell", "church bell", "handbell", "plate", "gong", "cymbal", "singing bowl", "steel pan",
	"wood block", "temple block", "bottle"}

type shapeKind int

const (
	shFlat  shapeKind = iota // every mode equally
	shBar                    // free-free beam
	shTine                   // clamped-free beam
	shPlate                  // J(kr) + C I(kr) times cos(m theta)
	shDome                   // J0(j r)
	shRing                   // a shell: the higher the nodal diameters, the nearer the lip
	shArch                   // an undercut bar: the mode shapes of the solved beam (see undercut.go)
)

// partial is a mode of an instrument
type partial struct {
	ratio, amp, decay float64
	kind              shapeKind
	n, m              int     // the number of the beam mode, the nodal diameters (a degenerate pair when above 0)
	k, c, norm        float64 // plate: the wavenumber, the edge constant, the scale
	beatScale         float64 // a degenerate pair splits this much of the instrument's split (0 is 1)
	arch              *archBar
}

// model is what an instrument is made of
type model struct {
	parts []partial
	q     float64  // 0-1: the share of the damping that grows with the square of the frequency (thermoelastic)
	split float64  // how far apart the two modes of a degenerate pair are, as a fraction of their pitch
	arch  *archBar // an undercut bar: the ratios and shapes follow the undercut
}

var (
	modelOnce sync.Once
	models    [InstBottle + 1]*model
)

func modelOf(ins Instrument) *model {
	modelOnce.Do(buildModels)
	if ins <= Custom || int(ins) >= len(models) {
		return nil
	}
	if ins == InstMarimbaBar || ins == InstVibraphoneBar || ins == InstXylophoneBar {
		archOnce[ins-InstMarimbaBar].Do(func() { models[ins] = buildArchModel(ins) })
	}
	return models[ins]
}

var archOnce [3]sync.Once

// buildArchModel solves and tunes the arch of an undercut bar (a few tens of milliseconds, the first
// time one is played). A marimba and a vibraphone are tuned 1 : 4 : 10, with the arch shaped as a cut
// with steep shoulders (the real bars are cut like that); the vibraphone takes the deeper, wider one
// of the two arches that reach it, and rings far longer. A xylophone is tuned 1 : 3 : 6.
func buildArchModel(ins Instrument) *model {
	var a *archBar
	var amps []float64
	var q float64
	switch ins {
	case InstMarimbaBar:
		a, amps, q = buildArch(4, 10, 0.25, 0.65, 0.22), []float64{1, 0.5, 0.25, 0.1}, 0.1
	case InstVibraphoneBar:
		a, amps, q = buildArch(4, 10, 0.25, 0.8, 0.33), []float64{1, 0.35, 0.12, 0.04}, 0.03
	default:
		a, amps, q = buildArch(3, 6, 1, 0.5, 0.25), []float64{1, 0.6, 0.3, 0.12}, 0.12
	}
	md := &model{q: q, arch: a}
	for i, amp := range amps {
		md.parts = append(md.parts, partial{ratio: a.ratio(i, 1), amp: amp, decay: 1, kind: shArch, n: i + 1, arch: a})
	}
	return md
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
	case shArch:
		return p.arch.shape(p.n-1, pos, 1)
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
	// A church bell. The ratios are those measured on cast bells (Rossing, The Science of Percussion
	// Instruments, and Perrin et al. on English and Dutch bells): the hum an octave below the prime,
	// the prime, the minor tierce (a minor third over the prime, measured 1.18 to 1.20), the quint
	// (a fifth), the nominal (an octave over the prime, the pitch the ear calls the bell's), then the
	// major tierce (deciem) near 2.5, the undecim near 2.67, the twelfth near 3 and the upper
	// partials near 4.2 and 5.4; the strike note the ear hears is the nominal with the prime. Each is
	// a degenerate pair of modes (n = 2, 2, 3, 3, 4 ... nodal diameters) split a little, so they
	// beat: the hum and prime slowly (about 0.5 Hz), the tierce and nominal faster. The hum rings
	// longest, the high partials die within a few seconds. Amplitudes and decays are approximate.
	bell := ring(0.08, 0.003,
		[]float64{0.5, 1, 1.2, 1.5, 2, 2.52, 2.67, 3, 4.17, 5.43},
		[]float64{0.55, 0.8, 0.6, 0.4, 1, 0.4, 0.35, 0.3, 0.2, 0.12},
		[]float64{1.8, 1.1, 0.8, 0.7, 0.6, 0.45, 0.4, 0.35, 0.28, 0.2},
		[]int{2, 2, 3, 3, 4, 4, 5, 5, 6, 7})
	for i, sc := range []float64{0.5, 0.7, 1.6, 1, 1.3, 1.5, 1.2, 1.2, 1.4, 1.4} {
		bell.parts[i].beatScale = sc
	}
	models[InstChurchBell] = bell
	// A handbell: thin brass, tuned so its first partials are close to the harmonic series (1 : 2 : 3,
	// where a church bell has its 0.5 : 1 : 1.2 : 1.5 : 2); the upper ones stay inharmonic near 4.15
	// and 5.4 and die within a second or so. Approximate values after Rossing's handbell spectra.
	models[InstHandbell] = ring(0.1, 0.002,
		[]float64{1, 2.0, 3.0, 4.15, 5.4, 6.9, 8.5},
		[]float64{1, 0.8, 0.5, 0.35, 0.2, 0.1, 0.05},
		[]float64{1, 0.8, 0.6, 0.45, 0.35, 0.25, 0.18},
		[]int{2, 3, 4, 5, 6, 7, 8})
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
	disc := func(n int, q, split float64, amp, decay, wobble func(i int) float64) *model {
		md := &model{q: q, split: split}
		for i, z := range zs[:n] {
			p := partial{ratio: math.Pow(z.k/zs[0].k, 2) * (1 + wobble(i)), amp: amp(i), decay: decay(i), kind: shPlate, m: z.m, k: z.k}
			freeEdge(&p, false)
			md.parts = append(md.parts, p)
		}
		return md
	}
	// A tam-tam. The free-edge disc has its modes at the squares of the zeros of the derivatives of the
	// Bessel functions, a first approximation to a flat tam-tam (Rossing and Fletcher measured the
	// real ones to be denser and a little irregular above the first dozen modes: the hammered dish
	// and the centre boss shift them a few per cent, which the wobble here stands for). The low
	// modes ring for ten seconds and more, the upper ones die in a second or two; the energy that
	// moves between them is the nonlinearity (Config.Nonlinearity). Cymbals are the same, dense
	// (36 modes), with a stronger irregularity and every mode dying quickly.
	models[InstGong] = disc(22, 0.12, 0.002,
		func(i int) float64 { return math.Max(0.25, 1-0.04*float64(i)) },
		func(i int) float64 { return math.Max(0.4, 1-0.03*float64(i)) },
		func(i int) float64 {
			if i < 6 {
				return 0
			}
			return 0.015 * math.Sin(float64(i)*5.1+0.7)
		})
	models[InstCymbal] = disc(36, 0.3, 0.004,
		func(i int) float64 { return 0.5 + 0.5*math.Sin(float64(i)*2.1+1) },
		func(i int) float64 { return 1 },
		func(i int) float64 {
			if i < 4 {
				return 0
			}
			return 0.03 * math.Sin(float64(i)*3.7+0.3)
		})

	pan := &model{q: 0.1}
	zero0 := []float64{2.404826, 5.520078, 8.653728, 11.791534}
	for i, r := range []float64{1, 2.005, 2.99, 4.03} {
		pan.parts = append(pan.parts, partial{ratio: r, amp: []float64{1, 0.65, 0.35, 0.15}[i],
			decay: []float64{1, 0.8, 0.6, 0.4}[i], kind: shDome, n: i + 1, k: zero0[i]})
	}
	models[InstSteelPan] = pan

	// A wood block: a slit in a hollow block. The air in the box is a Helmholtz resonator, a short
	// strongly damped tone that gives the block its pitch (ratio 1, the same wherever it is struck);
	// the walls and the bar between the slit and the edge ring above it as bars do, a little
	// inharmonic. A temple block is bigger and rounder: its cavity rings longer, its walls speak
	// lower and softer. A bottle: the body's Helmholtz tone, the neck's air column (an odd multiple
	// of a quarter wave, near 2.8 and 5.1 times the body), and the thin glass, high and quiet.
	cavity := func(q float64, ratios, amps, decays []float64) *model {
		md := &model{q: q}
		md.parts = append(md.parts, partial{ratio: 1, amp: 1, decay: 1.5, kind: shFlat})
		for i := range ratios {
			md.parts = append(md.parts, partial{ratio: ratios[i], amp: amps[i], decay: decays[i], kind: shBar, n: i + 1})
		}
		return md
	}
	models[InstWoodBlock] = cavity(0.2, []float64{1.62, 2.75, 4.2, 6.3}, []float64{0.5, 0.3, 0.15, 0.08}, []float64{0.8, 0.6, 0.5, 0.4})
	models[InstTempleBlock] = cavity(0.1, []float64{1.41, 2.2, 3.4, 5.1}, []float64{0.45, 0.3, 0.18, 0.1}, []float64{1.2, 0.9, 0.7, 0.5})
	models[InstTempleBlock].parts[0].decay = 2.4
	bottle := &model{q: 0.05}
	for _, e := range [][3]float64{{1, 1, 1.6}, {2.8, 0.18, 0.5}, {5.1, 0.08, 0.35}, {7.4, 0.1, 0.8}} {
		bottle.parts = append(bottle.parts, partial{ratio: e[0], amp: e[1], decay: e[2], kind: shFlat})
	}
	models[InstBottle] = bottle
}

// instrument adds the modes of an instrument for the strike: their pitch (stretched by the
// inharmonicity), how long each rings (the damping of the instrument, thermoelastic growing with the
// square of the frequency, and the high damping on top), how much each is woken by the strike
// position, and the splitting of the degenerate pairs
func (v *Voice) instrument(md *model, pos, beat, highDamping, inharm, decay, tube float64, add func(f, amp, t60 float64, tube bool)) {
	cfg := &v.cfg
	r0 := md.parts[0].ratio
	undercut := 1.0
	if cfg.Undercut > 0 {
		undercut = math.Min(cfg.Undercut, 1)
	}
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
		ratio := p.ratio
		if p.arch != nil {
			ratio = p.arch.ratio(p.n-1, undercut)
		}
		f := v.freq * ratio
		if inharm > 0 {
			x := ratio / r0
			f = v.freq * r0 * x * math.Sqrt(math.Max(0.1, 1+0.02*inharm*(x*x-1)))
		}
		f *= detune
		t60 := decay * p.decay / ((1 - md.q) + md.q*ratio*ratio) * math.Pow(r0/ratio, highDamping)
		if k == 0 && tube > 0 {
			t60 *= 1 + 0.3*tube
		}
		amp := p.amp * p.shape(pos)
		if p.arch != nil {
			amp = p.amp * p.arch.shape(p.n-1, pos, undercut)
		}
		switch {
		case p.m > 0 && spread > 0 && md.split > 0:
			s := md.split * spread * splitScale
			if p.beatScale > 0 {
				s *= p.beatScale
			}
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
