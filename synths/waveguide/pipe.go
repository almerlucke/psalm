package waveguide

import (
	"math"

	"github.com/almerlucke/psalm/blocks/tanh"
)

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
	ampThen                            float64 // the loudness a period ago: the tone has grown up when it hasn't grown since
	// the player's ease: a flute that sounds an octave too high (overblown) gets less jet until it
	// falls back (1: all of the breath), and short counts the crossings in a row that were half a period
	ease    float64
	short   int
	lastAny float64
	// the length the tube has now: it follows trim at a limited speed, since a delay line whose length
	// jumps jumps in the sound (a click), and trim is set a period at a time
	trimNow float64
}

// pipeFluteExtra and pipeReedExtra are the samples the loop holds besides its delay lines and
// filters (reading the line's last sample, the interpolation), measured: the lines are shortened by them
const (
	pipeFluteExtra = 1.0
	pipeReedExtra  = 1.145
)

// a note speaks quickly: while the tone is weaker than speakAmp the loop gain is up to 1+speakBoost
// times its own, and falls to its own as the tone grows; speakKick is the pulse the tube starts with
const (
	speakAmp   = 0.25
	speakBoost = 0.25
	speakKick  = 0.12
	// a reed grows fastest with its pressure at reedBest (against the closing pressure of 1), for
	// reedLeanPeriods periods from the start of a note
	reedBest        = 0.84
	reedLeanPeriods = 20

	// reedSeed is the puff a reed's tube starts with: the line holds this, and the closed end turns it
	// upside down on the way back
	reedSeed = 0.3
)

// easeFloor is the least of the jet a player eases to: below it the tone of some flutes dies (found by
// trying every flute preset over the range: 0.8 keeps all of them sounding in their own register)
const easeFloor = 0.8

// easeNow is how much of the jet is left (1 when the player isn't easing)
func (p *pipeState) easeNow() float64 {
	if p.ease == 0 {
		return 1
	}
	return p.ease
}

// trimSlew is the most the length of the tube moves in a sample (the pitch keeper)
const trimSlew = 0.004

// fluteJet is how hard the jet drives the tube: the loop gain at the breath of 1, over what the
// tube loses
const fluteJet = 2.6

// fluteCut is the pole of the filter of the flute's tube for a pitch and the damping: the sound
// of the tube is darker the higher the damping, and the higher the pitch the less of its partials are
// left in the loop (the cutoff is a multiple of the pitch)
func fluteCut(f, damping, sr float64) float64 {
	fc := f * (1.6 + 4*(1-max(0, min(damping, 1))))
	return math.Exp(-2 * math.Pi * min(fc, sr*0.4) / sr)
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
func jetTable(x float64) float64 { return tanh.Tanh(x) }

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
		s := 0.05 + 0.45*max(0, min(v.cfg.Damping, 1))
		v.baseDelay = sr/v.freq/2 - s - pipeReedExtra
	} else {
		a := fluteCut(v.freq, v.cfg.Damping, sr)
		w := 2 * math.Pi * v.freq / sr
		v.pipe.a = a
		v.pipe.dc = dcPole(v.freq, sr)
		v.baseDelay = sr/v.freq - 2*onePoleDelay(a, w) - dcBlockDelay(v.pipe.dc, w) - pipeFluteExtra
	}
	v.baseDelay = max(v.baseDelay, 4)
	v.vibWait = int(v.cfg.VibratoDelay * sr)
	if v.cfg.Reed && !v.active {
		// the tube starts with a puff in it: the line holds a wave that the closed end turns upside
		// down on the way back, a square wave of the tube's own period, so a low note (whose period is
		// long, and whose growth takes a number of periods) doesn't start from nothing
		n := len(v.neck.buf)
		for k := 1; k <= int(v.baseDelay)+1 && k < n; k++ {
			v.neck.buf[((v.neck.pos-k)%n+n)%n] = reedSeed
		}
	}
}

