package waveguide

import "math"

// The parts of a real string: its stiffness (a chain of allpass filters that stretch the partials), its
// two polarizations, the shape of the pluck, the friction of the bow.

// maxSections is the most allpass sections of the stiffness
const maxSections = 12

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

// stiffnessAt is the B a string has at a pitch: the thick low strings are stiffer, B at 220 Hz is
// that of the setting
func stiffnessAt(b, f float64) float64 {
	return b * math.Max(0.3, math.Min(220/f, 6))
}

// fitDispersion finds the sections and the coefficient of the allpass chain that put the partials of
// a string of f0 where f0 n sqrt((1+B n^2)/(1+B)) is (the first stays at f0). It returns the delay
// the chain adds at the fundamental, to be taken off the loop, and the worst error in cents
func fitDispersion(b, f0, sr float64) (a float64, m int, delay, worst float64) {
	top := min(10, int(sr/6/f0))
	if b <= 0 || top < 2 {
		return 0, 0, 0, 0
	}
	period := sr / f0
	w1 := 2 * math.Pi * f0 / sr
	worstOf := func(a float64, m int) (float64, float64) {
		fm := float64(m)
		l0 := period - fm*phaseDelay(a, w1) // the rest of the loop
		e := 0.0
		for n := 2; n <= top; n++ {
			fn := float64(n)
			want := period / math.Sqrt((1+b*fn*fn)/(1+b))
			got := l0 + fm*phaseDelay(a, w1*fn*math.Sqrt((1+b*fn*fn)/(1+b)))
			e = math.Max(e, math.Abs(got-want)/want)
		}
		return e * 1200 / math.Ln2, fm * phaseDelay(a, w1)
	}
	worst = math.Inf(1)
	for mm := 1; mm <= maxSections; mm++ {
		bestA, bestE := 0.0, math.Inf(1)
		for a := -0.97; a < 0; a += 0.01 {
			if e, d := worstOf(a, mm); e < bestE && period-d > 4 {
				bestA, bestE = a, e
			}
		}
		for step := 0.005; step > 0.0002; step /= 2 {
			for _, a := range [2]float64{bestA - step, bestA + step} {
				if a >= 0 || a < -0.99 {
					continue
				}
				if e, d := worstOf(a, mm); e < bestE && period-d > 4 {
					bestA, bestE = a, e
				}
			}
		}
		if bestE < worst-0.05 {
			worst, a, m = bestE, bestA, mm
		}
	}
	if m == 0 {
		return 0, 0, 0, 0
	}
	_, delay = worstOf(a, m)
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
}

func (p *pol) clear() {
	clear(p.line)
	p.prev, p.apX, p.apY = 0, 0, 0
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
		y = p.disp.tick(y)
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
