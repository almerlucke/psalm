package piano

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
	"sync"

	"github.com/almerlucke/psalm/blocks/fft"
)

// The soundboard every string sounds through: the response of a board of spruce, made of its
// modes. A soundboard has a few sparse modes in the bass (the whole board moving) and more and more
// above, about one every 25 Hz in the middle; wood loses energy fast, so the modes have a low Q
// (they ring for a fraction of a second in the bass and a few milliseconds in the treble). The
// board radiates the lowest frequencies weakly and its response falls off above a few kHz. Every
// side of the stereo image hears the board from another place: the same modes with other
// amplitudes and phases
const (
	boardModesFrom    = 45.0    // Hz, the lowest mode
	boardModesTo      = 10000.0 // Hz, the highest
	boardSpacingLow   = 40.0    // Hz between the modes below boardSpacingAt (on average)
	boardSpacingHigh  = 22.0    // Hz between the modes above
	boardSpacingAt    = 250.0
	boardQLow         = 15.0 // quality of the lowest modes
	boardQ            = 30.0 // quality above a few hundred Hz
	boardResponseSecs = 0.5
	boardResponseFade = 0.1
	boardPartition    = 128 // the convolution block: the latency of the instrument in samples
	boardShared       = 0.5 // how much of every mode both sides hear the same
)

var boardResponses sync.Map // sample rate -> [][]float64

// boardResponse returns the impulse response of the soundboard at a sample rate (left, right), at
// unit energy per side
func boardResponse(sr float64) [][]float64 {
	if r, ok := boardResponses.Load(sr); ok {
		return r.([][]float64)
	}
	rnd := rand.New(rand.NewPCG(11, 21))
	n := int(boardResponseSecs * sr)
	fade := int(boardResponseFade * sr)
	ir := [][]float64{make([]float64, n), make([]float64, n)}

	for f := boardModesFrom; f < math.Min(boardModesTo, 0.45*sr); {
		// the board: quality rising from the bass, how much it radiates and its high roll off
		q := boardQLow + (boardQ-boardQLow)*math.Min(1, f/300)
		decay := math.Exp(-math.Pi * f / q / sr) // per sample
		radiation := (f / 300) / math.Sqrt(1+(f/300)*(f/300))
		rolloff := 1 / math.Sqrt(1+(f/2500)*(f/2500))
		level := radiation * rolloff / math.Sqrt(q)

		// a mode heard from a place on the board: a random amplitude and phase, half of it the same
		// for both sides (the board moves as a whole), so the sides don't cancel in mono
		mode := func() complex128 {
			amp := math.Sqrt(-2 * math.Log(1-rnd.Float64()))
			return cmplx.Rect(amp, 2*math.Pi*rnd.Float64())
		}
		shared := mode()
		for _, ch := range ir {
			c := complex(level*math.Sqrt(boardShared), 0)*shared + complex(level*math.Sqrt(1-boardShared), 0)*mode()
			re, im := real(c), imag(c)
			rotRe, rotIm := decay*math.Cos(2*math.Pi*f/sr), decay*math.Sin(2*math.Pi*f/sr)
			for i := range ch {
				ch[i] += re
				re, im = re*rotRe-im*rotIm, re*rotIm+im*rotRe
				if math.Abs(re)+math.Abs(im) < 1e-9 {
					break
				}
			}
		}

		spacing := boardSpacingHigh
		if f < boardSpacingAt {
			spacing = boardSpacingLow
		}
		f += spacing * (0.3 + 1.4*rnd.Float64())
	}

	for _, ch := range ir {
		energy := 0.0
		for i := range ch {
			if left := n - 1 - i; left < fade {
				ch[i] *= 0.5 - 0.5*math.Cos(math.Pi*float64(left)/float64(fade))
			}
			energy += ch[i] * ch[i]
		}
		scale := 1 / math.Sqrt(energy)
		for i := range ch {
			ch[i] *= scale
		}
	}
	r, _ := boardResponses.LoadOrStore(sr, ir)
	return r.([][]float64)
}

// boardPower returns the power response of a soundboard response (the mean of its sides) and the
// width of a bin in Hz
func boardPower(ir [][]float64, sr float64) ([]float64, float64) {
	size := 1
	for size < len(ir[0]) {
		size *= 2
	}
	size *= 2
	f := fft.New(size)
	power := make([]float64, size/2)
	re, im := make([]float64, size), make([]float64, size)
	for _, ch := range ir {
		clear(re)
		clear(im)
		copy(re, ch)
		f.Forward(re, im)
		for k := range power {
			power[k] += (re[k]*re[k] + im[k]*im[k]) / float64(len(ir))
		}
	}
	return power, sr / float64(size)
}

// voicing returns the gain of every key that evens out what the soundboard does to it, like a
// piano technician voicing the hammers. Every key is modelled as its partials (stretched by the
// stiffness, shaped by the hammer and the strike position, every one as loud as long as it rings):
// how loud they come through the board and the direct sound together, against a board that is
// even everywhere
func voicing(power []float64, hz, sr float64, cfg Config) *[128]float64 {
	mean := 0.0
	for _, p := range power {
		mean += p
	}
	mean /= float64(len(power))
	body := math.Max(0, math.Min(cfg.Body, 1))
	direct := 1 - 0.7*body
	ref := direct*direct + body*body
	beta := math.Max(0.02, math.Min(cfg.StrikePosition, 0.5))
	h := math.Max(0, math.Min(cfg.Hardness, 1))

	var gains [128]float64
	for key := range gains {
		pitch := math.Max(21, math.Min(float64(key), 108))
		f0 := 440 * math.Exp2((pitch-69)/12)
		b := inharmonicity(pitch) * math.Max(0, cfg.Stiffness)
		t60 := ringTime(pitch) * math.Max(0.05, cfg.Decay)

		// the hammer pulse at velocity 0.8
		contact := hammerContact(pitch) * 0.7 * (1.25 - 0.5*h)
		n := max(2, int(math.Round(contact*sr)))
		q := 1 + 3.2*(0.5+h)
		pulse := make([]float64, n)
		for i := range pulse {
			pulse[i] = math.Pow(math.Sin(math.Pi*(float64(i)+0.5)/float64(n)), q)
		}
		d := math.Round(beta * sr / f0)

		sum, weights := 0.0, 0.0
		for k := 1.0; k <= 16; k++ {
			f := k * f0 * math.Sqrt((1+b*k*k)/(1+b))
			if f > 0.45*sr {
				break
			}
			w := 2 * math.Pi * f / sr
			var re, im float64
			for i, x := range pulse {
				re += x * math.Cos(w*float64(i))
				im -= x * math.Sin(w*float64(i))
			}
			comb := 1 + strikeReflection*strikeReflection - 2*strikeReflection*math.Cos(w*d)
			decay := math.Min(1/(1/t60+highDamping*math.Max(0, cfg.Damping)*f*f), 3) // seconds, heard for 3
			kw := 1 + 1.5*f*f/(f*f+1500*1500)                                        // the ear: the highs count more
			weight := (re*re + im*im) * comb * decay * kw

			// the partial is a few Hz wide (the strings beat, they are not exactly in tune)
			bin := int(f / hz)
			spread := max(1, int(3/hz))
			p := 0.0
			for i := max(0, bin-spread); i <= min(len(power)-1, bin+spread); i++ {
				p += power[i]
			}
			p /= float64(2*spread + 1)
			sum += weight * (direct*direct + body*body*p/mean)
			weights += weight
		}
		gains[key] = 1
		if weights > 0 && sum > 0 {
			gains[key] = math.Sqrt(ref / (sum / weights))
		}
	}
	return &gains
}
