package supersaw

import "math"

// ensemble is the chorus of a string machine for a channel: three delays of a few milliseconds,
// each swept by the same slow LFO a third of a cycle apart
type ensemble struct {
	buf [2048]float64
	w   int
}

func (e *ensemble) clear() {
	e.buf = [2048]float64{}
}

// process takes a sample and returns it with the ensemble mixed in (amount 0-1), for the phase of the LFO
func (e *ensemble) process(x, amount, phase, sr float64) float64 {
	e.buf[e.w] = x
	var wet float64
	for k := range 3 {
		ph := phase + float64(k)/3
		delay := (7 + 2.5*math.Sin(2*math.Pi*ph)) * 0.001 * sr
		delay = min(delay, float64(len(e.buf)-2))
		pos := float64(e.w) - delay
		if pos < 0 {
			pos += float64(len(e.buf))
		}
		i := int(pos)
		frac := pos - float64(i)
		j := (i + 1) % len(e.buf)
		wet += e.buf[i]*(1-frac) + e.buf[j]*frac
	}
	e.w = (e.w + 1) % len(e.buf)
	return x*(1-0.5*amount) + wet/3*amount*1.2
}
