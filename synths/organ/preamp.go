package organ

import "math"

// preamp is the tube preamp of the organ: two triode stages that clip asymmetrically (a bias
// pushes the signal off the middle of the curve, so the even harmonics come with the odd), a coupling
// capacitor between and after them that takes the offset away, and the tone control between the stages,
// a tilt that trades the lows for the highs.
type preamp struct {
	lp       float64 // the tone control's low pass
	dc1, dc2 float64 // the coupling capacitors
	x1, x2   float64
}

// tube is one stage: a soft, asymmetrical curve with a slope of 1 at the centre, bias b
func tube(x, b float64) float64 {
	t := math.Tanh(b)
	return (math.Tanh(x+b) - t) / (1 - t*t)
}

// tilt is the tone control: 0.5 is flat, below it the highs go (up to 9 dB), above it the lows
func (p *preamp) tilt(x, tone, coef float64) float64 {
	p.lp += (x - p.lp) * coef
	t := tone - 0.5
	return p.lp*math.Exp2(-t*3) + (x-p.lp)*math.Exp2(t*3)
}

// process runs a sample through: drive 0 is clean (only the tone control acts), 1 is the stages
// driven hard; tone 0-1; coef is the tone filter's coefficient and dcCoef the coupling capacitors'
func (p *preamp) process(s, drive, tone, coef, dcCoef float64) float64 {
	toned := math.Abs(tone-0.5) > 1e-3
	if drive <= 0 {
		if toned {
			return p.tilt(s, tone, coef)
		}
		return s
	}
	d := math.Min(drive, 2)
	g1, g2 := 1+d*3, 1+d*5
	bias := 0.12 + 0.28*math.Min(d, 1)
	y := tube(s*g1, bias) / g1
	p.dc1 += (y - p.dc1) * dcCoef // the coupling capacitor
	y -= p.dc1
	if toned {
		y = p.tilt(y, tone, coef)
	}
	y = tube(y*g2, bias*0.6) / g2
	p.dc2 += (y - p.dc2) * dcCoef
	return y - p.dc2
}
