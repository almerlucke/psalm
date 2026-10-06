package piano

import (
	"math"
	"math/rand/v2"
	"sync"
)

// boardSeconds is the length of the soundboard response in the strike, the last boardFade of it
// fades out (a response that stops while it still sounds clicks)
const (
	boardSeconds = 0.35
	boardFade    = 0.15
)

var boards sync.Map // sample rate -> []float64

// soundboard returns the response of a soundboard to a tap at a sample rate: dense modes that die
// away fast, the highs fastest (a low body, a knock and a short bright click), at unit energy.
// The same instrument every time, so it has its own fixed seed
func soundboard(sr float64) []float64 {
	if b, ok := boards.Load(sr); ok {
		return b.([]float64)
	}
	rnd := rand.New(rand.NewPCG(7, 88))
	n := int(boardSeconds * sr)
	ir := make([]float64, n)

	onePole := func(fc float64) float64 { return 1 - math.Exp(-2*math.Pi*fc/sr) }
	var (
		lowA, midA, hiA = onePole(250), onePole(1800), onePole(2500)
		low1, low2      float64
		mid1, mid2      float64
		hi1             float64
	)
	for i := range ir {
		t := float64(i) / sr
		x := rnd.Float64()*2 - 1

		// the body: lows ringing longest
		low1 += lowA * (x - low1)
		low2 += lowA * (low1 - low2)
		// the knock: mids, shorter
		mid1 += midA * (x - mid1)
		mid2 += midA * (mid1 - mid2)
		mid := mid2 - low2
		// the click: highs, very short
		hi1 += hiA * (x - hi1)
		hi := x - hi1

		ir[i] = 6*low2*math.Exp(-t/0.07) + 2*mid*math.Exp(-t/0.03) + 0.6*hi*math.Exp(-t/0.006)
	}

	// no dc, a smooth end, unit energy
	hp, prev, out := 1-onePole(30), 0.0, 0.0
	fade := int(boardFade * sr)
	energy := 0.0
	for i, x := range ir {
		out = hp * (out + x - prev)
		prev = x
		ir[i] = out
		if left := n - 1 - i; left < fade {
			ir[i] *= 0.5 - 0.5*math.Cos(math.Pi*float64(left)/float64(fade))
		}
		energy += ir[i] * ir[i]
	}
	scale := 1 / math.Sqrt(energy)
	for i := range ir {
		ir[i] *= scale
	}
	b, _ := boards.LoadOrStore(sr, ir)
	return b.([]float64)
}

// excitation sets up the strike of the hammer: what reaches the strings and the knock of the
// soundboard heard directly. The hammer pushes the string for a few milliseconds: a smooth force
// pulse, shorter for the higher keys (smaller hammers) and for harder felt, and shorter and
// peakier the harder the key is played, because the felt stiffens as it is compressed (louder is
// brighter). A little of the soundboard response joins it for texture (commuted synthesis:
// everything is linear, so the soundboard can color the strike instead of the strings). The strike
// position leaves out the harmonics with a node where the hammer hits (not completely, a hammer is
// a few centimeters wide), and a soundboard radiates the lowest fundamentals weakly. The knock is
// the soundboard response to the strike, heard directly
func (v *Voice) excitation(f0, sr float64) {
	cfg := &v.cfg
	board := soundboard(sr)
	if cap(v.exc) < len(board) {
		v.exc = make([]float64, len(board))
		v.knock = make([]float64, len(board))
	}
	exc, knock := v.exc[:len(board)], v.knock[:len(board)]
	v.exc, v.knock = exc, knock

	h := math.Max(0, math.Min(v.hardness, 1))
	if v.soft {
		h = math.Max(0, h-softHardness)
	}

	// the hammer meets the string: the force it pushes with (the same push, the momentum, for
	// every strike: the level is set by the velocity, the shape of the push is the tone)
	f := hammerForce(v.hammer(v.pitch, h, f0), v.velocity, sr, &v.scratch)
	clear(exc)
	peak := 0.0
	for i, x := range f {
		if i < len(exc) {
			exc[i] = x
			peak = math.Max(peak, x)
		}
	}
	width := 1.0 / math.Max(peak, 1e-9) / sr // seconds, how long the push is (area 1 over its peak)

	// the knock and the texture are an impact: short, shorter in the treble (where the strings
	// die fast and a long thud or noise would outlast them, like a slow hammer)
	knockEnv, knockDecay := 1.0, math.Exp(-1/(interpolate(knockSeconds, v.pitch)*sr))

	// the soundboard texture and knock, through a lowpass as bright as the pulse
	s := math.Max(0, math.Min(cfg.Soundboard, 1))
	fc := math.Min(0.25*sr, 0.48/width)
	a := 1 - math.Exp(-2*math.Pi*fc/sr)
	l1, l2 := 0.0, 0.0
	for i, b := range board {
		l1 += a * (b - l1)
		l2 += a * (l1 - l2)
		exc[i] += textureLevel * s * l2 * knockEnv
		knock[i] = knockLevel * s * l2 * knockEnv
		knockEnv *= knockDecay
	}

	// the radiation: two one pole highpasses
	hp := math.Exp(-2 * math.Pi * radiationHz / sr)
	var h1, h1x, h2, h2x float64
	for i, x := range exc {
		h1 = hp * (h1 + x - h1x)
		h1x = x
		h2 = hp * (h2 + h1 - h2x)
		h2x = h1
		exc[i] = h2
	}

	// the strike position: the wave reflected from the near end follows the direct one, a comb
	// with notches at the harmonics that have a node where the hammer hits
	beta := math.Max(0.02, math.Min(v.strikeAt, 0.5))
	d := int(math.Round(beta * sr / f0))
	if d >= 1 {
		for i := len(exc) - 1; i >= d; i-- {
			exc[i] -= strikeReflection * exc[i-d]
		}
	}
}

