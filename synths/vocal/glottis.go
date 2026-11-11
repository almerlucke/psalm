package vocal

import (
	"math"

	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
)

// The glottal source: the flow of air through the vocal folds as one cycle of a Rosenberg-Klatt
// pulse with an open quotient, a speed quotient (how much sooner it opens than closes) and a return
// (the soft end of the closing, a one-pole low-pass on the pulse, whose cut-off is the voice's tilt).
// What the voice sings is the derivative of the flow: the lips radiate it. The pulse is band-limited
// at the instant the folds snap shut, the one hard edge of the cycle, by the windowed sinc of the oscillators (blosc.BLEP).

const sinN = 2048

var sinTab [sinN + 1]float64

func init() {
	for i := range sinTab {
		sinTab[i] = math.Sin(2 * math.Pi * float64(i) / sinN)
	}
}

// sinTurn is sin(2 pi u) by table
func sinTurn(u float64) float64 {
	u -= math.Floor(u)
	x := u * sinN
	i := int(x)
	f := x - float64(i)
	return sinTab[i] + (sinTab[i+1]-sinTab[i])*f
}

// glottalScale keeps the pulse of a modal voice (open quotient 0.6, speed quotient 1.4) at a peak of
// about 1: a pressed voice has a sharper closing and so a louder, brighter pulse
const glottalScale = 0.159

// glottis makes the pulses of one singer, with the small irregularities of a real voice: jitter (each
// cycle a little longer or shorter) and shimmer (each a little louder or softer)
type glottis struct {
	phase  float64
	mul    float64 // this cycle's length against the nominal one
	amp    float64 // this cycle's amplitude
	rng    randSource
	cycle  float64 // the length in samples of the cycle that began last
	cycles int     // the cycles begun
	fl     float64 // the flow of the Liljencrants-Fant pulse so far this cycle
}

// tick returns the next sample of the flow's derivative and the flow (0-1), for a pitch of inc
// (cycles a sample)
func (g *glottis) tick(inc, oq, sq, jitter, shimmer float64) (dv, flow float64) {
	if g.mul == 0 {
		g.mul, g.amp = 1, 1
	}
	dt := min(inc/g.mul, 0.45)
	tp := oq * sq / (1 + sq)
	tn := oq - tp
	p := g.phase
	switch {
	case p < tp:
		x := p / tp
		dv = glottalScale * math.Pi / (2 * tp) * sinTurn(x/2)
		flow = 0.5 * (1 - sinTurn(x/2+0.25))
	case p < oq:
		y := (p - tp) / tn
		dv = -glottalScale * math.Pi / (2 * tn) * sinTurn(y/4)
		flow = sinTurn(y/4 + 0.25)
	}
	// the closing is a step in the derivative (from -A to 0)
	t := p - oq
	if t < 0 {
		t++
	}
	dv += glottalScale * math.Pi / (2 * tn) / 2 * blosc.BLEP(t, dt)
	// and the derivative has two corners, where the folds begin to open (from flat to rising) and where they are open the widest (from the opening to the closing)
	a1, a2 := glottalScale*math.Pi/(2*tp), glottalScale*math.Pi/(2*tn)
	slopeOpen := a1 * math.Pi / tp
	slopePeak := a1*math.Pi/tp - a2*math.Pi/(2*tn)
	tPeak := p - tp
	if tPeak < 0 {
		tPeak++
	}
	dv += 0.5 * dt * (slopeOpen*blosc.BLAMP(p, dt) + slopePeak*blosc.BLAMP(tPeak, dt))
	dv *= g.amp
	g.advance(dt, inc, jitter, shimmer)
	return dv, flow
}

