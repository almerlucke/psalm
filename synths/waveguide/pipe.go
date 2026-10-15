package waveguide

import "math"

// The pipe is a tube that is blown. The tube is a delay line the sound goes up and down, closed
// by a filter that takes the high partials and the steady pressure out of what comes back.
//
// A flute is blown over an edge: what the tube sends back steers an air jet, which crosses to the
// edge in a time (the jet delay, a part of the tube's) and drives the tube again through a curve
// that saturates (tanh). The more breath, the more gain in the loop, up to where it saturates; the
// embouchure sets the jet off the edge (an asymmetry: the even partials) and the air is noise in the
// jet. A clarinet has a reed instead: the breath against the pressure in the tube opens and closes
// it (a clipped line, after the STK), and the tube, closed at one end, plays only the odd partials,
// an octave lower than an open one of its length.
//
// A tube that is driven hard sounds a little off its length. The voice measures the period of the
// sound and trims the length until it is the pitch's (a few cents or better, whatever the breath,
// the embouchure or the damping).

// pipeState is the filters of a blown pipe
type pipeState struct {
	lp, lp1    float64
	dcX, dcY   float64
	a, dc      float64 // the poles of the flute's tube and the filter of the steady pressure
	outX, outY float64
	// the pitch keeper of the flute: the period of the tube's sound is measured, and the tube's length
	// trimmed until it is the pitch's (the jet bends it by the breath, the embouchure and the damping)
	clock, lastCross, prevT, amp, trim float64
}

// pipeFluteExtra and pipeReedExtra are the samples the loop holds besides its delay lines and
// filters (reading the line's last sample, the interpolation), measured: the lines are shortened by them
const (
	pipeFluteExtra = 1.0
	pipeReedExtra  = 1.145
)

// fluteJet is how hard the jet drives the tube: the loop gain at the breath of 1, over what the
// tube loses
const fluteJet = 2.6

// fluteCut is the pole of the filter of the flute's tube for a pitch and the damping: the sound
// of the tube is darker the higher the damping, and the higher the pitch the less of its partials are
// left in the loop (the cutoff is a multiple of the pitch)
func fluteCut(f, damping, sr float64) float64 {
	fc := f * (1.6 + 4*(1-math.Max(0, math.Min(damping, 1))))
	return math.Exp(-2 * math.Pi * math.Min(fc, sr*0.4) / sr)
}

// onePoleDelay is the phase delay in samples of y = (1-a) x + a y[-1] at a pitch w (radians per sample)
func onePoleDelay(a, w float64) float64 {
	return math.Atan2(a*math.Sin(w), 1-a*math.Cos(w)) / w
}

// dcPole is the pole of the filter that takes the steady pressure out of the loop: it cuts below the
// pitch, or the loop would sing slowly below it
func dcPole(f, sr float64) float64 { return math.Exp(-2 * math.Pi * 0.4 * f / sr) }

// dcBlockDelay is the phase delay of y = x - x[-1] + p y[-1], negative (it leads)
func dcBlockDelay(p, w float64) float64 {
	// H = (1 - z^-1) / (1 - p z^-1) at z = e^jw
	re, im := 1-math.Cos(w), math.Sin(w)
	dre, dim := 1-p*math.Cos(w), p*math.Sin(w)
	return -(math.Atan2(im, re) - math.Atan2(dim, dre)) / w
}

// jetTable is the curve of the air jet at the edge: it drives the tube until it saturates
func jetTable(x float64) float64 { return math.Tanh(x) }

// pipeNote sets the length of the tube
func (v *Voice) pipeNote() {
	sr := v.Config().SampleRate
	if !v.active {
		v.neck.clear()
		v.bridge.clear()
		v.pipe = pipeState{}
		v.bowEnv = 0
		v.vibPhase = 0
	}
	if v.cfg.Reed {
		// a closed tube: the sound goes up and down in a period, the delay line is half of it
		s := 0.05 + 0.45*math.Max(0, math.Min(v.cfg.Damping, 1))
		v.baseDelay = sr/v.freq/2 - s - pipeReedExtra
	} else {
		a := fluteCut(v.freq, v.cfg.Damping, sr)
		w := 2 * math.Pi * v.freq / sr
		v.pipe.a = a
		v.pipe.dc = dcPole(v.freq, sr)
		v.baseDelay = sr/v.freq - 2*onePoleDelay(a, w) - dcBlockDelay(v.pipe.dc, w) - pipeFluteExtra
	}
	v.baseDelay = math.Max(v.baseDelay, 4)
	v.vibWait = int(v.cfg.VibratoDelay * sr)
}

// pipeTick returns a sample of the tube: ratio stretches the tube (lower pitch above 1), breath
// is the pressure, noise the air and embouchure the jet or the reed
func (v *Voice) pipeTick(ratio, breath, noise, emb, sr float64) float64 {
	p := &v.pipe
	v.rng = v.rng*1664525 + 1013904223
	n := float64(int32(v.rng)) / 2147483648
	b := math.Max(0, math.Min(breath, 1.5)) * v.bowEnv
	noise = math.Max(0, math.Min(noise, 1))
	emb = math.Max(0, math.Min(emb, 1))
	L := v.baseDelay*ratio + p.trim
	var out float64
	if v.cfg.Reed {
		press := 0.9 * b * (1 + noise*0.15*n)
		v.neck.len = L
		s := 0.05 + 0.45*math.Max(0, math.Min(v.cfg.Damping, 1))
		p.lp = (1-s)*v.neck.out + s*p.lp
		t := -0.95 * p.lp
		pd := t - press
		slope := -(0.25 + 0.35*emb)
		reed := math.Max(-1, math.Min(0.7+slope*pd, 1))
		v.neck.tick(press + pd*reed)
		out = v.neck.out
	} else {
		jr := 0.2 + 0.3*emb
		v.neck.len = L * (1 - jr)
		v.bridge.len = L * jr
		a := p.a
		p.lp1 = (1-a)*v.neck.out + a*p.lp1
		p.lp = (1-a)*p.lp1 + a*p.lp
		// the tube holds no steady pressure
		t := p.lp - p.dcX + p.dc*p.dcY
		p.dcX, p.dcY = p.lp, t
		// the jet, off the edge by the embouchure, with the air in it
		x := t + 0.5*emb + 0.2*noise*n
		x = v.bridge.tick(x)
		v.neck.tick(fluteJet * b * jetTable(x))
		out = v.neck.out
	}
	// what leaves the tube has no steady pressure either
	y := out - p.outX + 0.995*p.outY
	p.outX, p.outY = out, y
	// keep the pitch: the period of the sound against the pitch's, a rising crossing to the next
	// (a reed's line is half the period)
	p.clock++
	p.amp = math.Max(math.Abs(y), p.amp*0.999)
	if p.prevT < 0 && y >= 0 {
		cross := p.clock - y/(y-p.prevT)
		pt := sr / v.freq * ratio
		if pm := cross - p.lastCross; pm > 0.7*pt {
			if p.lastCross > 0 && p.amp > 0.05 && pm < 1.3*pt {
				k := 0.3
				if v.cfg.Reed {
					k = 0.15
				}
				p.trim = math.Max(-0.06*pt, math.Min(p.trim+k*(pt-pm), 0.06*pt))
			}
			p.lastCross = cross
		}
	}
	p.prevT = y
	if math.IsNaN(y) || math.Abs(y) > 50 {
		v.neck.clear()
		v.bridge.clear()
		*p = pipeState{a: p.a, dc: p.dc}
		return 0
	}
	return y
}
