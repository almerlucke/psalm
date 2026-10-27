package blosc

import (
	"github.com/almerlucke/psalm/blocks/sine"
	"math"
)

type Waveform int

const (
	Sine Waveform = iota
	Cosine
	Triangle
	Square
	Rectangle
	Sawtooth
	Ramp
	ModifiedTriangle
	ModifiedSquare
	HalfWaveRectifiedSine
	FullWaveRectifiedSine
	TriangularPulse
	TrapezoidFixed
	TrapezoidVariable
)

const (
	twoPi float64 = math.Pi * 2.0
)

type Blosc struct {
	fc  float64
	t   float64
	dt  float64
	pw  float64
	amp float64
	wf  Waveform
}

func New(fc float64, sr float64, phase float64, pw float64, amp float64, wf Waveform) *Blosc {
	m := &Blosc{}

	m.SetFrequency(fc, sr)
	m.SetPhase(phase)
	m.SetPulseWidth(pw)
	m.SetWaveform(wf)
	m.SetAmplitude(amp)

	return m
}

func (b *Blosc) SetFrequency(fc float64, sr float64) {
	b.fc = fc
	b.dt = fc / sr
}

func (b *Blosc) SetPhase(t float64) {
	b.t = t

	if b.t >= 0.0 {
		b.t -= integerPart(b.t)
	} else {
		b.t += 1.0 - integerPart(b.t)
	}
}

// Phase returns the current phase in the range 0-1
func (b *Blosc) Phase() float64 {
	return b.t
}

func (b *Blosc) SetPulseWidth(pw float64) {
	b.pw = math.Max(0.0, math.Min(1.0, pw))
}

func (b *Blosc) SetWaveform(wf Waveform) {
	b.wf = wf
}

func (b *Blosc) SetAmplitude(amp float64) {
	b.amp = amp
}

func (b *Blosc) Tick(phaseMod float64, sync bool, syncPhase float64) float64 {
	sample := b.get()

	if sync {
		b.t = syncPhase
	} else {
		b.t += b.dt + phaseMod
	}

	if b.t >= 0.0 {
		b.t -= integerPart(b.t)
	} else {
		b.t += 1.0 - integerPart(b.t)
	}

	return sample
}

func (b *Blosc) get() float64 {
	switch b.wf {
	case Sine:
		return b.sin()
	case Cosine:
		return b.cos()
	case Triangle:
		return b.tri()
	case Square:
		return b.sqr()
	case Rectangle:
		return b.rect()
	case Sawtooth:
		return b.saw()
	case Ramp:
		return b.ramp()
	case ModifiedTriangle:
		return b.tri2()
	case ModifiedSquare:
		return b.sqr2()
	case HalfWaveRectifiedSine:
		return b.half()
	case FullWaveRectifiedSine:
		return b.full()
	case TriangularPulse:
		return b.trip()
	case TrapezoidFixed:
		return b.trap()
	case TrapezoidVariable:
		return b.trap2()
	default:
		return 0.0
	}
}

func (b *Blosc) sin() float64 {
	return b.amp * sine.Cycle(b.t)
}

func (b *Blosc) cos() float64 {
	return b.amp * sine.CosCycle(b.t)
}

func (b *Blosc) half() float64 {
	t2 := b.t + 0.5
	t2 -= integerPart(t2)

	y := -2.0 / math.Pi

	if b.t < 0.5 {
		y = 2.0*sine.Cycle(b.t) - 2.0/math.Pi
	}

	y += twoPi * b.dt * (blamp(b.t, b.dt) + blamp(t2, b.dt))

	return b.amp * y
}

func (b *Blosc) full() float64 {
	t1 := b.t + 0.25
	t1 -= integerPart(t1)

	y := 2.0*sine.Cycle(0.5*t1) - 4.0/math.Pi
	y += twoPi * b.dt * blamp(t1, b.dt)

	return b.amp * y
}

func (b *Blosc) tri() float64 {
	t1 := b.t + 0.25
	t1 -= integerPart(t1)

	t2 := b.t + 0.75
	t2 -= integerPart(t2)

	y := b.t * 4.0

	if y >= 3.0 {
		y -= 4.0
	} else if y > 1.0 {
		y = 2.0 - y
	}

	y += 4.0 * b.dt * (blamp(t1, b.dt) - blamp(t2, b.dt))

	return b.amp * y
}

