package waveguide

import (
	"math"
	"sync"
)

// The parts of a real string: its stiffness (a chain of allpass filters that stretch the partials), its
// two polarizations, the shape of the pluck, the friction of the bow.

// maxSections is the most allpass sections of the stiffness: a wound bass string needs dozens
const maxSections = 40

// gentleSections is the most sections the fit of a gentle stiffness takes before the design of a
// stiff one is called for
const gentleSections = 12

// dispChain is the stiffness of a string: allpass sections that delay the low partials more than the
// high ones, which stretches the partials to f0 n sqrt(1+B n^2)
type dispChain struct {
	a    float64
	m    int
	x, y [maxSections]float64
}

func (d *dispChain) tick(in float64) float64 {
	for i := 0; i < d.m; i++ {
		out := d.a*in + d.x[i] - d.a*d.y[i]
		d.x[i], d.y[i] = in, out
		in = out
	}
	return in
}

func (d *dispChain) clear() {
	d.x = [maxSections]float64{}
	d.y = [maxSections]float64{}
}

// phaseDelay is the delay in samples of the allpass (a + z^-1)/(1 + a z^-1) at w radians a sample
func phaseDelay(a, w float64) float64 {
	if w < 1e-6 {
		return (1 - a) / (1 + a)
	}
	return 1 - 2/w*math.Atan(a*math.Sin(w)/(1+a*math.Cos(w)))
}

// groupDelay is the slope of the phase of the same allpass
func groupDelay(a, w float64) float64 {
	return (1 - a*a) / (1 + 2*a*math.Cos(w) + a*a)
}

// stiffnessAt is the B a string has at a pitch: the thick low strings are stiffer, B at 220 Hz is
// that of the setting
func stiffnessAt(b, f float64) float64 {
	return b * math.Max(0.3, math.Min(220/f, 6))
}

// stretch is the ratio of the n-th partial of a stiff string to the n-th harmonic (the first is kept
// at f0)
func stretch(b float64, n int) float64 {
	fn := float64(n)
	return math.Sqrt((1 + b*fn*fn) / (1 + b))
}

// dispersionCents are the cents the partials 2..top lie off where a string of stiffness b has them,
// for a chain of m sections of a, the rest of the loop (a pure delay) taking the first to f0: the
// loop's phase at each partial against 2 pi n, over the slope of the phase. It also returns the delay
// the chain adds at the fundamental
func dispersionCents(a float64, m int, b, f0, sr float64, top int) (worst, delay float64) {
	w1 := 2 * math.Pi * f0 / sr
	fm := float64(m)
	l0 := 2*math.Pi/w1 - fm*phaseDelay(a, w1) // the rest of the loop
	for n := 2; n <= top; n++ {
		wn := w1 * float64(n) * stretch(b, n)
		g := l0 + fm*groupDelay(a, wn)
		r := wn*(l0+fm*phaseDelay(a, wn)) - 2*math.Pi*float64(n)
		worst = math.Max(worst, math.Abs(r/(wn*g))*1200/math.Ln2)
	}
	return worst, fm * phaseDelay(a, w1)
}

// fitSections finds the sections and the one coefficient of the allpass chain that put the partials
// where measure says the least wrong (worst error in cents, and the delay the chain adds at the
// fundamental, which has to leave room in the loop), up to maxM sections; a stop below good ends the
// search
func fitSections(period float64, maxM int, good float64, measure func(a float64, m int) (float64, float64)) (a float64, m int, worst float64) {
	worst = math.Inf(1)
	for mm := 1; mm <= maxM; mm++ {
		bestA, bestE := 0.0, math.Inf(1)
		try := func(a float64) {
			if e, d := measure(a, mm); e < bestE && period-d > 4 {
				bestA, bestE = a, e
			}
		}
		for a := -0.97; a < 0; a += 0.01 {
			try(a)
		}
		for step := 0.005; step > 0.0002; step /= 2 {
			for _, a := range [2]float64{bestA - step, bestA + step} {
				if a < 0 && a >= -0.99 {
					try(a)
				}
			}
		}
		if bestE < worst-0.05 {
			worst, a, m = bestE, bestA, mm
		}
		if worst < good {
			break
		}
	}
	return
}

// dispersionCache keeps the designs: the same note again costs nothing
var (
	dispersionMu    sync.Mutex
	dispersionCache = map[[3]float64]dispDesign{}
)

type dispDesign struct {
	a        float64
	m        int
	delay, w float64
}