// advance moves the phase on by dt and, at the end of a cycle, draws the length and the loudness of the
// next one
func (g *glottis) advance(dt, inc, jitter, shimmer float64) {
	g.phase += dt
	if g.phase >= 1 {
		g.phase--
		if g.phase >= 1 { // a pitch too high to follow
			g.phase = 0
		}
		g.cycles++
		g.fl = 0
		g.mul, g.amp = 1, 1
		if jitter > 0 {
			g.mul = max(0.5, min(1+jitter*math.Sqrt(3)*(g.rng.float()*2-1), 1.5))
		}
		if shimmer > 0 {
			g.amp = max(0.3, 1+shimmer*math.Sqrt(3)*(g.rng.float()*2-1))
		}
		g.cycle = g.mul / max(inc, 1e-9)
	}
}

// tickLF is tick for the Liljencrants-Fant pulse p
func (g *glottis) tickLF(inc float64, p *lfPulse, jitter, shimmer float64) (dv, flow float64) {
	if g.mul == 0 {
		g.mul, g.amp = 1, 1
	}
	dt := min(inc/g.mul, 0.45)
	d := p.at(g.phase) + p.corners(g.phase, dt)
	g.fl += d * dt
	flow = max(0, min(g.fl, 1))
	dv = d * p.out * g.amp
	g.advance(dt, inc, jitter, shimmer)
	return dv, flow
}

// The Liljencrants-Fant model of the flow's derivative over one cycle (time in cycles, the period 1):
// from the opening to the instant te the folds close at, an exponentially growing sine (growth alpha,
// half a period of it in tp, so the flow peaks at tp); then a return of the flow to zero, an
// exponential of time constant 1/eps that ends in the cycle's end, with ta its effective length. The
// negative peak at te is Ee = 1, and alpha is solved so the derivative integrates to zero: the flow
// ends where it began. The pulse is kept at a flow peak of 1 (for the breath that follows it), and the voice sings it with the
// closing peak, the excitation, at the same strength whatever the quality: a breathy voice has more of the
// fundamental, the loudness above it stays.
type lfPulse struct {
	tp, te, ta float64
	alpha, eps float64
	e0, ret    float64 // the amplitudes of the two parts
	tb, ebt    float64 // 1-te and exp(-eps tb)
	out        float64 // brings the closing peak to 1
	ok         bool
}

// lfRd derives tp, te and ta (in periods) from the single voice-quality parameter Rd (0.3 pressed to
// 2.7 breathy), by Fant's regression of Ra, Rk and Rg on Rd
func lfRd(rd float64) (tp, te, ta float64) {
	rd = max(0.3, min(rd, 2.7))
	ra := (-1 + 4.8*rd) / 100
	rk := (22.4 + 11.8*rd) / 100
	rg := rk / (4 * (0.11*rd/(0.5+1.2*rk) - ra))
	tp = 1 / (2 * rg)
	return tp, tp * (rk + 1), ra
}