func (b *Blosc) tri2() float64 {
	pw := math.Max(0.0001, math.Min(0.9999, b.pw))

	t1 := b.t + 0.5*pw
	t1 -= integerPart(t1)

	t2 := b.t + 1.0 - 0.5*pw
	t2 -= integerPart(t2)

	y := b.t * 2.0

	if y >= 2.0-pw {
		y = (y - 2.0) / pw
	} else if y >= pw {
		y = 1.0 - (y-pw)/(1.0-pw)
	} else {
		y /= pw
	}

	y += b.dt / (pw - pw*pw) * (blamp(t1, b.dt) - blamp(t2, b.dt))

	return b.amp * y
}

func (b *Blosc) trip() float64 {
	t1 := b.t + 0.75 + 0.5*b.pw
	t1 -= integerPart(t1)

	var y float64

	if t1 >= b.pw {
		y = -b.pw
	} else {
		y = 4.0 * t1
		if y >= 2.0*b.pw {
			y = 4.0 - y/b.pw - b.pw
		} else {
			y = y/b.pw - b.pw
		}
	}

	if b.pw > 0.0 {
		t2 := t1 + 1.0 - 0.5*b.pw
		t2 -= integerPart(t2)

		t3 := t1 + 1.0 - b.pw
		t3 -= integerPart(t3)
		y += 2.0 * b.dt / b.pw * (blamp(t1, b.dt) - 2*blamp(t2, b.dt) + blamp(t3, b.dt))
	}

	return b.amp * y
}

func (b *Blosc) trap() float64 {
	y := 4 * b.t

	if y >= 3.0 {
		y -= 4.0
	} else if y > 1.0 {
		y = 2.0 - y
	}

	y = math.Max(-1, math.Min(1.0, 2.0*y))

	t1 := b.t + 0.125
	t1 -= integerPart(t1)

	t2 := t1 + 0.5
	t2 -= integerPart(t2)

	// Triangle #1
	y += 4.0 * b.dt * (blamp(t1, b.dt) - blamp(t2, b.dt))

	t1 = b.t + 0.375
	t1 -= integerPart(t1)

	t2 = t1 + 0.5
	t2 -= integerPart(t2)

	// Triangle #2
	y += 4.0 * b.dt * (blamp(t1, b.dt) - blamp(t2, b.dt))

	return b.amp * y
}

func (b *Blosc) trap2() float64 {
	pw := math.Min(0.9999, b.pw)
	scale := 1.0 / (1.0 - pw)

	y := 4.0 * b.t
	if y >= 3.0 {
		y -= 4.0
	} else if y > 1.0 {
		y = 2.0 - y
	}

	y = math.Max(-1.0, math.Min(1.0, scale*y))

	t1 := b.t + 0.25 - 0.25*pw
	t1 -= integerPart(t1)

	t2 := t1 + 0.5
	t2 -= integerPart(t2)

	// Triangle #1
	y += scale * 2.0 * b.dt * (blamp(t1, b.dt) - blamp(t2, b.dt))

	t1 = b.t + 0.25 + 0.25*pw
	t1 -= integerPart(t1)

	t2 = t1 + 0.5
	t2 -= integerPart(t2)

	// Triangle #2
	y += scale * 2.0 * b.dt * (blamp(t1, b.dt) - blamp(t2, b.dt))

	return b.amp * y
}

func (b *Blosc) sqr() float64 {
	t2 := b.t + 0.5
	t2 -= integerPart(t2)

	var y float64

	if b.t < 0.5 {
		y = 1.0
	} else {
		y = -1.0
	}

	y += blep(b.t, b.dt) - blep(t2, b.dt)

	return b.amp * y
}

func (b *Blosc) sqr2() float64 {
	t1 := b.t + 0.875 + 0.25*(b.pw-0.5)
	t1 -= integerPart(t1)

	t2 := b.t + 0.375 + 0.25*(b.pw-0.5)
	t2 -= integerPart(t2)

	// Square #1
	var y float64

	if t1 < 0.5 {
		y = 1.0
	} else {
		y = -1.0
	}

	y += blep(t1, b.dt) - blep(t2, b.dt)

	t1 += 0.5 * (1.0 - b.pw)
	t1 -= integerPart(t1)

	t2 += 0.5 * (1.0 - b.pw)
	t2 -= integerPart(t2)

	// Square #2
	if t1 < 0.5 {
		y += 1.0
	} else {
		y -= 1.0
	}

	y += blep(t1, b.dt) - blep(t2, b.dt)

	return b.amp * 0.5 * y
}

func (b *Blosc) rect() float64 {
	t2 := b.t + 1.0 - b.pw
	t2 -= integerPart(t2)

	y := -2.0 * b.pw
	if b.t < b.pw {
		y += 2
	}

	y += blep(b.t, b.dt) - blep(t2, b.dt)

	return b.amp * y
}