// fitDispersion finds the sections and the coefficient of the allpass chain that put the partials of
// a string of stiffness b at f0 where f0 n sqrt((1+B n^2)/(1+B)) is (the first stays at f0). It
// returns the delay the chain adds at the fundamental, to be taken off the loop, and the worst error
// in cents of the partials up to the 14th. A gentle stiffness takes a few sections; a stiff string (a
// wound bass, a piano string) a designed chain of many, searched against the phase the loop has at
// each partial
func fitDispersion(b, f0, sr float64) (a float64, m int, delay, worst float64) {
	top := min(14, int(sr/6/f0))
	if b <= 0 || top < 2 {
		return 0, 0, 0, 0
	}
	key := [3]float64{b, f0, sr}
	dispersionMu.Lock()
	d, ok := dispersionCache[key]
	dispersionMu.Unlock()
	if ok {
		return d.a, d.m, d.delay, d.w
	}
	period := sr / f0
	w1 := 2 * math.Pi * f0 / sr
	// gentle: the partials up to the 10th by their phase delay
	lowTop := min(10, top)
	a, m, _ = fitSections(period, gentleSections, 0, func(a float64, m int) (float64, float64) {
		fm := float64(m)
		l0 := period - fm*phaseDelay(a, w1)
		e := 0.0
		for n := 2; n <= lowTop; n++ {
			fn := float64(n)
			want := period / stretch(b, n)
			got := l0 + fm*phaseDelay(a, w1*fn*stretch(b, n))
			e = math.Max(e, math.Abs(got-want)/want)
		}
		return e * 1200 / math.Ln2, fm * phaseDelay(a, w1)
	})
	if m > 0 {
		worst, delay = dispersionCents(a, m, b, f0, sr, top)
	}
	if m == 0 || worst > 3 {
		// stiff: the exact fit, as many sections as it takes
		a, m, _ = fitSections(period, maxSections, 1, func(a float64, m int) (float64, float64) {
			return dispersionCents(a, m, b, f0, sr, top)
		})
		if m > 0 {
			worst, delay = dispersionCents(a, m, b, f0, sr, top)
		}
	}
	if m == 0 {
		return 0, 0, 0, 0
	}
	dispersionMu.Lock()
	if len(dispersionCache) > 4096 {
		clear(dispersionCache)
	}
	dispersionCache[key] = dispDesign{a, m, delay, worst}
	dispersionMu.Unlock()
	return a, m, delay, worst
}

// pol is one polarization of the string: a loop
type pol struct {
	line []float64
	pos  int
	n    int     // the whole samples of the loop
	ap   float64 // the allpass coefficient for the fraction
	apX  float64 // the allpass state
	apY  float64
	prev float64 // the loop filter: y = (1-s) x + s x[-1]
	disp dispChain
	pen  float64 // how far the string was in the fret

	// a legato changes the stiffness under a ringing string: the old chain fades out
	oldDisp dispChain
	fade    int
	fadeLen int
}

func (p *pol) clear() {
	clear(p.line)
	p.prev, p.apX, p.apY, p.pen, p.fade = 0, 0, 0, 0, 0
	p.disp.clear()
}

// filter reads the loop back and runs it through the allpass of the fraction, the stiffness and the
// loss filter
func (p *pol) filter(s float64, stiff bool) float64 {
	size := len(p.line)
	x := p.line[(p.pos-p.n+size)%size]
	y := p.ap*x + p.apX - p.ap*p.apY
	p.apX, p.apY = x, y
	if stiff {
		if p.fade > 0 {
			yo := p.oldDisp.tick(y)
			w := 1 - float64(p.fade)/float64(p.fadeLen)
			y = (1-w)*yo + w*p.disp.tick(y)
			p.fade--
		} else {
			y = p.disp.tick(y)
		}
	}
	f := (1-s)*y + s*p.prev
	p.prev = y
	return f
}

func (p *pol) push(in float64) {
	p.line[p.pos] = in
	p.pos++
	if p.pos >= len(p.line) {
		p.pos = 0
	}
}

// at is what was written to the loop delay samples ago (1 or more, read between the samples)
func (p *pol) at(delay float64) float64 {
	size := len(p.line)
	fi := math.Floor(delay)
	fr := delay - fi
	i := ((p.pos-int(fi))%size + size) % size
	return p.line[i]*(1-fr) + p.line[(i-1+size)%size]*fr
}

// add writes a value into what was written delay samples ago (between two samples)
func (p *pol) add(delay, x float64) {
	size := len(p.line)
	fi := math.Floor(delay)
	fr := delay - fi
	i := ((p.pos-int(fi))%size + size) % size
	p.line[i] += x * (1 - fr)
	p.line[(i-1+size)%size] += x * fr
}

// damperLoss is the share of a partial the finger on a node takes each time round the loop
const damperLoss = 0.3

