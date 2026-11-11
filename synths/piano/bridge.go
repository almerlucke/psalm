package piano

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
)

// The bridge. The strings of all keys end on the same bridge, which sits on the soundboard: the
// force the strings put on it moves the board, and the motion of the bridge goes back into every
// string, so the strings are loaded by the board's admittance. A string loses most where one of
// its partials meets a mode of the board and little between the modes, and the energy of one
// string drives the strings of the keys beside it. That is what makes a chord more than the sum of
// its notes and what the sustain pedal washes with.
//
// The admittance is a bank of resonant modes shared by all voices (not one bank per voice): the
// forces of the strings are summed every sample, the modes ring, and their summed velocity goes
// back to the strings. The modes are sparse in the bass (the whole board moving) and denser and
// denser above, as many per octave as the board has until they overlap; wood is lossy, so they are
// damped (a low Q, lower still in the treble). Their strength is fitted so the mean loss of a
// string is what the bridge drain of the key is (see bridgeDrain): the modes shape how it loses,
// not how much. A string and the bridge exchange energy with the same coefficient, so the coupling
// never adds energy
const (
	bridgeFrom   = 35.0   // Hz, the lowest mode
	bridgeTo     = 8000.0 // Hz, the highest
	bridgeSpread = 0.35   // how much the strength of a mode varies (log, before the fit)
)

// bridgeModesPerOctave is how many modes the board has in the octaves from bridgeFrom up: the
// density rises until the modes overlap
var bridgeModesPerOctave = []int{2, 3, 5, 7, 10, 12, 12, 8}

// bridgeMode is a resonance of the board, a complex one pole resonator driven by the force
type bridgeMode struct {
	pole, z complex128
	gain    float64
}

// bridge is the shared admittance of the board at the feet of the strings
type bridge struct {
	modes  []bridgeMode
	f      float64 // the force of this sample, summed over the strings
	v      float64 // the velocity it moves the bridge with
	silent int     // samples without a force
}

// bridgeQ is the quality of a mode of the board: wood is lossy, the treble more
func bridgeQ(f float64) float64 {
	return max(6, (15+15*min(1, f/300))/(1+(f/2500)*(f/2500)))
}

// bridgeTarget is the mean real part of the admittance the modes are fitted to: what the old
// drain of a key was, the board taking the low frequencies of the bridge more than the highs
func bridgeTarget(f float64) float64 {
	return drainHigh + (drainLow-drainHigh)/(1+(f/drainHz)*(f/drainHz))
}

// admittance returns the real part of the admittance of the modes at a frequency
func (b *bridge) admittance(f, sr float64) float64 {
	w := 2 * math.Pi * f / sr
	e := cmplx.Exp(complex(0, -w))
	ei := cmplx.Exp(complex(0, w))
	y := 0.0
	for _, m := range b.modes {
		// the resonator takes the real part of a one sided response: the mean of both sides
		y += m.gain * 0.5 * (real(1/(1-m.pole*e)) + real(1/(1-m.pole*ei)))
	}
	return y
}

// newBridge builds the bridge of a soundboard at a sample rate. The modes are the same every
// time; the stream they are drawn from belongs to the bridge, not to the instrument
func newBridge(sr float64) *bridge {
	rnd := rand.New(rand.NewPCG(71, 83))
	b := &bridge{}
	amps := []float64{}
	lo := bridgeFrom
	for _, n := range bridgeModesPerOctave {
		for i := 0; i < n; i++ {
			// spread over the octave with a jitter, none on top of another
			f := lo * math.Exp2((float64(i)+0.2+0.6*rnd.Float64())/float64(n))
			if f >= min(bridgeTo, 0.45*sr) {
				continue
			}
			q := bridgeQ(f)
			r := math.Exp(-math.Pi * f / q / sr)
			b.modes = append(b.modes, bridgeMode{pole: cmplx.Rect(r, 2*math.Pi*f/sr)})
			amps = append(amps, math.Exp(bridgeSpread*rnd.NormFloat64()))
		}
		lo *= 2
	}
	freq := func(m bridgeMode) float64 { return cmplx.Phase(m.pole) * sr / (2 * math.Pi) }
	for k := range b.modes {
		b.modes[k].gain = 2 * (1 - cmplx.Abs(b.modes[k].pole)) * amps[k]
	}
	// fit: each mode scaled so the mean admittance around it is the target
	for range 8 {
		for k := range b.modes {
			f := freq(b.modes[k])
			mean := 0.0
			const points = 24
			for j := range points {
				mean += b.admittance(f*math.Exp2(-0.4+0.8*float64(j)/(points-1)), sr)
			}
			mean /= points
			if mean > 0 {
				b.modes[k].gain *= math.Pow(bridgeTarget(f)/mean, 0.7)
			}
		}
	}
	return b
}

// tick moves the bridge by the force of this sample
func (b *bridge) tick() {
	f := b.f
	b.f = 0
	if f == 0 {
		b.silent = min(b.silent+1, 30001)
		if b.silent > 30000 {
			if b.silent == 30001 {
				b.reset()
				b.silent = 30001
			}
			b.v = 0
			return
		}
	} else {
		b.silent = 0
	}
	cf := complex(f, 0)
	v := 0.0
	for k := range b.modes {
		m := &b.modes[k]
		m.z = m.pole*m.z + cf
		v += m.gain * real(m.z)
	}
	b.v = v
}

// reset silences the modes
func (b *bridge) reset() {
	for k := range b.modes {
		b.modes[k].z = 0
	}
	b.f, b.v, b.silent = 0, 0, 0
}
