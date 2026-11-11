package piano

import "math"

// Where the piano is heard from: how wide the keys spread, the first reflections (the lid, the
// floor, the walls of a stage, each darker than the sound itself) and, from further away, the air
// taking some of the highs. A reverb adds the room after that

// reflection is one early reflection: its delay, level, how dark it is, and which side it comes
// from and goes to (0 left, 1 right, -1 both, the mid of the sides)
type reflection struct {
	ms, gain, hz float64
	from, to     int
}

type perspective struct {
	width       float64 // how far the keys spread, times the Width setting
	side        float64 // how wide the sound is as a whole (the side of mid/side)
	airHz       float64 // a lowpass for the distance, 0 is none
	reflections []reflection
}

var perspectives = map[Position]perspective{
	// on the bench: the keys from left to right, the lid above, the floor below
	Player: {width: 1, side: 1, reflections: []reflection{
		{ms: 3.0, gain: 0.22, hz: 4000, from: -1, to: -1},
		{ms: 5.5, gain: 0.15, hz: 2500, from: -1, to: -1},
	}},
	// close microphones over the open lid: wide, clear and present, the lid very near
	Studio: {width: 1.3, side: 1.1, reflections: []reflection{
		{ms: 1.2, gain: 0.18, hz: 7000, from: 0, to: 0},
		{ms: 1.6, gain: 0.18, hz: 7000, from: 1, to: 1},
		{ms: 4.5, gain: 0.1, hz: 3000, from: -1, to: -1},
	}},
	// in the audience: the piano narrower, the open lid throwing the sound out, the floor and the
	// walls of the stage, a little air
	Hall: {width: 0.35, side: 0.6, airHz: 7000, reflections: []reflection{
		{ms: 2.4, gain: 0.4, hz: 5000, from: -1, to: -1},
		{ms: 5.8, gain: 0.3, hz: 3000, from: -1, to: -1},
		{ms: 11, gain: 0.22, hz: 2500, from: 0, to: 0},
		{ms: 14, gain: 0.22, hz: 2500, from: 1, to: 1},
		{ms: 21, gain: 0.15, hz: 1800, from: -1, to: -1},
	}},
}

func perspectiveOf(p Position) perspective {
	if q, ok := perspectives[p]; ok {
		return q
	}
	return perspectives[Player]
}

// listener renders a perspective
type listener struct {
	q      perspective
	buf    [2][]float64
	pos    int
	delays []int
	coefs  []float64
	states []float64
	direct float64
	air    float64
	airL   float64
	airR   float64
}

func newListener(p Position, sr float64) *listener {
	q := perspectiveOf(p)
	n := int(0.03*sr) + 1
	l := &listener{q: q, buf: [2][]float64{make([]float64, n), make([]float64, n)}}
	energy := 1.0
	for _, r := range q.reflections {
		l.delays = append(l.delays, max(1, int(r.ms*0.001*sr)))
		l.coefs = append(l.coefs, 1-math.Exp(-2*math.Pi*r.hz/sr))
		l.states = append(l.states, 0)
		energy += r.gain * r.gain
	}
	l.direct = 1 / math.Sqrt(energy) // about as loud with the reflections as without
	if q.airHz > 0 {
		l.air = 1 - math.Exp(-2*math.Pi*q.airHz/sr)
	}
	return l
}

// process hears a sample from the perspective
func (l *listener) process(left, right float64) (float64, float64) {
	// the width of the whole
	mid, side := (left+right)/2, (left-right)/2*l.q.side
	left, right = mid+side, mid-side

	n := len(l.buf[0])
	l.buf[0][l.pos], l.buf[1][l.pos] = left, right
	outL, outR := left*l.direct, right*l.direct
	for i, r := range l.q.reflections {
		at := (l.pos - l.delays[i] + n) % n
		var x float64
		switch r.from {
		case 0, 1:
			x = l.buf[r.from][at]
		default:
			x = (l.buf[0][at] + l.buf[1][at]) / 2
		}
		l.states[i] += l.coefs[i] * (x - l.states[i])
		y := l.states[i] * r.gain * l.direct
		switch r.to {
		case 0:
			outL += y
		case 1:
			outR += y
		default:
			outL += y
			outR += y
		}
	}
	l.pos = (l.pos + 1) % n

	if l.air > 0 {
		l.airL += l.air * (outL - l.airL)
		l.airR += l.air * (outR - l.airR)
		outL, outR = l.airL, l.airR
	}
	return outL, outR
}

// keyPan returns where a key sounds in the stereo field (0 left, 1 right): low keys on the left,
// as wide as the Width setting and the position
func keyPan(pitch float64, cfg *Config) float64 {
	w := max(0, min(cfg.Width*perspectiveOf(cfg.Position).width, 1))
	return 0.5 + 0.5*w*max(-1, min((pitch-64)/44, 1))
}
