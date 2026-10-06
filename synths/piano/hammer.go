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

	h := math.Max(0, math.Min(cfg.Hardness, 1))
	if v.soft {
		h = math.Max(0, h-softHardness)
	}
	vel := v.velocity

	// the force pulse: contact time and peakiness
	contact := hammerContact(v.pitch) * (1.5 - vel) * (1.25 - 0.5*h)
	q := 1 + 4*vel*(0.5+h)
	if v.soft {
		// the soft pedal: the hammer strikes with an unworn, softer part of the felt
		contact *= 1.25
		q = 1 + (q-1)*0.75
	}
	n := max(2, int(math.Round(contact*sr)))
	area := 0.0
	clear(exc)
	for i := range min(n, len(exc)) {
		f := math.Pow(math.Sin(math.Pi*(float64(i)+0.5)/float64(n)), q)
		exc[i] = f
		area += f
	}
	for i := range min(n, len(exc)) {
		exc[i] /= area // the same push (momentum) for every pulse, the level is set by the velocity
	}

	// the soundboard texture and knock, through a lowpass as bright as the pulse
	s := math.Max(0, math.Min(cfg.Soundboard, 1))
	fc := math.Min(0.25*sr, 0.6/(contact/math.Sqrt(q)))
	a := 1 - math.Exp(-2*math.Pi*fc/sr)
	l1, l2 := 0.0, 0.0
	for i, b := range board {
		l1 += a * (b - l1)
		l2 += a * (l1 - l2)
		exc[i] += textureLevel * s * l2
		knock[i] = knockLevel * s * l2
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
	beta := math.Max(0.02, math.Min(cfg.StrikePosition, 0.5))
	d := int(math.Round(beta * sr / f0))
	if d >= 1 {
		for i := len(exc) - 1; i >= d; i-- {
			exc[i] -= strikeReflection * exc[i-d]
		}
	}
}

// hammerContact is how long a hammer of a key touches the string at medium velocity, from about
// 4 ms in the bass to under 1 ms at the top
func hammerContact(pitch float64) float64 { return 0.004 * math.Exp2(-(pitch-21)/32) }

const (
	textureLevel     = 0.15 // the soundboard response in the strike, at Soundboard 1
	knockLevel       = 1.0  // the knock heard directly, at Soundboard 1
	radiationHz      = 50.0 // the soundboard radiates the fundamentals below this weakly
	strikeReflection = 0.85 // depth of the strike position notches
)