func (b *Blosc) saw() float64 {
	t1 := b.t + 0.5
	t1 -= integerPart(t1)

	y := 2.0*t1 - 1.0
	y -= blep(t1, b.dt)

	return b.amp * y
}

func (b *Blosc) ramp() float64 {
	t1 := b.t
	t1 -= integerPart(t1)

	y := 1.0 - 2.0*t1
	y += blep(t1, b.dt)

	return b.amp * y
}

func squareNumber(x float64) float64 {
	return x * x
}

// blep is the correction to add to a waveform with a jump of height 2 at phase 0 (and at every whole phase) so that it is band limited: the difference between the
// jump after a low pass of windowed sinc (8 samples to both sides, Blackman-Harris) and the jump itself, seen from the phase t (0-1) with a step of dt a sample.
// The jumps of the cycles before and after the one at hand are added when the kernel is longer than a cycle (high notes), so the alias stays low up the keyboard
func blep(t float64, dt float64) float64 {
	if dt <= 0 || math.IsNaN(t) || math.IsNaN(dt) {
		return 0
	}

	sum := 0.0

	for k := 0.0; ; k++ { // the jumps behind, in samples after them
		d := (t + k) / dt
		if d >= bleBreadth {
			break
		}

		sum += bleResidual(d)
	}

	for k := 0.0; ; k++ { // and ahead
		d := (t - 1 - k) / dt
		if d <= -bleBreadth {
			break
		}

		sum += bleResidual(d)
	}

	return sum
}

const (
	bleBreadth = 16   // the kernel reaches this many samples to both sides of the jump
	bleSteps   = 1024 // the table has this many entries to a sample
	bleCutoff  = 0.9  // the low pass of the kernel, as a part of the Nyquist frequency
	bleBeta    = 9.0  // the shape of the Kaiser window
)

// bessel0 is the modified Bessel function of the first kind and order 0
func bessel0(x float64) float64 {
	sum, term := 1.0, 1.0

	for k := 1.0; k < 50; k++ {
		term *= (x / (2 * k)) * (x / (2 * k))
		sum += term
	}

	return sum
}

var bleBefore, bleAfter = makeBleTables()

// makeBleTables are the residual of a jump of height 2 for the distances before it (-bleBreadth up to 0) and after it (0 up to bleBreadth), in samples: the integral of
// the windowed sinc minus the jump. The two sides are kept apart, so that the interpolation does not reach over the jump
func makeBleTables() (before, after []float64) {
	n := 2 * bleBreadth * bleSteps
	kernel := make([]float64, n+1)
	total := 0.0

	for i := range kernel {
		x := float64(i)/bleSteps - bleBreadth // samples from the jump
		w := float64(i) / float64(n)          // 0-1 over the window
		r := 2*w - 1
		window := bessel0(bleBeta*math.Sqrt(1-r*r)) / bessel0(bleBeta) // Kaiser
		sinc := bleCutoff

		if x != 0 {
			sinc = math.Sin(math.Pi*bleCutoff*x) / (math.Pi * x)
		}

		kernel[i] = sinc * window
		total += kernel[i]
	}

	total -= 0.5 * (kernel[0] + kernel[n])
	half := bleBreadth * bleSteps
	before, after = make([]float64, half+1), make([]float64, half+1)
	acc := 0.0

	for i := 0; i <= n; i++ {
		acc += 0.5 * kernel[i] // the integral up to this entry (the trapezoid: half of the entry itself)
		step := 2 * (acc / total)
		acc += 0.5 * kernel[i]

		if i <= half {
			before[i] = step
		}

		if i >= half {
			after[i-half] = step - 2
		}
	}

	return before, after
}

// bleResidual is the residual at d samples after a jump (negative: before it), linearly interpolated
func bleResidual(d float64) float64 {
	table := bleAfter
	pos := d * bleSteps

	if d < 0 {
		table = bleBefore
		pos = (d + bleBreadth) * bleSteps
	}

	i := int(pos)
	if i < 0 || i >= len(table)-1 {
		return 0
	}

	frac := pos - float64(i)

	return table[i] + frac*(table[i+1]-table[i])
}

func blamp(t float64, dt float64) float64 {
	if t < dt {
		t = t/dt - 1.0
		return -1.0 / 3.0 * squareNumber(t) * t
	} else if t > 1.0-dt {
		t = (t-1.0)/dt + 1.0
		return 1.0 / 3.0 * squareNumber(t) * t
	}

	return 0
}

func integerPart(x float64) float64 {
	ip, _ := math.Modf(x)
	return ip
}
