package piano

import (
	"math"
	"math/cmplx"
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
	knockKey := interpolate(knockByKey, v.pitch) // the treble ticks softly, its notes are nearly pure
	// the impact grows faster than the strike: a key played softly hardly knocks
	knockKey *= math.Pow(hammerVelocity(v.velocity)/hammerVelocity(0.8), knockTouch)

	// the soundboard texture and knock, through a lowpass as bright as the pulse; the knock also
	// through a highpass that rises with the key (a small hammer near the thin edge of the board
	// ticks, a big one in the middle of it thuds)
	s := math.Max(0, math.Min(cfg.Soundboard, 1))
	fc := math.Min(math.Min(0.25*sr, 0.48/width), knockMaxHz) // a soft tick, not a click
	a := 1 - math.Exp(-2*math.Pi*fc/sr)
	ka := 1 - math.Exp(-2*math.Pi*interpolate(knockLowHz, v.pitch)/sr)
	l1, l2, k1, k2 := 0.0, 0.0, 0.0, 0.0
	for i, b := range board {
		l1 += a * (b - l1)
		l2 += a * (l1 - l2)
		exc[i] += textureLevel * s * l2 * knockEnv
		k1 += ka * (l2 - k1)
		k2 += ka * (l2 - k1 - k2)
		knock[i] = knockLevel * s * (l2 - k1 - k2) * knockEnv * knockKey
		knockEnv *= knockDecay
	}

	// in the treble the felt touches the string for longer than a period (hammerContact keeps the
	// push shorter, so it leaves no key out): that smooths away what is above the fundamental, a
	// steep lowpass just above it. It comes in gradually over the keys (from where the felt
	// touches 1.1 periods, fully at smoothingSpan more), and leaves the fundamental as loud
	if lp, ok := smoothingPole(f0, v.pitch, h, sr); ok {
		gain := 1 / smoothingGain(lp, f0, sr)
		var s1, s2, s3, s4 float64
		for i, x := range exc {
			s1 += lp * (x - s1)
			s2 += lp * (s1 - s2)
			s3 += lp * (s2 - s3)
			s4 += lp * (s3 - s4)
			exc[i] = s4 * gain
		}
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

	// a string can't ring below its fundamental: what the push has down there would pass the
	// strings as an unpitched thump, louder than the note itself in the treble (where the push is
	// longer than a period). A steep highpass under the fundamental (Butterworth, two biquads)
	w := 2 * math.Pi * math.Min(fundamentalCut*f0, 0.45*sr) / sr
	for _, q := range [2]float64{0.5412, 1.3066} {
		alpha := math.Sin(w) / (2 * q)
		a0 := 1 + alpha
		b0 := (1 + math.Cos(w)) / 2 / a0
		b1 := -2 * b0
		a1, a2 := -2*math.Cos(w)/a0, (1-alpha)/a0
		var x1, x2, y1, y2 float64
		for i, x := range exc {
			y := b0*x + b1*x1 + b0*x2 - a1*y1 - a2*y2
			x2, x1 = x1, x
			y2, y1 = y1, y
			exc[i] = y
		}
	}

	// the strike position: the wave reflected from the near end follows the direct one, a comb
	// with notches at the harmonics that have a node where the hammer hits
	beta := math.Max(0.02, math.Min(v.strikeAt, 0.5))
	// (in the treble the notches are far above hearing, and a comb of a sample or two is only a
	// crude highpass: left out)
	if d, ok := strikeComb(f0, beta, sr); ok {
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
		contact *= 1.8 // the soft pedal: an unworn, softer part of the felt
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
	feltExponent    = [][2]float64{{21, 5}, {48, 4}, {60, 4}, {108, 4}}               // the felt stiffening, more in the treble
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
// hammerContact is how long a hammer of a key touches the string at medium velocity (seconds),
// fitted to the brightness of a recorded concert grand
func hammerContact(pitch float64) float64 {
	// at most 1.1 periods of the string: then the fundamental is below the first null of the
	// spectrum of the push (a longer push would leave out the fundamental of some keys) and the
	// second partial beyond it (the treble stays nearly pure)
	f0 := 440 * math.Exp2((pitch-69)/12)
	return math.Min(interpolate(contactTable, pitch)/1000, 1.1/f0)
}

var contactTable = [][2]float64{{21, 2.4}, {36, 1.6}, {48, 1.62}, {60, 1.63}, {72, 1.57}, {84, 1.4}, {96, 1.3}, {108, 0.9}} // ms

// knockByKey is how loud the knock of a key is: in the bass fitted to stay under the note (from
// 18 dB under it at key 53 to 24 dB at the bottom, where a low thud stands out most)
var knockByKey = [][2]float64{{21, 0.43}, {24, 0.42}, {27, 0.41}, {30, 0.45}, {33, 0.53}, {36, 0.5}, {39, 0.45},
	{42, 0.46}, {45, 0.91}, {48, 0.94}, {51, 0.92}, {54, 0.76}, {57, 1}, {72, 1}, {84, 0.5}, {96, 0.25}, {108, 0.15}}

// trebleSmoothing is where (times the fundamental) the long felt contact of the treble smooths the
// strike away
const trebleSmoothing = 1.4

// smoothingSpan is how many periods more of contact the treble smoothing takes to come in fully
const smoothingSpan = 0.35

// smoothingPole returns the coefficient of the four one pole lowpasses of the treble smoothing of
// a key with a hardness of its felt, if it has any
func smoothingPole(f0, pitch, hardness, sr float64) (float64, bool) {
	periods := f0 * interpolate(contactTable, pitch) / 1000
	if periods <= 1.1 {
		return 0, false
	}
	// a hard hammer (a toy piano's, on metal) touches shortly and leaves the overtones
	smoothing := trebleSmoothing * (1 + 10*math.Max(0, hardness-0.6))
	smoothing /= math.Max(math.Min((periods-1.1)/smoothingSpan, 1), 0.05)
	return 1 - math.Exp(-2*math.Pi*math.Min(smoothing*f0, 0.45*sr)/sr), true
}

// smoothingGain returns the gain of the treble smoothing at a frequency
func smoothingGain(lp, f, sr float64) float64 {
	at := lp / cmplx.Abs(1-complex(1-lp, 0)*cmplx.Exp(complex(0, -2*math.Pi*f/sr)))
	return at * at * at * at
}

// strikeComb returns whether the strike position notches are made for a key (in the treble they
// are far above hearing) and the delay that makes them
func strikeComb(f0, beta, sr float64) (int, bool) {
	d := int(math.Round(beta * sr / f0))
	return d, d >= 1 && f0/beta < 0.3*sr
}

// knockLowHz is where the knock of a key is highpassed: a low thud in the bass, a tick at the top
var knockLowHz = [][2]float64{{21, 30}, {60, 80}, {72, 200}, {84, 450}, {96, 900}, {108, 1500}}

// knockTouch is how much faster than the strike the knock grows with the speed of the hammer
const knockTouch = 0.5

// knockMaxHz is the brightest the knock gets
const knockMaxHz = 4000.0

// knockSeconds is how long the knock of a key rings (to a third)
var knockSeconds = [][2]float64{{21, 0.05}, {60, 0.04}, {84, 0.02}, {108, 0.012}}

const (
	textureLevel     = 0.15 // the soundboard response in the strike, at Soundboard 1
	knockLevel       = 1.0  // the knock heard directly, at Soundboard 1
	radiationHz      = 80.0 // the soundboard radiates the fundamentals below this weakly
	strikeReflection = 0.85 // depth of the strike position notches
	fundamentalCut   = 0.6  // the strike is highpassed here (times the fundamental), see excitation
)

// keyStrikeAt returns where the hammer of a key hits: the setting, a little different for every key
func keyStrikeAt(cfg *Config, key int) float64 {
	return cfg.StrikePosition*strikeRegister(cfg, float64(key)) + math.Max(0, math.Min(cfg.Variation, 1))*0.008*keyRandom(key, 2)
}

// strikeRegister returns the factor of the strike point of a pitch: the hammers of a piano hit
// the long bass strings at 1/8 of their length, the strike moves towards the end (1/12) in the
// treble, where a hammer is as wide as a fair part of the string and the fundamental should
// speak. The setting is the bass, the register moves it to strikeTreble times that at the top
func strikeRegister(cfg *Config, pitch float64) float64 {
	amount := math.Max(0, math.Min(cfg.StrikeRegister, 1))
	x := math.Max(0, math.Min((pitch-strikeFrom)/(strikeTo-strikeFrom), 1))
	x = x * x * (3 - 2*x)
	return 1 - amount*x*(1-strikeTreble)
}

const (
	strikeFrom   = 48.0
	strikeTo     = 100.0
	strikeTreble = 2.0 / 3 // 1/8 to 1/12
)

// prepareHammers fits the felt of every key now (see feltStiffness), so no strike has to
func (v *Voice) prepareHammers() {
	for key := 21; key <= 108; key++ {
		pitch := float64(key)
		v.hammer(pitch, 0.5, 440*math.Exp2((stretched(pitch, &v.cfg)-69)/12))
	}
}