// hammer is the hammer of a key and the string it strikes
type hammer struct {
	mass     float64 // kg
	k, p     float64 // the felt: its force is k compression^p (it stiffens as it is compressed)
	alpha    float64 // seconds, the hysteresis of the felt (it pushes harder going in than out)
	z        float64 // the impedance of the string, kg/s
	period   float64 // seconds, of the string
	strikeAt float64 // where on the string it hits, part of its length
}

// hammer returns the hammer of a key with a hardness of its felt: heavier and softer in the bass,
// lighter and harder at the top. The stiffness of the felt is set so the hammer touches the
// string as long as hammerContact says at velocity 0.8
func (v *Voice) hammer(pitch, hardness, f0 float64) hammer {
	h := hammer{
		mass:     interpolate(hammerMass, pitch) / 1000,
		p:        interpolate(feltExponent, pitch),
		alpha:    feltHysteresis,
		z:        interpolate(stringImpedance, pitch),
		period:   1 / f0,
		strikeAt: math.Max(0.02, math.Min(keyStrikeAt(&v.cfg, keyOf(pitch)), 0.5)),
	}
	// the felt of the key, fitted once (see feltStiffness), and how this strike differs: a harder
	// or softer felt touches the string shorter or longer, the stiffness goes with the contact
	// time to the power p+1
	base := math.Max(0, math.Min(v.cfg.Hardness+math.Max(0, math.Min(v.cfg.Variation, 1))*0.06*keyRandom(keyOf(pitch), 1), 1))
	h.k = feltStiffness(v.Config().SampleRate, pitch, base, h.strikeAt, f0)
	contact := (1.25 - 0.5*hardness) / (1.25 - 0.5*base)
	if v.soft {
		contact *= 1.5 // the soft pedal: an unworn, softer part of the felt
	}
	h.k *= math.Pow(contact, -(h.p + 1))
	return h
}

type feltKey struct {
	sr, pitch, hardness, strikeAt float64
}

var felts sync.Map // feltKey -> float64

