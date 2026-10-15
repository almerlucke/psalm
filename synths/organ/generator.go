package organ

import "math"

// The tone generator of a Hammond: a motor turns a shaft at 20 Hz (1200 rpm, synchronous), and
// twelve gear pairs step that down to the twelve wheels of the top octave of the generator. Gears
// have whole numbers of teeth, so the semitones come out a little off equal temperament (the
// characteristic slight mistuning of the instrument); every other octave is an exact doubling.
// Each wheel is a disc with teeth (a bump per tooth) passing a pickup coil, which makes a nearly,
// but not quite, sinusoidal voltage.

const shaftHz = 20.0

// gearRatios are the driving and driven teeth of the twelve gear pairs, C to B; with the two teeth
// of the slowest wheels the frequency of a wheel is 2 * 20 Hz * driving / driven
var gearRatios = [12][2]int{
	{85, 104}, {84, 97}, {78, 85}, {35, 36}, {103, 100}, {24, 22},
	{37, 32}, {49, 40}, {61, 47}, {33, 24}, {67, 46}, {71, 46},
}

// wheelHz returns the frequency of a tonewheel, 0 (C1, 32.69 Hz) to 90 (F#8)
func wheelHz(w int) float64 {
	g := gearRatios[w%12]
	return 2 * shaftHz * float64(g[0]) / float64(g[1]) * float64(int(1)<<uint(w/12))
}

// tempered returns the equal tempered frequency of a wheel
func tempered(w int) float64 { return 440 * math.Exp2(float64(w+lowWheel-69)/12) }

// wheelCents returns how far a wheel is from equal temperament, in cents
func wheelCents(w int) float64 { return 1200 * math.Log2(wheelHz(w)/tempered(w)) }

// shape is how a wheel and its pickup differ from a sine: a second and a third harmonic, with a
// phase each (in cycles). It differs a little from wheel to wheel, deterministically.
type shape struct{ a2, p2, a3, p3 float64 }

var shapes = func() (s [numWheels]shape) {
	hash := func(w, k int) float64 {
		x := math.Sin(float64(w)*12.9898+float64(k)*78.233) * 43758.5453
		return x - math.Floor(x)
	}
	for w := range s {
		// the pickup rounds the higher wheels off, the lower ones show their teeth more
		fall := 1 - 0.5*float64(w)/numWheels
		s[w] = shape{
			a2: fall * (0.018 + 0.016*hash(w, 1)),
			p2: hash(w, 2),
			a3: fall * (0.008 + 0.01*hash(w, 3)),
			p3: hash(w, 4),
		}
	}
	return
}()

// sinCycles is a sine of x cycles from the table
func sinCycles(x float64) float64 {
	x -= math.Floor(x)
	p := x * tableSize
	i := int(p)
	f := p - float64(i)
	return sineTable[i] + (sineTable[i+1]-sineTable[i])*f
}

// contactHash is a number from 0 to 1 that is always the same for a key and a contact
func contactHash(pitch, contact int) float64 {
	x := math.Sin(float64(pitch)*91.7+float64(contact)*37.31) * 24634.6345
	return x - math.Floor(x)
}

// contact is one of the nine contacts of a key: it connects a wheel to the bus bar of a drawbar.
// The contacts of a key close one after the other, a few milliseconds from the first to the last,
// and may bounce as they close: at a random point of the wheel's cycle the bus jumps, which is the
// click, and the staggered closing is the percussive attack of the organ.
type contact struct {
	env     float64 // how closed it is (the closing is a little smoothed)
	pend    float64 // where it goes when the wait is over: 1 closed, 0 open
	target  float64
	wait    int // samples until it takes pend
	bounce  int // samples to the bounce (target goes 0) after closing
	rebound int // samples until it closes again after the bounce
}

// press and release schedule a contact a delay from now
func (c *contact) press(delay int) { c.pend, c.wait = 1, delay+1 }
func (c *contact) release(delay int) {
	c.pend, c.wait = 0, delay+1
}

// step moves a contact a sample on, toward its target at the closing or opening speed
func (c *contact) step(bounceIn, bounceOut int, attack, release float64) {
	if c.wait > 0 {
		if c.wait--; c.wait == 0 {
			c.target = c.pend
			if c.pend == 1 && bounceIn > 0 {
				c.bounce = bounceIn
			}
		}
	}
	if c.bounce > 0 {
		if c.bounce--; c.bounce == 0 {
			c.target, c.rebound = 0, bounceOut
		}
	} else if c.rebound > 0 {
		if c.rebound--; c.rebound == 0 {
			c.target = c.pend
		}
	}
	if c.target > c.env {
		c.env += (c.target - c.env) * attack
	} else {
		c.env += (c.target - c.env) * release
	}
}

// idle says whether a contact is open, at rest and silent
func (c *contact) idle() bool {
	return c.wait == 0 && c.bounce == 0 && c.rebound == 0 && c.target == 0 && c.env < 1e-6
}
