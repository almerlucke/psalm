package vocal

import "math"

// The glottal source: the flow of air through the vocal folds as one cycle of a Rosenberg-Klatt
// pulse with an open quotient, a speed quotient (how much sooner it opens than closes) and a return
// (the soft end of the closing, a one-pole low-pass on the pulse, whose cut-off is the voice's tilt).
// What the voice sings is the derivative of the flow: the lips radiate it. The pulse is band-limited
// at the instant the folds snap shut, the one hard edge of the cycle.

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
}

// polyBLEP is the correction of a unit step at t = 0 (phase 0-1, dt the step per sample)
func polyBLEP(t, dt float64) float64 {
	if t < dt {
		t /= dt
		return t + t - t*t - 1
	}
	if t > 1-dt {
		t = (t - 1) / dt
		return t*t + t + t + 1
	}
	return 0
}

// tick returns the next sample of the flow's derivative and the flow (0-1), for a pitch of inc
// (cycles a sample)
func (g *glottis) tick(inc, oq, sq, jitter, shimmer float64) (dv, flow float64) {
	if g.mul == 0 {
		g.mul, g.amp = 1, 1
	}
	dt := math.Min(inc/g.mul, 0.45)
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
	dv += glottalScale * math.Pi / (2 * tn) / 2 * polyBLEP(t, dt)
	dv *= g.amp
	g.phase += dt
	if g.phase >= 1 {
		g.phase--
		if g.phase >= 1 { // a pitch too high to follow
			g.phase = 0
		}
		g.cycles++
		g.mul, g.amp = 1, 1
		if jitter > 0 {
			g.mul = math.Max(0.5, math.Min(1+jitter*math.Sqrt(3)*(g.rng.float()*2-1), 1.5))
		}
		if shimmer > 0 {
			g.amp = math.Max(0.3, 1+shimmer*math.Sqrt(3)*(g.rng.float()*2-1))
		}
		g.cycle = g.mul / math.Max(inc, 1e-9)
	}
	return dv, flow
}
