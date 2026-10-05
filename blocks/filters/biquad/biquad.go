package biquad

import "math"

type Type int

const (
	LowPass Type = iota
	HighPass
	BandPass // constant 0 dB peak gain
	Notch
	Peak // peaking EQ, boosts or cuts around the frequency
	LowShelf
	HighShelf
	AllPass
)

// Biquad is a second order filter with the classic audio EQ cookbook (RBJ) coefficients, processed
// in transposed direct form II
type Biquad struct {
	typ    Type
	sr     float64
	fc     float64
	q      float64
	gainDB float64

	b0, b1, b2, a1, a2 float64
	z1, z2             float64
}

// New creates a biquad, q sets the bandwidth / resonance (0.707 is flat for low and high pass),
// gainDB is used by the peak and shelf types
func New(typ Type, fc float64, q float64, gainDB float64, sr float64) *Biquad {
	b := &Biquad{typ: typ, sr: sr, fc: fc, q: q, gainDB: gainDB}
	b.update()
	return b
}

func (b *Biquad) Set(typ Type, fc float64, q float64, gainDB float64) {
	b.typ = typ
	b.fc = fc
	b.q = q
	b.gainDB = gainDB
	b.update()
}

func (b *Biquad) SetFrequency(fc float64) {
	b.fc = fc
	b.update()
}

func (b *Biquad) update() {
	fc := math.Max(1.0, math.Min(b.fc, b.sr*0.49))
	q := math.Max(0.05, b.q)

	w0 := 2.0 * math.Pi * fc / b.sr
	cosW := math.Cos(w0)
	alpha := math.Sin(w0) / (2.0 * q)
	A := math.Pow(10.0, b.gainDB/40.0)

	var b0, b1, b2, a0, a1, a2 float64

	switch b.typ {
	case HighPass:
		b0, b1, b2 = (1.0+cosW)/2.0, -(1.0 + cosW), (1.0+cosW)/2.0
		a0, a1, a2 = 1.0+alpha, -2.0*cosW, 1.0-alpha
	case BandPass:
		b0, b1, b2 = alpha, 0.0, -alpha
		a0, a1, a2 = 1.0+alpha, -2.0*cosW, 1.0-alpha
	case Notch:
		b0, b1, b2 = 1.0, -2.0*cosW, 1.0
		a0, a1, a2 = 1.0+alpha, -2.0*cosW, 1.0-alpha
	case Peak:
		b0, b1, b2 = 1.0+alpha*A, -2.0*cosW, 1.0-alpha*A
		a0, a1, a2 = 1.0+alpha/A, -2.0*cosW, 1.0-alpha/A
	case LowShelf:
		sq := 2.0 * math.Sqrt(A) * alpha
		b0 = A * ((A + 1.0) - (A-1.0)*cosW + sq)
		b1 = 2.0 * A * ((A - 1.0) - (A+1.0)*cosW)
		b2 = A * ((A + 1.0) - (A-1.0)*cosW - sq)
		a0 = (A + 1.0) + (A-1.0)*cosW + sq
		a1 = -2.0 * ((A - 1.0) + (A+1.0)*cosW)
		a2 = (A + 1.0) + (A-1.0)*cosW - sq
	case HighShelf:
		sq := 2.0 * math.Sqrt(A) * alpha
		b0 = A * ((A + 1.0) + (A-1.0)*cosW + sq)
		b1 = -2.0 * A * ((A - 1.0) + (A+1.0)*cosW)
		b2 = A * ((A + 1.0) + (A-1.0)*cosW - sq)
		a0 = (A + 1.0) - (A-1.0)*cosW + sq
		a1 = 2.0 * ((A - 1.0) - (A+1.0)*cosW)
		a2 = (A + 1.0) - (A-1.0)*cosW - sq
	case AllPass:
		b0, b1, b2 = 1.0-alpha, -2.0*cosW, 1.0+alpha
		a0, a1, a2 = 1.0+alpha, -2.0*cosW, 1.0-alpha
	default: // LowPass
		b0, b1, b2 = (1.0-cosW)/2.0, 1.0-cosW, (1.0-cosW)/2.0
		a0, a1, a2 = 1.0+alpha, -2.0*cosW, 1.0-alpha
	}

	b.b0, b.b1, b.b2 = b0/a0, b1/a0, b2/a0
	b.a1, b.a2 = a1/a0, a2/a0
}

func (b *Biquad) Clear() {
	b.z1, b.z2 = 0.0, 0.0
}

func (b *Biquad) Process(x float64) float64 {
	y := b.b0*x + b.z1
	b.z1 = b.b1*x - b.a1*y + b.z2
	b.z2 = b.b2*x - b.a2*y
	return y
}