// damp is the finger laid on the string at 1/k of its length, the loop's input in passing it: the
// partials that have a node there pass, for the others the finger is where the string moves and takes
// a share (the input and its k-1 copies a k-th of the loop apart, averaged, against the input)
func (p *pol) damp(in, step float64, k int) float64 {
	sum := in
	for j := 1; j < k; j++ {
		sum += p.at(float64(j) * step)
	}
	return (1-damperLoss)*in + damperLoss*sum/float64(k)
}

// fret contact: the spring of the fret and the damper in it, as shares of how far the string is in
const (
	fretSpring = 0.9
	fretDamper = 0.3
)

// contact is the string meeting a fret at d samples along the loop (from the bridge, there and
// back): the displacement there is the difference of the waves going to the bridge and from it, and
// where it goes lower than the gap (the action) the fret pushes back, a stiff one-sided spring and a
// damper in the way in, written into the two waves
func (p *pol) contact(period, d, gap float64) {
	a1, a2 := period-d, d
	u := (p.at(a1) - p.at(a2)) / 2
	pen := -u - gap
	if pen <= 0 {
		p.pen = 0
		return
	}
	f := fretSpring*pen + fretDamper*(pen-p.pen)
	p.pen = pen
	if f > 0 {
		p.add(a1, f)
		p.add(a2, -f)
	}
}

// polarization settings, for a Polarization of p (0-1)
const (
	polDetune   = 0.0029 // the sideways string is about 5 cents lower
	polOutput   = 0.5    // how much of the sideways string reaches the bridge
	polSideways = 0.4    // the pluck's push sideways, a part of p
)

// polGains are the gains per loop of the up and down string and the sideways one: the first
// loses its energy to the bridge and the body, the second hardly does
func polGains(g, p float64) (float64, float64) {
	return math.Pow(g, 1/(1-0.6*p)), math.Pow(g, 1/(1+p))
}

// polCoupling is the part of a string that crosses over to the other at the bridge
func polCoupling(p float64) float64 { return 0.0004 + 0.0015*p }

// shapeExcite fills the loop (n samples, the string there and back) with a string pulled aside by a
// finger or a pick: a triangle held at pick (0-0.5 of the string) with the odd extension for the
// way back, rounded by the width of what pulls it
func shapeExcite(ex []float64, n int, pick, width, hard float64) {
	l := float64(n) / 2
	a := math.Max(1, math.Min(math.Round(pick*l), l-1))
	for i := range n {
		x, sign := float64(i), 1.0
		if x > l {
			x, sign = 2*l-x, -1
		}
		y := x / a
		if x > a {
			y = (l - x) / (l - a)
		}
		ex[i] = sign * math.Max(0, y)
	}
	ex[n] = ex[0]
	// the contact: a finger is soft and wide, a pick thin; two passes of a moving average round the tip
	w := int(math.Round(float64(n) * width * (1 - 0.85*hard)))
	if w < 2 {
		return
	}
	w |= 1
	h := w / 2
	box := make([]float64, n)
	for range 2 {
		copy(box, ex[:n])
		sum := 0.0
		for j := -h; j <= h; j++ {
			sum += box[(j+n)%n]
		}
		for i := range n {
			ex[i] = sum / float64(w)
			sum += box[(i+h+1)%n] - box[(i-h+n)%n]
		}
	}
	ex[n] = ex[0]
}

// friction constants of the bow
const (
	frictionStatic  = 1.0   // the coefficient of sticking
	frictionDynamic = 0.35  // of sliding
	frictionSlope   = 0.03  // the relative speed the sliding friction falls over (the Stribeck curve)
	frictionHeat    = 0.45  // how much a hot rosin lets go
	frictionPower   = 4.0   // the heat a slip makes
	frictionCool    = 0.002 // seconds the rosin takes to heat and cool
)

// friction is the push the bow gives the string for a difference dv between its speed and the speed
// the string would have, for a force on the string (the bow table with a memory): the bow sticks
// while the friction can hold the string, and slips when it can't, the friction then falling as
// the slip grows (the Stribeck curve) and, the rosin heating, falling more the longer it slips
func (v *Voice) friction(dv, force, grain float64) float64 {
	a := math.Abs(dv)
	mus := frictionDynamic + (frictionStatic-frictionDynamic)*(1-frictionHeat*v.bowHeat)
	force *= grain
	var p float64
	if a <= force*mus {
		p = a
	} else {
		lo, hi := 0.0, a
		for range 12 {
			mid := (lo + hi) / 2
			mu := frictionDynamic + (mus-frictionDynamic)*math.Exp(-(a-mid)/frictionSlope)
			if mid < force*mu {
				lo = mid
			} else {
				hi = mid
			}
		}
		p = (lo + hi) / 2
	}
	heat := math.Min(1, (a-p)*force*frictionPower)
	v.bowHeat += (heat - v.bowHeat) * v.heatRate
	if dv < 0 {
		return -p
	}
	return p
}