// feltStiffness returns the stiffness of the felt of a key: fitted so that at velocity 0.8 the
// hammer pushes as sharply as hammerContact says (the push is as wide as a pulse of that contact),
// the other velocities follow from the felt and the string. Fitted once per key and kept
func feltStiffness(sr, pitch, hardness, strikeAt, f0 float64) float64 {
	key := feltKey{sr, math.Round(pitch*4) / 4, math.Round(hardness*100) / 100, math.Round(strikeAt*1000) / 1000}
	if k, ok := felts.Load(key); ok {
		return k.(float64)
	}
	h := hammer{
		mass:     interpolate(hammerMass, pitch) / 1000,
		p:        interpolate(feltExponent, pitch),
		alpha:    feltHysteresis,
		z:        interpolate(stringImpedance, pitch),
		period:   1 / f0,
		strikeAt: strikeAt,
	}
	// the target: as sharp as a pulse of the contact time (its area over its peak)
	contact := hammerContact(pitch) * 0.7 * (1.25 - 0.5*hardness)
	n := max(2, int(math.Round(contact*sr)))
	q := 1 + 3.2*(0.5+hardness)
	area := 0.0
	for i := range n {
		area += math.Pow(math.Sin(math.Pi*(float64(i)+0.5)/float64(n)), q)
	}
	target := area / sr // seconds

	vref := hammerVelocity(0.8)
	width := func(k float64) float64 {
		h.k = k
		f := hammerForce(h, 0.8, sr, nil)
		peak := 0.0
		for _, x := range f {
			peak = math.Max(peak, x)
		}
		return 1 / peak / sr
	}
	// a bracket around the stiffness (wider with a softer felt), then regula falsi on the logs
	// (the Illinois variant), the width falls as the stiffness rises
	compression := contact * vref / 2.9
	k := (h.p + 1) * h.mass * vref * vref / (2 * math.Pow(compression, h.p+1))
	miss := func(logK float64) float64 { return math.Log(width(math.Exp(logK)) / target) }
	a, fa := math.Log(k), 0.0
	fa = miss(a)
	b, fb := a, fa
	step := math.Log(10) * (h.p + 1) / 2
	bracketed := false
	for range 8 {
		if fa > 0 {
			b += step
		} else {
			b -= step
		}
		fb = miss(b)
		if (fa > 0) != (fb > 0) {
			bracketed = true
			break
		}
		a, fa = b, fb
	}
	if !bracketed { // out of reach: the closest
		felts.Store(key, math.Exp(b))
		return math.Exp(b)
	}
	k1 := b
	side := 0
	for range 16 {
		if math.Abs(fb) < 0.02 {
			break
		}
		k1 = b - fb*(b-a)/(fb-fa)
		f1 := miss(k1)
		if (f1 > 0) == (fb > 0) {
			b, fb = k1, f1
			if side == -1 {
				fa /= 2
			}
			side = -1
		} else {
			a, fa = b, fb
			b, fb = k1, f1
			if side == 1 {
				fa /= 2
			}
			side = 1
		}
	}
	k1 = math.Exp(b)
	felts.Store(key, k1)
	return k1
}

// hammerVelocity is how fast the hammer flies at a velocity, from about 0.25 m/s (the softest
// pianissimo) to 6 m/s (fortissimo)
func hammerVelocity(velocity float64) float64 { return 0.25 + 5.75*math.Pow(velocity, 1.6) }

// the hammers and strings of a grand, by key
var (
	hammerMass      = [][2]float64{{21, 11}, {60, 8.7}, {108, 6}}                     // grams
	feltExponent    = [][2]float64{{21, 3}, {60, 3.5}, {108, 4}}                      // the felt stiffening, more in the treble
	stringImpedance = [][2]float64{{21, 24}, {45, 32}, {60, 40}, {84, 56}, {108, 64}} // kg/s, an effective string: firmer than the bare string, so the felt shapes the push (the bridge, the other strings and the sharp front of the wave this simple string leaves out)
)

const (
	feltHysteresis   = 1.5e-4 // seconds
	hammerOversample = 4
	hammerMaxSeconds = 0.03
)