// lfSolve solves the pulse of the times tp < te < 1 and the return time ta
func lfSolve(tp, te, ta float64) lfPulse {
	tp = max(0.05, min(tp, 0.9))
	te = max(tp+0.02, min(te, 0.98))
	tb := 1 - te
	ta = max(1e-4, min(ta, 0.9*tb))
	p := lfPulse{tp: tp, te: te, ta: ta, tb: tb}
	// eps: eps ta = 1 - exp(-eps tb), by Newton from 1/ta
	eps := 1 / ta
	for range 40 {
		e := math.Exp(-eps * tb)
		f := eps*ta - 1 + e
		df := ta - tb*e
		if df == 0 {
			break
		}
		next := eps - f/df
		if next <= 0 {
			next = eps / 2
		}
		if math.Abs(next-eps) < 1e-12*eps {
			eps = next
			break
		}
		eps = next
	}
	p.eps = eps
	p.ebt = math.Exp(-eps * tb)
	w := math.Pi / tp
	sw, cw := math.Sincos(w * te)
	// the integral of the derivative over the cycle, for Ee = 1, as a function of alpha
	ret := -(1 / (eps * ta)) * ((1-p.ebt)/eps - tb*p.ebt)
	g := func(a float64) float64 {
		open := -((a*sw - w*cw) + w*math.Exp(-a*te)) / (sw * (a*a + w*w))
		return open + ret
	}
	a, ok := 0.0, false
	// Newton from a typical value, with a numerical slope
	a = 4
	for range 30 {
		f := g(a)
		if math.Abs(f) < 1e-12 {
			ok = true
			break
		}
		h := 1e-5 * max(1, math.Abs(a))
		df := (g(a+h) - g(a-h)) / (2 * h)
		if df == 0 || math.IsNaN(df) {
			break
		}
		a -= f / df
		if a < -60 || a > 400 || math.IsNaN(a) {
			break
		}
	}
	if !ok || math.Abs(g(a)) > 1e-8 {
		// a scan for the sign change, then bisection
		lo, flo := -60.0, g(-60)
		for x := -59.0; x <= 400; x += 1 {
			fx := g(x)
			if flo*fx <= 0 {
				l, h := lo, x
				for range 60 {
					m := (l + h) / 2
					if g(l)*g(m) <= 0 {
						h = m
					} else {
						l = m
					}
				}
				a, ok = (l+h)/2, true
				break
			}
			lo, flo = x, fx
		}
	}
	p.alpha = a
	p.ok = ok || math.Abs(g(a)) < 1e-6
	p.ret = -1 / (eps * ta)
	p.e0 = -1 / (math.Exp(a*te) * sw)
	// scale to a flow peak of 1 (at tp the derivative crosses zero)
	peak := p.e0 * w * (math.Exp(a*tp) + 1) / (a*a + w*w)
	if peak > 1e-9 {
		p.e0 /= peak
		p.ret /= peak
	}
	if ee := -p.ret * (1 - p.ebt); ee > 1e-9 {
		p.out = 1 / ee
	}
	return p
}

// at is the derivative of the flow at time t of the cycle (0-1)
func (p *lfPulse) at(t float64) float64 {
	if t < p.te {
		return p.e0 * math.Exp(p.alpha*t) * sinTurn(t/(2*p.tp))
	}
	return p.ret * (math.Exp(-p.eps*(t-p.te)) - p.ebt)
}

// corners is the correction of the two corners of the derivative at the phase t (0-1), a step of dt a sample: where the cycle begins (the return has run out, with its slope,
// and the opening begins with another) and where the folds close (the opening turns into the return)
func (p *lfPulse) corners(t, dt float64) float64 {
	w := math.Pi / p.tp
	slopeOpen := p.e0 * w                           // at 0, the opening
	slopeEnd := -p.ret * p.eps * p.ebt              // at 1, the return
	atTe := p.e0 * math.Exp(p.alpha*p.te)           // the opening, at te
	sinTe, cosTe := math.Sincos(w * p.te)           //
	slopeBefore := atTe * (p.alpha*sinTe + w*cosTe) // the opening's slope at te
	slopeAfter := -p.ret * p.eps                    // the return's slope at te
	tClose := t - p.te
	if tClose < 0 {
		tClose++
	}
	return 0.5 * dt * ((slopeOpen-slopeEnd)*blosc.BLAMP(t, dt) + (slopeAfter-slopeBefore)*blosc.BLAMP(tClose, dt))
}

// harmN is how many harmonics of the source's spectrum are tabulated
const harmN = 64

// sourceHarmonics is the power of the first harmonics of the pulse the voice sings (one cycle of the
// derivative of the flow, the same for every pitch: what changes with the pitch is only where in Hz the
// harmonics stand). lf is nil for the Rosenberg-Klatt pulse
func sourceHarmonics(oq, sq float64, lf *lfPulse) (p [harmN]float64) {
	const n = 512
	var g glottis
	x := make([]float64, n)
	for i := range x {
		if lf != nil {
			x[i], _ = g.tickLF(1.0/n, lf, 0, 0)
		} else {
			x[i], _ = g.tick(1.0/n, oq, sq, 0, 0)
		}
	}
	for h := 1; h <= harmN; h++ {
		var re, im float64
		for i, v := range x {
			a := 2 * math.Pi * float64(h*i) / n
			re += v * math.Cos(a)
			im += v * math.Sin(a)
		}
		p[h-1] = (re*re + im*im) / (n * n)
	}
	return
}