// pipeTick returns a sample of the tube: ratio stretches the tube (lower pitch above 1), breath
// is the pressure, noise the air and embouchure the jet or the reed
func (v *Voice) pipeTick(ratio, breath, noise, emb, sr float64) float64 {
	p := &v.pipe
	v.rng = v.rng*1664525 + 1013904223
	n := float64(int32(v.rng)) / 2147483648
	b := max(0, min(breath, 1.5)) * v.bowEnv
	noise = max(0, min(noise, 1))
	emb = max(0, min(emb, 1))
	// the trim moves the length a little at a time (about 0.004 samples a sample, a few cents of pitch
	// a period at the most), never in a jump
	if d := p.trim - p.trimNow; d != 0 {
		p.trimNow += max(-trimSlew, min(d, trimSlew))
	}
	L := v.baseDelay*ratio + p.trimNow
	// a note speaks quickly: while the tone is weaker than speakAmp the loop gain is higher than its own
	// (the weaker, the higher), so the sound builds up in a few periods whatever the pitch (a low note
	// has a long period, and with a gain barely above one it would take half a second), and the gain
	// falls to its own as the tone grows. The steady sound is the loop's own (the clipping decides how
	// loud it gets)
	pt0 := sr / v.freq * ratio
	boost := 1.0
	if p.amp < speakAmp {
		boost += speakBoost * (1 - p.amp/speakAmp)
	}
	// and it starts with a kick, the pressure step of the breath arriving: a small pulse the tube
	// rings with at its own period, which the loop then grows from instead of from nothing
	kick := 0.0
	if p.clock < 1 {
		kick = speakKick * min(1, max(0.3, breath))
	}
	var out float64
	if v.cfg.Reed {
		press := 0.9 * b * (1 + noise*0.15*n)
		// the player leans on the reed at the start, up to the pressure where it grows fastest (any more
		// and it closes: the tone dies), for a number of periods: the tone is up by then and the
		// pressure is the preset's own
		if nominal := 0.9 * min(breath, 1.5); nominal > 0 && nominal < reedBest {
			if age := p.clock / (reedLeanPeriods * pt0); age < 1 {
				press *= 1 + (reedBest/nominal-1)*(1-age)
			}
		}
		v.neck.len = L
		s := 0.05 + 0.45*max(0, min(v.cfg.Damping, 1))
		p.lp = (1-s)*v.neck.out + s*p.lp
		t := -0.95 * p.lp
		pd := t - press
		slope := -(0.25 + 0.35*emb)
		reed := max(-1, min(0.7+slope*pd, 1))
		v.neck.tick(press + pd*reed + kick)
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
		v.neck.tick(fluteJet*boost*p.easeNow()*b*jetTable(x) + kick)
		out = v.neck.out
	}
	// what leaves the tube has no steady pressure either
	y := out - p.outX + 0.995*p.outY
	p.outX, p.outY = out, y
	// keep the pitch: the period of the sound against the pitch's, a rising crossing to the next
	// (a reed's line is half the period)
	p.clock++
	p.amp = max(math.Abs(y), p.amp*0.999)
	if p.prevT < 0 && y >= 0 {
		cross := p.clock - y/(y-p.prevT)
		pt := sr / v.freq * ratio
		if !v.cfg.Reed && v.cfg.Breath <= 1 {
			// every rising crossing, one after the other: an octave up is crossing after crossing a
			// half period apart (noise, or a partial, gives crossings that are not)
			if gap := cross - p.lastAny; p.lastAny > 0 && p.amp > 0.1 && math.Abs(gap-0.5*pt) < 0.06*pt {
				if p.short++; p.short > 12 {
					p.ease = max(easeFloor, p.easeNow()*0.98)
				}
			} else {
				p.short = 0
			}
			p.lastAny = cross
		}
		if pm := cross - p.lastCross; pm > 0.7*pt {
			// only when the breath has come all the way up: the first periods of a note are not the tube's
			// sound yet, and trimming by them throws the length about
			grown := p.amp > 0.1 && math.Abs(p.amp-p.ampThen) < 0.04*p.amp // steady: not still growing
			p.ampThen = p.amp
			if p.lastCross > 0 && grown && pm < 1.3*pt && v.bowEnv > 0.98 {
				k := 0.3
				if v.cfg.Reed {
					k = 0.15
				}
				step := max(-0.01*pt, min(k*(pt-pm), 0.01*pt)) // a period moves it by a hundredth at the most
				p.trim = max(-0.06*pt, min(p.trim+step, 0.06*pt))
			}
			p.lastCross = cross
		}
	}
	p.prevT = y
	// what is heard comes up over the preset's attack (the tube is already ringing, from the puff it
	// started with, so the sound would begin with a step)
	if fadeN := min(max(v.cfg.BowAttack, 0.01), 0.2) * sr; p.clock < fadeN {
		y *= 0.5 - 0.5*math.Cos(math.Pi*p.clock/fadeN)
	}
	if math.IsNaN(y) || math.Abs(y) > 50 {
		v.neck.clear()
		v.bridge.clear()
		*p = pipeState{a: p.a, dc: p.dc}
		return 0
	}
	return y
}