// hammerForce simulates a hammer flying at a velocity into a string and returns the force it
// pushes with, a sample at a time, at area 1. The string gives way under the hammer (its
// impedance) and the waves leaving the strike point come back from both ends upside down: from the
// near end while the hammer still touches the string in the treble, pushing it away, so the hammer
// can leave and hit again. The felt stiffens as it is compressed, so the harder the strike, the
// shorter and peakier the push and the brighter the tone
func hammerForce(h hammer, velocity, sr float64, scratch *hammerScratch) []float64 {
	dt := 1 / (sr * hammerOversample)
	steps := int(hammerMaxSeconds / dt)
	near := max(1, int(math.Round(2*h.strikeAt*h.period/dt)))
	far := max(1, int(math.Round(2*(1-h.strikeAt)*h.period/dt)))
	both := max(1, int(math.Round(2*h.period/dt)))
	if scratch == nil {
		scratch = &hammerScratch{}
	}
	if cap(scratch.history) < both+1 {
		scratch.history = make([]float64, both+1)
	}
	history := scratch.history[:both+1] // the force, the latest at pos
	clear(history)
	pos := 0
	at := func(d int) float64 { return history[((pos-d)%len(history)+len(history))%len(history)] }

	var (
		yh, vh      = 0.0, hammerVelocity(velocity) // the hammer, flying at the string
		ys          = 0.0                           // the string under it
		prevSpring  = 0.0
		out         = scratch.out[:0]
		sum         = 0.0
		lastContact = 0
	)
	// every step is solved for the compression where the hammer, the felt and the string agree
	// (implicit: the string gives way under the hammer like a damper, too fast for a plain step)
	c := dt*dt/h.mass + dt/(2*h.z)
	hyst := 1 + h.alpha/dt
	// felt returns the force and its slope at a compression
	felt := func(x float64) (float64, float64) {
		if x <= 0 {
			return 0, 0
		}
		spring := math.Exp(h.p * math.Log(x))
		f := h.k * (spring*hyst - h.alpha*prevSpring/dt)
		if f <= 0 {
			return 0, 0
		}
		return f, h.k * hyst * h.p * spring / x
	}
	guess := 0.0
	for step := range steps {
		// the waves coming back from the ends of the string
		back := -at(near-1) - at(far-1) + at(both-1)
		free := yh + dt*vh - ys - dt*back/(2*h.z) // the compression without a force
		compression := free
		force := 0.0
		if free > 0 {
			// compression + c force(compression) = free, between 0 and free (Newton, kept in the
			// bracket, from the compression of the step before)
			lo, hi := 0.0, free
			x := math.Min(math.Max(guess, free*1e-3), free)
			for range 30 {
				f, slope := felt(x)
				g := x + c*f - free
				if g > 0 {
					hi = x
				} else {
					lo = x
				}
				next := x - g/(1+c*slope)
				if next <= lo || next >= hi || math.IsNaN(next) {
					next = (lo + hi) / 2
				}
				if math.Abs(next-x) <= 1e-10*x {
					x = next
					break
				}
				x = next
			}
			compression = x
			force, _ = felt(x)
		}
		guess = compression
		if compression > 0 {
			prevSpring = math.Exp(h.p * math.Log(compression))
			lastContact = step
		} else {
			prevSpring = 0
		}
		vh -= force / h.mass * dt
		yh += vh * dt
		ys += dt * (force + back) / (2 * h.z)
		pos = (pos + 1) % len(history)
		history[pos] = force

		sum += force
		if (step+1)%hammerOversample == 0 {
			out = append(out, sum/hammerOversample)
			sum = 0
		}
		// done: the hammer has left the string for a while
		if compression < 0 && step-lastContact > int(0.0015/dt) {
			break
		}
	}
	area := 0.0
	for _, x := range out {
		area += x
	}
	if area <= 0 {
		scratch.out = append(out[:0], 1)
		return scratch.out
	}
	for i := range out {
		out[i] /= area
	}
	scratch.out = out
	return out
}

// hammerScratch holds what a simulation needs, kept by a voice so a strike allocates nothing
type hammerScratch struct {
	history, out []float64
}

// hammerContact is how long a hammer of a key touches the string at medium velocity, from about
// 4 ms in the bass to under 1 ms at the top
func hammerContact(pitch float64) float64 { return 0.004 * math.Exp2(-(pitch-21)/32) }

// knockSeconds is how long the knock of a key rings (to a third)
var knockSeconds = [][2]float64{{21, 0.08}, {60, 0.04}, {84, 0.02}, {108, 0.012}}

const (
	textureLevel     = 0.15 // the soundboard response in the strike, at Soundboard 1
	knockLevel       = 1.0  // the knock heard directly, at Soundboard 1
	radiationHz      = 50.0 // the soundboard radiates the fundamentals below this weakly
	strikeReflection = 0.85 // depth of the strike position notches
)

// keyStrikeAt returns where the hammer of a key hits: the setting, a little different for every key
func keyStrikeAt(cfg *Config, key int) float64 {
	return cfg.StrikePosition + math.Max(0, math.Min(cfg.Variation, 1))*0.008*keyRandom(key, 2)
}

// prepareHammers fits the felt of every key now (see feltStiffness), so no strike has to
func (v *Voice) prepareHammers() {
	for key := 21; key <= 108; key++ {
		pitch := float64(key)
		v.hammer(pitch, 0.5, 440*math.Exp2((stretched(pitch, &v.cfg)-69)/12))
	}
}
